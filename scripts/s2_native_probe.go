//go:build ignore

// S2 candidate-only native black-box probe. This tool is NOT shipped with
// BoxWeave and runs exclusively inside the authorized single-use GH VM lab.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Count wire-level progress notifications on the official MCP client.
// Immediate durable Tool admission must not fabricate progress for native work.
var nativeProgressNotifications atomic.Int64

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
	c := mcp.NewClient(&mcp.Implementation{Name: "s2-native-independent-prober", Version: "1"}, &mcp.ClientOptions{
		ProgressNotificationHandler: func(context.Context, *mcp.ProgressNotificationClientRequest) {
			nativeProgressNotifications.Add(1)
		},
	})
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

// guestRead runs actual guest argv through the candidate MCP binary. A
// nonzero guest exit is a completed native Exec, not an MCP transport error.
// Deliberate network-denial probes must never be mistaken for missing ping.
func guestRead(ctx context.Context, session *mcp.ClientSession, uuid, key, command string) (string, int) {
	request := call(ctx, session, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-c", command},
		"idempotency_key": key,
	})
	opID := id(request)
	wait(ctx, session, opID)
	result := call(ctx, session, "operation_output_get", map[string]any{
		"operation_id": opID, "stream": "stdout", "limit": 4096,
	})
	encoded := field(result, "content_base64")
	body, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		panic(err)
	}
	exit, ok := result["exit_code"].(float64)
	check(ok, "native guest network probe returned no actual exit code")
	return strings.TrimSpace(string(body)), int(exit)
}

// Wait for guest DHCP/RA to converge after native Running. This is a LAB
// readiness test, not an inference that RUNNING proves GuestReady.
func guestAddress(ctx context.Context, session *mcp.ClientSession, uuid string, family int, name string) string {
	script := fmt.Sprintf("ip -o -%d addr show dev eth0 scope global | awk 'NR==1 {split($4,a,\"/\"); print a[1]}'", family)
	for attempt := 0; attempt < 12; attempt++ {
		key := fmt.Sprintf("native-s2-address-%s-v%d-attempt-%02d", name, family, attempt)
		result, exit := guestRead(ctx, session, uuid, key, script)
		ip := net.ParseIP(result)
		if exit == 0 && ip != nil && (family == 4 && ip.To4() != nil ||
			family == 6 && ip.To4() == nil) {
			return result
		}
		time.Sleep(time.Second)
	}
	panic(fmt.Sprintf("native IPv%d guest address absent on %s; packet isolation unverified", family, name))
}

func guestGateway(ctx context.Context, session *mcp.ClientSession, uuid, name string, family int) string {
	script := fmt.Sprintf("ip -%d route show default | awk 'NR==1 {print $3}'", family)
	for attempt := 0; attempt < 12; attempt++ {
		key := fmt.Sprintf("native-s2-gateway-%s-v%d-%02d", name, family, attempt)
		result, exit := guestRead(ctx, session, uuid, key, script)
		ip := net.ParseIP(result)
		if exit == 0 && ip != nil && (family == 4 && ip.To4() != nil ||
			family == 6 && ip.To4() == nil) {
			return result
		}
		time.Sleep(time.Second)
	}
	panic(fmt.Sprintf("no IPv%d gateway on %s: cannot distinguish ACL denial from broken guest routing", family, name))
}

