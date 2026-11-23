package recovery

import (
	"context"
	"errors"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// Native health is an explicit read-only inspection of a completed original
// Create, not a reconciliation or privilege to alter the NIC.
func (s Service) observeHealthyCreatePin(ctx context.Context, t state.RecoveryTarget, v View) (View, state.RecoveryProof) {
	block := func(reason string) (View, state.RecoveryProof) {
		v.Disposition, v.Reason = state.RecoveryBlocked, reason
		return v, state.RecoveryProof{Kind: state.RecoveryBlocked, Reason: reason}
	}
	pin, e := s.Store.ReadOriginalCreatePin(ctx, t)
	if e != nil {
		return block("CREATE_PIN_LEDGER_CONFLICT")
	}
	n, ok := s.Native.(interface {
		VerifyPinnedDeleteNIC(context.Context, string, string, string, string, string, string, string) error
		ObserveOriginalCreateStaticPeerBinding(context.Context, string, string, string) (native.VerifiedStaticPeerBinding, error)
	})
	if !ok {
		return block("CREATE_NATIVE_PIN_OBSERVER_UNAVAILABLE")
	}
	if e = n.VerifyPinnedDeleteNIC(ctx, t.Name, t.UUID, pin.Bridge, pin.ACL, pin.MAC, pin.IPv4, pin.IPv6); e != nil {
		if errors.Is(e, native.ErrUnsafeNetwork) {
			return block("CREATE_NATIVE_PIN_DRIFT")
		}
		return block("CREATE_NATIVE_PIN_UNVERIFIED")
	}
	observed, e := n.ObserveOriginalCreateStaticPeerBinding(ctx, t.Name, t.UUID, t.ID)
	if e != nil {
		return block("CREATE_NATIVE_PIN_UNVERIFIED")
	}
	if state.VerifiedCreateBinding(observed) != pin {
		return block("CREATE_NATIVE_PIN_DRIFT")
	}
	v.NativeObservation, v.EvidenceSource = "EXACT_NATIVE_PIN_MATCHES_LEDGER", "SQLITE_AND_NATIVE"
	v.Disposition, v.Reason = state.RecoveryNotNeeded, "CREATE_PIN_LEDGER_COMPLETE"
	return v, state.RecoveryProof{Kind: state.RecoveryNotNeeded, Reason: v.Reason}
}
