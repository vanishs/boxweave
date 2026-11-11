//go:build ignore

// An independently compiled, pinned pre-2026-07-28 official MCP SDK client.
// Run only against the already-started disposable S2 CI service. This
// intentionally never submits a valid native mutation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type authenticatedTransport struct{ token string }

func (t authenticatedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	copy.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(copy)
}

func verify(ok bool, description string) {
	if !ok {
		fmt.Fprintln(os.Stderr, "S2_OLDER_SDK_FAIL:", description)
		os.Exit(45)
	}
}

func invoke(ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) (*mcp.CallToolResult, error) {
	return session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
}

func readBoxes(ctx context.Context, session *mcp.ClientSession) {
	result, err := invoke(ctx, session, "box_list", map[string]any{})
	verify(err == nil && result != nil && !result.IsError, "old SDK cannot read box_list")
	var value map[string]any
	if result.StructuredContent != nil {
		data, e := json.Marshal(result.StructuredContent)
		verify(e == nil && json.Unmarshal(data, &value) == nil, "malformed structured box_list")
	}
	if value == nil {
		for _, entry := range result.Content {
			text, ok := entry.(*mcp.TextContent)
			if ok && json.Unmarshal([]byte(text.Text), &value) == nil {
				break
			}
		}
	}
	verify(value != nil, "neither structured nor legacy text box_list is readable")
	boxes, ok := value["boxes"].([]any)
	verify(ok && len(boxes) == 0, "unexpected owned Box before or after negative legacy calls")
}

func main() {
	verify(os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted",
		"refusing to execute outside one-use CI")
	endpoint, token := os.Getenv("BW_S2_LEGACY_MCP_URL"), os.Getenv("BW_S2_LEGACY_TOKEN")
	verify(strings.HasPrefix(endpoint, "http://127.0.0.1:") &&
		strings.HasSuffix(endpoint, "/mcp") && len(token) > 0,
		"endpoint must be local and caller must be authenticated")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "boxweave-s2-pinned-old-sdk", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Transport: authenticatedTransport{token: token},
			Timeout:   20 * time.Second,
		},
	}, nil)
	verify(err == nil && session != nil, "old SDK could not negotiate MCP HTTP connection")
	defer session.Close()

	list, err := session.ListTools(ctx, nil)
	verify(err == nil && list != nil, "old SDK tools/list failed")
	known := map[string]bool{
		"host_status": true, "box_create": true, "box_list": true, "box_get": true,
		"box_update": true, "box_start": true, "box_stop": true, "box_delete": true,
		"box_exec": true, "operation_get": true, "operation_list": true,
		"operation_output_get": true, "operation_cancel": true,
	}
	verify(len(list.Tools) == len(known), "old SDK did not discover all 13 public tools")
	for _, tool := range list.Tools {
		verify(known[tool.Name], "unknown tool disclosed by old SDK: "+tool.Name)
		delete(known, tool.Name)
	}
	verify(len(known) == 0, "missing one or more public tools in legacy discovery")
	for _, name := range []string{"host_status", "operation_list"} {
		result, callErr := invoke(ctx, session, name, map[string]any{})
		verify(callErr == nil && result != nil && !result.IsError,
			"valid legacy read failed: "+name)
		verify(result.StructuredContent != nil || len(result.Content) > 0,
			"valid legacy read missing both result formats: "+name)
	}
	readBoxes(ctx, session)

	// Every negative uses only a nonexistent owned identity or a locally
	// invalid name; even if schema validation regresses, no native Box is
	// created, changed or deleted. All twelve S2 tools are exercised.
	absent := "00000000000000000000000000000000"
	negative := []struct {
		name string
		args map[string]any
	}{
		{"box_create", map[string]any{"name": "INVALID NAME", "idempotency_key": "s2-old-sdk-invalid-create"}},
		{"box_get", map[string]any{"box_id": absent}},
		{"box_update", map[string]any{"box_id": absent, "cpu": 2, "idempotency_key": "s2-old-sdk-invalid-update"}},
		{"box_start", map[string]any{"box_id": absent, "idempotency_key": "s2-old-sdk-invalid-start"}},
		{"box_stop", map[string]any{"box_id": absent, "idempotency_key": "s2-old-sdk-invalid-stop"}},
		{"box_delete", map[string]any{"box_id": absent, "idempotency_key": "s2-old-sdk-invalid-delete"}},
		{"box_exec", map[string]any{"box_id": absent, "argv": []string{"/bin/true"}, "idempotency_key": "s2-old-sdk-invalid-exec"}},
		{"operation_get", map[string]any{"operation_id": absent}},
		{"operation_output_get", map[string]any{"operation_id": absent, "stream": "stdout"}},
		{"operation_cancel", map[string]any{"target_operation_id": absent, "idempotency_key": "s2-old-sdk-invalid-cancel"}},
	}
	for _, sample := range negative {
		result, callErr := invoke(ctx, session, sample.name, sample.args)
		verify(callErr != nil || result != nil && result.IsError,
			"old SDK negative unexpectedly succeeded: "+sample.name)
		// Verify rejection was not simply a disconnected transport.
		check, checkErr := invoke(ctx, session, "host_status", map[string]any{})
		verify(checkErr == nil && check != nil && !check.IsError,
			"old SDK transport disconnected after "+sample.name)
	}
	readBoxes(ctx, session)
	fmt.Println("S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_LEGACY_PROTOCOL_13_TOOLS_NO_NATIVE_EFFECT=PASS")
}
