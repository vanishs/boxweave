package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const bh018InjectedNativeID = "ac4fdc24-1111-4222-8333-6fdf6b6ee141"

func bh018InjectedAtomicFixture(t *testing.T) (*Store, OwnedBox, Admission, string) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(dir, "bh018-atomic.sqlite")
	s, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	_, token, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	box := OwnedBox{UUID: "bh018-atomic-owned-uuid", Project: "bh018-test-layer",
		Name: "bh018-atomic-owned", Incarnation: "bh018-atomic-owned-uuid", State: "OWNED"}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO owned_boxes
		(uuid,project,name,incarnation,state,created_at)
		VALUES (?,?,?,?,'OWNED',?)`, box.UUID, box.Project, box.Name, box.Incarnation,
		time.Now().UnixNano()); err != nil {
		t.Fatal(err)
	}
	intent := Admission{Layer: box.Project, ActorID: token.ID, TokenID: token.ID,
		Key: "bh018-atomic-original-key", RequestID: "bh018-atomic-original",
		Action: "box_exec", ResourceKey: "box:" + box.Name,
		BoxUUID: box.UUID, Incarnation: box.Incarnation,
		SafeIntent: []byte(`{"digest":"bh018-atomic-test-only"}`)}
	return s, box, intent, db
}

func bh018InjectedAdmitVerifying(t *testing.T, s *Store, in Admission) string {
	t.Helper()
	ctx := context.Background()
	r, err := s.Admit(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.NativeAck(ctx, r.ID, bh018InjectedNativeID); err != nil {
		t.Fatal(err)
	}
	return r.ID
}

func bh018InjectedPaths(box OwnedBox) (string, string) {
	p := "/1.0/instances/" + box.Name + "/logs/exec-output/exec_" + bh018InjectedNativeID
	return p + ".stdout", p + ".stderr"
}

// Trigger rejection occurs AFTER the SQL UPDATE and exec_outputs INSERT but
// BEFORE the final event INSERT. A genuinely single SQLite transaction must
// roll back ALL changes and preserve the existing effect claim. This is an
// injected SQLite failure, not a simulated native Incus response.
func TestBH018AtomicFinalizationRollsBackEverythingOnEventInsertFailure(t *testing.T) {
	ctx := context.Background()
	s, box, intent, _ := bh018InjectedAtomicFixture(t)
	id := bh018InjectedAdmitVerifying(t, s, intent)
	out, errOut := bh018InjectedPaths(box)
	_, err := s.db.ExecContext(ctx, `CREATE TRIGGER bh018_abort_verified_event
		BEFORE INSERT ON operation_events
		WHEN NEW.state='SUCCEEDED' AND NEW.safe_detail='VERIFIED_NATIVE_EXEC_TERMINAL'
		BEGIN SELECT RAISE(ABORT, 'bh018-injected-before-final-commit'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeVerifiedExec(ctx, id, box, bh018InjectedNativeID, out, errOut, 127); err == nil {
		t.Fatal("injected terminal event write failure was ignored")
	}
	r, err := s.Get(ctx, id)
	if err != nil || r.State != "VERIFYING" || r.NativeOperationID != bh018InjectedNativeID {
		t.Fatalf("partially committed original terminal: %+v %v", r, err)
	}
	if _, err = s.ExecOutput(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("output row survived rolled-back finalization: %v", err)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'", id).Scan(&count); err != nil || count != 0 {
		t.Fatalf("partially committed success event: count=%d err=%v", count, err)
	}
	// Crash-style recovery cannot trust a terminal receipt whose transaction
	// failed; retain UNKNOWN and the Box claim, never redispatch the guest.
	if n, err := s.Recover(ctx); err != nil || n != 1 {
		t.Fatalf("crash recovery did not preserve uncommitted final: n=%d err=%v", n, err)
	}
	r, err = s.Get(ctx, id)
	if err != nil || r.State != "UNKNOWN" {
		t.Fatalf("unverified final unexpectedly unlocked: %+v %v", r, err)
	}
	intent.Key = "bh018-after-rollback-separate-new-command"
	if _, err := s.Admit(ctx, intent); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("rolled back final lost UNKNOWN lock: %v", err)
	}
}

