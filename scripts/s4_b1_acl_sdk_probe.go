//go:build ignore

// Disposable S4-B1 probe for official Incus 7.6 SDK ACL/ETag and NIC identity.
// This is not a BoxWeave controller implementation or an installed MCP test.
package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		panic("mode required: etag | grant | revoke | read")
	}
	client, err := incus.ConnectIncusUnix("/var/lib/incus/unix.socket", nil)
	must(err)
	if !client.HasExtension("network_acl") {
		panic("network_acl SDK extension unavailable")
	}
	aclName := "bw-s4b1-native"
	acl, etag, err := client.GetNetworkACL(aclName)
	must(err)
	for _, name := range []string{"s4b1a", "s4b1b"} {
		instance, _, err := client.GetInstance(name)
		must(err)
		uuid := instance.Config["volatile.uuid"]
		dev := instance.ExpandedDevices["eth0"]
		if uuid == "" || dev["security.acls"] != aclName ||
			dev["security.acls.default.ingress.action"] != "reject" ||
			dev["security.ipv4_filtering"] != "true" ||
			dev["security.ipv6_filtering"] != "true" ||
			dev["security.mac_filtering"] != "true" {
			panic("exact native UUID/effective NIC safety mismatch: " + name)
		}
		fmt.Printf("S4_B1_NATIVE_%s_UUID=%s\n", strings.ToUpper(name), uuid)
	}
	if len(acl.UsedBy) < 2 {
		panic("ACL used_by lacks two owned NIC references")
	}
	fmt.Printf("S4_B1_ACL_USED_BY_COUNT=%d\n", len(acl.UsedBy))

	switch os.Args[1] {
	case "etag":
		newACL := acl.Writable()
		newACL.Description = "S4-B1 native SDK ETag checked"
		must(client.UpdateNetworkACL(aclName, newACL, etag))
		err = client.UpdateNetworkACL(aclName, newACL, etag)
		if !api.StatusErrorCheck(err, http.StatusPreconditionFailed) {
			panic(fmt.Sprintf("stale ACL ETag did not fail HTTP 412: %v", err))
		}
		updated, _, err := client.GetNetworkACL(aclName)
		must(err)
		if updated.Description != newACL.Description {
			panic("ETag accepted write not visible on independent read")
		}
		fmt.Println("S4_B1_SDK_ACL_ETAG_412_NO_OVERWRITE=PASS")
	case "grant":
		if len(os.Args) != 6 {
			panic("grant needs A4 B4 A6 B6")
		}
		if len(acl.Ingress) != 0 {
			panic("unexpected ingress before grant")
		}
		rules := []api.NetworkACLRule{
			{Action: "allow", State: "enabled", Source: os.Args[2] + "/32", Destination: os.Args[3] + "/32", Description: "S4-B1 A to B IPv4 only"},
			{Action: "allow", State: "enabled", Source: os.Args[4] + "/128", Destination: os.Args[5] + "/128", Description: "S4-B1 A to B IPv6 only"},
		}
		write := acl.Writable()
		write.Ingress = rules
		must(client.UpdateNetworkACL(aclName, write, etag))
		after, _, err := client.GetNetworkACL(aclName)
		must(err)
		if len(after.Ingress) != 2 || after.Ingress[0].Source != rules[0].Source ||
			after.Ingress[1].Destination != rules[1].Destination {
			panic("grant ACL native readback mismatch")
		}
		fmt.Println("S4_B1_SDK_TWO_FAMILY_ACL_GRANT_READBACK=PASS")
	case "revoke":
		write := acl.Writable()
		write.Ingress = []api.NetworkACLRule{}
		must(client.UpdateNetworkACL(aclName, write, etag))
		after, _, err := client.GetNetworkACL(aclName)
		must(err)
		if len(after.Ingress) != 0 {
			panic("revoked ACL rules persist")
		}
		fmt.Println("S4_B1_SDK_TWO_FAMILY_ACL_REVOKE_READBACK=PASS")
	case "read":
		fmt.Printf("S4_B1_SDK_ACL_INGRESS_COUNT=%d\n", len(acl.Ingress))
	default:
		panic("unknown mode")
	}
}
