package install

import (
	"reflect"
	"strings"
	"testing"
)

// TestS4B2NativeFaultLabStaleETag measures two competing snapshot writers
// against a disposable Incus ACL. This is a native SDK probe, not a product
// peer authorization or evidence of atomic per-NIC kernel reload.
func TestS4B2NativeFaultLabStaleETag(t *testing.T) {
	c := s4b1NativeProbe(t)
	before, staleTag, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || len(before.Ingress) != 2 {
		t.Fatal("expected granted ACL", err)
	}
	first := before.NetworkACLPut
	first.Description = "S4-B2 ETag competing-writer native probe"
	if err := c.UpdateNetworkACL(s4b2DedicatedACL, first, staleTag); err != nil {
		t.Fatal(err)
	}
	afterFirst, freshTag, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || freshTag == staleTag || afterFirst.Description != first.Description {
		t.Fatal("first writer not independently observed", err)
	}
	second := before.NetworkACLPut
	second.Description = "S4-B2 stale writer must never win"
	err = c.UpdateNetworkACL(s4b2DedicatedACL, second, staleTag)
	if err == nil {
		t.Fatal("stale second writer silently overwrote newer ACL")
	}
	normalized := strings.ToLower(err.Error())
	if !strings.Contains(normalized, "412") && !strings.Contains(normalized, "precondition") && !strings.Contains(normalized, "etag") {
		t.Fatalf("unexpected stale writer error: %v", err)
	}
	final, finalTag, err := c.GetNetworkACL(s4b2DedicatedACL)
	if err != nil || finalTag != freshTag || !reflect.DeepEqual(final.NetworkACLPut, afterFirst.NetworkACLPut) {
		t.Fatal("second writer changed ACL despite stale ETag", err)
	}
	t.Log("S4_B2_FAULT_LAB_STALE_SECOND_WRITER_REJECTED=PASS")
}

// This test requires A/B/C and an additional NIC-consuming instance created
// by the isolated native workflow. The guard must refuse the unknown consumer
// even though the native provider places it in the same Incus Project.
func TestS4B2NativeFaultLabUnknownSameProjectConsumer(t *testing.T) {
	c := s4b1NativeProbe(t)
	p := c.UseProject(s4b2DedicatedProject)
	owned := map[string]string{}
	for _, name := range []string{"s4b2a", "s4b2b", "s4b2c"} {
		inst, _, err := p.GetInstance(name)
		if err != nil {
			t.Fatal(err)
		}
		uuid := inst.Config["volatile.uuid"]
		if uuid == "" {
			t.Fatalf("missing native UUID of %s", name)
		}
		owned[name] = uuid
	}
	unknown, _, err := p.GetInstance("s4b2unknown")
	if err != nil || unknown.Config["volatile.uuid"] == "" {
		t.Fatal("unknown native instance not present", err)
	}
	nw, _, err := c.GetNetwork(s4b2DedicatedBridge)
	if err != nil {
		t.Fatal(err)
	}
	present := false
	for _, used := range nw.UsedBy {
		if strings.Contains(used, "/instances/s4b2unknown") {
			present = true
		}
	}
	if !present {
		t.Fatalf("unknown same-project consumer absent from root UsedBy: %v", nw.UsedBy)
	}
	if err := nativePeerBridgeConsumersOwned(nw.UsedBy, s4b2DedicatedProject, owned); err == nil {
		t.Fatal("unknown same-project NIC was incorrectly treated as BoxWeave-owned")
	}
	t.Log("S4_B2_FAULT_LAB_UNKNOWN_SAME_PROJECT_NATIVE_CONSUMER_REJECTED=PASS")
}
