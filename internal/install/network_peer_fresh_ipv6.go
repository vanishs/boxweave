package install

import (
	"context"
	"errors"
	"net/netip"
	"slices"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

// Prove exclusivity of the entire fresh IPv6 prefix, not just endpoints.
func nativePeerFreshIPv6Scope(bridge string, gateway netip.Prefix, networks []api.Network, routes []nativePeerHostRoute) (string, error) {
	if !gateway.IsValid() || !gateway.Addr().Is6() || gateway.Addr().Is4In6() || !gateway.Addr().IsPrivate() ||
		gateway.Bits() != 64 || gateway.Addr() == gateway.Masked().Addr() || len(networks) == 0 || len(networks) > 512 || len(routes) == 0 || len(routes) > 4096 {
		return "", errors.New("unverified fresh IPv6 scope")
	}
	var seen []string
	owners, connected, local, anycast := 0, 0, 0, 0
	for _, n := range networks {
		if !n.Managed {
			continue
		}
		project := n.Project
		if project == "" {
			project = "default"
		}
		raw := n.Config["ipv6.address"]
		if project == "default" && n.Name == bridge {
			if n.Type != "bridge" || raw != gateway.String() {
				return "", errors.New("fresh IPv6 owner bridge drift")
			}
			owners++
			continue
		}
		if raw == "" || raw == "none" {
			continue
		}
		p, e := netip.ParsePrefix(raw)
		if e != nil || !p.Addr().Is6() || p.Addr().Is4In6() || p.Overlaps(gateway) {
			return "", errors.New("foreign native IPv6 network overlap or uncertain prefix")
		}
		seen = append(seen, project+"/"+n.Name+"="+p.String())
	}
	for _, r := range routes {
		p, e := nativePeerHostRoutePrefix(r.Dst, true)
		if e != nil {
			return "", e
		}
		if p.Bits() == 0 || !p.Overlaps(gateway) {
			continue
		}
		if r.Dev != bridge || (r.PrefSrc != "" && r.PrefSrc != gateway.Addr().String()) {
			return "", errors.New("foreign host IPv6 route overlaps fresh bridge")
		}
		switch {
		case (r.Type == "" || r.Type == "unicast") && p.Masked() == gateway.Masked():
			connected++
		case r.Type == "local" && p.Bits() == 128 && p.Addr() == gateway.Addr():
			local++
		// Linux installs the subnet-router anycast address as a local
		// /128 at the exact network base, even before the first guest.
		case r.Type == "anycast" && p.Bits() == 128 && p.Addr() == gateway.Masked().Addr():
			anycast++
		default:
			return "", errors.New("unexpected overlapping IPv6 host route")
		}
	}
	if owners != 1 || connected != 1 || local != 1 || anycast > 1 {
		return "", errors.New("missing exact IPv6 bridge and gateway route proof")
	}
	slices.Sort(seen)
	return nativePrepareStateDigest("fresh-ipv6-v1", bridge, gateway.String(), seen), nil
}
func nativePeerFreshIPv6Snapshot(ctx context.Context, root incus.InstanceServer, bridge, raw string) (string, error) {
	gateway, err := netip.ParsePrefix(raw)
	if err != nil {
		return "", err
	}
	networks, err := root.GetNetworksAllProjects()
	if err != nil {
		return "", err
	}
	data, err := nativePeerHostIPOS(ctx, "-j", "-6", "route", "show", "table", "all")
	if err != nil {
		return "", err
	}
	routes, err := nativePeerHostRoutesParse(data)
	if err != nil {
		return "", err
	}
	return nativePeerFreshIPv6Scope(bridge, gateway, networks, routes)
}
