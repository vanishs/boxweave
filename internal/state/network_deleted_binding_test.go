package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vanishs/boxweave/internal/peerscope"
	"net/netip"
	"testing"
)

// Synthetic exact terminal rows supplement real Scoped fixture transitions;
// this is a large SQLite history oracle, not a native lifecycle claim.
func seedReleasedNetworkHistory(t *testing.T, s *Store, n int) {
	t.Helper()
	tx, e := s.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := tx.Exec(q, args...); e != nil {
			t.Fatal(e)
		}
	}
	for i := 0; i < n; i++ {
		uuid := fmt.Sprintf("22222222-2222-4222-8222-%012d", i+1)
		name := fmt.Sprintf("retired-%04d", i)
		create, del := fmt.Sprintf("%032x", 2*i+1), fmt.Sprintf("%032x", 2*i+2)
		acl, e := peerscope.ACLName(purePeerProject, purePeerBridge, uuid)
		if e != nil {
			t.Fatal(e)
		}
		b := VerifiedCreateBinding{purePeerBridge, acl, fmt.Sprintf("02:aa:bb:cc:%02x:%02x", i/256, i%256), fmt.Sprintf("10.200.%d.%d", (i+1)/256, (i+1)%256), fmt.Sprintf("fd10:200::%x", i+1)}
		raw, _ := json.Marshal(ScopedCreateIntent{Schema: 2, Project: purePeerProject, Bridge: purePeerBridge, Name: name, CPU: 1, MemoryMiB: 256, DiskGiB: 2, Options: json.RawMessage("{}")})
		d, _ := json.Marshal(struct {
			BoxID string `json:"box_id"`
		}{uuid})
		exec("INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at,deleted_by_operation_id,deleted_at) VALUES(?,?,?,?,'DELETED',1,?,2)", uuid, purePeerProject, name, uuid, del)
		for _, op := range []struct {
			id, action, event string
			intent            []byte
			keys              []string
		}{
			{create, "box_create", "ORIGINAL_CREATE_UUID_ATOMIC", raw, append([]string{"box:" + name}, b.reservationKeys()...)},
			{del, "box_delete", "SCOPED_DELETE_ACL_ABSENT_ATOMIC", d, []string{"box:" + name, deleteProjectKey(purePeerProject)}},
		} {
			exec("INSERT INTO operations(id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,intent_digest,safe_intent,resource_key,box_uuid,incarnation,native_operation_id,state,created_at,updated_at) VALUES(?,?,?,?,'actor','token',?,?,'digest',?,?,?,?,?,'SUCCEEDED',1,2)", op.id, op.id, op.id, purePeerProject, op.id, op.action, op.intent, "box:"+name, uuid, uuid, "native-"+op.id)
			exec("INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,2,'SUCCEEDED',?)", op.id, op.event)
			for _, key := range op.keys {
				exec("INSERT INTO operation_resource_claims(operation_id,layer,resource_key) VALUES(?,?,?)", op.id, purePeerProject, key)
			}
		}
		exec("INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES(?,?,?,'eth0',?,?,?,?,?,'RELEASED',2)", purePeerProject, uuid, uuid, b.Bridge, b.ACL, b.MAC, b.IPv4, b.IPv6)
		for _, a := range []struct{ family, ip string }{{"ipv4", b.IPv4}, {"ipv6", b.IPv6}} {
			exec("INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,operation_id,observed_at) VALUES(?,?,?,?,?,?,'RELEASED',?,2)", b.Bridge, a.family, a.ip, purePeerProject, uuid, uuid, del)
		}
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
}
func TestNetworkHistoryBeyond256AndExactCorruption(t *testing.T) {
	ctx := context.Background()
	s, _, token := n3ScopedOwnedFixture(t)
	purePeerComplete(t, s, purePeerAdmit(t, s, token, "large-grant", 1, 2, true))
	purePeerComplete(t, s, purePeerAdmit(t, s, token, "large-revoke", 1, 2, false))
	seedReleasedNetworkHistory(t, s, 2048)
	got, e := s.ReadCompleteScopedNetworkAuditLedger(ctx, purePeerProject, purePeerBridge)
	if e != nil || len(got.Boxes) != 3 || len(got.Retired) != 2048 {
		t.Fatal(len(got.Boxes), len(got.Retired), e)
	}
	policy, e := s.ReadScopedTerminalPeerPolicy(ctx, purePeerProject, purePeerBridge)
	if e != nil || len(policy.Revoked) != 1 || len(policy.Active) != 0 {
		t.Fatal(policy, e)
	}
	holds, e := s.ReadScopedCreateAddressHolds(ctx, purePeerProject, purePeerBridge)
	if e != nil || !holds.IPs[netip.MustParseAddr(got.Retired[2047].IPv4)] {
		t.Fatal("last hold", e)
	}
	if _, e = s.db.Exec("UPDATE network_address_fences SET operation_id='foreign' WHERE box_uuid=? AND family='ipv6'", got.Retired[2047].UUID); e != nil {
		t.Fatal(e)
	}
	got, e = s.ReadCompleteScopedNetworkAuditLedger(ctx, purePeerProject, purePeerBridge)
	if !errors.Is(e, ErrResourceLocked) || len(got.Boxes) != 0 || len(got.Retired) != 0 {
		t.Fatal(got, e)
	}
}

func TestNetworkHistoryRejectsOutstandingDependencies(t *testing.T) {
	ctx := context.Background()
	for i, stmt := range []string{
		"UPDATE operations SET state='UNKNOWN' WHERE action='box_delete'",
		"UPDATE network_address_fences SET state='QUARANTINED' WHERE family='ipv6'",
		"INSERT INTO forward_claims(forward_id,project,box_uuid,incarnation,native_device,protocol,host_bind,host_port,guest_ip,guest_port,state) VALUES('old','user-1001','22222222-2222-4222-8222-000000000001','other-incarnation','bwpf-old','tcp','127.0.0.1',12345,'127.0.0.1',80,'UNKNOWN')",
		"INSERT INTO peer_grants(project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status) VALUES('user-1001','22222222-2222-4222-8222-000000000001','other-incarnation','other','other',1,'CONFIRMED')",
		"INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('other','ipv4','10.205.0.1','user-1001','22222222-2222-4222-8222-000000000001','22222222-2222-4222-8222-000000000001','RELEASED',2)",
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s := newStore(t)
			seedReleasedNetworkHistory(t, s, 1)
			r := purePeerReadOnly(t, s)
			if ledger, e := r.ReadScopedRetirementLedger(ctx, purePeerProject, purePeerBridge); e != nil || len(ledger.Retired) != 1 {
				t.Fatal(ledger, e)
			}
			if _, e := s.db.Exec(stmt); e != nil {
				t.Fatal(e)
			}
			if _, e := r.ReadScopedRetirementLedger(ctx, purePeerProject, purePeerBridge); !errors.Is(e, ErrResourceLocked) {
				t.Fatal("corrupt retirement accepted", e)
			}
		})
	}
}
