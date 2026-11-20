package native

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

type fakePeerACLDriver struct {
	instances            map[string]*api.Instance
	events               []string
	failUpdate, failStop bool
}

func (f *fakePeerACLDriver) names() ([]string, error) {
	out := make([]string, 0, len(f.instances))
	for name := range f.instances {
		out = append(out, name)
	}
	return out, nil
}
func (f *fakePeerACLDriver) get(n string) (*api.Instance, string, error) {
	inst, ok := f.instances[n]
	if !ok {
		return nil, "", ErrNotOwned
	}
	out := *inst
	return &out, "fake-etag-" + n, nil
}
func (f *fakePeerACLDriver) disableAutostart(n string, put api.InstancePut, tag string) (string, error) {
	if tag != "fake-etag-"+n {
		return "", ErrUnverified
	}
	f.events = append(f.events, "disable:"+n)
	f.instances[n].Config = put.Config
	if f.failUpdate {
		return "", errors.New("lost update acknowledgement")
	}
	return "disable-" + n, nil
}
func (f *fakePeerACLDriver) stop(n string) (string, error) {
	f.events = append(f.events, "stop:"+n)
	f.instances[n].Status = "Stopped"
	if f.failStop {
		return "", errors.New("lost stop acknowledgement")
	}
	return "stop-" + n, nil
}
func (f *fakePeerACLDriver) terminal(id string) (bool, error) { return id != "", nil }

