package state

import (
	"context"
	"database/sql"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPR177D0ExtensionLiveReadTransactionEpoch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	rw, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()
	ro, err := OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	tx, err := ro.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var n int
	query := "SELECT COUNT(*) FROM oauth_attempts WHERE kind='d0x-epoch'"
	if err = tx.QueryRowContext(ctx, query).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	if _, err = rw.db.ExecContext(ctx, "INSERT INTO oauth_attempts(at,kind) VALUES(1,'d0x-epoch')"); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRowContext(ctx, query).Scan(&n); err != nil || n != 0 {
		t.Fatal("single snapshot changed", n, err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = ro.db.QueryRowContext(ctx, query).Scan(&n); err != nil || n != 1 {
		t.Fatal("new snapshot stale", n, err)
	}
	t.Log("D0X_LIVE_SQLITE_TRANSACTION_STABLE_NEW_EPOCH_VISIBLE")
}

func TestPR177D0ExtensionScopedRetirementStatusMismatch(t *testing.T) {
	ctx := context.Background()
	s, b, _ := n3RetiredFixture(t)
	l, err := s.ReadCompleteScopedNetworkAuditLedger(ctx, b.Project, "incusbr-1001")
	if err != nil || len(l.Retired) != 1 {
		t.Fatal(l, err)
	}
	holds, err := s.ReadCreateAddressHolds(ctx, b.Project, l.Bridge)
	if err != nil {
		t.Fatal(err)
	}
	retired := l.Retired[0]
	if !holds.IPs[netip.MustParseAddr(retired.IPv4)] || !holds.IPs[netip.MustParseAddr(retired.IPv6)] || !holds.MACs[retired.MAC] {
		t.Fatal("valid retired address or MAC no longer held")
	}
	// The current maintenance observation contains retained identity history,
	// never an obsolete reclamation queue.
	snap := DiagnosticSnapshot{store: s}
	observed, err := snap.ReadScopedMaintenanceLedger(ctx, b.Project)
	if err != nil || observed.Retired != 1 || len(observed.LegacyPending) != 0 || len(observed.LegacyCompactions) != 0 {
		t.Fatal(observed, err)
	}
}

func TestPR177D0ExtensionOriginalCredentialRevocation(t *testing.T) {
	ctx := context.Background()
	s, ro, id := pr178ScopedReadyToStage(t, true)
	if err := s.StageAndClaimScopedPeerOriginal(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, id, "DISPATCHING", "SCOPED_PEER_V2_ROOT_OUTCOME_UNVERIFIED"); err != nil {
		t.Fatal(err)
	}
	target, err := s.ReadRecoveryTarget(ctx, "user-1001", id)
	if err != nil {
		t.Fatal(err)
	}
	receipt := scopedTestReceipt(t, ro, id, true)
	var oldToken string
	if err = s.db.QueryRow("SELECT token_id FROM operations WHERE id=?", id).Scan(&oldToken); err != nil {
		t.Fatal(err)
	}
	if err = s.RevokeToken(ctx, oldToken); err != nil {
		t.Fatal(err)
	}
	if _, err = ro.ReadRootScopedPeerUnknownOriginal(ctx, id, "user-1001", purePeerBridge); !errors.Is(err, ErrRevoked) {
		t.Fatal("revoked original did not fail closed", err)
	}
	// A fresh authenticated recovery actor alone cannot re-authorize a
	// revoked original or settle using its formerly valid receipt.
	_, tok, err := s.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	req := RecoveryRequest{Layer: target.Layer, ActorID: tok.ID, TokenID: tok.ID, OriginalID: id, Key: "d0x-revoked-original-recovery"}
	p := RecoveryProof{Kind: RecoverySettle, Reason: "SCOPED_PEER_COMPLETED_NO_REPLAY", UUID: target.UUID, ScopedPeer: receipt, Fingerprint: "synthetic-d0x-revoked"}
	if _, err = s.CommitRecovery(ctx, req, target, p); err == nil {
		t.Fatal("new actor overrode original revocation")
	}
	if _, err = s.ReplayRecovery(ctx, req); !errors.Is(err, ErrNotFound) {
		t.Fatal("refused transaction recorded successful recovery", err)
	}
	got, err := s.ReadRecoveryNavigation(ctx, target.Layer, id)
	if err != nil || got.ClaimProtection != "RETAINED" {
		t.Fatal(got, err)
	}
	t.Log("D0X_REVOKED_ORIGINAL_RETAINS_CLAIMS_NEW_ACTOR_NOT_ENOUGH")
}
