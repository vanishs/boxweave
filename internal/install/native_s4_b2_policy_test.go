package install

import (
	"net/netip"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func s4b2UnitBridge() *api.Network {
	return &api.Network{
		Type: "bridge", Managed: true, Status: "Created",
		NetworkPut: api.NetworkPut{Config: api.ConfigMap{
			"ipv4.address": "10.55.8.1/24",
			"ipv6.address": "fd55:8::1/64",
		}},
	}
}

func s4b2TestPair() nativePeerBinding {
	return nativePeerBinding{
		SourceUUID: "box-a-uuid", TargetUUID: "box-b-uuid",
		SourceIPv4: netip.MustParseAddr("10.55.8.11"),
		TargetIPv4: netip.MustParseAddr("10.55.8.12"),
		SourceIPv6: netip.MustParseAddr("fd55:8::11"),
		TargetIPv6: netip.MustParseAddr("fd55:8::12"),
	}
}
func s4b2CIDRMatches(raw string, addr netip.Addr) bool {
	for _, s := range strings.Split(raw, ",") {
		p, err := netip.ParsePrefix(s)
		if err == nil && p.Contains(addr) {
			return true
		}
	}
	return false
}
func TestS4B2ExactPeerCompilerKeepsS1AndForeignDenials(t *testing.T) {
	target := NativePrepareTarget{ACL: "bw-s4b2-unit"}
	network := s4b2UnitBridge()
	s1, err := nativePrepareACL(target, network)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := nativeACLForVerifiedPeers(target, network, nil)
	if err != nil || !reflect.DeepEqual(s1, unchanged) {
		t.Fatal("zero peers changed exact S1 ACL", err)
	}
	pair := s4b2TestPair()
	proposed, err := nativeACLForVerifiedPeers(target, network, []nativePeerBinding{pair})
	if err != nil {
		t.Fatal(err)
	}
	if len(proposed.Ingress) != 2 || len(proposed.Egress) != 4 {
		t.Fatalf("unexpected peer rules ingress=%d egress=%d", len(proposed.Ingress), len(proposed.Egress))
	}
	for i, v := range []struct{ target, foreign netip.Addr }{
		{pair.TargetIPv4, netip.MustParseAddr("10.55.8.99")},
		{pair.TargetIPv6, netip.MustParseAddr("fd55:8::99")},
	} {
		if s4b2CIDRMatches(proposed.Egress[i].Destination, v.target) {
			t.Fatal("approved target remains in blocked reject range")
		}
		if !s4b2CIDRMatches(proposed.Egress[i].Destination, v.foreign) {
			t.Fatal("foreign same-bridge IP was unintentionally exempted")
		}
		if proposed.Ingress[i].Action != "allow" || proposed.Egress[i+2].Action != "allow" {
			t.Fatal("both explicit ingress/egress are required for internet=false")
		}
	}
	if _, err := nativeACLForVerifiedPeers(target, network, []nativePeerBinding{pair, pair}); err == nil {
		t.Fatal("duplicate peer rule accepted")
	}
	reused := pair
	reused.TargetUUID = "different-c-uuid"
	if _, err := nativeACLForVerifiedPeers(target, network, []nativePeerBinding{pair, reused}); err == nil {
		t.Fatal("same address assigned to two native identities")
	}
	outside := pair
	outside.TargetIPv4 = netip.MustParseAddr("192.168.1.3")
	if _, err := nativeACLForVerifiedPeers(target, network, []nativePeerBinding{outside}); err == nil {
		t.Fatal("out-of-bridge address admitted")
	}
}

func s4b2NativeEnv(t *testing.T) nativePeerBinding {
	t.Helper()
	pair := nativePeerBinding{}
	get := func(key string) netip.Addr {
		raw := os.Getenv(key)
		ip, err := netip.ParseAddr(raw)
		if err != nil {
			t.Fatalf("required exact native %s missing: %v", key, err)
		}
		return ip
	}
	pair.SourceIPv4 = get("S4B2_A4")
	pair.TargetIPv4 = get("S4B2_B4")
	pair.SourceIPv6 = get("S4B2_A6")
	pair.TargetIPv6 = get("S4B2_B6")
	return pair
}

// This probe is valid ONLY on a fresh disposable native Incus runner with
// BOXWEAVE_DISPOSABLE_NATIVE_PROBE=s4b1-ci-20261116. It is not a live grant.
func TestS4B2NativeExactTargetEgressAndIngress(t *testing.T) {
	c := s4b1NativeProbe(t)
	pair := s4b2NativeEnv(t)
	for _, spec := range []struct {
		name string
		uuid *string
	}{
		{"s4b2a", &pair.SourceUUID}, {"s4b2b", &pair.TargetUUID},
	} {
		inst, _, err := c.GetInstance(spec.name)
		if err != nil {
			t.Fatal(err)
		}
		*spec.uuid = inst.Config["volatile.uuid"]
		dev := inst.ExpandedDevices["eth0"]
		if *spec.uuid == "" || dev["security.acls"] != s4b1ProbeACL ||
			dev["security.acls.default.ingress.action"] != "reject" ||
			dev["security.ipv4_filtering"] != "true" ||
			dev["security.ipv6_filtering"] != "true" ||
			dev["security.mac_filtering"] != "true" {
			t.Fatalf("%s missing exact S1 native NIC/UUID", spec.name)
		}
	}
	network, _, err := c.GetNetwork("incusbr0")
	if err != nil {
		t.Fatal(err)
	}
	acl, tag, err := c.GetNetworkACL(s4b1ProbeACL)
	if err != nil {
		t.Fatal(err)
	}
	base, err := nativePrepareACL(NativePrepareTarget{ACL: s4b1ProbeACL}, network)
	if err != nil || !reflect.DeepEqual(acl.Egress, base.Egress) || len(acl.Ingress) != 0 {
		t.Fatal("native fixture is not exact S1 starting ACL", err)
	}
	desired, err := nativeACLForVerifiedPeers(NativePrepareTarget{ACL: s4b1ProbeACL}, network, []nativePeerBinding{pair})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateNetworkACL(s4b1ProbeACL, desired.NetworkACLPut, tag); err != nil {
		t.Fatal(err)
	}
	got, _, err := c.GetNetworkACL(s4b1ProbeACL)
	if err != nil || !reflect.DeepEqual(got.Ingress, desired.Ingress) || !reflect.DeepEqual(got.Egress, desired.Egress) {
		t.Fatal("native exact-host ACL ETag write/readback mismatch", err)
	}
	t.Log("S4_B2_NATIVE_EXACT_PEER_S1_POLICY_SDK_READBACK=PASS")
}
func TestS4B2NativeRevokeBeforeIPRelease(t *testing.T) {
	c := s4b1NativeProbe(t)
	network, _, err := c.GetNetwork("incusbr0")
	if err != nil {
		t.Fatal(err)
	}
	acl, tag, err := c.GetNetworkACL(s4b1ProbeACL)
	if err != nil {
		t.Fatal(err)
	}
	if len(acl.Ingress) != 2 || len(acl.Egress) != 4 {
		t.Fatal("expected exact previous peer grant")
	}
	baseline, err := nativePrepareACL(NativePrepareTarget{ACL: s4b1ProbeACL}, network)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateNetworkACL(s4b1ProbeACL, baseline.NetworkACLPut, tag); err != nil {
		t.Fatal(err)
	}
	got, _, err := c.GetNetworkACL(s4b1ProbeACL)
	if err != nil || !reflect.DeepEqual(got.Egress, baseline.Egress) || len(got.Ingress) != 0 {
		t.Fatal("native ACL revoke not independently read back", err)
	}
	t.Log("S4_B2_NATIVE_REVOKE_ACL_BEFORE_RELEASE_READBACK=PASS")
}
