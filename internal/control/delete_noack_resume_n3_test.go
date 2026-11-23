package control

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// Resume only an exact previous disposable worker-crash fixture. The
// test-specific trigger is dropped BEFORE product reopening, never disable
// strict SQLite verification, and never send a new Delete request.
func TestN3NativeMCPDeleteACKLostRestartEvidence(t *testing.T) {
	id := os.Getenv("BW_N3_DELETE_NOACK_RESUME_ORIGINAL")
	if id == "" {
		t.Skip("explicit exact prior test operation required")
	}
	project, _ := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	r := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	c, e := incus.ConnectIncusUnixWithContext(ctx, r.Socket, nil)
	if e != nil {
		t.Fatal(e)
	}
	scope := c.UseProject(project)
	names, e := scope.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("unexpected test native inventory", names, e)
	}
	if e = r.ConfirmDeleteAbsence(ctx, n3NoACKDeleteName); e != nil {
		t.Fatal(e)
	}
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	dst := filepath.Join(home, "n3-delete-noack-native-20261122", "state.db")
	db, e := sql.Open("sqlite3", dst)
	if e != nil {
		t.Fatal(e)
	}
	var phase, nativeID, uuid, name, work string
	e = db.QueryRowContext(ctx, `SELECT o.state,o.native_operation_id,o.box_uuid,b.name,o.work_ref
 FROM operations o JOIN owned_boxes b ON b.uuid=o.box_uuid AND b.project=o.layer AND b.incarnation=o.incarnation
 WHERE o.id=? AND o.layer=? AND o.action='box_delete'`, id, project).Scan(&phase, &nativeID, &uuid, &name, &work)
	if e != nil || (phase != "DISPATCHING" && phase != "UNKNOWN") || nativeID != "" || name != n3NoACKDeleteName || work != "n3/delete-ack-lost" {
		t.Fatal("wrong pre-restart original", phase, nativeID, uuid, name, work, e)
	}
	var triggers int
	if e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name='n3_delete_ack_fault'").Scan(&triggers); e != nil || triggers > 1 || (triggers == 0 && phase != "UNKNOWN") {
		t.Fatal("test cut not recorded", triggers, e)
	}
	var create string
	if e = db.QueryRowContext(ctx, "SELECT id FROM operations WHERE action='box_create' AND layer=? AND box_uuid=? AND state='SUCCEEDED'", project, uuid).Scan(&create); e != nil {
		t.Fatal(e)
	}
	if triggers == 1 {
		if _, e = db.ExecContext(ctx, "DROP TRIGGER n3_delete_ack_fault"); e != nil {
			t.Fatal(e)
		}
	}
	if e = db.Close(); e != nil {
		t.Fatal(e)
	}
	store, e := state.Open(dst)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	n, e := store.Recover(ctx)
	if e != nil || (n != 1 && n != 0) {
		t.Fatal("crash restart did not preserve UNKNOWN", n, e)
	}
	call := n3AckMCP(t, ctx, store, r)
	diag := call("operation_recover", map[string]any{"operation_id": id, "mode": "inspect"})["diagnosis"].(map[string]any)
	if diag["disposition"] != "BLOCKED" || diag["reason"] != "DEPENDENT_DELETE_FINAL_PROOF_REQUIRED" || diag["claim_protection"] != "RETAINED" {
		t.Fatal("lost native ACK fabricated causal success", diag)
	}
	args := map[string]any{"operation_id": id, "mode": "reconcile", "idempotency_key": "n3-noack-delete-protection-reconcile"}
	res := call("operation_recover", args)
	rec := res["recovery"].(map[string]any)
	op := rec["operation"].(map[string]any)
	if rec["disposition"] != "BLOCKED" || op["state"] != "FAILED" || res["native_retried"] != false {
		t.Fatal(res)
	}
	replay := call("operation_recover", args)["recovery"].(map[string]any)["operation"].(map[string]any)
	if replay["duplicate"] != true || replay["operation_id"] != op["operation_id"] {
		t.Fatal(replay)
	}
	original, e := store.Get(ctx, id)
	if e != nil || original.State != "UNKNOWN" || original.NativeOperationID != "" || original.BoxUUID != uuid {
		t.Fatal(original, e)
	}
	db, e = sql.Open("sqlite3", dst)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	for _, v := range []struct {
		query string
		want  int
	}{
		{"SELECT COUNT(*) FROM owned_boxes WHERE uuid=? AND state='OWNED'", 1},
		{"SELECT COUNT(*) FROM network_bindings WHERE box_uuid=? AND state='PINNED'", 1},
		{"SELECT COUNT(*) FROM network_address_fences WHERE box_uuid=? AND state='RESERVED' AND operation_id='" + create + "'", 2},
		{"SELECT COUNT(*) FROM network_address_fences WHERE box_uuid=? AND state='RELEASED'", 0},
	} {
		var count int
		if e = db.QueryRowContext(ctx, v.query, uuid).Scan(&count); e != nil || count != v.want {
			t.Fatal("protection altered", v, count, e)
		}
	}
	names, e = scope.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("native effect on restart", names, e)
	}
	fmt.Printf("N3_NATIVE_MCP_DELETE_ACK_LOST_FAIL_CLOSED=PASS create=%s delete=%s recovery=%s uuid=%s\n", create, id, op["operation_id"], uuid)
}
