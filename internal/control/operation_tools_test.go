package control

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

func TestS2MCPDurableOperationReadAndActorIsolation(t *testing.T) {
	st, httpServer, admin, viewer := fixture(t)
	ctx := context.Background()
	adminInfo, err := st.VerifyToken(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := st.Admit(ctx, state.Admission{
		Layer: "bw-test", ActorID: adminInfo.ID, TokenID: adminInfo.ID,
		Key: "mcp-s2-read-original-key-01", RequestID: "original",
		Action: "box_create", SafeIntent: []byte(`{"name":"safe-test"}`),
		WorkRef: "acceptance/session-1", AgentRef: "first-agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	connect := func(token string) (*mcp.ClientSession, context.CancelFunc) {
		t.Helper()
		requestContext, cancel := context.WithTimeout(ctx, 10*time.Second)
		client := mcp.NewClient(&mcp.Implementation{Name: "s2-reads", Version: "1"}, nil)
		session, err := client.Connect(requestContext, &mcp.StreamableClientTransport{
			Endpoint: httpServer.URL + "/mcp",
			HTTPClient: &http.Client{Transport: bearerTransport{
				base: http.DefaultTransport, token: token,
			}},
			DisableStandaloneSSE: true, MaxRetries: -1,
		}, nil)
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		return session, cancel
	}
	session, cancel := connect(admin)
	defer cancel()
	defer session.Close()
	listing, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	reads := map[string]bool{"host_status": true, "box_list": true, "box_get": true,
		"operation_get": true, "operation_list": true, "operation_output_get": true}
	for _, tool := range listing.Tools {
		if tool.Annotations == nil || tool.Annotations.ReadOnlyHint != reads[tool.Name] {
			t.Fatalf("incorrect side-effect annotation: %s", tool.Name)
		}
	}
	got, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_get", Arguments: map[string]any{"operation_id": receipt.ID, "wait_ms": 0},
	})
	if err != nil || got.IsError {
		t.Fatalf("original MCP operation_get: %+v %v", got, err)
	}
	encoded, err := json.Marshal(got.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), receipt.ID) || !strings.Contains(string(encoded), receipt.TrackingID) {
		t.Fatalf("MCP operation_get lost original IDs: %s", encoded)
	}
	got, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_list", Arguments: map[string]any{"work_ref": "acceptance/session-1", "limit": 1},
	})
	if err != nil || got.IsError {
		t.Fatalf("original MCP operation_list: %+v %v", got, err)
	}
	encoded, err = json.Marshal(got.StructuredContent)
	if err != nil || !strings.Contains(string(encoded), receipt.ID) {
		t.Fatalf("MCP operation_list lost original receipt: %s %v", encoded, err)
	}
	// A different authenticated token cannot borrow authority using the same
	// self-declared work or Agent label.
	viewSession, viewCancel := connect(viewer)
	defer viewCancel()
	defer viewSession.Close()
	hidden, err := viewSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_list", Arguments: map[string]any{"work_ref": "acceptance/session-1"},
	})
	if err != nil || hidden.IsError {
		t.Fatalf("scoped viewer list should return empty page: %+v %v", hidden, err)
	}
	encoded, err = json.Marshal(hidden.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), receipt.ID) {
		t.Fatalf("work_ref leaked original operation across actors: %s", encoded)
	}
	denied, err := viewSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_get", Arguments: map[string]any{"operation_id": receipt.ID},
	})
	if err == nil && denied != nil && !denied.IsError {
		t.Fatalf("viewer guessed original operation id successfully: %+v", denied)
	}
	invalidWait, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_get", Arguments: map[string]any{"operation_id": receipt.ID, "wait_ms": 20001},
	})
	if err == nil && invalidWait != nil && !invalidWait.IsError {
		t.Fatal("unbounded operation wait was accepted")
	}
	// A read never claims or dispatches the original admission.
	unchanged, err := st.Get(ctx, receipt.ID)
	if err != nil || unchanged.State != "ADMITTED" {
		t.Fatalf("MCP read produced a side effect: %+v %v", unchanged, err)
	}
}
