//go:build s1lab

package install

import (
	"os"
	"os/exec"
)

func nativeMigrationLabAptCommand(cmd *exec.Cmd) *exec.Cmd {
	if nativeMigrationLabMarkerOK() && os.Getenv("BW_S1_NATIVE_TEST_APT_FAIL_BEFORE_EFFECT") == "1" {
		return exec.Command("/bin/sh", "-c", "exit 42")
	}
	return cmd
}

func nativeMigrationLabMarkerOK() bool {
	marker, err := os.ReadFile("/etc/boxweave-s1-disposable-migration-fixture")
	return err == nil && string(marker) == "EXPLICIT_DISPOSABLE_S1_LAB_ONLY\n"
}

// NEVER use -tags=s1lab in a release artifact. Explicitly disposable
// integration fixture only; the marker and environment flag are both required.
func nativeMigrationLabCrashRequested() bool {
	if os.Getenv("BW_S1_NATIVE_TEST_CRASH_AFTER_APT") != "1" {
		return false
	}
	return nativeMigrationLabMarkerOK()
}
