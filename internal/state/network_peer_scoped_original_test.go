package state

import (
	"context"
	"errors"
	"testing"
)

// Use actual Scoped creation, admission and terminal SQLite history.
// Synthetic Root receipts are not a native kernel witness.
func pr178ScopedOriginalFixture(t *testing.T, allow bool) (*Store, *Store, string) {
	return pr178ScopedOriginalFixtureMode(t, allow, "pending")
}
func pr178ScopedOriginalFixtureMode(t *testing.T, allow bool, mode string) (*Store, *Store, string) {
	t.Helper()
	ctx := context.Background()
	s, _, old := n3ScopedOwnedFixture(t)
	purePeerComplete(t, s, purePeerAdmit(t, s, old, "sibling", 3, 1, true))
	if !allow || mode == "ready-revoke" || mode == "ready-regrant" {
		purePeerComplete(t, s, purePeerAdmit(t, s, old, "initial-grant", 1, 2, true))
	}
	if mode == "ready-regrant" {
		purePeerComplete(t, s, purePeerAdmit(t, s, old, "initial-revoke", 1, 2, false))
	}
	token := purePeerNewToken(t, s)
	receipt := purePeerAdmit(t, s, token, "next", 1, 2, allow)
	if mode == "pending" {
		if e := s.StageScopedPeerOriginal(ctx, receipt.ID); e != nil {
			t.Fatal(e)
		}
	}
	return s, purePeerReadOnly(t, s), receipt.ID
}
func TestPR178ScopedOriginalReadOnlyDirectionGrantAndRevoke(t *testing.T) {
	for _, allow := range []bool{true, false} {
		t.Run(map[bool]string{true: "grant", false: "revoke"}[allow], func(t *testing.T) {
			s, r, id := pr178ScopedOriginalFixture(t, allow)
			ctx := context.Background()
			if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", purePeerBridge); e == nil {
				t.Fatal("unclaimed scoped original gained root permission")
			}
			if e := s.Claim(ctx, id); e != nil {
				t.Fatal(e)
			}
			proof, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", purePeerBridge)
			if e != nil {
				t.Fatal("scoped root original not verified", e)
			}
			if proof.RecoveryOnly || proof.Intent.Schema != 2 ||
				proof.Intent.Allow != allow || len(proof.Inventory.Boxes) != 3 ||
				proof.OriginalID != id || len(proof.Revoked) != 0 {
				t.Fatalf("incomplete v2 proof %+v", proof)
			}
			if allow {
				if proof.Regrant || len(proof.RevokedBefore) != 0 || len(proof.RevokedAfter) != 0 {
					t.Fatal("fresh grant manufactured revoked history")
				}
				if len(proof.Before) != 1 || len(proof.After) != 2 ||
					proof.After[1].SourceUUID != s4b2RootThird {
					t.Fatal("grant changed unrelated confirmed C->A", proof.Before, proof.After)
				}
			} else {
				if proof.Regrant || len(proof.RevokedBefore) != 0 || len(proof.RevokedAfter) != 1 ||
					proof.RevokedAfter[0].SourceUUID != s4b2RootSrc {
					t.Fatal("revoke lost target DROP after-image")
				}
				if len(proof.Before) != 2 || len(proof.After) != 1 ||
					proof.After[0].SourceUUID != s4b2RootThird {
					t.Fatal("revoke changed unrelated confirmed C->A", proof.Before, proof.After)
				}
			}
			// The old v1 proof MUST never accept a v2 operation, even with
			// a legitimate already-pinned legacy ACL name on the same bridge.
			if _, e := r.ReadRootPeerGrant(ctx, id, "user-1001", purePeerBridge, "bw-s4b2"); e == nil {
				t.Fatal("old shared-ACL Root accepted v2 original")
			}
			if _, e := r.ReadRootPeerRevocation(ctx, id, "user-1001", purePeerBridge, "bw-s4b2"); e == nil {
				t.Fatal("old revoke-only Root accepted v2 original")
			}
			if _, e := r.ReadRootScopedPeerUnknownOriginal(ctx, id, "user-1001", purePeerBridge); e == nil {
				t.Fatal("DISPATCHING original accepted as UNKNOWN recovery")
			}
			if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1002", purePeerBridge); e == nil {
				t.Fatal("foreign tenant asked for someone else's grant")
			}
			if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", "other"); e == nil {
				t.Fatal("foreign bridge asked for scoped native proof")
			}
			if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", purePeerBridge); e != nil {
				t.Fatal("read-only proof unexpectedly consumed original", e)
			}
		})
	}
}
func TestPR178ScopedOriginalUNKNOWNReadOnlyNeverDispatch(t *testing.T) {
	s, r, id := pr178ScopedOriginalFixture(t, true)
	ctx := context.Background()
	if e := s.Claim(ctx, id); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec("UPDATE operations SET state='UNKNOWN' WHERE id=?", id); e != nil {
		t.Fatal(e)
	}
	if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", purePeerBridge); e == nil {
		t.Fatal("UNKNOWN original was approved for native write")
	}
	proof, e := r.ReadRootScopedPeerUnknownOriginal(ctx, id, "user-1001", purePeerBridge)
	if e != nil || !proof.RecoveryOnly || len(proof.After) != 2 {
		t.Fatal("UNKNOWN should be read-only and retain original Project", e)
	}
}
func TestPR178ScopedOriginalFailsClosedForOriginalAndLedgerDrift(t *testing.T) {
	scenarios := []struct{ name, sql string }{
		{"unknown-native-grant", "UPDATE peer_grants SET status='UNKNOWN' WHERE status='PENDING'"},
		{"wrong-pending-address", "UPDATE peer_grants SET source_ipv4='10.43.18.99' WHERE status='PENDING'"},
		{"extra-missing-pending", "UPDATE peer_grants SET status='REVOKED' WHERE status='PENDING'"},
		{"unlinked-pending", "UPDATE peer_grants SET operation_id='' WHERE status='PENDING'"},
		{"forged-v2-schema", "UPDATE operations SET safe_intent=replace(safe_intent,'\"schema\":2','\"schema\":1') WHERE action='box_peer_set' AND state!='SUCCEEDED'"},
		{"uncanonical-intent", "UPDATE operations SET safe_intent=' { } ' WHERE action='box_peer_set' AND state!='SUCCEEDED'"},
		{"wrong-original-action", "UPDATE operations SET action='box_start' WHERE action='box_peer_set' AND state!='SUCCEEDED'"},
		{"wrong-actor", "UPDATE operations SET actor_id='' WHERE action='box_peer_set' AND state!='SUCCEEDED'"},
		{"wrong-layer", "UPDATE operations SET layer='foreign' WHERE action='box_peer_set' AND state!='SUCCEEDED'"},
		{"wrong-source", "UPDATE operations SET box_uuid='foreign' WHERE action='box_peer_set' AND state!='SUCCEEDED'"},
		{"invalid-resource", "UPDATE operations SET resource_key='acl:default/bw-s4b2' WHERE action='box_peer_set' AND state!='SUCCEEDED'"},
		{"missing-guard-claim", "DELETE FROM operation_resource_claims WHERE resource_key LIKE 'peer-guard:%' AND operation_id IN (SELECT id FROM operations WHERE action='box_peer_set' AND state!='SUCCEEDED')"},
		{"missing-own-acl", "DELETE FROM operation_resource_claims WHERE resource_key LIKE 'acl:%' AND operation_id IN (SELECT id FROM operations WHERE action='box_peer_set' AND state!='SUCCEEDED')"},
		{"missing-target-claim", "DELETE FROM operation_resource_claims WHERE resource_key='box:" + s4b2RootDst + "' AND operation_id IN (SELECT id FROM operations WHERE action='box_peer_set' AND state!='SUCCEEDED')"},
		{"extra-foreign-claim", `INSERT INTO operation_resource_claims(operation_id,layer,resource_key)
   SELECT id,layer,'box:foreign' FROM operations WHERE action='box_peer_set' AND state!='SUCCEEDED'`},
		{"revoked-credential", "UPDATE credentials SET revoked=1 WHERE id IN (SELECT token_id FROM operations WHERE action='box_peer_set' AND state!='SUCCEEDED')"},
		{"missing-sibling", "DELETE FROM network_bindings WHERE box_uuid='" + s4b2RootThird + "'"},
		{"partially-migrated-acl", "UPDATE network_bindings SET acl='bw-s4b2' WHERE box_uuid='" + s4b2RootThird + "'"},
		{"foreign-address", "UPDATE network_address_fences SET state='QUARANTINED' WHERE address='fd43:18::11'"},
		{"confirmed-peer-drift", "UPDATE peer_grants SET source_ipv4='10.43.18.98' WHERE status='CONFIRMED'"},
		{"stale-native-incarnation", "UPDATE peer_grants SET target_incarnation='foreign' WHERE status='CONFIRMED'"},
		{"unknown-sibling-peer", "UPDATE peer_grants SET status='UNKNOWN' WHERE status='CONFIRMED'"},
	}
	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			s, r, id := pr178ScopedOriginalFixture(t, true)
			if e := s.Claim(context.Background(), id); e != nil {
				t.Fatal(e)
			}
			res, e := s.db.Exec(tc.sql)
			if e != nil {
				t.Fatal(e)
			}
			if n, e := res.RowsAffected(); e != nil || n == 0 {
				t.Fatal("tamper did not change target", n, e)
			}
			got, e := r.ReadRootScopedPeerOriginal(context.Background(), id, "user-1001", purePeerBridge)
			if e == nil || len(got.After) != 0 {
				t.Fatalf("drift gave forged scoped Root proof: %+v", got)
			}
		})
	}
}
func TestPR178ScopedOriginalOtherActivePeerAndCancelRefused(t *testing.T) {
	s, r, id := pr178ScopedOriginalFixture(t, true)
	ctx := context.Background()
	if e := s.Claim(ctx, id); e != nil {
		t.Fatal(e)
	}
	ctx2, cancel := context.WithCancel(ctx)
	cancel()
	if _, e := r.ReadRootScopedPeerOriginal(ctx2, id, "user-1001", purePeerBridge); !errors.Is(e, context.Canceled) {
		t.Fatal("cancelled ledger proof accepted", e)
	}
	if _, e := s.db.Exec(`INSERT INTO operations
 (id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,
 intent_digest,safe_intent,resource_key,box_uuid,incarnation,state,created_at,updated_at)
 SELECT 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',tracking_id,request_id,layer,actor_id,token_id,
 'unique-other-key',action,intent_digest,safe_intent,'peer-guard:incusbr-1001',
 box_uuid,incarnation,'UNKNOWN',created_at,updated_at FROM operations WHERE id=?`, id); e != nil {
		t.Fatal(e)
	}
	if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", purePeerBridge); e == nil {
		t.Fatal("unknown sibling operation was ignored")
	}
}
