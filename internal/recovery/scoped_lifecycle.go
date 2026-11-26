package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// No controller-side replay of Create, start, Delete or privileged ACL
// writes is reachable from this observer. Partial journals and missing
// original ACKs remain blocked, even if native name or absence matches.
func (s Service) observeScopedLifecycle(ctx context.Context, t state.RecoveryTarget, v View) (View, state.RecoveryProof) {
	block := func(reason string) (View, state.RecoveryProof) {
		v.Disposition, v.Reason = state.RecoveryBlocked, reason
		return v, state.RecoveryProof{Kind: state.RecoveryBlocked, Reason: reason}
	}
	if t.NativeID == "" {
		return block("SCOPED_LIFECYCLE_ORIGINAL_ACK_UNPROVED")
	}
	checker, ok := s.reader().(interface {
		CheckScopedLifecycleRecovery(context.Context, state.RecoveryTarget, state.RecoveryProof) error
	})
	if !ok || s.ScopedLifecycleWitness == nil || s.Native == nil {
		return block("SCOPED_LIFECYCLE_ROOT_PROOF_REQUIRED")
	}
	w, e := s.ScopedLifecycleWitness(ctx, t.ID)
	r := w.Record
	if e != nil || !w.RecoveryOnly || r.Schema != 2 || r.Status != "VERIFIED" || r.OriginalID != t.ID || r.NativeID != t.NativeID || r.Project != s.Layer || r.Name != t.Name || r.UUID == "" {
		return block("SCOPED_LIFECYCLE_ROOT_PROOF_REQUIRED")
	}
	var observed string
	switch t.Action {
	case "box_create":
		if r.Kind != "CREATE" {
			return block("SCOPED_LIFECYCLE_ROOT_PROOF_REQUIRED")
		}
		var intent state.ScopedCreateIntent
		var options native.InstanceOptions
		if json.Unmarshal([]byte(t.SafeIntent), &intent) != nil || json.Unmarshal(intent.Options, &options) != nil || intent.Schema != 2 || intent.Project != t.Layer || intent.Name != t.Name || intent.Bridge != r.Binding.Bridge || intent.Start != r.Start || intent.Internet != r.Internet || intent.BootAutostart != r.BootAutostart {
			return block("SCOPED_CREATE_INTENT_UNVERIFIED")
		}
		options = options.ForCreate()
		if *options.BootAutostart != intent.BootAutostart {
			return block("SCOPED_CREATE_INTENT_UNVERIFIED")
		}
		pinReader, ok := s.Native.(interface {
			VerifyPinnedDeleteNIC(context.Context, string, string, string, string, string, string, string) error
		})
		if !ok {
			return block("SCOPED_CREATE_BINDING_UNVERIFIED")
		}
		for i := 0; i < 2; i++ {
			b, o, e := ObserveCreate(ctx, s.Native, t.Name, t.ID)
			if e != nil || b.UUID != r.UUID || b.Project != r.Project || b.CPU != strconv.Itoa(intent.CPU) || b.Memory != fmt.Sprintf("%dMiB", intent.MemoryMiB) || b.RootDiskSize != fmt.Sprintf("%dGiB", intent.DiskGiB) || b.Internet != intent.Internet || b.Nesting != intent.Nesting || !options.Matches(b.NativeOptions) || intent.Start && b.NativeState != "Running" && b.NativeState != "Ready" || !intent.Start && b.NativeState != "Stopped" || o.Fingerprint == "" || i > 0 && o.Fingerprint != observed {
				return block("SCOPED_CREATE_FINAL_POSTSTATE_UNVERIFIED")
			}
			x := r.Binding
			if e = pinReader.VerifyPinnedDeleteNIC(ctx, r.Name, r.UUID, x.Bridge, x.ACL, x.MAC, x.IPv4, x.IPv6); e != nil {
				return block("SCOPED_CREATE_BINDING_UNVERIFIED")
			}
			observed = o.Fingerprint
		}
	case "box_delete":
		if r.Kind != "DELETE" || r.UUID != t.UUID || t.Incarnation != r.UUID {
			return block("SCOPED_LIFECYCLE_ROOT_PROOF_REQUIRED")
		}
		for i := 0; i < 2; i++ {
			o, e := s.Native.ObserveOwnedBox(ctx, t.Name, t.UUID)
			if e != nil || o.State != "ABSENT_UNCERTAIN" || o.Project != t.Layer || o.Name != t.Name || o.ExpectedUUID != t.UUID || o.Fingerprint == "" || i > 0 && observed != o.Fingerprint {
				return block("SCOPED_DELETE_FINAL_ABSENCE_UNVERIFIED")
			}
			observed = o.Fingerprint
		}
	default:
		return block("SCOPED_LIFECYCLE_ROOT_PROOF_REQUIRED")
	}
	p := state.RecoveryProof{Kind: state.RecoverySettle, Reason: "SCOPED_LIFECYCLE_COMPLETED_NO_REPLAY", NativeID: r.NativeID, UUID: r.UUID, ScopedLifecycle: w, Fingerprint: fingerprint(t.ID, w.OriginalSHA256, r.PlanSHA256, observed)}
	if t.Action == "box_create" {
		p.CreateBinding = r.Binding
	}
	if e = checker.CheckScopedLifecycleRecovery(ctx, t, p); e != nil {
		return block("SCOPED_LIFECYCLE_ORIGINAL_OR_CLAIMS_CHANGED")
	}
	v.Disposition, v.ProposedAction, v.Reason = "SAFE_RECOVER", state.RecoverySettle, p.Reason
	v.NativeObservation, v.EvidenceSource, v.ObservedNativeUUID = "ROOT_COMPLETED_AND_NATIVE_POSTSTATE_VERIFIED", "SQLITE_AND_ROOT_JOURNAL", r.UUID
	return v, p
}
