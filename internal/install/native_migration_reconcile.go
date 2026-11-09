package install

import (
	"errors"
	"os"
)

// Reconcile NEVER replays APT or restores a snapshot. It only independently
// re-reads the exact native state after a crash or failed final postcheck.
// Reviewed UNKNOWN stays UNKNOWN unless the requested native postcondition is
// actually proved. No public MCP operation can call this root-local function.
func NativeMigrationReconcile(hostDigest, expectedVersion string, approved bool) (NativeMigrationJournal, error) {
	j, err := readMigrationJournal()
	if err != nil {
		return j, err
	}
	if os.Geteuid() != 0 || !approved || hostDigest != j.Plan.HostDigest {
		return j, errors.New("explicit local host-bound recovery review required")
	}
	if err = assertNativeBackup(j); err != nil {
		return j, err
	}
	var settledPhase string
	switch j.Phase {
	case "APT_MAY_HAVE_RUN", "APT_OUTCOME_UNKNOWN", "APT_RETURNED_NEEDS_VERIFICATION", "POSTCHECK_UNKNOWN":
		if expectedVersion != j.Plan.NewPackage {
			return j, errors.New("new native package postcondition not asserted")
		}
		settledPhase = "SUCCEEDED_SNAPSHOT_RETAINED"
	case "ROLLBACK_UNCERTAIN":
		if expectedVersion != j.Plan.OldPackage {
			return j, errors.New("original native package postcondition not asserted")
		}
		settledPhase = "ROLLED_BACK_SNAPSHOT_RETAINED"
	default:
		return j, errors.New("not an unresolved native migration postcondition")
	}
	if err = verifyMigratedNative(expectedVersion); err != nil {
		return j, err
	}
	if err = setMigrationPhase(&j, settledPhase); err != nil {
		return j, err
	}
	return NativeMigrationStatus()
}
