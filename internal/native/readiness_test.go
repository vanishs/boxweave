package native

import (
	"github.com/lxc/incus/v7/shared/api"
	"testing"
	"time"
)

func TestS4CReadinessNativeAddressAndExitProof(t *testing.T) {
	full := &api.InstanceFull{Instance: api.Instance{
		InstancePut: api.InstancePut{Config: api.ConfigMap{"volatile.eth0.hwaddr": "00:16:3e:11:22:33"}},
		Status:      "Running", LastUsedAt: time.Now(),
		ExpandedDevices: api.DevicesMap{"eth0": {"type": "nic", "network": "bwbr0", "security.acls": "bw-private", "security.acls.default.ingress.action": "reject", "security.ipv4_filtering": "true", "security.ipv6_filtering": "true", "security.mac_filtering": "true", "ipv4.address": "10.0.0.9"}},
	}, State: &api.InstanceState{Status: "Running", Network: map[string]api.InstanceStateNetwork{"eth0": {State: "up", Hwaddr: "00:16:3e:11:22:33", Addresses: []api.InstanceStateNetworkAddress{
		{Family: "inet", Address: "10.0.0.9", Scope: "global"},
		{Family: "inet", Address: "10.0.0.8", Scope: "global"},
		{Family: "inet6", Address: "fd00::9", Scope: "global"},
		{Family: "inet6", Address: "fe80::9", Scope: "link"},
		{Family: "inet", Address: "127.0.0.1", Scope: "global"},
	}}}}}
	o, e := readinessObservation(full)
	if e != nil || !o.Running || !o.NetworkObserved || len(o.Addresses) != 2 {
		t.Fatalf("wrong owned address proof: %+v %v", o, e)
	}
	full.ExpandedDevices["eth0"]["security.ipv6_filtering"] = "false"
	if _, e = readinessObservation(full); e == nil {
		t.Fatal("unsafe NIC accepted")
	}
	for _, tc := range []struct {
		status api.StatusCode
		meta   map[string]any
		err    string
		ok     bool
		exit   int
	}{
		{api.Success, map[string]any{"return": float64(0)}, "", true, 0},
		{api.Success, map[string]any{"return": float64(8)}, "", true, 8},
		{api.Success, nil, "", false, 0},
		{api.Failure, map[string]any{"return": float64(0)}, "", false, 0},
		{api.Failure, map[string]any{"return": float64(127)}, "Command not found", true, 127},
		{api.Running, map[string]any{"return": float64(0)}, "", false, 0},
	} {
		exit, e := ReadinessProbeExit(&api.Operation{StatusCode: tc.status, Metadata: tc.meta, Err: tc.err})
		if (e == nil) != tc.ok || tc.ok && exit != tc.exit {
			t.Fatalf("bad terminal evidence: %d %v", exit, e)
		}
	}
}
