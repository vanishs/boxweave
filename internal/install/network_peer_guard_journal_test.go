package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func pr178GuardJournalScope(t *testing.T) nativePeerGuardJournalScope {
	t.Helper()
	cd := pr178Grant(pr178C, pr178D, "10.55.8.13", "10.55.8.14", "fd55:8::13", "fd55:8::14")
	ab := pr178Grant(pr178A, pr178B, "10.55.8.11", "10.55.8.12", "fd55:8::11", "fd55:8::12")
	before := pr178Guard(t, pr178GuardMembers(), []state.PeerGrantBinding{cd}, nil)
	after := pr178Guard(t, pr178GuardMembers(), []state.PeerGrantBinding{ab, cd}, nil)
	return nativePeerGuardJournalScope{
		OriginalID: strings.Repeat("a", 32), Project: "user-1001", Bridge: "incusbr-1001",
		ACL: "bw-pr178", SourceUUID: pr178A, TargetUUID: pr178B,
		LedgerSHA256: strings.Repeat("1", 64), NativeSHA256: strings.Repeat("2", 64),
		BeforeSHA256: before.DigestSHA256, AfterSHA256: after.DigestSHA256,
	}
}
func pr178GuardJournalDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "root-owned-originals")
}
func pr178GuardOutcome(s nativePeerGuardJournalScope) nativePeerGuardReplaceOutcome {
	return nativePeerGuardReplaceOutcome{
		Attempted: true, KernelVerified: true, BeforePlanSHA: s.BeforeSHA256, AfterPlanSHA: s.AfterSHA256,
	}
}
func TestPR178GuardJournalFsyncIntentBeforeEffectAndNeverReplays(t *testing.T) {
	s := pr178GuardJournalScope(t)
	dir := pr178GuardJournalDir(t)
	uid := uint32(os.Geteuid())
	called := 0
	effect := func(ctx context.Context) (nativePeerGuardReplaceOutcome, error) {
		called++
		j, err := nativePeerGuardJournalLoadAt(dir, uid, s.OriginalID)
		if err != nil || j.Status != "DISPATCH_UNKNOWN" {
			t.Fatalf("kernel mutation ran without durable persisted root intent: %+v %v", j, err)
		}
		return pr178GuardOutcome(s), nil
	}
	receipt, err := nativePeerGuardJournalOnce(context.Background(), dir, uid, s, effect, nil)
	if err != nil || receipt.Status != "GUARD_STEP_KERNEL_VERIFIED" ||
		!receipt.GuardStepVerified || !receipt.NativeAttempted || called != 1 {
		t.Fatalf("verified stage not durably recorded: %+v %v", receipt, err)
	}
	record, err := nativePeerGuardJournalLoadAt(dir, uid, s.OriginalID)
	if err != nil || record.Status != "GUARD_STEP_KERNEL_VERIFIED" ||
		record.Scope != s || !record.Outcome.KernelVerified {
		t.Fatal("durable root guard stage not independently read back", err)
	}
	receipt, err = nativePeerGuardJournalOnce(context.Background(), dir, uid, s, effect, nil)
	if err == nil || called != 1 || receipt.GuardStepVerified {
		t.Fatalf("completed original was blindly re-dispatched: %+v %v", receipt, err)
	}
}
func TestPR178GuardJournalUncertainEffectAndLostACKPreserveOriginal(t *testing.T) {
	cases := []struct {
		name         string
		run          func(nativePeerGuardJournalScope) (nativePeerGuardReplaceOutcome, error)
		beforeCommit func() error
	}{
		{"applied-native-batch-lost-ACK", func(s nativePeerGuardJournalScope) (nativePeerGuardReplaceOutcome, error) {
			return nativePeerGuardReplaceOutcome{
				Attempted: true, BeforePlanSHA: s.BeforeSHA256, AfterPlanSHA: s.AfterSHA256,
			}, errors.New("kernel applied but ACK lost")
		}, nil},
		{"post-kernel-witness-unknown", func(s nativePeerGuardJournalScope) (nativePeerGuardReplaceOutcome, error) {
			return nativePeerGuardReplaceOutcome{
				Attempted: true, BeforePlanSHA: s.BeforeSHA256, AfterPlanSHA: s.AfterSHA256,
			}, errors.New("strict nft JSON readback unavailable")
		}, nil},
		{"incomplete-success-receipt", func(s nativePeerGuardJournalScope) (nativePeerGuardReplaceOutcome, error) {
			return nativePeerGuardReplaceOutcome{
				Attempted: true, BeforePlanSHA: s.BeforeSHA256, AfterPlanSHA: s.AfterSHA256,
			}, nil
		}, nil},
		{"wrong-preimage-SHA", func(s nativePeerGuardJournalScope) (nativePeerGuardReplaceOutcome, error) {
			x := pr178GuardOutcome(s)
			x.BeforePlanSHA = strings.Repeat("9", 64)
			return x, nil
		}, nil},
		{"durable-completion-fault", func(s nativePeerGuardJournalScope) (nativePeerGuardReplaceOutcome, error) {
			return pr178GuardOutcome(s), nil
		}, func() error { return errors.New("injected fsync/rename refused") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := pr178GuardJournalScope(t)
			dir := pr178GuardJournalDir(t)
			uid := uint32(os.Geteuid())
			writes := 0
			receipt, err := nativePeerGuardJournalOnce(context.Background(), dir, uid, s,
				func(ctx context.Context) (nativePeerGuardReplaceOutcome, error) {
					writes++
					return tc.run(s)
				}, tc.beforeCommit)
			if err == nil || receipt.Status != "DISPATCH_UNKNOWN" ||
				receipt.GuardStepVerified || writes != 1 {
				t.Fatalf("failed native step claimed successful completion: %+v %v writes=%d", receipt, err, writes)
			}
			recorded, e := nativePeerGuardJournalLoadAt(dir, uid, s.OriginalID)
			if e != nil || recorded.Status != "DISPATCH_UNKNOWN" || recorded.Outcome.Attempted {
				t.Fatalf("UNKNOWN original not retained safely: %+v %v", recorded, e)
			}
			_, retryErr := nativePeerGuardJournalOnce(context.Background(), dir, uid, s,
				func(context.Context) (nativePeerGuardReplaceOutcome, error) {
					writes++
					return pr178GuardOutcome(s), nil
				}, nil)
			if retryErr == nil || writes != 1 {
				t.Fatal("crash/reconnect replayed ambiguous kernel write", retryErr)
			}
		})
	}
}
func TestPR178GuardJournalRejectsUnsafeOriginalScopeAndFilesystem(t *testing.T) {
	s := pr178GuardJournalScope(t)
	cases := []struct {
		name   string
		change func(*nativePeerGuardJournalScope)
	}{
		{"nonhex-original", func(v *nativePeerGuardJournalScope) { v.OriginalID = "../../root" }},
		{"foreign-project", func(v *nativePeerGuardJournalScope) { v.Project = "default" }},
		{"foreign-bridge", func(v *nativePeerGuardJournalScope) { v.Bridge = "../incusbr0" }},
		{"foreign-acl", func(v *nativePeerGuardJournalScope) { v.ACL = "incus-acl" }},
		{"same-box", func(v *nativePeerGuardJournalScope) { v.TargetUUID = v.SourceUUID }},
		{"invalid-uuid", func(v *nativePeerGuardJournalScope) { v.SourceUUID = "forged" }},
		{"empty-ledger-sha", func(v *nativePeerGuardJournalScope) { v.LedgerSHA256 = "" }},
		{"upper-hex-native-sha", func(v *nativePeerGuardJournalScope) { v.NativeSHA256 = strings.Repeat("A", 64) }},
		{"same-policy-digest", func(v *nativePeerGuardJournalScope) { v.AfterSHA256 = v.BeforeSHA256 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bad := s
			tc.change(&bad)
			dir := pr178GuardJournalDir(t)
			writes := 0
			_, err := nativePeerGuardJournalOnce(context.Background(), dir, uint32(os.Geteuid()), bad,
				func(context.Context) (nativePeerGuardReplaceOutcome, error) {
					writes++
					return pr178GuardOutcome(bad), nil
				}, nil)
			if err == nil || writes != 0 {
				t.Fatal("unsafe original selected root write", err)
			}
		})
	}
	t.Run("symlink-journal", func(t *testing.T) {
		dir := pr178GuardJournalDir(t)
		uid := uint32(os.Geteuid())
		if err := nativePeerGuardJournalPrepareAt(dir, uid); err != nil {
			t.Fatal(err)
		}
		file, err := nativePeerGuardJournalPathAt(dir, s.OriginalID)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(t.TempDir(), "foreign"), file); err != nil {
			t.Fatal(err)
		}
		writes := 0
		_, err = nativePeerGuardJournalOnce(context.Background(), dir, uid, s,
			func(context.Context) (nativePeerGuardReplaceOutcome, error) {
				writes++
				return pr178GuardOutcome(s), nil
			}, nil)
		if err == nil || writes != 0 {
			t.Fatal("untrusted symlink journal was overwritten")
		}
	})
	t.Run("world-writable-journal-directory", func(t *testing.T) {
		dir := pr178GuardJournalDir(t)
		if err := os.Mkdir(dir, 0777); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dir, 0777); err != nil {
			t.Fatal(err)
		}
		writes := 0
		_, err := nativePeerGuardJournalOnce(context.Background(), dir, uint32(os.Geteuid()), s,
			func(context.Context) (nativePeerGuardReplaceOutcome, error) {
				writes++
				return pr178GuardOutcome(s), nil
			}, nil)
		if err == nil || writes != 0 {
			t.Fatal("root intent saved under writable foreign directory")
		}
	})
	t.Run("wrong-dir-owner", func(t *testing.T) {
		dir := pr178GuardJournalDir(t)
		_, err := nativePeerGuardJournalOnce(context.Background(), dir, uint32(os.Geteuid())+1, s,
			func(context.Context) (nativePeerGuardReplaceOutcome, error) { return pr178GuardOutcome(s), nil }, nil)
		if err == nil {
			t.Fatal("incorrect Root journal owner accepted")
		}
	})
	t.Run("truncated-existing-original", func(t *testing.T) {
		dir := pr178GuardJournalDir(t)
		uid := uint32(os.Geteuid())
		if err := nativePeerGuardJournalPrepareAt(dir, uid); err != nil {
			t.Fatal(err)
		}
		path, err := nativePeerGuardJournalPathAt(dir, s.OriginalID)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := nativePeerGuardJournalLoadAt(dir, uid, s.OriginalID); err == nil {
			t.Fatal("corrupted journal was treated as a recoverable completed root intent")
		}
		called := false
		_, err = nativePeerGuardJournalOnce(context.Background(), dir, uid, s,
			func(context.Context) (nativePeerGuardReplaceOutcome, error) {
				called = true
				return pr178GuardOutcome(s), nil
			}, nil)
		if err == nil || called {
			t.Fatal("truncated existing intent permitted replay")
		}
	})
	t.Run("cancelled-before-journal", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		dir := pr178GuardJournalDir(t)
		called := false
		_, err := nativePeerGuardJournalOnce(ctx, dir, uint32(os.Geteuid()), s,
			func(context.Context) (nativePeerGuardReplaceOutcome, error) {
				called = true
				return pr178GuardOutcome(s), nil
			}, nil)
		if !errors.Is(err, context.Canceled) || called {
			t.Fatal("cancelled context admitted native write")
		}
		if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("cancelled intent wrote durable root scope")
		}
	})
}
