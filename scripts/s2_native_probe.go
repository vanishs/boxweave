//go:build ignore

// S2 candidate-only native black-box probe. This tool is NOT shipped with
// BoxWeave and runs exclusively inside the authorized single-use GH VM lab.
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

func (t bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(cp)
}
func check(cond bool, msg string) {
	if !cond {
		fmt.Fprintln(os.Stderr, "S2_NATIVE_FAIL:", msg)
		os.Exit(40)
	}
}
func decoded(result *mcp.CallToolResult) map[string]any {
	if result == nil || result.IsError {
		panic(fmt.Sprintf("native MCP result returned error: %+v", result))
	}
	buf, err := json.Marshal(result.StructuredContent)
	if err != nil {
		panic(err)
	}
	var v map[string]any
	if err = json.Unmarshal(buf, &v); err != nil {
		panic(err)
	}
	return v
}
func field(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}
func child(m map[string]any, key string) map[string]any {
	out, _ := m[key].(map[string]any)
	return out
}
func call(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) map[string]any {
	res, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		panic(fmt.Errorf("%s: %w", name, err))
	}
	return decoded(res)
}
func wait(ctx context.Context, s *mcp.ClientSession, opID string) map[string]any {
	deadline := time.Now().Add(210 * time.Second)
	for time.Now().Before(deadline) {
		r := child(call(ctx, s, "operation_get", map[string]any{"operation_id": opID, "wait_ms": 5000}), "operation")
		switch field(r, "state") {
		case "SUCCEEDED":
			return r
		case "FAILED", "UNKNOWN", "CANCELLED":
			panic(fmt.Sprintf("native operation %s stopped in %q: %+v", opID, field(r, "state"), r))
		}
	}
	panic("native operation wait exceeded independent 210s budget")
}
func connect(ctx context.Context, url, token string) *mcp.ClientSession {
	c := mcp.NewClient(&mcp.Implementation{Name: "s2-native-independent-prober", Version: "1"}, nil)
	session, err := c.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: url, HTTPClient: &http.Client{Transport: bearer{token: token}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		panic(err)
	}
	return session
}
func id(m map[string]any) string {
	s := field(m, "operation_id")
	check(s != "", "missing original operation ID")
	return s
}
func main() {
	url, token := os.Getenv("BW_S2_MCP_URL"), os.Getenv("BW_S2_ADMIN_TOKEN")
	viewer := os.Getenv("BW_S2_VIEWER_TOKEN")
	check(strings.HasPrefix(url, "http://127.0.0.1:"), "MCP must be loopback")
	check(token != "" && viewer != "", "both independent credentials required")
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	admin := connect(ctx, url, token)
	defer admin.Close()
	view := connect(ctx, url, viewer)
	defer view.Close()
	inv, err := admin.ListTools(ctx, nil)
	if err != nil {
		panic(err)
	}
	expected := map[string]bool{
		"host_status": true, "box_create": true, "box_list": true, "box_get": true,
		"box_update": true, "box_start": true, "box_stop": true, "box_delete": true,
		"box_exec": true, "operation_get": true, "operation_list": true,
		"operation_output_get": true, "operation_cancel": true,
	}
	check(len(inv.Tools) == len(expected), "13/13 actual S1+S2 discovery")
	for _, t := range inv.Tools {
		check(expected[t.Name], "unexpected Tool "+t.Name)
	}
	fmt.Println("S2_NATIVE_MCP_DISCOVERY_13_OF_13=PASS")

	denied, err := view.CallTool(ctx, &mcp.CallToolParams{Name: "box_create",
		Arguments: map[string]any{"name": "s2-denied", "idempotency_key": "native-viewer-create-denied-01"}})
	check(err != nil || denied != nil && denied.IsError, "viewer created native Box")
	fmt.Println("S2_NATIVE_VIEWER_CREATE_REFUSED=PASS")

	name := "bw-s2-ghci-110a"
	key := "native-s2-create-once-20261110"
	args := map[string]any{"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": false, "idempotency_key": key, "work_ref": "lab/s2", "agent_ref": "prober-01"}
	created := call(ctx, admin, "box_create", args)
	original := id(created)
	again := call(ctx, admin, "box_create", args)
	check(id(again) == original, "same key unexpectedly created second operation")
	fmt.Println("S2_NATIVE_CREATE_IDEMPOTENCY=PASS")
	contradictory := map[string]any{}
	for k, v := range args {
		contradictory[k] = v
	}
	contradictory["cpu"] = 2
	conflict, conflictErr := admin.CallTool(ctx, &mcp.CallToolParams{Name: "box_create", Arguments: contradictory})
	check(conflictErr != nil || conflict != nil && conflict.IsError,
		"same actor/key with different native create intent was accepted")
	distinctKey := map[string]any{}
	for k, v := range args {
		distinctKey[k] = v
	}
	distinctKey["idempotency_key"] = "native-s2-conflicting-name-second-key-01"
	other, otherErr := admin.CallTool(ctx, &mcp.CallToolParams{Name: "box_create", Arguments: distinctKey})
	if otherErr == nil && other != nil && !other.IsError {
		// If original create already completed, the new key may be
		// admitted but MUST fail before any duplicate native create.
		retryOp := id(decoded(other))
		outcome := ""
		for i := 0; i < 50; i++ {
			receipt := child(call(ctx, admin, "operation_get", map[string]any{
				"operation_id": retryOp, "wait_ms": 300}), "operation")
			outcome = field(receipt, "state")
			if outcome == "FAILED" || outcome == "UNKNOWN" || outcome == "SUCCEEDED" {
				break
			}
		}
		check(outcome == "FAILED", "different key made duplicate native create: "+outcome)
	}
	fmt.Println("S2_NATIVE_SAME_KEY_CONFLICT_AND_DIFFERENT_KEY_DUPLICATE_DENIED=PASS")
	cr := wait(ctx, admin, original)
	uuid := field(cr, "box_uuid")
	check(uuid != "", "verified native create did not persist UUID")
	box := child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(field(box, "box_id") == uuid && field(box, "native_state") == "Stopped",
		"created native UUID/stopped state not confirmed")
	fmt.Println("S2_NATIVE_CREATE_STOPPED_UUID=PASS")

	foreignList := call(ctx, view, "operation_list", map[string]any{"work_ref": "lab/s2"})
	private, _ := json.Marshal(foreignList)
	check(!strings.Contains(string(private), original), "viewer borrowed original work_ref across token actors")
	fmt.Println("S2_NATIVE_OPERATION_WORK_SCOPE=PASS")

	// The real installed server must list only its bound native UUID, not
	// an unrelated Project, guessed instance name, or database-only identity.
	listed := call(ctx, admin, "box_list", map[string]any{"limit": 20})
	foundBoxes, ok := listed["boxes"].([]any)
	check(ok && len(foundBoxes) == 1, "box_list did not return exactly the owned native Box")
	foundBox, ok := foundBoxes[0].(map[string]any)
	check(ok && field(foundBox, "box_id") == uuid, "box_list returned a foreign or stale Box")
	fmt.Println("S2_NATIVE_BOX_LIST_UUID=PASS")

	// Change CPU and RAM in a single typed native UpdateInstance without
	// replacing inherited NIC/root disk configuration.
	changed := call(ctx, admin, "box_update", map[string]any{
		"box_id": uuid, "cpu": 2, "memory_mib": 768,
		"idempotency_key": "native-s2-update-cpu-memory-01"})
	wait(ctx, admin, id(changed))
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(field(box, "cpu") == "2" && field(box, "memory") == "768MiB" &&
		field(box, "root_disk_size") == "2GiB",
		"CPU/RAM update erased omitted disk or did not persist")
	fmt.Println("S2_NATIVE_CPU_MEMORY_UPDATE_PRESERVES_DISK=PASS")

	grown := call(ctx, admin, "box_update", map[string]any{
		"box_id": uuid, "disk_gib": 3,
		"idempotency_key": "native-s2-grow-owned-disk-01"})
	wait(ctx, admin, id(grown))
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(field(box, "root_disk_size") == "3GiB" && field(box, "cpu") == "2",
		"disk growth failed or wiped CPU setting")
	fmt.Println("S2_NATIVE_DISK_GROWTH_PRESERVES_CPU=PASS")

	shrink := call(ctx, admin, "box_update", map[string]any{
		"box_id": uuid, "disk_gib": 2,
		"idempotency_key": "native-s2-shrink-must-be-denied-01"})
	shrinkID := id(shrink)
	var shrinkState string
	for i := 0; i < 30; i++ {
		shrinkState = field(child(call(ctx, admin, "operation_get", map[string]any{
			"operation_id": shrinkID, "wait_ms": 300}), "operation"), "state")
		if shrinkState == "FAILED" || shrinkState == "UNKNOWN" || shrinkState == "SUCCEEDED" {
			break
		}
	}
	check(shrinkState == "FAILED", "disk shrink was accepted or ambiguously dispatched: "+shrinkState)
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(field(box, "root_disk_size") == "3GiB", "disk shrink changed native root device")
	fmt.Println("S2_NATIVE_DISK_SHRINK_DENIED=PASS")

	start := call(ctx, admin, "box_start", map[string]any{"box_id": uuid,
		"idempotency_key": "native-s2-start-000000001"})
	sr := wait(ctx, admin, id(start))
	check(field(sr, "state") == "SUCCEEDED", "native start failed")
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(field(box, "native_state") == "Running", "native box did not start")
	fmt.Println("S2_NATIVE_START=PASS")

	cmd := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-c", "printf s2-out; printf s2-err >&2; exit 7"},
		"idempotency_key": "native-s2-exec-nonzero-001",
		"work_ref":        "lab/s2", "agent_ref": "prober-02"})
	execID := id(cmd)
	er := wait(ctx, admin, execID)
	check(field(er, "state") == "SUCCEEDED", "guest command execution not recorded")
	admin.Close()
	admin = connect(ctx, url, token)
	defer admin.Close()
	for _, mode := range []struct{ name, want string }{{"stdout", "s2-out"}, {"stderr", "s2-err"}} {
		stream := call(ctx, admin, "operation_output_get", map[string]any{
			"operation_id": execID, "stream": mode.name, "limit": 4096})
		raw, err := base64.StdEncoding.DecodeString(field(stream, "content_base64"))
		if err != nil {
			panic(err)
		}
		check(strings.Contains(string(raw), mode.want), mode.name+" recorded output mismatch")
		check(stream["exit_code"] == float64(7), mode.name+" did not retain observed guest exit 7")
		check(stream["command_success"] == false, mode.name+" falsely marked guest exit 7 as success")
	}
	fmt.Println("S2_NATIVE_EXEC_RECONNECT_EXIT_7_STDOUT_STDERR=PASS")

	noisy := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-c", "yes x | head -c 70000"},
		"idempotency_key": "native-s2-noisy-paginated-exec-001"})
	noisyID := id(noisy)
	wait(ctx, admin, noisyID)
	firstPage := call(ctx, admin, "operation_output_get", map[string]any{
		"operation_id": noisyID, "stream": "stdout", "offset": 0, "limit": 65536})
	firstBytes, err := base64.StdEncoding.DecodeString(field(firstPage, "content_base64"))
	if err != nil {
		panic(err)
	}
	check(len(firstBytes) == 65536 && firstPage["ended"] == false,
		"noisy output first page must be bounded, not silently truncated")
	secondPage := call(ctx, admin, "operation_output_get", map[string]any{
		"operation_id": noisyID, "stream": "stdout", "offset": 65536, "limit": 65536})
	secondBytes, err := base64.StdEncoding.DecodeString(field(secondPage, "content_base64"))
	if err != nil {
		panic(err)
	}
	check(len(firstBytes)+len(secondBytes) == 70000 && secondPage["ended"] == true,
		"noisy output second page did not resume original native evidence")
	fmt.Println("S2_NATIVE_NOISY_EXEC_PAGINATION_70K=PASS")

	delayed := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-c", "sleep 1; printf delayed-ok"},
		"timeout_ms": 5000, "idempotency_key": "native-s2-delayed-time-bounded-exec-01"})
	delayedID := id(delayed)
	wait(ctx, admin, delayedID)
	delayedOut := call(ctx, admin, "operation_output_get", map[string]any{
		"operation_id": delayedID, "stream": "stdout"})
	delayedData, err := base64.StdEncoding.DecodeString(field(delayedOut, "content_base64"))
	if err != nil {
		panic(err)
	}
	check(string(delayedData) == "delayed-ok" && delayedOut["exit_code"] == float64(0) &&
		delayedOut["command_success"] == true, "bounded delayed Exec lost exit/output")
	fmt.Println("S2_NATIVE_DELAYED_EXEC_TIMEOUT_BUDGET=PASS")

	quiet := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/true"},
		"idempotency_key": "native-s2-quiet-exec-zero-001"})
	quietID := id(quiet)
	wait(ctx, admin, quietID)
	quietOut := call(ctx, admin, "operation_output_get", map[string]any{
		"operation_id": quietID, "stream": "stdout"})
	check(field(quietOut, "content_base64") == "" && quietOut["ended"] == true &&
		quietOut["exit_code"] == float64(0), "quiet Exec fabricated output or lost return")
	fmt.Println("S2_NATIVE_QUIET_EXEC_RECORDED_ZERO_EXIT=PASS")

	// A deliberately cancelled read-side long poll cannot cancel or undo
	// the already accepted native Exec. Use a fresh client for the broken
	// HTTP request so the original actor session remains independent.
	slow := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-c", "sleep 3; printf still-ran"},
		"idempotency_key": "native-s2-cancel-only-read-poll-001"})
	slowID := id(slow)
	poller := connect(ctx, url, token)
	readCtx, stopReading := context.WithTimeout(ctx, 250*time.Millisecond)
	_, waitErr := poller.CallTool(readCtx, &mcp.CallToolParams{Name: "operation_get",
		Arguments: map[string]any{"operation_id": slowID, "wait_ms": 20000}})
	stopReading()
	_ = poller.Close()
	check(waitErr != nil, "canceled long read unexpectedly claimed native completion")
	finished := wait(ctx, admin, slowID)
	check(field(finished, "state") == "SUCCEEDED",
		"canceled read-side poll changed original native execution")
	pollOut := call(ctx, admin, "operation_output_get", map[string]any{
		"operation_id": slowID, "stream": "stdout"})
	pollBytes, pollErr := base64.StdEncoding.DecodeString(field(pollOut, "content_base64"))
	if pollErr != nil {
		panic(pollErr)
	}
	check(string(pollBytes) == "still-ran" && pollOut["exit_code"] == float64(0),
		"client read cancellation propagated to guest Exec")
	fmt.Println("S2_NATIVE_CANCELLED_LONG_POLL_IS_READ_ONLY=PASS")

	// Native detached/recorded Exec cannot be cancelled after it finishes.
	// The cancellation request has its own failed receipt and must not
	// retroactively change the original verified guest completion.
	cancelled := call(ctx, admin, "operation_cancel", map[string]any{
		"target_operation_id": execID, "idempotency_key": "native-s2-exec-late-cancel-01"})
	cancelReceipt := child(cancelled, "cancel_operation")
	cancelID := field(cancelReceipt, "operation_id")
	check(cancelID != "" && cancelID != execID, "cancellation forged original operation ID")
	var cancelState string
	for i := 0; i < 30; i++ {
		cancelState = field(child(call(ctx, admin, "operation_get", map[string]any{
			"operation_id": cancelID, "wait_ms": 300}), "operation"), "state")
		if cancelState == "FAILED" || cancelState == "UNKNOWN" || cancelState == "SUCCEEDED" {
			break
		}
	}
	check(cancelState == "FAILED", "detached Exec cancellation lied: "+cancelState)
	execAgain := child(call(ctx, admin, "operation_get", map[string]any{"operation_id": execID}), "operation")
	check(field(execAgain, "state") == "SUCCEEDED",
		"unsupported cancellation rewrote original Exec")
	fmt.Println("S2_NATIVE_EXEC_UNSUPPORTED_CANCEL_TRUTHFUL=PASS")

	stop := call(ctx, admin, "box_stop", map[string]any{"box_id": uuid,
		"idempotency_key": "native-s2-stop-0000000001"})
	wait(ctx, admin, id(stop))
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(field(box, "native_state") == "Stopped", "native Box failed to stop")
	fmt.Println("S2_NATIVE_STOP=PASS")

	del := call(ctx, admin, "box_delete", map[string]any{"box_id": uuid,
		"idempotency_key": "native-s2-delete-00000001"})
	wait(ctx, admin, id(del))
	replayedDelete := call(ctx, admin, "box_delete", map[string]any{"box_id": uuid,
		"idempotency_key": "native-s2-delete-00000001"})
	check(id(replayedDelete) == id(del) && replayedDelete["duplicate"] == true,
		"original delete did not replay after native instance disappeared")
	fmt.Println("S2_NATIVE_DELETE_VERIFIED_AND_REPLAY=PASS")

	empty := call(ctx, admin, "box_list", map[string]any{})
	remaining, ok := empty["boxes"].([]any)
	check(ok && len(remaining) == 0, "tombstoned Box still appeared in owned inventory")
	fmt.Println("S2_NATIVE_DELETED_BOX_NOT_LISTED=PASS")
	fmt.Println("S2_NATIVE_END_TO_END=PASS")
}
