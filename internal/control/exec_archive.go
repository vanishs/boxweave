package control

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/vanishs/boxweave/internal/state"
	"golang.org/x/sys/unix"
)

// The controller archives only a bounded prefix of each verified native stream.
// The files belong to the private controller state directory, not to an Incus Box.
// A full archive is not a prerequisite for reporting a *verified* guest exit.
const (
	execArchiveStreamLimit int64 = 1 << 20
	execArchiveTotalLimit  int64 = 256 << 20
	execArchiveFileLimit         = 4096
	execArchiveSweepLimit        = 64
)

var (
	execArchiveLock sync.Mutex
	execArchiveName = regexp.MustCompile("^[0-9a-f]{32}\\.(stdout|stderr)$")
	execArchiveID   = regexp.MustCompile("^[0-9a-f]{32}$")
)

type recordedStreamRead func(context.Context, string, int64, int64) ([]byte, bool, error)

func execArchiveDirectory(st *state.Store, create bool) (string, error) {
	if st == nil || st.ExecArchiveDir() == "" {
		return "", errors.New("exec archive is not configured")
	}
	root := st.ExecArchiveDir()
	if create {
		if err := os.Mkdir(root, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return "", err
		}
	}
	info, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.IsDir() || info.Mode().Perm()&0077 != 0 ||
		owner.Uid != uint32(os.Geteuid()) {
		return "", errors.New("exec archive directory must be private and owned by the service")
	}
	return root, nil
}

func execArchiveFile(root, opID, stream string) (string, error) {
	if !execArchiveID.MatchString(opID) || (stream != "stdout" && stream != "stderr") {
		return "", state.ErrInvalid
	}
	return filepath.Join(root, opID+"."+stream), nil
}

func checkedArchiveFile(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 ||
		stat.Nlink != 1 || stat.Uid != uint32(os.Geteuid()) ||
		info.Size() < 0 || info.Size() > execArchiveStreamLimit {
		return errors.New("unsafe exec archive file")
	}
	return nil
}

// Capture runs before the verified Exec SQLite finalization while the original
// Box claim is still held. Archive failure does not falsify a native exit:
// output_get will fall back to native if possible and otherwise report missing.
func (s *Service) captureExecArchive(ctx context.Context, original, stdout, stderr string) {
	for _, item := range []struct{ stream, path string }{{"stdout", stdout}, {"stderr", stderr}} {
		if item.path == "" {
			continue
		}
		if err := s.captureExecStream(ctx, original, item.stream, item.path, s.Native.ReadRecordedOutput); err != nil {
			slog.WarnContext(context.Background(), "exec_archive_capture_unavailable",
				"operation_id", original, "stream", item.stream, "reason", "BOUNDED_ARCHIVE_UNAVAILABLE")
		}
	}
}

