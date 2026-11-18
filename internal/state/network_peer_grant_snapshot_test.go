package state

import (
	"context"
	"errors"
	"testing"
)

func TestS4B3ReadExactPeerCompilerSnapshot(t *testing.T) {
	s, id := s4b3GrantFixture(t)
	ctx := context.Background()
	if err := s.StagePeerGrant(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadPeerGrantSnapshot(ctx, "user-1001", "bwbr0", "bw-s4b2", ""); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("unconfirmed live grant visible as confirmed: %v", err)
	}
	got, err := s.ReadPeerGrantSnapshot(ctx, "user-1001", "bwbr0", "bw-s4b2", id)
	if err != nil || len(got) != 1 || got[0].SourceUUID != s4b2RootSrc ||
		got[0].TargetUUID != s4b2RootDst || got[0].TargetIPv6 != "fd43:18::12" {
		t.Fatalf("bad exact directional candidate %+v: %v", got, err)
	}
	// Existing separately confirmed B -> C is retained in the native compiler
	// snapshot; one new grant must never overwrite an unrelated live grant.
	_, err = s.db.ExecContext(ctx, `INSERT INTO peer_grants
  (project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status,
  source_ipv4,target_ipv4,source_ipv6,target_ipv6)
  VALUES(?,?,?,?,?,1,'CONFIRMED',?,?,?,?)`,
		"user-1001", s4b2RootDst, s4b2RootDst, s4b2RootThird, s4b2RootThird,
		"10.243.18.12", "10.243.18.13", "fd43:18::12", "fd43:18::13")
	if err != nil {
		t.Fatal(err)
	}
	got, err = s.ReadPeerGrantSnapshot(ctx, "user-1001", "bwbr0", "bw-s4b2", id)
	if err != nil || len(got) != 2 || got[0].SourceUUID != s4b2RootSrc ||
		got[1].SourceUUID != s4b2RootDst || got[1].TargetUUID != s4b2RootThird {
		t.Fatalf("whole directional allowlist was not preserved: %+v %v", got, err)
	}
	t.Log("S4_B3_WHOLE_CONFIRMED_PLUS_ONE_PENDING_DIRECTIONAL_COMPILER_SNAPSHOT=PASS")
}

func TestS4B3ReadPeerCompilerSnapshotFailsClosed(t *testing.T) {
	cases := []struct{ name, sql string }{
		{"lost-original", "UPDATE operations SET state='UNKNOWN'"},
		{"wrong-role", "UPDATE credentials SET role='viewer'"},
		{"expired-token", "UPDATE credentials SET revoked=1"},
		{"bad-address", "UPDATE peer_grants SET source_ipv4='10.243.18.99'"},
		{"wrong-incarnation", "UPDATE peer_grants SET target_incarnation='reused'"},
		{"unverified-foreign-sibling", "DELETE FROM network_bindings WHERE box_uuid='" + s4b2RootThird + "'"},
		{"release-address", "UPDATE network_address_fences SET state='RELEASED' WHERE address='fd43:18::12'"},
		{"fake-confirmation", "UPDATE peer_grants SET status='CONFIRMED' WHERE status='PENDING'"},
		{"unknown-grant", "UPDATE peer_grants SET status='UNKNOWN'"},
		{"wrong-original", "UPDATE peer_grants SET operation_id='other'"},
		{"lost-staged-target-claim", "DELETE FROM operation_resource_claims WHERE resource_key='box:" + s4b2RootDst + "'"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, id := s4b3GrantFixture(t)
			if err := s.StagePeerGrant(context.Background(), id); err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.Exec(tc.sql); err != nil {
				t.Fatal(err)
			}
			if got, err := s.ReadPeerGrantSnapshot(context.Background(), "user-1001", "bwbr0", "bw-s4b2", id); err == nil {
				t.Fatalf("unsafe compiler would produce grant %+v", got)
			}
		})
	}
	t.Log("S4_B3_COMPILER_SNAPSHOT_UNKNOWN_FOREIGN_IDENTITY_FAIL_CLOSED=PASS")
}
