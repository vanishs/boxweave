//go:build ignore

// Test-only SQLite final-receipt failure after an ACTUAL native Incus Create
// binds its exact UUID. The trigger exists only in a disposable active test
// DB; it is never installed by the product or on shared/retained databases.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	_ "github.com/mattn/go-sqlite3"
)

type bearer struct{ token string }

func (t bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(cp)
}
func ensure(ok bool, what string) {
	if !ok {
		panic("S2_P1_BOUND_CREATE_FAULT_FAIL: "+what)
	}
}
func invoke(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) (map[string]any, error) {
	r, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return nil, err
	}
	if r == nil || r.IsError {
		return nil, fmt.Errorf("MCP tool refusal: %s", name)
	}
	var out map[string]any
	data, err := json.Marshal(r.StructuredContent)
	if err == nil {
		err = json.Unmarshal(data, &out)
	}
	return out, err
}
func stringField(m map[string]any, name string) string { v, _ := m[name].(string); return v }
func main() {
	dir := os.Getenv("BW_S2_BOUND_LAB")
	token := os.Getenv("BW_S2_BOUND_TOKEN")
	project := os.Getenv("BW_S2_BOUND_PROJECT")
	ensure(os.Geteuid() != 0 && os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		filepath.Base(dir) == "p1-bound-receipt" &&
		strings.HasPrefix(dir, "/tmp/bw-v1-s2-ghci.") &&
		strings.HasPrefix(project, "user-") && token != "",
		"original fixture must be one-use restricted CI")
	ctx, cancel := context.WithTimeout(context.Background(), 170*time.Second)
	defer cancel()
	db, err := sql.Open("sqlite3", filepath.Join(dir, "ledger.sqlite"))
	ensure(err == nil, "unable to open exact disposable ledger")
	defer db.Close()
	// No schema rewrite or external changes: install a test-only trigger AFTER
	// service startup so a successful real native Create can ACK and bind UUID
	// but the last controller-only SUCCEEDED update is deliberately refused.
	_, err = db.ExecContext(ctx, `CREATE TRIGGER s2_test_last_receipt_block
		BEFORE UPDATE OF state ON operations
		WHEN OLD.action='box_create' AND OLD.state='VERIFYING'
			AND NEW.state='SUCCEEDED'
		BEGIN SELECT RAISE(FAIL, 'S2_TEST_FINAL_RECEIPT_ABORT'); END`)
	ensure(err == nil, "test-only final receipt trigger not installed")
	client := mcp.NewClient(&mcp.Implementation{Name: "s2-bound-original-native-prober", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: "http://127.0.0.1:18663/mcp",
		HTTPClient: &http.Client{Transport: bearer{token}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	ensure(err == nil && session != nil, "original admin client connection refused")
	defer session.Close()
	name := "bw-s2-p1-bound-final-110u"
	args := map[string]any{"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": false, "internet": false, "idempotency_key": "s2-bound-final-original-create"}
	created, err := invoke(ctx, session, "box_create", args)
	ensure(err == nil, "real native Create admission refused")
	originalID := stringField(created, "operation_id")
	ensure(len(originalID) == 32, "original create ID missing")
	var pending map[string]any
	for i := 0; i < 90; i++ {
		result, e := invoke(ctx, session, "operation_get", map[string]any{
			"operation_id": originalID, "wait_ms": 1500,
		})
		ensure(e == nil, "cannot query original while verifying")
		pending, _ = result["operation"].(map[string]any)
		ensure(pending != nil, "missing immutable original receipt")
		if state := stringField(pending, "state"); state == "UNKNOWN" {
			break
		} else if state == "FAILED" || state == "SUCCEEDED" || state == "CANCELLED" {
			panic("S2_P1_BOUND_CREATE_FAULT_FAIL: original incorrectly terminal: "+state)
		}
	}
	uuid, nativeID := stringField(pending, "box_uuid"), stringField(pending, "native_operation_id")
	ensure(stringField(pending, "state") == "UNKNOWN" && len(uuid) == 36 && nativeID != "",
		"native Create binding and retained original UNKNOWN not durable")
	next, _ := pending["next_action"].(map[string]any)
	ensure(next != nil && stringField(next, "kind") == "HOST_LOCAL_REVIEW",
		"failed final receipt improperly advertised automatic retry")
	var bound, evidence, unexpected int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM owned_boxes
		WHERE uuid=? AND project=? AND name=? AND state='OWNED'`, uuid, project, name).Scan(&bound)
	ensure(err == nil && bound == 1, "native UUID not atomically bound to original")
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events
		WHERE operation_id=? AND state='VERIFYING' AND safe_detail='NATIVE_UUID_CONFIRMED'`, originalID).Scan(&evidence)
	ensure(err == nil && evidence == 1, "original UUID bind audit event missing")

	// Same-key replay is the *original UNKNOWN* and never a second Create.
	replay, err := invoke(ctx, session, "box_create", args)
	ensure(err == nil && stringField(replay, "operation_id") == originalID &&
		stringField(replay, "state") == "UNKNOWN",
		"same original key redispatched or changed receipt")
	otherCreate := map[string]any{}
	for k, v := range args { otherCreate[k] = v }
	otherCreate["idempotency_key"] = "s2-bound-final-different-create"
	_, err = invoke(ctx, session, "box_create", otherCreate)
	ensure(err != nil, "different-key Create bypassed bound original lock")
	_, err = invoke(ctx, session, "box_delete", map[string]any{
		"box_id": uuid, "idempotency_key": "s2-bound-final-different-delete",
	})
	ensure(err != nil, "different-key Delete raced past bound UNKNOWN original")
	box, err := invoke(ctx, session, "box_get", map[string]any{"box_id": uuid})
	ensure(err == nil, "known UUID became unreadable during refused Delete")
	observed, _ := box["box"].(map[string]any)
	ensure(observed != nil && stringField(observed, "box_id") == uuid &&
		stringField(observed, "native_state") == "Stopped",
		"foreign, missing or modified native Box after refused Delete")
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE action='box_delete'
		OR (action='box_create' AND id!=?)`, originalID).Scan(&unexpected)
	ensure(err == nil && unexpected == 0, "rejected mutation inserted another original")

	data, err := json.Marshal(map[string]string{"box_id": uuid, "original_id": originalID,
		"native_id": nativeID, "name": name})
	ensure(err == nil && os.WriteFile(filepath.Join(dir, "original.json"), data, 0600) == nil,
		"test-only original native witness not written")
	fmt.Println("S2_P1_REAL_NATIVE_CREATE_BOUND_UUID_FINAL_RECEIPT_UNKNOWN_NO_REDISPATCH=PASS")
	fmt.Println("S2_P1_REAL_BOUND_UNKNOWN_DIFFERENT_KEY_CREATE_DELETE_REFUSED=PASS")
}
