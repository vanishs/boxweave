package control

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// The installed/native probe is separate: this test validates real OAuth HTTP
// and the official MCP client against a synthetic SQLite terminal receipt.
// It does not claim a real Incus effect.
func TestS4OAuthRefreshHTTPMCPOriginalReceiptNotRedispatched(t *testing.T) {
	st, server, password := newOAuthServerTest(t)
	issued := completeOAuth(t, server, password, "mcp:admin")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	oldToken, err := st.VerifyToken(ctx, issued.access)
	if err != nil || oldToken.ActorID == oldToken.ID {
		t.Fatalf("no stable OAuth grant actor: %+v %v", oldToken, err)
	}
	const name = "s4-http-replay-only"
	const key = "s4-http-oauth-grant-replay-0001"
	intent, err := safeJSON(struct {
		Name      string                 `json:"name"`
		CPU       int                    `json:"cpu"`
		MemoryMiB int64                  `json:"memory_mib"`
		DiskGiB   int64                  `json:"disk_gib"`
		Image     string                 `json:"image,omitempty"`
		Start     bool                   `json:"start"`
		Nesting   bool                   `json:"nesting"`
		Internet  bool                   `json:"internet"`
		Options   native.InstanceOptions `json:"instance_options"`
	}{Name: name, CPU: 1, MemoryMiB: 1024, DiskGiB: 10, Start: true, Internet: true})
	if err != nil {
		t.Fatal(err)
	}
	original, err := st.Admit(ctx, state.Admission{
		Layer: "test", ActorID: oldToken.ActorID, TokenID: oldToken.ID,
		Key: key, RequestID: "original-http", Action: "box_create",
		ResourceKey: "box:" + name, SafeIntent: intent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Claim(ctx, original.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.NativeAck(ctx, original.ID, "test-only-synthetic-op-not-native"); err != nil {
		t.Fatal(err)
	}
	if err := st.Verified(ctx, original.ID); err != nil {
		t.Fatal(err)
	}
	form := url.Values{
		"grant_type": {"refresh_token"}, "client_id": {issued.client},
		"refresh_token": {issued.refresh}, "resource": {oauthPublicOrigin + "/mcp"},
	}
	refreshed := oauthReq(t, server, "POST", "/token", form.Encode(),
		"application/x-www-form-urlencoded", "", oauthPublicHost)
	if refreshed.StatusCode != http.StatusOK {
		t.Fatalf("refresh returned HTTP %d", refreshed.StatusCode)
	}
	pair := jsonFrom(t, refreshed)
	accessB, ok := pair["access_token"].(string)
	if !ok {
		t.Fatal("refresh returned no access token")
	}
	tokenB, err := st.VerifyToken(ctx, accessB)
	if err != nil || tokenB.ID == oldToken.ID || tokenB.ActorID != oldToken.ActorID {
		t.Fatalf("refresh did not rotate credential within original grant: %+v %v", tokenB, err)
	}
	if _, err := st.VerifyToken(ctx, issued.access); !errors.Is(err, state.ErrRevoked) {
		t.Fatalf("expired access A remained valid: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "s4-oauth-replay", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             server.URL + "/mcp",
		HTTPClient:           &http.Client{Transport: hostOAuthTransport{rawToken: accessB}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	reply, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "box_create", Arguments: map[string]any{
			"name": name, "idempotency_key": key,
			"cpu": 1, "memory_mib": 1024, "disk_gib": 10,
		},
	})
	if err != nil || reply == nil || reply.IsError {
		t.Fatalf("refreshed original HTTP MCP rejected: %+v %v", reply, err)
	}
	encoded, err := json.Marshal(reply.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var got state.Receipt
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Duplicate || got.ID != original.ID || got.TrackingID != original.TrackingID ||
		got.State != "SUCCEEDED" || got.NativeOperationID != "test-only-synthetic-op-not-native" {
		t.Fatalf("second credential did not replay immutable original: %+v", got)
	}
	// The synthetic original is not a real native Box. A second path through
	// native Create would fail: a successful reply proves replay bypassed it.
	if _, err := st.Replay(ctx, state.Admission{
		Layer: "test", ActorID: oldToken.ActorID, TokenID: oldToken.ID,
		Key: key, Action: "box_create", SafeIntent: intent,
	}); !errors.Is(err, state.ErrRevoked) {
		t.Fatalf("old credential borrowed original: %v", err)
	}
}

func TestS4OAuthViewerHistoryAfterRefreshHTTPMCP(t *testing.T) {
	st, server, password := newOAuthServerTest(t)
	issued := completeOAuth(t, server, password, "mcp:viewer")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	old, err := st.VerifyToken(ctx, issued.access)
	if err != nil {
		t.Fatal(err)
	}
	original, err := st.Admit(ctx, state.Admission{
		Layer: "test", ActorID: old.ActorID, TokenID: old.ID,
		Key: "s4-viewer-history-immutable-original", RequestID: "viewer",
		Action: "test-viewer-observation", SafeIntent: []byte(`{"read":true}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"grant_type": {"refresh_token"}, "client_id": {issued.client},
		"refresh_token": {issued.refresh}, "resource": {oauthPublicOrigin + "/mcp"}}
	res := oauthReq(t, server, "POST", "/token", form.Encode(),
		"application/x-www-form-urlencoded", "", oauthPublicHost)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("viewer refresh HTTP %d", res.StatusCode)
	}
	pair := jsonFrom(t, res)
	access, ok := pair["access_token"].(string)
	if !ok {
		t.Fatal("viewer did not receive new credential")
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "s4-viewer-history", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             server.URL + "/mcp",
		HTTPClient:           &http.Client{Transport: hostOAuthTransport{rawToken: access}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, request := range []mcp.CallToolParams{
		{Name: "operation_get", Arguments: map[string]any{"operation_id": original.ID}},
		{Name: "operation_list", Arguments: map[string]any{"limit": 5}},
	} {
		result, err := session.CallTool(ctx, &request)
		if err != nil || result == nil || result.IsError {
			t.Fatalf("refreshed viewer %s: %+v %v", request.Name, result, err)
		}
		encoded, _ := json.Marshal(result.StructuredContent)
		if !json.Valid(encoded) {
			t.Fatal("malformed viewer response")
		}
		if !strings.Contains(string(encoded), original.ID) {
			t.Fatalf("viewer history lost original receipt in %s", request.Name)
		}
	}
}
