//go:build ignore

// Original native Create ACK-loss black-box verifier using the official MCP
// SDK client against the installed BoxWeave candidate and a one-use Incus VM.
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

const (
	name = "bw-s2-ack-native-110d"
	key  = "native-s2-create-ack-dropped-once-001"
)

type saved struct {
	OriginalID string `json:"original_id"`
}

type authTransport struct{ token string }

func (a authTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c := r.Clone(r.Context())
	c.Header.Set("Authorization", "Bearer "+a.token)
	return http.DefaultTransport.RoundTrip(c)
}

func require(ok bool, msg string) {
	if !ok {
		panic("S2_ACK_LOSS_ACCEPTANCE_FAILED: " + msg)
	}
}

func session(ctx context.Context) *mcp.ClientSession {
	addr, secret := os.Getenv("BW_S2_MCP_URL"), os.Getenv("BW_S2_ADMIN_TOKEN")
	require(strings.HasPrefix(addr, "http://127.0.0.1:18652/") && secret != "",
		"only private disposable MCP listener and explicit authenticated actor allowed")
	client := mcp.NewClient(&mcp.Implementation{Name: "s2-native-ack-loss-verifier", Version: "1"}, nil)
	out, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: addr, HTTPClient: &http.Client{Transport: authTransport{token: secret}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		panic(err)
	}
	return out
}

func invoke(ctx context.Context, s *mcp.ClientSession, tool string, args map[string]any) (map[string]any, error) {
	r, err := s.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		return nil, err
	}
	if r == nil || r.IsError {
		return nil, fmt.Errorf("tool %s refused: %+v", tool, r)
	}
	b, err := json.Marshal(r.StructuredContent)
	if err != nil {
		return nil, err
	}
	var parsed map[string]any
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func field(m map[string]any, k string) string {
	out, _ := m[k].(string)
	return out
}

func child(m map[string]any, k string) map[string]any {
	out, _ := m[k].(map[string]any)
	return out
}

func createArgs() map[string]any {
	return map[string]any{
		"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": false, "internet": true, "idempotency_key": key,
		"work_ref": "lab/lost-native-ack", "agent_ref": "ack-verifier",
	}
}

func checkUnknown(ctx context.Context, s *mcp.ClientSession, id string) map[string]any {
	var seen map[string]any
	for n := 0; n < 100; n++ {
		reply, err := invoke(ctx, s, "operation_get", map[string]any{
			"operation_id": id, "wait_ms": 300,
		})
		if err != nil {
			panic(err)
		}
		seen = child(reply, "operation")
		switch field(seen, "state") {
		case "UNKNOWN":
			require(field(seen, "native_operation_id") == "" &&
				field(seen, "error_class") == "NATIVE_OUTCOME_AMBIGUOUS" &&
				field(child(seen, "next_action"), "kind") == "HOST_LOCAL_REVIEW",
				"ACK-loss original lacked safe UNKNOWN lock or lied about native ID")
			return seen
		case "FAILED", "SUCCEEDED", "CANCELLED":
			panic(fmt.Sprintf("ACK-loss original incorrectly settled: %+v", seen))
		}
	}
	panic("original ACK loss never reached terminal safe UNKNOWN")
}

func replayMustNotRedispatch(ctx context.Context, s *mcp.ClientSession, originalID string) {
	replayed, err := invoke(ctx, s, "box_create", createArgs())
	if err != nil {
		panic(err)
	}
	require(field(replayed, "operation_id") == originalID &&
		replayed["duplicate"] == true, "same request key lost original UNKNOWN receipt")
	changed := createArgs()
	changed["idempotency_key"] = "native-s2-new-key-must-not-dispatch-001"
	unexpected, err := invoke(ctx, s, "box_create", changed)
	require(err != nil, fmt.Sprintf("different-key native create bypassed UNKNOWN name lock: %+v", unexpected))
	checkUnknown(ctx, s, originalID)
}

