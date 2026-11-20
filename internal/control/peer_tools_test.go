package control

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestS4B3PublicPeerToolExactSchemaAndAdminBoundary(t *testing.T) {
	_, ts, admin, viewer := fixture(t)
	cases := []struct {
		name, token string
		args        map[string]any
	}{
		{"viewer", viewer, map[string]any{"source_box_id": "one", "target_box_id": "two", "allow": true, "idempotency_key": "s4b3-viewer-refused-2026"}},
		{"unpinned", admin, map[string]any{"source_box_id": "one", "target_box_id": "two", "allow": true, "idempotency_key": "s4b3-unpinned-scope-denied"}},
		{"same-uuid", admin, map[string]any{"source_box_id": "same", "target_box_id": "same", "allow": true, "idempotency_key": "s4b3-same-uuid-denied-2026"}},
		{"root-path", admin, map[string]any{"source_box_id": "one", "target_box_id": "two", "allow": true, "idempotency_key": "s4b3-extra-root-path-denied", "host_root_path": "/host"}},
		{"native-ip", admin, map[string]any{"source_box_id": "one", "target_box_id": "two", "allow": true, "idempotency_key": "s4b3-extra-native-ip-denied", "ipv4": "10.0.0.2"}},
		{"missing-allow", admin, map[string]any{"source_box_id": "one", "target_box_id": "two", "idempotency_key": "s4b3-missing-allow-denied"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			client := mcp.NewClient(&mcp.Implementation{Name: "b3-schema-client", Version: "1"}, nil)
			sess, err := client.Connect(ctx, &mcp.StreamableClientTransport{
				Endpoint: ts.URL + "/mcp", HTTPClient: &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: tc.token}},
				DisableStandaloneSSE: true, MaxRetries: -1,
			}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer sess.Close()
			tools, err := sess.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, tool := range tools.Tools {
				if tool.Name == "box_peer_set" {
					found = true
				}
			}
			if !found {
				t.Fatal("public typed peer tool not registered")
			}
			got, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "box_peer_set", Arguments: tc.args})
			if err == nil && (got == nil || !got.IsError) {
				t.Fatalf("peer writer bypassed auth, schema or pinned scope: %+v", got)
			}
		})
	}
}
