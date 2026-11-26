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
	"github.com/vanishs/boxweave/internal/recovery"
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

func refused(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any, code, category, admission, originalID string) {
	r, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	ensure(err == nil && r != nil && r.IsError && len(r.Content) == 1, "negative request was not a structured MCP refusal")
	part, ok := r.Content[0].(*mcp.TextContent)
	var failure struct {
		Code          string           `json:"code"`
		Category      string           `json:"category"`
		Admission     string           `json:"admission"`
		OriginalState string           `json:"original_state"`
		Next          state.NextAction `json:"next_action"`
	}
	ensure(ok && json.Unmarshal([]byte(part.Text), &failure) == nil, "negative request missing typed refusal")
	ensure(failure.Code == code && failure.Category == category && failure.Admission == admission,
		"negative request had unexpected refusal class: "+name+" "+failure.Code)
	if originalID != "" {
		ensure(failure.OriginalState == "UNKNOWN" && failure.Next.Tool == "operation_recover" &&
			failure.Next.Kind == "INSPECT_RECOVERY" && failure.Next.Mode == "inspect" &&
			failure.Next.OperationID == originalID, "refusal lost exact UNKNOWN recovery navigation")
	} else {
		ensure(failure.Next.OperationID == "" && failure.OriginalState == "" &&
			failure.Next.Kind == "CHECK_AUTHORIZED_SCOPE", "unowned refusal exposed original identity")
	}
}

