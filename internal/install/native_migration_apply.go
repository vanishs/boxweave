package install

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// NativeMigrationApply is ROOT-LOCAL ONLY and is always protected by the
// boxweave maintenance flock at the CLI boundary. It is not an MCP tool.
// Once APT_MAY_HAVE_RUN is durable, errors/crashes are UNKNOWN: NO AUTO RETRY.
func NativeMigrationApply(from, to, reviewedSHA, hostDigest string, approve, downtime bool) (NativeMigrationJournal, error) {
	var j NativeMigrationJournal
	if os.Geteuid() != 0 || !approve || !downtime {
		return j, errors.New("exact root local migration and native downtime approval required")
	}
	p, err := NativeMigrationReview(from, to)
	if err != nil {
		return j, err
	}
	if p.PlanSHA256 != reviewedSHA || p.HostDigest != hostDigest {
		return j, errors.New("native signed APT plan or exact host digest changed")
	}
	if err = os.Mkdir(nativeMigrationDir, 0700); err != nil {
		return j, err
	}
	j = NativeMigrationJournal{Schema: 1, Phase: "NEW", Plan: p, CreatedUnix: time.Now().Unix()}
	if err = setMigrationPhase(&j, "PREPARING_OLD_PACKAGE_BACKUP"); err != nil {
		return j, err
	}
	// Save exact signed OLD package artifacts before any daemon interruption.
	c := exec.Command("apt-get", "download", "incus="+from, "incus-client="+from)
	c.Dir = nativeMigrationDir
	if err = c.Run(); err != nil {
		return j, fmt.Errorf("old signed package download failed, no native stop: %w", err)
	}
	selectDeb := func(prefix string) (string, string, error) {
		matches, err := filepath.Glob(filepath.Join(nativeMigrationDir, prefix+"_*.deb"))
		if err != nil || len(matches) != 1 {
			return "", "", errors.New("exactly one old pinned deb required")
		}
		if err = os.Chmod(matches[0], 0600); err != nil {
			return "", "", err
		}
		h, err := shaNative(matches[0])
		return filepath.Base(matches[0]), h, err
	}
	j.OldServerDeb, j.ServerSHA, err = selectDeb("incus")
	if err != nil {
		return j, err
	}
	j.OldClientDeb, j.ClientSHA, err = selectDeb("incus-client")
	if err != nil {
		return j, err
	}
	if err = setMigrationPhase(&j, "OLD_PACKAGES_SAVED"); err != nil {
		return j, err
	}
	// Repeat inventory immediately before native stop. Never take over a guest
	// that appeared after the operator reviewed the signed plan.
	projs, err := migrationInventory()
	if err != nil {
		return j, err
	}
	if strings.Join(projs, ",") != strings.Join(p.Projects, ",") {
		return j, errors.New("native project inventory changed since approval")
	}
	if err = setMigrationPhase(&j, "DAEMON_STOP_REQUESTED"); err != nil {
		return j, err
	}
	if err = stopMigrationDaemon(); err != nil {
		_ = setMigrationPhase(&j, "STOP_UNCERTAIN")
		return j, fmt.Errorf("native stop uncertain, inspect before recovery: %w", err)
	}
	if err = setMigrationPhase(&j, "NATIVE_STOPPED"); err != nil {
		return j, err
	}
	backup := filepath.Join(nativeMigrationDir, "native-state.tar")
	if st, e := os.Lstat("/var/lib/incus"); e != nil || !st.IsDir() || st.Mode().IsRegular() {
		return j, errors.New("native state directory missing after daemon stop")
	}
	tar := exec.Command("tar", "--one-file-system",
		"--exclude=incus/unix.socket", "--exclude=incus/unix.socket.user",
		"--exclude=incus/guestapi/sock", "-C", "/var/lib", "-cf", backup, "incus")
	if err = tar.Run(); err != nil {
		_ = setMigrationPhase(&j, "SNAPSHOT_UNCERTAIN")
		return j, fmt.Errorf("offline native backup failed, inspect manually: %w", err)
	}
	if err = os.Chmod(backup, 0600); err != nil {
		return j, err
	}
	j.SnapshotSHA, err = shaNative(backup)
	if err != nil {
		return j, err
	}
	if err = assertNativeBackup(j); err != nil {
		return j, err
	}
	if err = setMigrationPhase(&j, "SNAPSHOT_READY"); err != nil {
		return j, err
	}
	// This is the irreversible uncertainty boundary: from here an APT exit,
	// loss of SSH or process kill never triggers automatic package re-install.
	if err = setMigrationPhase(&j, "APT_MAY_HAVE_RUN"); err != nil {
		return j, err
	}
	cmd := exec.Command("apt-get", "install", "-y", "--no-install-recommends",
		"incus="+to, "incus-client="+to)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	if err = cmd.Run(); err != nil {
		_ = setMigrationPhase(&j, "APT_OUTCOME_UNKNOWN")
		return j, fmt.Errorf("native APT upgrade outcome unknown: %w; no automatic retry", err)
	}
	// A privileged lab-only OS exit allows a real post-APT/pre-ACK crash test,
	// requiring an explicit root-owned sentinel inside the disposable guest.
	if os.Getenv("BW_S1_NATIVE_TEST_CRASH_AFTER_APT") == "1" {
		if marker, e := os.ReadFile("/etc/boxweave-s1-disposable-migration-fixture"); e == nil &&
			string(marker) == "EXPLICIT_DISPOSABLE_S1_LAB_ONLY\n" {
			os.Exit(79)
		}
	}
	if err = setMigrationPhase(&j, "APT_RETURNED_NEEDS_VERIFICATION"); err != nil {
		return j, err
	}
	if err = verifyMigratedNative(to); err != nil {
		_ = setMigrationPhase(&j, "POSTCHECK_UNKNOWN")
		return j, fmt.Errorf("new daemon/native postcheck uncertain: %w", err)
	}
	if err = setMigrationPhase(&j, "SUCCEEDED_SNAPSHOT_RETAINED"); err != nil {
		return j, err
	}
	return NativeMigrationStatus()
}

func stopMigrationDaemon() error {
	if err := runLocal("systemctl", "stop",
		"incus-user.socket", "incus-user.service", "incus.socket", "incus.service"); err != nil {
		return err
	}
	if err := exec.Command("systemctl", "is-active", "--quiet", "incus.service").Run(); err == nil {
		return errors.New("Incus daemon did not stop")
	}
	return nil
}

func verifyMigratedNative(version string) error {
	pkg, ok, err := installedIncusPackage()
	if err != nil || !ok || pkg != version {
		return errors.New("native daemon package postcondition mismatch")
	}
	client, err := installedClientVersion()
	if err != nil || client != version {
		return errors.New("native client package postcondition mismatch")
	}
	if err = runLocal("systemctl", "start", "incus.socket", "incus.service"); err != nil {
		return fmt.Errorf("cannot start changed native daemon: %w", err)
	}
	if err = exec.Command("systemctl", "is-active", "--quiet", "incus.service").Run(); err != nil {
		return errors.New("native daemon not running")
	}
	if _, err = os.Lstat("/var/lib/incus/unix.socket.user"); err != nil {
		return err
	}
	if _, err = migrationInventory(); err != nil {
		return fmt.Errorf("native isolated ownership lost: %w", err)
	}
	if out, err := exec.Command("runuser", "-u", "boxweave", "--",
		"incus", "profile", "show", "default").CombinedOutput(); err != nil ||
		!strings.Contains(string(out), "pool: default") || !strings.Contains(string(out), "path: /") {
		return errors.New("restricted native Project root profile not restored")
	}
	return nil
}
