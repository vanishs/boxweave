package install

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const scopedGuardUnitName = "boxweave-scoped-guard.service"
const scopedGuardUnitPath = "/etc/systemd/system/" + scopedGuardUnitName
const scopedLifecycleUnitName = "boxweave-scoped-lifecycle.service"
const scopedLifecycleUnitPath = "/etc/systemd/system/" + scopedLifecycleUnitName
const scopedIncusDropIn = "/etc/systemd/system/incus.service.d/50-boxweave-scoped-guard.conf"

const scopedGuardUnitText = "[Unit]\nDescription=BoxWeave scoped initial guard before native autostart\nAfter=nftables.service\nBefore=incus.service incus-startup.service\n\n[Service]\nType=oneshot\nExecStart=/usr/local/bin/boxweave native-scoped-guard-boot\nRemainAfterExit=yes\nUMask=0077\n"
const scopedLifecycleUnitText = "[Unit]\nDescription=BoxWeave scoped lifecycle Root verifier\nRequires=incus.service boxweave-scoped-guard.service\nAfter=incus.service boxweave-scoped-guard.service\n\n[Service]\nType=simple\nExecStart=/usr/local/bin/boxweave native-scoped-lifecycle-serve\nRestart=on-failure\nRestartSec=3s\nUMask=0077\n"
const scopedIncusDropInText = "[Unit]\nRequires=boxweave-scoped-guard.service\nAfter=boxweave-scoped-guard.service\n"

