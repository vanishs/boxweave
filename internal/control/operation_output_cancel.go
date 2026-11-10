package control

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/state"
)

type operationOutputInput struct {
	OperationID string `json:"operation_id"`
	Stream      string `json:"stream"`
	Offset      int64  `json:"offset,omitempty"`
	Limit       int64  `json:"limit,omitempty"`
}

type operationOutputResult struct {
	TrackingID     string `json:"tracking_id"`
	OperationID    string `json:"operation_id"`
	Stream         string `json:"stream"`
	ContentBase64  string `json:"content_base64"`
	Offset         int64  `json:"offset"`
	NextOffset     int64  `json:"next_offset"`
	Ended          bool   `json:"ended"`
	Truncated      bool   `json:"truncated"`
	ExitCode       *int   `json:"exit_code,omitempty"`
	CommandSuccess *bool  `json:"command_success,omitempty"`
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
	mcp.AddTool(server, &mcp.Tool{
		Name:        "operation_output_get",
		Description: "Read up to 65536 bytes from the original recorded guest Exec stdout or stderr by offset, with observed nonzero exit retained.",
		Annotations: readToolAnnotations(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in operationOutputInput) (*mcp.CallToolResult, operationOutputResult, error) {
		actor, role, layer, err := s.operationPrincipal(req)
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
		original, err := s.Store.GetScopedOperation(ctx, layer, actor, role, in.OperationID)
		if err != nil {
			return nil, operationOutputResult{}, err
		}
		if original.Action != "box_exec" || original.BoxUUID == "" {
			return nil, operationOutputResult{}, state.ErrNotFound
		}
		box, err := s.Store.OwnedBox(ctx, layer, original.BoxUUID)
		if err != nil {
			return nil, operationOutputResult{}, err
		}
		if _, _, err = s.Native.GetExactBox(ctx, box.Name, box.UUID); err != nil {
			return nil, operationOutputResult{}, err
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
			Stream: in.Stream, Offset: in.Offset, Truncated: out.Truncated, ExitCode: out.ExitCode}
		if out.ExitCode != nil {
			ok := *out.ExitCode == 0
			result.CommandSuccess = &ok
		}
		if path == "" {
			result.Ended = true
			return &mcp.CallToolResult{}, result, nil
		}
		if in.Offset == native.MaxRecordedOutputBytes {
			result.Ended, result.Truncated = true, true
			result.NextOffset = in.Offset
			return &mcp.CallToolResult{}, result, nil
		}
		data, ended, err := s.Native.ReadRecordedOutput(ctx, path, in.Offset, in.Limit)
		if err != nil {
			return nil, operationOutputResult{}, err
		}
		result.ContentBase64 = base64.StdEncoding.EncodeToString(data)
		result.NextOffset = in.Offset + int64(len(data))
		result.Ended = ended
		if result.NextOffset >= native.MaxRecordedOutputBytes && !ended {
			result.Ended, result.Truncated = true, true
		}
		return &mcp.CallToolResult{}, result, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "operation_cancel",
		Description: "Journal a separate cancellation request for a proven queued-undispatched owned Operation. Detached running Exec/native operations are not cancelable with the proved SDK path.",
		Annotations: mutationAnnotations(true),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in cancelOperationInput) (*mcp.CallToolResult, cancelOperationResult, error) {
		actor, layer, err := requireAdmin(req, s)
		if err != nil {
			return nil, cancelOperationResult{}, err
		}
		target, err := s.Store.GetScopedOperation(ctx, layer, actor, "admin", in.TargetOperationID)
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
			if err := s.Store.CancelQueued(ctx, layer, target.ID); err != nil {
				return "", state.PreDispatchError{Class: "ORIGINAL_NOT_QUEUED_CANCEL_UNSUPPORTED"}
			}
			return "", state.ErrNoNativeChange
		}
		verify := func(ctx context.Context) error {
			got, err := s.Store.GetScopedOperation(ctx, layer, actor, "admin", target.ID)
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
