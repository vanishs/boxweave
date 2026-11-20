package install

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/state"
)

// Disposable CI only: confirm that protected SQLite cannot substitute for
// Incus root MAC/UUID/bridge readback, even with a complete pinned ledger.
func TestS4B2NativeStoreLedgerBridgeAudit(t *testing.T) {
	if os.Getenv("BOXWEAVE_S4B2_DISPOSABLE_ROOT_AUDIT") != "20261117-read-only" ||
		os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("disposable root Incus only")
	}
	if os.Geteuid() != 0 {
		t.Fatal("root-only disposable native proof")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	// A GitHub runner can have a non-private inherited test directory.
	// Match the product's SQLite security requirements instead of relaxing
	// Store.Open: the immediate DB parent must be an actual private 0700 dir.
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "owned.sqlite")
	st, err := state.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := nativePeerBridgeAuditFromStoreWithClient(ctx, st, root, "bwowned", "bwbr0", "bw-s4b2-native"); err == nil {
		t.Fatal("missing native binding/dual-stack claims accepted")
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	own := root.UseProject("bwowned")
	for _, name := range []string{"s4b2a", "s4b2b", "s4b2c"} {
		box, _, err := own.GetInstance(name)
		if err != nil {
			t.Fatal(err)
		}
		uuid := box.Config["volatile.uuid"]
		mac := strings.ToLower(box.Config["volatile.eth0.hwaddr"])
		if mac == "" {
			mac = strings.ToLower(box.ExpandedDevices["eth0"]["hwaddr"])
		}
		ipv4, ipv6 := box.Devices["eth0"]["ipv4.address"], box.Devices["eth0"]["ipv6.address"]
		if uuid == "" || mac == "" || ipv4 == "" || ipv6 == "" {
			t.Fatal("missing original native identity", name)
		}
		if _, err = db.ExecContext(ctx, "INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,'OWNED',?)", uuid, "bwowned", name, uuid, time.Now().UnixNano()); err != nil {
			t.Fatal(err)
		}
		if _, err = db.ExecContext(ctx, "INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES(?,?,?,?,?,?,?,?,?,'PINNED',?)", "bwowned", uuid, uuid, "eth0", "bwbr0", "bw-s4b2-native", mac, ipv4, ipv6, time.Now().UnixNano()); err != nil {
			t.Fatal(err)
		}
		for _, p := range []struct{ family, address string }{{"ipv4", ipv4}, {"ipv6", ipv6}} {
			if _, err = db.ExecContext(ctx, "INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0',?,?,?, ?,?,'RESERVED',?)", p.family, p.address, "bwowned", uuid, uuid, time.Now().UnixNano()); err != nil {
				t.Fatal(err)
			}
		}
	}
	// Reopen this live WAL database via the dedicated root-audit read-only
	// path; the root verifier must not take a writable controller handle.
	reader, err := state.OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal("native root ledger cannot open read-only", err)
	}
	defer reader.Close()
	result, err := nativePeerBridgeAuditFromStoreWithClient(ctx, reader, root, "bwowned", "bwbr0", "bw-s4b2-native")
	if err != nil || result.OwnedBoxes != 3 || len(result.NativeDigestSHA256) != 64 {
		t.Fatalf("full SQLite-to-native root proof: %+v: %v", result, err)
	}
	t.Log("S4_B2_SQLITE_COMPLETE_LEDGER_TO_ROOT_INCUS_AUDIT=PASS")
	t.Log("S4_B2_ROOT_SQLITE_READ_ONLY_WAL_AUDIT=PASS")
	t.Log("S4_B2_ROOT_SQLITE_NATIVE_DOUBLE_SNAPSHOT_STABLE=PASS")
	if _, err = db.ExecContext(ctx, "UPDATE network_bindings SET mac='02:16:3e:fe:fe:fe' WHERE box_uuid=(SELECT uuid FROM owned_boxes WHERE name='s4b2a')"); err != nil {
		t.Fatal(err)
	}
	if _, err = nativePeerBridgeAuditFromStoreWithClient(ctx, reader, root, "bwowned", "bwbr0", "bw-s4b2-native"); err == nil {
		t.Fatal("root native MAC drift accepted despite internally consistent SQLite")
	}
	t.Log("S4_B2_SQLITE_MAC_CLAIM_NATIVE_ROOT_MISMATCH_REFUSED=PASS")
	if _, err = db.ExecContext(ctx, "UPDATE network_address_fences SET state='QUARANTINED' WHERE family='ipv6'"); err != nil {
		t.Fatal(err)
	}
	if _, err = nativePeerBridgeAuditFromStoreWithClient(ctx, reader, root, "bwowned", "bwbr0", "bw-s4b2-native"); err == nil {
		t.Fatal("quarantined dual-stack address admitted")
	}
	t.Log("S4_B2_SQLITE_ADDRESS_QUARANTINE_REFUSED=PASS")
}
