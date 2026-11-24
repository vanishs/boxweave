// Package diagnostic defines the versioned, redacted observation contract
// shared by Doctor and later Recover planning. It grants no effect authority.
package diagnostic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"runtime"
	"time"

	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/state"
)

const ReportVersion = 1

type Source struct {
	State   string `json:"state"`
	Reason  string `json:"reason,omitempty"`
	EpochNS int64  `json:"epoch_ns,omitempty"`
}
type Issue struct {
	ID             string `json:"id"`
	Domain         string `json:"domain"`
	Code           string `json:"code"`
	Resource       string `json:"resource"`
	Classification string `json:"classification"`
	Repairability  string `json:"repairability"`
	OriginalID     string `json:"original_id,omitempty"`
	BoxID          string `json:"box_id,omitempty"`
}
type Report struct {
	Version             int                            `json:"version"`
	AsOfNS              int64                          `json:"as_of_ns"`
	Scope               string                         `json:"scope"`
	GoVersion           string                         `json:"-"`
	Coverage            string                         `json:"coverage"`
	CrossSourceAtomic   bool                           `json:"cross_source_atomic"`
	Database            Source                         `json:"database"`
	Native              Source                         `json:"native"`
	MCP                 Source                         `json:"mcp"`
	Installation        install.DiagnosticInstallation `json:"installation"`
	ScopedLedger        *state.DiagnosticScopedSummary `json:"scoped_ledger,omitempty"`
	Ledger              *state.DiagnosticLedger        `json:"ledger,omitempty"`
	ActivityCoverage    Source                         `json:"activity_coverage"`
	NativeInventory     Source                         `json:"native_inventory"`
	NativeInventoryInfo *state.NativeNetworkAuditInfo  `json:"native_inventory_info,omitempty"`
	ForwardDevices      Source                         `json:"forward_devices"`
	ForwardVerified     int                            `json:"forward_verified"`
	ForwardPolicy       Source                         `json:"forward_policy"`
	AddressCapacity     Source                         `json:"address_capacity"`
	RootPolicy          install.DiagnosticRootPolicy   `json:"root_policy"`
	OAuth               Source                         `json:"oauth"`
	StorageQuota        Source                         `json:"storage_quota"`
	Tunnel              Source                         `json:"tunnel"`
	PacketReachability  Source                         `json:"packet_reachability"`

	Issues []Issue `json:"issues"`
}

func NewReport(scope string) Report {
	return Report{Version: ReportVersion, AsOfNS: time.Now().UnixNano(), Scope: scope, GoVersion: runtime.Version(), Coverage: "PARTIAL_D2_DIAGNOSIS", ActivityCoverage: Source{State: "NOT_CHECKED"}, NativeInventory: Source{State: "NOT_CHECKED"}, ForwardDevices: Source{State: "NOT_CHECKED"}, ForwardPolicy: Source{State: "NOT_CHECKED"}, AddressCapacity: Source{State: "NOT_CHECKED"}, RootPolicy: install.DiagnosticRootPolicy{State: "NOT_CHECKED", Kernel: "NOT_CHECKED", Conntrack: "NOT_CHECKED"}, OAuth: Source{State: "NOT_CHECKED"}, StorageQuota: Source{State: "NOT_CHECKED"}, Tunnel: Source{State: "NOT_CHECKED", Reason: "EXTERNAL_REACHABILITY_NOT_PROBED"}, PacketReachability: Source{State: "NOT_CHECKED", Reason: "CONFIGURATION_IS_NOT_PACKET_PROOF"}, MCP: Source{State: "NOT_CHECKED", Reason: "LEGACY_MCP_READY_IS_SCHEMA_ONLY"}, Issues: []Issue{}}
}

// IDs bind immutable scope and reason, not changing observations or
// timestamps. They are identifiers, never proof, approval or replay keys.
func IssueID(scope, domain, code, resource string) string {
	raw, _ := json.Marshal([]string{"boxweave-issue-v1", scope, domain, code, resource})
	h := sha256.Sum256(raw)
	return "d1-" + hex.EncodeToString(h[:])
}
func (r *Report) AddIssue(domain, code, resource, class string) {
	if code == "" {
		code = "EVIDENCE_UNAVAILABLE"
	}
	id := IssueID(r.Scope, domain, code, resource)
	for _, i := range r.Issues {
		if i.ID == id {
			return
		}
	}
	r.Issues = append(r.Issues, Issue{ID: id, Domain: domain, Code: code, Resource: resource, Classification: class, Repairability: "NOT_EVALUATED"})
}
func DatabaseReason(err error) string {
	for _, known := range []error{state.ErrDiagnosticMissing, state.ErrDiagnosticUnsafe, state.ErrDiagnosticBusy, state.ErrDiagnosticBound, state.ErrDiagnosticInvalid} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return "DIAGNOSTIC_DB_UNVERIFIED"
}
