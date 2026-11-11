//go:build ignore

// Real official MCP SDK proof of a preconfigured, separately mounted
// alternate root pool with omitted disk_gib default. No exposed pool selector.
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
)

type tokenTransport struct{ token string }

func (t tokenTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(cp)
}
func assert(ok bool, s string) {
	if !ok {
		panic("S2_P2_ALT_POOL_FAIL: " + s)
	}
}
func call(ctx context.Context, s *mcp.ClientSession, n string, a map[string]any) (map[string]any, error) {
	out, e := s.CallTool(ctx, &mcp.CallToolParams{Name: n, Arguments: a})
	if e != nil {
		return nil, e
	}
	if out == nil || out.IsError {
		return nil, fmt.Errorf("MCP rejection %s", n)
	}
	raw, e := json.Marshal(out.StructuredContent)
	if e != nil {
		return nil, e
	}
	var v map[string]any
	e = json.Unmarshal(raw, &v)
	return v, e
}
func field(v map[string]any, n string) string         { s, _ := v[n].(string); return s }
func child(v map[string]any, n string) map[string]any { m, _ := v[n].(map[string]any); return m }
func completed(ctx context.Context, s *mcp.ClientSession, id string) map[string]any {
	for i := 0; i < 120; i++ {
		p, e := call(ctx, s, "operation_get", map[string]any{"operation_id": id, "wait_ms": 700})
		assert(e == nil, "original operation inaccessible")
		op := child(p, "operation")
		state := field(op, "state")
		if state == "SUCCEEDED" {
			return op
		}
		assert(state != "FAILED" && state != "UNKNOWN" && state != "CANCELLED",
			"native default pool original unsafe state "+state)
	}
	panic("S2_P2_ALT_POOL_FAIL: original never verified")
}
func main() {
	dir := os.Getenv("BW_S2_ALT_LAB")
	token := os.Getenv("BW_S2_ALT_TOKEN")
	project := os.Getenv("BW_S2_ALT_PROJECT")
	assert(os.Geteuid() != 0 && os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		filepath.Base(dir) == "p2-alt-pool" && strings.HasPrefix(dir, "/tmp/bw-v1-s2-ghci.") &&
		strings.HasPrefix(project, "user-") && strings.HasPrefix(token, "bw1_"),
		"only single-use restricted CI")
	ctx, cancel := context.WithTimeout(context.Background(), 170*time.Second)
	defer cancel()
	cli := mcp.NewClient(&mcp.Implementation{Name: "s2-p2-alternate-root-profile", Version: "1"}, nil)
	s, e := cli.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: "http://127.0.0.1:18666/mcp",
		HTTPClient:           &http.Client{Transport: tokenTransport{token}},
		DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	assert(e == nil && s != nil, "independent MCP connection failed")
	defer s.Close()
	status, e := call(ctx, s, "host_status", map[string]any{})
	assert(e == nil, "alternate pool host status inaccessible")
	native := child(status, "native")
	assert(native != nil && native["selected_storage_pool_ready"] == true &&
		native["selected_storage_quota_verified"] == true,
		"alternate selected pool did not pass strict native quota preflight")
	initial, e := call(ctx, s, "box_list", map[string]any{})
	list, _ := initial["boxes"].([]any)
	assert(e == nil && len(list) == 0, "nonempty test Project before alternate pool")
	// Reject any request to select a pool other than the one chosen and
	// validated by the restricted Project's own default profile.
	_, e = call(ctx, s, "box_create", map[string]any{
		"name": "bw-s2-altpool-unsafe-110w", "root_pool": "default",
		"start": false, "idempotency_key": "s2-altpool-unapproved-selector"})
	assert(e != nil, "unsupported pool selector silently accepted")
	again, e := call(ctx, s, "box_list", map[string]any{})
	after, _ := again["boxes"].([]any)
	assert(e == nil && len(after) == 0, "unsupported pool selected/created a native instance")
	createArgs := map[string]any{
		"name": "bw-s2-alternate-pool-110w", "cpu": 1, "memory_mib": 512,
		"start": false, "internet": false,
		"idempotency_key": "s2-altpool-omitted-size-original"}
	receipt, e := call(ctx, s, "box_create", createArgs)
	assert(e == nil && len(field(receipt, "operation_id")) == 32,
		"valid omitted-size Create was not admitted")
	created := completed(ctx, s, field(receipt, "operation_id"))
	uuid := field(created, "box_uuid")
	assert(len(uuid) == 36, "alt pool create did not bind genuine native UUID")
	read, e := call(ctx, s, "box_get", map[string]any{"box_id": uuid})
	box := child(read, "box")
	assert(e == nil && box != nil && field(box, "box_id") == uuid &&
		field(box, "native_state") == "Stopped" &&
		field(box, "root_disk_size") == "10GiB",
		"omitted disk default not observed as exact 10GiB native root")
	data, e := json.Marshal(map[string]string{"box_id": uuid, "create_id": field(receipt, "operation_id")})
	assert(e == nil && os.WriteFile(filepath.Join(dir, "original.json"), data, 0600) == nil, "save exact native witness")
	// Retain the *live* instance until an independent Incus administrator
	// confirms expanded root device, actual UUID and selected pool, rather
	// than merely trusting the MCP response or checking after deletion.
	assert(os.WriteFile(filepath.Join(dir, "live.ready"), []byte("native UUID bound; awaiting independent readback\n"), 0600) == nil,
		"cannot signal independent native readback")
	confirmed := false
	for i := 0; i < 240; i++ {
		if _, err := os.Stat(filepath.Join(dir, "native-verified.release")); err == nil {
			confirmed = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert(confirmed, "independent live Incus pool witness never released")
	fmt.Println("S2_P2_ALTERNATE_SELECTED_PROFILE_POOL_DEFAULT_TEN_GIB=PASS")
	del, e := call(ctx, s, "box_delete", map[string]any{
		"box_id": uuid, "idempotency_key": "s2-altpool-original-delete"})
	assert(e == nil && len(field(del, "operation_id")) == 32, "exact owned alt Box not deletable")
	completed(ctx, s, field(del, "operation_id"))
	empty, e := call(ctx, s, "box_list", map[string]any{})
	boxes, _ := empty["boxes"].([]any)
	assert(e == nil && len(boxes) == 0, "owned Box left after exact deletion")
	fmt.Println("S2_P2_ALTERNATE_POOL_NO_EXPLICIT_SELECTION_NO_RESIDUAL_BOX=PASS")
}
