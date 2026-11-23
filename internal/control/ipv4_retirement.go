package control

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/state"
)

// No caller can supply a root selector, digest or address. Only the
// installed root-authenticated original Delete witness can return a proof.
// Normal native Delete success is independent of this optional maintenance;
// any missing helper, live sibling, stale history or error leaves the fence.
func (s *Service) tryRetireDeletedAddress(ctx context.Context, original string) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	_, err := s.Store.PrepareIPv4Compaction(ctx, s.Native.Project, original, func(ctx context.Context, c state.IPv4RetirementCandidate) (string, error) {
		return install.NativePeerRootCallRetirementPreview(ctx, c.DeleteOperationID)
	})
	if err != nil {
		s.recordRetirementDeferred(ctx, original, err)
		return
	}
	err = s.Store.RetireVerifiedIPv4(ctx, s.Native.Project, original, func(ctx context.Context, c state.IPv4RetirementCandidate) (state.IPv4RetirementCandidate, string, error) {
		proof, e := install.NativePeerRootCallRetirement(ctx, c.DeleteOperationID)
		return proof.Candidate, proof.SHA256, e
	})
	if err != nil {
		s.recordRetirementDeferred(ctx, original, err)
		return
	}
	slog.Info("network_address_retirement_verified", "operation", original)
}

// SweepIPv4Retirements revisits only completed original Deletes. The
// narrow root helper may compact an exact retired DROP policy; a completed
// native effect is never replayed. No public mutation selector is accepted.
func (s *Service) SweepIPv4Retirements(ctx context.Context) {
	if s == nil || s.Store == nil {
		return
	}
	if !install.NativeRetirementSocketPresent() {
		return
	}
	status, e := s.Store.ReadIPv4RetirementStatus(ctx, s.Native.Project)
	if e != nil || status.Pending == 0 {
		return
	}
	ids, e := s.Store.ClaimPendingIPv4Retirements(ctx, s.Native.Project, 2)
	if e != nil {
		return
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		s.tryRetireDeletedAddress(ctx, id)
	}
}

// Keep diagnostic labels bounded and avoid leaking root/foreign inventories.
func retirementFailureClass(e error) string {
	switch {
	case errors.Is(e, context.DeadlineExceeded), errors.Is(e, context.Canceled):
		return "RETIREMENT_DEADLINE"
	case errors.Is(e, state.ErrResourceLocked), errors.Is(e, state.ErrState):
		return "RETIREMENT_LEDGER_NOT_QUIESCENT_OR_INELIGIBLE"
	default:
		return install.NativeRetirementFailureClass(e)
	}
}

// A separate maintenance loop keeps address retirement independent of
// seven-day Exec retention and the original Delete worker. Only one bounded
// sweep runs at a time; a failed proof leaves the address pinned.
const ipv4RetirementInterval = 15 * time.Second

func (s *Service) ipv4RetirementLoop(ctx context.Context) {
	runIPv4RetirementLoop(ctx, ipv4RetirementInterval, s.SweepIPv4Retirements)
}
func runIPv4RetirementLoop(ctx context.Context, interval time.Duration, sweep func(context.Context)) {
	for ctx.Err() == nil {
		work, cancel := context.WithTimeout(ctx, 100*time.Second)
		sweep(work)
		cancel()
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
func (s *Service) recordRetirementDeferred(ctx context.Context, original string, err error) {
	reason := retirementFailureClass(err)
	slog.Info("network_address_retirement_deferred", "operation", original, "reason", reason)
	dctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if e := s.Store.RecordIPv4RetirementDeferred(dctx, s.Native.Project, original, reason); e != nil {
		slog.Warn("network_address_retirement_diagnostic_unavailable", "operation", original)
	}
}
