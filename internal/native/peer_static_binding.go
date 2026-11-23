package native

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	incus "github.com/lxc/incus/v7/client"
	"net"
	"net/netip"
	"reflect"
	"slices"
	"strings"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/networklimits"
)

// VerifiedStaticPeerBinding is an independently observed local NIC/static
// lease identity, not authorization to grant connectivity. PRS must still
// perform the full root bridge/global-overlap/kernel audit.
type VerifiedStaticPeerBinding struct {
	Bridge, ACL, MAC, IPv4, IPv6 string
}

func staticPeerBindingSnapshot(inst *api.Instance, nw *api.Network, p4, p6 netip.Prefix, leases []api.NetworkLease, project, name, uuid, original string) (VerifiedStaticPeerBinding, error) {
	var zero VerifiedStaticPeerBinding
	if inst == nil || nw == nil || original == "" || inst.Config[createOperationKey] != original ||
		inst.Name != name || inst.Config["volatile.uuid"] != uuid || (inst.Project != "" && inst.Project != project) {
		return zero, ErrNotOwned
	}
	local, ok := inst.Devices["eth0"]
	if !ok || local["type"] != "nic" {
		return zero, ErrUnsafeNetwork
	}
	mac, err := peerACLMAC(inst.Config["volatile.eth0.hwaddr"])
	if err != nil {
		mac, err = peerACLMAC(local["hwaddr"])
	}
	if err != nil {
		return zero, err
	}
	b := VerifiedStaticPeerBinding{local["network"], local["security.acls"], mac, local["ipv4.address"], local["ipv6.address"]}
	if b.Bridge == "" || b.ACL == "" || nw.Name != b.Bridge || nw.Type != "bridge" || !nw.Managed || nw.Status != "Created" || nw.Config["security.acls"] != "" || len(nw.Locations) > 1 {
		return zero, ErrUnsafeNetwork
	}
	v4, e := netip.ParseAddr(b.IPv4)
	if e != nil || !v4.Is4() || !v4.IsGlobalUnicast() || v4.String() != b.IPv4 {
		return zero, ErrUnsafeNetwork
	}
	v6, e := netip.ParseAddr(b.IPv6)
	if e != nil || !v6.Is6() || v6.Is4In6() || !v6.IsGlobalUnicast() || v6.String() != b.IPv6 {
		return zero, ErrUnsafeNetwork
	}
	for _, x := range []struct {
		prefix netip.Prefix
		ip     netip.Addr
	}{{p4, v4}, {p6, v6}} {
		if !x.prefix.IsValid() || !x.prefix.Contains(x.ip) || x.prefix.Addr() == x.ip || x.prefix.Masked().Addr() == x.ip {
			return zero, ErrUnsafeNetwork
		}
	}
	if err = ValidatePeerACLBox(inst, project, b.Bridge, b.ACL, PeerACLBox{Name: name, UUID: uuid, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6}, false); err != nil {
		return zero, err
	}
	seen := map[string]int{}
	for _, l := range leases {
		address, e := netip.ParseAddr(l.Address)
		matchIP := e == nil && (address == v4 || address == v6)
		matchMAC := strings.EqualFold(l.Hwaddr, b.MAC)
		matchName := l.Hostname == name
		if !matchIP && !matchMAC && !matchName {
			continue
		}
		// A matching dynamic-only record is never sufficient. Incus may
		// return a matching static + dynamic SLAAC lease simultaneously.
		if !matchIP || !matchMAC || !matchName || l.Location != "" && len(nw.Locations) == 1 && l.Location != nw.Locations[0] {
			return zero, ErrUnsafeNetwork
		}
		switch l.Type {
		case "static":
			seen[address.String()]++
		case "dynamic":
		default:
			return zero, ErrUnsafeNetwork
		}
	}
	if seen[b.IPv4] != 1 || seen[b.IPv6] != 1 {
		return zero, ErrUnsafeNetwork
	}
	return b, nil
}

