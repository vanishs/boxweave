package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"net/netip"
	"reflect"
	"regexp"
	"strings"
)

// Fixed installed ledger, shared by native lifecycle and retirement readers.
const nativePeerRootInstalledDB = "/var/lib/boxweave/ledger.sqlite"

var safePrereqName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)

type NativePrepareTarget struct {
	Project string `json:"project"`
	Pool    string `json:"pool"`
	Network string `json:"network"`
	ACL     string `json:"acl"`
}

func nativePrepareProfile(t NativePrepareTarget, original api.ProfilePut) api.ProfilePut {
	p := original
	p.Devices = api.DevicesMap{
		"root": {"type": "disk", "path": "/", "pool": t.Pool},
		"eth0": {"type": "nic", "name": "eth0", "network": t.Network, "security.acls": t.ACL,
			"security.acls.default.ingress.action": "reject",
			"security.ipv4_filtering":              "true", "security.ipv6_filtering": "true", "security.mac_filtering": "true"},
	}
	return p
}
func nativePrepareProject(original api.ProjectPut) api.ProjectPut {
	p := original
	p.Config = api.ConfigMap{}
	for k, v := range original.Config {
		p.Config[k] = v
	}
	p.Config["restricted.containers.nesting"] = "allow"
	// Only root-reviewed preparation on an empty owned Project grants
	// native proxy devices. No service install-stage or MCP side effect.
	p.Config["restricted.devices.proxy"] = "allow"
	// Incus multi-user bootstrapping can permit home-path mounts and GPUs.
	// This dedicated MCP service needs neither; tighten only its own Project.
	p.Config["restricted.devices.disk"] = "managed"
	p.Config["restricted.devices.gpu"] = "block"
	return p
}

// Incus already creates a bounded root disk and a private NIC for new
// restricted users. Only these unchanged defaults can be prepared, never
// an arbitrary host mount, a proxy, foreign bridge or a preexisting ACL.
func nativePrepareProfileEligible(devices api.DevicesMap, t NativePrepareTarget) error {
	if len(devices) > 2 {
		return errors.New("owned profile contains unreviewed devices")
	}
	for name, d := range devices {
		switch name {
		case "root":
			if d["type"] != "disk" || d["path"] != "/" || d["source"] != "" || len(d) != 3 || d["pool"] == "" {
				return errors.New("root device not the safe native default")
			}
		case "eth0":
			if d["type"] != "nic" || d["network"] != t.Network ||
				(d["name"] != "" && d["name"] != "eth0") ||
				d["security.acls"] != "" || d["security.acls.default.ingress.action"] != "" ||
				d["security.acls.default.egress.action"] != "" ||
				d["security.ipv4_filtering"] != "" || d["security.ipv6_filtering"] != "" ||
				d["security.mac_filtering"] != "" || len(d) > 4 {
				return errors.New("owned NIC not untouched native bridge default")
			}
		default:
			return errors.New("unexpected profile device or host mount")
		}
	}
	return nil
}

// Internet=true must not permit traffic to OTHER native Projects through
// host-routed private networks. On Incus 7.6 reject rules take precedence
// over allow rules regardless of their listed order (not first-match).
// We therefore carve exactly the selected bridge gateway ADDRESS from the
// otherwise-denied private CIDRs, without admitting adjacent bridge guests.
// For internet=false, the NIC's normal egress default reject still applies
// to that gateway (apart from Incus baseline DHCP/DNS/RA services).
//
// IPv6 link-local fe80::/10 MUST remain available: on a real Incus 7.6
// bridge, blocking it prevents Router Advertisements and breaks IPv6 routing.
// Link-local addresses cannot be routed to another Project subnet.
// This blocks private/CGNAT/ULA routing, not administrator-created foreign
// networks that use global unicast addresses. Such networks remain a
// host-specific BH-009 physical deployment review requirement.
const nativeACLBlockedIPv4 = "0.0.0.0/8,10.0.0.0/8,100.64.0.0/10,127.0.0.0/8,169.254.0.0/16,172.16.0.0/12,192.0.0.0/24,192.0.2.0/24,192.168.0.0/16,198.18.0.0/15,198.51.100.0/24,203.0.113.0/24,224.0.0.0/4,240.0.0.0/4"
const nativeACLBlockedIPv6 = "fc00::/7,::1/128"

