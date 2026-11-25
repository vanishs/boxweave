package install

import (
	"context"
	"errors"
	"os"
)

const ScopedDrainFence = "/var/lib/boxweave-scoped-drain-v1.json"

// Drain differs from the final retirement fence: the exact scoped Root
// lifecycle/guard must remain available until the approved owned operations
// have settled. No new public Controller or other maintenance can start.
func CheckScopedDrainFence() error {
	if _, e := os.Lstat(ScopedDrainFence); !errors.Is(e, os.ErrNotExist) {
		return errors.New("SCOPED_DRAIN_IN_PROGRESS: preserve approved original and review scoped resume")
	}
	return nil
}
func CheckScopedDrainCommand(command string) error {
	switch command {
	case "doctor", "doctor-native-worker", "build-info", "install-status",
		"native-scoped-guard-boot", "native-scoped-lifecycle-serve",
		"native-scoped-retirement-plan", "native-scoped-retirement-resume-plan",
		"native-scoped-retirement-restore", "native-scoped-retirement-guard",
		"native-scoped-drain-step", "scoped-drain-worker":
		return nil
	default:
		return CheckScopedDrainFence()
	}
}

func ReadScopedDrainFence() ([]byte, error) {
	if os.Geteuid() != 0 {
		return nil, errors.New("root retirement receipt required")
	}
	b, _, e := diagnosticReadFile(ScopedDrainFence, 0, true, 1024*1024)
	return b, e
}

func CheckScopedDrainControllerInactive(ctx context.Context) error {
	return scopedRetirementInactive(ctx, "boxweave.service")
}
