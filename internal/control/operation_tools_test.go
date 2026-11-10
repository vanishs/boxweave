package control

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

func TestS2MCPDurableOperationReadAndActorIsolation(t *testing.T) {
	st, httpServer, admin, viewer := fixture(t)
	ctx := context.Background()
	adminInfo, err := st.VerifyToken(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := st.Admit(ctx, state.Admission{
		Layer: "bw-test", ActorID: adminInfo.ID, TokenID: adminInfo.ID,
		Key: "mcp-s2-read-original-key-01", RequestID: "original",
		Action: "box_create", SafeIntent: []byte(`{"name":"safe-test"}`),
		WorkRef: "acceptance/session-1", AgentRef: "first-agent",
	})
	if err != nil {
		t.Fatal(err)
	}

	connect := func(token string) (*mcp.ClientSession, context.CancelFunc) {
		t.Helper()
		requestContext, cancel := context.WithTimeout(ctx, 10*time.Second)
		client := mcp.NewClient(&mcp.Implementation{Name: "s2-reads", Version: "1"}, nil)
		session, err := client.Connect(requestContext, &mcp.StreamableClientTransport{
			Endpoint: httpServer.URL + "/mcp",
			HTTPClient: &http.Client{Transport: bearerTransport{
				base: http.DefaultTransport, token: token,
			}},
			DisableStandaloneSSE: true, MaxRetries: -1,
		}, nil)
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		return session, cancel
	}
	session, cancel := connect(admin)
	defer cancel()
	defer session.Close()
	listing, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	reads := map[string]bool{"host_status": true, "box_list": true, "box_get": true,
		"operation_get": true, "operation_list": true, "operation_output_get": true}
	for _, tool := range listing.Tools {
		if tool.Annotations == nil || tool.Annotations.ReadOnlyHint != reads[tool.Name] {
			t.Fatalf("incorrect side-effect annotation: %s", tool.Name)
		}
	}
	got, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_get", Arguments: map[string]any{"operation_id": receipt.ID, "wait_ms": 0},
	})
	if err != nil || got.IsError {
		t.Fatalf("original MCP operation_get: %+v %v", got, err)
	}
	encoded, err := json.Marshal(got.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), receipt.ID) || !strings.Contains(string(encoded), receipt.TrackingID) {
		t.Fatalf("MCP operation_get lost original IDs: %s", encoded)
	}
	got, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_list", Arguments: map[string]any{"work_ref": "acceptance/session-1", "limit": 1},
	})
	if err != nil || got.IsError {
		t.Fatalf("original MCP operation_list: %+v %v", got, err)
	}
	encoded, err = json.Marshal(got.StructuredContent)
	if err != nil || !strings.Contains(string(encoded), receipt.ID) {
		t.Fatalf("MCP operation_list lost original receipt: %s %v", encoded, err)
	}
	// A different authenticated token cannot borrow authority using the same
	// self-declared work or Agent label.
	viewSession, viewCancel := connect(viewer)
	defer viewCancel()
	defer viewSession.Close()
	hidden, err := viewSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_list", Arguments: map[string]any{"work_ref": "acceptance/session-1"},
	})
	if err != nil || hidden.IsError {
		t.Fatalf("scoped viewer list should return empty page: %+v %v", hidden, err)
	}
	encoded, err = json.Marshal(hidden.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), receipt.ID) {
		t.Fatalf("work_ref leaked original operation across actors: %s", encoded)
	}
	denied, err := viewSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_get", Arguments: map[string]any{"operation_id": receipt.ID},
	})
	if err == nil && denied != nil && !denied.IsError {
		t.Fatalf("viewer guessed original operation id successfully: %+v", denied)
	}
	invalidWait, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "operation_get", Arguments: map[string]any{"operation_id": receipt.ID, "wait_ms": 20001},
	})
	if err == nil && invalidWait != nil && !invalidWait.IsError {
		t.Fatal("unbounded operation wait was accepted")
	}
	// A read never claims or dispatches the original admission.
	unchanged, err := st.Get(ctx, receipt.ID)
	if err != nil || unchanged.State != "ADMITTED" {
		t.Fatalf("MCP read produced a side effect: %+v %v", unchanged, err)
	}
}

func TestS2ViewerCannotMutateAnyNativeTool(t *testing.T) {
	st, ts, admin, viewer := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "s2-negative-viewer", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             ts.URL + "/mcp",
		HTTPClient:           &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: viewer}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{"box_create", map[string]any{"name": "negative-box", "idempotency_key": "viewer-unauthorized-create-1"}},
		{"box_update", map[string]any{"box_id": "forged", "cpu": 2, "idempotency_key": "viewer-unauthorized-update-1"}},
		{"box_start", map[string]any{"box_id": "forged", "idempotency_key": "viewer-unauthorized-start-1"}},
		{"box_stop", map[string]any{"box_id": "forged", "idempotency_key": "viewer-unauthorized-stop-01"}},
		{"box_delete", map[string]any{"box_id": "forged", "idempotency_key": "viewer-unauthorized-delete1"}},
		{"box_exec", map[string]any{"box_id": "forged", "argv": []string{"true"}, "idempotency_key": "viewer-unauthorized-exec-01"}},
		{"operation_cancel", map[string]any{"target_operation_id": "forged", "idempotency_key": "viewer-unauthorized-cancel"}},
	} {
		r, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.name, Arguments: tc.args})
		if err == nil && r != nil && !r.IsError {
			t.Fatalf("viewer mutation succeeded: %s %+v", tc.name, r)
		}
	}
	tok, err := st.VerifyToken(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	ops, err := st.ListOperations(ctx, state.OperationQuery{
		Layer: "bw-test", Role: "admin", Requester: tok.ID,
	})
	if err != nil || len(ops.Operations) != 0 {
		t.Fatalf("viewer denial created a durable native admission: %+v %v", ops, err)
	}
}

