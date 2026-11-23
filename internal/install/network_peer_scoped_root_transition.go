package install

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/networklimits"
	"github.com/vanishs/boxweave/internal/state"
)

// NativePeerScopedRootEvidence is a ROOT-COLLECTED read-only inventory,
// never a request field, native writer argument or MCP JSON input.
// Root must independently acquire it from the exact Incus Project/bridge
// and recheck immutable SQLite original + kernel port/nft witnesses.
type NativePeerScopedRootEvidence struct {
	AllProjectNames   []string
	Observed          map[string]NativePeerGuardObservedOwner
	RootNetworkUsedBy []string
}

// NativePeerScopedRootTransition is only a *policy proposal*. It is NOT
// signed authorization, a root journal, a safe ACL migration or an instruction
// to send a native effect. Every network writer MUST reprove exact kernel
// preimage, durable original/claims, locks and postimage independently.
type NativePeerScopedRootTransition struct {
	OriginalID, Project, Bridge     string
	SourceUUID, TargetUUID          string
	SourceACLName, TargetACLName    string
	GuardBefore, GuardAfter         NativePeerGuardPlan
	SourceACLBefore, SourceACLAfter api.NetworkACLsPost
	TargetACLBefore, TargetACLAfter api.NetworkACLsPost
	UnrelatedACLsVerifiedUnchanged  int
	OriginalBootAutostart           map[string]string
	ExpectedBeforeByUUID            map[string]api.NetworkACLsPost
	ExpectedAfterByUUID             map[string]api.NetworkACLsPost
}

// NativePeerScopedRootTransitionFromEvidence joins immutable scoped v2
// SQLite read-only original evidence with a complete independent read-only
// native Incus inventory and compiles exactly one A->B dual-stack change.
// Nothing here can update ACLs, nftables, a Box or boot.autostart.
func NativePeerScopedRootTransitionFromEvidence(proof state.ScopedPeerOriginalProof,
	network *api.Network, obs NativePeerScopedRootEvidence) (NativePeerScopedRootTransition, error) {
	var zero NativePeerScopedRootTransition
	i := proof.Intent
	if proof.RecoveryOnly || i.Schema != 2 || i.SourceUUID == i.TargetUUID ||
		i.SourceUUID == "" || i.TargetUUID == "" || proof.OriginalID == "" ||
		len(proof.OriginalID) != 32 || strings.Trim(proof.OriginalID, "0123456789abcdef") != "" ||
		i.Project != proof.Inventory.Project || i.Bridge != proof.Inventory.Bridge ||
		!strings.HasPrefix(i.Project, "user-") ||
		network == nil || network.Name != "" && network.Name != i.Bridge ||
		len(proof.Inventory.Boxes) < 2 || len(proof.Inventory.Boxes) > networklimits.MaxProjectBoxes {
		return zero, errors.New("Root v2 original/bridge/complete SQLite witness untrusted")
	}
	if !nativePeerRootExactV2Diff(i, proof) {
		return zero, errors.New("Root v2 direction or old/new DROP delta differs from A->B original")
	}
	if len(obs.AllProjectNames) != len(proof.Inventory.Boxes) ||
		len(obs.Observed) != len(proof.Inventory.Boxes) {
		return zero, errors.New("native Incus project inventory missing owned Box")
	}
	claims := make([]native.PeerACLBox, 0, len(proof.Inventory.Boxes))
	identities := map[string]state.ScopedNetworkAuditBinding{}
	for _, box := range proof.Inventory.Boxes {
		if _, exists := identities[box.UUID]; exists ||
			box.UUID == "" || box.Name == "" || box.Incarnation == "" ||
			box.ACL == "" || box.IPv4 == "" || box.IPv6 == "" || box.MAC == "" {
			return zero, errors.New("SQLite native owner missing or duplicate")
		}
		acl, e := NativePeerScopedACLName(i.Project, i.Bridge, box.UUID)
		if e != nil || acl != box.ACL {
			return zero, fmt.Errorf("SQLite ACL pin not exact for %q", box.Name)
		}
		identities[box.UUID] = box
		claims = append(claims, native.PeerACLBox{Name: box.Name, UUID: box.UUID,
			MAC: box.MAC, IPv4: box.IPv4, IPv6: box.IPv6})
	}
	source, hasSource := identities[i.SourceUUID]
	target, hasTarget := identities[i.TargetUUID]
	if !hasSource || !hasTarget || source.Incarnation != i.SourceIncarnation ||
		target.Incarnation != i.TargetIncarnation {
		return zero, errors.New("v2 original references foreign or reincarnated owner")
	}
	members, e := NativePeerGuardValidateScopedInventory(i.Project, i.Bridge,
		claims, obs.AllProjectNames, obs.Observed, obs.RootNetworkUsedBy)
	if e != nil {
		return zero, fmt.Errorf("native Root project owner proof failed: %w", e)
	}
	targetConfig := NativePrepareTarget{Project: i.Project, Network: i.Bridge, ACL: source.ACL}
	before, e := NativePeerGuardFromSnapshot(targetConfig, network, members,
		proof.Before, proof.RevokedBefore)
	if e != nil {
		return zero, fmt.Errorf("invalid preimage global guard: %w", e)
	}
	after, e := NativePeerGuardFromSnapshot(targetConfig, network, members,
		proof.After, proof.RevokedAfter)
	if e != nil {
		return zero, fmt.Errorf("invalid postimage global guard: %w", e)
	}
	if e = nativePeerGuardValidateOneDirectionDelta(before, after); e != nil {
		return zero, fmt.Errorf("ambiguous or out-of-scope native guard delta: %w", e)
	}
	out := NativePeerScopedRootTransition{
		OriginalID: proof.OriginalID, Project: i.Project, Bridge: i.Bridge,
		SourceUUID: source.UUID, TargetUUID: target.UUID,
		SourceACLName: source.ACL, TargetACLName: target.ACL,
		GuardBefore: before, GuardAfter: after,
		OriginalBootAutostart: map[string]string{},
		ExpectedBeforeByUUID:  map[string]api.NetworkACLsPost{},
		ExpectedAfterByUUID:   map[string]api.NetworkACLsPost{},
	}
	for _, member := range members {
		box := identities[member.UUID]
		observation, ok := obs.Observed[box.Name]
		if !ok || observation.Instance == nil {
			return zero, errors.New("native Box disappeared during read-only policy compilation")
		}
		boot := observation.Instance.Config["boot.autostart"]
		if boot != "" && boot != "true" && boot != "false" {
			return zero, errors.New("unexpected native autostart value in guarded original")
		}
		out.OriginalBootAutostart[box.UUID] = boot
		subject := NativePeerACLSubject{UUID: box.UUID, IPv4: box.IPv4, IPv6: box.IPv6}
		config := NativePrepareTarget{Project: i.Project, Network: i.Bridge, ACL: box.ACL}
		beforeACL, err1 := NativePeerScopedACLFromPolicySnapshot(config, network,
			proof.Before, proof.RevokedBefore, subject)
		afterACL, err2 := NativePeerScopedACLFromPolicySnapshot(config, network,
			proof.After, proof.RevokedAfter, subject)
		if err1 != nil || err2 != nil {
			return zero, errors.New("native per-Box ACL compiler refused global protected snapshot")
		}
		out.ExpectedBeforeByUUID[box.UUID] = beforeACL
		out.ExpectedAfterByUUID[box.UUID] = afterACL
		switch box.UUID {
		case source.UUID:
			out.SourceACLBefore = beforeACL
			out.SourceACLAfter = afterACL
		case target.UUID:
			out.TargetACLBefore = beforeACL
			out.TargetACLAfter = afterACL
		default:
			if !reflect.DeepEqual(beforeACL, afterACL) {
				return zero, errors.New("unrelated Box ACL would be mutated by Peer original")
			}
			out.UnrelatedACLsVerifiedUnchanged++
		}
	}
	if len(out.OriginalBootAutostart) != len(claims) ||
		(reflect.DeepEqual(out.SourceACLBefore, out.SourceACLAfter) &&
			reflect.DeepEqual(out.TargetACLBefore, out.TargetACLAfter)) {
		return zero, errors.New("scoped Root plan has no exact native A/B ACL change")
	}
	return out, nil
}

