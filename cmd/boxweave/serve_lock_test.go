package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestS2ServeLockPreventsSecondOwnerAndReopensAfterRelease(t *testing.T) {
	database := filepath.Join(t.TempDir(), "ledger.sqlite")
	first, err := acquireServeLock(database)
	if err != nil {
		t.Fatal(err)
	}
	if second, err := acquireServeLock(database); err == nil {
		_ = second.Close()
		t.Fatal("second active controller acquired same SQLite serve lock")
	} else if !strings.Contains(err.Error(), "already active") {
		t.Fatalf("unexpected second serve refusal: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := acquireServeLock(database)
	if err != nil {
		t.Fatalf("serve lock not re-acquired after original exit: %v", err)
	}
	if err := reopened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestS2ServeLockRefusesSymlinkOrWorldReadableFile(t *testing.T) {
	database := filepath.Join(t.TempDir(), "ledger.sqlite")
	target := filepath.Join(t.TempDir(), "not-our-lock")
	if err := os.WriteFile(target, []byte("not ours"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, database+".serve.lock"); err != nil {
		t.Fatal(err)
	}
	if f, err := acquireServeLock(database); err == nil {
		_ = f.Close()
		t.Fatal("symlink to unrelated host file adopted as serve lock")
	}
	if err := os.Remove(database + ".serve.lock"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(database+".serve.lock", []byte{}, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(database+".serve.lock", 0644); err != nil {
		t.Fatal(err)
	}
	if f, err := acquireServeLock(database); err == nil {
		_ = f.Close()
		t.Fatal("unsafe lock permissions accepted")
	}
}
