package control

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

func TestS5A2HTTPRecoveryRefusesCrossLayerAndUntrustedApproval(t *testing.T) {
	st, ts, admin, other := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	token, e := st.VerifyToken(ctx, admin)
	if e != nil {
		t.Fatal(e)
	}
	seed := func(layer, key string) state.Receipt {
		t.Helper()
		r, e := st.Admit(ctx, state.Admission{Layer: layer, ActorID: token.ID, TokenID: token.ID, Key: key, Action: "operation_cancel", RequestID: "test-original", SafeIntent: []byte("{}")})
		if e != nil {
			t.Fatal(e)
		}
		if e = st.Claim(ctx, r.ID); e != nil {
			t.Fatal(e)
		}
		if e = st.Unknown(ctx, r.ID, "DISPATCHING", "TEST_ORIGINAL_UNKNOWN"); e != nil {
			t.Fatal(e)
		}
		return r
	}
	original := seed("bw-test", "recovery-http-owned-original")
	foreign := seed("foreign-layer", "recovery-http-foreign-original")
	connect := func(raw string) *mcp.ClientSession {
		t.Helper()
		s, e := activityMCPClient(ctx, ts.URL, &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: raw}})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { s.Close() })
		return s
	}
	s := connect(admin)
	call := func(c *mcp.ClientSession, args map[string]any) (map[string]any, bool) {
		t.Helper()
		r, e := c.CallTool(ctx, &mcp.CallToolParams{Name: "operation_recover", Arguments: args})
		if e != nil || r == nil || r.IsError {
			return nil, false
		}
		return b4MCPData(t, r, e), true
	}
	denied := []map[string]any{
		{"operation_id": foreign.ID, "mode": "inspect"},
		{"operation_id": foreign.ID, "mode": "reconcile", "idempotency_key": "foreign-layer-recovery-key"},
		{"operation_id": original.ID, "mode": "reconcile", "idempotency_key": "agent-self-approval-key", "approved": true},
		{"operation_id": original.ID, "mode": "release", "idempotency_key": "agent-release-recovery-key"},
		{"operation_id": original.ID, "mode": "inspect", "project": "foreign-layer"},
		{"operation_id": original.ID, "mode": "reconcile"},
	}
	for _, args := range denied {
		if _, ok := call(s, args); ok {
			t.Fatalf("unsafe accepted request %+v", args)
		}
	}
	inspect, ok := call(s, map[string]any{"operation_id": original.ID, "mode": "inspect"})
	if !ok {
		t.Fatal("inspection failed")
	}
	diagnosis := inspect["diagnosis"].(map[string]any)
	if diagnosis["disposition"] != "BLOCKED" || diagnosis["claim_protection"] != "RETAINED" || inspect["recovery"] != nil {
		t.Fatalf("unexpected inspection %+v", inspect)
	}
	args := map[string]any{"operation_id": original.ID, "mode": "reconcile", "idempotency_key": "recovery-http-fresh-key-001"}
	first, ok := call(s, args)
	if !ok {
		t.Fatal("bounded refusal was not journaled")
	}
	record := first["recovery"].(map[string]any)
	op := record["operation"].(map[string]any)
	if record["disposition"] != "BLOCKED" || op["state"] != "FAILED" || op["operation_id"] == original.ID {
		t.Fatalf("unexpected receipt %+v", record)
	}
	again, ok := call(s, args)
	if !ok {
		t.Fatal("same key replay failed")
	}
	op2 := again["recovery"].(map[string]any)["operation"].(map[string]any)
	if op2["operation_id"] != op["operation_id"] || op2["duplicate"] != true {
		t.Fatalf("lost recovery receipt %+v", again)
	}
	s2 := connect(other)
	independently, ok := call(s2, args)
	if !ok {
		t.Fatal("same-Layer independent actor may inspect/recover")
	}
	op3 := independently["recovery"].(map[string]any)["operation"].(map[string]any)
	if op3["operation_id"] == op["operation_id"] {
		t.Fatal("borrowed another actor's key")
	}
	g, e := st.Get(ctx, original.ID)
	if e != nil || g.State != "UNKNOWN" || g.ErrorClass != "TEST_ORIGINAL_UNKNOWN" {
		t.Fatalf("original changed %+v %v", g, e)
	}
}
