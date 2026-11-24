package install

import (
	"context"
	"errors"
	"fmt"

	"github.com/vanishs/boxweave/internal/native"
)

// The exact original-direction tuples are selected ONLY by independently
// verified Root inventory. Caller must establish host-wide address/route
// exclusivity, a durable stage intent, and a protected deny guard. This is
// intentionally not the legacy all-stopped bidirectional purge.
func nativeScopedDirectionalConntrackPurge(ctx context.Context, run nativePeerConntrackRun, a, b native.PeerACLBox, deny func(context.Context) error) (NativePeerConntrackReport, error) {
	var out NativePeerConntrackReport
	if ctx == nil || run == nil || deny == nil || !scopedPeerUUID.MatchString(a.UUID) || !scopedPeerUUID.MatchString(b.UUID) || a.UUID == b.UUID {
		return out, errors.New("unverified directional conntrack scope")
	}
	for _, p := range []struct {
		family, src, dst string
		v6               bool
	}{{"ipv4", a.IPv4, b.IPv4, false}, {"ipv6", a.IPv6, b.IPv6, true}} {
		x, e := nativePeerConntrackAddr(p.src, p.v6)
		y, f := nativePeerConntrackAddr(p.dst, p.v6)
		if e != nil || f != nil || x == y || x.String() != p.src || y.String() != p.dst || x.Zone() != "" || y.Zone() != "" {
			return out, errors.New("invalid exact original tuple")
		}
	}
	out.PairIPv4 = a.IPv4 + "|" + b.IPv4
	out.PairIPv6 = a.IPv6 + "|" + b.IPv6
	for _, p := range []struct{ family, src, dst string }{{"ipv4", a.IPv4, b.IPv4}, {"ipv6", a.IPv6, b.IPv6}} {
		if e := ctx.Err(); e != nil {
			return out, e
		}
		if e := deny(ctx); e != nil {
			return out, e
		}
		found, e := nativePeerConntrackList(ctx, run, p.family, p.src, p.dst)
		if e != nil {
			return out, e
		}
		out.ProbedScopes++
		if !found {
			continue
		}
		if e = deny(ctx); e != nil {
			return out, e
		}
		if e = ctx.Err(); e != nil {
			return out, e
		}
		// Restrict to both address family and ORIGINAL src/dst. Never -r/-q
		// or reverse tuples, protocol-wide flush, or retry of uncertain -D.
		if _, e = run(ctx, "-D", "-f", p.family, "-s", p.src, "-d", p.dst); e != nil {
			return out, fmt.Errorf("directional conntrack effect UNKNOWN: %w", e)
		}
		out.DeletedScopes++
		found, e = nativePeerConntrackList(ctx, run, p.family, p.src, p.dst)
		if e != nil || found {
			return out, errors.New("directional conntrack postimage uncertain")
		}
		if e = deny(ctx); e != nil {
			return out, e
		}
	}
	if e := deny(ctx); e != nil {
		return out, e
	}
	return out, ctx.Err()
}
