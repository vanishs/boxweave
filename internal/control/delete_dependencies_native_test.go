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

// Runs last in the existing one-use B4 native fixture. The native Box
// is fixture-seeded, but all forwards and the final Delete use HTTP MCP.
func TestB5NativeMCPForwardDependentDelete(t *testing.T) {
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
	path := filepath.Join(private, "b5.sqlite")
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
	svc := &Service{Store: st, Native: native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}, BuildVersion: "b5-native-mcp-test", BuildCommit: "source-exact"}
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
	for _, b := range bindings {
		res := call("port_forward_create", map[string]any{"box_id": uuid, "protocol": b.proto, "host_bind": "127.0.0.1", "host_port": b.host, "guest_port": b.guest, "idempotency_key": "b5-disposable-create-" + b.proto + "-001"})
		id, ok := res["operation_id"].(string)
		if !ok {
			t.Fatal(res)
		}
		ids = append(ids, id)
		b4MCPWait(t, sess, id)
		b4MCPHostEcho(t, b.proto, "127.0.0.1", b.host, true)
	}
	args := map[string]any{"box_id": uuid, "idempotency_key": "b5-disposable-dependent-delete-001"}
	res := call("box_delete", args)
	parent, ok := res["operation_id"].(string)
	if !ok {
		t.Fatal(res)
	}
	b4MCPWait(t, sess, parent)
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
	fmt.Println("S4_B5_NATIVE_MCP_FORWARD_DEPENDENT_DELETE=PASS")
}
