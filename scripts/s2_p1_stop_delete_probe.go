//go:build ignore

// Official Go MCP SDK P1 original Stop/Delete lost native 202 ACK black box.
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
)

type original struct {
	BoxID    string `json:"box_id"`
	CreateID string `json:"create_id"`
	ActionID string `json:"action_id"`
}
type bearer struct{ token string }

func (b bearer) RoundTrip(req *http.Request) (*http.Response, error) {
	cp := req.Clone(req.Context())
	cp.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(cp)
}
func require(b bool, s string) {
	if !b {
		panic("S2_P1_STOP_DELETE_ACK_LOSS_FAIL: " + s)
	}
}
func connect(ctx context.Context, token string) *mcp.ClientSession {
	client := mcp.NewClient(&mcp.Implementation{Name: "s2-p1-stop-delete-loss", Version: "1"}, nil)
	s, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: "http://127.0.0.1:18658/mcp",
		HTTPClient: &http.Client{Transport: bearer{token}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	if err != nil {
		panic(err)
	}
	return s
}
func invoke(ctx context.Context, s *mcp.ClientSession, name string, args map[string]any) (map[string]any, error) {
	r, err := s.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return nil, err
	}
	if r == nil || r.IsError {
		return nil, fmt.Errorf("tool %s refused", name)
	}
	raw, err := json.Marshal(r.StructuredContent)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	err = json.Unmarshal(raw, &out)
	return out, err
}
func field(m map[string]any, k string) string         { v, _ := m[k].(string); return v }
func child(m map[string]any, k string) map[string]any { v, _ := m[k].(map[string]any); return v }
func read(path string) original {
	var x original
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	if err = json.Unmarshal(data, &x); err != nil {
		panic(err)
	}
	return x
}
func write(path string, x original) {
	data, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile(path, data, 0600); err != nil {
		panic(err)
	}
}
func actionArgs(phase, boxID string) map[string]any {
	return map[string]any{"box_id": boxID, "idempotency_key": "s2-p1-" + phase + "-native-202-ack-loss-original",
		"work_ref": "lab/p1-native-ack-loss", "agent_ref": "p1-ack-verifier"}
}
func main() {
	require(len(os.Args) == 2, "expected mode seed/mutate/poll/resume")
	mode := os.Args[1]
	require(mode == "seed" || mode == "mutate" || mode == "poll" || mode == "resume", "unknown test mode")
	dir, project, token := os.Getenv("BW_S2_P1_DIR"), os.Getenv("BW_S2_P1_PROJECT"), os.Getenv("BW_S2_P1_TOKEN")
	phase := strings.TrimPrefix(filepath.Base(dir), "p1-")
	require(strings.HasPrefix(dir, "/tmp/bw-v1-s2-ghci.") &&
		(phase == "stop" || phase == "delete") && strings.HasPrefix(project, "user-") &&
		os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		os.Geteuid() != 0 && strings.HasPrefix(token, "bw1_"),
		"only nonroot one-use GitHub Incus Project")
	name := "bw-s2-p1-" + phase + "-110p"
	path := filepath.Join(dir, "original.json")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	s := connect(ctx, token)
	defer s.Close()
	switch mode {
	case "seed":
		args := map[string]any{"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
			"start": phase == "stop", "internet": true, "idempotency_key": "s2-p1-" + phase + "-seed-once"}
		out, err := invoke(ctx, s, "box_create", args)
		require(err == nil && field(out, "operation_id") != "", "seed native Create not accepted")
		x := original{CreateID: field(out, "operation_id")}
		for n := 0; n < 120; n++ {
			res, e := invoke(ctx, s, "operation_get", map[string]any{"operation_id": x.CreateID, "wait_ms": 400})
			require(e == nil, "seed native Create not queryable")
			op := child(res, "operation")
			if field(op, "state") == "SUCCEEDED" {
				x.BoxID = field(op, "box_uuid")
				break
			}
			require(field(op, "state") != "FAILED" && field(op, "state") != "CANCELLED" && field(op, "state") != "UNKNOWN",
				"native seed not safely verified")
		}
		require(len(x.BoxID) == 36, "actual stopped/running native Box UUID not recorded")
		write(path, x)
		fmt.Println("S2_P1_STOP_DELETE_NATIVE_OWNED_REAL_UUID_SEEDED=PASS")
	case "mutate":
		x := read(path)
		require(x.BoxID != "" && x.ActionID == "", "original mutation already started")
		res, err := invoke(ctx, s, "box_"+phase, actionArgs(phase, x.BoxID))
		require(err == nil && field(res, "operation_id") != "", "original native Stop/Delete not admitted")
		x.ActionID = field(res, "operation_id")
		write(path, x)
		fmt.Println("S2_P1_STOP_DELETE_ORIGINAL_MUTATION_ADMITTED=PASS")
	case "poll", "resume":
		x := read(path)
		require(len(x.BoxID) == 36 && len(x.ActionID) == 32, "missing immutable original IDs")
		var op map[string]any
		for n := 0; n < 60; n++ {
			res, err := invoke(ctx, s, "operation_get", map[string]any{"operation_id": x.ActionID, "wait_ms": 300})
			require(err == nil, "original Stop/Delete receipt missing")
			op = child(res, "operation")
			if field(op, "state") == "UNKNOWN" {
				break
			}
			require(field(op, "state") != "SUCCEEDED" && field(op, "state") != "FAILED" && field(op, "state") != "CANCELLED",
				"lost ACK improperly settled original mutation")
		}
		require(field(op, "state") == "UNKNOWN" && field(op, "box_uuid") == x.BoxID &&
			field(op, "native_operation_id") == "" && field(child(op, "next_action"), "kind") == "HOST_LOCAL_REVIEW",
			"lost native ACK did not persist original UNKNOWN/box identity")
		if mode == "resume" {
			again, err := invoke(ctx, s, "box_"+phase, actionArgs(phase, x.BoxID))
			require(err == nil && field(again, "operation_id") == x.ActionID && again["duplicate"] == true,
				"same-key Stop/Delete replay lost immutable original receipt")
			newArgs := actionArgs(phase, x.BoxID)
			newArgs["idempotency_key"] = "s2-p1-" + phase + "-new-key-must-not-dispatch"
			_, err = invoke(ctx, s, "box_"+phase, newArgs)
			require(err != nil, "new-key Stop/Delete bypassed original UNKNOWN lock")
			fmt.Println("S2_P1_NATIVE_" + strings.ToUpper(phase) + "_LOST_ACK_RESTART_UNKNOWN_NO_REPLAY=PASS")
		} else {
			fmt.Println("S2_P1_NATIVE_" + strings.ToUpper(phase) + "_LOST_ACK_BEFORE_RESTART_UNKNOWN=PASS")
		}
	}
}
