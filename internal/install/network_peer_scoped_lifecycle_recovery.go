package install

import (
	"context"
	"errors"
	"net/http"
	"reflect"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/peerscope"
	"github.com/vanishs/boxweave/internal/state"
)

// Only complete, immutable Root journals can be observed here. This path
// never calls a lifecycle apply, creates a journal, or changes native state.
func nativeScopedLifecycleRecover(ctx context.Context, db *state.Store, id, action string) (state.ScopedLifecycleRecoveryProof, error) {
	var zero state.ScopedLifecycleRecoveryProof
	kind := ""
	switch action {
	case "box_create":
		kind = "CREATE"
	case "box_delete":
		kind = "DELETE"
	default:
		return zero, errors.New("invalid lifecycle recovery action")
	}
	r, _, e := nativePeerFreshVerifiedPlan()
	if e != nil {
		return zero, e
	}
	if e = nativePeerFreshQuarantineReceipt(r); e != nil {
		return zero, e
	}
	rec, e := nativeScopedCompletedLifecycleReceipt(id, kind)
	if e != nil || rec.HostSHA256 != r.HostSHA256 || rec.GuardSHA256 != r.GuardSHA256 || rec.Project != r.Project || rec.Binding.Bridge != r.Bridge {
		return zero, errors.New("completed original lifecycle journal unavailable")
	}
	c, e := prepareClient(ctx)
	if e != nil {
		return zero, e
	}
	c = c.UseProject("default")
	own := c.UseProject(r.Project)
	originalSHA := ""
	for i := 0; i < 2; i++ {
		if e = nativeScopedRecoveryIdle(ctx, own, rec.NativeID); e != nil {
			return zero, e
		}
		_, _, nw, e := nativePeerFreshQuarantineRead(ctx, c, r, "")
		if e != nil {
			return zero, e
		}
		var sha string
		if kind == "CREATE" {
			p, e := db.ReadRootScopedCreateUnknownOriginal(ctx, id, r.Project, r.Bridge)
			if e != nil {
				return zero, e
			}
			b := p.Quarantine
			b.ACL, e = peerscope.ACLName(r.Project, r.Bridge, rec.UUID)
			if e != nil || b != rec.Binding || p.NativeID != rec.NativeID || p.Intent.Name != rec.Name || p.Intent.Start != rec.Start || p.Intent.Internet != rec.Internet || p.Intent.BootAutostart != rec.BootAutostart {
				return zero, errors.New("scoped Create original differs from completed journal")
			}
			if e = nativeScopedCompletedCreateRead(ctx, c, r, nw, db, p, rec); e != nil {
				return zero, e
			}
			sha = state.ScopedLifecycleOriginalSHA256(p)
		} else {
			p, e := db.ReadRootScopedDeleteUnknownOriginal(ctx, id, r.Project, r.Bridge)
			if e != nil {
				return zero, e
			}
			created, e := nativeScopedCreateReceipt(p.CreateID)
			if e != nil || created.HostSHA256 != r.HostSHA256 || created.GuardSHA256 != r.GuardSHA256 || created.Project != r.Project || created.UUID != p.Box.UUID || created.Name != p.Box.Name || created.Binding != p.Binding || p.NativeID != rec.NativeID || p.Box.UUID != rec.UUID || p.Box.Name != rec.Name || p.Binding != rec.Binding || rec.Start || rec.Internet || rec.BootAutostart {
				return zero, errors.New("scoped Delete original differs from completed journal")
			}
			if _, _, e = own.GetInstance(rec.Name); !api.StatusErrorCheck(e, http.StatusNotFound) {
				return zero, errors.New("scoped Delete absence unverified")
			}
			if _, _, e = c.GetNetworkACL(rec.Binding.ACL); !api.StatusErrorCheck(e, http.StatusNotFound) {
				return zero, errors.New("scoped Delete ACL absence unverified")
			}
			if e = nativeScopedLifecycleInventory(ctx, c, db, r, nw, "", "", rec.UUID); e != nil {
				return zero, e
			}
			sha = state.ScopedLifecycleOriginalSHA256(p)
		}
		if sha == "" || i > 0 && sha != originalSHA {
			return zero, errors.New("scoped lifecycle original changed")
		}
		originalSHA = sha
		if _, e = nativePeerFreshGuardCurrent(ctx, r.Project, r.Bridge); e != nil {
			return zero, e
		}
		if e = nativeScopedRecoveryIdle(ctx, own, rec.NativeID); e != nil {
			return zero, e
		}
	}
	again, e := nativeScopedCompletedLifecycleReceipt(id, kind)
	if e != nil || again != rec {
		return zero, errors.New("scoped lifecycle journal changed")
	}
	return state.ScopedLifecycleRecoveryProof{Record: rec, OriginalSHA256: originalSHA, RecoveryOnly: true}, nil
}

func nativeScopedRecoveryIdle(ctx context.Context, c incus.InstanceServer, id string) error {
	if id == "" {
		return errors.New("original native ACK absent")
	}
	if e := nativeScopedCreateOriginalInactive(ctx, c, id); e != nil {
		return e
	}
	ops, e := c.GetOperations()
	if e != nil {
		return e
	}
	for _, o := range ops {
		if o.ID == "" || !o.StatusCode.IsFinal() {
			return errors.New("native Project not quiescent")
		}
	}
	return ctx.Err()
}

func nativeScopedCompletedCreateRead(ctx context.Context, c incus.InstanceServer, r nativePeerFreshJournalRecord, nw *api.Network, db *state.Store, p state.ScopedCreateOriginalProof, rec scopedLifecycleRecord) error {
	inst, tag, e := c.UseProject(r.Project).GetInstance(rec.Name)
	if e != nil || inst == nil || tag == "" || inst.Config["volatile.uuid"] != rec.UUID {
		return errors.New("scoped Create exact UUID unverified")
	}
	if e = nativeScopedCreateShapeCheck(inst, p, rec.Binding.ACL, rec.BootAutostart, rec.Internet, rec.Start); e != nil {
		return e
	}
	want, e := NativePeerScopedACLFromPolicySnapshot(NativePrepareTarget{Project: r.Project, Network: r.Bridge, ACL: rec.Binding.ACL}, nw, nil, nil, NativePeerACLSubject{UUID: rec.UUID, IPv4: rec.Binding.IPv4, IPv6: rec.Binding.IPv6})
	if e != nil {
		return e
	}
	actual, at, e := c.GetNetworkACL(rec.Binding.ACL)
	if e != nil || actual == nil || at == "" || actual.Project != "default" || !reflect.DeepEqual(actual.Writable(), want.NetworkACLPut) || len(actual.UsedBy) != 1 || !nativePeerGuardOwnedURL(actual.UsedBy[0], "instances", rec.Name, r.Project) {
		return errors.New("completed Create ACL unverified")
	}
	if e = nativeScopedCreateLeaseCheck(c, r.Bridge, rec.Name, rec.Binding, !rec.Start); e != nil {
		return e
	}
	return nativeScopedLifecycleInventory(ctx, c, db, r, nw, rec.Name, rec.Binding.ACL, "")
}
