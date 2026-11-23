package install

import (
	"context"
	"errors"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

func TestRetirementConntrackRequiresExplicitZero(t *testing.T) {
	ctx := context.Background()
	for _, s := range []struct {
		output string
		err    error
		ok     bool
	}{
		{"conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.\n", nil, true},
		{"conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.\n", nil, false},
		{"tcp src=10.4.5.6 dst=10.4.5.7\nconntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.\n", nil, false},
		{"", nil, false},
		{"conntrack v1.4.8 (conntrack-tools): 0 flow entries have been shown.\n", errors.New("transport failed"), false},
	} {
		run := func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) != 5 || args[0] != "-L" || args[1] != "-f" || args[2] != "ipv4" || args[3] != "--reply-dst" || args[4] != "10.4.5.6" {
				t.Fatal(args)
			}
			return []byte(s.output), s.err
		}
		if e := retirementConntrackZero(ctx, run, "ipv4", "--reply-dst", "10.4.5.6"); (e == nil) != s.ok {
			t.Fatal(s, e)
		}
	}
}
func TestRetirementKernelExactReferences(t *testing.T) {
	b := state.NetworkAuditBinding{UUID: "old", MAC: "02:00:00:00:00:01", IPv4: "10.4.5.6", IPv6: "fd00::6"}
	for _, x := range []struct {
		raw, kind string
		ok        bool
	}{
		{`{"nftables":[{"table":{"family":"inet","name":"incus"}},{"rule":{"expr":[{"match":{"right":{"prefix":{"addr":"10.4.5.0","len":24}}}}]}}]}`, "nft", true},
		{`{"nftables":[{"table":{"family":"inet","name":"incus"}},{"rule":{"expr":[{"match":{"right":{"prefix":{"addr":"10.4.5.6","len":32}}}}]}}]}`, "nft", false},
		{`{"nftables":[{"table":{"family":"inet","name":"incus"}},{"rule":{"expr":[{"dnat":{"addr":"10.4.5.6"}}]}}]}`, "nft", false},
		{`{"nftables":[]}`, "nft", false},
		{`[{"dst":"fd00::6","lladdr":"01:02:03:04:05:06"}]`, "neighbors", false},
		{`[{"mac":"02:00:00:00:00:01","dev":"eth0"}]`, "fdb", false},
		{`[{"dst":"fd00::6","state":["FAILED"]}]`, "neighbors", true},
		{`[{"dst":"fd00::6","state":["FAILED"],"lladdr":"02:00:00:00:00:01"}]`, "neighbors", false},
		{`[{"dst":"fd00::6","state":["FAILED","STALE"]}]`, "neighbors", false},
		{`[{"dst":"fd00::6","state":["FAILED"],"router":null}]`, "neighbors", false},
		{`[{"dst":"10.4.5.6","state":["FAILED"]}]`, "neighbors", false},
		{`[{"dst":"fd00::6","state":["FAILED"]},{"dst":"10.4.5.6","lladdr":"01:02:03:04:05:06"}]`, "neighbors", false},
		{`[]`, "neighbors", true},
		{`{}`, "neighbors", false},
	} {
		if e := retirementKernelReferenceAbsent([]byte(x.raw), b, x.kind); (e == nil) != x.ok {
			t.Fatal(x, e)
		}
	}
	for _, x := range []api.Instance{
		{InstancePut: api.InstancePut{Config: api.ConfigMap{"volatile.uuid": "old"}}},
		{ExpandedDevices: api.DevicesMap{"eth0": {"type": "nic", "ipv4.address": b.IPv4}}},
		{ExpandedDevices: api.DevicesMap{"proxy": {"type": "proxy", "connect": "tcp:" + b.IPv4 + ":80"}}},
	} {
		if e := retirementNativeIdentityAbsent([]api.Instance{x}, b); e == nil {
			t.Fatal("stale native identity accepted")
		}
	}
}
