package control

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

func activityOAuthSession(t *testing.T, ctx context.Context, endpoint, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "activity-p1-oauth", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: endpoint + "/mcp",
		HTTPClient: &http.Client{Transport: hostOAuthTransport{rawToken: token}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func activityOAuthRead(t *testing.T, ctx context.Context, session *mcp.ClientSession, view string, args map[string]any, want string) string {
	t.Helper()
	if args == nil {
		args = map[string]any{}
	}
	args["view"] = view
	got, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "box_activity_get", Arguments: args})
	if err != nil || got == nil || got.IsError {
		t.Fatalf("authorized Activity %s refused: %+v %v", view, got, err)
	}
	raw, err := json.Marshal(got.StructuredContent)
	if err != nil || !strings.Contains(string(raw), want) {
		t.Fatalf("Activity %s lost evidence %q: %s, %v", view, want, raw, err)
	}
	return string(raw)
}

func activityOAuthMustDeny(t *testing.T, ctx context.Context, session *mcp.ClientSession, view string, args map[string]any) {
	t.Helper()
	if args == nil {
		args = map[string]any{}
	}
	args["view"] = view
	got, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "box_activity_get", Arguments: args})
	if err == nil && got != nil && !got.IsError {
		t.Fatalf("invalidated OAuth bearer disclosed Activity %s: %+v", view, got.StructuredContent)
	}
}

