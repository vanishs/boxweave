package install

import (
	"net/netip"
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
	if _, err := NativePrepareArchive(t.Context(), strings.Repeat("0", 64), false); err == nil {
		t.Fatal("nonroot archive accepted")
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

// TestS25PrivateEgressBlocksRoutedForeignProjects locks the actual Incus
// reject-over-allow rule precedence. CIDRs MUST be disjoint from the single
// selected bridge gateway; an overlapping allow rule is ineffective.
func TestS25PrivateEgressBlocksRoutedForeignProjects(t *testing.T) {
	target := NativePrepareTarget{Project: "user-1001", ACL: "bw-s25-inner-owned"}
	network := &api.Network{
		Name: "incusbr-1001", Type: "bridge", Managed: true, Status: "Created",
		NetworkPut: api.NetworkPut{Config: api.ConfigMap{
			"ipv4.address": "10.223.5.1/24",
			"ipv6.address": "fd42:3af8:722d:a113::1/64",
		}},
	}
	acl, err := nativePrepareACL(target, network)
	if err != nil {
		t.Fatal(err)
	}
	if len(acl.Egress) != 2 || acl.Egress[0].Action != "reject" ||
		acl.Egress[1].Action != "reject" ||
		acl.Egress[0].State != "enabled" || acl.Egress[1].State != "enabled" {
		t.Fatalf("deny-only private egress ACL missing: %+v", acl.Egress)
	}
	checkDenied := func(family int, addr string, want bool) {
		t.Helper()
		rule := acl.Egress[0]
		if family == 6 {
			rule = acl.Egress[1]
		}
		ip := netip.MustParseAddr(addr)
		found := false
		for _, cidr := range strings.Split(rule.Destination, ",") {
			if netip.MustParsePrefix(cidr).Contains(ip) {
				if found {
					t.Fatalf("overlapping rule CIDRs for %s", addr)
				}
				found = true
			}
		}
		if found != want {
			t.Errorf("ACL denied %s: got=%v want=%v", addr, found, want)
		}
	}
	// Exactly one gateway IP is excluded. Sibling guests in the same
	// subnet and foreign host-routed projects are still blocked.
	for _, addr := range []string{
		"10.223.5.2", "10.179.98.182", "192.168.50.4",
		"172.20.0.2", "100.64.4.2", "169.254.6.3",
	} {
		checkDenied(4, addr, true)
	}
	checkDenied(4, "10.223.5.1", false)
	checkDenied(4, "1.1.1.1", false)
	for _, addr := range []string{
		"fd42:3af8:722d:a113::2", "fd42:abcd::1", "fe80::abcd",
	} {
		checkDenied(6, addr, true)
	}
	checkDenied(6, "fd42:3af8:722d:a113::1", false)
	checkDenied(6, "2001:db8:25:2026::2", false)
	got := &api.NetworkACL{NetworkACLPost: acl.NetworkACLPost, NetworkACLPut: acl.NetworkACLPut}
	if !nativeACLPrivateEgressSafe(got, network) {
		t.Fatal("exact native ACL policy not accepted")
	}
	got.Egress = nil
	if nativeACLPrivateEgressSafe(got, network) {
		t.Fatal("empty foreign-project-permissive ACL unexpectedly accepted")
	}
	got.Egress = append([]api.NetworkACLRule(nil), acl.Egress...)
	got.Egress[0].Destination = "10.0.0.0/8"
	if nativeACLPrivateEgressSafe(got, network) {
		t.Fatal("ACL that blocks own gateway not refused")
	}
	got.Egress = append([]api.NetworkACLRule(nil), acl.Egress...)
	got.Egress[0].Action = "allow"
	if nativeACLPrivateEgressSafe(got, network) {
		t.Fatal("private destination explicit allow incorrectly accepted")
	}
	got.Egress = append([]api.NetworkACLRule(nil), acl.Egress...)
	got.Egress[0], got.Egress[1] = got.Egress[1], got.Egress[0]
	if nativeACLPrivateEgressSafe(got, network) {
		t.Fatal("unreviewed ACL rule mutation incorrectly accepted")
	}
}

func TestS25PrivateEgressRefusesUnreviewedBridgeGateway(t *testing.T) {
	target := NativePrepareTarget{Project: "user-1001", ACL: "bw-s25-inner-owned"}
	network := &api.Network{Type: "bridge", Managed: true, Status: "Created",
		NetworkPut: api.NetworkPut{Config: api.ConfigMap{"ipv4.address": "auto"}}}
	if _, err := nativePrepareACL(target, network); err == nil {
		t.Fatal("autogenerated/unspecified gateway unexpectedly passed review")
	}
	network.Config["ipv4.address"] = "10.4.1.1/24"
	network.Config["ipv6.address"] = "none"
	got, err := nativePrepareACL(target, network)
	if err != nil || len(got.Egress) != 2 {
		t.Fatalf("IPv6-disabled bridge must still deny ULA destinations: %+v, %v", got.Egress, err)
	}
	network.Config["ipv6.address"] = "not-a-prefix"
	if _, err = nativePrepareACL(target, network); err == nil {
		t.Fatal("invalid IPv6 bridge gateway not rejected")
	}
}
