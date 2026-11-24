package state

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/vanishs/boxweave/internal/networklimits"
	"github.com/vanishs/boxweave/internal/peerscope"
)

func pr178ScopedStoreFixture(t *testing.T) *Store {
	t.Helper()
	s := newStore(t)
	s4b2InsertAuditFixture(t, s)
	for i := 0; i < 3; i++ {
		uuid := fmt.Sprintf("11111111-1111-4111-8111-%012d", i+1)
		acl, err := peerscope.ACLName("user-1001", "bwbr0", uuid)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.db.Exec("UPDATE network_bindings SET acl=? WHERE project='user-1001' AND box_uuid=?", acl, uuid); err != nil {
			t.Fatal(err)
		}
	}
	return s
}
func TestPR178ScopedAuditLedgerReadOnlyExactPinnedPerBoxACL(t *testing.T) {
	st := pr178ScopedStoreFixture(t)
	ctx := context.Background()
	got, err := st.ReadCompleteScopedNetworkAuditLedger(ctx, "user-1001", "bwbr0")
	if err != nil || len(got.Boxes) != 3 {
		t.Fatalf("complete per-Box state not independently readable: %+v %v", got, err)
	}
	seen := map[string]bool{}
	for i, b := range got.Boxes {
		if seen[b.ACL] || b.ACL == "" || b.IPv4 == "" || b.IPv6 == "" || b.MAC == "" || b.Incarnation == "" {
			t.Fatalf("reused or incomplete pinned exact ACL %+v", b)
		}
		expected, e := peerscope.ACLName(got.Project, got.Bridge, b.UUID)
		if e != nil || expected != b.ACL {
			t.Fatal("per-Box ACL is not deterministically owned", e)
		}
		seen[b.ACL] = true
		if i > 0 && got.Boxes[i-1].UUID >= b.UUID {
			t.Fatal("nondeterministic scoped ledger")
		}
	}
	// Old state model must remain strictly shared-ACL-only: the migration
	// source and target are distinct read-only APIs until separately reviewed.
	if _, e := st.ReadCompleteNetworkAuditLedger(ctx, "user-1001", "bwbr0", "bw-s4b2"); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("old shared ACL reader silently adopted per-Box ACLs", e)
	}
	legacy := newStore(t)
	s4b2InsertAuditFixture(t, legacy)
	if _, e := legacy.ReadCompleteScopedNetworkAuditLedger(ctx, "user-1001", "bwbr0"); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("current shared-ACL database was silently treated as fully migrated", e)
	}
}
func TestPR178ScopedAuditLedgerRejectsPartialMigrationAndNativeIdentityDrift(t *testing.T) {
	scenarios := []struct{ name, sql string }{
		{"missing-one-converted-acl", "UPDATE network_bindings SET acl='bw-s4b2' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='c')"},
		{"shared-acl-after-migration", "UPDATE network_bindings SET acl=(SELECT acl FROM network_bindings WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='a')) WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='b')"},
		{"foreign-acl", "UPDATE network_bindings SET acl='bw-peer-foreign' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='a')"},
		{"deleted-binding", "DELETE FROM network_bindings WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='a')"},
		{"wrong-incarnation", "UPDATE network_bindings SET incarnation='forged' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='b')"},
		{"wrong-network", "UPDATE network_bindings SET bridge='other' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='b')"},
		{"wrong-nic", "UPDATE network_bindings SET nic='eth2' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='b')"},
		{"bad-mac", "UPDATE network_bindings SET mac='01:00:5e:11:11:11' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='a')"},
		{"invalid-ipv6", "UPDATE network_bindings SET ipv6='fd43:0018::0011' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='a')"},
		{"quarantined", "UPDATE network_bindings SET state='QUARANTINED' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='a')"},
		{"released-history", "UPDATE network_bindings SET state='RELEASED' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='a')"},
		{"missing-fence", "DELETE FROM network_address_fences WHERE family='ipv6' AND address='fd43:18::12'"},
		{"foreign-fence", "UPDATE network_address_fences SET project='foreign' WHERE address='10.243.18.11'"},
		{"reused-fence", "UPDATE network_address_fences SET box_uuid='forged' WHERE address='10.243.18.11'"},
		{"released-fence", "UPDATE network_address_fences SET state='RELEASED' WHERE address='10.243.18.11'"},
		{"missing-owned", "UPDATE owned_boxes SET state='DELETED' WHERE name='b'"},
		{"unexpected-unpinned-box", "INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES('22222222-2222-4222-8222-222222222222','user-1001','foreign','22222222-2222-4222-8222-222222222222','OWNED',1)"},
		{"foreign-binding", "INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES('foreign','f','f','eth0','bwbr0','bw-foreign','02:16:3e:11:11:34','10.243.18.45','fd43:18::45','PINNED',1)"},
		{"foreign-lease", "INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv4','10.243.18.90','foreign','f','f','RESERVED',1)"},
	}
	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			st := pr178ScopedStoreFixture(t)
			if _, err := st.db.Exec(tc.sql); err != nil {
				t.Fatal(err)
			}
			got, err := st.ReadCompleteScopedNetworkAuditLedger(context.Background(), "user-1001", "bwbr0")
			if !errors.Is(err, ErrResourceLocked) || len(got.Boxes) != 0 {
				t.Fatalf("drifted per-Box ACL state accidentally authorized: %+v %v", got, err)
			}
		})
	}
}
func TestPR178ScopedAuditLedgerRefusesUntrustedRequests(t *testing.T) {
	st := pr178ScopedStoreFixture(t)
	for _, scope := range []struct{ project, bridge string }{
		{"default", "bwbr0"}, {"other", "bwbr0"}, {"user-1001", "../../root"}, {"user-1001", "other"}, {"user-1001", ""},
	} {
		got, e := st.ReadCompleteScopedNetworkAuditLedger(context.Background(), scope.project, scope.bridge)
		if e == nil || len(got.Boxes) != 0 {
			t.Fatal("untrusted scope accepted", scope)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := st.ReadCompleteScopedNetworkAuditLedger(ctx, "user-1001", "bwbr0"); !errors.Is(e, context.Canceled) {
		t.Fatal("cancelled root audit read context accepted", e)
	}
}

func TestPR178ScopedAuditLedgerSQLNeverAdmitsDuplicateBridgeAddress(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"mac", "UPDATE network_bindings SET mac=(SELECT mac FROM network_bindings WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='b')) WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='a')"},
		{"ipv4", "UPDATE network_bindings SET ipv4=(SELECT ipv4 FROM network_bindings WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='b')) WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='a')"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := pr178ScopedStoreFixture(t)
			if _, err := s.db.Exec(tc.sql); err == nil {
				t.Fatal("native SQLite unique bridge address/MAC constraint unexpectedly absent")
			}
			if _, err := s.ReadCompleteScopedNetworkAuditLedger(context.Background(), "user-1001", "bwbr0"); err != nil {
				t.Fatal("failed unsafe SQL statement modified valid inventory", err)
			}
		})
	}
}

