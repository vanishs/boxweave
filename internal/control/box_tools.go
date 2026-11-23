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
	"strconv"
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
	native.InstanceOptions
	IdempotencyKey string `json:"idempotency_key"`
	Name           string `json:"name"`
	CPU            int    `json:"cpu,omitempty"`
	MemoryMiB      int64  `json:"memory_mib,omitempty"`
	DiskGiB        int64  `json:"disk_gib,omitempty"`
	Image          string `json:"image,omitempty"`
	Start          *bool  `json:"start,omitempty"`
	Nesting        bool   `json:"security.nesting,omitempty" jsonschema:"Incus nesting; independent of LXC AppArmor generated-profile nesting. Does not automatically authorize safe recursive Incus."`
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

type getBoxInput struct {
	BoxID  string `json:"box_id"`
	Detail bool   `json:"detail,omitempty" jsonschema:"Set true to include runtime State, Snapshots, and Backups; otherwise return basic instance information."`
}

type getBoxResult struct {
	TrackingID   string            `json:"tracking_id"`
	Box          native.BoxState   `json:"box"`
	InstanceFull map[string]any    `json:"instance_full,omitempty"`
	ObservedAt   *time.Time        `json:"observed_at,omitempty"`
	Attention    attentionSnapshot `json:"attention"`
}

type operateBoxInput struct {
	BoxID          string `json:"box_id"`
	IdempotencyKey string `json:"idempotency_key"`
	WorkRef        string `json:"work_ref,omitempty"`
	AgentRef       string `json:"agent_ref,omitempty"`
}

type updateBoxInput struct {
	native.InstanceOptions
	BoxID          string `json:"box_id"`
	IdempotencyKey string `json:"idempotency_key"`
	CPU            *int   `json:"cpu,omitempty"`
	MemoryMiB      *int64 `json:"memory_mib,omitempty"`
	DiskGiB        *int64 `json:"disk_gib,omitempty"`
	Nesting        *bool  `json:"security.nesting,omitempty" jsonschema:"Toggle Incus nesting. Changing it does not grant a recursive AppArmor or UID/GID delegation capability."`
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
	principal, err := s.requestPrincipal(req)
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
		Layer: principal.Layer, ActorID: principal.ActorID, TokenID: principal.TokenID,
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
	principal, err := s.requestPrincipal(req)
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
		Layer: principal.Layer, ActorID: principal.ActorID, TokenID: principal.TokenID, Key: key, Action: action,
		BoxUUID: boxUUID, Incarnation: incarnation, ResourceKey: resourceKey,
		WorkRef: work, AgentRef: agent, SafeIntent: b, RequestID: requestID, CleanupDependencies: action == "box_delete",
	})
}

func (s *Service) worker() state.Worker {
	return state.Worker{Store: s.Store, Budget: 15 * time.Minute}
}

func preflightNative(err error) error {
	if err == nil {
		return nil
	}
	if class := native.CapabilityDenialClass(err); class != "" {
		return state.PreDispatchError{Class: class}
	}
	if errors.Is(err, native.ErrNativeExecQuotaLow) {
		return state.PreDispatchError{Class: "NATIVE_EXEC_SPOOL_QUOTA_HEADROOM_LOW"}
	}
	if errors.Is(err, native.ErrNativeUnsupported) ||
		errors.Is(err, native.ErrUnsafeNetwork) ||
		errors.Is(err, native.ErrNotOwned) ||
		errors.Is(err, native.ErrNativeDependents) {
		return state.PreDispatchError{Class: "SAFE_NATIVE_PREFLIGHT_DENIED"}
	}
	return err
}

func (s *Service) readOwnedLedger(ctx context.Context, req *mcp.CallToolRequest, uuid string) (state.OwnedBox, error) {
	principal, err := s.requestPrincipal(req)
	if err != nil {
		return state.OwnedBox{}, err
	}
	box, err := s.Store.OwnedBox(ctx, principal.Layer, uuid)
	if errors.Is(err, state.ErrNotFound) {
		// A missing/foreign UUID is deliberately indistinguishable. Do not
		// expose the backing Operation-store error as the user-facing Box type.
		return state.OwnedBox{}, fmt.Errorf("BOX_NOT_FOUND_OR_NOT_OWNED: %w", err)
	}
	if err != nil {
		return box, err
	}
	return box, nil
}

