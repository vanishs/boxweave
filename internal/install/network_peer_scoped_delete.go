package install

import (
	"context"
	"errors"
	"net/http"
	"reflect"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

type scopedDeleteNativeProof struct {
	Original state.ScopedDeleteOriginalProof
	Create   scopedLifecycleRecord
	ACL      api.NetworkACLPut
	ETag     string
}

func nativeScopedCreateReceipt(id string) (scopedLifecycleRecord, error) {
	var zero scopedLifecycleRecord
	path, err := scopedLifecyclePath(id)
	if err != nil {
		return zero, err
	}
	d, err := nativePeerFreshJournalOpenAt(path, 0)
	if err != nil {
		return zero, err
	}
	defer d.Close()
	a, err := scopedLifecycleRead(d, "DISPATCH_UNKNOWN")
	if err != nil {
		return zero, err
	}
	if a.Kind != "CREATE" || a.OriginalID != id {
		return zero, errors.New("wrong original Create receipt")
	}
	for _, status := range []string{"ACL_VERIFIED", "VERIFIED"} {
		r, err := scopedLifecycleRead(d, status)
		a.Status = status
		if err != nil || a != r {
			return zero, errors.New("incomplete scoped Create receipt")
		}
	}
	return a, nil
}
func nativeScopedDeleteReview(ctx context.Context, c incus.InstanceServer, db *state.Store, id string, r nativePeerFreshJournalRecord) (scopedDeleteNativeProof, error) {
	var zero scopedDeleteNativeProof
	p, err := db.ReadRootScopedDeleteOriginal(ctx, id, r.Project, r.Bridge)
	if err != nil {
		return zero, err
	}
	rec, err := nativeScopedCreateReceipt(p.CreateID)
	if err != nil || rec.Project != r.Project || rec.HostSHA256 != r.HostSHA256 || rec.GuardSHA256 != r.GuardSHA256 || rec.Name != p.Box.Name || rec.UUID != p.Box.UUID || rec.Binding != p.Binding {
		return zero, errors.New("scoped Delete missing original Root Create evidence")
	}
	own := c.UseProject(r.Project)
	if err = nativeScopedCreateOriginalInactive(ctx, own, p.NativeID); err != nil {
		return zero, err
	}
	if _, _, err = own.GetInstance(p.Box.Name); !api.StatusErrorCheck(err, http.StatusNotFound) {
		return zero, errors.New("scoped Delete instance absence unverified")
	}
	_, _, nw, err := nativePeerFreshQuarantineRead(ctx, c, r, "")
	if err != nil {
		return zero, err
	}
	if err = nativeScopedLifecycleInventory(ctx, c, db, r, nw, "", p.Binding.ACL, p.Box.UUID); err != nil {
		return zero, err
	}
	want, err := NativePeerScopedACLFromPolicySnapshot(NativePrepareTarget{Project: r.Project, Network: r.Bridge, ACL: p.Binding.ACL}, nw, nil, nil, NativePeerACLSubject{UUID: p.Box.UUID, IPv4: p.Binding.IPv4, IPv6: p.Binding.IPv6})
	if err != nil {
		return zero, err
	}
	a, tag, err := c.GetNetworkACL(p.Binding.ACL)
	if err != nil || a == nil || tag == "" || a.Project != "default" || len(a.UsedBy) != 0 || !reflect.DeepEqual(a.Writable(), want.NetworkACLPut) {
		return zero, errors.New("scoped Delete ACL is not exact detached original")
	}
	return scopedDeleteNativeProof{p, rec, a.Writable(), tag}, nil
}

func nativeScopedDeleteApply(ctx context.Context, db *state.Store, id string) (scopedLifecycleRecord, error) {
	var zero scopedLifecycleRecord
	r, _, err := nativePeerFreshVerifiedPlan()
	if err != nil {
		return zero, err
	}
	if err = nativePeerFreshQuarantineReceipt(r); err != nil {
		return zero, err
	}
	if _, err = nativePeerFreshGuardCurrent(ctx, r.Project, r.Bridge); err != nil {
		return zero, err
	}
	c, err := prepareClient(ctx)
	if err != nil {
		return zero, err
	}
	c = c.UseProject("default")
	p, err := nativeScopedDeleteReview(ctx, c, db, id, r)
	if err != nil {
		return zero, err
	}
	path, err := scopedLifecyclePath(id)
	if err != nil {
		return zero, err
	}
	d, err := nativePeerFreshJournalCreateAt(path, 0)
	if err != nil {
		return zero, err
	}
	defer d.Close()
	rec := scopedLifecycleRecord{Schema: 2, Kind: "DELETE", Status: "DISPATCH_UNKNOWN", OriginalID: id, NativeID: p.Original.NativeID, HostSHA256: r.HostSHA256, GuardSHA256: r.GuardSHA256,
		Project: r.Project, Name: p.Original.Box.Name, UUID: p.Original.Box.UUID, Binding: p.Original.Binding, PlanSHA256: nativePrepareStateDigest("scoped-delete-v2", p)}
	if err = scopedLifecycleWrite(d, rec); err != nil {
		return zero, err
	}
	again, err := nativeScopedDeleteReview(ctx, c, db, id, r)
	if err != nil || !reflect.DeepEqual(again, p) {
		return rec, errors.New("scoped Delete preimage changed; preserve UNKNOWN")
	}
	if _, err = nativePeerFreshGuardCurrent(ctx, r.Project, r.Bridge); err != nil {
		return rec, err
	}
	if err = c.DeleteNetworkACL(rec.Binding.ACL); err != nil {
		return rec, err
	}
	for i := 0; i < 2; i++ {
		if _, _, err = c.GetNetworkACL(rec.Binding.ACL); !api.StatusErrorCheck(err, http.StatusNotFound) {
			return rec, errors.New("scoped Delete ACL absence unverified")
		}
		if _, _, err = c.UseProject(r.Project).GetInstance(rec.Name); !api.StatusErrorCheck(err, http.StatusNotFound) {
			return rec, errors.New("scoped Delete instance absence changed")
		}
		_, _, nw, err := nativePeerFreshQuarantineRead(ctx, c, r, "")
		if err != nil {
			return rec, err
		}
		if err = nativeScopedLifecycleInventory(ctx, c, db, r, nw, "", "", rec.UUID); err != nil {
			return rec, err
		}
		if _, err = nativePeerFreshGuardCurrent(ctx, r.Project, r.Bridge); err != nil {
			return rec, err
		}
	}
	rec.Status = "VERIFIED"
	if err = scopedLifecycleWrite(d, rec); err != nil {
		return rec, err
	}
	return rec, nil
}
