package state

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// A native side effect has already returned its ID when the operation event
// insert fails. SQLite must roll back the entire NativeAck transition and the
// live worker must mark the original UNKNOWN rather than silently exit.
func TestS2NativeAckEventRollbackRetainsOriginalUnknown(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ack-fault.sqlite")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, token, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	a := Admission{
		Layer: "s2-ack-fault", ActorID: token.ID, TokenID: token.ID,
		Key: "s2-native-ack-lost-sqlite-001", RequestID: "original",
		Action: "box_update", ResourceKey: "box:locked-update",
		BoxUUID: "native-uuid-ack-test", Incarnation: "native-uuid-ack-test",
		SafeIntent: []byte(`{"box_id":"native-uuid-ack-test","cpu":2}`),
	}
	r, err := s.Admit(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	// TEMP trigger affects only this connection and is not persisted as an
	// undeclared schema change. It fails the second statement of NativeAck's
	// transaction after the state UPDATE but before COMMIT.
	if _, err := s.db.Exec(`CREATE TEMP TRIGGER s2_fault_ack_event
		BEFORE INSERT ON operation_events WHEN NEW.state='VERIFYING'
		BEGIN SELECT RAISE(ABORT, 'S2 injected native-ack journal error'); END`); err != nil {
		t.Fatal(err)
	}
	dispatched, verified := 0, 0
	Worker{Store: s, Budget: 5 * time.Second}.Execute(r.ID,
		func(context.Context) (string, error) {
			dispatched++
			return "native-accepted-once", nil
		},
		func(context.Context) error {
			verified++
			return nil
		})
	if dispatched != 1 || verified != 0 {
		t.Fatalf("native effect count=%d postread count=%d", dispatched, verified)
	}
	got, err := s.Get(ctx, r.ID)
	if err != nil || got.State != "UNKNOWN" || got.ErrorClass != "NATIVE_ACK_DURABILITY_UNVERIFIED" ||
		got.NativeOperationID != "native-accepted-once" || got.NextAction.Kind != "HOST_LOCAL_REVIEW" {
		t.Fatalf("NativeAck failed yet original was not safely UNKNOWN: %+v %v", got, err)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='VERIFYING'", r.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("torn NativeAck event committed: count=%d err=%v", n, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err = reopened.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	again, err := reopened.Replay(ctx, a)
	if err != nil || !again.Duplicate || again.ID != r.ID || again.State != "UNKNOWN" {
		t.Fatalf("reopening lost original unknown/replay lock: %+v %v", again, err)
	}
	a.Key = "s2-native-ack-lost-sqlite-002"
	if _, err = reopened.Admit(ctx, a); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("second native dispatch admitted after fault: %v", err)
	}
}

// This probe uses a new OS process: it is killed while its SQLite transaction
// contains both journal changes but has NOT committed. SQLite WAL rollback,
// Recover, replay and the name lock are verified on a newly opened Store.
// This does not simulate a kernel crash during COMMIT fsync.
func TestS2SQLiteKilledDuringUncommittedNativeAck(t *testing.T) {
	if os.Getenv("BOXWEAVE_S2_UNCOMMITTED_ACK_CHILD") == "1" {
		ctx := context.Background()
		s, err := Open(os.Getenv("BOXWEAVE_S2_ACK_DB"))
		if err != nil {
			t.Fatal(err)
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		id := os.Getenv("BOXWEAVE_S2_ACK_ID")
		if _, err := tx.ExecContext(ctx,
			"UPDATE operations SET state='VERIFYING',native_operation_id='uncommitted-native-id' WHERE id=? AND state='DISPATCHING'", id); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO operation_events(operation_id,at,state) VALUES (?,?,?)",
			id, time.Now().UnixNano(), "VERIFYING"); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(os.Getenv("BOXWEAVE_S2_ACK_STAGED"), []byte("staged\n"), 0600); err != nil {
			t.Fatal(err)
		}
		// Do not close Store, roll back, or let defer execute. SIGKILL must
		// leave real WAL recovery to the next process.
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Kill(); err != nil {
			t.Fatal(err)
		}
		os.Exit(20)
	}
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "kill-uncommitted-ack.sqlite")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, token, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	a := Admission{
		Layer: "s2-torn-ack", ActorID: token.ID, TokenID: token.ID,
		Key: "s2-killed-uncommitted-native-ack-001", RequestID: "original",
		Action: "box_update", ResourceKey: "box:locked-on-kill",
		BoxUUID: "native-uuid-kill", Incarnation: "native-uuid-kill",
		SafeIntent: []byte(`{"box_id":"native-uuid-kill","cpu":2}`),
	}
	r, err := s.Admit(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "transaction-staged")
	cmd := exec.Command(os.Args[0], "-test.run=^TestS2SQLiteKilledDuringUncommittedNativeAck$")
	cmd.Env = append(os.Environ(),
		"BOXWEAVE_S2_UNCOMMITTED_ACK_CHILD=1",
		"BOXWEAVE_S2_ACK_DB="+path,
		"BOXWEAVE_S2_ACK_ID="+r.ID,
		"BOXWEAVE_S2_ACK_STAGED="+marker)
	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != -1 {
		t.Fatalf("test helper did not die from SIGKILL: %v", err)
	}
	if _, err := os.ReadFile(marker); err != nil {
		t.Fatalf("helper never staged a native ACK transaction: %v", err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	before, err := reopened.Get(ctx, r.ID)
	if err != nil || before.State != "DISPATCHING" || before.NativeOperationID != "" {
		t.Fatalf("uncommitted transaction leaked into durable journal: %+v %v", before, err)
	}
	if n, err := reopened.Recover(ctx); err != nil || n != 1 {
		t.Fatalf("restart did not handle original dispatch: count=%d err=%v", n, err)
	}
	again, err := reopened.Replay(ctx, a)
	if err != nil || !again.Duplicate || again.ID != r.ID || again.State != "UNKNOWN" ||
		again.NextAction.Kind != "HOST_LOCAL_REVIEW" {
		t.Fatalf("original replay after SIGKILL: %+v %v", again, err)
	}
	a.Key = "s2-killed-uncommitted-native-ack-002"
	if _, err = reopened.Admit(ctx, a); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("crash permitted a second native effect: %v", err)
	}
}
