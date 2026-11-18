package install

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
)

func s4b2JournalPlan() NativePeerConntrackPlan {
	return NativePeerConntrackPlan{
		Project: "user-1001", Bridge: "bwbr0", ACL: "bw-peer",
		SourceUUID: "uuid-a", TargetUUID: "uuid-b",
		HostSHA256:         strings.Repeat("a", 64),
		LedgerSHA256:       strings.Repeat("b", 64),
		NativeDigestSHA256: strings.Repeat("c", 64),
		PlanSHA256:         strings.Repeat("d", 64),
	}
}

func TestS4B2RootConntrackDurableOneShotReceipt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "root-journal.json")
	plan := s4b2JournalPlan()
	calls := 0
	effect := func(context.Context) (NativePeerConntrackReport, error) {
		calls++
		if _, err := nativePeerConntrackLoadJournal(path); err != nil {
			t.Fatal("native effect before durable UNKNOWN:", err)
		}
		return NativePeerConntrackReport{ProbedScopes: 4, DeletedScopes: 2, ReadyForPeer: false, NativeDigestSHA256: plan.NativeDigestSHA256}, nil
	}
	receipt, err := nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true, effect)
	if err != nil || calls != 1 || receipt.Status != "COMPLETE_VERIFIED" || !receipt.MutationComplete ||
		receipt.ReadyForPeer || receipt.ProbedScopes != 4 || receipt.DeletedScopes != 2 {
		t.Fatalf("one-shot completed journal not durable: %+v %v calls=%d", receipt, err, calls)
	}
	j, err := nativePeerConntrackLoadJournal(path)
	if err != nil || j.Status != "COMPLETE_VERIFIED" || j.Plan != plan || j.Report.DeletedScopes != 2 {
		t.Fatal("verified result not persisted", j, err)
	}
	if _, err := nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true, effect); err == nil || calls != 1 {
		t.Fatal("second dispatch replayed same approved root effect", calls, err)
	}
	t.Log("S4_B2_ROOT_CONNTRACK_DURABLE_ONE_SHOT=PASS")
}

func TestS4B2RootConntrackUnknownNeverRetriesAfterNativeFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unknown.json")
	plan := s4b2JournalPlan()
	calls := 0
	effect := func(context.Context) (NativePeerConntrackReport, error) {
		calls++
		return NativePeerConntrackReport{DeletedScopes: 1}, errors.New("ACK LOST AFTER KERNEL DELETE")
	}
	r, err := nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true, effect)
	if err == nil || calls != 1 || r.Status != "DISPATCH_UNKNOWN" || r.MutationComplete {
		t.Fatal(r, err, calls)
	}
	j, err := nativePeerConntrackLoadJournal(path)
	if err != nil || j.Status != "DISPATCH_UNKNOWN" {
		t.Fatal(j, err)
	}
	if _, err := nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true, effect); err == nil || calls != 1 {
		t.Fatal("lost native ACK triggered replay", err, calls)
	}
}

func TestS4B2RootConntrackCrashLeavesOriginalUnknownAndBlocksReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crash.json")
	plan := s4b2JournalPlan()
	calls := 0
	func() {
		defer func() { _ = recover() }()
		_, _ = nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true,
			func(context.Context) (NativePeerConntrackReport, error) {
				calls++
				j, err := nativePeerConntrackLoadJournal(path)
				if err != nil || j.Status != "DISPATCH_UNKNOWN" {
					t.Fatal("crash preceded fsynced intent", j, err)
				}
				panic("simulated abrupt root worker death")
			})
	}()
	if calls != 1 {
		t.Fatal("crash fixture never dispatched")
	}
	j, err := nativePeerConntrackLoadJournal(path)
	if err != nil || j.Status != "DISPATCH_UNKNOWN" {
		t.Fatal(j, err)
	}
	if _, err := nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true,
		func(context.Context) (NativePeerConntrackReport, error) {
			calls++
			return NativePeerConntrackReport{}, nil
		}); err == nil || calls != 1 {
		t.Fatal("restarted root worker replayed original unknown", err, calls)
	}
	t.Log("S4_B2_ROOT_CONNTRACK_CRASH_ORIGINAL_UNKNOWN_NO_REPLAY=PASS")
}

func TestS4B2RootConntrackCannotDispatchWithoutExactApproval(t *testing.T) {
	plan := s4b2JournalPlan()
	for _, tc := range []struct {
		name, sha string
		approved  bool
	}{
		{"not-approved", plan.PlanSHA256, false},
		{"wrong-plan", strings.Repeat("e", 64), true},
		{"empty-plan", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "journal.json")
			calls := 0
			_, err := nativePeerConntrackDispatchOnce(t.Context(), path, plan, tc.sha, tc.approved,
				func(context.Context) (NativePeerConntrackReport, error) {
					calls++
					return NativePeerConntrackReport{}, nil
				})
			if err == nil || calls != 0 {
				t.Fatal("unsafe unapproved dispatch", calls, err)
			}
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				t.Fatal("unapproved request created root journal", err)
			}
		})
	}
}

func TestS4B2RootConntrackFailedFinalWriteCannotReportSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.json")
	plan := s4b2JournalPlan()
	calls := 0
	receipt, err := nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true,
		func(context.Context) (NativePeerConntrackReport, error) {
			calls++
			if err := os.Chmod(path, 0644); err != nil {
				t.Fatal(err)
			}
			return NativePeerConntrackReport{ProbedScopes: 4, DeletedScopes: 4}, nil
		})
	if err == nil || receipt.Status != "DISPATCH_UNKNOWN" || receipt.MutationComplete || calls != 1 {
		t.Fatal("failed final journal persistence falsely accepted", receipt, err)
	}
	if _, err := nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true,
		func(context.Context) (NativePeerConntrackReport, error) {
			calls++
			return NativePeerConntrackReport{}, nil
		}); err == nil || calls != 1 {
		t.Fatal("failed completion allowed native retry", err, calls)
	}
}

