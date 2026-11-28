package control

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

// Real candidate handler and official SDK over loopback HTTP; only the
// native witnesses in the disposable fixture are synthetic. No host socket.
func TestS5A5BoxHandoffLostRecoveryAndArchive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "a5-handoff.sqlite")
	st, e := state.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { st.Close() })
	raw, first, e := st.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	const work = "s5a5/handoff"
	const uuid = "87654321-4321-4abc-8def-abcdef012345"
	const nativeID = "12345678-1234-4abc-8def-123456789abc"
	const content = "fixture-archived-command-result"
	const output = "/1.0/instances/handoff/logs/exec-output/exec_" + nativeID + ".stdout"
	if e = st.RegisterOwnedBox(ctx, uuid, "bw-test", "handoff", uuid); e != nil {
		t.Fatal(e)
	}
	seed := func(layer, key, action, box, resource string) state.Receipt {
		t.Helper()
		r, e := st.AdmitScoped(ctx, state.Admission{Layer: layer, ActorID: first.ActorID, TokenID: first.ID, Key: key, RequestID: key, Action: action, BoxUUID: box, Incarnation: box, ResourceKey: resource, SafeIntent: []byte("{}"), WorkRef: work, AgentRef: "agent-one"})
		if e != nil {
			t.Fatal(e)
		}
		if e = st.Claim(ctx, r.ID); e != nil {
			t.Fatal(e)
		}
		return r
	}
	exec := seed("bw-test", "a5-handoff-exec-original", "box_exec", uuid, "box:handoff")
	if e = st.NativeAck(ctx, exec.ID, nativeID); e != nil {
		t.Fatal(e)
	}
	if e = st.Unknown(ctx, exec.ID, "VERIFYING", "TEST_FIXTURE_ACK_LOST"); e != nil {
		t.Fatal(e)
	}
	target, e := st.ReadRecoveryTarget(ctx, "bw-test", exec.ID)
	if e != nil {
		t.Fatal(e)
	}
	svc := &Service{Store: st}
	reader := func(_ context.Context, _ string, offset, limit int64) ([]byte, bool, error) {
		if offset >= int64(len(content)) {
			return nil, true, nil
		}
		end := min(offset+limit, int64(len(content)))
		return []byte(content[offset:end]), end == int64(len(content)), nil
	}
	if e = svc.captureExecStream(ctx, exec.ID, "stdout", output, reader); e != nil {
		t.Fatal(e)
	}
	// Explicit synthetic independent native proof at the state boundary.
	// The installed native CI independently exercises actual MCP settlement.
	exit := 7
	settled, e := st.CommitScopedRecovery(ctx, state.RecoveryRequest{Layer: "bw-test", ActorID: first.ActorID, TokenID: first.ID, OriginalID: exec.ID, Key: "a5-handoff-fixture-settle", WorkRef: work}, target, state.RecoveryProof{Kind: state.RecoverySettle, Reason: "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED", NativeID: nativeID, UUID: uuid, Fingerprint: "synthetic-test-witness", Exit: &exit, StdoutPath: output})
	if e != nil || settled.Disposition != "SETTLED" {
		t.Fatal("fixture settlement", e)
	}
	box, e := st.OwnedBox(ctx, "bw-test", uuid)
	if e != nil {
		t.Fatal(e)
	}
	if e = st.TombstoneOwnedBox(ctx, box); e != nil {
		t.Fatal(e)
	}
	blocked := seed("bw-test", "a5-handoff-blocked-original", "operation_cancel", "", "")
	if e = st.Unknown(ctx, blocked.ID, "DISPATCHING", "TEST_FIXTURE_UNKNOWN"); e != nil {
		t.Fatal(e)
	}
	foreign := seed("foreign", "a5-handoff-foreign-original", "operation_cancel", "", "")
	if e = st.Unknown(ctx, foreign.ID, "DISPATCHING", "TEST_FIXTURE_FOREIGN"); e != nil {
		t.Fatal(e)
	}
	server := activityMCPServer(st)
	lossy := &activityLostReplyTransport{token: raw}
	s1, e := activityMCPClient(ctx, server.URL, &http.Client{Transport: lossy})
	if e != nil {
		server.Close()
		t.Fatal(e)
	}
	if _, e = s1.ListTools(ctx, nil); e != nil {
		t.Fatal(e)
	}
	args := map[string]any{"operation_id": blocked.ID, "mode": "reconcile", "idempotency_key": "a5-handoff-lost-recovery-reply", "work_ref": work}
	lossy.armed.Store(true)
	_, e = s1.CallTool(ctx, &mcp.CallToolParams{Name: "operation_recover", Arguments: args})
	if e == nil || !strings.Contains(e.Error(), "ACTIVITY_PROBE_INTENTIONALLY_LOST_HTTP_REPLY") {
		t.Fatal("expected actual lost response", e)
	}
	s1.Close()
	server.Close()
	if e = st.Close(); e != nil {
		t.Fatal(e)
	}
	st, e = state.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	raw2, second, e := st.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	if first.ActorID == second.ActorID || first.ID == second.ID {
		t.Fatal("actor not independent")
	}
	server = activityMCPServer(st)
	defer server.Close()
	connect := func(token string) *mcp.ClientSession {
		t.Helper()
		s, e := activityMCPClient(ctx, server.URL, &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: token}})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { s.Close() })
		return s
	}
	s2 := connect(raw2)
	s1 = connect(raw)
	call := func(s *mcp.ClientSession, name string, args map[string]any) map[string]any {
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
			t.Fatal("foreign operation disclosed")
		}
		var m map[string]any
		if e = json.Unmarshal(b, &m); e != nil {
			t.Fatal(e)
		}
		return m
	}
	obj := func(v any) map[string]any {
		t.Helper()
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("expected object: %+v", v)
		}
		return m
	}
	deny := func(s *mcp.ClientSession, name string, args map[string]any) {
		t.Helper()
		r, e := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if e == nil && r != nil && !r.IsError {
			t.Fatalf("unexpected success %s", name)
		}
	}
	activity := func(s *mcp.ClientSession, args map[string]any) map[string]any {
		return obj(call(s, "box_activity_get", args)["activity"])
	}
	nav := func(s *mcp.ClientSession, id string) map[string]any {
		return obj(call(s, "operation_get", map[string]any{"operation_id": id})["recovery_navigation"])
	}
	// New actor knows the work label only; all IDs are recovered via readers.
	cursor := ""
	ids := map[string]bool{}
	for i := 0; i < 10; i++ {
		query := map[string]any{"view": "overview", "work_ref": work, "limit": 1}
		if cursor != "" {
			query["cursor"] = cursor
		}
		page := activity(s2, query)
		for _, x := range page["operations"].([]any) {
			id := obj(x)["operation_id"].(string)
			if ids[id] {
				t.Fatal("duplicate page")
			}
			ids[id] = true
		}
		next, _ := page["next_cursor"].(string)
		if next == "" {
			break
		}
		if cursor == "" {
			deny(s1, "box_activity_get", map[string]any{"view": "overview", "work_ref": work, "limit": 1, "cursor": next})
		}
		cursor = next
	}
	if !ids[exec.ID] || !ids[blocked.ID] || !ids[settled.Receipt.ID] || len(ids) != 4 {
		t.Fatalf("incomplete handoff pagination: %v", ids)
	}
	issues := activity(s2, map[string]any{"view": "issues", "work_ref": work})
	if len(issues["issues"].([]any)) != 1 || obj(issues["issues"].([]any)[0])["operation_id"] != blocked.ID {
		t.Fatalf("wrong current blockers %+v", issues)
	}
	if !strings.Contains(stringMustJSON(issues["history"]), exec.ID) {
		t.Fatal("recovered history missing")
	}
	n := nav(s2, blocked.ID)
	links := n["recent_recoveries"].([]any)
	if len(links) != 1 || n["claim_protection"] != "RETAINED" {
		t.Fatal("lost blocked receipt")
	}
	lostID := obj(links[0])["operation_id"].(string)
	if !ids[lostID] || obj(nav(s2, lostID)["original_link"])["original_operation_id"] != blocked.ID {
		t.Fatal("reverse link missing")
	}
	replay := obj(call(s1, "operation_recover", args)["recovery"])
	if op := obj(replay["operation"]); op["operation_id"] != lostID || op["duplicate"] != true {
		t.Fatal("same actor replay changed receipt")
	}
	if obj(call(s2, "operation_recover", map[string]any{"operation_id": blocked.ID, "mode": "inspect"})["diagnosis"])["disposition"] != "BLOCKED" {
		t.Fatal("unsupported original unlocked")
	}
	if obj(call(s2, "host_status", map[string]any{})["attention"])["attention_complete"] != false {
		t.Fatal("misleading host completeness")
	}
	ev := obj(activity(s2, map[string]any{"view": "evidence", "operation_id": exec.ID})["evidence"])
	if ev["owned_box_state"] != "DELETED" || obj(ev["recovery_navigation"])["disposition"] != "RECOVERY_SETTLED" || obj(ev["exec"])["exit_code"] != float64(7) {
		t.Fatal("deleted evidence missing")
	}
	if obj(nav(s2, settled.Receipt.ID)["original_link"])["original_operation_id"] != exec.ID {
		t.Fatal("settled reverse link")
	}
	var got []byte
	offset := 0
	for i := 0; i < 20; i++ {
		r := call(s2, "operation_output_get", map[string]any{"operation_id": exec.ID, "stream": "stdout", "offset": offset, "limit": 5})
		data, e := base64.StdEncoding.DecodeString(r["content_base64"].(string))
		if e != nil || r["availability"] != "available" || r["exit_code"] != float64(7) || r["command_success"] != false {
			t.Fatal("archive read", e)
		}
		got = append(got, data...)
		offset = int(r["next_offset"].(float64))
		if r["ended"] == true {
			break
		}
	}
	if string(got) != content {
		t.Fatal("archive pagination")
	}
	missing := call(s2, "operation_output_get", map[string]any{"operation_id": exec.ID, "stream": "stderr"})
	if missing["availability"] != "unavailable" || missing["content_base64"] != "" {
		t.Fatal("missing archive not explicit")
	}
	db, e := sql.Open("sqlite3", "file:"+path+"?_busy_timeout=5000")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.ExecContext(ctx, "UPDATE exec_outputs SET observed_at=? WHERE operation_id=?", time.Now().Add(-8*24*time.Hour).UnixNano(), exec.ID); e != nil {
		t.Fatal(e)
	}
	expired := call(s2, "operation_output_get", map[string]any{"operation_id": exec.ID, "stream": "stdout"})
	if expired["availability"] != "expired" || expired["content_base64"] != "" || expired["exit_code"] != float64(7) {
		t.Fatal("expiry not explicit")
	}
	for _, name := range []string{"operation_get", "operation_recover", "box_activity_get", "operation_output_get"} {
		a := map[string]any{"operation_id": foreign.ID}
		switch name {
		case "operation_recover":
			a["mode"] = "inspect"
		case "box_activity_get":
			a["view"] = "evidence"
		case "operation_output_get":
			a["stream"] = "stdout"
		}
		deny(s2, name, a)
	}
	if e = st.RevokeToken(ctx, first.ID); e != nil {
		t.Fatal(e)
	}
	deny(s1, "operation_get", map[string]any{"operation_id": blocked.ID})
	if nav(s2, blocked.ID)["claim_protection"] != "RETAINED" {
		t.Fatal("second actor lost access")
	}
	if e = st.RevokeToken(ctx, second.ID); e != nil {
		t.Fatal(e)
	}
	deny(s2, "operation_output_get", map[string]any{"operation_id": exec.ID, "stream": "stdout"})
	deny(s2, "box_activity_get", map[string]any{"view": "evidence", "operation_id": lostID})
	var count int
	if e = db.QueryRow("SELECT count(*) FROM operations WHERE layer='bw-test'").Scan(&count); e != nil || count != 4 {
		t.Fatal("unexpected new original or recovery", count, e)
	}
	t.Log("S5A5_BOX_TWO_ACTOR_LOST_RECOVERY_ARCHIVE_HANDOFF=PASS")
}
