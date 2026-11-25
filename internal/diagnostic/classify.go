package diagnostic

import (
	"github.com/vanishs/boxweave/internal/state"
	"strings"
)

func (r *Report) AddRoutedIssue(domain, code, resource, class, route string) {
	r.AddIssue(domain, code, resource, class)
	id := IssueID(r.Scope, domain, code, resource)
	for i := range r.Issues {
		if r.Issues[i].ID == id {
			r.Issues[i].Repairability = route
			return
		}
	}
}
func (r *Report) classifyActivity(a state.ActivityIssue, history bool) {
	// Only typed producer reasons are allowed; raw SQLite refs/labels
	// must not become report content or arbitrary repair commands.
	ref := state.SafeDiagnosticRef(a.ResourceRef)
	if a.OperationID != "" {
		ref = state.SafeDiagnosticRef(a.OperationID) + "/" + ref
	}
	class := "UNVERIFIED"
	route := "INSPECT_EVIDENCE"
	if history {
		class = "HISTORICAL"
		route = "NOT_APPLICABLE"
	}
	switch a.Disposition {
	case "ACTIVE_DEPENDENCY_NOT_ERROR":
		return
	case "IN_PROGRESS_OR_STALLED":
		class = "IN_PROGRESS"
		route = "OBSERVE_OR_INSPECT"
	case "MCP_RECOVERY_INSPECTION_REQUIRED":
		route = "MCP_INSPECT_REQUIRED"
	case "HOST_REVIEW_REQUIRED":
		route = "HOST_REVIEW_REQUIRED"
	case "NATIVE_VERIFICATION_REQUIRED", "NATIVE_ORIGIN_UNVERIFIED":
		route = "NATIVE_VERIFICATION_REQUIRED"
	}
	domain := "operations"
	if strings.HasPrefix(a.Reason, "NETWORK_") || strings.HasPrefix(a.Reason, "NATIVE_") || strings.Contains(a.Reason, "ADDRESS_FENCE") {
		domain = "network"
	}
	if strings.Contains(a.Reason, "PEER") {
		domain = "peer"
	}
	if strings.Contains(a.Reason, "FORWARD") {
		domain = "forward"
	}
	r.AddRoutedIssue(domain, a.Reason, ref, class, route)
	// Exact original/box IDs, not native names or addresses.
	id := IssueID(r.Scope, domain, a.Reason, ref)
	for i := range r.Issues {
		if r.Issues[i].ID == id {
			if a.OperationID != "" {
				r.Issues[i].OriginalID = state.SafeDiagnosticRef(a.OperationID)
			}
			if a.BoxUUID != "" {
				r.Issues[i].BoxID = state.SafeDiagnosticRef(a.BoxUUID)
			}
		}
	}
}
func (r *Report) ClassifyLedger(l state.DiagnosticLedger) {
	r.Ledger = &l
	r.ActivityCoverage = Source{State: "OBSERVED", Reason: "BOUNDED_LOCAL_SQLITE_ACTIVITY"}
	if l.Activity.Truncated || l.ActiveTruncated {
		r.ActivityCoverage = Source{State: "PARTIAL", Reason: "BOUNDED_ACTIVE_SAMPLE"}
	}
	for _, a := range l.Activity.Issues {
		r.classifyActivity(a, false)
	}
	for _, a := range l.Activity.History {
		r.classifyActivity(a, true)
	}
	if l.PeerCapacity != nil {
		p := l.PeerCapacity
		if !p.NewDirectionAvailable {
			r.AddRoutedIssue("capacity", "SCOPED_PEER_RETAINED_DIRECTION_LIMIT", "peer-ledger", "CONFIRMED", "NO_RESET_OR_COMPACTION")
		}
		if !p.NewGrantHistoryAvailable {
			r.AddRoutedIssue("capacity", "SCOPED_PEER_GRANT_HISTORY_RESERVE", "peer-ledger", "CONFIRMED", "NO_RESET_OR_COMPACTION")
		}

	}
	if p := l.ScopedIdentityCapacity; p != nil && (p.Level == "EXHAUSTED" || p.Level == "OVER_LIMIT") {
		r.AddRoutedIssue("capacity", "SCOPED_RETIRED_IDENTITY_CAPACITY", "ledger", "CONFIRMED", "NO_RESET_OR_COMPACTION")
	}
	if l.LegacyRetirement != nil && l.LegacyRetirement.Pending > 0 {
		r.AddRoutedIssue("network", "LEGACY_IPV4_RETIREMENT_PENDING", "ledger", "IN_PROGRESS", "INSPECT_LEGACY_RETIREMENT")
	}
	if !l.HoldsObserved {
		r.AddRoutedIssue("capacity", "ADDRESS_HOLDS_UNVERIFIED", "ledger", "UNVERIFIED", "INSPECT_EVIDENCE")
	}
	if l.OAuth.OriginConfigured && !l.OAuth.PasswordConfigured {
		r.AddRoutedIssue("oauth", "OAUTH_ORIGIN_WITHOUT_PASSWORD", "oauth", "CONFIRMED", "LOCAL_CONFIGURATION_REVIEW")
	}
	// OAuth disabled is not a failure; even enabled config doesn't prove auth.
	r.OAuth = Source{State: "OBSERVED", Reason: "CONFIG_ONLY_AUTHENTICATED_FLOW_NOT_CHECKED"}
}
func (r *Report) ClassifyNative(n NativeObservation) {
	r.NativeInventory = n.Inventory
	r.NativeInventoryInfo = n.InventoryInfo
	r.ForwardDevices = n.Forward
	r.ForwardVerified = n.ForwardVerified
	r.AddressCapacity = n.AddressCapacitySource
	if n.Inventory.State != "OBSERVED" {
		r.AddRoutedIssue("network", n.Inventory.Reason, "project", "UNVERIFIED", "NATIVE_VERIFICATION_REQUIRED")
	}
	if n.IssuesTruncated {
		r.NativeInventory.State = "PARTIAL"
		r.NativeInventory.Reason = "NATIVE_ISSUES_BOUNDED"
	}
	for _, i := range n.Issues {
		r.classifyActivity(i, false)
	}
	if n.AddressCapacity != nil && n.AddressCapacity.RemainingIPv4UpperBound == 0 {
		r.AddRoutedIssue("capacity", "IPV4_UPPER_BOUND_EXHAUSTED", "project", "CONFIRMED", "NO_AUTOMATIC_ADDRESS_REUSE")
	}
	if !n.Status.StorageQuotaVerified {
		r.StorageQuota = Source{State: "UNVERIFIED", Reason: "NO_CURRENT_INDEPENDENT_QUOTA_WITNESS"}
	} else {
		r.StorageQuota = Source{State: "OBSERVED", Reason: "RECENT_HOST_QUOTA_WITNESS_NOT_CONTINUOUS"}
	}
}
func (r *Report) ClassifyInstallation() {
	i := r.Installation
	if i.Mode == "CONFLICT" {
		r.AddRoutedIssue("installation", "ARCHITECTURE_EVIDENCE_CONFLICT", "installation", "CONFIRMED", "HOST_REVIEW_REQUIRED")
	}
	if i.ServiceUnit == "FAILED" || i.ServiceUnit == "INACTIVE" {
		r.AddRoutedIssue("service", "CONTROLLER_"+i.ServiceUnit, "controller", "CONFIRMED", "HOST_REVIEW_REQUIRED")
	}
	if i.Mode == "SCOPED_V2" || i.Candidate == "SCOPED_V2" && i.Runtime == "VERIFIED" {
		if i.LifecycleUnit == "FAILED" || i.LifecycleUnit == "INACTIVE" || i.LifecycleUnit == "TRANSITIONING" {
			r.AddRoutedIssue("root", "SCOPED_LIFECYCLE_"+i.LifecycleUnit, "root-helper", "CONFIRMED", "INSPECT_ROOT_EVIDENCE")
		}
		if i.GuardUnit != "ACTIVE" {
			r.AddRoutedIssue("root", "SCOPED_BOOT_GUARD_NOT_ACTIVE", "root-guard", "UNVERIFIED", "HOST_REVIEW_REQUIRED")
		}
	}
}
