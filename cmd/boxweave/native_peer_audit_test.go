package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestS4B2NativePeerAuditCLIRefusesMissingSQLiteWithoutCreatingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-created.sqlite")
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"boxweave", "native-peer-audit", "--db", path, "--project", "user-1001", "--network", "bwbr0", "--acl", "bw-s4b2"}
	if err := run(); err == nil {
		t.Fatal("native root audit accepted missing protected SQLite")
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read-only CLI created a database: %v", err)
	}
}
