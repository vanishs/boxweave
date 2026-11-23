package state

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"
)

// Construct only a DB-level v2 original, not an installed MCP or Root
// dispatcher. The old controller never issues this intent schema.
func pr178ScopedOriginalFixture(t *testing.T, allow bool) (*Store, *Store, string) {
	t.Helper()
	ctx := context.Background()
	s := pr178ScopedStoreFixture(t)
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	src, dst, third := s4b2RootSrc, s4b2RootDst, s4b2RootThird
	intent := ScopedPeerOriginalIntent{Schema: 2, Project: "user-1001", Bridge: "bwbr0",
		SourceUUID: src, SourceIncarnation: src, TargetUUID: dst, TargetIncarnation: dst, Allow: allow}
	raw, e := json.Marshal(intent)
	if e != nil {
		t.Fatal(e)
	}
	claims, e := scopedPeerResourceClaims(intent)
	if e != nil {
		t.Fatal(e)
	}
	receipt, e := s.Admit(ctx, Admission{
		Layer: intent.Project, ActorID: tok.ID, TokenID: tok.ID,
		Key:       "pr178-scoped-peer-original-unique-12345",
		RequestID: "pr178-scoped-peer-test", Action: "box_peer_set", SafeIntent: raw,
		BoxUUID: src, Incarnation: src, ResourceKey: "peer-guard:" + intent.Bridge,
		ResourceKeys: []string{claims[0], claims[1], claims[2], claims[3]},
	})
	if e != nil {
		t.Fatal(e)
	}
	// A confirmed C->A Peer must stay *byte-for-byte* unchanged by A->B,
	// regardless of the grant/revoke direction of this original.
	_, e = s.db.Exec(`INSERT INTO peer_grants
 (project,source_uuid,source_incarnation,target_uuid,target_incarnation,
 desired,status,source_ipv4,target_ipv4,source_ipv6,target_ipv6,operation_id)
 VALUES('user-1001',?,?,?, ?,1,'CONFIRMED','10.243.18.13','10.243.18.11',
 'fd43:18::13','fd43:18::11','')`, third, third, src, src)
	if e != nil {
		t.Fatal(e)
	}
	desired := 0
	if allow {
		desired = 1
	}
	_, e = s.db.Exec(`INSERT INTO peer_grants
 (project,source_uuid,source_incarnation,target_uuid,target_incarnation,
 desired,status,source_ipv4,target_ipv4,source_ipv6,target_ipv6,operation_id)
 VALUES('user-1001',?,?,?, ?,?,'PENDING','10.243.18.11','10.243.18.12',
 'fd43:18::11','fd43:18::12',?)`, src, src, dst, dst, desired, receipt.ID)
	if e != nil {
		t.Fatal(e)
	}
	var seq int
	var name, path string
	if e = s.db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path); e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(path, 0600); e != nil {
		t.Fatal(e)
	}
	reader, e := OpenNetworkAuditReadOnly(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = reader.Close() })
	return s, reader, receipt.ID
}
func TestPR178ScopedOriginalReadOnlyDirectionGrantAndRevoke(t *testing.T) {
	for _, allow := range []bool{true, false} {
		t.Run(map[bool]string{true: "grant", false: "revoke"}[allow], func(t *testing.T) {
			s, r, id := pr178ScopedOriginalFixture(t, allow)
			ctx := context.Background()
			if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", "bwbr0"); e == nil {
				t.Fatal("unclaimed scoped original gained root permission")
			}
			if e := s.Claim(ctx, id); e != nil {
				t.Fatal(e)
			}
			proof, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", "bwbr0")
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
			if _, e := r.ReadRootPeerGrant(ctx, id, "user-1001", "bwbr0", "bw-s4b2"); e == nil {
				t.Fatal("old shared-ACL Root accepted v2 original")
			}
			if _, e := r.ReadRootPeerRevocation(ctx, id, "user-1001", "bwbr0", "bw-s4b2"); e == nil {
				t.Fatal("old revoke-only Root accepted v2 original")
			}
			if _, e := r.ReadRootScopedPeerUnknownOriginal(ctx, id, "user-1001", "bwbr0"); e == nil {
				t.Fatal("DISPATCHING original accepted as UNKNOWN recovery")
			}
			if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1002", "bwbr0"); e == nil {
				t.Fatal("foreign tenant asked for someone else's grant")
			}
			if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", "other"); e == nil {
				t.Fatal("foreign bridge asked for scoped native proof")
			}
			if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", "bwbr0"); e != nil {
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
	if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", "bwbr0"); e == nil {
		t.Fatal("UNKNOWN original was approved for native write")
	}
	proof, e := r.ReadRootScopedPeerUnknownOriginal(ctx, id, "user-1001", "bwbr0")
	if e != nil || !proof.RecoveryOnly || len(proof.After) != 2 {
		t.Fatal("UNKNOWN should be read-only and retain original Project", e)
	}
}
func TestPR178ScopedOriginalFailsClosedForOriginalAndLedgerDrift(t *testing.T) {
	scenarios := []struct{ name, sql string }{
		{"unknown-native-grant", "UPDATE peer_grants SET status='UNKNOWN' WHERE status='PENDING'"},
		{"wrong-pending-address", "UPDATE peer_grants SET source_ipv4='10.243.18.99' WHERE status='PENDING'"},
		{"extra-missing-pending", "UPDATE peer_grants SET status='REVOKED' WHERE status='PENDING'"},
		{"unlinked-pending", "UPDATE peer_grants SET operation_id='' WHERE status='PENDING'"},
		{"forged-v2-schema", "UPDATE operations SET safe_intent=replace(safe_intent,'\"schema\":2','\"schema\":1')"},
		{"uncanonical-intent", "UPDATE operations SET safe_intent=' { } '"},
		{"wrong-original-action", "UPDATE operations SET action='box_start'"},
		{"wrong-actor", "UPDATE operations SET actor_id=''"},
		{"wrong-layer", "UPDATE operations SET layer='foreign'"},
		{"wrong-source", "UPDATE operations SET box_uuid='foreign'"},
		{"invalid-resource", "UPDATE operations SET resource_key='acl:default/bw-s4b2'"},
		{"missing-guard-claim", "DELETE FROM operation_resource_claims WHERE resource_key LIKE 'peer-guard:%'"},
		{"missing-own-acl", "DELETE FROM operation_resource_claims WHERE resource_key LIKE 'acl:%'"},
		{"missing-target-claim", "DELETE FROM operation_resource_claims WHERE resource_key='box:" + s4b2RootDst + "'"},
		{"extra-foreign-claim", `INSERT INTO operation_resource_claims(operation_id,layer,resource_key)
   SELECT id,layer,'box:foreign' FROM operations`},
		{"revoked-credential", "UPDATE credentials SET revoked=1"},
		{"missing-sibling", "DELETE FROM network_bindings WHERE box_uuid='" + s4b2RootThird + "'"},
		{"partially-migrated-acl", "UPDATE network_bindings SET acl='bw-s4b2' WHERE box_uuid='" + s4b2RootThird + "'"},
		{"foreign-address", "UPDATE network_address_fences SET state='QUARANTINED' WHERE address='fd43:18::11'"},
		{"confirmed-peer-drift", "UPDATE peer_grants SET source_ipv4='10.243.18.98' WHERE status='CONFIRMED'"},
		{"stale-native-incarnation", "UPDATE peer_grants SET target_incarnation='foreign' WHERE status='CONFIRMED'"},
		{"unknown-sibling-peer", "UPDATE peer_grants SET status='UNKNOWN' WHERE status='CONFIRMED'"},
	}
	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			s, r, id := pr178ScopedOriginalFixture(t, true)
			if e := s.Claim(context.Background(), id); e != nil {
				t.Fatal(e)
			}
			if _, e := s.db.Exec(tc.sql); e != nil {
				t.Fatal(e)
			}
			got, e := r.ReadRootScopedPeerOriginal(context.Background(), id, "user-1001", "bwbr0")
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
	if _, e := r.ReadRootScopedPeerOriginal(ctx2, id, "user-1001", "bwbr0"); !errors.Is(e, context.Canceled) {
		t.Fatal("cancelled ledger proof accepted", e)
	}
	if _, e := s.db.Exec(`INSERT INTO operations
 (id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,
 intent_digest,safe_intent,resource_key,box_uuid,incarnation,state,created_at,updated_at)
 SELECT 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',tracking_id,request_id,layer,actor_id,token_id,
 'unique-other-key',action,intent_digest,safe_intent,'peer-guard:bwbr0',
 box_uuid,incarnation,'UNKNOWN',created_at,updated_at FROM operations WHERE id=?`, id); e != nil {
		t.Fatal(e)
	}
	if _, e := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", "bwbr0"); e == nil {
		t.Fatal("unknown sibling operation was ignored")
	}
}
