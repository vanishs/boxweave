package install

import (
	"net/netip"
	"reflect"
	"strings"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

const (
	pr178A = "11111111-1111-4111-8111-111111111111"
	pr178B = "22222222-2222-4222-8222-222222222222"
	pr178C = "33333333-3333-4333-8333-333333333333"
	pr178D = "44444444-4444-4444-8444-444444444444"
)

func pr178Grant(src, dst, a4, b4, a6, b6 string) state.PeerGrantBinding {
	return state.PeerGrantBinding{SourceUUID: src, SourceIncarnation: src,
		TargetUUID: dst, TargetIncarnation: dst,
		SourceIPv4: a4, TargetIPv4: b4, SourceIPv6: a6, TargetIPv6: b6}
}
func pr178Subject(id, v4, v6 string) NativePeerACLSubject {
	return NativePeerACLSubject{UUID: id, IPv4: v4, IPv6: v6}
}
func pr178Compile(t *testing.T, subject NativePeerACLSubject,
	active, revoked []state.PeerGrantBinding) api.NetworkACLsPost {
	t.Helper()
	acl, err := NativePeerScopedACLFromPolicySnapshot(
		NativePrepareTarget{ACL: "bw-pr178-lab"}, s4b2UnitBridge(),
		active, revoked, subject)
	if err != nil {
		t.Fatal(err)
	}
	return acl
}

func TestPR178ScopedACLUnaffectedOwnerHasIdenticalPolicy(t *testing.T) {
	ab := pr178Grant(pr178A, pr178B, "10.55.8.11", "10.55.8.12", "fd55:8::11", "fd55:8::12")
	cd := pr178Grant(pr178C, pr178D, "10.55.8.13", "10.55.8.14", "fd55:8::13", "fd55:8::14")
	a := pr178Subject(pr178A, "10.55.8.11", "fd55:8::11")
	b := pr178Subject(pr178B, "10.55.8.12", "fd55:8::12")
	c := pr178Subject(pr178C, "10.55.8.13", "fd55:8::13")
	d := pr178Subject(pr178D, "10.55.8.14", "fd55:8::14")

	for _, unaffected := range []NativePeerACLSubject{c, d} {
		before := pr178Compile(t, unaffected, []state.PeerGrantBinding{cd}, nil)
		after := pr178Compile(t, unaffected, []state.PeerGrantBinding{ab, cd}, nil)
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("C/D policy changed from unrelated A->B grant: %+v", unaffected)
		}
		afterRevoke := pr178Compile(t, unaffected, []state.PeerGrantBinding{cd}, []state.PeerGrantBinding{ab})
		if !reflect.DeepEqual(before, afterRevoke) {
			t.Fatalf("C/D policy changed from unrelated A->B revoke: %+v", unaffected)
		}
		for _, reserved := range []netip.Addr{
			netip.MustParseAddr("10.55.8.11"),
			netip.MustParseAddr("10.55.8.12"),
			netip.MustParseAddr("fd55:8::11"),
			netip.MustParseAddr("fd55:8::12"),
		} {
			ix := 0
			if reserved.Is6() {
				ix = 1
			}
			if !s4b2CIDRMatches(before.Egress[ix].Destination, reserved) {
				t.Fatalf("C/D has unauthorized egress exception to A/B: %s", reserved)
			}
		}
	}
	for _, affected := range []NativePeerACLSubject{a, b} {
		before := pr178Compile(t, affected, nil, nil)
		after := pr178Compile(t, affected, []state.PeerGrantBinding{ab}, nil)
		if reflect.DeepEqual(before, after) || len(after.Ingress) != 2 || len(after.Egress) != 4 {
			t.Fatalf("A/B directed grant not reflected in scoped dual-stack ACL: %s", affected.UUID)
		}
		// The protected base reject still blocks unrelated same-bridge peers.
		for _, addr := range []netip.Addr{
			netip.MustParseAddr("10.55.8.13"),
			netip.MustParseAddr("fd55:8::13"),
		} {
			ix := 0
			if addr.Is6() {
				ix = 1
			}
			if !s4b2CIDRMatches(after.Egress[ix].Destination, addr) {
				t.Fatalf("A/B unexpectedly carved unauthorized address %s", addr)
			}
		}
		for _, rule := range after.Ingress {
			if rule.Action != "allow" || !strings.Contains(rule.Source, "/") ||
				!strings.Contains(rule.Destination, "/") {
				t.Fatal("scoped peer grant permits nonexact ingress", rule)
			}
		}
		tombstone := pr178Compile(t, affected, nil, []state.PeerGrantBinding{ab})
		if len(tombstone.Ingress) != 4 || len(tombstone.Egress) != 6 {
			t.Fatalf("scoped exact revoke lost IPv4/IPv6 and reverse drops: %+v", tombstone)
		}
		for _, rule := range tombstone.Ingress {
			if rule.Action != "drop" {
				t.Fatal("revoked direction had non-DROP ingress", rule)
			}
		}
	}
	t.Log("PR178_SCOPED_POLICY_ACD_CONTINUITY_COMPILER=PASS")
}

