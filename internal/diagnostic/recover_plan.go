package diagnostic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/vanishs/boxweave/internal/recovery"
	"github.com/vanishs/boxweave/internal/state"
)

const RecoverPlanVersion = 1
const RecoverPlanTTL = 2 * time.Minute

// RecoverPlan is a local reviewed observation, not a credential,
// approval, idempotency key or executable request. D3 never applies it.
type RecoverPlan struct {
	Version                  int               `json:"version"`
	IssueID                  string            `json:"issue_id"`
	Scope                    string            `json:"scope"`
	Mode                     string            `json:"mode"`
	IssuedNS                 int64             `json:"issued_ns"`
	ExpiresNS                int64             `json:"expires_ns,omitempty"`
	Status                   string            `json:"status"`
	Reason                   string            `json:"reason"`
	Issue                    *Issue            `json:"issue,omitempty"`
	OriginalID               string            `json:"original_id,omitempty"`
	OriginalAction           string            `json:"original_action,omitempty"`
	OriginalState            string            `json:"original_state,omitempty"`
	OriginalRevision         int64             `json:"original_revision,omitempty"`
	BoxID                    string            `json:"box_id,omitempty"`
	Incarnation              string            `json:"incarnation,omitempty"`
	ClaimCount               int               `json:"claim_count"`
	Resources                []string          `json:"affected_resources"`
	ObserverUID              int               `json:"observer_uid"`
	OriginalCredentialActive *bool             `json:"original_credential_active,omitempty"`
	InstallationSHA256       string            `json:"installation_sha256,omitempty"`
	OriginalSHA256           string            `json:"original_sha256,omitempty"`
	LedgerSHA256             string            `json:"ledger_sha256,omitempty"`
	EvidenceEpochNS          int64             `json:"evidence_epoch_ns,omitempty"`
	EffectiveForwardPolicy   string            `json:"effective_forward_policy,omitempty"`
	AffectedBoxes            []PlanBoxIdentity `json:"affected_boxes,omitempty"`

	Witness           *recovery.PlanWitness `json:"witness,omitempty"`
	Leaf              string                `json:"leaf"`
	Route             string                `json:"route"`
	Authority         string                `json:"required_authority"`
	Effects           []string              `json:"planned_effects"`
	Preconditions     []string              `json:"blocking_preconditions"`
	Postconditions    []string              `json:"postconditions"`
	Inspect           *PlanMCPInspect       `json:"mcp_inspect,omitempty"`
	CanRepair         bool                  `json:"can_repair"`
	ApplyAvailable    bool                  `json:"apply_available"`
	NativeRetried     bool                  `json:"native_retried"`
	CrossSourceAtomic bool                  `json:"cross_source_atomic"`
	ReviewSHA256      string                `json:"review_sha256,omitempty"`
	originalVerified  bool
}
type PlanBoxIdentity struct {
	Role        string `json:"role"`
	UUID        string `json:"uuid"`
	Incarnation string `json:"incarnation"`
}
type PlanMCPInspect struct {
	Tool      string `json:"tool"`
	Arguments struct {
		OperationID string `json:"operation_id"`
		Mode        string `json:"mode"`
	} `json:"arguments"`
}

