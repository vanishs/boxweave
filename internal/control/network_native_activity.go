package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"slices"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// This is an explicit, on-demand existing MCP view, NOT a native mutation.
// Every page re-reads the full bounded native Project and SQLite snapshots.
// Never expose unowned native guest names, config, addresses or UUIDs.
func (s *Service) readNativeNetworkActivity(ctx context.Context, p RequestPrincipal, in activityGetInput) (state.ActivityResult, error) {
	if p.Layer != s.Native.Project || in.WorkRef != "" || in.AgentRef != "" || in.OperationID != "" ||
		in.Limit < 0 || in.Limit > 50 || len(in.Cursor) > 2048 || len(in.BoxID) > 128 {
		return state.ActivityResult{}, state.ErrInvalid
	}
	ledger, e := s.Store.ReadNativeNetworkLedger(ctx, p.Layer)
	if e != nil {
		return state.ActivityResult{}, e
	}
	expected := DiagnosticNetworkExpected(ledger)
	scan, e := s.Native.ScanProjectNativeNetwork(ctx, expected)
	if e != nil {
		// Read failures do not prove empty native inventory. Report a
		// typed, read-only incomplete result without leaking SDK errors.
		info := state.NativeNetworkAuditInfo{Owned: len(ledger.Boxes),
			OrphanBindings: ledger.OrphanBindings, OrphanFences: ledger.OrphanFences,
			Unsettled: ledger.Unsettled, InventoryScanned: false, CrossSourceAtomic: false}
		issues := []state.ActivityIssue{{Reason: "NATIVE_PROJECT_INVENTORY_UNVERIFIED",
			Disposition: "NATIVE_VERIFICATION_REQUIRED", ResourceRef: "network-audit:project",
			State: "UNVERIFIED", NextAction: &state.NextAction{Kind: "READ_STATUS", Tool: "host_status"}}}
		raw, _ := json.Marshal(ledger)
		sum := sha256.Sum256(raw)
		return s.Store.PageNativeNetworkIssues(ctx, p.Layer, p.TokenID, in.BoxID, in.Limit, in.Cursor, hex.EncodeToString(sum[:]), issues, info)
	}
	if scan.Project != p.Layer {
		return state.ActivityResult{}, state.ErrResourceLocked
	}
	// A consistent SQLite snapshot at the start is not a license to infer
	// that its pins stayed unchanged while native I/O ran. Re-read the exact
	// same bounded Project ledger and refuse to label mixed observations
	// healthy if either independent source changed during the scan.
	after, e := s.Store.ReadNativeNetworkLedger(ctx, p.Layer)
	if e != nil {
		return state.ActivityResult{}, e
	}
	if !reflect.DeepEqual(ledger, after) {
		info := state.NativeNetworkAuditInfo{Owned: len(after.Boxes), Native: scan.ObservedCount,
			Unowned: scan.UnownedCount, OrphanBindings: after.OrphanBindings,
			OrphanFences: after.OrphanFences, Unsettled: after.Unsettled,
			InventoryScanned: true, CrossSourceAtomic: false}
		issues := []state.ActivityIssue{{Reason: "NETWORK_LEDGER_CHANGED_DURING_AUDIT",
			Disposition: "NATIVE_VERIFICATION_REQUIRED", State: "UNVERIFIED",
			ResourceRef: "network-audit:project",
			NextAction:  &state.NextAction{Kind: "INSPECT_SCOPE", Tool: "box_activity_get", View: "issues"}}}
		raw, _ := json.Marshal(struct {
			First, After state.NativeNetworkLedger
			Scan         native.NetworkScanResult
		}{ledger, after, scan})
		digest := sha256.Sum256(raw)
		return s.Store.PageNativeNetworkIssues(ctx, p.Layer, p.TokenID, in.BoxID, in.Limit, in.Cursor, hex.EncodeToString(digest[:]), issues, info)
	}
	issues := projectNativeNetworkIssues(ledger, scan, in.BoxID)
	info := state.NativeNetworkAuditInfo{Owned: len(ledger.Boxes), Native: scan.ObservedCount,
		Unowned: scan.UnownedCount, OrphanBindings: ledger.OrphanBindings, OrphanFences: ledger.OrphanFences,
		Unsettled: ledger.Unsettled, InventoryScanned: true, CrossSourceAtomic: false}
	digestInput, _ := json.Marshal(struct {
		Ledger state.NativeNetworkLedger
		Scan   native.NetworkScanResult
		Issues []state.ActivityIssue
	}{ledger, scan, issues})
	hash := sha256.Sum256(digestInput)
	return s.Store.PageNativeNetworkIssues(ctx, p.Layer, p.TokenID, in.BoxID, in.Limit, in.Cursor, hex.EncodeToString(hash[:]), issues, info)
}

// DiagnosticNetworkExpected shares the exact pin predicate with Doctor.
func DiagnosticNetworkExpected(ledger state.NativeNetworkLedger) []native.NetworkScanExpected {
	expected := make([]native.NetworkScanExpected, 0, len(ledger.Boxes))
	for _, b := range ledger.Boxes {
		exact := b.OriginalCount == 1 && b.OriginalProof && b.OriginalState == "SUCCEEDED" && b.OriginalResource == "box:"+b.Name &&
			b.BindingCount == 1 && b.BindingState == "PINNED" && b.NIC == "eth0" &&
			b.Bridge != "" && b.ACL != "" && b.MAC != "" && b.IPv4 != "" && b.IPv6 != ""
		expected = append(expected, native.NetworkScanExpected{
			Name: b.Name, UUID: b.UUID, OriginalID: b.OriginalID, Bridge: b.Bridge, ACL: b.ACL, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6, HasPin: exact,
		})
	}
	return expected
}

