package control

import (
	"context"
	"errors"
	"strings"

	"github.com/vanishs/boxweave/internal/state"
)

// Durable evidence only; missing evidence never proves non-execution.
type operationDiagnostic struct {
	EvidenceSource string `json:"evidence_source"`
	WaitOutcome    string `json:"wait_outcome"`
	CauseDomain    string `json:"cause_domain"`
	NativeReceipt  string `json:"native_receipt"`
	GuestOutcome   string `json:"guest_outcome"`
	GuestExitCode  *int   `json:"guest_exit_code,omitempty"`
	OutputMetadata string `json:"output_metadata"`
	StdoutRecorded bool   `json:"stdout_recorded"`
	StderrRecorded bool   `json:"stderr_recorded"`
}

func operationTerminal(status string) bool {
	switch status {
	case "SUCCEEDED", "FAILED", "UNKNOWN", "CANCELLED", "TIMED_OUT":
		return true
	}
	return false
}

func diagnosticCause(r state.Receipt) string {
	if r.ErrorClass == "" {
		if r.State == "UNKNOWN" || r.State == "FAILED" {
			return "UNCLASSIFIED"
		}
		return "NONE_RECORDED"
	}
	switch r.ErrorClass {
	case "NATIVE_ACK_DURABILITY_UNVERIFIED", "NATIVE_RECEIPT_RECORDING_UNVERIFIED", "FINAL_RECEIPT_DURABILITY_UNVERIFIED", "ATOMIC_EXEC_TERMINAL_UNVERIFIED", "ATOMIC_CREATE_TERMINAL_UNVERIFIED", "ATOMIC_DELETE_TERMINAL_UNVERIFIED", "ATOMIC_FORWARD_TERMINAL_UNVERIFIED", "EXEC_SQLITE_FINALIZATION_UNVERIFIED":
		return "SQLITE_DURABILITY"
	case "EXEC_NATIVE_RECEIPT_MISSING", "EXEC_NATIVE_OBSERVATION_DEADLINE", "EXEC_NATIVE_READ_UNVERIFIED", "NATIVE_OUTCOME_AMBIGUOUS", "NATIVE_RECEIPT_WITH_EFFECT_ERROR", "NO_NATIVE_OPERATION_RECEIPT":
		return "NATIVE_OBSERVATION"
	case "EXEC_NATIVE_IDENTITY_UNVERIFIED", "EXEC_TERMINAL_EVIDENCE_UNVERIFIED", "POSTCONDITION_UNVERIFIED", "NOOP_POSTCONDITION_UNVERIFIED":
		return "POSTCONDITION"
	case "NATIVE_EXEC_SPOOL_QUOTA_EXHAUSTED_PRESTART":
		return "OUTPUT_SPOOL"
	}
	if strings.HasPrefix(r.ErrorClass, "NATIVE_CREATE_") || strings.HasPrefix(r.ErrorClass, "DELETE_DEPENDENCY_") {
		return "POSTCONDITION"
	}
	return "UNCLASSIFIED"
}

func buildOperationDiagnostic(r state.Receipt, waited bool) operationDiagnostic {
	d := operationDiagnostic{EvidenceSource: "SQLITE_DURABLE", WaitOutcome: "SNAPSHOT", CauseDomain: diagnosticCause(r), NativeReceipt: "NOT_RECORDED", GuestOutcome: "NOT_APPLICABLE", OutputMetadata: "NOT_APPLICABLE"}
	if operationTerminal(r.State) {
		d.WaitOutcome = "TERMINAL"
	} else if waited {
		d.WaitOutcome = "WAIT_BUDGET_EXPIRED"
	}
	if r.NativeOperationID != "" {
		d.NativeReceipt = "RECORDED"
	}
	if r.Action == "box_exec" {
		d.GuestOutcome, d.OutputMetadata = "UNPROVED", "UNPROVED"
	}
	return d
}

func (s *Service) readOperationDiagnostic(ctx context.Context, r state.Receipt, waited bool) operationDiagnostic {
	d := buildOperationDiagnostic(r, waited)
	if r.Action != "box_exec" || r.State != "SUCCEEDED" {
		return d
	}
	// Terminal metadata and SUCCEEDED are atomically finalized. Never infer
	// completion of an UNKNOWN from a partial or later output row.
	out, err := s.Store.ExecOutput(ctx, r.ID)
	if errors.Is(err, state.ErrNotFound) {
		d.OutputMetadata = "NOT_RECORDED"
		return d
	}
	if err != nil {
		d.OutputMetadata = "SQLITE_READ_UNVERIFIED"
		return d
	}
	d.StdoutRecorded, d.StderrRecorded = out.StdoutPath != "", out.StderrPath != ""
	d.OutputMetadata = "RECORDED_NOT_LIVE_VERIFIED"
	if out.ExitCode != nil {
		d.GuestExitCode = out.ExitCode
		d.GuestOutcome = "EXIT_NONZERO"
		if *out.ExitCode == 0 {
			d.GuestOutcome = "EXIT_ZERO"
		}
	}
	return d
}
