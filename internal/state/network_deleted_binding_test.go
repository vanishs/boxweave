package state

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func seedReleasedNetworkHistory(t *testing.T, s *Store, n int) {
	t.Helper()
	tx, e := s.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	for i := 0; i < n; i++ {
		uuid := fmt.Sprintf("retired-%04d", i)
		del := fmt.Sprintf("%032x", i+1)
		mac := fmt.Sprintf("02:aa:bb:cc:%02x:%02x", i/256, i%256)
		v4 := fmt.Sprintf("10.200.%d.%d", (i+1)/256, (i+1)%256)
		v6 := fmt.Sprintf("fd10:200::%x", i+1)
		if _, e = tx.Exec("INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at,deleted_by_operation_id,deleted_at) VALUES(?,'user-1001',?,?,'DELETED',1,?,2)", uuid, uuid, uuid, del); e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec("INSERT INTO operations(id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,intent_digest,safe_intent,resource_key,box_uuid,incarnation,native_operation_id,state,created_at,updated_at) VALUES(?,?,?,'user-1001','actor','token',?,'box_delete','digest','{}',?,?,?,?,'SUCCEEDED',1,2)", del, del, del, del, "box:"+uuid, uuid, uuid, "native-"+del); e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec("INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES('user-1001',?,?,'eth0','bwbr0','bw-s4b2',?,?,?,'RELEASED',2)", uuid, uuid, mac, v4, v6); e != nil {
			t.Fatal(e)
		}
		for _, a := range []struct{ family, ip string }{{"ipv4", v4}, {"ipv6", v6}} {
			if _, e = tx.Exec("INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,operation_id,observed_at) VALUES('bwbr0',?,?,'user-1001',?,?,'RELEASED',?,2)", a.family, a.ip, uuid, uuid, del); e != nil {
				t.Fatal(e)
			}
		}
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
}
func TestNetworkHistoryBeyond256AndExactCorruption(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	s4b2InsertAuditFixture(t, s)
	seedReleasedNetworkHistory(t, s, 2048)
	got, e := s.ReadCompleteNetworkAuditLedger(ctx, "user-1001", "bwbr0", "bw-s4b2")
	if e != nil || len(got.Boxes) != 3 {
		t.Fatal(got, e)
	}
	// Revoked rows can still be compiled using independently verified historic identities.
	if _, e = s.db.Exec("INSERT INTO peer_grants(project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status,source_ipv4,target_ipv4,source_ipv6,target_ipv6) VALUES('user-1001','retired-0000','retired-0000','11111111-1111-4111-8111-000000000001','11111111-1111-4111-8111-000000000001',0,'REVOKED','10.200.0.1','10.243.18.11','fd10:200::1','fd43:18::11')"); e != nil {
		t.Fatal(e)
	}
	revoked, e := s.ReadPeerRevokedSnapshot(ctx, "user-1001", "bwbr0", "bw-s4b2")
	if e != nil || len(revoked) != 1 {
		t.Fatal(revoked, e)
	}
	// A last-row corruption must fail, not disappear due to a limit.
	if _, e = s.db.Exec("UPDATE network_address_fences SET operation_id='foreign' WHERE box_uuid='retired-2047' AND family='ipv6'"); e != nil {
		t.Fatal(e)
	}
	got, e = s.ReadCompleteNetworkAuditLedger(ctx, "user-1001", "bwbr0", "bw-s4b2")
	if !errors.Is(e, ErrResourceLocked) || len(got.Boxes) != 0 {
		t.Fatal(got, e)
	}
}
func TestNetworkHistoryRejectsOutstandingDependencies(t *testing.T) {
	cases := []string{
		"UPDATE operations SET state='UNKNOWN' WHERE box_uuid='retired-0000'",
		"UPDATE network_address_fences SET state='QUARANTINED' WHERE box_uuid='retired-0000' AND family='ipv6'",
		"INSERT INTO forward_claims(forward_id,project,box_uuid,incarnation,native_device,protocol,host_bind,host_port,guest_ip,guest_port,state) VALUES('old','user-1001','retired-0000','other-incarnation','bwpf-old','tcp','127.0.0.1',12345,'127.0.0.1',80,'UNKNOWN')",
		"INSERT INTO peer_grants(project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status) VALUES('user-1001','retired-0000','other-incarnation','other','other',1,'CONFIRMED')",
		"INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('other','ipv4','10.205.0.1','user-1001','retired-0000','retired-0000','RELEASED',2)",
	}
	for i, stmt := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s := newStore(t)
			s4b2InsertAuditFixture(t, s)
			seedReleasedNetworkHistory(t, s, 1)
			if _, e := s.db.Exec(stmt); e != nil {
				t.Fatal(e)
			}
			if _, e := s.ReadCompleteNetworkAuditLedger(context.Background(), "user-1001", "bwbr0", "bw-s4b2"); !errors.Is(e, ErrResourceLocked) {
				t.Fatal(e)
			}
		})
	}
}
