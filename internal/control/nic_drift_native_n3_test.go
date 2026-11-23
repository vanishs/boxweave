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

const n3DriftLabName = "n3-nic-drift-lab"

// Disposable stopped native Box. The only out-of-band change is a temporary
// MAC in the dedicated fixture, restored before verified normal Delete.
func TestN3NativeMCPPhysicalNICDrift(t *testing.T) {
	project, source := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 220*time.Second)
	defer cancel()
	reader := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	c, e := incus.ConnectIncusUnixWithContext(ctx, reader.Socket, nil)
	if e != nil {
		t.Fatal(e)
	}
	scoped := c.UseProject(project)
	names, e := scoped.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("unexpected inventory", names, e)
	}
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(home, "n3-real-drift-20261122")
	if e = os.Mkdir(dir, 0700); e != nil {
		t.Fatal("one-use drift DB", e)
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
	defer store.Close()
	call := n3AckMCP(t, ctx, store, reader)
	cr := call("box_create", map[string]any{"name": n3DriftLabName, "start": false, "cpu": 1, "memory_mib": 512, "disk_gib": 4,
		"idempotency_key": "n3-physical-drift-native-create", "work_ref": "n3/physical-drift"})
	create := cr["operation_id"].(string)
	var r state.Receipt
	deadline := time.Now().Add(100 * time.Second)
	for time.Now().Before(deadline) {
		r, e = store.Get(ctx, create)
		if e != nil {
			t.Fatal(e)
		}
		if r.State == "SUCCEEDED" {
			break
		}
		if r.State == "FAILED" || r.State == "UNKNOWN" {
			t.Fatal("drift create refused", r)
		}
		time.Sleep(250 * time.Millisecond)
	}
	if r.State != "SUCCEEDED" || r.BoxUUID == "" {
		t.Fatal("create timeout", r)
	}
	pre := call("operation_recover", map[string]any{"operation_id": create, "mode": "inspect"})["diagnosis"].(map[string]any)
	if pre["disposition"] != "NOT_REQUIRED" || pre["reason"] != "CREATE_PIN_LEDGER_COMPLETE" {
		t.Fatal("initial native pin unhealthy", pre)
	}
	inst, etag, e := scoped.GetInstance(n3DriftLabName)
	if e != nil || inst.Status != "Stopped" || inst.Config["volatile.uuid"] != r.BoxUUID {
		t.Fatal("fixture identity", inst, e)
	}
	mac := inst.Devices["eth0"]["hwaddr"]
	if mac == "" || mac == "02:16:3e:fe:bd:01" {
		t.Fatal("unexpected original MAC", mac)
	}
	setMAC := func(callCtx context.Context, value string) error {
		current, etag, err := scoped.GetInstance(n3DriftLabName)
		if err != nil {
			return err
		}
		if current.Config["volatile.uuid"] != r.BoxUUID || current.Status != "Stopped" {
			return native.ErrNotOwned
		}
		put := current.Writable()
		copyNIC := map[string]string{}
		for k, v := range put.Devices["eth0"] {
			copyNIC[k] = v
		}
		copyNIC["hwaddr"] = value
		put.Devices["eth0"] = copyNIC
		op, err := scoped.UpdateInstance(n3DriftLabName, put, etag)
		if err != nil {
			return err
		}
		return nativeDone(callCtx, reader, op.Get().ID)
	}
	modified := false
	defer func() {
		if modified {
			restoreCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if e := setMAC(restoreCtx, mac); e != nil {
				t.Errorf("fixture MAC restore failed: %v", e)
			}
		}
	}()
	if etag == "" {
		t.Fatal("missing original native ETag")
	}
	if e = setMAC(ctx, "02:16:3e:fe:bd:01"); e != nil {
		t.Fatal("temporary stopped NIC drift", e)
	}
	modified = true
	drift := call("operation_recover", map[string]any{"operation_id": create, "mode": "inspect"})["diagnosis"].(map[string]any)
	if drift["disposition"] != "BLOCKED" || drift["reason"] != "CREATE_NATIVE_PIN_DRIFT" {
		t.Fatal("out-of-band NIC drift not detected", drift)
	}
	if e = setMAC(ctx, mac); e != nil {
		t.Fatal("native NIC restoration failed", e)
	}
	modified = false
	healthy := call("operation_recover", map[string]any{"operation_id": create, "mode": "inspect"})["diagnosis"].(map[string]any)
	if healthy["disposition"] != "NOT_REQUIRED" || healthy["reason"] != "CREATE_PIN_LEDGER_COMPLETE" {
		t.Fatal("NIC restore unproved", healthy)
	}
	dr := call("box_delete", map[string]any{"box_id": r.BoxUUID, "idempotency_key": "n3-physical-drift-native-delete", "work_ref": "n3/physical-drift"})
	did := dr["operation_id"].(string)
	deadline = time.Now().Add(100 * time.Second)
	for time.Now().Before(deadline) {
		row, e := store.Get(ctx, did)
		if e != nil {
			t.Fatal(e)
		}
		if row.State == "SUCCEEDED" {
			break
		}
		if row.State == "UNKNOWN" || row.State == "FAILED" {
			t.Fatal("final safe Delete failed", row)
		}
		time.Sleep(250 * time.Millisecond)
	}
	del, e := store.Get(ctx, did)
	if e != nil || del.State != "SUCCEEDED" {
		t.Fatal("native delete timeout", del, e)
	}
	if e = reader.ConfirmDeleteAbsence(ctx, n3DriftLabName); e != nil {
		t.Fatal(e)
	}
	names, e = scoped.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("original guest changed", names, e)
	}
	fmt.Printf("N3_NATIVE_MCP_NIC_DRIFT=PASS create=%s delete=%s uuid=%s\n", create, did, r.BoxUUID)
}
