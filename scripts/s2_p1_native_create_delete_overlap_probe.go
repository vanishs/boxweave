//go:build ignore

// Official MCP SDK native Create/Delete overlap: a separate debugger suspends
// ONLY the Create worker at its final SQLite Verified call, after its actual
// native UUID has been bound. This file never ships in the product.
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
		// An HTTP/MCP refusal is not enough to establish a safe lock:
		// distinguish the expected original-resource lock from generic
		// internal/server errors and invalid request failures.
		encoded, encodeErr := json.Marshal(r.Content)
		must(encodeErr == nil && strings.Contains(string(encoded),
			"original unresolved resource claim is still locked"),
			"unexpected error instead of original bound-UUID resource lock on "+n)
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
		must(state != "FAILED" && state != "UNKNOWN" && state != "CANCELLED", "original Create failed after debugger release: "+state)
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
	created, ok := call(ctx, mainSession, "box_create", createArgs)
	must(ok && len(field(created, "operation_id")) == 32, "native original Create admission failed")
	id := field(created, "operation_id")
	awaitMarker(ctx, filepath.Join(dir, "held"))
	st, e := state.Open(filepath.Join(dir, "ledger.sqlite"))
	must(e == nil, "cannot inspect test-only SQLite after native bind")
	defer st.Close()
	original, e := st.Get(ctx, id)
	must(e == nil && original.State == "VERIFYING" && len(original.BoxUUID) == 36 &&
		original.NativeOperationID != "", "real CREATE did not bind exact UUID while VERIFYING")
	uuid := original.BoxUUID
	owned, e := st.OwnedBox(ctx, project, uuid)
	must(e == nil && owned.UUID == uuid && owned.Name == name, "original bound UUID not owned atomically")
	evidence, _ := json.Marshal(map[string]string{"uuid": uuid, "original_id": id, "native_id": original.NativeOperationID})
	newMarker(filepath.Join(dir, "bound.json"), string(evidence))
	awaitMarker(ctx, filepath.Join(dir, "native-verified"))
	// Source-line debugger is still holding only Create's worker goroutine,
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
	db, e := sql.Open("sqlite3", filepath.Join(dir, "ledger.sqlite"))
	must(e == nil, "cannot independently read exact SQLite")
	defer db.Close()
	var deleteCount int
	e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations WHERE action='box_delete'").Scan(&deleteCount)
	must(e == nil && deleteCount == 0, "rejected concurrent Delete inserted durable work")
	verify, e := st.Get(ctx, id)
	must(e == nil && verify.State == "VERIFYING" && verify.BoxUUID == uuid,
		"real concurrent Delete changed still-held original")
	fmt.Println("S2_P1_REAL_NATIVE_BOUND_VERIFYING_TWELVE_SDK_DELETE_REFUSED=PASS")
	// The test-only Go overlay pauses only box_create; subsequent verified
	// box_delete does not need any GDB-style disarm marker.
	newMarker(filepath.Join(dir, "release"), "safe to complete native original\n")
	final := finish(ctx, mainSession, id)
	must(field(final, "box_uuid") == uuid, "final Create changed original native UUID")
	// Only after the original is VERIFIED may a fresh new-key Delete be admitted.
	removal, ok := call(ctx, mainSession, "box_delete", map[string]any{
		"box_id": uuid, "idempotency_key": "s2-overlap-delete-after-verified"})
	must(ok && len(field(removal, "operation_id")) == 32, "verified original blocked legitimate Delete")
	finish(ctx, mainSession, field(removal, "operation_id"))
	fmt.Println("S2_P1_REAL_NATIVE_OVERLAP_RELEASE_VERIFIED_DELETE_SUCCEEDED=PASS")
}