func nativePing(ctx context.Context, session *mcp.ClientSession, uuid, key, destination string, family int) int {
	ip := net.ParseIP(destination)
	check(ip != nil && (family == 4 && ip.To4() != nil || family == 6 && ip.To4() == nil),
		"guest ping destination not an inspected address")
	command := fmt.Sprintf("ping -%d -n -c 1 -W 2 %s", family, destination)
	if family == 6 && ip.IsLinkLocalUnicast() {
		// A link-local gateway must be addressed with an explicit egress
		// interface, otherwise "invalid scope" looks like an ACL denial.
		command = fmt.Sprintf("ping -6 -I eth0 -n -c 1 -W 2 %s", destination)
	}
	_, exit := guestRead(ctx, session, uuid, key, command)
	return exit
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
		// A real SDK ListTools over HTTP must advertise safety/caching
		// annotations, not just tool names in local registration code.
		read := map[string]bool{
			"host_status": true, "box_list": true, "box_get": true,
			"operation_get": true, "operation_list": true,
			"operation_output_get": true,
		}[t.Name]
		check(t.Annotations != nil && t.Annotations.ReadOnlyHint == read &&
			t.Annotations.IdempotentHint == read &&
			t.Annotations.OpenWorldHint != nil && !*t.Annotations.OpenWorldHint &&
			t.Annotations.DestructiveHint != nil,
			"actual advertised MCP annotations do not match tool mutation/reader contract: "+t.Name)
		if read {
			check(!*t.Annotations.DestructiveHint,
				"read-only MCP Tool advertised destructive action: "+t.Name)
		}
	}
	fmt.Println("S2_NATIVE_MCP_DISCOVERY_13_OF_13=PASS")
	fmt.Println("S2_NATIVE_MCP_TOOL_ANNOTATIONS_ACTUALLY_ADVERTISED=PASS")

	denied, err := view.CallTool(ctx, &mcp.CallToolParams{Name: "box_create",
		Arguments: map[string]any{"name": "s2-denied", "idempotency_key": "native-viewer-create-denied-01"}})
	check(err != nil || denied != nil && denied.IsError, "viewer created native Box")
	fmt.Println("S2_NATIVE_VIEWER_CREATE_REFUSED=PASS")

	name := "bw-s2-ghci-110a"
	key := "native-s2-create-once-20261110"
	args := map[string]any{"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": false, "idempotency_key": key, "work_ref": "lab/s2", "agent_ref": "prober-01"}
	// Official SDK progress-token request: native asynchronous admission is
	// immediately durable and is not a long-lived MCP progress stream.
	createWithProgress := &mcp.CallToolParams{Name: "box_create", Arguments: args}
	createWithProgress.SetProgressToken("s2-native-create-optional-progress-001")
	createResponse, createErr := admin.CallTool(ctx, createWithProgress)
	check(createErr == nil && createResponse != nil && !createResponse.IsError,
		"progress-token native create failed immediate durable admission")
	created := decoded(createResponse)
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
	check(box["internet"] == true,
		"omitted internet must create a Box with effective egress allow")
	fmt.Println("S2_NATIVE_CREATE_DEFAULT_INTERNET_ALLOW=PASS")
	fmt.Println("S2_NATIVE_CREATE_STOPPED_UUID=PASS")

	noopStop := call(ctx, admin, "box_stop", map[string]any{"box_id": uuid,
		"idempotency_key": "native-s2-already-stopped-noop-001"})
	noopStopResult := wait(ctx, admin, id(noopStop))
	check(field(noopStopResult, "native_operation_id") == "",
		"already-stopped no-op invented a native Operation ID")
	fmt.Println("S2_NATIVE_STOP_ALREADY_STOPPED_NOOP=PASS")

	// Real client->HTTP MCP->typed server metadata extraction, not a
	// fabricated in-process CallToolRequest. A non-meta legacy caller is
	// already checked above; metadata labels are attribution, not auth.
	metaStop, metaErr := admin.CallTool(ctx, &mcp.CallToolParams{
		Meta: mcp.Meta{
			"io.boxweave.work_ref":  "lab/native-meta",
			"io.boxweave.agent_ref": "native-agent-meta",
		},
		Name: "box_stop",
		Arguments: map[string]any{
			"box_id": uuid, "idempotency_key": "native-s2-meta-attribution-stop-01",
		},
	})
	check(metaErr == nil && metaStop != nil && !metaStop.IsError,
		"official Go SDK metadata-only request rejected unexpectedly")
	metaID := id(decoded(metaStop))
	metaReceipt := wait(ctx, admin, metaID)
	check(field(metaReceipt, "work_ref") == "lab/native-meta" &&
		field(metaReceipt, "agent_ref") == "native-agent-meta",
		"metadata labels were not immutably stored in original receipt")
	metaPage := child(call(ctx, admin, "operation_list", map[string]any{
		"work_ref": "lab/native-meta", "agent_ref": "native-agent-meta"}), "page")
	foundMeta := false
	if records, ok := metaPage["operations"].([]any); ok {
		for _, raw := range records {
			if op, ok := raw.(map[string]any); ok && field(op, "operation_id") == metaID {
				foundMeta = true
			}
		}
	}
	check(foundMeta, "metadata-correlated original cannot be found by authorized list")
	conflictingMeta, metaConflictErr := admin.CallTool(ctx, &mcp.CallToolParams{
		Meta: mcp.Meta{"io.boxweave.work_ref": "lab/native-meta"},
		Name: "box_stop",
		Arguments: map[string]any{
			"box_id": uuid, "idempotency_key": "native-s2-meta-mismatch-deny-01",
			"work_ref": "lab/conflicting-client-arg",
		},
	})
	check(metaConflictErr != nil || conflictingMeta != nil && conflictingMeta.IsError,
		"metadata/argument mismatch escaped typed conflict validation")
	fmt.Println("S2_NATIVE_SDK_META_FALLBACK_CONFLICT_AND_INDEXED_SEARCH=PASS")

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

	internetOff := call(ctx, admin, "box_update", map[string]any{
		"box_id": uuid, "internet": false,
		"idempotency_key": "native-s2-internet-explicit-deny-001"})
	wait(ctx, admin, id(internetOff))
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(box["internet"] == false && field(box, "cpu") == "2" &&
		field(box, "memory") == "768MiB" && field(box, "root_disk_size") == "2GiB",
		"per-box internet=false lost typed resource config or failed native readback")
	internetOn := call(ctx, admin, "box_update", map[string]any{
		"box_id": uuid, "internet": true,
		"idempotency_key": "native-s2-internet-explicit-allow-001"})
	wait(ctx, admin, id(internetOn))
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(box["internet"] == true && field(box, "cpu") == "2" &&
		field(box, "memory") == "768MiB" && field(box, "root_disk_size") == "2GiB",
		"per-box internet=true lost typed resource config or failed native readback")
	fmt.Println("S2_NATIVE_INTERNET_EGRESS_FALSE_TRUE_READBACK_PRESERVES_RESOURCES=PASS")

	grown := call(ctx, admin, "box_update", map[string]any{
		"box_id": uuid, "disk_gib": 3,
		"idempotency_key": "native-s2-grow-owned-disk-01"})
	wait(ctx, admin, id(grown))
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(field(box, "root_disk_size") == "3GiB" && field(box, "cpu") == "2",
		"disk growth failed or wiped CPU setting")
	fmt.Println("S2_NATIVE_DISK_GROWTH_PRESERVES_CPU=PASS")

	nestingOn := call(ctx, admin, "box_update", map[string]any{
		"box_id": uuid, "nesting": true,
		"idempotency_key": "native-s2-opt-in-nesting-on-0001"})
	wait(ctx, admin, id(nestingOn))
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(box["nesting"] == true && field(box, "cpu") == "2",
		"approved native nesting did not take effect or erased CPU")
	nestingOff := call(ctx, admin, "box_update", map[string]any{
		"box_id": uuid, "nesting": false,
		"idempotency_key": "native-s2-opt-in-nesting-off-001"})
	wait(ctx, admin, id(nestingOff))
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(box["nesting"] == false && field(box, "root_disk_size") == "3GiB",
		"explicitly disabled nesting not effective or erased disk size")
	fmt.Println("S2_NATIVE_OPT_IN_NESTING_ENABLE_DISABLE=PASS")

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

	noopStart := call(ctx, admin, "box_start", map[string]any{"box_id": uuid,
		"idempotency_key": "native-s2-already-running-noop-001"})
	noopStartResult := wait(ctx, admin, id(noopStart))
	check(field(noopStartResult, "native_operation_id") == "",
		"already-running no-op invented a native Operation ID")
	fmt.Println("S2_NATIVE_START_ALREADY_RUNNING_NOOP=PASS")

	// S2 first-start packet gate: both Boxes are created through official
	// MCP calls; only the installed non-root Project is used. The existing
	// native shared ACL has no peer-allow rules. A/B must each reach their
	// assigned bridge gateway (working routing), but not one another.
	peerCreate := call(ctx, admin, "box_create", map[string]any{
		"name": "bw-s2-peer-native-110c", "cpu": 1, "memory_mib": 512,
		"disk_gib": 2, "start": true, "internet": true,
		"idempotency_key": "native-s2-second-private-box-start-001",
	})
	peerReceipt := wait(ctx, admin, id(peerCreate))
	peerUUID := field(peerReceipt, "box_uuid")
	check(peerUUID != "" && peerUUID != uuid,
		"new native peer did not get its separate true incarnation")
	peerBox := child(call(ctx, admin, "box_get", map[string]any{"box_id": peerUUID}), "box")
	check(field(peerBox, "native_state") == "Running" && peerBox["internet"] == true,
		"native first-start peer not running with internet=true")
	for _, entry := range []struct{ boxID, label string }{{uuid, "a"}, {peerUUID, "b"}} {
		_, pingExit := guestRead(ctx, admin, entry.boxID,
			"native-s2-ping-capability-"+entry.label+"-01", "command -v ping")
		check(pingExit == 0, "guest ping missing; cannot count missing executable as ACL denial")
	}
	a4 := guestAddress(ctx, admin, uuid, 4, "a")
	a6 := guestAddress(ctx, admin, uuid, 6, "a")
	b4 := guestAddress(ctx, admin, peerUUID, 4, "b")
	b6 := guestAddress(ctx, admin, peerUUID, 6, "b")
	check(a4 != b4 && a6 != b6, "two native guest NICs inherited the same address")
	ga4 := guestGateway(ctx, admin, uuid, "a", 4)
	gb4 := guestGateway(ctx, admin, peerUUID, "b", 4)
	ga6 := guestGateway(ctx, admin, uuid, "a", 6)
	gb6 := guestGateway(ctx, admin, peerUUID, "b", 6)
	check(nativePing(ctx, admin, uuid, "native-s2-gateway-a4-positive", ga4, 4) == 0 &&
		nativePing(ctx, admin, peerUUID, "native-s2-gateway-b4-positive", gb4, 4) == 0,
		"baseline egress cannot even reach each guest's native bridge gateway")
	check(nativePing(ctx, admin, uuid, "native-s2-gateway-a6-positive", ga6, 6) == 0 &&
		nativePing(ctx, admin, peerUUID, "native-s2-gateway-b6-positive", gb6, 6) == 0,
		"IPv6 baseline cannot reach assigned native gateway; no valid egress control")
	fmt.Println("S2_NATIVE_PACKET_IPV6_GATEWAY_BASELINE=PASS")
	check(nativePing(ctx, admin, uuid, "native-s2-private-a-to-b-v4-deny", b4, 4) == 1 &&
		nativePing(ctx, admin, peerUUID, "native-s2-private-b-to-a-v4-deny", a4, 4) == 1,
		"default ingress ACL did not deny IPv4 between owned Boxes")
	check(nativePing(ctx, admin, uuid, "native-s2-private-a-to-b-v6-deny", b6, 6) == 1 &&
		nativePing(ctx, admin, peerUUID, "native-s2-private-b-to-a-v6-deny", a6, 6) == 1,
		"default ingress ACL did not deny IPv6 between owned Boxes")
	fmt.Println("S2_NATIVE_PACKET_IPV4_IPV6_SAME_BRIDGE_DEFAULT_DENY=PASS")

	// Optional external WAN control, never part of a fake PASS. A known
	// fixed HTTPS endpoint with guest curl proves real public routing only
	// when online control succeeds *before* and *after* the offline test.
	// Some ephemeral hosted runners cannot provide guest WAN reachability.
	externalSupported := false
	_, hasCurl := guestRead(ctx, admin, uuid, "native-s2-wan-curl-available-01",
		"command -v curl")
	if hasCurl == 0 {
		body, status := guestRead(ctx, admin, uuid, "native-s2-wan-online-before-01",
			"curl -4 -L -sS -o /dev/null -w '%{http_code}' --max-time 7 --connect-timeout 5 https://example.com")
		if status == 0 && strings.HasPrefix(body, "2") {
			externalSupported = true
		} else {
			fmt.Println("S2_NATIVE_EXTERNAL_IPV4_WAN_BASELINE=UNVERIFIED")
		}
	} else {
		fmt.Println("S2_NATIVE_EXTERNAL_IPV4_WAN_CURL=UNVERIFIED_MISSING_CLIENT")
	}

	// Real IPv4 traffic must change with the per-Box internet flag, while
	// the separate private-peer default ingress denial remains unchanged.
	offline := call(ctx, admin, "box_update", map[string]any{
		"box_id": uuid, "internet": false,
		"idempotency_key": "native-s2-running-offline-packets-01",
	})
	wait(ctx, admin, id(offline))
	check(nativePing(ctx, admin, uuid,
		"native-s2-running-offline-gateway-denied", ga4, 4) != 0,
		"internet=false still sent IPv4 packets to native gateway")
	check(nativePing(ctx, admin, uuid,
		"native-s2-running-offline-ipv6-gateway-denied", ga6, 6) != 0,
		"internet=false still sent IPv6 packets to native gateway")
	if externalSupported {
		_, publicOfflineExit := guestRead(ctx, admin, uuid,
			"native-s2-wan-offline-01",
			"curl -4 -L -sS -o /dev/null -w '%{http_code}' --max-time 7 --connect-timeout 5 https://example.com")
		check(publicOfflineExit != 0, "public HTTPS reached WAN despite internet=false")
	}
	online := call(ctx, admin, "box_update", map[string]any{
		"box_id": uuid, "internet": true,
		"idempotency_key": "native-s2-running-online-packets-01",
	})
	wait(ctx, admin, id(online))
	check(nativePing(ctx, admin, uuid,
		"native-s2-running-online-gateway-restored", ga4, 4) == 0 &&
		nativePing(ctx, admin, uuid,
			"native-s2-running-online-ipv6-gateway-restored", ga6, 6) == 0,
		"internet=true did not restore real native IPv4 AND IPv6 gateway egress")
	fmt.Println("S2_NATIVE_PACKET_INTERNET_TOGGLE_IPV4_IPV6_GATEWAY_EGRESS=PASS")
	check(nativePing(ctx, admin, uuid,
		"native-s2-peer-still-denied-after-internet-on-v4", b4, 4) == 1 &&
		nativePing(ctx, admin, uuid,
			"native-s2-peer-still-denied-after-internet-on-v6", b6, 6) == 1,
		"re-enabling internet bypassed default private peer IPv4/IPv6 ingress")
	fmt.Println("S2_NATIVE_PACKET_INTERNET_ON_DOES_NOT_UNBLOCK_PEERS=PASS")
	if externalSupported {
		publicOnline, publicExit := guestRead(ctx, admin, uuid,
			"native-s2-wan-online-after-01",
			"curl -4 -L -sS -o /dev/null -w '%{http_code}' --max-time 7 --connect-timeout 5 https://example.com")
		check(publicExit == 0 && strings.HasPrefix(publicOnline, "2"),
			"public WAN did not recover after internet=true")
		fmt.Println("S2_NATIVE_PUBLIC_WAN_HTTPS_TOGGLE=PASS")
	}
	peerStop := call(ctx, admin, "box_stop", map[string]any{
		"box_id": peerUUID, "idempotency_key": "native-s2-peer-clean-stop-01"})
	wait(ctx, admin, id(peerStop))
	peerDelete := call(ctx, admin, "box_delete", map[string]any{
		"box_id": peerUUID, "idempotency_key": "native-s2-peer-clean-delete-01"})
	wait(ctx, admin, id(peerDelete))
	fmt.Println("S2_NATIVE_PACKET_PEER_EXACT_CLEANUP=PASS")

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

	// Read cap is a public API budget, not proof that Incus truncated its
	// underlying recorded files. Produce >16MiB on each real guest stream,
	// then demand an exactly bounded page and explicit truncation marker.
	large := call(ctx, admin, "box_exec", map[string]any{
		"box_id": uuid, "argv": []string{"/bin/sh", "-c",
			"head -c 16777220 /dev/zero | tr '\\000' Z; head -c 16777220 /dev/zero | tr '\\000' Z >&2"},
		"idempotency_key": "native-s2-dual-stream-over-16m-001",
	})
	largeID := id(large)
	wait(ctx, admin, largeID)
	for _, stream := range []string{"stdout", "stderr"} {
		bounded := call(ctx, admin, "operation_output_get", map[string]any{
			"operation_id": largeID, "stream": stream,
			"offset": 16777200, "limit": 65536,
		})
		chunk, e := base64.StdEncoding.DecodeString(field(bounded, "content_base64"))
		if e != nil {
			panic(e)
		}
		fmt.Printf("S2_NATIVE_DUAL_STREAM_META stream=%s got=%d ended=%v truncated=%v next=%v exit=%v\n",
			stream, len(chunk), bounded["ended"], bounded["truncated"], bounded["next_offset"], bounded["exit_code"])
		check(len(chunk) == 16 && strings.Trim(string(chunk), "Z") == "" &&
			bounded["ended"] == true && bounded["truncated"] == true &&
			bounded["next_offset"] == float64(16777216),
			"real recorded "+stream+" breached 16MiB public read cap or hid truncation")
	}
	tooLarge, tooLargeErr := admin.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_output_get", Arguments: map[string]any{
			"operation_id": largeID, "stream": "stdout",
			"offset": 16777217, "limit": 1,
		}})
	check(tooLargeErr != nil || tooLarge != nil && tooLarge.IsError,
		"oversized native recorded output offset unexpectedly accepted")
	fmt.Println("S2_NATIVE_DUAL_STREAM_16M_OUTPUT_READ_BUDGET=PASS")

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

	// A prior completed stop must not permanently lock the same UUID.
	// Its duplicate key is immutable, but subsequent legitimate Start/Stop
	// with distinct keys must really reach native Incus again.
	stopReplay := call(ctx, admin, "box_stop", map[string]any{"box_id": uuid,
		"idempotency_key": "native-s2-stop-0000000001"})
	check(id(stopReplay) == id(stop) && stopReplay["duplicate"] == true,
		"original Stop key was wrongly re-executed")
	secondStart := call(ctx, admin, "box_start", map[string]any{"box_id": uuid,
		"idempotency_key": "native-s2-start-second-cycle-001"})
	wait(ctx, admin, id(secondStart))
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(field(box, "native_state") == "Running", "second cycle failed to start original Box")
	secondStop := call(ctx, admin, "box_stop", map[string]any{"box_id": uuid,
		"idempotency_key": "native-s2-stop-second-cycle-001"})
	wait(ctx, admin, id(secondStop))
	box = child(call(ctx, admin, "box_get", map[string]any{"box_id": uuid}), "box")
	check(field(box, "native_state") == "Stopped", "second cycle failed to stop original Box")
	check(id(secondStop) != id(stop) && id(secondStart) != id(stop),
		"new keys returned stale completed Operation instead of native lifecycle")
	fmt.Println("S2_NATIVE_STOP_START_STOP_DISTINCT_KEYS=PASS")

	// Cursor comes from actual SQLite history on a real restricted native
	// service, not a fake in-process store. Its scope must be bound to
	// authenticated actor/layer/filter and cannot be transplanted.
	first := child(call(ctx, admin, "operation_list", map[string]any{
		"limit": 2}), "page")
	firstRows, ok := first["operations"].([]any)
	check(ok && len(firstRows) == 2, "first operation history page was not bounded")
	cursor := field(first, "next_cursor")
	check(cursor != "", "cursor missing despite known additional operations")
	second := child(call(ctx, admin, "operation_list", map[string]any{
		"limit": 2, "cursor": cursor}), "page")
	secondRows, ok := second["operations"].([]any)
	check(ok && len(secondRows) != 0 && len(secondRows) <= 2,
		"cursor continuation returned an empty/unbounded page")
	seen := map[string]bool{}
	for _, raw := range firstRows {
		if row, ok := raw.(map[string]any); ok {
			seen[field(row, "operation_id")] = true
		}
	}
	for _, raw := range secondRows {
		row, ok := raw.(map[string]any)
		check(ok && field(row, "operation_id") != "" &&
			!seen[field(row, "operation_id")],
			"cursor returned repeated original Operation")
	}
	mismatched, mismatchErr := admin.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_list", Arguments: map[string]any{
			"limit": 2, "cursor": cursor, "work_ref": "lab/native-meta"}})
	check(mismatchErr != nil || mismatched != nil && mismatched.IsError,
		"cursor from another filter was accepted")
	borrowed, borrowedErr := view.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_list", Arguments: map[string]any{
			"limit": 2, "cursor": cursor}})
	check(borrowedErr != nil || borrowed != nil && borrowed.IsError,
		"viewer reused administrator's HMAC cursor")
	forged, forgedErr := admin.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_list", Arguments: map[string]any{
			"limit": 2, "cursor": cursor + "x"}})
	check(forgedErr != nil || forged != nil && forged.IsError,
		"tampered operation-list cursor accepted")
	fmt.Println("S2_NATIVE_PRIVATE_INDEXED_CURSOR_TAMPER_AND_PAGINATION=PASS")

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

	// A native name may be reused only by a NEW independently verified
	// incarnation. The tombstoned UUID and original Delete key must never
	// acquire authority over the new guest.
	reused := call(ctx, admin, "box_create", map[string]any{
		"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2, "start": false,
		"internet":        false,
		"idempotency_key": "native-s2-reused-name-fresh-uuid-01"})
	reuseOutcome := wait(ctx, admin, id(reused))
	newUUID := field(reuseOutcome, "box_uuid")
	check(newUUID != "" && newUUID != uuid,
		"same-name new Box adopted the old native UUID or failed to create")
	stale, staleErr := admin.CallTool(ctx, &mcp.CallToolParams{Name: "box_get",
		Arguments: map[string]any{"box_id": uuid}})
	check(staleErr != nil || stale != nil && stale.IsError,
		"old tombstoned UUID accessed a new same-name Box")
	priorDelete := call(ctx, admin, "box_delete", map[string]any{"box_id": uuid,
		"idempotency_key": "native-s2-delete-00000001"})
	check(id(priorDelete) == id(del) && priorDelete["duplicate"] == true,
		"original Delete key lost immutability after name reuse")
	newBox := child(call(ctx, admin, "box_get", map[string]any{"box_id": newUUID}), "box")
	check(field(newBox, "box_id") == newUUID &&
		field(newBox, "native_state") == "Stopped" && newBox["internet"] == false,
		"old Delete incorrectly removed or modified new incarnation")
	lastDelete := call(ctx, admin, "box_delete", map[string]any{"box_id": newUUID,
		"idempotency_key": "native-s2-delete-new-incarnation-01"})
	wait(ctx, admin, id(lastDelete))
	fmt.Println("S2_NATIVE_NAME_REUSE_OLD_UUID_AND_DELETE_KEY_DENIED=PASS")

	// The original SDK request had a legitimate progress token. S2 does not
	// advertise or send synthetic progress for detached native operations.
	// A finite post-completion grace period also catches delayed notifications.
	time.Sleep(200 * time.Millisecond)
	check(nativeProgressNotifications.Load() == 0,
		"native async operation emitted unexpected MCP progress notification")
	fmt.Println("S2_NATIVE_ASYNC_PROGRESS_OPTIONAL_NO_FAKE_EVENTS=PASS")
	fmt.Println("S2_NATIVE_END_TO_END=PASS")
}