func (s *Service) captureExecStream(ctx context.Context, original, stream, nativePath string, reader recordedStreamRead) error {
	if reader == nil || nativePath == "" {
		return state.ErrInvalid
	}
	execArchiveLock.Lock()
	defer execArchiveLock.Unlock()
	root, err := execArchiveDirectory(s.Store, true)
	if err != nil {
		return err
	}
	target, err := execArchiveFile(root, original, stream)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(target); err == nil {
		return checkedArchiveFile(info) // verified immutable existing capture
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	if len(entries) >= execArchiveFileLimit {
		return errors.New("exec archive file count limit")
	}
	var used int64
	for _, entry := range entries {
		info, e := entry.Info()
		if e != nil || checkedArchiveFile(info) != nil {
			return errors.New("unsafe exec archive directory entry")
		}
		used += info.Size()
		if used > execArchiveTotalLimit {
			return errors.New("exec archive capacity exceeded")
		}
	}
	if used+execArchiveStreamLimit > execArchiveTotalLimit {
		return errors.New("exec archive capacity exceeded")
	}
	// Statfs is a narrow free-space preflight, NOT proof of Btrfs qgroup
	// enforcement. The installed host still needs independent quota review.
	var fs unix.Statfs_t
	if err := unix.Statfs(root, &fs); err != nil {
		return err
	}
	if fs.Bsize <= 0 || fs.Bavail < uint64(((64<<20)+execArchiveStreamLimit+fs.Bsize-1)/fs.Bsize) {
		return errors.New("exec archive free-space safety margin exhausted")
	}
	f, err := os.CreateTemp(root, "."+original+"."+stream+".tmp-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	var offset int64
	for offset < execArchiveStreamLimit {
		length := int64(65536)
		if execArchiveStreamLimit-offset < length {
			length = execArchiveStreamLimit - offset
		}
		data, ended, e := reader(ctx, nativePath, offset, length)
		if e != nil {
			_ = f.Close()
			return e
		}
		if int64(len(data)) > length || (!ended && len(data) == 0) {
			_ = f.Close()
			return errors.New("inconsistent native recorded stream")
		}
		if _, e = f.Write(data); e != nil {
			_ = f.Close()
			return e
		}
		offset += int64(len(data))
		if ended {
			break
		}
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// Linking a private temp file refuses overwrite of a prior original;
	// rename would silently replace an existing archived stream.
	if err := os.Link(f.Name(), target); err != nil {
		if errors.Is(err, os.ErrExist) {
			info, e := os.Lstat(target)
			if e == nil {
				return checkedArchiveFile(info)
			}
		}
		return err
	}
	d, err := os.Open(root)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// readExecArchive never opens a caller-provided path and refuses symbolic
// links/hardlinks. Its result is only exposed after original-operation and
// live TokenID authorization checks by operation_output_get.
func (s *Service) readExecArchive(original, stream string, offset, limit int64) ([]byte, bool, bool, bool, error) {
	root, err := execArchiveDirectory(s.Store, false)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, false, false, nil
	}
	if err != nil {
		return nil, false, false, false, err
	}
	path, err := execArchiveFile(root, original, stream)
	if err != nil {
		return nil, false, false, false, err
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil, false, false, false, nil
	}
	if err != nil {
		return nil, false, false, false, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, false, false, false, err
	}
	if err := checkedArchiveFile(info); err != nil {
		return nil, false, false, false, err
	}
	if offset >= info.Size() {
		return []byte{}, true, info.Size() >= execArchiveStreamLimit, true, nil
	}
	n := limit
	if n > info.Size()-offset {
		n = info.Size() - offset
	}
	buf := make([]byte, n)
	count, err := f.ReadAt(buf, offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false, false, false, err
	}
	buf = buf[:count]
	return buf, offset+int64(count) >= info.Size(), info.Size() >= execArchiveStreamLimit, true, nil
}

// SweepExecArchive removes only exact private archive names after the seven-day
// output witness TTL. Orphaned captures (finalize rolled back) use file mtime.
// It never issues native effects or removes the immutable SQLite operation.
func (s *Service) SweepExecArchive(ctx context.Context, now time.Time) (int, error) {
	execArchiveLock.Lock()
	defer execArchiveLock.Unlock()
	root, err := execArchiveDirectory(s.Store, false)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, err
	}
	expiredBefore := now.Add(-state.ExecRetention).UnixNano()
	removed := 0
	for _, entry := range entries {
		if ctx.Err() != nil {
			return removed, ctx.Err()
		}
		if removed >= execArchiveSweepLimit {
			break
		}
		name := entry.Name()
		if !execArchiveName.MatchString(name) && !strings.Contains(name, ".tmp-") {
			return removed, errors.New("unexpected exec archive directory entry")
		}
		info, err := entry.Info()
		if err != nil || checkedArchiveFile(info) != nil {
			return removed, errors.New("unsafe exec archive directory entry")
		}
		if !execArchiveName.MatchString(name) {
			if info.ModTime().UnixNano() > expiredBefore {
				continue
			}
		} else {
			opID := strings.SplitN(name, ".", 2)[0]
			witness, e := s.Store.ExecOutput(ctx, opID)
			switch {
			case e == nil:
				if witness.ObservedAt > expiredBefore {
					continue
				}
			case errors.Is(e, state.ErrNotFound):
				if info.ModTime().UnixNano() > expiredBefore {
					continue
				}
			default:
				return removed, e
			}
		}
		if err := os.Remove(filepath.Join(root, name)); err != nil {
			return removed, err
		}
		removed++
	}
	if removed > 0 {
		d, err := os.Open(root)
		if err != nil {
			return removed, err
		}
		err = d.Sync()
		_ = d.Close()
		if err != nil {
			return removed, fmt.Errorf("exec archive cleanup sync: %w", err)
		}
	}
	return removed, nil
}
