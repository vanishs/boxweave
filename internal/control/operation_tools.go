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
	TrackingID              string                   `json:"tracking_id"`
	Operation               state.Receipt            `json:"operation"`
	Version                 string                   `json:"version"`
	Commit                  string                   `json:"commit"`
	Readiness               *state.ReadinessResult   `json:"readiness,omitempty"`
	Diagnostic              operationDiagnostic      `json:"diagnostic"`
	Recovery                state.RecoveryNavigation `json:"recovery_navigation"`
	RecoverySnapshotMatches bool                     `json:"recovery_snapshot_matches_operation"`
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
		Description: "Read an authorized original Operation by operation_id: durable state, native ID, next_action and typed durable diagnostic plus bounded recovery_navigation (follow original_link/recent_recoveries or next_action; inspect before reconciling); recorded output metadata is not live output availability. Optional wait_ms (0..20000) only polls; it never cancels or retries. UNKNOWN may have dispatched native work: verify read-only post-state or request host review before further effects.",
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
			if in.WaitMS == 0 || operationTerminal(result.State) {
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
		readiness, err := s.Store.Readiness(ctx, result)
		if err != nil {
			return nil, operationGetResult{}, err
		}
		diagnostic := s.readOperationDiagnostic(ctx, result, in.WaitMS > 0)
		navCtx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
		navigation, navErr := s.Store.ReadRecoveryNavigation(navCtx, principal.Layer, result.ID)
		cancel()
		matches := navErr == nil && navigation.UpdatedAt == result.UpdatedAt && navigation.OriginalState == result.State
		if navErr != nil {
			navigation = state.RecoveryNavigation{Source: "SQLITE_UNAVAILABLE", Disposition: "SQLITE_READ_UNVERIFIED", ClaimProtection: "UNVERIFIED", Recent: []state.RecoveryLink{}, ApprovalCapability: "TRUSTED_CLIENT_APPROVAL_UNAVAILABLE", OutputAvailability: "NOT_CHECKED"}
		}
		if !matches {
			navigation.NextAction = state.NextAction{Kind: "READ_OPERATION", Tool: "operation_get", OperationID: result.ID}
			diagnostic.NextAction = navigation.NextAction
		} else if result.State == "UNKNOWN" || result.Action == "operation_recover" {
			diagnostic.NextAction = navigation.NextAction
		}
		if !s.ScopedPeerFresh && result.Action == "box_peer_set" && result.State == "ADMITTED" && matches {
			diagnostic.StatusClass = "LOCAL_APPROVAL_REQUIRED"
			diagnostic.NextAction = state.NextAction{Kind: "LOCAL_REVIEW_REQUIRED"}
			navigation.NextAction = diagnostic.NextAction
			navigation.Disposition = "LEGACY_PEER_LOCAL_APPROVAL_REQUIRED"
		}
		// A token that was valid when the HTTP request arrived may have
		// expired or been revoked during this bounded long poll. Refuse to
		// release the private original receipt after that event. Never
		// cancel/retry the durable operation as a side-effect.
		if err := s.Store.RequireActiveTokenID(ctx, principal.TokenID); err != nil {
			return nil, operationGetResult{}, err
		}
		return &mcp.CallToolResult{}, operationGetResult{TrackingID: tracking,
			Operation: result, Version: s.BuildVersion, Commit: s.BuildCommit, Readiness: readiness, Diagnostic: diagnostic, Recovery: navigation, RecoverySnapshotMatches: matches}, nil
	})

	addStrictTool(server, &mcp.Tool{
		Name:        "operation_list",
		Description: "List authorized durable Operations by work_ref, agent_ref, actor, action, state or bounded time/cursor filters. Use after reconnect to find an original operation_id, then call operation_get. Empty results never prove an effect did not occur; client labels do not grant access.",
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
