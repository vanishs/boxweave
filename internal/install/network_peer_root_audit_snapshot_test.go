package install

import (
	"github.com/lxc/incus/v7/shared/api"
	"testing"
)

// A same-size project or NIC mutation is a real cross-Project spoofing risk.
// The old audit compared lengths only; this regression suite captures that.
func TestS4B2RootAuditSnapshotDetectsSameSizeSecurityDrift(t *testing.T) {
	n := &api.Network{}
	n.Config = api.ConfigMap{"ipv4.address": "10.248.83.1/24", "ipv6.address": "fd83:248::1/64"}
	acl := &api.NetworkACL{}
	acl.NetworkACLPost.Name = "bw-s4b2-native"
	p := &api.Project{}
	p.Name = "bwowned"
	p.Config = api.ConfigMap{"restricted": "true", "restricted.networks.access": "bwbr0"}
	prof := &api.Profile{}
	prof.Name = "default"
	prof.Devices = api.DevicesMap{"eth0": {"type": "nic", "network": "bwbr0", "security.ipv4_filtering": "true"}}
	tag := "stable-etag"
	eq := func(n2 *api.Network, nt string, a2 *api.NetworkACL, at string, p2 *api.Project, pt string, f2 *api.Profile, ft string) bool {
		return nativePeerAuditSnapshotEqual(n, tag, acl, tag, p, tag, prof, tag, n2, nt, a2, at, p2, pt, f2, ft)
	}
	if !eq(n, tag, acl, tag, p, tag, prof, tag) {
		t.Fatal("stable identical snapshot refused")
	}
	n.UsedBy = []string{"/1.0/instances/a", "/1.0/instances/b"}
	n2order := *n
	n2order.UsedBy = []string{"/1.0/instances/b", "/1.0/instances/a"}
	if !eq(&n2order, tag, acl, tag, p, tag, prof, tag) {
		t.Fatal("same native UsedBy identities must not fail due only to listing order")
	}
	n2order.UsedBy = []string{"/1.0/instances/a", "/1.0/instances/foreign"}
	if eq(&n2order, tag, acl, tag, p, tag, prof, tag) {
		t.Fatal("foreign native bridge consumer accepted in equal-length UsedBy")
	}
	if eq(n, "new-etag", acl, tag, p, tag, prof, tag) {
		t.Fatal("network ETag drift ignored")
	}
	p2 := *p
	p2.Config = api.ConfigMap{"restricted": "true", "restricted.networks.access": "foreign-br"}
	if eq(n, tag, acl, tag, &p2, tag, prof, tag) {
		t.Fatal("same-size Project bridge scope mutation accepted")
	}
	prof2 := *prof
	prof2.Devices = api.DevicesMap{"eth0": {"type": "nic", "network": "foreign-br", "security.ipv4_filtering": "true"}}
	if eq(n, tag, acl, tag, p, tag, &prof2, tag) {
		t.Fatal("same-size NIC bridge mutation accepted")
	}
	prof2.Devices = api.DevicesMap{"eth0": {"type": "nic", "network": "bwbr0", "security.ipv4_filtering": "false"}}
	if eq(n, tag, acl, tag, p, tag, &prof2, tag) {
		t.Fatal("same-size NIC anti-spoof mutation accepted")
	}
	n2 := *n
	n2.Config = api.ConfigMap{"ipv4.address": "10.248.84.1/24", "ipv6.address": "fd83:248::1/64"}
	if eq(&n2, tag, acl, tag, p, tag, prof, tag) {
		t.Fatal("same-size native gateway mutation accepted")
	}
	a2 := *acl
	a2.Ingress = []api.NetworkACLRule{{Action: "allow", Source: "0.0.0.0/0"}}
	if eq(n, tag, &a2, tag, p, tag, prof, tag) {
		t.Fatal("same-ETag changed native ACL admitted")
	}
	if eq(nil, tag, acl, tag, p, tag, prof, tag) {
		t.Fatal("missing native network admitted")
	}
}
