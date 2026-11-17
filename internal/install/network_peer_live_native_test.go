package install

import (
	"reflect"
	"testing"
)

// Deliberately unsafe-live mutation EXCLUSIVELY in a freshly provisioned
// throwaway Incus 7.6 CI host, for firewall/conntrack research. Production
// writes MUST use the all-sibling native pre-PUT quiescence barrier and
// durable UNKNOWN claims; these tests are not a public Peer implementation.
func TestS4B2NativeCompiledLiveTrackedFlowDrop(t *testing.T) {
	c := s4b1NativeProbe(t)
	pair := s4b2NativeEnv(t)
	owner := c.UseProject(s4b2DedicatedProject)
	a, _, err := owner.GetInstance("s4b2a")
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := owner.GetInstance("s4b2b")
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != "Running" || b.Status != "Running" {
		t.Fatal("TCP/UDP source and target not live")
	}
	pair.SourceUUID = a.Config["volatile.uuid"]
	pair.TargetUUID = b.Config["volatile.uuid"]
	network, _, err := c.GetNetwork(s4b2DedicatedBridge)
	if err != nil {
		t.Fatal(err)
	}
	target := NativePrepareTarget{ACL: s4b2DedicatedACL}
	expected, err := nativeACLForVerifiedPeers(target, network, []nativePeerBinding{pair})
	if err != nil {
		t.Fatal(err)
	}
	got, etag, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || !reflect.DeepEqual(got.Ingress, expected.Ingress) || !reflect.DeepEqual(got.Egress, expected.Egress) {
		t.Fatal("guest TCP/UDP tracking not protected by expected native grant", err)
	}
	deny, err := nativeACLWithDeniedPeers(target, network, nil, []nativePeerBinding{pair})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.UpdateNetworkACL(s4b2DedicatedACL, deny.NetworkACLPut, etag); err != nil {
		t.Fatal(err)
	}
	updated, _, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || !reflect.DeepEqual(updated.Ingress, deny.Ingress) || !reflect.DeepEqual(updated.Egress, deny.Egress) {
		t.Fatal("native live DROP refused full ETag readback", err)
	}
	t.Log("S4_B2_DISPOSABLE_LIVE_ESTABLISHED_TCP_UDP_DUALSTACK_EXACT_DROP_NATIVE_READBACK=PASS")
}
func TestS4B2NativeCompiledLiveTrackedFlowRestore(t *testing.T) {
	c := s4b1NativeProbe(t)
	pair := s4b2NativeEnv(t)
	owner := c.UseProject(s4b2DedicatedProject)
	a, _, err := owner.GetInstance("s4b2a")
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := owner.GetInstance("s4b2b")
	if err != nil {
		t.Fatal(err)
	}
	pair.SourceUUID = a.Config["volatile.uuid"]
	pair.TargetUUID = b.Config["volatile.uuid"]
	network, _, err := c.GetNetwork(s4b2DedicatedBridge)
	if err != nil {
		t.Fatal(err)
	}
	target := NativePrepareTarget{ACL: s4b2DedicatedACL}
	expected, err := nativeACLWithDeniedPeers(target, network, nil, []nativePeerBinding{pair})
	if err != nil {
		t.Fatal(err)
	}
	current, etag, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || !reflect.DeepEqual(current.Ingress, expected.Ingress) || !reflect.DeepEqual(current.Egress, expected.Egress) {
		t.Fatal("native live tombstone mutated by unexpected actor", err)
	}
	granted, err := nativeACLForVerifiedPeers(target, network, []nativePeerBinding{pair})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.UpdateNetworkACL(s4b2DedicatedACL, granted.NetworkACLPut, etag); err != nil {
		t.Fatal(err)
	}
	readback, _, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || !reflect.DeepEqual(readback.Ingress, granted.Ingress) || !reflect.DeepEqual(readback.Egress, granted.Egress) {
		t.Fatal("live test fixture original permitted policy not restored", err)
	}
	t.Log("S4_B2_DISPOSABLE_LIVE_TCP_UDP_NATIVE_ACL_EXACT_RESTORE=PASS")
}
