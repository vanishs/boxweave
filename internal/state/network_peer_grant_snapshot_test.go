package state

import (
	"context"
	"errors"
	"testing"
)

func TestS4B3ReadExactPeerCompilerSnapshot(t *testing.T) {

	ctx := context.Background()
	s, r, id := pr178ScopedOriginalFixture(t, true)
	if _, e := r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge); e == nil {
		t.Fatal("unclaimed proof")
	}
	if e := s.Claim(ctx, id); e != nil {
		t.Fatal(e)
	}
	p, e := r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge)
	if e != nil || len(p.Before) != 1 || len(p.After) != 2 || p.After[0].SourceUUID != purePeerUUID(1) || p.After[0].TargetUUID != purePeerUUID(2) || p.After[0].TargetIPv6 != "fd43:18::12" || p.After[1] != p.Before[0] {
		t.Fatal(p, e)
	}
	if _, e = s.ReadPeerGrantSnapshot(ctx, purePeerProject, purePeerBridge, "bw-s4b2", id); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("shared compiler admitted", e)
	}
	t.Log("S4_B3_WHOLE_CONFIRMED_PLUS_ONE_PENDING_DIRECTIONAL_COMPILER_SNAPSHOT=PASS")

}

func TestS4B3ReadPeerCompilerSnapshotFailsClosed(t *testing.T) {

	ctx := context.Background()
	for _, sql := range []string{"UPDATE operations SET state='UNKNOWN' WHERE id=?", "UPDATE operations SET safe_intent='{}' WHERE id=?", "UPDATE credentials SET expires_at=0 WHERE id=(SELECT token_id FROM operations WHERE id=?)", "UPDATE peer_grants SET status='CONFIRMED' WHERE operation_id=?", "UPDATE peer_grants SET target_incarnation='reused' WHERE operation_id=?", "DELETE FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'acl:%'"} {
		t.Run(sql, func(t *testing.T) {
			s, r, id := pr178ScopedOriginalFixture(t, true)
			if e := s.Claim(ctx, id); e != nil {
				t.Fatal(e)
			}
			if _, e := r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge); e != nil {
				t.Fatal(e)
			}
			res, e := s.db.Exec(sql, id)
			if e != nil {
				t.Fatal(e)
			}
			if n, e := res.RowsAffected(); e != nil || n == 0 {
				t.Fatal(n, e)
			}
			if p, e := r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge); e == nil || len(p.After) != 0 {
				t.Fatal(p, e)
			}
		})
	}
	t.Log("S4_B3_COMPILER_SNAPSHOT_UNKNOWN_FOREIGN_IDENTITY_FAIL_CLOSED=PASS")

}

// Historical DROP rules are part of the next canonical whole-ACL projection,
// but must never survive a stale native UUID/address mapping unchecked.
func TestS4B3ReadHistoricalRevokedSnapshotRequiresCurrentLease(t *testing.T) {

	ctx := context.Background()
	s, b, token := n3ScopedOwnedFixture(t)
	purePeerComplete(t, s, purePeerAdmit(t, s, token, "snapshot-grant", 1, 2, true))
	purePeerComplete(t, s, purePeerAdmit(t, s, token, "snapshot-revoke", 1, 2, false))
	n3DeleteOwned(t, s, b, token)
	r := purePeerReadOnly(t, s)
	p, e := r.ReadScopedTerminalPeerPolicy(ctx, purePeerProject, purePeerBridge)
	if e != nil || len(p.Revoked) != 1 || p.Revoked[0].SourceUUID != b.UUID {
		t.Fatal(p, e)
	}
	if _, e = s.db.Exec("UPDATE peer_grants SET target_ipv6='fd43:18::99' WHERE status='REVOKED'"); e != nil {
		t.Fatal(e)
	}
	if _, e = r.ReadScopedTerminalPeerPolicy(ctx, purePeerProject, purePeerBridge); e == nil {
		t.Fatal("stale DROP")
	}

}
