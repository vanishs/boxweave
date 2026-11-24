package diagnostic

import (
	"encoding/json"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/state"
	"strings"
	"testing"
)

func TestPR177D2EvidenceClassesAndRedaction(t *testing.T) {
	r := NewReport("user-1001")
	l := state.DiagnosticLedger{HoldsObserved: true, Activity: state.ActivityResult{Issues: []state.ActivityIssue{
		{Reason: "PEER_ACTIVE_DEPENDENCY", Disposition: "ACTIVE_DEPENDENCY_NOT_ERROR", ResourceRef: "private-ip"},
		{Reason: "PROJECT_PEER_BARRIER", Disposition: "MCP_RECOVERY_INSPECTION_REQUIRED", ResourceRef: "private-guard", OperationID: strings.Repeat("a", 32)},
		{Reason: "UNRESOLVED_OPERATION", Disposition: "IN_PROGRESS_OR_STALLED", ResourceRef: "private-name"},
	}, History: []state.ActivityIssue{{Reason: "HISTORICAL_FAILED_OPERATION", Disposition: "TERMINAL_FAILED_NOT_ACTIVE", ResourceRef: "private-history"}}}}
	r.ClassifyLedger(l)
	if len(r.Issues) != 3 {
		t.Fatal(r.Issues)
	}
	classes := map[string]Issue{}
	for _, i := range r.Issues {
		classes[i.Code] = i
	}
	if classes["PROJECT_PEER_BARRIER"].Repairability != "MCP_INSPECT_REQUIRED" || classes["UNRESOLVED_OPERATION"].Classification != "IN_PROGRESS" || classes["HISTORICAL_FAILED_OPERATION"].Classification != "HISTORICAL" {
		t.Fatal(classes)
	}
	r.Installation = install.DiagnosticInstallation{Candidate: "SCOPED_V2", Runtime: "VERIFIED", ServiceUnit: "ACTIVE", LifecycleUnit: "TRANSITIONING", GuardUnit: "ACTIVE"}
	r.ClassifyInstallation()
	if len(r.Issues) != 4 {
		t.Fatal(r.Issues)
	}
	raw, e := json.Marshal(r)
	if e != nil || strings.Contains(string(raw), "private-") || strings.Contains(string(raw), "raw_intent") {
		t.Fatal("raw ref leaked", e)
	}
	if r.PacketReachability.State != "NOT_CHECKED" || r.Tunnel.State != "NOT_CHECKED" {
		t.Fatal("external proof fabricated")
	}
}
