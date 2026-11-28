package control

import (
	"context"
	"time"

	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// Parent retains its original durable Operation while each separately
// journaled child performs one exact native effect. No permission-restoring
// compensation (regrant/recreate proxy) is allowed after partial cleanup.
func (s *Service) executeDependentDelete(ctx context.Context, original string, box state.OwnedBox) (string, error) {
	return s.executeDependentDeleteWithWitness(ctx, original, box, nil)
}

// The optional witness is internal/test-only, never a production setting,
// API flag or source of approval. Normal dispatch passes nil.
func (s *Service) executeDependentDeleteWithWitness(ctx context.Context, original string, box state.OwnedBox, witness func(context.Context) error) (string, error) {
	cascade, forwards, e := s.Store.DeleteCascadePlan(ctx, original)
	if e != nil {
		return "", state.PreDispatchError{Class: "DELETE_DEPENDENCY_PREFLIGHT_DENIED"}
	}
	if !cascade {
		id, e := s.Native.DeleteBox(ctx, box.Name, box.UUID)
		return id, preflightNative(e)
	}
	pin, e := s.Store.DeletePinnedNIC(ctx, original)
	if e != nil {
		return "", state.PreDispatchError{Class: "DELETE_PINNED_LEDGER_UNVERIFIED"}
	}
	checkPin := func() error {
		if pin == nil {
			return nil
		}
		return s.Native.VerifyPinnedDeleteNIC(ctx, box.Name, box.UUID, pin.Bridge, pin.ACL, pin.MAC, pin.IPv4, pin.IPv6)
	}
	if pin != nil {
		dependent, e := s.Store.DeleteHasTrafficDependencies(ctx, original)
		if e != nil {
			return "", state.PreDispatchError{Class: "DELETE_DEPENDENCY_PREFLIGHT_DENIED"}
		}
		if !dependent {
			current, _, e := s.Native.GetExactBox(ctx, box.Name, box.UUID)
			if e != nil || current.NativeState != "Stopped" {
				return "", state.PreDispatchError{Class: "DELETE_REQUIRES_STOPPED"}
			}
		}
	}
	if e = checkPin(); e != nil {
		return "", state.PreDispatchError{Class: "DELETE_PINNED_NATIVE_UNVERIFIED"}
	}
	specs := make([]native.ForwardSpec, 0, len(forwards))
	for _, f := range forwards {
		specs = append(specs, specForClaim(f))
	}
	if e = s.Native.PreflightDependentDelete(ctx, box.Name, box.UUID, specs); e != nil {
		return "", state.PreDispatchError{Class: "DELETE_EXACT_NATIVE_DEPENDENCY_PREFLIGHT_DENIED"}
	}
	// After admitting any child, a failure must retain the parent UNKNOWN
	// even if final DeleteInstance itself has not been dispatched.
	uncertain := func() (string, error) {
		return "", state.VerificationUncertain{Class: "DELETE_DEPENDENCY_CHILD_OR_POSTSTATE_UNCERTAIN"}
	}
	for steps := 0; steps < 84; steps++ {
		current, _, e := s.Native.GetExactBox(ctx, box.Name, box.UUID)
		if e != nil {
			return uncertain()
		}
		needStop := current.NativeState == "Running" || current.NativeState == "Ready"
		if !needStop && current.NativeState != "Stopped" {
			return uncertain()
		}
		child, e := s.Store.NextDeleteChild(ctx, original, needStop)
		if e != nil {
			return uncertain()
		}
		if child.Receipt.ID == "" {
			if current.NativeState != "Stopped" {
				return uncertain()
			}
			if e = s.Store.AuthorizeCascadeFinal(ctx, original); e != nil {
				return uncertain()
			}
			if e = checkPin(); e != nil {
				return uncertain()
			}
			if e = s.Native.PreflightDependentDelete(ctx, box.Name, box.UUID, nil); e != nil {
				return uncertain()
			}
			if witness != nil {
				if e = witness(ctx); e != nil {
					return uncertain()
				}
			}
			// Only the final native delete ID is recorded on the parent. Child
			// native IDs retain their own original receipts.
			return s.Native.DeleteBox(ctx, box.Name, box.UUID)
		}
		id := child.Receipt.ID
		switch child.Receipt.Action {
		case "port_forward_delete":
			spec := specForClaim(child.Forward)
			var nativeID string
			s.worker().Execute(id, func(c context.Context) (string, error) {
				n, e := s.Native.UpdateOwnedForward(c, box.Name, box.UUID, spec, false)
				nativeID = n
				return n, preflightNative(e)
			}, func(c context.Context) error {
				if e := nativeDone(c, s.Native, nativeID); e != nil {
					return e
				}
				if e := s.Native.ObserveOwnedForward(c, box.Name, box.UUID, spec, false); e != nil {
					return e
				}
				if e := s.Store.FinalizeVerifiedForward(c, id, false); e != nil {
					return e
				}
				return state.ErrVerifiedForwardCommitted
			})
		case "box_peer_set":
			if e := s.worker().ExecuteScopedPeerOriginal(id, install.NativeScopedPeerCall); e != nil {
				return uncertain()
			}
		case "box_stop":
			var nativeID string
			s.worker().Execute(id, func(c context.Context) (string, error) {
				n, e := s.Native.SetPower(c, box.Name, box.UUID, "stop")
				nativeID = n
				return n, preflightNative(e)
			}, func(c context.Context) error {
				if e := nativeDone(c, s.Native, nativeID); e != nil {
					return e
				}
				current, _, e := s.Native.GetExactBox(c, box.Name, box.UUID)
				if e != nil || current.NativeState != "Stopped" {
					return native.ErrUnverified
				}
				return nil
			})
		default:
			return uncertain()
		}
		// Root operator may need to stage/review the peer child. The parent is
		// bounded by the existing worker budget; restart or timeout holds UNKNOWN.
		for {
			r, e := s.Store.Get(ctx, id)
			if e != nil {
				return uncertain()
			}
			if r.State == "SUCCEEDED" {
				break
			}
			if r.State == "FAILED" || r.State == "UNKNOWN" || r.State == "CANCELLED" {
				return uncertain()
			}
			select {
			case <-ctx.Done():
				_ = s.Store.CancelQueued(context.Background(), box.Project, id)
				return uncertain()
			case <-time.After(250 * time.Millisecond):
			}
		}
	}
	return uncertain()
}
