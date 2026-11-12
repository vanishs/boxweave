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
	final := wait(ctx, s, original)
	out := request(ctx, s, "operation_output_get", map[string]any{"operation_id": original, "stream": "stdout", "limit": 4096})
	buf, err := base64.StdEncoding.DecodeString(str(out, "content_base64"))
	if err != nil {
		panic(err)
	}
	exit, ok := out["exit_code"].(float64)
	must(ok, "guest Exec returned no verified exit code")
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
		"' '" + environmental("BW_S25_AGENT_SHA256") + "'"
	s := connectActor(ctx, "a")
	defer s.Close()
	final, out, exit := execGuest(ctx, s, m["uuid"], "s25-outer-l1-guest-bootstrap-once", cmd, "agent-outer-alpha", 900000)
	must(exit == 0, "L1 guest-side Incus/v1 bootstrap failed (review original Exec output, do not duplicate)")
	must(strings.Contains(out, "S25_INNER_READY=PASS"), "L1 installer success sentinel missing")
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
	case "nested-exec":
		nestedExec(ctx)
	case "cleanup":
		cleanup(ctx)
	case "inner-clean":
		cleanInnerViaParent(ctx)
	default:
		panic(errors.New("unknown LAB mode"))
	}
}
