package control

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// ToolFailure is additive error metadata and safe text, not the success
// output schema. Error responses never carry arbitrary native/SQLite strings.
type ToolFailure struct {
	Code          string           `json:"code"`
	Category      string           `json:"category"`
	Admission     string           `json:"admission"`
	Message       string           `json:"message"`
	OriginalState string           `json:"original_state,omitempty"`
	Next          state.NextAction `json:"next_action"`
}

func (e ToolFailure) Error() string { b, _ := json.Marshal(e); return string(b) }
func failure(code, category, admission, message string, next state.NextAction) ToolFailure {
	return ToolFailure{Code: code, Category: category, Admission: admission, Message: message, Next: next}
}
func noNext(kind string) state.NextAction { return state.NextAction{Kind: kind} }
func issuesNext() state.NextAction {
	return state.NextAction{Kind: "READ_ISSUES", Tool: "box_activity_get", View: "issues"}
}
func statusNext() state.NextAction { return state.NextAction{Kind: "READ_STATUS", Tool: "host_status"} }
func reconnectNext() state.NextAction {
	return state.NextAction{Kind: "READ_ORIGINALS", Tool: "operation_list"}
}
func keyToolFailure() ToolFailure {
	return failure("INVALID_IDEMPOTENCY_KEY", "CALLER_INPUT", "NOT_ADMITTED", "Use a distinct idempotency_key of 16..128 UTF-8 bytes; reuse only for identical intent.", noNext("CORRECT_INPUT"))
}
func admittedToolFailure(id string) ToolFailure {
	return failure("ORIGINAL_RECEIPT_RECHECK_REQUIRED", "EFFECT_UNVERIFIED", "ORIGINAL_RECORDED", "An original receipt exists. Read it; never retry with a new key.", state.NextAction{Kind: "READ_OPERATION", Tool: "operation_get", OperationID: id})
}
func newToolFailure(tool string, err error) ToolFailure {
	var f ToolFailure
	if errors.As(err, &f) {
		return f
	}
	var block state.AdmissionBlock
	if errors.As(err, &block) && validPrincipalID(block.OriginalID) {
		next := state.NextAction{Kind: "READ_OPERATION", Tool: "operation_get", OperationID: block.OriginalID, WaitMS: 1000}
		code, category, msg := "RESOURCE_BUSY", "NORMAL_WAIT", "Existing original is in progress. Poll it before submitting the refused request. No new operation was admitted."
		if block.State == "UNKNOWN" {
			code, category, msg = "UNKNOWN_BLOCKER", "EFFECT_UNVERIFIED", "Existing original has an uncertain effect. Inspect recovery; do not redispatch or unlock. No new operation was admitted."
			next = state.NextAction{Kind: "INSPECT_RECOVERY", Tool: "operation_recover", OperationID: block.OriginalID, Mode: "inspect"}
		}
		f := failure(code, category, "NOT_ADMITTED", msg, next)
		f.OriginalState = block.State
		return f
	}
	switch {
	case errors.Is(err, state.ErrRevoked) || err.Error() == "UNAUTHENTICATED":
		return failure("AUTHORIZATION_REQUIRED", "AUTHORIZATION", "UNVERIFIED", "Authenticate again. Inspect authorized original before any further effect.", noNext("AUTHENTICATE"))
	case errors.Is(err, state.ErrConflict):
		return failure("IDEMPOTENCY_CONFLICT", "CALLER_INPUT", "NOT_ADMITTED", "Key conflicts with original intent. Check authorized original; do not change keys to bypass it.", reconnectNext())
	case errors.Is(err, state.ErrInvalid):
		return failure("INVALID_INPUT", "CALLER_INPUT", "UNVERIFIED", "Invalid admission or arguments; check schema and original if effect status is uncertain.", noNext("CORRECT_INPUT"))
	case errors.Is(err, state.ErrProjectCapacity):
		return failure("PROJECT_BOX_CAPACITY_REACHED", "CAPACITY", "NOT_ADMITTED", "Owned and unresolved Create slots exhausted; inspect capacity, not a controller failure.", statusNext())
	case errors.Is(err, state.ErrOwnedName):
		return failure("OWNED_NAME_IN_USE", "CALLER_INPUT", "NOT_ADMITTED", "An owned name is already reserved; inspect existing owned Boxes.", state.NextAction{Kind: "LIST_BOXES", Tool: "box_list"})
	case errors.Is(err, state.ErrDependencyReview):
		return failure("DEPENDENCY_REVIEW_REQUIRED", "RESOURCE_PRECONDITION", "NOT_ADMITTED", "Inspect confirmed and unresolved dependencies before lifecycle changes.", issuesNext())
	case errors.Is(err, state.ErrNotFound), errors.Is(err, native.ErrNotOwned):
		code := "RESOURCE_NOT_FOUND_OR_NOT_OWNED"
		if strings.HasPrefix(err.Error(), "BOX_NOT_FOUND_OR_NOT_OWNED:") || errors.Is(err, native.ErrNotOwned) {
			code = "BOX_NOT_FOUND_OR_NOT_OWNED"
		}
		return failure(code, "AUTHORIZATION_OR_ABSENCE", "UNVERIFIED", "Resource not found or not owned; use authorized listing. No foreign identity is disclosed.", noNext("CHECK_AUTHORIZED_SCOPE"))
	case errors.Is(err, state.ErrResourceLocked):
		code, msg := "RESOURCE_SCOPE_UNVERIFIED_OR_LOCKED", "Resource protection or scope evidence is incomplete. Inspect authorized activity; escalate only if evidence remains unresolved."
		if tool == "box_peer_set" {
			code, msg = "PEER_SCOPE_UNVERIFIED_OR_LOCKED", "Verify owned UUIDs and inspect authorized activity; no peer operation was admitted for an unverified scope or lock."
		}
		return failure(code, "RESOURCE_PRECONDITION", "UNVERIFIED", msg, issuesNext())
	}
	if code := native.CapabilityDenialClass(err); code != "" {
		category := "NATIVE_PRECONDITION"
		if code == "NETWORK_ADDRESS_CAPACITY_EXHAUSTED" {
			category = "CAPACITY"
		}
		return failure(code, category, "UNVERIFIED", "Required safe capability or capacity unavailable; inspect host status and documented supported constraints.", statusNext())
	}
	switch {
	case errors.Is(err, native.ErrRecordedOutputUnavailable):
		return failure("RECORDED_OUTPUT_UNAVAILABLE", "OUTPUT", "UNVERIFIED", "Read original durable operation and output availability; missing output does not prove non-execution.", reconnectNext())
	case errors.Is(err, native.ErrNativeUnsupported), errors.Is(err, native.ErrUnsafeNetwork), errors.Is(err, native.ErrNativeDependents):
		return failure("SAFE_NATIVE_PREFLIGHT_DENIED", "NATIVE_PRECONDITION", "UNVERIFIED", "Native capability or dependency could not be safely verified; inspect current status and activity.", statusNext())
	case errors.Is(err, native.ErrUnverified):
		return failure("NATIVE_OBSERVATION_UNVERIFIED", "NATIVE_EVIDENCE", "UNVERIFIED", "Native evidence unavailable or inconsistent. Inspect originals and status before any effect.", reconnectNext())
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return failure("REQUEST_INTERRUPTED", "EFFECT_UNVERIFIED", "UNVERIFIED", "Request interrupted; inspect exact original or authorized operation list, never assume no effect.", reconnectNext())
	case err.Error() == "TRACKING_UNAVAILABLE", err.Error() == "CONTROLLER_NOT_CONFIGURED":
		return failure("CONTROLLER_UNAVAILABLE", "CONTROLLER", "UNVERIFIED", "Controller cannot provide verified evidence. Inspect status; local Doctor if still unavailable.", statusNext())
	default:
		return failure("CONTROLLER_OR_NATIVE_UNVERIFIED", "EVIDENCE_UNAVAILABLE", "UNVERIFIED", "Request could not be verified. Inspect originals and host status; local Doctor only if evidence remains unavailable.", reconnectNext())
	}
}

