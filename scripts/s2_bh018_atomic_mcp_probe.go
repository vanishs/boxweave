//go:build ignore

// BH-018 single-use, installed-MCP, real-Incus acceptance on a disposable
// GitHub Actions Incus runner. Not shipped. Never retries a distinct native
// effect or clears an uncertain/UNKNOWN original claim.
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

type bearerTransport struct{ token string }

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cp := req.Clone(req.Context())
	cp.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(cp)
}

type verifiedCase struct {
	Exit       int    `json:"exit"`
	OriginalID string `json:"original_id"`
	NativeID   string `json:"native_id"`
}

type checkpoint struct {
	BoxID     string         `json:"box_id"`
	CreateID  string         `json:"create_id"`
	Specials  []verifiedCase `json:"specials"`
	CreatedAt int64          `json:"created_at"`
}

func require(ok bool, format string, a ...any) {
	if !ok {
		panic(fmt.Sprintf("BH018_MCP_FAIL: "+format, a...))
	}
}

func getString(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}
func getMap(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	require(v != nil, "missing object %s", key)
	return v
}
func invoke(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) map[string]any {
	r, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	require(err == nil && r != nil && !r.IsError, "MCP %s rejected: %v", name, err)
	raw, err := json.Marshal(r.StructuredContent)
	require(err == nil, "cannot marshal %s: %v", name, err)
	var out map[string]any
	require(json.Unmarshal(raw, &out) == nil && out != nil, "malformed %s response", name)
	return out
}
func originalID(m map[string]any) string {
	id := getString(m, "operation_id")
	require(len(id) == 32, "lost immutable BW original operation ID: %q", id)
	return id
}
func poll(ctx context.Context, s *mcp.ClientSession, id string) map[string]any {
	deadline := time.Now().Add(220 * time.Second)
	for time.Now().Before(deadline) {
		r := getMap(invoke(ctx, s, "operation_get", map[string]any{
			"operation_id": id, "wait_ms": 5000,
		}), "operation")
		require(getString(r, "operation_id") == id, "original receipt ID drift")
		switch getString(r, "state") {
		case "SUCCEEDED":
			return r
		case "UNKNOWN", "FAILED", "CANCELLED":
			panic(fmt.Sprintf("BH018_MCP_FAIL: original %s stopped in %s", id, getString(r, "state")))
		}
	}
	panic("BH018_MCP_FAIL: terminal confirmation deadline exceeded; do not retry")
}
func readOutput(ctx context.Context, s *mcp.ClientSession, id, stream string, exit int, want string) {
	out := invoke(ctx, s, "operation_output_get", map[string]any{
		"operation_id": id, "stream": stream, "limit": 4096,
	})
	data, err := base64.StdEncoding.DecodeString(getString(out, "content_base64"))
	require(err == nil, "base64 output %s: %v", stream, err)
	require(string(data) == want, "unexpected %s for original %s: %q", stream, id, string(data))
	code, ok := out["exit_code"].(float64)
	require(ok && int(code) == exit, "original guest exit was not retained: %v", out["exit_code"])
	success, ok := out["command_success"].(bool)
	require(ok && success == (exit == 0), "confused native terminal and guest command_success")
}
func argv(tcExit int) []string {
	return []string{"/bin/sh", "-c", fmt.Sprintf(
		"printf 'BH018_EFFECT_%d\\n' >> /tmp/bh018-atomic-effect-%d; printf 'OUT_%d'; printf 'ERR_%d' >&2; exit %d",
		tcExit, tcExit, tcExit, tcExit, tcExit,
	)}
}
func key(tcExit int) string { return fmt.Sprintf("s2-bh018-original-exit-%d-001", tcExit) }

func connect(ctx context.Context) *mcp.ClientSession {
	url, token := os.Getenv("BW_S2_MCP_URL"), os.Getenv("BW_S2_ADMIN_TOKEN")
	require(strings.HasPrefix(url, "http://127.0.0.1:"), "MCP test endpoint must be loopback")
	require(token != "", "missing isolated admin token")
	c := mcp.NewClient(&mcp.Implementation{Name: "bh018-native-installed-test", Version: "1"}, nil)
	s, err := c.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             url,
		HTTPClient:           &http.Client{Transport: bearerTransport{token: token}},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
	require(err == nil, "official MCP client handshake: %v", err)
	return s
}

