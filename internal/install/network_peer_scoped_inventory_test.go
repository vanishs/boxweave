package install

import (
	"slices"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
)

func pr178ScopedInventoryFixture() (
	[]native.PeerACLBox, []string, map[string]NativePeerGuardObservedOwner, []string,
) {
	claims := []native.PeerACLBox{
		{Name: "pr178-a", UUID: pr178A, IPv4: "10.55.8.11", IPv6: "fd55:8::11", MAC: "10:66:6a:e2:1e:fd"},
		{Name: "pr178-b", UUID: pr178B, IPv4: "10.55.8.12", IPv6: "fd55:8::12", MAC: "10:66:6a:84:b1:ee"},
	}
	names := []string{"pr178-a", "pr178-b"}
	observed := map[string]NativePeerGuardObservedOwner{}
	links := []string{"/1.0/profiles/default?project=user-1001"}
	for i, b := range claims {
		acl, _ := NativePeerScopedACLName("user-1001", "incusbr-1001", b.UUID)
		entry := "/1.0/instances/" + b.Name + "?project=user-1001"
		boot := "true"
		if i == 1 {
			boot = "false"
		}
		observed[b.Name] = NativePeerGuardObservedOwner{
			Instance: &api.Instance{
				Name: b.Name, Project: "user-1001", Status: "Running",
				InstancePut: api.InstancePut{
					Config: api.ConfigMap{"volatile.uuid": b.UUID,
						"volatile.eth0.hwaddr": b.MAC, "boot.autostart": boot},
					Profiles: []string{"default"},
					Devices: api.DevicesMap{
						"eth0": {"type": "nic", "ipv4.address": b.IPv4, "ipv6.address": b.IPv6},
					},
				},
				ExpandedDevices: api.DevicesMap{
					"eth0": {
						"type": "nic", "name": "eth0", "network": "incusbr-1001",
						"security.acls":                        acl,
						"security.acls.default.ingress.action": "reject",
						"security.acls.default.egress.action":  "reject",
						"security.ipv4_filtering":              "true", "security.ipv6_filtering": "true",
						"security.mac_filtering": "true",
						"ipv4.address":           b.IPv4, "ipv6.address": b.IPv6,
					},
				},
			},
			ACL: &api.NetworkACL{
				NetworkACLPost: api.NetworkACLPost{Name: acl},
				Project:        "user-1001",
				UsedBy:         []string{entry},
			},
			ACLTag: "etag-for-" + b.Name,
		}
		links = append(links, entry)
	}
	return claims, names, observed, links
}

func TestPR178NativeScopedGuardCompleteProjectInventory(t *testing.T) {
	claims, names, observed, usedBy := pr178ScopedInventoryFixture()
	members, err := NativePeerGuardValidateScopedInventory(
		"user-1001", "incusbr-1001", claims, names, observed, usedBy)
	if err != nil || len(members) != len(claims) {
		t.Fatalf("exact native inventory rejected: %+v %v", members, err)
	}
	for i, m := range members {
		if m.UUID != claims[i].UUID || m.MAC != claims[i].MAC ||
			m.IPv4 != claims[i].IPv4 || m.IPv6 != claims[i].IPv6 {
			t.Fatal("native projection changed owner identity")
		}
	}
	b, err := NativePeerGuardFromSnapshot(pr178GuardTarget(), s4b2UnitBridge(), members, nil, nil)
	if err != nil || b.DigestSHA256 == "" {
		t.Fatal("verified owners could not compile guard", err)
	}
}
func TestPR178NativeScopedGuardRejectsIncompleteAndForeignInventory(t *testing.T) {
	cases := []struct {
		name   string
		change func(*[]native.PeerACLBox, *[]string, map[string]NativePeerGuardObservedOwner, *[]string)
	}{
		{"missing-sibling", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			*n = (*n)[:1]
		}},
		{"extra-sibling", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			*n = append(*n, "unknown-owner")
		}},
		{"duplicate-root-name", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			(*n)[1] = (*n)[0]
		}},
		{"foreign-network-profile", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			*u = append(*u, "/1.0/profiles/default?project=foreign")
		}},
		{"foreign-network-bridge-instance", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			*u = append(*u, "/1.0/instances/other?project=other")
		}},
		{"hidden-network-consumers", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			*u = nil
		}},
		{"missing-project-query", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			(*u)[0] = "/1.0/profiles/default"
		}},
		{"extra-usedby-query", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			(*u)[0] += "&foo=bar"
		}},
		{"acl-shared-consumer", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			a := o["pr178-a"]
			a.ACL.UsedBy = append(a.ACL.UsedBy, "/1.0/instances/pr178-b?project=user-1001")
			o["pr178-a"] = a
		}},
		{"acl-foreign-project", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			a := o["pr178-a"]
			a.ACL.Project = "other"
			o["pr178-a"] = a
		}},
		{"acl-hijacked-name", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			a := o["pr178-a"]
			a.ACL.Name = "bw-evil"
			o["pr178-a"] = a
		}},
		{"acl-stale-etag", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			a := o["pr178-a"]
			a.ACLTag = ""
			o["pr178-a"] = a
		}},
		{"wrong-native-uuid", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			a := o["pr178-a"]
			a.Instance.Config["volatile.uuid"] = pr178C
			o["pr178-a"] = a
		}},
		{"stale-mac", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			a := o["pr178-a"]
			a.Instance.Config["volatile.eth0.hwaddr"] = "10:66:6a:11:11:11"
			o["pr178-a"] = a
		}},
		{"duplicate-native-uuid", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			(*c)[1].UUID = (*c)[0].UUID
		}},
		{"duplicate-mac", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			(*c)[1].MAC = (*c)[0].MAC
		}},
		{"wrong-lease", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			a := o["pr178-a"]
			a.Instance.Devices["eth0"]["ipv4.address"] = "10.55.8.99"
			o["pr178-a"] = a
		}},
		{"unfiltered-ipv6", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			a := o["pr178-a"]
			a.Instance.ExpandedDevices["eth0"]["security.ipv6_filtering"] = "false"
			o["pr178-a"] = a
		}},
		{"stopped-foreigner", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			a := o["pr178-a"]
			a.Instance.Status = "Error"
			o["pr178-a"] = a
		}},
		{"alien-default-profile", func(c *[]native.PeerACLBox, n *[]string, o map[string]NativePeerGuardObservedOwner, u *[]string) {
			a := o["pr178-a"]
			a.Instance.Profiles = []string{"foreign-profile"}
			o["pr178-a"] = a
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims, names, observed, usedBy := pr178ScopedInventoryFixture()
			tc.change(&claims, &names, observed, &usedBy)
			if got, err := NativePeerGuardValidateScopedInventory(
				"user-1001", "incusbr-1001", claims, names, observed, usedBy); err == nil {
				t.Fatalf("unsafe complete Project scope accepted: %+v", got)
			}
		})
	}
	claims, names, observed, usedBy := pr178ScopedInventoryFixture()
	slices.Reverse(claims)
	slices.Reverse(names)
	slices.Reverse(usedBy)
	if _, err := NativePeerGuardValidateScopedInventory(
		"user-1001", "incusbr-1001", claims, names, observed, usedBy); err != nil {
		t.Fatal("root inventory should not depend on SDK list order:", err)
	}
}
