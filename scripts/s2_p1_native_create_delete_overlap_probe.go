//go:build ignore

// Official MCP SDK native Create/Delete overlap: a test-only source overlay
// pauses after native verification, before atomic UUID/terminal finalization.
// No bound-UUID VERIFYING window is fabricated. This file never ships.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

func must(b bool, msg string) {
	if !b {
		panic("S2_P1_NATIVE_OVERLAP_FAIL: " + msg)
	}
}

type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(cp)
}
func session(ctx context.Context, token string) *mcp.ClientSession {
	c := mcp.NewClient(&mcp.Implementation{Name: "s2-p1-verified-before-delete", Version: "1"}, nil)
	s, e := c.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: "http://127.0.0.1:18668/mcp",
		HTTPClient: &http.Client{Transport: bearer{token}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	must(e == nil && s != nil, "official MCP connection failed")
	return s
}
func call(ctx context.Context, s *mcp.ClientSession, n string, args map[string]any) (map[string]any, bool) {
	r, e := s.CallTool(ctx, &mcp.CallToolParams{Name: n, Arguments: args})
	must(e == nil && r != nil, "transport failed or stalled (cannot count as native lock refusal) on "+n)
	if r.IsError {
		// An independently known native UUID is not locally owned before
		// the atomic claim+terminal commit. Require this exact refusal;
		// transport/internal failures cannot count as ownership safety.
		// Match the public redacted ToolFailure, not the private store's
		// ErrNotFound string. The independent ledger check below still
		// requires zero Delete admissions and the unchanged original.
		must(len(r.Content) == 1, "unexpected refusal content count on "+n)
		part, ok := r.Content[0].(*mcp.TextContent)
		var refusal struct {
			Code      string `json:"code"`
			Category  string `json:"category"`
			Admission string `json:"admission"`
		}
		must(ok && json.Unmarshal([]byte(part.Text), &refusal) == nil,
			"missing typed ownership refusal on "+n)
		must(refusal.Code == "BOX_NOT_FOUND_OR_NOT_OWNED" &&
			refusal.Category == "AUTHORIZATION_OR_ABSENCE" &&
			refusal.Admission == "UNVERIFIED",
			"unexpected typed refusal instead of unbound-UUID refusal on "+n)
		return nil, false
	}
	b, e := json.Marshal(r.StructuredContent)
	must(e == nil, "structured response invalid")
	var v map[string]any
	must(json.Unmarshal(b, &v) == nil && v != nil, "MCP structured response missing")
	return v, true
}
func field(v map[string]any, k string) string         { x, _ := v[k].(string); return x }
func child(v map[string]any, k string) map[string]any { x, _ := v[k].(map[string]any); return x }
func present(path string) bool                        { _, e := os.Stat(path); return e == nil }
func newMarker(path, value string) {
	f, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	must(e == nil, "could not write original-only marker")
	_, e = f.WriteString(value)
	must(e == nil, "marker not writable")
	must(f.Sync() == nil && f.Close() == nil, "marker durable sync failure")
}
func awaitMarker(ctx context.Context, path string) {
	for i := 0; i < 900; i++ {
		if present(path) {
			return
		}
		select {
		case <-ctx.Done():
			panic("S2_P1_NATIVE_OVERLAP_FAIL: marker canceled " + path)
		case <-time.After(100 * time.Millisecond):
		}
	}
	panic("S2_P1_NATIVE_OVERLAP_FAIL: timeout waiting for " + path)
}
func finish(ctx context.Context, s *mcp.ClientSession, id string) map[string]any {
	for i := 0; i < 85; i++ {
		r, ok := call(ctx, s, "operation_get", map[string]any{"operation_id": id, "wait_ms": 750})
		must(ok, "final original receipt denied")
		op := child(r, "operation")
		must(op != nil, "original receipt missing")
		state := field(op, "state")
		if state == "SUCCEEDED" {
			return op
		}
		must(state != "FAILED" && state != "UNKNOWN" && state != "CANCELLED", "original failed after overlay release: "+state)
	}
	panic("S2_P1_NATIVE_OVERLAP_FAIL: verification did not finish")
}
func main() {
	dir := os.Getenv("BW_S2_OVERLAP_LAB")
	token := os.Getenv("BW_S2_OVERLAP_TOKEN")
	other := os.Getenv("BW_S2_OVERLAP_SECOND_TOKEN")
	project := os.Getenv("BW_S2_OVERLAP_PROJECT")
	must(os.Geteuid() != 0 && os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		filepath.Base(dir) == "p1-native-overlap" && strings.HasPrefix(dir, "/tmp/bw-v1-s2-ghci.") &&
		project == "user-"+os.Getenv("BW_S2_TEST_UID") &&
		strings.HasPrefix(token, "bw1_") && strings.HasPrefix(other, "bw1_"),
		"requires fresh restricted original-only native CI")
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	mainSession := session(ctx, token)
	defer mainSession.Close()
	name := "bw-s2-p1-overlap-110x"
	createArgs := map[string]any{
		"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": false, "internet": false,
		"idempotency_key": "s2-native-overlap-original-create",
	}
	scoped := os.Getenv("BW_S2_OVERLAP_SCOPED") == "1"
	path := filepath.Join(dir, "ledger.sqlite")
	if scoped {
		must(project == fmt.Sprintf("user-%d", os.Geteuid()), "exact restricted UID mismatch")
		path = "/var/lib/boxweave/ledger.sqlite"
	}
	db, e := sql.Open("sqlite3", "file:"+path+"?mode=ro&_query_only=1&_busy_timeout=5000")
	if !scoped {
		db.Close()
		db, e = sql.Open("sqlite3", path)
	}
	must(e == nil, "cannot inspect test ledger")
	defer db.Close()
	var priorDeletes int
	e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations WHERE action='box_delete'").Scan(&priorDeletes)
	must(e == nil, "read baseline delete count")
	created, ok := call(ctx, mainSession, "box_create", createArgs)
	must(ok && len(field(created, "operation_id")) == 32, "native original Create admission failed")
	id := field(created, "operation_id")
	awaitMarker(ctx, filepath.Join(dir, "held"))
	var st *state.Store
	if scoped {
		st, e = state.OpenNetworkAuditReadOnly(path)
	} else {
		st, e = state.Open(path)
	}
	must(e == nil, "cannot inspect test-only pre-atomic SQLite")
	defer st.Close()
	original, e := st.Get(ctx, id)
	must(e == nil && original.State == "VERIFYING" && original.BoxUUID == "" &&
		original.Incarnation == "" && original.NativeOperationID != "",
		"pre-atomic Create leaked UUID or lost original native receipt")
	heldBytes, e := os.ReadFile(filepath.Join(dir, "held"))
	must(e == nil, "pre-atomic native witness missing")
	var held map[string]string
	must(json.Unmarshal(heldBytes, &held) == nil && held["phase"] == "NATIVE_VERIFIED_BEFORE_ATOMIC_CREATE" &&
		held["original_id"] == id && held["native_id"] == original.NativeOperationID &&
		held["project"] == project && len(held["uuid"]) == 36, "original native witness identity mismatch")
	uuid := held["uuid"]
	_, e = st.OwnedBox(ctx, project, uuid)
	must(errors.Is(e, state.ErrNotFound), "uncommitted native UUID became locally owned")
	evidence, _ := json.Marshal(map[string]string{"uuid": uuid, "original_id": id, "native_id": original.NativeOperationID})
	newMarker(filepath.Join(dir, "bound.json"), string(evidence))
	awaitMarker(ctx, filepath.Join(dir, "native-verified"))
	// The source overlay is still holding only Create's worker goroutine,
	// so these real SDK calls must be genuinely handled by other goroutines.
	const count = 12
	var wg sync.WaitGroup
	errs := make(chan string, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			pctx, stop := context.WithTimeout(ctx, 9*time.Second)
			defer stop()
			tk := token
			if i%2 == 1 {
				tk = other
			}
			s := session(pctx, tk)
			defer s.Close()
			_, success := call(pctx, s, "box_delete", map[string]any{
				"box_id": uuid, "idempotency_key": fmt.Sprintf("s2-overlap-delete-%02d", i),
			})
			if success {
				errs <- fmt.Sprintf("delete-%d wrongly admitted", i)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for problem := range errs {
		panic("S2_P1_NATIVE_OVERLAP_FAIL: " + problem)
	}

	var deleteCount int
	e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations WHERE action='box_delete'").Scan(&deleteCount)
	must(e == nil && deleteCount == priorDeletes, "rejected concurrent Delete inserted durable work")
	verify, e := st.Get(ctx, id)
	must(e == nil && verify.State == "VERIFYING" && verify.BoxUUID == "" && verify.Incarnation == "",
		"real concurrent Delete changed still-held original")
	fmt.Println("S4_A_REAL_NATIVE_PREATOMIC_TWELVE_SDK_DELETE_REFUSED=PASS")
	if scoped {
		newMarker(filepath.Join(dir, "denied"), string(evidence))
		awaitMarker(ctx, filepath.Join(dir, "denials-verified"))
	}
	// The test-only Go overlay pauses only box_create; subsequent verified
	// box_delete does not need any GDB-style disarm marker.
	newMarker(filepath.Join(dir, "release"), "safe to complete native original\n")
	final := finish(ctx, mainSession, id)
	must(field(final, "box_uuid") == uuid, "final Create changed original native UUID")
	owned, e := st.OwnedBox(ctx, project, uuid)
	must(e == nil && owned.UUID == uuid && owned.Name == name && owned.Incarnation == uuid,
		"successful Create omitted atomic owned UUID")
	var createEvents int
	e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED' AND safe_detail='ORIGINAL_CREATE_UUID_ATOMIC'", id).Scan(&createEvents)
	must(e == nil && createEvents == 1, "successful Create omitted or duplicated atomic terminal event")
	if scoped {
		newMarker(filepath.Join(dir, "created"), string(evidence))
		awaitMarker(ctx, filepath.Join(dir, "create-verified"))
	}
	// Only after the original is VERIFIED may a fresh new-key Delete be admitted.
	removal, ok := call(ctx, mainSession, "box_delete", map[string]any{
		"box_id": uuid, "idempotency_key": "s2-overlap-delete-after-verified"})
	must(ok && len(field(removal, "operation_id")) == 32, "verified original blocked legitimate Delete")
	deleteID := field(removal, "operation_id")
	deleted := finish(ctx, mainSession, deleteID)
	if scoped {
		witness, err := json.Marshal(map[string]string{"create_id": id, "create_tracking": field(final, "tracking_id"), "create_native": field(final, "native_operation_id"), "delete_id": deleteID, "delete_tracking": field(deleted, "tracking_id"), "delete_native": field(deleted, "native_operation_id"), "uuid": uuid})
		must(err == nil, "cannot marshal final identity")
		newMarker(filepath.Join(dir, "completed.json"), string(witness))
	}
	var tombstone, deletedBy string
	var deletedAt int64
	e = db.QueryRowContext(ctx, "SELECT state,deleted_by_operation_id,deleted_at FROM owned_boxes WHERE uuid=?", uuid).Scan(&tombstone, &deletedBy, &deletedAt)
	must(e == nil && tombstone == "DELETED" && deletedBy == deleteID && deletedAt > 0,
		"successful Delete omitted original-operation tombstone")
	fmt.Printf("S4_A_OVERLAP_CREATE_BW_ID=%s NATIVE_ID=%s DELETE_BW_ID=%s BOX_UUID=%s PROJECT=%s\n", id, original.NativeOperationID, deleteID, uuid, project)
	fmt.Println("S4_A_REAL_NATIVE_OVERLAP_ATOMIC_CREATE_DELETE_TOMBSTONE=PASS")
}
