//go:build ignore

// Official MCP SDK black-box calls against installed non-root BoxWeave.
// This test alone issues and revokes its own disposable credentials.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

type bearer struct{ token string }

func (t bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(cp)
}
func check(ok bool, reason string) {
	if !ok {
		panic("S2_P1_PRIVATE_INFLIGHT_FAIL: " + reason)
	}
}
func connected(ctx context.Context, token string) *mcp.ClientSession {
	c := mcp.NewClient(&mcp.Implementation{Name: "s2-private-inflight-sdk", Version: "1"}, nil)
	s, e := c.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             "http://127.0.0.1:18662/mcp",
		HTTPClient:           &http.Client{Transport: bearer{token}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	check(e == nil && s != nil, "authenticated MCP connection unavailable")
	return s
}
func tool(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) map[string]any {
	result, e := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	check(e == nil && result != nil && !result.IsError, "positive native MCP call failed: "+name)
	raw, e := json.Marshal(result.StructuredContent)
	check(e == nil, "invalid result serialization")
	var out map[string]any
	check(json.Unmarshal(raw, &out) == nil && out != nil, "missing MCP object")
	return out
}
func value(m map[string]any, k string) string { v, _ := m[k].(string); return v }
func operation(ctx context.Context, s *mcp.ClientSession, receipt map[string]any) map[string]any {
	id := value(receipt, "operation_id")
	check(len(id) == 32, "native action has no original operation ID")
	for i := 0; i < 90; i++ {
		out := tool(ctx, s, "operation_get", map[string]any{"operation_id": id, "wait_ms": 2000})
		op, ok := out["operation"].(map[string]any)
		check(ok, "operation_get missing original")
		switch value(op, "state") {
		case "SUCCEEDED":
			return op
		case "FAILED", "UNKNOWN", "CANCELLED":
			panic("S2_P1_PRIVATE_INFLIGHT_FAIL: original not safely successful: " + id)
		}
	}
	panic("S2_P1_PRIVATE_INFLIGHT_FAIL: original timed out")
}

type original struct {
	UUID   string `json:"uuid"`
	ExecID string `json:"exec_id"`
}

func saved(dir string) original {
	data, e := os.ReadFile(filepath.Join(dir, "original.json"))
	check(e == nil, "original proof not found")
	var o original
	check(json.Unmarshal(data, &o) == nil && len(o.UUID) == 36 && len(o.ExecID) == 32,
		"original proof malformed")
	return o
}
func seed(ctx context.Context, dir, token string) {
	s := connected(ctx, token)
	defer s.Close()
	name := "bw-s2-read-revoke-110r"
	r := tool(ctx, s, "box_create", map[string]any{
		"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": true, "internet": false, "idempotency_key": "s2-private-read-create",
	})
	created := operation(ctx, s, r)
	uuid := value(created, "box_uuid")
	check(len(uuid) == 36, "native create did not bind UUID")
	exec := tool(ctx, s, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-c", "printf private-read-sentinel-110r"},
		"idempotency_key": "s2-private-read-exec",
	})
	execID := value(exec, "operation_id")
	operation(ctx, s, exec)
	output := tool(ctx, s, "operation_output_get", map[string]any{
		"operation_id": execID, "stream": "stdout", "offset": 0, "limit": 512,
	})
	check(value(output, "availability") == "available" && value(output, "content_base64") != "",
		"original stdout unreadable")
	data, e := json.Marshal(original{UUID: uuid, ExecID: execID})
	check(e == nil && os.WriteFile(filepath.Join(dir, "original.json"), data, 0600) == nil,
		"cannot persist test-only exact UUID")
	fmt.Println("S2_P1_PRIVATE_READ_ORIGINAL_NATIVE_BOX_EXEC_SEEDED=PASS")
}
func present(path string) bool { _, e := os.Stat(path); return e == nil }

type response struct {
	result *mcp.CallToolResult
	err    error
}

