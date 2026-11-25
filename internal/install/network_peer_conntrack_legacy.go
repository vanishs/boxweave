package install

import (
	"context"
	"errors"
	"fmt"
	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"net/netip"
	"os"
	"os/user"
	"strings"
)

// Legacy whole-bridge stop-fenced, bidirectional conntrack path.
// Keep until its callers and witnesses have been converted.
func nativePeerACLDropCovered(acl *api.NetworkACL, a, b native.PeerACLBox) error {
	if acl == nil {
		return errors.New("no native ACL readback")
	}
	a4, ea4 := nativePeerConntrackAddr(a.IPv4, false)
	a6, ea6 := nativePeerConntrackAddr(a.IPv6, true)
	b4, eb4 := nativePeerConntrackAddr(b.IPv4, false)
	b6, eb6 := nativePeerConntrackAddr(b.IPv6, true)
	if ea4 != nil || ea6 != nil || eb4 != nil || eb6 != nil ||
		a4 == b4 || a6 == b6 {
		return errors.New("unsafe native peer address identity")
	}
	for _, rules := range [][]api.NetworkACLRule{acl.Ingress, acl.Egress} {
		for _, fam := range []struct {
			x, y netip.Addr
			bits int
		}{
			{a4, b4, 32}, {b4, a4, 32}, {a6, b6, 128}, {b6, a6, 128},
		} {
			source := fmt.Sprintf("%s/%d", fam.x, fam.bits)
			dest := fmt.Sprintf("%s/%d", fam.y, fam.bits)
			found := false
			for _, rule := range rules {
				if rule.Action == "drop" && rule.State == "enabled" &&
					rule.Source == source && rule.Destination == dest && rule.Protocol == "" {
					found = true
				}
				if rule.Action == "allow" {
					// The result cannot claim a fail-closed peer if ANY other
					// allow could cover either endpoint, even via CIDR or wildcard.
					if nativePeerACLMatchesEndpoint(rule.Source, fam.x) &&
						nativePeerACLMatchesEndpoint(rule.Destination, fam.y) &&
						!nativePeerOnlyNeighborDiscovery(rule) {
						return errors.New("native ACL contains overlapping peer allow")
					}
				}
			}
			if !found {
				return errors.New("exact bidirectional dual-stack DROP absent from native ACL")
			}
		}
	}
	return nil
}
func nativePeerOnlyNeighborDiscovery(r api.NetworkACLRule) bool {
	return r.Protocol == "icmp6" && (r.ICMPType == "135" || r.ICMPType == "136")
}
func nativePeerACLMatchesEndpoint(pattern string, endpoint netip.Addr) bool {
	if pattern == "" {
		return true
	}
	for _, raw := range strings.Split(pattern, ",") {
		raw = strings.TrimSpace(raw)
		prefix, e := netip.ParsePrefix(raw)
		if e != nil {
			return true
		} // parse uncertainty must deny, not allow
		if prefix.Contains(endpoint) {
			return true
		}
	}
	return false
}

func nativePeerConntrackPurgeExact(ctx context.Context, run nativePeerConntrackRun,
	a, b native.PeerACLBox) (NativePeerConntrackReport, error) {
	var out NativePeerConntrackReport
	if run == nil {
		return out, errors.New("missing fixed root conntrack executor")
	}
	if a.UUID == "" || b.UUID == "" || a.UUID == b.UUID {
		return out, errors.New("unsafe native UUID scope")
	}
	for _, p := range []struct {
		raw string
		v6  bool
	}{
		{a.IPv4, false}, {b.IPv4, false}, {a.IPv6, true}, {b.IPv6, true},
	} {
		if _, err := nativePeerConntrackAddr(p.raw, p.v6); err != nil {
			return out, err
		}
	}
	if a.IPv4 == b.IPv4 || a.IPv6 == b.IPv6 {
		return out, errors.New("duplicate peer IP scope")
	}
	out.PairIPv4 = a.IPv4 + "|" + b.IPv4
	out.PairIPv6 = a.IPv6 + "|" + b.IPv6
	for _, pair := range []struct{ family, x, y string }{
		{"ipv4", a.IPv4, b.IPv4}, {"ipv4", b.IPv4, a.IPv4},
		{"ipv6", a.IPv6, b.IPv6}, {"ipv6", b.IPv6, a.IPv6},
	} {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		found, err := nativePeerConntrackList(ctx, run, pair.family, pair.x, pair.y)
		if err != nil {
			return out, err
		}
		out.ProbedScopes++
		if !found {
			continue
		}
		// NEVER repeat a native delete whose result was uncertain.
		if _, err := run(ctx, "-D", "-s", pair.x, "-d", pair.y); err != nil {
			return out, fmt.Errorf("exact conntrack delete uncertain: %w", err)
		}
		out.DeletedScopes++
		found, err = nativePeerConntrackList(ctx, run, pair.family, pair.x, pair.y)
		if err != nil {
			return out, err
		}
		if found {
			return out, errors.New("exact conntrack post-delete still contains session")
		}
	}
	return out, nil
}

