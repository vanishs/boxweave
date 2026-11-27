package install

import (
	"context"
	"errors"
	"net/http"
	"os"
	"reflect"
	"slices"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/peerscope"
	"github.com/vanishs/boxweave/internal/state"
)

// ScopedOfflineTarget binds the independently observed exact current owner,
// not a new business Delete or authorization to adopt an unknown effect.
type ScopedOfflineTarget struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	ACL            string `json:"acl"`
	InstanceSHA256 string `json:"instance_sha256"`
	ACLSHA256      string `json:"acl_sha256"`
}

// ScopedOfflineReview is read-only. It cannot be used as a normal retirement
// digest or as a permit. The archive and operator approval are separate.
type ScopedOfflineReview struct {
	Project      string                      `json:"project"`
	LedgerSHA256 string                      `json:"ledger_sha256"`
	ReviewSHA256 string                      `json:"review_sha256"`
	Originals    []state.MaintenanceOriginal `json:"originals"`
	Targets      []ScopedOfflineTarget       `json:"targets"`
	Table        string                      `json:"table"`
	Retired      int                         `json:"retired"`
}

func scopedOfflineStopped(ctx context.Context, archived bool) error {
	if os.Geteuid() != 0 {
		return errors.New("scoped offline review requires local root")
	}
	if e := scopedRetirementInactive(ctx, "boxweave.service"); e != nil {
		return e
	}
	for _, unit := range []string{"incus.service", scopedGuardUnitName} {
		active, e := scopedRetirementUnitProperty(ctx, unit, "ActiveState")
		if e != nil || active != "active" {
			return errors.New("scoped offline helper or Incus inactive")
		}
	}
	if archived {
		return scopedRetirementInactive(ctx, scopedLifecycleUnitName)
	}
	active, e := scopedRetirementUnitProperty(ctx, scopedLifecycleUnitName, "ActiveState")
	if e != nil {
		return e
	}
	if active != "active" {
		return scopedRetirementInactive(ctx, scopedLifecycleUnitName)
	}
	return nil
}

func scopedOfflineLedger(ctx context.Context, project, bridge string) (state.ScopedOfflineArchiveLedger, error) {
	var zero state.ScopedOfflineArchiveLedger
	s, e := state.OpenDiagnosticSnapshot(ctx, nativePeerRootInstalledDB)
	if e != nil {
		return zero, e
	}
	if s.Capture != "SQLITE_EXCLUSIVE_COLD_COPY" {
		_ = s.Close()
		return zero, errors.New("exclusive cold ledger proof required")
	}
	l, e := s.ReadScopedOfflineArchiveLedger(ctx, project, bridge)
	ce := s.Close()
	if e != nil {
		return zero, e
	}
	if ce != nil {
		return zero, ce
	}
	return l, nil
}

// A prefix consists of full instance+ACL pairs, optionally one detached
// ACL, then untouched owners. Missing or reordered effects are not adopted.
func scopedOfflinePrefix(states []int) (int, error) {
	prefix := 0
	untouched := false
	for _, v := range states {
		if v < 0 || v > 2 || untouched && v != 0 {
			return 0, errors.New("scoped offline non-prefix native state")
		}
		prefix += v
		if v != 2 {
			untouched = true
		}
	}
	return prefix, nil
}