func invalidate(ctx context.Context, st *state.Store, dir, method string, args map[string]any) {
	raw, t, e := st.IssueToken(ctx, "admin", time.Hour)
	check(e == nil && t.ID != "" && raw != "", "cannot create fresh per-read credential")
	s := connected(ctx, raw)
	defer s.Close()
	arm := filepath.Join(dir, "arm-"+method)
	held := filepath.Join(dir, "held-"+method)
	release := filepath.Join(dir, "release-"+method)
	check(!present(held) && !present(release), "old test hold/release file present")
	check(os.WriteFile(arm, []byte("arm\n"), 0600) == nil, "cannot arm native read hold")
	done := make(chan response, 1)
	go func() {
		r, err := s.CallTool(ctx, &mcp.CallToolParams{Name: method, Arguments: args})
		done <- response{r, err}
	}()
	observed := false
	for i := 0; i < 200; i++ {
		if present(held) {
			observed = true
			break
		}
		select {
		case <-done:
			panic("S2_P1_PRIVATE_INFLIGHT_FAIL: request finished before native GET was held: " + method)
		default:
		}
		time.Sleep(100 * time.Millisecond)
	}
	check(observed, "real successful native GET not held: "+method)
	// The proxy has already received HTTP 200 with real native bytes;
	// the product handler cannot yet return until its SDK call resumes.
	check(st.RevokeToken(ctx, t.ID) == nil, "cannot revoke exact in-flight bearer")
	_, e = st.VerifyToken(ctx, raw)
	check(e != nil, "bearer remained authorized during delayed read")
	check(os.WriteFile(release, []byte("release after revoke\n"), 0600) == nil,
		"cannot release the single held native GET")
	var got response
	select {
	case got = <-done:
	case <-time.After(18 * time.Second):
		panic("S2_P1_PRIVATE_INFLIGHT_FAIL: stalled MCP read after native release: " + method)
	}
	check(got.err != nil || got.result != nil && got.result.IsError,
		"revoked in-flight bearer received private native content: "+method)
	if got.result != nil {
		data, e := json.Marshal(got.result.StructuredContent)
		check(e == nil && !strings.Contains(string(data), "private-read-sentinel-110r") &&
			!strings.Contains(string(data), "bw-s2-read-revoke-110r"),
			"private native data leaked to revoked token: "+method)
	}
	check(os.Remove(arm) == nil, "arm removal failed")
	fmt.Println("S2_P1_PRIVATE_READ_REVOKED_AFTER_REAL_NATIVE_HTTP_200_" + strings.ToUpper(method) + "=PASS")
}
func read(ctx context.Context, dir, project string) {
	st, e := state.Open(filepath.Join(dir, "ledger.sqlite"))
	check(e == nil, "installed SQLite inaccessible to test actor")
	defer st.Close()
	o := saved(dir)
	// No artificial handler, no SDK mock, and no native mutation here.
	// Each read's real Incus response is captured before bearer revocation.
	for _, test := range []struct {
		name string
		args map[string]any
	}{
		{"host_status", map[string]any{}},
		{"box_list", map[string]any{}},
		{"box_get", map[string]any{"box_id": o.UUID}},
		{"operation_output_get", map[string]any{"operation_id": o.ExecID, "stream": "stdout", "offset": 0, "limit": 512}},
	} {
		invalidate(ctx, st, dir, test.name, test.args)
	}
	_ = project
	fmt.Println("S2_P1_FOUR_REAL_MCP_NATIVE_PRIVATE_READ_BEARER_REVOKE_NO_LEAK=PASS")
}
func cleanup(ctx context.Context, dir, token string) {
	o := saved(dir)
	s := connected(ctx, token)
	defer s.Close()
	stop := tool(ctx, s, "box_stop", map[string]any{
		"box_id": o.UUID, "idempotency_key": "s2-private-read-stop",
	})
	operation(ctx, s, stop)
	del := tool(ctx, s, "box_delete", map[string]any{
		"box_id": o.UUID, "idempotency_key": "s2-private-read-delete",
	})
	operation(ctx, s, del)
	fmt.Println("S2_P1_PRIVATE_READ_EXACT_FIXTURE_CLEANUP=PASS")
}
func main() {
	dir := os.Getenv("BW_S2_PRIVATE_READ_DIR")
	project := os.Getenv("BW_S2_PRIVATE_READ_PROJECT")
	token := os.Getenv("BW_S2_PRIVATE_READ_SEED_TOKEN")
	check(os.Geteuid() != 0 && os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		strings.HasPrefix(dir, "/tmp/bw-v1-s2-ghci.") &&
		filepath.Base(dir) == "p1-private-read" && strings.HasPrefix(project, "user-") &&
		token != "", "test cannot run outside one-use restricted CI")
	ctx, cancel := context.WithTimeout(context.Background(), 260*time.Second)
	defer cancel()
	switch os.Args[1] {
	case "seed":
		seed(ctx, dir, token)
	case "read":
		read(ctx, dir, project)
	case "cleanup":
		cleanup(ctx, dir, token)
	default:
		panic("unknown disposable test mode")
	}
}
