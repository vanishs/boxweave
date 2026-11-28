package state

import (
	"context"
	"testing"
)

func s4b3RootAddOtherConfirmed(t *testing.T, s *Store) {
	t.Helper()
	_, err := s.db.Exec("INSERT INTO peer_grants "+
		"(project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status,"+
		"source_ipv4,target_ipv4,source_ipv6,target_ipv6) VALUES(?,?,?,?,?,1,'CONFIRMED',?,?,?,?)",
		"user-1001", s4b2RootDst, s4b2RootDst, s4b2RootThird, s4b2RootThird,
		"10.243.18.12", "10.243.18.13", "fd43:18::12", "fd43:18::13")
	if err != nil {
		t.Fatal(err)
	}
}

func TestS4B3RevokeOneDirectionPreservesOtherConfirmed(t *testing.T) {

	ctx := context.Background()
	s, r, id := pr178ScopedReadyToStage(t, false)
	if e := s.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
		t.Fatal(e)
	}
	p, e := r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge)
	if e != nil || len(p.Before) != 2 || len(p.After) != 1 || len(p.RevokedAfter) != 1 || p.After[0].SourceUUID != purePeerUUID(3) || p.After[0].TargetUUID != purePeerUUID(1) || p.RevokedAfter[0].SourceUUID != purePeerUUID(1) {
		t.Fatal(p, e)
	}
	t.Log("S4_B3_REVOKE_ONE_DIRECTION_OTHER_CONFIRMED_STAYS=PASS")

}

func TestS4B3RevokeSnapshotRejectsUnknownOrStaleSibling(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"unknown", "UPDATE peer_grants SET status='UNKNOWN' WHERE source_uuid='" + s4b2RootDst + "'"},
		{"stale-ip", "UPDATE peer_grants SET target_ipv4='10.243.18.99' WHERE source_uuid='" + s4b2RootDst + "'"},
		{"lost-claim", "DELETE FROM operation_resource_claims WHERE resource_key='acl:default/bw-s4b2'"},
		{"unknown-original", "UPDATE operations SET state='UNKNOWN' WHERE action='box_peer_set'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, reader, original := s4b2RootOperationFixture(t)
			s4b2PendingExistingGrantToConfirmed(t, s)
			s4b3RootAddOtherConfirmed(t, s)
			if err := s.StageRootPeerRevocation(ctx, original); err != nil {
				t.Fatal(err)
			}
			if err := s.Claim(ctx, original); err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.ExecContext(ctx, tc.sql); err != nil {
				t.Fatal(err)
			}
			if before, after, err := reader.ReadPeerRevocationSnapshot(ctx, "user-1001", "bwbr0", "bw-s4b2", original); err == nil {
				t.Fatalf("unsafe snapshot accepted: %+v %+v", before, after)
			}
		})
	}
}
