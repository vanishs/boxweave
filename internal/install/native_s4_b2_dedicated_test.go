package install

import (
	"net/netip"
	"reflect"
	"testing"
)

const s4b2DedicatedACL = "bw-s4b2-native"
const s4b2DedicatedBridge = "bwbr0"
const s4b2DedicatedProject = "bwowned"

// All three tests below are gated by s4b1NativeProbe, which refuses to run
// without a root-local disposable Incus 7.6 CI daemon and exact opt-in.
// They are *not* an installed BoxWeave Peer writer.
func TestS4B2NativeDedicatedBridgeACLSetup(t *testing.T) {
	c := s4b1NativeProbe(t)
	netw, _, err := c.GetNetwork(s4b2DedicatedBridge)
	if err != nil {
		t.Fatal(err)
	}
	if !netw.Managed || netw.Type != "bridge" {
		t.Fatal("not dedicated managed bridge")
	}
	base, err := nativePrepareACL(NativePrepareTarget{ACL: s4b2DedicatedACL}, netw)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.CreateNetworkACL(base); err != nil {
		t.Fatal(err)
	}
	got, _, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || !reflect.DeepEqual(got.Egress, base.Egress) || len(got.Ingress) != 0 {
		t.Fatal("dedicated bridge exact S1 starting ACL mismatch", err)
	}
	t.Log("S4_B2_NATIVE_DEDICATED_EXACT_S1_ACL=PASS")
}

func TestS4B2NativeDedicatedBridgeExactPeer(t *testing.T) {
	c := s4b1NativeProbe(t)
	p := c.UseProject(s4b2DedicatedProject)
	pair := s4b2NativeEnv(t)
	names := map[string]string{}
	for _, spec := range []struct {
		name string
		id   *string
	}{
		{"s4b2a", &pair.SourceUUID}, {"s4b2b", &pair.TargetUUID}, {"s4b2c", nil},
	} {
		inst, _, err := p.GetInstance(spec.name)
		if err != nil {
			t.Fatal(err)
		}
		uuid := inst.Config["volatile.uuid"]
		if uuid == "" {
			t.Fatalf("native UUID missing for %s", spec.name)
		}
		names[spec.name] = uuid
		if spec.id != nil {
			*spec.id = uuid
		}
		dev := inst.ExpandedDevices["eth0"]
		if dev["security.acls"] != s4b2DedicatedACL ||
			dev["network"] != s4b2DedicatedBridge ||
			dev["security.acls.default.ingress.action"] != "reject" ||
			dev["security.acls.default.egress.action"] != "reject" ||
			dev["security.ipv4_filtering"] != "true" ||
			dev["security.ipv6_filtering"] != "true" ||
			dev["security.mac_filtering"] != "true" ||
			dev["ipv4.address"] == "" || dev["ipv6.address"] == "" {
			t.Fatalf("%s lacks exact owned anti-spoof static NIC", spec.name)
		}
	}
	nw, _, err := c.GetNetwork(s4b2DedicatedBridge)
	if err != nil {
		t.Fatal(err)
	}
	if err := nativePeerBridgeConsumersOwned(nw.UsedBy, s4b2DedicatedProject, names); err != nil {
		t.Fatalf("dedicated bridge has an unknown native consumer: %v %v", err, nw.UsedBy)
	}
	for _, spec := range []struct {
		name       string
		ipv4, ipv6 netip.Addr
	}{
		{"s4b2a", pair.SourceIPv4, pair.SourceIPv6}, {"s4b2b", pair.TargetIPv4, pair.TargetIPv6},
	} {
		inst, _, err := p.GetInstance(spec.name)
		if err != nil {
			t.Fatal(err)
		}
		dev := inst.ExpandedDevices["eth0"]
		if dev["ipv4.address"] != spec.ipv4.String() || dev["ipv6.address"] != spec.ipv6.String() {
			t.Fatalf("native static NIC identity for %s differs from guest", spec.name)
		}
	}
	acl, tag, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil {
		t.Fatal(err)
	}
	base, err := nativePrepareACL(NativePrepareTarget{ACL: s4b2DedicatedACL}, nw)
	if err != nil || len(acl.Ingress) != 0 || !reflect.DeepEqual(acl.Egress, base.Egress) {
		t.Fatal("native ACL fixture changed before exact grant", err)
	}
	desired, err := nativeACLForVerifiedPeers(NativePrepareTarget{ACL: s4b2DedicatedACL}, nw, []nativePeerBinding{pair})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateNetworkACL(s4b2DedicatedACL, desired.NetworkACLPut, tag); err != nil {
		t.Fatal(err)
	}
	observed, _, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || !reflect.DeepEqual(observed.Egress, desired.Egress) || !reflect.DeepEqual(observed.Ingress, desired.Ingress) {
		t.Fatal("dedicated native ACL readback failed", err)
	}
	t.Log("S4_B2_NATIVE_DEDICATED_EXCLUSIVE_BRIDGE_NIC_IDENTITY_AND_EXACT_ACL=PASS")
}

func TestS4B2NativeDedicatedBridgeRevoke(t *testing.T) {
	c := s4b1NativeProbe(t)
	nw, _, err := c.GetNetwork(s4b2DedicatedBridge)
	if err != nil {
		t.Fatal(err)
	}
	acl, tag, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || len(acl.Ingress) != 2 {
		t.Fatal("dedicated peer not granted", err)
	}
	base, err := nativePrepareACL(NativePrepareTarget{ACL: s4b2DedicatedACL}, nw)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateNetworkACL(s4b2DedicatedACL, base.NetworkACLPut, tag); err != nil {
		t.Fatal(err)
	}
	after, _, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || len(after.Ingress) != 0 || !reflect.DeepEqual(after.Egress, base.Egress) {
		t.Fatal("dedicated peer revoke native poststate unverified", err)
	}
	t.Log("S4_B2_NATIVE_DEDICATED_REVOKE_BEFORE_ADDRESS_RELEASE=PASS")
}
