package control

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type waitReadyInput struct {
	BoxID          string   `json:"box_id"`
	IdempotencyKey string   `json:"idempotency_key"`
	Checks         []string `json:"checks,omitempty" jsonschema:"Requested conditions: running, exec, ipv4, ipv6. Default running and exec."`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty" jsonschema:"Deadline in seconds (1..600); default 300."`
	WorkRef        string   `json:"work_ref,omitempty"`
	AgentRef       string   `json:"agent_ref,omitempty"`
}

func (s *Service) registerReadinessTool(server *mcp.Server) {
	addStrictTool(server, &mcp.Tool{Name: state.ReadinessAction, Description: "Admit a bounded asynchronous readiness observation for an exact owned Box. Default checks running and fixed timeout-bounded native Exec with zero guest exit; optional verified NIC IPv4/IPv6 assignment. Poll operation_get for readiness; not guest-declared Ready or internet reachability. No caller command or output.", Annotations: mutationAnnotations(false)},
		func(ctx context.Context, req *mcp.CallToolRequest, in waitReadyInput) (*mcp.CallToolResult, state.Receipt, error) {
			intent, e := state.NormalizeReadinessIntent(state.ReadinessIntent{BoxID: in.BoxID, Checks: in.Checks, TimeoutSeconds: in.TimeoutSeconds})
			if e != nil {
				return nil, state.Receipt{}, e
			}
			prior, found, e := s.replayNative(ctx, req, state.ReadinessAction, in.IdempotencyKey, in.BoxID, in.WorkRef, in.AgentRef, intent)
			if e != nil {
				return nil, state.Receipt{}, e
			}
			if found {
				return &mcp.CallToolResult{}, prior, nil
			}
			box, e := s.readOwned(ctx, req, in.BoxID)
			if e != nil {
				return nil, state.Receipt{}, e
			}
			rec, e := s.admitNative(ctx, req, state.ReadinessAction, in.IdempotencyKey, box.UUID, box.Incarnation, "box:"+box.Name, in.WorkRef, in.AgentRef, intent)
			if e != nil {
				return nil, state.Receipt{}, e
			}
			if !rec.Duplicate {
				go s.runReadiness(rec, box, intent)
			}
			return &mcp.CallToolResult{}, rec, nil
		})
}

func (s *Service) observeReady(ctx context.Context, box state.OwnedBox, i state.ReadinessIntent) (state.ReadinessSample, native.ReadyObservation, error) {
	sample := state.NewReadinessSample(i)
	current, e := s.Store.OwnedBox(ctx, box.Project, box.UUID)
	if e != nil {
		return sample, native.ReadyObservation{}, e
	}
	if current != box {
		return sample, native.ReadyObservation{}, native.ErrNotOwned
	}
	network := slices.Contains(i.Checks, "ipv4") || slices.Contains(i.Checks, "ipv6")
	obs, e := s.Native.ObserveReadiness(ctx, box.Name, box.UUID, network)
	if e != nil {
		return sample, obs, e
	}
	sample.CheckedAt = time.Now().UnixNano()
	if sample.Running != "not_checked" {
		sample.Running = "not_ready"
		if obs.Running {
			sample.Running = "ready"
		}
	}
	if !obs.Running || obs.NetworkObserved {
		if sample.IPv4Assigned != "not_checked" {
			sample.IPv4Assigned = "not_ready"
		}
		if sample.IPv6Assigned != "not_checked" {
			sample.IPv6Assigned = "not_ready"
		}
	}
	for _, a := range obs.Addresses {
		if !slices.Contains(i.Checks, a.Family) {
			continue
		}
		ok, e := s.Store.ReadinessAddressAllowed(ctx, box.Project, box.UUID, box.Incarnation, a.Bridge, a.Family, a.Address)
		if e != nil {
			return sample, obs, e
		}
		if !ok {
			continue
		}
		if a.Family == "ipv4" {
			sample.IPv4Assigned = "ready"
		}
		if a.Family == "ipv6" {
			sample.IPv6Assigned = "ready"
		}
	}
	return sample, obs, nil
}

func readinessConditions(sample state.ReadinessSample, obs native.ReadyObservation, exec bool) bool {
	// Exec and addresses depend on a running native instance even when
	// running was not explicitly selected as a displayed condition.
	if !obs.Running {
		return false
	}
	if exec {
		sample.ExecReady = "not_checked"
	}
	return sample.AllReady()
}

