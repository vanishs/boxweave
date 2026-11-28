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
		Description: "Inspect an exact authorized original Operation. Scoped Create/Delete require the durable original native ID, completed Root lifecycle journal and independent final poststate; incomplete effects remain UNKNOWN. Pinned lifecycle recovery verifies the exact durable NIC tuple. Scoped Peer requires the completed original Root journal and independent policy proof. Forward recovery requires exact original native success and dependency proof. mode=inspect is read-only; mode=reconcile requires its own idempotency_key and returns a separate durable recovery receipt. Historical unsupported originals remain review-only. Optional stream inspects original Exec GC; release requires local review, not MCP approval. No original replay, native dispatch, adoption or blanket unlock.",
		Annotations: mutationAnnotations(false)},
		func(ctx context.Context, req *mcp.CallToolRequest, in operationRecoverInput) (*mcp.CallToolResult, recovery.Result, error) {
			principal, e := s.requestPrincipal(req)
			if e != nil {
				return nil, recovery.Result{}, e
			}
			ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()
			svc := s.recoveryObserver(principal.Layer)
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
			next := state.NextAction{Kind: "READ_ORIGINAL", Tool: "operation_get", OperationID: out.View.OriginalID}
			if out.Recovery == nil {
				switch out.View.Disposition {
				case "SAFE_RECOVER":
					next = state.NextAction{Kind: "RECONCILE_ORIGINAL", Tool: "operation_recover", OperationID: out.View.OriginalID, Mode: "reconcile"}
				case state.RecoveryApproval:
					next = state.NextAction{Kind: "LOCAL_REVIEW_REQUIRED"}
				case state.RecoveryBlocked:
					next = state.NextAction{Kind: "READ_EVIDENCE", Tool: "box_activity_get", OperationID: out.View.OriginalID, View: "evidence"}
				}
			} else {
				next = state.NextAction{Kind: "READ_ORIGINAL", Tool: "operation_get", OperationID: out.View.OriginalID}
			}
			out.NextAction = &next
			return privateReadReply(ctx, s, principal.TokenID, out)
		})
}

// Recovery witnesses are mutually exclusive. A Scoped controller must not
// fall back to the historical PRS socket for a legacy original.
func (s *Service) recoveryObserver(layer string) recovery.Service {
	svc := recovery.Service{Store: s.Store, Native: s.Native, Layer: layer}
	svc.ScopedPeerWitness = install.NativeScopedPeerRecoverCall
	svc.ScopedLifecycleWitness = install.NativeScopedLifecycleRecoverCall
	return svc
}
