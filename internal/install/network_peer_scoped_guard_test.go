package install

import (
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func pr178GuardMembers() []NativePeerGuardMember {
	return []NativePeerGuardMember{
		{UUID: pr178A, MAC: "10:66:6a:e2:1e:fd", IPv4: "10.55.8.11", IPv6: "fd55:8::11"},
		{UUID: pr178B, MAC: "10:66:6a:84:b1:ee", IPv4: "10.55.8.12", IPv6: "fd55:8::12"},
		{UUID: pr178C, MAC: "10:66:6a:9a:19:4a", IPv4: "10.55.8.13", IPv6: "fd55:8::13"},
		{UUID: pr178D, MAC: "10:66:6a:4c:5e:0f", IPv4: "10.55.8.14", IPv6: "fd55:8::14"},
	}
}
func pr178GuardTarget() NativePrepareTarget {
	return NativePrepareTarget{Project: "user-1001", Network: "incusbr-1001", ACL: "bw-pr178-lab"}
}
func pr178Guard(t *testing.T, members []NativePeerGuardMember, grants, revoked []state.PeerGrantBinding) NativePeerGuardPlan {
	t.Helper()
	r, e := NativePeerGuardFromSnapshot(pr178GuardTarget(), s4b2UnitBridge(), members, grants, revoked)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestPR178GuardNonDisruptiveDirectedDualStackAtomicText(t *testing.T) {
	cd := pr178Grant(pr178C, pr178D, "10.55.8.13", "10.55.8.14", "fd55:8::13", "fd55:8::14")
	ab := pr178Grant(pr178A, pr178B, "10.55.8.11", "10.55.8.12", "fd55:8::11", "fd55:8::12")
	members := pr178GuardMembers()
	before := pr178Guard(t, members, []state.PeerGrantBinding{cd}, nil)
	after := pr178Guard(t, members, []state.PeerGrantBinding{ab, cd}, nil)
	reversed := pr178Guard(t, []NativePeerGuardMember{members[3], members[1], members[0], members[2]},
		[]state.PeerGrantBinding{cd, ab}, nil)
	if before.DigestSHA256 == after.DigestSHA256 || after != reversed {
		t.Fatal("guard does not vary by approved grant or is order-unstable")
	}
	if !strings.Contains(after.Install, "type filter hook forward priority -250; policy accept;") ||
		!strings.HasPrefix(after.Replace, "flush chain bridge "+after.Table+" protect\n") ||
		strings.Contains(after.Replace, "delete table") ||
		strings.Contains(after.Replace, "flush ruleset") {
		t.Fatal("guard lacks restricted bridge forward hook or uses global nft mutation")
	}
	for _, s := range []string{
		"ip saddr 10.55.8.11 ip daddr 10.55.8.12 accept",
		"ip6 saddr fd55:8::11 ip6 daddr fd55:8::12 accept",
		"ip saddr 10.55.8.12 ip daddr 10.55.8.11 ct state established,related accept",
		"ip6 saddr fd55:8::12 ip6 daddr fd55:8::11 ct state established,related accept",
		"ip saddr 10.55.8.13 ip daddr 10.55.8.14 accept",
		"ip6 saddr fd55:8::13 ip6 daddr fd55:8::14 accept",
		"ip saddr 10.55.8.0/24 ip daddr 10.55.8.0/24 counter drop",
		"ip6 saddr fd55:8::/64 ip6 daddr fd55:8::/64 counter drop",
		"ip6 saddr fe80::/10 ip6 daddr fe80::/10 counter drop",
		"icmpv6 type { nd-neighbor-solicit, nd-neighbor-advert } accept",
	} {
		if !strings.Contains(after.Install, s) {
			t.Fatalf("missing bounded guard policy: %q", s)
		}
	}
	if strings.Contains(after.Install, "ip saddr 10.55.8.12 ip daddr 10.55.8.11 accept") ||
		strings.Contains(after.Install, "ip6 saddr fd55:8::12 ip6 daddr fd55:8::11 accept") {
		t.Fatal("replies became unconditionally authorized reverse grant")
	}
	if strings.Contains(before.Install, "ip saddr 10.55.8.11 ip daddr 10.55.8.12 accept") {
		t.Fatal("prior guard mistakenly admitted A->B")
	}
	afterRevoke := pr178Guard(t, members, []state.PeerGrantBinding{cd}, []state.PeerGrantBinding{ab})
	if afterRevoke.Install != before.Install {
		t.Fatal("revocation failed to remove allow and retain C/D unchanged")
	}
	if afterRevoke.DigestSHA256 == after.DigestSHA256 {
		t.Fatal("guard digest unchanged after revoke")
	}
}
func TestPR178GuardRefusesUnsafeNativeInventoryAndSnapshots(t *testing.T) {
	ab := pr178Grant(pr178A, pr178B, "10.55.8.11", "10.55.8.12", "fd55:8::11", "fd55:8::12")
	members := pr178GuardMembers()
	cases := []struct {
		name   string
		change func(*[]NativePeerGuardMember, *[]state.PeerGrantBinding, *NativePrepareTarget)
	}{
		{"missing-box", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) { *m = (*m)[:1] }},
		{"mac-duplicate", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			(*m)[1].MAC = (*m)[0].MAC
		}},
		{"mac-multicast", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			(*m)[1].MAC = "01:00:5e:00:00:01"
		}},
		{"mac-injection", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			(*m)[1].MAC = "bad\nadd table bridge evil"
		}},
		{"uuid-forged", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			(*m)[0].UUID = "not-uuid"
		}},
		{"ipv4-reused", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			(*m)[1].IPv4 = (*m)[0].IPv4
		}},
		{"ipv6-reused", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			(*m)[1].IPv6 = (*m)[0].IPv6
		}},
		{"ipv6-zone", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			(*m)[0].IPv6 = "fd55:8::11%eth0"
		}},
		{"ipv4-foreign", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			(*m)[0].IPv4 = "192.168.44.11"
		}},
		{"identity-drift", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			(*m)[0].IPv4 = "10.55.8.19"
		}},
		{"hidden-foreign-grant", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			*g = append(*g, pr178Grant(pr178C, "55555555-5555-4555-8555-555555555555", "10.55.8.13", "10.55.8.99", "fd55:8::13", "fd55:8::99"))
		}},
		{"bridge-injection", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			t.Network = "incusbr;flush ruleset"
		}},
		{"project-injection", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			t.Project = "user-1001;evil"
		}},
		{"foreign-acl", func(m *[]NativePeerGuardMember, g *[]state.PeerGrantBinding, t *NativePrepareTarget) {
			t.ACL = "incus-managed"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := append([]NativePeerGuardMember(nil), members...)
			g := []state.PeerGrantBinding{ab}
			target := pr178GuardTarget()
			tc.change(&m, &g, &target)
			plan, err := NativePeerGuardFromSnapshot(target, s4b2UnitBridge(), m, g, nil)
			if err == nil {
				t.Fatalf("unsafe policy generated: %+v", plan)
			}
		})
	}
}
