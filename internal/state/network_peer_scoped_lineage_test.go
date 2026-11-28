package state

import (
	"context"
	"errors"
	"testing"
)

// These test mutations target a PREVIOUS already-SUCCEEDED Peer original,
// not the v2 original to be staged. A prior success/status bit alone is
// insufficient without the canonical immutable intent, claimed native
// resources and the original kernel-verified terminal event.
func TestPR178V2ScopedRootRefusesForgedScopedPeerHistory(t *testing.T) {
	cases := []struct{ name, sql string }{
		{"deleted-verified-id", "UPDATE peer_grants SET last_verified_operation_id=NULL WHERE source_uuid='" + s4b2RootThird + "'"},
		{"verified-id-points-to-current-original", "UPDATE peer_grants SET last_verified_operation_id=(SELECT id FROM operations WHERE state='DISPATCHING') WHERE source_uuid='" + s4b2RootThird + "'"},
		{"forged-confirmation-row", "UPDATE peer_grants SET operation_id='cccccccccccccccccccccccccccccccc' WHERE source_uuid='" + s4b2RootThird + "'"},
		{"terminal-unknown", "UPDATE operations SET state='UNKNOWN' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-failed", "UPDATE operations SET state='FAILED' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-wrong-action", "UPDATE operations SET action='box_start' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-wrong-layer", "UPDATE operations SET layer='user-1002' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-wrong-box", "UPDATE operations SET box_uuid='foreign' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-wrong-source-inc", "UPDATE operations SET incarnation='foreign' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-empty-actor", "UPDATE operations SET actor_id='' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-no-token", "UPDATE operations SET token_id='' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-faked-native-claim", "UPDATE operations SET resource_key='acl:default/foreign' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-JSON-drift", "UPDATE operations SET safe_intent='{}' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-missing-box-claim", "DELETE FROM operation_resource_claims WHERE resource_key='box:" + s4b2RootThird + "'"},
		{"terminal-missing-all-claims", "DELETE FROM operation_resource_claims WHERE operation_id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-missing-native-completion", "DELETE FROM operation_events WHERE state='SUCCEEDED' AND operation_id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-faked-native-completion", "UPDATE operation_events SET safe_detail='SPURIOUS_SUCCESS' WHERE state='SUCCEEDED' AND operation_id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-reversed-intent", "UPDATE operations SET safe_intent=replace(safe_intent,'\"allow\":true','\"allow\":false') WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
		{"terminal-foreign-bridge", "UPDATE operations SET safe_intent=replace(safe_intent,'incusbr-1001','foreign-bridge') WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE source_uuid='" + s4b2RootThird + "')"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, r, id := pr178ScopedOriginalFixture(t, true)
			if err := s.Claim(context.Background(), id); err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.Exec(tc.sql); err != nil {
				t.Fatal(err)
			}
			got, err := r.ReadRootScopedPeerOriginal(context.Background(), id, "user-1001", purePeerBridge)
			if err == nil || len(got.After) != 0 {
				t.Fatalf("forged previous successful C->A operation authorized v2 A->B: %+v", got)
			}
		})
	}
}
func TestPR178V2StageRefusesForgedScopedPeerHistory(t *testing.T) {
	cases := []struct{ name, sql string }{
		{"missing-verified-id", "UPDATE peer_grants SET last_verified_operation_id=NULL WHERE source_uuid='" + s4b2RootThird + "'"},
		{"spoof-past-native-event", "UPDATE operation_events SET safe_detail='UNVERIFIED' WHERE state='SUCCEEDED' AND operation_id IN (SELECT last_verified_operation_id FROM peer_grants)"},
		{"orphan-success", "UPDATE operations SET state='UNKNOWN' WHERE id IN (SELECT last_verified_operation_id FROM peer_grants)"},
		{"forged-historical-claim", "DELETE FROM operation_resource_claims WHERE resource_key LIKE 'acl:%' AND operation_id IN (SELECT last_verified_operation_id FROM peer_grants)"},
		{"missing-historical-intent", "UPDATE operations SET safe_intent='{}' WHERE id IN (SELECT last_verified_operation_id FROM peer_grants)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _, id := pr178ScopedReadyToStage(t, true)
			if _, err := s.db.Exec(tc.sql); err != nil {
				t.Fatal(err)
			}
			if err := s.StageScopedPeerOriginal(context.Background(), id); err == nil {
				t.Fatal("v2 pending peer committed with forged successful C->A lineage")
			}
			var n int
			if err := s.db.QueryRow("SELECT COUNT(*) FROM peer_grants WHERE operation_id=? AND status='PENDING'", id).Scan(&n); err != nil || n != 0 {
				t.Fatal("rejected original wrote a pending pair", err, n)
			}
		})
	}
}
func TestPR178V2RegrantRefusesForgedCompletedRevokeLineage(t *testing.T) {
	cases := []struct{ name, sql string }{
		{"revoke-verified-null", "UPDATE peer_grants SET last_verified_operation_id=NULL WHERE status='REVOKED'"},
		{"revoke-original-mismatch", "UPDATE peer_grants SET operation_id='' WHERE status='REVOKED'"},
		{"revoke-original-in-UNKNOWN", "UPDATE operations SET state='UNKNOWN' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE status='REVOKED')"},
		{"revoke-terminal-event-missing", "DELETE FROM operation_events WHERE state='SUCCEEDED' AND operation_id=(SELECT last_verified_operation_id FROM peer_grants WHERE status='REVOKED')"},
		{"revoke-provenance-foreign", "UPDATE operations SET resource_key='acl:default/foreign' WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE status='REVOKED')"},
		{"revoke-forged-allow", "UPDATE operations SET safe_intent=replace(safe_intent,'\"allow\":false','\"allow\":true') WHERE id=(SELECT last_verified_operation_id FROM peer_grants WHERE status='REVOKED')"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _, id := pr178ScopedOriginalFixtureMode(t, true, "ready-regrant")
			if _, err := s.db.Exec(tc.sql); err != nil {
				t.Fatal(err)
			}
			if err := s.StageScopedPeerOriginal(context.Background(), id); err == nil {
				t.Fatal("regrant trusted an unverifiable revoked tombstone")
			}
		})
	}
}
func TestPR178V2OldCredentialRevocationAfterSuccessKeepsHistoricalProof(t *testing.T) {
	s, r, id := pr178ScopedOriginalFixture(t, true)
	ctx := context.Background()
	var oldToken string
	err := s.db.QueryRow(`SELECT o.token_id FROM peer_grants p
 JOIN operations o ON o.id=p.last_verified_operation_id
 WHERE p.source_uuid=? AND p.status='CONFIRMED'`, s4b2RootThird).Scan(&oldToken)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RevokeToken(ctx, oldToken); err != nil {
		t.Fatal(err)
	}
	if err = s.Claim(ctx, id); err != nil {
		t.Fatal(err)
	}
	proof, err := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", purePeerBridge)
	if err != nil || len(proof.Before) != 1 || len(proof.After) != 2 {
		t.Fatal("completed historical root proof was mistaken for a currently active credential", err)
	}
}
func TestPR178V2UnknownHistoryDoesNotPermitNativeRecovery(t *testing.T) {
	s, r, id := pr178ScopedOriginalFixture(t, true)
	ctx := context.Background()
	if err := s.Claim(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("UPDATE peer_grants SET last_verified_operation_id=NULL WHERE status='CONFIRMED'"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", purePeerBridge); !errors.Is(err, ErrResourceLocked) {
		t.Fatal("missing past kernel event was treated as current authority", err)
	}
}
