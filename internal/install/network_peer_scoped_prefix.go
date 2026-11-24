package install

import (
	"context"
	"errors"
	"maps"
	"reflect"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// Exact whole-tenant native image at one prefix. Never compare only the
// modified ACL: sibling ACLs, instance identity/config/boot, network ETag
// and every live bridge port must remain unchanged.
func nativeScopedPeerPrefix(ctx context.Context, r nativePeerScopedRootReader, proof state.ScopedPeerOriginalProof, plan NativePeerScopedRootTransition, base nativePeerScopedSDKSnapshot, sysfs string, guard nativePeerGuardWitness, prefix int) (nativePeerScopedSDKSnapshot, string, error) {
	var zero nativePeerScopedSDKSnapshot
	steps, e := nativePeerScopedRootEffectSequence(plan)
	if e != nil || r == nil || guard == nil || prefix < 0 || prefix > len(steps) {
		return zero, "", errors.New("invalid scoped Peer prefix")
	}
	want := plan
	want.ExpectedBeforeByUUID = maps.Clone(plan.ExpectedBeforeByUUID)
	changed := map[string]bool{}
	g := plan.GuardBefore
	for _, s := range steps[:prefix] {
		switch s.Kind {
		case nativePeerScopedACLSource, nativePeerScopedACLTarget:
			want.ExpectedBeforeByUUID[s.OwnerUUID] = plan.ExpectedAfterByUUID[s.OwnerUUID]
			changed[s.OwnerUUID] = true
		case nativePeerScopedGuardBatch:
			g = plan.GuardAfter
		}
	}
	claims := make([]native.PeerACLBox, 0, len(proof.Inventory.Boxes))
	for _, b := range proof.Inventory.Boxes {
		claims = append(claims, native.PeerACLBox{Name: b.Name, UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6})
	}
	var first nativePeerScopedSDKSnapshot
	for i := 0; i < 2; i++ {
		now, e := nativePeerScopedRootReadSnapshot(ctx, r, proof)
		if e != nil {
			return zero, "", e
		}
		if !reflect.DeepEqual(now.Network, base.Network) || now.NetworkETag != base.NetworkETag || !maps.Equal(now.InstanceETags, base.InstanceETags) ||
			!reflect.DeepEqual(now.Evidence.AllProjectNames, base.Evidence.AllProjectNames) || !reflect.DeepEqual(now.Evidence.RootNetworkUsedBy, base.Evidence.RootNetworkUsedBy) {
			return zero, "", errors.New("scoped native bridge or instance ETag drift")
		}
		for name, o := range now.Evidence.Observed {
			b, ok := base.Evidence.Observed[name]
			if !ok || o.Instance == nil || !reflect.DeepEqual(b.Instance, o.Instance) {
				return zero, "", errors.New("scoped native instance drift")
			}
			uuid := o.Instance.Config["volatile.uuid"]
			if changed[uuid] == (o.ACLTag == b.ACLTag) {
				return zero, "", errors.New("unexpected native ACL ETag transition")
			}
		}
		if _, e = NativePeerGuardValidateScopedInventory(plan.Project, plan.Bridge, claims, now.Evidence.AllProjectNames, now.Evidence.Observed, now.Evidence.RootNetworkUsedBy); e != nil {
			return zero, "", e
		}
		if _, e = NativePeerScopedVerifyObservedACLPreimage(want, now.Evidence); e != nil {
			return zero, "", e
		}
		if e = nativePeerGuardKernelPortsExactAt(ctx, sysfs, plan.Bridge, claims, now.Evidence.Observed); e != nil {
			return zero, "", e
		}
		if e = guard(ctx, g); e != nil {
			return zero, "", e
		}
		if i == 0 {
			first = now
		} else if !reflect.DeepEqual(now, first) {
			return zero, "", errors.New("native scoped prefix changed during independent reads")
		}
	}
	if e = ctx.Err(); e != nil {
		return zero, "", e
	}
	return first, nativePrepareStateDigest("scoped-peer-prefix-v2", prefix, g.DigestSHA256, first), nil
}
