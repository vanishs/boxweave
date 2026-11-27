package recovery

import (
	"context"
	"github.com/vanishs/boxweave/internal/state"
)

func (s Service) observeLegacyUnboundCreate(ctx context.Context, t state.RecoveryTarget, v View) (View, state.RecoveryProof) {
	if t.CreatePinStaged {
		return s.observeStagedCreate(ctx, t, v)
	}
	if t.CreatePinRequired {
		return finishRecoveryObservation(v, state.RecoveryBlocked, "CREATE_STATIC_RESERVATION_MISSING")
	}
	box, obs, e := ObserveCreate(ctx, s.Native, t.Name, t.ID)
	if e != nil || box.Project != s.Layer {
		return finishRecoveryObservation(v, state.RecoveryBlocked, "CREATE_MARKER_IDENTITY_UNPROVED")
	}
	v.ObservedNativeUUID = box.UUID
	v.NativeObservation = obs.State
	v.EvidenceSource = "SQLITE_AND_NATIVE"
	view, p := finishRecoveryObservation(v, state.RecoveryBind, "CREATE_MARKER_EXACT_IDENTITY")
	p.UUID, p.Fingerprint = box.UUID, obs.Fingerprint
	return view, p
}

func (s Service) observeLegacyDelete(ctx context.Context, t state.RecoveryTarget, v View) (View, state.RecoveryProof) {
	obs, e := s.Native.ObserveUnknownDelete(ctx, t.Name, t.UUID, t.NativeID)
	if e != nil {
		return finishRecoveryObservation(v, state.RecoveryBlocked, "DELETE_NATIVE_OBSERVATION_UNVERIFIED")
	}
	v.NativeObservation = obs.Class
	v.EvidenceSource = "SQLITE_AND_NATIVE"
	if obs.NativeID != t.NativeID || obs.Box.Project != t.Layer || obs.Box.ExpectedUUID != t.UUID || obs.Box.Name != t.Name {
		return finishRecoveryObservation(v, state.RecoveryBlocked, "DELETE_IDENTITY_UNPROVED")
	}
	if obs.Class != "ORIGINAL_TERMINAL_DELETE_VERIFIED" || obs.Box.State != "ABSENT_UNCERTAIN" {
		if t.DependentDelete {
			return finishRecoveryObservation(v, state.RecoveryBlocked, "DEPENDENT_DELETE_FINAL_PROOF_REQUIRED")
		}
		return finishRecoveryObservation(v, state.RecoveryApproval, "DELETE_TERMINAL_ABSENCE_NOT_PROVED")
	}
	view, p := finishRecoveryObservation(v, state.RecoverySettle, "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED")
	p.UUID, p.NativeID, p.Fingerprint = t.UUID, t.NativeID, obs.Fingerprint
	return view, p
}
