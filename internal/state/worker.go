package state

import (
	"context"
	"errors"
	"time"
)

// NativeEffect is injected by a specific, typed domain handler. There is no
// public generic command runner or automatic replay after an ambiguous failure.
type NativeEffect func(context.Context) (nativeOperationID string, err error)
type VerifyEffect func(context.Context) error

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
		// A native SDK error/timeout cannot prove the request was not dispatched.
		_ = w.Store.Unknown(context.Background(), operationID, "DISPATCHING", "NATIVE_OUTCOME_AMBIGUOUS")
		return
	}
	if nativeID == "" {
		_ = w.Store.Unknown(context.Background(), operationID, "DISPATCHING", "NO_NATIVE_OPERATION_RECEIPT")
		return
	}
	if err = w.Store.NativeAck(context.Background(), operationID, nativeID); err != nil {
		return
	}
	if err = verify(ctx); err != nil {
		_ = w.Store.Unknown(context.Background(), operationID, "VERIFYING", "POSTCONDITION_UNVERIFIED")
		return
	}
	_ = w.Store.Verified(context.Background(), operationID)
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
