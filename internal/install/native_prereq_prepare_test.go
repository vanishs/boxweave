package install

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func TestS25NativePrepareDigestRejectsChangedNativeOrTarget(t *testing.T) {
	base := NativePreparePlan{
		Target:     NativePrepareTarget{Project: "user-1002", Pool: "default", Network: "incusbr0", ACL: "bw-s25-exclusive"},
		HostSHA256: "host", NativeStateSHA256: "etag-and-native-data",
		Actions: []string{"a"}, Warnings: []string{"quota-unverified"},
	}
	original := nativePrepareDigest(base)
	base.ReviewSHA256 = original
	if nativePrepareDigest(base) != original {
		t.Fatal("digest includes its own value")
	}
	base.Target.Pool = "foreign"
	if nativePrepareDigest(base) == original {
		t.Fatal("pool substitution accepted")
	}
	base.Target.Pool = "default"
	base.Target.ACL = "bw-other"
	if nativePrepareDigest(base) == original {
		t.Fatal("ACL substitution accepted")
	}
	base.Target.ACL = "bw-s25-exclusive"
	base.NativeStateSHA256 = "changed-profile"
	if nativePrepareDigest(base) == original {
		t.Fatal("ETag/native state substitution accepted")
	}
}

func TestS25NativePrepareProfileContainsOnlySafeDevices(t *testing.T) {
	target := NativePrepareTarget{Project: "user-1002", Pool: "default", Network: "incusbr0", ACL: "bw-s25-exclusive"}
	before := api.ProfilePut{Description: "owned profile", Config: api.ConfigMap{"user.foo": "bar"}, Devices: api.DevicesMap{}}
	after := nativePrepareProfile(target, before)
	if !reflect.DeepEqual(before.Config, after.Config) || after.Description != before.Description {
		t.Fatal("unrelated owned profile fields changed")
	}
	pool, network, acl, ok := prereqDevices(after.Devices)
	if !ok || pool != target.Pool || network != target.Network || acl != target.ACL {
		t.Fatal("native prepared Profile is unsafe")
	}
	after.Devices["eth0"]["security.ipv6_filtering"] = "false"
	if _, _, _, ok := prereqDevices(after.Devices); ok {
		t.Fatal("weakened anti-spoof NIC accepted")
	}
}

func TestS25NativePrepareProjectOnlyWidensOwnedNesting(t *testing.T) {
	before := api.ProjectPut{Description: "restricted", Config: api.ConfigMap{"restricted": "true", "features.images": "true"}}
	after := nativePrepareProject(before)
	if before.Config["restricted.containers.nesting"] != "" {
		t.Fatal("original mutated")
	}
	if after.Config["restricted.containers.nesting"] != "allow" || after.Config["restricted"] != "true" ||
		after.Config["features.images"] != "true" || after.Config["restricted.devices.disk"] != "managed" ||
		after.Config["restricted.devices.gpu"] != "block" {
		t.Fatal("Project restrictions lost")
	}
}

func TestS25NativePrepareRefusesUnapprovedNonRootAccess(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root-only negative is exercised by dedicated CLI suite")
	}
	target := NativePrepareTarget{Project: "user-1001", Pool: "default", Network: "incusbr0", ACL: "bw-s25"}
	if _, err := NativePreparePlanReview(t.Context(), target); err == nil || !strings.Contains(err.Error(), "root") {
		t.Fatalf("nonroot review accepted %v", err)
	}
	if _, err := NativePrepareApply(t.Context(), target, strings.Repeat("0", 64), false); err == nil {
		t.Fatal("unapproved native mutation accepted")
	}
	if _, err := NativePrepareStatus(); err == nil {
		t.Fatal("nonroot journal read accepted")
	}
	if _, err := NativePrepareRollback(t.Context(), strings.Repeat("0", 64), false); err == nil {
		t.Fatal("nonroot rollback accepted")
	}
}

func TestS25NativeAutoUserProfileEligibility(t *testing.T) {
	target := NativePrepareTarget{Project: "user-1002", Pool: "default", Network: "incusbr-1002", ACL: "bw-s25-lab"}
	defaults := api.DevicesMap{
		"root": {"type": "disk", "pool": "default", "path": "/"},
		"eth0": {"type": "nic", "name": "eth0", "network": "incusbr-1002"},
	}
	if err := nativePrepareProfileEligible(defaults, target); err != nil {
		t.Fatalf("real Incus-created default profile must be eligible: %v", err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(api.DevicesMap)
	}{
		{"foreign_device", func(d api.DevicesMap) { d["proxy"] = map[string]string{"type": "proxy"} }},
		{"host_disk", func(d api.DevicesMap) { d["root"]["source"] = "/etc" }},
		{"foreign_network", func(d api.DevicesMap) { d["eth0"]["network"] = "incusbr0" }},
		{"preexisting_acl", func(d api.DevicesMap) { d["eth0"]["security.acls"] = "foreign" }},
		{"unapproved_filter", func(d api.DevicesMap) { d["eth0"]["security.ipv6_filtering"] = "false" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copied := api.DevicesMap{}
			for n, dev := range defaults {
				copied[n] = map[string]string{}
				for k, v := range dev {
					copied[n][k] = v
				}
			}
			tc.mutate(copied)
			if err := nativePrepareProfileEligible(copied, target); err == nil {
				t.Fatal("preparer accepted unsafe existing native Profile")
			}
		})
	}
}
