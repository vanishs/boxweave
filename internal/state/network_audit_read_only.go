package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// OpenNetworkAuditReadOnly opens ONLY an existing protected owner database.
// It never creates directories/DBs or performs writable WAL initialization.
// This root-audit reader grants no ability to admit or mutate state.
func OpenNetworkAuditReadOnly(path string) (*Store, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path ||
		strings.ContainsAny(path, "?#\x00") {
		return nil, errors.New("absolute canonical literal audit database path required")
	}
	parent := filepath.Dir(path)
	physical, err := filepath.EvalSymlinks(parent)
	if err != nil || physical != parent {
		return nil, errors.New("audit database parent missing or contains symlink alias")
	}
	pstat, err := os.Lstat(parent)
	if err != nil || !pstat.IsDir() || pstat.Mode().Perm()&0077 != 0 {
		return nil, errors.New("audit database parent must be a private real directory")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm()&0077 != 0 {
		return nil, errors.New("audit database must be a private existing regular file")
	}
	// Refuse a symlink swap between Lstat and OpenFile.
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	opened, err := f.Stat()
	_ = f.Close()
	if err != nil || !opened.Mode().IsRegular() || opened.Mode().Perm()&0077 != 0 ||
		!os.SameFile(before, opened) {
		return nil, errors.New("audit database identity or permissions changed")
	}
	ino, ok := opened.Sys().(*syscall.Stat_t)
	if !ok || ino.Nlink != 1 {
		return nil, errors.New("audit database hardlinks or unknown inode refused")
	}
	uri := (&url.URL{Scheme: "file", Path: path,
		RawQuery: "mode=ro&_query_only=1&_busy_timeout=5000&_foreign_keys=on"}).String()
	db, err := sql.Open("sqlite3", uri)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	out := &Store{db: db, auditReadOnly: true}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	// Query-only is a connection-level safeguard; even a future read
	// helper accidentally attempting DML must fail without any write.
	if _, err := db.Exec("PRAGMA query_only=ON"); err != nil {
		_ = db.Close()
		return nil, err
	}
	var readOnly int
	if err := db.QueryRowContext(context.Background(), "PRAGMA query_only").Scan(&readOnly); err != nil || readOnly != 1 {
		_ = db.Close()
		return nil, fmt.Errorf("SQLite audit connection did not enforce query-only mode: %v", err)
	}
	if err := out.validateExistingSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return out, nil
}
