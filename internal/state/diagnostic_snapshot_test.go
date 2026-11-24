package state

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// Source writer MUST be another process: closing an fd in the same
// process can release SQLite's POSIX locks regardless of OFD locking.
func TestPR177D1WriterProcess(t *testing.T) {
	path := os.Getenv("BOXWEAVE_D1_WRITER_FIXTURE")
	if path == "" {
		return
	}
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, e = s.db.Exec("PRAGMA wal_autocheckpoint=0"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("INSERT INTO oauth_attempts(at,kind) VALUES(1,'d1-committed')"); e != nil {
		t.Fatal(e)
	}
	fmt.Println("D1_READY")
	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		switch scan.Text() {
		case "hold":
			tx, e := s.db.Begin()
			if e != nil {
				t.Fatal(e)
			}
			if _, e = tx.Exec("INSERT INTO oauth_attempts(at,kind) VALUES(2,'d1-uncommitted')"); e != nil {
				t.Fatal(e)
			}
			fmt.Println("D1_HELD")
			if !scan.Scan() || scan.Text() != "commit" {
				t.Fatal("fixture protocol")
			}
			if e = tx.Commit(); e != nil {
				t.Fatal(e)
			}
			fmt.Println("D1_COMMITTED")
		case "checkpoint":
			if _, e = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); e != nil {
				t.Fatal(e)
			}
			fmt.Println("D1_CHECKPOINTED")
		case "exit":
			return
		default:
			t.Fatal("fixture protocol")
		}
	}
}

type d1Writer struct {
	cmd *exec.Cmd
	in  *os.File
	out *bufio.Scanner
}

