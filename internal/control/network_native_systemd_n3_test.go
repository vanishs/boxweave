package control

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// Explicit gated test. The systemd unit and private SQLite clone are staged
// solely inside the isolated nested L1 Box by the separate shell harness.
// Never run against the physical L0 host or its installed boxweave.service.
func TestN3NativeInstalledSystemdMCPAcrossRestart(t *testing.T) {
	if os.Getenv("BW_N3_INSTALLED_UNIT_LAB") != "n3-l1-isolated-systemd-20261122" {
		t.Skip("isolated nested systemd lab only")
	}
	if os.Geteuid() != 1001 {
		t.Fatal("dedicated nonroot boxweave user required")
	}
	phase := os.Getenv("BW_N3_INSTALLED_UNIT_PHASE")
	if phase != "healthy" && phase != "tampered" && phase != "restored" {
		t.Fatal("unsupported unit phase")
	}
	project, _ := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	const dbPath = "/var/lib/boxweave-n3-p8/ledger.sqlite"
	const tokenPath = "/var/lib/boxweave-n3-p8/test-token.secret"
	s, e := state.Open(dbPath)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var raw string
	if phase == "healthy" {
		token, _, issueErr := s.IssueToken(ctx, 75*time.Minute)
		if issueErr != nil {
			t.Fatal(issueErr)
		}
		f, createErr := os.OpenFile(tokenPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if createErr != nil {
			t.Fatal(createErr)
		}
		_, createErr = f.WriteString(token)
		closeErr := f.Close()
		if createErr != nil || closeErr != nil {
			t.Fatal(createErr, closeErr)
		}
		raw = token
	} else {
		bytes, readErr := os.ReadFile(tokenPath)
		if readErr != nil {
			t.Fatal(readErr)
		}
		raw = string(bytes)
	}
	reader := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	client := mcp.NewClient(&mcp.Implementation{Name: "n3-isolated-installed-unit", Version: "1"}, nil)
	session, e := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             "http://127.0.0.1:18773/mcp",
		HTTPClient:           &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: raw}},
		DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer session.Close()
	call := func(name string, args map[string]any) map[string]any {
		response, callErr := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		return b4MCPData(t, response, callErr)
	}
	status := call("host_status", map[string]any{})
	if status["version"] != "n3-p8-systemd-lab" || status["commit"] != "70102d1af42dce165f84b79120689bb2d94d9908" {
		t.Fatal("not the source-exact independently installed unit", status["version"], status["commit"])
	}
	result := call("box_activity_get", map[string]any{"view": "native_network", "limit": 1})["activity"].(map[string]any)
	info := result["native_network"].(map[string]any)
	if info["owned"] != float64(1) || info["native"] != float64(1) || info["unowned_native"] != float64(0) ||
		info["orphan_bindings"] != float64(0) || info["orphan_fences"] != float64(0) ||
		info["entire_project_inventory_scanned"] != true || info["cross_source_atomic"] != false ||
		result["source"] != "SQLITE_AND_NATIVE_READONLY" || result["attention_complete"] != false {
		t.Fatal("lost full actual installed native network evidence", result)
	}
	issues, ok := result["issues"].([]any)
	if !ok && result["issues"] != nil {
		t.Fatal("malformed issues", result)
	}
	if phase == "tampered" {
		if len(issues) != 1 || issues[0].(map[string]any)["reason"] != "NATIVE_DB_PIN_UNVERIFIED" {
			t.Fatal("installed service lost secure original Create proof refusal", issues)
		}
	} else if len(issues) != 0 {
		t.Fatal("healthy native Box misclassified", issues)
	}
	ledger, e := s.ReadNativeNetworkLedger(ctx, project)
	if e != nil || len(ledger.Boxes) != 1 {
		t.Fatal("lost exact owner", ledger, e)
	}
	b := ledger.Boxes[0]
	box, e := reader.ObserveCreatedByOperation(ctx, b.Name, b.OriginalID)
	if e != nil || box.UUID != b.UUID || box.NativeState != "Stopped" {
		t.Fatal("real native original changed", box, e)
	}
	// The same private bearer token was created in the first phase and used
	// without reissuance after both real systemd restarts.
	if phase != "healthy" {
		if _, e = os.Stat(filepath.Join(filepath.Dir(dbPath), "test-token.secret")); e != nil {
			t.Fatal(e)
		}
	}
	t.Logf("N3_NATIVE_INSTALLED_SYSTEMD_MCP=PASS phase=%s box=%s original=%s", phase, b.UUID, b.OriginalID)
}
