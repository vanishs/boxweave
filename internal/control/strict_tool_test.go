package control

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SDK schema and server runtime must agree: callers cannot request an
// unsupported Project or root pool and have that intent silently omitted.
func TestS2AllPublicToolSchemasRejectUnknownArguments(t *testing.T) {
	_, server, admin, _ := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "s2-strict-schema", Version: "1"}, nil)
	s, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: server.URL + "/mcp",
		HTTPClient: &http.Client{Transport: bearerTransport{
			base: http.DefaultTransport, token: admin,
		}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tools, err := s.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 13 {
		t.Fatalf("all 13 S1+S2 typed tools must be discoverable: %v", err)
	}
	for _, tool := range tools.Tools {
		raw, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		schema := string(raw)
		if !strings.Contains(schema, "\"additionalProperties\":false") {
			t.Fatalf("public tool %s is open to silently discarded intent: %s", tool.Name, schema)
		}
	}
	for _, name := range []string{"host_status", "box_list"} {
		ok, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: map[string]any{}})
		if err != nil || ok == nil || ok.IsError {
			t.Fatalf("valid empty request for %s now rejected: %+v %v", name, ok, err)
		}
		denied, err := s.CallTool(ctx, &mcp.CallToolParams{
			Name:      name,
			Arguments: map[string]any{"native_project": "foreign-incus-project"},
		})
		if err == nil && (denied == nil || !denied.IsError) {
			t.Fatalf("unknown native Project silently ignored by %s: %+v", name, denied)
		}
	}
	denied, err := s.CallTool(ctx, &mcp.CallToolParams{
		Name: "box_create",
		Arguments: map[string]any{
			"name":            "bw-unknown-storage-pool",
			"idempotency_key": "s2-strict-schema-unsupported-root-pool",
			"cpu":             1, "memory_mib": 512, "disk_gib": 2,
			"start": false, "root_pool": "foreign-root-pool",
		},
	})
	if err == nil && (denied == nil || !denied.IsError) {
		t.Fatalf("box_create ignored explicit unsupported root_pool: %+v", denied)
	}
}

// Nested JSON model errors must fail in the official SDK or the typed Tool
// decoder, before any native handler can interpret them as a safe default.
func TestS2NestedModelErrorsNeverBecomeNativeIntent(t *testing.T) {
	_, server, admin, _ := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cli := mcp.NewClient(&mcp.Implementation{Name: "s2-nested-model-negative", Version: "1"}, nil)
	s, err := cli.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp",
		HTTPClient:           &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: admin}},
		DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, trial := range []struct {
		name, label string
		args        map[string]any
	}{
		{"box_create", "nested_start", map[string]any{"name": "bw-s2-model-invalid", "idempotency_key": "s2-model-invalid-create", "start": map[string]any{"value": true}}},
		{"box_update", "nested_cpu", map[string]any{"box_id": "not-owned", "idempotency_key": "s2-model-invalid-update", "cpu": map[string]any{"value": 1}}},
		{"box_exec", "nested_argv", map[string]any{"box_id": "not-owned", "idempotency_key": "s2-model-invalid-exec", "argv": []any{map[string]any{"shell": "sh"}}}},
		{"box_exec", "nested_env_value", map[string]any{"box_id": "not-owned", "idempotency_key": "s2-model-invalid-env", "argv": []string{"true"}, "env": map[string]any{"SAFE_KEY": map[string]any{"value": "secret"}}}},
		{"operation_list", "nested_since_ns", map[string]any{"since_ns": map[string]any{"value": 100}}},
		{"operation_get", "nested_wait_ms", map[string]any{"operation_id": "nonexistent", "wait_ms": map[string]any{"value": 15}}},
		{"operation_output_get", "nested_offset", map[string]any{"operation_id": "nonexistent", "stream": "stdout", "offset": map[string]any{"value": 1}}},
		{"box_get", "array_uuid", map[string]any{"box_id": []any{"wrong"}}},
	} {
		t.Run(trial.label, func(t *testing.T) {
			r, err := s.CallTool(ctx, &mcp.CallToolParams{Name: trial.name, Arguments: trial.args})
			if err == nil && (r == nil || !r.IsError) {
				t.Fatalf("nested invalid model accepted %s: %+v", trial.name, r)
			}
		})
	}
}
