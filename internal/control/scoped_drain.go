package control

import (
	"context"
	"errors"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

// ScopedDrainStep is not a public tool or authorization mechanism. The
// private worker receives one root-approved fixed effect via peer credentials.
type ScopedDrainStep struct {
	Action  string `json:"action"`
	Box     string `json:"box"`
	Target  string `json:"target,omitempty"`
	Forward string `json:"forward,omitempty"`
	Key     string `json:"key"`
}

func (s *Service) RunScopedDrainStep(ctx context.Context, actor state.ScopedDrainActor, step ScopedDrainStep, expected string, readOnly bool) (state.Receipt, error) {
	var zero state.Receipt
	if s == nil || !s.ScopedPeerFresh || s.Store == nil || !validPrincipalID(actor.ID) {
		return zero, state.ErrInvalid
	}
	// First inspect exact original, even if subsequent state has changed.
	prior, e := s.Store.ScopedDrainOriginal(ctx, s.Native.Project, actor.ID, step.Key)
	if e == nil {
		if prior.Action != step.Action || prior.BoxUUID != step.Box || prior.Incarnation != step.Box || prior.WorkRef != "scoped-drain-"+actor.ID {
			return zero, state.ErrConflict
		}
		if prior.State != "SUCCEEDED" {
			return prior, errors.New("SCOPED_DRAIN_ORIGINAL_UNSETTLED: preserve original; never redispatch")
		}
		return prior, nil
	}
	if !errors.Is(e, state.ErrNotFound) {
		return zero, e
	}
	if readOnly {
		return zero, state.ErrNotFound
	}
	l, e := s.Store.ReadScopedDrainWitness(ctx, s.Native.Project)
	if e != nil || l.ObservationDigest() != expected {
		return zero, state.ErrConflict
	}
	// Check the complete scoped native ledger, including exact pinned
	// bindings and retained lineages, before every new effect.
	if _, e = s.Store.ReadCompleteScopedNetworkAuditLedger(ctx, s.Native.Project, "incusbr-"+s.Native.Project[len("user-"):]); e != nil {
		return zero, e
	}
	if e = s.Store.EnsureScopedDrainActor(ctx, actor); e != nil {
		return zero, e
	}
	req := &mcp.CallToolRequest{Extra: &mcp.RequestExtra{TokenInfo: &auth.TokenInfo{UserID: actor.ID, Extra: map[string]any{"token_id": actor.ID}}}}
	work := "scoped-drain-" + actor.ID
	var r state.Receipt
	switch step.Action {
	case "box_peer_set":
		if step.Target == "" || step.Forward != "" || step.Box == step.Target {
			return zero, state.ErrInvalid
		}
		ok := false
		for _, p := range l.Peers {
			if p.Source == step.Box && p.Target == step.Target && p.SourceIncarnation == p.Source && p.TargetIncarnation == p.Target && p.State == "CONFIRMED" {
				ok = true
			}
		}
		if !ok {
			return zero, state.ErrConflict
		}
		_, r, e = s.operatePeerSet(ctx, req, peerSetInput{SourceBoxID: step.Box, TargetBoxID: step.Target, Allow: false, IdempotencyKey: step.Key, WorkRef: work})
	case "port_forward_delete":
		if step.Forward == "" || step.Target != "" {
			return zero, state.ErrInvalid
		}
		ok := false
		for _, f := range l.Forwards {
			if f.ID == step.Forward && f.Box == step.Box && f.Incarnation == f.Box && f.State == "CONFIRMED" {
				ok = true
			}
		}
		if !ok {
			return zero, state.ErrConflict
		}
		_, r, e = s.operateForwardDelete(ctx, req, forwardDeleteInput{ForwardID: step.Forward, IdempotencyKey: step.Key, WorkRef: work})
	case "box_stop", "box_delete":
		if step.Target != "" || step.Forward != "" {
			return zero, state.ErrInvalid
		}
		ok := false
		for _, b := range l.Boxes {
			if b.UUID == step.Box && b.Incarnation == b.UUID && b.CreateProof && b.BindingState == "PINNED" {
				ok = true
			}
		}
		if !ok {
			return zero, state.ErrConflict
		}
		// Never let the ordinary Delete cascade admit a dependency outside
		// the separately approved directional and Forward effect inventory.
		if step.Action == "box_delete" {
			for _, p := range l.Peers {
				if p.State != "REVOKED" && (p.Source == step.Box || p.Target == step.Box) {
					return zero, state.ErrResourceLocked
				}
			}
			for _, f := range l.Forwards {
				if f.State != "REVOKED" && f.Box == step.Box {
					return zero, state.ErrResourceLocked
				}
			}
		}
		in := operateBoxInput{BoxID: step.Box, IdempotencyKey: step.Key, WorkRef: work}
		if step.Action == "box_stop" {
			_, r, e = s.operateBoxPower(ctx, req, in, "box_stop")
		} else {
			_, r, e = s.operateBoxDelete(ctx, req, in)
		}
	default:
		return zero, state.ErrInvalid
	}
	if e != nil {
		return r, e
	}
	if r.ID == "" {
		return zero, state.ErrConflict
	}
	// Worker budgets and durable receipts remain the existing business
	// Operation protocol. Timeout never authorizes new native dispatch.
	for {
		prior, e = s.Store.Get(ctx, r.ID)
		if e != nil {
			return r, e
		}
		switch prior.State {
		case "SUCCEEDED":
			return prior, nil
		case "FAILED", "UNKNOWN", "CANCELLED", "TIMED_OUT":
			return prior, errors.New("SCOPED_DRAIN_ORIGINAL_UNSETTLED: preserve original")
		}
		select {
		case <-ctx.Done():
			return prior, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}