// Real OAuth DCR, browser grant, token rotation and official SDK calls all
// operate on a single authentic HTTP MCP server; no mocked authorization.
func TestActivityOAuthRefreshRotationRevokedGrantAndForeignLayer(t *testing.T) {
	st, ts, pwd := newOAuthServerTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	first := completeOAuth(t, ts, pwd, "")
	token, err := st.VerifyOAuthAccess(ctx, first.access)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := st.Admit(ctx, state.Admission{
		Layer: "test", ActorID: token.ActorID, TokenID: token.ID,
		Key: "activity-oauth-p1-original-20261119", RequestID: "oauth-p1",
		Action: "box_exec", SafeIntent: []byte("{}"),
		ResourceKey: "box:owned", BoxUUID: "bcfe32e0-3acf-4f21-8a66-332a84d741a4",
		Incarnation: "bcfe32e0-3acf-4f21-8a66-332a84d741a4",
		WorkRef: "p1/oauth-evidence",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Claim(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.Unknown(ctx, rec.ID, "DISPATCHING", "UNVERIFIED_NATIVE_REPLY"); err != nil {
		t.Fatal(err)
	}
	foreign, err := st.Admit(ctx, state.Admission{
		Layer: "other-project", ActorID: token.ActorID, TokenID: token.ID,
		Key: "activity-oauth-foreign-layer-20261119", RequestID: "oauth-foreign",
		Action: "box_exec", SafeIntent: []byte("{}"), ResourceKey: "box:foreign",
	})
	if err != nil {
		t.Fatal(err)
	}
	sess := activityOAuthSession(t, ctx, ts.URL, first.access)
	activityOAuthRead(t, ctx, sess, "issues", map[string]any{"work_ref": "p1/oauth-evidence"}, rec.ID)
	activityOAuthMustDeny(t, ctx, sess, "evidence", map[string]any{"operation_id": foreign.ID})

	// Rotation consumes the old refresh and revokes prior access inside one
	// SQLite transaction. Same OAuth Grant remains the same Actor identity.
	form := url.Values{"grant_type": {"refresh_token"}, "client_id": {first.client},
		"refresh_token": {first.refresh}, "resource": {oauthPublicOrigin + "/mcp"}}
	res := oauthReq(t, ts, "POST", "/token", form.Encode(), "application/x-www-form-urlencoded", "", oauthPublicHost)
	if res.StatusCode != 200 {
		t.Fatalf("refresh HTTP %d", res.StatusCode)
	}
	pair := jsonFrom(t, res)
	newAccess := pair["access_token"].(string)
	newRefresh := pair["refresh_token"].(string)
	refreshed, err := st.VerifyOAuthAccess(ctx, newAccess)
	if err != nil || refreshed.ActorID != token.ActorID || refreshed.ID == token.ID {
		t.Fatalf("rotation did not retain Actor and issue new Token: %+v %v", refreshed, err)
	}
	for _, view := range []string{"overview", "issues", "evidence"} {
		params := map[string]any{}
		if view == "evidence" {
			params["operation_id"] = rec.ID
		}
		activityOAuthMustDeny(t, ctx, sess, view, params)
	}
	newSess := activityOAuthSession(t, ctx, ts.URL, newAccess)
	activityOAuthRead(t, ctx, newSess, "evidence", map[string]any{"operation_id": rec.ID}, "UNVERIFIED_NATIVE_REPLY")

	// Independent OAuth grant/Actor can recover the same-Layer original but
	// cannot guess its way into a foreign Layer through labels/IDs.
	next := completeOAuth(t, ts, pwd, "")
	second, err := st.VerifyOAuthAccess(ctx, next.access)
	if err != nil || second.ActorID == token.ActorID {
		t.Fatalf("fresh OAuth grant lacked independent Actor: %+v %v", second, err)
	}
	nextSess := activityOAuthSession(t, ctx, ts.URL, next.access)
	activityOAuthRead(t, ctx, nextSess, "overview", map[string]any{"work_ref": "p1/oauth-evidence"}, rec.ID)
	activityOAuthMustDeny(t, ctx, nextSess, "evidence", map[string]any{"operation_id": foreign.ID})

	// A revoked grant may neither read Activity nor use its current refresh.
	if err := st.RevokeOAuthGrant(ctx, refreshed.ActorID); err != nil {
		t.Fatal(err)
	}
	for _, view := range []string{"overview", "issues", "evidence"} {
		params := map[string]any{}
		if view == "evidence" {
			params["operation_id"] = rec.ID
		}
		activityOAuthMustDeny(t, ctx, newSess, view, params)
	}
	form.Set("refresh_token", newRefresh)
	res = oauthReq(t, ts, "POST", "/token", form.Encode(), "application/x-www-form-urlencoded", "", oauthPublicHost)
	if res.StatusCode != 400 {
		t.Fatalf("revoked OAuth grant refreshed again: HTTP %d", res.StatusCode)
	}
	res.Body.Close()
	activityOAuthRead(t, ctx, nextSess, "issues", map[string]any{"work_ref": "p1/oauth-evidence"}, rec.ID)
}

// Concurrent reads must never disclose data after the revocation transaction
// completes. Reads racing before commit may be allowed; every later attempt
// on the same session must fail, even under concurrent new-grant reads.
func TestActivityOAuthConcurrentReadersRevokedGrantNoPostRevokeLeak(t *testing.T) {
	st, ts, pwd := newOAuthServerTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	first := completeOAuth(t, ts, pwd, "")
	tok, err := st.VerifyOAuthAccess(ctx, first.access)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := st.Admit(ctx, state.Admission{
		Layer: "test", ActorID: tok.ActorID, TokenID: tok.ID,
		Key: "activity-p1-concurrent-oauth-20261119", RequestID: "p1-auth-race",
		Action: "box_exec", SafeIntent: []byte("{}"), ResourceKey: "box:auth-race",
		WorkRef: "p1/auth-race",
	})
	if err != nil {
		t.Fatal(err)
	}
	sess := activityOAuthSession(t, ctx, ts.URL, first.access)
	activityOAuthRead(t, ctx, sess, "overview", map[string]any{"work_ref": "p1/auth-race"}, rec.ID)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wg.Done()
			<-start
			// Responses beginning before the revoke commit are not evidence
			// of forbidden post-revoke disclosure.
			_, _ = sess.CallTool(ctx, &mcp.CallToolParams{Name: "box_activity_get",
				Arguments: map[string]any{"view": "overview", "limit": 1}})
		}()
	}
	close(start)
	if err := st.RevokeOAuthGrant(ctx, tok.ActorID); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	for i := 0; i < 5; i++ {
		for _, view := range []string{"overview", "issues", "evidence"} {
			params := map[string]any{}
			if view == "evidence" {
				params["operation_id"] = rec.ID
			}
			activityOAuthMustDeny(t, ctx, sess, view, params)
		}
	}
}
