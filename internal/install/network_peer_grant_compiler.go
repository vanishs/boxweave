package install

import (
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

func NativePeerACLFromGrantSnapshot(t NativePrepareTarget, network *api.Network, rows []state.PeerGrantBinding) (api.NetworkACLsPost, error) {
	return NativePeerACLFromPolicySnapshot(t, network, rows, nil)
}

// Compile the complete durable policy: confirmed directions PLUS exact
// historical revoked DROP tombstones. Never remove tombstones merely because
// the next original concerns another pair.
func NativePeerACLFromPolicySnapshot(t NativePrepareTarget, network *api.Network,
	active, revoked []state.PeerGrantBinding) (api.NetworkACLsPost, error) {
	live, err := nativePeerBindingsFromSnapshot(active)
	if err != nil {
		return api.NetworkACLsPost{}, err
	}
	denied, err := nativePeerBindingsFromSnapshot(revoked)
	if err != nil {
		return api.NetworkACLsPost{}, err
	}
	return nativeACLWithDeniedPeers(t, network, live, denied)
}
