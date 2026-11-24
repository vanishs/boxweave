package install

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/networklimits"
	"github.com/vanishs/boxweave/internal/state"
)

// NativePeerGuardMember is one independently pinned native UUID, MAC and
// dual-stack lease. Snapshot compilation NEVER establishes that pin: the root
// dispatcher must prove immutable Incus identity, UsedBy, bridge isolation,
// lease ownership and the original operation before applying any policy.
type NativePeerGuardMember struct {
	UUID, MAC, IPv4, IPv6 string
}

// NativePeerGuardPlan is a deterministic PURE nftables policy proposal.
// Install/Replace must NOT be executed from MCP, a guest, or by any caller
// that has not independently proven ownership and guarded boot/recovery.
// An nftables accept in one bridge chain does not bypass another chain's
// drop. The terminal drop is intentionally restricted to managed-to-managed
// traffic so unrelated host/forward traffic is unaffected.
type NativePeerGuardPlan struct {
	Table, Install, Replace, ExpectedKernelJSON, DigestSHA256 string
}

func NativePeerGuardFromSnapshot(t NativePrepareTarget, network *api.Network,
	members []NativePeerGuardMember, active, revoked []state.PeerGrantBinding) (NativePeerGuardPlan, error) {
	return nativePeerGuardCompile(t, network, members, active, revoked, false)
}

// Only fresh provisioning may propose the empty initial guard.
func nativePeerFreshInitialGuard(t NativePrepareTarget, network *api.Network) (NativePeerGuardPlan, error) {
	return nativePeerGuardCompile(t, network, nil, nil, nil, true)
}

