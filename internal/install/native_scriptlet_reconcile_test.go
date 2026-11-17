package install

import (
	"path/filepath"
	"testing"
)

func TestS4B2ScriptletReconcileACKReadback(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		target, _, _, _, _ := scriptletTestFixture()
		f := fakeScriptletFixture()
		plan, err := nativeScriptletInspectWithClient(t.Context(), f, target, "host")
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "journal.json")
		if rollback {
			if _, err = nativeScriptletApplyWithClient(t.Context(), f, target, "host", path, plan.ReviewSHA256, true); err != nil {
				t.Fatal(err)
			}
		}
		f.failMode = "lost_ack_after_dispatch"
		if rollback {
			_, err = nativeScriptletRollbackWithClient(t.Context(), f, path, plan.ReviewSHA256, true)
		} else {
			_, err = nativeScriptletApplyWithClient(t.Context(), f, target, "host", path, plan.ReviewSHA256, true)
		}
		if err == nil {
			t.Fatal("expected ACK loss")
		}
		count := f.updateCount
		want := "ROUTE_APPLIED_PARITY_PENDING"
		if rollback {
			want = "ROLLED_BACK"
		}
		for i := 0; i < 2; i++ {
			got, err := nativeScriptletReconcileWithClient(t.Context(), f, "host", path, plan.ReviewSHA256, true)
			if err != nil || got.Status != want || got.ReadyForPeer || f.updateCount != count {
				t.Fatal("unsafe reconciliation", rollback, got, err)
			}
		}
	}
}
func TestS4B2ScriptletReconcilePreservesUnknown(t *testing.T) {
	for _, mode := range []string{"before_dispatch", "partial_after_dispatch"} {
		target, _, _, _, _ := scriptletTestFixture()
		f := fakeScriptletFixture()
		plan, err := nativeScriptletInspectWithClient(t.Context(), f, target, "host")
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "journal.json")
		f.failMode = mode
		if _, err = nativeScriptletApplyWithClient(t.Context(), f, target, "host", path, plan.ReviewSHA256, true); err == nil {
			t.Fatal("expected ambiguity")
		}
		for _, c := range []struct {
			host, sha string
			approve   bool
		}{{"host", plan.ReviewSHA256, false}, {"foreign-host", plan.ReviewSHA256, true}, {"host", "deadbeef", true}, {"host", plan.ReviewSHA256, true}} {
			got, err := nativeScriptletReconcileWithClient(t.Context(), f, c.host, path, c.sha, c.approve)
			if err == nil || got.ReadyForPeer {
				t.Fatal("uncertain result promoted", got)
			}
		}
		j, err := nativeScriptletLoadJournal(path)
		if err != nil || j.Status != "DISPATCH_UNKNOWN" || f.updateCount != 1 {
			t.Fatal("unknown lost or native replay", j.Status, err)
		}
	}
}
