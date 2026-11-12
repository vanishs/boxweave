package install

import (
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func TestPrereqDevicesFailClosed(t *testing.T) {
	safe := api.DevicesMap{
		"root": {"type": "disk", "path": "/", "pool": "bw-pool"},
		"eth0": {"type": "nic", "network": "bw-br", "security.acls": "bw-acl",
			"security.acls.default.ingress.action": "reject",
			"security.ipv4_filtering":              "true", "security.ipv6_filtering": "true", "security.mac_filtering": "true"},
	}
	pool, net, acl, ok := prereqDevices(safe)
	if !ok || pool != "bw-pool" || net != "bw-br" || acl != "bw-acl" {
		t.Fatalf("safe devices rejected: %q %q %q %v", pool, net, acl, ok)
	}
	tests := []struct {
		name   string
		change func(api.DevicesMap)
	}{
		{"missing_ipv6_filter", func(m api.DevicesMap) { delete(m["eth0"], "security.ipv6_filtering") }},
		{"open_ingress", func(m api.DevicesMap) { m["eth0"]["security.acls.default.ingress.action"] = "allow" }},
		{"shared_acls", func(m api.DevicesMap) { m["eth0"]["security.acls"] = "bw-acl,foreign" }},
		{"missing_root", func(m api.DevicesMap) { delete(m, "root") }},
		{"host_mount", func(m api.DevicesMap) { m["etc"] = map[string]string{"type": "disk", "path": "/etc", "source": "/etc"} }},
		{"proxy", func(m api.DevicesMap) { m["hostproxy"] = map[string]string{"type": "proxy"} }},
		{"preexisting_root_volume", func(m api.DevicesMap) { m["root"]["source"] = "foreign" }},
		{"second_nic", func(m api.DevicesMap) { m["eth1"] = map[string]string{"type": "nic"} }},
		{"unsafe_pool", func(m api.DevicesMap) { m["root"]["pool"] = "../foreign" }},
		{"unnamed_acl", func(m api.DevicesMap) { m["eth0"]["security.acls"] = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := api.DevicesMap{}
			for n, d := range safe {
				cp[n] = map[string]string{}
				for k, v := range d {
					cp[n][k] = v
				}
			}
			tt.change(cp)
			if _, _, _, ok := prereqDevices(cp); ok {
				t.Fatal("unsafe profile accepted")
			}
		})
	}
}

func TestPrereqACLExclusivity(t *testing.T) {
	const proj = "user-1001"
	tests := []struct {
		refs []string
		ok   bool
	}{
		{[]string{"/1.0/profiles/default?project=user-1001"}, true},
		{[]string{"/1.0/profiles/default?project=user-1001", "/1.0/instances/owned?project=user-1001"}, true},
		{[]string{"/1.0/profiles/default"}, false},
		{[]string{"/1.0/networks/incusbr0?project=user-1001"}, false},
		{[]string{"/1.0/profiles/default?project=default"}, false},
		{[]string{"/1.0/profiles/default?project=user-1001", "/1.0/instances/foreign?project=default"}, false},
		{[]string{"/1.0/instances/owned?project=user-1001"}, false},
		{[]string{"/1.0/profiles/default?project=user-1001&other=yes"}, false},
		{nil, false},
	}
	for _, tt := range tests {
		if got := aclExclusiveUse(tt.refs, proj); got != tt.ok {
			t.Errorf("ACL used_by=%q: got %v want %v", tt.refs, got, tt.ok)
		}
	}
}
func TestPrereqDigestBindsNativeIdentity(t *testing.T) {
	r := NativePrereqReport{HostSHA256: "host-a", Project: "user-1001", Pool: "pool",
		Network: "bridge", ACL: "acl", ImageFingerprint: "image",
		Checks: []NativePrereqCheck{{ID: "dir_pool_backing", OK: true, Detail: "free 10G"}}}
	hash := prereqDigest(r)
	r.Checks[0].Detail = "free 9G"
	if prereqDigest(r) != hash {
		t.Fatal("volatile human-readable details changed review digest")
	}
	r.ImageFingerprint = "other-image"
	if prereqDigest(r) == hash {
		t.Fatal("new image must require reapproval")
	}
	r.ImageFingerprint = "image"
	r.Checks[0].OK = false
	if prereqDigest(r) == hash {
		t.Fatal("new failed prerequisite must require reapproval")
	}
	r.Checks[0].OK = true
	r.QuotaMountPrecondition = true
	if prereqDigest(r) == hash {
		t.Fatal("changed quota evidence must require reapproval")
	}
}

func TestConfinedProjectInvocationIgnoresOperatorIncusConfig(t *testing.T) {
	args := confinedProjectListArgs("/home/boxweave")
	want := []string{"-u", "boxweave", "--", "env", "-u", "INCUS_CONF", "-u", "INCUS_CONFIG_DIR", "-u", "XDG_CONFIG_HOME",
		"HOME=/home/boxweave", "XDG_CONFIG_HOME=/home/boxweave/.config", "incus", "project", "list", "--format", "csv"}
	if len(args) != len(want) {
		t.Fatalf("unexpected restricted native invocation %v", args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("arg[%d]=%q want %q", i, args[i], want[i])
		}
	}
}
