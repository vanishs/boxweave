package control

import (
	"context"
	"github.com/vanishs/boxweave/internal/install"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/recovery"
	"github.com/vanishs/boxweave/internal/state"
)

type operationRecoverInput struct {
	OperationID    string `json:"operation_id"`
	Mode           string `json:"mode"`
	Stream         string `json:"stream,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	WorkRef        string `json:"work_ref,omitempty"`
	AgentRef       string `json:"agent_ref,omitempty"`
}

func (s *Service) registerRecoveryTool(server *mcp.Server) {
	addStrictTool(server, &mcp.Tool{Name: "operation_recover",
		Description: "Inspect the exact authorized original Operation and optionally reconcile only independently proven original native terminal/poststate or bind an exact unbound legacy Create identity without releasing UNKNOWN. Staged static Creates require exact original intent, native poststate and both address reservations; ownership, pin, fences and original settlement commit atomically. A completed modern Create may restore only its own missing binding or fence rows after native UUID/static lease proof; conflicting records are never overwritten. Healthy completed Create checks current pinned NIC without mutation. A lost Create ACK can be settled from exact original marker/static poststate and Project quiescence without inventing a native ID. Pinned lifecycle recovery also verifies the exact durable NIC tuple. mode=inspect is read-only; mode=reconcile requires its own idempotency_key and returns a durable recovery receipt separate from the original. Optional stream examines original Exec GC review. Forward and dependent Delete require exact original success and dependency proof; Peer requires completed PRS journal and independent root read-only kernel verification. Unproved outcomes remain blocked or require approval (not available here). No original replay, native dispatch, or blanket unlock.",
		Annotations: mutationAnnotations(false)},
		func(ctx context.Context, req *mcp.CallToolRequest, in operationRecoverInput) (*mcp.CallToolResult, recovery.Result, error) {
			principal, e := s.requestPrincipal(req)
			if e != nil {
				return nil, recovery.Result{}, e
			}
			ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()
			svc := recovery.Service{Store: s.Store, Native: s.Native, Layer: principal.Layer, PeerWitness: recovery.ObserveCompletedPeer}
			if s.ScopedPeerFresh {
				svc.ScopedPeerWitness = install.NativeScopedPeerCall
			}
			var out recovery.Result
			switch in.Mode {
			case "inspect":
				if in.IdempotencyKey != "" || in.WorkRef != "" || in.AgentRef != "" {
					return nil, out, state.ErrInvalid
				}
				out, e = svc.Inspect(ctx, in.OperationID, in.Stream)
			case "reconcile":
				out, e = svc.Reconcile(ctx, state.RecoveryRequest{Layer: principal.Layer, ActorID: principal.ActorID, TokenID: principal.TokenID, OriginalID: in.OperationID, Key: in.IdempotencyKey, Stream: in.Stream, WorkRef: in.WorkRef, AgentRef: in.AgentRef})
			default:
				return nil, out, state.ErrInvalid
			}
			if e != nil {
				return nil, out, e
			}
			return privateReadReply(ctx, s, principal.TokenID, out)
		})
}