func testStaleNativeETag(ctx context.Context, s *mcp.ClientSession) {
	const fixture = "bw-s2-etag-native-110e"
	created, err := invoke(ctx, s, "box_create", map[string]any{
		"name": fixture, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": false, "internet": true,
		"idempotency_key": "native-s2-stale-etag-create-once-001",
	})
	if err != nil {
		panic(err)
	}
	originalCreate := field(created, "operation_id")
	require(originalCreate != "", "native stale-ETag fixture was not admitted")
	for n := 0; n < 100; n++ {
		ret, err := invoke(ctx, s, "operation_get", map[string]any{"operation_id": originalCreate, "wait_ms": 400})
		if err != nil {
			panic(err)
		}
		receipt := child(ret, "operation")
		if field(receipt, "state") == "SUCCEEDED" {
			break
		}
		if field(receipt, "state") == "FAILED" || field(receipt, "state") == "UNKNOWN" {
			panic(fmt.Sprintf("native stale-ETag fixture failed create: %+v", receipt))
		}
		if n == 99 {
			panic("native stale-ETag fixture create never completed")
		}
	}
	boxID := field(childMustBox(ctx, s, fixture), "box_id")
	require(boxID != "", "no exact owned UUID for ETag fixture")
	// The proxy advances this exact owned native instance ETag between
	// BoxWeave's GET and PUT. Its original stale PUT must fail closed.
	changes := map[string]any{
		"box_id": boxID,
		"cpu":    2, "idempotency_key": "native-s2-stale-etag-update-001",
	}
	reply, err := invoke(ctx, s, "box_update", changes)
	if err != nil {
		panic(err)
	}
	oid := field(reply, "operation_id")
	require(oid != "", "stale native ETag original Update not admitted")
	checkUnknown(ctx, s, oid)
	again, err := invoke(ctx, s, "box_update", changes)
	if err != nil {
		panic(err)
	}
	require(field(again, "operation_id") == oid && again["duplicate"] == true,
		"original stale native ETag operation not returned on same-key replay")
	mismatch := map[string]any{
		"box_id": changes["box_id"], "cpu": 3,
		"idempotency_key": "native-s2-stale-etag-different-key-deny-01",
	}
	rejected, err := invoke(ctx, s, "box_update", mismatch)
	require(err != nil, fmt.Sprintf("stale ETag UNKNOWN lock bypassed with new key: %+v", rejected))
	fmt.Println("S2_NATIVE_STALE_ETAG_ORIGINAL_UNKNOWN_AND_LOCKED=PASS")
}

func childMustBox(ctx context.Context, s *mcp.ClientSession, name string) map[string]any {
	page, err := invoke(ctx, s, "box_list", map[string]any{})
	if err != nil {
		panic(err)
	}
	raw, ok := page["boxes"].([]any)
	require(ok, "owned Box list absent")
	for _, v := range raw {
		if row, ok := v.(map[string]any); ok && field(row, "name") == name {
			return row
		}
	}
	panic("exact owned ETag fixture missing from authorized list")
}

type savedUpdate struct {
	OriginalID string `json:"original_id"`
	BoxID      string `json:"box_id"`
}

func updateAckArgs(boxID string) map[string]any {
	return map[string]any{
		"box_id": boxID, "cpu": 2,
		"idempotency_key": "native-s2-update-accepted-ack-dropped-once-001",
	}
}

func checkUpdateAckLocked(ctx context.Context, s *mcp.ClientSession, original savedUpdate) {
	checkUnknown(ctx, s, original.OriginalID)
	again, err := invoke(ctx, s, "box_update", updateAckArgs(original.BoxID))
	if err != nil || field(again, "operation_id") != original.OriginalID || again["duplicate"] != true {
		panic(fmt.Sprintf("accepted native Update lost original receipt: %+v %v", again, err))
	}
	mismatch := map[string]any{
		"box_id": original.BoxID, "cpu": 3,
		"idempotency_key": "native-s2-update-new-key-must-be-denied-001",
	}
	rejected, err := invoke(ctx, s, "box_update", mismatch)
	require(err != nil, fmt.Sprintf("new-key Update bypassed original unknown: %+v", rejected))
}

