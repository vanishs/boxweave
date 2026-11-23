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

// Reuses ONLY the prior one-use nested native guest. The original DB is
// cloned before corruption; production SQLite and native NIC are untouched.
func TestN3NativeMCPCreatePinRepair(t *testing.T) {
	project, source := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	reader := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	client, e := incus.ConnectIncusUnixWithContext(ctx, reader.Socket, nil)
	if e != nil {
		t.Fatal(e)
	}
	scoped := client.UseProject(project)
	names, e := scoped.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("unexpected native inventory", names, e)
	}
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(home, "n3-pin-repair-native-20261122")
	if e = os.Mkdir(dir, 0700); e != nil {
		t.Fatal("one-use repair proof directory", e)
	}
	repair := filepath.Join(dir, "state.db")
	src, e := sql.Open("sqlite3", source)
	if e != nil {
		t.Fatal(e)
	}
	defer src.Close()
	if _, e = src.ExecContext(ctx, "VACUUM INTO ?", repair); e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(repair, 0600); e != nil {
		t.Fatal(e)
	}
	s, e := state.Open(repair)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var id string
	if e = src.QueryRowContext(ctx, "SELECT id FROM operations WHERE action='box_create' AND work_ref='n3/create-ack-cut'").Scan(&id); e != nil {
		t.Fatal(e)
	}
	target, e := s.ReadRecoveryTarget(ctx, project, id)
	if e != nil || target.State != "SUCCEEDED" || target.NativeID != "" || target.Name != n3AckName || target.UUID == "" {
		t.Fatal(target, e)
	}
	box, e := reader.ObserveCreatedByOperation(ctx, n3AckName, id)
	if e != nil || box.UUID != target.UUID {
		t.Fatal(box, e)
	}
	pin, e := reader.ObserveOriginalCreateStaticPeerBinding(ctx, n3AckName, box.UUID, id)
	if e != nil {
		t.Fatal(e)
	}
	// A duplicated private fixture proves how a crash/restore inconsistency
	// appears to the real MCP without changing the authentic native guest.
	db, e := sql.Open("sqlite3", repair)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.ExecContext(ctx, "DELETE FROM network_bindings WHERE project=? AND box_uuid=?", project, target.UUID); e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, "DELETE FROM network_address_fences WHERE project=? AND box_uuid=? AND family='ipv6'", project, target.UUID); e != nil {
		t.Fatal(e)
	}
	st, e := s.CreatePinRepairStatus(ctx, target)
	if e != nil || st != "MISSING" {
		t.Fatal(st, e)
	}
	if _, e = s.ReadCompleteNetworkAuditLedger(ctx, project, pin.Bridge, pin.ACL); e == nil {
		t.Fatal("unrepaired ledger accepted")
	}
	call := n3AckMCP(t, ctx, s, reader)
	diag := call("operation_recover", map[string]any{"operation_id": id, "mode": "inspect"})["diagnosis"].(map[string]any)
	if diag["disposition"] != "SAFE_RECOVER" || diag["reason"] != "CREATE_PIN_LEDGER_REPAIRED" {
		t.Fatal("native repair not safe", diag)
	}
	args := map[string]any{"operation_id": id, "mode": "reconcile", "idempotency_key": "n3-native-pin-repair-reconcile"}
	first := call("operation_recover", args)
	recovery := first["recovery"].(map[string]any)
	op := recovery["operation"].(map[string]any)
	if recovery["disposition"] != "SETTLED" || op["state"] != "SUCCEEDED" || op["operation_id"] == id || first["native_retried"] != false {
		t.Fatal(first)
	}
	replay := call("operation_recover", args)["recovery"].(map[string]any)["operation"].(map[string]any)
	if replay["duplicate"] != true || replay["operation_id"] != op["operation_id"] {
		t.Fatal(replay)
	}
	after, e := s.ReadRecoveryTarget(ctx, project, id)
	if e != nil || after.State != "SUCCEEDED" || after.NativeID != "" || after.UUID != target.UUID {
		t.Fatal(after, e)
	}
	st, e = s.CreatePinRepairStatus(ctx, after)
	if e != nil || st != "COMPLETE" {
		t.Fatal(st, e)
	}
	ledger, e := s.ReadCompleteNetworkAuditLedger(ctx, project, pin.Bridge, pin.ACL)
	if e != nil || len(ledger.Boxes) != 1 || ledger.Boxes[0].UUID != box.UUID || ledger.Boxes[0].IPv4 != pin.IPv4 || ledger.Boxes[0].IPv6 != pin.IPv6 {
		t.Fatal(ledger, e)
	}
	names, e = scoped.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("unexpected native effect", names, e)
	}
	fmt.Printf("N3_NATIVE_MCP_PIN_REPAIR=PASS original=%s recovered=%s uuid=%s\n", id, op["operation_id"], box.UUID)
}
