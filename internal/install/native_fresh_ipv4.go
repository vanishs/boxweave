package install

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"slices"
	"strings"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

// This policy is part of the approved fresh bootstrap digest. Existing
// daemons and existing BoxWeave networks never enter the resizing path.
const freshIPv4Policy = "fresh daemon only: target /21 (2045 nominal IPv4 identities, not 2045 live Boxes); exact prefix and pool are unknown until native bootstrap and exclusive route/lease proof; retain the verified original pool if no nonoverlapping /21 exists; never resize an existing BoxWeave bridge"

type freshIPv4Scope struct {
	Before  api.NetworkPut
	ETag    string
	Prefix  netip.Prefix
	Blocked []netip.Prefix
}

func freshIPv4Usable(p netip.Prefix) int {
	if !p.IsValid() || !p.Addr().Is4() || p.Bits() < 16 || p.Bits() > 30 {
		return 0
	}
	return (1 << (32 - p.Bits())) - 3
}

func freshIPv4Disjoint(p netip.Prefix, blocked []netip.Prefix) bool {
	for _, b := range blocked {
		if p.Overlaps(b) {
			return false
		}
	}
	return true
}

// Preserve the old gateway when safe. Otherwise try deterministic private
// /21 candidates, avoiding every observed native network and host route.
// Never guess an available network from an absent/incomplete inventory.
func freshIPv4Select(old netip.Prefix, blocked []netip.Prefix) (netip.Prefix, error) {
	if freshIPv4Usable(old) == 0 || !old.Addr().IsPrivate() || old.Addr() == old.Masked().Addr() {
		return netip.Prefix{}, errors.New("invalid fresh IPv4 bridge gateway")
	}
	// Old bridge itself must not overlap another network or host route.
	if !freshIPv4Disjoint(old, blocked) {
		return netip.Prefix{}, errors.New("fresh bridge already overlaps foreign native network or host route")
	}
	if old.Bits() <= 21 {
		return old, nil
	}
	wider := netip.PrefixFrom(old.Addr(), 21)
	if freshIPv4Disjoint(wider, blocked) {
		return wider, nil
	}
	for second := 0; second < 256; second++ {
		for third := 0; third < 256; third += 8 {
			a := netip.AddrFrom4([4]byte{10, byte(second), byte(third), 1})
			p := netip.PrefixFrom(a, 21)
			if freshIPv4Disjoint(p, blocked) {
				return p, nil
			}
		}
	}
	// The smaller native pool is honest and still subject to exact allocator
	// collision checks before CreateInstance. No unsafe wider subnet.
	return old, nil
}

func freshIPv4RouteBlocks(bridge string, old netip.Prefix, rows []nativePeerHostRoute) ([]netip.Prefix, error) {
	var blocked []netip.Prefix
	connected, local := false, false
	for _, r := range rows {
		p, e := nativePeerHostRoutePrefix(r.Dst, false)
		if e != nil {
			return nil, e
		}
		if p.Bits() == 0 {
			continue
		} // ordinary default routing
		if r.Dev == bridge && old.Masked().Contains(p.Masked().Addr()) && p.Bits() >= old.Bits() && (r.PrefSrc == "" || r.PrefSrc == old.Addr().String()) {
			switch r.Type {
			case "", "unicast":
				if p.Masked() == old.Masked() {
					connected = true
					continue
				}
			case "local":
				if p.Bits() == 32 && p.Addr() == old.Addr() {
					local = true
					continue
				}
			case "broadcast":
				// Only a /32 in the current owned subnet is an expected
				// kernel broadcast route, never an unrelated route.
				if p.Bits() == 32 {
					continue
				}
			}
		}
		blocked = append(blocked, p.Masked())
	}
	if !connected || !local {
		return nil, errors.New("fresh bridge gateway/kernel connected route not independently observed")
	}
	return blocked, nil
}

