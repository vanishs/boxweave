package state

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestS4B2ReadOnlyAuditRefusesMissingUnsafeDB(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "owner.sqlite")
	if _, err := OpenNetworkAuditReadOnly(path); err == nil {
		t.Fatal("missing database created")
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("missing database changed", err)
	}
	rw, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := rw.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ro.db.Exec("INSERT INTO oauth_attempts(at,kind) VALUES(1,'blocked')"); err == nil {
		t.Fatal("readonly mutates")
	}
	if v, e := ro.Schema(context.Background()); e != nil || v != SchemaVersion {
		t.Fatal(v, e)
	}
	if err := ro.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenNetworkAuditReadOnly(path); err == nil {
		t.Fatal("public parent admitted")
	}
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenNetworkAuditReadOnly(path); err == nil {
		t.Fatal("public file admitted")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "symlink.sqlite")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenNetworkAuditReadOnly(link); err == nil {
		t.Fatal("symlink followed")
	}
	hard := filepath.Join(dir, "hard.sqlite")
	if err := os.Link(path, hard); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenNetworkAuditReadOnly(path); err == nil {
		t.Fatal("hardlink admitted")
	}
	if err := os.Remove(hard); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing", "db")
	if _, err := OpenNetworkAuditReadOnly(missing); err == nil {
		t.Fatal("missing parent created")
	}
	if _, err := os.Lstat(filepath.Dir(missing)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("missing parent became visible")
	}
	if _, err := OpenNetworkAuditReadOnly("relative.sqlite"); err == nil {
		t.Fatal("relative path admitted")
	}
}

func TestS4B2ReadOnlyAuditSeesLiveWALAndRefusesSchemaDrift(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	rw, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rw.db.Exec("INSERT INTO oauth_attempts(at,kind) VALUES(7,'wal-visible')"); err != nil {
		t.Fatal(err)
	}
	ro, err := OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	var n int
	if err := ro.db.QueryRow("SELECT COUNT(*) FROM oauth_attempts WHERE kind='wal-visible'").Scan(&n); err != nil || n != 1 {
		t.Fatal("WAL not visible", n, err)
	}
	if _, err := ro.db.Exec("DELETE FROM oauth_attempts"); err == nil {
		t.Fatal("read-only write admitted")
	}
	if err := ro.Close(); err != nil {
		t.Fatal(err)
	}
	if err := rw.Close(); err != nil {
		t.Fatal(err)
	}
	foreign, err := sql.Open("sqlite3", "file:"+path+"?_busy_timeout=5000")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := foreign.Exec("CREATE TABLE FOREIGN_SHADOW(id TEXT)"); err != nil {
		t.Fatal(err)
	}
	if err := foreign.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenNetworkAuditReadOnly(path); err == nil {
		t.Fatal("foreign schema accepted")
	}
}

func TestAuditReaderReleasesIdleAndRejectsReplacement(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	rw, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = rw.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	assertIdle := func() {
		t.Helper()
		if got := ro.db.Stats(); got.OpenConnections != 0 || got.InUse != 0 || got.Idle != 0 {
			t.Fatalf("audit reader retained physical connection: %+v", got)
		}
	}
	assertIdle()
	if v, err := ro.Schema(context.Background()); err != nil || v != SchemaVersion {
		t.Fatal(v, err)
	}
	assertIdle()
	// The original inode must remain pinned after unlink, without a SQLite
	// connection. A valid replacement at the same path must not be adopted.
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	replacement, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = replacement.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ro.Schema(context.Background()); err == nil {
		t.Fatal("audit reader adopted replacement database")
	}
	assertIdle()
	if _, err := ro.auditConnector.fileAnchor.Stat(); err != nil {
		t.Fatal("lost inode anchor", err)
	}
	if err := ro.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ro.auditConnector.fileAnchor.Stat(); err == nil {
		t.Fatal("leaked file anchor")
	}
	if _, err := ro.auditConnector.parentAnchor.Stat(); err == nil {
		t.Fatal("leaked parent anchor")
	}
}
