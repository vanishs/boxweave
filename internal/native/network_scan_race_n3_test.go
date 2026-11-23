package native

import (
	"context"
	"errors"
	"github.com/lxc/incus/v7/shared/api"
	"testing"
)

type n3ChangingNetworkSnapshot struct {
	mode                                                string
	networkReads, leaseReads, stateReads, instanceReads int
}

func (f *n3ChangingNetworkSnapshot) GetInstanceNames(api.InstanceType) ([]string, error) {
	return []string{"alpha"}, nil
}
func (f *n3ChangingNetworkSnapshot) GetInstances(api.InstanceType) ([]api.Instance, error) {
	f.instanceReads++
	nic := map[string]string{"type": "nic", "name": "eth0", "network": "bwbr0",
		"security.acls": "bw-peer", "security.acls.default.ingress.action": "reject",
		"security.ipv4_filtering": "true", "security.ipv6_filtering": "true", "security.mac_filtering": "true",
		"ipv4.address": "10.1.2.15", "ipv6.address": "fd01:2::15", "hwaddr": "02:16:3e:11:22:33"}
	cfg := api.ConfigMap{"volatile.uuid": "uuid-a", "volatile.eth0.hwaddr": "02:16:3e:11:22:33", createOperationKey: "original"}
	if f.mode == "instance-change" && f.instanceReads > 1 {
		cfg["user.boxweave.create_operation"] = "wrong"
	}
	inst := api.Instance{
		Name: "alpha", Project: "user-1001", Status: "Stopped",
		InstancePut:     api.InstancePut{Config: cfg, Profiles: []string{"default"}, Devices: api.DevicesMap{"eth0": nic}},
		ExpandedDevices: api.DevicesMap{"eth0": nic},
	}
	return []api.Instance{inst}, nil
}
func (f *n3ChangingNetworkSnapshot) GetNetwork(string) (*api.Network, string, error) {
	f.networkReads++
	nw := &api.Network{NetworkPut: api.NetworkPut{Config: api.ConfigMap{
		"ipv4.address": "10.1.2.1/24", "ipv6.address": "fd01:2::1/64"}},
		Name: "bwbr0", Type: "bridge", Managed: true, Status: "Created"}
	etag := "stable"
	if f.networkReads > 1 {
		if f.mode == "network-etag" {
			etag = "changed"
		}
		if f.mode == "network-config" {
			nw.Config["security.acls"] = "foreign"
		}
		if f.mode == "network-error" {
			return nil, "", errors.New("network unavailable")
		}
	}
	return nw, etag, nil
}
func (f *n3ChangingNetworkSnapshot) GetNetworkState(string) (*api.NetworkState, error) {
	f.stateReads++
	ipv4 := "10.1.2.1"
	if f.stateReads > 1 && f.mode == "gateway-change" {
		ipv4 = "10.1.2.99"
	}
	return &api.NetworkState{
		State: "up", Bridge: &api.NetworkStateBridge{},
		Addresses: []api.NetworkStateAddress{
			{Family: "inet", Address: ipv4, Netmask: "24", Scope: "global"},
			{Family: "inet6", Address: "fd01:2::1", Netmask: "64", Scope: "global"},
		}}, nil
}
func (f *n3ChangingNetworkSnapshot) GetNetworkLeases(string) ([]api.NetworkLease, error) {
	f.leaseReads++
	m := "02:16:3e:11:22:33"
	a := api.NetworkLease{Type: "static", Hostname: "alpha", Hwaddr: m, Address: "10.1.2.15"}
	b := api.NetworkLease{Type: "static", Hostname: "alpha", Hwaddr: m, Address: "fd01:2::15"}
	if f.leaseReads > 1 {
		switch f.mode {
		case "lease-removed":
			return []api.NetworkLease{a}, nil
		case "lease-duplicate":
			return []api.NetworkLease{a, b, b}, nil
		case "lease-owner-change":
			b.Hostname = "foreign"
			return []api.NetworkLease{a, b}, nil
		case "lease-error":
			return nil, errors.New("lease list read failed")
		}
	}
	if f.mode == "lease-reordered" && f.leaseReads > 1 {
		return []api.NetworkLease{b, a}, nil
	}
	return []api.NetworkLease{a, b}, nil
}

func TestN3NativeSnapshotFullNetworkLeaseRaceIsFailClosed(t *testing.T) {
	expected := []NetworkScanExpected{{
		Name: "alpha", UUID: "uuid-a", OriginalID: "original", Bridge: "bwbr0", ACL: "bw-peer",
		MAC: "02:16:3e:11:22:33", IPv4: "10.1.2.15", IPv6: "fd01:2::15", HasPin: true,
	}}
	for _, tc := range []struct {
		mode string
		pass bool
	}{
		{"stable", true}, {"lease-reordered", true},
		{"lease-removed", false}, {"lease-duplicate", false}, {"lease-owner-change", false},
		{"lease-error", false}, {"network-etag", false}, {"network-config", false},
		{"network-error", false}, {"gateway-change", false}, {"instance-change", false},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			f := &n3ChangingNetworkSnapshot{mode: tc.mode}
			got, e := scanNativeProject(context.Background(), f, "user-1001", expected)
			if tc.pass {
				if e != nil || len(got.Issues) != 0 || got.ObservationDigest == "" || f.leaseReads != 2 || f.networkReads != 2 || f.stateReads != 2 || f.instanceReads != 2 {
					t.Fatal("stable witnesses rejected", e, got, f)
				}
			} else if e == nil {
				t.Fatal("native changed between reads but scan claimed complete", got)
			}
		})
	}
}

func TestN3NativeCancelledOrExpiredInventoryNeverClears(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &n3ChangingNetworkSnapshot{}
	got, e := scanNativeProject(ctx, f, "user-1001", nil)
	if !errors.Is(e, context.Canceled) || got.ObservationDigest != "" || f.networkReads != 0 || f.leaseReads != 0 {
		t.Fatal("cancelled inventory reported healthy", got, e)
	}
}
