package control

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

const oauthPublicOrigin = "https://lab.trycloudflare.com"
const oauthPublicHost = "lab.trycloudflare.com"
const oauthCallback = "https://chatgpt.com/connector_platform_oauth_redirect"

func newOAuthServerTest(t *testing.T) (*state.Store, *httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	st, err := state.Open(dir + "/state.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	pwd := "fixture-approval-password-1111"
	if err = st.SetupOAuthPassword(context.Background(), pwd); err != nil {
		t.Fatal(err)
	}
	if err = st.SetOAuthOrigin(context.Background(), oauthPublicOrigin); err != nil {
		t.Fatal(err)
	}
	svc := &Service{Store: st, Native: native.Reader{Socket: "/nonexistent", Project: "test"}, Listener: "127.0.0.1:18650", BuildVersion: "oauth-unit"}
	ts := httptest.NewServer(svc.Handler())
	t.Cleanup(ts.Close)
	return st, ts, pwd
}
func oauthReq(t *testing.T, ts *httptest.Server, method, path, body, contentType, origin, host string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if host != "" {
		req.Host = host
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}
func jsonFrom(t *testing.T, r *http.Response) map[string]any {
	t.Helper()
	defer r.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(r.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

type oauthFixtureHTTP struct{ client, challenge, verifier, code, access, refresh, role string }

func completeOAuth(t *testing.T, ts *httptest.Server, pwd, scope string) oauthFixtureHTTP {
	t.Helper()
	body := `{"client_name":"ChatGPT browser","redirect_uris":["https://chatgpt.com/connector_platform_oauth_redirect"],"token_endpoint_auth_method":"none"}`
	rr := oauthReq(t, ts, "POST", "/register", body, "application/json", "", "")
	if rr.StatusCode != 201 {
		t.Fatalf("DCR HTTP %d: %+v", rr.StatusCode, jsonFrom(t, rr))
	}
	reg := jsonFrom(t, rr)
	clientID := reg["client_id"].(string)
	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNO1234567890._~-"
	d := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(d[:])
	query := url.Values{"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {oauthCallback}, "code_challenge_method": {"S256"}, "code_challenge": {challenge}, "resource": {oauthPublicOrigin + "/mcp"}, "scope": {scope}, "state": {"state-for-browser-123456789"}}
	page := oauthReq(t, ts, "GET", "/authorize?"+query.Encode(), "", "", "", oauthPublicHost)
	if page.StatusCode != 200 {
		t.Fatalf("authorize GET %d", page.StatusCode)
	}
	if rp := page.Header.Get("Referrer-Policy"); rp != "same-origin" {
		t.Fatalf("wrong form referrer: %q", rp)
	}
	if csp := page.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "form-action 'self' https://chatgpt.com;") {
		t.Fatalf("bad callback CSP: %q", csp)
	}
	data, _ := io.ReadAll(page.Body)
	page.Body.Close()
	re := regexp.MustCompile(`name="pending" value="([^"]+)"`)
	m := re.FindSubmatch(data)
	if len(m) != 2 {
		t.Fatalf("pending absent")
	}
	form := url.Values{"pending": {string(m[1])}, "password": {pwd}, "approve": {"yes"}}.Encode()
	// Browser consent MUST carry the canonical public Origin.
	rejected := oauthReq(t, ts, "POST", "/authorize", form, "application/x-www-form-urlencoded", "null", oauthPublicHost)
	if rejected.StatusCode != 403 {
		t.Fatalf("Origin null accepted %d", rejected.StatusCode)
	}
	rejected.Body.Close()
	consent := oauthReq(t, ts, "POST", "/authorize", form, "application/x-www-form-urlencoded", oauthPublicOrigin, oauthPublicHost)
	if consent.StatusCode != 302 {
		t.Fatalf("authorize POST: HTTP %d", consent.StatusCode)
	}
	redirect := consent.Header.Get("Location")
	consent.Body.Close()
	u, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "https" || u.Host != "chatgpt.com" || u.Query().Get("state") != "state-for-browser-123456789" {
		t.Fatalf("invalid exact callback")
	}
	code := u.Query().Get("code")
	if code == "" {
		t.Fatal("missing code")
	}
	replay := oauthReq(t, ts, "POST", "/authorize", form, "application/x-www-form-urlencoded", oauthPublicOrigin, oauthPublicHost)
	if replay.StatusCode != 403 {
		t.Fatalf("replayed approval accepted %d", replay.StatusCode)
	}
	replay.Body.Close()
	tok := url.Values{"client_id": {clientID}, "grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {oauthCallback}, "code_verifier": {verifier}, "resource": {oauthPublicOrigin + "/mcp"}}
	wrong := url.Values{"client_id": {clientID}, "grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {oauthCallback}, "code_verifier": {"bad-verifier"}, "resource": {oauthPublicOrigin + "/mcp"}}
	invalid := oauthReq(t, ts, "POST", "/token", wrong.Encode(), "application/x-www-form-urlencoded", "", oauthPublicHost)
	if invalid.StatusCode != 400 {
		t.Fatalf("invalid PKCE accepted %d", invalid.StatusCode)
	}
	invalid.Body.Close()
	issued := oauthReq(t, ts, "POST", "/token", tok.Encode(), "application/x-www-form-urlencoded", "", oauthPublicHost)
	if issued.StatusCode != 200 {
		t.Fatalf("token HTTP %d: %+v", issued.StatusCode, jsonFrom(t, issued))
	}
	pair := jsonFrom(t, issued)
	if pair["token_type"] != "Bearer" || pair["scope"] != "mcp:"+expectedOAuthRole(scope) || pair["expires_in"] != float64(3600) {
		t.Fatalf("bad issued token metadata: %+v", pair)
	}
	return oauthFixtureHTTP{client: clientID, challenge: challenge, verifier: verifier, code: code, access: pair["access_token"].(string), refresh: pair["refresh_token"].(string), role: scope}
}

type hostOAuthTransport struct{ rawToken string }

func (h hostOAuthTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Host = oauthPublicHost
	cp.Header.Set("Authorization", "Bearer "+h.rawToken)
	return http.DefaultTransport.RoundTrip(cp)
}
func TestEmbeddedOAuthMetadataDCRBrowserCallbackAndSDK(t *testing.T) {
	st, ts, pwd := newOAuthServerTest(t)
	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp"} {
		res := oauthReq(t, ts, "GET", path, "", "", "", oauthPublicHost)
		if res.StatusCode != 200 {
			t.Fatalf("metadata %s: %d", path, res.StatusCode)
		}
		data := jsonFrom(t, res)
		if data["resource"] != oauthPublicOrigin+"/mcp" {
			t.Fatalf("wrong resource %+v", data)
		}
	}
	as := jsonFrom(t, oauthReq(t, ts, "GET", "/.well-known/oauth-authorization-server", "", "", "", oauthPublicHost))
	if as["issuer"] != oauthPublicOrigin || as["registration_endpoint"] != oauthPublicOrigin+"/register" {
		t.Fatalf("AS metadata %+v", as)
	}
	for _, tc := range []struct {
		name, host, origin string
		want               int
	}{
		{"foreign-host", "example.invalid", "", 403},
		{"forwarded", "attacker.invalid", "", 403},
		{"null-origin", oauthPublicHost, "null", 403},
		{"host-valid", oauthPublicHost, "", 401},
	} {
		res := oauthReq(t, ts, "POST", "/mcp", "{}", "application/json", tc.origin, tc.host)
		if res.StatusCode != tc.want {
			t.Fatalf("%s returned %d, want %d", tc.name, res.StatusCode, tc.want)
		}
		if tc.want == 401 && !strings.Contains(res.Header.Get("WWW-Authenticate"), oauthPublicOrigin+"/.well-known/oauth-protected-resource/mcp") {
			t.Fatalf("missing RFC9728 auth challenge")
		}
		res.Body.Close()
	}
	f := completeOAuth(t, ts, pwd, "mcp:viewer mcp:admin")
	if tok, err := st.VerifyToken(context.Background(), f.access); err != nil || tok.Role != "admin" {
		t.Fatalf("token not verified: %v", err)
	}
	sessctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "real-oauth-sdk-client", Version: "1"}, nil)
	session, err := client.Connect(sessctx, &mcp.StreamableClientTransport{
		Endpoint: ts.URL + "/mcp", HTTPClient: &http.Client{Transport: hostOAuthTransport{rawToken: f.access}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatalf("MCP initialize: %v", err)
	}
	defer session.Close()
	tools, err := session.ListTools(sessctx, nil)
	if err != nil || len(tools.Tools) != 13 {
		t.Fatalf("authenticated official SDK tools/list: %v tools=%d", err, len(tools.Tools))
	}
	status, err := session.CallTool(sessctx, &mcp.CallToolParams{Name: "host_status", Arguments: map[string]any{}})
	if err != nil || status.IsError {
		t.Fatalf("host_status: %+v %v", status, err)
	}
	form := url.Values{"grant_type": {"refresh_token"}, "client_id": {f.client}, "refresh_token": {f.refresh}, "resource": {oauthPublicOrigin + "/mcp"}}
	refreshed := oauthReq(t, ts, "POST", "/token", form.Encode(), "application/x-www-form-urlencoded", "", oauthPublicHost)
	if refreshed.StatusCode != 200 {
		t.Fatalf("refresh HTTP %d", refreshed.StatusCode)
	}
	newPair := jsonFrom(t, refreshed)
	if _, err = st.VerifyToken(context.Background(), f.access); err == nil {
		t.Fatal("old access not revoked")
	}
	if _, err = st.VerifyToken(context.Background(), newPair["access_token"].(string)); err != nil {
		t.Fatalf("refreshed access: %v", err)
	}
}
func TestEmbeddedOAuthMethodOriginAndDCRNegatives(t *testing.T) {
	_, ts, pwd := newOAuthServerTest(t)
	_ = pwd
	for _, tc := range []struct {
		method, path, body, typ, origin, host string
		want                                  int
	}{
		{"POST", "/", "{}", "application/json", "", oauthPublicHost, 404},
		{"GET", "/token", "", "", "", oauthPublicHost, 405},
		{"POST", "/register", `{"client_name":"evil","redirect_uris":["http://evil.com/cb"]}`, "application/json", "", oauthPublicHost, 400},
		{"POST", "/register", `{"client_name":"evil","redirect_uris":["https://evil.com/cb"],"token_endpoint_auth_method":"client_secret_basic"}`, "application/json", "", oauthPublicHost, 400},
		{"POST", "/token", "grant_type=password&client_id=bad", "application/x-www-form-urlencoded", "", oauthPublicHost, 400},
		{"POST", "/mcp", "{}", "application/json", "https://attacker.invalid", oauthPublicHost, 403},
		{"POST", "/mcp", "{}", "application/json", "", "localhost.evil:1234", 403},
	} {
		res := oauthReq(t, ts, tc.method, tc.path, tc.body, tc.typ, tc.origin, tc.host)
		if res.StatusCode != tc.want {
			t.Fatalf("%s %s: HTTP %d want %d", tc.method, tc.path, res.StatusCode, tc.want)
		}
		res.Body.Close()
	}
}
func TestEmbeddedOAuthJSONResponsesNoSSE(t *testing.T) {
	_, ts, pwd := newOAuthServerTest(t)
	f := completeOAuth(t, ts, pwd, "mcp:admin")
	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"no-sse","version":"1"}}}`
	req, err := http.NewRequest("POST", ts.URL+"/mcp", bytes.NewBufferString(init))
	if err != nil {
		t.Fatal(err)
	}
	req.Host = oauthPublicHost
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+f.access)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("initialize HTTP %d", res.StatusCode)
	}
	if strings.Contains(res.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatal("SSE incompatible with Quick Tunnel")
	}
	if !strings.Contains(res.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("non-JSON response %q", res.Header.Get("Content-Type"))
	}
}

func expectedOAuthRole(scope string) string {
	for _, name := range strings.Fields(scope) {
		if name == "mcp:admin" {
			return "admin"
		}
	}
	return "viewer"
}
func TestEmbeddedOAuthViewerDeniedExecAndToolRefreshWithoutParams(t *testing.T) {
	_, ts, pwd := newOAuthServerTest(t)
	f := completeOAuth(t, ts, pwd, "mcp:viewer")
	client := mcp.NewClient(&mcp.Implementation{Name: "viewer-negative", Version: "1"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sess, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp", HTTPClient: &http.Client{Transport: hostOAuthTransport{rawToken: f.access}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	result, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "box_exec", Arguments: map[string]any{"box_id": "nonexistent-owned-id", "idempotency_key": "viewer-denied-intent", "argv": []string{"true"}}})
	if err == nil && (result == nil || !result.IsError) {
		t.Fatalf("viewer executed write: %+v", result)
	}
	payload := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	req, err := http.NewRequest("POST", ts.URL+"/mcp", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Host = oauthPublicHost
	req.Header.Set("Authorization", "Bearer "+f.access)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || !bytes.Contains(raw, []byte("tools")) {
		t.Fatalf("omitted optional params rejected HTTP %d", res.StatusCode)
	}
}