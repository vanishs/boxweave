package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"strconv"
	"strings"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// NativeIPv4RetirementProof contains a read-only exact-original observation.
// It cannot move a fence itself. The installed controller MUST hold
// the project-wide exclusion/port barrier through the final writer commit.
// No new public Tool, root mutation, or arbitrary native selector is added.
type NativeIPv4RetirementProof struct {
	Candidate state.IPv4RetirementCandidate
	SHA256    string
}

type retirementObservation struct {
	NativeScope, GlobalScope, RouteScope string
	ACLHash                              string
	NFTSHA256, NeighborSHA256, FDBSHA256 string
	ConntrackScopes                      int
	ForwardHistorySHA256                 string
}

func retirementSHA(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Exact no-flow witness for one filter. A nonzero/unknown result is never
// interpreted as absence; this path does not purge or retry any kernel effect.
func retirementConntrackZero(ctx context.Context, run nativePeerConntrackRun, family, selector, ip string) error {
	if run == nil {
		return errors.New("missing fixed conntrack reader")
	}
	if _, e := nativePeerConntrackAddr(ip, family == "ipv6"); e != nil {
		return e
	}
	switch selector {
	case "--orig-src", "--orig-dst", "--reply-src", "--reply-dst":
	default:
		return errors.New("invalid retirement conntrack selector")
	}
	if family != "ipv4" && family != "ipv6" {
		return errors.New("invalid retirement family")
	}
	raw, cmdErr := run(ctx, "-L", "-f", family, selector, ip)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if len(raw) == 0 || len(raw) > 65536 {
		return errors.New("retirement conntrack output missing or excessive")
	}
	line := strings.TrimSpace(string(raw))
	// conntrack-tools 1.4.x emits the count (also when zero exits 1).
	if strings.Contains(line, "\n") || !strings.HasPrefix(line, "conntrack v") || !(strings.HasSuffix(line, ": 0 flow entries have been shown.") || strings.HasSuffix(line, ": 0 flow entries shown.")) {
		return fmt.Errorf("retired endpoint still has conntrack state or readback is unverified")
	}
	if cmdErr != nil {
		var exit *exec.ExitError
		if !errors.As(cmdErr, &exit) || exit.ExitCode() != 1 {
			return errors.New("retirement conntrack read failed")
		}
	}
	return nil
}

// Conservative exact-reference scan of kernel JSON. In particular, an
// old address in an exact NAT/ACL/filter set or a residual neighbour/MAC
// blocks reuse. Broad network prefixes are not mistaken for host literals;
// full native ACL equality is separately required.
func retirementKernelReferenceAbsent(raw []byte, binding state.NetworkAuditBinding, kind string) error {
	if len(raw) == 0 || len(raw) > 8*1024*1024 {
		return errors.New("retirement kernel inventory missing or excessive")
	}
	var value any
	if e := json.Unmarshal(raw, &value); e != nil {
		return errors.New("unparseable retirement kernel inventory")
	}
	if kind == "nft" {
		root, ok := value.(map[string]any)
		if !ok {
			return errors.New("unexpected nft ruleset shape")
		}
		rows, ok := root["nftables"].([]any)
		if !ok || len(rows) == 0 {
			return errors.New("empty nft ruleset")
		}
		// At least one table is required; an absent firewall is not proof.
		tables := 0
		for _, row := range rows {
			m, ok := row.(map[string]any)
			if !ok {
				return errors.New("invalid nft object")
			}
			if _, ok = m["table"]; ok {
				tables++
			}
		}
		if tables == 0 {
			return errors.New("native nft firewall tables unobserved")
		}
	} else if _, ok := value.([]any); !ok {
		return errors.New("unexpected kernel table shape")
	}
	// A failed IPv6 neighbor lookup without a link-layer mapping is a
	// negative cache entry, not a usable reference to the retired endpoint.
	// IPv6 itself is never recycled. Only accept the exact iproute2 shape;
	// any MAC, router flag, additional state or field remains fail-closed.
	if kind == "neighbors" {
		rows := value.([]any)
		kept := make([]any, 0, len(rows))
		for _, row := range rows {
			m, ok := row.(map[string]any)
			if ok && len(m) == 2 && m["dst"] == binding.IPv6 {
				states, ok := m["state"].([]any)
				if ok && len(states) == 1 && states[0] == "FAILED" {
					continue
				}
			}
			kept = append(kept, row)
		}
		value = kept
	}
	exact := map[string]bool{binding.IPv4: true, binding.IPv4 + "/32": true, binding.IPv6: true, binding.IPv6 + "/128": true, binding.MAC: true}
	var scan func(any, int) error
	scan = func(v any, depth int) error {
		if depth > 40 {
			return errors.New("excessive kernel inventory nesting")
		}
		switch u := v.(type) {
		case string:
			if exact[strings.ToLower(u)] {
				return errors.New("retired exact address or MAC still referenced by kernel")
			}
		case []any:
			for _, x := range u {
				if e := scan(x, depth+1); e != nil {
					return e
				}
			}
		case map[string]any:
			for k, x := range u {
				if k == "prefix" {
					p, ok := x.(map[string]any)
					if !ok {
						return errors.New("invalid kernel prefix")
					}
					addr, aok := p["addr"].(string)
					bits, bok := p["len"].(float64)
					if !aok || !bok || bits != float64(int(bits)) {
						return errors.New("invalid kernel IP prefix")
					}
					ip, e := netip.ParseAddr(addr)
					if e != nil || bits < 0 || bits > float64(ip.BitLen()) {
						return errors.New("invalid kernel IP prefix")
					}
					if int(bits) == ip.BitLen() && exact[ip.String()] {
						return errors.New("retired exact host prefix remains")
					}
					continue
				}
				if e := scan(x, depth+1); e != nil {
					return e
				}
			}
		}
		return nil
	}
	return scan(value, 0)
}

func retirementKernelCommand(ctx context.Context, path string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = []string{"LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	return cmd.CombinedOutput()
}

func retirementNativeIdentityAbsent(instances []api.Instance, b state.NetworkAuditBinding) error {
	for _, inst := range instances {
		if inst.Config["volatile.uuid"] == b.UUID {
			return errors.New("retired UUID still exists in native inventory")
		}
		for dev, d := range inst.ExpandedDevices {
			if d["ipv4.address"] == b.IPv4 || d["ipv6.address"] == b.IPv6 || strings.EqualFold(d["hwaddr"], b.MAC) || strings.EqualFold(inst.Config["volatile."+dev+".hwaddr"], b.MAC) {
				return errors.New("retired address or MAC still used by native device")
			}
			if d["type"] == "proxy" {
				for _, raw := range []string{d["connect"], d["listen"]} {
					if strings.Contains(raw, b.IPv4+":") || strings.Contains(raw, "["+b.IPv6+"]:") {
						return errors.New("retired address still referenced by native proxy")
					}
				}
			}
		}
	}
	return nil
}

func observeRetirement(ctx context.Context, root incus.InstanceServer, policy state.IPv4RetirementPolicy) (retirementObservation, error) {
	var out retirementObservation
	c := policy.Candidate
	b := c.Binding
	if e := nativePeerBridgeKernelNoLivePorts(ctx, c.Bridge); e != nil {
		return out, retirementFailure("RETIREMENT_BRIDGE_NOT_QUIESCENT", e)
	}
	target := nativePeerTargetFromLedger(policy.Ledger)
	nativeAudit, e := nativePeerBridgeAuditScoped(ctx, root, target, true)
	if e != nil {
		return out, retirementFailure("RETIREMENT_NATIVE_IDENTITY_UNVERIFIED", e)
	}
	out.NativeScope = nativeAudit.NativeDigestSHA256
	instances, e := root.GetInstancesAllProjects(api.InstanceTypeAny)
	if e != nil {
		return out, e
	}
	if e = retirementNativeIdentityAbsent(instances, b); e != nil {
		return out, retirementFailure("RETIREMENT_NATIVE_IDENTITY_UNVERIFIED", e)
	}
	// Native proxies are local instance devices, not arbitrary forwards.
	// For each verified historical claim refuse if its device or any
	// local socket on its old protocol/port remains (conservative across
	// unrelated binds; never kill a foreign listener or inspect its owner).
	for _, f := range policy.Forwards {
		if f.HostPort < 1 || f.HostPort > 65535 || (f.Protocol != "tcp" && f.Protocol != "udp") {
			return out, retirementFailure("RETIREMENT_PROXY_NOT_CLEAR", errors.New("invalid historical proxy proof"))
		}
		for _, inst := range instances {
			if _, ok := inst.ExpandedDevices[f.NativeDevice]; ok {
				return out, retirementFailure("RETIREMENT_PROXY_NOT_CLEAR", errors.New("retired native proxy device still exists"))
			}
		}
		family := "-tan"
		if f.Protocol == "udp" {
			family = "-uan"
		}
		raw, e := retirementKernelCommand(ctx, "/usr/bin/ss", "-H", family, "sport = :"+strconv.Itoa(f.HostPort))
		if e != nil || len(strings.TrimSpace(string(raw))) != 0 {
			return out, retirementFailure("RETIREMENT_PROXY_NOT_CLEAR", errors.New("retired proxy local socket remains or cannot be verified"))
		}
	}
	data, e := json.Marshal(policy.Forwards)
	if e != nil {
		return out, e
	}
	out.ForwardHistorySHA256 = retirementSHA(data)

	// The kernel bridge must remain actually quiescent; never stop another
	// user's or sibling's guest from this read-only capability.
	for _, inst := range instances {
		p := inst.Project
		if p == "" {
			p = "default"
		}
		if p == c.Project && inst.Status != "Stopped" {
			return out, retirementFailure("RETIREMENT_BRIDGE_NOT_QUIESCENT", errors.New("retirement requires all owned guests stopped"))
		}
	}
	nw, _, e := root.GetNetwork(c.Bridge)
	if e != nil || nw == nil {
		return out, errors.New("retirement bridge unavailable")
	}
	acl, _, e := root.GetNetworkACL(c.ACL)
	if e != nil || acl == nil {
		return out, errors.New("retirement ACL unavailable")
	}
	want, e := NativePeerACLFromPolicySnapshot(NativePrepareTarget{Project: c.Project, Network: c.Bridge, ACL: c.ACL}, nw, policy.Active, policy.Revoked)
	if e != nil || !nativePeerACLMatches(acl, want) {
		return out, retirementFailure("RETIREMENT_POLICY_UNVERIFIED", errors.New("retirement native ACL differs from complete durable policy"))
	}
	out.ACLHash = nativePrepareStateDigest(want)
	// The old address must still belong to this exclusive native subnet;
	// never recycle a gateway, network, broadcast, or foreign IP.
	for _, v := range []struct{ key, raw string }{{"ipv4.address", b.IPv4}, {"ipv6.address", b.IPv6}} {
		prefix, e := netip.ParsePrefix(nw.Config[v.key])
		ip, ie := netip.ParseAddr(v.raw)
		if e != nil || ie != nil || !prefix.Contains(ip) || ip == prefix.Addr() || ip == prefix.Masked().Addr() {
			return out, errors.New("retirement address outside native host range")
		}
		if ip.Is4() {
			a := prefix.Masked().Addr().As4()
			x := ip.As4()
			last := uint32(a[0])<<24 | uint32(a[1])<<16 | uint32(a[2])<<8 | uint32(a[3])
			last += uint32(1)<<uint(32-prefix.Bits()) - 1
			ipn := uint32(x[0])<<24 | uint32(x[1])<<16 | uint32(x[2])<<8 | uint32(x[3])
			if ipn == last {
				return out, errors.New("retirement broadcast address refused")
			}
		}
	}
	leases, e := root.GetNetworkLeases(c.Bridge)
	if e != nil || len(leases) > 131072 {
		return out, retirementFailure("RETIREMENT_LEASE_NOT_CLEAR", errors.New("retirement lease inventory unverified"))
	}
	for _, l := range leases {
		if _, e := netip.ParseAddr(l.Address); e != nil {
			return out, retirementFailure("RETIREMENT_LEASE_NOT_CLEAR", errors.New("unparseable native lease"))
		}
		if l.Address == b.IPv4 || l.Address == b.IPv6 || strings.EqualFold(l.Hwaddr, b.MAC) {
			return out, retirementFailure("RETIREMENT_LEASE_NOT_CLEAR", errors.New("retired native lease still exists"))
		}
	}
	forwards, e := root.GetNetworkForwards(c.Bridge)
	if e != nil || len(forwards) != 0 {
		return out, errors.New("native bridge has unverified network forwards")
	}
	onlyOld := NativePeerBridgeAuditTarget{Project: c.Project, Bridge: c.Bridge, ACL: c.ACL, Boxes: []native.PeerACLBox{{Name: b.Name, UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6}}}
	networks, e := root.GetNetworksAllProjects()
	if e != nil {
		return out, e
	}
	out.GlobalScope, e = nativeNetworkScopeFromInventory(onlyOld, networks, instances, 1)
	if e != nil {
		return out, e
	}
	var routes [2][]nativePeerHostRoute
	for i, f := range []string{"-4", "-6"} {
		raw, e := nativePeerHostIPOS(ctx, "-j", f, "route", "show", "table", "all")
		if e != nil {
			return out, e
		}
		routes[i], e = nativePeerHostRoutesParse(raw)
		if e != nil {
			return out, e
		}
	}
	out.RouteScope, e = nativeHostRouteScopeFromRows(onlyOld, routes[0], routes[1], 1)
	if e != nil {
		return out, e
	}
	for _, probe := range []struct {
		path, kind string
		args       []string
		dest       *string
	}{
		{"/usr/sbin/nft", "nft", []string{"-j", "list", "ruleset"}, &out.NFTSHA256},
		{"/usr/sbin/ip", "neighbors", []string{"-j", "neigh", "show", "dev", c.Bridge}, &out.NeighborSHA256},
		{"/usr/sbin/bridge", "fdb", []string{"-j", "fdb", "show", "br", c.Bridge}, &out.FDBSHA256},
	} {
		raw, e := retirementKernelCommand(ctx, probe.path, probe.args...)
		if e != nil {
			return out, retirementFailure("RETIREMENT_KERNEL_REFERENCE_UNVERIFIED", errors.New("retirement kernel inventory unavailable"))
		}
		if e = retirementKernelReferenceAbsent(raw, b, probe.kind); e != nil {
			return out, retirementFailure("RETIREMENT_KERNEL_REFERENCE_UNVERIFIED", e)
		}
		*probe.dest = retirementSHA(raw)
	}
	for _, f := range []struct{ family, ip string }{{"ipv4", b.IPv4}, {"ipv6", b.IPv6}} {
		for _, sel := range []string{"--orig-src", "--orig-dst", "--reply-src", "--reply-dst"} {
			if e = retirementConntrackZero(ctx, nativePeerConntrackOS, f.family, sel, f.ip); e != nil {
				return out, retirementFailure("RETIREMENT_CONNTRACK_NOT_CLEAR", e)
			}
			out.ConntrackScopes++
		}
	}
	if e = nativePeerBridgeKernelNoLivePorts(ctx, c.Bridge); e != nil {
		return out, e
	}
	return out, nil
}

// NativeIPv4RetirementReview reads only the exact protected original, root
// Incus inventory and fixed kernel probes. It never accepts an IP selector
// or purges conntrack, edits a rule, stops a Box or writes SQLite. The
// installed controller holds its writer reservation across this root witness.
func NativeIPv4RetirementReview(ctx context.Context, s *state.Store, project, deleteOriginal string) (NativeIPv4RetirementProof, error) {
	var zero NativeIPv4RetirementProof
	if os.Geteuid() != 0 || s == nil || !s.IsNetworkAuditReadOnly() {
		return zero, errors.New("protected root read-only retirement witness required")
	}
	u, e := user.Lookup("boxweave")
	if e != nil || project != "user-"+u.Uid {
		return zero, errors.New("retirement Project differs from installed identity")
	}
	c, e := s.ReadIPv4RetirementCandidate(ctx, project, deleteOriginal)
	if e != nil {
		return zero, e
	}
	if c.RequiresPeerCompaction {
		return zero, errors.New("read-only retirement cannot certify un-compacted Peer DROP history")
	}
	policy, e := s.ReadIPv4RetirementPolicy(ctx, c)
	if e != nil {
		return zero, e
	}
	root, e := prepareClient(ctx)
	if e != nil {
		return zero, e
	}
	before, e := observeRetirement(ctx, root, policy)
	if e != nil {
		return zero, e
	}
	middle, e := s.ReadIPv4RetirementPolicy(ctx, c)
	if e != nil || !reflect.DeepEqual(policy, middle) {
		return zero, errors.New("retirement protected policy changed")
	}
	after, e := observeRetirement(ctx, root, middle)
	if e != nil {
		return zero, e
	}
	if before.NativeScope != after.NativeScope || before.GlobalScope != after.GlobalScope || before.RouteScope != after.RouteScope || before.ACLHash != after.ACLHash || before.ForwardHistorySHA256 != after.ForwardHistorySHA256 || before.ConntrackScopes != 8 || after.ConntrackScopes != 8 {
		return zero, errors.New("retirement native scope changed between observations")
	}
	final, e := s.ReadIPv4RetirementPolicy(ctx, c)
	if e != nil || !reflect.DeepEqual(policy, final) {
		return zero, errors.New("retirement protected policy changed after kernel read")
	}
	data, e := json.Marshal(struct {
		Candidate     state.IPv4RetirementCandidate
		Before, After retirementObservation
	}{c, before, after})
	if e != nil {
		return zero, e
	}
	return NativeIPv4RetirementProof{Candidate: c, SHA256: retirementSHA(append([]byte("BoxWeave IPv4 retirement proof v1\n"), data...))}, nil
}
