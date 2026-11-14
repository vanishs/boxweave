package native

import (
	"errors"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

// These terminal shapes were verified against native Incus 7.6, with
// independent Guest-file side-effect readback for BOTH 126 and 127.
// Incus's lxcCmd.Wait maps the actual exit status to a misleading native
// error, irrespective of whether the command was actually launched.
// This proves only that the original native Exec has ended. It never
// establishes no side effects, nor permits any automatic command replay.
func TestBH018NativeSpecialExitIsFinishedButNotGuestSuccess(t *testing.T) {
	const id = "bdf9c7b6-384d-4ac0-a30c-fd78308a5b90"
	const name = "owned-guest"
	prefix := "/1.0/instances/" + name + "/logs/exec-output/exec_" + id
	for _, tc := range []struct {
		name string
		err  string
		exit int
	}{
		{name: "missing-executable-return-127", err: "Command not found", exit: 127},
		{name: "ran-wrote-file-return-127", err: "Command not found", exit: 127},
		{name: "ran-wrote-file-return-126", err: "Command not executable", exit: 126},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := &api.Operation{ID: id, StatusCode: api.Failure, Err: tc.err,
				Metadata: map[string]any{
					"return": float64(tc.exit),
					"output": map[string]any{"1": prefix + ".stdout", "2": prefix + ".stderr"},
				}}
			ev, err := ExecEvidenceFromFinal(op)
			if err != nil || ev.Exit == nil || *ev.Exit != tc.exit {
				t.Fatalf("lost verified native terminal guest exit: ev=%+v err=%v", ev, err)
			}
			if !ExecEvidenceMatchesOriginal(ev, name, id) {
				t.Fatalf("rejected real original recorded guest output: %+v", ev)
			}
			if *ev.Exit == 0 {
				t.Fatal("native operation completion falsely means guest command success")
			}
		})
	}
}

// Native Failure is still an UNKNOWN/unverified receipt unless an exact
// source-backed native special terminal signature and bounded output survive.
// A fabricated terminal state may NEVER be used to release a resource claim.
func TestBH018OtherNativeFailureRemainsUnknown(t *testing.T) {
	const path = "/1.0/instances/owned-guest/logs/exec-output/exec_deadbeef.stdout"
	for _, tc := range []struct {
		name   string
		status api.StatusCode
		err    string
		exit   any
		output any
	}{
		{"unknown-error-127", api.Failure, "opaque native failure", float64(127), map[string]any{"1": path}},
		{"mismatched-error-126", api.Failure, "Command not found", float64(126), map[string]any{"1": path}},
		{"mismatched-error-127", api.Failure, "Command not executable", float64(127), map[string]any{"1": path}},
		{"generic-failure-1", api.Failure, "", float64(1), map[string]any{"1": path}},
		{"native-still-running", api.Running, "Command not found", float64(127), map[string]any{"1": path}},
		{"missing-guest-exit", api.Failure, "Command not found", nil, map[string]any{"1": path}},
		{"invalid-guest-exit", api.Failure, "Command not found", "127", map[string]any{"1": path}},
		{"missing-original-logs", api.Failure, "Command not found", float64(127), nil},
		{"foreign-output-url", api.Failure, "Command not found", float64(127), map[string]any{"1": "https://evil.test/etc/shadow"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := &api.Operation{StatusCode: tc.status, Err: tc.err,
				Metadata: map[string]any{"return": tc.exit, "output": tc.output}}
			ev, err := ExecEvidenceFromFinal(op)
			if !errors.Is(err, ErrUnverified) {
				t.Fatalf("ambiguous native failure treated as complete: %+v, %v", ev, err)
			}
			if ev.Exit != nil || ev.StdoutPath != "" || ev.StderrPath != "" {
				t.Fatalf("leaked fake terminal evidence: %+v", ev)
			}
		})
	}
}

func TestBH018NativeExecOutputBoundToOriginalBoxAndOperation(t *testing.T) {
	const id = "bdf9c7b6-384d-4ac0-a30c-fd78308a5b90"
	const name = "owned-guest"
	prefix := "/1.0/instances/" + name + "/logs/exec-output/exec_" + id
	code := 127
	ok := ExecEvidence{Exit: &code, StdoutPath: prefix + ".stdout", StderrPath: prefix + ".stderr"}
	if !ExecEvidenceMatchesOriginal(ok, name, id) {
		t.Fatal("actual matching evidence rejected")
	}
	for _, tc := range []struct {
		name string
		ev   ExecEvidence
		box  string
		id   string
	}{
		{"other-box", ok, "different-box", id},
		{"other-operation", ok, name, "00000000-0000-0000-0000-000000000000"},
		{"foreign-output", ExecEvidence{Exit: &code, StdoutPath: "/1.0/instances/foreign/logs/exec-output/exec_" + id + ".stdout"}, name, id},
		{"wrong-native-id-filename", ExecEvidence{Exit: &code, StderrPath: prefix + "X.stderr"}, name, id},
		{"wrong-stream", ExecEvidence{Exit: &code, StdoutPath: prefix + ".stderr"}, name, id},
		{"invalid-guest", ok, "../owned-guest", id},
		{"invalid-native-id", ok, name, "wrong/id"},
		{"missing-output", ExecEvidence{Exit: &code}, name, id},
		{"missing-exit", ExecEvidence{StdoutPath: prefix + ".stdout"}, name, id},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if ExecEvidenceMatchesOriginal(tc.ev, tc.box, tc.id) {
				t.Fatalf("foreign/ambiguous native Exec admitted: %+v", tc)
			}
		})
	}
}

func TestBH018SuccessfulNativeExecMayHaveNonzeroGuestExit(t *testing.T) {
	const id = "bdf9c7b6-384d-4ac0-a30c-fd78308a5b90"
	prefix := "/1.0/instances/owned-guest/logs/exec-output/exec_" + id
	op := &api.Operation{ID: id, StatusCode: api.Success, Metadata: map[string]any{
		"return": float64(1),
		"output": map[string]any{"1": prefix + ".stdout", "2": prefix + ".stderr"},
	}}
	ev, err := ExecEvidenceFromFinal(op)
	if err != nil || ev.Exit == nil || *ev.Exit != 1 || !ExecEvidenceMatchesOriginal(ev, "owned-guest", id) {
		t.Fatalf("verified native success with guest /bin/false exit lost: %+v %v", ev, err)
	}
}