func TestPR178ScopedAuditLedger1024SQLiteCapacity(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	tx, err := st.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for i := 0; i < networklimits.MaxProjectBoxes; i++ {
		uuid := fmt.Sprintf("11111111-1111-4111-8111-%012d", i+1)
		name := fmt.Sprintf("capacity-%04d", i)
		mac := fmt.Sprintf("02:16:3e:11:%02x:%02x", i/256, i%256)
		v4 := fmt.Sprintf("10.243.%d.%d", 16+(i+2)/256, (i+2)%256)
		v6 := fmt.Sprintf("fd43:18::%x", i+2)
		acl, e := peerscope.ACLName("user-1001", "bwbr0", uuid)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec("INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,'user-1001',?,?,'OWNED',1)", uuid, name, uuid); e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec("INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES('user-1001',?,?,'eth0','bwbr0',?,?,?,?,'PINNED',1)", uuid, uuid, acl, mac, v4, v6); e != nil {
			t.Fatal(e)
		}
		for _, f := range []struct{ family, ip string }{{"ipv4", v4}, {"ipv6", v6}} {
			if _, e = tx.Exec("INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0',?,?,'user-1001',?,?,'RESERVED',1)", f.family, f.ip, uuid, uuid); e != nil {
				t.Fatal(e)
			}
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	ledger, err := st.ReadCompleteScopedNetworkAuditLedger(ctx, "user-1001", "bwbr0")
	if err != nil || len(ledger.Boxes) != networklimits.MaxProjectBoxes {
		t.Fatal("complete 1024 native SQLite rows not accepted", len(ledger.Boxes), err)
	}
	if _, err = st.db.Exec("INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES('22222222-2222-4222-8222-222222222222','user-1001','overflow','22222222-2222-4222-8222-222222222222','OWNED',1)"); err != nil {
		t.Fatal(err)
	}
	ledger, err = st.ReadCompleteScopedNetworkAuditLedger(ctx, "user-1001", "bwbr0")
	if !errors.Is(err, ErrResourceLocked) || len(ledger.Boxes) != 0 {
		t.Fatal("1025th Box was accepted", len(ledger.Boxes), err)
	}
}
