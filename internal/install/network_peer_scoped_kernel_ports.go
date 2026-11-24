package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/networklimits"
)

var nativePeerGuardHostVethName = regexp.MustCompile(`^veth[a-zA-Z0-9_-]{1,11}$`)

// nativePeerGuardKernelPortsExactAt is a read-only kernel port fence for
// NONDISRUPTIVE per-Box ACL operations. It replaces neither the exact root
// Incus+SQLite ownership review nor the guard nft readback. Unlike the older
// all-STOP fence it permits the pinned C/D veth ports to stay UP, but refuses
// unexpected raw veths, stale native host names, and any missing port.
//
// Only the privileged root collector may supply observed instance readbacks;
// no MCP input may select a port or supply a trusted NIC state. A root writer
// must repeat the check immediately before/after every native effect and
// hold the original resource locks throughout.
func nativePeerGuardKernelPortsExactAt(ctx context.Context, sysClassNet, bridge string,
	claims []native.PeerACLBox, observed map[string]NativePeerGuardObservedOwner) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !safePrereqName.MatchString(bridge) ||
		bridge == "." || bridge == ".." || strings.ContainsAny(bridge, "/\\") ||
		len(claims) == 0 || len(claims) > networklimits.MaxProjectBoxes ||
		len(observed) != len(claims) {
		return errors.New("missing complete protected bridge port inventory")
	}
	expected := make(map[string]string, len(claims))
	seenNames := make(map[string]bool, len(claims))
	for _, b := range claims {
		if !safePrereqName.MatchString(b.Name) || b.UUID == "" || seenNames[b.Name] {
			return errors.New("untrusted duplicate native Box in kernel port audit")
		}
		seenNames[b.Name] = true
		o, ok := observed[b.Name]
		if !ok || o.Instance == nil || o.Instance.Name != b.Name ||
			o.Instance.Config["volatile.uuid"] != b.UUID {
			return fmt.Errorf("native instance identity missing for kernel port audit: %s", b.Name)
		}
		inst := o.Instance
		hostName := inst.Config["volatile.eth0.host_name"]
		switch inst.Status {
		case "Running", "Ready":
			if !nativePeerGuardHostVethName.MatchString(hostName) ||
				expected[hostName] != "" {
				return errors.New("untrusted or duplicate running native veth name")
			}
			expected[hostName] = b.Name
		case "Stopped":
			// A stopped Box must have no active NIC bridge port; its volatile
			// host_name can legitimately still record an old retired veth.
			if hostName != "" && !nativePeerGuardHostVethName.MatchString(hostName) {
				return errors.New("stopped native Box retains untrusted veth name")
			}
		default:
			return errors.New("native Box in uncertain lifecycle state")
		}
	}
	dir := filepath.Join(sysClassNet, bridge, "brif")
	first, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("kernel bridge port inventory unavailable: %w", err)
	}
	if len(first) != len(expected) {
		return fmt.Errorf("kernel bridge has %d live ports, expected %d exact owned NICs", len(first), len(expected))
	}
	names := make([]string, 0, len(first))
	for _, port := range first {
		name := port.Name()
		if expected[name] == "" || !nativePeerGuardHostVethName.MatchString(name) ||
			(port.Type()&os.ModeSymlink) == 0 {
			return fmt.Errorf("foreign or uncertain live kernel bridge port %q", name)
		}
		names = append(names, name)
	}
	slices.Sort(names)
	if err := ctx.Err(); err != nil {
		return err
	}
	// Detect an ordinary concurrent hotplug between inventory reads; a
	// privileged out-of-band host actor can still race afterward.
	second, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("kernel port inventory changed during review: %w", err)
	}
	names2 := make([]string, 0, len(second))
	for _, port := range second {
		if (port.Type() & os.ModeSymlink) == 0 {
			return errors.New("kernel bridge port is no longer a native veth symlink")
		}
		names2 = append(names2, port.Name())
	}
	slices.Sort(names2)
	if !slices.Equal(names, names2) {
		return errors.New("native bridge port inventory changed between kernel reads")
	}
	return nil
}

func nativePeerGuardKernelPortsExact(ctx context.Context, bridge string,
	claims []native.PeerACLBox, observed map[string]NativePeerGuardObservedOwner) error {
	return nativePeerGuardKernelPortsExactAt(ctx, "/sys/class/net", bridge, claims, observed)
}
