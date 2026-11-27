package recovery

import (
	"context"
	"github.com/vanishs/boxweave/internal/state"
)

func (s Service) observeLegacyPeer(ctx context.Context, t state.RecoveryTarget, v View) (View, state.RecoveryProof) {
	if !t.PeerReady || s.PeerWitness == nil {
		return finishRecoveryObservation(v, state.RecoveryBlocked, "PEER_ROOT_COMPLETED_PROOF_REQUIRED")
	}
	proof, e := s.PeerWitness(ctx, t.ID)
	if e != nil || !state.ValidCompletedPeerProof(proof) {
		return finishRecoveryObservation(v, state.RecoveryBlocked, "PEER_ROOT_COMPLETED_PROOF_REQUIRED")
	}
	v.NativeObservation = "ROOT_COMPLETED_KERNEL_VERIFIED"
	v.EvidenceSource = "SQLITE_AND_ROOT_JOURNAL"
	view, p := finishRecoveryObservation(v, state.RecoverySettle, "ROOT_COMPLETED_POSTSTATE_VERIFIED")
	p.UUID, p.NativeID, p.Peer = t.UUID, t.NativeID, proof
	p.Fingerprint = fingerprint(t.ID, t.UUID, proof.PlanSHA256, proof.PostNativeDigestSHA256)
	return view, p
}
