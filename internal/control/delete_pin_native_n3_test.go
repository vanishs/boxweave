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

const n3DeleteLabName = "n3-delete-ack-lab"

// Real nested/native test: a second, disposable Box is created in a copy
// of the already-proven original DB; original DB/instance are never touched.
func TestN3NativeMCPModernDeleteTombstoneCut(t *testing.T) {
	project, source := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	reader := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	c, e := incus.ConnectIncusUnixWithContext(ctx, reader.Socket, nil)
	if e != nil {
		t.Fatal(e)
	}
	scoped := c.UseProject(project)
	names, e := scoped.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("unsafe native fixture inventory", names, e)
	}
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(home, "n3-delete-native-20261122")
	if e = os.Mkdir(dir, 0700); e != nil {
		t.Fatal("test DB directory not one-use", e)
	}
	dest := filepath.Join(dir, "state.db")
	copyDB, e := sql.Open("sqlite3", source)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = copyDB.ExecContext(ctx, "VACUUM INTO ?", dest); e != nil {
		t.Fatal(e)
	}
	if e = copyDB.Close(); e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(dest, 0600); e != nil {
		t.Fatal(e)
	}
	store, e := state.Open(dest)
	if e != nil {
		t.Fatal(e)
	}
	call := n3AckMCP(t, ctx, store, reader)
	created := call("box_create", map[string]any{
		"name": n3DeleteLabName, "start": false, "cpu": 1, "memory_mib": 512, "disk_gib": 4,
		"idempotency_key": "n3-delete-fixture-native-create", "work_ref": "n3/delete-native-cut",
	})
	cid := created["operation_id"].(string)
	var cRec state.Receipt
	until := time.Now().Add(100 * time.Second)
	for time.Now().Before(until) {
		cRec, e = store.Get(ctx, cid)
		if e != nil {
			t.Fatal(e)
		}
		if cRec.State == "SUCCEEDED" {
			break
		}
		if cRec.State == "UNKNOWN" || cRec.State == "FAILED" {
			t.Fatal("disposable create failed", cRec)
		}
		time.Sleep(250 * time.Millisecond)
	}
	if cRec.State != "SUCCEEDED" || cRec.BoxUUID == "" {
		t.Fatal("create deadline", cRec)
	}
	pin, e := reader.ObserveOriginalCreateStaticPeerBinding(ctx, n3DeleteLabName, cRec.BoxUUID, cid)
	if e != nil {
		t.Fatal("original dual-stack marker", e)
	}
	before, e := store.ReadCompleteNetworkAuditLedger(ctx, project, pin.Bridge, pin.ACL)
	if e != nil || len(before.Boxes) != 2 {
		t.Fatal("two native pinned records required", before, e)
	}
	// Deliberate failure in cloned DB, for this named disposable Box only.
	fault, e := sql.Open("sqlite3", dest)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = fault.ExecContext(ctx, `CREATE TRIGGER n3_delete_fault BEFORE UPDATE OF state ON owned_boxes
 WHEN OLD.name='n3-delete-ack-lab' AND NEW.state='DELETED'
 BEGIN SELECT RAISE(ABORT,'fixture cut after native Delete'); END`); e != nil {
		t.Fatal(e)
	}
	deleted := call("box_delete", map[string]any{
		"box_id": cRec.BoxUUID, "idempotency_key": "n3-delete-fixture-native-original",
		"work_ref": "n3/delete-native-cut",
	})
	did := deleted["operation_id"].(string)
	var dRec state.Receipt
	until = time.Now().Add(110 * time.Second)
	for time.Now().Before(until) {
		dRec, e = store.Get(ctx, did)
		if e != nil {
			t.Fatal(e)
		}
		if dRec.State == "UNKNOWN" {
			break
		}
		if dRec.State == "FAILED" || dRec.State == "SUCCEEDED" {
			t.Fatal("fault did not preserve UNKNOWN", dRec)
		}
		time.Sleep(250 * time.Millisecond)
	}
	if dRec.State != "UNKNOWN" || dRec.NativeOperationID == "" || dRec.BoxUUID != cRec.BoxUUID {
		t.Fatal("native terminal tombstone fault absent", dRec)
	}
	if e = reader.ConfirmDeleteAbsence(ctx, n3DeleteLabName); e != nil {
		t.Fatal(e)
	}
	observed, e := reader.ObserveUnknownDelete(ctx, n3DeleteLabName, dRec.BoxUUID, dRec.NativeOperationID)
	if e != nil || observed.Class != "ORIGINAL_TERMINAL_DELETE_VERIFIED" {
		t.Fatal("exact native Delete proof", observed, e)
	}
	if _, e = fault.ExecContext(ctx, "DROP TRIGGER n3_delete_fault"); e != nil {
		t.Fatal(e)
	}
	if e = fault.Close(); e != nil {
		t.Fatal(e)
	}
	if e = store.Close(); e != nil {
		t.Fatal(e)
	}
	reopened, e := state.Open(dest)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if n, e := reopened.Recover(ctx); e != nil || n != 0 {
		t.Fatal("unexpected nonterminal DB after restart", n, e)
	}
	call2 := n3AckMCP(t, ctx, reopened, reader)
	diag := call2("operation_recover", map[string]any{"operation_id": did, "mode": "inspect"})["diagnosis"].(map[string]any)
	if diag["disposition"] != "SAFE_RECOVER" || diag["reason"] != "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED" {
		t.Fatal("not recoverable exact native delete", diag)
	}
	args := map[string]any{"operation_id": did, "mode": "reconcile", "idempotency_key": "n3-delete-fixture-native-recover"}
	result := call2("operation_recover", args)
	recovery := result["recovery"].(map[string]any)
	op := recovery["operation"].(map[string]any)
	if recovery["disposition"] != "SETTLED" || op["state"] != "SUCCEEDED" || result["native_retried"] != false {
		t.Fatal("native Delete recovery refused", result)
	}
	replay := call2("operation_recover", args)["recovery"].(map[string]any)["operation"].(map[string]any)
	if replay["duplicate"] != true || replay["operation_id"] != op["operation_id"] {
		t.Fatal("replay mismatch", replay)
	}
	post, e := reopened.Get(ctx, did)
	if e != nil || post.State != "SUCCEEDED" || post.NativeOperationID != dRec.NativeOperationID {
		t.Fatal(post, e)
	}
	db, e := sql.Open("sqlite3", dest)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	for _, q := range []struct {
		sql  string
		want int
	}{
		{"SELECT COUNT(*) FROM owned_boxes WHERE uuid=? AND state='DELETED'", 1},
		{"SELECT COUNT(*) FROM network_bindings WHERE box_uuid=? AND state='RELEASED'", 1},
		{"SELECT COUNT(*) FROM network_address_fences WHERE box_uuid=? AND state='RELEASED' AND operation_id='" + did + "'", 2},
	} {
		var n int
		if e = db.QueryRowContext(ctx, q.sql, dRec.BoxUUID).Scan(&n); e != nil || n != q.want {
			t.Fatal("tombstone not atomic", q, n, e)
		}
	}
	history, e := reopened.ReadIPv4RetirementCandidate(ctx, project, did)
	if e != nil || history.CreateOperationID != cid {
		t.Fatal("retained original address lineage", history, e)
	}
	ledger, e := reopened.ReadCompleteNetworkAuditLedger(ctx, project, pin.Bridge, pin.ACL)
	if e != nil || len(ledger.Boxes) != 1 || ledger.Boxes[0].Name != n3AckName {
		t.Fatal("historical native deletion not audited", ledger, e)
	}
	names, e = scoped.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("original fixture changed", names, e)
	}
	fmt.Printf("N3_NATIVE_MCP_DELETE_TOMBSTONE_CUT=PASS create=%s delete=%s recovery=%s uuid=%s\n", cid, did, op["operation_id"], dRec.BoxUUID)
}
