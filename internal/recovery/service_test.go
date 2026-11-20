package recovery

import (
	"context"
	"errors"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
	"testing"
)

const nu = "12345678-1234-4abc-8def-123456789abc"
const bu = "87654321-4321-4abc-8def-abcdef012345"

type witness struct {
	Observer
	box             native.BoxState
	op              *api.Operation
	opErr, errorBox error
	markerErr       error
	page            native.GuestFilePage
	deleteObs       native.DeleteRecoveryObservation
	privateErr      error
}

func (w witness) GetExactBox(context.Context, string, string) (native.BoxState, string, error) {
	return w.box, "etag", w.errorBox
}
func (w witness) CheckExactPrivateNIC(context.Context, string, string) error { return w.privateErr }
func (w witness) NativeOperation(context.Context, string) (*api.Operation, error) {
	return w.op, w.opErr
}
func (w witness) ObserveCreatedByOperation(context.Context, string, string) (native.BoxState, error) {
	return w.box, w.markerErr
}
func (w witness) ObserveOwnedBox(context.Context, string, string) (native.BoxObservation, error) {
	return native.BoxObservation{State: "PRESENT_UUID_CONFIRMED", ExpectedUUID: w.box.UUID, Name: w.box.Name, Project: w.box.Project, Fingerprint: "marker"}, w.errorBox
}
func (w witness) ObserveUnknownDelete(context.Context, string, string, string) (native.DeleteRecoveryObservation, error) {
	return w.deleteObs, w.opErr
}
func (w witness) ReadGuestFile(context.Context, string, string, string, int64, int) (native.GuestFilePage, error) {
	return w.page, nil
}
func original(action string) state.RecoveryTarget {
	return state.RecoveryTarget{ID: "0123456789abcdef0123456789abcdef", Action: action, State: "UNKNOWN", Layer: "layer", NativeID: nu, UUID: bu, Incarnation: bu, Resource: "box:owned", Name: "owned", OwnedState: "OWNED"}
}
func known() witness {
	return witness{box: native.BoxState{UUID: bu, Name: "owned", Project: "layer", NativeState: "Running", CPU: "2", Memory: "512MiB"}, op: &api.Operation{ID: nu, StatusCode: api.Success}}
}
func TestS5A2DomainProofMatrix(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name         string
		change       func(*state.RecoveryTarget, *witness)
		kind, reason string
	}{
		{"start", func(t *state.RecoveryTarget, w *witness) {}, "SETTLE", "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED"},
		{"power mismatch", func(t *state.RecoveryTarget, w *witness) { w.box.NativeState = "Stopped" }, "APPROVAL_REQUIRED", "POWER_POSTSTATE_UNVERIFIED"},
		{"stop", func(t *state.RecoveryTarget, w *witness) { t.Action = "box_stop"; w.box.NativeState = "Stopped" }, "SETTLE", "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED"},
		{"changed uuid", func(t *state.RecoveryTarget, w *witness) { w.box.UUID = "other" }, "BLOCKED", "EXACT_NATIVE_UUID_UNVERIFIED"},
		{"foreign layer", func(t *state.RecoveryTarget, w *witness) { w.box.Project = "foreign" }, "BLOCKED", "EXACT_NATIVE_UUID_UNVERIFIED"},
		{"missing ack", func(t *state.RecoveryTarget, w *witness) { t.NativeID = "" }, "APPROVAL_REQUIRED", "ORIGINAL_NATIVE_ACK_MISSING"},
		{"unavailable evidence", func(t *state.RecoveryTarget, w *witness) { w.opErr = errors.New("unavailable") }, "BLOCKED", "ORIGINAL_NATIVE_RECEIPT_UNVERIFIED"},
		{"native active", func(t *state.RecoveryTarget, w *witness) { w.op.StatusCode = api.Running }, "BLOCKED", "ORIGINAL_NATIVE_ACTIVE"},
		{"native failure", func(t *state.RecoveryTarget, w *witness) { w.op.StatusCode = api.Failure }, "APPROVAL_REQUIRED", "NATIVE_TERMINAL_SUCCESS_UNPROVED"},
		{"wrong native id", func(t *state.RecoveryTarget, w *witness) { w.op.ID = "different" }, "BLOCKED", "ORIGINAL_NATIVE_RECEIPT_UNVERIFIED"},
		{"private NIC drift", func(t *state.RecoveryTarget, w *witness) { w.privateErr = errors.New("drift") }, "BLOCKED", "PRIVATE_NIC_UNVERIFIED"},
		{"update", func(t *state.RecoveryTarget, w *witness) {
			t.Action = "box_update"
			t.SafeIntent = "{\"box_id\":\"" + bu + "\",\"changes\":{\"CPU\":2}}"
		}, "SETTLE", "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED"},
		{"update mismatch", func(t *state.RecoveryTarget, w *witness) {
			t.Action = "box_update"
			t.SafeIntent = "{\"box_id\":\"" + bu + "\",\"changes\":{\"CPU\":3}}"
		}, "APPROVAL_REQUIRED", "UPDATE_POSTSTATE_UNVERIFIED"},
		{"create marker", func(t *state.RecoveryTarget, w *witness) {
			t.Action = "box_create"
			t.UUID = ""
			t.Incarnation = ""
			t.OwnedState = ""
			t.NativeID = ""
		}, "BIND_IDENTITY", "CREATE_MARKER_EXACT_IDENTITY"},
		{"create no marker", func(t *state.RecoveryTarget, w *witness) {
			t.Action = "box_create"
			t.UUID = ""
			t.Incarnation = ""
			w.markerErr = errors.New("wrong marker")
		}, "BLOCKED", "CREATE_MARKER_IDENTITY_UNPROVED"},
		{"bound create", func(t *state.RecoveryTarget, w *witness) { t.Action = "box_create" }, "APPROVAL_REQUIRED", "CREATE_IDENTITY_BOUND_OUTCOME_UNPROVED"},
		{"delete", func(t *state.RecoveryTarget, w *witness) {
			t.Action = "box_delete"
			w.deleteObs.Box = native.BoxObservation{Project: t.Layer, Name: t.Name, ExpectedUUID: t.UUID, State: "ABSENT_UNCERTAIN"}
			w.deleteObs.NativeID = t.NativeID
			w.deleteObs.Class = "ORIGINAL_TERMINAL_DELETE_VERIFIED"
			w.deleteObs.Fingerprint = "DELETE_SUCCESS:test"
		}, "SETTLE", "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED"},
		{"delete absent alone", func(t *state.RecoveryTarget, w *witness) {
			t.Action = "box_delete"
			w.deleteObs.Box = native.BoxObservation{Project: t.Layer, Name: t.Name, ExpectedUUID: t.UUID, State: "ABSENT_UNCERTAIN"}
			w.deleteObs.NativeID = t.NativeID
			w.deleteObs.Class = "NATIVE_RECEIPT_UNAVAILABLE"
		}, "APPROVAL_REQUIRED", "DELETE_TERMINAL_ABSENCE_NOT_PROVED"},
		{"dependent delete", func(t *state.RecoveryTarget, w *witness) { t.Action = "box_delete"; t.DependentDelete = true }, "BLOCKED", "NETWORK_RECOVERY_DEFERRED_A4"},
		{"peer", func(t *state.RecoveryTarget, w *witness) { t.Action = "box_peer_set" }, "BLOCKED", "NETWORK_RECOVERY_DEFERRED_A4"},
		{"forward", func(t *state.RecoveryTarget, w *witness) { t.Action = "port_forward_create" }, "BLOCKED", "NETWORK_RECOVERY_DEFERRED_A4"},
		{"readiness", func(t *state.RecoveryTarget, w *witness) { t.Action = state.ReadinessAction }, "APPROVAL_REQUIRED", "READINESS_HISTORICAL_OUTCOME_UNPROVED"},
		{"matching file", func(t *state.RecoveryTarget, w *witness) {
			t.Action = "box_file_apply"
			t.SafeIntent = "{\"box_id\":\"" + bu + "\",\"path\":\"/tmp/marker\",\"mode\":\"create\",\"bytes\":3,\"sha256\":\"abc\"}"
			w.page = native.GuestFilePage{WholeFile: true, Size: 3, PageSHA256: "abc"}
		}, "APPROVAL_REQUIRED", "FILE_CONTENT_MATCHES_CAUSALITY_UNPROVED"},
		{"Exec verified nonzero", func(t *state.RecoveryTarget, w *witness) {
			t.Action = "box_exec"
			w.op.StatusCode = api.Failure
			w.op.Err = "Command not found"
			w.op.Metadata = map[string]any{"return": float64(127), "output": map[string]any{"1": "/1.0/instances/owned/logs/exec-output/exec_" + nu + ".stdout", "2": "/1.0/instances/owned/logs/exec-output/exec_" + nu + ".stderr"}}
		}, "SETTLE", "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED"},
		{"Exec missing metadata", func(t *state.RecoveryTarget, w *witness) { t.Action = "box_exec" }, "APPROVAL_REQUIRED", "EXEC_TERMINAL_EVIDENCE_UNPROVED"},
		{"released", func(t *state.RecoveryTarget, w *witness) { t.Released = true }, "NOT_REQUIRED", "ORIGINAL_ALREADY_REVIEWED_RELEASED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := original("box_start")
			w := known()
			tc.change(&target, &w)
			v, p := (Service{Native: w, Layer: "layer"}).observe(ctx, target, "")
			if p.Kind != tc.kind || p.Reason != tc.reason {
				t.Fatalf("%+v %+v", v, p)
			}
			if target.State == "UNKNOWN" && !target.Released && v.ClaimProtection != "RETAINED" {
				t.Fatalf("inspection unlocked %+v", v)
			}
			if p.Kind == state.RecoverySettle && (p.NativeID != nu || p.UUID != bu || p.Fingerprint == "") {
				t.Fatalf("missing exact proof %+v", p)
			}
		})
	}
}