// ObserveOriginalCreateStaticPeerBinding is read-only and rechecks native
// config, lease identity and native object ETags. It refuses dynamic-only
// leases, stale exact UUIDs, duplicate addresses, and ambiguous static
// ownership. The restricted view is not a substitute for root global audit.
func (r Reader) ObserveOriginalCreateStaticPeerBinding(ctx context.Context, name, uuid, original string) (VerifiedStaticPeerBinding, error) {
	var zero VerifiedStaticPeerBinding
	c, inst, etag, err := r.confinedExactBox(ctx, name, uuid)
	if err != nil {
		return zero, err
	}
	if etag == "" || !c.HasExtension("network_leases") {
		return zero, ErrNativeUnsupported
	}
	bridge := inst.Devices["eth0"]["network"]
	nw, nt, err := c.GetNetwork(bridge)
	if err != nil {
		return zero, err
	}
	if nt == "" {
		return zero, ErrUnverified
	}
	project, pt, err := c.GetProject(r.Project)
	if err != nil {
		return zero, err
	}
	if pt == "" || project == nil || project.Config["restricted"] != "true" || !PeerManagedNICPolicy(project.Config["restricted.devices.nic"]) || project.Config["restricted.networks.access"] != bridge {
		return zero, ErrUnsafeNetwork
	}
	leases, err := c.GetNetworkLeases(bridge)
	if err != nil {
		return zero, err
	}
	p4, p6, err := peerLivePrefixes(c, nw)
	if err != nil {
		return zero, err
	}
	a, err := staticPeerBindingSnapshot(inst, nw, p4, p6, leases, r.Project, name, uuid, original)
	if err != nil {
		return zero, err
	}
	again, et2, err := c.GetInstance(name)
	if err != nil {
		return zero, err
	}
	nw2, nt2, err := c.GetNetwork(bridge)
	if err != nil {
		return zero, err
	}
	p2, pt2, err := c.GetProject(r.Project)
	if err != nil {
		return zero, err
	}
	leases2, err := c.GetNetworkLeases(bridge)
	if err != nil {
		return zero, err
	}
	q4, q6, err := peerLivePrefixes(c, nw2)
	if err != nil || p4 != q4 || p6 != q6 {
		return zero, ErrUnverified
	}
	b, err := staticPeerBindingSnapshot(again, nw2, q4, q6, leases2, r.Project, name, uuid, original)
	if err != nil || a != b || etag != et2 || nt != nt2 || pt != pt2 || !reflect.DeepEqual(nw.Config, nw2.Config) || !reflect.DeepEqual(project.Config, p2.Config) {
		return zero, ErrUnverified
	}
	return a, nil
}

// ReserveCreatePeerBinding is an internal durable reservation callback.
type ReserveCreatePeerBinding func(VerifiedStaticPeerBinding) (bool, error)

// selectCreatePeerBinding chooses a free IPv4 and a MAC-derived SLAAC IPv6
// in the exact dedicated managed bridge. Never treat a merely observed
// dynamic lease as a static IP. The Incus Create request supplies both
// static addresses and the exact MAC in one native effect.
type ReadCreatePeerHolds func(context.Context, string, string) (networklimits.AddressHolds, error)

type createPeerInventory struct {
	bridge, acl string
	p4, p6      netip.Prefix
	holds       networklimits.AddressHolds
}

func readCreatePeerInventory(c incus.InstanceServer, project string, proj *api.Project, devices api.DevicesMap) (createPeerInventory, error) {
	var zero createPeerInventory
	nic := devices["eth0"]
	if !nicPrivate(devices) || nic["type"] != "nic" || nic["name"] != "eth0" || nic["ipv4.address"] != "" || nic["ipv6.address"] != "" || nic["hwaddr"] != "" || proj == nil || proj.Config["restricted"] != "true" || !PeerManagedNICPolicy(proj.Config["restricted.devices.nic"]) || proj.Config["restricted.networks.access"] != nic["network"] || !c.HasExtension("network_leases") {
		return zero, ErrUnsafeNetwork
	}
	nw, etag, err := c.GetNetwork(nic["network"])
	if err != nil {
		return zero, err
	}
	if nw == nil || etag == "" || !nw.Managed || nw.Type != "bridge" || nw.Status != "Created" || nw.Name != nic["network"] || nw.Config["security.acls"] != "" || len(nw.Locations) > 1 || nw.Config["ipv4.dhcp"] == "false" {
		return zero, ErrUnsafeNetwork
	}
	p4, p6, err := peerLivePrefixes(c, nw)
	if err != nil {
		return zero, err
	}
	leases, err := c.GetNetworkLeases(nw.Name)
	if err != nil {
		return zero, err
	}
	if len(leases) > 131072 {
		return zero, ErrNativeUnsupported
	}
	usedIP := map[netip.Addr]bool{}
	usedMAC := map[string]bool{}
	for _, l := range leases {
		ip, e := netip.ParseAddr(l.Address)
		if e != nil {
			return zero, ErrUnsafeNetwork
		}
		usedIP[ip] = true
		if l.Hwaddr != "" {
			m, e := peerACLMAC(l.Hwaddr)
			if e != nil {
				return zero, e
			}
			usedMAC[m] = true
		}
	}
	// Also read stopped and running native devices, including unowned siblings.
	// The root Peer audit independently checks the full global bridge inventory.
	names, err := c.GetInstanceNames(api.InstanceTypeAny)
	if err != nil {
		return zero, err
	}
	if len(names) > networklimits.MaxProjectBoxes {
		return zero, ErrNativeUnsupported
	}
	instances, err := c.GetInstances(api.InstanceTypeAny)
	if err != nil {
		return zero, err
	}
	if len(instances) != len(names) {
		return zero, ErrUnsafeNetwork
	}
	actual := make([]string, 0, len(instances))
	for _, inst := range instances {
		if inst.Name == "" || (inst.Project != "" && inst.Project != project) {
			return zero, ErrUnsafeNetwork
		}
		actual = append(actual, inst.Name)
	}
	slices.Sort(names)
	slices.Sort(actual)
	if !slices.Equal(names, actual) || len(slices.Compact(slices.Clone(actual))) != len(actual) {
		return zero, ErrUnsafeNetwork
	}
	for _, inst := range instances {
		for dev, d := range inst.ExpandedDevices {
			if d["type"] != "nic" || d["network"] != nw.Name {
				continue
			}
			for _, k := range []string{"ipv4.address", "ipv6.address"} {
				if d[k] != "" {
					ip, e := netip.ParseAddr(d[k])
					if e != nil {
						return zero, ErrUnsafeNetwork
					}
					usedIP[ip] = true
				}
			}
			for _, raw := range []string{d["hwaddr"], inst.Config["volatile."+dev+".hwaddr"]} {
				if raw != "" {
					mac, e := peerACLMAC(raw)
					if e != nil {
						return zero, e
					}
					usedMAC[mac] = true
				}
			}
		}
	}
	return createPeerInventory{nw.Name, nic["security.acls"], p4, p6, networklimits.AddressHolds{IPs: usedIP, MACs: usedMAC}}, nil
}