func TestPR178ScopedACLRejectsAllUntrustedSnapshotsAndOwnerDrift(t *testing.T) {
	ab := pr178Grant(pr178A, pr178B, "10.55.8.11", "10.55.8.12", "fd55:8::11", "fd55:8::12")
	cd := pr178Grant(pr178C, pr178D, "10.55.8.13", "10.55.8.14", "fd55:8::13", "fd55:8::14")
	owner := pr178Subject(pr178A, "10.55.8.11", "fd55:8::11")
	cases := []struct {
		name            string
		owner           NativePeerACLSubject
		active, revoked []state.PeerGrantBinding
	}{
		{"owner-forged-uuid", pr178Subject("untrusted", "10.55.8.11", "fd55:8::11"), []state.PeerGrantBinding{ab}, nil},
		{"owner-ipv4-drift", pr178Subject(pr178A, "10.55.8.99", "fd55:8::11"), []state.PeerGrantBinding{ab}, nil},
		{"owner-ipv6-drift", pr178Subject(pr178A, "10.55.8.11", "fd55:8::99"), []state.PeerGrantBinding{ab}, nil},
		{"owner-foreign-address", pr178Subject(pr178A, "192.168.1.4", "fd55:8::11"), nil, nil},
		{"owner-gateway", pr178Subject(pr178A, "10.55.8.1", "fd55:8::11"), nil, nil},
		{"owner-noncanonical", pr178Subject(pr178A, "10.55.8.11", "fd55:0008::0011"), nil, nil},
		{"global-duplicate-address", owner, []state.PeerGrantBinding{ab, pr178Grant(pr178C, pr178D, "10.55.8.12", "10.55.8.14", "fd55:8::13", "fd55:8::14")}, nil},
		{"global-invalid-foreign", owner, []state.PeerGrantBinding{ab, pr178Grant(pr178C, pr178D, "192.168.1.13", "10.55.8.14", "fd55:8::13", "fd55:8::14")}, nil},
		{"global-invalid-incarnation", owner, []state.PeerGrantBinding{ab, func() state.PeerGrantBinding { x := cd; x.TargetIncarnation = ""; return x }()}, nil},
		{"global-reverse-live-and-revoked", owner, []state.PeerGrantBinding{pr178Grant(pr178B, pr178A, "10.55.8.12", "10.55.8.11", "fd55:8::12", "fd55:8::11")}, []state.PeerGrantBinding{ab}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := NativePeerScopedACLFromPolicySnapshot(
				NativePrepareTarget{ACL: "bw-pr178-lab"}, s4b2UnitBridge(),
				tc.active, tc.revoked, tc.owner); err == nil {
				t.Fatalf("unsafe scoped input compiled %+v", got)
			}
		})
	}
}

func TestPR178ScopedACLNamePinnedToProjectBridgeAndNativeUUID(t *testing.T) {
	name, err := NativePeerScopedACLName("user-1001", "incusbr-1001", pr178A)
	if err != nil || !strings.HasPrefix(name, "bw-peer-") || len(name) > 63 {
		t.Fatalf("derived invalid bounded ACL name: %q %v", name, err)
	}
	again, err := NativePeerScopedACLName("user-1001", "incusbr-1001", pr178A)
	if err != nil || again != name {
		t.Fatal("scoped ACL name nondeterministic")
	}
	for _, tc := range []struct{ p, n, id string }{
		{"user-1002", "incusbr-1001", pr178A},
		{"user-1001", "incusbr-1002", pr178A},
		{"user-1001", "incusbr-1001", pr178B},
	} {
		other, e := NativePeerScopedACLName(tc.p, tc.n, tc.id)
		if e != nil || other == name {
			t.Fatal("distinct owner scope reused ACL name")
		}
	}
	for _, tc := range []struct{ p, n, id string }{
		{"../../root", "incusbr-1001", pr178A},
		{"user-1001", "../bridge", pr178A},
		{"user-1001", "incusbr-1001", "untrusted"},
	} {
		if _, e := NativePeerScopedACLName(tc.p, tc.n, tc.id); e == nil {
			t.Fatal("invalid scope accepted")
		}
	}
}
