package install

import (
	"context"
	"errors"
	"io/fs"
)

// Fixed sequence for a future trusted fresh installer under its setup lock.
// All capabilities are internal; there is no production dispatch or API here.
// Unknown outcomes keep the durable directory and MUST NOT be replayed.
func nativePeerFreshBootstrapOnce(ctx context.Context, dir string, uid uint32, hostSHA string, upgrade bool,
	lstat func(string) (fs.FileInfo, error),
	inspect func(context.Context) (nativePeerFreshEvidence, error),
	absent func(context.Context, NativePeerGuardPlan) error,
	install func(context.Context, NativePeerGuardPlan) error,
	witness func(context.Context, NativePeerGuardPlan) error,
) (nativePeerFreshJournalRecord, error) {
	var zero nativePeerFreshJournalRecord
	if !nativePeerGuardSHA256Valid(hostSHA) || absent == nil || install == nil || witness == nil {
		return zero, errors.New("missing trusted fresh bootstrap capability")
	}
	before, err := nativePeerFreshAdmission(ctx, upgrade, lstat, inspect)
	if err != nil {
		return zero, err
	}

	if err = absent(ctx, before.Guard); err != nil {
		return zero, err
	}
	if err = ctx.Err(); err != nil {
		return zero, err
	}
	d, err := nativePeerFreshJournalCreateAt(dir, uid)
	if err != nil {
		return zero, err
	}
	defer d.Close()
	intent := nativePeerFreshJournalRecord{Schema: 1, Mode: nativePeerFreshMode, HostSHA256: hostSHA,
		Project: before.Project, Bridge: before.Bridge, NativeSHA256: before.NativeSHA256, GuardSHA256: before.Guard.DigestSHA256, Status: "DISPATCH_UNKNOWN"}
	if err = nativePeerFreshJournalWrite(d, intent); err != nil {
		return zero, err
	}
	again, err := inspect(ctx)
	if err != nil || again != before {
		return intent, errors.New("fresh native preimage changed; preserve UNKNOWN")
	}
	if err = absent(ctx, before.Guard); err != nil {
		return intent, err
	}
	if err = ctx.Err(); err != nil {
		return intent, err
	}
	if err = nativePeerFreshPathsAbsentExceptJournal(lstat, true); err != nil {
		return intent, err
	}
	if err = install(ctx, before.Guard); err != nil {
		return intent, err
	}
	if err = witness(ctx, before.Guard); err != nil {
		return intent, err
	}
	after, err := inspect(ctx)
	if err != nil || after != before {
		return intent, errors.New("fresh native postimage changed; preserve UNKNOWN")
	}
	if err = witness(ctx, before.Guard); err != nil {
		return intent, err
	}
	if err = ctx.Err(); err != nil {
		return intent, err
	}
	if err = nativePeerFreshPathsAbsentExceptJournal(lstat, true); err != nil {
		return intent, err
	}
	verified := intent
	verified.Status = "INITIAL_GUARD_VERIFIED"
	if err = nativePeerFreshJournalWrite(d, verified); err != nil {
		return intent, err
	}
	return verified, nil
}
