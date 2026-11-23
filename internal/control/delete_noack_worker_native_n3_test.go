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

const n3NoACKDeleteWorkerName = "n3-delete-noack-worker-lab"

// Fresh one-use real native Delete ACK double-write fault with the new live Worker fallback.
// Absence with no native receipt is intentionally NOT repaired or released.
func TestN3NativeMCPDeleteACKLostWorkerLiveFallback(t *testing.T) {
	project, source := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 230*time.Second)
	defer cancel()
	reader := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	c, e := incus.ConnectIncusUnixWithContext(ctx, reader.Socket, nil)
	if e != nil {
		t.Fatal(e)
	}
	scoped := c.UseProject(project)
	names, e := scoped.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("unexpected native state", names, e)
	}
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(home, "n3-delete-noack-worker-native-20261122")
	if e = os.Mkdir(dir, 0700); e != nil {
		t.Fatal("one-use lab DB path", e)
	}
	dst := filepath.Join(dir, "state.db")
	src, e := sql.Open("sqlite3", source)
	if e != nil {
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
	store, e := state.Open(dst)
	if e != nil {
		t.Fatal(e)
	}
	call := n3AckMCP(t, ctx, store, reader)
	create := call("box_create", map[string]any{
		"name": n3NoACKDeleteWorkerName, "start": false, "cpu": 1, "memory_mib": 512, "disk_gib": 4,
		"idempotency_key": "n3-worker-noack-delete-disposable-create", "work_ref": "n3/delete-ack-lost-worker",
	})
	cid := create["operation_id"].(string)
	var cr state.Receipt
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		cr, e = store.Get(ctx, cid)
		if e != nil {
			t.Fatal(e)
		}
		if cr.State == "SUCCEEDED" {
			break
		}
		if cr.State == "FAILED" || cr.State == "UNKNOWN" {
			t.Fatal("disposable Create failed", cr)
		}
		time.Sleep(250 * time.Millisecond)
	}
	if cr.State != "SUCCEEDED" || cr.BoxUUID == "" {
		t.Fatal("native Create deadline", cr)
	}
	db, e := sql.Open("sqlite3", dst)
	if e != nil {
		t.Fatal(e)
	}
	// Product DB opened BEFORE test trigger. A native ACK write after the
	// operation was sent will fail; worker may mark only an UNKNOWN.
	if _, e = db.ExecContext(ctx, `CREATE TRIGGER n3_delete_ack_fault BEFORE UPDATE OF native_operation_id ON operations
 WHEN OLD.action='box_delete' AND OLD.native_operation_id='' AND NEW.native_operation_id!=''
 BEGIN SELECT RAISE(ABORT,'test-only lost Delete ACK'); END`); e != nil {
		t.Fatal(e)
	}
	del := call("box_delete", map[string]any{
		"box_id": cr.BoxUUID, "idempotency_key": "n3-worker-noack-delete-original",
		"work_ref": "n3/delete-ack-lost-worker",
	})
	did := del["operation_id"].(string)
	var dr state.Receipt
	deadline = time.Now().Add(100 * time.Second)
	for time.Now().Before(deadline) {
		dr, e = store.Get(ctx, did)
		if e != nil {
			t.Fatal(e)
		}
		if dr.State == "UNKNOWN" {
			break
		}
		if dr.State == "SUCCEEDED" || dr.State == "FAILED" {
			t.Fatal("unsafe ACK cut outcome", dr)
		}
		time.Sleep(250 * time.Millisecond)
	}
	if dr.State != "UNKNOWN" || dr.NativeOperationID != "" || dr.BoxUUID != cr.BoxUUID {
		t.Fatal("ACK cut failed", dr)
	}
	if e = reader.ConfirmDeleteAbsence(ctx, n3NoACKDeleteWorkerName); e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, "DROP TRIGGER n3_delete_ack_fault"); e != nil {
		t.Fatal(e)
	}
	if e = db.Close(); e != nil {
		t.Fatal(e)
	}
	if e = store.Close(); e != nil {
		t.Fatal(e)
	}
	reopened, e := state.Open(dst)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if n, e := reopened.Recover(ctx); e != nil || n != 0 {
		t.Fatal("unexpected terminal after recovery", n, e)
	}
	call2 := n3AckMCP(t, ctx, reopened, reader)
	diag := call2("operation_recover", map[string]any{"operation_id": did, "mode": "inspect"})["diagnosis"].(map[string]any)
	if diag["disposition"] != "BLOCKED" || diag["reason"] != "DEPENDENT_DELETE_FINAL_PROOF_REQUIRED" || diag["claim_protection"] != "RETAINED" {
		t.Fatal("absent native Delete ACK falsely proved", diag)
	}
	args := map[string]any{"operation_id": did, "mode": "reconcile", "idempotency_key": "n3-worker-noack-delete-protection-reconcile"}
	first := call2("operation_recover", args)
	rec := first["recovery"].(map[string]any)
	op := rec["operation"].(map[string]any)
	if rec["disposition"] != "BLOCKED" || op["state"] != "FAILED" || first["native_retried"] != false {
		t.Fatal(first)
	}
	replay := call2("operation_recover", args)["recovery"].(map[string]any)["operation"].(map[string]any)
	if replay["duplicate"] != true || replay["operation_id"] != op["operation_id"] {
		t.Fatal(replay)
	}
	original, e := reopened.Get(ctx, did)
	if e != nil || original.State != "UNKNOWN" || original.NativeOperationID != "" {
		t.Fatal("lost original protection", original, e)
	}
	db, e = sql.Open("sqlite3", dst)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	for _, tc := range []struct {
		sql  string
		want int
	}{
		{"SELECT COUNT(*) FROM owned_boxes WHERE uuid=? AND state='OWNED'", 1},
		{"SELECT COUNT(*) FROM network_bindings WHERE box_uuid=? AND state='PINNED'", 1},
		{"SELECT COUNT(*) FROM network_address_fences WHERE box_uuid=? AND state='RESERVED' AND operation_id='" + cid + "'", 2},
	} {
		var count int
		if e = db.QueryRowContext(ctx, tc.sql, cr.BoxUUID).Scan(&count); e != nil || count != tc.want {
			t.Fatal("unproven Delete released safety barrier", tc, count, e)
		}
	}
	names, e = scoped.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("unexpected native guest", names, e)
	}
	fmt.Printf("N3_NATIVE_MCP_DELETE_ACK_LOST_LIVE_UNKNOWN=PASS create=%s delete=%s recovery=%s uuid=%s\n", cid, did, op["operation_id"], cr.BoxUUID)
}
