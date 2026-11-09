package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// A-080: unknown migration history must not be silently adopted even when
// the six declared tables still have their original names and DDL.
func TestS1ForeignSQLiteMigrationHistoryFailClosed(t *testing.T) {
	cases := []struct {
		name     string
		mutation string
	}{
		{"duplicate-version", "INSERT INTO schema_version(version) VALUES (1)"},
		{"mixed-version-history", "INSERT INTO schema_version(version) VALUES (2)"},
		{"missing-version-history", "DELETE FROM schema_version"},
		{"unrecognized-migration-table", "CREATE TABLE schema_migrations (name TEXT PRIMARY KEY)"},
		{"extra-application-table", "CREATE TABLE external_owner (value TEXT)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Chmod(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "ledger.sqlite")
			original, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			raw, tok, err := original.IssueToken(context.Background(), "admin", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			if err := original.Close(); err != nil {
				t.Fatal(err)
			}
			external, err := sql.Open("sqlite3", "file:"+path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := external.Exec(tc.mutation); err != nil {
				t.Fatal(err)
			}
			if err := external.Close(); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			adopted, err := Open(path)
			if err == nil {
				_ = adopted.Close()
				t.Fatal("unexpected foreign migration history was adopted")
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("rejected Open changed original database")
			}
			check, err := sql.Open("sqlite3", "file:"+path)
			if err != nil {
				t.Fatal(err)
			}
			var count int
			err = check.QueryRow("SELECT COUNT(*) FROM credentials WHERE id=? AND token_digest=?", tok.ID, DigestToken(raw)).Scan(&count)
			_ = check.Close()
			if err != nil || count != 1 {
				t.Fatalf("original credential lost: count=%d err=%v", count, err)
			}
		})
	}
}

// A-080: concurrent local openers must preserve the existing operation
// identity and its original Tracking ID.
func TestS1ConcurrentSQLiteOpenAndOriginalOperation(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, tok, err := first.IssueToken(context.Background(), "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := first.Admit(context.Background(), Admission{
		Layer: "L0", ActorID: tok.ID, TokenID: tok.ID, Key: "simultaneous-start-source-only",
		Action: "A080-source-check-no-native-effect", SafeIntent: []byte(`{"action":"source-only"}`),
		RequestID: "first",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	const n = 12
	start := make(chan struct{})
	errorsCh := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			s, err := Open(path)
			if err != nil {
				errorsCh <- err
				return
			}
			defer s.Close()
			version, err := s.Schema(context.Background())
			if err != nil || version != SchemaVersion {
				errorsCh <- fmt.Errorf("schema=%d err=%v", version, err)
				return
			}
			op, err := s.Get(context.Background(), initial.ID)
			if err != nil || op.ID != initial.ID || op.TrackingID != initial.TrackingID {
				errorsCh <- fmt.Errorf("lost original operation: %+v %v", op, err)
				return
			}
			errorsCh <- nil
		}()
	}
	close(start)
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

// A-080: a process lost during an uncommitted DDL change must not establish
// a false new schema or discard the original operation record.
func TestS1AbortedSQLiteMigrationTransaction(t *testing.T) {
	if os.Getenv("BW_S1_MIGRATION_CRASH_CHILD") == "1" {
		db, err := sql.Open("sqlite3", "file:"+os.Getenv("BW_S1_MIGRATION_CRASH_DB"))
		if err != nil {
			panic(err)
		}
		tx, err := db.Begin()
		if err != nil {
			panic(err)
		}
		if _, err = tx.Exec("ALTER TABLE operations ADD COLUMN interrupted_migration TEXT"); err != nil {
			panic(err)
		}
		os.Exit(79)
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, tok, err := s.IssueToken(context.Background(), "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	op, err := s.Admit(context.Background(), Admission{
		Layer: "L0", ActorID: tok.ID, TokenID: tok.ID, Key: "migration-crash-source-only",
		Action: "A080-source-test", SafeIntent: []byte(`{"source":true}`), RequestID: "first",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestS1AbortedSQLiteMigrationTransaction$")
	cmd.Env = append(os.Environ(), "BW_S1_MIGRATION_CRASH_CHILD=1", "BW_S1_MIGRATION_CRASH_DB="+path)
	err = cmd.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 79 {
		t.Fatalf("fault-injection child must exit 79, got: %v", err)
	}
	recovered, err := Open(path)
	if err != nil {
		t.Fatalf("SQLite atomic DDL recovery failed: %v", err)
	}
	defer recovered.Close()
	got, err := recovered.Get(context.Background(), op.ID)
	if err != nil || got.ID != op.ID || got.TrackingID != op.TrackingID {
		t.Fatalf("original operation lost after rollback: %+v %v", got, err)
	}
}
