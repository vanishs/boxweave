package state

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func s4b3GrantFixture(t *testing.T) (*Store, string) {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	s4b2InsertAuditFixture(t, s)
	_, tok, err := s.IssueToken(ctx,time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	intent := RootPeerRevokeSafeIntent{
		Project: "user-1001", Bridge: "bwbr0", ACL: "bw-s4b2",
		SourceUUID: s4b2RootSrc, SourceIncarnation: s4b2RootSrc,
		TargetUUID: s4b2RootDst, TargetIncarnation: s4b2RootDst, Allow: true,
	}
	raw, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := s.Admit(ctx, Admission{
		Layer: intent.Project, ActorID: tok.ID, TokenID: tok.ID,
		Key: "b3-grant-original-1", RequestID: "b3-grant-original-1",
		Action: "box_peer_set", SafeIntent: raw, BoxUUID: s4b2RootSrc, Incarnation: s4b2RootSrc,
		ResourceKey:  "acl:default/bw-s4b2",
		ResourceKeys: []string{"box:" + s4b2RootSrc, "box:" + s4b2RootDst},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s, rec.ID
}

func TestS4B3DurableGrantStageNoNativeEffect(t *testing.T) {
	s, id := s4b3GrantFixture(t)
	ctx := context.Background()
	if err := s.StagePeerGrant(ctx, id); err != nil {
		t.Fatal(err)
	}
	var desired int
	var status, linked, src4, dst6 string
	err := s.db.QueryRowContext(ctx, `SELECT desired,status,operation_id,source_ipv4,target_ipv6
  FROM peer_grants WHERE project='user-1001'`).Scan(&desired, &status, &linked, &src4, &dst6)
	if err != nil || desired != 1 || status != "PENDING" || linked != id ||
		src4 != "10.243.18.11" || dst6 != "fd43:18::12" {
		t.Fatalf("staged peer grant lost exact identity: %d %s %s %s %s %v", desired, status, linked, src4, dst6, err)
	}
	rec, err := s.Get(ctx, id)
	if err != nil || rec.State != "ADMITTED" {
		t.Fatalf("staging misreported native success: %+v %v", rec, err)
	}
	if err := s.StagePeerGrant(ctx, id); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("one original staged twice: %v", err)
	}
	t.Log("S4_B3_GRANT_STAGED_INERT_SQLITE_PENDING_NO_NATIVE_SUCCESS=PASS")
}

func TestS4B3GrantStageFailsClosedOnActorAndNetworkDrift(t *testing.T) {
	cases := []struct{ name, sql string }{
		{"revoked-credential", "UPDATE credentials SET revoked=1"},
		{"revoked-token", "UPDATE credentials SET revoked=1"},
		{"wrong-source-box", "UPDATE operations SET box_uuid='foreign'"},
		{"wrong-operation-action", "UPDATE operations SET action='box_exec'"},
		{"unknown-original", "UPDATE operations SET state='UNKNOWN'"},
		{"uncanonical-intent", "UPDATE operations SET safe_intent='{}'"},
		{"lost-target-claim", "DELETE FROM operation_resource_claims WHERE resource_key='box:" + s4b2RootDst + "'"},
		{"forged-extra-claim", "INSERT INTO operation_resource_claims(operation_id,layer,resource_key) SELECT id,layer,'box:extra' FROM operations"},
		{"lost-sibling-nic", "DELETE FROM network_bindings WHERE box_uuid='" + s4b2RootThird + "'"},
		{"target-uuid-changed", "UPDATE owned_boxes SET incarnation='replaced' WHERE uuid='" + s4b2RootDst + "'"},
		{"wrong-mac", "UPDATE network_bindings SET mac='01:00:5e:01:01:01' WHERE box_uuid='" + s4b2RootSrc + "'"},
		{"quarantined-ip", "UPDATE network_address_fences SET state='QUARANTINED' WHERE address='10.243.18.12'"},
		{"foreign-lease", "INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv4','10.243.18.90','foreign','f','f','RESERVED',1)"},
		{"existing-confirmed", `INSERT INTO peer_grants(project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status) VALUES('user-1001','` + s4b2RootSrc + `','` + s4b2RootSrc + `','` + s4b2RootDst + `','` + s4b2RootDst + `',1,'CONFIRMED')`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, id := s4b3GrantFixture(t)
			if _, err := s.db.Exec(tc.sql); err != nil {
				t.Fatal(err)
			}
			if err := s.StagePeerGrant(context.Background(), id); err == nil {
				t.Fatal("unauthorized/stale grant became PENDING")
			}
			var n int
			if err := s.db.QueryRow("SELECT COUNT(*) FROM peer_grants WHERE operation_id=? AND status='PENDING'", id).Scan(&n); err != nil || n != 0 {
				t.Fatalf("rejected original mutated grant: count=%d err=%v", n, err)
			}
		})
	}
	t.Log("S4_B3_GRANT_IDENTITY_FENCE_ACTOR_FAIL_CLOSED=PASS")
}

func TestS4B3RegrantOnlyExactRevokedIncarnation(t *testing.T) {
	s, id := s4b3GrantFixture(t)
	_, err := s.db.Exec(`INSERT INTO peer_grants(project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status,
 source_ipv4,target_ipv4,source_ipv6,target_ipv6) VALUES(?,?,?,?,?,0,'REVOKED',?,?,?,?)`,
		"user-1001", s4b2RootSrc, s4b2RootSrc, s4b2RootDst, s4b2RootDst,
		"10.243.18.11", "10.243.18.12", "fd43:18::11", "fd43:18::12")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.StagePeerGrant(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	var status, linked string
	if err := s.db.QueryRow("SELECT status,operation_id FROM peer_grants WHERE project='user-1001'").Scan(&status, &linked); err != nil || status != "PENDING" || linked != id {
		t.Fatalf("exact revoked regrant did not retain new operation: %s %s %v", status, linked, err)
	}
	t.Log("S4_B3_REGRANT_EXACT_REVOKED_UUID_AND_LEASE_ONLY=PASS")
}
