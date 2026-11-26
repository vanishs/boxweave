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
// host path or PRS digest. It creates only a durable, operator-gated original.
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
		Description: "Admit a directional private-network grant/revoke for two exact UUID-owned Boxes in one operator-pinned Project/ACL. UUIDs must come from box_list. Missing/foreign IDs remain non-enumerating. Unverified scope/bindings/audit may refuse as PEER_SCOPE_UNVERIFIED_OR_LOCKED; an exact admitted blocker may identify an authorized original to poll or inspect. Never bypass or blindly retry. Fresh scoped installations automatically execute the exact admitted original through the authenticated local Root service; poll the original Operation. Legacy installations require host-local approval. No retry of uncertain effects.",
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
	if s.ScopedPeerFresh {
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
	return s.admitLegacyPeerSet(ctx, req, in, principal, work, agent)
}

// admitLegacyPeerSet retains the old local-review original. Authentication,
// validation and labels are resolved by operatePeerSet before either path.
func (s *Service) admitLegacyPeerSet(ctx context.Context, req *mcp.CallToolRequest, in peerSetInput, principal RequestPrincipal, work, agent string) (*mcp.CallToolResult, state.Receipt, error) {
	previous, already, err := s.Store.ReplayPeerSet(ctx, principal.Layer, principal.ActorID, principal.TokenID,
		in.IdempotencyKey, in.SourceBoxID, in.TargetBoxID, in.Allow, work, agent)
	if err != nil {
		return nil, state.Receipt{}, err
	}
	if already {
		return &mcp.CallToolResult{}, previous, nil
	}
	bridge, acl, err := s.Store.ResolvePeerAuditScope(ctx, principal.Layer, in.SourceBoxID, in.TargetBoxID)
	if err != nil {
		return nil, state.Receipt{}, peerScopeRefusal(err)
	}
	ledger, err := s.Store.ReadCompleteNetworkAuditLedger(ctx, principal.Layer, bridge, acl)
	if err != nil {
		return nil, state.Receipt{}, peerScopeRefusal(state.ErrResourceLocked)
	}
	var src, dst state.NetworkAuditBinding
	for _, b := range ledger.Boxes {
		if b.UUID == in.SourceBoxID {
			src = b
		}
		if b.UUID == in.TargetBoxID {
			dst = b
		}
	}
	if src.UUID == "" || dst.UUID == "" || src.UUID == dst.UUID {
		return nil, state.Receipt{}, state.ErrResourceLocked
	}
	intent := state.RootPeerRevokeSafeIntent{
		Project: principal.Layer, Bridge: bridge, ACL: acl,
		SourceUUID: src.UUID, SourceIncarnation: src.Incarnation,
		TargetUUID: dst.UUID, TargetIncarnation: dst.Incarnation,
		Allow: in.Allow,
	}
	safe, err := safeJSON(intent)
	if err != nil {
		return nil, state.Receipt{}, err
	}
	requestID, err := newTrackingID()
	if err != nil {
		return nil, state.Receipt{}, errors.New("TRACKING_UNAVAILABLE")
	}
	admission := state.Admission{
		Layer: principal.Layer, ActorID: principal.ActorID, TokenID: principal.TokenID,
		Key: in.IdempotencyKey, Action: "box_peer_set",
		BoxUUID: src.UUID, Incarnation: src.Incarnation,
		ResourceKey:  "acl:default/" + acl,
		ResourceKeys: []string{"box:" + src.UUID, "box:" + dst.UUID},
		SafeIntent:   safe, RequestID: requestID, WorkRef: work, AgentRef: agent,
	}
	for _, b := range []state.NetworkAuditBinding{src, dst} {
		owned, err := s.readOwned(ctx, req, b.UUID)
		if err != nil {
			return nil, state.Receipt{}, err
		}
		if owned.UUID != b.UUID || owned.Incarnation != b.Incarnation {
			return nil, state.Receipt{}, state.ErrResourceLocked
		}
	}
	result, err := s.Store.Admit(ctx, admission)
	return &mcp.CallToolResult{}, result, err
}
