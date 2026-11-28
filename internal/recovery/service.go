package recovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// Read-only native seam, shared with the root-reviewed CLI. No mutation
// method is reachable from recovery; test doubles cannot issue native work.
type Observer interface {
	ObserveOwnedBox(context.Context, string, string) (native.BoxObservation, error)
	ObserveCreatedByOperation(context.Context, string, string) (native.BoxState, error)
	ObserveUnknownDelete(context.Context, string, string, string) (native.DeleteRecoveryObservation, error)
	GetExactBox(context.Context, string, string) (native.BoxState, string, error)
	CheckExactPrivateNIC(context.Context, string, string) error
	NativeOperation(context.Context, string) (*api.Operation, error)
	ReadRecordedOutput(context.Context, string, int64, int64) ([]byte, bool, error)
	ReadGuestFile(context.Context, string, string, string, int64, int) (native.GuestFilePage, error)
}

// ObservationStore exposes only the existing read predicates. Diagnostic
// snapshots implement this interface without exporting their private Store.
type ObservationStore interface {
	ReadRecoveryTarget(context.Context, string, string) (state.RecoveryTarget, error)
	ExecGCReviewInspect(context.Context, string, string) (state.ExecGCReview, error)
	CheckDependentDeleteRecovery(context.Context, state.RecoveryTarget) error
	CheckRecoveryCreateBinding(context.Context, state.RecoveryTarget, state.RecoveryProof) error
	CreatePinRepairStatus(context.Context, state.RecoveryTarget) (string, error)
	ReadOriginalCreatePin(context.Context, state.RecoveryTarget) (state.VerifiedCreateBinding, error)
	CheckCreatePinRepair(context.Context, state.RecoveryTarget, state.RecoveryProof) error
}
type Service struct {
	ReadStore ObservationStore

	Store                  *state.Store
	Native                 Observer
	Layer                  string
	ScopedPeerWitness      func(context.Context, string) (state.ScopedPeerRootReceipt, error)
	ScopedLifecycleWitness func(context.Context, string) (state.ScopedLifecycleRecoveryProof, error)
}

func (s Service) reader() ObservationStore {
	if s.ReadStore != nil {
		return s.ReadStore
	}
	if s.Store != nil {
		return s.Store
	}
	return nil
}

type View struct {
	OriginalID         string `json:"original_operation_id"`
	OriginalAction     string `json:"original_action"`
	OriginalState      string `json:"original_state"`
	BoxID              string `json:"box_id,omitempty"`
	Incarnation        string `json:"incarnation,omitempty"`
	OriginalNativeID   string `json:"original_native_operation_id,omitempty"`
	ClaimProtection    string `json:"claim_protection"`
	Disposition        string `json:"disposition"`
	Reason             string `json:"reason"`
	ProposedAction     string `json:"proposed_action,omitempty"`
	NativeObservation  string `json:"native_observation"`
	EvidenceSource     string `json:"evidence_source"`
	ObservedNativeUUID string `json:"observed_native_uuid,omitempty"`
	GuestExitCode      *int   `json:"guest_exit_code,omitempty"`
	GCStream           string `json:"gc_stream,omitempty"`
}
type Result struct {
	View          View                  `json:"diagnosis"`
	Recovery      *state.RecoveryRecord `json:"recovery,omitempty"`
	NextAction    *state.NextAction     `json:"next_action,omitempty"`
	NativeRetried bool                  `json:"native_retried"`
}

