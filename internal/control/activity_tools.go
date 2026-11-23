package control

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

// Activity is a read-only SQLite evidence projection. A work/agent label is
// not an identity or a task lock; the authenticated Bearer principal supplies
// the only Layer and TokenID authority.
type activityGetInput struct {
	View        string `json:"view,omitempty"`
	BoxID       string `json:"box_id,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
	WorkRef     string `json:"work_ref,omitempty"`
	AgentRef    string `json:"agent_ref,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Cursor      string `json:"cursor,omitempty"`
}

type activityGetResult struct {
	TrackingID string               `json:"tracking_id"`
	Activity   state.ActivityResult `json:"activity"`
}

func (s *Service) registerActivityTools(server *mcp.Server) {
	addStrictTool(server, &mcp.Tool{
		Name:        "box_activity_get",
		Description: "Read BoxWeave's durable same-Project SQLite activity after reconnect. view=overview (default) lists recent originals; view=native_network performs an on-demand complete bounded same-Project SQLite+native inventory comparison (including exact static dual-stack leases), returning signed paged diagnosis but never a root/global trust proof or mutation authorization; issues identifies missing or inconsistent owned network pins/address fences and separates current blockers/dependencies (with dispositions) from a small historical released-claim/failed-operation sample and explicit bounded coverage; evidence requires operation_id and returns original events, resource claims and safe proof references including recovery_navigation; Box-scoped issues include Project-wide barriers (labels narrow scope). Optional box_id/work_ref/agent_ref narrow SQLite scope; native_network supports box_id, limit, cursor but refuses work_ref/agent_ref and does not infer root/global identity; limit 1..50 and signed next_cursor paginate overview/issues. This does not prove native or guest completion, and UNKNOWN must never be blindly retried.",
		Annotations: readToolAnnotations(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in activityGetInput) (*mcp.CallToolResult, activityGetResult, error) {
		principal, err := s.requestPrincipal(req)
		if err != nil {
			return nil, activityGetResult{}, err
		}
		if in.View == "" {
			in.View = "overview"
		}
		tracking, err := newTrackingID()
		if err != nil {
			return nil, activityGetResult{}, errors.New("TRACKING_UNAVAILABLE")
		}
		if in.View == "native_network" {
			result, err := s.readNativeNetworkActivity(ctx, principal, in)
			if err != nil {
				return nil, activityGetResult{}, err
			}
			return privateReadReply(ctx, s, principal.TokenID, activityGetResult{TrackingID: tracking, Activity: result})
		}
		result, err := s.Store.ReadActivity(ctx, state.ActivityQuery{
			Layer: principal.Layer, Requester: principal.TokenID,
			View: in.View, BoxUUID: in.BoxID, OperationID: in.OperationID,
			WorkRef: in.WorkRef, AgentRef: in.AgentRef,
			Limit: in.Limit, Cursor: in.Cursor,
		})
		if err != nil {
			return nil, activityGetResult{}, err
		}
		// Revocation during/after the SQL snapshot must suppress disclosure.
		return privateReadReply(ctx, s, principal.TokenID, activityGetResult{
			TrackingID: tracking, Activity: result,
		})
	})
}