func (i *createPeerInventory) mergeHolds(ctx context.Context, project string, read ReadCreatePeerHolds) error {
	if read == nil {
		return ErrUnsafeNetwork
	}
	h, e := read(ctx, project, i.bridge)
	if e != nil {
		return e
	}
	for ip := range h.IPs {
		i.holds.IPs[ip] = true
	}
	for mac := range h.MACs {
		i.holds.MACs[mac] = true
	}
	return nil
}

func selectCreatePeerBinding(ctx context.Context, c incus.InstanceServer, project string, proj *api.Project, devices api.DevicesMap, original string, reserve ReserveCreatePeerBinding, read ReadCreatePeerHolds) (VerifiedStaticPeerBinding, error) {
	var zero VerifiedStaticPeerBinding
	if original == "" || reserve == nil {
		return zero, ErrUnsafeNetwork
	}
	i, e := readCreatePeerInventory(c, project, proj, devices)
	if e != nil {
		return zero, e
	}
	if e = i.mergeHolds(ctx, project, read); e != nil {
		return zero, e
	}
	return chooseCreatePeerBinding(i.bridge, i.acl, original, i.p4, i.p6, i.holds.IPs, i.holds.MACs, reserve)
}

// AddressCapacity describes IPv4 candidates in the current confined native
// and durable ledger snapshot, NOT authorization to allocate or a guarantee
// of dual-stack, storage or compute availability.
type AddressCapacity struct {
	IPv4Prefix              string `json:"ipv4_prefix"`
	NominalIPv4             int    `json:"nominal_ipv4"`
	BlockedIPv4             int    `json:"blocked_ipv4"`
	RemainingIPv4UpperBound int    `json:"remaining_ipv4_upper_bound"`
}

func (i createPeerInventory) capacity() AddressCapacity {
	out := AddressCapacity{IPv4Prefix: i.p4.String(), NominalIPv4: (1 << (32 - i.p4.Bits())) - 3}
	first := i.p4.Masked().Addr()
	last := binary.BigEndian.Uint32(first.AsSlice()) + uint32(1<<(32-i.p4.Bits())) - 1
	for a := range i.holds.IPs {
		if a.Is4() && i.p4.Contains(a) && a != first && a != i.p4.Addr() && binary.BigEndian.Uint32(a.AsSlice()) != last {
			out.BlockedIPv4++
		}
	}
	out.RemainingIPv4UpperBound = max(0, out.NominalIPv4-out.BlockedIPv4)
	return out
}

func (r Reader) ReadCreateAddressCapacity(ctx context.Context, read ReadCreatePeerHolds) (AddressCapacity, error) {
	var zero AddressCapacity
	c, e := r.confined(ctx)
	if e != nil {
		return zero, e
	}
	proj, _, e := c.GetProject(r.Project)
	if e != nil {
		return zero, e
	}
	p, _, e := c.GetProfile("default")
	if e != nil || p == nil {
		return zero, ErrUnsafeNetwork
	}
	// Reconstruct the exact private NIC as used by Create, without resolving
	// or downloading an image, changing native state, or reserving an IP.
	devices, e := ownedPrivateNICEgress(api.DevicesMap{}, p.Devices, true)
	if e != nil {
		return zero, e
	}
	i, e := readCreatePeerInventory(c, r.Project, proj, devices)
	if e != nil {
		return zero, e
	}
	if e = i.mergeHolds(ctx, r.Project, read); e != nil {
		return zero, e
	}
	return i.capacity(), nil
}

