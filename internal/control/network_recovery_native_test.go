package control

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// One-use real Incus fixture. A SQLite trigger cuts only the final commit
// after real native completion, leaving the original UNKNOWN. Recovery must
// use the original native receipt, exact poststate, and a distinct receipt.
func TestS5A4NativeMCPNetworkRecovery(t *testing.T) {
	if os.Getenv("BOXWEAVE_B4_MCP_ONE_USE") != "s4b4-disposable-only-20261119" || os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("disposable native MCP only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	project := "user-" + strconv.Itoa(os.Getuid())
	c, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket.user", nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, _, err := c.UseProject(project).GetInstance("s4b4native")
	if err != nil {
		t.Fatal(err)
	}
	uuid := inst.Config["volatile.uuid"]
	if uuid == "" {
		t.Fatal("fixture native UUID absent")
	}
	private := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(private, "a4.sqlite")
	st, err := state.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, "INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,?,?)", uuid, project, "s4b4native", uuid, "OWNED", time.Now().UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := st.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{Store: st, Native: native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project, ExternalForwardAllow: "bwbr0/tcp/[fd83:248::1]:23163,bwbr0/udp/[fd83:248::1]:23164"}, BuildVersion: "a4-native-mcp-test", BuildCommit: "source-exact"}
	srv := httptest.NewUnstartedServer(svc.Handler())
	svc.Listener = srv.Listener.Addr().String()
	srv.Start()
	defer srv.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "s4-b5-native-client", Version: "1"}, nil)
	sess, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp", HTTPClient: &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: token}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	type binding struct {
		proto       string
		host, guest int
	}
	bindings := []binding{{"tcp", 24111, 19091}, {"udp", 24112, 19092}}
	ids := []string{}
	call := func(name string, args map[string]any) map[string]any {
		r, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		return b4MCPData(t, r, e)
	}
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER s5a4_fault BEFORE UPDATE OF state ON operations WHEN OLD.state IN ('VERIFYING','DISPATCHING') AND NEW.state='SUCCEEDED' AND OLD.work_ref='s5a4/recovery-fault' AND OLD.action IN ('port_forward_create','port_forward_delete','box_delete') BEGIN SELECT RAISE(ABORT,'s5a4 terminal commit cut'); END`); err != nil {
		t.Fatal(err)
	}
	recoverOriginal := func(original string) {
		t.Helper()
		var prior state.Receipt
		deadline := time.Now().Add(40 * time.Second)
		for time.Now().Before(deadline) {
			var e error
			prior, e = st.Get(ctx, original)
			if e != nil {
				t.Fatal(e)
			}
			if prior.State == "UNKNOWN" {
				break
			}
			if prior.State == "SUCCEEDED" || prior.State == "FAILED" || prior.State == "CANCELLED" {
				t.Fatalf("fault cut did not retain UNKNOWN: %+v", prior)
			}
			time.Sleep(100 * time.Millisecond)
		}
		if prior.State != "UNKNOWN" || prior.NativeOperationID == "" {
			t.Fatalf("missing original witness: %+v", prior)
		}
		args := map[string]any{"operation_id": original, "mode": "reconcile", "idempotency_key": "s5a4-recover-" + original}
		first := call("operation_recover", args)
		rec := first["recovery"].(map[string]any)
		op := rec["operation"].(map[string]any)
		if rec["disposition"] != "SETTLED" || rec["original_operation_id"] != original || op["state"] != "SUCCEEDED" || op["operation_id"] == original || first["native_retried"] != false {
			t.Fatalf("recovery failed: %+v", first)
		}
		replay := call("operation_recover", args)["recovery"].(map[string]any)["operation"].(map[string]any)
		if replay["operation_id"] != op["operation_id"] || replay["duplicate"] != true {
			t.Fatal("recovery receipt replay changed")
		}
		after, e := st.Get(ctx, original)
		if e != nil || after.State != "SUCCEEDED" || after.NativeOperationID != prior.NativeOperationID {
			t.Fatalf("original changed: %+v %v", after, e)
		}
		var n int
		if e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='UNKNOWN'", original).Scan(&n); e != nil || n < 1 {
			t.Fatalf("lost UNKNOWN history %d %v", n, e)
		}
	}
	for _, f := range []struct {
		proto, bind string
		host, guest int
	}{
		{"tcp", "127.0.0.1", 23161, 19091}, {"udp", "127.0.0.1", 23162, 19092},
		{"tcp", "fd83:248::1", 23163, 19091}, {"udp", "fd83:248::1", 23164, 19092},
	} {
		key := strconv.Itoa(f.host)
		original := call("port_forward_create", map[string]any{"box_id": uuid, "protocol": f.proto, "host_bind": f.bind, "host_port": f.host, "guest_port": f.guest, "confirm_external_exposure": f.bind != "127.0.0.1", "idempotency_key": "s5a4-fwd-create-" + key, "work_ref": "s5a4/recovery-fault"})["operation_id"].(string)
		recoverOriginal(original)
		b4MCPHostEcho(t, f.proto, f.bind, f.host, true)
		removal := call("port_forward_delete", map[string]any{"forward_id": original, "idempotency_key": "s5a4-fwd-delete-" + key, "work_ref": "s5a4/recovery-fault"})["operation_id"].(string)
		recoverOriginal(removal)
		b4MCPHostEcho(t, f.proto, f.bind, f.host, false)
	}
	fmt.Println("S5_A4_FORWARD_REAL_IPV4_IPV6_TCP_UDP_RECOVERY=PASS")
	for _, b := range bindings {
		res := call("port_forward_create", map[string]any{"box_id": uuid, "protocol": b.proto, "host_bind": "127.0.0.1", "host_port": b.host, "guest_port": b.guest, "idempotency_key": "a4-disposable-create-" + b.proto + "-001"})
		id, ok := res["operation_id"].(string)
		if !ok {
			t.Fatal(res)
		}
		ids = append(ids, id)
		b4MCPWait(t, sess, id)
		b4MCPHostEcho(t, b.proto, "127.0.0.1", b.host, true)
	}
	if _, err := db.ExecContext(ctx, "DROP TRIGGER s5a4_fault"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER s5a4_fault BEFORE UPDATE OF state ON operations WHEN OLD.state IN ('VERIFYING','DISPATCHING') AND OLD.action='box_delete' AND OLD.work_ref='s5a4/recovery-fault' AND NEW.state='SUCCEEDED' BEGIN SELECT RAISE(ABORT,'s5a4 dependent terminal commit cut'); END`); err != nil {
		t.Fatal(err)
	}
	args := map[string]any{"box_id": uuid, "idempotency_key": "a4-disposable-dependent-delete-001", "work_ref": "s5a4/recovery-fault"}
	res := call("box_delete", args)
	parent, ok := res["operation_id"].(string)
	if !ok {
		t.Fatal(res)
	}
	recoverOriginal(parent)
	if err = svc.Native.ConfirmDeleteAbsence(ctx, "s4b4native"); err != nil {
		t.Fatal(err)
	}
	for i, b := range bindings {
		b4MCPHostEcho(t, b.proto, "127.0.0.1", b.host, false)
		f, e := st.GetForward(ctx, project, ids[i])
		if e != nil || f.State != "REVOKED" {
			t.Fatalf("forward claim not revoked: %+v %v", f, e)
		}
	}
	repeat := call("box_delete", args)
	if repeat["operation_id"] != parent || repeat["duplicate"] != true {
		t.Fatalf("parent replay changed: %+v", repeat)
	}
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM owned_boxes WHERE uuid=? AND state='DELETED' AND deleted_by_operation_id=?", uuid, parent).Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("missing exact tombstone %d %v", count, err)
	}
	for _, check := range []struct {
		action string
		n      int
	}{{"port_forward_delete", 2}, {"box_stop", 1}} {
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations o JOIN operation_resource_claims c ON c.operation_id=o.id WHERE c.resource_key=? AND o.action=? AND o.state='SUCCEEDED' AND o.native_operation_id!=''", "delete-parent:"+parent, check.action).Scan(&count)
		if err != nil || count != check.n {
			t.Fatalf("missing linked native child %s: %d %v", check.action, count, err)
		}
	}
	fmt.Println("S5_A4_DEPENDENT_DELETE_NATIVE_RECOVERY=PASS")
}
