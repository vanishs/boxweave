package install

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/state"
)

func TestS4B2ConntrackScopeMustComeFromFullSQLiteLedger(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "owned.sqlite")
	s, err := state.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ids := []string{"11111111-1111-4111-8111-000000000001", "11111111-1111-4111-8111-000000000002", "11111111-1111-4111-8111-000000000003"}
	for i, id := range ids {
		now := time.Now().UnixNano()
		v4 := fmt.Sprintf("10.243.18.%d", i+11)
		v6 := fmt.Sprintf("fd43:18::%d", i+11)
		mac := fmt.Sprintf("02:16:3e:11:12:%02x", i+1)
		for _, q := range []struct {
			sql  string
			args []any
		}{
			{"INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,'OWNED',?)", []any{id, "user-1001", fmt.Sprintf("b%d", i), id, now}},
			{"INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES(?,?,?,?,?,?,?,?,?,'PINNED',?)", []any{"user-1001", id, id, "eth0", "bwbr0", "bw-s4b2", mac, v4, v6, now}},
			{"INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv4',?,?,?,?,'RESERVED',?)", []any{v4, "user-1001", id, id, now}},
			{"INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv6',?,?,?,?,'RESERVED',?)", []any{v6, "user-1001", id, id, now}},
		} {
			if _, err := db.Exec(q.sql, q.args...); err != nil {
				t.Fatal(err)
			}
		}
	}
	scope, ledger, err := nativePeerConntrackTargetFromStore(t.Context(), s, "user-1001", "bwbr0", "bw-s4b2", ids[0], ids[1])
	if err != nil || len(scope.Boxes) != 3 || len(ledger.Boxes) != 3 || scope.Boxes[0].UUID != ids[0] {
		t.Fatal("complete bounded identity was not derived from SQLite", scope, err)
	}
	for _, pair := range [][2]string{{ids[0], ids[0]}, {ids[0], "foreign-uuid"}, {"", ids[1]}} {
		if _, _, err := nativePeerConntrackTargetFromStore(t.Context(), s, "user-1001", "bwbr0", "bw-s4b2", pair[0], pair[1]); err == nil {
			t.Fatal("unbound pair UUID passed state gate", pair)
		}
	}
	if _, err := db.Exec("UPDATE network_address_fences SET state='QUARANTINED' WHERE box_uuid=?", ids[1]); err != nil {
		t.Fatal(err)
	}
	if _, _, err := nativePeerConntrackTargetFromStore(t.Context(), s, "user-1001", "bwbr0", "bw-s4b2", ids[0], ids[1]); err == nil {
		t.Fatal("quarantined IPv4/IPv6 scope used for root conntrack")
	}
}

func TestScopedConntrackUsesProtectedSQLiteLedger(t *testing.T) {
	fixture, _ := pr178RootSDKFixture(t)
	_, root, _, id := scopedInstallStore(t, fixture)

	proof, err := root.ReadRootScopedPeerOriginal(
		t.Context(),
		id,
		fixture.Intent.Project,
		fixture.Intent.Bridge,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(proof.Inventory.Boxes) != len(fixture.Inventory.Boxes) {
		t.Fatal("incomplete scoped ledger")
	}

	if proof.Intent.SourceUUID != fixture.Intent.SourceUUID ||
		proof.Intent.TargetUUID != fixture.Intent.TargetUUID {
		t.Fatal("incorrect scoped pair")
	}
}