func validSHA(s string) bool { return len(s) == 64 && strings.Trim(s, "0123456789abcdef") == "" }
func ValidIssueID(id string) bool {
	return strings.HasPrefix(id, "d1-") && validSHA(strings.TrimPrefix(id, "d1-"))
}
func validOriginalID(id string) bool {
	return len(id) == 32 && strings.Trim(id, "0123456789abcdef") == ""
}
func FindIssue(r Report, id string) (Issue, bool) {
	if !ValidIssueID(id) {
		return Issue{}, false
	}
	for _, i := range r.Issues {
		if i.ID == id && IssueID(r.Scope, i.Domain, i.Code, i.Resource) == id {
			return i, true
		}
	}
	return Issue{}, false
}
func NewRecoverPlan(r Report, id string, uid int, now time.Time) RecoverPlan {
	p := RecoverPlan{Version: RecoverPlanVersion, IssueID: id, Scope: r.Scope, Mode: r.Installation.Mode, IssuedNS: now.UnixNano(), Status: "BLOCKED_UNPROVEN", Reason: "ISSUE_NOT_PRESENT_OR_OUTSIDE_COVERAGE", ObserverUID: uid, Resources: []string{}, Effects: []string{}, Preconditions: []string{}, Postconditions: []string{}, Leaf: "NONE", Route: "RERUN_DOCTOR", Authority: "NONE"}
	i, ok := FindIssue(r, id)
	if !ok {
		p.Preconditions = []string{"FRESH_MATCHING_ISSUE_REQUIRED"}
		return p
	}
	p.Issue = &i
	p.InstallationSHA256 = r.Installation.IdentitySHA256
	p.Reason = "NO_SUPPORTED_AUTOMATIC_LEAF"
	p.Route = i.Repairability
	if i.Classification == "HISTORICAL" {
		p.Status = "HISTORICAL"
		p.Reason = "HISTORICAL_NOT_ACTIVE"
		p.Route = "NOT_APPLICABLE"
		return p
	}
	switch i.Repairability {
	case "NO_RESET_OR_COMPACTION", "NO_AUTOMATIC_ADDRESS_REUSE":
		p.Status = "UNSUPPORTED"
		p.Reason = "RETENTION_OR_CAPACITY_IS_NOT_REPAIRABLE_BY_RESET"
		return p
	case "LOCAL_CONFIGURATION_REVIEW", "LOCAL_DEPENDENCY_REVIEW", "INSPECT_INSTALLED_VERSION":
		p.Status = "OPERATOR_REVIEW_REQUIRED"
		p.Authority = "LOCAL_OPERATOR"
		return p
	case "HOST_REVIEW_REQUIRED", "INSPECT_ROOT_EVIDENCE", "INSPECT_LEGACY_RETIREMENT":
		p.Authority = "LOCAL_ROOT_REVIEW"
	}
	if i.Code == "EXEC_OUTPUT_GC_BLOCKER" {
		p.Status = "OPERATOR_REVIEW_REQUIRED"
		p.Route = "EXISTING_LOCAL_GC_REVIEW"
		p.Authority = "LOCAL_ROOT_REVIEW"
		p.Reason = "EXACT_STREAM_AND_EXISTING_GC_REVIEW_REQUIRED"
		return p
	}
	if validOriginalID(i.OriginalID) {
		p.OriginalID = i.OriginalID
		p.Inspect = &PlanMCPInspect{Tool: "operation_recover"}
		p.Inspect.Arguments.OperationID = i.OriginalID
		p.Inspect.Arguments.Mode = "inspect"
		p.Route = "MCP_INSPECT_REQUIRED"
		p.Authority = "AUTHORIZED_EXISTING_MCP_PRINCIPAL"
		p.Reason = "EXACT_ORIGINAL_PROOF_REQUIRED"
	}
	return p
}
func (p *RecoverPlan) Block(reason string) {
	p.Status = "BLOCKED_UNPROVEN"
	p.originalVerified = false
	p.Reason = reason
	p.CanRepair = false
	p.ReviewSHA256 = ""
	p.ExpiresNS = 0
	p.Leaf = "NONE"
	p.Effects = []string{}
	p.Postconditions = []string{}
	p.Preconditions = []string{reason}
}
func (p *RecoverPlan) SetOriginal(o state.DiagnosticOriginal) bool {
	t := o.Target
	if p.Issue == nil || p.OriginalID != t.ID || p.Scope != t.Layer || !validOriginalID(t.ID) || !validSHA(o.Digest) || p.Issue.BoxID != "" && p.Issue.BoxID != state.SafeDiagnosticRef(t.UUID) {
		p.Block("ORIGINAL_SCOPE_OR_IDENTITY_CHANGED")
		return false
	}
	if t.Action == "" || len(t.Action) > 64 || strings.Trim(t.Action, "abcdefghijklmnopqrstuvwxyz_") != "" || t.State == "" || len(t.State) > 32 || strings.Trim(t.State, "ABCDEFGHIJKLMNOPQRSTUVWXYZ_") != "" {
		p.Block("ORIGINAL_TYPE_UNVERIFIED")
		return false
	}
	p.OriginalAction = t.Action
	p.OriginalState = t.State
	p.OriginalRevision = t.UpdatedAt
	if t.UUID != "" {
		p.BoxID = state.SafeDiagnosticRef(t.UUID)
	}
	if t.Incarnation != "" {
		p.Incarnation = state.SafeDiagnosticRef(t.Incarnation)
	}
	if t.UUID != "" && t.Incarnation != "" {
		p.AffectedBoxes = append(p.AffectedBoxes, PlanBoxIdentity{"ORIGINAL", state.SafeDiagnosticRef(t.UUID), state.SafeDiagnosticRef(t.Incarnation)})
	}
	if t.ScopedPeer {
		var in state.ScopedPeerOriginalIntent
		if json.Unmarshal([]byte(t.SafeIntent), &in) != nil || in.Schema != 2 || in.Project != p.Scope || in.SourceUUID != t.UUID || in.SourceIncarnation != t.Incarnation || in.TargetUUID == "" || in.TargetIncarnation == "" || in.TargetUUID == in.SourceUUID {
			p.Block("SCOPED_PEER_INTENT_UNVERIFIED")
			return false
		}
		p.AffectedBoxes = append(p.AffectedBoxes, PlanBoxIdentity{"PEER_TARGET", state.SafeDiagnosticRef(in.TargetUUID), state.SafeDiagnosticRef(in.TargetIncarnation)})
	} else if t.Action == "box_peer_set" && t.PeerReady {
		in := t.Peer.Intent
		p.AffectedBoxes = append(p.AffectedBoxes, PlanBoxIdentity{"PEER_TARGET", state.SafeDiagnosticRef(in.TargetUUID), state.SafeDiagnosticRef(in.TargetIncarnation)})
	}
	p.OriginalSHA256 = o.Digest
	p.ClaimCount = len(o.Claims)
	v := o.CredentialActive
	p.OriginalCredentialActive = &v
	for _, k := range o.Claims {
		p.Resources = append(p.Resources, state.SafeDiagnosticRef(k))
	}
	if t.ScopedPeer && !o.CredentialActive {
		p.Block("SCOPED_ORIGINAL_CREDENTIAL_NOT_ACTIVE")
		return false
	}
	if t.Released {
		p.Block("ORIGINAL_REVIEWED_RELEASE_OR_NETWORK_REVIEW_REQUIRED")
		return false
	}
	if t.State != "UNKNOWN" && !(t.Action == "box_create" && t.State == "SUCCEEDED" && t.CreatePinRequired && t.CreatePinStaged) {
		p.Block("ORIGINAL_NOT_CURRENT_RECOVERY_TARGET")
		return false
	}

	if p.Mode != "SCOPED_V2" && p.Mode != "LEGACY" || !validSHA(p.InstallationSHA256) {
		p.Block("INSTALLED_ARCHITECTURE_OR_IDENTITY_UNVERIFIED")
		return false
	}
	if t.Action == "box_peer_set" && (t.ScopedPeer != (p.Mode == "SCOPED_V2")) {
		p.Block("ORIGINAL_ARCHITECTURE_MISMATCH")
		return false
	}
	if p.Mode == "SCOPED_V2" && (t.Action == "box_create" || t.Action == "box_delete") {
		p.Block("SCOPED_LIFECYCLE_RECOVERY_UNSUPPORTED")
		return false
	}
	p.originalVerified = true
	return true
}
func (p *RecoverPlan) SetWitness(w recovery.PlanWitness) {
	if !p.originalVerified || p.OriginalID == "" || w.OriginalID != p.OriginalID {
		p.Block("WITNESS_ORIGINAL_MISMATCH")
		return
	}
	p.Witness = &w
	if w.Kind == state.RecoveryNotNeeded {
		p.Status = "NOT_REQUIRED"
		p.Reason = w.Reason
		return
	}
	if w.Kind == state.RecoveryApproval {
		p.Status = "OPERATOR_REVIEW_REQUIRED"
		p.Reason = w.Reason
		p.Authority = "LOCAL_ROOT_REVIEW"
		return
	}
	if !validSHA(w.ProofSHA256) {
		p.Block(w.Reason)
		return
	}
	// Explicit supported leaves; no arbitrary command or generic release.
	switch w.Kind {
	case state.RecoverySettle:
		switch p.OriginalAction {
		case "box_start", "box_stop", "box_update", "box_exec", "box_create", "box_delete", "box_peer_set", "port_forward_create", "port_forward_delete":
			p.Leaf = "MCP_SETTLE_ORIGINAL_NO_REPLAY"
		default:
			p.Block("UNSUPPORTED_RECOVERY_LEAF")
			return
		}
	case state.RecoveryBind:
		if p.Mode != "LEGACY" || p.OriginalAction != "box_create" {
			p.Block("UNSUPPORTED_IDENTITY_BIND")
			return
		}
		p.Leaf = "MCP_BIND_ORIGINAL_IDENTITY"
	case state.RecoveryRepairPin:
		if p.Mode != "LEGACY" || p.OriginalAction != "box_create" {
			p.Block("UNSUPPORTED_PIN_REPAIR")
			return
		}
		p.Leaf = "MCP_REPAIR_ORIGINAL_PIN_LEDGER"
	default:
		p.Block("UNSUPPORTED_RECOVERY_LEAF")
		return
	}
	p.Status = "REVIEW_READY"
	p.Reason = "OBSERVED_SUPPORTED_LEAF_REQUIRES_FRESH_REVALIDATION"
	p.Route = "EXISTING_MCP_INSPECT_THEN_REVIEWED_RECONCILE"
	p.CanRepair = true
	p.Effects = []string{"ONLY_ORIGINAL_LEDGER_RECONCILIATION_VIA_EXISTING_RECOVERY"}
	p.Preconditions = []string{"FRESH_ORIGINAL_AND_NATIVE_PROOF_REQUIRED", "AUTHENTICATED_MCP_AUTHORITY_REQUIRED", "SEPARATE_RECOVERY_IDEMPOTENCY_KEY_REQUIRED", "D4_REVIEW_ADAPTER_NOT_IMPLEMENTED"}
	p.Postconditions = []string{"ORIGINAL_REOBSERVED", "EXACT_DURABLE_RECOVERY_RECEIPT", "NO_NATIVE_REPLAY", "UNRELATED_PROTECTION_UNCHANGED"}
	p.ExpiresNS = p.IssuedNS + RecoverPlanTTL.Nanoseconds()
	// Digest includes version, exact issue, installation, actor scope,
	// original revision and proof, bounded resources, leaf and review window.
	// It is never a signature or authorization.
	p.ReviewSHA256 = p.reviewDigest()
}
func (p RecoverPlan) reviewDigest() string {
	p.ReviewSHA256 = ""
	b, e := json.Marshal(p)
	if e != nil {
		return ""
	}
	h := sha256.Sum256(append([]byte("boxweave/recover-plan/v1\n"), b...))
	return hex.EncodeToString(h[:])
}

// ReviewMatches is pure structural verification for future D4. It does
// NOT verify current native state or authorize a caller.
func (p RecoverPlan) ReviewMatches(now time.Time) bool {
	return p.Version == RecoverPlanVersion && p.Status == "REVIEW_READY" && p.CanRepair && !p.ApplyAvailable && !p.NativeRetried && p.IssuedNS <= now.UnixNano() && now.UnixNano() < p.ExpiresNS && p.ExpiresNS == p.IssuedNS+RecoverPlanTTL.Nanoseconds() && validSHA(p.ReviewSHA256) && p.ReviewSHA256 == p.reviewDigest()
}
