package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// After the full native stop barrier, the dedicated Incus bridge must have no
// host-kernel ports. A foreign raw NIC may not be represented by the controller
// Project's inventory. This is necessary, not a global host transaction.
func nativePeerBridgePortsEmptyAt(ctx context.Context, sysClassNet, bridge string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !safePrereqName.MatchString(bridge) || bridge == "." || bridge == ".." ||
		strings.ContainsAny(bridge, "/\\") {
		return errors.New("untrusted native bridge name for kernel port audit")
	}
	ports, err := os.ReadDir(filepath.Join(sysClassNet, bridge, "brif"))
	if err != nil {
		return fmt.Errorf("native bridge port inventory unavailable: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(ports) != 0 {
		return fmt.Errorf("native bridge has %d live ports despite complete stopped fence", len(ports))
	}
	return nil
}

func nativePeerBridgeKernelNoLivePorts(ctx context.Context, bridge string) error {
	return nativePeerBridgePortsEmptyAt(ctx, "/sys/class/net", bridge)
}
