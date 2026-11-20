package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
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

func b4MCPData(t *testing.T, result *mcp.CallToolResult, err error) map[string]any {
	t.Helper()
	if err != nil || result == nil || result.IsError {
		t.Fatalf("actual MCP call refused: result=%+v err=%v", result, err)
	}
	raw, e := json.Marshal(result.StructuredContent)
	if e != nil {
		t.Fatal(e)
	}
	var m map[string]any
	if e = json.Unmarshal(raw, &m); e != nil {
		t.Fatal(e)
	}
	return m
}
func b4MCPWait(t *testing.T, sess *mcp.ClientSession, id string) {
	t.Helper()
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		r, e := sess.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "operation_get", Arguments: map[string]any{"operation_id": id, "wait_ms": 2000},
		})
		data := b4MCPData(t, r, e)
		op, ok := data["operation"].(map[string]any)
		if !ok {
			t.Fatalf("lost durable operation: %+v", data)
		}
		switch op["state"] {
		case "SUCCEEDED":
			return
		case "FAILED", "UNKNOWN", "CANCELLED":
			t.Fatalf("native operation did not pass: %+v", op)
		}
	}
	t.Fatalf("original %s did not settle within test budget", id)
}
func b4MCPHostEcho(t *testing.T, protocol string, port int, want bool) {
	t.Helper()
	conn, err := net.DialTimeout(protocol, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err != nil {
		if want {
			t.Fatal(err)
		}
		return
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	msg := []byte("b4-mcp-public-echo")
	_, sendErr := conn.Write(msg)
	got := make([]byte, len(msg))
	n, receiveErr := conn.Read(got)
	ok := sendErr == nil && receiveErr == nil && string(got[:n]) == string(msg)
	if want && !ok {
		t.Fatalf("MCP forwarding not verified: send=%v read=%v payload=%q", sendErr, receiveErr, got[:n])
	}
	if !want && ok {
		t.Fatal("deleted MCP forward still sent traffic")
	}
}

// A real authenticated SDK -> MCP Controller -> durable SQLite/Worker ->
// nonroot Incus 7.6 instance proxy -> host TCP/UDP packet test. The owned-box
// row is explicitly TEST-SEEDED for this disposable native fixture; this does
// not claim to reaccept the independently signed S4-A Box Create path.
func TestB4InstalledMCPNativeCreateListDelete(t *testing.T) {
	if os.Getenv("BOXWEAVE_B4_MCP_ONE_USE") != "s4b4-disposable-only-20261119" ||
		os.Getenv("GITHUB_ACTIONS") != "true" {
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
	// The Go testing temp root may be shared in the runner; Store.Open
	// intentionally requires the DIRECT SQLite parent to be 0700.
	private := filepath.Join(t.TempDir(), "controller-private")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(private, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(private, "b4.sqlite")
	st, err := state.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	testOnly, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = testOnly.ExecContext(ctx, "INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,?,?)", uuid, project, "s4b4native", uuid, "OWNED", time.Now().UnixNano())
	_ = testOnly.Close()
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := st.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{Store: st, Native: native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}, BuildVersion: "b4-native-mcp-test", BuildCommit: "source-exact"}
	srv := httptest.NewUnstartedServer(svc.Handler())
	svc.Listener = srv.Listener.Addr().String()
	srv.Start()
	defer srv.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "s4-b4-native-client", Version: "1"}, nil)
	sess, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: srv.URL + "/mcp", HTTPClient: &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: token}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	inventory, err := sess.ListTools(ctx, nil)
	if err != nil || len(inventory.Tools) != 21 {
		t.Fatalf("expected exact 21 live MCP tools: %v / %d", err, len(inventory.Tools))
	}
	for _, tc := range []struct {
		proto               string
		hostPort, guestPort int
	}{
		{"tcp", 23093, 19091}, {"udp", 23094, 19092},
	} {
		t.Run(tc.proto, func(t *testing.T) {
			args := map[string]any{"box_id": uuid, "protocol": tc.proto, "host_port": tc.hostPort,
				"guest_port": tc.guestPort, "idempotency_key": "s4b4-native-mcp-create-" + tc.proto + "-001"}
			first, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_create", Arguments: args})
			original := b4MCPData(t, first, e)
			opID, ok := original["operation_id"].(string)
			if !ok || len(opID) != 32 {
				t.Fatalf("no original forward receipt: %+v", original)
			}
			b4MCPWait(t, sess, opID)
			again, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_create", Arguments: args})
			replay := b4MCPData(t, again, e)
			if replay["operation_id"] != opID || replay["duplicate"] != true {
				t.Fatalf("new native effect on identical retry: %+v", replay)
			}
			listed, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_list", Arguments: map[string]any{"box_id": uuid}})
			data := b4MCPData(t, listed, e)
			items, ok := data["forwards"].([]any)
			if !ok || len(items) != 1 {
				t.Fatalf("MCP list lacks current forward: %+v", data)
			}
			item, ok := items[0].(map[string]any)
			if !ok || item["forward_id"] != opID || item["state"] != "CONFIRMED" || item["native_verified"] != true {
				t.Fatalf("not native verified: %+v", items[0])
			}
			b4MCPHostEcho(t, tc.proto, tc.hostPort, true)
			conflicting := map[string]any{"box_id": uuid, "protocol": tc.proto, "host_port": tc.hostPort,
				"guest_port": tc.guestPort, "idempotency_key": "s4b4-native-mcp-collision-" + tc.proto + "-002"}
			denied, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_create", Arguments: conflicting})
			if e == nil && (denied == nil || !denied.IsError) {
				t.Fatal("listener collision was accepted")
			}
			deleted, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_delete", Arguments: map[string]any{
				"forward_id": opID, "idempotency_key": "s4b4-native-mcp-delete-" + tc.proto + "-003",
			}})
			remove := b4MCPData(t, deleted, e)
			removeID, ok := remove["operation_id"].(string)
			if !ok {
				t.Fatalf("no delete original %+v", remove)
			}
			b4MCPWait(t, sess, removeID)
			b4MCPHostEcho(t, tc.proto, tc.hostPort, false)
			listed, e = sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_list", Arguments: map[string]any{"box_id": uuid}})
			data = b4MCPData(t, listed, e)
			items, ok = data["forwards"].([]any)
			if !ok || len(items) != 0 {
				t.Fatalf("revoked forward remains listed: %+v", data)
			}
			t.Logf("S4_B4_MCP_%s_REAL_CREATION_REPLAY_CONFLICT_PACKET_DELETE=PASS", tc.proto)
		})
	}
	fmt.Println("S4_B4_INSTALLED_AUTHENTICATED_21_TOOL_HTTP_MCP_ORIGINAL_SQLITE_INCUS_TCP_UDP=PASS")
}
