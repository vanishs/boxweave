package install

// Root-only, exact-identity Linux conntrack invalidation prerequisite for
// future B3 peer revoke. This code has no MCP handler, no root delegation,
// and no generic host shell: only a fixed conntrack binary with constrained
// kernel tuples can be reached after independent root Incus verification.
import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
)

type NativePeerConntrackReport struct {
	NativeDigestSHA256 string `json:"native_digest_sha256"`
	PairIPv4           string `json:"pair_ipv4"`
	PairIPv6           string `json:"pair_ipv6"`
	ProbedScopes       int    `json:"probed_scopes"`
	DeletedScopes      int    `json:"deleted_scopes"`
	// No readiness claim is made before an independently checked ACL
	// native per-NIC kernel state and the full controller transaction.
	ReadyForPeer bool `json:"ready_for_peer"`
}

type nativePeerConntrackRun func(context.Context, ...string) ([]byte, error)

func nativePeerConntrackOS(ctx context.Context, args ...string) ([]byte, error) {
	// Deliberately absolute, never PATH resolution or any shell arguments.
	cmd := exec.CommandContext(ctx, "/usr/sbin/conntrack", args...)
	cmd.Env = []string{"LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	return cmd.CombinedOutput()
}

func nativePeerConntrackAddr(raw string, ipv6 bool) (netip.Addr, error) {
	a, err := netip.ParseAddr(raw)
	if err != nil || !a.IsGlobalUnicast() || a.Is4In6() ||
		(ipv6 && !a.Is6()) || (!ipv6 && !a.Is4()) {
		return netip.Addr{}, errors.New("untrusted static native peer address")
	}
	return a, nil
}

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

// A successful process exit is NOT evidence that an empty stdout means
// zero kernel connections. Require an explicit conntrack zero diagnostic or
// original source/destination tuples. Unexpected, truncated or contradictory
// output preserves UNKNOWN; never issue a blind -D after an ambiguous read.
func nativePeerConntrackList(ctx context.Context, run nativePeerConntrackRun,
	family, source, target string) (bool, error) {
	if run == nil {
		return false, errors.New("missing bounded conntrack reader")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	output, cmdErr := run(ctx, "-L", "-f", family, "-s", source, "-d", target)
	if err := ctx.Err(); err != nil {
		return false, err
	}
	found, zero, nonzero := false, false, false
	for _, raw := range strings.Split(string(output), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		pos := strings.Index(line, " flow entries have been shown")
		if pos < 0 {
			pos = strings.Index(line, " flow entries shown")
		}
		if pos >= 0 {
			prefix := strings.Fields(strings.TrimSpace(line[:pos]))
			if len(prefix) == 0 {
				return false, errors.New("unparseable conntrack summary")
			}
			count, err := strconv.Atoi(prefix[len(prefix)-1])
			if err != nil || count < 0 {
				return false, errors.New("unparseable conntrack entry count")
			}
			if count == 0 {
				zero = true
			} else {
				nonzero = true
			}
			continue
		}
		firstSource, firstTarget := "", ""
		for _, f := range strings.Fields(line) {
			if strings.HasPrefix(f, "src=") && firstSource == "" {
				firstSource = strings.TrimPrefix(f, "src=")
			}
			if strings.HasPrefix(f, "dst=") && firstTarget == "" {
				firstTarget = strings.TrimPrefix(f, "dst=")
			}
		}
		if firstSource != source || firstTarget != target {
			return false, errors.New("conntrack inventory contains unverified tuple or diagnostic")
		}
		found = true
	}
	if (zero && found) || (zero && nonzero) {
		return false, errors.New("contradictory conntrack inventory readback")
	}
	if cmdErr != nil {
		if zero && !found && !nonzero {
			// Some conntrack-tools builds return code 1 for a filtered zero.
			return false, nil
		}
		return false, fmt.Errorf("bounded conntrack inventory uncertain: %w", cmdErr)
	}
	if found {
		return true, nil
	}
	if zero && !nonzero {
		return false, nil
	}
	return false, errors.New("conntrack readback lacks explicit zero or verified original tuple")
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
