package native

import (
	"github.com/lxc/incus/v7/shared/api"
	"testing"
)

func TestB4ForwardSpecAndExactProxyDevice(t *testing.T) {
	cases := []struct {
		bind     string
		protocol string
		hp, gp   int
		ack      bool
		valid    bool
	}{
		{"127.0.0.1", "tcp", 22001, 80, false, true},
		{"::1", "udp", 22002, 53, false, true},
		{"192.0.2.8", "tcp", 80, 8080, true, true},
		{"192.0.2.8", "tcp", 80, 8080, false, false},
		{"0.0.0.0", "tcp", 80, 80, true, true},
		{"::", "udp", 53, 53, true, true},
		{"0.0.0.0", "tcp", 80, 80, false, false},
		{"::", "udp", 53, 53, false, false},
		{"127.0.0.9", "tcp", 80, 80, false, false},
		{"127.0.0.1", "sctp", 80, 80, false, false},
		{"127.0.0.1", "tcp", 0, 80, false, false},
		{"127.0.0.1", "tcp", 80, 65536, false, false},
	}
	for _, tt := range cases {
		f := ForwardSpec{Device: "bwpf-0123456789abcdef", Protocol: tt.protocol, HostBind: tt.bind, HostPort: tt.hp, GuestPort: tt.gp, ExternalAcknowledged: tt.ack}
		if (f.Validate() == nil) != tt.valid {
			t.Fatalf("validation %+v", tt)
		}
		if !tt.valid {
			continue
		}
		original := api.DevicesMap{"root": {"type": "disk", "path": "/", "pool": "test"}, "eth0": {"type": "nic", "network": "bwbr0"}}
		added, err := forwardDeviceChange(original, f, true)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := original[f.Device]; ok {
			t.Fatal("mutated original devices")
		}
		if !exactForwardDevice(added[f.Device], f) {
			t.Fatalf("incorrect owned proxy %+v", added[f.Device])
		}
		if _, err := forwardDeviceChange(added, f, true); err != ErrNativeDependents {
			t.Fatalf("overwrote existing device: %v", err)
		}
		tampered := copyInstanceDevices(added)
		tampered[f.Device]["listen"] = "tcp:127.0.0.1:65535"
		if _, err := forwardDeviceChange(tampered, f, false); err != ErrNativeDependents {
			t.Fatalf("deleted foreign device: %v", err)
		}
		removed, err := forwardDeviceChange(added, f, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := removed[f.Device]; exists {
			t.Fatal("forward not removed")
		}
		if len(removed) != len(original) {
			t.Fatalf("unrelated devices removed %+v", removed)
		}
	}
}

func TestB4HostLocalForwardPolicy(t *testing.T) {
	f := ForwardSpec{Device: "bwpf-0123456789abcdef", Protocol: "tcp", HostBind: "192.0.2.8", HostPort: 23099, GuestPort: 8080, ExternalAcknowledged: true}
	if (Reader{}).AuthorizeForwardCreate(f) == nil {
		t.Fatal("caller acknowledgment bypassed host policy")
	}
	for _, bad := range []string{"eth0/tcp/0.0.0.0:80", "eth0/tcp/192.0.2.8:80-90", "eth0/tcp/192.0.2.8:080", "*/tcp/192.0.2.8:80", "eth0/udp/[::]:53", "eth0/tcp/127.0.0.1:80", "eth0/tcp/192.0.2.8:80,eth1/tcp/192.0.2.8:80"} {
		if _, err := ParseExternalForwardAllow(bad); err == nil {
			t.Fatalf("unsafe policy %q accepted", bad)
		}
	}
	if p, err := ParseExternalForwardAllow("eth0/tcp/192.0.2.8:23099,eth1/udp/[2001:db8::1]:53"); err != nil || len(p) != 2 {
		t.Fatalf("valid exact policy: %v %v", p, err)
	}
	if (Reader{ExternalForwardAllow: "notreal/tcp/192.0.2.8:23099"}).AuthorizeForwardCreate(f) == nil {
		t.Fatal("unassigned address accepted")
	}
	f.HostBind = "127.0.0.1"
	f.ExternalAcknowledged = false
	if (Reader{}).AuthorizeForwardCreate(f) != nil {
		t.Fatal("loopback default denied")
	}
}

func TestForwardOperatorAllPolicy(t *testing.T) {
	all := Reader{ExternalForwardAllow: ExternalForwardAll}
	if p, err := ParseExternalForwardAllow("all"); err != nil || len(p) != 1 {
		t.Fatalf("operator all policy rejected: %v %v", p, err)
	}
	for _, bad := range []string{"ALL", " all", "all,", "all,eth0/tcp/192.0.2.8:80", "*"} {
		if _, err := ParseExternalForwardAllow(bad); err == nil {
			t.Fatalf("ambiguous broad policy accepted: %q", bad)
		}
	}
	for _, bind := range []string{"0.0.0.0", "::"} {
		for _, proto := range []string{"tcp", "udp"} {
			f := ForwardSpec{Device: "bwpf-0123456789abcdef", Protocol: proto, HostBind: bind, HostPort: 23099, GuestPort: 8080, ExternalAcknowledged: true}
			if err := all.AuthorizeForwardCreate(f); err != nil {
				t.Fatalf("explicit operator policy refused %s %s: %v", proto, bind, err)
			}
			if (Reader{}).AuthorizeForwardCreate(f) == nil || (Reader{ExternalForwardAllow: "eth0/tcp/192.0.2.8:23099"}).AuthorizeForwardCreate(f) == nil {
				t.Fatal("wildcard bypassed empty or exact policy")
			}
			f.ExternalAcknowledged = false
			if all.AuthorizeForwardCreate(f) == nil {
				t.Fatal("wildcard bypassed explicit exposure acknowledgement")
			}
		}
	}
	for _, bind := range []string{"192.0.2.8", "224.0.0.1", "169.254.0.1", "127.0.0.9", "::ffff:127.0.0.1"} {
		f := ForwardSpec{Device: "bwpf-0123456789abcdef", Protocol: "tcp", HostBind: bind, HostPort: 23099, GuestPort: 8080, ExternalAcknowledged: true}
		if all.AuthorizeForwardCreate(f) == nil {
			t.Fatalf("foreign, special or noncanonical address accepted: %s", bind)
		}
	}
}
