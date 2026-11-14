package state

import (
	"context"
	"errors"
	"strings"
	"time"
)

// NativeEffect is injected by a specific, typed domain handler. There is no
// public generic command runner or automatic replay after an ambiguous failure.
type NativeEffect func(context.Context) (nativeOperationID string, err error)
type VerifyEffect func(context.Context) error

// VerifiedEffectFailure is returned only after a native operation has a
// terminal FAILED result and independent readback proves the relevant
// resource is absent. Ambiguous or residual outcomes must stay UNKNOWN.
type VerifiedEffectFailure struct{ Class string }

func (e VerifiedEffectFailure) Error() string { return e.Class }

// VerificationUncertain carries a bounded internal diagnostic class only.
// It is not native evidence and NEVER permits releasing an UNKNOWN claim.
type VerificationUncertain struct{ Class string }

func (e VerificationUncertain) Error() string { return e.Class }

// A local validation error is eligible for a terminal pre-dispatch failure
// only if the implementation proves no Incus mutation request was sent.
type PreDispatchError struct{ Class string }

func (e PreDispatchError) Error() string { return e.Class }

// An already achieved state is not a native Operation ID. Call verify again
// before acknowledging it, and never invent a native operation receipt.
var ErrNoNativeChange = errors.New("native readback proves no change necessary")

type Worker struct {
	Store  *Store
	Budget time.Duration
}

// Execute is started in its own goroutine by the internal worker, never on
// the inbound MCP context. S1 exposes no mutation tool; S2 will supply typed effects.
func (w Worker) Execute(operationID string, effect NativeEffect, verify VerifyEffect) {
	if w.Budget <= 0 {
		w.Budget = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), w.Budget)
	defer cancel()
	if err := w.Store.Claim(ctx, operationID); err != nil {
		return
	}
	nativeID, err := effect(ctx)
	if err != nil {
		// A native Operation ID contradicts even a typed pre-dispatch claim:
		// the request may have reached Incus. Preserve the original receipt,
		// leave the resource claim locked and never send a second effect.
		if nativeID != "" {
			if recordErr := w.Store.NativeEffectErrorUnknown(context.Background(), operationID, nativeID); recordErr != nil {
				_ = w.Store.Unknown(context.Background(), operationID, "DISPATCHING", "NATIVE_RECEIPT_RECORDING_UNVERIFIED")
			}
			return
		}
		var pre PreDispatchError
		if errors.As(err, &pre) {
			_ = w.Store.FailPreDispatch(context.Background(), operationID, pre.Class)
			return
		}
		if errors.Is(err, ErrNoNativeChange) {
			if verify(ctx) == nil {
				_ = w.Store.CompleteWithoutNative(context.Background(), operationID)
			} else {
				_ = w.Store.Unknown(context.Background(), operationID,
					"DISPATCHING", "NOOP_POSTCONDITION_UNVERIFIED")
			}
			return
		}
		// A native SDK error/timeout cannot prove the request was not dispatched.
		_ = w.Store.Unknown(context.Background(), operationID, "DISPATCHING", "NATIVE_OUTCOME_AMBIGUOUS")
		return
	}
	if nativeID == "" {
		_ = w.Store.Unknown(context.Background(), operationID, "DISPATCHING", "NO_NATIVE_OPERATION_RECEIPT")
		return
	}
	if err = w.Store.NativeAck(context.Background(), operationID, nativeID); err != nil {
		// The native effect may already have committed. The SQLite ACK
		// transaction may have rolled back, committed with a lost reply, or
		// remained inaccessible. Best-effort mark either possible journal
		// state UNKNOWN; NEVER retry the native effect or release its lock.
		markErr := w.Store.NativeAckUnknown(context.Background(), operationID, nativeID)
		if errors.Is(markErr, ErrState) {
			_ = w.Store.Unknown(context.Background(), operationID, "VERIFYING", "NATIVE_ACK_DURABILITY_UNVERIFIED")
		}
		return
	}
	if err = verify(ctx); err != nil {
		var proved VerifiedEffectFailure
		if errors.As(err, &proved) && proved.Class != "" {
			if w.Store.VerifiedFailed(context.Background(), operationID, proved.Class) == nil {
				return
			}
		}
		reason := "POSTCONDITION_UNVERIFIED"
		var uncertain VerificationUncertain
		if errors.As(err, &uncertain) && len(uncertain.Class) <= 100 &&
			strings.HasPrefix(uncertain.Class, "NATIVE_CREATE_") &&
			!strings.ContainsAny(uncertain.Class, "\x00\n\r") {
			reason = uncertain.Class
		}
		_ = w.Store.Unknown(context.Background(), operationID, "VERIFYING", reason)
		return
	}
	if err = w.Store.Verified(context.Background(), operationID); err != nil {
		// A lost final SQLite COMMIT acknowledgment must not be reported as
		// success without a durable receipt. If it did commit, this guarded
		// transition cannot overwrite the terminal SUCCEEDED state.
		_ = w.Store.Unknown(context.Background(), operationID, "VERIFYING", "FINAL_RECEIPT_DURABILITY_UNVERIFIED")
	}
}

func (w Worker) Submit(id string, effect NativeEffect, verify VerifyEffect) error {
	if w.Store == nil || effect == nil || verify == nil {
		return errors.New("worker not configured")
	}
	// Only a previously admitted task can be submitted. Duplicate submits
	// cannot trigger dispatch because Claim requires exactly ADMITTED.
	r, err := w.Store.Get(context.Background(), id)
	if err != nil {
		return err
	}
	if r.State != "ADMITTED" {
		return ErrState
	}
	go w.Execute(id, effect, verify)
	return nil
}
