//go:build ignore

// S4-A installed-only acceptance: real OAuth grant refresh over HTTP followed
// by the SAME official MCP box_exec idempotency key. The native effect happens
// in the disposable owned Guest once; an independent Incus CLI read checks it.
// Do not run against shared L0/L1 or any existing/historical UNKNOWN claim.
package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

const originalKey = "s4-oauth-native-one-effect-001"
const guestName = "bw-s2-bh018-atomic-native"
const effectFile = "/tmp/s4-oauth-original-effect"

func must(ok bool, class string) {
	if !ok {
		panic("S4_A_INSTALLED_OAUTH_" + class)
	}
}

type bearer struct{ access string }

func (b bearer) RoundTrip(req *http.Request) (*http.Response, error) {
	copyReq := req.Clone(req.Context())
	copyReq.Header.Set("Authorization", "Bearer "+b.access)
	return http.DefaultTransport.RoundTrip(copyReq)
}

func call(ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) map[string]any {
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	must(err == nil && result != nil && !result.IsError, "OFFICIAL_MCP_"+name+"_REJECTED")
	encoded, err := json.Marshal(result.StructuredContent)
	must(err == nil, "MCP_STRUCTURE_ENCODING_FAILED")
	var body map[string]any
	must(json.Unmarshal(encoded, &body) == nil && body != nil, "MCP_STRUCTURE_INVALID")
	return body
}

func value(body map[string]any, key string) string {
	text, _ := body[key].(string)
	return text
}

func waitFor(ctx context.Context, session *mcp.ClientSession, id string) map[string]any {
	for i := 0; i < 90; i++ {
		body := call(ctx, session, "operation_get", map[string]any{
			"operation_id": id, "wait_ms": 1500,
		})
		op, ok := body["operation"].(map[string]any)
		must(ok && value(op, "operation_id") == id, "ORIGINAL_RECEIPT_ID_MISMATCH")
		switch value(op, "state") {
		case "SUCCEEDED":
			return op
		case "FAILED", "UNKNOWN", "CANCELLED":
			panic("S4_A_INSTALLED_OAUTH_NATIVE_ORIGINAL_NOT_SUCCEEDED")
		}
	}
	panic("S4_A_INSTALLED_OAUTH_NATIVE_WAIT_TIMEOUT")
}

