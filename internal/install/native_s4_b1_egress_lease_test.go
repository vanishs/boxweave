package install

// Native, GitHub-hosted disposable-VM experiments only. This test NEVER runs
// against a shared daemon or under ordinary `go test` without an explicit opt-in.
import (
	"fmt"
	"net/netip"
	"os"
	"reflect"
	"strings"
	"testing"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

const s4b1ProbeACL = "bw-s4b1-native"

func s4b1NativeProbe(t *testing.T) incus.InstanceServer {
	t.Helper()
	if os.Getenv("BOXWEAVE_DISPOSABLE_NATIVE_PROBE") != "s4b1-ci-20261116" {
		t.Skip("requires explicit ephemeral CI-only native daemon")
	}
	if os.Geteuid() != 0 {
		t.Fatal("requires local throwaway runner administrator")
	}
	c, err := incus.ConnectIncusUnix("/var/lib/incus/unix.socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestS4B1ExactS1ACLNativeFixture(t *testing.T) {
	c := s4b1NativeProbe(t)
	network, _, err := c.GetNetwork("incusbr0")
	if err != nil {
		t.Fatal(err)
	}
	acl, err := nativePrepareACL(NativePrepareTarget{ACL: s4b1ProbeACL}, network)
	if err != nil {
		t.Fatal(err)
	}
	if len(acl.Ingress) != 0 || len(acl.Egress) != 2 || len(acl.Egress[0].Destination) == 0 || len(acl.Egress[1].Destination) == 0 {
		t.Fatal("not exact S1 deny-only ACL")
	}
	if err = c.CreateNetworkACL(acl); err != nil {
		t.Fatal(err)
	}
	got, _, err := c.GetNetworkACL(s4b1ProbeACL)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Egress, acl.Egress) || len(got.Ingress) != 0 {
		t.Fatal("S1 ACL native readback disagrees with actual source generator")
	}
	fmt.Printf("S4_B1_NATIVE_S1_EXACT_EGRESS_DENY_SOURCE_AND_READBACK=PASS ipv4=%s ipv6=%s\n", network.Config["ipv4.address"], network.Config["ipv6.address"])
}

func s4b1PrefixDifference(parent, excluded netip.Prefix) []netip.Prefix {
	parent = parent.Masked()
	excluded = excluded.Masked()
	if parent.Addr().Is4() != excluded.Addr().Is4() || !parent.Contains(excluded.Addr()) && !excluded.Contains(parent.Addr()) {
		return []netip.Prefix{parent}
	}
	if excluded.Bits() <= parent.Bits() {
		return nil
	}
	n := parent.Bits()
	first := netip.PrefixFrom(parent.Addr(), n+1).Masked()
	secondAddr := parent.Addr()
	if secondAddr.Is4() {
		v := secondAddr.As4()
		v[n/8] ^= byte(1 << uint(7-n%8))
		secondAddr = netip.AddrFrom4(v)
	} else {
		v := secondAddr.As16()
		v[n/8] ^= byte(1 << uint(7-n%8))
		secondAddr = netip.AddrFrom16(v)
	}
	second := netip.PrefixFrom(secondAddr, n+1).Masked()
	out := s4b1PrefixDifference(first, excluded)
	return append(out, s4b1PrefixDifference(second, excluded)...)
}

func TestS4B1CarveOnlyOwnedBridgeForNativeExperiment(t *testing.T) {
	c := s4b1NativeProbe(t)
	network, _, err := c.GetNetwork("incusbr0")
	if err != nil {
		t.Fatal(err)
	}
	acl, etag, err := c.GetNetworkACL(s4b1ProbeACL)
	if err != nil {
		t.Fatal(err)
	}
	s1, err := nativePrepareACL(NativePrepareTarget{ACL: s4b1ProbeACL}, network)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(acl.Egress, s1.Egress) {
		t.Fatal("not original unmodified S1 egress")
	}
	updated := acl.Writable()
	for i := range updated.Egress {
		var excluded netip.Prefix
		addr := network.Config["ipv4.address"]
		if i == 1 {
			addr = network.Config["ipv6.address"]
		}
		excluded, err = netip.ParsePrefix(addr)
		if err != nil {
			t.Fatal(err)
		}
		excluded = excluded.Masked()
		var pieces []string
		for _, raw := range strings.Split(updated.Egress[i].Destination, ",") {
			p, e := netip.ParsePrefix(strings.TrimSpace(raw))
			if e != nil {
				t.Fatal(e)
			}
			for _, part := range s4b1PrefixDifference(p, excluded) {
				pieces = append(pieces, part.String())
			}
		}
		updated.Egress[i].Destination = strings.Join(pieces, ",")
		for _, p := range pieces {
			q, _ := netip.ParsePrefix(p)
			if q.Contains(excluded.Addr()) {
				t.Fatalf("bridge subnet not removed from egress deny: %s", p)
			}
		}
	}
	if err = c.UpdateNetworkACL(s4b1ProbeACL, updated, etag); err != nil {
		t.Fatal(err)
	}
	after, _, err := c.GetNetworkACL(s4b1ProbeACL)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after.Egress, updated.Egress) {
		t.Fatal("test-only bridge-subnet carve not native-confirmed")
	}
	fmt.Printf("S4_B1_NATIVE_EXPERIMENT_CARVE_BRIDGE_SUBNET_TEST_ONLY=PASS ipv4=%s ipv6=%s\n", network.Config["ipv4.address"], network.Config["ipv6.address"])
}

func TestS4B1RevokeStaleRulesNativeExperiment(t *testing.T) {
	c := s4b1NativeProbe(t)
	acl, etag, err := c.GetNetworkACL(s4b1ProbeACL)
	if err != nil {
		t.Fatal(err)
	}
	if len(acl.Ingress) != 2 {
		t.Fatalf("expected stale A-B rules, got %d", len(acl.Ingress))
	}
	updated := acl.Writable()
	updated.Ingress = []api.NetworkACLRule{}
	if err = c.UpdateNetworkACL(s4b1ProbeACL, updated, etag); err != nil {
		t.Fatal(err)
	}
	observed, _, err := c.GetNetworkACL(s4b1ProbeACL)
	if err != nil {
		t.Fatal(err)
	}
	if len(observed.Ingress) != 0 {
		t.Fatal("stale rules remain")
	}
	fmt.Println("S4_B1_NATIVE_STALE_GRANT_EXPLICIT_REVOKE_READBACK=PASS")
}
