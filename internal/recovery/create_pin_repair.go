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

// A terminal original Create can be re-observed to repair only its own
// missing static ledger records. Never rewrite a mismatched existing row.
func (s Service) observeCreatePinRepair(ctx context.Context, t state.RecoveryTarget, v View) (View, state.RecoveryProof) {
	block := func(reason string) (View, state.RecoveryProof) {
		v.Disposition, v.Reason = state.RecoveryBlocked, reason
		return v, state.RecoveryProof{Kind: state.RecoveryBlocked, Reason: reason}
	}
	if s.Store == nil {
		return block("CREATE_PIN_REPAIR_STORE_UNAVAILABLE")
	}
	status, err := s.Store.CreatePinRepairStatus(ctx, t)
	if err != nil {
		return block("CREATE_PIN_REPAIR_LEDGER_UNVERIFIED")
	}
	switch status {
	case "COMPLETE":
		return s.observeHealthyCreatePin(ctx, t, v)
	case "MISSING":
	case "CONFLICT":
		return block("CREATE_PIN_LEDGER_CONFLICT")
	default:
		return block("CREATE_PIN_REPAIR_UNSUPPORTED")
	}
	n, ok := s.Native.(createRecoveryObserver)
	if !ok {
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
	if json.Unmarshal([]byte(t.SafeIntent), &in) != nil || in.Name != t.Name || in.CPU < 1 || in.MemoryMiB < 1 || in.DiskGiB < 1 ||
		in.Start == nil || in.Nesting == nil || in.Internet == nil || in.Options == nil {
		return block("CREATE_ORIGINAL_INTENT_UNVERIFIED")
	}
	noACK := t.NativeID == ""
	idle, hasIdle := s.Native.(interface{ ObserveCreateRecoveryIdle(context.Context) error })
	expired := false
	if noACK {
		if !hasIdle || idle.ObserveCreateRecoveryIdle(ctx) != nil {
			return block("CREATE_ORIGINAL_NATIVE_ACK_UNPROVED")
		}
	} else {
		op, e := s.Native.NativeOperation(ctx, t.NativeID)
		expired = api.StatusErrorCheck(e, http.StatusNotFound)
		if expired {
			if n.OriginalOperationInactive(ctx, t.NativeID) != nil {
				return block("CREATE_ORIGINAL_NATIVE_ACTIVE_OR_UNVERIFIED")
			}
		} else if e != nil || op == nil || op.ID != t.NativeID || op.StatusCode != api.Success {
			return block("CREATE_ORIGINAL_NATIVE_SUCCESS_UNVERIFIED")
		}
	}
	matches := func(b native.BoxState) bool {
		return b.Project == t.Layer && b.Name == t.Name && b.UUID == t.UUID && b.CPU == strconv.Itoa(in.CPU) &&
			b.Memory == fmt.Sprintf("%dMiB", in.MemoryMiB) && b.RootDiskSize == fmt.Sprintf("%dGiB", in.DiskGiB) &&
			b.Nesting == *in.Nesting && b.Internet == *in.Internet && in.Options.Matches(b.NativeOptions) &&
			((*in.Start && b.NativeState == "Running") || (!*in.Start && b.NativeState == "Stopped"))
	}
	box, first, e := ObserveCreate(ctx, s.Native, t.Name, t.ID)
	if e != nil || !matches(box) || first.Fingerprint == "" {
		return block("CREATE_ORIGINAL_POSTSTATE_UNVERIFIED")
	}
	pin, e := n.ObserveOriginalCreateStaticPeerBinding(ctx, t.Name, t.UUID, t.ID)
	if e != nil {
		return block("CREATE_STATIC_NATIVE_BINDING_UNVERIFIED")
	}
	second, again, e := ObserveCreate(ctx, s.Native, t.Name, t.ID)
	if e != nil || !matches(second) || again.Fingerprint != first.Fingerprint {
		return block("CREATE_ORIGINAL_POSTSTATE_CHANGED")
	}
	if noACK {
		if idle.ObserveCreateRecoveryIdle(ctx) != nil {
			return block("CREATE_ORIGINAL_NATIVE_ACK_UNPROVED")
		}
	} else if expired && n.OriginalOperationInactive(ctx, t.NativeID) != nil {
		return block("CREATE_ORIGINAL_NATIVE_ACTIVE_OR_UNVERIFIED")
	}
	p := state.RecoveryProof{Kind: state.RecoveryRepairPin, Reason: "CREATE_PIN_LEDGER_REPAIRED",
		NativeID: t.NativeID, UUID: t.UUID, CreateBinding: state.VerifiedCreateBinding(pin),
		Fingerprint: fingerprint(t.ID, t.NativeID, t.UUID, first.Fingerprint, pin.Bridge, pin.ACL, pin.MAC, pin.IPv4, pin.IPv6)}
	if e = s.Store.CheckCreatePinRepair(ctx, t, p); e != nil {
		return block("CREATE_PIN_REPAIR_CONFLICT_OR_BUSY")
	}
	v.Disposition, v.ProposedAction, v.Reason = "SAFE_RECOVER", state.RecoveryRepairPin, p.Reason
	v.NativeObservation, v.EvidenceSource, v.ObservedNativeUUID = "ORIGINAL_CREATE_STATIC_POSTSTATE_VERIFIED", "SQLITE_AND_NATIVE", t.UUID
	return v, p
}
