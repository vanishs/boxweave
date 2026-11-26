package recovery

import (
	"context"
	"errors"
	"fmt"
	"github.com/vanishs/boxweave/internal/state"
	"time"
)

// ReconcileReviewed is called only by the kernel-authenticated restricted
// local worker. No proof arrives from the caller. Reuse the existing
// observer and the exact same transaction/finalizer as MCP reconciliation.
func (s Service) ReconcileReviewed(ctx context.Context, r state.RecoveryRequest, recheck func(context.Context) error) (state.RecoveryRecord, error) {
	var zero state.RecoveryRecord
	if s.Store == nil || s.ReadStore != nil || s.Native == nil || r.Review == nil || r.LocalReviewer == "" || r.Layer != s.Layer || recheck == nil {
		return zero, state.ErrInvalid
	}
	old, e := s.Store.ReplayRecovery(ctx, r)
	if e == nil {
		return old, nil
	}
	if !errors.Is(e, state.ErrNotFound) {
		return zero, e
	}
	if time.Now().UnixNano() >= r.Review.ExpiresNS {
		return zero, fmt.Errorf("reviewed recovery: review expired: %w", state.ErrConflict)
	}
	o, l, e := s.Store.ReadReviewedRecoveryEvidence(ctx, r.Layer, r.OriginalID, r.Review.Mode)
	if e != nil || o.Digest != r.Review.OriginalSHA256 || l.ObservationDigest() != r.Review.LedgerSHA256 {
		return zero, fmt.Errorf("reviewed recovery: original or ledger evidence changed: %w", state.ErrConflict)
	}
	_, proof := s.observe(ctx, o.Target, "")
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	if state.RecoveryWitnessDigest(o.Target, proof) != r.Review.WitnessSHA256 {
		return zero, fmt.Errorf("reviewed recovery: native witness changed: %w", state.ErrConflict)
	}
	if e = recheck(ctx); e != nil {
		return zero, e
	}
	return s.commitRecovery(ctx, r, o.Target, proof)
}