func scopedOfflineNativeTargets(ctx context.Context, c incus.InstanceServer, r nativePeerFreshJournalRecord, nw *api.Network, l state.ScopedOfflineArchiveLedger, approved []ScopedOfflineTarget) ([]ScopedOfflineTarget, int, error) {
	fail := func(e error) ([]ScopedOfflineTarget, int, error) { return nil, 0, e }
	all, e := c.GetInstancesAllProjects(api.InstanceTypeAny)
	if e != nil || len(all) > len(l.Current) {
		return fail(errors.New("scoped offline native instance inventory differs"))
	}
	instances := map[string]api.Instance{}
	for _, i := range all {
		if i.Project != r.Project || i.Status != "Stopped" || i.Type != "container" || instances[i.Name].Name != "" {
			return fail(errors.New("scoped offline foreign or unstopped instance"))
		}
		instances[i.Name] = i
	}
	names, e := c.GetNetworkACLNames()
	if e != nil || len(names) > len(l.Current)+1 {
		return fail(errors.New("scoped offline ACL inventory differs"))
	}
	want := []string{peerscope.QuarantineACL}
	out := make([]ScopedOfflineTarget, 0, len(l.Current))
	states := []int{}
	if approved != nil && len(approved) != len(l.Current) {
		return fail(errors.New("scoped offline approved target count differs"))
	}
	for n, b := range l.Current {
		i, ok := instances[b.Name]
		if ok {
			if i.Config["volatile.uuid"] != b.UUID || i.Config["user.boxweave.create_operation"] != b.OriginalID {
				return fail(errors.New("scoped offline original instance mismatch"))
			}
			proof, e := scopedOfflineTarget(ctx, c, r, nw, l, b, &i)
			if e != nil {
				return fail(e)
			}
			if approved != nil && proof != approved[n] {
				return fail(errors.New("scoped offline target drift"))
			}
			out = append(out, proof)
			states = append(states, 0)
			want = append(want, b.ACL)
			delete(instances, b.Name)
			continue
		}
		if approved == nil {
			return fail(errors.New("scoped offline initial owner absent"))
		}
		a := approved[n]
		if a.UUID != b.UUID || a.Name != b.Name || a.ACL != b.ACL {
			return fail(errors.New("scoped offline approved identity differs"))
		}
		v, tag, e := c.GetNetworkACL(b.ACL)
		if api.StatusErrorCheck(e, http.StatusNotFound) {
			states = append(states, 2)
		} else if e == nil && v != nil && tag != "" && v.Project == "default" && len(v.UsedBy) == 0 && nativePrepareStateDigest(v.Writable()) == a.ACLSHA256 {
			states = append(states, 1)
			want = append(want, b.ACL)
		} else {
			return fail(errors.New("scoped offline detached ACL drift"))
		}
		out = append(out, a)
	}
	slices.Sort(names)
	slices.Sort(want)
	if len(instances) != 0 || !slices.Equal(names, want) {
		return fail(errors.New("scoped offline foreign native consumer"))
	}
	prefix, e := scopedOfflinePrefix(states)
	if e != nil {
		return fail(e)
	}
	return out, prefix, nil
}

func scopedOfflineTarget(ctx context.Context, c incus.InstanceServer, r nativePeerFreshJournalRecord, nw *api.Network, l state.ScopedOfflineArchiveLedger, b state.NativeNetworkBox, i *api.Instance) (ScopedOfflineTarget, error) {
	var zero ScopedOfflineTarget
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	for _, dev := range i.ExpandedDevices {
		if dev["type"] == "proxy" {
			return zero, errors.New("scoped offline unexpected proxy")
		}
	}
	claim := native.PeerACLBox{Name: b.Name, UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6}
	if e := native.ValidatePeerACLBox(i, r.Project, r.Bridge, b.ACL, claim, false); e != nil {
		return zero, e
	}
	binding := state.VerifiedCreateBinding{Bridge: b.Bridge, ACL: b.ACL, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6}
	if e := nativeScopedCreateLeaseCheck(c.UseProject(r.Project), r.Bridge, b.Name, binding, true); e != nil {
		return zero, e
	}
	want, e := NativePeerScopedACLFromPolicySnapshot(NativePrepareTarget{Project: r.Project, Network: r.Bridge, ACL: b.ACL}, nw, l.Policy.Active, l.Policy.Revoked, NativePeerACLSubject{UUID: b.UUID, IPv4: b.IPv4, IPv6: b.IPv6})
	if e != nil {
		return zero, e
	}
	a, tag, e := c.GetNetworkACL(b.ACL)
	if e != nil || a == nil || tag == "" || a.Project != "default" || !reflect.DeepEqual(a.Writable(), want.NetworkACLPut) || len(a.UsedBy) != 1 || !nativePeerGuardOwnedURL(a.UsedBy[0], "instances", b.Name, r.Project) {
		return zero, errors.New("scoped offline ACL preimage drift")
	}
	return ScopedOfflineTarget{b.UUID, b.Name, b.ACL, nativePrepareStateDigest(i.Writable(), i.ExpandedConfig, i.ExpandedDevices), nativePrepareStateDigest(a.Writable())}, nil
}

