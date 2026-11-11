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

// Legacy releases can return only TextContent. Never require the newer
// SDK's StructuredContent when validating a real native lifecycle.
func successfulObject(result *mcp.CallToolResult) map[string]any {
	verify(result != nil && !result.IsError, "legacy tool returned an error")
	var obj map[string]any
	if result.StructuredContent != nil {
		data, err := json.Marshal(result.StructuredContent)
		verify(err == nil && json.Unmarshal(data, &obj) == nil,
			"legacy structured result malformed")
	}
	if obj == nil {
		for _, entry := range result.Content {
			text, ok := entry.(*mcp.TextContent)
			if ok && json.Unmarshal([]byte(text.Text), &obj) == nil {
				break
			}
		}
	}
	verify(obj != nil, "legacy tool emitted no decodable object")
	return obj
}

func legacyCall(ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) map[string]any {
	result, err := invoke(ctx, session, name, args)
	verify(err == nil, "legacy tool request failed: "+name)
	return successfulObject(result)
}

func objectField(obj map[string]any, key string) string {
	value, _ := obj[key].(string)
	return value
}

func legacyOperationID(receipt map[string]any) string {
	id := objectField(receipt, "operation_id")
	verify(len(id) == 32, "missing exact original legacy operation ID")
	return id
}

func legacyCompleted(ctx context.Context, session *mcp.ClientSession, operationID string) map[string]any {
	for i := 0; i < 90; i++ {
		got := legacyCall(ctx, session, "operation_get", map[string]any{
			"operation_id": operationID, "wait_ms": 2000,
		})
		op, _ := got["operation"].(map[string]any)
		verify(op != nil, "legacy operation_get lost nested operation")
		switch objectField(op, "state") {
		case "SUCCEEDED":
			return op
		case "FAILED", "UNKNOWN", "CANCELLED":
			verify(false, "legacy operation left exact original not successful: "+operationID)
		}
	}
	verify(false, "legacy original operation did not complete within polling budget")
	return nil
}

func legacyLifecycle(ctx context.Context, session *mcp.ClientSession) {
	// This unique fixture exists only inside an already verified single-use
	// GitHub-hosted Incus Project and is deleted through the same MCP service.
	name := "bw-s2-legacy-compat-110e"
	create := legacyCall(ctx, session, "box_create", map[string]any{
		"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": false, "internet": false,
		"idempotency_key": "s2-old-sdk-positive-create",
	})
	created := legacyCompleted(ctx, session, legacyOperationID(create))
	uuid := objectField(created, "box_uuid")
	verify(uuid != "", "legacy create did not bind an exact native UUID")
	box := legacyCall(ctx, session, "box_get", map[string]any{"box_id": uuid})
	detail, _ := box["box"].(map[string]any)
	verify(detail != nil && objectField(detail, "box_id") == uuid &&
		objectField(detail, "native_state") == "Stopped",
		"legacy client could not observe exactly owned stopped guest")

	update := legacyCall(ctx, session, "box_update", map[string]any{
		"box_id": uuid, "cpu": 2,
		"idempotency_key": "s2-old-sdk-positive-update",
	})
	legacyCompleted(ctx, session, legacyOperationID(update))
	start := legacyCall(ctx, session, "box_start", map[string]any{
		"box_id": uuid, "idempotency_key": "s2-old-sdk-positive-start",
	})
	legacyCompleted(ctx, session, legacyOperationID(start))
	running := legacyCall(ctx, session, "box_get", map[string]any{"box_id": uuid})
	current, _ := running["box"].(map[string]any)
	verify(current != nil && objectField(current, "native_state") == "Running",
		"legacy client start not reflected by exact native read")

	exec := legacyCall(ctx, session, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-c", "printf legacy-sdk-ok"},
		"idempotency_key": "s2-old-sdk-positive-exec",
	})
	execID := legacyOperationID(exec)
	legacyCompleted(ctx, session, execID)
	out := legacyCall(ctx, session, "operation_output_get", map[string]any{
		"operation_id": execID, "stream": "stdout", "offset": 0, "limit": 16384,
	})
	verify(objectField(out, "availability") == "available" &&
		objectField(out, "operation_id") == execID,
		"legacy client could not retrieve its completed recorded guest output")
	legacyCall(ctx, session, "operation_list", map[string]any{})

	stop := legacyCall(ctx, session, "box_stop", map[string]any{
		"box_id": uuid, "idempotency_key": "s2-old-sdk-positive-stop",
	})
	legacyCompleted(ctx, session, legacyOperationID(stop))
	remove := legacyCall(ctx, session, "box_delete", map[string]any{
		"box_id": uuid, "idempotency_key": "s2-old-sdk-positive-delete",
	})
	legacyCompleted(ctx, session, legacyOperationID(remove))
	readBoxes(ctx, session)
	fmt.Println("S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_NATIVE_CREATE_UPDATE_EXEC_DELETE=PASS")
}

func main() {
	verify(os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted",
		"refusing to execute outside one-use CI")
	endpoint, token := os.Getenv("BW_S2_LEGACY_MCP_URL"), os.Getenv("BW_S2_LEGACY_TOKEN")
	verify(strings.HasPrefix(endpoint, "http://127.0.0.1:") &&
		strings.HasSuffix(endpoint, "/mcp") && len(token) > 0,
		"endpoint must be local and caller must be authenticated")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
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
	legacyLifecycle(ctx, session)
}