func TestS2MCPQueuedCancelOriginalReceiptNeverIssuesNativeID(t *testing.T) {
	st, ts, admin, _ := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tok, err := st.VerifyToken(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	target, err := st.Admit(ctx, state.Admission{
		Layer: "bw-test", ActorID: tok.ID, TokenID: tok.ID,
		Key: "s2-original-queued-exec-0001", Action: "box_exec",
		RequestID: "queued-before-worker", SafeIntent: []byte(`{"guest_command_digest":"safe"}`),
		ResourceKey: "box:alpha",
	})
	if err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "s2-queued-cancel", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             ts.URL + "/mcp",
		HTTPClient:           &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: admin}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	args := map[string]any{"target_operation_id": target.ID, "idempotency_key": "s2-cancel-queued-original-0001"}
	first, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "operation_cancel", Arguments: args})
	if err != nil || first.IsError {
		t.Fatalf("cancel admission: %+v %v", first, err)
	}
	body, err := json.Marshal(first.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		CancelOperation   state.Receipt `json:"cancel_operation"`
		TargetOperationID string        `json:"target_operation_id"`
	}
	if err = json.Unmarshal(body, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.CancelOperation.ID == "" || receipt.CancelOperation.ID == target.ID || receipt.TargetOperationID != target.ID {
		t.Fatalf("fake/non-distinct cancel receipt: %+v", receipt)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, e := st.Get(ctx, target.ID)
		if e != nil {
			t.Fatal(e)
		}
		if got.State == "CANCELLED" {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	got, err := st.Get(ctx, target.ID)
	if err != nil || got.State != "CANCELLED" || got.NativeOperationID != "" {
		t.Fatalf("cancelled queue still dispatchable or fake native id: %+v %v", got, err)
	}
	after, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "operation_cancel", Arguments: args})
	if err != nil || after.IsError {
		t.Fatalf("original cancel key replay: %+v %v", after, err)
	}
	bytes, err := json.Marshal(after.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var duplicate struct {
		CancelOperation state.Receipt `json:"cancel_operation"`
	}
	if err = json.Unmarshal(bytes, &duplicate); err != nil {
		t.Fatal(err)
	}
	if duplicate.CancelOperation.ID != receipt.CancelOperation.ID || !duplicate.CancelOperation.Duplicate {
		t.Fatalf("cancellation double dispatch or new receipt: original=%+v replay=%+v",
			receipt.CancelOperation, duplicate.CancelOperation)
	}
}

func TestS2MCPOriginalDeleteReplaysAfterBoxGoneWithoutNativeRead(t *testing.T) {
	store, httpServer, admin, _ := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	actor, err := store.VerifyToken(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	boxID := "native-uuid-stale"
	key := "s2-deleted-original-key-00001"
	original, err := store.Admit(ctx, state.Admission{
		Layer: "bw-test", ActorID: actor.ID, TokenID: actor.ID, Key: key,
		Action: "box_delete", RequestID: "original-delete",
		ResourceKey: "box:deleted-name", BoxUUID: boxID, Incarnation: boxID,
		SafeIntent: []byte(`{"box_id":"native-uuid-stale"}`),
		WorkRef:    "deleted/old-work", AgentRef: "first-agent",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Claim(ctx, original.ID); err != nil {
		t.Fatal(err)
	}
	if err = store.Unknown(ctx, original.ID, "DISPATCHING", "LOST_NATIVE_ACK"); err != nil {
		t.Fatal(err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "s2-replay-after-delete", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             httpServer.URL + "/mcp",
		HTTPClient:           &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: admin}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	out, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "box_delete", Arguments: map[string]any{
		"box_id": boxID, "idempotency_key": key,
	}})
	if err != nil || out.IsError {
		t.Fatalf("original delete was not read-only replayed: %+v %v", out, err)
	}
	encoded, err := json.Marshal(out.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var found state.Receipt
	if err = json.Unmarshal(encoded, &found); err != nil {
		t.Fatal(err)
	}
	if found.ID != original.ID || found.TrackingID != original.TrackingID ||
		!found.Duplicate || found.State != "UNKNOWN" ||
		found.WorkRef != "deleted/old-work" || found.NextAction.Kind != "HOST_LOCAL_REVIEW" {
		t.Fatalf("original uncertain native identity lost: %+v", found)
	}
	// A distinct key does NOT gain authority to delete a tombstoned/unknown
	// native Box merely by guessing the old UUID or copying the work label.
	other, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "box_delete", Arguments: map[string]any{
		"box_id": boxID, "idempotency_key": "s2-delete-new-dispatch-denied-01",
		"work_ref": "deleted/old-work",
	}})
	if err == nil && other != nil && !other.IsError {
		t.Fatal("new key bypassed native ownership readback after original UNKNOWN")
	}
}
