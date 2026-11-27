package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

// The COMMIT acknowledgement is lost to the caller after SQLite has actually
// committed. This injects the loss at the verifier boundary: it does not claim
// that the SQLite driver itself returned a lost network acknowledgement.
// Worker must not overwrite a committed terminal receipt with UNKNOWN or
// repeat the original guest effect.
func TestBH018WorkerLostFinalCommitAcknowledgementCannotOverwriteTerminal(t *testing.T) {
	ctx := context.Background()
	s, box, intent, db := bh018InjectedAtomicFixture(t)
	rec, err := s.AdmitScoped(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr := bh018InjectedPaths(box)
	effectCalls := 0
	verifierCalls := 0
	Worker{Store: s, Budget: time.Second}.Execute(rec.ID,
		func(context.Context) (string, error) {
			effectCalls++
			return bh018InjectedNativeID, nil
		},
		func(ctx context.Context) error {
			verifierCalls++
			if err := s.FinalizeVerifiedExec(ctx, rec.ID, box,
				bh018InjectedNativeID, stdout, stderr, 127); err != nil {
				return err
			}
			// Simulate a lost acknowledgement AFTER the successful COMMIT.
			return errors.New("bh018-injected-final-commit-reply-lost")
		})
	if effectCalls != 1 || verifierCalls != 1 {
		t.Fatalf("original effect or verifier repeated: effects=%d verifier=%d",
			effectCalls, verifierCalls)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if n, err := reopened.Recover(ctx); err != nil || n != 0 {
		t.Fatalf("durable verified receipt incorrectly recovered as UNKNOWN: n=%d err=%v", n, err)
	}
	got, err := reopened.Get(ctx, rec.ID)
	if err != nil || got.State != "SUCCEEDED" || got.NativeOperationID != bh018InjectedNativeID {
		t.Fatalf("lost COMMIT reply corrupted original receipt: %+v err=%v", got, err)
	}
	out, err := reopened.ExecOutput(ctx, rec.ID)
	if err != nil || out.ExitCode == nil || *out.ExitCode != 127 ||
		out.StdoutPath != stdout || out.StderrPath != stderr {
		t.Fatalf("lost COMMIT reply corrupted original guest exit/output: %+v err=%v", out, err)
	}
	var events int
	if err := reopened.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events
		WHERE operation_id=? AND state='SUCCEEDED'
		AND safe_detail='VERIFIED_NATIVE_EXEC_TERMINAL'`, rec.ID).Scan(&events); err != nil || events != 1 {
		t.Fatalf("expected exactly one durable final event, got %d err=%v", events, err)
	}
	replay, err := reopened.Replay(ctx, intent)
	if err != nil || !replay.Duplicate || replay.ID != rec.ID ||
		replay.NativeOperationID != bh018InjectedNativeID {
		t.Fatalf("original idempotency changed after lost reply: %+v err=%v", replay, err)
	}
	intent.Key = "bh018-after-lost-final-reply-new-command"
	if _, err := reopened.AdmitScoped(ctx, intent); err != nil {
		t.Fatalf("committed original still falsely holds resource lock: %v", err)
	}
}
