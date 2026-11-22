package native

import (
	"errors"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func TestProjectCapabilityReportAndRefusals(t *testing.T) {
	c := api.ConfigMap{"restricted": "true", "restricted.devices.proxy": "block", "restricted.devices.disk": "managed", "user.secret": "not-reported"}
	p := ProjectPermissions(c)
	if len(p) != 8 || p["restricted.devices.proxy"] != "block" || p["restricted.devices.disk"] != "managed" || p["restricted.containers.interception"] != "unset" || p["user.secret"] != "" {
		t.Fatalf("unexpected closed permission report: %v", p)
	}
	for _, val := range []string{"", "block", "other"} {
		c["restricted.devices.proxy"] = val
		if err := requireProjectProxy(c); !errors.Is(err, ErrNativeUnsupported) || CapabilityDenialClass(err) != "PROJECT_PROXY_DENIED" {
			t.Fatalf("proxy refusal: %v", err)
		}
	}
	c["restricted.devices.proxy"] = "allow"
	if err := requireProjectProxy(c); err != nil {
		t.Fatal(err)
	}
	c["restricted"] = "false"
	if requireProjectProxy(c) == nil {
		t.Fatal("unrestricted Project accepted")
	}
	lowlevel := "generated"
	yes := true
	cases := []struct {
		opts  InstanceOptions
		class string
	}{
		{InstanceOptions{AppArmorProfile: &lowlevel}, "PROJECT_LOWLEVEL_DENIED"},
		{InstanceOptions{InterceptMknod: &yes}, "PROJECT_INTERCEPTION_DENIED"},
		{InstanceOptions{IDMapIsolated: &yes}, "IDMAP_RESERVATION_UNAVAILABLE"},
		{InstanceOptions{Privileged: &yes}, "PRIVILEGED_BOX_DENIED"},
	}
	for _, tc := range cases {
		err := tc.opts.Apply(api.ConfigMap{}, c)
		if !errors.Is(err, ErrNativeUnsupported) || CapabilityDenialClass(err) != tc.class {
			t.Fatalf("%s: %v", tc.class, err)
		}
	}
}

func TestForwardPolicyModeAndDenial(t *testing.T) {
	for raw, want := range map[string]string{"": "disabled", "all": "all", "eth0/tcp/192.0.2.1:8080": "exact", "INVALID": "invalid"} {
		if got := (Reader{ExternalForwardAllow: raw}).ExternalForwardPolicyMode(); got != want {
			t.Fatalf("mode %s != %s", got, want)
		}
	}
	f := ForwardSpec{Device: "bwpf-0123456789abcdef", Protocol: "tcp", HostBind: "0.0.0.0", HostPort: 8080, GuestPort: 80, ExternalAcknowledged: true}
	if err := (Reader{}).AuthorizeForwardCreate(f); CapabilityDenialClass(err) != "FORWARD_EXTERNAL_POLICY_DENIED" || !errors.Is(err, ErrNativeUnsupported) {
		t.Fatal(err)
	}
	if err := (Reader{ExternalForwardAllow: "all"}).AuthorizeForwardCreate(f); err != nil {
		t.Fatal(err)
	}
	f.ExternalAcknowledged = false
	if (Reader{ExternalForwardAllow: "all"}).AuthorizeForwardCreate(f) == nil {
		t.Fatal("acknowledgment lost")
	}
}
