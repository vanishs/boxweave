package recovery

import (
	"context"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/state"
)

// ObserveCompletedPeer is a bounded read-only root socket call. The running
// trusted helper selects the already approved digest and repeats its existing
// full Project, journal, kernel and conntrack witness. No mutation fallback.
func ObserveCompletedPeer(ctx context.Context, original string) (state.RootPeerResult, error) {
	resp, e := install.NativePeerRootCallReconcileApproved(ctx, original)
	if e != nil || resp.Receipt == nil || resp.Receipt.Status != "COMPLETE_VERIFIED" {
		return state.RootPeerResult{}, state.ErrResourceLocked
	}
	r := resp.Receipt
	return state.RootPeerResult{Status: resp.Status, PlanSHA256: r.PlanSHA256, ProbedScopes: r.ProbedScopes, DeletedScopes: r.DeletedScopes, MutationComplete: r.MutationComplete, ReadyForPeer: r.ReadyForPeer, KernelVerified: r.KernelVerified, VerifiedNICs: r.VerifiedNICs, PostNativeDigestSHA256: r.PostNativeDigestSHA256}, nil
}