// PeerManagedNICPolicy follows the documented Incus v7.6 default.
func PeerManagedNICPolicy(mode string) bool { return mode == "" || mode == "managed" }

// Restricted Unix actors see redacted Network.Config. Read the actual bridge
// state; never guess a prefix from a lease. The root audit still checks config.
func peerLivePrefixes(c incus.InstanceServer, nw *api.Network) (netip.Prefix, netip.Prefix, error) {
	var z netip.Prefix
	if nw == nil || !nw.Managed || nw.Type != "bridge" || nw.Status != "Created" || nw.Name == "" {
		return z, z, ErrUnsafeNetwork
	}
	st, err := c.GetNetworkState(nw.Name)
	if err != nil {
		return z, z, err
	}
	return peerStatePrefixes(nw, st)
}
func peerStatePrefixes(nw *api.Network, st *api.NetworkState) (netip.Prefix, netip.Prefix, error) {
	var p4, p6, z netip.Prefix
	if nw == nil || st == nil || st.State != "up" || st.Bridge == nil {
		return z, z, ErrUnsafeNetwork
	}
	for _, a := range st.Addresses {
		if a.Scope != "global" {
			continue
		}
		p, e := netip.ParsePrefix(a.Address + "/" + a.Netmask)
		if e != nil || !p.Addr().IsGlobalUnicast() || p.Addr().Is4In6() {
			return z, z, ErrUnsafeNetwork
		}
		switch a.Family {
		case "inet":
			if p4.IsValid() || !p.Addr().Is4() || p.Bits() < 16 || p.Bits() > 30 {
				return z, z, ErrNativeUnsupported
			}
			p4 = p
		case "inet6":
			if p6.IsValid() || !p.Addr().Is6() || p.Bits() != 64 {
				return z, z, ErrNativeUnsupported
			}
			p6 = p
		default:
			return z, z, ErrUnsafeNetwork
		}
	}
	if !p4.IsValid() || !p6.IsValid() {
		return z, z, ErrNativeUnsupported
	}
	for k, p := range map[string]netip.Prefix{"ipv4.address": p4, "ipv6.address": p6} {
		if raw, ok := nw.Config[k]; ok {
			actual, e := netip.ParsePrefix(raw)
			if e != nil || actual != p {
				return z, z, ErrUnsafeNetwork
			}
		}
	}
	return p4, p6, nil
}
func chooseCreatePeerBinding(bridge, acl, original string, p4, p6 netip.Prefix, usedIP map[netip.Addr]bool, usedMAC map[string]bool, reserve ReserveCreatePeerBinding) (VerifiedStaticPeerBinding, error) {
	var zero VerifiedStaticPeerBinding
	if p4.Bits() < 16 || p4.Bits() > 30 || !p4.Addr().Is4() || p6.Bits() != 64 || !p6.Addr().Is6() || reserve == nil {
		return zero, ErrNativeUnsupported
	}
	size := uint32(1) << uint(32-p4.Bits())
	base := p4.Masked().Addr().As4()
	baseN := binary.BigEndian.Uint32(base[:])
	digest := sha256.Sum256([]byte("boxweave-create-pin-v1:" + original))
	offset := binary.BigEndian.Uint32(digest[:4]) % (size - 2)
	for i := uint32(0); i < size-2; i++ {
		var raw [4]byte
		binary.BigEndian.PutUint32(raw[:], baseN+1+(offset+i)%(size-2))
		v4 := netip.AddrFrom4(raw)
		if v4 == p4.Addr() || usedIP[v4] || !v4.IsGlobalUnicast() {
			continue
		}
		h := sha256.Sum256([]byte(fmt.Sprintf("boxweave-create-mac-v1:%s:%d", original, i)))
		macRaw := []byte{(h[0] | 2) & 0xfe, h[1], h[2], h[3], h[4], h[5]}
		mac := net.HardwareAddr(macRaw).String()
		ipv6 := p6.Masked().Addr().As16()
		copy(ipv6[8:], []byte{macRaw[0] ^ 2, macRaw[1], macRaw[2], 0xff, 0xfe, macRaw[3], macRaw[4], macRaw[5]})
		v6 := netip.AddrFrom16(ipv6)
		if v6 == p6.Addr() || usedIP[v6] || usedMAC[mac] || !v6.IsGlobalUnicast() {
			continue
		}
		b := VerifiedStaticPeerBinding{bridge, acl, mac, v4.String(), v6.String()}
		ok, err := reserve(b)
		if err != nil {
			return zero, err
		}
		if ok {
			return b, nil
		}
	}
	return zero, ErrNetworkAddressCapacityExhausted
}
