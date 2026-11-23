package install

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// This fixture models a ROOT READ-ONLY complete Incus/SQLite scope.
// It does NOT grant Root auth, create native ACLs or sign a durable
// original. The live v1w^~)vv2 ACL migration remains a distinct acceptance.
func pr178FourRootScopedFixture(t *testing.T) (state.ScopedPeerOriginalProof, *api.Network, NativePeerScopedRootEvidence) {
	t.Helper()
	claims, names, observed, usedBy := pr178ScopedInventoryFixture()
	for index, m := range pr178GuardMembers()[2:] {
		name := "pr178-c"
		if index == 1 {
			name = "pr178-d"
		}
		b := native.PeerACLBox{Name: name, UUID: m.UUID, IPv4: m.IPv4, IPv6: m.IPv6, MAC: m.MAC}
		claims = append(claims, b)
		names = append(names, name)
		link := "/1.0/instances/" + name + "?project=user-1001"
		usedBy = append(usedBy, link)
		acl, e := NativePeerScopedACLName("user-1001", "incusbr-1001", b.UUID)
		if e != nil {
			t.Fatal(e)
		}
		boot := "true"
		if index == 1 {
			boot = "false"
		}
		observed[name] = NativePeerGuardObservedOwner{
			Instance: &api.Instance{
				Name: name, Project: "user-1001", Status: "Running",
				InstancePut: api.InstancePut{
					Config: api.ConfigMap{"volatile.uuid": b.UUID,
						"volatile.eth0.hwaddr": b.MAC, "boot.autostart": boot},
					Profiles: []string{"default"},
					Devices: api.DevicesMap{"eth0": {"type": "nic",
						"ipv4.address": b.IPv4, "ipv6.address": b.IPv6}},
				},
				ExpandedDevices: api.DevicesMap{"eth0": {
					"type": "nic", "name": "eth0", "network": "incusbr-1001",
					"security.acls":                        acl,
					"security.acls.default.ingress.action": "reject",
					"security.acls.default.egress.action":  "reject",
					"security.ipv4_filtering":              "true",
					"security.ipv6_filtering":              "true",
					"security.mac_filtering":               "true",
					"ipv4.address":                         b.IPv4, "ipv6.address": b.IPv6,
				}},
			},
			ACL: &api.NetworkACL{
				NetworkACLPost: api.NetworkACLPost{Name: acl},
				Project:        "default", UsedBy: []string{link},
			},
			ACLTag: "etag-for-" + name,
		}
	}
	inventory := state.ScopedNetworkAuditLedger{
		Project: "user-1001", Bridge: "incusbr-1001",
	}
	for _, b := range claims {
		acl, e := NativePeerScopedACLName("user-1001", "incusbr-1001", b.UUID)
		if e != nil {
			t.Fatal(e)
		}
		inventory.Boxes = append(inventory.Boxes, state.ScopedNetworkAuditBinding{
			NetworkAuditBinding: state.NetworkAuditBinding{
				Name: b.Name, UUID: b.UUID, Incarnation: b.UUID,
				MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6,
			},
			ACL: acl,
		})
	}
	slices.SortFunc(inventory.Boxes, func(a, b state.ScopedNetworkAuditBinding) int {
		return strings.Compare(a.UUID, b.UUID)
	})
	ab := pr178Grant(pr178A, pr178B, "10.55.8.11", "10.55.8.12", "fd55:8::11", "fd55:8::12")
	cd := pr178Grant(pr178C, pr178D, "10.55.8.13", "10.55.8.14", "fd55:8::13", "fd55:8::14")
	proof := state.ScopedPeerOriginalProof{
		OriginalID: strings.Repeat("a", 32),
		Intent: state.ScopedPeerOriginalIntent{
			Schema: 2, Project: "user-1001", Bridge: "incusbr-1001",
			SourceUUID: pr178A, SourceIncarnation: pr178A,
			TargetUUID: pr178B, TargetIncarnation: pr178B,
			Allow: true,
		},
		Inventory: inventory,
		Before:    []state.PeerGrantBinding{cd},
		After:     []state.PeerGrantBinding{cd, ab},
	}
	evidence := NativePeerScopedRootEvidence{
		AllProjectNames: names, Observed: observed, RootNetworkUsedBy: usedBy,
	}
	return proof, s4b2UnitBridge(), evidence
}
func TestPR178NativeRootScopeCombinesV2SQLiteAndIncusFourOwners(t *testing.T) {
	proof, network, evidence := pr178FourRootScopedFixture(t)
	out, e := NativePeerScopedRootTransitionFromEvidence(proof, network, evidence)
	if e != nil {
		t.Fatal("full Root-scoped immutable evidence not compiled", e)
	}
	if out.OriginalID != proof.OriginalID ||
		out.SourceUUID != pr178A || out.TargetUUID != pr178B ||
		out.GuardBefore.DigestSHA256 == out.GuardAfter.DigestSHA256 ||
		out.GuardBefore.Table != out.GuardAfter.Table ||
		out.UnrelatedACLsVerifiedUnchanged != 2 ||
		reflect.DeepEqual(out.SourceACLBefore, out.SourceACLAfter) ||
		reflect.DeepEqual(out.TargetACLBefore, out.TargetACLAfter) {
		t.Fatal("native A/B and unrelated C/D compile delta invalid")
	}
	if out.OriginalBootAutostart[pr178A] != "true" ||
		out.OriginalBootAutostart[pr178B] != "false" ||
		out.OriginalBootAutostart[pr178C] != "true" ||
		out.OriginalBootAutostart[pr178D] != "false" {
		t.Fatal("root-only plan failed to retain explicit per-Box autostart")
	}
	if evidence.Observed["pr178-b"].Instance.Config["boot.autostart"] != "false" {
		t.Fatal("pure compiler modified a Box's explicit boot preference")
	}
	t.Log("PR178_SCOPED_V2_SQLITE_ROOT_NATIVE_C_D_UNCHANGED=PASS")
}
func TestPR178NativeRootScopeRevokeAndRegrantKeepExactTombstones(t *testing.T) {
	for _, kind := range []string{"revoke", "regrant"} {
		t.Run(kind, func(t *testing.T) {
			proof, network, evidence := pr178FourRootScopedFixture(t)
			ab := proof.After[1]
			if kind == "revoke" {
				proof.Intent.Allow = false
				proof.Before = proof.After
				proof.After = proof.Before[:1]
				proof.RevokedBefore = nil
				proof.RevokedAfter = []state.PeerGrantBinding{ab}
			} else {
				proof.Regrant = true
				proof.RevokedBefore = []state.PeerGrantBinding{ab}
			}
			out, e := NativePeerScopedRootTransitionFromEvidence(proof, network, evidence)
			if e != nil || out.UnrelatedACLsVerifiedUnchanged != 2 {
				t.Fatal("root read-only revoke/regrant proof failed", e)
			}
			if out.GuardBefore.DigestSHA256 == out.GuardAfter.DigestSHA256 {
				t.Fatal("native guard did not remove/restore exact A/B right")
			}
		})
	}
}
func TestPR178NativeRootScopedPlanFailsClosedOnForeignOrDrift(t *testing.T) {
	cases := []struct {
		name   string
		change func(*state.ScopedPeerOriginalProof, *api.Network, *NativePeerScopedRootEvidence)
	}{
		{"UNKNOWN-not-root-authorized", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			p.RecoveryOnly = true
		}},
		{"wrong-original-id", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			p.OriginalID = "foreign"
		}},
		{"wrong-v2-schema", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			p.Intent.Schema = 1
		}},
		{"wrong-bridge", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			p.Intent.Bridge = "other"
		}},
		{"wrong-pair", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			p.Intent.TargetUUID = pr178C
		}},
		{"wrong-direction", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			p.Intent.Allow = false
		}},
		{"unrelated-C-D-dropped", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			p.After = p.After[1:]
		}},
		{"all-Box-ACL-changed", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			p.After = append(p.After, pr178Grant(pr178B, pr178A, "10.55.8.12", "10.55.8.11", "fd55:8::12", "fd55:8::11"))
		}},
		{"spurious-DROP-removed", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			p.RevokedBefore = []state.PeerGrantBinding{p.After[1]}
		}},
		{"wrong-regrant-flag", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			p.Regrant = true
		}},
		{"bridge-foreign-network-name", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			n.Name = "other"
		}},
		{"missing-SDK-sibling", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			o.AllProjectNames = o.AllProjectNames[:3]
		}},
		{"foreign-UsedBy", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			o.RootNetworkUsedBy = append(o.RootNetworkUsedBy, "/1.0/instances/alien?project=other")
		}},
		{"wrong-ACL-Project", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			o.Observed["pr178-b"].ACL.Project = "user-1001"
		}},
		{"foreign-owner-UUID", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			o.Observed["pr178-b"].Instance.Config["volatile.uuid"] = pr178C
		}},
		{"stale-ACL-ETag", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			x := o.Observed["pr178-b"]
			x.ACLTag = ""
			o.Observed["pr178-b"] = x
		}},
		{"scoped-ACL-name-swapped", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			p.Inventory.Boxes[0].ACL = "bw-peer-forged"
		}},
		{"foreign-native-IPv4", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			o.Observed["pr178-c"].Instance.Devices["eth0"]["ipv4.address"] = "10.55.8.99"
		}},
		{"unsafe-autostart", func(p *state.ScopedPeerOriginalProof, n *api.Network, o *NativePeerScopedRootEvidence) {
			o.Observed["pr178-d"].Instance.Config["boot.autostart"] = "garbage"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, n, o := pr178FourRootScopedFixture(t)
			tc.change(&p, n, &o)
			got, e := NativePeerScopedRootTransitionFromEvidence(p, n, o)
			if e == nil || got.OriginalID != "" {
				t.Fatalf("unsafe Root v2 native plan compiled: %+v", got)
			}
		})
	}
}
