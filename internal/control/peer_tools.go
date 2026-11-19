package control

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

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
		Description: "Admit a directional private-network grant/revoke for two UUID-owned Boxes in an already operator-pinned Project/ACL. Returns an original Operation receipt for operation_get; host-root operator must separately stage, approve the exact digest and run PRS. ADMITTED/PENDING does not mean connectivity changed; this MCP tool never directly changes native network state.",
		Annotations: mutationAnnotations(true),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in peerSetInput) (*mcp.CallToolResult, state.Receipt, error) {
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
			return nil, state.Receipt{}, err
		}
		ledger, err := s.Store.ReadCompleteNetworkAuditLedger(ctx, principal.Layer, bridge, acl)
		if err != nil {
			return nil, state.Receipt{}, state.ErrResourceLocked
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
	})
}
