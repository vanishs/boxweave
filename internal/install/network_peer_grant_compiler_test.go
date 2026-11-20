package install

import (
	"net/netip"
	"reflect"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func TestS4B3WholeDirectionalPeerACLCompiler(t *testing.T) {
	a := state.PeerGrantBinding{SourceUUID: "a", SourceIncarnation: "a", TargetUUID: "b", TargetIncarnation: "b",
		SourceIPv4: "10.55.8.11", TargetIPv4: "10.55.8.12", SourceIPv6: "fd55:8::11", TargetIPv6: "fd55:8::12"}
	b := state.PeerGrantBinding{SourceUUID: "b", SourceIncarnation: "b", TargetUUID: "c", TargetIncarnation: "c",
		SourceIPv4: "10.55.8.12", TargetIPv4: "10.55.8.13", SourceIPv6: "fd55:8::12", TargetIPv6: "fd55:8::13"}
	target := NativePrepareTarget{ACL: "bw-s4b2-unit"}
	nw := s4b2UnitBridge()
	acl, err := NativePeerACLFromGrantSnapshot(target, nw, []state.PeerGrantBinding{b, a})
	if err != nil || len(acl.Ingress) != 4 || len(acl.Egress) != 6 {
		t.Fatalf("multiple direction allowlist lost: %+v %v", acl, err)
	}
	swapped, err := NativePeerACLFromGrantSnapshot(target, nw, []state.PeerGrantBinding{a, b})
	if err != nil || !reflect.DeepEqual(acl, swapped) {
		t.Fatal("ACL generation not stable across SQLite iteration", err)
	}
	for _, c := range []struct{ src, dst string }{
		{"10.55.8.11/32", "10.55.8.12/32"},
		{"fd55:8::11/128", "fd55:8::12/128"},
		{"10.55.8.12/32", "10.55.8.13/32"},
		{"fd55:8::12/128", "fd55:8::13/128"},
	} {
		seen := false
		for _, rule := range acl.Ingress {
			if rule.Source == c.src && rule.Destination == c.dst && rule.Action == "allow" {
				seen = true
				break
			}
		}
		if !seen {
			t.Fatalf("missing exact directional rule %s -> %s", c.src, c.dst)
		}
	}
	if !s4b2CIDRMatches(acl.Egress[0].Destination, netip.MustParseAddr("10.55.8.99")) ||
		!s4b2CIDRMatches(acl.Egress[1].Destination, netip.MustParseAddr("fd55:8::99")) {
		t.Fatal("foreign private addresses unintentionally carved out of reject rules")
	}
	t.Log("S4_B3_WHOLE_ALLOWLIST_EXACT_DIRECTIONAL_IPV4_IPV6=PASS")
}

func TestS4B3ACLCompilerRejectsForgedOrNoncanonicalSnapshot(t *testing.T) {
	base := state.PeerGrantBinding{SourceUUID: "a", SourceIncarnation: "a", TargetUUID: "b", TargetIncarnation: "b",
		SourceIPv4: "10.55.8.11", TargetIPv4: "10.55.8.12", SourceIPv6: "fd55:8::11", TargetIPv6: "fd55:8::12"}
	cases := []struct {
		name   string
		mutate func(*state.PeerGrantBinding)
	}{
		{"missing-incarnation", func(p *state.PeerGrantBinding) { p.TargetIncarnation = "" }},
		{"foreign-private-ip", func(p *state.PeerGrantBinding) { p.TargetIPv4 = "192.168.50.10" }},
		{"ipv4-mapped", func(p *state.PeerGrantBinding) { p.SourceIPv6 = "::ffff:10.55.8.11" }},
		{"ip6-noncanonical", func(p *state.PeerGrantBinding) { p.SourceIPv6 = "fd55:0008::0011" }},
		{"invalid-v6", func(p *state.PeerGrantBinding) { p.TargetIPv6 = "not-ip" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := base
			tc.mutate(&item)
			if acl, err := NativePeerACLFromGrantSnapshot(NativePrepareTarget{ACL: "bw-s4b2-unit"}, s4b2UnitBridge(), []state.PeerGrantBinding{item}); err == nil {
				t.Fatalf("forged snapshot compiled %+v", acl)
			}
		})
	}
	t.Log("S4_B3_ACL_COMPILER_UNTRUSTED_ADDRESS_REFUSED=PASS")
}
