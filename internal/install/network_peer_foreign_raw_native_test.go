package install

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/native"
)

// Only the fresh CI daemon is permitted to attach a foreign unfiltered raw
// bridge NIC. The product root audit must refuse before any ACL write.
func TestS4B2NativeRootRejectsForeignRawNIC(t *testing.T) {
	if os.Getenv("BOXWEAVE_S4B2_FOREIGN_RAW_NIC") != "fresh-ci-only" ||
		os.Getenv("GITHUB_ACTIONS") != "true" || os.Geteuid() != 0 {
		t.Skip("disposable native foreign raw-NIC attack only")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	owner := root.UseProject("bwowned")
	scope := NativePeerBridgeAuditTarget{Project: "bwowned", Bridge: "bwbr0", ACL: "bw-s4b2-native"}
	for _, name := range []string{"s4b2a", "s4b2b", "s4b2c"} {
		inst, _, err := owner.GetInstance(name)
		if err != nil {
			t.Fatal(err)
		}
		mac := inst.Config["volatile.eth0.hwaddr"]
		if mac == "" {
			mac = inst.ExpandedDevices["eth0"]["hwaddr"]
		}
		scope.Boxes = append(scope.Boxes, native.PeerACLBox{
			Name: name, UUID: inst.Config["volatile.uuid"], MAC: mac,
			IPv4: inst.Devices["eth0"]["ipv4.address"],
			IPv6: inst.Devices["eth0"]["ipv6.address"],
		})
	}
	bridge, _, err := root.GetNetwork(scope.Bridge)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("S4_B2_FOREIGN_RAW_NIC_NATIVE_BRIDGE_USED_BY=%v", bridge.UsedBy)
	_, err = nativePeerBridgeAuditWithClient(ctx, root, scope)
	if err == nil {
		t.Fatal("foreign Project unfiltered bridged NIC was invisible to root bridge ACL safety audit")
	}
	if !strings.Contains(err.Error(), "foreign") && !strings.Contains(err.Error(), "non-BoxWeave") {
		t.Fatalf("root audit refusal must prove foreign network attachment, not another incidental error: %v", err)
	}
	t.Logf("S4_B2_FOREIGN_RAW_NIC_ROOT_AUDIT_DENIED=PASS reason=%v", err)
}