func (s *Service) readOwned(ctx context.Context, req *mcp.CallToolRequest, uuid string) (state.OwnedBox, error) {
	box, err := s.readOwnedLedger(ctx, req, uuid)
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
// A postread must prove the requested native shape as well as the user
// marker and effective ACL. A native operation that left a same-name but
// differently sized/resourced guest is NOT considered a complete Create.
func exactCreateNativeShape(g native.BoxState, in createBoxInput) bool {
	return g.CPU == strconv.Itoa(in.CPU) &&
		g.Memory == fmt.Sprintf("%dMiB", in.MemoryMiB) &&
		g.RootDiskSize == fmt.Sprintf("%dGiB", in.DiskGiB) &&
		g.Nesting == in.Nesting &&
		in.InstanceOptions.Matches(g.NativeOptions)
}

func (s *Service) observeOriginalCreateSettled(ctx context.Context, name, originalID string, start, internet, nativeFailed bool) (native.BoxState, error) {
	return settleOriginalCreateWithNativeFailure(ctx, 6*time.Second, func(readCtx context.Context) (native.BoxState, error) {
		return s.Native.ObserveCreatedByOperation(readCtx, name, originalID)
	}, start, internet, nativeFailed)
}

// The injected read-only observation makes the settle rules testable without
// inventing native operation replies or issuing additional mutations.
func settleOriginalCreate(ctx context.Context, budget time.Duration, observe func(context.Context) (native.BoxState, error), start, internet bool) (native.BoxState, error) {
	return settleOriginalCreateWithNativeFailure(ctx, budget, observe, start, internet, false)
}

// A terminal FAILED Incus Create operation must never be reconciled from an
// instantaneous Running observation: Incus can transiently report Running
// before a failed LXC start settles back to Stopped. Require two consistent
// STOPPED readbacks with the same native UUID, or leave the original UNKNOWN.
func settleOriginalCreateWithNativeFailure(ctx context.Context, budget time.Duration, observe func(context.Context) (native.BoxState, error), start, internet, nativeFailed bool) (native.BoxState, error) {
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	var stopped native.BoxState
	for {
		observed, err := observe(ctx)
		if errors.Is(err, native.ErrNotOwned) || errors.Is(err, native.ErrUnsafeNetwork) {
			return native.BoxState{}, state.VerificationUncertain{Class: "NATIVE_CREATE_IDENTITY_OR_NIC_UNVERIFIED"}
		}
		if err == nil {
			if observed.Internet != internet {
				return native.BoxState{}, state.VerificationUncertain{Class: "NATIVE_CREATE_EGRESS_UNVERIFIED"}
			}
			if nativeFailed {
				if observed.NativeState == "Stopped" && observed.UUID != "" {
					if stopped.UUID != "" && stopped.UUID != observed.UUID {
						return native.BoxState{}, state.VerificationUncertain{Class: "NATIVE_CREATE_IDENTITY_OR_NIC_UNVERIFIED"}
					}
					if stopped.UUID == observed.UUID && stopped.UUID != "" {
						return observed, nil
					}
					stopped = observed
				} else {
					stopped = native.BoxState{}
				}
			} else {
				if (start && observed.NativeState == "Running") ||
					(!start && observed.NativeState == "Stopped") {
					return observed, nil
				}
				if start && observed.NativeState == "Stopped" {
					stopped = observed
				}
			}
		}
		timer := time.NewTimer(350 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			if stopped.UUID != "" {
				// Re-read the exact UUID after settling to reject name reuse.
				finalCtx, finalCancel := context.WithTimeout(context.Background(), 2*time.Second)
				final, err := observe(finalCtx)
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

// Verify existing pin before and after a mutation; no implicit change to
// network bindings or address fences during ordinary power/resource operations.
func (s *Service) verifyLifecyclePin(ctx context.Context, id string, box state.OwnedBox) error {
	pin, err := s.Store.LifecyclePinnedNIC(ctx, id, box)
	if err != nil {
		return err
	}
	if pin == nil {
		return nil
	}
	return s.Native.VerifyPinnedDeleteNIC(ctx, box.Name, box.UUID, pin.Bridge, pin.ACL, pin.MAC, pin.IPv4, pin.IPv6)
}
func (s *Service) registerBoxTools(server *mcp.Server) {
	addStrictTool(server, &mcp.Tool{Name: "box_create",
		Description: "Admit creation of one owned container with typed Incus/LXC options; omitted booleans preserve defaults. Returns an asynchronous Operation, not guest readiness. Never disable host AppArmor; nested Incus requires independently verified Project/AppArmor/UID/GID constraints.",
		Annotations: mutationAnnotations(true)},
		func(ctx context.Context, req *mcp.CallToolRequest, in createBoxInput) (*mcp.CallToolResult, state.Receipt, error) {
			if !validBoxName.MatchString(in.Name) || !native.ValidImageChoice(in.Image) {
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
					Name      string                 `json:"name"`
					CPU       int                    `json:"cpu"`
					MemoryMiB int64                  `json:"memory_mib"`
					DiskGiB   int64                  `json:"disk_gib"`
					Image     string                 `json:"image,omitempty"`
					Start     bool                   `json:"start"`
					Nesting   bool                   `json:"nesting"`
					Internet  bool                   `json:"internet"`
					Options   native.InstanceOptions `json:"instance_options"`
				}{in.Name, in.CPU, in.MemoryMiB, in.DiskGiB, in.Image, start, in.Nesting, internet, in.InstanceOptions})
			if err != nil {
				return nil, state.Receipt{}, err
			}
			var nativeID string
			var reserved native.VerifiedStaticPeerBinding
			effect := func(ctx context.Context) (string, error) {
				id, e := s.Native.CreateWithPeerBinding(ctx, native.InstanceAction{
					OperationID: r.ID, Name: in.Name, CPU: in.CPU, MemoryMiB: in.MemoryMiB, DiskGiB: in.DiskGiB,
					Image: in.Image, Start: start, Nesting: in.Nesting, Internet: internet,
					Options: in.InstanceOptions}, func(b native.VerifiedStaticPeerBinding) (bool, error) {
					ok, err := s.Store.StageCreateBinding(ctx, r.ID, s.Native.Project, in.Name, state.VerifiedCreateBinding(b))
					if err == nil && ok {
						reserved = b
					}
					return ok, err
				}, s.Store.ReadCreateAddressHolds)
				nativeID = id
				return id, preflightNative(e)
			}
			verify := func(ctx context.Context) error {
				op, err := s.Native.ObserveUntilFinal(ctx, nativeID)
				originalNativeOutcomeMissing := false
				if api.StatusErrorCheck(err, http.StatusNotFound) {
					// Incus may discard a short Create Operation immediately
					// after finalization. Native 404 alone NEVER proves
					// success/failure: independently query the native active
					// Operation set twice, then verify exact postconditions.
					if err := s.Native.OriginalOperationInactive(ctx, nativeID); err != nil {
						return state.VerificationUncertain{Class: "NATIVE_CREATE_EXPIRED_OP_ACTIVE_OR_UNVERIFIED"}
					}
					originalNativeOutcomeMissing = true
				} else if err != nil {
					return state.VerificationUncertain{Class: "NATIVE_CREATE_FINAL_OP_UNVERIFIED"}
				}
				if !originalNativeOutcomeMissing && op.StatusCode != api.Success {
					if err := s.Native.ConfirmDeleteAbsence(ctx, in.Name); err == nil {
						return state.VerifiedEffectFailure{Class: "NATIVE_CREATE_FAILED_NO_INSTANCE"}
					}
				}
				observed, err := s.observeOriginalCreateSettled(ctx, in.Name, r.ID, start, internet,
					!originalNativeOutcomeMissing && op.StatusCode != api.Success)
				if err != nil {
					return err
				}
				if !exactCreateNativeShape(observed, in) {
					return state.VerificationUncertain{Class: "NATIVE_CREATE_RESOURCE_SHAPE_UNVERIFIED"}
				}
				if originalNativeOutcomeMissing {
					if err := s.Native.OriginalOperationInactive(ctx, nativeID); err != nil {
						return state.VerificationUncertain{Class: "NATIVE_CREATE_EXPIRED_OP_REAPPEARED"}
					}
				} else if op.StatusCode != api.Success && observed.NativeState != "Stopped" {
					return state.VerificationUncertain{Class: "NATIVE_CREATE_FAILED_STATE_UNVERIFIED"}
				}
				// Only the exact reserved static tuple, independently observed twice,
				// may be attached to the original Create UUID. Bounded read-only
				// settle handles lease publication lag, never retries Create.
				var proof native.VerifiedStaticPeerBinding
				proved := false
				deadline := time.Now().Add(8 * time.Second)
				for time.Now().Before(deadline) && ctx.Err() == nil {
					got, e := s.Native.ObserveOriginalCreateStaticPeerBinding(ctx, in.Name, observed.UUID, r.ID)
					if e == nil && got == reserved {
						proof = got
						proved = true
						break
					}
					select {
					case <-ctx.Done():
					case <-time.After(350 * time.Millisecond):
					}
				}
				if !proved {
					return state.VerificationUncertain{Class: "NATIVE_CREATE_STATIC_BINDING_UNVERIFIED"}
				}
				if (!originalNativeOutcomeMissing && op.StatusCode != api.Success) ||
					(start && observed.NativeState == "Stopped") {
					// Keep the exact native UUID claim for failed-start residue.
					// It is NOT a successful Create and needs explicit cleanup.
					if err := s.Store.RegisterCreatedBoxWithBinding(ctx, r.ID, s.Native.Project, in.Name, observed.UUID, state.VerifiedCreateBinding(proof)); err != nil {
						return state.VerificationUncertain{Class: "NATIVE_CREATE_UUID_LEDGER_UNVERIFIED"}
					}
					if !originalNativeOutcomeMissing && op.StatusCode != api.Success {
						return state.VerifiedEffectFailure{Class: "NATIVE_CREATE_FAILED_OWNED_STOPPED"}
					}
					return state.VerifiedEffectFailure{Class: "NATIVE_CREATE_GUEST_NOT_RUNNING_OWNED"}
				}
				if err := s.Store.FinalizeVerifiedCreateWithBinding(ctx, r.ID, s.Native.Project, in.Name, observed.UUID, state.VerifiedCreateBinding(proof)); err != nil {
					return state.VerificationUncertain{Class: "NATIVE_CREATE_UUID_LEDGER_UNVERIFIED"}
				}
				return state.ErrVerifiedCreateCommitted
			}
			r, err = s.submitNative(r, effect, verify)
			return &mcp.CallToolResult{}, r, err
		})

	addStrictTool(server, &mcp.Tool{Name: "box_list",
		Description: "List only UUID-registered owned Boxes with independently observed exact native identity.",
		Annotations: readToolAnnotations()},
		func(ctx context.Context, req *mcp.CallToolRequest, in listBoxesInput) (*mcp.CallToolResult, listBoxesResult, error) {
			principal, err := s.requestPrincipal(req)
			if err != nil {
				return nil, listBoxesResult{}, err
			}
			records, next, err := s.Store.ListOwnedBoxes(ctx, principal.Layer, in.After, in.Limit)
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
			return privateReadReply(ctx, s, principal.TokenID, out)
		})

	addStrictTool(server, &mcp.Tool{Name: "box_get",
		Description: "Get exact locally owned Box UUID and independently observed native state; native Running is not guest readiness. attention lists bounded SQLite-visible blockers/dependencies, independent of live native state. Optional detail=true returns native InstanceFull including potentially sensitive configuration, State, Snapshots and Backups. A missing or foreign UUID returns BOX_NOT_FOUND_OR_NOT_OWNED without revealing another Project.",
		Annotations: readToolAnnotations()},
		func(ctx context.Context, req *mcp.CallToolRequest, in getBoxInput) (*mcp.CallToolResult, getBoxResult, error) {
			principal, err := s.requestPrincipal(req)
			if err != nil {
				return nil, getBoxResult{}, err
			}
			if in.Detail {
				return s.readDetailedBox(ctx, req, principal, in.BoxID)
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
			attention := s.readAttention(ctx, principal, box.UUID)
			return privateReadReply(ctx, s, principal.TokenID, getBoxResult{TrackingID: tracking, Box: status, Attention: attention})
		})

	for _, action := range []string{"box_start", "box_stop"} {
		action := action
		desc := "Start one exact owned Box with verified private network preconditions."
		if action == "box_stop" {
			desc = "Stop one exact owned running Box without host execution."
		}
		addStrictTool(server, &mcp.Tool{Name: action, Description: desc, Annotations: mutationAnnotations(false)},
			func(ctx context.Context, req *mcp.CallToolRequest, in operateBoxInput) (*mcp.CallToolResult, state.Receipt, error) {
				if _, err := s.requestPrincipal(req); err != nil {
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
					if e := s.verifyLifecyclePin(ctx, r.ID, box); e != nil {
						return "", state.PreDispatchError{Class: "LIFECYCLE_PIN_UNVERIFIED"}
					}
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
					return s.verifyLifecyclePin(ctx, r.ID, box)
				}
				r, err = s.submitNative(r, effect, verify)
				return &mcp.CallToolResult{}, r, err
			})
	}

	addStrictTool(server, &mcp.Tool{Name: "box_update",
		Description: "Patch exact owned Box by presence-aware typed dotted Incus options; explicit false differs from omission. CPU 1..64, memory_mib 128..262144 and disk_gib 2..2048; invalid bounds are rejected before admission unless an original key is replayed. Mapping, AppArmor, and interception changes require safe stopped-instance capability; never permit privileged or raw config fallbacks.",
		Annotations: mutationAnnotations(true)},
		func(ctx context.Context, req *mcp.CallToolRequest, in updateBoxInput) (*mcp.CallToolResult, state.Receipt, error) {
			if _, err := s.requestPrincipal(req); err != nil {
				return nil, state.Receipt{}, err
			}
			if in.Internet == nil && in.CPU == nil && in.MemoryMiB == nil &&
				in.DiskGiB == nil && in.Nesting == nil && in.InstanceOptions.IsZero() {
				return nil, state.Receipt{}, native.ErrNativeUnsupported
			}
			changes := native.InstanceChanges{CPU: in.CPU, MemoryMiB: in.MemoryMiB, DiskGiB: in.DiskGiB,
				Nesting: in.Nesting, Internet: in.Internet, Options: in.InstanceOptions}
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
			// Reject impossible native dimensions before admitting a durable
			// operation. Replay precedes this check so historical originals
			// (including failed ones) retain their exact idempotency receipt.
			if in.CPU != nil && (*in.CPU < 1 || *in.CPU > 64) ||
				in.MemoryMiB != nil && (*in.MemoryMiB < 128 || *in.MemoryMiB > 262144) ||
				in.DiskGiB != nil && (*in.DiskGiB < 2 || *in.DiskGiB > 2048) {
				return nil, state.Receipt{}, state.ErrInvalid
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
				if e := s.verifyLifecyclePin(ctx, r.ID, box); e != nil {
					return "", state.PreDispatchError{Class: "LIFECYCLE_PIN_UNVERIFIED"}
				}
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
				if !native.InstanceChangesMatch(changes, actual) {
					return native.ErrUnverified
				}
				return s.verifyLifecyclePin(ctx, r.ID, box)
			}
			r, err = s.submitNative(r, effect, verify)
			return &mcp.CallToolResult{}, r, err
		})

	addStrictTool(server, &mcp.Tool{Name: "box_delete",
		Description: "Delete an exact owned Box. With confirmed Forward/Peer dependencies, the original reserves the Project and serially cleans exact linked child Operations before stopping and deleting; Peer requires existing host-local PRS review/approval. Without dependencies the Box must already be STOPPED. Unknown or foreign dependencies refuse; never blindly retry UNKNOWN.",
		Annotations: mutationAnnotations(true)},
		func(ctx context.Context, req *mcp.CallToolRequest, in operateBoxInput) (*mcp.CallToolResult, state.Receipt, error) {
			if _, err := s.requestPrincipal(req); err != nil {
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
				id, e := s.executeDependentDelete(ctx, r.ID, box)
				nativeID = id
				return id, e
			}
			verify := func(ctx context.Context) error {
				if err := nativeDone(ctx, s.Native, nativeID); err != nil {
					return err
				}
				if err := s.Native.ConfirmDeleteAbsence(ctx, box.Name); err != nil {
					return err
				}
				if err := s.Store.FinalizeVerifiedDelete(ctx, r.ID, box); err != nil {
					return state.VerificationUncertain{Class: "NATIVE_DELETE_TOMBSTONE_UNVERIFIED"}
				}
				// The original Delete has already committed. The independent bounded
				// retirement loop discovers this tombstone; never hold the Delete
				// worker open for an optional root witness.
				return state.ErrVerifiedDeleteCommitted
			}
			r, err = s.submitNative(r, effect, verify)
			return &mcp.CallToolResult{}, r, err
		})

	addStrictTool(server, &mcp.Tool{Name: "box_exec",
		Description: "Execute a nonempty argv array asynchronously INSIDE an exact owned guest with typed guest UID/GID; read original stdout/stderr and observed exit_code via operation_output_get. Operation SUCCEEDED is not proof of guest exit 0.",
		Annotations: mutationAnnotations(true)},
		func(ctx context.Context, req *mcp.CallToolRequest, in execBoxInput) (*mcp.CallToolResult, state.Receipt, error) {
			if _, err := s.requestPrincipal(req); err != nil {
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
					class := "EXEC_NATIVE_READ_UNVERIFIED"
					if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
						class = "EXEC_NATIVE_OBSERVATION_DEADLINE"
					} else if api.StatusErrorCheck(err, http.StatusNotFound) {
						class = "EXEC_NATIVE_RECEIPT_MISSING"
					}
					return state.VerificationUncertain{Class: class}
				}
				// The original Incus terminal result and its recorded log
				// paths must belong to this exact admitted guest/Operation.
				if final == nil || final.ID != nativeID {
					return state.VerificationUncertain{Class: "EXEC_NATIVE_IDENTITY_UNVERIFIED"}
				}
				// Incus 7.6 creates recorded logs BEFORE it calls Guest
				// Exec. An exact failed log-open with no Guest metadata is
				// a proven terminal pre-start native failure, not UNKNOWN.
				// Unclassified native failures stay UNKNOWN/fail-closed.
				if native.ExecRecordingQuotaPreStartFailure(final, box.Name, nativeID) {
					return state.VerifiedEffectFailure{Class: "NATIVE_EXEC_SPOOL_QUOTA_EXHAUSTED_PRESTART"}
				}
				out, err := native.ExecEvidenceFromFinal(final)
				if err != nil {
					return state.VerificationUncertain{Class: "EXEC_TERMINAL_EVIDENCE_UNVERIFIED"}
				}
				if !native.ExecEvidenceMatchesOriginal(out, box.Name, nativeID) {
					return state.VerificationUncertain{Class: "EXEC_TERMINAL_EVIDENCE_UNVERIFIED"}
				}
				if out.Exit == nil {
					return state.VerificationUncertain{Class: "EXEC_TERMINAL_EVIDENCE_UNVERIFIED"}
				}
				// Capture bounded private output while the original Exec claim still
				// protects the Box from concurrent Delete. A spool failure must
				// not erase an independently verified native terminal result.
				s.captureExecArchive(ctx, r.ID, out.StdoutPath, out.StderrPath)
				// The final native Exec receipt, Guest exit, paths and
				// BoxWeave SUCCEEDED event are ONE SQLite commit. Never
				// leave a separate output-row / final-state crash gap.
				if err := s.Store.FinalizeVerifiedExec(ctx, r.ID, box,
					nativeID, out.StdoutPath, out.StderrPath, *out.Exit); err != nil {
					return state.VerificationUncertain{Class: "EXEC_SQLITE_FINALIZATION_UNVERIFIED"}
				}
				return state.ErrVerifiedExecCommitted
			}
			r, err = s.submitNative(r, effect, verify)
			return &mcp.CallToolResult{}, r, err
		})
}

// Ensure the wire handler imports only ordinary typed SDK/native data and
// does not create an extra transport, host-shell endpoint or task service.
var _ = http.StatusNotFound
