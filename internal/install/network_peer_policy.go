package install

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/lxc/incus/v7/shared/api"
)

// Never carve a whole private bridge prefix to enable one directional peer.
// An exception is one exact address; a deleted/reused address is unsafe until
// dependent native rules are demonstrably revoked before lease release.
func nativeACLExceptAddresses(blocked string, exceptions []netip.Addr) (string, error) {
	var out []string
	for _, raw := range strings.Split(blocked, ",") {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(raw))
		if err != nil {
			return "", fmt.Errorf("invalid blocked prefix: %w", err)
		}
		parts := []netip.Prefix{prefix.Masked()}
		for _, addr := range exceptions {
			if !addr.IsValid() || addr.Is4In6() {
				return "", errors.New("invalid exact ACL exception address")
			}
			var next []netip.Prefix
			for _, p := range parts {
				next = append(next, nativeACLSubtractHost(p, addr)...)
			}
			parts = next
		}
		for _, p := range parts {
			out = append(out, p.String())
		}
	}
	return strings.Join(out, ","), nil
}

func nativeACLSubtractHost(prefix netip.Prefix, address netip.Addr) []netip.Prefix {
	prefix = prefix.Masked()
	if !prefix.Contains(address) {
		return []netip.Prefix{prefix}
	}
	var out []netip.Prefix
	for prefix.Bits() < address.BitLen() {
		bit := prefix.Bits()
		left := netip.PrefixFrom(prefix.Addr(), bit+1).Masked()
		nextAddr := prefix.Addr()
		if address.Is4() {
			raw := nextAddr.As4()
			raw[bit/8] ^= byte(1 << uint(7-bit%8))
			nextAddr = netip.AddrFrom4(raw)
		} else {
			raw := nextAddr.As16()
			raw[bit/8] ^= byte(1 << uint(7-bit%8))
			nextAddr = netip.AddrFrom16(raw)
		}
		right := netip.PrefixFrom(nextAddr, bit+1).Masked()
		if left.Contains(address) {
			out = append(out, right)
			prefix = left
		} else {
			out = append(out, left)
			prefix = right
		}
	}
	return out
}

