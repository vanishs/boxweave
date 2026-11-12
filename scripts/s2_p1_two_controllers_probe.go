//go:build ignore

// S2 disposable-only two-controller black-box: two separately credentialed
// restricted Incus OS users/projects and independent SQLite-backed MCP services.
// This is not a host orchestrator, and never adopts any shared or UNKNOWN Box.
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

type actorTransport struct{ token string }

func (a actorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cp := req.Clone(req.Context())
	cp.Header.Set("Authorization", "Bearer "+a.token)
	return http.DefaultTransport.RoundTrip(cp)
}
func assert(v bool, reason string) {
	if !v {
		panic("S2_TWO_CONTROLLERS_FAIL: " + reason)
	}
}
func join(ctx context.Context, url, token string) *mcp.ClientSession {
	assert(strings.HasPrefix(url, "http://127.0.0.1:") && strings.HasSuffix(url, "/mcp") && token != "",
		"controller endpoint/credential not isolated")
	c := mcp.NewClient(&mcp.Implementation{Name: "s2-two-controller-independent", Version: "1"}, nil)
	s, e := c.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: url, HTTPClient: &http.Client{Transport: actorTransport{token: token}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	assert(e == nil && s != nil, "cannot connect to independent controller")
	return s
}
func call(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) map[string]any {
	r, e := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	assert(e == nil && r != nil && !r.IsError, "unexpected tool error: "+name)
	b, e := json.Marshal(r.StructuredContent)
	assert(e == nil, "cannot decode tool JSON")
	var obj map[string]any
	assert(json.Unmarshal(b, &obj) == nil && obj != nil, "invalid tool result")
	return obj
}
func field(v map[string]any, k string) string { s, _ := v[k].(string); return s }
func opID(r map[string]any) string {
	id := field(r, "operation_id")
	assert(len(id) == 32, "missing original operation ID")
	return id
}
func completed(ctx context.Context, s *mcp.ClientSession, id string) map[string]any {
	for i := 0; i < 90; i++ {
		r := call(ctx, s, "operation_get", map[string]any{"operation_id": id, "wait_ms": 2000})
		op, _ := r["operation"].(map[string]any)
		assert(op != nil, "operation response missing")
		switch field(op, "state") {
		case "SUCCEEDED":
			return op
		case "FAILED", "UNKNOWN", "CANCELLED":
			panic("S2_TWO_CONTROLLERS_FAIL: unsafe terminal original " + id)
		}
	}
	panic("S2_TWO_CONTROLLERS_FAIL: original timed out")
}
func refuse(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) {
	r, e := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	assert(e != nil || r != nil && r.IsError, "cross-project access succeeded: "+name)
}
func single(ctx context.Context, s *mcp.ClientSession, expected string) {
	v := call(ctx, s, "box_list", map[string]any{})
	boxes, ok := v["boxes"].([]any)
	assert(ok && len(boxes) == 1, "controller sees wrong number of Boxes")
	b, ok := boxes[0].(map[string]any)
	assert(ok && field(b, "box_id") == expected, "controller sees foreign UUID")
}
func empty(ctx context.Context, s *mcp.ClientSession) {
	v := call(ctx, s, "box_list", map[string]any{})
	boxes, ok := v["boxes"].([]any)
	assert(ok && len(boxes) == 0, "controller retained unexpected live Box")
}
func main() {
	assert(os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted",
		"one-use CI only")
	ctx, cancel := context.WithTimeout(context.Background(), 260*time.Second)
	defer cancel()
	a := join(ctx, os.Getenv("BW_S2_DUAL_A_URL"), os.Getenv("BW_S2_DUAL_A_TOKEN"))
	defer a.Close()
	b := join(ctx, os.Getenv("BW_S2_DUAL_B_URL"), os.Getenv("BW_S2_DUAL_B_TOKEN"))
	defer b.Close()
	// Same native name, but independent original keys/projects/OS identities.
	name := "bw-s2-dualctrl-owned-110f"
	makeBox := func(s *mcp.ClientSession, key string) string {
		r := call(ctx, s, "box_create", map[string]any{"name": name, "cpu": 1,
			"memory_mib": 512, "disk_gib": 2, "start": false, "internet": false, "idempotency_key": key})
		done := completed(ctx, s, opID(r))
		u := field(done, "box_uuid")
		assert(u != "", "verified native UUID not bound")
		return u
	}
	u1 := makeBox(a, "s2-two-controller-original-a")
	u2 := makeBox(b, "s2-two-controller-original-b")
	assert(u1 != u2, "two Projects share native UUID")
	single(ctx, a, u1)
	single(ctx, b, u2)
	refuse(ctx, a, "box_get", map[string]any{"box_id": u2})
	refuse(ctx, b, "box_get", map[string]any{"box_id": u1})
	refuse(ctx, a, "box_delete", map[string]any{"box_id": u2, "idempotency_key": "s2-cross-project-a-deny"})
	refuse(ctx, b, "box_delete", map[string]any{"box_id": u1, "idempotency_key": "s2-cross-project-b-deny"})
	single(ctx, a, u1)
	single(ctx, b, u2)
	dir := os.Getenv("BW_S2_DUAL_WITNESS_DIR")
	assert(strings.HasPrefix(dir, "/tmp/bw-v1-s2-ghci."), "unsafe independent witness path")
	assert(os.WriteFile(dir+"/dualctrl.ready", []byte(u1+"\n"+u2+"\n"), 0600) == nil,
		"cannot publish independent native readback request")
	released := false
	for i := 0; i < 200; i++ {
		if _, e := os.Stat(dir + "/dualctrl.release"); e == nil {
			released = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert(released, "independent admin witness not released")
	for _, box := range []struct {
		s       *mcp.ClientSession
		id, key string
	}{
		{a, u1, "s2-two-controller-delete-a"},
		{b, u2, "s2-two-controller-delete-b"},
	} {
		d := call(ctx, box.s, "box_delete", map[string]any{"box_id": box.id, "idempotency_key": box.key})
		completed(ctx, box.s, opID(d))
	}
	empty(ctx, a)
	empty(ctx, b)
	fmt.Println("S2_P1_TWO_REAL_OS_IDENTITIES_TWO_CONTROLLER_PROJECTS_SAME_NAME_NO_CROSS_ACCESS=PASS")
}
