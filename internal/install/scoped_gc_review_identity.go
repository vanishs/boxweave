package install

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
)

// VerifyScopedGCReviewIdentity rechecks the public/static installed evidence
// against the exact manifest already verified by the local Root parent.
// It is NOT a Root journal proof or an authorization function. The parent
// must hold the maintenance lock and call Status before passing its verified
// manifest over the existing credential-checked recovery socket. Only the
// exact GC review actions use this path, including while the service is down.
func VerifyScopedGCReviewIdentity(p Plan, reviewed Manifest) error {
	if err := safe(p); err != nil {
		return err
	}
	uid, _, err := ServiceIdentity(p)
	if err != nil {
		return err
	}
	if p.User != "boxweave" || p.Socket != "/var/lib/incus/unix.socket.user" || p.Project != "user-"+strconv.Itoa(uid) ||
		(os.Geteuid() != 0 && os.Geteuid() != uid) {
		return errors.New("scoped GC review requires the fixed installed restricted identity")
	}
	if err := CheckScopedServeMode(true); err != nil {
		return err
	}
	raw, _, err := diagnosticReadFile(manifestPath(p), uint32(uid), true, 65536)
	if err != nil {
		return err
	}
	var current Manifest
	if json.Unmarshal(raw, &current) != nil || !matchScopedGCReviewManifest(p, reviewed, current) {
		return errors.New("scoped GC reviewed manifest changed")
	}
	binary, binaryStat, err := diagnosticReadFile(p.BinaryPath, 0, false, 128<<20)
	if err != nil || diagnosticHash(binary) != reviewed.BinarySHA256 {
		return errors.New("scoped GC installed binary identity unverified")
	}
	running, err := os.Stat("/proc/self/exe")
	if err != nil || !os.SameFile(binaryStat, running) {
		return errors.New("scoped GC reviewer is not the installed executable")
	}
	unit, _, err := diagnosticReadFile(p.UnitPath, 0, false, 65536)
	if err != nil || diagnosticHash(unit) != reviewed.UnitSHA256 {
		return errors.New("scoped GC installed unit identity unverified")
	}
	// The same three fixed Root-owned files were checked by Status. Read
	// their static bytes again without entering the protected Root journal.
	for _, u := range []struct{ path, body string }{{scopedGuardUnitPath, scopedGuardUnitText}, {scopedLifecycleUnitPath, scopedLifecycleUnitText}, {scopedIncusDropIn, scopedIncusDropInText}} {
		b, _, err := diagnosticReadFile(u.path, 0, false, 65536)
		if err != nil || string(b) != u.body {
			return errors.New("scoped GC fixed unit identity changed")
		}
	}
	return nil
}

func matchScopedGCReviewManifest(p Plan, reviewed, current Manifest) bool {
	if reviewed.Schema != manifestSchema || reviewed.ScopedPeerMode != nativePeerFreshMode ||
		reviewed.StateDir != p.StateDir || reviewed.BinaryPath != p.BinaryPath ||
		reviewed.Project != p.Project || reviewed.ServiceUser != p.User || reviewed.Socket != p.Socket ||
		reviewed.RetirementUnitSHA256 != "" || reviewed.RetirementBridge != "" || reviewed.RetirementACL != "" || reviewed.RetirementNativeReviewSHA != "" ||
		!validRetirementReviewSHA(reviewed.BinarySHA256) || !validRetirementReviewSHA(reviewed.UnitSHA256) {
		return false
	}
	// Status observes service activity, unlike the persisted manifest. No
	// other reviewed field is exempt from the exact comparison.
	current.ServiceActive = reviewed.ServiceActive
	return reviewed == current
}
