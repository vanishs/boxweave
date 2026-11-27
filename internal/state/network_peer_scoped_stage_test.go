package state

import (
	"context"
	"errors"
	"testing"
)

func pr178ScopedReadyToStage(t *testing.T, allow bool) (*Store, *Store, string) {
	t.Helper()
	if allow {
		return pr178ScopedOriginalFixtureMode(t, true, "ready-grant")
	}
	return pr178ScopedOriginalFixtureMode(t, false, "ready-revoke")
}
func TestPR178V2StageScopedPeerGrantAndRevokeAtomic(t *testing.T) {
	for _, allow := range []bool{true, false} {
		t.Run(map[bool]string{true: "grant", false: "revoke"}[allow], func(t *testing.T) {
			s, r, id := pr178ScopedReadyToStage(t, allow)
			ctx := context.Background()
			if e := s.StageScopedPeerOriginal(ctx, id); e != nil {
				t.Fatal("v2 stage rejected valid complete native ledger:", e)
			}
			if e := s.StageScopedPeerOriginal(ctx, id); e == nil {
				t.Fatal("duplicate native original stage committed twice")
			}
			var desired int
			var state, original string
			e := s.db.QueryRow("SELECT desired,status,operation_id FROM peer_grants WHERE project='user-1001' AND source_uuid=? AND target_uuid=?", s4b2RootSrc, s4b2RootDst).
				Scan(&desired, &state, &original)
			if e != nil || state != "PENDING" || original != id ||
				allow && desired != 1 || !allow && desired != 0 {
				t.Fatalf("v2 desired transition not atomic: desired=%d status=%s original=%s e=%v", desired, state, original, e)
			}
			var n int
			if e = s.db.QueryRow("SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND safe_detail='SCOPED_PEER_V2_STAGED_ONLY_NO_NATIVE_EFFECT'", id).Scan(&n); e != nil || n != 1 {
				t.Fatal("inert scoped original staging not audited", e, n)
			}
			if e = s.Claim(ctx, id); e != nil {
				t.Fatal(e)
			}
			proof, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", purePeerBridge)
			if e != nil || proof.RecoveryOnly || len(proof.Inventory.Boxes) != 3 {
				t.Fatal("native v2 staging failed Root SQLite original proof", e)
			}
			if allow && (len(proof.Before) != 1 || len(proof.After) != 2) ||
				!allow && (len(proof.Before) != 2 || len(proof.After) != 1) {
				t.Fatal("scoped staged peer mutated unrelated C->A direction")
			}
			if _, e = r.ReadRootPeerGrant(ctx, id, "user-1001", purePeerBridge, "bw-s4b2"); e == nil {
				t.Fatal("legacy Root B3 accepted v2 scoped operation")
			}
		})
	}
}
func TestPR178V2StageScopedPeerRegrantOnlyExactTombstone(t *testing.T) {
	s, r, id := pr178ScopedOriginalFixtureMode(t, true, "ready-regrant")
	ctx := context.Background()
	if e := s.StageScopedPeerOriginal(ctx, id); e != nil {
		t.Fatal("v2 regrant of same pinned revoked Box incarnation rejected", e)
	}
	var marker int
	if e := s.db.QueryRow("SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND safe_detail='SCOPED_PEER_V2_REGRANT_FROM_REVOKED'", id).Scan(&marker); e != nil || marker != 1 {
		t.Fatal("revoked direction missing durable native DROP-before marker", e)
	}
	if e := s.Claim(ctx, id); e != nil {
		t.Fatal(e)
	}
	proof, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", purePeerBridge)
	if e != nil || len(proof.After) != 2 || !proof.Regrant ||
		len(proof.RevokedBefore) != 1 || len(proof.RevokedAfter) != 0 {
		t.Fatal("regrant lost prior DROP tombstone or kept it after explicit grant", e)
	}
}
func TestPR178V2StageScopedPeerRejectsUnsafeLedgerAndOriginal(t *testing.T) {
	cases := []struct{ name, sql string }{
		{"partial-tenant-ACL", "UPDATE network_bindings SET acl='bw-s4b2' WHERE box_uuid='" + s4b2RootThird + "'"},
		{"quarantined-ipv6", "UPDATE network_address_fences SET state='QUARANTINED' WHERE address='fd43:18::13'"},
		{"lost-A-peer-lease", "DELETE FROM network_address_fences WHERE address='10.43.18.11'"},
		{"old-v1-original", "UPDATE operations SET safe_intent='{}' WHERE action='box_peer_set' AND state!='SUCCEEDED'"},
		{"wrong-guard-lock", "UPDATE operations SET resource_key='acl:default/bw-s4b2' WHERE action='box_peer_set' AND state!='SUCCEEDED'"},
		{"missing-scoped-ACL-claim", "DELETE FROM operation_resource_claims WHERE resource_key LIKE 'acl:%' AND operation_id IN (SELECT id FROM operations WHERE action='box_peer_set' AND state!='SUCCEEDED')"},
		{"missing-target-Box-claim", "DELETE FROM operation_resource_claims WHERE resource_key='box:" + s4b2RootDst + "' AND operation_id IN (SELECT id FROM operations WHERE action='box_peer_set' AND state!='SUCCEEDED')"},
		{"foreign-claim", `INSERT INTO operation_resource_claims(operation_id,layer,resource_key)
     SELECT id,layer,'box:foreign' FROM operations WHERE action='box_peer_set' AND state!='SUCCEEDED'`},
		{"revoked-token", "UPDATE credentials SET revoked=1 WHERE id IN (SELECT token_id FROM operations WHERE action='box_peer_set' AND state!='SUCCEEDED')"},
		{"wrong-actor", "UPDATE operations SET actor_id='foreign' WHERE action='box_peer_set' AND state!='SUCCEEDED'"},
		{"pending-sibling-peer", `INSERT INTO peer_grants
    (project,source_uuid,source_incarnation,target_uuid,target_incarnation,
     desired,status,source_ipv4,target_ipv4,source_ipv6,target_ipv6,operation_id)
    VALUES('user-1001','` + s4b2RootThird + `','` + s4b2RootThird + `','` + s4b2RootDst + `','` + s4b2RootDst + `',
     1,'PENDING','10.43.18.13','10.43.18.12','fd43:18::13','fd43:18::12','other')`},
		{"malformed-confirmed-peer", "UPDATE peer_grants SET source_ipv4='10.43.18.88' WHERE status='CONFIRMED'"},
		{"reused-confirmed-pair", `INSERT INTO peer_grants
    (project,source_uuid,source_incarnation,target_uuid,target_incarnation,
     desired,status,source_ipv4,target_ipv4,source_ipv6,target_ipv6,operation_id)
    VALUES('user-1001','` + s4b2RootSrc + `','` + s4b2RootSrc + `','` + s4b2RootDst + `','` + s4b2RootDst + `',
     1,'CONFIRMED','10.43.18.11','10.43.18.12','fd43:18::11','fd43:18::12','')`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _, id := pr178ScopedReadyToStage(t, true)
			res, e := s.db.Exec(tc.sql)
			if e != nil {
				t.Fatal(e)
			}
			if n, e := res.RowsAffected(); e != nil || n == 0 {
				t.Fatal("tamper did not change target", n, e)
			}
			if e := s.StageScopedPeerOriginal(context.Background(), id); e == nil {
				t.Fatal("unsafe v2 staging modified network grants")
			}
			var n int
			if e := s.db.QueryRow("SELECT COUNT(*) FROM peer_grants WHERE source_uuid=? AND target_uuid=? AND status='PENDING' AND operation_id=?", s4b2RootSrc, s4b2RootDst, id).Scan(&n); e != nil || n != 0 {
				t.Fatal("native A->B pending row committed despite rejected original", e, n)
			}
		})
	}
}
func TestPR178V2StageScopedPeerCancelledAndRevokeMissingRejected(t *testing.T) {
	s, _, id := pr178ScopedReadyToStage(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := s.StageScopedPeerOriginal(ctx, id); !errors.Is(e, context.Canceled) {
		t.Fatal("cancelled operation wrote v2 pending state", e)
	}
	s2, _, revokeID := pr178ScopedOriginalFixture(t, false)
	if _, e := s2.db.Exec("DELETE FROM peer_grants WHERE status='PENDING'"); e != nil {
		t.Fatal(e)
	}
	if e := s2.StageScopedPeerOriginal(context.Background(), revokeID); e == nil {
		t.Fatal("revoke without a CONFIRMED peer silently passed")
	}
}