func NativeScopedFreshReview(ctx context.Context) (NativePeerFreshGuardReview, error) {
	if err := scopedFreshEffectiveAbsent(); err != nil {
		return NativePeerFreshGuardReview{}, err
	}
	if err := verifyScopedPrerequisiteExecutables(); err != nil {
		return NativePeerFreshGuardReview{}, err
	}
	return nativePeerFreshGuardReview(ctx)
}
func NativeScopedGuardBoot(ctx context.Context) error {
	r, _, err := nativePeerFreshVerifiedPlan()
	if err != nil {
		return err
	}
	if err = nativePeerFreshQuarantineReceipt(r); err != nil {
		return err
	}
	return nativePeerFreshGuardRestore(ctx)
}
func scopedFreshEffectiveAbsent() error {
	if os.Geteuid() != 0 {
		return errors.New("scoped fresh install requires local root")
	}
	if err := nativePeerFreshPathsAbsent(os.Lstat); err != nil {
		return err
	}
	for _, unit := range []string{"boxweave.service", filepath.Base(nativePeerRootUnitPath), nativeRetirementUnitName, scopedGuardUnitName, scopedLifecycleUnitName} {
		out, err := exec.Command("systemctl", "show", "-p", "LoadState", "--value", unit).Output()
		if err != nil || strings.TrimSpace(string(out)) != "not-found" {
			return errors.New("existing or unverified effective service: " + unit)
		}
	}
	return nil
}
func scopedFreshPrepare(p Plan) error {
	if p.RetirementTarget != nil || p.User != "boxweave" || !nativePeerGuardSHA256Valid(p.ScopedFreshReviewSHA) {
		return errors.New("scoped fresh requires exact reviewed dedicated installation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	r, err := NativeScopedFreshReview(ctx)
	if err != nil {
		return err
	}
	if r.Project != p.Project || r.ReviewSHA256 != p.ScopedFreshReviewSHA {
		return errors.New("scoped fresh reviewed scope changed")
	}
	if _, err = nativePeerFreshGuardApply(ctx, r.ReviewSHA256, true); err != nil {
		return err
	}
	return nativePeerFreshQuarantineSetup(ctx)
}
func scopedWriteExclusive(path, body string) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return err
	}
	physical, err := filepath.EvalSymlinks(parent)
	if err != nil || physical != parent {
		return errors.New("untrusted scoped unit parent")
	}
	st, err := os.Lstat(parent)
	if err != nil {
		return err
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok || sys.Uid != 0 || !st.IsDir() || st.Mode().Perm()&0022 != 0 {
		return errors.New("unsafe scoped unit parent")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.WriteString(body)
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
	d, err := os.Open(parent)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// CheckScopedServeMode prevents an installed scoped journal from silently
// entering the legacy controller, including when an operator omits the flag.
func CheckScopedServeMode(enabled bool) error {
	st, err := os.Lstat(nativePeerFreshJournalDir)
	if errors.Is(err, os.ErrNotExist) {
		if enabled {
			return errors.New("scoped installation journal missing")
		}
		return refuseScopedLegacy()
	}
	if err != nil {
		return err
	}
	owner, ok := st.Sys().(*syscall.Stat_t)
	if !ok || owner.Uid != 0 || !st.IsDir() || st.Mode().Perm() != 0700 || !enabled {
		return errors.New("scoped journal requires scoped controller mode")
	}
	return nil
}

func refuseScopedLegacy() error {
	return refuseScopedLegacyPaths(os.Lstat)
}

func refuseScopedLegacyPaths(lstat func(string) (fs.FileInfo, error)) error {
	for _, path := range []string{nativePeerFreshJournalDir, scopedGuardUnitPath,
		scopedLifecycleUnitPath, scopedIncusDropIn, scopedLifecycleSocket} {
		if info, err := lstat(path); info != nil || !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("scoped fresh evidence exists or cannot be inspected: %s; legacy maintenance unsupported", path)
		}
	}
	return nil
}
func scopedStageUnits() error {
	for _, u := range []struct{ path, body string }{{scopedGuardUnitPath, scopedGuardUnitText}, {scopedLifecycleUnitPath, scopedLifecycleUnitText}, {scopedIncusDropIn, scopedIncusDropInText}} {
		if err := scopedWriteExclusive(u.path, u.body); err != nil {
			return err
		}
	}
	return nil
}
func verifyScopedInstalled(m Manifest) error {
	if m.ScopedPeerMode == "" {
		return refuseScopedLegacy()
	}
	if m.ScopedPeerMode != nativePeerFreshMode || m.RetirementUnitSHA256 != "" {
		return errors.New("unexpected scoped installer identity")
	}
	r, _, err := nativePeerFreshVerifiedPlan()
	if err != nil {
		return err
	}
	if r.Project != m.Project {
		return errors.New("scoped installer project mismatch")
	}
	if err = nativePeerFreshQuarantineReceipt(r); err != nil {
		return err
	}
	for _, u := range []struct{ path, body string }{{scopedGuardUnitPath, scopedGuardUnitText}, {scopedLifecycleUnitPath, scopedLifecycleUnitText}, {scopedIncusDropIn, scopedIncusDropInText}} {
		st, err := os.Lstat(u.path)
		if err != nil {
			return err
		}
		own, ok := st.Sys().(*syscall.Stat_t)
		if !ok || own.Uid != 0 || own.Nlink != 1 || !st.Mode().IsRegular() || st.Mode().Perm()&0022 != 0 {
			return errors.New("untrusted scoped unit")
		}
		body, err := os.ReadFile(u.path)
		if err != nil || string(body) != u.body {
			return errors.New("scoped unit identity changed")
		}
	}
	return nil
}
func activateScopedGuard() error {
	if err := runLocal("systemctl", "start", scopedGuardUnitName); err != nil {
		return err
	}
	for _, prop := range []string{"Requires", "After"} {
		raw, err := exec.Command("systemctl", "show", "-p", prop, "--value", "incus.service").Output()
		if err != nil {
			return err
		}
		found := false
		for _, v := range strings.Fields(string(raw)) {
			if v == scopedGuardUnitName {
				found = true
			}
		}
		if !found {
			return errors.New("effective Incus boot guard dependency missing")
		}
	}
	return nil
}
func observeScopedInstalled(m Manifest) error {
	if m.ScopedPeerMode == "" {
		return nil
	}
	if err := verifyScopedInstalled(m); err != nil {
		return err
	}
	for _, name := range []string{scopedGuardUnitName, scopedLifecycleUnitName} {
		if err := runLocal("systemctl", "is-active", "--quiet", name); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r, _, err := nativePeerFreshVerifiedPlan()
	if err != nil {
		return err
	}
	_, err = nativePeerFreshGuardCurrent(ctx, r.Project, r.Bridge)
	return err
}
