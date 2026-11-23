package control

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

const n3AckName = "n3-create-ack-lab"
const n3AckMarker = "/run/boxweave-n3-create-disposable-lab"

// One-use nested lab only. No production fault switches and never an admin
// socket. Preserve the private test DB even after the abrupt child exit.
func n3AckLab(t *testing.T) (string, string) {
	t.Helper()
	if os.Getenv("BOXWEAVE_N3_CREATE_BOX_LAB") != "n3-create-disposable-20261122" {
		t.Skip("explicit one-use native Box lab only")
	}
	if os.Getuid() == 0 {
		t.Fatal("native test must use restricted account")
	}
	f, e := os.Lstat(n3AckMarker)
	if e != nil || !f.Mode().IsRegular() || f.Mode().Perm()&0022 != 0 {
		t.Fatal("unsafe lab marker", e)
	}
	owner, ok := f.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 {
		t.Fatal("lab marker not root-owned")
	}
	p := "user-" + strconv.Itoa(os.Getuid())
	raw, e := os.ReadFile(n3AckMarker)
	if e != nil || strings.TrimSpace(string(raw)) != p+" "+n3AckName {
		t.Fatal("wrong disposable project/name", e)
	}
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	return p, filepath.Join(home, "n3-create-ack-20261122", "state.db")
}
func n3AckMCP(t *testing.T, ctx context.Context, s *state.Store, r native.Reader) func(string, map[string]any) map[string]any {
	t.Helper()
	tok, _, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	svc := &Service{Store: s, Native: r, BuildVersion: "n3-ack-native-test", BuildCommit: "source-exact"}
	srv := httptest.NewUnstartedServer(svc.Handler())
	svc.Listener = srv.Listener.Addr().String()
	srv.Start()
	t.Cleanup(srv.Close)
	cli := mcp.NewClient(&mcp.Implementation{Name: "n3-ack-client", Version: "1"}, nil)
	sess, e := cli.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp", HTTPClient: &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: tok}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { sess.Close() })
	return func(name string, args map[string]any) map[string]any {
		t.Helper()
		out, e := sess.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		return b4MCPData(t, out, e)
	}
}
func TestN3NativeMCPCreateACKLoss(t *testing.T) {
	project, path := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	r := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	if os.Getenv("BW_N3_CREATE_CHILD") == "1" {
		s, e := state.Open(path)
		if e != nil {
			t.Fatal(e)
		}
		db, e := sql.Open("sqlite3", path)
		if e != nil {
			t.Fatal(e)
		}
		_, e = db.ExecContext(ctx, `CREATE TRIGGER n3_ack_fault BEFORE UPDATE OF native_operation_id ON operations
    WHEN OLD.action='box_create' AND OLD.work_ref='n3/create-ack-cut' AND OLD.native_operation_id='' AND NEW.native_operation_id!=''
    BEGIN SELECT RAISE(ABORT,'test native ACK durability cut'); END`)
		if e != nil {
			t.Fatal(e)
		}
		call := n3AckMCP(t, ctx, s, r)
		original := call("box_create", map[string]any{"name": n3AckName, "start": false, "cpu": 1, "memory_mib": 512, "disk_gib": 4, "idempotency_key": "n3-native-create-ack-original", "work_ref": "n3/create-ack-cut"})["operation_id"].(string)
		deadline := time.Now().Add(100 * time.Second)
		for time.Now().Before(deadline) {
			b, e := r.ObserveCreatedByOperation(ctx, n3AckName, original)
			if e == nil && b.UUID != "" && b.NativeState == "Stopped" {
				if pin, e := r.ObserveOriginalCreateStaticPeerBinding(ctx, n3AckName, b.UUID, original); e == nil && pin.IPv4 != "" && pin.IPv6 != "" {
					target, e := s.ReadRecoveryTarget(ctx, project, original)
					if e != nil || target.State != "DISPATCHING" || target.NativeID != "" || target.UUID != "" || !target.CreatePinStaged {
						t.Fatal("fault did not preserve original", target, e)
					}
					fmt.Printf("N3_NATIVE_CREATE_ACK_CUT original=%s uuid=%s\n", original, b.UUID)
					os.Exit(73)
				}
			}
			rec, e := s.Get(ctx, original)
			if e != nil {
				t.Fatal(e)
			}
			if rec.State == "FAILED" || rec.State == "SUCCEEDED" {
				t.Fatal("unexpected original terminal", rec)
			}
			time.Sleep(400 * time.Millisecond)
		}
		t.Fatal("real native static poststate did not settle")
	}
	c, e := incus.ConnectIncusUnixWithContext(ctx, r.Socket, nil)
	if e != nil {
		t.Fatal(e)
	}
	scope := c.UseProject(project)
	names, e := scope.GetInstanceNames(api.InstanceTypeAny)
	if e != nil {
		t.Fatal(e)
	}
	resume := os.Getenv("BW_N3_CREATE_RESUME_ORIGINAL")
	if resume == "" {
		if len(names) != 0 {
			t.Fatal("native lab must be empty")
		}
		if e = os.Mkdir(filepath.Dir(path), 0700); e != nil {
			t.Fatal("one-use evidence directory", e)
		}
		child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestN3NativeMCPCreateACKLoss$", "-test.v")
		child.Env = append(os.Environ(), "BW_N3_CREATE_CHILD=1")
		out, e := child.CombinedOutput()
		fmt.Print(string(out))
		exit, ok := e.(*exec.ExitError)
		if !ok || exit.ExitCode() != 73 {
			t.Fatalf("missing real ACK cut: %v", e)
		}
	} else {
		if len(names) != 1 || names[0] != n3AckName {
			t.Fatal("wrong resumed fixture", names)
		}
	}
	// Remove only the test-owned fault trigger before invoking the strict
	// product schema validator. Never relax the production validator.
	db, e := sql.Open("sqlite3", path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var original string
	if e = db.QueryRowContext(ctx, "SELECT id FROM operations WHERE action='box_create' AND work_ref='n3/create-ack-cut'").Scan(&original); e != nil {
		t.Fatal(e)
	}
	if resume != "" && resume != original {
		t.Fatal("resume original mismatch")
	}
	if _, e = db.ExecContext(ctx, "DROP TRIGGER IF EXISTS n3_ack_fault"); e != nil {
		t.Fatal(e)
	}
	s, e := state.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if n, e := s.Recover(ctx); e != nil || n > 1 || (resume == "" && n != 1) {
		t.Fatal("restart recovery", n, e)
	}
	before, e := s.Get(ctx, original)
	if e != nil || before.State != "UNKNOWN" || before.NativeOperationID != "" || before.BoxUUID != "" {
		t.Fatal(before, e)
	}
	call := n3AckMCP(t, ctx, s, r)
	diag := call("operation_recover", map[string]any{"operation_id": original, "mode": "inspect"})["diagnosis"].(map[string]any)
	if diag["disposition"] != "SAFE_RECOVER" || diag["reason"] != "CREATE_ACK_LOST_STATIC_POSTSTATE_VERIFIED" || diag["claim_protection"] != "RETAINED" {
		t.Fatal("wrong diagnosis", diag)
	}
	args := map[string]any{"operation_id": original, "mode": "reconcile", "idempotency_key": "n3-native-create-ack-reconcile"}
	first := call("operation_recover", args)
	recovery := first["recovery"].(map[string]any)
	op := recovery["operation"].(map[string]any)
	if recovery["disposition"] != "SETTLED" || op["state"] != "SUCCEEDED" || op["operation_id"] == original || first["native_retried"] != false {
		t.Fatal(first)
	}
	replay := call("operation_recover", args)["recovery"].(map[string]any)["operation"].(map[string]any)
	if replay["operation_id"] != op["operation_id"] || replay["duplicate"] != true {
		t.Fatal(replay)
	}
	after, e := s.Get(ctx, original)
	if e != nil || after.State != "SUCCEEDED" || after.NativeOperationID != "" || after.BoxUUID == "" {
		t.Fatal(after, e)
	}
	pin, e := r.ObserveOriginalCreateStaticPeerBinding(ctx, n3AckName, after.BoxUUID, original)
	if e != nil {
		t.Fatal(e)
	}
	ledger, e := s.ReadCompleteNetworkAuditLedger(ctx, project, pin.Bridge, pin.ACL)
	if e != nil || len(ledger.Boxes) != 1 || ledger.Boxes[0].UUID != after.BoxUUID || ledger.Boxes[0].IPv4 != pin.IPv4 || ledger.Boxes[0].IPv6 != pin.IPv6 {
		t.Fatal(ledger, e)
	}
	names, e = scope.GetInstanceNames(api.InstanceTypeAny)
	if e != nil || len(names) != 1 || names[0] != n3AckName {
		t.Fatal("native replay or wrong fixture", names, e)
	}
	fmt.Printf("N3_NATIVE_MCP_CREATE_ACK_LOSS=PASS original=%s recovery=%s uuid=%s\n", original, op["operation_id"], after.BoxUUID)
}