// A one-direction nft delta must be EXACTLY the protected A->B original,
// never an unrelated C->D or reverse B->A action. Historical DROP
// changes must also match the same one-direction grant/revoke/regrant.
func nativePeerRootExactV2Diff(i state.ScopedPeerOriginalIntent, p state.ScopedPeerOriginalProof) bool {
	exact := func(v state.PeerGrantBinding) bool {
		return v.SourceUUID == i.SourceUUID && v.TargetUUID == i.TargetUUID &&
			v.SourceIncarnation == i.SourceIncarnation &&
			v.TargetIncarnation == i.TargetIncarnation
	}
	distinct := func(a []state.PeerGrantBinding) bool {
		seen := map[string]bool{}
		for _, v := range a {
			key := v.SourceUUID + "/" + v.TargetUUID
			if v.SourceUUID == "" || v.TargetUUID == "" || v.SourceUUID == v.TargetUUID || seen[key] {
				return false
			}
			seen[key] = true
		}
		return true
	}
	delta := func(before, after []state.PeerGrantBinding, add bool) bool {
		if !distinct(before) || !distinct(after) {
			return false
		}
		lower, upper := before, after
		if !add {
			lower, upper = after, before
		}
		if len(upper) != len(lower)+1 {
			return false
		}
		for _, v := range lower {
			if !slices.Contains(upper, v) {
				return false
			}
		}
		changed := 0
		for _, v := range upper {
			if !slices.Contains(lower, v) {
				if !exact(v) {
					return false
				}
				changed++
			}
		}
		return changed == 1
	}
	same := func(a, b []state.PeerGrantBinding) bool {
		if !distinct(a) || !distinct(b) || len(a) != len(b) {
			return false
		}
		for _, v := range a {
			if !slices.Contains(b, v) {
				return false
			}
		}
		return true
	}
	if i.Allow {
		if !delta(p.Before, p.After, true) {
			return false
		}
		if p.Regrant {
			return delta(p.RevokedBefore, p.RevokedAfter, false)
		}
		return same(p.RevokedBefore, p.RevokedAfter)
	}
	if p.Regrant || !delta(p.Before, p.After, false) {
		return false
	}
	return delta(p.RevokedBefore, p.RevokedAfter, true)
}
