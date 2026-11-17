package install

import (
	"net/netip"
	"reflect"
	"strings"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func revocationTestNetwork() *api.Network {
	n := &api.Network{}
	n.Managed = true
	n.Type = "bridge"
	n.Status = "Created"
	n.Config = api.ConfigMap{
		"ipv4.address": "10.248.83.1/24",
		"ipv6.address": "fd83:248::1/64",
	}
	return n
}
func revocationTestPair(dstUUID, dst4, dst6 string) nativePeerBinding {
	return nativePeerBinding{
		SourceUUID: "uuid-a", TargetUUID: dstUUID,
		SourceIPv4: netip.MustParseAddr("10.248.83.11"), TargetIPv4: netip.MustParseAddr(dst4),
		SourceIPv6: netip.MustParseAddr("fd83:248::11"), TargetIPv6: netip.MustParseAddr(dst6),
	}
}
func TestS4B2RevokedPeerCompilesDualStackDropWithoutGrant(t *testing.T) {
	n := revocationTestNetwork()
	target := NativePrepareTarget{ACL: "bw-test-drop", Project: "bwowned"}
	pair := revocationTestPair("uuid-b", "10.248.83.12", "fd83:248::12")
	base, err := nativePrepareACL(target, n)
	if err != nil {
		t.Fatal(err)
	}
	acl, err := nativeACLWithDeniedPeers(target, n, nil, []nativePeerBinding{pair})
	if err != nil {
		t.Fatal(err)
	}
	if len(acl.Ingress) != 2 || len(acl.Egress) != 4 {
		t.Fatalf("wrong direction and family coverage: ingress %d egress %d", len(acl.Ingress), len(acl.Egress))
	}
	for _, rules := range [][]api.NetworkACLRule{acl.Ingress, acl.Egress[2:]} {
		for i, rule := range rules {
			if rule.Action != "drop" || rule.State != "enabled" || rule.Protocol != "" ||
				rule.Source == "" || rule.Destination == "" || strings.Contains(rule.Source, ",") || strings.Contains(rule.Destination, ",") {
				t.Fatalf("unexpected tombstone %d: %+v", i, rule)
			}
		}
	}
	for _, rule := range append(append([]api.NetworkACLRule{}, acl.Ingress...), acl.Egress...) {
		if rule.Action == "allow" {
			t.Fatal("revoked pair was granted while compiling tombstone")
		}
	}
	if !reflect.DeepEqual(base.Egress[0].Destination, acl.Egress[0].Destination) {
		// Target /32 may be carved only while an exact native address is
		// claimed, with the DROP on that same pair always present. This
		// is not an independent broad egress permission.
		if !strings.Contains(acl.Egress[0].Destination, ",") {
			t.Fatal("unexpected non-specific egress carveout")
		}
	}
}
func TestS4B2RevokedAndActivePeerDisjointAndDeterministic(t *testing.T) {
	tgt := NativePrepareTarget{ACL: "bw-test-drop", Project: "bwowned"}
	n := revocationTestNetwork()
	revoked := revocationTestPair("uuid-b", "10.248.83.12", "fd83:248::12")
	active := revocationTestPair("uuid-c", "10.248.83.13", "fd83:248::13")
	acl, err := nativeACLWithDeniedPeers(tgt, n, []nativePeerBinding{active}, []nativePeerBinding{revoked})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, rule := range acl.Ingress {
		counts[rule.Action]++
	}
	if counts["allow"] != 2 || counts["drop"] != 2 {
		t.Fatalf("unexpected native pairs: %+v", counts)
	}
	deny4 := exactPeerRuleKey(revoked.SourceIPv4, revoked.TargetIPv4, 32)
	deny6 := exactPeerRuleKey(revoked.SourceIPv6, revoked.TargetIPv6, 128)
	for _, rule := range acl.Ingress {
		key := rule.Source + "|" + rule.Destination
		if (key == deny4 || key == deny6) && rule.Action != "drop" {
			t.Fatalf("native revoked grant not suppressed %+v", rule)
		}
	}
	for _, variant := range []struct{ active, denied []nativePeerBinding }{
		{active: []nativePeerBinding{active}, denied: []nativePeerBinding{revoked}},
	} {
		again, err := nativeACLWithDeniedPeers(tgt, n, variant.active, variant.denied)
		if err != nil || !reflect.DeepEqual(acl, again) {
			t.Fatal("nondeterministic or unsafe denied pair compiler", err)
		}
	}
	if _, err := nativeACLWithDeniedPeers(tgt, n, []nativePeerBinding{revoked}, []nativePeerBinding{revoked}); err == nil {
		t.Fatal("same exact UUID pair both active and revoked")
	}
	if _, err := nativeACLWithDeniedPeers(tgt, n, nil, []nativePeerBinding{{SourceUUID: "uuid-a", TargetUUID: "uuid-b", SourceIPv4: netip.MustParseAddr("127.0.0.1")}}); err == nil {
		t.Fatal("malformed native tombstone identity admitted")
	}
}
func TestS4B2EmptyTombstonePreservesExactOldPolicy(t *testing.T) {
	tgt := NativePrepareTarget{ACL: "bw-test-drop", Project: "bwowned"}
	n := revocationTestNetwork()
	active := revocationTestPair("uuid-b", "10.248.83.12", "fd83:248::12")
	base, err := nativeACLForVerifiedPeers(tgt, n, []nativePeerBinding{active})
	if err != nil {
		t.Fatal(err)
	}
	next, err := nativeACLWithDeniedPeers(tgt, n, []nativePeerBinding{active}, nil)
	if err != nil || !reflect.DeepEqual(base, next) {
		t.Fatal("preexisting grant policy changed without revocation", err)
	}
}
