package install

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestPR178GuardJournaledNativeDirectionNoReplay(t *testing.T) {
	before, after := pr178GuardTransitionPlans(t)
	scope := pr178GuardJournalScope(t)
	uid := uint32(os.Geteuid())
	dir := pr178GuardJournalDir(t)
	current := before
	writes := 0
	witness := func(_ context.Context, p NativePeerGuardPlan) error {
		if current.DigestSHA256 != p.DigestSHA256 {
			return errors.New("actual kernel image mismatch")
		}
		return nil
	}
	writer := func(_ context.Context, script string) error {
		if script != after.Replace {
			return errors.New("wrong native nft policy")
		}
		writes++
		current = after
		return nil
	}
	r, e := nativePeerGuardJournaledReplaceOnce(context.Background(), dir, uid, scope, before, after, witness, writer, nil)
	if e != nil || !r.GuardStepVerified || writes != 1 {
		t.Fatalf("root guard first original failed to complete: %+v %v", r, e)
	}
	j, e := nativePeerGuardJournalLoadAt(dir, uid, scope.OriginalID)
	if e != nil || j.Status != "GUARD_STEP_KERNEL_VERIFIED" {
		t.Fatal("guard original not durably committed", e)
	}
	_, e = nativePeerGuardJournaledReplaceOnce(context.Background(), dir, uid, scope, before, after, witness, writer, nil)
	if e == nil || writes != 1 {
		t.Fatal("old completed original replayed")
	}
	forged := scope
	forged.OriginalID = strings.Repeat("b", 32)
	forged.AfterSHA256 = strings.Repeat("f", 64)
	_, e = nativePeerGuardJournaledReplaceOnce(context.Background(), pr178GuardJournalDir(t), uid, forged, before, after,
		witness, writer, nil)
	if e == nil || writes != 1 {
		t.Fatal("forged after-policy permitted foreign write")
	}
}
func TestPR178GuardJournaledNativeEffectAppliedButACKLost(t *testing.T) {
	before, after := pr178GuardTransitionPlans(t)
	scope := pr178GuardJournalScope(t)
	uid := uint32(os.Geteuid())
	dir := pr178GuardJournalDir(t)
	actual := before
	writes := 0
	witness := func(_ context.Context, p NativePeerGuardPlan) error {
		if actual.DigestSHA256 != p.DigestSHA256 {
			return errors.New("kernel image mismatch")
		}
		return nil
	}
	writer := func(_ context.Context, _ string) error {
		writes++
		actual = after
		return errors.New("kernel committed but ACK lost")
	}
	receipt, e := nativePeerGuardJournaledReplaceOnce(context.Background(), dir, uid, scope, before, after, witness, writer, nil)
	if e == nil || receipt.Status != "DISPATCH_UNKNOWN" || !receipt.NativeAttempted ||
		receipt.GuardStepVerified || writes != 1 {
		t.Fatalf("applied-but-ACK-lost falsely completed: %+v %v", receipt, e)
	}
	if err := witness(context.Background(), after); err != nil {
		t.Fatal("kernel after state not independently observable", err)
	}
	journal, err := nativePeerGuardJournalLoadAt(dir, uid, scope.OriginalID)
	if err != nil || journal.Status != "DISPATCH_UNKNOWN" {
		t.Fatal("lost ACK not durably UNKNOWN", err)
	}
	_, e = nativePeerGuardJournaledReplaceOnce(context.Background(), dir, uid, scope, before, after, witness, writer, nil)
	if e == nil || writes != 1 {
		t.Fatal("ambiguous native effect replayed")
	}
}
