package install

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"syscall"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/peerscope"
	"golang.org/x/sys/unix"
)

const scopedRetirementPreimageName = "retirement-preimage.json"

// Written by the original scoped fresh setup before its first Project,
// Profile or ACL effect. Older installs have only a digest and are never
// allowed to synthesize a preimage from today's native configuration.
type scopedRetirementPreimage struct {
	Schema        int
	Origin        nativePeerFreshJournalRecord
	BeforeProject api.ProjectPut
	BeforeProfile api.ProfilePut
	AfterProject  api.ProjectPut
	AfterProfile  api.ProfilePut
	Bridge        api.NetworkPut
	Quarantine    api.NetworkACLsPost
}

func scopedRetirementPreimageValid(p scopedRetirementPreimage, r nativePeerFreshJournalRecord) bool {
	r.Status = "QUARANTINE_DISPATCH_UNKNOWN"
	target := NativePrepareTarget{Project: r.Project, Pool: p.BeforeProfile.Devices["root"]["pool"], Network: r.Bridge, ACL: peerscope.QuarantineACL}
	want := nativePrepareProfile(target, p.BeforeProfile)
	want.Devices["eth0"]["security.acls.default.egress.action"] = "reject"
	acl, err := nativePeerFreshQuarantinePolicy(r, &api.Network{Name: r.Bridge, Type: "bridge", Managed: true, Status: "Created", NetworkPut: p.Bridge})
	return p.Schema == 1 && p.Origin == r && nativePeerFreshJournalRecordValid(r) &&
		p.BeforeProject.Config["restricted"] == "true" && p.BeforeProject.Config["features.networks"] == "false" &&
		p.BeforeProject.Config["restricted.networks.access"] == r.Bridge &&
		safePrereqName.MatchString(target.Pool) &&
		nativePrepareProfileEligible(p.BeforeProfile.Devices, target) == nil &&
		reflect.DeepEqual(p.AfterProject, nativePrepareProject(p.BeforeProject)) &&
		reflect.DeepEqual(p.AfterProfile, want) && err == nil && reflect.DeepEqual(p.Quarantine, acl)
}

func scopedRetirementPreimageWrite(d *os.File, p scopedRetirementPreimage) error {
	if !scopedRetirementPreimageValid(p, p.Origin) {
		return errors.New("invalid scoped retirement preimage")
	}
	data, err := json.Marshal(p)
	if err != nil || len(data) > 262143 {
		return errors.New("invalid scoped retirement preimage size")
	}
	fd, err := unix.Openat(int(d.Fd()), scopedRetirementPreimageName, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), scopedRetirementPreimageName)
	_, err = f.Write(append(data, '\n'))
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

func scopedRetirementPreimageRead(r nativePeerFreshJournalRecord) (scopedRetirementPreimage, error) {
	var p scopedRetirementPreimage
	d, err := nativePeerFreshJournalOpenAt(nativePeerFreshJournalDir, 0)
	if err != nil {
		return p, err
	}
	defer d.Close()
	fd, err := unix.Openat(int(d.Fd()), scopedRetirementPreimageName, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return p, errors.New("SCOPED_RETIREMENT_PREIMAGE_MISSING: no safe reconstruction; retain original installation")
	}
	f := os.NewFile(uintptr(fd), scopedRetirementPreimageName)
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return p, err
	}
	own, ok := st.Sys().(*syscall.Stat_t)
	if !ok || own.Uid != 0 || own.Nlink != 1 || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
		return p, errors.New("untrusted scoped retirement preimage")
	}
	data, err := io.ReadAll(io.LimitReader(f, 262145))
	if err != nil || len(data) > 262144 || json.Unmarshal(data, &p) != nil {
		return p, errors.New("unreadable scoped retirement preimage")
	}
	canonical, err := json.Marshal(p)
	if err != nil || !bytes.Equal(data, append(canonical, '\n')) || !scopedRetirementPreimageValid(p, r) {
		return p, errors.New("scoped retirement preimage identity mismatch")
	}
	return p, nil
}
