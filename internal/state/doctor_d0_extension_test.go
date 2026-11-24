package state

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// Test-only private snapshot of a deliberately stationary writer. Never
// certify an arbitrary live file copy as a coherent snapshot.
func TestPR177D0ExtensionPrivateWALSnapshot(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	rw, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = rw.Close(); err != nil {
		t.Fatal(err)
	}
	rw, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()
	if _, err = rw.db.Exec("PRAGMA wal_autocheckpoint=0"); err != nil {
		t.Fatal(err)
	}
	if _, err = rw.db.Exec("INSERT INTO oauth_attempts(at,kind) VALUES(9,'d0-snapshot')"); err != nil {
		t.Fatal(err)
	}
	before := d0FileNames(t, dir)
	files := map[string][]byte{}
	for _, name := range []string{"ledger.sqlite", "ledger.sqlite-wal", "ledger.sqlite-shm"} {
		files[name], err = os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
	}
	// Writer intentionally stays open but makes no further writes.
	// Rebuild index ONLY on private copy, never copy stale SHM.
	dst := t.TempDir()
	if err = os.Chmod(dst, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ledger.sqlite", "ledger.sqlite-wal"} {
		if err = os.WriteFile(filepath.Join(dst, name), files[name], 0600); err != nil {
			t.Fatal(err)
		}
	}
	copyStore, err := OpenNetworkAuditReadOnly(filepath.Join(dst, "ledger.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err = copyStore.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_attempts WHERE kind='d0-snapshot'").Scan(&n); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if err = copyStore.Close(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, d0FileNames(t, dir)) {
		t.Fatal("source inventory changed")
	}
	for name, want := range files {
		got, e := os.ReadFile(filepath.Join(dir, name))
		if e != nil || !reflect.DeepEqual(got, want) {
			t.Fatal("source file changed", name, e)
		}
	}
	t.Log("D0X_PRIVATE_STATIONARY_WAL_COPY_VISIBLE_SOURCE_UNCHANGED")
	// A main-file-only immutable copy silently misses committed data.
	bare := filepath.Join(t.TempDir(), "bare.sqlite")
	if err = os.WriteFile(bare, files["ledger.sqlite"], 0600); err != nil {
		t.Fatal(err)
	}
	uri := (&url.URL{Scheme: "file", Path: bare, RawQuery: "mode=ro&immutable=1&_query_only=1"}).String()
	db, err := sql.Open("sqlite3", uri)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.QueryRow("SELECT COUNT(*) FROM oauth_attempts WHERE kind='d0-snapshot'").Scan(&n); err != nil || n != 0 {
		t.Fatal("main-only snapshot unexpectedly current", n, err)
	}
	t.Log("D0X_MAIN_ONLY_COPY_STALE")
}

func TestPR177D0ExtensionScopedRecoveryReceiptAndReview(t *testing.T) {
	ctx := context.Background()
	for _, allow := range []bool{true, false} {
		t.Run(map[bool]string{true: "grant", false: "revoke"}[allow], func(t *testing.T) {
			s, ro, id := pr178ScopedReadyToStage(t, allow)
			if err := s.StageAndClaimScopedPeerOriginal(ctx, id); err != nil {
				t.Fatal(err)
			}
			if err := s.Unknown(ctx, id, "DISPATCHING", "SCOPED_PEER_V2_ROOT_OUTCOME_UNVERIFIED"); err != nil {
				t.Fatal(err)
			}
			target, err := s.ReadRecoveryTarget(ctx, "user-1001", id)
			if err != nil || !target.ScopedPeer || target.NativeID != "" || target.State != "UNKNOWN" {
				t.Fatal(target, err)
			}
			_, tok, err := s.IssueToken(ctx, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			// Report and evidence reads are not a recovery attempt.
			before, err := s.ReadRecoveryNavigation(ctx, target.Layer, id)
			if err != nil || before.ClaimProtection != "RETAINED" || before.Disposition != "INSPECT_REQUIRED" {
				t.Fatal(before, err)
			}
			view, err := s.ReadActivity(ctx, ActivityQuery{Layer: target.Layer, Requester: tok.ID, View: "issues", Limit: 50})
			if err != nil {
				t.Fatal(err)
			}
			seen := false
			for _, i := range view.Issues {
				if i.OperationID == id && i.Reason == "PROJECT_PEER_BARRIER" && i.Disposition == "MCP_RECOVERY_INSPECTION_REQUIRED" {
					seen = true
				}
			}
			if !seen {
				t.Fatal("expected scoped MCP inspection route", view)
			}
			t.Log("D2_SCOPED_ACTIVITY_MCP_INSPECT_NOT_PROOF")
			req := RecoveryRequest{Layer: target.Layer, ActorID: tok.ID, TokenID: tok.ID, OriginalID: id, Key: "d0x-blocked-original-recovery"}
			blocked, err := s.CommitRecovery(ctx, req, target, RecoveryProof{Kind: RecoveryBlocked, Reason: "SCOPED_PEER_ROOT_PROOF_REQUIRED"})
			if err != nil || blocked.Receipt.State != "FAILED" || blocked.Disposition != RecoveryBlocked {
				t.Fatal(blocked, err)
			}
			got, err := s.ReadRecoveryTarget(ctx, target.Layer, id)
			if err != nil || got != target {
				t.Fatal("blocked receipt changed original", got, err)
			}
			receipt := scopedTestReceipt(t, ro, id, true)
			proof := RecoveryProof{Kind: RecoverySettle, Reason: "SCOPED_PEER_COMPLETED_NO_REPLAY", UUID: target.UUID, ScopedPeer: receipt, Fingerprint: "synthetic-d0x-witness"}
			old, err := s.CommitRecovery(ctx, req, target, proof)
			if err != nil || !old.Receipt.Duplicate || old.Receipt.ID != blocked.Receipt.ID || old.Disposition != RecoveryBlocked {
				t.Fatal("same key did not replay blocked result", old, err)
			}
			t.Log("D0X_BLOCKED_KEY_REPLAY_REMAINS_BLOCKED")
			req.Key = "d0x-complete-original-recovery"
			// A new receipt is still insufficient if original revision changed.
			stale := target
			stale.UpdatedAt--
			if _, err = s.CommitRecovery(ctx, req, stale, proof); !errors.Is(err, ErrConflict) {
				t.Fatal("stale original accepted", err)
			}
			wrong := proof
			wrong.ScopedPeer.RecoveryOnly = false
			if _, err = s.CommitRecovery(ctx, req, target, wrong); err == nil {
				t.Fatal("non-recovery receipt accepted")
			}
			wrong = proof
			wrong.ScopedPeer.ProofSHA256 = wrong.ScopedPeer.PlanSHA256
			if _, err = s.CommitRecovery(ctx, req, target, wrong); err == nil {
				t.Fatal("wrong scoped proof accepted")
			}
			if _, err = s.ReplayRecovery(ctx, req); !errors.Is(err, ErrNotFound) {
				t.Fatal("failed plan consumed key", err)
			}
			settled, err := s.CommitRecovery(ctx, req, target, proof)
			if err != nil || settled.Receipt.State != "SUCCEEDED" || settled.Disposition != "SETTLED" || settled.Receipt.NativeOperationID != "" {
				t.Fatal(settled, err)
			}
			got, err = s.ReadRecoveryTarget(ctx, target.Layer, id)
			if err != nil || got.State != "SUCCEEDED" || got.NativeID != "" || got.Released {
				t.Fatal(got, err)
			}
			nav, err := s.ReadRecoveryNavigation(ctx, target.Layer, id)
			if err != nil || nav.Disposition != "RECOVERY_SETTLED" {
				t.Fatal(nav, err)
			}
			var live, n int
			if err = s.db.QueryRow("SELECT COUNT(*) FROM operation_resource_claims c JOIN operations o ON c.operation_id=o.id WHERE c.operation_id=? AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')", id).Scan(&live); err != nil || live != 0 {
				t.Fatal(live, err)
			}
			if err = s.db.QueryRow("SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=?", id).Scan(&n); err != nil || n != 5 {
				t.Fatal(n, err)
			}
			dup, err := s.ReplayRecovery(ctx, req)
			if err != nil || !dup.Receipt.Duplicate || dup.Receipt.ID != settled.Receipt.ID {
				t.Fatal(dup, err)
			}
			t.Log("D0X_SCOPED_SETTLEMENT_AUDIT_REPLAY_LIVE_CLAIMS_ZERO_HISTORY_FIVE")
		})
	}
}
