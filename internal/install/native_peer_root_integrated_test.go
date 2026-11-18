package install

import (
	"net/netip"
	"strings"
	"testing"
)

func TestS4B2InstalledRootKernelPerNICReadbackRefusesMissingOrAllowedRules(t *testing.T) {
	pair := nativePeerBinding{
		SourceIPv4: netip.MustParseAddr("10.248.83.11"),
		TargetIPv4: netip.MustParseAddr("10.248.83.12"),
		SourceIPv6: netip.MustParseAddr("fd83:248::11"),
		TargetIPv6: netip.MustParseAddr("fd83:248::12"),
	}
	lines := []string{
		"ip saddr 10.248.83.11 ip daddr 10.248.83.12 drop",
		"ip saddr 10.248.83.12 ip daddr 10.248.83.11 drop",
		"ip6 saddr fd83:248::11 ip6 daddr fd83:248::12 drop",
		"ip6 saddr fd83:248::12 ip6 daddr fd83:248::11 drop",
	}
	fixture := func(boxes []string) string {
		var rules strings.Builder
		rules.WriteString("table inet incus {\n")
		for _, name := range boxes {
			for _, chain := range []string{"in", "fwd", "out"} {
				rules.WriteString("    chain " + chain + ".user-1001_" + name + ".eth0 {\n")
				for _, line := range lines {
					rules.WriteString("        " + line + "\n")
				}
				rules.WriteString("    }\n")
			}
		}
		rules.WriteString("}\n")
		return rules.String()
	}
	valid := fixture([]string{"a", "b", "c"})
	check := func(name, raw string, boxes []string) {
		t.Helper()
		if err := nativePeerRootVerifyNFT(raw, "user-1001", boxes, pair); err == nil {
			t.Fatalf("%s accepted incomplete or unsafe native per-NIC nft state", name)
		}
	}
	if err := nativePeerRootVerifyNFT(valid, "user-1001", []string{"a", "b", "c"}, pair); err != nil {
		t.Fatal("exact complete synthetic kernel witness rejected", err)
	}
	check("missing-drop", strings.Replace(valid, lines[1], "", 1), []string{"a", "b", "c"})
	check("old-accept", strings.Replace(valid, lines[0], strings.Replace(lines[0], "drop", "accept", 1), 1), []string{"a", "b", "c"})
	check("missing-NIC", strings.Replace(valid, "chain fwd.user-1001_b.eth0", "chain fwd.foreign_b.eth0", 1), []string{"a", "b", "c"})
	check("repeated-chain", valid+fixture([]string{"a"}), []string{"a", "b", "c"})
	check("missing-foreign-target", valid, []string{"a", "b", "c", "foreign"})
	t.Log("S4_B2_INSTALLED_ROOT_PER_NIC_NFT_EXACT_DROP_FAIL_CLOSED=PASS")
}
