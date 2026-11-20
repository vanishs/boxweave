package native

import (
	"net"
	"strconv"
	"strings"
)

// ParseExternalForwardAllow validates operator-managed startup policy.
// Entries: interface/protocol/IP:port; IPv6 addresses use brackets.
func ParseExternalForwardAllow(raw string) (map[string]string, error) {
	rules := map[string]string{}
	if raw == "" {
		return rules, nil
	}
	if len(raw) > 8192 {
		return nil, ErrNativeUnsupported
	}
	for _, entry := range strings.Split(raw, ",") {
		parts := strings.Split(entry, "/")
		if len(parts) != 3 || parts[0] == "" || len(parts[0]) > 15 || strings.ContainsAny(parts[0], " *?[]\t\r\n") {
			return nil, ErrNativeUnsupported
		}
		host, port, err := net.SplitHostPort(parts[2])
		if err != nil {
			return nil, ErrNativeUnsupported
		}
		p, err := strconv.Atoi(port)
		if err != nil || strconv.Itoa(p) != port {
			return nil, ErrNativeUnsupported
		}
		f := ForwardSpec{Device: "bwpf-0123456789abcdef", Protocol: parts[1], HostBind: host, HostPort: p, GuestPort: 1, ExternalAcknowledged: true}
		if f.Validate() != nil || net.ParseIP(host).IsLoopback() || parts[2] != net.JoinHostPort(host, port) {
			return nil, ErrNativeUnsupported
		}
		key := parts[1] + "/" + parts[2]
		if _, exists := rules[key]; exists {
			return nil, ErrNativeUnsupported
		}
		rules[key] = parts[0]
	}
	return rules, nil
}

// AuthorizeForwardCreate checks trusted startup policy and the current unique
// local interface address. This is NOT an ingress ACL or public reachability proof.
// Existing exact-owned forwards remain observable/deletable after policy removal.
func (r Reader) AuthorizeForwardCreate(f ForwardSpec) error {
	if err := f.Validate(); err != nil {
		return err
	}
	ip := net.ParseIP(f.HostBind)
	if ip.IsLoopback() {
		return nil
	}
	rules, err := ParseExternalForwardAllow(r.ExternalForwardAllow)
	if err != nil {
		return err
	}
	iface, allowed := rules[f.Protocol+"/"+net.JoinHostPort(f.HostBind, strconv.Itoa(f.HostPort))]
	if !allowed {
		return ErrNativeUnsupported
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return ErrNativeUnsupported
	}
	matches := 0
	for _, dev := range interfaces {
		addresses, err := dev.Addrs()
		if err != nil {
			return ErrNativeUnsupported
		}
		for _, a := range addresses {
			address, _, err := net.ParseCIDR(a.String())
			if err == nil && address.Equal(ip) {
				if dev.Name != iface || dev.Flags&net.FlagUp == 0 || dev.Flags&net.FlagLoopback != 0 {
					return ErrNativeUnsupported
				}
				matches++
			}
		}
	}
	if matches != 1 {
		return ErrNativeUnsupported
	}
	return nil
}
