//go:build !s1lab

package install

import (
	"os"
	"testing"
)

func TestProductionNativeMigrationNeverAcceptsEnvironmentFaultInjection(t *testing.T) {
	t.Setenv("BW_S1_NATIVE_TEST_CRASH_AFTER_APT", "1")
	if nativeMigrationLabCrashRequested() {
		t.Fatal("untrusted environment can force production native migration to exit")
	}
	if os.Getenv("BW_S1_NATIVE_TEST_CRASH_AFTER_APT") != "1" {
		t.Fatal("test fixture did not set environment")
	}
}
