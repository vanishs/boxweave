package install

import (
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func TestNativePeerACLWholePolicyComparison(t *testing.T) {
	before := api.NetworkACLsPost{NetworkACLPut: api.NetworkACLPut{
		Ingress: []api.NetworkACLRule{{Action: "drop"}},
		Egress: []api.NetworkACLRule{{Action: "allow"}},
	}}
	same := api.NetworkACLsPost{NetworkACLPut: api.NetworkACLPut{
		Ingress: []api.NetworkACLRule{{Action: "drop"}},
		Egress: []api.NetworkACLRule{{Action: "allow"}},
	}}
	changed := api.NetworkACLsPost{NetworkACLPut: api.NetworkACLPut{
		Ingress: []api.NetworkACLRule{{Action: "allow"}},
		Egress: []api.NetworkACLRule{{Action: "allow"}},
	}}
	if !nativePeerACLMatches(&api.NetworkACL{NetworkACLPut: before.NetworkACLPut}, before) ||
		nativePeerACLMatches(nil, before) ||
		nativePeerACLMatches(&api.NetworkACL{NetworkACLPut: changed.NetworkACLPut}, before) {
		t.Fatal("native ordered full-ACL comparison failed")
	}
	if nativePeerACLPolicyChanges(before, same) || !nativePeerACLPolicyChanges(before, changed) {
		t.Fatal("pure grant/revoke no-op/refusal detection failed")
	}
	// Do not collapse nil and empty rule slices: the exact SDK native
	// observed policy must match the reviewed snapshot, not a loose subset.
	empty := api.NetworkACLsPost{NetworkACLPut: api.NetworkACLPut{Ingress: []api.NetworkACLRule{}}}
	if nativePeerACLMatches(&api.NetworkACL{}, empty) {
		t.Fatal("native nil versus empty rule set was silently normalized")
	}
}
