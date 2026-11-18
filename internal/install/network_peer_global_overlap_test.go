package install

import (
	"slices"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
)

func s4b2GlobalNetworkScopeFixture() (NativePeerBridgeAuditTarget, []api.Network, []api.Instance) {
	t := NativePeerBridgeAuditTarget{Project: "bwowned", Bridge: "bwbr0",
		Boxes: []native.PeerACLBox{
			{Name: "a", UUID: "a", IPv4: "10.248.83.11", IPv6: "fd83:248::11"},
			{Name: "b", UUID: "b", IPv4: "10.248.83.12", IPv6: "fd83:248::12"},
		}}
	networks := []api.Network{
		{Name: "bwbr0", Project: "default", Type: "bridge", Managed: true,
			NetworkPut: api.NetworkPut{Config: api.ConfigMap{"ipv4.address": "10.248.83.1/24", "ipv6.address": "fd83:248::1/64"}}},
		{Name: "other", Project: "foreign", Type: "bridge", Managed: true,
			NetworkPut: api.NetworkPut{Config: api.ConfigMap{"ipv4.address": "10.199.22.1/24", "ipv6.address": "fd99:22::1/64"}}},
	}
	foreign := []api.Instance{{Name: "f", Project: "foreign",
		ExpandedDevices: api.DevicesMap{"eth0": {"type": "nic", "network": "other",
			"ipv4.address": "10.199.22.22", "ipv6.address": "fd99:22::22"}}}}
	return t, networks, foreign
}

func TestS4B2GlobalRootNetworkIPOnlyConntrackScope(t *testing.T) {
	scope, networks, instances := s4b2GlobalNetworkScopeFixture()
	baseline, err := nativePeerGlobalNetworkScopeFromInventory(scope, networks, instances)
	if err != nil || len(baseline) != 64 {
		t.Fatalf("non-overlapping global root scope refused: %q %v", baseline, err)
	}
	reordered := slices.Clone(networks)
	reordered[0], reordered[1] = reordered[1], reordered[0]
	reorderedDigest, err := nativePeerGlobalNetworkScopeFromInventory(scope, reordered, instances)
	if err != nil || reorderedDigest != baseline {
		t.Fatalf("global scope digest depends on project inventory order: %s %v", reorderedDigest, err)
	}
	tests := []struct {
		name   string
		change func([]api.Network, []api.Instance) ([]api.Network, []api.Instance)
	}{
		{"owned-bridge-wrong-ipv4-prefix", func(n []api.Network, i []api.Instance) ([]api.Network, []api.Instance) {
			n[0].Config["ipv4.address"] = "10.55.3.1/24"
			return n, i
		}},
		{"owned-bridge-wrong-ipv6-prefix", func(n []api.Network, i []api.Instance) ([]api.Network, []api.Instance) {
			n[0].Config["ipv6.address"] = "fd31:65::1/64"
			return n, i
		}},
		{"foreign-same-ipv4-bridge", func(n []api.Network, i []api.Instance) ([]api.Network, []api.Instance) {
			n[1].Config["ipv4.address"] = "10.248.83.1/24"
			return n, i
		}},
		{"foreign-same-ipv6-bridge", func(n []api.Network, i []api.Instance) ([]api.Network, []api.Instance) {
			n[1].Config["ipv6.address"] = "fd83:248::1/64"
			return n, i
		}},
		{"foreign-broad-route", func(n []api.Network, i []api.Instance) ([]api.Network, []api.Instance) {
			n[1].Config["ipv4.address"] = "0.0.0.0/0"
			return n, i
		}},
		{"foreign-static-nic-reused-source-ip", func(n []api.Network, i []api.Instance) ([]api.Network, []api.Instance) {
			i[0].ExpandedDevices["eth0"]["ipv4.address"] = "10.248.83.11"
			return n, i
		}},
		{"foreign-static-nic-reused-target-ipv6", func(n []api.Network, i []api.Instance) ([]api.Network, []api.Instance) {
			i[0].ExpandedDevices["eth0"]["ipv6.address"] = "fd83:248::12"
			return n, i
		}},
		{"invalid-global-address", func(n []api.Network, i []api.Instance) ([]api.Network, []api.Instance) {
			n[1].Config["ipv4.address"] = "auto"
			return n, i
		}},
		{"owner-network-missing", func(n []api.Network, i []api.Instance) ([]api.Network, []api.Instance) {
			return n[1:], i
		}},
		{"owner-network-duplicated", func(n []api.Network, i []api.Instance) ([]api.Network, []api.Instance) {
			return append(n, n[0]), i
		}},
		{"empty-global-inventory", func(n []api.Network, i []api.Instance) ([]api.Network, []api.Instance) {
			return nil, i
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, nextN, nextI := s4b2GlobalNetworkScopeFixture()
			nn, ii := tc.change(nextN, nextI)
			if hash, err := nativePeerGlobalNetworkScopeFromInventory(scope, nn, ii); err == nil || hash != "" {
				t.Fatalf("unsafe host topology accepted: hash=%q error=%v", hash, err)
			}
		})
	}
	t.Log("S4_B2_GLOBAL_ROOT_NETWORK_OVERLAP_AND_FOREIGN_NIC_REFUSED=PASS")
}
