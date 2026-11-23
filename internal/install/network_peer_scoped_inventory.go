package install

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/networklimits"
)

// NativePeerGuardObservedOwner is an SDK read-only snapshot, never an MCP
// argument. The eventual privileged caller must independently obtain these
// values from the root Incus socket and immutable SQLite native binding.
type NativePeerGuardObservedOwner struct {
	Instance *api.Instance
	ACL      *api.NetworkACL
	ACLTag   string
}

// nativePeerGuardOwnedURL accepts only one exact native Project resource,
// never a host path, foreign Project, subresource, or extra query parameter.
func nativePeerGuardOwnedURL(raw, kind, name, project string) bool {
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Scheme != "" || u.Host != "" ||
		u.Fragment != "" || u.RawFragment != "" ||
		u.Path != "/1.0/"+kind+"/"+name {
		return false
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(query) != 1 || len(query["project"]) != 1 ||
		query.Get("project") != project {
		return false
	}
	return true
}

// NativePeerGuardValidateScopedInventory is a fail-closed, READ-ONLY native
// provenance snapshot check. No ACL, nftables, Incus or autostart writes.
// It binds a complete all-Box Project inventory to dedicated ACL ownership
// before compiling any new A/B policy. The caller MUST independently prove:
// root Incus and SQLite identity, the host-wide bridge/ACL UsedBy inventory,
// absence of raw/foreign bridge ports and a protected original/lock journal.
func NativePeerGuardValidateScopedInventory(project, bridge string,
	claims []native.PeerACLBox, allProjectNames []string,
	observed map[string]NativePeerGuardObservedOwner,
	rootNetworkUsedBy []string) ([]NativePeerGuardMember, error) {
	if !strings.HasPrefix(project, "user-") ||
		!safePrereqName.MatchString(project) ||
		!safePrereqName.MatchString(bridge) ||
		len(claims) == 0 || len(claims) > networklimits.MaxProjectBoxes ||
		len(allProjectNames) != len(claims) ||
		len(observed) != len(claims) ||
		len(rootNetworkUsedBy) == 0 {
		return nil, errors.New("guard scope or complete root native inventory unavailable")
	}
	owned := make(map[string]native.PeerACLBox, len(claims))
	ips4, ips6, macs, ids := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, claim := range claims {
		if claim.Name == "" || !scopedPeerUUID.MatchString(claim.UUID) ||
			owned[claim.Name].Name != "" ||
			ips4[claim.IPv4] || ips6[claim.IPv6] || macs[claim.MAC] || ids[claim.UUID] {
			return nil, errors.New("duplicate or invalid exact guard native lease ownership")
		}
		owned[claim.Name] = claim
		ips4[claim.IPv4], ips6[claim.IPv6], macs[claim.MAC], ids[claim.UUID] = true, true, true, true
	}
	names := slices.Clone(allProjectNames)
	slices.Sort(names)
	for i, name := range names {
		if i > 0 && names[i-1] == name {
			return nil, errors.New("duplicate root Project instance")
		}
		if _, ok := owned[name]; !ok {
			return nil, errors.New("root Project has unclaimed native instance")
		}
	}
	// Profiles may own a network in Incus. Only the exact owned default
	// profile is admitted, alongside exact owned instance references.
	// Unknown consumers are not treated as harmless labels.
	usedBySeen := map[string]bool{}
	for _, raw := range rootNetworkUsedBy {
		if usedBySeen[raw] {
			return nil, errors.New("duplicate native bridge consumer")
		}
		usedBySeen[raw] = true
		if nativePeerGuardOwnedURL(raw, "profiles", "default", project) {
			continue
		}
		allowed := false
		for _, name := range names {
			if nativePeerGuardOwnedURL(raw, "instances", name, project) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("foreign or unrecognized native bridge consumer: %q", raw)
		}
	}

	members := make([]NativePeerGuardMember, 0, len(names))
	aclNames := map[string]bool{}
	for _, name := range names {
		b := owned[name]
		o, ok := observed[name]
		if !ok || o.Instance == nil || o.ACL == nil || o.ACLTag == "" {
			return nil, fmt.Errorf("missing exact root readback for %q", name)
		}
		aclName, err := NativePeerScopedACLName(project, bridge, b.UUID)
		if err != nil || aclNames[aclName] {
			return nil, errors.New("duplicate or untrusted per-Box ACL name")
		}
		aclNames[aclName] = true
		if err = native.ValidatePeerACLBox(o.Instance, project, bridge, aclName, b, false); err != nil {
			return nil, fmt.Errorf("native peer identity/anti-spoof/lease check %q: %w", name, err)
		}
		if (o.ACL.Project != "" && o.ACL.Project != project) ||
			o.ACL.Name != aclName || len(o.ACL.UsedBy) != 1 ||
			!nativePeerGuardOwnedURL(o.ACL.UsedBy[0], "instances", name, project) {
			return nil, fmt.Errorf("native peer ACL not exclusively used by pinned Box %q", name)
		}
		members = append(members, NativePeerGuardMember{
			UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6,
		})
	}
	return members, nil
}