// Return a set of DISJOINT CIDRs matching the blocked destinations except
// the single reviewed gateway. It is important that no reject CIDR contains
// the allowed gateway: Incus prioritizes reject over allow.
func nativeACLWithoutGateway(ranges string, gateway netip.Addr) (string, error) {
	var out []string
	for _, raw := range strings.Split(ranges, ",") {
		p, err := netip.ParsePrefix(raw)
		if err != nil {
			return "", fmt.Errorf("invalid private ACL CIDR %q: %w", raw, err)
		}
		p = p.Masked()
		if !gateway.IsValid() || !p.Contains(gateway) {
			out = append(out, p.String())
			continue
		}
		for p.Bits() < p.Addr().BitLen() {
			n := p.Bits()
			left := netip.PrefixFrom(p.Addr(), n+1).Masked()
			var other netip.Addr
			if p.Addr().Is4() {
				a := p.Addr().As4()
				a[n/8] ^= byte(1 << uint(7-n%8))
				other = netip.AddrFrom4(a)
			} else {
				a := p.Addr().As16()
				a[n/8] ^= byte(1 << uint(7-n%8))
				other = netip.AddrFrom16(a)
			}
			right := netip.PrefixFrom(other, n+1).Masked()
			if left.Contains(gateway) {
				out = append(out, right.String())
				p = left
			} else {
				out = append(out, left.String())
				p = right
			}
		}
		// The final /32 or /128, containing ONLY the trusted gateway,
		// is deliberately omitted. All other private targets are denied.
	}
	return strings.Join(out, ","), nil
}

func nativeACLGateway(config api.ConfigMap, key string, ipv6 bool) (netip.Addr, error) {
	raw := strings.TrimSpace(config[key])
	if raw == "" || raw == "none" {
		if ipv6 {
			return netip.Addr{}, nil // no v6 gateway, still block ULA/LL
		}
		return netip.Addr{}, errors.New("missing IPv4 native bridge gateway")
	}
	prefix, err := netip.ParsePrefix(raw)
	if err != nil || !prefix.IsValid() {
		return netip.Addr{}, errors.New("selected native bridge gateway CIDR unparseable")
	}
	addr := prefix.Addr()
	if ipv6 && !addr.Is6() || !ipv6 && !addr.Is4() ||
		addr.IsUnspecified() || addr.IsLoopback() || addr.IsMulticast() {
		return netip.Addr{}, errors.New("invalid native bridge gateway family or address")
	}
	return addr, nil
}

func nativePrepareACL(t NativePrepareTarget, network *api.Network) (api.NetworkACLsPost, error) {
	if network == nil || network.Type != "bridge" || !network.Managed ||
		network.Status != "Created" {
		return api.NetworkACLsPost{}, errors.New("selected native network not a live managed bridge")
	}
	v4, err := nativeACLGateway(network.Config, "ipv4.address", false)
	if err != nil {
		return api.NetworkACLsPost{}, err
	}
	v6, err := nativeACLGateway(network.Config, "ipv6.address", true)
	if err != nil {
		return api.NetworkACLsPost{}, err
	}
	blockedV4, err := nativeACLWithoutGateway(nativeACLBlockedIPv4, v4)
	if err != nil {
		return api.NetworkACLsPost{}, err
	}
	blockedV6, err := nativeACLWithoutGateway(nativeACLBlockedIPv6, v6)
	if err != nil {
		return api.NetworkACLsPost{}, err
	}
	// No explicit allows: the per-NIC Internet switch remains authoritative.
	egress := []api.NetworkACLRule{
		{Action: "reject", State: "enabled", Destination: blockedV4,
			Description: "Deny routed private and reserved IPv4 networks except exact bridge gateway"},
		{Action: "reject", State: "enabled", Destination: blockedV6,
			Description: "Deny routed ULA IPv6 except gateway, preserve link-local RA"},
	}
	return api.NetworkACLsPost{
		NetworkACLPost: api.NetworkACLPost{Name: t.ACL},
		NetworkACLPut: api.NetworkACLPut{Description: "BoxWeave v1 exclusive to " + t.Project,
			Ingress: []api.NetworkACLRule{}, Egress: egress, Config: api.ConfigMap{}},
	}, nil
}

func nativeACLPrivateEgressSafe(acl *api.NetworkACL, network *api.Network) bool {
	if acl == nil || network == nil {
		return false
	}
	expected, err := nativePrepareACL(NativePrepareTarget{ACL: acl.Name}, network)
	if err != nil {
		return false
	}
	// The exact gateway carve-out, destination CIDR set and deny-only rules
	// must be unchanged; simply checking ACL name or used_by is not enough.
	return reflect.DeepEqual(acl.Egress, expected.Egress) &&
		len(acl.Ingress) == 0
}
func nativePrepareStateDigest(objects ...any) string {
	h := sha256.New()
	for _, v := range objects {
		b, _ := json.Marshal(v)
		_, _ = h.Write(b)
		_, _ = h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}
func prepareClient(ctx context.Context) (incus.InstanceServer, error) {
	return incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
}
