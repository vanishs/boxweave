package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/native"
)

func pr178PortFixture(t *testing.T) (string, []native.PeerACLBox, map[string]NativePeerGuardObservedOwner, string) {
	t.Helper()
	claims, _, observed, _ := pr178ScopedInventoryFixture()
	root := t.TempDir()
	bridge := "incusbr-1001"
	dir := filepath.Join(root, bridge, "brif")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for i, b := range claims {
		ob := observed[b.Name]
		ob.Instance.Config["volatile.eth0.host_name"] = "veth0000000" + string(rune('a'+i))
		if err := os.Symlink("../../../fake", filepath.Join(dir, ob.Instance.Config["volatile.eth0.host_name"])); err != nil {
			t.Fatal(err)
		}
		observed[b.Name] = ob
	}
	return root, claims, observed, bridge
}
func TestPR178ExactKernelPortsOnlyOwnedRunningBoxVeths(t *testing.T) {
	root, claims, obs, bridge := pr178PortFixture(t)
	if err := nativePeerGuardKernelPortsExactAt(context.Background(), root, bridge, claims, obs); err != nil {
		t.Fatal(err)
	}
	b := obs["pr178-b"]
	b.Instance.Status = "Stopped"
	obs["pr178-b"] = b
	if err := os.Remove(filepath.Join(root, bridge, "brif", b.Instance.Config["volatile.eth0.host_name"])); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerGuardKernelPortsExactAt(context.Background(), root, bridge, claims, obs); err != nil {
		t.Fatal("stopped Box must not be assigned a live bridge port:", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := nativePeerGuardKernelPortsExactAt(ctx, root, bridge, claims, obs); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled original must not pass native port audit", err)
	}
}
func TestPR178ExactKernelPortsRejectsForeignRawAndDrift(t *testing.T) {
	cases := []struct {
		name   string
		change func(*testing.T, string, []native.PeerACLBox, map[string]NativePeerGuardObservedOwner, string)
	}{
		{"raw-foreign-veth", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			if err := os.Symlink("../../../fake", filepath.Join(r, n, "brif", "vethforeign1")); err != nil {
				t.Fatal(err)
			}
		}},
		{"non-veth-foreign-port", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			if err := os.Symlink("../../../fake", filepath.Join(r, n, "brif", "wlan0")); err != nil {
				t.Fatal(err)
			}
		}},
		{"missing-live-veth", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			if err := os.Remove(filepath.Join(r, n, "brif", o["pr178-a"].Instance.Config["volatile.eth0.host_name"])); err != nil {
				t.Fatal(err)
			}
		}},
		{"host-name-swapped", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			a := o["pr178-a"]
			a.Instance.Config["volatile.eth0.host_name"] = "vethforged01"
			o["pr178-a"] = a
		}},
		{"same-host-name-duplicate", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			a := o["pr178-a"]
			v := o["pr178-b"]
			v.Instance.Config["volatile.eth0.host_name"] = a.Instance.Config["volatile.eth0.host_name"]
			o["pr178-b"] = v
		}},
		{"bad-volatile-uuid", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			a := o["pr178-a"]
			a.Instance.Config["volatile.uuid"] = pr178C
			o["pr178-a"] = a
		}},
		{"stopped-but-port-present", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			a := o["pr178-a"]
			a.Instance.Status = "Stopped"
			o["pr178-a"] = a
		}},
		{"transient-status", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			a := o["pr178-a"]
			a.Instance.Status = "Error"
			o["pr178-a"] = a
		}},
		{"duplicate-claim", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			b[1] = b[0]
		}},
		{"unknown-instance", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			delete(o, "pr178-a")
		}},
		{"bad-name-path", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			b[0].Name = "../../etc"
		}},
		{"not-symlink", func(t *testing.T, r string, b []native.PeerACLBox, o map[string]NativePeerGuardObservedOwner, n string) {
			p := filepath.Join(r, n, "brif", o["pr178-a"].Instance.Config["volatile.eth0.host_name"])
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte("fake"), 0600); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, claims, obs, bridge := pr178PortFixture(t)
			tc.change(t, root, claims, obs, bridge)
			if err := nativePeerGuardKernelPortsExactAt(context.Background(), root, bridge, claims, obs); err == nil {
				t.Fatalf("unsafe kernel bridge port drift accepted: %s", tc.name)
			}
		})
	}
	root, claims, obs, _ := pr178PortFixture(t)
	for _, bridge := range []string{"../../etc", "bad/bridge", ""} {
		if err := nativePeerGuardKernelPortsExactAt(context.Background(), root, bridge, claims, obs); err == nil {
			t.Fatal("untrusted bridge path accepted:", bridge)
		}
	}
	if err := nativePeerGuardKernelPortsExactAt(context.Background(), filepath.Join(root, "missing"), "incusbr-1001", claims, obs); err == nil {
		t.Fatal("missing kernel sysfs refused no readback")
	}
	if !strings.HasPrefix(obs["pr178-a"].Instance.Config["volatile.eth0.host_name"], "veth") {
		t.Fatal("bad native veth fixture")
	}
}