func start(ctx context.Context, s *mcp.ClientSession, stateFile string) {
	_, err := os.Stat(stateFile)
	require(os.IsNotExist(err), "test checkpoint already exists; no duplicate create")
	create := invoke(ctx, s, "box_create", map[string]any{
		"name": "bw-s2-bh018-atomic-native",
		"cpu":  1, "memory_mib": 512, "disk_gib": 2,
		"start": true, "internet": false,
		"idempotency_key": "s2-bh018-atomic-native-create-001",
	})
	createID := originalID(create)
	finalCreate := poll(ctx, s, createID)
	boxID := getString(finalCreate, "box_uuid")
	require(len(boxID) == 36, "real Box missing original native UUID")
	require(getString(finalCreate, "native_operation_id") != "", "real Create lacked native operation ID")
	fmt.Printf("BH018_MCP_CREATE_BOX_UUID=%s BW_ID=%s NATIVE_ID=%s\n", boxID, createID, getString(finalCreate, "native_operation_id"))
	cp := checkpoint{BoxID: boxID, CreateID: createID, CreatedAt: time.Now().Unix()}
	for _, exit := range []int{126, 127} {
		command := argv(exit)
		request := map[string]any{
			"box_id": boxID, "argv": command,
			"idempotency_key": key(exit),
		}
		effect := invoke(ctx, s, "box_exec", request)
		id := originalID(effect)
		final := poll(ctx, s, id)
		nativeID := getString(final, "native_operation_id")
		require(len(nativeID) == 36, "lost original native Incus Exec ID at exit %d", exit)
		fmt.Printf("BH018_MCP_EXEC_EXIT_%d_BW_ID=%s NATIVE_ID=%s\n", exit, id, nativeID)
		readOutput(ctx, s, id, "stdout", exit, fmt.Sprintf("OUT_%d", exit))
		readOutput(ctx, s, id, "stderr", exit, fmt.Sprintf("ERR_%d", exit))
		cp.Specials = append(cp.Specials, verifiedCase{
			Exit: exit, OriginalID: id, NativeID: nativeID,
		})
		fmt.Printf("BH018_NATIVE_REAL_HTTP_MCP_EXIT_%d_ATOMIC_RECEIPT=PASS\n", exit)
	}
	raw, err := json.MarshalIndent(cp, "", "  ")
	require(err == nil, "checkpoint JSON failure")
	require(os.WriteFile(stateFile, raw, 0600) == nil, "checkpoint write failure; original effects must not be retried")
	fmt.Println("BH018_NATIVE_REAL_HTTP_MCP_ATOMIC_START=PASS")
}
func resume(ctx context.Context, s *mcp.ClientSession, stateFile string) {
	fmt.Println("BH018_MCP_RESUME_PHASE=CHECKPOINT_READ")
	raw, err := os.ReadFile(stateFile)
	require(err == nil, "checkpoint unavailable: %v", err)
	var cp checkpoint
	require(json.Unmarshal(raw, &cp) == nil && len(cp.BoxID) == 36 && len(cp.Specials) == 2,
		"invalid original-only restart checkpoint")
	fmt.Println("BH018_MCP_RESUME_PHASE=CREATE_READBACK")
	require(getString(poll(ctx, s, cp.CreateID), "box_uuid") == cp.BoxID,
		"reopened project adopted a different native instance")
	fmt.Println("BH018_MCP_RESUME_PHASE=CREATE_VERIFIED")

	for _, original := range cp.Specials {
		fmt.Printf("BH018_MCP_RESUME_PHASE=ORIGINAL_%d_READBACK\n", original.Exit)
		final := poll(ctx, s, original.OriginalID)
		require(getString(final, "native_operation_id") == original.NativeID,
			"original native Incus Exec ID changed across restart")
		fmt.Printf("BH018_MCP_REOPEN_EXIT_%d_BW_ID=%s NATIVE_ID=%s\n", original.Exit, original.OriginalID, original.NativeID)
		readOutput(ctx, s, original.OriginalID, "stdout", original.Exit,
			fmt.Sprintf("OUT_%d", original.Exit))
		readOutput(ctx, s, original.OriginalID, "stderr", original.Exit,
			fmt.Sprintf("ERR_%d", original.Exit))
		fmt.Printf("BH018_MCP_RESUME_PHASE=ORIGINAL_%d_OUTPUT_VERIFIED\n", original.Exit)

		// Exactly the SAME authorized idempotency key + argv: only a
		// read-only replay of the original, never a second native POST.
		fmt.Printf("BH018_MCP_RESUME_PHASE=ORIGINAL_%d_REPLAY\n", original.Exit)
		replay := invoke(ctx, s, "box_exec", map[string]any{
			"box_id": cp.BoxID, "argv": argv(original.Exit),
			"idempotency_key": key(original.Exit),
		})
		require(originalID(replay) == original.OriginalID && replay["duplicate"] == true &&
			getString(replay, "native_operation_id") == original.NativeID,
			"original dedupe after restart lost provenance or reissued Guest effects")
		fmt.Printf("BH018_MCP_RESUME_PHASE=ORIGINAL_%d_REPLAY_VERIFIED\n", original.Exit)

		// A NEW request is allowed only after the verified original
		// native terminal. Independently read guest file, prove exact
		// effect count = one, and verify the fresh Exec itself succeeded.
		fmt.Printf("BH018_MCP_RESUME_PHASE=ORIGINAL_%d_FRESH_READBACK\n", original.Exit)
		check := invoke(ctx, s, "box_exec", map[string]any{
			"box_id": cp.BoxID,
			"argv": []string{"/bin/cat",
				fmt.Sprintf("/tmp/bh018-atomic-effect-%d", original.Exit)},
			"idempotency_key": fmt.Sprintf("s2-bh018-independent-readback-%d-001", original.Exit),
		})
		fmt.Printf("BH018_MCP_RESUME_PHASE=ORIGINAL_%d_FRESH_CALL_RETURNED\n", original.Exit)
		readID := originalID(check)
		fmt.Printf("BH018_MCP_RESUME_PHASE=ORIGINAL_%d_FRESH_ID_VALIDATED\n", original.Exit)
		// An admitted fresh readback is new, independent work, never a retry
		// of an uncertain original. Print only its immutable nonsecret ID.
		fmt.Printf("BH018_MCP_FRESH_ADMITTED_EXIT_%d_BW_ID=%s\n", original.Exit, readID)
		fresh := poll(ctx, s, readID)
		freshNativeID := getString(fresh, "native_operation_id")
		require(len(freshNativeID) == 36, "new readback lacks native identity")
		fmt.Printf("BH018_MCP_READBACK_EXIT_%d_BW_ID=%s NATIVE_ID=%s\n", original.Exit, readID, freshNativeID)
		readOutput(ctx, s, readID, "stdout", 0,
			fmt.Sprintf("BH018_EFFECT_%d\n", original.Exit))
		fmt.Printf("BH018_NATIVE_REAL_HTTP_MCP_EXIT_%d_DEDUPE_READBACK_AFTER_RESTART=PASS\n",
			original.Exit)
	}
	stop := invoke(ctx, s, "box_stop", map[string]any{
		"box_id": cp.BoxID, "idempotency_key": "s2-bh018-atomic-native-stop-001",
	})
	stopID := originalID(stop)
	stopReceipt := poll(ctx, s, stopID)
	fmt.Printf("BH018_MCP_CLEAN_STOP_BW_ID=%s NATIVE_ID=%s\n", stopID, getString(stopReceipt, "native_operation_id"))
	del := invoke(ctx, s, "box_delete", map[string]any{
		"box_id": cp.BoxID, "idempotency_key": "s2-bh018-atomic-native-delete-001",
	})
	delID := originalID(del)
	delReceipt := poll(ctx, s, delID)
	fmt.Printf("BH018_MCP_CLEAN_DELETE_BW_ID=%s NATIVE_ID=%s\n", delID, getString(delReceipt, "native_operation_id"))
	list := invoke(ctx, s, "box_list", map[string]any{})
	rows, ok := list["boxes"].([]any)
	require(ok && len(rows) == 0, "disposable owned instance not fully deleted")
	fmt.Println("BH018_NATIVE_REAL_HTTP_MCP_ATOMIC_RESTART_AND_CLEANUP=PASS")
}

func main() {
	require(len(os.Args) == 2 && (os.Args[1] == "start" || os.Args[1] == "resume"),
		"mode must be start or resume")
	stateFile := os.Getenv("BW_S2_BH018_CHECKPOINT")
	require(stateFile != "", "checkpoint path not set")
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	if os.Args[1] == "resume" {
		fmt.Println("BH018_MCP_RESUME_PHASE=CONNECT")
	}
	s := connect(ctx)
	defer s.Close()
	if os.Args[1] == "resume" {
		fmt.Println("BH018_MCP_RESUME_PHASE=CONNECTED")
	}
	if os.Args[1] == "start" {
		start(ctx, s, stateFile)
	} else {
		resume(ctx, s, stateFile)
	}
}
