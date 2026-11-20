package state

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const rNative = "12345678-1234-4abc-8def-123456789abc"
const rUUID = "87654321-4321-4abc-8def-abcdef012345"

func recoveryOriginal(t *testing.T, s *Store, action, uuid string) (RecoveryRequest, RecoveryTarget) {
	t.Helper()
	ctx := context.Background()
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	if uuid != "" {
		if e = s.RegisterOwnedBox(ctx, uuid, "recovery-test", "owned", uuid); e != nil {
			t.Fatal(e)
		}
	}
	orig, e := s.Admit(ctx, Admission{Layer: "recovery-test", ActorID: tok.ID, TokenID: tok.ID, Key: "original-recovery-fixture-key", Action: action, RequestID: "original", ResourceKey: "box:owned", BoxUUID: uuid, Incarnation: uuid, SafeIntent: []byte(`{"name":"owned"}`)})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, orig.ID); e != nil {
		t.Fatal(e)
	}
	from := "DISPATCHING"
	if uuid != "" {
		if e = s.NativeAck(ctx, orig.ID, rNative); e != nil {
			t.Fatal(e)
		}
		from = "VERIFYING"
	}
	if e = s.Unknown(ctx, orig.ID, from, "TEST_ORIGINAL_UNKNOWN"); e != nil {
		t.Fatal(e)
	}
	r := RecoveryRequest{Layer: "recovery-test", ActorID: tok.ID, TokenID: tok.ID, OriginalID: orig.ID, Key: "new-recovery-request-key-001", WorkRef: "s5a2/test"}
	target, e := s.ReadRecoveryTarget(ctx, r.Layer, r.OriginalID)
	if e != nil {
		t.Fatal(e)
	}
	return r, target
}
func recoverProof(t RecoveryTarget) RecoveryProof {
	return RecoveryProof{Kind: RecoverySettle, Reason: "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED", NativeID: t.NativeID, UUID: t.UUID, Fingerprint: "test-proof"}
}
func TestS5A2RecoveryAtomicReceiptAndReplay(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "recovery.db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	r, target := recoveryOriginal(t, s, "box_start", rUUID)
	// Simulated lost COMMIT response: discard return before reopening DB.
	first, e := s.CommitRecovery(ctx, r, target, recoverProof(target))
	if e != nil {
		t.Fatal(e)
	}
	if first.Receipt.ID == target.ID || first.Receipt.State != "SUCCEEDED" || first.Receipt.NativeOperationID != "" || first.Disposition != "SETTLED" {
		t.Fatalf("wrong recovery receipt %+v", first)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	duplicate, e := s.ReplayRecovery(ctx, r)
	if e != nil || !duplicate.Receipt.Duplicate || duplicate.Receipt.ID != first.Receipt.ID || duplicate.Disposition != "SETTLED" {
		t.Fatalf("lost receipt %+v %v", duplicate, e)
	}
	current, e := s.ReadRecoveryTarget(ctx, r.Layer, r.OriginalID)
	if e != nil || current.State != "SUCCEEDED" || current.NativeID != rNative || current.Released {
		t.Fatalf("wrong original %+v %v", current, e)
	}
	var n int
	if e = s.db.QueryRow("SELECT count(*) FROM operation_events WHERE operation_id=? AND state='UNKNOWN'", target.ID).Scan(&n); e != nil || n != 1 {
		t.Fatalf("lost unknown history: %d %v", n, e)
	}
	if e = s.db.QueryRow("SELECT count(*) FROM recovery_audit WHERE operation_id=? AND decision='RECOVERY_SETTLED'", target.ID).Scan(&n); e != nil || n != 1 {
		t.Fatalf("lost audit: %d %v", n, e)
	}
	if _, e = s.ReadRecoveryTarget(ctx, "foreign", target.ID); !errors.Is(e, ErrNotFound) {
		t.Fatalf("cross-layer allowed: %v", e)
	}
	wrong := r
	wrong.Stream = "stdout"
	if _, e = s.ReplayRecovery(ctx, wrong); !errors.Is(e, ErrConflict) {
		t.Fatalf("same key changed intent: %v", e)
	}
	wrong = r
	wrong.WorkRef = "different"
	if _, e = s.ReplayRecovery(ctx, wrong); !errors.Is(e, ErrConflict) {
		t.Fatalf("same key changed attribution: %v", e)
	}
	if e = s.RevokeToken(ctx, r.TokenID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ReplayRecovery(ctx, r); !errors.Is(e, ErrRevoked) {
		t.Fatalf("revoked replay: %v", e)
	}
}
func TestS5A2RecoveryCommitFaultRollsBackBoth(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	r, target := recoveryOriginal(t, s, "box_stop", rUUID)
	// Test-only fault inside the same SQLite transaction, after the original
	// terminal UPDATE but before its recovery receipt event can commit.
	if _, e := s.db.Exec(`CREATE TRIGGER s5a2_fail BEFORE INSERT ON operation_events WHEN NEW.safe_detail LIKE 'SETTLED:%' BEGIN SELECT RAISE(ABORT,'injected recovery event fault'); END`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.CommitRecovery(ctx, r, target, recoverProof(target)); e == nil {
		t.Fatal("fault not observed")
	}
	if got, e := s.ReadRecoveryTarget(ctx, r.Layer, target.ID); e != nil || got != target {
		t.Fatalf("partial original commit %+v %v", got, e)
	}
	if _, e := s.ReplayRecovery(ctx, r); !errors.Is(e, ErrNotFound) {
		t.Fatalf("partial recovery receipt: %v", e)
	}
	if _, e := s.db.Exec("DROP TRIGGER s5a2_fail"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.CommitRecovery(ctx, r, target, recoverProof(target)); e != nil {
		t.Fatal(e)
	}
}
func TestS5A2RecoveryConcurrentSameKeyAndDifferentKey(t *testing.T) {
	for _, same := range []bool{true, false} {
		t.Run(fmt.Sprint(same), func(t *testing.T) {
			s := newStore(t)
			ctx := context.Background()
			r, target := recoveryOriginal(t, s, "box_start", rUUID)
			type result struct {
				v RecoveryRecord
				e error
			}
			results := make(chan result, 2)
			var wg sync.WaitGroup
			for i := range 2 {
				req := r
				if !same {
					req.Key = fmt.Sprintf("distinct-recovery-key-%03d", i)
				}
				wg.Go(func() { v, e := s.CommitRecovery(ctx, req, target, recoverProof(target)); results <- result{v, e} })
			}
			wg.Wait()
			close(results)
			var ids []string
			conflicts := 0
			for x := range results {
				if errors.Is(x.e, ErrConflict) {
					conflicts++
					continue
				}
				if x.e != nil {
					t.Fatal(x.e)
				}
				ids = append(ids, x.v.Receipt.ID)
			}
			if same && (len(ids) != 2 || ids[0] != ids[1] || conflicts != 0) {
				t.Fatalf("same key %v conflicts=%d", ids, conflicts)
			}
			if !same && (len(ids) != 1 || conflicts != 1) {
				t.Fatalf("distinct keys %v conflicts=%d", ids, conflicts)
			}
		})
	}
}
func TestS5A2RecoveryBindDoesNotUnlockOrInventNativeAck(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	r, target := recoveryOriginal(t, s, "box_create", "")
	p := RecoveryProof{Kind: RecoveryBind, Reason: "CREATE_MARKER_EXACT_IDENTITY", UUID: rUUID, Fingerprint: "test-marker"}
	got, e := s.CommitRecovery(ctx, r, target, p)
	if e != nil || got.Disposition != "IDENTITY_BOUND" || got.Receipt.BoxUUID != rUUID {
		t.Fatalf("binding %+v %v", got, e)
	}
	current, e := s.ReadRecoveryTarget(ctx, r.Layer, target.ID)
	if e != nil || current.State != "UNKNOWN" || current.UUID != rUUID || current.NativeID != "" || current.Released {
		t.Fatalf("unsafe binding %+v %v", current, e)
	}
	_, e = s.Admit(ctx, Admission{Layer: r.Layer, ActorID: r.ActorID, TokenID: r.TokenID, Key: "never-admit-under-original-lock", Action: "box_exec", RequestID: "refusal", ResourceKey: "box:owned", BoxUUID: rUUID, Incarnation: rUUID, SafeIntent: []byte("{}")})
	if !errors.Is(e, ErrResourceLocked) {
		t.Fatalf("identity binding unlocked UNKNOWN: %v", e)
	}
}
func TestS5A2RecoveryExecEvidenceAndDeleteAtomic(t *testing.T) {
	ctx := context.Background()
	for _, action := range []string{"box_exec", "box_delete"} {
		t.Run(action, func(t *testing.T) {
			s := newStore(t)
			r, target := recoveryOriginal(t, s, action, rUUID)
			p := recoverProof(target)
			if action == "box_exec" {
				exit := 127
				p.Exit = &exit
				p.StdoutPath = "/1.0/instances/owned/logs/exec-output/exec_" + rNative + ".stdout"
				p.StderrPath = "/1.0/instances/owned/logs/exec-output/exec_" + rNative + ".stderr"
				bad := p
				bad.StdoutPath = "/1.0/instances/foreign/logs/exec-output/exec_" + rNative + ".stdout"
				if _, e := s.CommitRecovery(ctx, r, target, bad); !errors.Is(e, ErrInvalid) {
					t.Fatalf("foreign path accepted %v", e)
				}
			} else {
				p.Fingerprint = "DELETE_SUCCESS:independent-fixture"
			}
			got, e := s.CommitRecovery(ctx, r, target, p)
			if e != nil || got.Disposition != "SETTLED" {
				t.Fatalf("%+v %v", got, e)
			}
			if action == "box_exec" {
				out, e := s.ExecOutput(ctx, target.ID)
				if e != nil || out.ExitCode == nil || *out.ExitCode != 127 || out.StdoutPath != p.StdoutPath {
					t.Fatalf("lost guest exit %+v %v", out, e)
				}
			} else {
				var phase, by string
				if e = s.db.QueryRow("SELECT state,deleted_by_operation_id FROM owned_boxes WHERE uuid=?", rUUID).Scan(&phase, &by); e != nil || phase != "DELETED" || by != target.ID {
					t.Fatalf("lost original tombstone %s %s %v", phase, by, e)
				}
			}
		})
	}
}
func TestS5A2RecoveryRefusalAndLocalParity(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	r, target := recoveryOriginal(t, s, "box_exec", rUUID)
	p := RecoveryProof{Kind: RecoveryApproval, Reason: "ORIGINAL_NATIVE_RECEIPT_UNAVAILABLE"}
	got, e := s.CommitRecovery(ctx, r, target, p)
	if e != nil || got.Disposition != "APPROVAL_REQUIRED" || got.Receipt.State != "FAILED" {
		t.Fatalf("refusal %+v %v", got, e)
	}
	current, e := s.ReadRecoveryTarget(ctx, r.Layer, target.ID)
	if e != nil || current != target {
		t.Fatalf("refusal modified original %+v %v", current, e)
	}
	local := RecoveryRequest{Layer: r.Layer, LocalReviewer: "local-operator", OriginalID: target.ID, Key: r.Key}
	reviewed, e := s.CommitRecovery(ctx, local, target, p)
	if e != nil || reviewed.Receipt.ID == got.Receipt.ID {
		t.Fatalf("local parity / key isolation %+v %v", reviewed, e)
	}
	if e = s.RevokeToken(ctx, r.TokenID); e != nil {
		t.Fatal(e)
	}
	r.Key = "revoked-recovery-request-new"
	if _, e = s.CommitRecovery(ctx, r, target, p); !errors.Is(e, ErrRevoked) {
		t.Fatalf("revoked before commit: %v", e)
	}
}
