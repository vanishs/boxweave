package control

import (
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

// Characterize mapping coverage, not a desired permanent UNCLASSIFIED result.
func TestPR177D0ExtensionScopedCauseCoverage(t *testing.T) {
	for _, tc := range []struct{ phase, reason, cause string }{
		{"FAILED", "SCOPED_PEER_V2_HISTORY_CAPACITY", "CAPACITY"},
		{"FAILED", "SCOPED_PEER_V2_PRE_DISPATCH_REFUSED", "ADMISSION_PRECONDITION"},
		{"UNKNOWN", "SCOPED_PEER_V2_ROOT_OUTCOME_UNVERIFIED", "ROOT_OUTCOME_UNVERIFIED"},
		{"FAILED", "RECOVERY_BLOCKED", "RECOVERY_EVIDENCE"},
	} {
		d := buildOperationDiagnostic(state.Receipt{State: tc.phase, Action: "box_peer_set", ErrorClass: tc.reason}, false)
		if d.CauseDomain != tc.cause || d.NativeReceipt != "NOT_RECORDED" || d.GuestOutcome != "NOT_APPLICABLE" {
			t.Fatal(d)
		}
		t.Logf("D0X_CAUSE_COVERAGE phase=%s reason=%s cause=%s", tc.phase, tc.reason, d.CauseDomain)
	}
}
