package install

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

// Package-private write capability. No MCP/socket caller may supply one.
type nativePeerScopedACLDriver interface {
	getACL(string) (*api.NetworkACL, string, error)
	casACL(string, api.NetworkACLPut, string) error
}
type nativePeerScopedIncusACLDriver struct{ root incus.InstanceServer }

func (d nativePeerScopedIncusACLDriver) getACL(name string) (*api.NetworkACL, string, error) {
	return d.root.UseProject("default").GetNetworkACL(name)
}
func (d nativePeerScopedIncusACLDriver) casACL(name string, put api.NetworkACLPut, etag string) error {
	return d.root.UseProject("default").UpdateNetworkACL(name, put, etag)
}

// A single ACL phase receipt is NOT a completed Peer operation. An uncertain
// ETag PUT must be fenced by the future durable Root original, never replayed.
type nativePeerScopedACLWriteReceipt struct {
	OriginalID, OwnerUUID, ACLName, OldETag, NewETag string
	Attempted, Verified                              bool
}

func nativePeerScopedACLFullMatch(got *api.NetworkACL, want api.NetworkACLsPost, usedBy string) bool {
	if got == nil || got.Project != "default" || got.Name != want.Name ||
		len(got.UsedBy) != 1 || got.UsedBy[0] != usedBy ||
		got.Description != want.Description || !maps.Equal(got.Config, want.Config) {
		return false
	}
	a, e1 := nativePeerScopedCanonicalRules(got.Ingress)
	b, e2 := nativePeerScopedCanonicalRules(want.Ingress)
	c, e3 := nativePeerScopedCanonicalRules(got.Egress)
	d, e4 := nativePeerScopedCanonicalRules(want.Egress)
	return e1 == nil && e2 == nil && e3 == nil && e4 == nil &&
		reflect.DeepEqual(a, b) && reflect.DeepEqual(c, d)
}

// One scoped owner; never a Project stop or boot.autostart mutation.
// On grant, guard remains in Before(DROP) until A/B ACLs are ready.
// On revoke, guard must already be After(DROP) before A/B ACL cleanup.
func nativePeerScopedACLReplaceOnce(ctx context.Context, plan NativePeerScopedRootTransition,
	uuid string, observed NativePeerGuardObservedOwner,
	guard nativePeerGuardWitness, driver nativePeerScopedACLDriver) (nativePeerScopedACLWriteReceipt, error) {
	rec := nativePeerScopedACLWriteReceipt{OriginalID: plan.OriginalID, OwnerUUID: uuid}
	if ctx == nil || guard == nil || driver == nil {
		return rec, errors.New("missing Root capability")
	}
	if err := ctx.Err(); err != nil {
		return rec, err
	}
	if _, err := nativePeerScopedRootEffectSequence(plan); err != nil {
		return rec, err
	}
	if uuid != plan.SourceUUID && uuid != plan.TargetUUID {
		return rec, errors.New("only exact A/B ACL may be written")
	}
	if len(plan.ExpectedBeforeByUUID) < 2 || len(plan.ExpectedBeforeByUUID) != len(plan.ExpectedAfterByUUID) ||
		len(plan.OriginalBootAutostart) != len(plan.ExpectedBeforeByUUID) ||
		plan.UnrelatedACLsVerifiedUnchanged != len(plan.ExpectedBeforeByUUID)-2 {
		return rec, errors.New("incomplete Project preimage")
	}
	for id, b := range plan.ExpectedBeforeByUUID {
		a, ok := plan.ExpectedAfterByUUID[id]
		if !ok || id != plan.SourceUUID && id != plan.TargetUUID && !reflect.DeepEqual(b, a) {
			return rec, errors.New("unrelated C/D ACL would change")
		}
	}
	before, hasB := plan.ExpectedBeforeByUUID[uuid]
	after, hasA := plan.ExpectedAfterByUUID[uuid]
	if !hasB || !hasA || reflect.DeepEqual(before, after) || before.Name == "" ||
		before.Name != after.Name ||
		uuid == plan.SourceUUID && before.Name != plan.SourceACLName ||
		uuid == plan.TargetUUID && before.Name != plan.TargetACLName {
		return rec, errors.New("no exact scoped ACL change")
	}
	if observed.Instance == nil || observed.ACL == nil || observed.ACLTag == "" ||
		observed.Instance.Project != plan.Project ||
		observed.Instance.Config["volatile.uuid"] != uuid ||
		observed.Instance.Config["boot.autostart"] != plan.OriginalBootAutostart[uuid] ||
		len(observed.ACL.UsedBy) != 1 ||
		!nativePeerGuardOwnedURL(observed.ACL.UsedBy[0], "instances", observed.Instance.Name, plan.Project) ||
		!nativePeerScopedACLFullMatch(observed.ACL, before, observed.ACL.UsedBy[0]) {
		return rec, errors.New("native owner, UsedBy or ACL preimage differs")
	}
	rec.ACLName = before.Name
	rec.OldETag = observed.ACLTag
	protected := plan.GuardBefore
	if !plan.IntentAllow {
		protected = plan.GuardAfter
	}
	if err := guard(ctx, protected); err != nil {
		return rec, fmt.Errorf("guard protection unavailable: %w", err)
	}
	current, etag, err := driver.getACL(before.Name)
	if err != nil || etag != observed.ACLTag ||
		!nativePeerScopedACLFullMatch(current, before, observed.ACL.UsedBy[0]) {
		return rec, errors.New("native ACL content or ETag changed before CAS")
	}
	if err := ctx.Err(); err != nil {
		return rec, err
	}
	if err := guard(ctx, protected); err != nil {
		return rec, errors.New("guard changed immediately before ACL CAS")
	}
	// Root must fsync an original-bound UNKNOWN ACL-stage intent BEFORE here.
	rec.Attempted = true
	if err := driver.casACL(before.Name, after.NetworkACLPut, etag); err != nil {
		return rec, fmt.Errorf("native ACL CAS result UNKNOWN, do not replay: %w", err)
	}
	applied, newTag, err := driver.getACL(before.Name)
	if err != nil || newTag == "" || newTag == etag ||
		!nativePeerScopedACLFullMatch(applied, after, observed.ACL.UsedBy[0]) {
		return rec, errors.New("ACL post-ACK readback UNKNOWN, do not replay")
	}
	if err := guard(ctx, protected); err != nil {
		return rec, errors.New("post-ACL guard UNKNOWN, do not replay")
	}
	rec.NewETag = newTag
	rec.Verified = true
	return rec, nil
}