// Network and dependent Delete barriers cannot be retired by a generic
// legacy RELEASED audit row. Preserve the row as history but report the
// retained protection and require explicit review; never infer settlement.
func protectedNetworkRecovery(t state.RecoveryTarget) bool {
	return t.ScopedLifecycle || t.DependentDelete || (t.Action == "box_create" && (t.CreatePinStaged || t.CreatePinRequired)) || t.Action == "box_peer_set" || t.Action == "port_forward_create" || t.Action == "port_forward_delete"
}
func base(t state.RecoveryTarget, stream string) View {
	v := View{OriginalID: t.ID, OriginalAction: t.Action, OriginalState: t.State, BoxID: t.UUID, Incarnation: t.Incarnation, OriginalNativeID: t.NativeID, ClaimProtection: "NOT_UNKNOWN", NativeObservation: "NOT_CHECKED", EvidenceSource: "SQLITE_DURABLE", GCStream: stream}
	if t.State == "UNKNOWN" {
		v.ClaimProtection = "RETAINED"
		if t.Released && !protectedNetworkRecovery(t) {
			v.ClaimProtection = "REVIEWED_RELEASED"
		}
	}
	return v
}
func validStream(s string) bool { return s == "" || s == "stdout" || s == "stderr" }

func ObserveCreate(ctx context.Context, n Observer, name, id string) (native.BoxState, native.BoxObservation, error) {
	box, e := n.ObserveCreatedByOperation(ctx, name, id)
	if e != nil {
		return box, native.BoxObservation{}, e
	}
	obs, e := n.ObserveOwnedBox(ctx, name, box.UUID)
	if e != nil {
		return box, obs, e
	}
	if box.UUID == "" || box.Name != name || obs.Name != name || obs.Project != box.Project || obs.State != "PRESENT_UUID_CONFIRMED" || obs.ExpectedUUID != box.UUID {
		return box, obs, native.ErrUnverified
	}
	return box, obs, nil
}
func ObserveGC(ctx context.Context, n Observer, r state.ExecGCReview) (string, error) {
	box, _, e := n.GetExactBox(ctx, r.Name, r.BoxUUID)
	if e != nil {
		return "", e
	}
	if box.UUID != r.BoxUUID || box.Project != r.Project {
		return "", native.ErrNotOwned
	}
	_, _, e = n.ReadRecordedOutput(ctx, r.Path, 0, 1)
	if errors.Is(e, native.ErrRecordedOutputUnavailable) {
		return "NATIVE_MISSING", nil
	}
	if e != nil {
		return "", e
	}
	return "NATIVE_PRESENT", nil
}

func (s Service) Inspect(ctx context.Context, id, stream string) (Result, error) {
	if !validStream(stream) || s.reader() == nil {
		return Result{}, state.ErrInvalid
	}
	t, e := s.reader().ReadRecoveryTarget(ctx, s.Layer, id)
	if e != nil {
		return Result{}, e
	}
	v, _ := s.observe(ctx, t, stream)
	return Result{View: v}, nil
}
func (s Service) Reconcile(ctx context.Context, r state.RecoveryRequest) (Result, error) {
	if s.ReadStore != nil || s.Store == nil || r.Review != nil {
		return Result{}, state.ErrInvalid
	}
	if r.Layer != s.Layer {
		return Result{}, state.ErrInvalid
	}
	old, e := s.Store.ReplayRecovery(ctx, r)
	if e == nil {
		t, e := s.Store.ReadRecoveryTarget(ctx, s.Layer, r.OriginalID)
		if e != nil {
			return Result{}, e
		}
		v := base(t, r.Stream)
		v.Disposition, v.Reason = old.Disposition, old.Reason
		v.NativeObservation = "NOT_REOBSERVED_REPLAY"
		if r.Stream != "" {
			v.ClaimProtection = "GC_NOT_REOBSERVED"
		}
		return Result{View: v, Recovery: &old}, nil
	}
	if !errors.Is(e, state.ErrNotFound) {
		return Result{}, e
	}
	t, e := s.Store.ReadRecoveryTarget(ctx, s.Layer, r.OriginalID)
	if e != nil {
		return Result{}, e
	}
	v, p := s.observe(ctx, t, r.Stream)
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	record, e := s.commitRecovery(ctx, r, t, p)
	if e != nil {
		return Result{}, e
	}
	current, e := s.Store.ReadRecoveryTarget(ctx, s.Layer, t.ID)
	if e != nil {
		return Result{}, e
	}
	latest := base(current, r.Stream)
	if record.Receipt.Duplicate {
		latest.NativeObservation = "NOT_REOBSERVED_REPLAY"
		if r.Stream != "" {
			latest.ClaimProtection = "GC_NOT_REOBSERVED"
		}
	} else {
		latest.NativeObservation = v.NativeObservation
		latest.EvidenceSource = v.EvidenceSource
		latest.ObservedNativeUUID = v.ObservedNativeUUID
		latest.GuestExitCode = v.GuestExitCode
		if r.Stream != "" {
			latest.ClaimProtection = v.ClaimProtection
		}
	}
	latest.Disposition, latest.Reason = record.Disposition, record.Reason
	return Result{View: latest, Recovery: &record}, nil
}
func (s Service) commitRecovery(ctx context.Context, r state.RecoveryRequest, t state.RecoveryTarget, p state.RecoveryProof) (state.RecoveryRecord, error) {
	return s.Store.CommitScopedRecovery(ctx, r, t, p)
}

