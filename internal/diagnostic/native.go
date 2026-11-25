package diagnostic

import (
	"context"
	"github.com/vanishs/boxweave/internal/control"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// NativeObservation is bounded, redacted evidence from the restricted
// actor. It never contains raw Incus config, foreign IDs or credentials.
type NativeObservation struct {
	Status                native.Status                 `json:"status"`
	Inventory             Source                        `json:"inventory"`
	InventoryInfo         *state.NativeNetworkAuditInfo `json:"inventory_info,omitempty"`
	Issues                []state.ActivityIssue         `json:"issues,omitempty"`
	IssuesTruncated       bool                          `json:"issues_truncated"`
	Forward               Source                        `json:"forward"`
	ForwardVerified       int                           `json:"forward_verified"`
	AddressCapacity       *native.AddressCapacity       `json:"address_capacity,omitempty"`
	AddressCapacitySource Source                        `json:"address_capacity_source"`
	LedgerDigest          string                        `json:"ledger_digest,omitempty"`
}

func ReadNativeObservation(ctx context.Context, r native.Reader, st native.Status, snap *state.DiagnosticSnapshot, l *state.DiagnosticLedger) NativeObservation {
	out := NativeObservation{Status: st, Inventory: Source{State: "UNVERIFIED", Reason: "NATIVE_INVENTORY_NOT_OBSERVED"}, Forward: Source{State: "UNVERIFIED", Reason: "FORWARD_DEVICE_NOT_OBSERVED"}, AddressCapacitySource: Source{State: "UNVERIFIED", Reason: "ADDRESS_CAPACITY_NOT_OBSERVED"}}
	if snap == nil || l == nil {
		return out
	}
	out.LedgerDigest = l.ObservationDigest()
	if !st.Ready {
		return out
	}
	// This is an exact bounded comparison, not packet or kernel ACL proof.
	if !l.NetworkObserved {
		out.Inventory.Reason = "SQLITE_NETWORK_LEDGER_UNVERIFIED"
	}
	if l.NetworkObserved {
		out.Inventory.Reason = "NATIVE_PROJECT_INVENTORY_UNVERIFIED"
		scan, e := r.ScanProjectNativeNetwork(ctx, control.DiagnosticNetworkExpected(l.Network))
		if e == nil && scan.Project == r.Project {
			out.Inventory = Source{State: "OBSERVED", Reason: "RESTRICTED_NIC_AND_LEASE_COMPARISON", EpochNS: snap.EpochNS}
			out.InventoryInfo = &state.NativeNetworkAuditInfo{Owned: len(l.Network.Boxes), Native: scan.ObservedCount, Unowned: scan.UnownedCount, OrphanBindings: l.Network.OrphanBindings, OrphanFences: l.Network.OrphanFences, Unsettled: l.Network.Unsettled, InventoryScanned: true, CrossSourceAtomic: false}
			issues := control.DiagnosticNetworkIssues(l.Network, scan)
			if len(issues) > 50 {
				out.IssuesTruncated = true
				issues = issues[:50]
			}
			out.Issues = issues
		}
	}
	out.Forward = Source{State: "OBSERVED", Reason: "CONFIRMED_FORWARD_DEVICE_ONLY", EpochNS: snap.EpochNS}
	for _, f := range l.Forwards {
		if ctx.Err() != nil {
			out.Forward = Source{State: "UNVERIFIED", Reason: "FORWARD_SCAN_DEADLINE"}
			break
		}
		spec := native.ForwardSpec{Device: f.Device, Protocol: f.Protocol, HostBind: f.Bind, HostPort: f.HostPort, GuestPort: f.GuestPort, ExternalAcknowledged: true}
		if f.Name == "" || r.ObserveOwnedForward(ctx, f.Name, f.UUID, spec, true) != nil {
			out.Issues = append(out.Issues, state.ActivityIssue{Reason: "FORWARD_DEVICE_UNVERIFIED", Disposition: "NATIVE_VERIFICATION_REQUIRED", State: "UNVERIFIED", BoxUUID: f.UUID, ResourceRef: "forward:" + f.ID})
			out.Forward = Source{State: "PARTIAL", Reason: "FORWARD_DEVICE_PROOF_INCOMPLETE"}
		} else {
			out.ForwardVerified++
		}
	}
	if l.ForwardTruncated || l.ForwardUnresolved > 0 {
		out.Forward = Source{State: "PARTIAL", Reason: "FORWARD_UNRESOLVED_OR_BOUNDED"}
	}
	if cap, e := r.ReadCreateAddressCapacity(ctx, snap.ReadAddressHolds); e == nil {
		// The verified confined bridge prefix is the same typed capacity
		// field as host_status. This is an upper bound, not allocation proof.
		out.AddressCapacity = &cap
		out.AddressCapacitySource = Source{State: "OBSERVED", Reason: "IPV4_UPPER_BOUND_NOT_ALLOCATION_PROOF", EpochNS: snap.EpochNS}
	}
	return out
}