type scopedOfflineObservation struct {
	Review ScopedOfflineReview
	Prefix int
}

func scopedOfflineReviewRead(ctx context.Context, project string, approved *ScopedOfflineReview) (scopedOfflineObservation, error) {
	var zero scopedOfflineObservation
	if e := scopedOfflineStopped(ctx, approved != nil); e != nil {
		return zero, e
	}
	r, initial, e := nativePeerFreshVerifiedPlan()
	if e != nil || r.Project != project {
		return zero, errors.New("scoped offline original scope unverified")
	}
	if e = nativePeerFreshQuarantineReceipt(r); e != nil {
		return zero, e
	}
	pre, e := scopedRetirementPreimageRead(r)
	if e != nil {
		return zero, e
	}
	guard, last, e := scopedPeerBootCurrent(nativePeerFreshJournalDir, 0, r, initial, "")
	if e != nil || len(last.Active) != 0 {
		return zero, errors.New("scoped offline terminal guard unverified")
	}
	if e = NativePeerGuardKernelReadback(ctx, guard); e != nil {
		return zero, e
	}
	l, e := scopedOfflineLedger(ctx, project, r.Bridge)
	if e != nil {
		return zero, e
	}
	receipts, e := scopedRetirementReceiptSet(r, l.Inventory, l.Current)
	if e != nil {
		return zero, e
	}
	c, e := prepareClient(ctx)
	if e != nil {
		return zero, e
	}
	c = c.UseProject("default")
	pf, pr, nw, e := nativePeerFreshQuarantineRead(ctx, c, r, "")
	if e != nil {
		return zero, e
	}
	owners := map[string]string{}
	for _, b := range l.Current {
		owners[b.Name] = b.UUID
	}
	if !reflect.DeepEqual(pf.Writable(), pre.AfterProfile) || !reflect.DeepEqual(pr.Writable(), pre.AfterProject) || !reflect.DeepEqual(nw.Writable(), pre.Bridge) || nativePeerBridgeConsumersOwned(nw.UsedBy, project, owners) != nil {
		return zero, errors.New("scoped offline native preimage drift")
	}
	if e = nativePeerBridgeKernelNoLivePorts(ctx, r.Bridge); e != nil {
		return zero, e
	}
	ops, e := c.UseProject(project).GetOperations()
	if e != nil || len(ops) > 8192 {
		return zero, errors.New("scoped offline native operations unverified")
	}
	for _, op := range ops {
		if !op.StatusCode.IsFinal() {
			return zero, errors.New("scoped offline native operation active")
		}
	}
	var expected []ScopedOfflineTarget
	if approved != nil {
		expected = approved.Targets
	}
	targets, prefix, e := scopedOfflineNativeTargets(ctx, c, r, nw, l, expected)
	if e != nil {
		return zero, e
	}
	if e = scopedOfflineStopped(ctx, approved != nil); e != nil {
		return zero, e
	}
	review := ScopedOfflineReview{Project: project, LedgerSHA256: l.LedgerSHA256, Originals: l.Originals, Targets: targets, Table: guard.Table, Retired: len(l.Inventory.Retired)}
	review.ReviewSHA256 = nativePrepareStateDigest("scoped-offline-archive-review-v1", r, pre, l, receipts, last, guard.DigestSHA256, review)
	if approved != nil && !reflect.DeepEqual(review, *approved) {
		return zero, errors.New("scoped offline original review drift")
	}
	return scopedOfflineObservation{review, prefix}, nil
}

func NativeScopedOfflineArchiveReview(ctx context.Context, project string) (ScopedOfflineReview, error) {
	a, e := scopedOfflineReviewRead(ctx, project, nil)
	if e != nil {
		return a.Review, e
	}
	b, e := scopedOfflineReviewRead(ctx, project, nil)
	if e != nil || !reflect.DeepEqual(a, b) || ctx.Err() != nil {
		return ScopedOfflineReview{}, errors.New("scoped offline review changed or unavailable")
	}
	return b.Review, nil
}
