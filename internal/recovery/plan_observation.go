package recovery

import (
	"context"

	"github.com/vanishs/boxweave/internal/state"
)

// PlanWitness is a redacted observation only; not a commit payload.
type PlanWitness struct {
	OriginalID        string `json:"original_id"`
	Kind              string `json:"kind"`
	Reason            string `json:"reason"`
	NativeObservation string `json:"native_observation"`
	EvidenceSource    string `json:"evidence_source"`
	ProofSHA256       string `json:"proof_sha256,omitempty"`
	ObservedBox       string `json:"observed_box,omitempty"`
}

func (s Service) InspectPlan(ctx context.Context, t state.RecoveryTarget) (PlanWitness, error) {
	// The plan adapter can never contain a recovery writer.
	if s.ReadStore == nil || s.Store != nil || s.Layer != t.Layer || s.Native == nil {
		return PlanWitness{}, state.ErrInvalid
	}
	current, e := s.ReadStore.ReadRecoveryTarget(ctx, t.Layer, t.ID)
	if e != nil || current != t {
		return PlanWitness{}, state.ErrConflict
	}
	v, p := s.observe(ctx, t, "")
	if ctx.Err() != nil {
		return PlanWitness{}, ctx.Err()
	}
	current, e = s.ReadStore.ReadRecoveryTarget(ctx, t.Layer, t.ID)
	if e != nil || current != t {
		return PlanWitness{}, state.ErrConflict
	}
	out := PlanWitness{OriginalID: t.ID, Kind: p.Kind, Reason: v.Reason, NativeObservation: v.NativeObservation, EvidenceSource: v.EvidenceSource}
	if p.UUID != "" {
		out.ObservedBox = state.SafeDiagnosticRef(p.UUID)
	}
	// Scope and exact original intent are private. The digest is evidence,
	// not authority. Missing fingerprint is never review-ready.
	if (p.Kind == state.RecoverySettle || p.Kind == state.RecoveryBind || p.Kind == state.RecoveryRepairPin) && p.Fingerprint != "" {
		if s.checkPlanRecovery(ctx, t, p) != nil {
			out.Kind = state.RecoveryBlocked
			out.Reason = "RECOVERY_PRECONDITIONS_UNVERIFIED"
			return out, nil
		}
		out.ProofSHA256 = state.RecoveryWitnessDigest(t, p)
		if out.ProofSHA256 == "" {
			return PlanWitness{}, state.ErrInvalid
		}
	}
	return out, nil
}

// No Scoped plan can fall back to a Legacy precondition checker.
func (s Service) checkPlanRecovery(ctx context.Context, t state.RecoveryTarget, p state.RecoveryProof) error {
	if s.ScopedMode {
		checker, ok := s.ReadStore.(interface {
			CheckScopedPlanRecoveryProof(context.Context, state.RecoveryTarget, state.RecoveryProof) error
		})
		if !ok {
			return state.ErrResourceLocked
		}
		return checker.CheckScopedPlanRecoveryProof(ctx, t, p)
	}
	checker, ok := s.ReadStore.(interface {
		CheckPlanRecoveryProof(context.Context, state.RecoveryTarget, state.RecoveryProof) error
	})
	if !ok {
		return state.ErrResourceLocked
	}
	return checker.CheckPlanRecoveryProof(ctx, t, p)
}
