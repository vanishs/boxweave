package install

import (
	"errors"
	"github.com/vanishs/boxweave/internal/state"
	"net/netip"
)

// nativePeerBinding is a proposed exact-identity peer rule, NOT evidence that
// addresses belong to the named Boxes. Native UUID/NIC/MAC/lease exclusivity
// and all ACL consumers require independent checks before use.
type nativePeerBinding struct {
	SourceUUID, TargetUUID string
	SourceIPv4, TargetIPv4 netip.Addr
	SourceIPv6, TargetIPv6 netip.Addr
}

// NativePeerACLFromGrantSnapshot compiles the WHOLE verified directional
// SQLite allowlist rather than overwriting an ACL with a single new peer.
// This is pure compilation: it never performs SDK calls, unquiesces a NIC,
// or treats a SQLite row as evidence of host-native lease ownership.
func nativePeerBindingsFromSnapshot(rows []state.PeerGrantBinding) ([]nativePeerBinding, error) {
	if len(rows) > 32 {
		return nil, errors.New("too many durable peer grants or tombstones")
	}
	pairs := make([]nativePeerBinding, 0, len(rows))
	parse := func(raw string, v6 bool) (netip.Addr, error) {
		ip, err := netip.ParseAddr(raw)
		if err != nil || !ip.IsGlobalUnicast() || ip.Is4In6() || ip.Is6() != v6 || ip.String() != raw {
			return netip.Addr{}, errors.New("noncanonical or untrusted durable peer IP")
		}
		return ip, nil
	}
	for _, row := range rows {
		a4, e1 := parse(row.SourceIPv4, false)
		b4, e2 := parse(row.TargetIPv4, false)
		a6, e3 := parse(row.SourceIPv6, true)
		b6, e4 := parse(row.TargetIPv6, true)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil ||
			row.SourceUUID == "" || row.TargetUUID == "" || row.SourceUUID == row.TargetUUID ||
			row.SourceIncarnation == "" || row.TargetIncarnation == "" {
			return nil, errors.New("untrusted durable peer binding")
		}
		pairs = append(pairs, nativePeerBinding{
			SourceUUID: row.SourceUUID, TargetUUID: row.TargetUUID,
			SourceIPv4: a4, TargetIPv4: b4, SourceIPv6: a6, TargetIPv6: b6,
		})
	}
	return pairs, nil
}
