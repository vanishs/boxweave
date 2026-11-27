package recovery

import (
	"context"
	"github.com/vanishs/boxweave/internal/state"
)

func (s Service) observeScopedPeer(ctx context.Context, t state.RecoveryTarget, v View) (View, state.RecoveryProof) {
	if s.ScopedPeerWitness == nil {
		return finishRecoveryObservation(v, state.RecoveryBlocked, "SCOPED_PEER_ROOT_PROOF_REQUIRED")
	}
	proof, e := s.ScopedPeerWitness(ctx, t.ID)
	if e != nil || !proof.RecoveryOnly || proof.OriginalID != t.ID || proof.Project != t.Layer || proof.Status != "COMPLETE_VERIFIED" || proof.VerifiedSteps != 4 {
		return finishRecoveryObservation(v, state.RecoveryBlocked, "SCOPED_PEER_ROOT_PROOF_REQUIRED")
	}
	v.NativeObservation = "ROOT_COMPLETED_KERNEL_VERIFIED"
	v.EvidenceSource = "SQLITE_AND_ROOT_JOURNAL"
	view, p := finishRecoveryObservation(v, state.RecoverySettle, "SCOPED_PEER_COMPLETED_NO_REPLAY")
	p.UUID, p.ScopedPeer = t.UUID, proof
	p.Fingerprint = fingerprint(t.ID, proof.ProofSHA256, proof.PlanSHA256, proof.FinalSHA256)
	return view, p
}