func finishRecoveryObservation(v View, kind, reason string) (View, state.RecoveryProof) {
	v.Disposition, v.Reason = kind, reason
	if kind == state.RecoverySettle || kind == state.RecoveryBind {
		v.Disposition = "SAFE_RECOVER"
		v.ProposedAction = kind
	}
	return v, state.RecoveryProof{Kind: kind, Reason: reason}
}

func (s Service) observe(ctx context.Context, t state.RecoveryTarget, stream string) (View, state.RecoveryProof) {
	v := base(t, stream)
	finish := func(kind, reason string) (View, state.RecoveryProof) {
		return finishRecoveryObservation(v, kind, reason)
	}
	if stream != "" {
		if t.Action != "box_exec" {
			return finish(state.RecoveryBlocked, "GC_REQUIRES_ORIGINAL_EXEC")
		}
		gc, e := s.reader().ExecGCReviewInspect(ctx, t.ID, stream)
		if e != nil || gc.Project != s.Layer {
			return finish(state.RecoveryBlocked, "GC_REVIEW_NOT_AVAILABLE")
		}
		v.ClaimProtection = "GC_REVIEW_RETAINED"
		observed, e := ObserveGC(ctx, s.Native, gc)
		if e != nil {
			return finish(state.RecoveryBlocked, "GC_NATIVE_OBSERVATION_UNVERIFIED")
		}
		v.NativeObservation = observed
		v.EvidenceSource = "SQLITE_AND_NATIVE"
		return finish(state.RecoveryApproval, "GC_RELEASE_REQUIRES_APPROVAL_NO_EXPIRY_INFERENCE")
	}
	if t.Action == "operation_recover" {
		return finish(state.RecoveryBlocked, "RECOVERY_OF_RECOVERY_UNSUPPORTED")
	}
	// A fresh-only controller cannot adopt a Legacy original, even if
	// an exact native name or successful old receipt remains observable.
	if (t.Action == "box_create" || t.Action == "box_delete") && !t.ScopedLifecycle ||
		t.Action == "box_peer_set" && !t.ScopedPeer {
		return finish(state.RecoveryBlocked, "SCOPED_ORIGINAL_REQUIRED")
	}
	if t.Action == "box_create" && t.State == "SUCCEEDED" && t.CreatePinRequired && t.CreatePinStaged && t.OwnedState == "OWNED" {
		return s.observeCreatePinRepair(ctx, t, v)
	}
	if t.State != "UNKNOWN" {
		switch t.State {
		case "SUCCEEDED", "FAILED", "CANCELLED", "TIMED_OUT":
			return finish(state.RecoveryNotNeeded, "ORIGINAL_TERMINAL")
		default:
			return finish(state.RecoveryBlocked, "ORIGINAL_NOT_UNKNOWN")
		}
	}
	if t.Released {
		if protectedNetworkRecovery(t) {
			return finish(state.RecoveryBlocked, "NETWORK_LEGACY_RELEASE_REVIEW_REQUIRED")
		}
		return finish(state.RecoveryNotNeeded, "ORIGINAL_ALREADY_REVIEWED_RELEASED")
	}
	if t.ScopedLifecycle {
		return s.observeScopedLifecycle(ctx, t, v)
	}
	if t.Action == "box_peer_set" {
		return s.observeScopedPeer(ctx, t, v)
	}
	if t.DependentDelete && (s.reader() == nil || s.reader().CheckDependentDeleteRecovery(ctx, t) != nil) {
		return finish(state.RecoveryBlocked, "DEPENDENT_DELETE_FINAL_PROOF_REQUIRED")
	}
	if t.Action == "port_forward_create" || t.Action == "port_forward_delete" {
		if !t.ForwardReady {
			return finish(state.RecoveryBlocked, "FORWARD_ORIGINAL_CLAIM_UNVERIFIED")
		}
		observer, ok := s.Native.(interface {
			ObserveOwnedForward(context.Context, string, string, native.ForwardSpec, bool) error
			AuthorizeForwardCreate(native.ForwardSpec) error
		})
		if !ok {
			return finish(state.RecoveryBlocked, "FORWARD_NATIVE_OBSERVER_UNAVAILABLE")
		}
		if t.OwnedState != "OWNED" || t.UUID == "" || t.Incarnation == "" || t.Name == "" || t.Resource != "box:"+t.Name || t.NativeID == "" {
			return finish(state.RecoveryBlocked, "FORWARD_ORIGINAL_IDENTITY_UNVERIFIED")
		}
		op, e := s.Native.NativeOperation(ctx, t.NativeID)
		if e != nil || op == nil || op.ID != t.NativeID || op.StatusCode != api.Success {
			return finish(state.RecoveryBlocked, "FORWARD_ORIGINAL_TERMINAL_UNVERIFIED")
		}
		f := t.Forward
		spec := native.ForwardSpec{Device: f.NativeDevice, Protocol: f.Protocol, HostBind: f.HostBind, HostPort: f.HostPort, GuestPort: f.GuestPort, ExternalAcknowledged: true}
		create := t.Action == "port_forward_create"
		if create && observer.AuthorizeForwardCreate(spec) != nil {
			return finish(state.RecoveryBlocked, "FORWARD_CURRENT_EXPOSURE_UNVERIFIED")
		}
		if e = observer.ObserveOwnedForward(ctx, t.Name, t.UUID, spec, create); e != nil {
			return finish(state.RecoveryBlocked, "FORWARD_DEVICE_POSTSTATE_UNVERIFIED")
		}
		v.NativeObservation = "ORIGINAL_TERMINAL_DEVICE_VERIFIED"
		v.EvidenceSource = "SQLITE_AND_NATIVE"
		view, p := finish(state.RecoverySettle, "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED")
		p.UUID, p.NativeID, p.Forward = t.UUID, t.NativeID, f
		p.Fingerprint = fingerprint(t.ID, t.NativeID, t.UUID, f.NativeDevice, f.Protocol, f.HostBind, fmt.Sprint(f.HostPort), fmt.Sprint(f.GuestPort), fmt.Sprint(create))
		return view, p
	}
	if t.Action == "operation_cancel" {
		var in struct {
			Target string `json:"target_operation_id"`
		}
		if json.Unmarshal([]byte(t.SafeIntent), &in) == nil {
			target, e := s.reader().ReadRecoveryTarget(ctx, s.Layer, in.Target)
			if e == nil && target.State == "CANCELLED" && target.NativeID == "" {
				return finish(state.RecoveryBlocked, "TARGET_CANCELLED_ORIGINAL_CAUSALITY_UNPROVED")
			}
		}
		return finish(state.RecoveryBlocked, "CANCEL_OUTCOME_UNPROVED")
	}
	if t.Name == "" || t.Resource != "box:"+t.Name {
		return finish(state.RecoveryBlocked, "EXACT_ORIGINAL_OWNERSHIP_UNPROVED")
	}

	if t.OwnedState != "OWNED" || t.UUID == "" || t.Incarnation == "" {
		return finish(state.RecoveryBlocked, "EXACT_ORIGINAL_OWNERSHIP_UNPROVED")
	}

	box, etag, e := s.Native.GetExactBox(ctx, t.Name, t.UUID)
	if e != nil || box.UUID != t.UUID || box.Project != s.Layer {
		return finish(state.RecoveryBlocked, "EXACT_NATIVE_UUID_UNVERIFIED")
	}
	v.NativeObservation = "PRESENT_UUID_CONFIRMED"
	v.EvidenceSource = "SQLITE_AND_NATIVE"
	switch t.Action {
	case "box_create":
		return finish(state.RecoveryApproval, "CREATE_IDENTITY_BOUND_OUTCOME_UNPROVED")
	case state.ReadinessAction:
		return finish(state.RecoveryApproval, "READINESS_HISTORICAL_OUTCOME_UNPROVED")
	case "box_file_apply":
		// Persisted keys use snake case; explicit tags avoid accidental matches.
		var saved struct {
			BoxID  string `json:"box_id"`
			Path   string `json:"path"`
			Mode   string `json:"mode"`
			Bytes  int    `json:"bytes"`
			SHA256 string `json:"sha256"`
		}
		if json.Unmarshal([]byte(t.SafeIntent), &saved) != nil || saved.BoxID != t.UUID || saved.Mode != "create" || !native.SafeGuestFilePath(saved.Path) || saved.Bytes < 0 || saved.Bytes > native.MaxGuestFileBytes {
			return finish(state.RecoveryBlocked, "FILE_ORIGINAL_INTENT_UNVERIFIED")
		}
		page, e := s.Native.ReadGuestFile(ctx, t.Name, t.UUID, saved.Path, 0, native.MaxGuestFileBytes)
		if e == nil && page.WholeFile && page.Size == int64(saved.Bytes) && page.PageSHA256 == saved.SHA256 {
			return finish(state.RecoveryApproval, "FILE_CONTENT_MATCHES_CAUSALITY_UNPROVED")
		}
		return finish(state.RecoveryApproval, "FILE_ORIGINAL_EFFECT_UNPROVED")
	case "box_exec", "box_start", "box_stop", "box_update":
	default:
		return finish(state.RecoveryBlocked, "DOMAIN_RECOVERY_UNSUPPORTED")
	}
	if t.NativeID == "" {
		return finish(state.RecoveryApproval, "ORIGINAL_NATIVE_ACK_MISSING")
	}
	op, e := s.Native.NativeOperation(ctx, t.NativeID)
	if api.StatusErrorCheck(e, http.StatusNotFound) {
		return finish(state.RecoveryApproval, "ORIGINAL_NATIVE_RECEIPT_UNAVAILABLE")
	}
	if e != nil || op == nil || op.ID != t.NativeID {
		return finish(state.RecoveryBlocked, "ORIGINAL_NATIVE_RECEIPT_UNVERIFIED")
	}
	if !op.StatusCode.IsFinal() {
		v.NativeObservation = "NATIVE_ACTIVE"
		return finish(state.RecoveryBlocked, "ORIGINAL_NATIVE_ACTIVE")
	}
	v.NativeObservation = "NATIVE_TERMINAL"
	if t.Action == "box_exec" {
		ev, e := native.ExecEvidenceFromFinal(op)
		if e != nil || !native.ExecEvidenceMatchesOriginal(ev, t.Name, t.NativeID) {
			return finish(state.RecoveryApproval, "EXEC_TERMINAL_EVIDENCE_UNPROVED")
		}
		// Repeat exact native identity after observing terminal. Recorded paths
		// and exit are not proof that output bytes remain accessible.
		again, _, e := s.Native.GetExactBox(ctx, t.Name, t.UUID)
		if e != nil || again.UUID != t.UUID || again.Project != s.Layer {
			return finish(state.RecoveryBlocked, "EXACT_NATIVE_UUID_UNVERIFIED")
		}
		v.GuestExitCode = ev.Exit
		view, p := finish(state.RecoverySettle, "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED")
		p.NativeID, p.UUID, p.Exit, p.StdoutPath, p.StderrPath = t.NativeID, t.UUID, ev.Exit, ev.StdoutPath, ev.StderrPath
		p.Fingerprint = fingerprint(t.ID, t.NativeID, t.UUID, fmt.Sprint(*ev.Exit), ev.StdoutPath, ev.StderrPath)
		return view, p
	}
	if op.StatusCode != api.Success {
		return finish(state.RecoveryApproval, "NATIVE_TERMINAL_SUCCESS_UNPROVED")
	}
	// Re-read after the native terminal; the original UNKNOWN protection
	// prevents other BoxWeave mutations until the single SQLite commit.
	box, etag, e = s.Native.GetExactBox(ctx, t.Name, t.UUID)
	if e != nil || box.UUID != t.UUID || box.Project != s.Layer {
		return finish(state.RecoveryBlocked, "EXACT_NATIVE_UUID_UNVERIFIED")
	}
	switch t.LifecyclePinStatus {
	case "MISSING":
		return finish(state.RecoveryBlocked, "LIFECYCLE_STATIC_LEDGER_MISSING")
	case "CONFLICT":
		return finish(state.RecoveryBlocked, "LIFECYCLE_STATIC_LEDGER_MISMATCH")
	case "PINNED":
		n, ok := s.Native.(interface {
			VerifyPinnedDeleteNIC(context.Context, string, string, string, string, string, string, string) error
		})
		if !ok {
			return finish(state.RecoveryBlocked, "LIFECYCLE_STATIC_OBSERVER_UNAVAILABLE")
		}
		pin := t.LifecycleBinding
		if e = n.VerifyPinnedDeleteNIC(ctx, t.Name, t.UUID, pin.Bridge, pin.ACL, pin.MAC, pin.IPv4, pin.IPv6); e != nil {
			if errors.Is(e, native.ErrUnsafeNetwork) {
				return finish(state.RecoveryBlocked, "NATIVE_PINNED_NIC_DRIFT")
			}
			return finish(state.RecoveryBlocked, "NATIVE_PINNED_NIC_UNVERIFIED")
		}
		again, tag, err := s.Native.GetExactBox(ctx, t.Name, t.UUID)
		if err != nil || tag == "" || etag != tag || again.UUID != box.UUID || again.Project != box.Project || again.NativeState != box.NativeState {
			return finish(state.RecoveryBlocked, "LIFECYCLE_NATIVE_POSTSTATE_CHANGED")
		}
	}
	if e = s.Native.CheckExactPrivateNIC(ctx, t.Name, t.UUID); e != nil {
		return finish(state.RecoveryBlocked, "PRIVATE_NIC_UNVERIFIED")
	}
	switch t.Action {
	case "box_start":
		if box.NativeState != "Running" {
			return finish(state.RecoveryApproval, "POWER_POSTSTATE_UNVERIFIED")
		}
	case "box_stop":
		if box.NativeState != "Stopped" {
			return finish(state.RecoveryApproval, "POWER_POSTSTATE_UNVERIFIED")
		}
	case "box_update":
		var in struct {
			BoxID   string                 `json:"box_id"`
			Changes native.InstanceChanges `json:"changes"`
		}
		if json.Unmarshal([]byte(t.SafeIntent), &in) != nil || in.BoxID != t.UUID || !native.InstanceChangesMatch(in.Changes, box) {
			return finish(state.RecoveryApproval, "UPDATE_POSTSTATE_UNVERIFIED")
		}
	}
	view, p := finish(state.RecoverySettle, "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED")
	p.NativeID, p.UUID, p.Fingerprint = t.NativeID, t.UUID, fingerprint(t.ID, t.NativeID, t.UUID, etag, box.NativeState)
	if t.LifecyclePinStatus == "PINNED" {
		p.LifecycleBinding = t.LifecycleBinding
		pin := p.LifecycleBinding
		p.Fingerprint = fingerprint(p.Fingerprint, pin.Bridge, pin.ACL, pin.MAC, pin.IPv4, pin.IPv6)
	}
	return view, p
}

func fingerprint(parts ...string) string {
	raw, _ := json.Marshal(parts)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
