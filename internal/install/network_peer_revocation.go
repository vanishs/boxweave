package install

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/lxc/incus/v7/shared/api"
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
	tombstones := make(map[string]struct{}, len(denied)*2)
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
		}
	}
	return result, nil
}
func exactPeerRuleKey(source, target netip.Addr, bits int) string {
	return fmt.Sprintf("%s/%d|%s/%d", source, bits, target, bits)
}
