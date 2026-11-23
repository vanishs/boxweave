package install

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

func pr178NativeACLPreimageFixture(t *testing.T) (
	NativePeerScopedRootTransition, NativePeerScopedRootEvidence) {
	t.Helper()
	proof, network, observed := pr178FourRootScopedFixture(t)
	plan, e := NativePeerScopedRootTransitionFromEvidence(proof, network, observed)
	if e != nil {
		t.Fatal(e)
	}
	for _, box := range proof.Inventory.Boxes {
		expected, ok := plan.ExpectedBeforeByUUID[box.UUID]
		if !ok {
			t.Fatal("missing exact native ACL preimage for sibling Box", box.Name)
		}
		owner := observed.Observed[box.Name]
		owner.ACL.NetworkACLPut = api.NetworkACLPut{
			Description: expected.Description,
			Config:      maps.Clone(expected.Config),
			Ingress:     slices.Clone(expected.Ingress),
			Egress:      slices.Clone(expected.Egress),
		}
		observed.Observed[box.Name] = owner
	}
	return plan, observed
}
func TestPR178NativeRootACLFullProjectReadbackExactBeforeImage(t *testing.T) {
	plan, observed := pr178NativeACLPreimageFixture(t)
	witness, e := NativePeerScopedVerifyObservedACLPreimage(plan, observed)
	if e != nil || witness.ACLCount != 4 || len(witness.SHA256) != 64 ||
		strings.Trim(witness.SHA256, "0123456789abcdef") != "" {
		t.Fatalf("all-Project read-only ACL preimage not proven %+v %v", witness, e)
	}
	// Rule lists are order-INDEPENDENT in real Incus 7.6. The digest must be
	// canonical under pure rule reordering but bind exact content and ETags.
	c := observed.Observed["pr178-c"]
	slices.Reverse(c.ACL.Ingress)
	slices.Reverse(c.ACL.Egress)
	observed.Observed["pr178-c"] = c
	again, e := NativePeerScopedVerifyObservedACLPreimage(plan, observed)
	if e != nil || !reflect.DeepEqual(witness, again) {
		t.Fatal("independent native rule list order changed root ACL witness digest", e)
	}
	// A new ETag is not itself a policy change, but MUST change the original
	// digest so it cannot be mistaken for a stale preimage at write time.
	b := observed.Observed["pr178-b"]
	b.ACLTag = "new-cas-etag"
	observed.Observed["pr178-b"] = b
	changed, e := NativePeerScopedVerifyObservedACLPreimage(plan, observed)
	if e != nil || changed.SHA256 == witness.SHA256 {
		t.Fatal("stale native ETag did not change the digest for a Root CAS", e)
	}
}
func TestPR178NativeRootACLPreimageRejectsForeignAcceptAndDrift(t *testing.T) {
	cases := []struct {
		name   string
		change func(NativePeerScopedRootTransition, *NativePeerScopedRootEvidence)
	}{
		{"added-C-ingress-accept", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-c"]
			x.ACL.Ingress = append(x.ACL.Ingress, api.NetworkACLRule{
				Action: "accept", Source: "0.0.0.0/0", State: "enabled",
			})
			o.Observed["pr178-c"] = x
		}},
		{"added-D-egress-accept", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-d"]
			x.ACL.Egress = append(x.ACL.Egress, api.NetworkACLRule{
				Action: "accept", Destination: "0.0.0.0/0", State: "enabled",
			})
			o.Observed["pr178-d"] = x
		}},
		{"changed-C-egress-action", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-c"]
			if x.ACL.Egress[0].Action == "accept" {
				x.ACL.Egress[0].Action = "reject"
			} else {
				x.ACL.Egress[0].Action = "accept"
			}
			o.Observed["pr178-c"] = x
		}},
		{"deleted-C-ingress-rule", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-c"]
			x.ACL.Ingress = nil
			o.Observed["pr178-c"] = x
		}},
		{"deleted-B-egress-rule", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-b"]
			x.ACL.Egress = x.ACL.Egress[:len(x.ACL.Egress)-1]
			o.Observed["pr178-b"] = x
		}},
		{"duplicated-exact-rule", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-c"]
			x.ACL.Egress = append(x.ACL.Egress, x.ACL.Egress[0])
			o.Observed["pr178-c"] = x
		}},
		{"foreign-ACL-config", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-a"]
			x.ACL.Config = api.ConfigMap{"user.foreign": "true"}
			o.Observed["pr178-a"] = x
		}},
		{"changed-description", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-b"]
			x.ACL.Description = "not the exact owned BoxWeave ACL"
			o.Observed["pr178-b"] = x
		}},
		{"wrong-ACL-name", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-b"]
			x.ACL.Name = "bw-unrelated"
			o.Observed["pr178-b"] = x
		}},
		{"missing-ETag", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-b"]
			x.ACLTag = ""
			o.Observed["pr178-b"] = x
		}},
		{"ACL-used-by-foreign", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-a"]
			x.ACL.UsedBy = []string{"/1.0/instances/foreign?project=other"}
			o.Observed["pr178-a"] = x
		}},
		{"wrong-ACL-project", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-b"]
			x.ACL.Project = "user-1001"
			o.Observed["pr178-b"] = x
		}},
		{"unexpected-boot-start", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-b"]
			x.Instance.Config["boot.autostart"] = "true"
			o.Observed["pr178-b"] = x
		}},
		{"missing-C-Box", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			delete(o.Observed, "pr178-c")
		}},
		{"missing-project-owner", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			o.AllProjectNames = o.AllProjectNames[:3]
		}},
		{"foreign-bridge-consumer", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			o.RootNetworkUsedBy = append(o.RootNetworkUsedBy, "/1.0/instances/other?project=other")
		}},
		{"duplicate-bridge-consumer", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			o.RootNetworkUsedBy = append(o.RootNetworkUsedBy, o.RootNetworkUsedBy[0])
		}},
		{"missing-original-C-ACL", func(p NativePeerScopedRootTransition, o *NativePeerScopedRootEvidence) {
			delete(p.ExpectedBeforeByUUID, pr178C)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, o := pr178NativeACLPreimageFixture(t)
			tc.change(p, &o)
			witness, e := NativePeerScopedVerifyObservedACLPreimage(p, o)
			if e == nil || witness.SHA256 != "" {
				t.Fatalf("forged native full-Project ACL preimage passed: %+v", witness)
			}
		})
	}
}
func TestPR178NativeRootACLRegrantRequiresExistingExactDrop(t *testing.T) {
	proof, network, observed := pr178FourRootScopedFixture(t)
	proof.Regrant = true
	ab := proof.After[1]
	proof.RevokedBefore = []state.PeerGrantBinding{ab}
	plan, e := NativePeerScopedRootTransitionFromEvidence(proof, network, observed)
	if e != nil {
		t.Fatal(e)
	}
	for _, b := range proof.Inventory.Boxes {
		x := observed.Observed[b.Name]
		expected := plan.ExpectedBeforeByUUID[b.UUID]
		x.ACL.NetworkACLPut = api.NetworkACLPut{
			Description: expected.Description,
			Config:      maps.Clone(expected.Config),
			Ingress:     slices.Clone(expected.Ingress),
			Egress:      slices.Clone(expected.Egress),
		}
		observed.Observed[b.Name] = x
	}
	if _, e = NativePeerScopedVerifyObservedACLPreimage(plan, observed); e != nil {
		t.Fatal("verified old A->B DROP was not accepted on regrant", e)
	}
	a := observed.Observed["pr178-a"]
	a.ACL.Ingress = nil
	observed.Observed["pr178-a"] = a
	if _, e = NativePeerScopedVerifyObservedACLPreimage(plan, observed); e == nil {
		t.Fatal("regrant silently erased native DROP preimage before authorization")
	}
}