func testAcceptedUpdateAckLoss(ctx context.Context, s *mcp.ClientSession, resume bool) {
	const fixture = "bw-s2-update-ack-native-110f"
	file := os.Getenv("BW_S2_UPDATE_ACK_STATE_FILE")
	require(strings.HasPrefix(file, "/tmp/bw-v1-s2-ghci.") &&
		strings.HasSuffix(file, "/update-ack-original.json"), "disposable Update ACK checkpoint required")
	if resume {
		raw, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}
		var original savedUpdate
		if err := json.Unmarshal(raw, &original); err != nil {
			panic(err)
		}
		require(original.OriginalID != "" && original.BoxID != "", "original Update witness missing")
		checkUpdateAckLocked(ctx, s, original)
		fmt.Println("S2_NATIVE_UPDATE_ACCEPTED_LOST_ACK_RESTART_NO_REPLAY=PASS")
		return
	}
	created, err := invoke(ctx, s, "box_create", map[string]any{
		"name": fixture, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": false, "internet": true,
		"idempotency_key": "native-s2-update-ack-fixture-create-001",
	})
	if err != nil {
		panic(err)
	}
	createID := field(created, "operation_id")
	require(createID != "", "new native Update fixture missing original create")
	// An earlier ACK-loss test was deliberately cleaned up by the one-use
	// administrator, while its own journal still retains a historical claim.
	// box_list intentionally refuses to hide that missing native UUID. Read
	// only this just-finished original Create operation, which atomically
	// records the exact owned Box UUID on independent successful postread.
	var boxID string
	for n := 0; n < 120; n++ {
		ret, err := invoke(ctx, s, "operation_get", map[string]any{
			"operation_id": createID, "wait_ms": 400,
		})
		if err != nil {
			panic(err)
		}
		r := child(ret, "operation")
		if field(r, "state") == "SUCCEEDED" {
			boxID = field(r, "box_uuid")
			break
		}
		require(field(r, "state") != "UNKNOWN" && field(r, "state") != "FAILED" &&
			field(r, "state") != "CANCELLED", "Update ACK fixture was not safely created")
		require(n < 119, "native Update fixture never completed")
	}
	require(boxID != "", "verified original Create receipt lacks exact Box UUID")
	update, err := invoke(ctx, s, "box_update", updateAckArgs(boxID))
	if err != nil {
		panic(err)
	}
	original := savedUpdate{OriginalID: field(update, "operation_id"), BoxID: boxID}
	require(original.OriginalID != "", "native Update not durably admitted before ACK loss")
	checkUpdateAckLocked(ctx, s, original)
	raw, err := json.Marshal(original)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(file, raw, 0600); err != nil {
		panic(err)
	}
	fmt.Println("S2_NATIVE_UPDATE_ACCEPTED_LOST_ACK_ORIGINAL_UNKNOWN=PASS")
}

type savedPostread struct {
	OriginalID string `json:"original_id"`
	BoxID      string `json:"box_id"`
}

func postreadArgs(boxID string) map[string]any {
	return map[string]any{
		"box_id": boxID, "cpu": 2,
		"idempotency_key": "native-s2-update-accepted-postread-failed-once-001",
	}
}

func checkPostreadLocked(ctx context.Context, s *mcp.ClientSession, original savedPostread) {
	var observed map[string]any
	for n := 0; n < 100; n++ {
		reply, err := invoke(ctx, s, "operation_get", map[string]any{
			"operation_id": original.OriginalID, "wait_ms": 300,
		})
		if err != nil {
			panic(err)
		}
		observed = child(reply, "operation")
		state := field(observed, "state")
		if state == "UNKNOWN" {
			require(field(observed, "native_operation_id") != "" &&
				field(observed, "error_class") == "POSTCONDITION_UNVERIFIED" &&
				field(child(observed, "next_action"), "kind") == "HOST_LOCAL_REVIEW",
				"accepted native Update lost postread but no original native ID/UNKNOWN receipt")
			break
		}
		require(state != "FAILED" && state != "SUCCEEDED" && state != "CANCELLED",
			"unverified native postcondition incorrectly settled operation")
		require(n < 99, "native postread fault did not settle to UNKNOWN")
	}
	again, err := invoke(ctx, s, "box_update", postreadArgs(original.BoxID))
	require(err == nil && field(again, "operation_id") == original.OriginalID &&
		again["duplicate"] == true, "same-key Update postread replay was not original")
	different := postreadArgs(original.BoxID)
	different["idempotency_key"] = "native-s2-postread-new-key-must-deny-001"
	different["cpu"] = 3
	bad, err := invoke(ctx, s, "box_update", different)
	require(err != nil, fmt.Sprintf("new-key Update bypassed postread UNKNOWN lock: %+v", bad))
}

