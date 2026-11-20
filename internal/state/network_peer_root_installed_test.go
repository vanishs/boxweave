package state

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestS4B2InstalledRootControllerExactOperationVerifiedWithoutCallbacks(t *testing.T) {
	s, _, id := s4b2RootOperationFixture(t)
	s4b2PendingExistingGrantToConfirmed(t, s)
	if err := s.StageRootPeerRevocation(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	calls := 0
	invoke := func(_ context.Context, original, sha string) (RootPeerResult, error) {
		calls++
		if original != id {
			t.Fatal("foreign original reached installed root")
		}
		if sha == "" {
			return RootPeerResult{Status: "REVIEW_ONLY", PlanSHA256: strings.Repeat("a", 64)}, nil
		}
		if sha != strings.Repeat("a", 64) {
			t.Fatal("unreviewed root digest")
		}
		return RootPeerResult{Status: "COMPLETE_VERIFIED", PlanSHA256: sha, ProbedScopes: 4, DeletedScopes: 2,
			MutationComplete: true, KernelVerified: true, VerifiedNICs: 3,
			PostNativeDigestSHA256: strings.Repeat("b", 64)}, nil
	}
	w := Worker{Store: s}
	if err := w.ExecutePreparedRootPeerRevocation(id, invoke); err != nil {
		t.Fatal(err)
	}
	result, err := s.Get(context.Background(), id)
	if err != nil || result.State != "SUCCEEDED" || calls != 2 {
		t.Fatalf("integrated worker did not close original: %+v %v calls=%d", result, err, calls)
	}
	if err := w.ExecutePreparedRootPeerRevocation(id, invoke); err == nil || calls != 2 {
		t.Fatal("successful installed root operation replayed")
	}
	t.Log("S4_B2_INSTALLED_ROOT_CONTROLLER_TYPED_ORIGINAL_ATOMIC_FINALIZE=PASS")
}

func TestS4B2InstalledRootControllerRejectsUnverifiedKernelAndNeverReplays(t *testing.T) {
	for _, failure := range []string{"lost-ack", "kernel-unknown", "incomplete-digest"} {
		t.Run(failure, func(t *testing.T) {
			s, _, id := s4b2RootOperationFixture(t)
			s4b2PendingExistingGrantToConfirmed(t, s)
			if err := s.StageRootPeerRevocation(context.Background(), id); err != nil {
				t.Fatal(err)
			}
			if err := s.Claim(context.Background(), id); err != nil {
				t.Fatal(err)
			}
			calls := 0
			fn := func(_ context.Context, original, sha string) (RootPeerResult, error) {
				calls++
				if sha == "" {
					return RootPeerResult{Status: "REVIEW_ONLY", PlanSHA256: strings.Repeat("a", 64)}, nil
				}
				res := RootPeerResult{Status: "COMPLETE_VERIFIED", PlanSHA256: sha, ProbedScopes: 4, DeletedScopes: 2,
					MutationComplete: true, KernelVerified: true, VerifiedNICs: 3,
					PostNativeDigestSHA256: strings.Repeat("b", 64)}
				if failure == "lost-ack" {
					return RootPeerResult{}, errors.New("root response lost")
				}
				if failure == "kernel-unknown" {
					res.KernelVerified = false
				}
				if failure == "incomplete-digest" {
					res.PostNativeDigestSHA256 = ""
				}
				return res, nil
			}
			w := Worker{Store: s}
			if err := w.ExecutePreparedRootPeerRevocation(id, fn); err == nil {
				t.Fatal("unsafe root evidence committed")
			}
			rec, err := s.Get(context.Background(), id)
			if err != nil || rec.State != "UNKNOWN" {
				t.Fatalf("unverified root result not fenced: %+v %v", rec, err)
			}
			before := calls
			if err := w.ExecutePreparedRootPeerRevocation(id, fn); err == nil || calls != before {
				t.Fatal("ambiguous installed root original was replayed")
			}
		})
	}
	t.Log("S4_B2_INSTALLED_ROOT_CONTROLLER_LOST_ACK_KERNEL_FAIL_CLOSED=PASS")
}
