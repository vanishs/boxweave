package control

import (
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func TestS5A1OperationDiagnosticDoesNotInferOutcome(t *testing.T) {
	for _, tc := range []struct {
		state, cause, native, wantDomain, wantWait, wantNative string
		waited                                                 bool
	}{
		{"ADMITTED", "", "", "NONE_RECORDED", "SNAPSHOT", "NOT_RECORDED", false},
		{"VERIFYING", "", "id", "NONE_RECORDED", "WAIT_BUDGET_EXPIRED", "RECORDED", true},
		{"UNKNOWN", "EXEC_NATIVE_RECEIPT_MISSING", "id", "NATIVE_OBSERVATION", "TERMINAL", "RECORDED", true},
		{"UNKNOWN", "EXEC_SQLITE_FINALIZATION_UNVERIFIED", "id", "SQLITE_DURABILITY", "TERMINAL", "RECORDED", false},
		{"UNKNOWN", "EXEC_TERMINAL_EVIDENCE_UNVERIFIED", "id", "POSTCONDITION", "TERMINAL", "RECORDED", false},
		{"FAILED", "NATIVE_EXEC_SPOOL_QUOTA_EXHAUSTED_PRESTART", "id", "OUTPUT_SPOOL", "TERMINAL", "RECORDED", false},
		{"SUCCEEDED", "", "id", "NONE_RECORDED", "TERMINAL", "RECORDED", false},
		{"UNKNOWN", "unknown old error", "", "UNCLASSIFIED", "TERMINAL", "NOT_RECORDED", false},
	} {
		t.Run(tc.state+"/"+tc.cause, func(t *testing.T) {
			d := buildOperationDiagnostic(state.Receipt{Action: "box_exec", State: tc.state, ErrorClass: tc.cause, NativeOperationID: tc.native}, tc.waited)
			if d.EvidenceSource != "SQLITE_DURABLE" || d.CauseDomain != tc.wantDomain || d.WaitOutcome != tc.wantWait || d.NativeReceipt != tc.wantNative || d.GuestExitCode != nil || d.GuestOutcome != "UNPROVED" || d.OutputMetadata != "UNPROVED" {
				t.Fatalf("unsafe diagnostic inference: %+v", d)
			}
		})
	}
}
