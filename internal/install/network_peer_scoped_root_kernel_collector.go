package install

import (
	"context"
	"errors"
	"reflect"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// Package-private read-only evidence. Not Root authorization, a lock, or
// permission to PUT ACLs, edit nft rules, purge conntrack, or finalize Peers.
type nativePeerScopedRootKernelReadResult struct {
	Read              nativePeerScopedRootReadResult
	GuardBeforeSHA256 string
	VerifiedBoxCount  int
	KernelGuardChecks int
}

// Two independent full double-pass SDK captures frame two sysfs and nft
// preimage witnesses. Every unrelated sibling must remain fully accounted.
// The future native effect owner must still hold durable locks and repeat
// this witness at the actual irreversible mutation boundary.
func nativePeerScopedRootKernelWithDriver(ctx context.Context,
	r nativePeerScopedRootReader, proof state.ScopedPeerOriginalProof,
	sysClassNet string, guard nativePeerGuardWitness,
) (nativePeerScopedRootKernelReadResult, error) {
	var zero nativePeerScopedRootKernelReadResult
	if ctx == nil || r == nil || guard == nil || sysClassNet == "" {
		return zero, errors.New("missing Root SDK, kernel guard or sysfs reader")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	first, err := nativePeerScopedRootReadOnlyWithDriver(ctx, r, proof)
	if err != nil {
		return zero, err
	}
	claims := make([]native.PeerACLBox, 0, len(proof.Inventory.Boxes))
	for _, b := range proof.Inventory.Boxes {
		claims = append(claims, native.PeerACLBox{
			Name: b.Name, UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6,
		})
	}
	if err = nativePeerGuardKernelPortsExactAt(ctx, sysClassNet, proof.Intent.Bridge,
		claims, first.observed); err != nil {
		return zero, errors.New("Root bridge kernel ports do not match full tenant SDK")
	}
	if err = guard(ctx, first.Plan.GuardBefore); err != nil {
		return zero, errors.New("Root nft kernel guard exact preimage not verified")
	}
	if err = ctx.Err(); err != nil {
		return zero, err
	}
	second, err := nativePeerScopedRootReadOnlyWithDriver(ctx, r, proof)
	if err != nil {
		return zero, err
	}
	if !reflect.DeepEqual(first, second) {
		return zero, errors.New("Root tenant/ACL/bridge/ETag drift between independent kernel reads")
	}
	if err = nativePeerGuardKernelPortsExactAt(ctx, sysClassNet, proof.Intent.Bridge,
		claims, second.observed); err != nil {
		return zero, errors.New("Root kernel veth/bridge port inventory changed during read")
	}
	if err = guard(ctx, second.Plan.GuardBefore); err != nil {
		return zero, errors.New("Root nft kernel guard drifted during independent read")
	}
	if err = ctx.Err(); err != nil {
		return zero, err
	}
	return nativePeerScopedRootKernelReadResult{
		Read: second, GuardBeforeSHA256: second.Plan.GuardBefore.DigestSHA256,
		VerifiedBoxCount: len(claims), KernelGuardChecks: 2,
	}, nil
}

// This real Root adapter remains READ-ONLY. Its caller must independently
// authenticate the restricted SQLite descriptor and local Incus SDK.
// An SDK pointer is not a credential and does not pin the kernel state.
func nativePeerScopedRootCollectKernelReadOnly(ctx context.Context,
	store *state.Store, root incus.InstanceServer,
	original, project, bridge string) (nativePeerScopedRootKernelReadResult, error) {
	var zero nativePeerScopedRootKernelReadResult
	if ctx == nil || store == nil || root == nil {
		return zero, errors.New("missing protected original or Root Unix SDK")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	first, err := store.ReadRootScopedPeerOriginal(ctx, original, project, bridge)
	if err != nil {
		return zero, err
	}
	result, err := nativePeerScopedRootKernelWithDriver(ctx,
		nativePeerScopedIncusReader{root: root}, first,
		"/sys/class/net", NativePeerGuardKernelReadback)
	if err != nil {
		return zero, err
	}
	second, err := store.ReadRootScopedPeerOriginal(ctx, original, project, bridge)
	if err != nil || !reflect.DeepEqual(first, second) {
		return zero, errors.New("protected Root SQLite original or peer history changed during kernel preflight")
	}
	return result, nil
}
