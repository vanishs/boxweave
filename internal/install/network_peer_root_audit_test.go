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

// Real official Incus root-side bridge inventory. Only runs on a fresh
// disposable CI daemon, NEVER an L0 shared host or production controller.
func TestS4B2NativeRootBridgeAudit(t *testing.T) {
	if os.Getenv("BOXWEAVE_S4B2_DISPOSABLE_ROOT_AUDIT") != "20261117-read-only" {
		t.Skip("root-only disposable native test")
	}
	if os.Geteuid() != 0 {
		t.Fatal("host-root test required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	own := root.UseProject("bwowned")
	var t0 NativePeerBridgeAuditTarget
	t0.Project = "bwowned"
	t0.Bridge = "bwbr0"
	t0.ACL = "bw-s4b2-native"
	for _, name := range []string{"s4b2a", "s4b2b", "s4b2c"} {
		inst, _, err := own.GetInstance(name)
		if err != nil {
			t.Fatal(err)
		}
		mac := inst.Config["volatile.eth0.hwaddr"]
		if mac == "" {
			mac = inst.ExpandedDevices["eth0"]["hwaddr"]
		}
		t0.Boxes = append(t0.Boxes, native.PeerACLBox{
			Name: name, UUID: inst.Config["volatile.uuid"], MAC: mac,
			IPv4: inst.Devices["eth0"]["ipv4.address"],
			IPv6: inst.Devices["eth0"]["ipv6.address"],
		})
	}
	snap, err := nativePeerBridgeAuditWithClient(ctx, root, t0)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.NativeDigestSHA256) != 64 || snap.OwnedBoxes != 3 {
		t.Fatalf("bad exact root audit: %+v", snap)
	}
	t.Logf("S4_B2_ROOT_NATIVE_BRIDGE_AUDIT_READONLY_PASS digest=%s boxes=%d",
		snap.NativeDigestSHA256, snap.OwnedBoxes)
	bad := t0
	bad.Boxes = append([]native.PeerACLBox(nil), t0.Boxes...)
	bad.Boxes[1].MAC = "00:16:3e:fe:fe:fe"
	if _, err := nativePeerBridgeAuditWithClient(ctx, root, bad); err == nil {
		t.Fatal("foreign MAC owner spoof accepted")
	}
	t.Log("S4_B2_ROOT_NATIVE_MAC_DRIFT_REFUSED=PASS")
	bad = t0
	bad.Boxes = t0.Boxes[:2]
	if _, err := nativePeerBridgeAuditWithClient(ctx, root, bad); err == nil {
		t.Fatal("missing third native NIC accepted")
	}
	t.Log("S4_B2_ROOT_NATIVE_INCOMPLETE_SIBLINGS_REFUSED=PASS")
	bad = t0
	bad.Bridge = "untrusted-br"
	if _, err := nativePeerBridgeAuditWithClient(ctx, root, bad); err == nil {
		t.Fatal("unknown bridge accepted")
	}
	t.Log("S4_B2_ROOT_NATIVE_FOREIGN_BRIDGE_REFUSED=PASS")
}
func TestS4B2RootBridgeAuditRejectsUntrustedScope(t *testing.T) {
	bad := NativePeerBridgeAuditTarget{Project: "default", Bridge: "bwbr0", ACL: "bw-s4b2"}
	if _, err := nativePeerBridgeAuditWithClient(context.Background(), nil, bad); err == nil {
		t.Fatal("default Project admitted")
	}
	bad.Project = "bwowned"
	bad.ACL = strings.Repeat("x", 200)
	if _, err := nativePeerBridgeAuditWithClient(context.Background(), nil, bad); err == nil {
		t.Fatal("unbounded ACL scope admitted")
	}
}
