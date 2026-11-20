package state

import (
	"context"
	"errors"
	"time"
)

// SynchronousACLEffect is an official Incus Get/ETag/UpdateNetworkACL write.
// Unlike instance mutations, the SDK returns an error, not a native Operation.
// The caller supplies independently verified ACL/NIC/kernel postconditions.
type SynchronousACLEffect func(context.Context) error

// SynchronousACLIsolation must prove the native endpoints are stopped,
// autostart is disabled, dual-stack addresses stay pinned and every affected
// NIC is contained before ACL PUT. Unknown results preserve that barrier.
// A no-op is NEVER a valid installed-controller implementation.
type SynchronousACLIsolation func(context.Context) error

// AckSynchronousACL durably marks a successful synchronous SDK return.
// It deliberately does NOT manufacture a native Operation ID, or imply every
// NIC's in-kernel ACL reload converged: the verifier must establish that.
func (s *Store) AckSynchronousACL(ctx context.Context, id string) error {
	return s.transition(ctx, id, "DISPATCHING", "VERIFYING", "", "")
}

// ExecuteSynchronousACL is a separate typed path, not ErrNoNativeChange and
// not the normal asynchronous Worker.Execute. Dispatch intent is journaled
// first. On timeout, any SDK error after dispatch or uncertain poststate,
// retain UNKNOWN and the exact shared ACL/Box claims; never blindly retry.
func (w Worker) ExecuteSynchronousACL(id string, isolate SynchronousACLIsolation, effect SynchronousACLEffect, verify VerifyEffect) {
	if w.Store == nil || isolate == nil || effect == nil || verify == nil {
		return
	}
	rec, err := w.Store.Get(context.Background(), id)
	if err != nil || !(rec.Action == "box_peer_set" || rec.Action == "network_acl_reconcile") {
		return
	}
	if w.Budget <= 0 {
		w.Budget = 2 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), w.Budget)
	defer cancel()
	if err = w.Store.Claim(ctx, id); err != nil {
		return
	}
	if err = isolate(ctx); err != nil {
		_ = w.Store.Unknown(context.Background(), id, "DISPATCHING", "SYNC_ACL_ISOLATION_UNVERIFIED")
		return
	}
	if err = effect(ctx); err != nil {
		_ = w.Store.Unknown(context.Background(), id, "DISPATCHING", "SYNC_ACL_NATIVE_OUTCOME_AMBIGUOUS")
		return
	}
	if err = w.Store.AckSynchronousACL(context.Background(), id); err != nil {
		_ = w.Store.Unknown(context.Background(), id, "DISPATCHING", "SYNC_ACL_ACK_DURABILITY_UNVERIFIED")
		_ = w.Store.Unknown(context.Background(), id, "VERIFYING", "SYNC_ACL_ACK_DURABILITY_UNVERIFIED")
		return
	}
	if err = verify(ctx); err != nil {
		_ = w.Store.Unknown(context.Background(), id, "VERIFYING", "SYNC_ACL_POSTSTATE_UNVERIFIED")
		return
	}
	if err = w.Store.Verified(context.Background(), id); err != nil {
		_ = w.Store.Unknown(context.Background(), id, "VERIFYING", "SYNC_ACL_FINAL_DURABILITY_UNVERIFIED")
	}
}

func (w Worker) SubmitSynchronousACL(id string, isolate SynchronousACLIsolation, effect SynchronousACLEffect, verify VerifyEffect) error {
	if w.Store == nil || isolate == nil || effect == nil || verify == nil {
		return errors.New("ACL worker not configured")
	}
	r, err := w.Store.Get(context.Background(), id)
	if err != nil {
		return err
	}
	if r.Action != "box_peer_set" && r.Action != "network_acl_reconcile" {
		return ErrInvalid
	}
	if r.State != "ADMITTED" {
		return ErrState
	}
	go w.ExecuteSynchronousACL(id, isolate, effect, verify)
	return nil
}
