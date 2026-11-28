package state

import (
	"context"
	"errors"
	"testing"
)

func TestS4B3PeerOriginalModeSelectorNeverAuthorizesWrongBranch(t *testing.T) {

	ctx := context.Background()
	for _, allow := range []bool{true, false} {
		s, r, id := pr178ScopedReadyToStage(t, allow)
		if _, e := r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge); e == nil {
			t.Fatal("ADMITTED")
		}
		if e := s.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
			t.Fatal(e)
		}
		p, e := r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge)
		if e != nil || p.Intent.Allow != allow || p.RecoveryOnly {
			t.Fatal(p, e)
		}
		if _, e = r.ReadRootScopedPeerUnknownOriginal(ctx, id, purePeerProject, purePeerBridge); e == nil {
			t.Fatal("wrong phase")
		}
		if _, e = s.PeerOperationDirection(ctx, id, "DISPATCHING"); !errors.Is(e, ErrResourceLocked) {
			t.Fatal("old selector accepted Scoped", e)
		}
		if _, e = s.db.Exec("UPDATE operations SET safe_intent='{}' WHERE id=?", id); e != nil {
			t.Fatal(e)
		}
		if _, e = r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge); e == nil {
			t.Fatal("tampered direction")
		}
	}
	t.Log("S4_B3_CONTROLLER_CLI_ORIGINAL_DIRECTION_STRICT_NONREPLAY=PASS")

}
