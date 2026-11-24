package install

import (
	"context"
	"errors"
	"reflect"

	"github.com/vanishs/boxweave/internal/state"
)

// Fixed production-capability composition, private until the installed
// socket, lifecycle policy coexistence and real native acceptance pass.
// The caller must hold a Root-wide lifecycle lock. No caller may supply
// native tuples, ACL names, journal paths or executable commands.
func nativeScopedPeerApply(ctx context.Context, db *state.Store, id string) (state.ScopedPeerRootReceipt, error) {
	var zero state.ScopedPeerRootReceipt
	if db == nil || !db.IsNetworkAuditReadOnly() || !scopedLifecycleID.MatchString(id) {
		return zero, errors.New("untrusted scoped Root original")
	}
	r, initial, e := nativePeerFreshVerifiedPlan()
	if e != nil {
		return zero, e
	}
	if e = nativePeerFreshQuarantineReceipt(r); e != nil {
		return zero, e
	}
	current, _, e := scopedPeerBootCurrent(nativePeerFreshJournalDir, 0, r, initial, "")
	if e != nil {
		return zero, e
	}
	if e = NativePeerGuardKernelReadback(ctx, current); e != nil {
		return zero, e
	}
	c, e := prepareClient(ctx)
	if e != nil {
		return zero, e
	}
	c = c.UseProject("default")
	read := func(ctx context.Context) (state.ScopedPeerOriginalProof, error) {
		return db.ReadRootScopedPeerOriginal(ctx, id, r.Project, r.Bridge)
	}
	proof, e := read(ctx)
	if e != nil {
		return zero, e
	}
	// Bind the durable boot preimage to the freshly compiled native plan
	// before the whole-original executor is permitted to create a journal.
	base, e := nativePeerScopedRootReadSnapshot(ctx, nativePeerScopedIncusReader{root: c}, proof)
	if e != nil {
		return zero, e
	}
	plan, e := NativePeerScopedRootTransitionFromEvidence(proof, base.Network, base.Evidence)
	if e != nil || plan.GuardBefore != current {
		return zero, errors.New("current published guard differs from staged original")
	}
	rec, e := nativeScopedPeerExecute(ctx, nativePeerFreshJournalDir, 0, r.HostSHA256, read, nativePeerScopedIncusReader{root: c}, nativePeerScopedIncusACLDriver{root: c}, "/sys/class/net", NativePeerGuardKernelReadback, nativePeerGuardFixedNftBatch, nativePeerConntrackOS,
		func(ctx context.Context, p state.ScopedPeerOriginalProof, s nativePeerScopedSDKSnapshot) (string, error) {
			if !reflect.DeepEqual(proof, p) {
				return "", errors.New("Root original changed")
			}
			return nativeScopedPeerGlobal(ctx, c, db, r, p, s)
		})
	if e != nil {
		return zero, e
	}
	again, e := read(ctx)
	if e != nil || !reflect.DeepEqual(again, proof) {
		return zero, errors.New("original changed before policy publication")
	}
	if e = scopedPeerBootPublish(ctx, nativePeerFreshJournalDir, 0, r, initial, proof, rec, NativePeerGuardKernelReadback); e != nil {
		return zero, e
	}
	return rec, nil
}
