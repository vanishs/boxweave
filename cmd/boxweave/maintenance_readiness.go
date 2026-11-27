package main

import (
	"github.com/vanishs/boxweave/internal/state"
	"sort"
	"strings"
)

// This projection uses exactly H1's typed observations, no new scanner or
// mutation authority. The coordinator additionally verifies local inventory,
// archive provenance, full effect set, approval and post-fence proof.
type maintenanceBlocker struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Resource string `json:"resource"`
	Next     string `json:"next_action"`
}
type maintenanceReadiness struct {
	Installation   string               `json:"installation"`
	Retirement     string               `json:"retirement"`
	Purge          string               `json:"purge"`
	Route          string               `json:"route"`
	Blockers       []maintenanceBlocker `json:"observation_blockers"`
	Complete       bool                 `json:"effect_set_complete"`
	ApplyAvailable bool                 `json:"apply_available"`
}

func summarizeMaintenance(o maintenanceReport) maintenanceReadiness {
	r := maintenanceReadiness{Installation: "UNVERIFIED", Retirement: "REVIEW_ONLY", Purge: "SEPARATE_COMPLETED_RETIREMENT_REVIEW", Route: "NORMAL_CLEANUP_REVIEW", Blockers: []maintenanceBlocker{}}
	i, n, l := o.Installation, o.Native, o.Ledger
	add := func(code, severity, resource, next string) {
		r.Blockers = append(r.Blockers, maintenanceBlocker{code, severity, resource, next})
		if severity == "HARD" {
			r.Retirement = "BLOCKED"
		}
	}
	hard := func(code, resource, next string) { add(code, "HARD", resource, next) }
	offline := i.Identity.Runtime != "VERIFIED"
	if i.Adapter != "PINNED_RC13" && i.Adapter != "PINNED_RC14" && i.Adapter != "EXACT_RUNNING_OBSERVER_BINARY" {
		hard("INSTALLATION_IDENTITY_UNVERIFIED", "installation", "Inspect exact manifest and binary; do not adopt")
	} else if i.Identity.Runtime == "VERIFIED" {
		r.Installation = "RUNTIME_OBSERVED"
	} else {
		r.Installation = "STATIC_IDENTITY_ONLY"
	}
	if i.Mode != "SCOPED_V2" || i.Identity.Mode == "CONFLICT" {
		r.Installation = "UNSUPPORTED_OR_UNVERIFIED"
		hard("CURRENT_SCOPED_INSTALLATION_REQUIRED", "installation", "Preserve historical or mixed artifacts; use the exact original reviewed lifecycle")
	}
	if i.Preimage.State != "OBSERVED" {
		hard("PREIMAGE_UNVERIFIED", "native_preimage", "Preserve original preparation journal; no reconstructed baseline")
	}
	if o.Coverage != "BOUNDED_SOURCES_OBSERVED" {
		hard("SOURCE_COVERAGE_INCOMPLETE", "observation", "Repeat read-only observation; missing is not empty")
	}
	if o.Database.State != "OBSERVED" || l == nil {
		hard("LEDGER_UNVERIFIED", "ledger", "Inspect durable original ledger without starting controller")
		l = &state.MaintenanceLedger{}
	} else if len(o.LedgerSHA256) != 64 || strings.Trim(o.LedgerSHA256, "0123456789abcdef") != "" {
		hard("LEDGER_WITNESS_MISSING", "ledger", "Use H1B observation adapter")
	}
	if n.Evidence.State != "OBSERVED" {
		hard("NATIVE_INVENTORY_UNVERIFIED", "native", "Inspect Incus locally; never start it for preview")
	}
	if l.ForeignRows != 0 {
		hard("FOREIGN_LEDGER_SCOPE", "ledger", "Preserve foreign rows; do not adopt or remove")
	}
	if n.Unowned != 0 || len(n.OwnedUnverified) > 0 || n.OwnershipComparison != "OBSERVED" {
		hard("NATIVE_IDENTITY_CONFLICT", "native", "Resolve exact UUID/incarnation and original Create evidence")
	}
	if l.OrphanBindings != 0 || l.OrphanFences != 0 {
		hard("ORPHAN_NETWORK_EVIDENCE", "ledger", "Inspect original binding and fence lineage")
	}
	if n.ActiveOperations != 0 {
		offline = true
		add("NATIVE_OPERATIONS_ACTIVE", "REVIEW", "native", "Drain and independently verify quiescence in a later reviewed phase")
	}
	if n.OtherProjects != 0 {
		add("OTHER_PROJECTS_PRESERVED", "INFO", "native", "Preserve unrelated Projects; fresh-install prerequisites are separate")
	}
	for _, x := range []struct{ key, value string }{{"project_config", n.ProjectConfig}, {"profile_config", n.ProfileConfig}} {
		if x.value != "BEFORE" && x.value != "AFTER" {
			hard("NATIVE_CONFIG_UNVERIFIED", x.key, "Compare recorded preimage; no overwrite of drift")
		}
	}
	if n.BridgeConfig != "MANAGED_BRIDGE_OBSERVED" && n.BridgeConfig != "RECORDED_PREIMAGE_MATCH" {
		hard("BRIDGE_CONFIG_UNVERIFIED", "bridge", "Preserve bridge and verify exact native configuration")
	}
	for _, x := range []struct{ key, value string }{{"bridge_consumers", n.BridgeConsumers}, {"base_acl_consumers", n.BaseACLConsumers}} {
		if x.value != "OWNED_REFERENCES_ONLY" && x.value != "NO_REFERENCES" && x.value != "NOT_APPLICABLE" {
			hard("FOREIGN_OR_UNVERIFIED_CONSUMER", x.key, "Inspect references; never alter foreign resources")
		}
	}
	if n.BaseACLConfig != "RECORDED_PREIMAGE_MATCH" && n.BaseACLConfig != "ABSENT" || n.BaseACLConfig == "ABSENT" && n.ProfileConfig == "AFTER" {
		hard("BASE_ACL_DELTA_UNVERIFIED", "base_acl", "Verify exact managed delta before restoration")
	}
	if n.BoxACLReferences != "OBSERVED" {
		hard("BOX_ACL_UNVERIFIED", "box_acl", "Verify per-Box ACL ownership and consumer evidence")
	}
	if n.ForwardReferences != "OBSERVED" {
		hard("FORWARD_DEVICE_UNVERIFIED", "forward", "Verify exact proxy identity and endpoints")
	}
	if i.Mode == "SCOPED_V2" && (i.RootPolicy.Kernel != "OBSERVED" || o.RootLedgerComparison != "MATCH") {
		hard("ROOT_POLICY_UNVERIFIED", "scoped_guard", "Preserve guard and original journal; do not replay")
	}
	if i.Identity.Runtime != "VERIFIED" {
		add("CONTROLLER_RUNTIME_UNVERIFIED", "REVIEW", "controller", "Use offline retirement review, not a forced restart")
	}
	present := map[string]bool{}
	for _, a := range i.ArtifactEvidence {
		if a.Evidence.State == "PRESENT" {
			present[a.ID] = true
		} else if a.Evidence.State != "ABSENT" {
			hard("ARTIFACT_UNSAFE_OR_UNVERIFIED", a.ID, "Inspect fixed path metadata and ownership")
		}
	}
	for _, u := range i.Units {
		if i.Mode == "SCOPED_V2" && u.DropIns == "KNOWN_LXC255_SCOPED_RETIREMENT_REVIEW_REQUIRED" && o.Coverage == "BOUNDED_SOURCES_OBSERVED" && o.Database.State == "OBSERVED" {
			add("LXC_SCOPED_RETIREMENT_REVIEW_REQUIRED", "REVIEW", u.Name, "Use exact scoped uninstall plan; verify inherited policy and approved original scope; no runtime hardening approval")
		} else if u.DropIns != "ABSENT" {
			hard("EFFECTIVE_DROPIN_UNREVIEWED", u.Name, "Review effective systemd configuration; do not remove inherited policy")
		}
		if u.EffectiveFile.State != "OBSERVED" && u.EffectiveFile.State != "ABSENT" {
			hard("EFFECTIVE_UNIT_UNVERIFIED", u.Name, "Verify exact installed fragment")
		}
	}
	legacy, scoped, missingScoped, legacyRoot := false, false, false, false
	for _, id := range []string{"legacy_preparation", "legacy_root_unit", "legacy_retirement_unit", "legacy_root_socket", "legacy_retirement_socket", "legacy_root_operations", "legacy_root_lock", "legacy_root_journal"} {
		legacy = legacy || present[id]
	}
	for _, id := range []string{"scoped_root_journal", "scoped_guard_unit", "scoped_lifecycle_unit", "scoped_incus_dropin", "scoped_lifecycle_socket"} {
		scoped = scoped || present[id]
		if id != "scoped_lifecycle_socket" && !present[id] {
			missingScoped = true
		}
	}
	for _, id := range []string{"legacy_root_unit", "legacy_root_socket", "legacy_root_operations", "legacy_root_journal"} {
		legacyRoot = legacyRoot || present[id]
	}
	if legacy && scoped || i.Mode == "SCOPED_V2" && legacy || i.Mode == "LEGACY" && scoped {
		hard("MIXED_INSTALLATION_ARTIFACTS", "installation", "Retain both original witnesses; no cross-mode adoption")
	}
	if i.Mode == "SCOPED_V2" && missingScoped {
		hard("SCOPED_ARTIFACTS_INCOMPLETE", "installation", "Verify exact installed guard and lifecycle contract")
	}
	if legacyRoot {
		offline = true
		hard("LEGACY_ROOT_ORIGINAL_REVIEW_REQUIRED", "legacy_root", "Verify exact original stage, unit and journal; preserve UNKNOWN")
	}
	if i.Mode == "LEGACY" {
		for _, id := range l.LegacyPending {
			offline = true
			add("LEGACY_RETIREMENT_QUEUE_PRESENT", "REVIEW", id, "Preserve exact original Delete evidence and verify retirement")
		}
	}
	for _, id := range l.LegacyCompactions {
		offline = true
		hard("LEGACY_COMPACTION_PENDING", id, "Verify original Root compaction; preserve journal and fences")
	}
	for _, x := range l.Originals {
		offline = true
		code, severity := "ORIGINAL_IN_FLIGHT", "REVIEW"
		if x.State == "UNKNOWN" {
			code = "ORIGINAL_UNKNOWN"
		}
		if x.Action == "box_create" {
			severity = "HARD"
		}
		add(code, severity, x.ID, "Preserve original receipt and claims; review exact native outcome without replay")
	}
	boxes := map[[2]string]bool{}
	matched := map[string]bool{}
	for _, id := range n.OwnedMatched {
		matched[id] = true
	}
	for _, b := range l.Boxes {
		boxes[[2]string{b.UUID, b.Incarnation}] = true
		if !matched[b.UUID] || !b.CreateProof || b.BindingState != "PINNED" {
			hard("BOX_ORIGIN_UNVERIFIED", b.UUID, "Inspect original Create, binding and native identity")
		}
	}
	for _, p := range l.Peers {
		if p.State == "REVOKED" {
			continue
		}
		if !boxes[[2]string{p.Source, p.SourceIncarnation}] || !boxes[[2]string{p.Target, p.TargetIncarnation}] || !matched[p.Source] || !matched[p.Target] || p.State != "CONFIRMED" || p.LastVerifiedOriginal == "" {
			offline = true
			id := p.Original
			if id == "" {
				id = p.Source
			}
			add("PEER_ORIGINAL_UNVERIFIED", "REVIEW", id, "Review original directional policy; no replay")
		}
	}
	for _, f := range l.Forwards {
		if f.State == "REVOKED" {
			continue
		}
		if !boxes[[2]string{f.Box, f.Incarnation}] || !matched[f.Box] || f.State != "CONFIRMED" || n.ForwardReferences != "OBSERVED" {
			offline = true
			add("FORWARD_ORIGINAL_UNVERIFIED", "REVIEW", f.ID, "Review original proxy without redispatch")
		}
	}
	if offline {
		r.Route = "OFFLINE_RETIREMENT_REVIEW"
	}
	rank := map[string]int{"HARD": 0, "REVIEW": 1, "INFO": 2}
	sort.Slice(r.Blockers, func(a, b int) bool {
		x, y := r.Blockers[a], r.Blockers[b]
		if x.Severity != y.Severity {
			return rank[x.Severity] < rank[y.Severity]
		}
		if x.Code != y.Code {
			return x.Code < y.Code
		}
		return x.Resource < y.Resource
	})
	return r
}
