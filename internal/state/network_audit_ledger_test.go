package state

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func s4b2InsertAuditFixture(t *testing.T, s *Store) {
	t.Helper()
	for i, name := range []string{"a", "b", "c"} {
		id := fmt.Sprintf("11111111-1111-4111-8111-%012d", i+1)
		mac := fmt.Sprintf("02:16:3e:11:12:%02x", i+1)
		v4 := fmt.Sprintf("10.243.18.%d", i+11)
		v6 := fmt.Sprintf("fd43:18::%d", i+11)
		for _, a := range []struct {
			q string
			v []any
		}{
			{"INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,'OWNED',?)", []any{id, "user-1001", name, id, time.Now().UnixNano()}},
			{"INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES(?,?,?,?,?,?,?,?,?,'PINNED',?)", []any{"user-1001", id, id, "eth0", "bwbr0", "bw-s4b2", mac, v4, v6, time.Now().UnixNano()}},
			{"INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv4',?,?,?,?,'RESERVED',?)", []any{v4, "user-1001", id, id, time.Now().UnixNano()}},
			{"INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv6',?,?,?,?,'RESERVED',?)", []any{v6, "user-1001", id, id, time.Now().UnixNano()}},
		} {
			if _, err := s.db.Exec(a.q, a.v...); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestS4B2CompleteNetworkAuditLedger(t *testing.T) {
	s := newStore(t)
	s4b2InsertAuditFixture(t, s)
	out, err := s.ReadCompleteNetworkAuditLedger(context.Background(), "user-1001", "bwbr0", "bw-s4b2")
	if err != nil || len(out.Boxes) != 3 {
		t.Fatal(out, err)
	}
	for i, b := range out.Boxes {
		if b.UUID == "" || b.UUID != b.Incarnation || b.MAC == "" || b.IPv4 == "" || b.IPv6 == "" {
			t.Fatal(b)
		}
		if i > 0 && out.Boxes[i-1].UUID >= b.UUID {
			t.Fatal("not deterministic")
		}
	}
	if _, err := s.ReadCompleteNetworkAuditLedger(context.Background(), "user-1001", "other", "bw-s4b2"); !errors.Is(err, ErrResourceLocked) {
		t.Fatal(err)
	}
	if _, err := s.ReadCompleteNetworkAuditLedger(context.Background(), "default", "bwbr0", "bw-s4b2"); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
}
func TestS4B2NetworkAuditLedgerRefusesDrift(t *testing.T) {
	cases := []struct{ name, sql string }{
		{"unbound", "DELETE FROM network_bindings"},
		{"missing-owned", "DELETE FROM network_bindings WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='c')"},
		{"bad-mac", "UPDATE network_bindings SET mac='01:00:5e:11:11:11' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='a')"},
		{"wrong-acl", "UPDATE network_bindings SET acl='foreign' WHERE nic='eth0'"},
		{"other-incarnation", "UPDATE network_bindings SET incarnation='replaced' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='b')"},
		{"missing-ipv4-fence", "DELETE FROM network_address_fences WHERE address='10.243.18.12'"},
		{"quarantine", "UPDATE network_address_fences SET state='QUARANTINED' WHERE address='fd43:18::12'"},
		{"recycle", "UPDATE network_address_fences SET box_uuid='new-uuid' WHERE address='10.243.18.12'"},
		{"released", "UPDATE network_address_fences SET state='RELEASED' WHERE family='ipv4'"},
		{"foreign-binding", "INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES('foreign','f','f','eth0','bwbr0','bw-s4b2','02:16:3e:11:11:11','10.243.18.45','fd43:18::45','PINNED',1)"},
		{"foreign-lease", "INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv4','10.243.18.90','foreign','f','f','RESERVED',1)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore(t)
			s4b2InsertAuditFixture(t, s)
			if _, err := s.db.Exec(tc.sql); err != nil {
				t.Fatal(err)
			}
			got, err := s.ReadCompleteNetworkAuditLedger(context.Background(), "user-1001", "bwbr0", "bw-s4b2")
			if !errors.Is(err, ErrResourceLocked) || len(got.Boxes) != 0 {
				t.Fatalf("unsafe ledger accepted %+v: %v", got, err)
			}
		})
	}
}
