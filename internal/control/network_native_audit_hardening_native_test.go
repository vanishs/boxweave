package control

import (
	"context"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Re-open the already validated one-use clone, without touching native
// config or the authentic original Create's private DB.
func TestN3NativeMCPFullInventoryHardenedReadOnly(t *testing.T) {
	project, _ := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	s, e := state.Open(filepath.Join(home, "n3-full-project-scan-v2-20261122", "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	ledger, e := s.ReadNativeNetworkLedger(ctx, project)
	if e != nil || len(ledger.Boxes) != 1 || !ledger.Boxes[0].OriginalProof {
		t.Fatal("provenance not complete", ledger, e)
	}
	r := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	expected := []native.NetworkScanExpected{{Name: ledger.Boxes[0].Name, UUID: ledger.Boxes[0].UUID,
		OriginalID: ledger.Boxes[0].OriginalID, Bridge: ledger.Boxes[0].Bridge, ACL: ledger.Boxes[0].ACL,
		MAC: ledger.Boxes[0].MAC, IPv4: ledger.Boxes[0].IPv4, IPv6: ledger.Boxes[0].IPv6, HasPin: true}}
	nativeView, e := r.ScanProjectNativeNetwork(ctx, expected)
	if e != nil || nativeView.ObservationDigest == "" || len(nativeView.Issues) != 0 || nativeView.ObservedCount != 1 {
		t.Fatal("unverified full native inventory", nativeView, e)
	}
	call := n3AckMCP(t, ctx, s, r)
	result := call("box_activity_get", map[string]any{"view": "native_network", "limit": 1})["activity"].(map[string]any)
	info := result["native_network"].(map[string]any)
	if info["owned"] != float64(1) || info["native"] != float64(1) || info["entire_project_inventory_scanned"] != true {
		t.Fatal("MCP full-scope lost", result)
	}
	if v, ok := result["issues"].([]any); ok && len(v) > 0 {
		t.Fatal("unexpected healthy regression", v)
	}
	t.Logf("N3_NATIVE_MCP_HARDENED_FULL_INVENTORY=PASS original=%s", ledger.Boxes[0].OriginalID)
}
