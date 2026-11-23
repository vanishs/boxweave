package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"slices"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

// This is an ADDITIONAL host-wide fail-closed fence, not proof that Linux
// conntrack has per-Project ownership. The original-direction IP-only -D
// removes *all* matching port tuples (proved on native Incus 7.6). Do not
// permit it while another managed network or foreign known NIC can overlap.
func nativePeerGlobalNetworkScopeSnapshot(ctx context.Context, root incus.InstanceServer,
	scope NativePeerBridgeAuditTarget) (string, error) {
	if root == nil {
		return "", errors.New("missing independently privileged native root")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	networks, err := root.GetNetworksAllProjects()
	if err != nil {
		return "", fmt.Errorf("all-Project native network inventory unavailable: %w", err)
	}
	instances, err := root.GetInstancesAllProjects(api.InstanceTypeAny)
	if err != nil {
		return "", fmt.Errorf("all-Project native NIC inventory unavailable: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return nativePeerGlobalNetworkScopeFromInventory(scope, networks, instances)
}

func nativePeerGlobalNetworkScopeFromInventory(scope NativePeerBridgeAuditTarget,
	networks []api.Network, instances []api.Instance) (string, error) {
	return nativeNetworkScopeFromInventory(scope, networks, instances, 2)
}
func nativeNetworkScopeFromInventory(scope NativePeerBridgeAuditTarget,
	networks []api.Network, instances []api.Instance, minBoxes int) (string, error) {
	if scope.Project == "" || scope.Bridge == "" || len(scope.Boxes) < minBoxes ||
		len(networks) == 0 || len(networks) > 512 || len(instances) > 4096 {
		return "", errors.New("incomplete or unbounded host-wide network inventory")
	}
	addresses := make([]netip.Addr, 0, len(scope.Boxes)*2)
	for _, b := range scope.Boxes {
		for _, raw := range []string{b.IPv4, b.IPv6} {
			a, err := netip.ParseAddr(raw)
			if err != nil || !a.IsGlobalUnicast() || a.Is4In6() {
				return "", errors.New("untrusted peer source/destination address")
			}
			addresses = append(addresses, a)
		}
	}
	// A global inventory must include exactly the root-owned bridge. An
	// absent bridge cannot prove that the list is complete.
	owners := 0
	ranges := []struct{ Project, Name, V4, V6 string }{}
	for _, nw := range networks {
		project := nw.Project
		if project == "" {
			project = "default"
		}
		if project == "default" && nw.Name == scope.Bridge && nw.Managed &&
			nw.Type == "bridge" {
			owners++
			// Reject owner addresses outside the reviewed managed bridge,
			// rather than trusting a claimed Box NIC with foreign routing.
			for _, family := range []struct {
				key string
				v6  bool
			}{{"ipv4.address", false}, {"ipv6.address", true}} {
				prefix, err := netip.ParsePrefix(nw.Config[family.key])
				if err != nil || prefix.Addr().Is4In6() || prefix.Addr().Is6() != family.v6 {
					return "", errors.New("root bridge address range is not a verified dual-stack prefix")
				}
				for _, addr := range addresses {
					if addr.Is6() == family.v6 && !prefix.Contains(addr) {
						return "", errors.New("peer source/target address lies outside root-owned bridge")
					}
				}
			}
			continue
		}
		if !nw.Managed {
			// Unmanaged links are not tracked as Incus network objects;
			// the separate physical/host routing admission remains required.
			continue
		}
		if nw.Name == "" || project == "" {
			return "", errors.New("ambiguous foreign managed network")
		}
		for _, raw := range []string{nw.Config["ipv4.address"], nw.Config["ipv6.address"]} {
			if raw == "" || raw == "none" {
				continue
			}
			prefix, err := netip.ParsePrefix(raw)
			if err != nil || prefix.Addr().Is4In6() {
				return "", errors.New("unparseable global native network address range")
			}
			for _, peerIP := range addresses {
				if prefix.Contains(peerIP) {
					return "", fmt.Errorf("foreign native network %q/%q overlaps protected peer address", project, nw.Name)
				}
			}
		}
		ranges = append(ranges, struct{ Project, Name, V4, V6 string }{
			project, nw.Name, nw.Config["ipv4.address"], nw.Config["ipv6.address"]})
	}
	if owners != 1 {
		return "", errors.New("missing or ambiguous globally owned root network")
	}
	type foreignNIC struct{ Project, Name, NIC, IPv4, IPv6 string }
	foreign := []foreignNIC{}
	for _, inst := range instances {
		project := inst.Project
		if project == "" {
			project = "default"
		}
		if project == scope.Project {
			continue // exact owner siblings are independently audited
		}
		if inst.Name == "" || project == "" {
			return "", errors.New("ambiguous foreign instance identity")
		}
		for name, dev := range inst.ExpandedDevices {
			if dev["type"] != "nic" {
				continue
			}
			for _, raw := range []string{dev["ipv4.address"], dev["ipv6.address"]} {
				if raw == "" || raw == "none" || raw == "auto" {
					continue
				}
				addr, err := netip.ParseAddr(raw)
				if err != nil || addr.Is4In6() {
					return "", errors.New("unparseable foreign NIC static address")
				}
				for _, peerIP := range addresses {
					if addr == peerIP {
						return "", fmt.Errorf("foreign Project %q NIC %q has identical protected peer IP", project, name)
					}
				}
			}
			foreign = append(foreign, foreignNIC{
				project, inst.Name, name, dev["ipv4.address"], dev["ipv6.address"]})
		}
	}
	slices.SortFunc(ranges, func(a, b struct{ Project, Name, V4, V6 string }) int {
		if a.Project != b.Project {
			if a.Project < b.Project {
				return -1
			}
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	slices.SortFunc(foreign, func(a, b foreignNIC) int {
		if a.Project != b.Project {
			if a.Project < b.Project {
				return -1
			}
			return 1
		}
		if a.Name != b.Name {
			if a.Name < b.Name {
				return -1
			}
			return 1
		}
		if a.NIC < b.NIC {
			return -1
		}
		if a.NIC > b.NIC {
			return 1
		}
		return 0
	})
	data, err := json.Marshal(struct {
		Project     string
		Bridge      string
		Ranges      any
		ForeignNICs any
	}{scope.Project, scope.Bridge, ranges, foreign})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append([]byte("boxweave/b2/native-global-ranges/v1\n"), data...))
	return hex.EncodeToString(sum[:]), nil
}
