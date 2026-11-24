package install

import (
	"context"
	"errors"
)

// Non-exported fixed composition. This is NOT wired to the current Root
// socket, controller, MCP or installer. Future root dispatch MUST first
// independently verify and lock the authenticated original, full SQLite and
// Incus scope, bridge ports, nft guard, original actor, and exact A/B pair.
func nativePeerGuardJournaledReplaceOnce(ctx context.Context, dir string, uid uint32,
	scope nativePeerGuardJournalScope, before, after NativePeerGuardPlan,
	witness nativePeerGuardWitness, write nativePeerGuardBatch,
	beforeCommit func() error) (nativePeerGuardJournalReceipt, error) {
	none := nativePeerGuardJournalReceipt{
		Status: "NOT_DISPATCHED", OriginalID: scope.OriginalID,
		BeforeSHA256: scope.BeforeSHA256, AfterSHA256: scope.AfterSHA256,
	}
	if scope.BeforeSHA256 != before.DigestSHA256 ||
		scope.AfterSHA256 != after.DigestSHA256 ||
		nativePeerGuardValidateOneDirectionDelta(before, after) != nil {
		return none, errors.New("durable guard journal cannot authorize unverified native policy delta")
	}
	if witness == nil || write == nil {
		return none, errors.New("missing independent kernel witness or fixed root batch executor")
	}
	return nativePeerGuardJournalOnce(ctx, dir, uid, scope,
		func(ctx context.Context) (nativePeerGuardReplaceOutcome, error) {
			return nativePeerGuardReplaceOnce(ctx, before, after, witness, write)
		}, beforeCommit)
}
