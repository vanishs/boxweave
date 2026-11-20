package install

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// nativePeerRootVerifyGrantNFT checks EFFECTIVE per-NIC Incus nft chains,
// rather than accepting ACL API GET as evidence of grant convergence.
// Every resumed NIC must have all three unique chains. For each pair where
// both endpoints resumed, both endpoints must expose the exact permitted
// source -> target IPv4/IPv6 ACCEPT; reverse and sibling permissions without
// a matching complete snapshot grant must not be present anywhere.
func nativePeerRootVerifyGrantNFT(rules, project string, boxes []native.PeerACLBox,
	resumed []string, grants []state.PeerGrantBinding) error {
	if !safePrereqName.MatchString(project) || len(boxes) < 2 || len(resumed) == 0 ||
		len(rules) == 0 {
		return errors.New("incomplete native grant NIC/ACL kernel evidence")
	}
	owners := map[string]native.PeerACLBox{}
	names := map[string]string{}
	for _, b := range boxes {
		if b.UUID == "" || !safePrereqName.MatchString(b.Name) || owners[b.UUID].UUID != "" {
			return errors.New("duplicate or invalid native grant NIC scope")
		}
		owners[b.UUID] = b
		names[b.Name] = b.UUID
	}
	allowed := map[string]bool{}
	for _, g := range grants {
		a, oka := owners[g.SourceUUID]
		b, okb := owners[g.TargetUUID]
		if !oka || !okb || g.SourceUUID == g.TargetUUID ||
			a.IPv4 != g.SourceIPv4 || a.IPv6 != g.SourceIPv6 ||
			b.IPv4 != g.TargetIPv4 || b.IPv6 != g.TargetIPv6 {
			return errors.New("kernel grant binding not backed by original native NIC identity")
		}
		k := g.SourceUUID + "|" + g.TargetUUID
		if allowed[k] {
			return errors.New("duplicate directional grant in native kernel witness")
		}
		allowed[k] = true
	}
	resumedUUID := map[string]bool{}
	chains := map[string][]string{}
	for _, name := range resumed {
		id := names[name]
		if id == "" || resumedUUID[id] {
			return errors.New("resumed NIC not in native protected Project")
		}
		resumedUUID[id] = true
		for _, prefix := range []string{"in", "fwd", "out"} {
			ch := prefix + "." + project + "_" + name + ".eth0"
			re := regexp.MustCompile("(?ms)^\\s*chain " + regexp.QuoteMeta(ch) + " \\{(.*?)^\\s*\\}")
			matches := re.FindAllStringSubmatch(rules, -1)
			if len(matches) != 1 {
				return fmt.Errorf("missing or repeated native grant per-NIC chain %s", ch)
			}
			chains[id] = append(chains[id], matches[0][1])
		}
	}
	for srcID, a := range owners {
		for dstID, b := range owners {
			if srcID == dstID {
				continue
			}
			isAllowed := allowed[srcID+"|"+dstID]
			for _, family := range []struct{ prefix, src, dst string }{
				{"ip", a.IPv4, b.IPv4}, {"ip6", a.IPv6, b.IPv6},
			} {
				pattern := family.prefix + " saddr " + family.src + " " + family.prefix + " daddr " + family.dst + " accept"
				atSrc, atDst := false, false
				for id, bodies := range chains {
					for _, body := range bodies {
						if strings.Contains(body, pattern) {
							if !isAllowed {
								return fmt.Errorf("unauthorized directional kernel ACCEPT %s to %s", srcID, dstID)
							}
							if id == srcID {
								atSrc = true
							}
							if id == dstID {
								atDst = true
							}
						}
					}
				}
				if isAllowed && resumedUUID[srcID] && resumedUUID[dstID] && (!atSrc || !atDst) {
					return fmt.Errorf("no exact directional dual-NIC grant ACCEPT %s to %s", srcID, dstID)
				}
			}
		}
	}
	return nil
}