// Incus 7.6 reports a lease for each bridge gateway even with zero
// endpoints. Accept only those exact native observations, not guest leases.
func freshGatewayLeasesOnly(bridge string, v4, v6 netip.Addr, leases []api.NetworkLease) bool {
	if len(leases) != 2 {
		return false
	}
	seen := map[netip.Addr]bool{}
	for _, l := range leases {
		ip, e := netip.ParseAddr(l.Address)
		if e != nil || l.Hostname != bridge+".gw" || l.Type != "gateway" || l.Hwaddr != "" || l.Location != "" || (ip != v4 && ip != v6) || seen[ip] {
			return false
		}
		seen[ip] = true
	}
	return seen[v4] && seen[v6]
}

func inspectFreshIPv4(ctx context.Context, c incus.InstanceServer, project, bridge string) (freshIPv4Scope, error) {
	var zero freshIPv4Scope
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	projects, e := c.GetProjectNames()
	if e != nil || len(projects) != 2 || !slices.Contains(projects, "default") || !slices.Contains(projects, project) {
		return zero, errors.New("fresh native project inventory not exclusive")
	}
	instances, e := c.GetInstancesAllProjects(api.InstanceTypeAny)
	if e != nil || len(instances) != 0 {
		return zero, errors.New("fresh IPv4 provisioning refuses any existing native guest")
	}
	pr, _, e := c.GetProject(project)
	if e != nil || pr == nil || pr.Config["restricted"] != "true" || pr.Config["features.networks"] != "false" || pr.Config["restricted.networks.access"] != bridge {
		return zero, errors.New("fresh restricted project bridge ownership unverified")
	}
	pf, _, e := c.UseProject(project).GetProfile("default")
	if e != nil || pf == nil || len(pf.UsedBy) != 0 || pf.Devices["eth0"]["network"] != bridge || nativePrepareProfileEligible(pf.Devices, NativePrepareTarget{Network: bridge}) != nil {
		return zero, errors.New("fresh default profile not an unused native default")
	}
	nw, tag, e := c.GetNetwork(bridge)
	if e != nil || nw == nil || tag == "" || !nw.Managed || nw.Type != "bridge" || nw.Status != "Created" || (nw.Project != "" && nw.Project != "default") || nw.Description != "Network for user restricted project "+project || len(nw.Locations) > 1 || len(nw.UsedBy) != 1 || nw.UsedBy[0] != "/1.0/profiles/default?project="+project {
		return zero, errors.New("fresh native bridge not exclusive")
	}
	// Incus 7.6 default automatic bridge config. Refuse customized DHCP,
	// routes, external interfaces, ACLs or any other operator configuration.
	if len(nw.Config) != 4 || nw.Config["ipv4.nat"] != "true" || nw.Config["ipv6.nat"] != "true" {
		return zero, errors.New("fresh bridge configuration is not the untouched native default")
	}
	old, e := netip.ParsePrefix(nw.Config["ipv4.address"])
	if e != nil || freshIPv4Usable(old) == 0 {
		return zero, errors.New("fresh native IPv4 gateway invalid")
	}
	v6, e := netip.ParsePrefix(nw.Config["ipv6.address"])
	if e != nil || !v6.Addr().Is6() || v6.Addr().Is4In6() || v6.Bits() != 64 {
		return zero, errors.New("fresh native IPv6 gateway invalid")
	}
	leases, e := c.GetNetworkLeases(bridge)
	if e != nil || !freshGatewayLeasesOnly(bridge, old.Addr(), v6.Addr(), leases) {
		return zero, errors.New("fresh bridge lease inventory must contain exactly its own gateway identities")
	}
	forwards, e := c.GetNetworkForwardAddresses(bridge)
	if e != nil || len(forwards) != 0 {
		return zero, errors.New("fresh bridge network forward inventory not empty")
	}
	all, e := c.GetNetworksAllProjects()
	if e != nil || len(all) == 0 || len(all) > 512 {
		return zero, errors.New("global native network inventory unavailable")
	}
	var blocked []netip.Prefix
	owners := 0
	for _, n := range all {
		p := n.Project
		if p == "" {
			p = "default"
		}
		if n.Name == bridge && p == "default" && n.Managed {
			owners++
			if !reflect.DeepEqual(n.Writable(), nw.Writable()) {
				return zero, errors.New("fresh bridge inventory drift")
			}
			continue
		}
		if !n.Managed {
			continue
		}
		raw := n.Config["ipv4.address"]
		if raw == "" || raw == "none" {
			continue
		}
		v, e := netip.ParsePrefix(raw)
		if e != nil || !v.Addr().Is4() {
			return zero, errors.New("foreign native network has unverified IPv4 prefix")
		}
		blocked = append(blocked, v.Masked())
	}
	if owners != 1 {
		return zero, errors.New("global owned bridge inventory ambiguous")
	}
	raw, e := nativePeerHostIPOS(ctx, "-j", "-4", "route", "show", "table", "all")
	if e != nil {
		return zero, e
	}
	routes, e := nativePeerHostRoutesParse(raw)
	if e != nil {
		return zero, e
	}
	host, e := freshIPv4RouteBlocks(bridge, old, routes)
	if e != nil {
		return zero, e
	}
	blocked = append(blocked, host...)
	slices.SortFunc(blocked, func(a, b netip.Prefix) int { return strings.Compare(a.String(), b.String()) })
	blocked = slices.Compact(blocked)
	return freshIPv4Scope{Before: nw.Writable(), ETag: tag, Prefix: old, Blocked: blocked}, nil
}

