//go:build ignore

// CI-only official MCP SDK proof of credential revocation/expiry during
// a private operation_get long poll. Controller-only ADMITTED rows, no Incus writes.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(cp)
}
func check(ok bool, reason string) {
	if !ok {
		panic("S2_P1_TOKEN_INFLIGHT_FAIL: " + reason)
	}
}
func session(ctx context.Context, token string) *mcp.ClientSession {
	client := mcp.NewClient(&mcp.Implementation{Name: "s2-inflight-auth-blackbox", Version: "1"}, nil)
	s, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             "http://127.0.0.1:18659/mcp",
		HTTPClient:           &http.Client{Transport: bearer{token}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	check(err == nil && s != nil, "authenticated MCP session not available")
	return s
}

type reply struct {
	result *mcp.CallToolResult
	err    error
}

func one(ctx context.Context, st *state.Store, project, mode string) {
	ttl := time.Hour
	if mode == "expire" {
		ttl = 3 * time.Second
	}
	raw, actor, err := st.IssueToken(ctx, "viewer", ttl)
	check(err == nil && len(raw) == 68 && actor.ID != "", "test viewer not created")
	original, err := st.Admit(ctx, state.Admission{
		Layer: project, ActorID: actor.ID, TokenID: actor.ID,
		Key: "s2-p1-inflight-" + mode + "-original-key", RequestID: "initial-private-poll",
		Action: "box_exec", SafeIntent: []byte("{\"kind\":\"never-dispatched\"}"),
		WorkRef: "lab/inflight-credential", AgentRef: mode,
	})
	check(err == nil && original.State == "ADMITTED", "test-only original not journaled")
	_, err = st.VerifyToken(ctx, raw)
	check(err == nil, "test bearer was invalid before request")
	s := session(ctx, raw)
	defer s.Close()
	done := make(chan reply, 1)
	go func() {
		out, e := s.CallTool(ctx, &mcp.CallToolParams{
			Name: "operation_get", Arguments: map[string]any{"operation_id": original.ID, "wait_ms": 5000},
		})
		done <- reply{out, e}
	}()
	select {
	case <-done:
		panic("S2_P1_TOKEN_INFLIGHT_FAIL: request completed before invalidation")
	case <-time.After(750 * time.Millisecond):
	}
	if mode == "revoke" {
		err = st.RevokeToken(ctx, actor.ID)
		check(err == nil, "exact test credential revocation failed")
	}
	out := <-done
	check(out.err == nil || !strings.Contains(out.err.Error(), "deadline exceeded"),
		"transport timed out instead of checking live credential")
	check(out.err != nil || out.result != nil && out.result.IsError,
		"invalidated credential received private operation result")
	if out.result != nil {
		b, e := json.Marshal(out.result.StructuredContent)
		check(e == nil && !strings.Contains(string(b), original.ID),
			"private original included in invalidated credential response")
	}
	_, err = st.VerifyToken(ctx, raw)
	check(err != nil, "credential was not expired/revoked at response")
	saved, err := st.Get(ctx, original.ID)
	check(err == nil && saved.ID == original.ID && saved.State == "ADMITTED" &&
		saved.NativeOperationID == "", "credential read affected original operation state")
	if mode == "revoke" {
		fmt.Println("S2_P1_INSTALLED_MCP_REVOKED_DURING_READ_NO_LEAK=PASS")
	}
	if mode == "expire" {
		fmt.Println("S2_P1_INSTALLED_MCP_EXPIRED_DURING_READ_NO_LEAK=PASS")
	}
}

// A signed history cursor is fixed to the first-page admission watermark,
// while the state filter must reflect real durable changes on later pages.
// The real installed HTTP handler is called using the official Go MCP SDK.
func statusChangeDuringInstalledPaging(ctx context.Context, st *state.Store, project string) {
	raw, actor, err := st.IssueToken(ctx, "viewer", time.Hour)
	check(err == nil && actor.ID != "", "cannot create isolated query actor")
	ids := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		receipt, e := st.Admit(ctx, state.Admission{
			Layer: project, ActorID: actor.ID, TokenID: actor.ID,
			Key:       fmt.Sprintf("s2-p1-installed-cursor-%02d-original", i),
			RequestID: fmt.Sprintf("cursor-request-%02d", i),
			Action:    "box_exec", SafeIntent: []byte("{\"kind\":\"controller-only\"}"),
			WorkRef: "lab/p1-installed-cursor", AgentRef: "p1-cursor",
		})
		check(e == nil && receipt.State == "ADMITTED",
			"cannot seed durable private original for installed cursor")
		ids = append(ids, receipt.ID)
	}
	s := session(ctx, raw)
	defer s.Close()
	read := func(args map[string]any) map[string]any {
		r, e := s.CallTool(ctx, &mcp.CallToolParams{Name: "operation_list", Arguments: args})
		check(e == nil && r != nil && !r.IsError, "installed private history query failed")
		b, e := json.Marshal(r.StructuredContent)
		check(e == nil, "installed history response cannot be decoded")
		var body map[string]any
		check(json.Unmarshal(b, &body) == nil, "invalid installed history JSON")
		page, ok := body["page"].(map[string]any)
		check(ok, "missing installed history page")
		return page
	}
	args := map[string]any{
		"work_ref": "lab/p1-installed-cursor", "state": "ADMITTED", "limit": 2,
	}
	first := read(args)
	rows1, ok := first["operations"].([]any)
	check(ok && len(rows1) == 2, "first page must contain two ADMITTED originals")
	cursor, ok := first["next_cursor"].(string)
	check(ok && cursor != "", "first installed page lacked HMAC cursor")
	check(st.CancelQueued(ctx, project, ids[4]) == nil,
		"could not transition unvisited original to CANCELLED")
	args["cursor"] = cursor
	second := read(args)
	rows2, ok := second["operations"].([]any)
	check(ok && len(rows2) == 2, "cursor unexpectedly counted cancelled original")
	seen := map[string]bool{}
	for _, item := range append(rows1, rows2...) {
		row, ok := item.(map[string]any)
		check(ok && row["state"] == "ADMITTED", "cursor emitted out-of-filter status")
		id, ok := row["operation_id"].(string)
		if !ok {
			id, _ = row["id"].(string)
		}
		check(id != "" && id != ids[4] && !seen[id], "cursor leaked/duplicated original")
		seen[id] = true
	}
	check(len(seen) == 4, "status-changing cursor lost existing ADMITTED original")
	fresh := read(map[string]any{
		"work_ref": "lab/p1-installed-cursor", "state": "CANCELLED", "limit": 10,
	})
	rows, ok := fresh["operations"].([]any)
	check(ok && len(rows) == 1, "fresh cursor did not reflect newly CANCELLED original")
	row, ok := rows[0].(map[string]any)
	check(ok && row["state"] == "CANCELLED", "fresh cursor misclassified new state")
	fmt.Println("S2_P1_REAL_INSTALLED_MCP_STATUS_CHANGED_DURING_SIGNED_CURSOR=PASS")
}

func main() {
	dir, project := os.Getenv("BW_S2_AUTH_LAB"), os.Getenv("BW_S2_AUTH_PROJECT")
	check(os.Geteuid() != 0 && os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		strings.HasPrefix(dir, "/tmp/bw-v1-s2-ghci.") &&
		filepath.Base(dir) == "p1-token-inflight" &&
		strings.HasPrefix(project, "user-"), "test outside disposable confined CI")
	ctx, cancel := context.WithTimeout(context.Background(), 27*time.Second)
	defer cancel()
	st, err := state.Open(filepath.Join(dir, "ledger.sqlite"))
	check(err == nil, "installed test SQLite unavailable")
	defer st.Close()
	one(ctx, st, project, "revoke")
	one(ctx, st, project, "expire")
	statusChangeDuringInstalledPaging(ctx, st, project)
	fmt.Println("S2_P1_INSTALLED_INFLIGHT_BEARER_REVOCATION_AND_EXPIRY=PASS")
}