func (s *Service) runReadiness(rec state.Receipt, box state.OwnedBox, intent state.ReadinessIntent) {
	deadline := time.Unix(0, rec.CreatedAt).Add(time.Duration(intent.TimeoutSeconds) * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	sample := state.NewReadinessSample(intent)
	exec := slices.Contains(intent.Checks, "exec")
	save := func(from, to, reason string) error {
		// The durable verdict must survive even when the observation deadline
		// expired; never reuse the cancelled native request context for SQLite.
		c, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		return s.Store.SaveReadiness(c, rec.ID, from, to, reason, sample)
	}
	for {
		if ctx.Err() != nil {
			_ = save("OBSERVING", "TIMED_OUT", "READINESS_DEADLINE_ELAPSED")
			return
		}
		if e := s.Store.ReadinessActorActive(ctx, rec.ID); e != nil {
			_ = save("OBSERVING", "FAILED", "READINESS_AUTH_OR_STATE_UNAVAILABLE")
			return
		}
		next, obs, e := s.observeReady(ctx, box, intent)
		if e == nil {
			sample = next
		}
		if errors.Is(e, state.ErrNotFound) || errors.Is(e, native.ErrNotOwned) {
			_ = save("OBSERVING", "FAILED", "EXACT_BOX_IDENTITY_LOST")
			return
		}
		if errors.Is(e, native.ErrNativeUnsupported) || errors.Is(e, native.ErrUnsafeNetwork) {
			_ = save("OBSERVING", "FAILED", "READINESS_NATIVE_UNSUPPORTED_OR_UNSAFE")
			return
		}
		reason := ""
		if e != nil {
			sample = state.NewReadinessSample(intent)
			reason = "READINESS_OBSERVATION_UNAVAILABLE"
		}
		if e == nil && readinessConditions(sample, obs, exec) {
			// Ordinary lifecycle tools can run during OBSERVING; acquire a short
			// point-of-use fence and repeat exact native observation before probe.
			claimErr := s.Store.ClaimReadiness(ctx, rec.ID)
			switch {
			case claimErr == nil:
				confirmed, before, checkErr := s.observeReady(ctx, box, intent)
				if checkErr != nil || !readinessConditions(confirmed, before, exec) {
					sample = confirmed
					if errors.Is(checkErr, state.ErrNotFound) || errors.Is(checkErr, native.ErrNotOwned) {
						_ = save("DISPATCHING", "FAILED", "EXACT_BOX_IDENTITY_LOST")
						return
					}
					if e = save("DISPATCHING", "OBSERVING", "READINESS_CHANGED_BEFORE_PROBE"); e != nil {
						return
					}
					break
				}
				sample = confirmed
				if ctx.Err() != nil {
					_ = save("DISPATCHING", "TIMED_OUT", "READINESS_DEADLINE_ELAPSED")
					return
				}
				if e = s.Store.ReadinessActorActive(ctx, rec.ID); e != nil {
					_ = save("DISPATCHING", "FAILED", "AUTH_REVOKED_BEFORE_PROBE")
					return
				}
				if !exec {
					_ = save("DISPATCHING", "SUCCEEDED", "")
					return
				}
				// No second probe is issued for this original: a lost or ambiguous
				// receipt stays UNKNOWN. Nonzero terminal exit is a witnessed failure.
				id, dispatchErr := s.Native.DispatchReadinessProbe(ctx, box.Name, box.UUID)
				if dispatchErr != nil || id == "" {
					sample.ExecReady = "unknown"
					if id != "" {
						_ = s.Store.NativeEffectErrorUnknown(context.Background(), rec.ID, id)
					} else {
						_ = save("DISPATCHING", "UNKNOWN", "READINESS_PROBE_ACK_UNVERIFIED")
					}
					return
				}
				if e = s.Store.NativeAck(context.Background(), rec.ID, id); e != nil {
					_ = s.Store.NativeAckUnknown(context.Background(), rec.ID, id)
					return
				}
				terminal, e := s.Native.ObserveUntilFinal(ctx, id)
				if e != nil || terminal == nil || terminal.ID != id {
					sample.ExecReady = "unknown"
					_ = save("VERIFYING", "UNKNOWN", "READINESS_PROBE_TERMINAL_UNVERIFIED")
					return
				}
				exit, e := native.ReadinessProbeExit(terminal)
				if e != nil {
					sample.ExecReady = "unknown"
					_ = save("VERIFYING", "UNKNOWN", "READINESS_PROBE_EXIT_UNVERIFIED")
					return
				}
				sample.GuestExit = &exit
				if exit != 0 {
					sample.ExecReady = "not_ready"
					_ = save("VERIFYING", "FAILED", "READINESS_PROBE_NONZERO_OR_UNSUPPORTED")
					return
				}
				after, post, e := s.observeReady(ctx, box, intent)
				if e != nil || !readinessConditions(after, post, true) || before.Started.IsZero() || !before.Started.Equal(post.Started) {
					// Probe completed, but readiness must be for the same current boot.
					sample.ExecReady = "unknown"
					_ = save("VERIFYING", "FAILED", "READINESS_POSTCONDITION_CHANGED")
					return
				}
				sample = after
				sample.ExecReady = "ready"
				sample.GuestExit = &exit
				if ctx.Err() != nil {
					_ = save("VERIFYING", "TIMED_OUT", "READINESS_DEADLINE_ELAPSED")
					return
				}
				if e = s.Store.ReadinessActorActive(ctx, rec.ID); e != nil {
					_ = save("VERIFYING", "FAILED", "READINESS_AUTH_REVOKED")
					return
				}
				_ = save("VERIFYING", "SUCCEEDED", "")
				return
			case errors.Is(claimErr, state.ErrResourceLocked):
				reason = "READINESS_WAITING_FOR_NATIVE_FENCE"
			default:
				_ = save("OBSERVING", "FAILED", "READINESS_FENCE_OR_AUTH_UNAVAILABLE")
				return
			}
		}
		if e = save("OBSERVING", "OBSERVING", reason); e != nil {
			return
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}
