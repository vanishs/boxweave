package state

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	sqlite3 "github.com/mattn/go-sqlite3"
)

// This helper is an entirely separate test process with a private SQLite
// database. The parent test kills ONLY its own child PID, never an Incus
// daemon, host service, or a previously admitted real Guest operation.
func TestBH018SIGKILLFinalizerChild(t *testing.T) {
	stage := os.Getenv("BW_BH018_SIGKILL_STAGE")
	if stage == "" {
		return
	}
	if stage != "before_commit" && stage != "after_commit" {
		t.Fatal("unexpected SIGKILL stage")
	}
	path := os.Getenv("BW_BH018_SIGKILL_DB")
	marker := os.Getenv("BW_BH018_SIGKILL_MARKER")
	opID := os.Getenv("BW_BH018_SIGKILL_OPID")
	if path == "" || marker == "" || opID == "" {
		t.Fatal("missing isolated subprocess test identity")
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	box := OwnedBox{UUID: "bh018-atomic-owned-uuid", Project: "bh018-test-layer",
		Name: "bh018-atomic-owned", Incarnation: "bh018-atomic-owned-uuid", State: "OWNED"}
	stdout, stderr := bh018InjectedPaths(box)
	if stage == "before_commit" {
		// Temporary trigger: one SQLite connection, never stored in schema.
		// It runs AFTER original status + output INSERT + final event INSERT
		// but BEFORE SQLite can commit the transaction. Parent SIGKILL leaves
		// the real WAL uncommitted. Unlike RAISE(ABORT), this is a process kill.
		old := s.db
		sql.Register("bh018_sqlite_sigkill_helper", &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("bh018_pause_before_commit", func() int {
					if err := os.WriteFile(marker, []byte("in-transaction"), 0600); err != nil {
						panic(err)
					}
					select {}
				}, false)
			},
		})
		newDB, err := sql.Open("bh018_sqlite_sigkill_helper",
			"file:"+path+"?_busy_timeout=5000&_foreign_keys=on")
		if err != nil {
			t.Fatal(err)
		}
		newDB.SetMaxOpenConns(1)
		if err := old.Close(); err != nil {
			t.Fatal(err)
		}
		s.db = newDB
		if _, err := s.db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
			t.Fatal(err)
		}
		if _, err := s.db.Exec(`PRAGMA synchronous=FULL`); err != nil {
			t.Fatal(err)
		}
		if _, err := s.db.Exec(`CREATE TEMP TRIGGER bh018_pause_final_event
			AFTER INSERT ON main.operation_events
			WHEN NEW.state='SUCCEEDED' AND NEW.safe_detail='VERIFIED_NATIVE_EXEC_TERMINAL'
			BEGIN SELECT bh018_pause_before_commit(); END`); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.FinalizeVerifiedExec(context.Background(), opID, box,
		bh018InjectedNativeID, stdout, stderr, 126); err != nil {
		t.Fatalf("finalization did not reach crash boundary: %v", err)
	}
	if stage == "after_commit" {
		// The SQLite COMMIT has returned and the durable status is written;
		// deliberately stop before any Worker readback/application reply.
		if err := os.WriteFile(marker, []byte("durable-commit"), 0600); err != nil {
			t.Fatal(err)
		}
		select {}
	}
	t.Fatal("before-COMMIT trigger unexpectedly returned without SIGKILL")
}

func TestBH018ActualSIGKILLBeforeAndAfterAtomicExecCommit(t *testing.T) {
	for _, stage := range []string{"before_commit", "after_commit"} {
		t.Run(stage, func(t *testing.T) {
			ctx := context.Background()
			s, box, intent, db := bh018InjectedAtomicFixture(t)
			opID := bh018InjectedAdmitVerifying(t, s, intent)
			stdout, stderr := bh018InjectedPaths(box)
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(t.TempDir(), "crash-boundary.marker")
			cmd := exec.Command(os.Args[0], "-test.run=^TestBH018SIGKILLFinalizerChild$")
			cmd.Env = append(os.Environ(),
				"BW_BH018_SIGKILL_STAGE="+stage,
				"BW_BH018_SIGKILL_DB="+db,
				"BW_BH018_SIGKILL_MARKER="+marker,
				"BW_BH018_SIGKILL_OPID="+opID)
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			}()
			deadline := time.Now().Add(12 * time.Second)
			for {
				if _, err := os.Stat(marker); err == nil {
					break
				}
				if time.Now().After(deadline) {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
					t.Fatalf("SIGKILL boundary never reached: %s", output.String())
				}
				time.Sleep(20 * time.Millisecond)
			}
			if err := cmd.Process.Kill(); err != nil {
				t.Fatalf("cannot kill isolated child PID: %v", err)
			}
			_ = cmd.Wait()
			reopened, err := Open(db)
			if err != nil {
				t.Fatalf("SQLite file failed to reopen after SIGKILL: %v", err)
			}
			defer reopened.Close()
			got, err := reopened.Get(ctx, opID)
			if err != nil || got.NativeOperationID != bh018InjectedNativeID {
				t.Fatalf("original native identity lost after SIGKILL: %+v %v", got, err)
			}
			var events int
			if err := reopened.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events
				WHERE operation_id=? AND state='SUCCEEDED'
				AND safe_detail='VERIFIED_NATIVE_EXEC_TERMINAL'`, opID).Scan(&events); err != nil {
				t.Fatal(err)
			}
			if stage == "before_commit" {
				if got.State != "VERIFYING" || events != 0 {
					t.Fatalf("uncommitted final survived SIGKILL: %+v events=%d", got, events)
				}
				if _, err := reopened.ExecOutput(ctx, opID); !errors.Is(err, ErrNotFound) {
					t.Fatalf("uncommitted output survived SIGKILL: %v", err)
				}
				if n, err := reopened.Recover(ctx); err != nil || n != 1 {
					t.Fatalf("uncommitted result was not recovered fail-closed: n=%d %v", n, err)
				}
				intent.Key = "bh018-sigkill-new-key-remains-locked"
				if _, err := reopened.AdmitScoped(ctx, intent); !errors.Is(err, ErrResourceLocked) {
					t.Fatalf("crashed original released lock without evidence: %v", err)
				}
			} else {
				if got.State != "SUCCEEDED" || events != 1 {
					t.Fatalf("committed original was lost or double-recorded: %+v events=%d", got, events)
				}
				out, err := reopened.ExecOutput(ctx, opID)
				if err != nil || out.ExitCode == nil || *out.ExitCode != 126 ||
					out.StdoutPath != stdout || out.StderrPath != stderr {
					t.Fatalf("committed original exit/path lost: %+v %v", out, err)
				}
				if n, err := reopened.Recover(ctx); err != nil || n != 0 {
					t.Fatalf("committed final falsely recovered UNKNOWN: n=%d %v", n, err)
				}
				replay, err := reopened.Replay(ctx, intent)
				if err != nil || !replay.Duplicate || replay.ID != opID {
					t.Fatalf("original key no longer idempotent: %+v %v", replay, err)
				}
				intent.Key = "bh018-sigkill-new-key-after-real-commit"
				if _, err := reopened.AdmitScoped(ctx, intent); err != nil {
					t.Fatalf("fully committed original kept false resource lock: %v", err)
				}
			}
			t.Logf("BH018_SQLITE_CHILD_SIGKILL_%s=PASS original=%s", stage, opID)
		})
	}
}

// Keep the test narrowly scoped to a local SQLite subprocess. Actual HTTP
// MCP, Incus 7.6 effects and an independent verifier are separate acceptance.
var _ = fmt.Sprintf
