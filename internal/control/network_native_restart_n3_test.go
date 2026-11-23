package control

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// Three actual OS processes with three newly initialized MCP server sessions.
// Parent only changes private SQLite clone, never native or the original DB.
func TestN3NativeMCPNetworkViewAfterProcessRestart(t *testing.T) {
	project, source := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(home, "n3-project-native-process-restart-v3-20261122", "state.db")
	nativeReader := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	if os.Getenv("BW_N3_NETWORK_RESTART_CHILD") == "1" {
		s, e := state.Open(path)
		if e != nil {
			t.Fatal(e)
		}
		defer s.Close()
		call := n3AckMCP(t, ctx, s, nativeReader)
		result := call("box_activity_get", map[string]any{"view": "native_network", "limit": 1})["activity"].(map[string]any)
		info := result["native_network"].(map[string]any)
		if info["owned"] != float64(1) || info["native"] != float64(1) || info["unowned_native"] != float64(0) || info["entire_project_inventory_scanned"] != true || result["attention_complete"] != false {
			t.Fatal("lost independent native inventory", result)
		}
		expected := os.Getenv("BW_N3_NETWORK_RESTART_EXPECT")
		issues, ok := result["issues"].([]any)
		if !ok && result["issues"] != nil {
			t.Fatal("issues unreadable", result)
		}
		if expected == "healthy" && len(issues) != 0 {
			t.Fatal("healthy restart incorrectly blocked", issues)
		}
		if expected == "drift" && (len(issues) != 1 || issues[0].(map[string]any)["reason"] != "NATIVE_DB_PIN_UNVERIFIED") {
			t.Fatal("restart failed to discover DB drift", issues)
		}
		t.Logf("N3_NATIVE_RESTART_CHILD=PASS state=%s", expected)
		return
	}
	c, e := incus.ConnectIncusUnixWithContext(ctx, nativeReader.Socket, nil)
	if e != nil {
		t.Fatal(e)
	}
	names, e := c.UseProject(project).GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("unexpected native prestate", names, e)
	}
	if e = os.Mkdir(filepath.Dir(path), 0700); e != nil {
		t.Fatal("one-use restart fixture", e)
	}
	src, e := sql.Open("sqlite3", source)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = src.ExecContext(ctx, "VACUUM INTO ?", path); e != nil {
		t.Fatal(e)
	}
	if e = src.Close(); e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(path, 0600); e != nil {
		t.Fatal(e)
	}
	db, e := sql.Open("sqlite3", path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var uuid, mac, original string
	if e = db.QueryRowContext(ctx, "SELECT c.box_uuid,n.mac,c.id FROM operations c JOIN network_bindings n ON n.project=c.layer AND n.box_uuid=c.box_uuid WHERE c.layer=? AND c.action='box_create' AND c.work_ref='n3/create-ack-cut'", project).Scan(&uuid, &mac, &original); e != nil {
		t.Fatal(e)
	}
	run := func(expected string) {
		child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestN3NativeMCPNetworkViewAfterProcessRestart$", "-test.v")
		child.Env = append(os.Environ(), "BW_N3_NETWORK_RESTART_CHILD=1", "BW_N3_NETWORK_RESTART_EXPECT="+expected)
		bytes, e := child.CombinedOutput()
		if e != nil || !strings.Contains(string(bytes), "N3_NATIVE_RESTART_CHILD=PASS state="+expected) {
			t.Fatalf("new MCP process %s failed: %v output=%s", expected, e, bytes)
		}
	}
	run("healthy")
	if _, e = db.ExecContext(ctx, "UPDATE network_bindings SET mac='02:16:3e:ab:cd:ef' WHERE project=? AND box_uuid=?", project, uuid); e != nil {
		t.Fatal(e)
	}
	run("drift")
	if _, e = db.ExecContext(ctx, "UPDATE network_bindings SET mac=? WHERE project=? AND box_uuid=?", mac, project, uuid); e != nil {
		t.Fatal(e)
	}
	run("healthy")
	box, e := nativeReader.ObserveCreatedByOperation(ctx, n3AckName, original)
	if e != nil || box.UUID != uuid || box.NativeState != "Stopped" {
		t.Fatal("native original not preserved", box, e)
	}
	names, e = c.UseProject(project).GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("native inventory changed", names, e)
	}
	t.Logf("N3_NATIVE_MCP_NETWORK_PROCESS_RESTART=PASS original=%s uuid=%s", original, uuid)
}