func TestS4B2RootConntrackConcurrentApprovalOnlyOneOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.json")
	plan := s4b2JournalPlan()
	var calls atomic.Int32
	const n = 6
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = nativePeerConntrackDispatchOnce(context.Background(), path, plan, plan.PlanSHA256, true,
				func(context.Context) (NativePeerConntrackReport, error) {
					calls.Add(1)
					return NativePeerConntrackReport{ProbedScopes: 4, NativeDigestSHA256: plan.NativeDigestSHA256}, nil
				})
		}()
	}
	wg.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatalf("concurrent root replay dispatched %d times", got)
	}
	j, err := nativePeerConntrackLoadJournal(path)
	if err != nil || j.Status != "COMPLETE_VERIFIED" {
		t.Fatal(j, err)
	}
}

func TestS4B2RootConntrackPartialVerificationPreservesUnknown(t *testing.T) {
	plan := s4b2JournalPlan()
	checks := []struct {
		name   string
		report NativePeerConntrackReport
	}{
		{"three-directions", NativePeerConntrackReport{ProbedScopes: 3, NativeDigestSHA256: plan.NativeDigestSHA256}},
		{"too-many-directions", NativePeerConntrackReport{ProbedScopes: 5, NativeDigestSHA256: plan.NativeDigestSHA256}},
		{"native-digest-drift", NativePeerConntrackReport{ProbedScopes: 4, NativeDigestSHA256: strings.Repeat("e", 64)}},
		{"invalid-deleted-count", NativePeerConntrackReport{ProbedScopes: 4, DeletedScopes: 5, NativeDigestSHA256: plan.NativeDigestSHA256}},
		{"false-ready", NativePeerConntrackReport{ProbedScopes: 4, NativeDigestSHA256: plan.NativeDigestSHA256, ReadyForPeer: true}},
	}
	for _, test := range checks {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "journal.json")
			calls := 0
			receipt, err := nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true,
				func(context.Context) (NativePeerConntrackReport, error) { calls++; return test.report, nil })
			if err == nil || calls != 1 || receipt.Status != "DISPATCH_UNKNOWN" || receipt.MutationComplete || receipt.ReadyForPeer {
				t.Fatal("partial root result falsely verified", receipt, err)
			}
			j, err := nativePeerConntrackLoadJournal(path)
			if err != nil || j.Status != "DISPATCH_UNKNOWN" {
				t.Fatal(j, err)
			}
		})
	}
	t.Log("S4_B2_ROOT_CONNTRACK_PARTIAL_SCOPE_OR_DIGEST_REFUSED=PASS")
}

// TestS4B2RootConntrackActualSIGKILLAfterPersistedIntent uses a REAL
// abrupt OS process kill (not a panic or timeout) after the intent has
// been fsynced. This tests the durable no-replay fence in a new process,
// while deliberately avoiding touching any host-native Incus/conntrack.
func TestS4B2RootConntrackActualSIGKILLAfterPersistedIntent(t *testing.T) {
	path := os.Getenv("BOXWEAVE_B2_KILL_INTENT_PATH")
	marker := os.Getenv("BOXWEAVE_B2_KILL_MARKER")
	if path != "" || marker != "" {
		if path == "" || marker == "" {
			t.Fatal("incomplete crash-subprocess fixture")
		}
		plan := s4b2JournalPlan()
		_, _ = nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true,
			func(context.Context) (NativePeerConntrackReport, error) {
				j, err := nativePeerConntrackLoadJournal(path)
				if err != nil || j.Status != "DISPATCH_UNKNOWN" {
					t.Fatal("native effect preceded durable UNKNOWN", err)
				}
				if err := os.WriteFile(marker, []byte("ROOT_INTENT_FSYNCED"), 0600); err != nil {
					t.Fatal(err)
				}
				_ = syscall.Kill(os.Getpid(), syscall.SIGKILL)
				panic("SIGKILL unexpectedly returned")
			})
		t.Fatal("native root crash subprocess did not die")
	}
	dir := t.TempDir()
	path = filepath.Join(dir, "crash-original.json")
	marker = filepath.Join(dir, "durable-witness")
	command := exec.Command(os.Args[0], "-test.run=^TestS4B2RootConntrackActualSIGKILLAfterPersistedIntent$")
	command.Env = append(os.Environ(), "BOXWEAVE_B2_KILL_INTENT_PATH="+path, "BOXWEAVE_B2_KILL_MARKER="+marker)
	err := command.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("root process was not killed: %v", err)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
		t.Fatalf("root process did not die from the expected real SIGKILL: %v", status)
	}
	b, err := os.ReadFile(marker)
	if err != nil || string(b) != "ROOT_INTENT_FSYNCED" {
		t.Fatal("root intent was not persisted before SIGKILL", err)
	}
	journal, err := nativePeerConntrackLoadJournal(path)
	if err != nil || journal.Status != "DISPATCH_UNKNOWN" {
		t.Fatal("root journal after actual SIGKILL was not UNKNOWN", journal, err)
	}
	count := 0
	plan := s4b2JournalPlan()
	_, err = nativePeerConntrackDispatchOnce(t.Context(), path, plan, plan.PlanSHA256, true,
		func(context.Context) (NativePeerConntrackReport, error) {
			count++
			return NativePeerConntrackReport{}, nil
		})
	if err == nil || count != 0 {
		t.Fatal("crashed original was redispatched", err, count)
	}
	t.Log("S4_B2_ROOT_ACTUAL_SIGKILL_DURABLE_UNKNOWN_NO_REPLAY=PASS")
}
