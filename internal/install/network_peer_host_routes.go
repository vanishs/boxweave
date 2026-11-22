package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os/exec"
	"slices"
	"strings"

	"github.com/vanishs/boxweave/internal/networklimits"
)

// Linux conntrack belongs to the host network namespace, not an Incus
// Project. A root operation using only -s/-d must therefore refuse even a
// foreign *unmanaged* host route that could own one of the same IPs.
// This checks the actual root namespace routing table in addition to the
// independent Incus all-Projects inventory; it is NOT a netns/zone proof.
type nativePeerHostRoute struct {
	Dst     string `json:"dst"`
	Dev     string `json:"dev"`
	Type    string `json:"type"`
	PrefSrc string `json:"prefsrc"`
}

type nativePeerHostIPRun func(context.Context, ...string) ([]byte, error)

func nativePeerHostIPOS(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/sbin/ip", args...)
	cmd.Env = []string{"LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	return cmd.CombinedOutput()
}

func nativePeerHostRoutesParse(data []byte) ([]nativePeerHostRoute, error) {
	if len(data) == 0 || len(data) > 1024*1024 {
		return nil, errors.New("missing or excessive kernel route inventory")
	}
	var rows []nativePeerHostRoute
	if err := json.Unmarshal(data, &rows); err != nil || rows == nil || len(rows) > 4096 {
		return nil, errors.New("unparseable or unbounded kernel route inventory")
	}
	return rows, nil
}

func nativePeerHostRoutePrefix(dst string, ipv6 bool) (netip.Prefix, error) {
	if dst == "default" {
		if ipv6 {
			return netip.MustParsePrefix("::/0"), nil
		}
		return netip.MustParsePrefix("0.0.0.0/0"), nil
	}
	p, err := netip.ParsePrefix(dst)
	if err != nil {
		a, e := netip.ParseAddr(dst)
		if e != nil {
			return netip.Prefix{}, errors.New("unparseable kernel route destination")
		}
		bits := 32
		if ipv6 {
			bits = 128
		}
		p = netip.PrefixFrom(a, bits)
	}
	if !p.IsValid() || p.Addr().Is4In6() || p.Addr().Is6() != ipv6 {
		return netip.Prefix{}, errors.New("kernel route family drift")
	}
	return p, nil
}

func nativePeerHostRouteScopeFromRows(t NativePeerBridgeAuditTarget,
	v4, v6 []nativePeerHostRoute) (string, error) {
	if !safePrereqName.MatchString(t.Bridge) || t.Bridge == "." ||
		t.Bridge == ".." || len(t.Boxes) < 2 || len(t.Boxes) > networklimits.MaxProjectBoxes {
		return "", errors.New("invalid root bridge route scope")
	}
	entries := []struct {
		ip   netip.Addr
		fam6 bool
	}{}
	for _, b := range t.Boxes {
		for i, raw := range []string{b.IPv4, b.IPv6} {
			a, err := nativePeerConntrackAddr(raw, i == 1)
			if err != nil {
				return "", err
			}
			entries = append(entries, struct {
				ip   netip.Addr
				fam6 bool
			}{a, i == 1})
		}
	}
	for _, family := range []struct {
		rows []nativePeerHostRoute
		v6   bool
	}{{v4, false}, {v6, true}} {
		if len(family.rows) == 0 || len(family.rows) > 4096 {
			return "", errors.New("missing or unbounded host route family")
		}
		covered := map[netip.Addr]int{}
		for _, route := range family.rows {
			if route.Dst == "" || strings.Contains(route.Dev, "/") || strings.ContainsRune(route.Dev, 0) {
				return "", errors.New("ambiguous native host route")
			}
			prefix, err := nativePeerHostRoutePrefix(route.Dst, family.v6)
			if err != nil {
				return "", err
			}
			for _, peer := range entries {
				if peer.fam6 != family.v6 {
					continue
				}
				if route.PrefSrc != "" && route.PrefSrc == peer.ip.String() &&
					route.Dev != t.Bridge {
					return "", errors.New("foreign native host preferred source duplicates peer IP")
				}
				if !prefix.Contains(peer.ip) || prefix.Bits() == 0 {
					continue
				}
				if route.Dev != t.Bridge || (route.Type != "" && route.Type != "unicast") {
					return "", fmt.Errorf("non-owner native host route overlaps protected peer address")
				}
				covered[peer.ip]++
			}
		}
		for _, peer := range entries {
			if peer.fam6 == family.v6 && covered[peer.ip] != 1 {
				return "", errors.New("protected peer lacks one exclusive root bridge route")
			}
		}
	}
	slices.SortFunc(v4, func(a, b nativePeerHostRoute) int {
		return strings.Compare(a.Dst+"|"+a.Dev+"|"+a.Type+"|"+a.PrefSrc,
			b.Dst+"|"+b.Dev+"|"+b.Type+"|"+b.PrefSrc)
	})
	slices.SortFunc(v6, func(a, b nativePeerHostRoute) int {
		return strings.Compare(a.Dst+"|"+a.Dev+"|"+a.Type+"|"+a.PrefSrc,
			b.Dst+"|"+b.Dev+"|"+b.Type+"|"+b.PrefSrc)
	})
	data, err := json.Marshal(struct {
		Bridge string
		IPv4   []nativePeerHostRoute
		IPv6   []nativePeerHostRoute
	}{t.Bridge, v4, v6})
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(append([]byte("boxweave/b2/root-host-routes/v1\n"), data...))
	return hex.EncodeToString(h[:]), nil
}

func nativePeerHostRoutesWithRun(ctx context.Context, t NativePeerBridgeAuditTarget,
	run nativePeerHostIPRun) (string, error) {
	if run == nil || len(t.Boxes) < 2 {
		return "", errors.New("missing host routing executor or complete peer ledger")
	}
	var family [2][]nativePeerHostRoute
	for i, af := range []string{"-4", "-6"} {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		out, err := run(ctx, "-j", af, "route", "show", "table", "all")
		if err != nil {
			return "", fmt.Errorf("native host routing inventory unavailable: %w", err)
		}
		family[i], err = nativePeerHostRoutesParse(out)
		if err != nil {
			return "", err
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return nativePeerHostRouteScopeFromRows(t, family[0], family[1])
}

func nativePeerHostRoutes(ctx context.Context, t NativePeerBridgeAuditTarget) (string, error) {
	return nativePeerHostRoutesWithRun(ctx, t, nativePeerHostIPOS)
}
