package state

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// DiagnosticSnapshot is an observation-only handle to a private captured
// epoch. It cannot be supplied as a live Root authorization witness.
type DiagnosticSnapshot struct {
	store   *Store
	dir     string
	EpochNS int64
	Capture string
}

var (
	ErrDiagnosticMissing = errors.New("DIAGNOSTIC_DB_MISSING")
	ErrDiagnosticUnsafe  = errors.New("DIAGNOSTIC_DB_IDENTITY_UNVERIFIED")
	ErrDiagnosticBusy    = errors.New("DIAGNOSTIC_DB_BUSY_OR_UNSTABLE")
	ErrDiagnosticBound   = errors.New("DIAGNOSTIC_DB_CAPTURE_LIMIT")
	ErrDiagnosticInvalid = errors.New("DIAGNOSTIC_DB_SCHEMA_OR_INTEGRITY_UNVERIFIED")
)

const diagnosticMaxBytes int64 = 32 << 20
const diagnosticLockBudget = 2 * time.Second

// SQLite's pinned default Unix VFS (go-sqlite3 v1.14.52):
// PENDING_BYTE, RESERVED_BYTE, SHARED_FIRST/SHARED_SIZE; WAL write,
// checkpoint, recovery and deadman-switch offsets. OFD locks conflict
// with SQLite's POSIX locks. This collector MUST run in a standalone
// process without source SQLite connections (closing an fd can drop
// same-process POSIX locks). No SQLite
// connection is ever opened on the source by this collector.
// Supported only for the local cooperative BoxWeave SQLite writer.
const (
	diagnosticDBLock     = 0x40000000
	diagnosticDBLockLen  = 512
	diagnosticWALLock    = 120
	diagnosticWALLockLen = 3
	diagnosticDMS        = 128
)

