//go:build ignore

// Test-only SQLite atomic-finalization failure after an ACTUAL native Create.
// UUID/claim/terminal must all roll back. The trigger exists only in a test
// DB; it is never installed by the product or on shared/retained databases.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type bearer struct{ token string }

func (t bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(cp)
}
func ensure(ok bool, what string) {
	if !ok {
		panic("S2_P1_BOUND_CREATE_FAULT_FAIL: " + what)
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

func refused(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any, expected error) {
	r, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	ensure(err == nil && r != nil && r.IsError, "negative request was not a structured MCP refusal")
	content, err := json.Marshal(r.Content)
	ensure(err == nil && strings.Contains(string(content), expected.Error()), "negative request had unexpected refusal class")
}

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
	// service startup. Native Create can ACK; the combined UUID ownership +
	// SUCCEEDED transaction must roll back together when its update is refused.
	_, err = db.ExecContext(ctx, `CREATE TRIGGER s2_test_last_receipt_block
		BEFORE UPDATE OF state ON operations
		WHEN OLD.action='box_create' AND OLD.state='VERIFYING'
			AND NEW.state='SUCCEEDED'
		BEGIN SELECT RAISE(FAIL, 'S2_TEST_FINAL_RECEIPT_ABORT'); END`)
	ensure(err == nil, "test-only final receipt trigger not installed")
	client := mcp.NewClient(&mcp.Implementation{Name: "s2-bound-original-native-prober", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             "http://127.0.0.1:18663/mcp",
		HTTPClient:           &http.Client{Transport: bearer{token}},
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
			panic("S2_P1_BOUND_CREATE_FAULT_FAIL: original incorrectly terminal: " + state)
		}
	}
	nativeID := stringField(pending, "native_operation_id")
	ensure(stringField(pending, "state") == "UNKNOWN" && stringField(pending, "box_uuid") == "" &&
		stringField(pending, "incarnation") == "" && nativeID != "",
		"atomic Create rollback leaked UUID or lost original UNKNOWN/native ID")
	reader := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	observed, err := reader.ObserveCreatedByOperation(ctx, name, originalID)
	ensure(err == nil && len(observed.UUID) == 36 && observed.Name == name &&
		observed.Project == project && observed.NativeState == "Stopped",
		"independent native original marker/UUID/Project/Stopped state missing")
	uuid := observed.UUID
	next, _ := pending["next_action"].(map[string]any)
	ensure(next != nil && stringField(next, "kind") == "HOST_LOCAL_REVIEW",
		"failed final receipt improperly advertised automatic retry")
	var bound, evidence, unexpected int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM owned_boxes
		WHERE uuid=? OR (project=? AND name=?)`, uuid, project, name).Scan(&bound)
	ensure(err == nil && bound == 0, "failed atomic Create leaked ownership claim")
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events
		WHERE operation_id=? AND (state='SUCCEEDED' OR safe_detail='NATIVE_UUID_CONFIRMED')`, originalID).Scan(&evidence)
	ensure(err == nil && evidence == 0, "failed atomic Create leaked terminal or split UUID event")

	// Same-key replay is the *original UNKNOWN* and never a second Create.
	replay, err := invoke(ctx, session, "box_create", args)
	ensure(err == nil && stringField(replay, "operation_id") == originalID &&
		stringField(replay, "state") == "UNKNOWN" && replay["duplicate"] == true,
		"same original key redispatched or changed receipt")
	otherCreate := map[string]any{}
	for k, v := range args {
		otherCreate[k] = v
	}
	otherCreate["idempotency_key"] = "s2-bound-final-different-create"
	refused(ctx, session, "box_create", otherCreate, state.ErrResourceLocked)
	refused(ctx, session, "box_delete", map[string]any{
		"box_id": uuid, "idempotency_key": "s2-bound-final-different-delete",
	}, state.ErrNotFound)
	refused(ctx, session, "box_get", map[string]any{"box_id": uuid}, state.ErrNotFound)
	after, err := reader.ObserveCreatedByOperation(ctx, name, originalID)
	ensure(err == nil && after.UUID == uuid && after.NativeState == "Stopped",
		"original native UUID changed after refused Delete")
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE action='box_delete'
		OR (action='box_create' AND id!=?)`, originalID).Scan(&unexpected)
	ensure(err == nil && unexpected == 0, "rejected mutation inserted another original")

	// Exercise explicit reviewed identity binding ONLY on a consistent backup.
	// This is not a product retry, adoption, state resolution or lock release.
	copyDir := filepath.Join(dir, "atomic-bind-copy")
	ensure(os.Mkdir(copyDir, 0700) == nil, "isolated identity-review copy directory unavailable")
	copyDB := filepath.Join(copyDir, "ledger.sqlite")
	backup := exec.CommandContext(ctx, "python3", "-c", `import os,sqlite3,sys
with sqlite3.connect("file:"+sys.argv[1]+"?mode=ro",uri=True) as src:
    with sqlite3.connect(sys.argv[2]) as dst:
        src.backup(dst)
        dst.execute("DROP TRIGGER s2_test_last_receipt_block")
os.chmod(sys.argv[2],0o600)`, filepath.Join(dir, "ledger.sqlite"), copyDB)
	ensure(backup.Run() == nil, "consistent standalone SQLite backup unavailable")
	copyStore, err := state.Open(copyDB)
	ensure(err == nil, "isolated identity-review backup refused")
	defer copyStore.Close()
	snap, err := copyStore.CreateRecoveryInspect(ctx, originalID)
	ensure(err == nil && snap.NativeOperationID == nativeID && snap.Project == project && snap.Name == name,
		"copy lost original unbound UNKNOWN identity")
	readback, err := reader.ObserveOwnedBox(ctx, name, uuid)
	ensure(err == nil && readback.State == "PRESENT_UUID_CONFIRMED", "independent native readback unavailable")
	challenge, err := copyStore.IssueCreateBindingChallenge(ctx, snap, "ci-atomic-create-verifier", readback.Fingerprint)
	ensure(err == nil, "copy-only review challenge refused")
	err = copyStore.BindUnknownCreatedBox(ctx, snap, "ci-atomic-create-verifier", challenge, uuid,
		"forged-observation", "independent original native identity verified", true)
	ensure(err != nil, "copy-only binding accepted forged observation")
	err = copyStore.BindUnknownCreatedBox(ctx, snap, "ci-atomic-create-verifier", challenge, uuid,
		readback.Fingerprint, "independent original native identity verified", true)
	ensure(err == nil, "copy-only exact identity binding refused")
	copyOriginal, err := copyStore.Get(ctx, originalID)
	ensure(err == nil && copyOriginal.State == "UNKNOWN" && copyOriginal.BoxUUID == uuid &&
		copyOriginal.NativeOperationID == nativeID && copyOriginal.NextAction.Kind == "HOST_LOCAL_REVIEW",
		"copy-only binding resolved UNKNOWN or lost immutable native identity")
	err = copyStore.BindUnknownCreatedBox(ctx, snap, "ci-atomic-create-verifier", challenge, uuid,
		readback.Fingerprint, "replayed binding challenge must be refused", true)
	ensure(errors.Is(err, state.ErrConflict), "copy-only original binding challenge replay accepted")
	var originalUUID, originalState string
	err = db.QueryRowContext(ctx, "SELECT box_uuid,state FROM operations WHERE id=?", originalID).Scan(&originalUUID, &originalState)
	ensure(err == nil && originalUUID == "" && originalState == "UNKNOWN", "copy-only review changed original journal")
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM owned_boxes").Scan(&bound)
	ensure(err == nil && bound == 0, "copy-only review changed original ownership")
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_audit WHERE operation_id=?", originalID).Scan(&evidence)
	ensure(err == nil && evidence == 0, "copy-only review changed original recovery audit")

	data, err := json.Marshal(map[string]string{"box_id": uuid, "original_id": originalID,
		"native_id": nativeID, "name": name})
	ensure(err == nil && os.WriteFile(filepath.Join(dir, "original.json"), data, 0600) == nil,
		"test-only original native witness not written")
	fmt.Printf("S4_A_ATOMIC_ROLLBACK_CREATE_BW_ID=%s NATIVE_ID=%s BOX_UUID=%s PROJECT=%s\n", originalID, nativeID, uuid, project)
	fmt.Println("S4_A_REAL_NATIVE_CREATE_ATOMIC_ROLLBACK_UNBOUND_UNKNOWN_NO_REDISPATCH=PASS")
	fmt.Println("S4_A_REAL_UNBOUND_UNKNOWN_DIFFERENT_KEY_CREATE_DELETE_REFUSED=PASS")
	fmt.Println("S4_A_REAL_NATIVE_CREATE_ATOMIC_ROLLBACK_COPY_ONLY_BIND_ORIGINAL_UNCHANGED=PASS")
}