func nativePeerGuardCompile(t NativePrepareTarget, network *api.Network,
	members []NativePeerGuardMember, active, revoked []state.PeerGrantBinding, fresh bool) (NativePeerGuardPlan, error) {
	var zero NativePeerGuardPlan
	if !safePrereqName.MatchString(t.Project) ||
		!safePrereqName.MatchString(t.Network) ||
		!safePrereqName.MatchString(t.ACL) ||
		!strings.HasPrefix(t.ACL, "bw-") ||
		network == nil || (network.Name != "" && network.Name != t.Network) ||
		(!fresh && len(members) == 0) || len(members) > networklimits.MaxProjectBoxes ||
		(fresh && (len(members) != 0 || len(active) != 0 || len(revoked) != 0)) {
		return zero, errors.New("untrusted scoped nft guard Project, bridge, ACL or complete inventory")
	}
	live, err := nativePeerBindingsFromSnapshot(active)
	if err != nil {
		return zero, err
	}
	denied, err := nativePeerBindingsFromSnapshot(revoked)
	if err != nil {
		return zero, err
	}
	if _, err := nativeScopedACLWithDeniedPeers(t, network, live, denied); err != nil {
		return zero, fmt.Errorf("unsafe full peer grant/tombstone snapshot: %w", err)
	}
	base4, err4 := netip.ParsePrefix(network.Config["ipv4.address"])
	base6, err6 := netip.ParsePrefix(network.Config["ipv6.address"])
	gateway4, g4err := nativeACLGateway(network.Config, "ipv4.address", false)
	gateway6, g6err := nativeACLGateway(network.Config, "ipv6.address", true)
	if err4 != nil || err6 != nil || g4err != nil || g6err != nil ||
		!base4.Addr().Is4() || !base6.Addr().Is6() ||
		gateway4 == (netip.Addr{}) || gateway6 == (netip.Addr{}) {
		return zero, errors.New("untrusted native guard bridge prefixes/gateways")
	}

	byUUID := make(map[string]NativePeerGuardMember, len(members))
	seen4, seen6, seenMAC := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, member := range members {
		ip4, e4 := netip.ParseAddr(member.IPv4)
		ip6, e6 := netip.ParseAddr(member.IPv6)
		mac, em := net.ParseMAC(member.MAC)
		if !scopedPeerUUID.MatchString(member.UUID) ||
			e4 != nil || e6 != nil || em != nil ||
			!ip4.Is4() || !ip6.Is6() || ip6.Is4In6() ||
			!ip4.IsGlobalUnicast() || !ip6.IsGlobalUnicast() ||
			member.IPv4 != ip4.String() || member.IPv6 != ip6.String() ||
			!base4.Contains(ip4) || !base6.Contains(ip6) ||
			ip4 == gateway4 || ip6 == gateway6 ||
			ip4 == base4.Masked().Addr() || ip6 == base6.Masked().Addr() ||
			len(mac) != 6 || mac[0]&1 != 0 ||
			member.MAC != strings.ToLower(mac.String()) ||
			byUUID[member.UUID].UUID != "" ||
			seen4[member.IPv4] || seen6[member.IPv6] || seenMAC[member.MAC] {
			return zero, errors.New("guard member duplicated, spoofed or outside exact bridge")
		}
		byUUID[member.UUID] = member
		seen4[member.IPv4], seen6[member.IPv6], seenMAC[member.MAC] = true, true, true
	}
	// Every durable live OR revoked binding must exist in the exact native
	// inventory; a missing formerly deleted Box cannot silently lose drops.
	for _, pair := range append(slices.Clone(active), revoked...) {
		src, srcOK := byUUID[pair.SourceUUID]
		dst, dstOK := byUUID[pair.TargetUUID]
		if !srcOK || !dstOK ||
			src.IPv4 != pair.SourceIPv4 || src.IPv6 != pair.SourceIPv6 ||
			dst.IPv4 != pair.TargetIPv4 || dst.IPv6 != pair.TargetIPv6 {
			return zero, errors.New("guard pair references missing or reincarnated native Box")
		}
	}

	// No untrusted field can become nft syntax: names are fixed-regexp
	// checked, IPs are canonical netip and MAC is a canonical 6-byte unicast.
	digest := sha256.Sum256([]byte(t.Project + "\x00" + t.Network))
	name := "bw_pg_" + hex.EncodeToString(digest[:8])
	prefix := fmt.Sprintf("meta ibrname \"%s\" ", t.Network)
	var rules []string
	pairs := slices.Clone(active)
	slices.SortFunc(pairs, func(a, b state.PeerGrantBinding) int {
		return strings.Compare(a.SourceUUID+"/"+a.TargetUUID, b.SourceUUID+"/"+b.TargetUUID)
	})
	for _, pair := range pairs {
		src, dst := byUUID[pair.SourceUUID], byUUID[pair.TargetUUID]
		// Both halves must prove the conntrack direction. If B->A is
		// independently granted, its original-direction rule must not
		// keep replies to a revoked A->B connection alive (or vice versa).
		// RELATED is not an authorization for a distinct data connection.
		for _, fam := range []struct{ proto, source, target string }{
			{"ip", src.IPv4, dst.IPv4}, {"ip6", src.IPv6, dst.IPv6},
		} {
			forward := fmt.Sprintf("%sether saddr %s ether daddr %s %s saddr %s %s daddr %s",
				prefix, src.MAC, dst.MAC, fam.proto, fam.source, fam.proto, fam.target)
			backward := fmt.Sprintf("%sether saddr %s ether daddr %s %s saddr %s %s daddr %s",
				prefix, dst.MAC, src.MAC, fam.proto, fam.target, fam.proto, fam.source)
			rules = append(rules, forward+" ct direction original ct state established,new accept", backward+" ct direction reply ct state established accept")
		}
	}
	// Restrict an entire exact Project bridge prefix rather than only the
	// current lease set: an unexpected source/destination address must not
	// bypass the guard during a failed native ACL fanout or lease reuse.
	// Routed gateway traffic uses INPUT/OUTPUT, not bridge FORWARD.
	// IPv6 link-local unicast can reach peer services outside the global
	// ULA prefix; permit only mandatory neighbor discovery there.
	rules = append(rules,
		fmt.Sprintf("%sip saddr %s ip daddr %s counter drop",
			prefix, base4.Masked(), base4.Masked()),
		// Neighbor discovery must not depend on connection tracking:
		// a solicited unicast NA can use a global source/destination.
		// Only ND with the mandatory on-link hop limit is exempt.
		fmt.Sprintf("%sip6 hoplimit 255 icmpv6 type { nd-neighbor-solicit, nd-neighbor-advert } accept", prefix),
		// Cover both ULA/link-local directions, including mixed pairs.
		fmt.Sprintf("%sip6 saddr { %s, fe80::/10 } ip6 daddr %s counter drop", prefix, base6.Masked(), base6.Masked()),
		fmt.Sprintf("%sip6 saddr { %s, fe80::/10 } ip6 daddr fe80::/10 counter drop", prefix, base6.Masked()),
	)
	var installed, replaced strings.Builder
	fmt.Fprintf(&installed, "table bridge %s {\n chain protect {\n  type filter hook forward priority -250; policy accept;\n", name)
	fmt.Fprintf(&replaced, "flush chain bridge %s protect\n", name)
	for _, rule := range rules {
		fmt.Fprintf(&installed, "  %s\n", rule)
		fmt.Fprintf(&replaced, "add rule bridge %s protect %s\n", name, rule)
	}
	installed.WriteString(" }\n}\n")
	expected, err := nativePeerGuardExpectedKernelJSON(t.Network, base4, base6, byUUID, active)
	if err != nil {
		return zero, err
	}
	plan := NativePeerGuardPlan{
		Table: name, Install: installed.String(), Replace: replaced.String(),
		ExpectedKernelJSON: expected,
	}
	plan.DigestSHA256 = nativePeerGuardDigest(plan)
	return plan, nil
}