func testAcceptedUpdatePostread(ctx context.Context, s *mcp.ClientSession, resume bool) {
	const fixture = "bw-s2-update-postread-native-110g"
	file := os.Getenv("BW_S2_POSTREAD_STATE_FILE")
	require(strings.HasPrefix(file, "/tmp/bw-v1-s2-ghci.") &&
		strings.HasSuffix(file, "/postread-original.json"), "disposable postread checkpoint required")
	if resume {
		raw, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}
		var original savedPostread
		if err := json.Unmarshal(raw, &original); err != nil {
			panic(err)
		}
		require(original.OriginalID != "" && original.BoxID != "", "postread original checkpoint missing")
		checkPostreadLocked(ctx, s, original)
		fmt.Println("S2_NATIVE_UPDATE_POSTREAD_UNKNOWN_RESTART_NO_REPLAY=PASS")
		return
	}
	created, err := invoke(ctx, s, "box_create", map[string]any{
		"name": fixture, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": false, "internet": true,
		"idempotency_key": "native-s2-update-postread-fixture-create-001",
	})
	if err != nil {
		panic(err)
	}
	createID := field(created, "operation_id")
	require(createID != "", "postread test fixture create missing original ID")
	var boxID string
	for n := 0; n < 120; n++ {
		ret, e := invoke(ctx, s, "operation_get", map[string]any{
			"operation_id": createID, "wait_ms": 400,
		})
		if e != nil {
			panic(e)
		}
		r := child(ret, "operation")
		if field(r, "state") == "SUCCEEDED" {
			boxID = field(r, "box_uuid")
			break
		}
		require(field(r, "state") != "FAILED" && field(r, "state") != "UNKNOWN",
			"postread fixture create failed")
		require(n < 119, "postread fixture create did not finish")
	}
	require(boxID != "", "verified postread fixture missing exact native UUID")
	update, err := invoke(ctx, s, "box_update", postreadArgs(boxID))
	if err != nil {
		panic(err)
	}
	original := savedPostread{OriginalID: field(update, "operation_id"), BoxID: boxID}
	require(original.OriginalID != "", "postread Update admission missing original ID")
	checkPostreadLocked(ctx, s, original)
	raw, err := json.Marshal(original)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(file, raw, 0600); err != nil {
		panic(err)
	}
	fmt.Println("S2_NATIVE_UPDATE_ACCEPTED_ACK_DELIVERED_POSTREAD_UNKNOWN=PASS")
}

func main() {
	require(len(os.Args) == 2 && (os.Args[1] == "start" || os.Args[1] == "resume" || os.Args[1] == "etag" ||
		os.Args[1] == "update_ack" || os.Args[1] == "update_ack_resume" ||
		os.Args[1] == "postread" || os.Args[1] == "postread_resume" ||
		os.Args[1] == "unbound_delete"),
		"start/resume/etag/update_ack/update_ack_resume mode required")
	file := os.Getenv("BW_S2_ACK_STATE_FILE")
	require(strings.HasPrefix(file, "/tmp/bw-v1-s2-ghci.") &&
		strings.HasSuffix(file, "/ack-original.json"), "only disposable original checkpoint permitted")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	s := session(ctx)
	defer s.Close()
	if os.Args[1] == "postread" || os.Args[1] == "postread_resume" {
		testAcceptedUpdatePostread(ctx, s, os.Args[1] == "postread_resume")
		return
	}
	if os.Args[1] == "etag" {
		testStaleNativeETag(ctx, s)
		return
	}
	if os.Args[1] == "update_ack" || os.Args[1] == "update_ack_resume" {
		testAcceptedUpdateAckLoss(ctx, s, os.Args[1] == "update_ack_resume")
		return
	}
	if os.Args[1] == "unbound_delete" {
		// The native Incus operation may already have accepted Create 202
		// and assigned this exact UUID; BoxWeave must NOT adopt/delete it
		// until its original durable Create has independently verified it.
		actual := os.Getenv("BW_S2_ACK_NATIVE_UUID")
		require(len(actual) == 36, "native admin's exact UUID witness missing")
		data, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}
		var original saved
		if err := json.Unmarshal(data, &original); err != nil {
			panic(err)
		}
		require(original.OriginalID != "", "original accepted Create receipt missing")
		checkUnknown(ctx, s, original.OriginalID)
		result, err := invoke(ctx, s, "box_delete", map[string]any{
			"box_id":          actual,
			"idempotency_key": "s2-p1-unbound-create-delete-must-deny",
		})
		require(err != nil, fmt.Sprintf("unbound real native UUID was deleted or adopted: %+v", result))
		checkUnknown(ctx, s, original.OriginalID)
		fmt.Println("S2_P1_REAL_UNBOUND_CREATE_UUID_DELETE_REFUSED_WITHOUT_REDISPATCH=PASS")
		return
	}
	if os.Args[1] == "start" {
		created, err := invoke(ctx, s, "box_create", createArgs())
		if err != nil {
			panic(err)
		}
		id := field(created, "operation_id")
		require(id != "", "native create was not admitted before ACK lost")
		checkUnknown(ctx, s, id)
		replayMustNotRedispatch(ctx, s, id)
		data, err := json.Marshal(saved{OriginalID: id})
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(file, data, 0600); err != nil {
			panic(err)
		}
		fmt.Println("S2_NATIVE_CREATE_ACK_LOSS_ORIGINAL_UNKNOWN=PASS")
		return
	}
	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	var original saved
	if err := json.Unmarshal(data, &original); err != nil {
		panic(err)
	}
	require(original.OriginalID != "", "missing original checkpoint on restart")
	checkUnknown(ctx, s, original.OriginalID)
	replayMustNotRedispatch(ctx, s, original.OriginalID)
	fmt.Println("S2_NATIVE_CREATE_ACK_LOSS_RESTART_NO_REPLAY=PASS")
}
