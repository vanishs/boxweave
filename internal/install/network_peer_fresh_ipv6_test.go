package install

import (
	"github.com/lxc/incus/v7/shared/api"
	"net/netip"
	"testing"
)

func TestPR178FreshIPv6RouteScope(t *testing.T) {
	const bridge = "incusbr-1001"
	gw := netip.MustParsePrefix("fd42:f960:51a3:edc1::1/64")
	nets := []api.Network{{Name: bridge, Type: "bridge", Managed: true, NetworkPut: api.NetworkPut{Config: api.ConfigMap{"ipv6.address": gw.String()}}}}
	rows := []nativePeerHostRoute{{Dst: gw.Masked().String(), Dev: bridge}, {Dst: gw.Addr().String(), Dev: bridge, Type: "local"}, {Dst: gw.Masked().Addr().String(), Dev: bridge, Type: "anycast"}}
	if _, e := nativePeerFreshIPv6Scope(bridge, gw, nets, rows); e != nil {
		t.Fatal(e)
	}
	for _, bad := range []nativePeerHostRoute{
		{Dst: gw.Masked().Addr().String(), Dev: bridge, Type: "anycast"},
		{Dst: gw.Masked().Addr().String(), Dev: "eth0", Type: "anycast"},
		{Dst: gw.Addr().String(), Dev: bridge, Type: "anycast"},
		{Dst: "fd42:f960:51a3:edc1::2", Dev: bridge, Type: "local"},
		{Dst: "fd42:f960:51a3::/48", Dev: "eth0"},
		{Dst: gw.Masked().String(), Dev: bridge},
	} {
		if _, e := nativePeerFreshIPv6Scope(bridge, gw, nets, append(append([]nativePeerHostRoute{}, rows...), bad)); e == nil {
			t.Fatal("accepted", bad)
		}
	}
	if _, e := nativePeerFreshIPv6Scope(bridge, gw, nets, rows[:1]); e == nil {
		t.Fatal("missing gateway")
	}
	foreign := api.Network{Name: "other", Type: "bridge", Managed: true, NetworkPut: api.NetworkPut{Config: api.ConfigMap{"ipv6.address": "fd42:f960:51a3:edc1::2/64"}}}
	if _, e := nativePeerFreshIPv6Scope(bridge, gw, append(nets, foreign), rows); e == nil {
		t.Fatal("foreign overlapping prefix")
	}
}
