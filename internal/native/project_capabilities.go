package native

import (
	"errors"
	"fmt"
)

var (
	ErrNetworkAddressCapacityExhausted = fmt.Errorf("NETWORK_ADDRESS_CAPACITY_EXHAUSTED: no verified unreserved dual-stack address tuple remains: %w", ErrNativeUnsupported)
	ErrProjectProxyDenied              = fmt.Errorf("PROJECT_PROXY_DENIED: restricted.devices.proxy=allow requires host approval: %w", ErrNativeUnsupported)
	ErrProjectNestingDenied            = fmt.Errorf("PROJECT_NESTING_DENIED: restricted.containers.nesting=allow required: %w", ErrNativeUnsupported)
	ErrProjectInterceptionDenied       = fmt.Errorf("PROJECT_INTERCEPTION_DENIED: restricted.containers.interception=allow requires host approval: %w", ErrNativeUnsupported)
	ErrProjectLowlevelDenied           = fmt.Errorf("PROJECT_LOWLEVEL_DENIED: restricted.containers.lowlevel=allow requires separate host review: %w", ErrNativeUnsupported)
	ErrForwardPolicyDenied             = fmt.Errorf("FORWARD_EXTERNAL_POLICY_DENIED: effective host service policy does not permit this external bind: %w", ErrNativeUnsupported)
	ErrForwardAddressUnverified        = fmt.Errorf("FORWARD_HOST_ADDRESS_UNVERIFIED: address not uniquely verified on an allowed active host interface: %w", ErrNativeUnsupported)
	ErrIDMapReservationUnavailable     = fmt.Errorf("IDMAP_RESERVATION_UNAVAILABLE: persistent UID/GID reservation is not implemented: %w", ErrNativeUnsupported)
	ErrPrivilegedBoxDenied             = fmt.Errorf("PRIVILEGED_BOX_DENIED: privileged guests are not supported by this confined runtime: %w", ErrNativeUnsupported)
)

// CapabilityDenialClass maps only known pre-dispatch failures to durable diagnostics.
func CapabilityDenialClass(err error) string {
	for _, v := range []struct {
		err   error
		class string
	}{
		{ErrNetworkAddressCapacityExhausted, "NETWORK_ADDRESS_CAPACITY_EXHAUSTED"},
		{ErrProjectProxyDenied, "PROJECT_PROXY_DENIED"},
		{ErrProjectNestingDenied, "PROJECT_NESTING_DENIED"},
		{ErrProjectInterceptionDenied, "PROJECT_INTERCEPTION_DENIED"},
		{ErrProjectLowlevelDenied, "PROJECT_LOWLEVEL_DENIED"},
		{ErrForwardPolicyDenied, "FORWARD_EXTERNAL_POLICY_DENIED"},
		{ErrForwardAddressUnverified, "FORWARD_HOST_ADDRESS_UNVERIFIED"},
		{ErrIDMapReservationUnavailable, "IDMAP_RESERVATION_UNAVAILABLE"},
		{ErrPrivilegedBoxDenied, "PRIVILEGED_BOX_DENIED"},
	} {
		if errors.Is(err, v.err) {
			return v.class
		}
	}
	return ""
}

// ProjectPermissions reports only audited configuration, not proof of kernel
// capability, successful guest operation or packet reachability.
func ProjectPermissions(config map[string]string) map[string]string {
	out := map[string]string{}
	for _, key := range []string{
		"restricted.devices.proxy", "restricted.containers.nesting",
		"restricted.containers.interception", "restricted.containers.lowlevel",
		"restricted.containers.privilege", "restricted.devices.disk",
		"restricted.devices.gpu", "restricted.devices.nic",
	} {
		switch v := config[key]; v {
		case "":
			out[key] = "unset"
		case "allow", "block", "managed", "unprivileged", "isolated":
			out[key] = v
		default:
			out[key] = "other"
		}
	}
	return out
}

// ExternalForwardPolicyMode does not disclose individual interface addresses.
func (r Reader) ExternalForwardPolicyMode() string {
	rules, err := ParseExternalForwardAllow(r.ExternalForwardAllow)
	if err != nil {
		return "invalid"
	}
	if _, all := rules[ExternalForwardAll]; all {
		return "all"
	}
	if len(rules) == 0 {
		return "disabled"
	}
	return "exact"
}

func requireProjectProxy(config map[string]string) error {
	if config["restricted"] != "true" {
		return ErrNativeUnsupported
	}
	if config["restricted.devices.proxy"] != "allow" {
		return ErrProjectProxyDenied
	}
	return nil
}
