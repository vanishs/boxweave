package native

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

// Opt-in ONLY from the fresh disposable Incus 7.6 CI runner, with its own
// isolated project and address space. Never run against a shared L0 daemon.
func TestS4B2NativeSDKSharedACLQuiescence(t *testing.T) {
	if os.Getenv("BOXWEAVE_S4B2_DISPOSABLE_BARRIER") != "20261117-single-use" {
		t.Skip("requires disposable Incus native host")
	}
	if os.Geteuid() != 0 {
		t.Fatal("test-only root bridge inspection required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	host, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	project := host.UseProject("bwowned")
	names := []string{"s4b2a", "s4b2b", "s4b2c"}
	boxes := make([]PeerACLBox, 0, len(names))
	for _, name := range names {
		inst, _, err := project.GetInstance(name)
		if err != nil {
			t.Fatal(err)
		}
		mac := inst.Config["volatile.eth0.hwaddr"]
		if mac == "" {
			mac = inst.ExpandedDevices["eth0"]["hwaddr"]
		}
		box := PeerACLBox{
			Name: name, UUID: inst.Config["volatile.uuid"], MAC: mac,
			IPv4: inst.Devices["eth0"]["ipv4.address"], IPv6: inst.Devices["eth0"]["ipv6.address"],
		}
		t.Logf("NATIVE_PRE_BARRIER name=%s uuid=%s mac=%s v4=%s v6=%s status=%s",
			box.Name, box.UUID, box.MAC, box.IPv4, box.IPv6, inst.Status)
		boxes = append(boxes, box)
	}
	checkCount := 0
	check := func(ctx context.Context) error {
		checkCount++
		network, _, err := host.GetNetwork("bwbr0")
		if err != nil {
			return err
		}
		expect := []string{"/1.0/profiles/default?project=bwowned",
			"/1.0/instances/s4b2a?project=bwowned",
			"/1.0/instances/s4b2b?project=bwowned",
			"/1.0/instances/s4b2c?project=bwowned"}
		if len(network.UsedBy) != len(expect) {
			return fmt.Errorf("unexpected bridge used_by: %v", network.UsedBy)
		}
		for _, ref := range network.UsedBy {
			if !slices.Contains(expect, ref) {
				return fmt.Errorf("foreign bridge consumer: %s", ref)
			}
		}
		return nil
	}
	d := &tracingPeerACLSDK{peerACLSDK: peerACLSDK{c: project}}
	if err := isolatePeerACL(ctx, d, "bwowned", "bwbr0", "bw-s4b2-native", boxes, check); err != nil {
		t.Fatal(err)
	}
	if checkCount != 2 {
		t.Fatalf("root used_by check not repeated: %d", checkCount)
	}
	if len(d.nativeOperations) != 6 {
		t.Fatalf("missing native operation original IDs: %v", d.nativeOperations)
	}
	for _, name := range names {
		inst, _, err := project.GetInstance(name)
		if err != nil {
			t.Fatal(err)
		}
		if inst.Status != "Stopped" || inst.Config["boot.autostart"] != "false" {
			t.Fatalf("native isolation not retained for %s: status=%s autostart=%q", name, inst.Status, inst.Config["boot.autostart"])
		}
		t.Logf("NATIVE_POST_BARRIER name=%s uuid=%s status=%s autostart=%s", name, inst.Config["volatile.uuid"], inst.Status, inst.Config["boot.autostart"])
	}
	t.Logf("S4_B2_NATIVE_SDK_THREE_NIC_BARRIER=PASS native_operation_ids=%v", d.nativeOperations)
}

type tracingPeerACLSDK struct {
	peerACLSDK
	nativeOperations []string
}

func (t *tracingPeerACLSDK) disableAutostart(n string, p api.InstancePut, etag string) (string, error) {
	id, err := t.peerACLSDK.disableAutostart(n, p, etag)
	if id != "" {
		t.nativeOperations = append(t.nativeOperations, id)
	}
	return id, err
}
func (t *tracingPeerACLSDK) stop(n string) (string, error) {
	id, err := t.peerACLSDK.stop(n)
	if id != "" {
		t.nativeOperations = append(t.nativeOperations, id)
	}
	return id, err
}
