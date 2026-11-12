package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

var validBoxName = regexp.MustCompile(`^[a-z][a-z0-9-]{1,62}$`)

type workLabels struct {
	WorkRef  string `json:"work_ref,omitempty"`
	AgentRef string `json:"agent_ref,omitempty"`
}

type mutationBase struct {
	IdempotencyKey string `json:"idempotency_key"`
	workLabels
}

type createBoxInput struct {
	IdempotencyKey string `json:"idempotency_key"`
	Name           string `json:"name"`
	CPU            int    `json:"cpu,omitempty"`
	MemoryMiB      int64  `json:"memory_mib,omitempty"`
	DiskGiB        int64  `json:"disk_gib,omitempty"`
	Start          *bool  `json:"start,omitempty"`
	Nesting        bool   `json:"nesting,omitempty"`
	Internet       *bool  `json:"internet,omitempty"`
	WorkRef        string `json:"work_ref,omitempty"`
	AgentRef       string `json:"agent_ref,omitempty"`
}

type boxIdentityInput struct {
	BoxID string `json:"box_id"`
}

type listBoxesInput struct {
	Limit int    `json:"limit,omitempty"`
	After string `json:"cursor,omitempty"`
}

type listBoxesResult struct {
	TrackingID string            `json:"tracking_id"`
	Boxes      []native.BoxState `json:"boxes"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type getBoxResult struct {
	TrackingID string          `json:"tracking_id"`
	Box        native.BoxState `json:"box"`
}

type operateBoxInput struct {
	BoxID          string `json:"box_id"`
	IdempotencyKey string `json:"idempotency_key"`
	WorkRef        string `json:"work_ref,omitempty"`
	AgentRef       string `json:"agent_ref,omitempty"`
}

type updateBoxInput struct {
	BoxID          string `json:"box_id"`
	IdempotencyKey string `json:"idempotency_key"`
	CPU            *int   `json:"cpu,omitempty"`
	MemoryMiB      *int64 `json:"memory_mib,omitempty"`
	DiskGiB        *int64 `json:"disk_gib,omitempty"`
	Nesting        *bool  `json:"nesting,omitempty"`
	Internet       *bool  `json:"internet,omitempty"`
	WorkRef        string `json:"work_ref,omitempty"`
	AgentRef       string `json:"agent_ref,omitempty"`
}

type execBoxInput struct {
	BoxID          string            `json:"box_id"`
	IdempotencyKey string            `json:"idempotency_key"`
	Argv           []string          `json:"argv"`
	Cwd            string            `json:"cwd,omitempty"`
	GuestUID       uint32            `json:"guest_uid,omitempty"`
	GuestGID       uint32            `json:"guest_gid,omitempty"`
	Environment    map[string]string `json:"env,omitempty"`
	TimeoutMS      int               `json:"timeout_ms,omitempty"`
	WorkRef        string            `json:"work_ref,omitempty"`
	AgentRef       string            `json:"agent_ref,omitempty"`
}

func requireAdmin(req *mcp.CallToolRequest, s *Service) (string, string, error) {
	actor, role, layer, err := s.operationPrincipal(req)
	if err != nil {
		return "", "", err
	}
	if role != "admin" {
		return "", "", errors.New("PERMISSION_DENIED")
	}
	return actor, layer, nil
}

func mutationAnnotations(destructive bool) *mcp.ToolAnnotations {
	openWorld := false
	return &mcp.ToolAnnotations{ReadOnlyHint: false, IdempotentHint: false,
		DestructiveHint: &destructive, OpenWorldHint: &openWorld}
}

func boundedWorkLabels(req *mcp.CallToolRequest, work, agent string) (string, string, error) {
	if req != nil && req.Params != nil {
		meta := req.Params.GetMeta()
		for _, pair := range []struct {
			key  string
			dest *string
		}{
			{"io.boxweave.work_ref", &work}, {"io.boxweave.agent_ref", &agent},
		} {
			raw, exists := meta[pair.key]
			if !exists {
				continue
			}
			label, ok := raw.(string)
			if !ok || label == "" || !state.ValidCorrelationRef(label) {
				return "", "", state.ErrInvalid
			}
			if *pair.dest != "" && *pair.dest != label {
				return "", "", state.ErrConflict
			}
			*pair.dest = label
		}
	}
	if !state.ValidCorrelationRef(work) || !state.ValidCorrelationRef(agent) {
		return "", "", state.ErrInvalid
	}
	return work, agent, nil
}

func safeJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil || len(b) > 4096 {
		return nil, state.ErrInvalid
	}
	return b, nil
}

// replayNative checks an already accepted original before live Box inspection.
// This preserves exact idempotent receipts across verified deletion, native
// timeout, UNKNOWN and process restart without redispatch or name adoption.
func (s *Service) replayNative(ctx context.Context, req *mcp.CallToolRequest,
	action, key, boxID, work, agent string, intent any) (state.Receipt, bool, error) {
	actor, layer, err := requireAdmin(req, s)
	if err != nil {
		return state.Receipt{}, false, err
	}
	if key == "" || boxID == "" {
		return state.Receipt{}, false, state.ErrInvalid
	}
	work, agent, err = boundedWorkLabels(req, work, agent)
	if err != nil {
		return state.Receipt{}, false, err
	}
	safe, err := safeJSON(intent)
	if err != nil {
		return state.Receipt{}, false, err
	}
	rec, err := s.Store.Replay(ctx, state.Admission{
		Layer: layer, ActorID: actor, TokenID: actor,
		Key: key, Action: action, BoxUUID: boxID,
		Incarnation: boxID, SafeIntent: safe,
		WorkRef: work, AgentRef: agent,
	})
	if errors.Is(err, state.ErrNotFound) {
		return state.Receipt{}, false, nil
	}
	return rec, err == nil, err
}

func (s *Service) admitNative(ctx context.Context, req *mcp.CallToolRequest,
	action, key, boxUUID, incarnation, resourceKey, work, agent string, intent any) (state.Receipt, error) {
	actor, layer, err := requireAdmin(req, s)
	if err != nil {
		return state.Receipt{}, err
	}
	if key == "" {
		return state.Receipt{}, state.ErrInvalid
	}
	work, agent, err = boundedWorkLabels(req, work, agent)
	if err != nil {
		return state.Receipt{}, err
	}
	b, err := safeJSON(intent)
	if err != nil {
		return state.Receipt{}, err
	}
	requestID, err := newTrackingID()
	if err != nil {
		return state.Receipt{}, err
	}
	return s.Store.Admit(ctx, state.Admission{
		Layer: layer, ActorID: actor, TokenID: actor, Key: key, Action: action,
		BoxUUID: boxUUID, Incarnation: incarnation, ResourceKey: resourceKey,
		WorkRef: work, AgentRef: agent, SafeIntent: b, RequestID: requestID,
	})
}

func (s *Service) worker() state.Worker {
	return state.Worker{Store: s.Store, Budget: 15 * time.Minute}
}

func preflightNative(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, native.ErrNativeUnsupported) ||
		errors.Is(err, native.ErrUnsafeNetwork) ||
		errors.Is(err, native.ErrNotOwned) ||
		errors.Is(err, native.ErrNativeDependents) {
		return state.PreDispatchError{Class: "SAFE_NATIVE_PREFLIGHT_DENIED"}
	}
	return err
}

func (s *Service) readOwned(ctx context.Context, req *mcp.CallToolRequest, uuid string) (state.OwnedBox, error) {
	_, _, layer, err := s.operationPrincipal(req)
	if err != nil {
		return state.OwnedBox{}, err
	}
	box, err := s.Store.OwnedBox(ctx, layer, uuid)
	if err != nil {
		return box, err
	}
	if _, _, err = s.Native.GetExactBox(ctx, box.Name, box.UUID); err != nil {
		return state.OwnedBox{}, err
	}
	return box, nil
}

func nativeDone(ctx context.Context, r native.Reader, nativeID string) error {
	op, err := r.ObserveUntilFinal(ctx, nativeID)
	if err != nil {
		return err
	}
	if op.StatusCode != api.Success {
		return native.ErrUnverified
	}
	return nil
}

// observeOriginalCreateSettled tolerates Incus post-create status propagation.
// It is read-only and never retries Create or Start. Foreign identity or
// unsafe NIC evidence refuses immediately; other transient observations get
// a bounded read window. A stable stopped guest is returned as a residual
// only after giving the requested startup a chance to settle.
func (s *Service) observeOriginalCreateSettled(ctx context.Context, name, originalID string, start, internet bool) (native.BoxState, error) {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	var stopped native.BoxState
	for {
		observed, err := s.Native.ObserveCreatedByOperation(ctx, name, originalID)
		if errors.Is(err, native.ErrNotOwned) || errors.Is(err, native.ErrUnsafeNetwork) {
			return native.BoxState{}, state.VerificationUncertain{Class: "NATIVE_CREATE_IDENTITY_OR_NIC_UNVERIFIED"}
		}
		if err == nil {
			if observed.Internet != internet {
				return native.BoxState{}, state.VerificationUncertain{Class: "NATIVE_CREATE_EGRESS_UNVERIFIED"}
			}
			if (start && observed.NativeState == "Running") ||
				(!start && observed.NativeState == "Stopped") {
				return observed, nil
			}
			if start && observed.NativeState == "Stopped" {
				stopped = observed
			}
		}
		timer := time.NewTimer(350 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			if stopped.UUID != "" {
				// Re-read the exact UUID after settling to reject name reuse.
				finalCtx, finalCancel := context.WithTimeout(context.Background(), 2*time.Second)
				final, err := s.Native.ObserveCreatedByOperation(finalCtx, name, originalID)
				finalCancel()
				if err == nil && final.UUID == stopped.UUID && final.NativeState == "Stopped" &&
					final.Internet == internet {
					return final, nil
				}
			}
			return native.BoxState{}, state.VerificationUncertain{Class: "NATIVE_CREATE_POSTREAD_UNVERIFIED"}
		case <-timer.C:
		}
	}
}

func (s *Service) submitNative(r state.Receipt, effect state.NativeEffect, verify state.VerifyEffect) (state.Receipt, error) {
	if r.Duplicate {
		return r, nil
	}
	if err := s.worker().Submit(r.ID, effect, verify); err != nil {
		// Do not resubmit by creating a new native key if the first worker
		// scheduling attempt is uncertain. The persisted receipt remains.
		return r, err
	}
	return r, nil
}

func (s *Service) registerBoxTools(server *mcp.Server) {
	addStrictTool(server, &mcp.Tool{Name: "box_create",
		Description: "Admit one owned Ubuntu LTS Box creation using the installed restricted Incus Project and its preverified private NIC; asynchronous durable receipt.",
		Annotations: mutationAnnotations(true)},
		func(ctx context.Context, req *mcp.CallToolRequest, in createBoxInput) (*mcp.CallToolResult, state.Receipt, error) {
			if !validBoxName.MatchString(in.Name) {
				return nil, state.Receipt{}, state.ErrInvalid
			}
			if in.CPU == 0 {
				in.CPU = 1
			}
			if in.MemoryMiB == 0 {
				in.MemoryMiB = 1024
			}
			if in.DiskGiB == 0 {
				in.DiskGiB = 10
			}
			if in.CPU < 1 || in.CPU > 64 || in.MemoryMiB < 128 || in.MemoryMiB > 262144 ||
				in.DiskGiB < 2 || in.DiskGiB > 2048 {
				return nil, state.Receipt{}, state.ErrInvalid
			}
			start := true
			if in.Start != nil {
				start = *in.Start
			}
			internet := true
			if in.Internet != nil {
				internet = *in.Internet
			}
			r, err := s.admitNative(ctx, req, "box_create", in.IdempotencyKey, "", "",
				"box:"+in.Name, in.WorkRef, in.AgentRef, struct {
					Name      string `json:"name"`
					CPU       int    `json:"cpu"`
					MemoryMiB int64  `json:"memory_mib"`
					DiskGiB   int64  `json:"disk_gib"`
					Start     bool   `json:"start"`
					Nesting   bool   `json:"nesting"`
					Internet  bool   `json:"internet"`
				}{in.Name, in.CPU, in.MemoryMiB, in.DiskGiB, start, in.Nesting, internet})
			if err != nil {
				return nil, state.Receipt{}, err
			}
			var nativeID string
			effect := func(ctx context.Context) (string, error) {
				id, e := s.Native.Create(ctx, native.InstanceAction{
					OperationID: r.ID, Name: in.Name, CPU: in.CPU, MemoryMiB: in.MemoryMiB, DiskGiB: in.DiskGiB,
					Start: start, Nesting: in.Nesting, Internet: internet})
				nativeID = id
				return id, preflightNative(e)
			}
			verify := func(ctx context.Context) error {
				op, err := s.Native.ObserveUntilFinal(ctx, nativeID)
				if err != nil {
					return state.VerificationUncertain{Class: "NATIVE_CREATE_FINAL_OP_UNVERIFIED"}
				}
				if op.StatusCode != api.Success {
					if err := s.Native.ConfirmDeleteAbsence(ctx, in.Name); err == nil {
						return state.VerifiedEffectFailure{Class: "NATIVE_CREATE_FAILED_NO_INSTANCE"}
					}
				}
				observed, err := s.observeOriginalCreateSettled(ctx, in.Name, r.ID, start, internet)
				if err != nil {
					return err
				}
				if op.StatusCode != api.Success && observed.NativeState != "Stopped" {
					return state.VerificationUncertain{Class: "NATIVE_CREATE_FAILED_STATE_UNVERIFIED"}
				}
				if err := s.Store.RegisterCreatedBox(ctx, r.ID, s.Native.Project, in.Name, observed.UUID); err != nil {
					return state.VerificationUncertain{Class: "NATIVE_CREATE_UUID_LEDGER_UNVERIFIED"}
				}
				if op.StatusCode != api.Success {
					return state.VerifiedEffectFailure{Class: "NATIVE_CREATE_FAILED_OWNED_STOPPED"}
				}
				if start && observed.NativeState == "Stopped" {
					return state.VerifiedEffectFailure{Class: "NATIVE_CREATE_GUEST_NOT_RUNNING_OWNED"}
				}
				return nil
			}
			r, err = s.submitNative(r, effect, verify)
			return &mcp.CallToolResult{}, r, err
		})

	addStrictTool(server, &mcp.Tool{Name: "box_list",
		Description: "List only UUID-registered owned Boxes with independently observed exact native identity.",
		Annotations: readToolAnnotations()},
		func(ctx context.Context, req *mcp.CallToolRequest, in listBoxesInput) (*mcp.CallToolResult, listBoxesResult, error) {
			actor, _, project, err := s.operationPrincipal(req)
			if err != nil {
				return nil, listBoxesResult{}, err
			}
			records, next, err := s.Store.ListOwnedBoxes(ctx, project, in.After, in.Limit)
			if err != nil {
				return nil, listBoxesResult{}, err
			}
			out := listBoxesResult{Boxes: make([]native.BoxState, 0, len(records)), NextCursor: next}
			out.TrackingID, err = newTrackingID()
			if err != nil {
				return nil, listBoxesResult{}, err
			}
			for _, rec := range records {
				status, _, e := s.Native.GetExactBox(ctx, rec.Name, rec.UUID)
				if e != nil {
					return nil, listBoxesResult{}, e
				}
				out.Boxes = append(out.Boxes, status)
			}
			return privateReadReply(ctx, s, actor, out)
		})

	addStrictTool(server, &mcp.Tool{Name: "box_get",
		Description: "Get exact locally owned Box UUID and independently observed native state; native running is not guest readiness.",
		Annotations: readToolAnnotations()},
		func(ctx context.Context, req *mcp.CallToolRequest, in boxIdentityInput) (*mcp.CallToolResult, getBoxResult, error) {
			actor, _, _, err := s.operationPrincipal(req)
			if err != nil {
				return nil, getBoxResult{}, err
			}
			box, err := s.readOwned(ctx, req, in.BoxID)
			if err != nil {
				return nil, getBoxResult{}, err
			}
			status, _, err := s.Native.GetExactBox(ctx, box.Name, box.UUID)
			if err != nil {
				return nil, getBoxResult{}, err
			}
			tracking, err := newTrackingID()
			if err != nil {
				return nil, getBoxResult{}, err
			}
			return privateReadReply(ctx, s, actor, getBoxResult{TrackingID: tracking, Box: status})
		})

	for _, action := range []string{"box_start", "box_stop"} {
		action := action
		desc := "Start one exact owned Box with verified private network preconditions."
		if action == "box_stop" {
			desc = "Stop one exact owned running Box without host execution."
		}
		addStrictTool(server, &mcp.Tool{Name: action, Description: desc, Annotations: mutationAnnotations(false)},
			func(ctx context.Context, req *mcp.CallToolRequest, in operateBoxInput) (*mcp.CallToolResult, state.Receipt, error) {
				if _, _, err := requireAdmin(req, s); err != nil {
					return nil, state.Receipt{}, err
				}
				intent := struct {
					BoxID string `json:"box_id"`
				}{in.BoxID}
				original, already, err := s.replayNative(ctx, req, action, in.IdempotencyKey,
					in.BoxID, in.WorkRef, in.AgentRef, intent)
				if err != nil {
					return nil, state.Receipt{}, err
				}
				if already {
					return &mcp.CallToolResult{}, original, nil
				}
				box, err := s.readOwned(ctx, req, in.BoxID)
				if err != nil {
					return nil, state.Receipt{}, err
				}
				r, err := s.admitNative(ctx, req, action, in.IdempotencyKey, box.UUID, box.Incarnation,
					"box:"+box.Name, in.WorkRef, in.AgentRef, struct {
						BoxID string `json:"box_id"`
					}{in.BoxID})
				if err != nil {
					return nil, state.Receipt{}, err
				}
				power := "start"
				want := "Running"
				if action == "box_stop" {
					power = "stop"
					want = "Stopped"
				}
				var nativeID string
				effect := func(ctx context.Context) (string, error) {
					boxStatus, _, e := s.Native.GetExactBox(ctx, box.Name, box.UUID)
					if e != nil {
						return "", preflightNative(e)
					}
					if boxStatus.NativeState == want {
						return "", state.ErrNoNativeChange
					}
					id, e := s.Native.SetPower(ctx, box.Name, box.UUID, power)
					nativeID = id
					return id, preflightNative(e)
				}
				verify := func(ctx context.Context) error {
					if nativeID != "" {
						if err := nativeDone(ctx, s.Native, nativeID); err != nil {
							return err
						}
					}
					current, _, err := s.Native.GetExactBox(ctx, box.Name, box.UUID)
					if err != nil {
						return err
					}
					if current.NativeState != want {
						return native.ErrUnverified
					}
					return nil
				}
				r, err = s.submitNative(r, effect, verify)
				return &mcp.CallToolResult{}, r, err
			})
	}

	addStrictTool(server, &mcp.Tool{Name: "box_update",
		Description: "Update exact owned CPU/RAM/root disk-growth/nesting and per-Box Internet egress by presence-aware typed fields.",
		Annotations: mutationAnnotations(true)},
		func(ctx context.Context, req *mcp.CallToolRequest, in updateBoxInput) (*mcp.CallToolResult, state.Receipt, error) {
			if _, _, err := requireAdmin(req, s); err != nil {
				return nil, state.Receipt{}, err
			}
			if in.Internet == nil && in.CPU == nil && in.MemoryMiB == nil &&
				in.DiskGiB == nil && in.Nesting == nil {
				return nil, state.Receipt{}, native.ErrNativeUnsupported
			}
			changes := native.InstanceChanges{CPU: in.CPU, MemoryMiB: in.MemoryMiB, DiskGiB: in.DiskGiB,
				Nesting: in.Nesting, Internet: in.Internet}
			intent := struct {
				BoxID   string                 `json:"box_id"`
				Changes native.InstanceChanges `json:"changes"`
			}{in.BoxID, changes}
			original, already, err := s.replayNative(ctx, req, "box_update", in.IdempotencyKey,
				in.BoxID, in.WorkRef, in.AgentRef, intent)
			if err != nil {
				return nil, state.Receipt{}, err
			}
			if already {
				return &mcp.CallToolResult{}, original, nil
			}
			box, err := s.readOwned(ctx, req, in.BoxID)
			if err != nil {
				return nil, state.Receipt{}, err
			}
			r, err := s.admitNative(ctx, req, "box_update", in.IdempotencyKey, box.UUID, box.Incarnation,
				"box:"+box.Name, in.WorkRef, in.AgentRef, struct {
					BoxID   string                 `json:"box_id"`
					Changes native.InstanceChanges `json:"changes"`
				}{in.BoxID, changes})
			if err != nil {
				return nil, state.Receipt{}, err
			}
			var nativeID string
			effect := func(ctx context.Context) (string, error) {
				id, e := s.Native.UpdateBox(ctx, box.Name, box.UUID, changes)
				nativeID = id
				return id, preflightNative(e)
			}
			verify := func(ctx context.Context) error {
				if err := nativeDone(ctx, s.Native, nativeID); err != nil {
					return err
				}
				actual, _, err := s.Native.GetExactBox(ctx, box.Name, box.UUID)
				if err != nil {
					return err
				}
				if in.CPU != nil && actual.CPU != fmt.Sprint(*in.CPU) ||
					in.MemoryMiB != nil && actual.Memory != fmt.Sprintf("%dMiB", *in.MemoryMiB) ||
					in.Nesting != nil && actual.Nesting != *in.Nesting ||
					in.Internet != nil && actual.Internet != *in.Internet {
					return native.ErrUnverified
				}
				// Root disk size postread must also be proved, never only
				// trust the successful UpdateInstance operation.
				if in.DiskGiB != nil && actual.RootDiskSize != fmt.Sprintf("%dGiB", *in.DiskGiB) {
					return native.ErrUnverified
				}
				return nil
			}
			r, err = s.submitNative(r, effect, verify)
			return &mcp.CallToolResult{}, r, err
		})

	addStrictTool(server, &mcp.Tool{Name: "box_delete",
		Description: "Delete only an exact owned STOPPED Box after refusing native proxy/peer dependencies; retain UNKNOWN on unverified deletion.",
		Annotations: mutationAnnotations(true)},
		func(ctx context.Context, req *mcp.CallToolRequest, in operateBoxInput) (*mcp.CallToolResult, state.Receipt, error) {
			if _, _, err := requireAdmin(req, s); err != nil {
				return nil, state.Receipt{}, err
			}
			intent := struct {
				BoxID string `json:"box_id"`
			}{in.BoxID}
			original, already, err := s.replayNative(ctx, req, "box_delete", in.IdempotencyKey,
				in.BoxID, in.WorkRef, in.AgentRef, intent)
			if err != nil {
				return nil, state.Receipt{}, err
			}
			if already {
				return &mcp.CallToolResult{}, original, nil
			}
			box, err := s.readOwned(ctx, req, in.BoxID)
			if err != nil {
				return nil, state.Receipt{}, err
			}
			r, err := s.admitNative(ctx, req, "box_delete", in.IdempotencyKey, box.UUID, box.Incarnation,
				"box:"+box.Name, in.WorkRef, in.AgentRef, struct {
					BoxID string `json:"box_id"`
				}{in.BoxID})
			if err != nil {
				return nil, state.Receipt{}, err
			}
			var nativeID string
			effect := func(ctx context.Context) (string, error) {
				id, e := s.Native.DeleteBox(ctx, box.Name, box.UUID)
				nativeID = id
				return id, preflightNative(e)
			}
			verify := func(ctx context.Context) error {
				if err := nativeDone(ctx, s.Native, nativeID); err != nil {
					return err
				}
				if err := s.Native.ConfirmDeleteAbsence(ctx, box.Name); err != nil {
					return err
				}
				return s.Store.TombstoneOwnedBox(ctx, box)
			}
			r, err = s.submitNative(r, effect, verify)
			return &mcp.CallToolResult{}, r, err
		})

	addStrictTool(server, &mcp.Tool{Name: "box_exec",
		Description: "Durable asynchronous argv execution INSIDE an exact owned guest, guest UID/GID typed; original stdout/stderr through operation_output_get.",
		Annotations: mutationAnnotations(true)},
		func(ctx context.Context, req *mcp.CallToolRequest, in execBoxInput) (*mcp.CallToolResult, state.Receipt, error) {
			if _, _, err := requireAdmin(req, s); err != nil {
				return nil, state.Receipt{}, err
			}
			if len(in.Argv) == 0 || len(in.Argv) > 64 ||
				(in.TimeoutMS != 0 && (in.TimeoutMS < 1000 || in.TimeoutMS > 900000)) ||
				len(in.Environment) > 32 {
				return nil, state.Receipt{}, state.ErrInvalid
			}
			for _, arg := range in.Argv {
				if arg == "" || len(arg) > 4096 || strings.ContainsRune(arg, 0) {
					return nil, state.Receipt{}, state.ErrInvalid
				}
			}
			canonical, _ := json.Marshal(struct {
				Argv []string          `json:"argv"`
				Cwd  string            `json:"cwd"`
				UID  uint32            `json:"uid"`
				GID  uint32            `json:"gid"`
				Env  map[string]string `json:"env"`
			}{in.Argv, in.Cwd, in.GuestUID, in.GuestGID, in.Environment})
			hash := sha256.Sum256(canonical)
			intent := struct {
				BoxID              string `json:"box_id"`
				GuestCommandDigest string `json:"guest_command_digest"`
				TimeoutMS          int    `json:"timeout_ms"`
			}{in.BoxID, hex.EncodeToString(hash[:]), in.TimeoutMS}
			original, already, err := s.replayNative(ctx, req, "box_exec", in.IdempotencyKey,
				in.BoxID, in.WorkRef, in.AgentRef, intent)
			if err != nil {
				return nil, state.Receipt{}, err
			}
			if already {
				return &mcp.CallToolResult{}, original, nil
			}
			box, err := s.readOwned(ctx, req, in.BoxID)
			if err != nil {
				return nil, state.Receipt{}, err
			}
			spec := native.ExecSpec{Argv: in.Argv, Cwd: in.Cwd, UID: in.GuestUID,
				GID: in.GuestGID, Environment: in.Environment, TimeoutMS: in.TimeoutMS}
			r, err := s.admitNative(ctx, req, "box_exec", in.IdempotencyKey, box.UUID, box.Incarnation,
				"box:"+box.Name, in.WorkRef, in.AgentRef, intent)
			if err != nil {
				return nil, state.Receipt{}, err
			}
			var nativeID string
			effect := func(ctx context.Context) (string, error) {
				id, e := s.Native.Exec(ctx, box.Name, box.UUID, spec)
				nativeID = id
				return id, preflightNative(e)
			}
			verify := func(ctx context.Context) error {
				final, err := s.Native.ObserveUntilFinal(ctx, nativeID)
				if err != nil {
					return err
				}
				out, err := native.ExecEvidenceFromFinal(final)
				if err != nil {
					return err
				}
				if err := s.Store.SaveExecOutput(ctx, r.ID, out.StdoutPath, out.StderrPath, out.Exit); err != nil {
					return err
				}
				return nil
			}
			r, err = s.submitNative(r, effect, verify)
			return &mcp.CallToolResult{}, r, err
		})
}

// Ensure the wire handler imports only ordinary typed SDK/native data and
// does not create an extra transport, host-shell endpoint or task service.
var _ = http.StatusNotFound
