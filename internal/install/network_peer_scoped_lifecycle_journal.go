package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/vanishs/boxweave/internal/peerscope"
	"github.com/vanishs/boxweave/internal/state"
	"golang.org/x/sys/unix"
)

var scopedLifecycleID = regexp.MustCompile("^[0-9a-f]{32}$")

type scopedLifecycleRecord struct {
	Schema                                                                                       int
	Kind, Status, OriginalID, NativeID, HostSHA256, GuardSHA256, Project, Name, UUID, PlanSHA256 string
	Binding                                                                                      state.VerifiedCreateBinding
	Start, Internet, BootAutostart                                                               bool
}

func scopedLifecycleValid(r scopedLifecycleRecord) bool {
	acl, e := peerscope.ACLName(r.Project, r.Binding.Bridge, r.UUID)
	return e == nil && r.Schema == 2 && (r.Kind == "CREATE" || r.Kind == "DELETE") && scopedLifecycleID.MatchString(r.OriginalID) &&
		r.NativeID != "" && len(r.NativeID) <= 128 && safePrereqName.MatchString(r.Name) && r.Binding.ACL == acl &&
		nativePeerGuardSHA256Valid(r.HostSHA256) && nativePeerGuardSHA256Valid(r.GuardSHA256) && nativePeerGuardSHA256Valid(r.PlanSHA256) &&
		(r.Status == "DISPATCH_UNKNOWN" || r.Kind == "CREATE" && r.Status == "ACL_VERIFIED" || r.Status == "VERIFIED")
}
func scopedLifecyclePath(id string) (string, error) {
	if !scopedLifecycleID.MatchString(id) {
		return "", errors.New("invalid scoped lifecycle original")
	}
	return filepath.Join(nativePeerFreshJournalDir, "op-"+id), nil
}
func scopedLifecycleFile(status string) string {
	switch status {
	case "DISPATCH_UNKNOWN":
		return "intent.json"
	case "ACL_VERIFIED":
		return "acl-verified.json"
	case "VERIFIED":
		return "verified.json"
	}
	return ""
}
func scopedLifecycleWrite(d *os.File, r scopedLifecycleRecord) error {
	if d == nil || !scopedLifecycleValid(r) {
		return errors.New("invalid scoped lifecycle record")
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	fd, err := unix.Openat(int(d.Fd()), scopedLifecycleFile(r.Status), unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), "scoped lifecycle journal")
	_, err = f.Write(append(b, '\n'))
	if err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err != nil {
		return err
	}
	if ce != nil {
		return ce
	}
	return d.Sync()
}
func scopedLifecycleRead(d *os.File, status string) (scopedLifecycleRecord, error) {
	var zero scopedLifecycleRecord
	name := scopedLifecycleFile(status)
	if d == nil || name == "" {
		return zero, errors.New("invalid scoped lifecycle read")
	}
	fd, err := unix.Openat(int(d.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return zero, err
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return zero, err
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok || sys.Uid != 0 || sys.Nlink != 1 || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return zero, errors.New("untrusted scoped journal inode")
	}
	b, err := io.ReadAll(io.LimitReader(f, 8193))
	if err != nil || len(b) > 8192 {
		return zero, errors.New("invalid scoped lifecycle journal size")
	}
	var got scopedLifecycleRecord
	if json.Unmarshal(b, &got) != nil || !scopedLifecycleValid(got) || got.Status != status {
		return zero, errors.New("invalid scoped lifecycle journal")
	}
	raw, err := json.Marshal(got)
	if err != nil || !bytes.Equal(b, append(raw, '\n')) {
		return zero, errors.New("noncanonical scoped journal")
	}
	return got, nil
}
