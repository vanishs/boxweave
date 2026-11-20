package control

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type operationOutputInput struct {
	OperationID string `json:"operation_id"`
	Stream      string `json:"stream"`
	Offset      int64  `json:"offset,omitempty"`
	Limit       int64  `json:"limit,omitempty"`
}

type operationOutputResult struct {
	TrackingID        string `json:"tracking_id"`
	OperationID       string `json:"operation_id"`
	Stream            string `json:"stream"`
	ContentBase64     string `json:"content_base64"`
	Offset            int64  `json:"offset"`
	NextOffset        int64  `json:"next_offset"`
	Ended             bool   `json:"ended"`
	Truncated         bool   `json:"truncated"`
	Availability      string `json:"availability"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	ExitCode          *int   `json:"exit_code,omitempty"`
	CommandSuccess    *bool  `json:"command_success,omitempty"`
}

type cancelOperationInput struct {
	TargetOperationID string `json:"target_operation_id"`
	IdempotencyKey    string `json:"idempotency_key"`
	WorkRef           string `json:"work_ref,omitempty"`
	AgentRef          string `json:"agent_ref,omitempty"`
}

type cancelOperationResult struct {
	TrackingID        string        `json:"tracking_id"`
	TargetOperationID string        `json:"target_operation_id"`
	CancelOperation   state.Receipt `json:"cancel_operation"`
}

func (s *Service) registerOperationOutputCancel(server *mcp.Server) {
	addStrictTool(server, &mcp.Tool{
		Name:        "operation_output_get",
		Description: "Read bounded 7-day private archived box_exec stdout/stderr by original operation_id even after Box Delete; fall back to original native recording if no archive exists. Expiration, missing output and truncation are explicit; command_success uses the separately recorded exit_code. Advance next_offset for pagination. Each stream is capped at 1MiB in the private archive.",
		Annotations: readToolAnnotations(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in operationOutputInput) (*mcp.CallToolResult, operationOutputResult, error) {
		principal, err := s.requestPrincipal(req)
		if err != nil {
			return nil, operationOutputResult{}, err
		}
		if (in.Stream != "stdout" && in.Stream != "stderr") ||
			in.Offset < 0 || in.Offset > native.MaxRecordedOutputBytes ||
			in.Limit < 0 || in.Limit > 65536 {
			return nil, operationOutputResult{}, state.ErrInvalid
		}
		if in.Limit == 0 {
			in.Limit = 16384
		}
		original, err := s.Store.GetScopedOperation(ctx, principal.Layer, principal.ActorID, in.OperationID)
		if err != nil {
			return nil, operationOutputResult{}, err
		}
		if original.Action != "box_exec" || original.BoxUUID == "" {
			return nil, operationOutputResult{}, state.ErrNotFound
		}
		out, err := s.Store.ExecOutput(ctx, original.ID)
		if err != nil {
			return nil, operationOutputResult{}, err
		}
		path := out.StdoutPath
		if in.Stream == "stderr" {
			path = out.StderrPath
		}
		tracking, err := newTrackingID()
		if err != nil {
			return nil, operationOutputResult{}, err
		}
		result := operationOutputResult{TrackingID: tracking, OperationID: original.ID,
			Stream: in.Stream, Offset: in.Offset, Truncated: out.Truncated, ExitCode: out.ExitCode, Availability: "available"}
		if out.ExitCode != nil {
			ok := *out.ExitCode == 0
			result.CommandSuccess = &ok
		}
		gcState, err := s.Store.ExecGCStreamState(ctx, original.ID, in.Stream)
		if err != nil {
			return nil, operationOutputResult{}, err
		}
		if gcState == "EXPIRED" {
			result.Ended, result.NextOffset = true, in.Offset
			result.Availability = "expired"
			result.UnavailableReason = "RETENTION_EXPIRED_VERIFIED"
			return privateReadReply(ctx, s, principal.TokenID, result)
		}
		// The controller archive is independent of the Incus instance and
		// must be checked before any live-Box lookup. TTL remains anchored
		// to the immutable Exec witness, not to a caller's read time.
		if out.ObservedAt > 0 && time.Now().UnixNano()-out.ObservedAt >= state.ExecRetention.Nanoseconds() {
			result.Ended, result.NextOffset = true, in.Offset
			result.Availability = "expired"
			result.UnavailableReason = "RETENTION_WINDOW_ELAPSED"
			return privateReadReply(ctx, s, principal.TokenID, result)
		}
		archive, ended, truncated, found, archiveErr := s.readExecArchive(original.ID, in.Stream, in.Offset, in.Limit)
		if archiveErr != nil {
			// A corrupt/private archive is a security failure, never a cue
			// to switch to a less-controlled arbitrary path.
			return nil, operationOutputResult{}, archiveErr
		}
		if found {
			result.ContentBase64 = base64.StdEncoding.EncodeToString(archive)
			result.NextOffset = in.Offset + int64(len(archive))
			result.Ended = ended
			result.Truncated = result.Truncated || truncated
			return privateReadReply(ctx, s, principal.TokenID, result)
		}
		box, err := s.Store.OwnedBox(ctx, principal.Layer, original.BoxUUID)
		if errors.Is(err, state.ErrNotFound) {
			result.Availability, result.UnavailableReason, result.Ended = "unavailable", "BOX_REMOVED_OUTPUT_NOT_ARCHIVED", true
			result.NextOffset = in.Offset
			return privateReadReply(ctx, s, principal.TokenID, result)
		}
		if err != nil {
			return nil, operationOutputResult{}, err
		}
		if _, _, err = s.Native.GetExactBox(ctx, box.Name, box.UUID); err != nil {
			result.Availability, result.UnavailableReason, result.Ended = "unavailable", "EXACT_NATIVE_BOX_NOT_AVAILABLE", true
			result.NextOffset = in.Offset
			return privateReadReply(ctx, s, principal.TokenID, result)
		}
		if path == "" {
			result.Ended = true
			result.Availability = "unavailable"
			result.UnavailableReason = "NO_NATIVE_RECORDED_STREAM_PATH"
			return privateReadReply(ctx, s, principal.TokenID, result)
		}
		if in.Offset == native.MaxRecordedOutputBytes {
			result.Ended, result.Truncated = true, true
			result.NextOffset = in.Offset
			return privateReadReply(ctx, s, principal.TokenID, result)
		}
		data, ended, err := s.Native.ReadRecordedOutput(ctx, path, in.Offset, in.Limit)
		if errors.Is(err, native.ErrRecordedOutputUnavailable) {
			result.Availability = "unavailable"
			result.UnavailableReason = "NATIVE_RECORDED_OUTPUT_MISSING"
			result.Ended = true
			result.NextOffset = in.Offset
			return privateReadReply(ctx, s, principal.TokenID, result)
		}
		if err != nil {
			return nil, operationOutputResult{}, err
		}
		result.ContentBase64 = base64.StdEncoding.EncodeToString(data)
		result.NextOffset = in.Offset + int64(len(data))
		result.Ended = ended
		if result.NextOffset >= native.MaxRecordedOutputBytes && !ended {
			result.Ended, result.Truncated = true, true
		}
		return privateReadReply(ctx, s, principal.TokenID, result)
	})

	addStrictTool(server, &mcp.Tool{
		Name:        "operation_cancel",
		Description: "Journal a separate cancellation request for a proven queued-undispatched owned Operation. Detached running Exec/native operations are not cancelable with the proved SDK path.",
		Annotations: mutationAnnotations(true),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in cancelOperationInput) (*mcp.CallToolResult, cancelOperationResult, error) {
		principal, err := s.requestPrincipal(req)
		if err != nil {
			return nil, cancelOperationResult{}, err
		}
		target, err := s.Store.GetScopedOperation(ctx, principal.Layer, principal.ActorID, in.TargetOperationID)
		if err != nil {
			return nil, cancelOperationResult{}, err
		}
		// Journal independently before acting. Retried keys return their original
		// cancellation receipt even when the target is now terminal.
		r, err := s.admitNative(ctx, req, "operation_cancel", in.IdempotencyKey,
			"", "", "cancel:"+target.ID, in.WorkRef, in.AgentRef,
			struct {
				Target string `json:"target_operation_id"`
			}{target.ID})
		if err != nil {
			return nil, cancelOperationResult{}, err
		}
		effect := func(ctx context.Context) (string, error) {
			if target.State != "ADMITTED" {
				return "", state.PreDispatchError{Class: "ORIGINAL_ALREADY_DISPATCHED_CANCEL_UNSUPPORTED"}
			}
			if err := s.Store.CancelQueued(ctx, principal.Layer, target.ID); err != nil {
				return "", state.PreDispatchError{Class: "ORIGINAL_NOT_QUEUED_CANCEL_UNSUPPORTED"}
			}
			return "", state.ErrNoNativeChange
		}
		verify := func(ctx context.Context) error {
			got, err := s.Store.GetScopedOperation(ctx, principal.Layer, principal.ActorID, target.ID)
			if err != nil {
				return err
			}
			if got.State != "CANCELLED" || got.NativeOperationID != "" {
				return errors.New("original queued cancellation unverified")
			}
			return nil
		}
		r, err = s.submitNative(r, effect, verify)
		return &mcp.CallToolResult{}, cancelOperationResult{
			TrackingID: r.TrackingID, TargetOperationID: target.ID, CancelOperation: r}, err
	})
}
