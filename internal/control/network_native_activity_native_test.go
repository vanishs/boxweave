package control

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// One-use test on an exact previously verified stopped native Box.
// ALL corruption is in a VACUUM clone; the real native Box never changes.
func TestN3NativeMCPFullProjectNetworkAudit(t *testing.T) {
	project, source := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	r := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	c, e := incus.ConnectIncusUnixWithContext(ctx, r.Socket, nil)
	if e != nil {
		t.Fatal(e)
	}
	scoped := c.UseProject(project)
	names, e := scoped.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("fixture inventory", names, e)
	}
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(home, "n3-full-project-scan-v2-20261122")
	if e = os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	copyPath := filepath.Join(dir, "state.db")
	db, e := sql.Open("sqlite3", source)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, "VACUUM INTO ?", copyPath); e != nil {
		t.Fatal(e)
	}
	if e = db.Close(); e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(copyPath, 0600); e != nil {
		t.Fatal(e)
	}
	s, e := state.Open(copyPath)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	call := n3AckMCP(t, ctx, s, r)
	check := func() map[string]any {
		result := call("box_activity_get", map[string]any{"view": "native_network", "limit": 1})["activity"].(map[string]any)
		audit := result["native_network"].(map[string]any)
		if audit["owned"] != float64(1) || audit["native"] != float64(1) || audit["unowned_native"] != float64(0) ||
			audit["entire_project_inventory_scanned"] != true || audit["cross_source_atomic"] != false ||
			result["attention_complete"] != false || result["source"] != "SQLITE_AND_NATIVE_READONLY" {
			t.Fatal("unsafe coverage", result)
		}
		return result
	}
	healthy := check()
	if list, ok := healthy["issues"].([]any); ok && len(list) != 0 {
		t.Fatal("healthy native pin misdiagnosed", healthy)
	}
	var original, uuid, mac string
	db, e = sql.Open("sqlite3", copyPath)
	if e != nil {
		t.Fatal(e)
	}
	if e = db.QueryRowContext(ctx, "SELECT o.id,o.box_uuid,n.mac FROM operations o JOIN network_bindings n ON n.box_uuid=o.box_uuid AND n.project=o.layer WHERE o.layer=? AND o.action='box_create' AND o.work_ref='n3/create-ack-cut'", project).Scan(&original, &uuid, &mac); e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, "UPDATE network_bindings SET mac='02:16:3e:de:ad:fe' WHERE project=? AND box_uuid=?", project, uuid); e != nil {
		t.Fatal(e)
	}
	drift := check()
	list, ok := drift["issues"].([]any)
	if !ok || len(list) != 1 || list[0].(map[string]any)["reason"] != "NATIVE_NIC_IDENTITY_DRIFT" || list[0].(map[string]any)["box_id"] != uuid {
		t.Fatal("native drift not detected", drift)
	}
	if _, e = db.ExecContext(ctx, "UPDATE operations SET resource_key='box:tampered' WHERE id=?", original); e != nil {
		t.Fatal(e)
	}
	seen := map[string]bool{}
	args := map[string]any{"view": "native_network", "limit": 1}
	for i := 0; i < 8; i++ {
		page := call("box_activity_get", args)["activity"].(map[string]any)
		items, ok := page["issues"].([]any)
		if !ok || len(items) != 1 {
			t.Fatal("page truncated", page)
		}
		issue := items[0].(map[string]any)
		if issue["box_id"] != uuid || issue["operation_id"] != original {
			t.Fatal("foreign/wrong original", issue)
		}
		seen[issue["reason"].(string)] = true
		cursor, _ := page["next_cursor"].(string)
		if cursor == "" {
			break
		}
		args["cursor"] = cursor
	}
	if !seen["NETWORK_CREATE_LINEAGE_CONFLICT"] || !seen["NATIVE_DB_PIN_UNVERIFIED"] || len(seen) != 2 {
		t.Fatal("missing root causes", seen)
	}
	if _, e = db.ExecContext(ctx, "UPDATE operations SET resource_key='box:'||? WHERE id=?", n3AckName, original); e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, "UPDATE network_bindings SET mac=? WHERE project=? AND box_uuid=?", mac, project, uuid); e != nil {
		t.Fatal(e)
	}
	if e = db.Close(); e != nil {
		t.Fatal(e)
	}
	healthy = check()
	if items, ok := healthy["issues"].([]any); ok && len(items) != 0 {
		t.Fatal("restored clone still damaged", items)
	}
	box, e := r.ObserveCreatedByOperation(ctx, n3AckName, original)
	if e != nil || box.UUID != uuid || box.NativeState != "Stopped" {
		t.Fatal("native original changed", box, e)
	}
	t.Logf("N3_NATIVE_MCP_FULL_NETWORK_INVENTORY=PASS original=%s uuid=%s", original, uuid)
}
