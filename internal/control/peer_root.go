package control

import (
	"context"
	"errors"

	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/state"
)

// PrepareRootPeerRevocation is a private installed-controller phase. It
// atomically moves an already admitted admin-owned original to DISPATCHING,
// without touching native state. The host-root operator can then review the
// exact pre-isolation digest and approve the bounded helper. No public MCP
// peer writer is exposed during B2.
func (s *Service) PrepareRootPeerRevocation(original string) error {
	if s == nil || s.Store == nil {
		return errors.New("missing durable controller")
	}
	ctx := context.Background()
	if err := s.Store.StageRootPeerRevocation(ctx, original); err != nil {
		return err
	}
	return s.Store.Claim(ctx, original)
}

// ExecutePreparedRootPeerRevocation has NO client-supplied callbacks or
// arbitrary native arguments. The separate host-root daemon performs all
// preflight, no-replay journaling, full-NIC isolation, ACL ETag DROP, exact
// conntrack and per-NIC nft verification before its positive response.
func (s *Service) ExecutePreparedRootPeerRevocation(original string) error {
	if s == nil || s.Store == nil {
		return errors.New("missing durable controller")
	}
	return (state.Worker{Store: s.Store}).ExecutePreparedRootPeerRevocation(
		original, func(ctx context.Context, id, sha string) (state.RootPeerResult, error) {
			report, err := install.NativePeerRootCall(ctx, id, sha)
			out := state.RootPeerResult{Status: report.Status, PlanSHA256: report.PlanSHA256}
			if report.Receipt != nil {
				out.PlanSHA256 = report.Receipt.PlanSHA256
				out.ProbedScopes = report.Receipt.ProbedScopes
				out.DeletedScopes = report.Receipt.DeletedScopes
				out.MutationComplete = report.Receipt.MutationComplete
				out.ReadyForPeer = report.Receipt.ReadyForPeer
				out.KernelVerified = report.Receipt.KernelVerified
				out.VerifiedNICs = report.Receipt.VerifiedNICs
				out.PostNativeDigestSHA256 = report.Receipt.PostNativeDigestSHA256
			}
			return out, err
		})
}

// ReconcileCompletedUnknownRootPeerRevocation accepts ONLY the exact already
// completed root journal and independent post-native read-only kernel proof.
// It never reapplies Stop, ACL PUT, conntrack or re-arms an UNKNOWN original.
func (s *Service) ReconcileCompletedUnknownRootPeerRevocation(original, sha string) error {
	if s == nil || s.Store == nil {
		return errors.New("missing restricted controller")
	}
	return (state.Worker{Store: s.Store}).ReconcileCompletedUnknownRootPeer(original, sha,
		func(ctx context.Context, id, digest string) (state.RootPeerResult, error) {
			resp, err := install.NativePeerRootCallReconcile(ctx, id, digest)
			out := state.RootPeerResult{Status: resp.Status}
			if resp.Receipt != nil {
				out.PlanSHA256 = resp.Receipt.PlanSHA256
				out.ProbedScopes = resp.Receipt.ProbedScopes
				out.DeletedScopes = resp.Receipt.DeletedScopes
				out.MutationComplete = resp.Receipt.MutationComplete
				out.ReadyForPeer = resp.Receipt.ReadyForPeer
				out.KernelVerified = resp.Receipt.KernelVerified
				out.VerifiedNICs = resp.Receipt.VerifiedNICs
				out.PostNativeDigestSHA256 = resp.Receipt.PostNativeDigestSHA256
			}
			return out, err
		})
}

// PrepareRootPeerGrant moves an already admitted, actor/admin-approved grant
// original into DISPATCHING. It does NOT execute an ACL PUT or assume that a
// staged SQLite row authorizes the host kernel. Root independently reviews.
func (s *Service) PrepareRootPeerGrant(original string) error {
	if s==nil || s.Store==nil {return errors.New("missing durable peer controller")}
	ctx:=context.Background()
	if err:=s.Store.StagePeerGrant(ctx,original);err!=nil{return err}
	return s.Store.Claim(ctx,original)
}

// ExecutePreparedRootPeerGrant passes ONLY the immutable original and an
// operator-approved SHA through the existing restricted PRS socket. Neither
// nonroot controller nor MCP supplies ACL, IP, host commands or root paths.
func (s *Service) ExecutePreparedRootPeerGrant(original string) error {
	if s==nil || s.Store==nil {return errors.New("missing durable peer controller")}
	return (state.Worker{Store:s.Store}).ExecutePreparedRootPeerGrant(original,
		func(ctx context.Context,id,sha string)(state.RootPeerResult,error){
			report,err:=install.NativePeerRootCall(ctx,id,sha)
			out:=state.RootPeerResult{Status:report.Status,PlanSHA256:report.PlanSHA256}
			if report.Receipt!=nil {
				out.PlanSHA256=report.Receipt.PlanSHA256
				out.ProbedScopes=report.Receipt.ProbedScopes
				out.DeletedScopes=report.Receipt.DeletedScopes
				out.MutationComplete=report.Receipt.MutationComplete
				out.ReadyForPeer=report.Receipt.ReadyForPeer
				out.KernelVerified=report.Receipt.KernelVerified
				out.VerifiedNICs=report.Receipt.VerifiedNICs
				out.PostNativeDigestSHA256=report.Receipt.PostNativeDigestSHA256
			}
			return out,err
		})
}
