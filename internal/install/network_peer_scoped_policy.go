package install

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/netip"
	"regexp"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

// NativePeerACLSubject is an independently verified, exact leased Box identity.
// The compiler alone is not a native permission to edit or attach any ACL.
type NativePeerACLSubject struct {
	UUID string
	IPv4 string
	IPv6 string
}

var scopedPeerUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// NativePeerScopedACLName derives a fixed, immutable per-Box ACL name without
// embedding caller-controlled paths, Projects or untrusted native commands.
// Actual provisioning MUST refuse adoption/collision and independently check
// UsedBy, host leases and the exact instance UUID before attaching it.
func NativePeerScopedACLName(project, bridge, uuid string) (string, error) {
	if !safePrereqName.MatchString(project) || !safePrereqName.MatchString(bridge) ||
		!scopedPeerUUID.MatchString(uuid) {
		return "", errors.New("invalid scoped peer ACL identity")
	}
	sum := sha256.Sum256([]byte(project + "\x00" + bridge + "\x00" + uuid))
	return "bw-peer-" + hex.EncodeToString(sum[:24]), nil
}

// NativePeerScopedACLFromPolicySnapshot compiles only the directions involving
// one pinned owner. Unlike a shared ACL, unrelated C/D policies do not change
// when an A->B direction is granted/revoked. This is a PURE compiler, not an
// approved on-line migration or a live ACL update implementation.
func NativePeerScopedACLFromPolicySnapshot(
	target NativePrepareTarget, bridge *api.Network,
	active, revoked []state.PeerGrantBinding, subject NativePeerACLSubject,
) (api.NetworkACLsPost, error) {
	var zero api.NetworkACLsPost
	if !scopedPeerUUID.MatchString(subject.UUID) || target.ACL == "" || bridge == nil {
		return zero, errors.New("scoped peer subject or network missing")
	}
	v4, e4 := netip.ParseAddr(subject.IPv4)
	v6, e6 := netip.ParseAddr(subject.IPv6)
	p4, p4err := netip.ParsePrefix(bridge.Config["ipv4.address"])
	p6, p6err := netip.ParsePrefix(bridge.Config["ipv6.address"])
	gw4, gw4err := nativeACLGateway(bridge.Config, "ipv4.address", false)
	gw6, gw6err := nativeACLGateway(bridge.Config, "ipv6.address", true)
	if e4 != nil || e6 != nil || p4err != nil || p6err != nil ||
		gw4err != nil || gw6err != nil ||
		!v4.Is4() || !v6.Is6() || v6.Is4In6() ||
		!v4.IsGlobalUnicast() || !v6.IsGlobalUnicast() ||
		v4.String() != subject.IPv4 || v6.String() != subject.IPv6 ||
		!p4.Contains(v4) || !p6.Contains(v6) ||
		v4 == gw4 || v6 == gw6 || v4 == p4.Masked().Addr() ||
		v6 == p6.Masked().Addr() {
		return zero, errors.New("scoped peer owner has untrusted native address")
	}
	allActive, err := nativePeerBindingsFromSnapshot(active)
	if err != nil {
		return zero, err
	}
	allRevoked, err := nativePeerBindingsFromSnapshot(revoked)
	if err != nil {
		return zero, err
	}
	// Validate the ENTIRE state before selection. Invalid C/D snapshots must
	// not be silently ignored when compiling A's ACL.
	if _, err := nativeACLWithDeniedPeers(target, bridge, allActive, allRevoked); err != nil {
		return zero, err
	}
	var scopedActive, scopedRevoked []nativePeerBinding
	collect := func(input []nativePeerBinding, dest *[]nativePeerBinding) error {
		for _, pair := range input {
			if pair.SourceUUID != subject.UUID && pair.TargetUUID != subject.UUID {
				continue
			}
			if (pair.SourceUUID == subject.UUID &&
				(pair.SourceIPv4 != v4 || pair.SourceIPv6 != v6)) ||
				(pair.TargetUUID == subject.UUID &&
					(pair.TargetIPv4 != v4 || pair.TargetIPv6 != v6)) {
				return errors.New("scoped peer owner identity/address drift")
			}
			*dest = append(*dest, pair)
		}
		return nil
	}
	if err := collect(allActive, &scopedActive); err != nil {
		return zero, err
	}
	if err := collect(allRevoked, &scopedRevoked); err != nil {
		return zero, err
	}
	return nativeACLWithDeniedPeers(target, bridge, scopedActive, scopedRevoked)
}
