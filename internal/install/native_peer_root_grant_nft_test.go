package install

import (
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

func TestS4B3GrantRootKernelWitnessExactDirectionalIPv4IPv6(t *testing.T) {
	boxes := []native.PeerACLBox{
		{Name: "a", UUID: "a", IPv4: "10.55.8.11", IPv6: "fd55:8::11"},
		{Name: "b", UUID: "b", IPv4: "10.55.8.12", IPv6: "fd55:8::12"},
		{Name: "c", UUID: "c", IPv4: "10.55.8.13", IPv6: "fd55:8::13"},
	}
	grant := []state.PeerGrantBinding{{SourceUUID: "a", TargetUUID: "b",
		SourceIPv4: "10.55.8.11", TargetIPv4: "10.55.8.12",
		SourceIPv6: "fd55:8::11", TargetIPv6: "fd55:8::12"}}
	allow4 := "ip saddr 10.55.8.11 ip daddr 10.55.8.12 accept"
	allow6 := "ip6 saddr fd55:8::11 ip6 daddr fd55:8::12 accept"
	makeRules := func() string {
		var s strings.Builder
		s.WriteString("table inet incus {\n")
		for _, b := range boxes {
			for _, c := range []string{"in", "fwd", "out"} {
				s.WriteString(" chain " + c + ".user-1001_" + b.Name + ".eth0 {\n")
				if b.Name == "a" || b.Name == "b" {
					s.WriteString(" " + allow4 + "\n " + allow6 + "\n")
				}
				s.WriteString(" }\n")
			}
		}
		s.WriteString("}\n")
		return s.String()
	}
	valid := makeRules()
	check := func(name, raw string, grants []state.PeerGrantBinding) {
		t.Helper()
		if err := nativePeerRootVerifyGrantNFT(raw, "user-1001", boxes, []string{"a", "b", "c"}, grants); err == nil {
			t.Fatalf("unsafe or incomplete grant accepted: %s", name)
		}
	}
	if err := nativePeerRootVerifyGrantNFT(valid, "user-1001", boxes, []string{"a", "b", "c"}, grant); err != nil {
		t.Fatal("exact synthetic dual-NIC directional grant refused", err)
	}
	check("missing-ip6", strings.ReplaceAll(valid, allow6, ""), grant)
	check("missing-destination", strings.ReplaceAll(valid, " "+allow4, ""), grant)
	check("reverse-unapproved", strings.Replace(valid, allow4, allow4+"\n ip saddr 10.55.8.12 ip daddr 10.55.8.11 accept", 1), grant)
	check("missing-chain", strings.Replace(valid, "chain fwd.user-1001_b.eth0", "chain fwd.foreign_b.eth0", 1), grant)
	check("duplicate-chain", valid+valid, grant)
	bad := append([]state.PeerGrantBinding{}, grant...)
	bad[0].SourceIPv4 = "10.55.8.99"
	check("stale-identity", valid, bad)
	t.Log("S4_B3_PRS_GRANT_NFT_DIRECTIONAL_DUALSTACK_FAIL_CLOSED=PASS")
}
