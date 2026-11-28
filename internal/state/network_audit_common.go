package state

import (
	"net"
	"net/netip"
	"strings"
)

// NetworkAuditBinding is a local claim, not native authentication.
type NetworkAuditBinding struct{ Name, UUID, Incarnation, MAC, IPv4, IPv6 string }

func auditName(s string) bool {
	if len(s) == 0 || len(s) > 100 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
func auditIP(s string, v6 bool) bool {
	ip, e := netip.ParseAddr(s)
	return e == nil && ip.IsGlobalUnicast() && !ip.Is4In6() && ip.Is6() == v6 && ip.String() == s
}
func auditMAC(s string) bool {
	m, e := net.ParseMAC(s)
	return e == nil && len(m) == 6 && m[0]&1 == 0 && m.String() == s && s == strings.ToLower(s)
}