// Only NativeDependencyBootstrap's exact fresh-daemon branch calls this.
// An uncertain update never retries. Independently observe the plan twice,
// perform one ETag-bound write on the new empty bridge, verify all poststate.
func provisionFreshIPv4() (string, error) {
	if exec.Command("systemctl", "is-active", "--quiet", "boxweave.service").Run() == nil {
		return "", errors.New("fresh IPv4 provisioning refuses active BoxWeave service")
	}
	// A re-created native daemon must not resize a previous controller's
	// network even when the service is currently inactive.
	for _, path := range []string{"/var/lib/boxweave/ledger.sqlite", "/etc/systemd/system/boxweave.service", nativePrepareJournalPath} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			return "", errors.New("fresh IPv4 provisioning refuses existing or unreadable BoxWeave state")
		}
	}
	u, e := user.Lookup("boxweave")
	if e != nil {
		return "", e
	}
	project, bridge := "user-"+u.Uid, "incusbr-"+u.Uid
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	c, e := prepareClient(ctx)
	if e != nil {
		return "", e
	}
	before, e := inspectFreshIPv4(ctx, c, project, bridge)
	if e != nil {
		return "", e
	}
	chosen, e := freshIPv4Select(before.Prefix, before.Blocked)
	if e != nil {
		return "", e
	}
	again, e := inspectFreshIPv4(ctx, c, project, bridge)
	if e != nil || !reflect.DeepEqual(before, again) {
		return "", errors.New("fresh IPv4 plan changed before effect")
	}
	if chosen == before.Prefix {
		return chosen.String(), nil
	}
	want := before.Before
	want.Config = api.ConfigMap{}
	for k, v := range before.Before.Config {
		want.Config[k] = v
	}
	want.Config["ipv4.address"] = chosen.String()
	if e = c.UpdateNetwork(bridge, want, before.ETag); e != nil {
		return "", fmt.Errorf("fresh IPv4 network update uncertain; inspect, never retry: %w", e)
	}
	after, e := inspectFreshIPv4(ctx, c, project, bridge)
	if e != nil || after.Prefix != chosen || !reflect.DeepEqual(after.Before, want) || !freshIPv4Disjoint(after.Prefix, after.Blocked) {
		return "", errors.New("fresh IPv4 update not independently verified; inspect before further effects")
	}
	return chosen.String(), nil
}
