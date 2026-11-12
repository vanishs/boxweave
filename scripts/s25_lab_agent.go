//go:build ignore

// Independent S2.5 Agent-process black-box client. No private native socket,
// host shell or test-only MCP endpoint: all lifecycle effects use the official
// Go MCP SDK, while a separate Verifier owns native read-only attestations.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type authorizedHTTP struct{ credential string }

func (a authorizedHTTP) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Header.Set("Authorization", "Bearer "+a.credential)
	return http.DefaultTransport.RoundTrip(cp)
}
func must(ok bool, reason string) {
	if !ok {
		panic("S25_FAIL: " + reason)
	}
}
func environmental(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic("missing required LAB environment " + k)
	}
	return v
}
func tokenFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "token=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "token="))
		}
	}
	panic("token fixture has no issued credential")
}
func session(ctx context.Context, file string) *mcp.ClientSession {
	c := mcp.NewClient(&mcp.Implementation{Name: "boxweave-s25-separated-agent", Version: "1"}, nil)
	sess, err := c.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             environmental("BW_S25_URL"),
		HTTPClient:           &http.Client{Transport: authorizedHTTP{tokenFile(file)}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		panic(fmt.Errorf("SDK initialize: %w", err))
	}
	return sess
}
func tools(ctx context.Context, c *mcp.ClientSession) {
	t, err := c.ListTools(ctx, nil)
	if err != nil {
		panic(err)
	}
	expected := map[string]bool{"host_status": true, "box_create": true, "box_get": true,
		"box_list": true, "box_update": true, "box_start": true, "box_stop": true, "box_delete": true,
		"box_exec": true, "operation_get": true, "operation_list": true,
		"operation_output_get": true, "operation_cancel": true}
	must(len(t.Tools) == len(expected), "MCP source contract changed: 13 S1+S2 public tools required")
	for _, tool := range t.Tools {
		must(expected[tool.Name], "unexpected public MCP Tool "+tool.Name)
	}
}
func reply(result *mcp.CallToolResult) map[string]any {
	must(result != nil && !result.IsError, "MCP reported Tool failure")
	b, e := json.Marshal(result.StructuredContent)
	if e != nil {
		panic(e)
	}
	var v map[string]any
	if e = json.Unmarshal(b, &v); e != nil {
		panic(e)
	}
	return v
}
func request(ctx context.Context, c *mcp.ClientSession, name string, args map[string]any) map[string]any {
	v, e := c.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if e != nil {
		panic(fmt.Errorf("%s MCP call: %w", name, e))
	}
	return reply(v)
}
func denied(ctx context.Context, c *mcp.ClientSession, name string, args map[string]any) bool {
	v, e := c.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	return e != nil || (v != nil && v.IsError)
}
func str(x map[string]any, k string) string { v, _ := x[k].(string); return v }
func child(x map[string]any, k string) map[string]any {
	v, _ := x[k].(map[string]any)
	must(v != nil, "missing structured field "+k)
	return v
}
func opid(x map[string]any) string {
	s := str(x, "operation_id")
	must(len(s) == 32, "missing original operation id")
	return s
}
func wait(ctx context.Context, c *mcp.ClientSession, id string) map[string]any {
	deadline := time.Now().Add(16 * time.Minute)
	for time.Now().Before(deadline) {
		r := child(request(ctx, c, "operation_get", map[string]any{"operation_id": id, "wait_ms": 5000}), "operation")
		switch str(r, "state") {
		case "SUCCEEDED":
			return r
		case "FAILED", "UNKNOWN", "CANCELLED":
			panic(fmt.Sprintf("original %s terminal=%s; preserve receipt, no re-dispatch", id, str(r, "state")))
		}
	}
	panic("original operation did not become final; preserve original id, never replay")
}
func checkpointPath() string { return environmental("BW_S25_CHECKPOINT") }
func readCheckpoint() map[string]string {
	buf, e := os.ReadFile(checkpointPath())
	if e != nil {
		panic(e)
	}
	var m map[string]string
	if e = json.Unmarshal(buf, &m); e != nil {
		panic(e)
	}
	return m
}
func saveCheckpoint(m map[string]string) {
	path := checkpointPath()
	b, e := json.MarshalIndent(m, "", "  ")
	if e != nil {
		panic(e)
	}
	f, e := os.OpenFile(path+".pending", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		panic(e)
	}
	if _, e = f.Write(append(b, '\n')); e != nil {
		panic(e)
	}
	if e = f.Sync(); e != nil {
		panic(e)
	}
	if e = f.Close(); e != nil {
		panic(e)
	}
	if e = os.Rename(path+".pending", path); e != nil {
		panic(e)
	}
	dir, e := os.Open(filepath.Dir(path))
	if e != nil {
		panic(e)
	}
	if e = dir.Sync(); e != nil {
		panic(e)
	}
	_ = dir.Close()
}
func connectActor(ctx context.Context, actor string) *mcp.ClientSession {
	switch actor {
	case "a":
		return session(ctx, environmental("BW_S25_ADMIN1_FILE"))
	case "b":
		return session(ctx, environmental("BW_S25_ADMIN2_FILE"))
	case "viewer":
		return session(ctx, environmental("BW_S25_VIEWER_FILE"))
	default:
		panic("unknown actor")
	}
}
func isInner() bool { return os.Getenv("BW_S25_INNER") == "1" }
func boxName() string {
	if isInner() {
		return "bw-s25-mcp-l2"
	}
	return "bw-s25-mcp-l1"
}
func firstCreateArgs() map[string]any {
	if isInner() {
		return map[string]any{"name": boxName(), "cpu": 1, "memory_mib": 512, "disk_gib": 2,
			"start": true, "nesting": false, "internet": true, "idempotency_key": "s25-inner-l2-once-20261111",
			"work_ref": "lab/s25/nested", "agent_ref": "agent-inner-alpha"}
	}
	return map[string]any{"name": boxName(), "cpu": 2, "memory_mib": 2500, "disk_gib": 3,
		"start": true, "nesting": true, "internet": true, "idempotency_key": "s25-outer-l1-once-20261111",
		"work_ref": "lab/s25/nested", "agent_ref": "agent-outer-alpha"}
}
func admit(ctx context.Context) {
	s := connectActor(ctx, "a")
	defer s.Close()
	tools(ctx, s)
	result := request(ctx, s, "box_create", firstCreateArgs())
	op := opid(result)
	saveCheckpoint(map[string]string{"layer": map[bool]string{true: "L1", false: "L0"}[isInner()],
		"name": boxName(), "create_op": op, "original_key": str(firstCreateArgs(), "idempotency_key"),
		"work_ref": "lab/s25/nested"})
	// Exit without polling / waiting: next independent Agent process must
	// reconstruct original operation from the external checkpoint, never redo.
	fmt.Printf("S25_ORIGINAL_ADMITTED_%s=%s\n", map[bool]string{true: "L2", false: "L1"}[isInner()], op)
	fmt.Println("S25_AGENT_PROCESS_DISCONNECTED_AFTER_ADMISSION=PASS")
}
func recoverOriginal(ctx context.Context) {
	m := readCheckpoint()
	s := connectActor(ctx, "b")
	defer s.Close()
	tools(ctx, s)
	final := wait(ctx, s, m["create_op"])
	uuid := str(final, "box_uuid")
	must(len(uuid) >= 32, "verified original create has no native Box UUID")
	box := child(request(ctx, s, "box_get", map[string]any{"box_id": uuid}), "box")
	must(str(box, "box_id") == uuid && str(box, "native_state") == "Running", "recover did not read original running native Box")
	if !isInner() {
		must(box["nesting"] == true, "MCP-created L1 nesting not enabled")
	}
	// The other authorized Agent observed the ORIGINAL admitted operation;
	// work/agent labels are not credentials.
	must(str(final, "work_ref") == "lab/s25/nested", "work correlation lost")
	m["uuid"] = uuid
	saveCheckpoint(m)
	s.Close()
	// Original actor reconnects and retries exactly the ORIGINAL key+intent:
	// returned durable original receipt must be identical; no native replay.
	original := connectActor(ctx, "a")
	defer original.Close()
	replay := request(ctx, original, "box_create", firstCreateArgs())
	must(opid(replay) == m["create_op"] && replay["duplicate"] == true, "reconnect redispatched native create")
	changed := firstCreateArgs()
	changed["cpu"] = 5
	must(denied(ctx, original, "box_create", changed),
		"same idempotency key with different intent was accepted")
	fmt.Println("S25_ORIGINAL_KEY_DIFFERENT_INTENT_DENIED=PASS")
	fmt.Printf("S25_RECOVERED_ORIGINAL_UUID=%s\n", uuid)
	fmt.Println("S25_SECOND_AGENT_RECOVERED_ORIGINAL_OPERATION=PASS")
	fmt.Println("S25_IDEMPOTENT_CREATE_NO_REDISPATCH=PASS")
}
func execGuest(ctx context.Context, s *mcp.ClientSession, uuid, key, script, agent string, timeout int) (map[string]any, string, int) {
	args := map[string]any{"box_id": uuid, "argv": []string{"/bin/sh", "-c", script},
		"timeout_ms": timeout, "idempotency_key": key,
		"work_ref": "lab/s25/nested", "agent_ref": agent}
	rec := request(ctx, s, "box_exec", args)
	original := opid(rec)
	// Persist immediately after admission and before any network wait. If
	// the Agent crashes or HTTP disconnects, this is the only safe original
	// identifier to read. Never infer nonexistence from a lost reply.
	m := readCheckpoint()
	m["last_guest_exec_op"] = original
	saveCheckpoint(m)
	fmt.Println("S25_GUEST_EXEC_ORIGINAL_OPERATION_ID=" + original)
	final := wait(ctx, s, original)
	out := request(ctx, s, "operation_output_get", map[string]any{"operation_id": original, "stream": "stdout", "limit": 4096})
	buf, err := base64.StdEncoding.DecodeString(str(out, "content_base64"))
	if err != nil {
		panic(err)
	}
	exit, ok := out["exit_code"].(float64)
	must(ok, "guest Exec returned no verified exit code")
	if int(exit) != 0 {
		// Diagnostic read only; never resubmit the possible Guest effect.
		// Restrict public CI logs to bounded non-secret error lines.
		stderr := ""
		if detail, e := s.CallTool(ctx, &mcp.CallToolParams{Name: "operation_output_get",
			Arguments: map[string]any{"operation_id": original, "stream": "stderr", "limit": 4096}}); e == nil && detail != nil && !detail.IsError {
			enc := str(reply(detail), "content_base64")
			b, _ := base64.StdEncoding.DecodeString(enc)
			stderr = string(b)
		}
		fmt.Fprintf(os.Stderr, "S25_GUEST_EXEC_NONZERO_EXIT=%d original=%s\n", int(exit), original)
		clean := func(v string) string {
			lines := strings.Split(v, "\n")
			if len(lines) > 16 {
				lines = lines[len(lines)-16:]
			}
			for i, line := range lines {
				if len(line) > 240 {
					line = line[:240] + "[TRUNCATED]"
				}
				low := strings.ToLower(line)
				if strings.Contains(low, "token=") || strings.Contains(low, "authorization:") ||
					strings.Contains(low, "bearer ") || strings.Contains(low, ".secret") {
					lines[i] = "[REDACTED]"
				} else {
					lines[i] = line
				}
			}
			return strings.Join(lines, " | ")
		}
		fmt.Fprintln(os.Stderr, "S25_GUEST_EXEC_STDOUT_TAIL="+clean(string(buf)))
		fmt.Fprintln(os.Stderr, "S25_GUEST_EXEC_STDERR_TAIL="+clean(stderr))
	}
	return final, string(buf), int(exit)
}
func bootstrap(ctx context.Context) {
	must(!isInner(), "outer bootstrap cannot run inside L1")
	m := readCheckpoint()
	must(m["uuid"] != "", "L1 original UUID missing")
	stage := environmental("BW_S25_STAGE_URL")
	sha := environmental("BW_S25_SETUP_SHA256")
	// Transport is a PRIVATE temporary root-only CI bridge server. The guest
	// verifies the pinned setup script digest, then setup validates candidate
	// and Agent binary digests before starting its independent L1 Incus+v1.
	cmd := "set -eu; curl -fsSL --max-time 45 " + stage + "/s25_inner_setup.sh -o /tmp/s25_inner_setup.sh; " +
		"printf '%s  %s\\n' '" + sha + "' /tmp/s25_inner_setup.sh | sha256sum -c -; " +
		"sh /tmp/s25_inner_setup.sh '" + stage + "' '" + environmental("BW_S25_BINARY_SHA256") +
		"' '" + environmental("BW_S25_AGENT_SHA256") + "' '" +
		environmental("BW_S25_PACKAGE_PROXY_URL") + "'"
	s := connectActor(ctx, "a")
	defer s.Close()
	final, out, exit := execGuest(ctx, s, m["uuid"], "s25-outer-l1-guest-bootstrap-once", cmd, "agent-outer-alpha", 900000)
	must(exit == 0, "L1 guest-side Incus/v1 bootstrap failed (review original Exec output, do not duplicate)")
	must(strings.Contains(out, "S25_INNER_READY=PASS"), "L1 installer success sentinel missing")
	m = readCheckpoint()
	m["bootstrap_op"] = str(final, "operation_id")
	saveCheckpoint(m)
	fmt.Println("S25_L1_GUEST_EXEC_NESTED_INCUS_V1_INSTALL=PASS")
}
func doubleAgent(ctx context.Context) {
	must(!isInner(), "two-client test requires L0")
	m := readCheckpoint()
	must(m["uuid"] != "", "L1 original missing")
	first := connectActor(ctx, "a")
	defer first.Close()
	second := connectActor(ctx, "b")
	defer second.Close()
	viewer := connectActor(ctx, "viewer")
	defer viewer.Close()
	work := "lab/s25/two-agents"
	// Independent agents need separate native resources for the positive
	// parallel-work proof. Competing Execs against one Box may safely refuse
	// because the worker enforces one mutable native resource at a time.
	peer := request(ctx, second, "box_create", map[string]any{
		"name": "bw-s25-mcp-peer", "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": true, "internet": true, "nesting": false,
		"idempotency_key": "s25-dual-agent-peer-create-once",
		"work_ref":        work, "agent_ref": "agent-s25-beta",
	})
	peerOp := opid(peer)
	m["peer_create_op"] = peerOp
	saveCheckpoint(m)
	peerFinal := wait(ctx, second, peerOp)
	peerUUID := str(peerFinal, "box_uuid")
	must(peerUUID != "" && peerUUID != m["uuid"], "second Agent did not receive distinct native Box")
	m["peer_uuid"] = peerUUID
	saveCheckpoint(m)
	fmt.Println("S25_SECOND_AGENT_DISTINCT_MCP_BOX=PASS")
	type output struct {
		actor, op string
		err       error
	}
	results := make(chan output, 2)
	var wg sync.WaitGroup
	for _, a := range []struct {
		c                       *mcp.ClientSession
		agent, key, text, boxID string
	}{
		{first, "agent-s25-alpha", "s25-agent-a-independent-exec", "s25-agent-alpha", m["uuid"]},
		{second, "agent-s25-beta", "s25-agent-b-independent-exec", "s25-agent-beta", peerUUID},
	} {
		wg.Add(1)
		go func(a struct {
			c                       *mcp.ClientSession
			agent, key, text, boxID string
		}) {
			defer wg.Done()
			var got output
			got.actor = a.agent
			defer func() {
				if v := recover(); v != nil {
					got.err = fmt.Errorf("%v", v)
				}
				results <- got
			}()
			rec := request(ctx, a.c, "box_exec", map[string]any{"box_id": a.boxID,
				"argv":       []string{"/bin/sh", "-c", "printf " + a.text},
				"timeout_ms": 60000, "idempotency_key": a.key, "work_ref": work, "agent_ref": a.agent})
			got.op = opid(rec)
			final := wait(ctx, a.c, got.op)
			must(str(final, "agent_ref") == a.agent && str(final, "work_ref") == work, "two-Agent attribution lost")
		}(a)
	}
	wg.Wait()
	close(results)
	ops := map[string]string{}
	for r := range results {
		if r.err != nil {
			panic(r.err)
		}
		must(r.op != "", "agent had no original operation")
		ops[r.actor] = r.op
	}
	must(len(ops) == 2 && ops["agent-s25-alpha"] != ops["agent-s25-beta"], "Agent Operations collapsed")
	must(denied(ctx, viewer, "operation_get", map[string]any{"operation_id": ops["agent-s25-alpha"]}),
		"viewer borrowed private operation by shared work label")
	must(denied(ctx, viewer, "box_create", map[string]any{"name": "bw-s25-forbidden", "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"idempotency_key": "s25-forbidden-viewer", "work_ref": work, "agent_ref": "agent-s25-alpha"}),
		"viewer used another Agent label for privilege")
	m["agent_a_exec"] = ops["agent-s25-alpha"]
	m["agent_b_exec"] = ops["agent-s25-beta"]
	saveCheckpoint(m)
	// The second Agent explicitly closes its own original temporary Box;
	// the independently verified L1 remains unchanged throughout.
	stop := request(ctx, second, "box_stop", map[string]any{"box_id": peerUUID,
		"idempotency_key": "s25-dual-agent-peer-stop-once", "work_ref": work,
		"agent_ref": "agent-s25-beta"})
	_ = wait(ctx, second, opid(stop))
	del := request(ctx, second, "box_delete", map[string]any{"box_id": peerUUID,
		"idempotency_key": "s25-dual-agent-peer-delete-once", "work_ref": work,
		"agent_ref": "agent-s25-beta"})
	_ = wait(ctx, second, opid(del))
	m["peer_stop_op"] = opid(stop)
	m["peer_delete_op"] = opid(del)
	saveCheckpoint(m)
	fmt.Println("S25_SECOND_AGENT_EXACT_PEER_CLEANUP=PASS")
	fmt.Println("S25_TWO_AUTHENTICATED_AGENT_CLIENTS_PARALLEL=PASS")
	fmt.Println("S25_SAME_WORK_DISTINCT_AGENT_OPERATIONS=PASS")
	fmt.Println("S25_VIEWER_CANNOT_BORROW_WORK_REF=PASS")
}

// quietLongPoll proves that a completed MCP request and a bounded
// quiet reader never automatically cancel an admitted native Guest Exec.
func quietLongPoll(ctx context.Context) {
	must(!isInner(), "quiet operation is scoped to isolated L0")
	m := readCheckpoint()
	a := connectActor(ctx, "a")
	defer a.Close()
	args := map[string]any{"box_id": m["uuid"],
		"argv":       []string{"/bin/sh", "-c", "sleep 5; printf s25-quiet-exec-done"},
		"timeout_ms": 15000, "idempotency_key": "s25-quiet-long-poll-once",
		"work_ref": "lab/s25/quiet", "agent_ref": "agent-outer-alpha"}
	rec := request(ctx, a, "box_exec", args)
	original := opid(rec)
	m["quiet_op"] = original
	saveCheckpoint(m)
	b := connectActor(ctx, "b")
	defer b.Close()
	early := child(request(ctx, b, "operation_get",
		map[string]any{"operation_id": original, "wait_ms": 150}), "operation")
	switch str(early, "state") {
	case "ADMITTED", "DISPATCHING", "VERIFYING":
	default:
		must(false, "quiet native Exec unexpectedly terminal before 5-second Guest sleep: "+str(early, "state"))
	}
	must(str(early, "operation_id") == original, "bounded wait lost original Operation ID")
	fmt.Println("S25_MCP_QUIET_RUNNING_BOUNDED_WAIT=PASS")
	_ = wait(ctx, b, original)
	out := request(ctx, b, "operation_output_get", map[string]any{
		"operation_id": original, "stream": "stdout", "limit": 4096,
	})
	text, err := base64.StdEncoding.DecodeString(str(out, "content_base64"))
	if err != nil {
		panic(err)
	}
	must(string(text) == "s25-quiet-exec-done",
		"quiet original completion/output lost after bounded read")
	fmt.Println("S25_MCP_QUIET_FINISH_UNCANCELLED=PASS")
}

func afterControllerRestart(ctx context.Context) {
	must(!isInner(), "restart cross-process proof is L0-only")
	m := readCheckpoint()
	must(m["uuid"] != "" && m["create_op"] != "" && m["agent_a_exec"] != "" && m["agent_b_exec"] != "",
		"original restart checkpoint incomplete")
	s := connectActor(ctx, "b")
	defer s.Close()
	tools(ctx, s)
	original := child(request(ctx, s, "operation_get", map[string]any{"operation_id": m["create_op"]}), "operation")
	must(str(original, "state") == "SUCCEEDED" && str(original, "box_uuid") == m["uuid"],
		"controller restart lost verified original create")
	for _, op := range []string{m["agent_a_exec"], m["agent_b_exec"], m["peer_delete_op"], m["quiet_op"]} {
		r := child(request(ctx, s, "operation_get", map[string]any{"operation_id": op}), "operation")
		must(str(r, "state") == "SUCCEEDED", "original Agent/cleanup receipt lost on service restart")
	}
	box := child(request(ctx, s, "box_get", map[string]any{"box_id": m["uuid"]}), "box")
	must(str(box, "box_id") == m["uuid"] && str(box, "native_state") == "Running",
		"controller restart did not independently reconcile exact native L1")
	a := connectActor(ctx, "a")
	defer a.Close()
	replay := request(ctx, a, "box_create", firstCreateArgs())
	must(opid(replay) == m["create_op"] && replay["duplicate"] == true,
		"after controller restart original create key dispatched again")
	fmt.Println("S25_CONTROLLER_RESTART_ORIGINAL_MULTI_AGENT_RECEIPTS=PASS")
	fmt.Println("S25_CONTROLLER_RESTART_NO_NATIVE_CREATE_REPLAY=PASS")
}
func rotatedViewer(ctx context.Context) {
	must(!isInner(), "token rotation assertion is L0-only")
	viewer := connectActor(ctx, "viewer")
	defer viewer.Close()
	tools(ctx, viewer)
	_ = request(ctx, viewer, "host_status", map[string]any{})
	must(denied(ctx, viewer, "box_create", map[string]any{
		"name":            "bw-s25-viewer-denied-rotated",
		"idempotency_key": "s25-viewer-rotated-denial", "cpu": 1, "memory_mib": 512, "disk_gib": 2,
	}), "new viewer token gained mutation privilege")
	fmt.Println("S25_ROTATED_VIEWER_SDK_HOST_STATUS_AND_WRITE_DENIAL=PASS")
}
func nestedExec(ctx context.Context) {
	must(isInner(), "L2 probe only inside nested L1")
	m := readCheckpoint()
	s := connectActor(ctx, "b")
	defer s.Close()
	_, out, exit := execGuest(ctx, s, m["uuid"], "s25-inner-l2-exec-zero", "printf nested-guest-exec", "agent-inner-beta", 90000)
	must(exit == 0 && strings.TrimSpace(out) == "nested-guest-exec", "L2 guest actual output mismatched")
	_, _, badExit := execGuest(ctx, s, m["uuid"], "s25-inner-l2-exec-nonzero", "exit 9", "agent-inner-beta", 90000)
	must(badExit == 9, "L2 nonzero original native exit not preserved")
	fmt.Println("S25_L1_MCP_L2_EXEC_EXIT_OUTPUT=PASS")
	// Complete the same UUID's v1 Update, Stop, Start, and post-start
	// observed Guest execution before the final MCP Stop/Delete.
	update := request(ctx, s, "box_update", map[string]any{
		"box_id": m["uuid"], "cpu": 2,
		"idempotency_key": "s25-inner-l2-update-cpu-once",
		"work_ref":        "lab/s25/nested", "agent_ref": "agent-inner-beta",
	})
	m["update_op"] = opid(update)
	saveCheckpoint(m)
	_ = wait(ctx, s, m["update_op"])
	box := child(request(ctx, s, "box_get", map[string]any{"box_id": m["uuid"]}), "box")
	must(str(box, "cpu") == "2", "L1 MCP Update CPU had no verified native post-state")
	stop := request(ctx, s, "box_stop", map[string]any{
		"box_id": m["uuid"], "idempotency_key": "s25-inner-l2-mid-stop-once",
		"work_ref": "lab/s25/nested", "agent_ref": "agent-inner-beta",
	})
	m["restart_stop_op"] = opid(stop)
	saveCheckpoint(m)
	_ = wait(ctx, s, m["restart_stop_op"])
	box = child(request(ctx, s, "box_get", map[string]any{"box_id": m["uuid"]}), "box")
	must(str(box, "native_state") == "Stopped", "L1 MCP Stop had no native stopped post-state")
	start := request(ctx, s, "box_start", map[string]any{
		"box_id": m["uuid"], "idempotency_key": "s25-inner-l2-mid-start-once",
		"work_ref": "lab/s25/nested", "agent_ref": "agent-inner-beta",
	})
	m["restart_start_op"] = opid(start)
	saveCheckpoint(m)
	_ = wait(ctx, s, m["restart_start_op"])
	box = child(request(ctx, s, "box_get", map[string]any{"box_id": m["uuid"]}), "box")
	must(str(box, "native_state") == "Running" &&
		str(box, "box_id") == m["uuid"], "restart changed UUID or not running")
	_, out, exit = execGuest(ctx, s, m["uuid"],
		"s25-inner-l2-after-start-exec-once",
		"printf s25-guest-after-start",
		"agent-inner-beta", 90000)
	must(exit == 0 && strings.TrimSpace(out) == "s25-guest-after-start",
		"restarted original L2 Guest Exec failed")
	fmt.Println("S25_L1_MCP_L2_UPDATE_STOP_START_SAME_UUID=PASS")
}
func cleanInnerViaParent(ctx context.Context) {
	must(!isInner(), "nested cleanup requires L0 authorized guest Exec")
	m := readCheckpoint()
	script := "BW_S25_ALLOWED_NESTED=1 BW_S25_INNER=1 " +
		"BW_S25_URL=http://127.0.0.1:18650/mcp " +
		"BW_S25_ADMIN1_FILE=/root/s25-admin1.secret " +
		"BW_S25_ADMIN2_FILE=/root/s25-admin2.secret " +
		"BW_S25_VIEWER_FILE=/root/s25-viewer.secret " +
		"BW_S25_CHECKPOINT=/root/s25-inner-checkpoint.json " +
		"/usr/local/bin/s25-lab-agent cleanup"
	s := connectActor(ctx, "a")
	defer s.Close()
	_, stdout, exit := execGuest(ctx, s, m["uuid"],
		"s25-outer-guest-exec-inner-l2-cleanup-once", script, "agent-outer-alpha", 180000)
	must(exit == 0 && strings.Contains(stdout, "S25_L1_MCP_STOP_DELETE=PASS"),
		"nested original L2 MCP delete did not complete; preserve original operation")
	fmt.Println("S25_L0_AUTHORIZED_GUEST_EXEC_L1_MCP_L2_CLEANUP=PASS")
}

// guestGitWorkflow uses only v1 MCP box_exec and durable operation output to
// prove BH-014 before the independent native/Git verifier reads the Guest.
func guestGitWorkflow(ctx context.Context) {
	must(!isInner(), "guest Git workflow must use L0 MCP-owned L1")
	m := readCheckpoint()
	must(m["uuid"] != "", "original L1 missing")
	s := connectActor(ctx, "a")
	defer s.Close()
	setupScript := "set -eu\ntest ! -e /root/s25-git-loop/.git\nmkdir -p /root/s25-git-loop\ncd /root/s25-git-loop\ngit init -q\ngit config user.name \"BoxWeave S2.5 Lab\"\ngit config user.email \"s25-lab@example.invalid\"\nprintf 'def double(value):\\n    return value + value + 1\\n' > calc.py\nprintf 'import unittest\\nfrom calc import double\\nclass TestCalc(unittest.TestCase):\\n    def test_double(self):\\n        self.assertEqual(double(2), 4)\\n' > test_calc.py\ngit add calc.py test_calc.py\ngit commit -q -m \"test: deliberately failing regression\"\necho S25_GIT_ORIGINAL_COMMIT=$(git rev-parse HEAD)\n"
	original, output, code := execGuest(ctx, s, m["uuid"],
		"s25-guest-git-setup-once", setupScript, "agent-outer-alpha", 120000)
	must(code == 0 && strings.Contains(output, "S25_GIT_ORIGINAL_COMMIT="), "Git setup failed")
	m["git_setup_op"] = str(original, "operation_id")
	saveCheckpoint(m)

	failScript := "set -eu\ncd /root/s25-git-loop\npython3 -B -m unittest -v test_calc\n"
	failed, _, code := execGuest(ctx, s, m["uuid"],
		"s25-guest-git-failing-test-once", failScript, "agent-outer-alpha", 120000)
	m["git_failure_op"] = str(failed, "operation_id")
	saveCheckpoint(m)
	must(code == 1, "deliberate Guest test did not fail with exit=1")
	errOutput := request(ctx, s, "operation_output_get", map[string]any{
		"operation_id": m["git_failure_op"], "stream": "stderr", "limit": 4096,
	})
	raw, err := base64.StdEncoding.DecodeString(str(errOutput, "content_base64"))
	if err != nil {
		panic(err)
	}
	must(strings.Contains(string(raw), "AssertionError: 5 != 4"),
		"original persisted native stderr lacks real regression failure")
	fmt.Println("S25_MCP_GUEST_GIT_TEST_ORIGINAL_FAILURE=PASS")

	repairScript := "set -eu\ncd /root/s25-git-loop\nsed -i 's/return value + value + 1/return value + value/' calc.py\ngit add calc.py\ngit commit -q -m \"fix: repair failing test\"\npython3 -B -m unittest -q test_calc\ntest \"$(git rev-list --count HEAD)\" = 2\ntest -z \"$(git status --porcelain)\"\necho S25_GIT_REPAIRED_COMMIT=$(git rev-parse HEAD)\necho S25_GUEST_GIT_FIX_TEST=PASS\n"
	fixed, stdout, code := execGuest(ctx, s, m["uuid"],
		"s25-guest-git-repair-and-retest-once", repairScript,
		"agent-outer-alpha", 120000)
	must(code == 0 && strings.Contains(stdout, "S25_GUEST_GIT_FIX_TEST=PASS"),
		"Guest repaired Git test did not pass")
	m["git_repair_op"] = str(fixed, "operation_id")
	saveCheckpoint(m)
	fmt.Println("S25_MCP_GUEST_GIT_FIX_AND_RETEST=PASS")
}

func cleanup(ctx context.Context) {
	m := readCheckpoint()
	must(m["uuid"] != "", "original Box missing")
	s := connectActor(ctx, "b")
	defer s.Close()
	stop := request(ctx, s, "box_stop", map[string]any{"box_id": m["uuid"],
		"idempotency_key": "s25-" + m["layer"] + "-stop-once",
		"work_ref":        "lab/s25/nested", "agent_ref": "agent-s25-cleanup"})
	_ = wait(ctx, s, opid(stop))
	del := request(ctx, s, "box_delete", map[string]any{"box_id": m["uuid"],
		"idempotency_key": "s25-" + m["layer"] + "-delete-once",
		"work_ref":        "lab/s25/nested", "agent_ref": "agent-s25-cleanup"})
	_ = wait(ctx, s, opid(del))
	m["stop_op"] = opid(stop)
	m["delete_op"] = opid(del)
	saveCheckpoint(m)
	fmt.Printf("S25_%s_MCP_STOP_DELETE=PASS\n", m["layer"])
}
func main() {
	if len(os.Args) != 2 {
		panic("usage: s25-lab-agent admit|recover|bootstrap|double-agent|nested-exec|cleanup")
	}
	if os.Getenv("GITHUB_ACTIONS") != "true" && os.Getenv("BW_S25_ALLOWED_NESTED") != "1" {
		panic("S25 isolated lab fixture only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 22*time.Minute)
	defer cancel()
	switch os.Args[1] {
	case "admit":
		admit(ctx)
	case "recover":
		recoverOriginal(ctx)
	case "bootstrap":
		bootstrap(ctx)
	case "double-agent":
		doubleAgent(ctx)
	case "quiet-long-poll":
		quietLongPoll(ctx)
	case "post-restart":
		afterControllerRestart(ctx)
	case "rotated-viewer":
		rotatedViewer(ctx)
	case "nested-exec":
		nestedExec(ctx)
	case "cleanup":
		cleanup(ctx)
	case "inner-clean":
		cleanInnerViaParent(ctx)
	case "guest-git":
		guestGitWorkflow(ctx)
	default:
		panic(errors.New("unknown LAB mode"))
	}
}
