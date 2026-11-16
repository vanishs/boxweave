package control

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestS3ThreeFileToolsAndPreNativeDeniedPaths(t *testing.T) {
	_, ts, admin, viewer := fixture(t)
	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "admin", token: admin},
		{name: "viewer", token: viewer},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			client := mcp.NewClient(&mcp.Implementation{Name: "s3-file-schema-test", Version: "1"}, nil)
			session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
				Endpoint: ts.URL + "/mcp",
				HTTPClient: &http.Client{Transport: bearerTransport{
					base: http.DefaultTransport, token: tc.token,
				}},
				DisableStandaloneSSE: true,
				MaxRetries:           -1,
			}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			list, err := session.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for _, tool := range list.Tools {
				seen[tool.Name] = true
			}
			for _, name := range []string{"box_file_list", "box_file_read", "box_file_apply"} {
				if !seen[name] {
					t.Fatalf("%s missing from actual MCP tool catalog", name)
				}
			}
			for _, call := range []struct {
				name string
				args map[string]any
			}{
				{name: "box_file_list", args: map[string]any{
					"box_id": "foreign", "path": "/tmp/../etc/shadow",
				}},
				{name: "box_file_read", args: map[string]any{
					"box_id": "foreign", "path": "/proc/self/fd/0",
				}},
				{name: "box_file_apply", args: map[string]any{
					"box_id": "foreign", "path": "/tmp/safe",
					"mode": "replace", "idempotency_key": "s3-file-mode-replace-001",
				}},
				{name: "box_file_apply", args: map[string]any{
					"box_id": "foreign", "path": "/tmp/safe",
					"mode": "delete", "idempotency_key": "s3-file-mode-delete-001",
				}},
				{name: "box_file_apply", args: map[string]any{
					"box_id": "foreign", "path": "/tmp/file",
					"mode": "create", "idempotency_key": "s3-file-bad-base64-001",
					"content_base64": "!invalid!",
				}},
			} {
				got, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name: call.name, Arguments: call.args,
				})
				if err == nil && got != nil && !got.IsError {
					t.Fatalf("%s %s accepted unsafe input: %+v", tc.name, call.name, got)
				}
			}
			if tc.name == "admin" {
				// Admin still cannot cause a native effect when a wire schema
				// contains an unsupported or silently ignored extra field.
				got, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name: "box_file_apply", Arguments: map[string]any{
						"box_id": "foreign", "path": "/tmp/safe",
						"mode": "create", "idempotency_key": "s3-file-extra-key-001",
						"unknown_host_path": "/var/lib/incus",
					},
				})
				if err == nil && got != nil && !got.IsError {
					t.Fatal("unknown host-path parameter was silently accepted")
				}
			}
		})
	}
}

func TestS3FileToolDescriptionsDiscloseUnsupportedConditionalModes(t *testing.T) {
	_, ts, admin, _ := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "s3-file-description-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: ts.URL + "/mcp",
		HTTPClient: &http.Client{Transport: bearerTransport{
			base: http.DefaultTransport, token: admin,
		}},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	list, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range list.Tools {
		if tool.Name == "box_file_apply" {
			if !strings.Contains(tool.Description, "UNSUPPORTED") ||
				!strings.Contains(tool.Description, "O_EXCL") {
				t.Fatalf("unsafe conditional writes are not clearly disclosed: %q", tool.Description)
			}
			return
		}
	}
	t.Fatal("file apply tool missing")
}
