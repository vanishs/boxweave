package state

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

const purePeerProject = "user-1001"
const purePeerBridge = "incusbr-1001"

func purePeerUUID(i int) string {
	return fmt.Sprintf("11111111-1111-4111-8111-%012d", i)
}
func purePeerAdmit(t *testing.T, s *Store, token, key string, src, dst int, allow bool) Receipt {
	t.Helper()
	r, e := s.AdmitScopedPeerSet(context.Background(), Admission{Layer: purePeerProject, ActorID: token, TokenID: token, Key: "pure-scoped-history-" + key, RequestID: key}, purePeerUUID(src), purePeerUUID(dst), allow)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func purePeerReceipt(t *testing.T, s *Store, r Receipt) ScopedPeerRootReceipt {
	t.Helper()
	ctx := context.Background()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	proof, e := readScopedPeerOriginalTx(ctx, tx, r.ID, purePeerProject, purePeerBridge, "DISPATCHING")
	if e != nil {
		t.Fatal(e)
	}
	// Synthetic completion only: the SQLite oracle cannot certify native effects.
	return ScopedPeerRootReceipt{OriginalID: r.ID, Project: purePeerProject, Bridge: purePeerBridge, Status: "COMPLETE_VERIFIED", ProofSHA256: ScopedPeerProofSHA256(proof), PlanSHA256: strings.Repeat("a", 64), FinalSHA256: strings.Repeat("b", 64), VerifiedSteps: 4}
}
func purePeerComplete(t *testing.T, s *Store, r Receipt) {
	t.Helper()
	ctx := context.Background()
	if e := s.StageAndClaimScopedPeerOriginal(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if e := s.FinalizeScopedPeerOriginal(ctx, purePeerReceipt(t, s, r)); e != nil {
		t.Fatal(e)
	}
}
func purePeerReadOnly(t *testing.T, s *Store) *Store {
	t.Helper()
	var seq int
	var name, path string
	if e := s.db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path); e != nil {
		t.Fatal(e)
	}
	r, e := OpenNetworkAuditReadOnly(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}
func purePeerNewToken(t *testing.T, s *Store) string {
	t.Helper()
	_, tok, e := s.IssueToken(context.Background(), time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	return tok.ID
}

func TestP1ScopedPurePeerHistoryLifecycle(t *testing.T) {
	ctx := context.Background()
	s, _, _ := n3ScopedOwnedFixture(t)
	old := purePeerNewToken(t, s)
	sibling := purePeerAdmit(t, s, old, "sibling", 3, 1, true)
	purePeerComplete(t, s, sibling)
	if e := s.RevokeToken(ctx, old); e != nil {
		t.Fatal(e)
	}
	token := purePeerNewToken(t, s)
	r := purePeerReadOnly(t, s)
	for _, tc := range []struct {
		key                                string
		allow, regrant                     bool
		before, after, revBefore, revAfter int
	}{
		{"grant", true, false, 1, 2, 0, 0},
		{"revoke", false, false, 2, 1, 0, 1},
		{"regrant", true, true, 1, 2, 1, 0},
	} {
		t.Run(tc.key, func(t *testing.T) {
			orig := purePeerAdmit(t, s, token, tc.key, 1, 2, tc.allow)
			if _, e := r.ReadRootScopedPeerOriginal(ctx, orig.ID, purePeerProject, purePeerBridge); e == nil {
				t.Fatal("ADMITTED original authorized Root")
			}
			if e := s.StageAndClaimScopedPeerOriginal(ctx, orig.ID); e != nil {
				t.Fatal(e)
			}
			p, e := r.ReadRootScopedPeerOriginal(ctx, orig.ID, purePeerProject, purePeerBridge)
			if e != nil {
				t.Fatal(e)
			}
			if p.OriginalID != orig.ID || p.RecoveryOnly || p.Regrant != tc.regrant || p.Intent.Allow != tc.allow || len(p.Inventory.Boxes) != 3 || len(p.Before) != tc.before || len(p.After) != tc.after || len(p.RevokedBefore) != tc.revBefore || len(p.RevokedAfter) != tc.revAfter {
				t.Fatalf("unexpected proof: %+v", p)
			}
			for _, list := range [][]PeerGrantBinding{p.Before, p.After} {
				found := false
				for _, pair := range list {
					if pair.SourceUUID == purePeerUUID(3) && pair.TargetUUID == purePeerUUID(1) {
						found = true
					}
				}
				if !found {
					t.Fatal("unrelated Scoped sibling direction lost")
				}
			}
			if _, e = r.ReadRootScopedPeerUnknownOriginal(ctx, orig.ID, purePeerProject, purePeerBridge); e == nil {
				t.Fatal("DISPATCHING accepted as UNKNOWN")
			}
			if _, e = r.ReadRootScopedPeerOriginal(ctx, orig.ID, "user-1002", purePeerBridge); e == nil {
				t.Fatal("foreign layer accepted")
			}
			if e = s.FinalizeScopedPeerOriginal(ctx, purePeerReceipt(t, s, orig)); e != nil {
				t.Fatal(e)
			}
			if _, e = r.ReadRootScopedPeerOriginal(ctx, orig.ID, purePeerProject, purePeerBridge); e == nil {
				t.Fatal("completed original redispatch")
			}
		})
	}
	var verified string
	if e := s.db.QueryRow("SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid=? AND target_uuid=?", purePeerUUID(3), purePeerUUID(1)).Scan(&verified); e != nil || verified != sibling.ID {
		t.Fatal("sibling provenance drift", verified, e)
	}
}

func TestP1ScopedPurePeerHistoryRefusesDrift(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ name, sql string }{
		{"missing-event", "DELETE FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'"},
		{"legacy-event", "UPDATE operation_events SET safe_detail='PEER_ROOT_SCOPED_GRANT_KERNEL_VERIFIED' WHERE operation_id=? AND state='SUCCEEDED'"},
		{"missing-claim", "DELETE FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'acl:%'"},
		{"bad-intent", "UPDATE operations SET safe_intent='{}' WHERE id=?"},
		{"wrong-state", "UPDATE operations SET state='UNKNOWN' WHERE id=?"},
		{"missing-link", "UPDATE peer_grants SET last_verified_operation_id=NULL WHERE last_verified_operation_id=?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := n3ScopedOwnedFixture(t)
			token := purePeerNewToken(t, s)
			sibling := purePeerAdmit(t, s, token, "sibling", 3, 1, true)
			purePeerComplete(t, s, sibling)
			orig := purePeerAdmit(t, s, token, "grant", 1, 2, true)
			if _, e := s.db.Exec(tc.sql, sibling.ID); e != nil {
				t.Fatal(e)
			}
			if e := s.StageAndClaimScopedPeerOriginal(ctx, orig.ID); !errors.Is(e, ErrResourceLocked) {
				t.Fatal("forged history allowed or wrong refusal", e)
			}
			var n int
			if e := s.db.QueryRow("SELECT COUNT(*) FROM peer_grants WHERE operation_id=? AND status='PENDING'", orig.ID).Scan(&n); e != nil || n != 0 {
				t.Fatal("partial pending state", n, e)
			}
		})
	}
}

func TestP1ScopedPurePeerHistoryUnknownRecovery(t *testing.T) {
	ctx := context.Background()
	s, _, _ := n3ScopedOwnedFixture(t)
	token := purePeerNewToken(t, s)
	sibling := purePeerAdmit(t, s, token, "sibling", 3, 1, true)
	purePeerComplete(t, s, sibling)
	purePeerComplete(t, s, purePeerAdmit(t, s, token, "grant", 1, 2, true))
	purePeerComplete(t, s, purePeerAdmit(t, s, token, "revoke", 1, 2, false))
	orig := purePeerAdmit(t, s, token, "regrant", 1, 2, true)
	if e := s.StageAndClaimScopedPeerOriginal(ctx, orig.ID); e != nil {
		t.Fatal(e)
	}
	receipt := purePeerReceipt(t, s, orig)
	if e := s.Unknown(ctx, orig.ID, "DISPATCHING", "TEST_LOST_ACK"); e != nil {
		t.Fatal(e)
	}
	reader := purePeerReadOnly(t, s)
	if _, e := reader.ReadRootScopedPeerOriginal(ctx, orig.ID, purePeerProject, purePeerBridge); e == nil {
		t.Fatal("UNKNOWN authorized native redispatch")
	}
	proof, e := reader.ReadRootScopedPeerUnknownOriginal(ctx, orig.ID, purePeerProject, purePeerBridge)
	if e != nil || !proof.RecoveryOnly || !proof.Regrant || len(proof.RevokedBefore) != 1 || len(proof.RevokedAfter) != 0 || ScopedPeerProofSHA256(proof) != receipt.ProofSHA256 {
		t.Fatal("regrant UNKNOWN proof drift", e)
	}
	req, target := a4Request(t, s, orig.ID)
	receipt.RecoveryOnly = true
	p := RecoveryProof{Kind: RecoverySettle, Reason: "SCOPED_PEER_COMPLETED_NO_REPLAY", Fingerprint: "synthetic-pure-history", ScopedPeer: receipt}
	bad := p
	bad.ScopedPeer.ProofSHA256 = strings.Repeat("0", 64)
	if _, e = s.CommitScopedRecovery(ctx, req, target, bad); e == nil {
		t.Fatal("forged history receipt accepted")
	}
	a4SettleMode(t, s, req, target, p, true)
	policy, e := s.ReadScopedTerminalPeerPolicy(ctx, purePeerProject, purePeerBridge)
	if e != nil || len(policy.Active) != 2 || len(policy.Revoked) != 0 {
		t.Fatal(policy, e)
	}
	var link string
	if e = s.db.QueryRow("SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid=? AND target_uuid=?", purePeerUUID(3), purePeerUUID(1)).Scan(&link); e != nil || link != sibling.ID {
		t.Fatal("sibling original changed", link, e)
	}
}
