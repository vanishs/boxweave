package control

import (
	"context"
	"net/http"
	"time"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// admitLegacyCreate is the pre-consolidation static-binding path. Keep it
// isolated until the remaining native witness gates permit retirement.
func (s *Service) admitLegacyCreate(ctx context.Context, req *mcp.CallToolRequest, in createBoxInput, start, internet bool) (*mcp.CallToolResult, state.Receipt, error) {
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
	// Preserve the admitted caller intent for legacy replay; materialize
	// the default only for an actually new native Create.
	in.InstanceOptions = in.InstanceOptions.ForCreate()
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
}

// finalizeLegacyDelete is the old tombstone and asynchronous retirement path.
func (s *Service) finalizeLegacyDelete(ctx context.Context, originalID string, box state.OwnedBox) error {
	if err := s.Store.FinalizeVerifiedDelete(ctx, originalID, box); err != nil {
		return state.VerificationUncertain{Class: "NATIVE_DELETE_TOMBSTONE_UNVERIFIED"}
	}
	// The original Delete has already committed. The independent bounded
	// retirement loop discovers this tombstone; never hold the Delete
	// worker open for an optional root witness.
	return state.ErrVerifiedDeleteCommitted
}
