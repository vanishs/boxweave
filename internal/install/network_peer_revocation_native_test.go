package install

import (
	"reflect"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

// This is exclusively a fresh privileged Incus 7.6 CI test; it is not a
// product ACL writer. The original native operation and ETag are observed
// without any host-wide conntrack manipulation.
func TestS4B2NativeCompiledDropTombstoneApply(t *testing.T) {
	c := s4b1NativeProbe(t)
	pair := s4b2NativeEnv(t)
	owned := c.UseProject(s4b2DedicatedProject)
	a, _, err := owned.GetInstance("s4b2a")
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := owned.GetInstance("s4b2b")
	if err != nil {
		t.Fatal(err)
	}
	pair.SourceUUID = a.Config["volatile.uuid"]
	pair.TargetUUID = b.Config["volatile.uuid"]
	network, _, err := c.GetNetwork(s4b2DedicatedBridge)
	if err != nil {
		t.Fatal(err)
	}
	acl, tag, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil {
		t.Fatal(err)
	}
	base, err := nativePrepareACL(NativePrepareTarget{ACL: s4b2DedicatedACL}, network)
	if err != nil || len(acl.Ingress) != 0 || !reflect.DeepEqual(acl.Egress, base.Egress) {
		t.Fatal("native exact revoke fixture was not the baseline", err)
	}
	desired, err := nativeACLWithDeniedPeers(NativePrepareTarget{ACL: s4b2DedicatedACL, Project: s4b2DedicatedProject}, network, nil, []nativePeerBinding{pair})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.UpdateNetworkACL(s4b2DedicatedACL, desired.NetworkACLPut, tag); err != nil {
		t.Fatalf("native exact directed dual-stack DROP ETag PUT failed: %v", err)
	}
	got, _, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || !reflect.DeepEqual(got.Ingress, desired.Ingress) || !reflect.DeepEqual(got.Egress, desired.Egress) {
		t.Fatal("native SDK DROP tombstone original readback differs", err)
	}
	for _, rules := range [][]api.NetworkACLRule{got.Ingress, got.Egress} {
		for _, r := range rules {
			if r.Action == "allow" {
				t.Fatal("native revoked ACL still contains allow")
			}
		}
	}
	t.Log("S4_B2_NATIVE_PRODUCT_COMPILED_DROP_DUALSTACK_APPLY_READBACK=PASS")
}
func TestS4B2NativeCompiledDropTombstoneRemove(t *testing.T) {
	c := s4b1NativeProbe(t)
	pair := s4b2NativeEnv(t)
	p := c.UseProject(s4b2DedicatedProject)
	a, _, err := p.GetInstance("s4b2a")
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := p.GetInstance("s4b2b")
	if err != nil {
		t.Fatal(err)
	}
	pair.SourceUUID = a.Config["volatile.uuid"]
	pair.TargetUUID = b.Config["volatile.uuid"]
	network, _, err := c.GetNetwork(s4b2DedicatedBridge)
	if err != nil {
		t.Fatal(err)
	}
	acl, etag, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil {
		t.Fatal(err)
	}
	desired, err := nativeACLWithDeniedPeers(NativePrepareTarget{ACL: s4b2DedicatedACL, Project: s4b2DedicatedProject}, network, nil, []nativePeerBinding{pair})
	if err != nil || !reflect.DeepEqual(acl.Ingress, desired.Ingress) || !reflect.DeepEqual(acl.Egress, desired.Egress) {
		t.Fatal("native DROP tombstone was altered before exact restore", err)
	}
	base, err := nativePrepareACL(NativePrepareTarget{ACL: s4b2DedicatedACL}, network)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.UpdateNetworkACL(s4b2DedicatedACL, base.NetworkACLPut, etag); err != nil {
		t.Fatalf("original native baseline restore refused: %v", err)
	}
	got, _, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || len(got.Ingress) != 0 || !reflect.DeepEqual(got.Egress, base.Egress) {
		t.Fatal("native original ACL rulebook failed exact restore", err)
	}
	t.Log("S4_B2_NATIVE_PRODUCT_COMPILED_DROP_EXACT_RESTORE=PASS")
}
