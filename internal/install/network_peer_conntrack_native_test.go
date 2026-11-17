package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
)

// Isolated Incus 7.6 root daemon only. Never enabled on shared L0.
// Run after a complete three-Box native stop/autostart barrier and an
// exact dual-stack DROP ETag write, before any device is started again.
func TestS4B2NativeRootPairConntrackAfterQuiescence(t *testing.T) {
	if os.Getenv("BOXWEAVE_DISPOSABLE_NATIVE_PROBE") != "s4b1-ci-20261116" ||
		os.Getenv("GITHUB_ACTIONS") != "true" || os.Geteuid() != 0 {
		t.Skip("root-only disposable native conntrack primitive")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	own := root.UseProject("bwowned")
	scope := NativePeerBridgeAuditTarget{Project: "bwowned", Bridge: "bwbr0", ACL: "bw-s4b2-native"}
	for _, name := range []string{"s4b2a", "s4b2b", "s4b2c"} {
		inst, _, err := own.GetInstance(name)
		if err != nil {
			t.Fatal(err)
		}
		mac := inst.Config["volatile.eth0.hwaddr"]
		if mac == "" {
			mac = inst.ExpandedDevices["eth0"]["hwaddr"]
		}
		if inst.Status != "Stopped" || inst.Config["boot.autostart"] != "false" {
			t.Fatalf("must run after complete native autostart-safe stop barrier %s", name)
		}
		scope.Boxes = append(scope.Boxes, native.PeerACLBox{
			Name: name, UUID: inst.Config["volatile.uuid"], MAC: mac,
			IPv4: inst.Devices["eth0"]["ipv4.address"],
			IPv6: inst.Devices["eth0"]["ipv6.address"],
		})
	}
	if err := nativePeerBridgeKernelNoLivePorts(ctx, scope.Bridge); err != nil {
		t.Fatal("stopped dedicated bridge still has kernel ports:", err)
	}
	// Independently attach a foreign raw veth without Incus's UsedBy.
	// Only on this disposable GitHub-owned Incus runner, never shared L0.
	cmd := func(args ...string) error {
		output, err := exec.CommandContext(ctx, "/usr/sbin/ip", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("host-only test ip %v: %s: %w", args, output, err)
		}
		return nil
	}
	if err := cmd("link", "add", "bwprb0", "type", "veth", "peer", "name", "bwprb1"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = exec.Command("/usr/sbin/ip", "link", "del", "bwprb0").Run() }()
	if err := cmd("link", "set", "bwprb0", "master", scope.Bridge); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerBridgeKernelNoLivePorts(ctx, scope.Bridge); err == nil {
		t.Fatal("unregistered raw host bridge NIC evaded kernel port guard")
	}
	t.Log("S4_B2_NATIVE_ROOT_UNREGISTERED_HOST_BRIDGE_PORT_REFUSED=PASS")
	if err := cmd("link", "del", "bwprb0"); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerBridgeKernelNoLivePorts(ctx, scope.Bridge); err != nil {
		t.Fatal("foreign test link cleanup failed:", err)
	}
	t.Log("S4_B2_NATIVE_ROOT_ZERO_HOST_BRIDGE_PORTS_AFTER_STOP=PASS")
	a, b := scope.Boxes[0], scope.Boxes[1]
	actual, _, err := root.GetNetworkACL(scope.ACL)
	if err != nil {
		t.Fatal(err)
	}
	if err := nativePeerACLDropCovered(actual, a, b); err != nil {
		t.Fatal("native actual dual-stack DROP missing: ", err)
	}
	receipt, err := nativePeerConntrackWithClient(ctx, root, scope, a.UUID, b.UUID, nativePeerConntrackOS)
	if err != nil {
		t.Fatal("root bounded conntrack invalidation uncertain: ", err)
	}
	if receipt.ReadyForPeer || receipt.ProbedScopes != 4 ||
		len(receipt.NativeDigestSHA256) != 64 || !strings.Contains(receipt.PairIPv4, "|") {
		t.Fatalf("scope proof incomplete: %+v", receipt)
	}
	t.Logf("S4_B2_PRODUCT_NATIVE_ROOT_EXACT_PAIR_CONNTRACK_READBACK_PASS scopes=%d deleted=%d digest=%s",
		receipt.ProbedScopes, receipt.DeletedScopes, receipt.NativeDigestSHA256)
	rejectCalls := 0
	refused, err := nativePeerConntrackWithClient(ctx, root, scope, "nonexistent-uuid", b.UUID,
		func(_ context.Context, _ ...string) ([]byte, error) { rejectCalls++; return nil, nil })
	if err == nil || rejectCalls != 0 || refused.ReadyForPeer {
		t.Fatal("foreign requested native identity reached root conntrack")
	}
	t.Log("S4_B2_NATIVE_ROOT_FOREIGN_UUID_CONNTRACK_REFUSED=PASS")
	bad := scope
	bad.Boxes = append([]native.PeerACLBox(nil), scope.Boxes...)
	bad.Boxes[2].MAC = "02:11:22:33:44:55"
	rejectCalls = 0
	if _, err := nativePeerConntrackWithClient(ctx, root, bad, a.UUID, b.UUID,
		func(_ context.Context, _ ...string) ([]byte, error) { rejectCalls++; return nil, nil }); err == nil || rejectCalls != 0 {
		t.Fatal("native MAC drift reached root conntrack")
	}
	t.Log("S4_B2_NATIVE_ROOT_FOREIGN_MAC_CONNTRACK_REFUSED=PASS")
	acl, _, err := root.GetNetworkACL(scope.ACL)
	if err != nil {
		t.Fatal(err)
	}
	if len(acl.Ingress) != 4 && len(acl.Ingress) != 0 {
		t.Logf("native ACL ingress size %d (additional owned entries)", len(acl.Ingress))
	}
	// The root helper is read-only on Incus; it never mutates ACL, Project
	// or Profile and never implies readiness for a network Peer Tool.
	if err := nativePeerACLDropCovered(acl, a, b); err != nil {
		t.Fatal(err)
	}
	if _, _, err := root.GetProject("bwowned"); err != nil {
		// The Project may need a project-scoped reader, but no state was touched.
		t.Logf("root project direct read unavailable: %v", err)
	}
	_ = api.NetworkACLRule{}
}
