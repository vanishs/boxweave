//go:build ignore

// Official authenticated Go MCP SDK black-box of a real Incus Update and a
// simultaneous SQLite WAL COMMIT crash. Never performs native cleanup.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type saved struct {
	CreateID   string `json:"create_id"`
	BoxID      string `json:"box_id"`
	UpdateID   string `json:"update_id"`
	TrackingID string `json:"tracking_id"`
}
type transport struct{ token string }

func (t transport) RoundTrip(r *http.Request) (*http.Response, error) {
	c := r.Clone(r.Context())
	c.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(c)
}
func must(ok bool, message string) {
	if !ok {
		panic("S2_P0_NATIVE_SQLITE_COMBINED_FAIL: " + message)
	}
}
func get(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}
func child(m map[string]any, key string) map[string]any {
	s, _ := m[key].(map[string]any)
	return s
}
func session(ctx context.Context, token string) *mcp.ClientSession {
	c := mcp.NewClient(&mcp.Implementation{Name: "s2-p0-combined-witness", Version: "1"}, nil)
	s, err := c.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: "http://127.0.0.1:18657/mcp",
		HTTPClient: &http.Client{Transport: transport{token}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		panic(err)
	}
	return s
}
func call(ctx context.Context, s *mcp.ClientSession, tool string, args map[string]any) (map[string]any, error) {
	reply, err := s.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil || reply == nil || reply.IsError {
		return nil, fmt.Errorf("MCP tool %s did not return successful typed receipt: %v", tool, err)
	}
	b, err := json.Marshal(reply.StructuredContent)
	if err != nil {
		return nil, err
	}
	var v map[string]any
	err = json.Unmarshal(b, &v)
	return v, err
}
func read(path string) saved {
	var s saved
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		panic(err)
	}
	return s
}
func write(path string, s saved) {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		panic(err)
	}
}
func updateArgs(uuid, phase string) map[string]any {
	return map[string]any{
		"box_id": uuid, "cpu": 2,
		"idempotency_key": "s2-p0-combined-native-wal-"+phase+"-original",
		"work_ref": "lab/p0-native-sqlite", "agent_ref": "p0-combined-verifier",
	}
}
func main() {
	must(len(os.Args) == 2 && (os.Args[1] == "seed" || os.Args[1] == "apply" || os.Args[1] == "resume"),
		"exact seed/apply/resume mode required")
	mode := os.Args[1]
	dir := os.Getenv("BW_S2_P0_DIR")
	phase := filepath.Base(dir)
	must((phase == "p0-before" || phase == "p0-after") &&
		strings.HasPrefix(dir, "/tmp/bw-v1-s2-ghci.") &&
		os.Getenv("GITHUB_ACTIONS") == "true" &&
		os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		os.Geteuid() != 0 && strings.HasPrefix(os.Getenv("BW_S2_P0_TOKEN"), "bw1_") &&
		strings.HasPrefix(os.Getenv("BW_S2_P0_PROJECT"), "user-"),
		"disposable nonroot GitHub native + SQLite fixture required")
	phase = strings.TrimPrefix(phase, "p0-")
	name := "bw-s2-p0-before-110j"
	if phase == "after" {
		name = "bw-s2-p0-after-110k"
	}
	path := filepath.Join(dir, "original.json")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	token := os.Getenv("BW_S2_P0_TOKEN")
	s := session(ctx, token)
	defer s.Close()
	switch mode {
	case "seed":
		created, err := call(ctx, s, "box_create", map[string]any{
			"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
			"start": false, "internet": true,
			"idempotency_key": "s2-p0-combined-seed-"+phase+"-once",
		})
		must(err == nil && get(created, "operation_id") != "", "durable fixture Create not admitted")
		record := saved{CreateID: get(created, "operation_id")}
		for i := 0; i < 120; i++ {
			x, e := call(ctx, s, "operation_get", map[string]any{"operation_id": record.CreateID, "wait_ms": 400})
			must(e == nil, "fixture Create not queryable")
			op := child(x, "operation")
			if get(op, "state") == "SUCCEEDED" {
				record.BoxID = get(op, "box_uuid")
				break
			}
			must(get(op, "state") != "UNKNOWN" && get(op, "state") != "FAILED" &&
				get(op, "state") != "CANCELLED", "fixture Create was not verified")
		}
		must(len(record.BoxID) == 36, "verified native fixture UUID absent")
		write(path, record)
		fmt.Println("S2_P0_COMBINED_NATIVE_VERIFIED_OWNED_FIXTURE=PASS")
	case "apply":
		record := read(path)
		must(record.BoxID != "" && record.UpdateID == "", "original P0 update already submitted")
		x, err := call(ctx, s, "box_update", updateArgs(record.BoxID, phase))
		must(err == nil && get(x, "operation_id") != "" && get(x, "tracking_id") != "",
			"original native Update not durably admitted")
		record.UpdateID, record.TrackingID = get(x, "operation_id"), get(x, "tracking_id")
		write(path, record)
		fmt.Println("S2_P0_COMBINED_ORIGINAL_NATIVE_UPDATE_ADMITTED=PASS")
	case "resume":
		record := read(path)
		must(record.UpdateID != "" && record.BoxID != "" && record.TrackingID != "",
			"lost original immutable Update receipt")
		x, err := call(ctx, s, "operation_get", map[string]any{"operation_id": record.UpdateID})
		must(err == nil, "original after native WAL crash unavailable")
		op := child(x, "operation")
		must(get(op, "state") == "UNKNOWN" && get(op, "tracking_id") == record.TrackingID &&
			get(op, "box_uuid") == record.BoxID &&
			get(op, "work_ref") == "lab/p0-native-sqlite" &&
			get(op, "agent_ref") == "p0-combined-verifier" &&
			get(child(op, "next_action"), "kind") == "HOST_LOCAL_REVIEW",
			"native+SQLite crash lost original identity, scope or safe UNKNOWN")
		again, e := call(ctx, s, "box_update", updateArgs(record.BoxID, phase))
		must(e == nil && get(again, "operation_id") == record.UpdateID &&
			again["duplicate"] == true, "same-key P0 mutation replayed rather than original")
		newArgs := updateArgs(record.BoxID, phase)
		newArgs["idempotency_key"] = "s2-p0-new-key-must-stay-locked-"+phase
		newArgs["cpu"] = 3
		_, e = call(ctx, s, "box_update", newArgs)
		must(e != nil, "new-key Update bypassed P0 original UNKNOWN lock")
		// Independent bounded client sessions read the same original through
		// official MCP HTTP while SQLite status and journal are stable.
		const clients = 8
		var wg sync.WaitGroup
		errors := make(chan error, clients)
		for i := 0; i < clients; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						errors <- fmt.Errorf("independent client panic: %v", r)
					}
				}()
				c, done := context.WithTimeout(context.Background(), 30*time.Second)
				defer done()
				peer := session(c, token)
				defer peer.Close()
				for n := 0; n < 4; n++ {
					out, er := call(c, peer, "operation_get", map[string]any{"operation_id": record.UpdateID, "wait_ms": 20})
					if er != nil || get(child(out, "operation"), "state") != "UNKNOWN" ||
						get(child(out, "operation"), "tracking_id") != record.TrackingID {
						errors <- fmt.Errorf("P0 concurrent observer invented terminal state: %v", er)
						return
					}
				}
			}()
		}
		wg.Wait()
		close(errors)
		for e := range errors {
			panic(e)
		}
		fmt.Println("S2_P0_REAL_NATIVE_SQLITE_CRASH_UNKNOWN_REPLAY_AND_CONCURRENT_READS=PASS")
	}
}
