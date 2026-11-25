package install

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

// nativeACLWithDeniedPeers compiles immutable exact directed DROP tombstones
// alongside independently verified live pairs. It does NOT perform native
// ACL writes or assert that any packet has been blocked. In particular,
// revocation must hold every affected address/Box claim until independent
// per-NIC kernel enforcement and flow tests have converged.
//
// Incus 7.6 orders ACL actions drop -> reject -> allow, independent of rule
// list order. The DROP entries are intentionally both ingress and egress
// because this ACL is attached directly to each owned instance NIC.
func nativeACLWithDeniedPeers(t NativePrepareTarget, network *api.Network, active, denied []nativePeerBinding) (api.NetworkACLsPost, error) {
	if len(active)+len(denied) > 32 {
		return api.NetworkACLsPost{}, errors.New("too many owned peer grants and revocation tombstones")
	}
	// Compiling all pairs first reuses the single strict native pair/UUID/
	// dual-stack/bridge-address validator, including across both sets. We
	// subsequently replace the disallowed entries; NONE of this intermediate
	// candidate is sent to Incus.
	all := make([]nativePeerBinding, 0, len(active)+len(denied))
	all = append(all, active...)
	all = append(all, denied...)
	result, err := nativeACLForVerifiedPeers(t, network, all)
	if err != nil {
		return result, err
	}
	if len(denied) == 0 {
		return result, nil
	}
	// A stateful reply to A->B traverses B->A. To guarantee that an
	// established session is revoked, tombstone both directions. If the
	// caller has an independent live B->A grant we cannot preserve it and
	// simultaneously guarantee revocation without per-flow primitives.
	// Fail closed rather than silently de-authorize a different grant.
	for _, block := range denied {
		for _, live := range active {
			if block.SourceUUID == live.TargetUUID && block.TargetUUID == live.SourceUUID {
				return api.NetworkACLsPost{}, errors.New("reverse peer grant must be revoked in same dependency set")
			}
		}
		for _, other := range denied {
			if block.SourceUUID == other.TargetUUID && block.TargetUUID == other.SourceUUID {
				return api.NetworkACLsPost{}, errors.New("duplicate symmetric revoked peer identity")
			}
		}
	}
	tombstones := make(map[string]struct{}, len(denied)*4)
	for _, binding := range denied {
		tombstones[exactPeerRuleKey(binding.SourceIPv4, binding.TargetIPv4, 32)] = struct{}{}
		tombstones[exactPeerRuleKey(binding.SourceIPv6, binding.TargetIPv6, 128)] = struct{}{}
	}
	removeGrant := func(rules []api.NetworkACLRule) []api.NetworkACLRule {
		kept := make([]api.NetworkACLRule, 0, len(rules))
		for _, rule := range rules {
			if rule.Action == "allow" {
				if _, denied := tombstones[rule.Source+"|"+rule.Destination]; denied {
					continue
				}
			}
			kept = append(kept, rule)
		}
		return kept
	}
	result.Ingress = removeGrant(result.Ingress)
	result.Egress = removeGrant(result.Egress)
	for _, binding := range denied {
		for _, flow := range []struct {
			source, dest netip.Addr
			bits         int
		}{
			{binding.SourceIPv4, binding.TargetIPv4, 32},
			{binding.SourceIPv6, binding.TargetIPv6, 128},
		} {
			block := api.NetworkACLRule{
				Action: "drop", State: "enabled",
				Source:      fmt.Sprintf("%s/%d", flow.source, flow.bits),
				Destination: fmt.Sprintf("%s/%d", flow.dest, flow.bits),
				Description: "BoxWeave exact revoked directed peer; hold original native lease and conntrack until independently safe",
			}
			result.Ingress = append(result.Ingress, block)
			result.Egress = append(result.Egress, block)
			// Conntrack reply traffic must be stopped as well. This is not a
			// new grant; the reverse-direction DROP remains exact-host only.
			reverse := block
			reverse.Source, reverse.Destination = block.Destination, block.Source
			result.Ingress = append(result.Ingress, reverse)
			result.Egress = append(result.Egress, reverse)
		}
	}
	return result, nil
}

// nativeACLRevokeOneFromSnapshot keeps every separately confirmed direction
// while replacing exactly the staged direction with host-exact DROP tombstones.
// Existing reverse live grants are deliberately refused: a stateful response
// cannot be simultaneously treated as both a revoked flow and a live new flow.
func nativeACLRevokeOneFromSnapshot(t NativePrepareTarget, network *api.Network,
	remaining []state.PeerGrantBinding, denied nativePeerBinding,
	historical ...state.PeerGrantBinding) (api.NetworkACLsPost, error) {
	live, err := nativePeerBindingsFromSnapshot(remaining)
	if err != nil {
		return api.NetworkACLsPost{}, err
	}
	old, err := nativePeerBindingsFromSnapshot(historical)
	if err != nil {
		return api.NetworkACLsPost{}, err
	}
	return nativeACLWithDeniedPeers(t, network, live, append(old, denied))
}
