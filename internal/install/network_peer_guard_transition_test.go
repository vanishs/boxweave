package install

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func pr178GuardTransitionPlans(t *testing.T) (NativePeerGuardPlan, NativePeerGuardPlan) {
	t.Helper()
	cd := pr178Grant(pr178C, pr178D, "10.55.8.13", "10.55.8.14", "fd55:8::13", "fd55:8::14")
	ab := pr178Grant(pr178A, pr178B, "10.55.8.11", "10.55.8.12", "fd55:8::11", "fd55:8::12")
	baseline := pr178Guard(t, pr178GuardMembers(), []state.PeerGrantBinding{cd}, nil)
	grant := pr178Guard(t, pr178GuardMembers(), []state.PeerGrantBinding{ab, cd}, nil)
	return baseline, grant
}
func TestPR178NftOneDirectionReplacePreservesUnrelatedCAndD(t *testing.T) {
	before, after := pr178GuardTransitionPlans(t)
	if err := nativePeerGuardValidateOneDirectionDelta(before, after); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerGuardValidateOneDirectionDelta(after, before); err != nil {
		t.Fatal(err)
	}
	current := before
	checked, writes := 0, 0
	witness := func(ctx context.Context, p NativePeerGuardPlan) error {
		checked++
		if current.ExpectedKernelJSON != p.ExpectedKernelJSON {
			return errors.New("missing kernel witness")
		}
		return nil
	}
	batch := func(ctx context.Context, script string) error {
		writes++
		if script != after.Replace || strings.Contains(script, "delete table") ||
			strings.Contains(script, "flush ruleset") {
			return errors.New("unsafe multi-bridge shell command")
		}
		current = after
		return nil
	}
	receipt, err := nativePeerGuardReplaceOnce(context.Background(), before, after, witness, batch)
	if err != nil || !receipt.Attempted || !receipt.KernelVerified ||
		checked != 2 || writes != 1 || receipt.BeforePlanSHA != before.DigestSHA256 ||
		receipt.AfterPlanSHA != after.DigestSHA256 {
		t.Fatalf("atomic one direction not witnessed: %+v %v checked=%d writes=%d", receipt, err, checked, writes)
	}
	receipt, err = nativePeerGuardReplaceOnce(context.Background(), after, before, witness,
		func(ctx context.Context, script string) error {
			if script != before.Replace {
				t.Fatal("revoke altered bounded target")
			}
			current = before
			writes++
			return nil
		})
	if err != nil || !receipt.KernelVerified || writes != 2 {
		t.Fatalf("revoke not bounded/verified: %+v %v", receipt, err)
	}
}
func TestPR178NftOneDirectionRejectsMultiPeerOrDefaultRuleMutation(t *testing.T) {
	before, after := pr178GuardTransitionPlans(t)
	scenarios := []struct {
		name     string
		bef, aft NativePeerGuardPlan
	}{
		{"no-change", before, before},
		{"cross-project-table", func() NativePeerGuardPlan {
			p := before
			p.Table = "bw_pg_deadbeefdeadbeef"
			p.DigestSHA256 = nativePeerGuardDigest(p)
			return p
		}(), after},
		{"forged-digest", before, func() NativePeerGuardPlan { p := after; p.DigestSHA256 = strings.Repeat("0", 64); return p }()},
		{"multiple-concurrent-grants", func() NativePeerGuardPlan {
			p := pr178Guard(t, pr178GuardMembers(), nil, nil)
			return p
		}(), after},
		{"change-other-grant-also", before, func() NativePeerGuardPlan {
			ab := pr178Grant(pr178A, pr178B, "10.55.8.11", "10.55.8.12", "fd55:8::11", "fd55:8::12")
			return pr178Guard(t, pr178GuardMembers(), []state.PeerGrantBinding{ab}, nil)
		}()},
		{"unexpected-terminal-deny", before, func() NativePeerGuardPlan {
			p := after
			var a [][]any
			if e := json.Unmarshal([]byte(p.ExpectedKernelJSON), &a); e != nil {
				t.Fatal(e)
			}
			last := a[len(a)-1]
			last[len(last)-1] = map[string]any{"accept": nil}
			b, e := json.Marshal(a)
			if e != nil {
				t.Fatal(e)
			}
			p.ExpectedKernelJSON = string(b)
			p.DigestSHA256 = nativePeerGuardDigest(p)
			return p
		}()},
		{"unbounded-native-batch", before, func() NativePeerGuardPlan {
			p := after
			p.Replace += "flush ruleset\n"
			p.DigestSHA256 = nativePeerGuardDigest(p)
			return p
		}()},
	}
	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r, err := nativePeerGuardReplaceOnce(context.Background(), tc.bef, tc.aft,
				func(context.Context, NativePeerGuardPlan) error { return nil },
				func(context.Context, string) error { calls++; return nil })
			if err == nil || calls != 0 || r.Attempted || r.KernelVerified {
				t.Fatalf("untrusted delta executed: %+v %v calls=%d", r, err, calls)
			}
		})
	}
}
func TestPR178NftOneDirectionUncertainACKAndKernelReadbackNeverRetry(t *testing.T) {
	before, after := pr178GuardTransitionPlans(t)
	test := func(t *testing.T, applyThenErr bool, postFault bool) {
		current := before
		writes := 0
		witness := func(_ context.Context, p NativePeerGuardPlan) error {
			if postFault && p.ExpectedKernelJSON == after.ExpectedKernelJSON {
				return errors.New("post-kernel nft JSON absent")
			}
			if current.ExpectedKernelJSON != p.ExpectedKernelJSON {
				return errors.New("preimage/postimage kernel mismatch")
			}
			return nil
		}
		batch := func(_ context.Context, _ string) error {
			writes++
			if applyThenErr {
				current = after
				return errors.New("simulated lost native ACK")
			}
			current = after
			return nil
		}
		r, err := nativePeerGuardReplaceOnce(context.Background(), before, after, witness, batch)
		if err == nil || !r.Attempted || r.KernelVerified || writes != 1 {
			t.Fatalf("uncertain kernel effect guessed as complete: %+v %v writes=%d", r, err, writes)
		}
	}
	t.Run("applied-but-ACK-lost", func(t *testing.T) { test(t, true, false) })
	t.Run("post-kernel-readback-missing", func(t *testing.T) { test(t, false, true) })

	t.Run("unverified-before-no-dispatch", func(t *testing.T) {
		writes := 0
		r, err := nativePeerGuardReplaceOnce(context.Background(), before, after,
			func(context.Context, NativePeerGuardPlan) error { return errors.New("foreign guard changed") },
			func(context.Context, string) error { writes++; return nil })
		if err == nil || writes != 0 || r.Attempted {
			t.Fatal("unsafe preimage wrote native nft")
		}
	})
	t.Run("cancelled-original-no-dispatch", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		writes := 0
		r, err := nativePeerGuardReplaceOnce(ctx, before, after,
			func(context.Context, NativePeerGuardPlan) error { return nil },
			func(context.Context, string) error { writes++; return nil })
		if !errors.Is(err, context.Canceled) || writes != 0 || r.Attempted {
			t.Fatal("cancelled original wrote kernel policy", err)
		}
	})
}
