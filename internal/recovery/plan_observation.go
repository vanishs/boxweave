package recovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

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
		checker, ok := s.ReadStore.(interface {
			CheckPlanRecoveryProof(context.Context, state.RecoveryTarget, state.RecoveryProof) error
		})
		if !ok || checker.CheckPlanRecoveryProof(ctx, t, p) != nil {
			out.Kind = state.RecoveryBlocked
			out.Reason = "RECOVERY_PRECONDITIONS_UNVERIFIED"
			return out, nil
		}
		b, e := json.Marshal(struct {
			Target state.RecoveryTarget
			Proof  state.RecoveryProof
		}{t, p})
		if e != nil {
			return PlanWitness{}, e
		}
		h := sha256.Sum256(append([]byte("boxweave/recover-plan-witness/v1\n"), b...))
		out.ProofSHA256 = hex.EncodeToString(h[:])
	}
	return out, nil
}
