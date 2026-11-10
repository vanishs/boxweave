package control

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (t bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	cp := r.Clone(r.Context())
	cp.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(cp)
}
func fixture(t *testing.T) (*state.Store, *httptest.Server, string, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	st, err := state.Open(filepath.Join(dir, "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	admin, a, err := st.IssueToken(context.Background(), "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	viewer, _, err := st.IssueToken(context.Background(), "viewer", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_ = a
	srv := &Service{Store: st, Native: native.Reader{Socket: "/nonexistent", Project: "bw-test"}, BuildVersion: "s1-test", BuildCommit: "test-commit", Listener: "127.0.0.1:18650"}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return st, ts, admin, viewer
}
func TestHTTPAuthorizationHostOriginAndRevocation(t *testing.T) {
	st, ts, admin, _ := fixture(t)
	for _, tc := range []struct {
		name, authorization, origin, host string
		want                              int
	}{
		{"missing", "", "", "", 401},
		{"bad", "Bearer wrong", "", "", 401},
		{"origin", "Bearer " + admin, "https://attacker.invalid", "", 403},
		{"host", "Bearer " + admin, "", "example.invalid:9000", 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader("{}"))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.host != "" {
				req.Host = tc.host
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Fatalf("HTTP %d, expected %d", resp.StatusCode, tc.want)
			}
		})
	}
	tok, _, err := st.IssueToken(context.Background(), "viewer", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tkn, err := st.VerifyToken(context.Background(), tok)
	if err != nil {
		t.Fatal(err)
	}
	if err = st.RevokeToken(context.Background(), tkn.ID); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatalf("revoked: HTTP %d", resp.StatusCode)
	}
}
func TestOfficialMCPHandshakeAndS2ReadTools(t *testing.T) {
	_, ts, admin, viewer := fixture(t)
	for _, tok := range []string{admin, viewer} {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		client := mcp.NewClient(&mcp.Implementation{Name: "s1-test-client", Version: "1"}, nil)
		transport := &mcp.StreamableClientTransport{
			Endpoint:             ts.URL + "/mcp",
			HTTPClient:           &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: tok}},
			DisableStandaloneSSE: true,
			MaxRetries:           -1,
		}
		sess, err := client.Connect(ctx, transport, nil)
		if err != nil {
			cancel()
			t.Fatalf("official MCP client connect: %v", err)
		}
		tools, err := sess.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("tools/list: %v", err)
		}
		gotNames := make(map[string]bool, len(tools.Tools))
		for _, tool := range tools.Tools {
			gotNames[tool.Name] = true
		}
		expected := []string{
			"host_status", "box_create", "box_list", "box_get", "box_update",
			"box_start", "box_stop", "box_delete", "box_exec",
			"operation_get", "operation_list", "operation_output_get", "operation_cancel",
		}
		if len(gotNames) != len(expected) {
			t.Fatalf("S2 requires exact 13-tool inventory: %+v", tools.Tools)
		}
		for _, name := range expected {
			if !gotNames[name] {
				t.Fatalf("missing S2 tool %q: %+v", name, tools.Tools)
			}
		}
		got, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "host_status", Arguments: map[string]any{}})
		if err != nil {
			t.Fatalf("host_status MCP call: %v", err)
		}
		if got.IsError {
			t.Fatalf("unexpected MCP tool error: %+v", got)
		}
		invalid, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: "host_status", Arguments: map[string]any{"unexpected_field": "must-reject"}})
		if err == nil && invalid != nil && !invalid.IsError {
			t.Fatal("unknown host_status input was silently ignored")
		}
		_ = sess.Close()
		cancel()
	}
}