func connect(ctx context.Context, endpoint, access string) *mcp.ClientSession {
	client := mcp.NewClient(&mcp.Implementation{Name: "s4-oauth-native", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: endpoint, HTTPClient: &http.Client{Transport: bearer{access: access}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	must(err == nil, "OFFICIAL_MCP_CONNECT_FAILED")
	return session
}

// These independently authored negative cases use the actual installed HTTP
// boundary. Refusal must leave both the admission count and Guest effect intact.
func refused(ctx context.Context, session *mcp.ClientSession, name string, args map[string]any, class string) {
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	must(err == nil && result != nil && result.IsError, class)
}

func unauthorized(ctx context.Context, endpoint, access, class string) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"s4-refused","version":"1"}}}`))
	must(err == nil, "NEGATIVE_HTTP_REQUEST_INVALID")
	request.Header.Set("Authorization", "Bearer "+access)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	response, err := http.DefaultClient.Do(request)
	must(err == nil && response != nil, "NEGATIVE_HTTP_READ_FAILED")
	defer response.Body.Close()
	must(response.StatusCode == http.StatusUnauthorized, class)
}

func main() {
	db := flag.String("db", "", "one-use installed SQLite ledger")
	project := flag.String("project", "", "single restricted Project")
	boxID := flag.String("box-id", "", "exact disposable owned Box UUID")
	endpoint := flag.String("endpoint", "", "single-use loopback installed MCP URL")
	flag.Parse()
	must(*db != "" && *project != "" && len(*boxID) >= 32 &&
		strings.HasPrefix(*endpoint, "http://127.0.0.1:"), "UNSAFE_RUNNER_TARGET")
	must(os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted", "NOT_DISPOSABLE_RUNNER")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	store, err := state.Open(*db)
	must(err == nil, "EXACT_LEDGER_UNAVAILABLE")
	defer store.Close()
	cfg, err := store.OAuthConfig(ctx)
	must(err == nil && !cfg.Enabled, "EXISTING_OAUTH_IDENTITY_PRESENT")
	passwordBytes := make([]byte, 32)
	_, err = rand.Read(passwordBytes)
	must(err == nil, "SECRET_GENERATION_FAILED")
	password := hex.EncodeToString(passwordBytes)
	const origin = "https://s4a-ephemeral-runner.example.com"
	const redirect = "https://chatgpt.com/connector_platform_oauth_redirect"
	if err := store.SetupOAuthPassword(ctx, password); err != nil {
		panic("S4_A_INSTALLED_OAUTH_SETUP_FAILED")
	}
	must(store.SetOAuthOrigin(ctx, origin) == nil, "ORIGIN_SETUP_FAILED")
	// This program uses the same durable OAuth issuance functions as the
	// standalone server, then uses real HTTP /token refresh and official /mcp.
	clientID, err := store.RegisterOAuthClient(ctx, "one-use S4 verifier", []string{redirect})
	must(err == nil, "CLIENT_SETUP_FAILED")
	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ._~"
	d := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(d[:])
	pending, err := store.BeginOAuth(ctx, clientID, redirect, "", challenge,
		origin+"/mcp", "s4-a-native-once-only")
	must(err == nil, "OAUTH_BEGIN_FAILED")
	code, _, _, err := store.ApproveOAuth(ctx, pending, password)
	must(err == nil, "OAUTH_APPROVAL_FAILED")
	first, err := store.ExchangeOAuth(ctx, clientID, redirect, code, verifier, origin+"/mcp")
	must(err == nil, "OAUTH_FIRST_ACCESS_FAILED")
	a, err := store.VerifyToken(ctx, first.Access)
	must(err == nil && a.ActorID == first.GrantID && a.ID != a.ActorID, "ORIGINAL_GRANT_ACTOR_INVALID")
	sessionA := connect(ctx, *endpoint, first.Access)
	defer sessionA.Close()
	argv := []string{"/bin/sh", "-c", "printf 'S4_OAUTH_ONCE\\n' >> " + effectFile}
	original := call(ctx, sessionA, "box_exec", map[string]any{
		"box_id": *boxID, "argv": argv, "idempotency_key": originalKey,
	})
	originalID := value(original, "operation_id")
	must(len(originalID) == 32 && original["duplicate"] != true, "ORIGINAL_ADMISSION_INVALID")
	terminal := waitFor(ctx, sessionA, originalID)
	nativeID := value(terminal, "native_operation_id")
	must(nativeID != "" && value(terminal, "state") == "SUCCEEDED", "NATIVE_ORIGINAL_NOT_VERIFIED")
	form := url.Values{
		"grant_type": {"refresh_token"}, "client_id": {clientID},
		"refresh_token": {first.Refresh}, "resource": {origin + "/mcp"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(*endpoint, "/mcp")+"/token", strings.NewReader(form.Encode()))
	must(err == nil, "REFRESH_REQUEST_FAILED")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := http.DefaultClient.Do(request)
	must(err == nil && response != nil && response.StatusCode == http.StatusOK,
		"REFRESH_HTTP_FAILED")
	var next struct {
		Access string `json:"access_token"`
	}
	err = json.NewDecoder(response.Body).Decode(&next)
	_ = response.Body.Close()
	must(err == nil && next.Access != "", "REFRESH_RESPONSE_INVALID")
	b, err := store.VerifyToken(ctx, next.Access)
	must(err == nil && b.ActorID == a.ActorID && b.ID != a.ID, "REFRESH_CHANGED_ACTOR")
	_, oldAccessErr := store.VerifyToken(ctx, first.Access)
	must(oldAccessErr != nil, "OLD_CREDENTIAL_STILL_AUTHORIZED")
	sessionB := connect(ctx, *endpoint, next.Access)
	defer sessionB.Close()
	replay := call(ctx, sessionB, "box_exec", map[string]any{
		"box_id": *boxID, "argv": argv, "idempotency_key": originalKey,
	})
	must(replay["duplicate"] == true && value(replay, "operation_id") == originalID &&
		value(replay, "tracking_id") == value(original, "tracking_id") &&
		value(replay, "native_operation_id") == nativeID &&
		value(replay, "state") == "SUCCEEDED", "REFRESH_REDISPATCHED_ORIGINAL")
	reloaded := call(ctx, sessionB, "operation_get", map[string]any{"operation_id": originalID})
	reloadedOp, ok := reloaded["operation"].(map[string]any)
	must(ok && value(reloadedOp, "native_operation_id") == nativeID,
		"REFRESH_LOST_ORIGINAL_NATIVE_RECEIPT")
	readDB, err := sql.Open("sqlite3", "file:"+*db+"?mode=ro&_query_only=1")
	must(err == nil, "NEGATIVE_SQLITE_READ_FAILED")
	defer readDB.Close()
	admissions := func() int {
		var count int
		must(readDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations").Scan(&count) == nil,
			"NEGATIVE_SQLITE_COUNT_FAILED")
		return count
	}
	beforeNegatives := admissions()
	unauthorized(ctx, *endpoint, first.Access, "OLD_CREDENTIAL_HTTP_ACCEPTED")
	issueGrant := func(scope, label string) state.OAuthPair {
		pending, err := store.BeginOAuth(ctx, clientID, redirect, scope, challenge,
			origin+"/mcp", "s4-a-negative-"+label)
		must(err == nil, "NEGATIVE_GRANT_BEGIN_FAILED")
		code, _, _, err := store.ApproveOAuth(ctx, pending, password)
		must(err == nil, "NEGATIVE_GRANT_APPROVAL_FAILED")
		pair, err := store.ExchangeOAuth(ctx, clientID, redirect, code, verifier, origin+"/mcp")
		must(err == nil, "NEGATIVE_GRANT_EXCHANGE_FAILED")
		return pair
	}
	other := issueGrant("", "independent")
	must(other.GrantID != first.GrantID, "INDEPENDENT_GRANT_REUSED_ACTOR")
	otherActor, err := store.VerifyToken(ctx, other.Access)
	must(err == nil && otherActor.ActorID == other.GrantID, "INDEPENDENT_GRANT_ACTOR_INVALID")
	otherSession := connect(ctx, *endpoint, other.Access)
	defer otherSession.Close()
	// Admin may read another actor's project-local history. Its idempotency
	// namespace is nevertheless separate: the SAME key with different safe
	// argv must admit a new harmless Exec, rather than borrow or conflict.
	separateActor := func(session *mcp.ClientSession, actor state.Token, label string) {
		admitted := call(ctx, session, "box_exec", map[string]any{
			"box_id": *boxID, "argv": []string{"/bin/true"}, "idempotency_key": originalKey,
		})
		id := value(admitted, "operation_id")
		must(len(id) == 32 && id != originalID && admitted["duplicate"] != true,
			"OTHER_ACTOR_BORROWED_ORIGINAL")
		terminal := waitFor(ctx, session, id)
		must(value(terminal, "native_operation_id") != "" &&
			value(terminal, "native_operation_id") != nativeID, "OTHER_ACTOR_BORROWED_NATIVE")
		var storedActor, storedCredential, storedUUID, storedProject, storedNative string
		must(readDB.QueryRowContext(ctx,
			"SELECT actor_id,token_id,box_uuid,layer,native_operation_id FROM operations WHERE id=?", id).Scan(
			&storedActor, &storedCredential, &storedUUID, &storedProject, &storedNative) == nil &&
			storedActor == actor.ActorID && storedCredential == actor.ID && storedUUID == *boxID &&
			storedProject == *project && storedNative == value(terminal, "native_operation_id"),
			"OTHER_ACTOR_SQLITE_NATIVE_IDENTITY_MISMATCH")
		fmt.Printf("S4_A_OAUTH_SEPARATE_ACTOR=%s BW_ID=%s NATIVE_ID=%s\n",
			label, id, value(terminal, "native_operation_id"))
	}
	separateActor(otherSession, otherActor, "INDEPENDENT_GRANT")
	// Separate authorized Actor history is visible; revoked grants are not.
	recovered := call(ctx, otherSession, "operation_get", map[string]any{"operation_id": originalID})
	must(recovered["operation"] != nil, "OTHER_ACTOR_CANNOT_TAKE_OVER_ORIGINAL")
	revoked := issueGrant("", "revoked-independent")
	must(store.RevokeOAuthGrant(ctx, revoked.GrantID) == nil, "INDEPENDENT_GRANT_REVOKE_FAILED")
	unauthorized(ctx, *endpoint, revoked.Access, "REVOKED_INDEPENDENT_GRANT_HTTP_ACCEPTED")
	manualAccess, manual, err := store.IssueToken(ctx, time.Minute)
	must(err == nil && manual.ActorID == manual.ID, "MANUAL_TOKEN_ACTOR_INVALID")
	manualSession := connect(ctx, *endpoint, manualAccess)
	defer manualSession.Close()
	separateActor(manualSession, manual, "MANUAL_TOKEN")
	must(store.RevokeToken(ctx, manual.ID) == nil, "MANUAL_TOKEN_REVOKE_FAILED")
	unauthorized(ctx, *endpoint, manualAccess, "REVOKED_MANUAL_HTTP_ACCEPTED")
	must(store.RevokeOAuthGrant(ctx, first.GrantID) == nil, "GRANT_REVOKE_FAILED")
	unauthorized(ctx, *endpoint, next.Access, "REVOKED_GRANT_HTTP_ACCEPTED")
	must(admissions() == beforeNegatives+2, "REFUSED_CALL_CREATED_ADMISSION")
	// Independent source of the exact Guest effect count. Native Incus CLI
	// reads a disposable guest file, bypassing BoxWeave's MCP and SQLite.
	nativeRead := exec.CommandContext(ctx, "incus", "exec", guestName, "--project", *project,
		"--", "/bin/cat", effectFile)
	guestOutput, err := nativeRead.Output()
	must(err == nil && string(guestOutput) == "S4_OAUTH_ONCE\n",
		"INDEPENDENT_GUEST_SIDE_EFFECT_NOT_EXACTLY_ONE")
	must(store.SetOAuthOrigin(ctx, "") == nil, "DISABLE_EPHEMERAL_OAUTH_FAILED")
	fmt.Printf("S4_A_OAUTH_NATIVE_ORIGINAL_BW_ID=%s NATIVE_ID=%s BOX_UUID=%s\n",
		originalID, nativeID, *boxID)
	fmt.Println("S4_A_REAL_INSTALLED_OAUTH_REFRESH_GRANT_REPLAY_ONCE=PASS")
	fmt.Println("S4_A_INSTALLED_OAUTH_REVOKED_GRANT_MANUAL_NO_ADMISSION=PASS")
	fmt.Println("S4_A_INDEPENDENT_GUEST_ONE_EFFECT=PASS")
}
