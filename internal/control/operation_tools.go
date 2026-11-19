package control

import (
	"context"
	"errors"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

// RequestPrincipal is the only handler-facing identity, derived from the
// SDK-verified Bearer request. Never use client labels or stable OAuth Grant
// IDs as independently revocable access credentials.
type RequestPrincipal struct {
	ActorID string
	TokenID string
	Layer   string
}

// requestPrincipal deliberately has no historical UserID-only fallback.
// OAuth refresh keeps ActorID stable while issuing a new TokenID.
func (s *Service) requestPrincipal(req *mcp.CallToolRequest) (RequestPrincipal, error) {
	if req == nil || req.Extra == nil || req.Extra.TokenInfo == nil {
		return RequestPrincipal{}, errors.New("UNAUTHENTICATED")
	}
	info := req.Extra.TokenInfo
	raw, present := info.Extra["token_id"]
	tokenID, ok := raw.(string)
	if !present || !ok || !validPrincipalID(info.UserID) || !validPrincipalID(tokenID) {
		return RequestPrincipal{}, errors.New("UNAUTHENTICATED")
	}
	if s == nil || s.Native.Project == "" || s.Store == nil {
		return RequestPrincipal{}, errors.New("CONTROLLER_NOT_CONFIGURED")
	}
	return RequestPrincipal{ActorID: info.UserID, TokenID: tokenID, Layer: s.Native.Project}, nil
}

func validPrincipalID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return false
		}
	}
	return true
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
	addStrictTool(server, &mcp.Tool{
		Name:        "operation_get",
		Description: "Read an authorized original operation_id after a timeout/reconnect; compare durable state and next_action before any further mutation. UNKNOWN may have dispatched native work: do not retry with a new key; require independent read-only post-state/host review. wait_ms 0..20000 only polls, never cancels or retries.",
		Annotations: readToolAnnotations(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in operationGetInput) (*mcp.CallToolResult, operationGetResult, error) {
		principal, err := s.requestPrincipal(req)
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
			result, err = s.Store.GetScopedOperation(ctx, principal.Layer, principal.ActorID, in.OperationID)
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
		// A token that was valid when the HTTP request arrived may have
		// expired or been revoked during this bounded long poll. Refuse to
		// release the private original receipt after that event. Never
		// cancel/retry the durable operation as a side-effect.
		if err := s.Store.RequireActiveTokenID(ctx, principal.TokenID); err != nil {
			return nil, operationGetResult{}, err
		}
		return &mcp.CallToolResult{}, operationGetResult{TrackingID: tracking,
			Operation: result, Version: s.BuildVersion, Commit: s.BuildCommit}, nil
	})

	addStrictTool(server, &mcp.Tool{
		Name:        "operation_list",
		Description: "After agent interruption/reconnect, search original Operations by known work_ref, or bounded action/time pages and cursor; follow exact operation_id with operation_get. Empty results do not prove no native effect or that work finished. Client work/Agent labels never grant visibility.",
		Annotations: readToolAnnotations(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in operationListInput) (*mcp.CallToolResult, operationListResult, error) {
		principal, err := s.requestPrincipal(req)
		if err != nil {
			return nil, operationListResult{}, err
		}
		tracking, err := newTrackingID()
		if err != nil {
			return nil, operationListResult{}, errors.New("TRACKING_UNAVAILABLE")
		}
		page, err := s.Store.ListOperations(ctx, state.OperationQuery{
			Layer: principal.Layer, Requester: principal.ActorID, ActorID: in.ActorID,
			WorkRef: in.WorkRef, AgentRef: in.AgentRef,
			Action: in.Action, State: in.State, SinceNS: in.SinceNS,
			UntilNS: in.UntilNS, Limit: in.Limit, Cursor: in.Cursor,
		})
		if err != nil {
			return nil, operationListResult{}, err
		}
		return privateReadReply(ctx, s, principal.TokenID, operationListResult{TrackingID: tracking, Page: page})
	})
}
