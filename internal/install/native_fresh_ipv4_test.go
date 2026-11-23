package install

import (
	"net/netip"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func TestFreshIPv4SelectVerifiedCapacity(t *testing.T) {
	old := netip.MustParsePrefix("10.46.78.1/24")
	// Keep the old IP, avoid neighboring native or host ranges.
	chosen, e := freshIPv4Select(old, []netip.Prefix{netip.MustParsePrefix("10.13.236.0/24"), netip.MustParsePrefix("10.249.1.0/24")})
	if e != nil || chosen.String() != "10.46.78.1/21" || freshIPv4Usable(chosen) != 2045 {
		t.Fatal(chosen, e)
	}
	blocked := []netip.Prefix{netip.MustParsePrefix("10.46.72.0/24"), netip.MustParsePrefix("10.0.0.0/21")}
	chosen, e = freshIPv4Select(old, blocked)
	if e != nil || chosen.String() != "10.0.8.1/21" || !freshIPv4Disjoint(chosen, blocked) {
		t.Fatal(chosen, e)
	}
	if _, e = freshIPv4Select(old, []netip.Prefix{netip.MustParsePrefix("10.46.78.21/32")}); e == nil {
		t.Fatal("old overlap accepted")
	}
	if _, e = freshIPv4Select(netip.MustParsePrefix("203.0.113.1/24"), nil); e == nil {
		t.Fatal("public bridge accepted")
	}
	old = netip.MustParsePrefix("192.168.20.1/24")
	chosen, e = freshIPv4Select(old, []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("192.168.21.0/24")})
	if e != nil || chosen != old || freshIPv4Usable(chosen) != 253 {
		t.Fatal(chosen, e)
	}
	large := netip.MustParsePrefix("10.67.0.1/16")
	if chosen, e = freshIPv4Select(large, nil); e != nil || chosen != large {
		t.Fatal(chosen, e)
	}
}

func TestFreshIPv4RouteInventoryProtectsForeignAndPolicyRoutes(t *testing.T) {
	old := netip.MustParsePrefix("10.46.78.1/24")
	rows := []nativePeerHostRoute{
		{Dst: "default", Dev: "eth0"},
		{Dst: "10.46.78.0/24", Dev: "incusbr-1001", PrefSrc: "10.46.78.1"},
		{Dst: "10.46.78.1", Dev: "incusbr-1001", Type: "local", PrefSrc: "10.46.78.1"},
		{Dst: "10.46.78.255", Dev: "incusbr-1001", Type: "broadcast", PrefSrc: "10.46.78.1"},
		{Dst: "10.46.76.0/24", Dev: "eth0"},
		{Dst: "10.249.1.0/24", Dev: "eth0"},
	}
	b, e := freshIPv4RouteBlocks("incusbr-1001", old, rows)
	if e != nil || len(b) != 2 {
		t.Fatal(b, e)
	}
	if p, e := freshIPv4Select(old, b); e != nil || p.Addr() == old.Addr() {
		t.Fatal("route overlap ignored", p, e)
	}
	if _, e := freshIPv4RouteBlocks("incusbr-1001", old, rows[:2]); e == nil {
		t.Fatal("missing local gateway proof")
	}
	for _, bad := range []nativePeerHostRoute{{Dst: "bad"}, {Dst: "fd00::/64"}, {Dst: "10.46.78.5", Type: "local", Dev: "eth0"}} {
		block, e := freshIPv4RouteBlocks("incusbr-1001", old, append(rows, bad))
		if e == nil {
			if _, e = freshIPv4Select(old, block); e == nil {
				t.Fatal("ambiguous or foreign route accepted", bad)
			}
		}
	}
}

func TestFreshGatewayLeaseInventory(t *testing.T) {
	v4 := netip.MustParseAddr("10.133.59.1")
	v6 := netip.MustParseAddr("fd42:3640:58e3:e13e::1")
	a := api.NetworkLease{Hostname: "incusbr-1001.gw", Address: v4.String(), Type: "gateway"}
	b := api.NetworkLease{Hostname: "incusbr-1001.gw", Address: v6.String(), Type: "gateway"}
	if !freshGatewayLeasesOnly("incusbr-1001", v4, v6, []api.NetworkLease{b, a}) {
		t.Fatal("exact gateway leases refused")
	}
	c := b
	c.Type = "static"
	d := b
	d.Address = "fd42:3640:58e3:e13e::2"
	z := b
	z.Hwaddr = "02:16:3e:11:22:33"
	for _, v := range [][]api.NetworkLease{nil, {a}, {a, a}, {a, c}, {a, d}, {a, z}, {a, b, c}} {
		if freshGatewayLeasesOnly("incusbr-1001", v4, v6, v) {
			t.Fatal("unsafe gateway inventory accepted", v)
		}
	}
}
