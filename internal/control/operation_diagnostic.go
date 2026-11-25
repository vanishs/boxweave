package control

import (
	"context"
	"errors"

	"github.com/vanishs/boxweave/internal/state"
)

// Durable evidence only; missing evidence never proves non-execution.
type operationDiagnostic struct {
	EvidenceSource string           `json:"evidence_source"`
	WaitOutcome    string           `json:"wait_outcome"`
	CauseDomain    string           `json:"cause_domain"`
	NativeReceipt  string           `json:"native_receipt"`
	GuestOutcome   string           `json:"guest_outcome"`
	GuestExitCode  *int             `json:"guest_exit_code,omitempty"`
	NextAction     state.NextAction `json:"next_action"`
	StatusClass    string           `json:"status_class"`
	OutputMetadata string           `json:"output_metadata"`
	StdoutRecorded bool             `json:"stdout_recorded"`
	StderrRecorded bool             `json:"stderr_recorded"`
}

func operationTerminal(status string) bool {
	switch status {
	case "SUCCEEDED", "FAILED", "UNKNOWN", "CANCELLED", "TIMED_OUT":
		return true
	}
	return false
}

func diagnosticCause(r state.Receipt) string { return state.OperationCause(r) }

func buildOperationDiagnostic(r state.Receipt, waited bool) operationDiagnostic {
	d := operationDiagnostic{EvidenceSource: "SQLITE_DURABLE", WaitOutcome: "SNAPSHOT", CauseDomain: diagnosticCause(r), NativeReceipt: "NOT_RECORDED", GuestOutcome: "NOT_APPLICABLE", OutputMetadata: "NOT_APPLICABLE"}
	d.StatusClass = "IN_PROGRESS"
	d.NextAction = state.NextAction{Kind: "READ_OPERATION", Tool: "operation_get", OperationID: r.ID}
	if r.State == "UNKNOWN" {
		d.StatusClass = "EFFECT_UNVERIFIED"
		d.NextAction = state.NextAction{Kind: "INSPECT_RECOVERY", Tool: "operation_recover", OperationID: r.ID, Mode: "inspect"}
	} else if operationTerminal(r.State) {
		d.StatusClass = "TERMINAL"
		d.NextAction = r.NextAction
		if d.NextAction.Kind == "" {
			d.NextAction = state.NextAction{Kind: "NONE"}
		}
		if d.CauseDomain == "CAPACITY" {
			d.StatusClass = "CAPACITY"
			d.NextAction = statusNext()
		}
		if r.State == "FAILED" && d.CauseDomain != "CAPACITY" {
			d.StatusClass = "FAILED"
			d.NextAction = state.NextAction{Kind: "READ_EVIDENCE", Tool: "box_activity_get", OperationID: r.ID, View: "evidence"}
		}
	}
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
