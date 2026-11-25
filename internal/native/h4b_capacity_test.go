package native

import (
	"context"
	"github.com/vanishs/boxweave/internal/networklimits"
	"net/netip"
	"testing"
)

func TestH4BAddressCapacityUnionAndRetention(t *testing.T) {
	p := netip.MustParsePrefix("10.64.54.1/24")
	h := networklimits.NewAddressHolds()
	for _, raw := range []string{"10.64.54.1", "10.64.54.2", "10.64.54.3", "10.64.54.255", "10.64.55.5", "fd00::2"} {
		h.IPs[netip.MustParseAddr(raw)] = true
	}
	i := createPeerInventory{p4: p, holds: h}
	d := networklimits.NewAddressHolds()
	// Native overlap, retired-only hold, out-of-prefix and reserved endpoints.
	for _, raw := range []string{"10.64.54.2", "10.64.54.4", "10.64.54.0", "10.64.54.1", "10.64.55.4", "fd00::3"} {
		d.IPs[netip.MustParseAddr(raw)] = true
	}
	if e := i.mergeHolds(context.Background(), "user-1001", func(context.Context, string, string) (networklimits.AddressHolds, error) { return d, nil }); e != nil {
		t.Fatal(e)
	}
	c := i.capacity()
	if c.IPv4Prefix != p.String() || c.NominalIPv4 != 253 || c.BlockedIPv4 != 3 || c.RemainingIPv4UpperBound != 250 || *c.DurableHeldIPv4 != 2 || *c.NativeOnlyBlockedIPv4 != 1 || c.Level != "AVAILABLE" {
		t.Fatal(c)
	}
	// /30 only has one usable address, which remains unavailable after deletion.
	i = createPeerInventory{p4: netip.MustParsePrefix("10.1.2.1/30"), holds: networklimits.NewAddressHolds()}
	d = networklimits.NewAddressHolds()
	d.IPs[netip.MustParseAddr("10.1.2.2")] = true
	if e := i.mergeHolds(context.Background(), "user-1001", func(context.Context, string, string) (networklimits.AddressHolds, error) { return d, nil }); e != nil {
		t.Fatal(e)
	}
	c = i.capacity()
	if c.BlockedIPv4 != 1 || c.RemainingIPv4UpperBound != 0 || c.Level != "EXHAUSTED" || *c.NativeOnlyBlockedIPv4 != 0 {
		t.Fatal(c)
	}
}
