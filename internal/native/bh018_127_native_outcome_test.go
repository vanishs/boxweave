package native

import (
	"errors"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

// BH-018 2026-11-14: genuine Incus 7.6 native Exec tests found that both an
// absent executable and an *existing shell that wrote a file before exit 127*
// returned Failure / "Command not found" / metadata.return=127.
// The latter Guest side effect was independently read back afterward.
// That native signature cannot prove absence of Guest effects and must not
// authorize a FAILED-as-predispatch result or safe retry.
func TestBH018NativeFailureWithReturn127NeverProvesNoGuestEffect(t *testing.T) {
	path := "/1.0/instances/owned-guest/logs/exec-output/exec_original.stdout"
	for _, tc := range []struct {
		name string
		err  string
	}{
		{name: "nonexistent-executable", err: "Command not found"},
		{name: "existing-shell-wrote-file-then-exited-127", err: "Command not found"},
		{name: "unknown-error-with-code-127", err: "opaque native failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := &api.Operation{
				StatusCode: api.Failure,
				Err:        tc.err,
				Metadata: map[string]any{
					"return": float64(127),
					"output": map[string]any{"1": path},
				},
			}
			ev, err := ExecEvidenceFromFinal(op)
			if !errors.Is(err, ErrUnverified) {
				t.Fatalf("failed native Operation must remain unverified, got %v", err)
			}
			if ev.Exit != nil || ev.StdoutPath != "" || ev.StderrPath != "" {
				t.Fatalf("unsafe success evidence extracted from native failure: %+v", ev)
			}
		})
	}
}

// Unlike a failed native Operation with return=127, a successful native Exec
// with an observed nonzero Guest exit records the exit code, not a false
// startup failure. The native case was /bin/false -> Success / return=1.
func TestBH018SuccessfulNativeExecMayHaveNonzeroGuestExit(t *testing.T) {
	op := &api.Operation{
		StatusCode: api.Success,
		Metadata: map[string]any{
			"return": float64(1),
			"output": map[string]any{
				"1": "/1.0/instances/owned-guest/logs/exec-output/exec_original.stdout",
				"2": "/1.0/instances/owned-guest/logs/exec-output/exec_original.stderr",
			},
		},
	}
	ev, err := ExecEvidenceFromFinal(op)
	if err != nil {
		t.Fatalf("native success with nonzero exit must be readable: %v", err)
	}
	if ev.Exit == nil || *ev.Exit != 1 || ev.StdoutPath == "" || ev.StderrPath == "" {
		t.Fatalf("incorrect verified nonzero Guest exit: %+v", ev)
	}
}
