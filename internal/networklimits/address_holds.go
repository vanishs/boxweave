package networklimits

import "net/netip"

// AddressHolds is an internal read-only allocation snapshot. The SQLite
// reservation transaction remains the authoritative collision check.
type AddressHolds struct {
	IPs  map[netip.Addr]bool
	MACs map[string]bool
}

func NewAddressHolds() AddressHolds {
	return AddressHolds{IPs: map[netip.Addr]bool{}, MACs: map[string]bool{}}
}
