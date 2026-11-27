package control

import (
	"context"
	"errors"
	"fmt"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

// peerScopeRefusal deliberately does not distinguish a foreign UUID, missing
// PINNED NIC, incomplete global audit ledger or an unresolved resource claim.
// Exposing one cause would leak inventory or invite unsafe retry/bypass.
func peerScopeRefusal(err error) error {
	var block state.AdmissionBlock
	if errors.As(err, &block) {
		return err
	}
	if errors.Is(err, state.ErrResourceLocked) {
		return fmt.Errorf("PEER_SCOPE_UNVERIFIED_OR_LOCKED: verify both owned UUIDs with box_list, inspect operation_list/activity issues, then inspect exact original before considering local review; no peer operation was admitted: %w", err)
	}
	return err
}

// A public request has no authority to select an ACL, IP, route, shell,
// host path or PRS digest. It creates only a durable, scope-verified original.
type peerSetInput struct {
	SourceBoxID    string `json:"source_box_id"`
	TargetBoxID    string `json:"target_box_id"`
	Allow          bool   `json:"allow"`
	IdempotencyKey string `json:"idempotency_key"`
	WorkRef        string `json:"work_ref,omitempty"`
	AgentRef       string `json:"agent_ref,omitempty"`
}

func (s *Service) registerPeerTools(server *mcp.Server) {
	addStrictTool(server, &mcp.Tool{
		Name:        "box_peer_set",
		Description: "Admit a directional private-network grant/revoke for two exact UUID-owned Boxes in one operator-pinned Project/ACL. UUIDs must come from box_list. Missing/foreign IDs remain non-enumerating. Unverified scope/bindings/audit may refuse as PEER_SCOPE_UNVERIFIED_OR_LOCKED; an exact admitted blocker may identify an authorized original to poll or inspect. Never bypass or blindly retry. The controller automatically executes the exact admitted original through the authenticated local Root service; poll the original Operation. No retry of uncertain effects.",
		Annotations: mutationAnnotations(true),
	}, s.operatePeerSet)
}
func (s *Service) operatePeerSet(ctx context.Context, req *mcp.CallToolRequest, in peerSetInput) (*mcp.CallToolResult, state.Receipt, error) {

	principal, err := s.requestPrincipal(req)
	if err != nil {
		return nil, state.Receipt{}, err
	}
	if in.SourceBoxID == "" || in.TargetBoxID == "" || in.SourceBoxID == in.TargetBoxID || in.IdempotencyKey == "" {
		return nil, state.Receipt{}, state.ErrInvalid
	}
	work, agent, err := boundedWorkLabels(req, in.WorkRef, in.AgentRef)
	if err != nil {
		return nil, state.Receipt{}, err
	}
	previous, already, e := s.Store.ReplayScopedPeerSet(ctx, principal.Layer, principal.ActorID, principal.TokenID, in.IdempotencyKey, in.SourceBoxID, in.TargetBoxID, in.Allow, work, agent)
	if e != nil {
		return nil, state.Receipt{}, e
	}
	if already {
		return &mcp.CallToolResult{}, previous, nil
	}
	for _, id := range []string{in.SourceBoxID, in.TargetBoxID} {
		if _, e = s.readOwned(ctx, req, id); e != nil {
			// Keep absent, foreign and mismatched native identities
			// indistinguishable from an unverified Peer scope.
			if errors.Is(e, state.ErrNotFound) || errors.Is(e, native.ErrNotOwned) {
				return nil, state.Receipt{}, peerScopeRefusal(state.ErrResourceLocked)
			}
			return nil, state.Receipt{}, e
		}
	}
	requestID, e := newTrackingID()
	if e != nil {
		return nil, state.Receipt{}, e
	}
	r, e := s.Store.AdmitScopedPeerSet(ctx, state.Admission{Layer: principal.Layer, ActorID: principal.ActorID, TokenID: principal.TokenID, Key: in.IdempotencyKey, RequestID: requestID, WorkRef: work, AgentRef: agent}, in.SourceBoxID, in.TargetBoxID, in.Allow)
	if e != nil {
		return nil, state.Receipt{}, peerScopeRefusal(e)
	}
	if !r.Duplicate {
		go s.worker().ExecuteScopedPeerOriginal(r.ID, install.NativeScopedPeerCall)
	}
	return &mcp.CallToolResult{}, r, nil
}