func peerACLFixture() (*fakePeerACLDriver, []PeerACLBox) {
	f := &fakePeerACLDriver{instances: map[string]*api.Instance{}}
	var boxes []PeerACLBox
	for i, name := range []string{"s4b2-a", "s4b2-b", "s4b2-c"} {
		mac := fmt.Sprintf("00:16:3e:51:42:%02x", i+1)
		v4 := fmt.Sprintf("10.42.83.%d", 11+i)
		v6 := fmt.Sprintf("fd83:42::%d", 11+i)
		b := PeerACLBox{Name: name, UUID: "test-uuid-" + name, MAC: mac, IPv4: v4, IPv6: v6}
		nic := map[string]string{
			"type": "nic", "name": "eth0", "network": "bwbr0", "security.acls": "bw-s4b2-native",
			"security.acls.default.ingress.action": "reject", "security.acls.default.egress.action": "reject",
			"security.ipv4_filtering": "true", "security.ipv6_filtering": "true", "security.mac_filtering": "true",
			"ipv4.address": v4, "ipv6.address": v6,
		}
		f.instances[name] = &api.Instance{
			Name: name, Project: "user-1001", Status: "Running",
			InstancePut: api.InstancePut{
				Config:   api.ConfigMap{"volatile.uuid": b.UUID, "volatile.eth0.hwaddr": mac, "boot.autostart": "true"},
				Profiles: []string{"default"},
				Devices:  api.DevicesMap{"eth0": {"type": "nic", "ipv4.address": v4, "ipv6.address": v6}},
			},
			ExpandedDevices: api.DevicesMap{"eth0": nic},
		}
		boxes = append(boxes, b)
	}
	return f, boxes
}
func runFakePeerACL(f *fakePeerACLDriver, boxes []PeerACLBox, check PeerACLBridgeCheck) error {
	return isolatePeerACL(context.Background(), f, "user-1001", "bwbr0", "bw-s4b2-native", boxes, check)
}
func TestS4B2QuiescentSharedACLStopsEveryNICBeforePUT(t *testing.T) {
	f, boxes := peerACLFixture()
	checkCount := 0
	err := runFakePeerACL(f, boxes, func(context.Context) error { checkCount++; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if checkCount != 2 {
		t.Fatalf("root bridge recheck count=%d", checkCount)
	}
	expected := []string{"disable:s4b2-a", "disable:s4b2-b", "disable:s4b2-c", "stop:s4b2-a", "stop:s4b2-b", "stop:s4b2-c"}
	if !slices.Equal(f.events, expected) {
		t.Fatalf("unexpected native write order: %v", f.events)
	}
	for _, inst := range f.instances {
		if inst.Config["boot.autostart"] != "false" || inst.Status != "Stopped" {
			t.Fatalf("native isolation not durable: %+v", inst)
		}
	}
}
func TestS4B2QuiescentSharedACLRejectsUnsafeBindingsBeforeSideEffect(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*fakePeerACLDriver, []PeerACLBox) []PeerACLBox
	}{
		{"missing-sibling", func(f *fakePeerACLDriver, b []PeerACLBox) []PeerACLBox { return b[:2] }},
		{"wrong-mac", func(f *fakePeerACLDriver, b []PeerACLBox) []PeerACLBox { b[1].MAC = "00:16:3e:51:42:fe"; return b }},
		{"duplicate-ipv4", func(f *fakePeerACLDriver, b []PeerACLBox) []PeerACLBox { b[1].IPv4 = b[0].IPv4; return b }},
		{"static-lease-missing", func(f *fakePeerACLDriver, b []PeerACLBox) []PeerACLBox {
			delete(f.instances[b[1].Name].Devices["eth0"], "ipv6.address")
			return b
		}},
		{"ipv6-anti-spoof-disabled", func(f *fakePeerACLDriver, b []PeerACLBox) []PeerACLBox {
			f.instances[b[0].Name].ExpandedDevices["eth0"]["security.ipv6_filtering"] = "false"
			return b
		}},
		{"foreign-network", func(f *fakePeerACLDriver, b []PeerACLBox) []PeerACLBox {
			f.instances[b[0].Name].ExpandedDevices["eth0"]["network"] = "foreign-bridge"
			return b
		}},
		{"missing-native-uuid", func(f *fakePeerACLDriver, b []PeerACLBox) []PeerACLBox {
			delete(f.instances[b[0].Name].Config, "volatile.uuid")
			return b
		}},
		{"unverified-native-mac", func(f *fakePeerACLDriver, b []PeerACLBox) []PeerACLBox {
			delete(f.instances[b[0].Name].Config, "volatile.eth0.hwaddr")
			return b
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, boxes := peerACLFixture()
			boxes = tc.mutate(f, boxes)
			err := runFakePeerACL(f, boxes, func(context.Context) error { return nil })
			if err == nil || len(f.events) != 0 {
				t.Fatalf("preflight bypass err=%v events=%v", err, f.events)
			}
		})
	}
}
func TestS4B2QuiescentSharedACLRootCheckMandatory(t *testing.T) {
	f, boxes := peerACLFixture()
	if err := runFakePeerACL(f, boxes, nil); err == nil || len(f.events) != 0 {
		t.Fatal(err, f.events)
	}
	if err := runFakePeerACL(f, boxes, func(context.Context) error { return errors.New("foreign NIC used_by") }); err == nil || len(f.events) != 0 {
		t.Fatal(err, f.events)
	}
}
func TestS4B2QuiescentSharedACLLostACKNeverResumes(t *testing.T) {
	for _, fail := range []string{"autostart", "stop"} {
		t.Run(fail, func(t *testing.T) {
			f, boxes := peerACLFixture()
			f.failUpdate = fail == "autostart"
			f.failStop = fail == "stop"
			err := runFakePeerACL(f, boxes, func(context.Context) error { return nil })
			if err == nil {
				t.Fatal("lost ACK incorrectly succeeded")
			}
			if len(f.events) == 0 {
				t.Fatal("fault did not reach native mutation")
			}
			if strings.Contains(strings.Join(f.events, ","), "start:") {
				t.Fatal("unsafe restart")
			}
			if fail == "autostart" {
				for _, e := range f.events {
					if strings.HasPrefix(e, "stop:") {
						t.Fatalf("stop after failed autostart acknowledgement: %v", f.events)
					}
				}
			}
		})
	}
}
