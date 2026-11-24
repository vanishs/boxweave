package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// Separate from the controller-owned SQLite directory. Neither the restricted
// daemon nor an MCP caller may create or supply this installation witness.
const nativePeerFreshJournalDir = "/var/lib/boxweave-scoped-peer-v2"

type nativePeerFreshJournalRecord struct {
	Schema                                                       int
	Mode, HostSHA256, Project, Bridge, NativeSHA256, GuardSHA256 string
	Status                                                       string
}

func nativePeerFreshJournalRecordValid(r nativePeerFreshJournalRecord) bool {
	return r.Schema == 1 && r.Mode == nativePeerFreshMode &&
		safePrereqName.MatchString(r.Project) && safePrereqName.MatchString(r.Bridge) &&
		nativePeerGuardSHA256Valid(r.HostSHA256) && nativePeerGuardSHA256Valid(r.NativeSHA256) &&
		nativePeerGuardSHA256Valid(r.GuardSHA256) &&
		(r.Status == "DISPATCH_UNKNOWN" || r.Status == "INITIAL_GUARD_VERIFIED")
}

// The production directory and owner are fixed; these parameters are only
// a package-private filesystem test seam. Never adopt an existing directory.
func nativePeerFreshJournalCreateAt(dir string, uid uint32) (*os.File, error) {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return nil, errors.New("invalid fresh journal path")
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		return nil, err
	}
	d, err := nativePeerFreshJournalOpenAt(dir, uid)
	if err != nil {
		return nil, err
	}
	parent, err := os.Open(filepath.Dir(dir))
	if err != nil {
		d.Close()
		return nil, err
	}
	err = parent.Sync()
	closeErr := parent.Close()
	if err != nil {
		d.Close()
		return nil, err
	}
	if closeErr != nil {
		d.Close()
		return nil, closeErr
	}
	return d, nil
}

func nativePeerFreshJournalOpenAt(dir string, uid uint32) (*os.File, error) {
	fd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	d := os.NewFile(uintptr(fd), dir)
	st, err := d.Stat()
	if err != nil {
		d.Close()
		return nil, err
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok || sys.Uid != uid || !st.IsDir() || st.Mode().Perm() != 0700 {
		d.Close()
		return nil, errors.New("untrusted fresh journal directory")
	}
	return d, nil
}

func nativePeerFreshJournalName(status string) string {
	if status == "DISPATCH_UNKNOWN" {
		return "intent.json"
	}
	if status == "INITIAL_GUARD_VERIFIED" {
		return "guard-verified.json"
	}
	return ""
}

func nativePeerFreshJournalWrite(d *os.File, r nativePeerFreshJournalRecord) error {
	if d == nil || !nativePeerFreshJournalRecordValid(r) {
		return errors.New("invalid fresh journal record")
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	fd, err := unix.Openat(int(d.Fd()), nativePeerFreshJournalName(r.Status), unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), "fresh journal record")
	_, err = f.Write(append(data, '\n'))
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return d.Sync()
}

func nativePeerFreshJournalRead(d *os.File, status string) (nativePeerFreshJournalRecord, error) {
	var zero nativePeerFreshJournalRecord
	name := nativePeerFreshJournalName(status)
	if d == nil || name == "" {
		return zero, errors.New("invalid fresh journal read")
	}
	fd, err := unix.Openat(int(d.Fd()), name, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return zero, err
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	ds, err := d.Stat()
	if err != nil {
		return zero, err
	}
	st, err := f.Stat()
	if err != nil {
		return zero, err
	}
	dsStat, ok := ds.Sys().(*syscall.Stat_t)
	stStat, ok2 := st.Sys().(*syscall.Stat_t)
	if !ok || !ok2 || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || stStat.Nlink != 1 || stStat.Uid != dsStat.Uid {
		return zero, errors.New("untrusted fresh journal inode")
	}
	data, err := io.ReadAll(io.LimitReader(f, 8193))
	if err != nil || len(data) > 8192 {
		return zero, errors.New("unreadable or oversized fresh journal")
	}
	var got nativePeerFreshJournalRecord
	if err = json.Unmarshal(data, &got); err != nil || !nativePeerFreshJournalRecordValid(got) || got.Status != status {
		return zero, errors.New("invalid fresh journal content")
	}
	canonical, err := json.Marshal(got)
	if err != nil || !bytes.Equal(data, append(canonical, '\n')) {
		return zero, errors.New("noncanonical fresh journal")
	}
	return got, nil
}

// Historical evidence of ONE initial guard, never current kernel readiness.
func nativePeerFreshJournalVerifiedAt(dir string, uid uint32, want nativePeerFreshJournalRecord) error {
	if !nativePeerFreshJournalRecordValid(want) || want.Status != "INITIAL_GUARD_VERIFIED" {
		return errors.New("invalid expected initial guard witness")
	}
	d, err := nativePeerFreshJournalOpenAt(dir, uid)
	if err != nil {
		return err
	}
	defer d.Close()
	intent, err := nativePeerFreshJournalRead(d, "DISPATCH_UNKNOWN")
	if err != nil {
		return err
	}
	receipt, err := nativePeerFreshJournalRead(d, "INITIAL_GUARD_VERIFIED")
	if err != nil {
		return err
	}
	intent.Status = "INITIAL_GUARD_VERIFIED"
	if intent != want || receipt != want {
		return errors.New("fresh guard receipt does not bind exact initial intent")
	}
	return nil
}