func diagnosticLock(f *os.File, kind int16, start, length int64) error {
	l := unix.Flock_t{Type: kind, Whence: io.SeekStart, Start: start, Len: length}
	if err := unix.FcntlFlock(f.Fd(), unix.F_OFD_SETLK, &l); err != nil {
		return ErrDiagnosticBusy
	}
	return nil
}
func diagnosticLiveShm(f *os.File) bool {
	l := unix.Flock_t{Type: unix.F_WRLCK, Whence: io.SeekStart, Start: diagnosticDMS, Len: 1}
	return unix.FcntlFlock(f.Fd(), unix.F_OFD_GETLK, &l) == nil && l.Type == unix.F_RDLCK
}
func diagnosticLocalFS(fd int) bool {
	var st unix.Statfs_t
	if unix.Fstatfs(fd, &st) != nil {
		return false
	}
	switch uint64(st.Type) {
	case 0xef53, 0x58465342, 0x9123683e, 0x01021994, 0x794c7630, 0x2fc12fc1:
		return true // ext, XFS, btrfs, tmpfs, overlay, ZFS (locks still required)
	}
	return false
}
func diagnosticDir(path string) (*os.File, unix.Stat_t, error) {
	var st unix.Stat_t
	fd, e := unix.Openat2(unix.AT_FDCWD, path, &unix.OpenHow{Flags: unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW, Resolve: unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_MAGICLINKS})
	if errors.Is(e, unix.ENOENT) {
		return nil, st, ErrDiagnosticMissing
	}
	if e != nil {
		return nil, st, ErrDiagnosticUnsafe
	}
	f := os.NewFile(uintptr(fd), "diagnostic-source-directory")
	if unix.Fstat(fd, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Mode&0077 != 0 || os.Geteuid() != 0 && st.Uid != uint32(os.Geteuid()) || !diagnosticLocalFS(fd) {
		f.Close()
		return nil, st, ErrDiagnosticUnsafe
	}
	return f, st, nil
}
func diagnosticFile(dir *os.File, owner unix.Stat_t, name string, flags int) (*os.File, unix.Stat_t, error) {
	var st unix.Stat_t
	fd, e := unix.Openat(int(dir.Fd()), name, flags|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_NOATIME, 0)
	if errors.Is(e, unix.ENOENT) {
		return nil, st, ErrDiagnosticMissing
	}
	if e != nil {
		return nil, st, ErrDiagnosticUnsafe
	}
	f := os.NewFile(uintptr(fd), "diagnostic-source-file")
	if unix.Fstat(fd, &st) != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Mode&0077 != 0 || st.Nlink != 1 || st.Uid != owner.Uid || st.Dev != owner.Dev || !diagnosticLocalFS(fd) {
		f.Close()
		return nil, st, ErrDiagnosticUnsafe
	}
	return f, st, nil
}
func diagnosticSame(a, b unix.Stat_t) bool {
	return a.Dev == b.Dev && a.Ino == b.Ino && a.Uid == b.Uid && a.Gid == b.Gid && a.Mode == b.Mode && a.Nlink == b.Nlink && a.Size == b.Size && a.Mtim == b.Mtim && a.Ctim == b.Ctim
}
func diagnosticPathSame(dir *os.File, name string, f *os.File, old unix.Stat_t) bool {
	var a, b unix.Stat_t
	return unix.Fstat(int(f.Fd()), &a) == nil && unix.Fstatat(int(dir.Fd()), name, &b, unix.AT_SYMLINK_NOFOLLOW) == nil && diagnosticSame(old, a) && diagnosticSame(a, b)
}
func diagnosticAbsent(dir *os.File, name string) bool {
	var st unix.Stat_t
	return errors.Is(unix.Fstatat(int(dir.Fd()), name, &st, unix.AT_SYMLINK_NOFOLLOW), unix.ENOENT)
}
func diagnosticCopy(ctx context.Context, f *os.File, size int64, dst string) error {
	if size < 0 || size > diagnosticMaxBytes {
		return ErrDiagnosticBound
	}
	out, e := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return ErrDiagnosticBound
	}
	defer out.Close()
	buf := make([]byte, 128<<10)
	for offset := int64(0); offset < size; {
		if ctx.Err() != nil {
			return ErrDiagnosticBound
		}
		n := int(min(int64(len(buf)), size-offset))
		if _, e = f.ReadAt(buf[:n], offset); e != nil {
			return ErrDiagnosticBusy
		}
		if _, e = out.Write(buf[:n]); e != nil {
			return ErrDiagnosticBound
		}
		offset += int64(n)
	}
	if ctx.Err() != nil {
		return ErrDiagnosticBound
	}
	return out.Close()
}

