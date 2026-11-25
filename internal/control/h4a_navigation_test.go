package control

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

func h4aFailure(t *testing.T, r *mcp.CallToolResult, e error) ToolFailure {
	t.Helper()
	if e != nil || r == nil || !r.IsError || r.StructuredContent != nil || len(r.Content) != 1 {
		t.Fatalf("not a compatible tool error: %+v %v", r, e)
	}
	text, ok := r.Content[0].(*mcp.TextContent)
	var f, meta ToolFailure
	if !ok || json.Unmarshal([]byte(text.Text), &f) != nil {
		t.Fatal("missing safe error text", r.Content)
	}
	raw, e := json.Marshal(r.Meta["io.boxweave.error"])
	if e != nil || json.Unmarshal(raw, &meta) != nil || f != meta {
		t.Fatal("metadata/text mismatch", f, string(raw), e)
	}
	return f
}
func TestH4AAuthenticatedSDKNavigation(t *testing.T) {
	st, server, admin, viewer := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	info, e := st.VerifyToken(ctx, admin)
	if e != nil {
		t.Fatal(e)
	}
	connect := func(token string) *mcp.ClientSession {
		c := mcp.NewClient(&mcp.Implementation{Name: "h4a-test", Version: "1"}, nil)
		s, e := c.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp", HTTPClient: &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: token}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { s.Close() })
		return s
	}
	c, v := connect(admin), connect(viewer)
	tools, e := c.ListTools(ctx, nil)
	if e != nil || len(tools.Tools) != 23 {
		t.Fatal(e, tools)
	}
	for _, tool := range tools.Tools {
		raw, _ := json.Marshal(tool.InputSchema)
		var schema struct {
			Properties map[string]map[string]any `json:"properties"`
		}
		if e = json.Unmarshal(raw, &schema); e != nil {
			t.Fatal(e)
		}
		if key, ok := schema.Properties["idempotency_key"]; ok {
			if key["maxLength"] != float64(128) || !strings.Contains(fmt.Sprint(key["description"]), "16..128 UTF-8 bytes") || tool.Name != "operation_recover" && key["minLength"] != float64(16) {
				t.Fatal(tool.Name, key)
			}
		}
	}
	// Exact original seeded through real Admit; no native dispatch. The
	// authenticated handler must refuse before it can reach the worker.
	a := state.Admission{Layer: testScopedProject, ActorID: info.ActorID, TokenID: info.ID, Key: "h4a-original-key-0001", Action: "box_create", ResourceKey: "box:h4a-locked", SafeIntent: []byte(`{"name":"h4a-locked"}`), RequestID: "h4a-fixture"}
	original, e := st.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	call := func(s *mcp.ClientSession, tool string, args map[string]any) ToolFailure {
		r, e := s.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
		return h4aFailure(t, r, e)
	}
	args := map[string]any{"name": "h4a-locked", "idempotency_key": "h4a-refused-key-0001", "work_ref": "h4a-refused"}
	f := call(c, "box_create", args)
	if f.Code != "RESOURCE_BUSY" || f.Admission != "NOT_ADMITTED" || f.OriginalState != "ADMITTED" || f.Next.OperationID != original.ID || f.Next.Tool != "operation_get" {
		t.Fatal(f)
	}
	// Authorized different actor sees the exact same-layer blocker but
	// never obtains the other actor's key or ability to replay its identity.
	if x := call(v, "box_create", args); x.Next.OperationID != original.ID {
		t.Fatal(x)
	}
	if e = st.Claim(ctx, original.ID); e != nil {
		t.Fatal(e)
	}
	if e = st.Unknown(ctx, original.ID, "DISPATCHING", "NATIVE_ACK_DURABILITY_UNVERIFIED"); e != nil {
		t.Fatal(e)
	}
	f = call(c, "box_create", args)
	if f.Code != "UNKNOWN_BLOCKER" || f.Next.Tool != "operation_recover" || f.Next.Mode != "inspect" || f.Next.OperationID != original.ID {
		t.Fatal(f)
	}
	raw, e := c.CallTool(ctx, &mcp.CallToolParams{Name: "operation_get", Arguments: map[string]any{"operation_id": original.ID}})
	data := b4MCPData(t, raw, e)
	diag := data["diagnostic"].(map[string]any)
	nav := data["recovery_navigation"].(map[string]any)
	if diag["status_class"] != "EFFECT_UNVERIFIED" || nav["disposition"] != "INSPECT_REQUIRED" || diag["next_action"].(map[string]any)["operation_id"] != original.ID {
		t.Fatal(diag, nav)
	}
	for _, key := range []string{strings.Repeat("x", 15), strings.Repeat("x", 129), strings.Repeat("界", 43)} {
		x := call(c, "box_create", map[string]any{"name": "h4a-locked", "idempotency_key": key})
		if x.Code != "INVALID_IDEMPOTENCY_KEY" || x.Admission != "NOT_ADMITTED" || x.Next.OperationID != "" || strings.Contains(x.Error(), key) {
			t.Fatal(x)
		}
	}
	// Schema rejection must not echo the untrusted secret argument.
	x := call(c, "box_exec", map[string]any{"box_id": "missing", "idempotency_key": "h4a-invalid-input-key-01", "argv": map[string]any{"secret": "SECRET_DO_NOT_REFLECT_H4A"}})
	if x.Code != "INVALID_INPUT" || strings.Contains(x.Error(), "SECRET_DO_NOT_REFLECT_H4A") {
		t.Fatal(x)
	}
	a.Layer = "foreign-project"
	a.Key = "foreign-original-key-0001"
	a.ResourceKey = "box:foreign-private"
	foreign, e := st.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	absent := call(c, "operation_get", map[string]any{"operation_id": strings.Repeat("0", 32)})
	hidden := call(c, "operation_get", map[string]any{"operation_id": foreign.ID})
	if absent != hidden || strings.Contains(hidden.Error(), foreign.ID) {
		t.Fatal(absent, hidden)
	}
	page, e := st.ListOperations(ctx, state.OperationQuery{Layer: testScopedProject, Requester: info.ActorID})
	if e != nil || len(page.Operations) != 1 || page.Operations[0].ID != original.ID || page.Operations[0].State != "UNKNOWN" {
		t.Fatal(page, e)
	}
	t.Log("PR188_H4A_AUTHENTICATED_SDK_NAVIGATION=PASS")
}