func d1Start(t *testing.T, path string) *d1Writer {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPR177D1WriterProcess$")
	cmd.Env = append(os.Environ(), "BOXWEAVE_D1_WRITER_FIXTURE="+path)
	r, w, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	cmd.Stdin = r
	output, e := cmd.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	cmd.Stderr = os.Stderr
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	r.Close()
	t.Cleanup(func() {
		w.Close()
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	x := &d1Writer{cmd: cmd, in: w, out: bufio.NewScanner(output)}
	x.expect(t, "D1_READY")
	return x
}
func (w *d1Writer) expect(t *testing.T, want string) {
	t.Helper()
	if !w.out.Scan() || w.out.Text() != want {
		t.Fatal("writer output", want, w.out.Text(), w.out.Err())
	}
}
func (w *d1Writer) send(t *testing.T, s string) {
	t.Helper()
	if _, e := fmt.Fprintln(w.in, s); e != nil {
		t.Fatal(e)
	}
}
func d1Contents(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	for _, n := range d0FileNames(t, dir) {
		b, e := os.ReadFile(filepath.Join(dir, n))
		if e != nil {
			t.Fatal(e)
		}
		out[n] = b
	}
	return out
}
func d1CheckRow(t *testing.T, s *DiagnosticSnapshot, kind string, want int) {
	t.Helper()
	var n int
	if e := s.store.db.QueryRow("SELECT COUNT(*) FROM oauth_attempts WHERE kind=?", kind).Scan(&n); e != nil || n != want {
		t.Fatal(kind, n, e)
	}
	if s.store.IsNetworkAuditReadOnly() {
		t.Fatal("snapshot became Root proof")
	}
}
func TestPR177D1LiveColdAndCrashSnapshots(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	w := d1Start(t, path)
	before := d1Contents(t, dir)
	snap, e := OpenDiagnosticSnapshot(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	if snap.Capture != "SQLITE_WAL_LOCKED_COPY" {
		t.Fatal(snap.Capture)
	}
	d1CheckRow(t, snap, "d1-committed", 1)
	if _, e = snap.store.db.Exec("DELETE FROM oauth_attempts"); e == nil {
		t.Fatal("snapshot writable")
	}
	private := snap.dir
	if e = snap.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(private); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("private copy retained", e)
	}
	if !reflect.DeepEqual(before, d1Contents(t, dir)) {
		t.Fatal("source DB/WAL/SHM changed")
	}
	w.send(t, "hold")
	w.expect(t, "D1_HELD")
	if s, e := OpenDiagnosticSnapshot(ctx, path); !errors.Is(e, ErrDiagnosticBusy) {
		if s != nil {
			s.Close()
		}
		t.Fatal("active writer not refused", e)
	}
	w.send(t, "commit")
	w.expect(t, "D1_COMMITTED")
	w.send(t, "checkpoint")
	w.expect(t, "D1_CHECKPOINTED")
	// Checkpointed active WAL can be zero length, still a coherent epoch.
	before = d1Contents(t, dir)
	snap, e = OpenDiagnosticSnapshot(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	d1CheckRow(t, snap, "d1-uncommitted", 1)
	snap.Close()
	if !reflect.DeepEqual(before, d1Contents(t, dir)) {
		t.Fatal("checkpointed source changed")
	}
	w.send(t, "exit")
	if e = w.cmd.Wait(); e != nil {
		t.Fatal(e)
	}
	before = d1Contents(t, dir)
	snap, e = OpenDiagnosticSnapshot(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	if snap.Capture != "SQLITE_EXCLUSIVE_COLD_COPY" {
		t.Fatal(snap.Capture)
	}
	d1CheckRow(t, snap, "d1-uncommitted", 1)
	snap.Close()
	if !reflect.DeepEqual(before, d1Contents(t, dir)) {
		t.Fatal("cold source changed")
	}
	t.Log("D1_LIVE_WAL_BUSY_CHECKPOINT_COLD_NO_SOURCE_EFFECTS_PASS")
	crashDir := t.TempDir()
	if e := os.Chmod(crashDir, 0700); e != nil {
		t.Fatal(e)
	}
	crashPath := filepath.Join(crashDir, "ledger.sqlite")
	crash := d1Start(t, crashPath)
	if e = crash.cmd.Process.Kill(); e != nil {
		t.Fatal(e)
	}
	_ = crash.cmd.Wait()
	// Test-only missing SHM after crash; committed WAL still exists.
	if e = os.Remove(crashPath + "-shm"); e != nil {
		t.Fatal(e)
	}
	before = d1Contents(t, crashDir)
	snap, e = OpenDiagnosticSnapshot(ctx, crashPath)
	if e != nil {
		t.Fatal(e)
	}
	d1CheckRow(t, snap, "d1-committed", 1)
	snap.Close()
	if !reflect.DeepEqual(before, d1Contents(t, crashDir)) {
		t.Fatal("crash source changed")
	}
	t.Log("D1_CRASH_WAL_NO_SHM_CAPTURE_PASS")
}

func TestPR177D1SnapshotRefusals(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	refuse := func(p string, want error) {
		t.Helper()
		s, e := OpenDiagnosticSnapshot(ctx, p)
		if s != nil {
			s.Close()
		}
		if !errors.Is(e, want) {
			t.Fatal(p, e, want)
		}
	}
	refuse(filepath.Join(dir, "missing", "db"), ErrDiagnosticMissing)
	if len(d0FileNames(t, dir)) != 0 {
		t.Fatal("created missing path")
	}
	rw, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = rw.Close(); e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(path, 0644); e != nil {
		t.Fatal(e)
	}
	refuse(path, ErrDiagnosticUnsafe)
	os.Chmod(path, 0600)
	if e = os.Symlink(path, filepath.Join(dir, "link")); e != nil {
		t.Fatal(e)
	}
	refuse(filepath.Join(dir, "link"), ErrDiagnosticUnsafe)
	if e = os.Link(path, filepath.Join(dir, "hard")); e != nil {
		t.Fatal(e)
	}
	refuse(path, ErrDiagnosticUnsafe)
	os.Remove(filepath.Join(dir, "hard"))
	if e = os.Symlink(dir, filepath.Join(t.TempDir(), "parent-link")); e != nil {
		t.Fatal(e)
	}
	aliasDir := t.TempDir()
	os.Symlink(dir, filepath.Join(aliasDir, "alias"))
	refuse(filepath.Join(aliasDir, "alias", "ledger.sqlite"), ErrDiagnosticUnsafe)
	// Source journal, unsafe WAL symlink, and hardlink sidecar all refuse.
	if e = os.WriteFile(path+"-journal", []byte("hot"), 0600); e != nil {
		t.Fatal(e)
	}
	refuse(path, ErrDiagnosticBusy)
	os.Remove(path + "-journal")
	if e = os.Symlink(path, path+"-wal"); e != nil {
		t.Fatal(e)
	}
	refuse(path, ErrDiagnosticUnsafe)
	os.Remove(path + "-wal")
	if e = os.Link(path, path+"-wal"); e != nil {
		t.Fatal(e)
	}
	refuse(path, ErrDiagnosticUnsafe)
	os.Remove(path + "-wal")
	bad := filepath.Join(dir, "bad")
	if e = os.WriteFile(bad, []byte("corrupt"), 0600); e != nil {
		t.Fatal(e)
	}
	before := d1Contents(t, dir)
	refuse(bad, ErrDiagnosticInvalid)
	if !reflect.DeepEqual(before, d1Contents(t, dir)) {
		t.Fatal("invalid source changed")
	}
	huge := filepath.Join(dir, "huge")
	if e = os.WriteFile(huge, []byte("huge"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.Truncate(huge, diagnosticMaxBytes+1); e != nil {
		t.Fatal(e)
	}
	refuse(huge, ErrDiagnosticBound)
	t.Log("D1_MISSING_UNSAFE_CORRUPT_BOUNDED_REFUSALS_PASS")
}

func TestPR177D1SnapshotSchemaAndEmptyScoped(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	snap, e := OpenDiagnosticSnapshot(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	summary, e := snap.ScopedLedgerSummary(ctx, "user-1001", "incusbr-1001")
	if e != nil || summary.Owned != 0 || summary.Retired != 0 {
		t.Fatal(summary, e)
	}
	snap.Close()
	// A foreign trigger must still fail exact schema validation.
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("CREATE TRIGGER foreign_d1 AFTER INSERT ON oauth_attempts BEGIN SELECT 1; END"); e != nil {
		t.Fatal(e)
	}
	s.Close()
	before := d1Contents(t, dir)
	if snap, e = OpenDiagnosticSnapshot(ctx, path); !errors.Is(e, ErrDiagnosticInvalid) {
		if snap != nil {
			snap.Close()
		}
		t.Fatal(e)
	}
	if !reflect.DeepEqual(before, d1Contents(t, dir)) {
		t.Fatal("foreign schema changed")
	}
	t.Log("D1_EMPTY_SCOPED_AND_SCHEMA_REFUSAL_PASS")
}
