package install

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

// Planning only: no native dispatch, root permission, conntrack execution,
// rollback or operation completion. A fresh-only v2 Root worker must fsync
// an UNKNOWN intent BEFORE each irreversible stage under the same locks.
type nativePeerScopedEffectKind string

const (
	nativePeerScopedACLSource      nativePeerScopedEffectKind = "ACL_SOURCE_CAS"
	nativePeerScopedACLTarget      nativePeerScopedEffectKind = "ACL_TARGET_CAS"
	nativePeerScopedGuardBatch     nativePeerScopedEffectKind = "GUARD_NFT_BATCH"
	nativePeerScopedExactConntrack nativePeerScopedEffectKind = "EXACT_A_B_CONNTRACK"
)

type nativePeerScopedEffectStep struct {
	Kind                 nativePeerScopedEffectKind
	OwnerUUID            string
	GuardMustMatchSHA256 string
}

// Exact transition contract for a fresh scoped Project. ACL edits never
// touch C/D, stop Box processes, or alter boot.autostart.
// GRANT: two scoped ACLs prepared under nft DROP -> exact A/B purge -> nft grant.
// REVOKE: nft DROP -> exact A/B purge -> two scoped ACL drops.
// A crash after ANY stage must leave one journaled original UNKNOWN.
func nativePeerScopedRootEffectSequence(p NativePeerScopedRootTransition) ([]nativePeerScopedEffectStep, error) {
	if len(p.OriginalID) != 32 || strings.Trim(p.OriginalID, "0123456789abcdef") != "" ||
		!scopedPeerUUID.MatchString(p.SourceUUID) || !scopedPeerUUID.MatchString(p.TargetUUID) ||
		p.SourceUUID == p.TargetUUID ||
		p.SourceACLName == "" || p.TargetACLName == "" ||
		p.SourceACLName == p.TargetACLName {
		return nil, errors.New("unknown fresh v2 original or A/B ACL ownership")
	}
	if err := nativePeerGuardValidateOneDirectionDelta(p.GuardBefore, p.GuardAfter); err != nil {
		return nil, err
	}
	// A valid one-direction delta can be either addition or removal. Bind
	// its polarity to the original before selecting the protected image.
	var beforeRules, afterRules [][]any
	if json.Unmarshal([]byte(p.GuardBefore.ExpectedKernelJSON), &beforeRules) != nil ||
		json.Unmarshal([]byte(p.GuardAfter.ExpectedKernelJSON), &afterRules) != nil ||
		(p.IntentAllow && len(afterRules) != len(beforeRules)+4) ||
		(!p.IntentAllow && len(beforeRules) != len(afterRules)+4) {
		return nil, errors.New("guard delta polarity differs from original")
	}
	if len(p.ExpectedBeforeByUUID) < 2 ||
		len(p.ExpectedBeforeByUUID) != len(p.ExpectedAfterByUUID) ||
		len(p.OriginalBootAutostart) != len(p.ExpectedBeforeByUUID) ||
		p.UnrelatedACLsVerifiedUnchanged != len(p.ExpectedBeforeByUUID)-2 {
		return nil, errors.New("incomplete whole-tenant scoped policy")
	}
	for uuid, b := range p.ExpectedBeforeByUUID {
		a, ok := p.ExpectedAfterByUUID[uuid]
		boot, hasBoot := p.OriginalBootAutostart[uuid]
		name, nameErr := NativePeerScopedACLName(p.Project, p.Bridge, uuid)
		if !ok || !hasBoot || (boot != "" && boot != "true" && boot != "false") ||
			nameErr != nil || b.Name != name || b.Name != a.Name {
			return nil, errors.New("orphan native ACL during v2 planning")
		}
		if uuid != p.SourceUUID && uuid != p.TargetUUID && !reflect.DeepEqual(a, b) {
			return nil, errors.New("C/D policy would be disrupted")
		}
		if (uuid == p.SourceUUID || uuid == p.TargetUUID) && reflect.DeepEqual(a, b) {
			return nil, errors.New("missing A/B native ACL transition")
		}
	}
	if p.ExpectedBeforeByUUID[p.SourceUUID].Name != p.SourceACLName ||
		p.ExpectedBeforeByUUID[p.TargetUUID].Name != p.TargetACLName {
		return nil, errors.New("native original ACL source/target mixed")
	}
	deny := p.GuardBefore.DigestSHA256
	if !p.IntentAllow {
		deny = p.GuardAfter.DigestSHA256
	}
	a := nativePeerScopedEffectStep{Kind: nativePeerScopedACLSource, OwnerUUID: p.SourceUUID, GuardMustMatchSHA256: deny}
	b := nativePeerScopedEffectStep{Kind: nativePeerScopedACLTarget, OwnerUUID: p.TargetUUID, GuardMustMatchSHA256: deny}
	g := nativePeerScopedEffectStep{Kind: nativePeerScopedGuardBatch, GuardMustMatchSHA256: p.GuardBefore.DigestSHA256}
	c := nativePeerScopedEffectStep{Kind: nativePeerScopedExactConntrack, GuardMustMatchSHA256: p.GuardAfter.DigestSHA256}
	if p.IntentAllow {
		c.GuardMustMatchSHA256 = deny
		return []nativePeerScopedEffectStep{a, b, c, g}, nil
	}
	// On revoke, the guard must already be DROP before conntrack or ACL writes.
	return []nativePeerScopedEffectStep{g, c, a, b}, nil
}
