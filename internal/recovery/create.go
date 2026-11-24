package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type createRecoveryObserver interface {
	ObserveOriginalCreateStaticPeerBinding(context.Context, string, string, string) (native.VerifiedStaticPeerBinding, error)
	OriginalOperationInactive(context.Context, string) error
}

// observeStagedCreate requires the exact original marker, durable address
// reservation, matching normalized intent, and original native terminal or
// independently inactive receipt. It never calls native Create.
func (s Service) observeStagedCreate(ctx context.Context, t state.RecoveryTarget, v View) (View, state.RecoveryProof) {
	block := func(reason string) (View, state.RecoveryProof) {
		v.Disposition, v.Reason = state.RecoveryBlocked, reason
		return v, state.RecoveryProof{Kind: state.RecoveryBlocked, Reason: reason}
	}
	n, ok := s.Native.(createRecoveryObserver)
	if !ok || s.reader() == nil {
		return block("CREATE_STATIC_RECOVERY_OBSERVER_UNAVAILABLE")
	}
	var in struct {
		Name      string                  `json:"name"`
		CPU       int                     `json:"cpu"`
		MemoryMiB int64                   `json:"memory_mib"`
		DiskGiB   int64                   `json:"disk_gib"`
		Start     *bool                   `json:"start"`
		Nesting   *bool                   `json:"nesting"`
		Internet  *bool                   `json:"internet"`
		Options   *native.InstanceOptions `json:"instance_options"`
	}
	if json.Unmarshal([]byte(t.SafeIntent), &in) != nil || in.Name != t.Name || in.CPU < 1 || in.MemoryMiB < 1 || in.DiskGiB < 1 || in.Start == nil || in.Nesting == nil || in.Internet == nil || in.Options == nil {
		return block("CREATE_ORIGINAL_INTENT_UNVERIFIED")
	}
	// An absent ACK is not a synthetic native receipt. The exact original
	// marker and static poststate must be bracketed by Project-wide native
	// quiescence; no assertion about a vanished operation's status is made.
	noACK := t.NativeID == ""
	idle, hasIdle := s.Native.(interface {
		ObserveCreateRecoveryIdle(context.Context) error
	})
	expired := false
	if noACK {
		if !hasIdle || idle.ObserveCreateRecoveryIdle(ctx) != nil {
			return block("CREATE_ORIGINAL_NATIVE_ACK_UNPROVED")
		}
	} else {
		op, err := s.Native.NativeOperation(ctx, t.NativeID)
		expired = api.StatusErrorCheck(err, http.StatusNotFound)
		if expired {
			if n.OriginalOperationInactive(ctx, t.NativeID) != nil {
				return block("CREATE_ORIGINAL_NATIVE_ACTIVE_OR_UNVERIFIED")
			}
		} else if err != nil || op == nil || op.ID != t.NativeID || op.StatusCode != api.Success {
			return block("CREATE_ORIGINAL_NATIVE_SUCCESS_UNVERIFIED")
		}
	}
	matches := func(b native.BoxState) bool {
		return b.Project == t.Layer && b.Name == t.Name && b.UUID != "" &&
			b.CPU == strconv.Itoa(in.CPU) && b.Memory == fmt.Sprintf("%dMiB", in.MemoryMiB) &&
			b.RootDiskSize == fmt.Sprintf("%dGiB", in.DiskGiB) && b.Nesting == *in.Nesting && b.Internet == *in.Internet &&
			in.Options.Matches(b.NativeOptions) && ((*in.Start && b.NativeState == "Running") || (!*in.Start && b.NativeState == "Stopped"))
	}
	box, obs, err := ObserveCreate(ctx, s.Native, t.Name, t.ID)
	if err != nil || !matches(box) || obs.Fingerprint == "" {
		return block("CREATE_ORIGINAL_POSTSTATE_UNVERIFIED")
	}
	b, err := n.ObserveOriginalCreateStaticPeerBinding(ctx, t.Name, box.UUID, t.ID)
	if err != nil {
		return block("CREATE_STATIC_NATIVE_BINDING_UNVERIFIED")
	}
	again, after, err := ObserveCreate(ctx, s.Native, t.Name, t.ID)
	if err != nil || !matches(again) || again.UUID != box.UUID || after.Fingerprint != obs.Fingerprint {
		return block("CREATE_ORIGINAL_POSTSTATE_CHANGED")
	}
	if expired && n.OriginalOperationInactive(ctx, t.NativeID) != nil {
		return block("CREATE_ORIGINAL_NATIVE_ACTIVE_OR_UNVERIFIED")
	}
	reason := "CREATE_ORIGINAL_POSTSTATE_VERIFIED"
	if noACK {
		if idle.ObserveCreateRecoveryIdle(ctx) != nil {
			return block("CREATE_ORIGINAL_NATIVE_ACK_UNPROVED")
		}
		reason = "CREATE_ACK_LOST_STATIC_POSTSTATE_VERIFIED"
	}
	p := state.RecoveryProof{Kind: state.RecoverySettle, Reason: reason, NativeID: t.NativeID, UUID: box.UUID,
		CreateBinding: state.VerifiedCreateBinding(b), Fingerprint: fingerprint(t.ID, t.NativeID, box.UUID, obs.Fingerprint, b.Bridge, b.ACL, b.MAC, b.IPv4, b.IPv6)}
	if err = s.reader().CheckRecoveryCreateBinding(ctx, t, p); err != nil {
		return block("CREATE_STATIC_RESERVATION_OR_LEDGER_CONFLICT")
	}
	v.Disposition, v.ProposedAction, v.Reason = "SAFE_RECOVER", state.RecoverySettle, p.Reason
	v.NativeObservation, v.EvidenceSource, v.ObservedNativeUUID = "ORIGINAL_CREATE_STATIC_POSTSTATE_VERIFIED", "SQLITE_AND_NATIVE", box.UUID
	return v, p
}