// The SDK wraps both typed handler errors and validation errors. Handle them
// at the response boundary, keeping success output schemas untouched and
// ensuring schema validation does not echo supplied secrets or raw values.
func (s *Service) redactToolError(ctx context.Context, req mcp.Request, result mcp.Result) {
	r, ok := result.(*mcp.CallToolResult)
	if !ok || r == nil || !r.IsError {
		return
	}
	q, ok := req.(*mcp.CallToolRequest)
	if !ok || q == nil {
		return
	}
	var f ToolFailure
	if e := r.GetError(); !errors.As(e, &f) {
		f = failure("CONTROLLER_OR_NATIVE_UNVERIFIED", "EVIDENCE_UNAVAILABLE", "UNVERIFIED", "Tool result could not be verified; inspect originals before further effects.", reconnectNext())
		// The pinned SDK uses this exact prefix before invoking any handler.
		// Never infer no effect from an unfamiliar error or result.
		if e != nil && strings.HasPrefix(e.Error(), "validating \"arguments\":") {
			f = failure("INVALID_INPUT", "CALLER_INPUT", "NOT_ADMITTED", "Arguments do not match the advertised schema; check required fields and types.", noNext("CORRECT_INPUT"))
		}
		// Detect key length without ever reflecting the supplied key.
		var args struct {
			Key *string `json:"idempotency_key"`
		}
		if f.Admission == "NOT_ADMITTED" && q.Params != nil && json.Unmarshal(q.Params.Arguments, &args) == nil && args.Key != nil && (len(*args.Key) < 16 || len(*args.Key) > 128) {
			f = keyToolFailure()
		}
	}
	// A credential revoked while a request was in flight must not receive
	// even a previously authorized original ID in an error response.
	if p, e := s.requestPrincipal(q); e != nil || s.Store.RequireActiveTokenID(ctx, p.TokenID) != nil {
		f = failure("AUTHORIZATION_REQUIRED", "AUTHORIZATION", "UNVERIFIED", "Authenticate again before inspecting originals.", noNext("AUTHENTICATE"))
	}
	r.Content = []mcp.Content{&mcp.TextContent{Text: f.Error()}}
	r.StructuredContent = nil
	if r.Meta == nil {
		r.Meta = mcp.Meta{}
	}
	r.Meta["io.boxweave.error"] = f
}
