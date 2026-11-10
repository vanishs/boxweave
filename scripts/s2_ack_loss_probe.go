//go:build ignore

// Original native Create ACK-loss black-box verifier using the official MCP
// SDK client against the installed BoxWeave candidate and a one-use Incus VM.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	name = "bw-s2-ack-native-110d"
	key  = "native-s2-create-ack-dropped-once-001"
)

type saved struct {
	OriginalID string `json:"original_id"`
}

type authTransport struct{ token string }

func (a authTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c := r.Clone(r.Context())
	c.Header.Set("Authorization", "Bearer "+a.token)
	return http.DefaultTransport.RoundTrip(c)
}

func require(ok bool, msg string) {
	if !ok {
		panic("S2_ACK_LOSS_ACCEPTANCE_FAILED: " + msg)
	}
}

func session(ctx context.Context) *mcp.ClientSession {
	addr, secret := os.Getenv("BW_S2_MCP_URL"), os.Getenv("BW_S2_ADMIN_TOKEN")
	require(strings.HasPrefix(addr, "http://127.0.0.1:18652/") && secret != "",
		"only private disposable MCP listener and explicit authenticated actor allowed")
	client := mcp.NewClient(&mcp.Implementation{Name: "s2-native-ack-loss-verifier", Version: "1"}, nil)
	out, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: addr, HTTPClient: &http.Client{Transport: authTransport{token: secret}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		panic(err)
	}
	return out
}

func invoke(ctx context.Context, s *mcp.ClientSession, tool string, args map[string]any) (map[string]any, error) {
	r, err := s.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		return nil, err
	}
	if r == nil || r.IsError {
		return nil, fmt.Errorf("tool %s refused: %+v", tool, r)
	}
	b, err := json.Marshal(r.StructuredContent)
	if err != nil {
		return nil, err
	}
	var parsed map[string]any
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func field(m map[string]any, k string) string {
	out, _ := m[k].(string)
	return out
}

func child(m map[string]any, k string) map[string]any {
	out, _ := m[k].(map[string]any)
	return out
}

func createArgs() map[string]any {
	return map[string]any{
		"name": name, "cpu": 1, "memory_mib": 512, "disk_gib": 2,
		"start": false, "internet": true, "idempotency_key": key,
		"work_ref": "lab/lost-native-ack", "agent_ref": "ack-verifier",
	}
}

func checkUnknown(ctx context.Context, s *mcp.ClientSession, id string) map[string]any {
	var seen map[string]any
	for n := 0; n < 100; n++ {
		reply, err := invoke(ctx, s, "operation_get", map[string]any{
			"operation_id": id, "wait_ms": 300,
		})
		if err != nil {
			panic(err)
		}
		seen = child(reply, "operation")
		switch field(seen, "state") {
		case "UNKNOWN":
			require(field(seen, "native_operation_id") == "" &&
				field(seen, "error_class") == "NATIVE_OUTCOME_AMBIGUOUS" &&
				field(child(seen, "next_action"), "kind") == "HOST_LOCAL_REVIEW",
				"ACK-loss original lacked safe UNKNOWN lock or lied about native ID")
			return seen
		case "FAILED", "SUCCEEDED", "CANCELLED":
			panic(fmt.Sprintf("ACK-loss original incorrectly settled: %+v", seen))
		}
	}
	panic("original ACK loss never reached terminal safe UNKNOWN")
}

func replayMustNotRedispatch(ctx context.Context, s *mcp.ClientSession, originalID string) {
	replayed, err := invoke(ctx, s, "box_create", createArgs())
	if err != nil {
		panic(err)
	}
	require(field(replayed, "operation_id") == originalID &&
		replayed["duplicate"] == true, "same request key lost original UNKNOWN receipt")
	changed := createArgs()
	changed["idempotency_key"] = "native-s2-new-key-must-not-dispatch-001"
	unexpected, err := invoke(ctx, s, "box_create", changed)
	require(err != nil, fmt.Sprintf("different-key native create bypassed UNKNOWN name lock: %+v", unexpected))
	checkUnknown(ctx, s, originalID)
}

func main() {
	require(len(os.Args) == 2 && (os.Args[1] == "start" || os.Args[1] == "resume"),
		"start/resume mode required")
	file := os.Getenv("BW_S2_ACK_STATE_FILE")
	require(strings.HasPrefix(file, "/tmp/bw-v1-s2-ghci.") &&
		strings.HasSuffix(file, "/ack-original.json"), "only disposable original checkpoint permitted")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	s := session(ctx)
	defer s.Close()
	if os.Args[1] == "start" {
		created, err := invoke(ctx, s, "box_create", createArgs())
		if err != nil {
			panic(err)
		}
		id := field(created, "operation_id")
		require(id != "", "native create was not admitted before ACK lost")
		checkUnknown(ctx, s, id)
		replayMustNotRedispatch(ctx, s, id)
		data, err := json.Marshal(saved{OriginalID: id})
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(file, data, 0600); err != nil {
			panic(err)
		}
		fmt.Println("S2_NATIVE_CREATE_ACK_LOSS_ORIGINAL_UNKNOWN=PASS")
		return
	}
	data, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	var original saved
	if err := json.Unmarshal(data, &original); err != nil {
		panic(err)
	}
	require(original.OriginalID != "", "missing original checkpoint on restart")
	checkUnknown(ctx, s, original.OriginalID)
	replayMustNotRedispatch(ctx, s, original.OriginalID)
	fmt.Println("S2_NATIVE_CREATE_ACK_LOSS_RESTART_NO_REPLAY=PASS")
}
