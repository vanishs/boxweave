package state

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

const s4b2RootSrc = "11111111-1111-4111-8111-000000000001"
const s4b2RootDst = "11111111-1111-4111-8111-000000000002"
const s4b2RootThird = "11111111-1111-4111-8111-000000000003"

func s4b2RootOperationFixture(t *testing.T) (*Store, *Store, string) {
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
		TargetUUID: s4b2RootDst, TargetIncarnation: s4b2RootDst,
		Allow: false,
	}
	safe, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Admit(ctx, Admission{
		Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID,
		Key: "s4b2-peer-root-original-0000001", RequestID: "s4b2-peer-root-original",
		Action: "box_peer_set", SafeIntent: safe,
		BoxUUID: s4b2RootSrc, Incarnation: s4b2RootSrc,
		ResourceKey:  "acl:default/bw-s4b2",
		ResourceKeys: []string{"box:" + s4b2RootSrc, "box:" + s4b2RootDst},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO peer_grants
	(project,source_uuid,source_incarnation,target_uuid,target_incarnation,
	desired,status,source_ipv4,target_ipv4,source_ipv6,target_ipv6,operation_id)
	VALUES(?,?,?,?,?,0,'PENDING',?,?,?,?,?)`,
		"user-1001", s4b2RootSrc, s4b2RootSrc, s4b2RootDst, s4b2RootDst,
		"10.243.18.11", "10.243.18.12", "fd43:18::11", "fd43:18::12", r.ID)
	if err != nil {
		t.Fatal(err)
	}
	var seq int
	var name, path string
	if err := s.db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	reader, err := OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	return s, reader, r.ID
}

func TestS4B2RootRevocationOriginalOnlyAndClaims(t *testing.T) {
	ctx := context.Background()
	s, reader, id := s4b2RootOperationFixture(t)
	if _, err := reader.ReadRootPeerRevocation(ctx, id, "user-1001", "bwbr0", "bw-s4b2"); err == nil {
		t.Fatal("unclaimed ADMITTED op reached root")
	}
	if err := s.Claim(ctx, id); err != nil {
		t.Fatal(err)
	}
	got, err := reader.ReadRootPeerRevocation(ctx, id, "user-1001", "bwbr0", "bw-s4b2")
	if err != nil || got.OperationID != id || got.Intent.Allow || got.Intent.SourceUUID != s4b2RootSrc {
		t.Fatalf("valid claimed original refused %+v %v", got, err)
	}
	t.Log("S4_B2_ROOT_DISPATCH_SQLITE_CLAIMED_ORIGINAL_ONLY=PASS")
	if _, err := reader.ReadRootPeerRevocation(ctx, "deadbeef", "user-1001", "bwbr0", "bw-s4b2"); err == nil {
		t.Fatal("short forged operation identifier accepted")
	}
	if _, err := reader.ReadRootPeerRevocation(ctx, id, "foreign", "bwbr0", "bw-s4b2"); err == nil {
		t.Fatal("wrong project accepted")
	}
	if _, err := reader.ReadRootPeerRevocation(ctx, id, "user-1001", "other", "bw-s4b2"); err == nil {
		t.Fatal("wrong bridge accepted")
	}
	if _, err := reader.ReadRootPeerRevocation(ctx, id, "user-1001", "bwbr0", "foreign"); err == nil {
		t.Fatal("wrong ACL accepted")
	}
}

func TestS4B2RootRevocationCannotBypassClaimsOrUNKNOWN(t *testing.T) {
	cases := []struct{ name, sql string }{
		{"unknown-original", "UPDATE operations SET state='UNKNOWN' WHERE action='box_peer_set'"},
		{"completed-original", "UPDATE operations SET state='SUCCEEDED' WHERE action='box_peer_set'"},
		{"foreign-action", "UPDATE operations SET action='box_start' WHERE action='box_peer_set'"},
		{"revoked-credential", "UPDATE credentials SET revoked=1"},
		{"revoked-token", "UPDATE credentials SET revoked=1"},
		{"wrong-layer", "UPDATE operations SET layer='other' WHERE action='box_peer_set'"},
		{"wrong-box", "UPDATE operations SET box_uuid='other' WHERE action='box_peer_set'"},
		{"bad-safe-intent", "UPDATE operations SET safe_intent='{}' WHERE action='box_peer_set'"},
		{"uncanonical-JSON", "UPDATE operations SET safe_intent=' { } ' WHERE action='box_peer_set'"},
		{"grant-allows", "UPDATE peer_grants SET desired=1"},
		{"grant-unknown", "UPDATE peer_grants SET status='UNKNOWN'"},
		{"grant-mismatched-address", "UPDATE peer_grants SET source_ipv4='10.243.18.99'"},
		{"grant-stale-incarnation", "UPDATE peer_grants SET target_incarnation='new-incarnation'"},
		{"grant-wrong-id", "UPDATE peer_grants SET operation_id='wrong'"},
		{"extra-grant-same-original", `INSERT INTO peer_grants
			(project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status,operation_id)
			SELECT project,source_uuid,source_incarnation,'third','third',0,'PENDING',operation_id FROM peer_grants`},
		{"missing-acl-claim", "DELETE FROM operation_resource_claims WHERE resource_key LIKE 'acl:%'"},
		{"missing-target-claim", "DELETE FROM operation_resource_claims WHERE resource_key='box:" + s4b2RootDst + "'"},
		{"extra-foreign-resource-claim", `INSERT INTO operation_resource_claims(operation_id,layer,resource_key)
			SELECT id,layer,'box:foreign' FROM operations WHERE action='box_peer_set'`},
		{"quarantined-address", "UPDATE network_address_fences SET state='QUARANTINED' WHERE address='10.243.18.11'"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, reader, id := s4b2RootOperationFixture(t)
			if err := s.Claim(ctx, id); err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.ExecContext(ctx, tc.sql); err != nil {
				t.Fatal(err)
			}
			if proof, err := reader.ReadRootPeerRevocation(ctx, id, "user-1001", "bwbr0", "bw-s4b2"); err == nil {
				t.Fatalf("unsafe root operation accepted: %+v", proof)
			}
		})
	}
	t.Log("S4_B2_ROOT_DISPATCH_UNKNOWN_REVOKED_CLAIMS_AND_FOREIGN_REFUSED=PASS")
}
