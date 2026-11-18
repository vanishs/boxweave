package install

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestS4B2RootPerOriginalConntrackJournalNoReplay(t *testing.T) {
	originalA := strings.Repeat("a", 32)
	originalB := strings.Repeat("b", 32)
	for _, bad := range []string{"", "A" + originalA[1:], originalA + "/", "../" + originalA} {
		if _, err := nativePeerConntrackOperationJournalPath(bad); err == nil {
			t.Fatalf("untrusted original accepted as root journal path: %q", bad)
		}
	}
	plan := s4b2JournalPlan()
	parent := t.TempDir()
	pathA := filepath.Join(parent, originalA+".json")
	pathB := filepath.Join(parent, originalB+".json")
	calls := 0
	effect := func(context.Context) (NativePeerConntrackReport, error) {
		calls++
		return NativePeerConntrackReport{
			ProbedScopes: 4, DeletedScopes: 2,
			NativeDigestSHA256: plan.NativeDigestSHA256,
		}, nil
	}
	r, err := nativePeerConntrackDispatchOnce(t.Context(), pathA, plan, plan.PlanSHA256, true, effect)
	if err != nil || r.Status != "COMPLETE_VERIFIED" || calls != 1 {
		t.Fatalf("root original A missing durable completion: %+v %v calls=%d", r, err, calls)
	}
	if _, err := nativePeerConntrackDispatchOnce(t.Context(), pathA, plan, plan.PlanSHA256, true, effect); err == nil || calls != 1 {
		t.Fatal("same immutable original replayed")
	}
	r, err = nativePeerConntrackDispatchOnce(t.Context(), pathB, plan, plan.PlanSHA256, true, effect)
	if err != nil || r.Status != "COMPLETE_VERIFIED" || calls != 2 {
		t.Fatal("different root original did not have separate journal", r, err, calls)
	}
	t.Log("S4_B2_ROOT_PER_ORIGINAL_JOURNAL_DISTINCT_AND_NONREPLAY=PASS")
}

func TestS4B2RootPerOriginalUncertainReceiptNeverReplay(t *testing.T) {
	plan := s4b2JournalPlan()
	path := filepath.Join(t.TempDir(), strings.Repeat("a", 32)+".json")
	n := 0
	broken := func(context.Context) (NativePeerConntrackReport, error) {
		n++
		return NativePeerConntrackReport{}, errors.New("native delete ACK lost")
	}
	r, err := nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true, broken)
	if err == nil || r.Status != "DISPATCH_UNKNOWN" || n != 1 {
		t.Fatal("lost ACK not fenced", r, err, n)
	}
	if _, err = nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true, broken); err == nil || n != 1 {
		t.Fatal("ambiguous original replayed after simulated root worker restart")
	}
	t.Log("S4_B2_ROOT_PER_ORIGINAL_UNKNOWN_FREEZES_DISPATCH=PASS")
}
