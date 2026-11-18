package install

import (
	"net/netip"
	"os"
	"testing"
)

// This invokes the production kernel nft readback parser against ACTUAL
// Incus 7.6 per-NIC chains in a fresh, host-root disposable CI runner,
// immediately after the native ETag dual-stack DROP has been installed.
// It is not evidence of an installed controller or a real root journal.
func TestS4B2NativeRootKernelPerNICDropReadback(t *testing.T) {
	if os.Getenv("BOXWEAVE_DISPOSABLE_NATIVE_PROBE") != "s4b1-ci-20261116" ||
		os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("fresh host-root Incus 7.6 native fixture only")
	}
	if os.Geteuid() != 0 {
		t.Fatal("independent native kernel readback requires host root")
	}
	rules, err := nativePeerRootKernelRules(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	pair := nativePeerBinding{
		SourceIPv4: netip.MustParseAddr("10.248.83.11"),
		TargetIPv4: netip.MustParseAddr("10.248.83.12"),
		SourceIPv6: netip.MustParseAddr("fd83:248::11"),
		TargetIPv6: netip.MustParseAddr("fd83:248::12"),
	}
	if err := nativePeerRootVerifyNFT(rules, "bwowned", []string{"s4b2a", "s4b2b", "s4b2c"}, pair); err != nil {
		t.Fatalf("real per-NIC nft DROP did not converge: %v", err)
	}
	t.Log("S4_B2_NATIVE_INTEGRATED_ROOT_NFT_DUALSTACK_PER_NIC=PASS")
}