// NativePeerConntrackPurge is a host-local root-only primitive, not a
// network Tool. Its argument ledger must be read from protected persistent
// service state by the eventual caller, never from an MCP request.
func NativePeerConntrackPurge(ctx context.Context, t NativePeerBridgeAuditTarget,
	sourceUUID, targetUUID string) (NativePeerConntrackReport, error) {
	var zero NativePeerConntrackReport
	if os.Geteuid() != 0 {
		return zero, errors.New("conntrack invalidation requires local root")
	}
	u, err := user.Lookup("boxweave")
	if err != nil || t.Project != "user-"+u.Uid {
		return zero, errors.New("conntrack scope is not the confined owned Project")
	}
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return zero, err
	}
	// The full stop fence must leave zero live ports on the dedicated host bridge.
	// Reject raw NICs that escaped a restricted-Project inventory.
	if err := nativePeerBridgeKernelNoLivePorts(ctx, t.Bridge); err != nil {
		return zero, err
	}
	report, err := nativePeerConntrackWithClient(ctx, root, t, sourceUUID, targetUUID, nativePeerConntrackOS)
	if err != nil {
		return report, err
	}
	if err := nativePeerBridgeKernelNoLivePorts(ctx, t.Bridge); err != nil {
		return report, fmt.Errorf("bridge acquired port during native conntrack purge: %w", err)
	}
	return report, nil
}

func nativePeerConntrackWithClient(ctx context.Context, root incus.InstanceServer,
	t NativePeerBridgeAuditTarget, srcUUID, dstUUID string,
	run nativePeerConntrackRun) (NativePeerConntrackReport, error) {
	var zero NativePeerConntrackReport
	if root == nil || run == nil {
		return zero, errors.New("missing root-only native proof")
	}
	audit, err := nativePeerBridgeAuditWithClient(ctx, root, t)
	if err != nil {
		return zero, err
	}
	globalBefore, err := nativePeerGlobalNetworkScopeSnapshot(ctx, root, t)
	if err != nil {
		return zero, fmt.Errorf("host-wide native network provenance uncertain before conntrack: %w", err)
	}
	var src, dst native.PeerACLBox
	if err := nativePeerConntrackAllStopped(ctx, root, t); err != nil {
		return zero, err
	}
	for _, b := range t.Boxes {
		if b.UUID == srcUUID {
			src = b
		}
		if b.UUID == dstUUID {
			dst = b
		}
	}
	if src.UUID == "" || dst.UUID == "" || src.UUID == dst.UUID {
		return zero, errors.New("pair UUID not in complete stopped ledger")
	}
	acl, _, err := root.GetNetworkACL(t.ACL)
	if err != nil {
		return zero, err
	}
	if err = nativePeerACLDropCovered(acl, src, dst); err != nil {
		return zero, err
	}
	repeat, err := nativePeerBridgeAuditWithClient(ctx, root, t)
	if err != nil || repeat.NativeDigestSHA256 != audit.NativeDigestSHA256 {
		return zero, errors.New("root network identity changed before conntrack")
	}
	if err := nativePeerConntrackAllStopped(ctx, root, t); err != nil {
		return zero, fmt.Errorf("native stop fence changed immediately before kernel operation: %w", err)
	}
	globalBeforeDispatch, err := nativePeerGlobalNetworkScopeSnapshot(ctx, root, t)
	if err != nil || globalBeforeDispatch != globalBefore {
		return zero, errors.New("native global network identity changed before conntrack")
	}
	report, err := nativePeerConntrackPurgeExact(ctx, run, src, dst)
	report.NativeDigestSHA256 = audit.NativeDigestSHA256
	if err != nil {
		return report, err
	}
	last, err := nativePeerBridgeAuditWithClient(ctx, root, t)
	if err != nil || last.NativeDigestSHA256 != audit.NativeDigestSHA256 {
		return report, errors.New("root network identity changed during conntrack")
	}
	if err := nativePeerConntrackAllStopped(ctx, root, t); err != nil {
		return report, fmt.Errorf("native stop fence changed after kernel operation: %w", err)
	}
	globalAfter, err := nativePeerGlobalNetworkScopeSnapshot(ctx, root, t)
	if err != nil || globalAfter != globalBefore {
		return report, errors.New("native global network identity changed during conntrack")
	}
	return report, nil
}

func nativePeerConntrackAllStopped(ctx context.Context, root incus.InstanceServer,
	t NativePeerBridgeAuditTarget) error {
	if root == nil || len(t.Boxes) == 0 {
		return errors.New("no root-owned complete stop fence")
	}
	own := root.UseProject(t.Project)
	for _, b := range t.Boxes {
		if err := ctx.Err(); err != nil {
			return err
		}
		inst, _, err := own.GetInstance(b.Name)
		if err != nil {
			return err
		}
		if err := native.ValidatePeerACLBox(inst, t.Project, t.Bridge, t.ACL, b, true); err != nil {
			return fmt.Errorf("native ACL consumer %q not stopped or identity drift: %w", b.Name, err)
		}
	}
	return nil
}
