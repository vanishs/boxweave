package control

import (
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

// Characterize mapping coverage, not a desired permanent UNCLASSIFIED result.
func TestPR177D0ExtensionScopedCauseCoverage(t *testing.T) {
	for _, tc := range []struct{ phase, reason string }{
		{"FAILED", "SCOPED_PEER_V2_HISTORY_CAPACITY"},
		{"FAILED", "SCOPED_PEER_V2_PRE_DISPATCH_REFUSED"},
		{"UNKNOWN", "SCOPED_PEER_V2_ROOT_OUTCOME_UNVERIFIED"},
		{"FAILED", "RECOVERY_BLOCKED"},
	} {
		d := buildOperationDiagnostic(state.Receipt{State: tc.phase, Action: "box_peer_set", ErrorClass: tc.reason}, false)
		if d.NativeReceipt != "NOT_RECORDED" || d.GuestOutcome != "NOT_APPLICABLE" {
			t.Fatal(d)
		}
		t.Logf("D0X_CAUSE_COVERAGE phase=%s reason=%s cause=%s", tc.phase, tc.reason, d.CauseDomain)
	}
}
