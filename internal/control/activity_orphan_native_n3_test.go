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

func TestN3NativeMCPOrphanNetworkRestoreActivity(t *testing.T) {
	project, source := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	r := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	c, e := incus.ConnectIncusUnixWithContext(ctx, r.Socket, nil)
	if e != nil {
		t.Fatal(e)
	}
	scoped := c.UseProject(project)
	names, e := scoped.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("unexpected native inventory", names, e)
	}
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(home, "n3-orphan-restore-native-20261122")
	if e = os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	dst := filepath.Join(dir, "state.db")
	src, e := sql.Open("sqlite3", source)
	if e != nil {
		t.Fatal(e)
	}
	var uuid, original string
	if e = src.QueryRowContext(ctx, "SELECT id,box_uuid FROM operations WHERE layer=? AND action='box_create' AND work_ref='n3/create-ack-cut'", project).Scan(&original, &uuid); e != nil {
		t.Fatal(e)
	}
	if _, e = src.ExecContext(ctx, "VACUUM INTO ?", dst); e != nil {
		t.Fatal(e)
	}
	if e = src.Close(); e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(dst, 0600); e != nil {
		t.Fatal(e)
	}
	broken, e := sql.Open("sqlite3", dst)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = broken.ExecContext(ctx, "UPDATE owned_boxes SET state='DELETED' WHERE project=? AND name=? AND uuid=? AND state='OWNED'", project, n3AckName, uuid); e != nil {
		t.Fatal(e)
	}
	if e = broken.Close(); e != nil {
		t.Fatal(e)
	}
	s, e := state.Open(dst)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	call := n3AckMCP(t, ctx, s, r)
	args := map[string]any{"view": "issues", "box_id": uuid, "limit": 1}
	seen := map[string]int{}
	for i := 0; i < 8; i++ {
		page := call("box_activity_get", args)["activity"].(map[string]any)
		if page["attention_complete"] != false {
			t.Fatal("reported full native coverage", page)
		}
		list, ok := page["issues"].([]any)
		if !ok || len(list) != 1 {
			t.Fatal("unreadable issue page", page)
		}
		issue := list[0].(map[string]any)
		name := issue["reason"].(string)
		seen[name]++
		if issue["box_id"] != uuid || issue["disposition"] != "INSPECT_EVIDENCE" {
			t.Fatal("unsafe issue attribution", issue)
		}
		if issue["operation_id"] != original {
			t.Fatal("original missing", issue)
		}
		next, ok := page["next_cursor"].(string)
		if !ok || next == "" {
			break
		}
		args["cursor"] = next
	}
	if seen["NETWORK_ORPHAN_BINDING"] != 1 || seen["NETWORK_ORPHAN_ADDRESS_FENCE"] != 2 || len(seen) != 2 {
		t.Fatal("orphan issue inventory incomplete", seen)
	}
	filtered := call("box_activity_get", map[string]any{"view": "issues", "box_id": uuid, "work_ref": "other-unrelated-work", "limit": 10})["activity"].(map[string]any)
	if list, ok := filtered["issues"].([]any); ok && len(list) != 0 {
		t.Fatal("wrong work attributed orphan", list)
	}
	box, e := r.ObserveCreatedByOperation(ctx, n3AckName, original)
	if e != nil || box.UUID != uuid || box.NativeState != "Stopped" {
		t.Fatal("real native fixture changed", box, e)
	}
	names, e = scoped.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("native inventory changed", names, e)
	}
	t.Logf("N3_NATIVE_MCP_ORPHAN_RESTORE_ACTIVITY=PASS original=%s uuid=%s", original, uuid)
}
