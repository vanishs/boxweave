package install

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// NativeMigrationRollback is a separate, explicit, root-approved and
// host-bound recovery operation. No original APT task is ever automatically
// replayed. A rollback uncertainty stays locked for manual investigation.
func NativeMigrationRollback(hostDigest, oldVersion string, approved, downtime bool) (NativeMigrationJournal, error) {
	j, err := readMigrationJournal()
	if err != nil {
		return j, err
	}
	if os.Geteuid() != 0 || !approved || !downtime || hostDigest != j.Plan.HostDigest ||
		oldVersion != j.Plan.OldPackage {
		return j, errors.New("exact root local host/old-version rollback and downtime approval required")
	}
	if j.Phase == "ROLLED_BACK_SNAPSHOT_RETAINED" {
		return j, errors.New("old native state already rolled back; use explicit finalization")
	}
	switch j.Phase {
	case "APT_MAY_HAVE_RUN", "APT_OUTCOME_UNKNOWN", "APT_RETURNED_NEEDS_VERIFICATION",
		"POSTCHECK_UNKNOWN", "SUCCEEDED_SNAPSHOT_RETAINED":
	default:
		return j, errors.New("ambiguous migration snapshot/recovery state; refuse rollback without manual investigation")
	}
	if err = assertNativeBackup(j); err != nil {
		return j, err
	}
	// If the changed daemon is reachable, it must still have ZERO instances
	// and the original restricted Projects; foreign consumers block rollback.
	if exec.Command("systemctl", "is-active", "--quiet", "incus.service").Run() == nil {
		if _, err = migrationInventory(); err != nil {
			return j, fmt.Errorf("new native consumer blocks rollback: %w", err)
		}
	}
	if err = setMigrationPhase(&j, "ROLLBACK_MAY_HAVE_RUN"); err != nil {
		return j, err
	}
	if err = stopMigrationDaemon(); err != nil {
		_ = setMigrationPhase(&j, "ROLLBACK_UNCERTAIN")
		return j, err
	}
	// Restore exact previously cached signed Ubuntu packages, never a floating
	// repository version. DPKG may have changed schema; restore old native DB
	// from the offline snapshot AFTER package downgrade and before re-start.
	server := filepath.Join(nativeMigrationDir, j.OldServerDeb)
	client := filepath.Join(nativeMigrationDir, j.OldClientDeb)
	cmd := exec.Command("apt-get", "install", "-y", "--allow-downgrades",
		"--no-install-recommends", server, client)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	if err = cmd.Run(); err != nil {
		_ = setMigrationPhase(&j, "ROLLBACK_UNCERTAIN")
		return j, fmt.Errorf("rollback package change uncertain: %w", err)
	}
	if err = stopMigrationDaemon(); err != nil {
		_ = setMigrationPhase(&j, "ROLLBACK_UNCERTAIN")
		return j, err
	}
	pkg, ok, e := installedIncusPackage()
	if e != nil || !ok || pkg != oldVersion {
		_ = setMigrationPhase(&j, "ROLLBACK_UNCERTAIN")
		return j, errors.New("old native package not restored; keep snapshot for manual recovery")
	}
	quarantine := "/var/lib/incus-native-migration-quarantine-" + strconv.FormatInt(j.CreatedUnix, 10)
	if _, e = os.Lstat(quarantine); !errors.Is(e, os.ErrNotExist) {
		_ = setMigrationPhase(&j, "ROLLBACK_UNCERTAIN")
		return j, errors.New("existing native quarantine requires manual review")
	}
	if err = os.Rename("/var/lib/incus", quarantine); err != nil {
		_ = setMigrationPhase(&j, "ROLLBACK_UNCERTAIN")
		return j, err
	}
	// Offline, digest-verified, root-only original native state. Never
	// overwrite a live database by extracting in-place.
	if err = runLocal("tar", "--no-same-owner", "-C", "/var/lib", "-xf",
		filepath.Join(nativeMigrationDir, "native-state.tar")); err != nil {
		_ = setMigrationPhase(&j, "ROLLBACK_UNCERTAIN")
		return j, fmt.Errorf("offline native snapshot restore uncertain: %w", err)
	}
	if err = verifyMigratedNative(oldVersion); err != nil {
		_ = setMigrationPhase(&j, "ROLLBACK_UNCERTAIN")
		return j, fmt.Errorf("restored native daemon postcheck failed: %w", err)
	}
	if err = setMigrationPhase(&j, "ROLLED_BACK_SNAPSHOT_RETAINED"); err != nil {
		return j, err
	}
	return NativeMigrationStatus()
}

// Keep backups and audit available for postmortem even after completing the
// recovery window; move root-owned directory to immutable-named archive.
// A second migration cannot start while the active journal remains.
func NativeMigrationFinalize(hostDigest, expectedVersion string, approved bool) (string, error) {
	j, err := readMigrationJournal()
	if err != nil {
		return "", err
	}
	if !approved || os.Geteuid() != 0 || hostDigest != j.Plan.HostDigest {
		return "", errors.New("root host-specific native migration finalization approval required")
	}
	var want string
	switch j.Phase {
	case "SUCCEEDED_SNAPSHOT_RETAINED":
		want = j.Plan.NewPackage
	case "ROLLED_BACK_SNAPSHOT_RETAINED":
		want = j.Plan.OldPackage
	default:
		return "", errors.New("native migration not conclusively healthy; cannot finalize")
	}
	if want != expectedVersion {
		return "", errors.New("expected native package version must match observed accepted outcome")
	}
	if err = assertNativeBackup(j); err != nil {
		return "", err
	}
	if err = verifyMigratedNative(want); err != nil {
		return "", err
	}
	archive := nativeMigrationDir + "-completed-" + strconv.FormatInt(j.CreatedUnix, 10)
	if strings.ContainsAny(archive, "\n\r") {
		return "", errors.New("unsafe archive path")
	}
	if _, err = os.Lstat(archive); !errors.Is(err, os.ErrNotExist) {
		return "", errors.New("native audit archive path already exists")
	}
	if err = setMigrationPhase(&j, "FINALIZED_"+j.Phase); err != nil {
		return "", err
	}
	if err = os.Rename(nativeMigrationDir, archive); err != nil {
		return "", err
	}
	return archive, nil
}
