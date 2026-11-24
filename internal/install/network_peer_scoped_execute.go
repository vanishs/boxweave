package install

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type scopedPeerOriginalReader func(context.Context) (state.ScopedPeerOriginalProof, error)
type scopedPeerGlobalWitness func(context.Context, state.ScopedPeerOriginalProof, nativePeerScopedSDKSnapshot) (string, error)

// Composition of the actual bounded native operations. The caller MUST
// hold the installed Root/lifecycle mutex and acquire the fixed Root SDK,
// durable original reader, host scope witness and root-owned journal.
// Not an installed dispatcher until policy boot publication/recovery is wired.
func nativeScopedPeerExecute(ctx context.Context, dir string, uid uint32, host string,
	read scopedPeerOriginalReader, r nativePeerScopedRootReader, driver nativePeerScopedACLDriver,
	sysfs string, guard nativePeerGuardWitness, batch nativePeerGuardBatch, conntrack nativePeerConntrackRun, global scopedPeerGlobalWitness) (state.ScopedPeerRootReceipt, error) {
	var zero state.ScopedPeerRootReceipt
	if ctx == nil || read == nil || r == nil || driver == nil || guard == nil || batch == nil || conntrack == nil || global == nil || !nativePeerGuardSHA256Valid(host) {
		return zero, errors.New("missing scoped Root execution capability")
	}
	proof, e := read(ctx)
	if e != nil || proof.RecoveryOnly {
		return zero, errors.New("unverified dispatch original")
	}
	base, e := nativePeerScopedRootReadSnapshot(ctx, r, proof)
	if e != nil {
		return zero, e
	}
	p, e := NativePeerScopedRootTransitionFromEvidence(proof, base.Network, base.Evidence)
	if e != nil {
		return zero, e
	}
	glob, e := global(ctx, proof, base)
	if e != nil || !nativePeerGuardSHA256Valid(glob) {
		return zero, fmt.Errorf("missing host-wide scope witness: %w", e)
	}
	scope := scopedPeerEffectJournalScope{HostSHA256: host, LedgerSHA256: state.ScopedPeerProofSHA256(proof), NativeSHA256: nativePrepareStateDigest("scoped-peer-native-preimage-v2", base, glob)}
	var latest nativePeerScopedSDKSnapshot
	prefix := 0
	witness := func(ctx context.Context, i int) (string, error) {
		a, e := read(ctx)
		if e != nil || !reflect.DeepEqual(a, proof) {
			return "", errors.New("original changed before prefix")
		}
		n, h, e := nativeScopedPeerPrefix(ctx, r, proof, p, base, sysfs, guard, i)
		if e != nil {
			return "", e
		}
		g, e := global(ctx, proof, n)
		if e != nil || g != glob {
			return "", errors.New("host-wide scope drift")
		}
		b, e := read(ctx)
		if e != nil || !reflect.DeepEqual(b, proof) {
			return "", errors.New("original changed after prefix")
		}
		latest = n
		prefix = i
		return nativePrepareStateDigest("scoped-peer-full-prefix-v2", h, g, scope.LedgerSHA256), nil
	}
	steps, e := nativePeerScopedRootEffectSequence(p)
	if e != nil {
		return zero, e
	}
	box := func(id string) native.PeerACLBox {
		for _, b := range proof.Inventory.Boxes {
			if b.UUID == id {
				return native.PeerACLBox{Name: b.Name, UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6}
			}
		}
		return native.PeerACLBox{}
	}
	apply := func(ctx context.Context, s nativePeerScopedEffectStep) error {
		if prefix >= len(steps) || s != steps[prefix] {
			return errors.New("unexpected scoped effect stage")
		}
		switch s.Kind {
		case nativePeerScopedACLSource, nativePeerScopedACLTarget:
			o, ok := latest.Evidence.Observed[box(s.OwnerUUID).Name]
			if !ok {
				return errors.New("missing native ACL owner")
			}
			receipt, e := nativePeerScopedACLReplaceOnce(ctx, p, s.OwnerUUID, o, guard, driver)
			if e != nil || !receipt.Verified {
				return errors.New("scoped ACL effect uncertain")
			}
		case nativePeerScopedGuardBatch:
			receipt, e := nativePeerGuardReplaceOnce(ctx, p.GuardBefore, p.GuardAfter, guard, batch)
			if e != nil || !receipt.KernelVerified {
				return errors.New("scoped Guard effect uncertain")
			}
		case nativePeerScopedExactConntrack:
			deny := p.GuardBefore
			if !p.IntentAllow {
				deny = p.GuardAfter
			}
			report, e := nativeScopedDirectionalConntrackPurge(ctx, conntrack, box(p.SourceUUID), box(p.TargetUUID), func(ctx context.Context) error { return guard(ctx, deny) })
			if e != nil || report.ProbedScopes != 2 {
				return errors.New("scoped conntrack effect uncertain")
			}
		default:
			return errors.New("unsupported scoped Peer effect")
		}
		return nil
	}
	rec, e := nativeScopedPeerEffectsOnce(ctx, dir, uid, p, scope, witness, apply)
	if e != nil || rec.Status != "COMPLETE_VERIFIED" {
		return zero, errors.New("scoped Root original outcome UNKNOWN")
	}
	return state.ScopedPeerRootReceipt{OriginalID: proof.OriginalID, Project: p.Project, Bridge: p.Bridge, Status: rec.Status, ProofSHA256: scope.LedgerSHA256, PlanSHA256: rec.PlanSHA256, FinalSHA256: rec.FinalSHA256, VerifiedSteps: rec.VerifiedSteps}, nil
}
