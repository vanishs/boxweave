package install

import (
	"reflect"

	"github.com/lxc/incus/v7/shared/api"
)

// nativePeerACLMatches compares the complete ordered ingress and egress rule
// sets against the independently compiled desired policy. Keep the Incus
// SDK ETag preconditions, protected Root checks, native readback and
// conntrack/UNKNOWN barriers at each caller; this is ONLY a pure comparison.
func nativePeerACLMatches(got *api.NetworkACL, want api.NetworkACLsPost) bool {
	return got != nil && reflect.DeepEqual(got.Ingress, want.Ingress) &&
		reflect.DeepEqual(got.Egress, want.Egress)
}

func nativePeerACLPolicyChanges(before, after api.NetworkACLsPost) bool {
	return !reflect.DeepEqual(before.Ingress, after.Ingress) ||
		!reflect.DeepEqual(before.Egress, after.Egress)
}