// DiagnosticNetworkIssues is observation-only; never establishes recovery authority.
func DiagnosticNetworkIssues(ledger state.NativeNetworkLedger, scan native.NetworkScanResult) []state.ActivityIssue {
	return projectNativeNetworkIssues(ledger, scan, "")
}
func projectNativeNetworkIssues(ledger state.NativeNetworkLedger, scan native.NetworkScanResult, boxID string) []state.ActivityIssue {
	issues := []state.ActivityIssue{}
	matched := map[string]state.NativeNetworkBox{}
	for _, b := range ledger.Boxes {
		matched[b.UUID] = b
	}
	add := func(id, original, reason, disposition string) {
		if boxID != "" && id != boxID {
			return
		}
		issue := state.ActivityIssue{BoxUUID: id, OperationID: original, Reason: reason, Disposition: disposition, State: "UNVERIFIED", ResourceRef: "network-audit:" + id}
		if id == "" {
			issue.ResourceRef = "network-audit:project"
		}
		if original != "" {
			issue.NextAction = &state.NextAction{Kind: "READ_EVIDENCE", Tool: "box_activity_get", OperationID: original, View: "evidence"}
			if disposition == "NATIVE_VERIFICATION_REQUIRED" &&
				(reason == "NETWORK_BINDING_MISSING" || reason == "NETWORK_ADDRESS_FENCE_CONFLICT") {
				issue.NextAction = &state.NextAction{Kind: "INSPECT_RECOVERY", Tool: "operation_recover", OperationID: original, Mode: "inspect"}
			}
		} else {
			issue.NextAction = &state.NextAction{Kind: "INSPECT_SCOPE", Tool: "box_activity_get", View: "issues"}
		}
		issues = append(issues, issue)
	}
	for _, b := range ledger.Boxes {
		switch {
		case b.OriginalCount > 1:
			add(b.UUID, b.OriginalID, "NETWORK_CREATE_LINEAGE_DUPLICATE", "INSPECT_EVIDENCE")
		case b.OriginalCount == 1 && (b.OriginalState != "SUCCEEDED" || b.OriginalResource != "box:"+b.Name || !b.OriginalProof):
			add(b.UUID, b.OriginalID, "NETWORK_CREATE_LINEAGE_CONFLICT", "INSPECT_EVIDENCE")
		case b.OriginalCount == 0 && b.BindingCount == 0:
			add(b.UUID, "", "NETWORK_LEGACY_UNPINNED_NATIVE_UNVERIFIED", "INSPECT_EVIDENCE")
		case b.OriginalCount == 0:
			add(b.UUID, "", "NETWORK_CREATE_LINEAGE_MISSING", "INSPECT_EVIDENCE")
		}
		if b.OriginalCount == 0 && b.BindingCount == 0 {
			// Legacy owned Box has no modern original/prospective pin.
			// Do not invent a corrupted modern Create from absent rows.
			continue
		}
		if b.BindingCount != 1 || b.NIC != "eth0" || b.BindingState != "PINNED" {
			kind := "NETWORK_BINDING_INVALID"
			if b.BindingCount == 0 {
				kind = "NETWORK_BINDING_MISSING"
			}
			add(b.UUID, b.OriginalID, kind, "NATIVE_VERIFICATION_REQUIRED")
		} else if b.FenceCount != 2 || b.FenceIPv4 != 1 || b.FenceIPv6 != 1 {
			add(b.UUID, b.OriginalID, "NETWORK_ADDRESS_FENCE_CONFLICT", "NATIVE_VERIFICATION_REQUIRED")
		}
	}
	for _, a := range scan.Issues {
		if a.Reason == "NATIVE_DB_PIN_UNVERIFIED" && matched[a.BoxID].BindingCount == 0 {
			continue
		}
		disposition := "NATIVE_VERIFICATION_REQUIRED"
		if a.Reason == "NATIVE_OWNED_BOX_MISSING" || a.Reason == "NATIVE_OWNED_UUID_DRIFT" || a.Reason == "NATIVE_UNOWNED_PROJECT_INSTANCE" {
			disposition = "NATIVE_ORIGIN_UNVERIFIED"
		}
		add(a.BoxID, matched[a.BoxID].OriginalID, a.Reason, disposition)
	}
	if boxID == "" {
		if ledger.OrphanBindings > 0 {
			add("", "", "NETWORK_ORPHAN_BINDING_PRESENT", "INSPECT_EVIDENCE")
		}
		if ledger.OrphanFences > 0 {
			add("", "", "NETWORK_ORPHAN_FENCE_PRESENT", "INSPECT_EVIDENCE")
		}
	}
	slices.SortFunc(issues, func(a, b state.ActivityIssue) int {
		if a.BoxUUID != b.BoxUUID {
			if a.BoxUUID < b.BoxUUID {
				return -1
			}
			return 1
		}
		if a.Reason < b.Reason {
			return -1
		}
		if a.Reason > b.Reason {
			return 1
		}
		return 0
	})
	return issues
}
