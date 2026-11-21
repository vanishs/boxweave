package native

import (
	"github.com/lxc/incus/v7/shared/api"
	"net/netip"
	"testing"
)

func TestStaticPeerBindingSnapshot(t *testing.T) {
	const name = "box-a"
	const uuid = "uuid-a"
	const original = "original"
	nic := map[string]string{"type": "nic", "name": "eth0", "network": "bwbr0", "security.acls": "bw-peer", "security.acls.default.ingress.action": "reject", "security.ipv4_filtering": "true", "security.ipv6_filtering": "true", "security.mac_filtering": "true", "ipv4.address": "10.1.2.15", "ipv6.address": "fd01:2::15", "hwaddr": "02:16:3e:11:22:33"}
	inst := &api.Instance{InstancePut: api.InstancePut{Config: api.ConfigMap{"volatile.uuid": uuid, createOperationKey: original, "volatile.eth0.hwaddr": nic["hwaddr"]}, Profiles: []string{"default"}, Devices: api.DevicesMap{"eth0": nic}}, Name: name, Project: "user-1001", Status: "Stopped", ExpandedDevices: api.DevicesMap{"eth0": nic}}
	nw := &api.Network{NetworkPut: api.NetworkPut{Config: api.ConfigMap{"ipv4.address": "10.1.2.1/24", "ipv6.address": "fd01:2::1/64"}}, Name: "bwbr0", Type: "bridge", Managed: true, Status: "Created"}
	v4 := api.NetworkLease{Hostname: name, Hwaddr: nic["hwaddr"], Address: nic["ipv4.address"], Type: "static"}
	v6 := api.NetworkLease{Hostname: name, Hwaddr: nic["hwaddr"], Address: nic["ipv6.address"], Type: "static"}
	dynamic := v6
	dynamic.Type = "dynamic"
	if got, err := staticPeerBindingSnapshot(inst, nw, netip.MustParsePrefix("10.1.2.1/24"), netip.MustParsePrefix("fd01:2::1/64"), []api.NetworkLease{v4, v6, dynamic}, "user-1001", name, uuid, original); err != nil || got.IPv6 != v6.Address {
		t.Fatal(got, err)
	}
	wrong := v6
	wrong.Hostname = "foreign"
	otherIP := dynamic
	otherIP.Address = "fd01:2::99"
	for _, leases := range [][]api.NetworkLease{{v4, dynamic}, {v4, v6, v6}, {v4, v6, wrong}, {v4, v6, otherIP}} {
		if got, err := staticPeerBindingSnapshot(inst, nw, netip.MustParsePrefix("10.1.2.1/24"), netip.MustParsePrefix("fd01:2::1/64"), leases, "user-1001", name, uuid, original); err == nil {
			t.Fatal("unsafe lease accepted", got)
		}
	}
	if got, err := staticPeerBindingSnapshot(inst, nw, netip.MustParsePrefix("10.1.2.1/24"), netip.MustParsePrefix("fd01:2::1/64"), []api.NetworkLease{v4, v6}, "user-1001", name, "replacement-uuid", original); err == nil {
		t.Fatal("wrong uuid", got)
	}
	if got, err := staticPeerBindingSnapshot(inst, nw, netip.MustParsePrefix("10.1.2.1/24"), netip.MustParsePrefix("fd01:2::1/64"), []api.NetworkLease{v4, v6}, "user-1001", name, uuid, "wrong-original"); err == nil {
		t.Fatal("wrong original", got)
	}
	inst.Devices["eth0"]["ipv6.address"] = ""
	if got, err := staticPeerBindingSnapshot(inst, nw, netip.MustParsePrefix("10.1.2.1/24"), netip.MustParsePrefix("fd01:2::1/64"), []api.NetworkLease{v4, v6}, "user-1001", name, uuid, original); err == nil {
		t.Fatal("dynamic config accepted", got)
	}
}

func TestPeerStatePrefixesRedacted(t *testing.T) {
	nw := &api.Network{}
	st := &api.NetworkState{State: "up", Bridge: &api.NetworkStateBridge{}, Addresses: []api.NetworkStateAddress{{Family: "inet", Address: "10.1.2.1", Netmask: "24", Scope: "global"}, {Family: "inet6", Address: "fd01:2::1", Netmask: "64", Scope: "global"}, {Family: "inet6", Address: "fe80::1", Netmask: "64", Scope: "link"}}}
	a, b, e := peerStatePrefixes(nw, st)
	if e != nil || a.String() != "10.1.2.1/24" || b.String() != "fd01:2::1/64" {
		t.Fatal(a, b, e)
	}
	st.Addresses = append(st.Addresses, api.NetworkStateAddress{Family: "inet", Address: "10.1.3.1", Netmask: "24", Scope: "global"})
	if _, _, e = peerStatePrefixes(nw, st); e == nil {
		t.Fatal("ambiguous gateway accepted")
	}
	st.Addresses = st.Addresses[:3]
	nw.Config = api.ConfigMap{"ipv4.address": "10.1.3.1/24"}
	if _, _, e = peerStatePrefixes(nw, st); e == nil {
		t.Fatal("conflicting config accepted")
	}
	for _, m := range []string{"allow", "block", "unknown"} {
		if PeerManagedNICPolicy(m) {
			t.Fatal(m)
		}
	}
	if !PeerManagedNICPolicy("") || !PeerManagedNICPolicy("managed") {
		t.Fatal("default")
	}
}
