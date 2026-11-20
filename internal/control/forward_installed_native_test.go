package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
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
func b4MCPHostEcho(t *testing.T, protocol, host string, port int, want bool) {
	t.Helper()
	conn, err := net.DialTimeout(protocol, net.JoinHostPort(host, strconv.Itoa(port)), time.Second)
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
	svc := &Service{Store: st, Native: native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project, ExternalForwardAllow: "bwbr0/tcp/10.248.83.1:23095,bwbr0/udp/10.248.83.1:23096,bwbr0/tcp/[fd83:248::1]:23097,bwbr0/udp/[fd83:248::1]:23098"}, BuildVersion: "b4-native-mcp-test", BuildCommit: "source-exact"}
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
	if err != nil || len(inventory.Tools) != 22 {
		t.Fatalf("expected exact 22 live MCP tools: %v / %d", err, len(inventory.Tools))
	}
	// Refuse missing acknowledgment, unmatched port, wildcard, and unapproved addresses.
	for i, tc := range []struct {
		bind string
		port int
		ack  bool
	}{
		{"10.248.83.1", 23095, false}, {"10.248.83.1", 23099, true}, {"10.248.83.2", 23095, true}, {"0.0.0.0", 23095, true},
	} {
		args := map[string]any{"box_id": uuid, "protocol": "tcp", "host_bind": tc.bind, "host_port": tc.port, "guest_port": 19091, "confirm_external_exposure": tc.ack, "idempotency_key": fmt.Sprintf("b4-exposure-negative-%03d", i)}
		result, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_create", Arguments: args})
		if e == nil && (result == nil || !result.IsError) {
			t.Fatalf("unapproved external admission %v", args)
		}
	}
	fmt.Println("S4_B4_MCP_EXTERNAL_DENY_MISSING_CONSENT_ADDRESS_PORT_WILDCARD=PASS")
	for _, tc := range []struct {
		name, proto, bind   string
		hostPort, guestPort int
	}{
		{"tcp", "tcp", "127.0.0.1", 23093, 19091},
		{"udp", "udp", "127.0.0.1", 23094, 19092},
		{"external-v4-tcp", "tcp", "10.248.83.1", 23095, 19091},
		{"external-v4-udp", "udp", "10.248.83.1", 23096, 19092},
		{"external-v6-tcp", "tcp", "fd83:248::1", 23097, 19091},
		{"external-v6-udp", "udp", "fd83:248::1", 23098, 19092},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{"box_id": uuid, "protocol": tc.proto, "host_bind": tc.bind, "host_port": tc.hostPort, "guest_port": tc.guestPort, "confirm_external_exposure": tc.bind != "127.0.0.1", "idempotency_key": "b4-mcp-create-" + tc.name + "-001"}
			other := cloneB4Args(args)
			other["idempotency_key"] = "b4-mcp-create-" + tc.name + "-002"
			original, args := b4CompetingMCPCalls(t, ctx, sess, "port_forward_create", args, other)
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
			b4MCPHostEcho(t, tc.proto, tc.bind, tc.hostPort, true)
			alternate := "10.248.83.1"
			if tc.bind != "127.0.0.1" {
				alternate = "127.0.0.1"
			}
			b4MCPHostEcho(t, tc.proto, alternate, tc.hostPort, false)
			conflicting := map[string]any{"box_id": uuid, "protocol": tc.proto, "host_port": tc.hostPort,
				"guest_port": tc.guestPort, "host_bind": tc.bind, "confirm_external_exposure": tc.bind != "127.0.0.1", "idempotency_key": "s4b4-native-mcp-collision-" + tc.name + "-003"}
			denied, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_create", Arguments: conflicting})
			if e == nil && (denied == nil || !denied.IsError) {
				t.Fatal("listener collision was accepted")
			}
			delArgs := map[string]any{"forward_id": opID, "idempotency_key": "b4-mcp-delete-" + tc.name + "-004"}
			rival := cloneB4Args(delArgs)
			rival["idempotency_key"] = "b4-mcp-delete-" + tc.name + "-005"
			remove, delArgs := b4CompetingMCPCalls(t, ctx, sess, "port_forward_delete", delArgs, rival)
			removeID, ok := remove["operation_id"].(string)
			if !ok {
				t.Fatalf("no delete original %+v", remove)
			}
			b4MCPWait(t, sess, removeID)
			againDelete, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_delete", Arguments: delArgs})
			deleteReplay := b4MCPData(t, againDelete, e)
			if deleteReplay["operation_id"] != removeID || deleteReplay["duplicate"] != true {
				t.Fatalf("delete replay %+v", deleteReplay)
			}
			b4MCPHostEcho(t, tc.proto, tc.bind, tc.hostPort, false)
			listed, e = sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_list", Arguments: map[string]any{"box_id": uuid}})
			data = b4MCPData(t, listed, e)
			items, ok = data["forwards"].([]any)
			if !ok || len(items) != 0 {
				t.Fatalf("revoked forward remains listed: %+v", data)
			}
			t.Logf("S4_B4_MCP_%s_REAL_CREATION_REPLAY_CONFLICT_PACKET_DELETE=PASS", tc.name)
		})
	}
	// A real foreign host listener occupies a TCP endpoint. Incus must not
	// replace it, and an incomplete native apply cannot become CONFIRMED.
	t.Run("occupied-host-listener", func(t *testing.T) {
		const port = 23105
		guard, e := net.Listen("tcp", "127.0.0.1:23105")
		if e != nil {
			t.Fatal(e)
		}
		defer guard.Close()
		args := map[string]any{"box_id": uuid, "protocol": "tcp", "host_port": port, "guest_port": 19091, "idempotency_key": "b4-occupied-real-listener-001"}
		result, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_create", Arguments: args})
		original := b4MCPData(t, result, e)
		id, ok := original["operation_id"].(string)
		if !ok {
			t.Fatal("missing occupied-listener original")
		}
		var op state.Receipt
		until := time.Now().Add(40 * time.Second)
		for time.Now().Before(until) {
			op, e = st.Get(ctx, id)
			if e != nil {
				t.Fatal(e)
			}
			if op.State == "UNKNOWN" || op.State == "FAILED" || op.State == "SUCCEEDED" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if op.State != "UNKNOWN" && op.State != "FAILED" {
			t.Fatalf("native occupied listener misreported %+v", op)
		}
		claim, e := st.GetForward(ctx, project, id)
		if e != nil {
			t.Fatal(e)
		}
		if op.State == "UNKNOWN" {
			if claim.State != "PENDING" {
				t.Fatalf("uncertain residual claim released %+v", claim)
			}
			next := cloneB4Args(args)
			next["idempotency_key"] = "b4-occupied-real-listener-002"
			denied, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "port_forward_create", Arguments: next})
			if e == nil && (denied == nil || !denied.IsError) {
				t.Fatal("fresh occupied listener original admitted")
			}
		} else if claim.State != "REVOKED" {
			t.Fatalf("proved no dispatch not released %+v", claim)
		}
		sentinel := "original-host-listener"
		accepted := make(chan error, 1)
		go func() {
			cn, e := guard.Accept()
			if e != nil {
				accepted <- e
				return
			}
			defer cn.Close()
			_, e = cn.Write([]byte(sentinel))
			accepted <- e
		}()
		cn, e := net.DialTimeout("tcp", "127.0.0.1:23105", time.Second)
		if e != nil {
			t.Fatal(e)
		}
		_ = cn.SetDeadline(time.Now().Add(time.Second))
		buf := make([]byte, len(sentinel))
		_, e = io.ReadFull(cn, buf)
		_ = cn.Close()
		if e != nil || string(buf) != sentinel {
			t.Fatalf("foreign host listener was replaced: %q %v", buf, e)
		}
		if e = <-accepted; e != nil {
			t.Fatal(e)
		}
		_ = guard.Close()
		// One-use host teardown, not a production UNKNOWN retry. If Incus retained
		// exact residual metadata, explicitly verify and remove only that test proxy.
		after, _, e := c.UseProject(project).GetInstance("s4b4native")
		if e != nil {
			t.Fatal(e)
		}
		if _, exists := after.Devices[claim.NativeDevice]; exists {
			if op.State != "UNKNOWN" {
				t.Fatal("failed receipt released a residual native device")
			}
			spec := specForClaim(claim)
			if e = svc.Native.ObserveOwnedForward(ctx, "s4b4native", uuid, spec, true); e != nil {
				t.Fatal(e)
			}
			nativeID, e := svc.Native.UpdateOwnedForward(ctx, "s4b4native", uuid, spec, false)
			if e != nil {
				t.Fatal(e)
			}
			if e = nativeDone(ctx, svc.Native, nativeID); e != nil {
				t.Fatal(e)
			}
		}
		after, _, e = c.UseProject(project).GetInstance("s4b4native")
		if e != nil {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(inst.Devices, after.Devices) {
			t.Fatalf("unrelated native devices changed: %v", after.Devices)
		}
		b4MCPHostEcho(t, "tcp", "127.0.0.1", port, false)
		fmt.Println("S4_B4_MCP_NATIVE_OCCUPIED_HOST_LISTENER_NO_FALSE_SUCCESS=PASS")
	})
	fmt.Println("S4_B4_INSTALLED_AUTHENTICATED_22_TOOL_HTTP_MCP_ORIGINAL_SQLITE_INCUS_TCP_UDP=PASS")
}

// Two different keys simultaneously contend for the same native resource.
// Exactly one may receive an original; the losing call may not dispatch.
func cloneB4Args(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
func b4CompetingMCPCalls(t *testing.T, ctx context.Context, sess *mcp.ClientSession, name string, first, second map[string]any) (map[string]any, map[string]any) {
	t.Helper()
	type response struct {
		result *mcp.CallToolResult
		err    error
		args   map[string]any
	}
	start := make(chan struct{})
	results := make(chan response, 2)
	for _, a := range []map[string]any{first, second} {
		go func(a map[string]any) {
			<-start
			r, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: a})
			results <- response{r, e, a}
		}(a)
	}
	close(start)
	var winner response
	count := 0
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err == nil && r.result != nil && !r.result.IsError {
			winner = r
			count++
		}
	}
	if count != 1 {
		t.Fatalf("%s competing distinct idempotency keys accepted %d", name, count)
	}
	return b4MCPData(t, winner.result, winner.err), winner.args
}
