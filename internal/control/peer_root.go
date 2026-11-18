package control

import (
	"context"
	"errors"

	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/state"
)

// SubmitRootPeerRevocation wires the restricted, durable controller worker
// to the host-approved SO_PEERCRED root helper. The public MCP Peer writer
// is intentionally NOT exposed in B2: it must supply a real native
// quarantine callback AND independent per-NIC nft/foreign packet verifier
// before this private path may ever be used for a caller's request.
func (s *Service) SubmitRootPeerRevocation(original string,
	isolate state.SynchronousACLIsolation, verify state.RootPeerVerifier) error {
	if s == nil || s.Store == nil || isolate == nil || verify == nil {
		return errors.New("installed root peer revocation requires durable controller and kernel verifier")
	}
	rootCall := func(ctx context.Context, id, sha string) (state.RootPeerResult, error) {
		report, err := install.NativePeerRootCall(ctx, id, sha)
		out := state.RootPeerResult{Status: report.Status, PlanSHA256: report.PlanSHA256}
		if report.Receipt != nil {
			out.PlanSHA256 = report.Receipt.PlanSHA256
			out.ProbedScopes = report.Receipt.ProbedScopes
			out.DeletedScopes = report.Receipt.DeletedScopes
			out.MutationComplete = report.Receipt.MutationComplete
			out.ReadyForPeer = report.Receipt.ReadyForPeer
		}
		return out, err
	}
	return (state.Worker{Store: s.Store}).SubmitRootPeerRevocation(original, isolate, rootCall, verify)
}
