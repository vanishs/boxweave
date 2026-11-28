package state

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/mattn/go-sqlite3"
	"golang.org/x/sys/unix"
)

func inspectNetworkAuditIdentity(path string) (os.FileInfo, os.FileInfo, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path ||
		strings.ContainsAny(path, "?#\x00") {
		return nil, nil, errors.New("absolute canonical literal audit database path required")
	}
	parent := filepath.Dir(path)
	physical, err := filepath.EvalSymlinks(parent)
	if err != nil || physical != parent {
		return nil, nil, errors.New("audit database parent missing or contains symlink alias")
	}
	pstat, err := os.Lstat(parent)
	if err != nil || !pstat.IsDir() || pstat.Mode().Perm()&0077 != 0 {
		return nil, nil, errors.New("audit database parent must be a private real directory")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm()&0077 != 0 {
		return nil, nil, errors.New("audit database must be a private existing regular file")
	}
	ino, ok := before.Sys().(*syscall.Stat_t)
	if !ok || ino.Nlink != 1 {
		return nil, nil, errors.New("audit database hardlinks or unknown inode refused")
	}
	return pstat, before, nil
}

// A protected audit pool may reopen its SQLite connection after idle release.
// Bind each new connection to the same verified directory and database inode.
type networkAuditConnector struct {
	path, uri                string
	parent, file             os.FileInfo
	driver                   *sqlite3.SQLiteDriver
	parentAnchor, fileAnchor *os.File
	closeOnce                sync.Once
	closeErr                 error
}

func (c *networkAuditConnector) close() error {
	c.closeOnce.Do(func() { c.closeErr = errors.Join(c.fileAnchor.Close(), c.parentAnchor.Close()) })
	return c.closeErr
}

// Pin inodes with O_PATH, not SQLite readers or ordinary data descriptors.
// O_PATH does not participate in POSIX file locks when closed. This prevents
// inode reuse while the pool is idle, without disturbing SQLite locks.
func pinNetworkAuditConnector(path, uri string, parent, file os.FileInfo) (*networkAuditConnector, error) {
	dir, err := os.OpenFile(filepath.Dir(path), unix.O_PATH|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, unix.O_PATH|unix.O_NOFOLLOW, 0)
	if err != nil {
		_ = dir.Close()
		return nil, err
	}
	c := &networkAuditConnector{path: path, uri: uri, parent: parent, file: file, driver: &sqlite3.SQLiteDriver{}, parentAnchor: dir, fileAnchor: f}
	ds, de := dir.Stat()
	fs, fe := f.Stat()
	if de != nil || fe != nil || !os.SameFile(parent, ds) || !os.SameFile(file, fs) {
		_ = c.close()
		return nil, errors.New("audit database or directory identity changed while pinning")
	}
	if err := c.verifyIdentity(); err != nil {
		_ = c.close()
		return nil, err
	}
	return c, nil
}

func (c *networkAuditConnector) Driver() driver.Driver { return c.driver }

func (c *networkAuditConnector) verifyIdentity() error {
	parent, file, err := inspectNetworkAuditIdentity(c.path)
	if err != nil {
		return err
	}
	if !os.SameFile(c.parent, parent) || !os.SameFile(c.file, file) {
		return errors.New("audit database or directory identity changed")
	}
	return nil
}

func (c *networkAuditConnector) Connect(ctx context.Context) (driver.Conn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.verifyIdentity(); err != nil {
		return nil, err
	}
	conn, err := c.driver.Open(c.uri)
	if err != nil {
		return nil, err
	}
	if err = c.verifyIdentity(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

// OpenNetworkAuditReadOnly opens ONLY an existing protected owner database.
// It never creates directories/DBs or performs writable WAL initialization.
// This root-audit reader grants no ability to admit or mutate state.
func OpenNetworkAuditReadOnly(path string) (*Store, error) {
	parent, file, err := inspectNetworkAuditIdentity(path)
	if err != nil {
		return nil, err
	}
	uri := (&url.URL{Scheme: "file", Path: path,
		RawQuery: "mode=ro&_query_only=1&_busy_timeout=5000&_foreign_keys=on"}).String()
	connector, err := pinNetworkAuditConnector(path, uri, parent, file)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)
	out := &Store{db: db, auditReadOnly: true, auditConnector: connector}
	if err := db.Ping(); err != nil {
		_ = out.Close()
		return nil, err
	}
	// Query-only is a connection-level safeguard; even a future read
	// helper accidentally attempting DML must fail without any write.
	if _, err := db.Exec("PRAGMA query_only=ON"); err != nil {
		_ = out.Close()
		return nil, err
	}
	var readOnly int
	if err := db.QueryRowContext(context.Background(), "PRAGMA query_only").Scan(&readOnly); err != nil || readOnly != 1 {
		_ = out.Close()
		return nil, fmt.Errorf("SQLite audit connection did not enforce query-only mode: %v", err)
	}
	if err := out.validateExistingSchema(); err != nil {
		_ = out.Close()
		return nil, err
	}
	// A lifetime-idle SQLite reader can pin the old WAL-index after the
	// controller dies between WAL sync and commit publication. Release
	// physical connections when queries/transactions finish; retain the
	// read-only URI, startup schema check and original file identity.
	db.SetMaxIdleConns(0)
	return out, nil
}
