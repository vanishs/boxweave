package state

import (
	"context"
	"testing"
)

func s4b2PendingExistingGrantToConfirmed(t *testing.T, s *Store) {
	t.Helper()
	if _, err := s.db.Exec(`UPDATE peer_grants
		SET desired=1,status='CONFIRMED',operation_id=''`); err != nil {
		t.Fatal(err)
	}
}

func TestS4B2ControllerStagesOriginalPeerRevocation(t *testing.T) {

	ctx := context.Background()
	s, r, id := pr178ScopedReadyToStage(t, false)
	if e := s.StageScopedPeerOriginal(ctx, id); e != nil {
		t.Fatal(e)
	}
	var desired int
	var status, linked string
	e := s.db.QueryRow("SELECT desired,status,operation_id FROM peer_grants WHERE source_uuid=? AND target_uuid=?", purePeerUUID(1), purePeerUUID(2)).Scan(&desired, &status, &linked)
	if e != nil || desired != 0 || status != "PENDING" || linked != id {
		t.Fatal(desired, status, linked, e)
	}
	if e = s.Claim(ctx, id); e != nil {
		t.Fatal(e)
	}
	p, e := r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge)
	if e != nil || p.OriginalID != id || p.Intent.Allow || len(p.RevokedAfter) != 1 {
		t.Fatal(p, e)
	}
	if e = s.StageScopedPeerOriginal(ctx, id); e == nil {
		t.Fatal("staged twice")
	}
	t.Log("S4_B2_CONTROLLER_DURABLE_REVOCATION_STAGE_TO_ROOT_PROOF=PASS")

}

func TestS4B2ControllerCannotStageInvalidPeerGrant(t *testing.T) {
	cases := []struct{ name, sql string }{
		{"unconfirmed-existing", "UPDATE peer_grants SET status='PENDING'"},
		{"already-revoked", "UPDATE peer_grants SET status='REVOKED',desired=0"},
		{"wrong-project", "UPDATE peer_grants SET project='foreign'"},
		{"wrong-source-incarnation", "UPDATE peer_grants SET source_incarnation='reincarnated'"},
		{"wrong-native-source-ip", "UPDATE peer_grants SET source_ipv6='fd43:18::99'"},
		{"revoked-credential", "UPDATE credentials SET revoked=1"},
		{"token-revoked", "UPDATE credentials SET revoked=1"},
		{"missing-ACL-claim", "DELETE FROM operation_resource_claims WHERE resource_key LIKE 'acl:%'"},
		{"additional-sibling-claim", `INSERT INTO operation_resource_claims(operation_id,layer,resource_key)
			SELECT id,layer,'box:foreign' FROM operations`},
		{"stale-operation", "UPDATE operations SET state='UNKNOWN'"},
		{"wrong-action", "UPDATE operations SET action='box_start'"},
		{"noncanonical-safe-intent", "UPDATE operations SET safe_intent='{}'"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, _, id := s4b2RootOperationFixture(t)
			s4b2PendingExistingGrantToConfirmed(t, s)
			if _, err := s.db.ExecContext(ctx, tc.sql); err != nil {
				t.Fatal(err)
			}
			if err := s.StageRootPeerRevocation(ctx, id); err == nil {
				t.Fatal("unauthorized peer grant mutated")
			}
			var count int
			if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM peer_grants
				WHERE operation_id=? AND desired=0 AND status='PENDING'`, id).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("invalid revoke staged a root mutation")
			}
		})
	}
	t.Log("S4_B2_CONTROLLER_STAGE_CROSS_ACTOR_AND_UNKNOWN_REFUSED=PASS")
}