// captureDiagnosticDB holds only nonblocking, short-lived cooperative
// filesystem locks while copying DB+WAL. It never creates or maps source
// sidecars, checkpoints, sets SQLite pragmas on source, or takes a serve lock.
// Cold capture requires exclusive SQLite DB range (open existing RW for the
// lock only, no writes); live capture requires the active WAL index and
// shared locks excluding all writer/checkpoint/recovery locks. Unsupported,
// busy, unsafe or oversized captures fail closed, not stale PASS.
func captureDiagnosticDB(ctx context.Context, path, dst string) (string, int64, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsAny(path, "?#\x00") {
		return "", 0, ErrDiagnosticUnsafe
	}
	parent, name := filepath.Dir(path), filepath.Base(path)
	dir, owner, e := diagnosticDir(parent)
	if e != nil {
		return "", 0, e
	}
	defer dir.Close()
	db, dbst, e := diagnosticFile(dir, owner, name, unix.O_RDONLY)
	if e != nil {
		return "", 0, e
	}
	defer db.Close()
	mode := "SQLITE_WAL_LOCKED_COPY"
	// Try cold proof before acquiring the shared DB lock on the RO fd.
	rw, rwst, re := diagnosticFile(dir, owner, name, unix.O_RDWR)
	if re == nil {
		defer rw.Close()
		if !diagnosticSame(dbst, rwst) {
			return "", 0, ErrDiagnosticBusy
		}
		if diagnosticLock(rw, unix.F_WRLCK, diagnosticDBLock, diagnosticDBLockLen) == nil {
			mode = "SQLITE_EXCLUSIVE_COLD_COPY"
		}
	}
	var shm *os.File
	var shmst unix.Stat_t
	if mode != "SQLITE_EXCLUSIVE_COLD_COPY" {
		if e = diagnosticLock(db, unix.F_RDLCK, diagnosticDBLock, diagnosticDBLockLen); e != nil {
			return "", 0, e
		}
		shm, shmst, e = diagnosticFile(dir, owner, name+"-shm", unix.O_RDONLY)
		if e != nil || shm == nil {
			return "", 0, ErrDiagnosticBusy
		}
		defer shm.Close()
		if !diagnosticLiveShm(shm) || diagnosticLock(shm, unix.F_RDLCK, diagnosticWALLock, diagnosticWALLockLen) != nil || !diagnosticLiveShm(shm) {
			return "", 0, ErrDiagnosticBusy
		}
	}
	// Refresh main stat AFTER acquiring the locks, since a legitimate
	// checkpoint might have completed while opening descriptors.
	if unix.Fstat(int(db.Fd()), &dbst) != nil || !diagnosticPathSame(dir, name, db, dbst) || !diagnosticAbsent(dir, name+"-journal") {
		return "", 0, ErrDiagnosticBusy
	}
	if mode == "SQLITE_EXCLUSIVE_COLD_COPY" && !diagnosticPathSame(dir, name, rw, dbst) {
		return "", 0, ErrDiagnosticBusy
	}
	if shm != nil {
		if unix.Fstat(int(shm.Fd()), &shmst) != nil || !diagnosticPathSame(dir, name+"-shm", shm, shmst) {
			return "", 0, ErrDiagnosticBusy
		}
	}
	wal, walst, we := diagnosticFile(dir, owner, name+"-wal", unix.O_RDONLY)
	if we != nil && !errors.Is(we, ErrDiagnosticMissing) {
		return "", 0, we
	}
	if wal == nil && shm != nil {
		return "", 0, ErrDiagnosticBusy
	}
	if wal != nil {
		defer wal.Close()
	}
	// Refuse unsafe stale SHM even when cold; we never copy it.
	if shm == nil && !diagnosticAbsent(dir, name+"-shm") {
		f, _, err := diagnosticFile(dir, owner, name+"-shm", unix.O_RDONLY)
		if err != nil {
			return "", 0, err
		}
		f.Close()
	}
	if dbst.Size <= 0 || dbst.Size > diagnosticMaxBytes || walst.Size < 0 || walst.Size > diagnosticMaxBytes-dbst.Size {
		return "", 0, ErrDiagnosticBound
	}
	copyCtx, cancel := context.WithTimeout(ctx, diagnosticLockBudget)
	defer cancel()
	epoch := time.Now().UnixNano()
	if e = diagnosticCopy(copyCtx, db, dbst.Size, dst); e != nil {
		return "", 0, e
	}
	if wal != nil {
		if e = diagnosticCopy(copyCtx, wal, walst.Size, dst+"-wal"); e != nil {
			return "", 0, e
		}
		if !diagnosticPathSame(dir, name+"-wal", wal, walst) {
			return "", 0, ErrDiagnosticBusy
		}
	} else if !diagnosticAbsent(dir, name+"-wal") {
		return "", 0, ErrDiagnosticBusy
	}
	if shm != nil {
		// SHM read marks can legitimately change; compare identity, not its
		// contents or timestamps. No writer/checkpoint/recovery can pass locks.
		var current unix.Stat_t
		if unix.Fstatat(int(dir.Fd()), name+"-shm", &current, unix.AT_SYMLINK_NOFOLLOW) != nil || current.Dev != shmst.Dev || current.Ino != shmst.Ino || current.Uid != shmst.Uid || current.Mode != shmst.Mode || current.Nlink != 1 || !diagnosticLiveShm(shm) {
			return "", 0, ErrDiagnosticBusy
		}
	}
	if !diagnosticPathSame(dir, name, db, dbst) || !diagnosticAbsent(dir, name+"-journal") || copyCtx.Err() != nil {
		return "", 0, ErrDiagnosticBusy
	}
	again, st, e := diagnosticDir(parent)
	if e != nil {
		return "", 0, ErrDiagnosticBusy
	}
	again.Close()
	if st.Dev != owner.Dev || st.Ino != owner.Ino || st.Uid != owner.Uid || st.Mode != owner.Mode {
		return "", 0, ErrDiagnosticBusy
	}
	return mode, epoch, nil
}

