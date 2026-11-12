//go:build ignore

// S2 test-only official MCP SDK quota-fault end-to-end observer.
// This binary is never shipped; it runs only on a verified disposable lab.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	req := r.Clone(r.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(req)
}

func check(ok bool, msg string) {
	if !ok {
		panic("S2_MCP_QUOTA_FAIL: " + msg)
	}
}

func connect(ctx context.Context, url, token string) *mcp.ClientSession {
	c := mcp.NewClient(&mcp.Implementation{Name: "s2-one-use-mcp-quota", Version: "1"}, nil)
	s, err := c.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: url, HTTPClient: &http.Client{Transport: bearer{token}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		panic(err)
	}
	return s
}

func call(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) map[string]any {
	r, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil || r == nil || r.IsError {
		panic(fmt.Sprintf("S2_MCP_QUOTA_FAIL: %s error: %v", name, err))
	}
	raw, err := json.Marshal(r.StructuredContent)
	if err != nil {
		panic(err)
	}
	var out map[string]any
	if err = json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

func child(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	check(v != nil, "missing typed object "+key)
	return v
}

func field(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func id(m map[string]any) string {
	v := field(m, "operation_id")
	check(len(v) == 32, "missing/invalid BoxWeave original operation ID")
	return v
}

func wait(ctx context.Context, s *mcp.ClientSession, opID string) map[string]any {
	end := time.Now().Add(2 * time.Minute)
	for time.Now().Before(end) {
		op := child(call(ctx, s, "operation_get", map[string]any{
			"operation_id": opID, "wait_ms": 1000,
		}), "operation")
		switch field(op, "state") {
		case "SUCCEEDED":
			return op
		case "UNKNOWN", "FAILED", "CANCELLED":
			panic("original BoxWeave quota Operation did not verify: " + field(op, "state"))
		}
	}
	panic("original BoxWeave quota operation polling exhausted")
}

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "create" && os.Args[1] != "seed" &&
		os.Args[1] != "flood" && os.Args[1] != "missing" && os.Args[1] != "cleanup") ||
		os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" {
		panic("MCP quota test only on disposable GitHub runner")
	}
	url, token := os.Getenv("BW_S2_MCP_QUOTA_URL"), os.Getenv("BW_S2_MCP_QUOTA_TOKEN")
	uuid := os.Getenv("BW_S2_MCP_QUOTA_UUID")
	check(strings.HasPrefix(url, "http://127.0.0.1:"), "must bind loopback")
	check(strings.HasPrefix(token, "bw1_"), "must have fresh CI-only administrator token")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	s := connect(ctx, url, token)
	defer s.Close()
	const name = "bw-s2-mcp-quota-110i"
	switch os.Args[1] {
	case "create":
		create := call(ctx, s, "box_create", map[string]any{
			"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
			"start": true, "idempotency_key": "native-s2-mcp-quota-create-once-001",
		})
		op := wait(ctx, s, id(create))
		uuid = field(op, "box_uuid")
		check(len(uuid) == 36, "real MCP native create UUID missing")
		box := child(call(ctx, s, "box_get", map[string]any{"box_id": uuid}), "box")
		check(field(box, "native_state") == "Running" &&
			field(box, "root_disk_size") == "2GiB", "incorrect native quota Box")
		fmt.Println("S2_MCP_QUOTA_BOX_UUID=" + uuid)
	case "seed":
		check(len(uuid) == 36, "missing quota box UUID")
		seed := call(ctx, s, "box_exec", map[string]any{
			"box_id": uuid, "argv": []string{"/bin/sh", "-c", "printf initial"},
			"idempotency_key": "native-s2-mcp-quota-seed-output-001",
		})
		opID := id(seed)
		wait(ctx, s, opID)
		out := call(ctx, s, "operation_output_get", map[string]any{
			"operation_id": opID, "stream": "stdout", "limit": 4096,
		})
		bytes, err := base64.StdEncoding.DecodeString(field(out, "content_base64"))
		check(err == nil && string(bytes) == "initial" && out["command_success"] == true,
			"seed original recorded output not verified")
		fmt.Println("S2_MCP_QUOTA_SEED_RECORDED=PASS")
	case "flood":
		check(len(uuid) == 36, "missing quota box UUID")
		args := map[string]any{
			"box_id":          uuid,
			"argv":            []string{"/bin/sh", "-c", "head -c 134217728 /dev/zero"},
			"timeout_ms":      30000,
			"idempotency_key": "native-s2-mcp-quota-guest-flood-once-001",
		}
		request := call(ctx, s, "box_exec", args)
		originalID := id(request)
		op := wait(ctx, s, originalID)
		nativeID := field(op, "native_operation_id")
		check(len(nativeID) == 36, "verified original native Exec operation ID missing")
		check(field(op, "state") == "SUCCEEDED", "native Guest Exec did not durably settle")
		s.Close()
		s = connect(ctx, url, token)
		defer s.Close()
		out := call(ctx, s, "operation_output_get", map[string]any{
			"operation_id": originalID, "stream": "stdout", "limit": 4096,
		})
		payload, err := base64.StdEncoding.DecodeString(field(out, "content_base64"))
		check(err == nil && len(payload) == 4096, "quota original data not retrievable")
		check(out["exit_code"] == float64(1) && out["command_success"] == false,
			"native Success/200 with guest exit 1 was incorrectly reported as command success")
		tail := call(ctx, s, "operation_output_get", map[string]any{
			"operation_id": originalID, "stream": "stdout",
			"offset": 16777200, "limit": 65536,
		})
		check(tail["truncated"] == true && tail["ended"] == true &&
			tail["next_offset"] == float64(16777216) &&
			tail["command_success"] == false,
			"public 16MiB page guard or guest failure was lost under native EDQUOT")
		replay := call(ctx, s, "box_exec", args)
		check(id(replay) == originalID && replay["duplicate"] == true,
			"same key duplicated the quota-failed original guest effect")
		fmt.Println("S2_MCP_QUOTA_OPERATION_ID=" + originalID)
		fmt.Println("S2_MCP_QUOTA_NATIVE_ID=" + nativeID)
		fmt.Println("S2_MCP_QUOTA_GUEST_EXIT_ONE_PUBLIC_OUTPUT=PASS")
	case "missing":
		check(len(uuid) == 36, "missing quota box UUID")
		originalID := os.Getenv("BW_S2_MCP_QUOTA_ORIGINAL_ID")
		check(len(originalID) == 32, "exact original Exec ID for missing-log test required")
		op := child(call(ctx, s, "operation_get", map[string]any{
			"operation_id": originalID,
		}), "operation")
		check(field(op, "state") == "SUCCEEDED" &&
			len(field(op, "native_operation_id")) == 36,
			"original operation identity changed after explicit native log deletion")
		out := call(ctx, s, "operation_output_get", map[string]any{
			"operation_id": originalID, "stream": "stdout", "limit": 4096,
		})
		check(field(out, "availability") == "unavailable" &&
			field(out, "unavailable_reason") == "NATIVE_RECORDED_OUTPUT_MISSING" &&
			out["ended"] == true && field(out, "content_base64") == "" &&
			out["exit_code"] == float64(1) && out["command_success"] == false,
			"missing native file was misreported as successful, expired or available")
		recheck := call(ctx, s, "operation_output_get", map[string]any{
			"operation_id": originalID, "stream": "stdout", "limit": 4096,
		})
		check(field(recheck, "availability") == "unavailable" &&
			field(recheck, "unavailable_reason") == "NATIVE_RECORDED_OUTPUT_MISSING",
			"repeat original missing-log read mutated immutable availability evidence")
		stderr := call(ctx, s, "operation_output_get", map[string]any{
			"operation_id": originalID, "stream": "stderr", "limit": 4096,
		})
		check(field(stderr, "availability") == "available" &&
			stderr["command_success"] == false,
			"missing stdout was wrongly projected onto untouched native stderr")
		fmt.Println("S2_MCP_EXACT_ORIGINAL_NATIVE_LOG_MISSING_TYPED_UNAVAILABLE=PASS")
	case "cleanup":
		check(len(uuid) == 36, "missing quota box UUID")
		stop := call(ctx, s, "box_stop", map[string]any{
			"box_id": uuid, "idempotency_key": "native-s2-mcp-quota-stop-after-filler-001",
		})
		wait(ctx, s, id(stop))
		del := call(ctx, s, "box_delete", map[string]any{
			"box_id": uuid, "idempotency_key": "native-s2-mcp-quota-delete-after-stop-01",
		})
		wait(ctx, s, id(del))
		boxes := call(ctx, s, "box_list", map[string]any{"limit": 20})
		remaining, ok := boxes["boxes"].([]any)
		check(ok && len(remaining) == 0, "quota test Box remained after deletion")
		fmt.Println("S2_MCP_QUOTA_EXACT_BOX_DELETED=PASS")
	}
}
