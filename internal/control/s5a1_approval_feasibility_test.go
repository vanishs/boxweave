package control

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Test only: no Box, real approval, credential, or mutation is issued here.
// Client-generated elicitation data is deliberately NOT an authorization.
func TestS5A1ElicitationFeasibility(t *testing.T) {
	for _, tc := range []struct {
		version, mode string
		handler       bool
		stateless     bool
	}{
		{"2026-07-28", "form", true, true},
		{"2026-07-28", "url", true, true},
		{"2026-07-28", "form", false, true},
		{"2025-11-25", "form", true, true},
		{"2025-11-25", "url", true, true},
		{"2025-11-25", "form", true, false},
		{"2025-11-25", "url", true, false},
	} {
		t.Run(tc.version+"/"+tc.mode+map[bool]string{true: "/stateless", false: "/stateful"}[tc.stateless]+map[bool]string{true: "/handler", false: "/nohandler"}[tc.handler], func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			server := mcp.NewServer(&mcp.Implementation{Name: "s5a1-inert", Version: "1"}, nil)
			mcp.AddTool(server, &mcp.Tool{Name: "probe"}, func(_ context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
				if len(req.Params.InputResponses) == 0 {
					p := &mcp.ElicitParams{Mode: tc.mode, Message: "Inert test prompt; no real authorization"}
					if tc.mode == "url" {
						p.URL = "https://example.invalid/s5a1"
						p.ElicitationID = "inert-probe"
					} else {
						p.RequestedSchema = map[string]any{"type": "object", "properties": map[string]any{"approved": map[string]any{"type": "boolean"}}}
					}
					return &mcp.CallToolResult{InputRequests: mcp.InputRequestMap{"probe": p}, RequestState: "inert"}, nil, nil
				}
				e, ok := req.Params.InputResponses["probe"].(*mcp.ElicitResult)
				if !ok {
					return &mcp.CallToolResult{IsError: true}, nil, nil
				}
				// Even an accepted result is not a proof of independent human presence.
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "CLIENT_" + e.Action + "_NOT_AUTHORIZATION"}}}, nil, nil
			})
			h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: tc.stateless, JSONResponse: true, PropagateRequestCancellation: false})
			ts := httptest.NewServer(h)
			defer ts.Close()
			opts := &mcp.ClientOptions{}
			if tc.handler {
				opts.Capabilities = &mcp.ClientCapabilities{Elicitation: &mcp.ElicitationCapabilities{Form: &mcp.FormElicitationCapabilities{}, URL: &mcp.URLElicitationCapabilities{}}}
				opts.ElicitationHandler = func(_ context.Context, _ *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
					e := &mcp.ElicitResult{Action: "accept"}
					if tc.mode == "form" {
						e.Content = map[string]any{"approved": true}
					}
					return e, nil
				}
			}
			c := mcp.NewClient(&mcp.Implementation{Name: "synthetic-client", Version: "1"}, opts)
			cs, err := c.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL, DisableStandaloneSSE: tc.stateless, MaxRetries: -1}, &mcp.ClientSessionOptions{ProtocolVersion: tc.version})
			if err != nil {
				t.Fatalf("connect: %v", err)
			}
			defer cs.Close()
			wantAvailable := tc.handler && (tc.version == "2026-07-28" || !tc.stateless)
			res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "probe"})
			if err != nil || res == nil || res.IsError {
				t.Logf("INTERACTION_UNAVAILABLE err=%v result_error=%t", err, res != nil && res.IsError)
				if wantAvailable {
					t.Fatal("expected supported interaction")
				}
				return
			}
			if !wantAvailable {
				t.Fatal("unexpected interaction on unavailable path")
			}
			if len(res.Content) != 1 {
				t.Fatalf("unexpected result: %+v", res)
			}
			got, ok := res.Content[0].(*mcp.TextContent)
			if !ok || got.Text != "CLIENT_accept_NOT_AUTHORIZATION" {
				t.Fatalf("unexpected reply: %+v", res)
			}
			t.Log("SYNTHETIC_CLIENT_ACCEPTED_WITHOUT_HUMAN; NOT_AUTHORIZATION")
		})
	}
}
