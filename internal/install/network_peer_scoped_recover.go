package install

import (
	"context"
	"errors"
	"reflect"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// Completed UNKNOWN recovery is strictly read-only: no stage replay,
// no ACL/Guard update, and no conntrack deletion. A published root policy
// and complete journal must bind the exact still-locked original.
func nativeScopedPeerRecoverWithDriver(ctx context.Context, dir string, uid uint32, r nativePeerFreshJournalRecord, initial NativePeerGuardPlan, read scopedPeerOriginalReader, driver nativePeerScopedRootReader, sysfs string, guard nativePeerGuardWitness, global scopedPeerGlobalWitness) (state.ScopedPeerRootReceipt, error) {
	var zero state.ScopedPeerRootReceipt
	if ctx == nil || read == nil || driver == nil || guard == nil || global == nil {
		return zero, errors.New("missing recovery witness")
	}
	proof, e := read(ctx)
	if e != nil || !proof.RecoveryOnly {
		return zero, errors.New("not an UNKNOWN scoped original")
	}
	current, last, e := scopedPeerBootCurrent(dir, uid, r, initial, "")
	if e != nil || last.OriginalID != proof.OriginalID || last.ProofSHA256 != state.ScopedPeerProofSHA256(proof) {
		return zero, errors.New("UNKNOWN original has no complete published policy")
	}
	// Compiler/read-only SDK functions are shared. Never pass this local
	// normalized copy to an executor, even if every postimage matches.
	p := proof
	p.RecoveryOnly = false
	claims := []native.PeerACLBox{}
	for _, b := range p.Inventory.Boxes {
		claims = append(claims, native.PeerACLBox{Name: b.Name, UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6})
	}
	var first nativePeerScopedSDKSnapshot
	var globalFirst string
	for i := 0; i < 2; i++ {
		s, e := nativePeerScopedRootReadSnapshot(ctx, driver, p)
		if e != nil {
			return zero, e
		}
		plan, e := NativePeerScopedRootTransitionFromEvidence(p, s.Network, s.Evidence)
		if e != nil || nativePrepareStateDigest("scoped-peer-full-sequence-v2", plan) != last.PlanSHA256 || plan.GuardAfter != current {
			return zero, errors.New("recovery plan differs from original")
		}
		plan.ExpectedBeforeByUUID = plan.ExpectedAfterByUUID
		if _, e = NativePeerScopedVerifyObservedACLPreimage(plan, s.Evidence); e != nil {
			return zero, e
		}
		if e = nativePeerGuardKernelPortsExactAt(ctx, sysfs, r.Bridge, claims, s.Evidence.Observed); e != nil {
			return zero, e
		}
		if e = guard(ctx, current); e != nil {
			return zero, e
		}
		g, e := global(ctx, proof, s)
		if e != nil || !nativePeerGuardSHA256Valid(g) {
			return zero, errors.New("recovery global scope unverified")
		}
		if i == 0 {
			first = s
			globalFirst = g
		} else if !reflect.DeepEqual(first, s) || g != globalFirst {
			return zero, errors.New("recovery native postimage drift")
		}
	}
	again, e := read(ctx)
	if e != nil || !reflect.DeepEqual(again, proof) {
		return zero, errors.New("UNKNOWN original changed")
	}
	final, latest, e := scopedPeerBootCurrent(dir, uid, r, initial, "")
	if e != nil || final != current || !reflect.DeepEqual(last, latest) {
		return zero, errors.New("recovery durable policy changed")
	}
	return state.ScopedPeerRootReceipt{OriginalID: proof.OriginalID, Project: r.Project, Bridge: r.Bridge, Status: "COMPLETE_VERIFIED", ProofSHA256: last.ProofSHA256, PlanSHA256: last.PlanSHA256, FinalSHA256: last.FinalSHA256, VerifiedSteps: 4, RecoveryOnly: true}, ctx.Err()
}

func nativeScopedPeerRecover(ctx context.Context, db *state.Store, id string) (state.ScopedPeerRootReceipt, error) {
	var zero state.ScopedPeerRootReceipt
	if db == nil || !db.IsNetworkAuditReadOnly() || !scopedLifecycleID.MatchString(id) {
		return zero, errors.New("unverified recovery original")
	}
	r, initial, e := nativePeerFreshVerifiedPlan()
	if e != nil {
		return zero, e
	}
	if e = nativePeerFreshQuarantineReceipt(r); e != nil {
		return zero, e
	}
	c, e := prepareClient(ctx)
	if e != nil {
		return zero, e
	}
	c = c.UseProject("default")
	return nativeScopedPeerRecoverWithDriver(ctx, nativePeerFreshJournalDir, 0, r, initial,
		func(ctx context.Context) (state.ScopedPeerOriginalProof, error) {
			return db.ReadRootScopedPeerUnknownOriginal(ctx, id, r.Project, r.Bridge)
		},
		nativePeerScopedIncusReader{root: c}, "/sys/class/net", NativePeerGuardKernelReadback,
		func(ctx context.Context, p state.ScopedPeerOriginalProof, s nativePeerScopedSDKSnapshot) (string, error) {
			return nativeScopedPeerGlobal(ctx, c, db, r, p, s)
		})
}
