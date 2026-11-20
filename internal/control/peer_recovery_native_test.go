package control

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// Only the dedicated hosted native PRS fixtures can invoke this test. The
// original was really executed (or cut) by the installed root worker.
func TestS5A4NativeMCPPeerRecovery(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("S5A4_PEER_ONE_USE") != "s5a4-hosted-peer-only" {
		t.Skip("one-use hosted PRS fixture only")
	}
	mode, id := os.Getenv("S5A4_PEER_MODE"), os.Getenv("S5A4_ORIGINAL")
	if mode != "settled" && mode != "blocked" {
		t.Fatal("bad mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	project := "user-" + strconv.Itoa(os.Getuid())
	st, e := state.Open("/var/lib/boxweave/ledger.sqlite")
	if e != nil {
		t.Fatal(e)
	}
	defer st.Close()
	prior, e := st.Get(ctx, id)
	if e != nil || prior.State != "UNKNOWN" || prior.Action != "box_peer_set" {
		t.Fatalf("unexpected original %+v %v", prior, e)
	}
	token, _, e := st.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	svc := &Service{Store: st, Native: native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}, BuildVersion: "a4-native-peer", BuildCommit: "source-exact"}
	server := httptest.NewUnstartedServer(svc.Handler())
	svc.Listener = server.Listener.Addr().String()
	server.Start()
	defer server.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "s5a4-native-peer-client", Version: "1"}, nil)
	sess, e := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp", HTTPClient: &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: token}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer sess.Close()
	call := func(args map[string]any) map[string]any {
		r, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "operation_recover", Arguments: args})
		return b4MCPData(t, r, e)
	}
	before := call(map[string]any{"operation_id": id, "mode": "inspect"})["diagnosis"].(map[string]any)
	want, ws, diag := "SETTLED", "SUCCEEDED", "SAFE_RECOVER"
	if mode == "blocked" {
		want, ws, diag = "BLOCKED", "UNKNOWN", "BLOCKED"
	}
	if before["disposition"] != diag {
		t.Fatalf("unexpected diagnosis %+v", before)
	}
	args := map[string]any{"operation_id": id, "mode": "reconcile", "idempotency_key": "s5a4-peer-recover-" + id}
	result := call(args)
	rec := result["recovery"].(map[string]any)
	op := rec["operation"].(map[string]any)
	if rec["disposition"] != want || rec["original_operation_id"] != id || op["operation_id"] == id || result["native_retried"] != false {
		t.Fatalf("unexpected receipt %+v", result)
	}
	replay := call(args)["recovery"].(map[string]any)["operation"].(map[string]any)
	if replay["operation_id"] != op["operation_id"] || replay["duplicate"] != true {
		t.Fatal("replay receipt mismatch")
	}
	after, e := st.Get(ctx, id)
	if e != nil || after.State != ws || after.NativeOperationID != prior.NativeOperationID {
		t.Fatalf("original changed unexpectedly %+v %v", after, e)
	}
	t.Logf("S5_A4_PEER_REAL_PRS_%s_RECOVERY=PASS", mode)
}
