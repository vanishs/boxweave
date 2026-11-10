package state

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestS2SQLiteAliasAndHardlinkCannotBypassServeLock(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "canonical.sqlite")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	originalDigest := sha256.Sum256(original)
	hardlink := filepath.Join(dir, "same-inode.sqlite")
	if err := os.Link(path, hardlink); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []string{path, hardlink} {
		reopened, err := Open(candidate)
		if err == nil {
			_ = reopened.Close()
			t.Fatalf("SQLite hardlink alias accepted for possible duplicate serve: %s", candidate)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || sha256.Sum256(after) != originalDigest {
		t.Fatalf("rejected hardlinks modified original SQLite: %v", err)
	}
	if err := os.Remove(hardlink); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("original DB did not reopen after alias removal: %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
	realDir := filepath.Join(dir, "real")
	if err := os.Mkdir(realDir, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Symlink(realDir, alias); err != nil {
		t.Fatal(err)
	}
	if opened, err := Open(filepath.Join(alias, "journal.sqlite")); err == nil {
		_ = opened.Close()
		t.Fatal("ancestor symlink could create alternate SQLite serve lock")
	}
	unclean := realDir + "/../real/journal.sqlite"
	if opened, err := Open(unclean); err == nil {
		_ = opened.Close()
		t.Fatal("non-canonical ancestor traversal could create alternate serve lock")
	}
}

func TestS2SQLiteTwoLocalWritersSerializeWithoutJournalDamage(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "writers.sqlite")
	one, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer one.Close()
	two, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer two.Close()
	ctx := context.Background()
	const writers = 40
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			write, read := one, two
			if i%2 != 0 {
				write, read = two, one
			}
			raw, issued, err := write.IssueToken(ctx, "viewer", time.Hour)
			if err != nil {
				errs <- err
				return
			}
			got, err := read.VerifyToken(ctx, raw)
			if err != nil {
				errs <- err
				return
			}
			if got.ID != issued.ID || got.Role != "viewer" {
				errs <- ErrState
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent SQLite credential write/read: %v", err)
	}
	var count int
	if err := one.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM credentials WHERE role='viewer'").Scan(&count); err != nil || count != writers {
		t.Fatalf("concurrent writes disappeared: got=%d err=%v", count, err)
	}
	// No mixed or partial journal schema may be introduced by concurrent CLI
	// writers. The local single-server flock is tested separately.
	if path == "" || strings.ContainsRune(path, 0) {
		t.Fatal("invalid test database path")
	}
}
