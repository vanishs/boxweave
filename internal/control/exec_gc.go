package control

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/vanishs/boxweave/internal/state"
)

const execGCInterval = 15 * time.Minute

// SweepExecRetention is an internal nonroot, bounded maintenance pass, not
// an MCP Tool, native daemon, host shell or user-selectable deletion endpoint.
// No stream is eligible until seven days after the durable final Exec witness.
func (s *Service) SweepExecRetention(ctx context.Context, now time.Time) (int, error) {
	ids, err := s.Store.ExecGCCandidates(ctx, now, 16)
	if err != nil {
		return 0, err
	}
	completed := 0
	for _, id := range ids {
		for _, stream := range []string{"stdout", "stderr"} {
			if ctx.Err() != nil {
				return completed, ctx.Err()
			}
			intent, err := s.Store.PrepareExecGC(ctx, id, stream, now)
			if errors.Is(err, state.ErrState) || errors.Is(err, state.ErrResourceLocked) {
				continue
			}
			if err != nil {
				return completed, err
			}
			// Per-attempt timeout bounds the exact native destructive request.
			// A timeout is always REVIEW_REQUIRED: NEVER retry a lost ACK.
			workCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			err = s.Native.DeleteExactRecordedOutput(workCtx, intent.Name, intent.BoxUUID,
				intent.NativeOperationID, intent.Stream, intent.Path)
			cancel()
			status := "EXPIRED"
			if err != nil {
				status = "REVIEW_REQUIRED"
			}
			if e := s.Store.FinishExecGC(ctx, intent.OperationID, intent.Stream, status); e != nil {
				return completed, e // DELETE_SENT survives, recover never replays it
			}
			if err != nil {
				slog.WarnContext(ctx, "exec_output_retention_requires_review",
					"operation_id", intent.OperationID, "stream", intent.Stream)
				continue
			}
			completed++
		}
	}
	return completed, nil
}

func (s *Service) execGCLoop(ctx context.Context) {
	run := func() {
		workCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		n, err := s.SweepExecRetention(workCtx, time.Now())
		if err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "exec_output_retention_sweep_failed", "error", err)
		} else if n > 0 {
			slog.InfoContext(ctx, "exec_output_retention_sweep_completed", "streams", n)
		}
	}
	run()
	ticker := time.NewTicker(execGCInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
