//go:build ignore

// Exact installed-candidate acceptance on the disposable S2 CI VM only.
// No production fault flag, no shared host/ledger, and no original replay.
package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	_ "github.com/mattn/go-sqlite3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
)

type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(cp)
}
func check(ok bool, msg string) {
	if !ok {
		panic("S5A2_NATIVE_FAIL: " + msg)
	}
}
func obj(x any) map[string]any {
	m, ok := x.(map[string]any)
	check(ok, "missing structured object")
	return m
}
func str(m map[string]any, k string) string { s, _ := m[k].(string); return s }

type lab struct {
	ctx    context.Context
	s      *mcp.ClientSession
	db     *sql.DB
	native incus.InstanceServer
	reader native.Reader
}

func (l lab) call(name string, args map[string]any) map[string]any {
	r, e := l.s.CallTool(l.ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	check(e == nil && r != nil && !r.IsError, "MCP refused "+name)
	raw, e := json.Marshal(r.StructuredContent)
	check(e == nil, "encode response")
	var m map[string]any
	check(json.Unmarshal(raw, &m) == nil && m != nil, "structured response")
	return m
}
func (l lab) deny(name string, args map[string]any) {
	r, e := l.s.CallTool(l.ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	check(e != nil || r == nil || r.IsError, "unexpected success "+name)
}
func (l lab) poll(id, want string) map[string]any {
	end := time.Now().Add(120 * time.Second)
	for time.Now().Before(end) {
		op := obj(l.call("operation_get", map[string]any{"operation_id": id, "wait_ms": 1000})["operation"])
		check(str(op, "operation_id") == id, "original ID drift")
		switch str(op, "state") {
		case "SUCCEEDED", "FAILED", "UNKNOWN", "CANCELLED", "TIMED_OUT":
			check(str(op, "state") == want, "wrong original terminal "+str(op, "state"))
			return op
		}
	}
	panic("S5A2_NATIVE_FAIL: original poll deadline")
}
func (l lab) original(name, key string, args map[string]any, fault bool) map[string]any {
	args["idempotency_key"] = "s5a2-original-" + key
	if fault {
		args["work_ref"] = "s5a2/recovery-fault"
	}
	r := l.call(name, args)
	id := str(r, "operation_id")
	check(len(id) == 32, "admission lost")
	fmt.Printf("S5A2_NATIVE_ADMITTED ACTION=%s BW_ID=%s\n", name, id)
	want := "SUCCEEDED"
	if fault {
		want = "UNKNOWN"
	}
	op := l.poll(id, want)
	fmt.Printf("S5A2_NATIVE_ORIGINAL ACTION=%s BW_ID=%s NATIVE_ID=%s STATE=%s\n", name, id, str(op, "native_operation_id"), want)
	return op
}
func (l lab) recover(op map[string]any, key, want string) map[string]any {
	id := str(op, "operation_id")
	var originalSeq, originalAt int64
	var originalDetail string
	check(l.db.QueryRow("SELECT seq,at,safe_detail FROM operation_events WHERE operation_id=? AND state='UNKNOWN' ORDER BY seq LIMIT 1", id).Scan(&originalSeq, &originalAt, &originalDetail) == nil, "original UNKNOWN event missing before recovery")
	var count int
	check(l.db.QueryRow("SELECT count(*) FROM operation_events WHERE operation_id=? AND state='UNKNOWN'", id).Scan(&count) == nil && count == 1, "unexpected original UNKNOWN history before recovery")
	args := map[string]any{"operation_id": id, "mode": "reconcile", "idempotency_key": "s5a2-recovery-" + key}
	out := l.call("operation_recover", args)
	rec := obj(out["recovery"])
	receipt := obj(rec["operation"])
	rid := str(receipt, "operation_id")
	fmt.Printf("S5A2_NATIVE_RECOVERY ACTION=%s ORIGINAL=%s RECOVERY=%s DISPOSITION=%s REASON=%s\n", str(op, "action"), id, rid, str(rec, "disposition"), str(rec, "reason"))
	check(rec["disposition"] == want && rec["original_operation_id"] == id && len(rid) == 32 && rid != id && str(receipt, "native_operation_id") == "" && out["native_retried"] == false, "wrong recovery receipt")
	dup := obj(l.call("operation_recover", args)["recovery"])
	dupReceipt := obj(dup["operation"])
	check(dupReceipt["operation_id"] == rid && dupReceipt["duplicate"] == true && dup["disposition"] == want, "lost same-key recovery receipt")
	var state, nativeID, linked string
	check(l.db.QueryRow("SELECT state,native_operation_id FROM operations WHERE id=?", id).Scan(&state, &nativeID) == nil && nativeID == str(op, "native_operation_id"), "original SQLite linkage")
	expected := "UNKNOWN"
	if want == "SETTLED" {
		expected = "SUCCEEDED"
	}
	check(state == expected, "original outcome or claim changed incorrectly")
	check(l.db.QueryRow("SELECT json_extract(safe_intent,'$.original_operation_id') FROM operations WHERE id=? AND action='operation_recover'", rid).Scan(&linked) == nil && linked == id, "durable recovery lineage")
	// An identity-only binding appends a second UNKNOWN event. The original
	// event must remain unchanged and the new event must occur exactly once.
	var at int64
	var detail, eventState string
	check(l.db.QueryRow("SELECT at,state,safe_detail FROM operation_events WHERE operation_id=? AND seq=?", id, originalSeq).Scan(&at, &eventState, &detail) == nil && at == originalAt && eventState == "UNKNOWN" && detail == originalDetail, "original UNKNOWN event changed")
	expectedUnknown := 1
	if want == "IDENTITY_BOUND" {
		expectedUnknown = 2
	}
	check(l.db.QueryRow("SELECT count(*) FROM operation_events WHERE operation_id=? AND state='UNKNOWN'", id).Scan(&count) == nil && count == expectedUnknown, "unexpected UNKNOWN event count after recovery")
	if want == "IDENTITY_BOUND" {
		check(l.db.QueryRow("SELECT count(*) FROM operation_events WHERE operation_id=? AND seq>? AND state='UNKNOWN' AND safe_detail='EXACT_NATIVE_UUID_BOUND_RECOVERY_NO_REPLAY'", id, originalSeq).Scan(&count) == nil && count == 1, "identity binding evidence missing or duplicated")
	}
	check(l.db.QueryRow("SELECT count(*) FROM recovery_audit WHERE operation_id=? AND decision=?", id, "RECOVERY_"+want).Scan(&count) == nil && count == 1, "recovery audit missing or duplicated")
	check(l.db.QueryRow("SELECT count(*) FROM recovery_audit WHERE operation_id=? AND decision='RELEASED'", id).Scan(&count) == nil && count == 0, "unapproved claim release")
	return out
}
func (l lab) terminal(op map[string]any, exit *int) {
	id := str(op, "native_operation_id")
	check(len(id) == 36, "original native identity missing")
	actual, _, e := l.native.GetOperation(id)
	check(e == nil && actual.ID == id && actual.StatusCode == api.Success, "independent original native terminal unavailable")
	if exit != nil {
		code, ok := actual.Metadata["return"].(float64)
		check(ok && int(code) == *exit, "native Guest exit mismatch")
	}
}
func main() {
	path := os.Getenv("S5A2_DB")
	check(os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" && os.Getenv("S5A2_RECOVERY_NATIVE_ONLY") == "true" && os.Getuid() != 0, "one-use CI guard")
	check(filepath.IsAbs(path) && strings.HasPrefix(path, "/tmp/bw-v1-s2-ghci.") && strings.HasSuffix(path, "/ledger/ledger.sqlite"), "exact disposable ledger required")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	token := os.Getenv("S5A2_TOKEN")
	check(token != "", "one-use authentication unavailable")
	cli := mcp.NewClient(&mcp.Implementation{Name: "s5a2-one-use", Version: "1"}, nil)
	sess, e := cli.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: "http://127.0.0.1:18650/mcp", HTTPClient: &http.Client{Transport: bearer{token}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	check(e == nil, "client initialization")
	defer sess.Close()
	db, e := sql.Open("sqlite3", "file:"+path+"?_busy_timeout=4000")
	check(e == nil, "fixture ledger open")
	defer db.Close()
	project := "user-" + strconv.Itoa(os.Getuid())
	c, e := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket.user", nil)
	check(e == nil, "native witness connection")
	l := lab{ctx: ctx, s: sess, db: db, native: c.UseProject(project), reader: native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}}
	inv, e := sess.ListTools(ctx, nil)
	check(e == nil && len(inv.Tools) == 23, "exact installed tool count")
	found := false
	for _, tool := range inv.Tools {
		if tool.Name == "operation_recover" {
			found = true
		}
	}
	check(found, "installed recovery tool missing")
	// Test-only, exact fault label on the disposable ledger. Original native
	// work runs through the actual binary. Only the final SQLite write fails;
	// recovery UNKNOWN->SUCCEEDED is intentionally not affected.
	_, e = db.Exec(`CREATE TRIGGER s5a2_only_ci_fault BEFORE UPDATE OF state ON operations WHEN OLD.state IN ('VERIFYING','DISPATCHING') AND NEW.state='SUCCEEDED' AND OLD.work_ref='s5a2/recovery-fault' BEGIN SELECT RAISE(ABORT,'S5A2_ONE_USE_FINALIZATION_FAULT'); END`)
	check(e == nil, "test fault installation")
	defer db.Exec("DROP TRIGGER IF EXISTS s5a2_only_ci_fault")
	mk := func(name, key string, start, fault bool) map[string]any {
		return l.original("box_create", key, map[string]any{"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2, "start": start, "internet": false}, fault)
	}
	name := "s5a2-recovery-native"
	created := mk(name, "create-fixture", true, false)
	uuid := str(created, "box_uuid")
	check(len(uuid) == 36, "owned UUID missing")
	exec := l.original("box_exec", "exec-once", map[string]any{"box_id": uuid, "argv": []string{"/bin/sh", "-c", "printf 's5a2-once\\n' >> /var/tmp/s5a2-once; printf 's5a2-stdout'"}}, true)
	zero := 0
	l.terminal(exec, &zero)
	l.recover(exec, "exec-once", "SETTLED")
	out := l.call("operation_output_get", map[string]any{"operation_id": str(exec, "operation_id"), "stream": "stdout"})
	data, e := base64.StdEncoding.DecodeString(str(out, "content_base64"))
	check(e == nil && string(data) == "s5a2-stdout" && out["exit_code"] == float64(0), "original recorded output/exit mismatch")
	page, e := l.reader.ReadGuestFile(ctx, name, uuid, "/var/tmp/s5a2-once", 0, 128)
	check(e == nil && page.WholeFile && string(page.Data) == "s5a2-once\n", "native guest effect not exactly once")
	fmt.Println("S5A2_NATIVE_EXEC_ORIGINAL_SQLITE_OUTPUT_AND_ONE_EFFECT=PASS")
	for _, tc := range []struct {
		action, key, want string
		values            map[string]any
	}{{"box_stop", "stop", "Stopped", nil}, {"box_update", "update", "Stopped", map[string]any{"cpu": 2}}, {"box_start", "start", "Running", nil}, {"box_stop", "stop-again", "Stopped", nil}} {
		args := map[string]any{"box_id": uuid}
		for k, v := range tc.values {
			args[k] = v
		}
		op := l.original(tc.action, tc.key, args, true)
		l.terminal(op, nil)
		l.recover(op, tc.key, "SETTLED")
		actual, _, e := l.reader.GetExactBox(ctx, name, uuid)
		check(e == nil && actual.NativeState == tc.want, "independent power postread")
		if tc.action == "box_update" {
			check(actual.CPU == "2", "independent update postread")
		}
	}
	fmt.Println("S5A2_NATIVE_POWER_UPDATE_ORIGINAL_TERMINAL_AND_POSTSTATE=PASS")
	op := l.original("box_delete", "delete", map[string]any{"box_id": uuid}, true)
	l.terminal(op, nil)
	l.recover(op, "delete", "SETTLED")
	_, _, e = l.native.GetInstance(name)
	check(api.StatusErrorCheck(e, http.StatusNotFound), "native deletion not proved")
	var phase, deletedBy string
	check(db.QueryRow("SELECT state,deleted_by_operation_id FROM owned_boxes WHERE uuid=?", uuid).Scan(&phase, &deletedBy) == nil && phase == "DELETED" && deletedBy == str(op, "operation_id"), "original delete tombstone")
	fmt.Println("S5A2_NATIVE_DELETE_TERMINAL_EXACT_ABSENCE_TOMBSTONE=PASS")
	// The identity-only bind MUST leave UNKNOWN and prevent new effects.
	name = "s5a2-identity-native"
	unbound := mk(name, "unbound-create", false, true)
	check(str(unbound, "box_uuid") == "", "fault did not leave original unbound")
	inst, _, e := l.native.GetInstance(name)
	check(e == nil && inst.Config["user.boxweave.create_operation"] == str(unbound, "operation_id"), "exact marker unverified")
	uuid = inst.Config["volatile.uuid"]
	check(len(uuid) == 36, "native identity missing")
	result := l.recover(unbound, "bind-create", "IDENTITY_BOUND")
	diagnosis := obj(result["diagnosis"])
	check(diagnosis["box_id"] == uuid && diagnosis["claim_protection"] == "RETAINED", "binding unlocked original")
	l.deny("box_start", map[string]any{"box_id": uuid, "idempotency_key": "s5a2-must-refuse-after-binding"})
	fmt.Println("S5A2_NATIVE_CREATE_BIND_IDENTITY_ONLY_LOCK_RETAINED=PASS")
	// Retain this UNKNOWN fixture; the disposable VM, not production
	// recovery, supplies teardown. No new operation on its held resources.
	fmt.Println("S5A2_NATIVE_INSTALLED_RECOVERY=PASS")
}
