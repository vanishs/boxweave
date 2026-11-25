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

func TestActivityOfficialMCPThreeViewsAndSameLayerTakeover(t *testing.T) {
	st, ts, admin, second := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tok, err := st.VerifyToken(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	original, err := st.Admit(ctx, state.Admission{
		Layer: testScopedProject, ActorID: tok.ActorID, TokenID: tok.ID,
		Key: "activity-mcp-original-20261119-01", RequestID: "activity-original",
		Action: "box_exec", SafeIntent: []byte("{}"), ResourceKey: "box:owned",
		BoxUUID:     "bcfe32e0-3acf-4f21-8a66-332a84d741a4",
		Incarnation: "bcfe32e0-3acf-4f21-8a66-332a84d741a4",
		WorkRef:     "work/reading", AgentRef: "agent/original",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = st.Claim(ctx, original.ID); err != nil {
		t.Fatal(err)
	}
	if err = st.Unknown(ctx, original.ID, "DISPATCHING", "SIMULATED_REPLY_LOST"); err != nil {
		t.Fatal(err)
	}
	if _, err = st.Admit(ctx, state.Admission{
		Layer: "foreign", ActorID: tok.ActorID, TokenID: tok.ID,
		Key: "activity-mcp-foreign-20261119-01", RequestID: "foreign",
		Action: "box_exec", SafeIntent: []byte("{}"), ResourceKey: "box:secret",
	}); err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "activity-client", Version: "1"}, nil)
	sess, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             ts.URL + "/mcp",
		HTTPClient:           &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: second}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	tools, err := sess.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tool := range tools.Tools {
		if tool.Name == "box_activity_get" {
			found = true
			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				t.Fatal("Activity must be read-only")
			}
		}
	}
	if !found || len(tools.Tools) != 23 {
		t.Fatalf("wrong SDK tool inventory: %d", len(tools.Tools))
	}
	for _, tc := range []struct {
		view string
		args map[string]any
		want string
	}{
		{"overview", map[string]any{"view": "overview", "limit": 5}, original.ID},
		{"issues", map[string]any{"view": "issues", "limit": 5}, "UNRESOLVED_OPERATION"},
		{"evidence", map[string]any{"view": "evidence", "operation_id": original.ID}, "SIMULATED_REPLY_LOST"},
	} {
		result, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "box_activity_get", Arguments: tc.args})
		if e != nil || result == nil || result.IsError {
			t.Fatalf("%s failed %+v %v", tc.view, result, e)
		}
		raw, e := json.Marshal(result.StructuredContent)
		if e != nil {
			t.Fatal(e)
		}
		if !strings.Contains(string(raw), tc.want) ||
			!strings.Contains(string(raw), "\"tracking_id\"") ||
			!strings.Contains(string(raw), "\"activity\"") ||
			strings.Contains(string(raw), "activity-foreign") {
			t.Fatalf("%s lost scope or evidence: %s", tc.view, raw)
		}
	}
	for _, args := range []map[string]any{
		{"view": "not-supported"}, {"view": "evidence"},
		{"view": "issues", "unexpected": "foreign"}, {"view": "issues", "limit": 51},
	} {
		result, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: "box_activity_get", Arguments: args})
		if e == nil && result != nil && !result.IsError {
			t.Fatalf("invalid input accepted: %+v", args)
		}
	}
}
