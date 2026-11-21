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

func TestS5A5HTTPReconnectRecoveryNavigationAndAttention(t *testing.T) {
	st, ts, raw, _ := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	tok, e := st.VerifyToken(ctx, raw)
	if e != nil {
		t.Fatal(e)
	}
	seed := func(layer, key string) state.Receipt {
		t.Helper()
		r, e := st.Admit(ctx, state.Admission{Layer: layer, ActorID: tok.ActorID, TokenID: tok.ID, Key: key, Action: "operation_cancel", RequestID: key, SafeIntent: []byte("{}"), WorkRef: "s5a5/reconnect"})
		if e != nil {
			t.Fatal(e)
		}
		if e = st.Claim(ctx, r.ID); e != nil {
			t.Fatal(e)
		}
		if e = st.Unknown(ctx, r.ID, "DISPATCHING", "TEST_UNKNOWN"); e != nil {
			t.Fatal(e)
		}
		return r
	}
	orig := seed("bw-test", "a5-http-unknown-original")
	foreign := seed("foreign", "a5-http-foreign-original")
	connect := func() *mcp.ClientSession {
		s, e := activityMCPClient(ctx, ts.URL, &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: raw}})
		if e != nil {
			t.Fatal(e)
		}
		return s
	}
	s := connect()
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		r, e := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if e != nil || r == nil || r.IsError {
			t.Fatalf("%s failed %v", name, e)
		}
		b, e := json.Marshal(r.StructuredContent)
		if e != nil {
			t.Fatal(e)
		}
		if strings.Contains(string(b), foreign.ID) {
			t.Fatal("foreign evidence leaked")
		}
		var out map[string]any
		if e = json.Unmarshal(b, &out); e != nil {
			t.Fatal(e)
		}
		return out
	}
	obj := func(x any) map[string]any {
		t.Helper()
		m, ok := x.(map[string]any)
		if !ok {
			t.Fatalf("expected object %+v", x)
		}
		return m
	}
	first := call("operation_get", map[string]any{"operation_id": orig.ID})
	nav := obj(first["recovery_navigation"])
	if first["recovery_snapshot_matches_operation"] != true || nav["claim_protection"] != "RETAINED" || obj(nav["next_action"])["mode"] != "inspect" {
		t.Fatalf("wrong nav %+v", first)
	}
	s.Close()
	s = connect()
	defer s.Close()
	issue := obj(call("box_activity_get", map[string]any{"view": "issues", "work_ref": "s5a5/reconnect"})["activity"])
	if len(issue["issues"].([]any)) != 1 || obj(issue["issues"].([]any)[0])["operation_id"] != orig.ID {
		t.Fatalf("reconnect issues %+v", issue)
	}
	hs := obj(call("host_status", map[string]any{})["attention"])
	if hs["native_live_verified"] != false || hs["attention_complete"] != false || !strings.Contains(stringMustJSON(hs), orig.ID) {
		t.Fatalf("host attention %+v", hs)
	}
	recovered := obj(call("operation_recover", map[string]any{"operation_id": orig.ID, "mode": "reconcile", "idempotency_key": "a5-blocked-recovery-request"})["recovery"])
	id := obj(recovered["operation"])["operation_id"].(string)
	if recovered["disposition"] != "BLOCKED" {
		t.Fatal("unsupported recovery unexpectedly succeeds")
	}
	v := obj(call("operation_get", map[string]any{"operation_id": orig.ID})["recovery_navigation"])
	links := v["recent_recoveries"].([]any)
	if len(links) != 1 || obj(links[0])["operation_id"] != id || v["claim_protection"] != "RETAINED" {
		t.Fatalf("missing original link %+v", v)
	}
	ev := obj(obj(call("box_activity_get", map[string]any{"view": "evidence", "operation_id": id})["activity"])["evidence"])
	rev := obj(obj(ev["recovery_navigation"])["original_link"])
	if rev["original_operation_id"] != orig.ID || rev["disposition"] != "BLOCKED" {
		t.Fatalf("reverse link %+v", rev)
	}
	if e = st.RevokeToken(ctx, tok.ID); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"operation_get", "box_activity_get", "host_status"} {
		args := map[string]any{}
		if name == "operation_get" {
			args["operation_id"] = id
		}
		out, e := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if e == nil && out != nil && !out.IsError {
			t.Fatalf("revoked access %s", name)
		}
	}
}
func stringMustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
