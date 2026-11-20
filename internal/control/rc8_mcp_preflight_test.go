package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

// RC8 regression: advertised schemas must not invite invalid native
// effects. The official SDK must reject them before any mutation admission.
func TestRC8AgentVisibleSchemasAndSafePreflight(t *testing.T) {
	_, server, admin, _ := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "rc8-mcp-preflight", Version: "1"}, nil)
	sess, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: server.URL + "/mcp", HTTPClient: &http.Client{Transport: bearerTransport{
			base: http.DefaultTransport, token: admin,
		}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	list, err := sess.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]map[string]any{}
	for _, tool := range list.Tools {
		b, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(b, &schema); err != nil {
			t.Fatal(err)
		}
		byName[tool.Name] = schema
	}
	property := func(tool, name string) map[string]any {
		t.Helper()
		schema := byName[tool]
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("missing %s properties", tool)
		}
		p, ok := props[name].(map[string]any)
		if !ok {
			t.Fatalf("missing %s.%s", tool, name)
		}
		return p
	}
	for _, tc := range []struct {
		tool, field string
		want        []string
	}{
		{"box_file_apply", "mode", []string{"create"}},
		{"operation_output_get", "stream", []string{"stdout", "stderr"}},
	} {
		p := property(tc.tool, tc.field)
		values, ok := p["enum"].([]any)
		if !ok || len(values) != len(tc.want) {
			t.Fatalf("%s.%s missing exact enum: %v", tc.tool, tc.field, p)
		}
		for i, v := range tc.want {
			if values[i] != v {
				t.Fatalf("%s.%s enum: got %v want %v", tc.tool, tc.field, values, tc.want)
			}
		}
	}
	argv := property("box_exec", "argv")
	if argv["type"] != "array" || argv["minItems"] != float64(1) {
		t.Fatalf("box_exec.argv nullable/empty in advertised schema: %v", argv)
	}
	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{"box_file_apply", map[string]any{"box_id": "not-owned", "path": "/tmp/demo", "mode": "replace", "idempotency_key": "rc8-replace-denied"}},
		{"operation_output_get", map[string]any{"operation_id": "00000000000000000000000000000000", "stream": "both"}},
		{"box_exec", map[string]any{"box_id": "not-owned", "argv": nil, "idempotency_key": "rc8-null-argv-denied"}},
		{"box_exec", map[string]any{"box_id": "not-owned", "argv": []string{}, "idempotency_key": "rc8-empty-argv-denied"}},
	} {
		r, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: tc.name, Arguments: tc.args})
		if err == nil && (r == nil || !r.IsError) {
			t.Fatalf("%s accepted invalid typed args: %+v", tc.name, r)
		}
	}
	// Out-of-range resources fail before Admit, without a new Operation.
	invalidUpdate, invalidErr := sess.CallTool(ctx, &mcp.CallToolParams{
		Name: "box_update", Arguments: map[string]any{
			"box_id": "00000000-0000-4000-8000-000000000001",
			"idempotency_key": "rc8-bounds-pre-admit",
			"work_ref": "rc8-bounds-not-admitted", "cpu": -1,
		},
	})
	if invalidErr == nil && (invalidUpdate == nil || !invalidUpdate.IsError) {
		t.Fatalf("invalid resource update reached admission: %+v", invalidUpdate)
	}
	query, queryErr := sess.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_list", Arguments: map[string]any{"work_ref": "rc8-bounds-not-admitted"},
	})
	if queryErr != nil || query == nil || query.IsError {
		t.Fatalf("cannot inspect rejected update audit: %+v, %v", query, queryErr)
	}
	for _, part := range query.Content {
		if v, ok := part.(*mcp.TextContent); ok && strings.Contains(v.Text, "box_update") {
			t.Fatalf("invalid dimensions produced a durable mutation: %s", v.Text)
		}
	}
	// Unknown peer IDs and incomplete PINNED network audit stay fail-closed,
	// but the public error must give an Agent the correct preflight category.
	peer, peerErr := sess.CallTool(ctx, &mcp.CallToolParams{
		Name: "box_peer_set", Arguments: map[string]any{
			"source_box_id": "00000000-0000-4000-8000-000000000001",
			"target_box_id": "00000000-0000-4000-8000-000000000002",
			"allow": false, "idempotency_key": "rc8-peer-unknown-scope",
		},
	})
	peerMessage := fmt.Sprint(peerErr)
	if peer != nil {
		for _, part := range peer.Content {
			if v, ok := part.(*mcp.TextContent); ok {
				peerMessage += v.Text
			}
		}
	}
	if !strings.Contains(peerMessage, "PEER_SCOPE_UNVERIFIED_OR_LOCKED") {
		t.Fatalf("unsafe peer preflight not classified: %s", peerMessage)
	}
	// The caller sees the target type, without revealing whether a foreign
	// Box exists in a different layer.
	got, err := sess.CallTool(ctx, &mcp.CallToolParams{
		Name: "box_get", Arguments: map[string]any{"box_id": "00000000-0000-4000-8000-000000000001"},
	})
	message := fmt.Sprint(err)
	if got != nil {
		for _, c := range got.Content {
			if text, ok := c.(*mcp.TextContent); ok {
				message += text.Text
			}
		}
	}
	if !strings.Contains(message, "BOX_NOT_FOUND_OR_NOT_OWNED") {
		t.Fatalf("missing Box ID still presented as an Operation error: %s", message)
	}
}

func TestRC8PeerScopeRefusalPreservesFailClosedClass(t *testing.T) {
	got := peerScopeRefusal(state.ErrResourceLocked)
	if !errors.Is(got, state.ErrResourceLocked) {
		t.Fatalf("peer refused status class lost: %v", got)
	}
	for _, marker := range []string{"PEER_SCOPE_UNVERIFIED_OR_LOCKED", "no peer operation was admitted"} {
		if !strings.Contains(got.Error(), marker) {
			t.Fatalf("peer refusal lacks safe guidance %q: %v", marker, got)
		}
	}
	if got := peerScopeRefusal(state.ErrInvalid); !errors.Is(got, state.ErrInvalid) {
		t.Fatalf("invalid arguments were misclassified as peer-scope lock: %v", got)
	}
}