func TestBH018AtomicFinalizationSurvivesDatabaseReopenAndRecovery(t *testing.T) {
	ctx := context.Background()
	s, box, intent, db := bh018InjectedAtomicFixture(t)
	id := bh018InjectedAdmitVerifying(t, s, intent)
	out, errOut := bh018InjectedPaths(box)
	if err := s.FinalizeVerifiedExec(ctx, id, box, bh018InjectedNativeID, out, errOut, 126); err != nil {
		t.Fatal(err)
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
		t.Fatalf("durable atomic success marked UNKNOWN on restart: n=%d err=%v", n, err)
	}
	r, err := reopened.Get(ctx, id)
	if err != nil || r.State != "SUCCEEDED" || r.NativeOperationID != bh018InjectedNativeID {
		t.Fatalf("original finished Exec lost after reopened SQLite: %+v %v", r, err)
	}
	ev, err := reopened.ExecOutput(ctx, id)
	if err != nil || ev.ExitCode == nil || *ev.ExitCode != 126 ||
		ev.StdoutPath != out || ev.StderrPath != errOut {
		t.Fatalf("durable output/exit mismatch: %+v %v", ev, err)
	}
	var count int
	if err := reopened.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events
		WHERE operation_id=? AND state='SUCCEEDED' AND safe_detail='VERIFIED_NATIVE_EXEC_TERMINAL'`,
		id).Scan(&count); err != nil || count != 1 {
		t.Fatalf("original final event not exactly once: %d %v", count, err)
	}
	if err := reopened.FinalizeVerifiedExec(ctx, id, box, bh018InjectedNativeID, out, errOut, 126); !errors.Is(err, ErrState) {
		t.Fatalf("duplicate finalization succeeded: %v", err)
	}
	if err := reopened.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events
		WHERE operation_id=? AND state='SUCCEEDED'`, id).Scan(&count); err != nil || count != 1 {
		t.Fatalf("final event replayed: %d %v", count, err)
	}
	same, err := reopened.Replay(ctx, intent)
	if err != nil || !same.Duplicate || same.ID != id || same.State != "SUCCEEDED" {
		t.Fatalf("original idempotency replay changed: %+v %v", same, err)
	}
	intent.Key = "bh018-after-confirmed-final-new-command"
	if _, err := reopened.Admit(ctx, intent); err != nil {
		t.Fatalf("verified terminal still holds ownership lock: %v", err)
	}
}

func TestBH018AtomicFinalizationRejectsForeignOrMalformedProvenance(t *testing.T) {
	ctx := context.Background()
	s, box, intent, _ := bh018InjectedAtomicFixture(t)
	id := bh018InjectedAdmitVerifying(t, s, intent)
	out, errOut := bh018InjectedPaths(box)
	tests := []struct {
		name   string
		box    OwnedBox
		id     string
		stdout string
		stderr string
	}{
		{"wrong-project", OwnedBox{UUID: box.UUID, Project: "foreign-project", Name: box.Name, Incarnation: box.Incarnation, State: "OWNED"}, bh018InjectedNativeID, out, errOut},
		{"wrong-incarnation", OwnedBox{UUID: box.UUID, Project: box.Project, Name: box.Name, Incarnation: "foreign-incarnation", State: "OWNED"}, bh018InjectedNativeID, out, errOut},
		{"wrong-name", OwnedBox{UUID: box.UUID, Project: box.Project, Name: "another-box", Incarnation: box.Incarnation, State: "OWNED"}, bh018InjectedNativeID, out, errOut},
		{"wrong-native-id", box, "00000000-0000-0000-0000-000000000000", out, errOut},
		{"foreign-output", box, bh018InjectedNativeID, "/1.0/instances/foreign/logs/exec-output/exec_" + bh018InjectedNativeID + ".stdout", errOut},
		{"swapped-stream", box, bh018InjectedNativeID, errOut, out},
		{"missing-both-paths", box, bh018InjectedNativeID, "", ""},
		{"malformed-native-id", box, "../../escape", out, errOut},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := s.FinalizeVerifiedExec(ctx, id, tc.box, tc.id, tc.stdout, tc.stderr, 127); err == nil {
				t.Fatal("unverified or foreign terminal was finalized")
			}
			r, err := s.Get(ctx, id)
			if err != nil || r.State != "VERIFYING" {
				t.Fatalf("foreign original mutated: %+v %v", r, err)
			}
			if _, err := s.ExecOutput(ctx, id); !errors.Is(err, ErrNotFound) {
				t.Fatalf("foreign output stored: %v", err)
			}
		})
	}
	if err := s.FinalizeVerifiedExec(ctx, id, box, bh018InjectedNativeID, out, errOut, 127); err != nil {
		t.Fatalf("legitimate verified final rejected after negative controls: %v", err)
	}
}
