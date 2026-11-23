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

func TestN3NetworkNativeAuditTypedScopeAndNoAdoption(t *testing.T) {
	b := state.NativeNetworkBox{UUID: "our-uuid", Incarnation: "our-uuid", Name: "alpha",
		OriginalID: "create-original", OriginalCount: 1, OriginalProof: true, OriginalState: "SUCCEEDED", OriginalResource: "box:alpha",
		BindingCount: 1, NIC: "eth0", Bridge: "bwbr", ACL: "acl", MAC: "02:16:3e:00:00:01", IPv4: "10.24.0.2", IPv6: "fd43::2",
		BindingState: "PINNED", FenceCount: 2, FenceIPv4: 1, FenceIPv6: 1}
	ledger := state.NativeNetworkLedger{Project: "user-1001", Boxes: []state.NativeNetworkBox{b}, OrphanBindings: 1, OrphanFences: 1}
	scan := native.NetworkScanResult{Project: "user-1001", ObservedCount: 2, UnownedCount: 1, Issues: []native.NetworkScanIssue{
		{BoxID: b.UUID, Reason: "NATIVE_NIC_IDENTITY_DRIFT"},
		{Reason: "NATIVE_UNOWNED_PROJECT_INSTANCE"},
	}}
	out := projectNativeNetworkIssues(ledger, scan, "")
	wanted := map[string]bool{"NATIVE_NIC_IDENTITY_DRIFT": false, "NATIVE_UNOWNED_PROJECT_INSTANCE": false, "NETWORK_ORPHAN_BINDING_PRESENT": false, "NETWORK_ORPHAN_FENCE_PRESENT": false}
	if len(out) != len(wanted) {
		t.Fatal(out)
	}
	for _, v := range out {
		if _, exists := wanted[v.Reason]; !exists {
			t.Fatal("unexpected issue", v)
		}
		wanted[v.Reason] = true
		if v.ResourceRef == "foreign-secret" || v.BoxUUID == "foreign-secret" {
			t.Fatal("foreign leak", v)
		}
		if v.NextAction == nil {
			t.Fatal("no recovery/evidence navigation", v)
		}
	}
	for reason, found := range wanted {
		if !found {
			t.Fatal("missing", reason)
		}
	}
	filtered := projectNativeNetworkIssues(ledger, scan, b.UUID)
	if len(filtered) != 1 || filtered[0].Reason != "NATIVE_NIC_IDENTITY_DRIFT" || filtered[0].OperationID != b.OriginalID {
		t.Fatal(filtered)
	}
	b.BindingCount = 0
	ledger.Boxes[0] = b
	missing := projectNativeNetworkIssues(ledger, native.NetworkScanResult{Project: "user-1001", Issues: []native.NetworkScanIssue{{BoxID: b.UUID, Reason: "NATIVE_DB_PIN_UNVERIFIED"}}}, b.UUID)
	if len(missing) != 1 || missing[0].Reason != "NETWORK_BINDING_MISSING" {
		t.Fatal("expected one actionable missing row", missing)
	}
}

func TestN3NativeNetworkObserverUnavailableNeverClaimsEmpty(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	s, e := state.Open(filepath.Join(dir, "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	project := "user-1001"
	svc := &Service{Store: s, Native: native.Reader{Project: project, Socket: "/nonexistent-incus"}}
	out, e := svc.readNativeNetworkActivity(ctx, RequestPrincipal{Layer: project, TokenID: tok.ID, ActorID: tok.ID}, activityGetInput{View: "native_network", Limit: 10})
	if e != nil || len(out.Issues) != 1 || out.Issues[0].Reason != "NATIVE_PROJECT_INVENTORY_UNVERIFIED" ||
		out.NativeNetwork == nil || out.NativeNetwork.InventoryScanned || out.CoverageDetails.NativeLiveVerified ||
		out.Source != "SQLITE_ONLY_NATIVE_UNAVAILABLE" || out.Coverage != "NATIVE_PROJECT_INVENTORY_UNVERIFIED" ||
		out.AttentionComplete || out.Issues[0].NextAction == nil || out.Issues[0].NextAction.Tool != "host_status" {
		t.Fatal("native reader failure incorrectly treated as all clear", out, e)
	}
}