// Verify the existing installed Scoped final-commit fault before the
// independent reviewed recovery. No trigger, native effect or DB write here.
func verifyScoped(dir, token, project string) {
	ensure(os.Geteuid() != 0 && project == fmt.Sprintf("user-%d", os.Geteuid()) &&
		os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		filepath.Dir(dir) == "/tmp" && strings.HasPrefix(filepath.Base(dir), "bw-n2-installed.") &&
		strings.HasPrefix(token, "bw1_"), "only exact installed one-use Scoped fixture")
	raw, err := os.ReadFile(filepath.Join(dir, "scoped-final-cut-original.json"))
	ensure(err == nil, "missing existing original checkpoint")
	var cp struct {
		Request    map[string]any `json:"request"`
		OriginalID string         `json:"original_id"`
		NativeID   string         `json:"native_id"`
		Tracking   string         `json:"tracking_id"`
		UUID       string         `json:"uuid"`
	}
	ensure(json.Unmarshal(raw, &cp) == nil && len(cp.OriginalID) == 32 && len(cp.Tracking) == 32 &&
		len(cp.NativeID) == 36 && len(cp.UUID) == 36 && cp.Request["name"] == "p1b-recovery-final-cut" &&
		cp.Request["work_ref"] == "p1b/scoped-create-final-cut" &&
		cp.Request["idempotency_key"] == "pr191-scoped-create-final-cut", "wrong existing Scoped fixture")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "s2-scoped-original-rollback", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: "http://127.0.0.1:18650/mcp",
		HTTPClient: &http.Client{Transport: bearer{token}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	ensure(err == nil && session != nil, "installed official SDK failed")
	defer session.Close()
	r, err := invoke(ctx, session, "operation_get", map[string]any{"operation_id": cp.OriginalID})
	op, _ := r["operation"].(map[string]any)
	ensure(err == nil && stringField(op, "state") == "UNKNOWN" && stringField(op, "box_uuid") == "" &&
		stringField(op, "incarnation") == "" && stringField(op, "native_operation_id") == cp.NativeID &&
		stringField(op, "tracking_id") == cp.Tracking, "existing original not exact unbound UNKNOWN")
	replay, err := invoke(ctx, session, "box_create", cp.Request)
	ensure(err == nil && stringField(replay, "operation_id") == cp.OriginalID &&
		stringField(replay, "native_operation_id") == cp.NativeID && stringField(replay, "tracking_id") == cp.Tracking &&
		stringField(replay, "state") == "UNKNOWN" && replay["duplicate"] == true, "replay changed original")
	next := map[string]any{}
	for k, v := range cp.Request {
		next[k] = v
	}
	next["idempotency_key"] = "pr191-scoped-final-cut-different-create"
	refused(ctx, session, "box_create", next, "UNKNOWN_BLOCKER", "EFFECT_UNVERIFIED", "NOT_ADMITTED", cp.OriginalID)
	refused(ctx, session, "box_delete", map[string]any{"box_id": cp.UUID, "idempotency_key": "pr191-scoped-final-cut-different-delete"},
		"BOX_NOT_FOUND_OR_NOT_OWNED", "AUTHORIZATION_OR_ABSENCE", "UNVERIFIED", "")
	refused(ctx, session, "box_get", map[string]any{"box_id": cp.UUID},
		"BOX_NOT_FOUND_OR_NOT_OWNED", "AUTHORIZATION_OR_ABSENCE", "UNVERIFIED", "")
	fmt.Println("P1B_SCOPED_CREATE_ROLLBACK_SAME_KEY_AND_TYPED_NEW_KEY_REFUSALS=PASS")
}

func main() {
	if os.Getenv("BW_S2_BOUND_SCOPED") == "1" {
		verifyScoped(os.Getenv("BW_S2_BOUND_LAB"), os.Getenv("BW_S2_BOUND_TOKEN"), os.Getenv("BW_S2_BOUND_PROJECT"))
		return
	}
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
	refused(ctx, session, "box_create", otherCreate, "UNKNOWN_BLOCKER", "EFFECT_UNVERIFIED", "NOT_ADMITTED", originalID)
	refused(ctx, session, "box_delete", map[string]any{
		"box_id": uuid, "idempotency_key": "s2-bound-final-different-delete",
	}, "BOX_NOT_FOUND_OR_NOT_OWNED", "AUTHORIZATION_OR_ABSENCE", "UNVERIFIED", "")
	refused(ctx, session, "box_get", map[string]any{"box_id": uuid}, "BOX_NOT_FOUND_OR_NOT_OWNED", "AUTHORIZATION_OR_ABSENCE", "UNVERIFIED", "")
	after, err := reader.ObserveCreatedByOperation(ctx, name, originalID)
	ensure(err == nil && after.UUID == uuid && after.NativeState == "Stopped",
		"original native UUID changed after refused Delete")
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE action='box_delete'
		OR (action='box_create' AND id!=?)`, originalID).Scan(&unexpected)
	ensure(err == nil && unexpected == 0, "rejected mutation inserted another original")

	// Exercise the real N3 staged Create recovery on an isolated backup.
	// This is not a product retry, native adoption or mutation of the live
	// UNKNOWN ledger. The test-only receipt trigger is removed ONLY in copy.
	copyDir := filepath.Join(dir, "atomic-bind-copy")
	ensure(os.Mkdir(copyDir, 0700) == nil, "isolated recovery copy directory unavailable")
	copyDB := filepath.Join(copyDir, "ledger.sqlite")
	backup := exec.CommandContext(ctx, "python3", "-c", `import os,sqlite3,sys
with sqlite3.connect("file:"+sys.argv[1]+"?mode=ro",uri=True) as src:
    with sqlite3.connect(sys.argv[2]) as dst:
        src.backup(dst)
        dst.execute("DROP TRIGGER s2_test_last_receipt_block")
os.chmod(sys.argv[2],0o600)`, filepath.Join(dir, "ledger.sqlite"), copyDB)
	ensure(backup.Run() == nil, "consistent standalone SQLite backup unavailable")
	copyStore, err := state.Open(copyDB)
	ensure(err == nil, "isolated staged recovery backup refused")
	defer copyStore.Close()
	snap, err := copyStore.CreateRecoveryInspect(ctx, originalID)
	ensure(err == nil && snap.NativeOperationID == nativeID && snap.Project == project && snap.Name == name,
		"copy lost original unbound UNKNOWN identity")
	readback, err := reader.ObserveOwnedBox(ctx, name, uuid)
	ensure(err == nil && readback.State == "PRESENT_UUID_CONFIRMED" && readback.Fingerprint != "",
		"independent native readback unavailable")
	challenge, err := copyStore.IssueCreateBindingChallenge(ctx, snap, "ci-atomic-create-verifier", readback.Fingerprint)
	ensure(err == nil, "copy-only review challenge refused")
	err = copyStore.BindUnknownCreatedBox(ctx, snap, "ci-atomic-create-verifier", challenge, uuid,
		"forged-observation", "independent original native identity verified", true)
	ensure(err != nil, "copy-only binding accepted forged observation")
	err = copyStore.BindUnknownCreatedBox(ctx, snap, "ci-atomic-create-verifier", challenge, uuid,
		readback.Fingerprint, "independent original native identity verified", true)
	ensure(errors.Is(err, state.ErrResourceLocked), "modern pinned Create accepted obsolete UUID-only binding")
	copyOriginal, err := copyStore.Get(ctx, originalID)
	ensure(err == nil && copyOriginal.State == "UNKNOWN" && copyOriginal.BoxUUID == "" &&
		copyOriginal.NativeOperationID == nativeID && copyOriginal.NextAction.Kind == "HOST_LOCAL_REVIEW",
		"legacy UUID-only refusal changed original UNKNOWN or its native ACK")

	var pinClaims int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_resource_claims
		WHERE operation_id=? AND resource_key LIKE 'create-pin:%'`, originalID).Scan(&pinClaims)
	ensure(err == nil && pinClaims == 4, "original staged Create lost a prospective address claim")
	svc := recovery.Service{Store: copyStore, Native: reader, Layer: project}
	diagnosis, err := svc.Inspect(ctx, originalID, "")
	ensure(err == nil && diagnosis.View.Disposition == "SAFE_RECOVER" &&
		diagnosis.View.ProposedAction == state.RecoverySettle &&
		diagnosis.View.Reason == "CREATE_ORIGINAL_POSTSTATE_VERIFIED" &&
		diagnosis.View.ObservedNativeUUID == uuid &&
		diagnosis.View.ClaimProtection == "RETAINED",
		"original native terminal, static lease/marker or staged address pin not verified")
	var actor, tokenID string
	err = db.QueryRowContext(ctx, "SELECT actor_id,token_id FROM operations WHERE id=?", originalID).Scan(&actor, &tokenID)
	ensure(err == nil && actor == tokenID, "original authenticated owner not recorded")
	request := state.RecoveryRequest{
		Layer: project, ActorID: actor, TokenID: tokenID,
		OriginalID: originalID, Key: "s2-native-final-rollback-copy-reconcile-001",
		WorkRef: "ci/staged-create-fault", AgentRef: "ci-atomic-create-verifier",
	}
	settled, err := svc.Reconcile(ctx, request)
	ensure(err == nil && settled.Recovery != nil && settled.Recovery.Disposition == "SETTLED" &&
		settled.Recovery.Receipt.State == "SUCCEEDED" && !settled.NativeRetried,
		"staged Create finalization rollback could not be recovered atomically on copy")
	copyOriginal, err = copyStore.Get(ctx, originalID)
	ensure(err == nil && copyOriginal.State == "SUCCEEDED" && copyOriginal.BoxUUID == uuid &&
		copyOriginal.NativeOperationID == nativeID, "copied original native ACK, UUID or success state was altered")
	copySQL, err := sql.Open("sqlite3", "file:"+copyDB+"?mode=ro&_foreign_keys=on")
	ensure(err == nil, "read-only recovered copy unavailable")
	defer copySQL.Close()
	var copyOwned, binding, fences int
	err = copySQL.QueryRowContext(ctx, "SELECT COUNT(*) FROM owned_boxes WHERE uuid=? AND project=? AND state='OWNED'", uuid, project).Scan(&copyOwned)
	ensure(err == nil && copyOwned == 1, "copy recovery missed original exact ownership")
	err = copySQL.QueryRowContext(ctx, "SELECT COUNT(*) FROM network_bindings WHERE project=? AND box_uuid=? AND state='PINNED'", project, uuid).Scan(&binding)
	ensure(err == nil && binding == 1, "copy recovery missed original static network binding")
	err = copySQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_address_fences
		WHERE project=? AND box_uuid=? AND operation_id=? AND state='RESERVED'`, project, uuid, originalID).Scan(&fences)
	ensure(err == nil && fences == 2, "copy recovery missed an original dual-stack fence")
	repeated, err := svc.Reconcile(ctx, request)
	ensure(err == nil && repeated.Recovery != nil && repeated.Recovery.Receipt.Duplicate &&
		repeated.Recovery.Receipt.ID == settled.Recovery.Receipt.ID && !repeated.NativeRetried,
		"copy recovery replay issued another original native mutation")
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
	fmt.Println("S4_A_REAL_NATIVE_CREATE_ATOMIC_ROLLBACK_COPY_ONLY_STAGED_RECOVERY_ORIGINAL_UNCHANGED=PASS")
}
