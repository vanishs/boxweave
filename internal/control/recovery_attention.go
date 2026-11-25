package control

import (
	"context"
	"github.com/vanishs/boxweave/internal/state"
	"time"
)

// Attention is explicitly separate from the Native readiness/Box observation.
// Its small sample never claims that all native or admission conditions hold.
type attentionSnapshot struct {
	Source             string                `json:"source"`
	Scope              string                `json:"scope"`
	BoxID              string                `json:"box_id,omitempty"`
	AsOfNS             int64                 `json:"as_of_ns"`
	Coverage           string                `json:"coverage"`
	Items              []state.ActivityIssue `json:"items"`
	More               bool                  `json:"more"`
	AttentionComplete  bool                  `json:"attention_complete"`
	NativeLiveVerified bool                  `json:"native_live_verified"`
	NextTool           string                `json:"next_tool"`
	NextView           string                `json:"next_view"`
	NextAction         state.NextAction      `json:"next_action"`
	ApprovalCapability string                `json:"approval_capability"`
}

func (s *Service) readAttention(ctx context.Context, p RequestPrincipal, uuid string) attentionSnapshot {
	a := attentionSnapshot{Source: "SQLITE_SNAPSHOT", Scope: p.Layer, BoxID: uuid, Coverage: "UNAVAILABLE", Items: []state.ActivityIssue{}, NextTool: "box_activity_get", NextView: "issues", NextAction: issuesNext(), ApprovalCapability: "TRUSTED_CLIENT_APPROVAL_UNAVAILABLE"}
	readCtx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	defer cancel()
	out, e := s.Store.ReadActivity(readCtx, state.ActivityQuery{Layer: p.Layer, Requester: p.TokenID, BoxUUID: uuid, View: "issues", Limit: 10, BlockersOnly: true})
	if e != nil {
		return a
	}
	a.AsOfNS = out.AsOfNS
	a.Coverage = "BOUNDED_BLOCKER_SAMPLE"
	a.Items = out.Issues
	a.More = out.Truncated
	if len(a.Items) == 1 && !a.More && a.Items[0].NextAction != nil && a.Items[0].Disposition != "ACTIVE_DEPENDENCY_NOT_ERROR" {
		a.NextAction = *a.Items[0].NextAction
	}
	return a
}
