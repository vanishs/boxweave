//go:build s4b3_native_ci

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
	"github.com/vanishs/boxweave/internal/state"
)

type peerBearer struct {
	token string
	base  http.RoundTripper
}

func (b peerBearer) RoundTrip(req *http.Request) (*http.Response, error) {
	cp := req.Clone(req.Context())
	cp.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(cp)
}
func main() {
	token, revoked, source, target := os.Getenv("S4B3_MCP_TOKEN"), os.Getenv("S4B3_REVOKED_TOKEN"), os.Getenv("S4B3_SOURCE_UUID"), os.Getenv("S4B3_TARGET_UUID")
	if token == "" || revoked == "" || source == "" || target == "" || source == target {
		panic("exact tokens and UUIDs required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	connect := func(token string) *mcp.ClientSession {
		c := mcp.NewClient(&mcp.Implementation{Name: "b3-installed-native-client", Version: "1"}, nil)
		s, err := c.Connect(ctx, &mcp.StreamableClientTransport{
			Endpoint:             "http://127.0.0.1:18650/mcp",
			HTTPClient:           &http.Client{Transport: peerBearer{token: token, base: http.DefaultTransport}},
			DisableStandaloneSSE: true, MaxRetries: -1,
		}, nil)
		if err != nil {
			panic(err)
		}
		return s
	}
	admin := connect(token)
	defer admin.Close()
	tools, err := admin.ListTools(ctx, nil)
	if err != nil {
		panic(err)
	}
	found := false
	for _, t := range tools.Tools {
		if t.Name == "box_peer_set" {
			found = true
		}
	}
	if !found {
		panic("missing registered box_peer_set")
	}
	key := "s4b3-installed-mcp-directional-grant-1118"
	args := map[string]any{"source_box_id": source, "target_box_id": target, "allow": true, "idempotency_key": key}
	reply, err := admin.CallTool(ctx, &mcp.CallToolParams{Name: "box_peer_set", Arguments: args})
	if err != nil || reply == nil || reply.IsError {
		// Print only a known typed failure class from this disposable runner.
		// Never echo native error bodies, headers or bearer credentials.
		class := "UNCLASSIFIED"
		if reply != nil {
			for _, part := range reply.Content {
				if msg, ok := part.(*mcp.TextContent); ok {
					for _, known := range []string{
						"RESOURCE_LOCKED", "INVALID", "PERMISSION_DENIED", "UNAUTHENTICATED",
						"CONFLICT", "NOT_FOUND", "REVOKED", "NOT_OWNED", "INCUS",
						"SQLITE", "DATABASE", "NATIVE", "UNKNOWN", "TOKEN",
					} {
						if strings.Contains(strings.ToUpper(msg.Text), known) {
							class = known
							break
						}
					}
				}
			}
		}
		fmt.Println("S4_B3_PUBLIC_MCP_ADMISSION_REFUSED_CLASS=" + class)
		panic("official MCP first admission refused; no native side effect")
	}
	encoded, err := json.Marshal(reply.StructuredContent)
	if err != nil {
		panic(err)
	}
	var receipt state.Receipt
	if err := json.Unmarshal(encoded, &receipt); err != nil || len(receipt.ID) != 32 || receipt.Action != "box_peer_set" || receipt.State != "ADMITTED" {
		panic(fmt.Sprintf("MCP original not durable: %+v %v", receipt, err))
	}
	second, err := admin.CallTool(ctx, &mcp.CallToolParams{Name: "box_peer_set", Arguments: args})
	if err != nil || second == nil || second.IsError {
		panic("MCP same-key replay rejected")
	}
	encoded, err = json.Marshal(second.StructuredContent)
	if err != nil {
		panic(err)
	}
	var duplicate state.Receipt
	if err := json.Unmarshal(encoded, &duplicate); err != nil || duplicate.ID != receipt.ID || !duplicate.Duplicate {
		panic("MCP same-key original was not exactly replayed")
	}
	changed := map[string]any{"source_box_id": source, "target_box_id": target, "allow": false, "idempotency_key": key}
	reject, err := admin.CallTool(ctx, &mcp.CallToolParams{Name: "box_peer_set", Arguments: changed})
	if err == nil && reject != nil && !reject.IsError {
		panic("same-key opposite direction mutation accepted")
	}
	forged := map[string]any{"source_box_id": source, "target_box_id": target, "allow": true, "idempotency_key": key, "acl": "forged"}
	reject, err = admin.CallTool(ctx, &mcp.CallToolParams{Name: "box_peer_set", Arguments: forged})
	if err == nil && reject != nil && !reject.IsError {
		panic("forged ACL parameter accepted")
	}
	read, err := admin.CallTool(ctx, &mcp.CallToolParams{Name: "operation_get", Arguments: map[string]any{"operation_id": receipt.ID}})
	if err != nil || read == nil || read.IsError {
		panic("operation_get missing original")
	}
	// A revoked credential cannot even reach the installed MCP tool dispatcher.
	negativeRequest, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://127.0.0.1:18650/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"box_peer_set","arguments":{}}}`))
	if err != nil { panic(err) }
	negativeRequest.Header.Set("Authorization", "Bearer "+revoked)
	negativeRequest.Header.Set("Content-Type", "application/json")
	negativeRequest.Header.Set("Accept", "application/json")
	negativeReply, err := http.DefaultClient.Do(negativeRequest)
	if err != nil { panic(err) }
	negativeReply.Body.Close()
	if negativeReply.StatusCode != http.StatusUnauthorized {
		panic("revoked credential reached directional network grant")
	}
	fmt.Printf("S4_B3_NATIVE_MCP_ORIGINAL=%s\n", receipt.ID)
	fmt.Println("S4_B3_NATIVE_OFFICIAL_MCP_AUTH_TOKEN_REPLAY_SCHEMA_REVOKED_DENIAL_OPERATION_GET=PASS")
}
