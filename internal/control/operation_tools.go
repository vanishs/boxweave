package control

import (
	"context"
	"errors"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

// The SQLite ledger is local to this controller. The installed restricted
// native Project supplies the local layer scope; work/agent labels never do.
func (s *Service) operationPrincipal(req *mcp.CallToolRequest) (actor, role, layer string, err error) {
	if req == nil || req.Extra == nil || req.Extra.TokenInfo == nil {
		return "", "", "", errors.New("UNAUTHENTICATED")
	}
	info := req.Extra.TokenInfo
	if info.UserID == "" {
		return "", "", "", errors.New("UNAUTHENTICATED")
	}
	switch {
	case contains(info.Scopes, "admin"):
		role = "admin"
	case contains(info.Scopes, "viewer"):
		role = "viewer"
	default:
		return "", "", "", errors.New("PERMISSION_DENIED")
	}
	if s.Native.Project == "" || s.Store == nil {
		return "", "", "", errors.New("CONTROLLER_NOT_CONFIGURED")
	}
	return info.UserID, role, s.Native.Project, nil
}

type operationGetInput struct {
	OperationID string `json:"operation_id"`
	WaitMS      int    `json:"wait_ms,omitempty"`
}

type operationGetResult struct {
	TrackingID string        `json:"tracking_id"`
	Operation  state.Receipt `json:"operation"`
	Version    string        `json:"version"`
	Commit     string        `json:"commit"`
}

type operationListInput struct {
	WorkRef  string `json:"work_ref,omitempty"`
	AgentRef string `json:"agent_ref,omitempty"`
	ActorID  string `json:"actor_id,omitempty"`
	Action   string `json:"action,omitempty"`
	State    string `json:"state,omitempty"`
	SinceNS  int64  `json:"since_ns,omitempty"`
	UntilNS  int64  `json:"until_ns,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Cursor   string `json:"cursor,omitempty"`
}

type operationListResult struct {
	TrackingID string              `json:"tracking_id"`
	Page       state.OperationPage `json:"page"`
}

func readToolAnnotations() *mcp.ToolAnnotations {
	destructive, openWorld := false, false
	return &mcp.ToolAnnotations{
		ReadOnlyHint: true, IdempotentHint: true,
		DestructiveHint: &destructive, OpenWorldHint: &openWorld,
	}
}

func (s *Service) registerOperationReadTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "operation_get",
		Description: "Read an original authorized durable Operation. wait_ms <= 20000 never cancels or retries native work.",
		Annotations: readToolAnnotations(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in operationGetInput) (*mcp.CallToolResult, operationGetResult, error) {
		actor, role, layer, err := s.operationPrincipal(req)
		if err != nil {
			return nil, operationGetResult{}, err
		}
		if in.WaitMS < 0 || in.WaitMS > 20000 {
			return nil, operationGetResult{}, state.ErrInvalid
		}
		tracking, err := newTrackingID()
		if err != nil {
			return nil, operationGetResult{}, errors.New("TRACKING_UNAVAILABLE")
		}
		var result state.Receipt
		deadline := time.Now().Add(time.Duration(in.WaitMS) * time.Millisecond)
		for {
			result, err = s.Store.GetScopedOperation(ctx, layer, actor, role, in.OperationID)
			if err != nil {
				return nil, operationGetResult{}, err
			}
			if in.WaitMS == 0 || result.State == "SUCCEEDED" || result.State == "FAILED" ||
				result.State == "UNKNOWN" || result.State == "CANCELLED" {
				break
			}
			remaining := time.Until(deadline)
			if remaining <= 0 {
				break
			}
			if remaining > 200*time.Millisecond {
				remaining = 200 * time.Millisecond
			}
			timer := time.NewTimer(remaining)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, operationGetResult{}, ctx.Err()
			case <-timer.C:
			}
		}
		return &mcp.CallToolResult{}, operationGetResult{TrackingID: tracking,
			Operation: result, Version: s.BuildVersion, Commit: s.BuildCommit}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "operation_list",
		Description: "Read a bounded, authenticated, indexed page of controller-local Operations. Client work/Agent labels never grant visibility.",
		Annotations: readToolAnnotations(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in operationListInput) (*mcp.CallToolResult, operationListResult, error) {
		actor, role, layer, err := s.operationPrincipal(req)
		if err != nil {
			return nil, operationListResult{}, err
		}
		tracking, err := newTrackingID()
		if err != nil {
			return nil, operationListResult{}, errors.New("TRACKING_UNAVAILABLE")
		}
		page, err := s.Store.ListOperations(ctx, state.OperationQuery{
			Layer: layer, Requester: actor, Role: role, ActorID: in.ActorID,
			WorkRef: in.WorkRef, AgentRef: in.AgentRef,
			Action: in.Action, State: in.State, SinceNS: in.SinceNS,
			UntilNS: in.UntilNS, Limit: in.Limit, Cursor: in.Cursor,
		})
		if err != nil {
			return nil, operationListResult{}, err
		}
		return &mcp.CallToolResult{}, operationListResult{TrackingID: tracking, Page: page}, nil
	})
}