func OpenDiagnosticSnapshot(ctx context.Context, path string) (*DiagnosticSnapshot, error) {
	if ctx.Err() != nil {
		return nil, ErrDiagnosticBound
	}
	// Use fixed volatile tmpfs, not caller-controlled TMPDIR. Snapshot may
	// contain secrets; never expose its location, remove on every normal path.
	var fs unix.Statfs_t
	st, e := os.Lstat("/dev/shm")
	if e != nil || !st.IsDir() || st.Mode()&os.ModeSticky == 0 || unix.Statfs("/dev/shm", &fs) != nil || uint64(fs.Type) != 0x01021994 {
		return nil, ErrDiagnosticUnsafe
	}
	dir, e := os.MkdirTemp("/dev/shm", "boxweave-doctor-")
	if e != nil {
		return nil, ErrDiagnosticBound
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(dir)
		}
	}()
	dst := filepath.Join(dir, "ledger.sqlite")
	mode, epoch, e := captureDiagnosticDB(ctx, path, dst)
	if e != nil {
		return nil, e
	}
	// Rebuild private WAL index using SQLite; never copy SHM. Use a
	// context-bounded query-only connection and existing schema validator.
	uri := (&url.URL{Scheme: "file", Path: dst, RawQuery: "mode=ro&_query_only=1&_busy_timeout=500&_foreign_keys=on"}).String()
	conn, e := sql.Open("sqlite3", uri)
	if e != nil {
		return nil, ErrDiagnosticInvalid
	}
	conn.SetMaxOpenConns(1)
	store := &Store{db: conn} // not a live Root audit handle
	if e = conn.PingContext(ctx); e == nil {
		e = store.validateExistingSchemaContext(ctx)
	}
	if e != nil {
		conn.Close()
		return nil, ErrDiagnosticInvalid
	}
	ok = true
	return &DiagnosticSnapshot{store: store, dir: dir, EpochNS: epoch, Capture: mode}, nil
}
func (s *DiagnosticSnapshot) Close() error {
	e := s.store.Close()
	ce := os.RemoveAll(s.dir)
	if e != nil {
		return e
	}
	return ce
}
func (s *DiagnosticSnapshot) Schema(ctx context.Context) (int, error) { return s.store.Schema(ctx) }
func (s *DiagnosticSnapshot) ProjectCapacity(ctx context.Context, project string) (ProjectBoxCapacity, error) {
	return s.store.ReadProjectBoxCapacity(ctx, project)
}

// ScopedLedgerSummary is diagnostic-only and permits a healthy empty
// scope. It does not loosen the mutation-grade Root validator.
type DiagnosticScopedSummary struct {
	Owned   int `json:"owned"`
	Retired int `json:"retired"`
}

func (s *DiagnosticSnapshot) ScopedLedgerSummary(ctx context.Context, project, bridge string) (DiagnosticScopedSummary, error) {
	var zero DiagnosticScopedSummary
	tx, e := s.store.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return zero, e
	}
	defer tx.Rollback()
	l, e := readCompleteScopedNetworkAuditLedgerModeTx(ctx, tx, project, bridge, true)
	if e != nil {
		return zero, e
	}
	if e = tx.Commit(); e != nil {
		return zero, e
	}
	return DiagnosticScopedSummary{Owned: len(l.Boxes), Retired: len(l.Retired)}, nil
}
