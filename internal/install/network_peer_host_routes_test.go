package install

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/native"
)

func s4b2HostRoutesFixture() (NativePeerBridgeAuditTarget, []nativePeerHostRoute, []nativePeerHostRoute) {
	scope := NativePeerBridgeAuditTarget{Project: "user-1001", Bridge: "bwbr0", ACL: "bw-s4b2",
		Boxes: []native.PeerACLBox{
			{UUID: "a", IPv4: "10.243.18.11", IPv6: "fd43:18::11"},
			{UUID: "b", IPv4: "10.243.18.12", IPv6: "fd43:18::12"},
		}}
	v4 := []nativePeerHostRoute{
		{Dst: "default", Dev: "eth0"},
		{Dst: "10.243.18.0/24", Dev: "bwbr0"},
		{Dst: "172.18.0.0/16", Dev: "docker0"},
	}
	v6 := []nativePeerHostRoute{
		{Dst: "default", Dev: "eth0"},
		{Dst: "fd43:18::/64", Dev: "bwbr0"},
		{Dst: "fe80::/64", Dev: "eth0"},
	}
	return scope, v4, v6
}

func TestS4B2RootHostRoutesPeerScopeRefusesUnmanagedOverlap(t *testing.T) {
	scope, v4, v6 := s4b2HostRoutesFixture()
	hash, err := nativePeerHostRouteScopeFromRows(scope, v4, v6)
	if err != nil || len(hash) != 64 {
		t.Fatalf("exclusive native route scope refused: %q %v", hash, err)
	}
	reverse := slices.Clone(v4)
	slices.Reverse(reverse)
	reversed, err := nativePeerHostRouteScopeFromRows(scope, reverse, v6)
	if err != nil || reversed != hash {
		t.Fatalf("route scope digest depends on JSON order: %q %v", reversed, err)
	}
	cases := []struct {
		name string
		edit func([]nativePeerHostRoute, []nativePeerHostRoute) ([]nativePeerHostRoute, []nativePeerHostRoute)
	}{
		{"foreign-unmanaged-same-ipv4-range", func(a, b []nativePeerHostRoute) ([]nativePeerHostRoute, []nativePeerHostRoute) {
			return append(a, nativePeerHostRoute{Dst: "10.243.18.0/24", Dev: "rawbr0"}), b
		}},
		{"foreign-unmanaged-same-ipv6-range", func(a, b []nativePeerHostRoute) ([]nativePeerHostRoute, []nativePeerHostRoute) {
			return a, append(b, nativePeerHostRoute{Dst: "fd43:18::/64", Dev: "rawbr0"})
		}},
		{"foreign-host-route-to-specific-peer", func(a, b []nativePeerHostRoute) ([]nativePeerHostRoute, []nativePeerHostRoute) {
			return append(a, nativePeerHostRoute{Dst: "10.243.18.11/32", Dev: "eth0"}), b
		}},
		{"foreign-blackhole-route", func(a, b []nativePeerHostRoute) ([]nativePeerHostRoute, []nativePeerHostRoute) {
			return append(a, nativePeerHostRoute{Dst: "10.243.18.12/32", Type: "blackhole"}), b
		}},
		{"owner-route-removed", func(a, b []nativePeerHostRoute) ([]nativePeerHostRoute, []nativePeerHostRoute) {
			return a[:1], b
		}},
		{"owner-route-duplicated", func(a, b []nativePeerHostRoute) ([]nativePeerHostRoute, []nativePeerHostRoute) {
			return append(a, a[1]), b
		}},
		{"malformed-route", func(a, b []nativePeerHostRoute) ([]nativePeerHostRoute, []nativePeerHostRoute) {
			return append(a, nativePeerHostRoute{Dst: "not-an-IP", Dev: "eth0"}), b
		}},
		{"empty-v6-readback", func(a, b []nativePeerHostRoute) ([]nativePeerHostRoute, []nativePeerHostRoute) {
			return a, nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, a, b := s4b2HostRoutesFixture()
			a, b = tc.edit(a, b)
			if hash, err := nativePeerHostRouteScopeFromRows(scope, a, b); err == nil || hash != "" {
				t.Fatalf("unsafe native route overlap authorized: hash=%s err=%v", hash, err)
			}
		})
	}
	t.Log("S4_B2_ROOT_UNMANAGED_KERNEL_ROUTE_OVERLAP_REFUSED=PASS")
}

func TestS4B2RootHostRoutesMustUseBoundedFixedIPReader(t *testing.T) {
	scope, _, _ := s4b2HostRoutesFixture()
	count := 0
	_, err := nativePeerHostRoutesWithRun(t.Context(), scope,
		func(_ context.Context, args ...string) ([]byte, error) {
			count++
			expected := []string{"-j", "-4", "route", "show", "table", "all"}
			if count == 2 {
				expected[1] = "-6"
			}
			if !slices.Equal(args, expected) {
				t.Fatalf("unbounded root IP argv: %q", args)
			}
			if count == 1 {
				return []byte(`[{"dst":"default","dev":"eth0"},{"dst":"10.243.18.0/24","dev":"bwbr0"}]`), nil
			}
			return []byte(`[{"dst":"default","dev":"eth0"},{"dst":"fd43:18::/64","dev":"bwbr0"}]`), nil
		})
	if err != nil || count != 2 {
		t.Fatalf("fixed root route reader failed: %v count=%d", err, count)
	}
	_, err = nativePeerHostRoutesWithRun(t.Context(), scope,
		func(context.Context, ...string) ([]byte, error) {
			return nil, errors.New("kernel route inventory denied")
		})
	if err == nil {
		t.Fatal("missing kernel root routes was treated as empty")
	}
	if strings.Contains(err.Error(), "not found") {
		t.Fatal("unexpected nondeterministic shell path")
	}
	t.Log("S4_B2_ROOT_FIXED_KERNEL_ROUTE_READER=PASS")
}
