package control

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// An authorized HTTP response is received by the simulated transport but
// never delivered to the caller. The service must not dispatch a replacement
// original; a new client discovers the prior durable state after DB reopen.
type activityLostReplyTransport struct {
	token string
	armed atomic.Bool
}

func (t *activityLostReplyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cp := req.Clone(req.Context())
	cp.Header.Set("Authorization", "Bearer "+t.token)
	resp, err := http.DefaultTransport.RoundTrip(cp)
	if err != nil {
		return nil, err
	}
	if !t.armed.CompareAndSwap(true, false) {
		return resp, nil
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return nil, errors.New("ACTIVITY_PROBE_INTENTIONALLY_LOST_HTTP_REPLY")
}

func activityMCPServer(st *state.Store) *httptest.Server {
	svc := &Service{ScopedPeerFresh: true, Store: st, Native: native.Reader{
		Socket: "/nonexistent", Project: "bw-test",
	}, BuildVersion: "reconnect-test", BuildCommit: "reconnect-test", Listener: "127.0.0.1:18650"}
	return httptest.NewServer(svc.Handler())
}

func activityMCPClient(ctx context.Context, endpoint string, client *http.Client) (*mcp.ClientSession, error) {
	c := mcp.NewClient(&mcp.Implementation{Name: "activity-reconnect", Version: "1"}, nil)
	return c.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: endpoint + "/mcp", HTTPClient: client,
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
}

func TestActivityRealHTTPReplyLossReopenControllerFreshActorFindsExactUnknown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(dir, "controller.sqlite")
	st, err := state.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	raw, first, err := st.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	work := "a2/recover-20261119"
	rec, err := st.AdmitScoped(ctx, state.Admission{
		Layer: "bw-test", ActorID: first.ActorID, TokenID: first.ID,
		Key: "a2-reconnect-lost-native-reply-001", RequestID: "lost-reply-first-client",
		Action: "box_exec", SafeIntent: []byte("{}"),
		BoxUUID:     "c118c9a0-2f03-4a2f-a63e-3982247e359e",
		Incarnation: "c118c9a0-2f03-4a2f-a63e-3982247e359e",
		ResourceKey: "box:interrupted", WorkRef: work, AgentRef: "first-client",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Claim(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.Unknown(ctx, rec.ID, "DISPATCHING", "NATIVE_ACK_UNCERTAIN"); err != nil {
		t.Fatal(err)
	}
	lossy := &activityLostReplyTransport{token: raw}
	httpServer := activityMCPServer(st)
	client, err := activityMCPClient(ctx, httpServer.URL, &http.Client{Transport: lossy})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListTools(ctx, nil); err != nil {
		t.Fatal(err)
	}
	lossy.armed.Store(true)
	_, err = client.CallTool(ctx, &mcp.CallToolParams{
		Name: "box_activity_get", Arguments: map[string]any{
			"view": "issues", "work_ref": work,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "ACTIVITY_PROBE_INTENTIONALLY_LOST_HTTP_REPLY") {
		t.Fatalf("Activity response loss was not actually observed: %v", err)
	}
	_ = client.Close()
	httpServer.Close()
	// Restart with the same durable database and a fresh independently
	// authenticated principal, without borrowing the original actor key.
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := state.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	nextRaw, second, err := reopened.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if second.ActorID == first.ActorID || second.ID == first.ID {
		t.Fatal("fresh principal reused earlier actor/token identity")
	}
	nextServer := activityMCPServer(reopened)
	defer nextServer.Close()
	nextClient, err := activityMCPClient(ctx, nextServer.URL,
		&http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: nextRaw}})
	if err != nil {
		t.Fatal(err)
	}
	defer nextClient.Close()
	for _, tc := range []struct {
		args map[string]any
		must string
	}{
		{map[string]any{"view": "issues", "work_ref": work}, rec.ID},
		{map[string]any{"view": "evidence", "operation_id": rec.ID}, "NATIVE_ACK_UNCERTAIN"},
		{map[string]any{"view": "overview", "work_ref": work}, rec.ID},
	} {
		reply, err := nextClient.CallTool(ctx, &mcp.CallToolParams{
			Name: "box_activity_get", Arguments: tc.args,
		})
		if err != nil || reply == nil || reply.IsError {
			t.Fatalf("fresh actor cannot find prior evidence: %+v, %v", reply, err)
		}
		b, err := json.Marshal(reply.StructuredContent)
		if err != nil || !strings.Contains(string(b), tc.must) {
			t.Fatalf("lost original after restart: %s, %v", b, err)
		}
	}
	receipt, err := reopened.GetScopedOperation(ctx, "bw-test", second.ActorID, rec.ID)
	if err != nil || receipt.State != "UNKNOWN" || receipt.ID != rec.ID {
		t.Fatalf("restart created/replayed wrong original: %+v, %v", receipt, err)
	}
	if receipt.NextAction.Kind != "HOST_LOCAL_REVIEW" {
		t.Fatalf("uncertain native original advertised unsafe automated retry: %+v", receipt.NextAction)
	}
	// A different Actor cannot bypass an UNKNOWN original by minting a new
	// idempotency key for the same Box. Do not dispatch another native effect.
	_, replayErr := reopened.AdmitScoped(ctx, state.Admission{
		Layer: "bw-test", ActorID: second.ActorID, TokenID: second.ID,
		Key: "a2-forbidden-new-key-after-unknown-001", RequestID: "second-agent-replay",
		Action: "box_exec", SafeIntent: []byte("{}"),
		BoxUUID: rec.BoxUUID, Incarnation: rec.Incarnation,
		ResourceKey: "box:interrupted", WorkRef: work, AgentRef: "second-client",
	})
	if !errors.Is(replayErr, state.ErrResourceLocked) {
		t.Fatalf("second Actor bypassed UNKNOWN resource barrier: %v", replayErr)
	}
	got, readErr := nextClient.CallTool(ctx, &mcp.CallToolParams{Name: "operation_get",
		Arguments: map[string]any{"operation_id": rec.ID}})
	if readErr != nil || got == nil || got.IsError {
		t.Fatalf("second Actor cannot read exact old original over MCP: %+v, %v", got, readErr)
	}
	receiptPayload, readErr := json.Marshal(got.StructuredContent)
	if readErr != nil || !strings.Contains(string(receiptPayload), "HOST_LOCAL_REVIEW") {
		t.Fatalf("second Actor MCP did not reveal safe next action: %s, %v", receiptPayload, readErr)
	}
	page, err := reopened.ListOperations(ctx, state.OperationQuery{
		Layer: "bw-test", Requester: second.ActorID, WorkRef: work,
	})
	if err != nil || len(page.Operations) != 1 || page.Operations[0].ID != rec.ID {
		t.Fatalf("response loss created new mutation or hid original: %+v, %v", page, err)
	}
}