// nativeACLForVerifiedPeers is a *compiler*, not a grant or an applied-state
// claim. The B2 controller does NOT enable peer writes: first prove native
// address pinning, cross-Project exclusivity and reload/UNKNOWN fencing.
// Explicit paired egress allows must also be tested for internet=false.
func nativeACLForVerifiedPeers(t NativePrepareTarget, network *api.Network, bindings []nativePeerBinding) (api.NetworkACLsPost, error) {
	base, err := nativePrepareACL(t, network)
	if err != nil || len(bindings) == 0 {
		return base, err
	}
	if len(bindings) > 32 {
		return base, errors.New("peer ACL limit exceeded")
	}
	v4, err := netip.ParsePrefix(network.Config["ipv4.address"])
	if err != nil {
		return base, errors.New("bridge IPv4 prefix unavailable")
	}
	v6, err := netip.ParsePrefix(network.Config["ipv6.address"])
	if err != nil {
		return base, errors.New("bridge IPv6 prefix unavailable")
	}
	gateway4, e4 := nativeACLGateway(network.Config, "ipv4.address", false)
	gateway6, e6 := nativeACLGateway(network.Config, "ipv6.address", true)
	if e4 != nil || e6 != nil {
		return base, errors.New("native bridge gateway identity unavailable")
	}
	v4, v6 = v4.Masked(), v6.Masked()
	sorted := slices.Clone(bindings)
	slices.SortFunc(sorted, func(a, b nativePeerBinding) int {
		return strings.Compare(a.SourceUUID+"/"+a.TargetUUID, b.SourceUUID+"/"+b.TargetUUID)
	})
	owners4, owners6 := map[netip.Addr]string{}, map[netip.Addr]string{}
	// A single native UUID must have one exact dual-stack pinned identity
	// across every directed grant. IP->UUID uniqueness alone is insufficient.
	uuid4, uuid6 := map[string]netip.Addr{}, map[string]netip.Addr{}
	seen := map[string]bool{}
	except4, except6 := []netip.Addr{v4.Addr()}, []netip.Addr{v6.Addr()}
	for _, p := range sorted {
		key := p.SourceUUID + "/" + p.TargetUUID
		if p.SourceUUID == "" || p.TargetUUID == "" || p.SourceUUID == p.TargetUUID || seen[key] {
			return base, errors.New("invalid or duplicate native peer pair")
		}
		seen[key] = true
		if !p.SourceIPv4.Is4() || !p.TargetIPv4.Is4() || !p.SourceIPv6.Is6() || !p.TargetIPv6.Is6() ||
			p.SourceIPv6.Is4In6() || p.TargetIPv6.Is4In6() ||
			p.SourceIPv4 == p.TargetIPv4 || p.SourceIPv6 == p.TargetIPv6 ||
			!v4.Contains(p.SourceIPv4) || !v4.Contains(p.TargetIPv4) ||
			!v6.Contains(p.SourceIPv6) || !v6.Contains(p.TargetIPv6) ||
			p.SourceIPv4 == v4.Addr() || p.TargetIPv4 == v4.Addr() ||
			p.SourceIPv6 == v6.Addr() || p.TargetIPv6 == v6.Addr() ||
			p.SourceIPv4 == gateway4 || p.TargetIPv4 == gateway4 ||
			p.SourceIPv6 == gateway6 || p.TargetIPv6 == gateway6 {
			return base, errors.New("peer IP is untrusted or outside selected bridge")
		}
		pairs := []struct {
			m  map[netip.Addr]string
			ip netip.Addr
			id string
		}{
			{owners4, p.SourceIPv4, p.SourceUUID}, {owners4, p.TargetIPv4, p.TargetUUID},
			{owners6, p.SourceIPv6, p.SourceUUID}, {owners6, p.TargetIPv6, p.TargetUUID},
		}
		for _, entry := range pairs {
			if old := entry.m[entry.ip]; old != "" && old != entry.id {
				return base, errors.New("address assigned to multiple native UUIDs")
			}
			entry.m[entry.ip] = entry.id
			uuidOwners := uuid4
			if entry.ip.Is6() {
				uuidOwners = uuid6
			}
			if previous, ok := uuidOwners[entry.id]; ok && previous != entry.ip {
				return base, errors.New("native UUID bound to more than one static address")
			}
			uuidOwners[entry.id] = entry.ip
		}
		// Native NDP response B -> A is addressed to A, not B.
		// The global reserved-destination reject takes precedence over
		// any later specific allow in Incus. Carve only the exact two
		// verified endpoint hosts, NEVER the entire bridge prefix.
		// The source/destination NIC rule still controls direction.
		except4 = append(except4, p.SourceIPv4, p.TargetIPv4)
		except6 = append(except6, p.SourceIPv6, p.TargetIPv6)
	}
	base.Egress[0].Destination, err = nativeACLExceptAddresses(nativeACLBlockedIPv4, except4)
	if err != nil {
		return base, err
	}
	base.Egress[1].Destination, err = nativeACLExceptAddresses(nativeACLBlockedIPv6, except6)
	if err != nil {
		return base, err
	}
	for _, p := range sorted {
		for _, f := range []struct {
			src, dst netip.Addr
			bits     int
		}{
			{p.SourceIPv4, p.TargetIPv4, 32}, {p.SourceIPv6, p.TargetIPv6, 128},
		} {
			rule := api.NetworkACLRule{
				Action: "allow", State: "enabled",
				Source:      fmt.Sprintf("%s/%d", f.src, f.bits),
				Destination: fmt.Sprintf("%s/%d", f.dst, f.bits),
				Description: "BoxWeave exact peer; native lease identity verified separately",
			}
			base.Ingress = append(base.Ingress, rule)
			base.Egress = append(base.Egress, rule)
		}
	}
	return base, nil
}

func exactPeerRuleKey(source, target netip.Addr, bits int) string {
	return fmt.Sprintf("%s/%d|%s/%d", source, bits, target, bits)
}
