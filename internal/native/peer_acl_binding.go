package native

import "github.com/lxc/incus/v7/shared/api"

// ValidatePeerACLBox is the same exact UUID/NIC/MAC/static dual-stack
// validation used by the native pre-PUT quiescence worker. A separate
// host-root read-only auditor can reuse it without relaxing the confined
// Reader or minting privileged credentials for the MCP service.
func ValidatePeerACLBox(inst *api.Instance, project, bridge, acl string, b PeerACLBox, stopped bool) error {
	return peerACLCheckBox(inst, project, bridge, acl, b, stopped)
}
