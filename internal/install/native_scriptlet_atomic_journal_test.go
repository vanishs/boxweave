package install

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func s4b2ScriptletIntentFixture() nativeScriptletJournal {
	return nativeScriptletJournal{
		Schema: 1, Status: "DISPATCH_UNKNOWN",
		Plan:          NativeScriptletPlan{ReviewSHA256: strings.Repeat("a", 64), SourceSHA256: strings.Repeat("b", 64)},
		BeforeConfig:  api.ConfigMap{"authorization.client.tls-restricted": "tls"},
		AppliedConfig: api.ConfigMap{"authorization.client.tls-restricted": "scriptlet", "authorization.scriptlet": "def authorize(*args): return False"},
	}
}
func TestS4B2ScriptletJournalFirstIntentExclusiveAndNoStateRewind(t *testing.T) {
	j := s4b2ScriptletIntentFixture()
	path := filepath.Join(t.TempDir(), "s4b2-root-intent.json")
	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := nativeScriptletSaveJournal(path, j); err == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("concurrent global Scriptlet first intent dispatched %d times", wins.Load())
	}
	actual, err := nativeScriptletLoadJournal(path)
	if err != nil || actual.Status != "DISPATCH_UNKNOWN" {
		t.Fatal("original journal damaged", err)
	}
	st, err := os.Stat(path)
	if err != nil || st.Mode().Perm() != 0600 {
		t.Fatal("native root journal permissions missing", err)
	}
	j.Status = "ROUTE_APPLIED_PARITY_PENDING"
	if err := nativeScriptletSaveJournal(path, j); err != nil {
		t.Fatal("valid native successful ACK transition", err)
	}
	j.Status = "DISPATCH_UNKNOWN"
	if err := nativeScriptletSaveJournal(path, j); err == nil {
		t.Fatal("original root request rearmed")
	}
	j.Status = "ROLLBACK_UNKNOWN"
	if err := nativeScriptletSaveJournal(path, j); err != nil {
		t.Fatal("rollback dispatch intent not recorded", err)
	}
	if err := nativeScriptletSaveJournal(path, j); err == nil {
		t.Fatal("uncertain native rollback was re-dispatched")
	}
	j.Status = "ROLLED_BACK"
	if err := nativeScriptletSaveJournal(path, j); err != nil {
		t.Fatal("verified final rollback not recorded", err)
	}
	j.Status = "ROUTE_APPLIED_PARITY_PENDING"
	if err := nativeScriptletSaveJournal(path, j); err == nil {
		t.Fatal("completed rollback journal illegally rearmed")
	}
}

func TestS4B2ScriptletRootLockRefusesConcurrentAndSymlink(t *testing.T) {
	lock := filepath.Join(t.TempDir(), "scriptlet-root.lock")
	done, err := nativeScriptletAcquireLock(lock)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := nativeScriptletAcquireLock(lock); err == nil {
		again()
		t.Fatal("another root authorization installer concurrently acquired exclusive lock")
	}
	done()
	done, err = nativeScriptletAcquireLock(lock)
	if err != nil {
		t.Fatal("release did not unblock subsequent host maintenance", err)
	}
	done()
	link := filepath.Join(t.TempDir(), "forged-lock")
	if err := os.Symlink(lock, link); err != nil {
		t.Fatal(err)
	}
	if closeIt, err := nativeScriptletAcquireLock(link); err == nil {
		closeIt()
		t.Fatal("symlink root auth lock accepted")
	}
	unsafe := filepath.Join(t.TempDir(), "world-lock")
	if err := os.WriteFile(unsafe, []byte("open"), 0600); err != nil {
		t.Fatal(err)
	}
	// A restrictive process umask must not turn the negative fixture into a safe file.
	if err := os.Chmod(unsafe, 0644); err != nil {
		t.Fatal(err)
	}
	if closeIt, err := nativeScriptletAcquireLock(unsafe); err == nil {
		closeIt()
		t.Fatal("world-readable root lock accepted")
	}
}

func TestS4B2ScriptletRollbackLostAckNeverReplaysUnknown(t *testing.T) {
	target, _, _, _, _ := scriptletTestFixture()
	f := fakeScriptletFixture()
	plan, err := nativeScriptletInspectWithClient(t.Context(), f, target, "host")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "scriptlet-root-journal.json")
	if _, err := nativeScriptletApplyWithClient(t.Context(), f, target, "host", path, plan.ReviewSHA256, true); err != nil {
		t.Fatal(err)
	}
	f.failMode = "lost_ack_after_dispatch"
	first, err := nativeScriptletRollbackWithClient(t.Context(), f, path, plan.ReviewSHA256, true)
	if err == nil || first.Status != "ROLLBACK_UNKNOWN" || f.updateCount != 2 {
		t.Fatal("lost native rollback response misclassified", err, first, f.updateCount)
	}
	second, err := nativeScriptletRollbackWithClient(t.Context(), f, path, plan.ReviewSHA256, true)
	if err == nil || second.Status != "ROLLBACK_UNKNOWN" || f.updateCount != 2 {
		t.Fatal("unknown native rollback was resent", err, second, f.updateCount)
	}
}

func TestS4B2ScriptletInterruptedOriginalIntentNeverDispatches(t *testing.T) {
	target, _, _, _, _ := scriptletTestFixture()
	f := fakeScriptletFixture()
	plan, err := nativeScriptletInspectWithClient(t.Context(), f, target, "host")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "interrupted-intent.json")
	if err := os.WriteFile(path, []byte("{\"schema\":1,\"status\":"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := nativeScriptletApplyWithClient(t.Context(), f, target, "host", path, plan.ReviewSHA256, true); err == nil {
		t.Fatal("interrupted persistent journal was silently adopted")
	}
	if _, err := nativeScriptletRollbackWithClient(t.Context(), f, path, plan.ReviewSHA256, true); err == nil {
		t.Fatal("interrupted persistent journal was silently rolled back")
	}
	if f.updateCount != 0 {
		t.Fatal("native Scriptlet mutation dispatched after interrupted intent")
	}
}
