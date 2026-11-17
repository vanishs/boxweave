package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

type s4B2JournalWriteFaultServer struct {
	*fakeScriptletRoot
	journal string
	reads   int
}

func (s *s4B2JournalWriteFaultServer) GetServer() (*api.Server, string, error) {
	server, etag, err := s.fakeScriptletRoot.GetServer()
	s.reads++
	if s.reads == 1 && err == nil {
		if e := os.Chmod(s.journal, 0644); e != nil {
			return nil, "", e
		}
	}
	return server, etag, err
}

func TestS4B2ReconcilePersistenceFailurePreservesConfirmedJournalStatus(t *testing.T) {
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
		expected := "DISPATCH_UNKNOWN"
		if rollback {
			expected = "ROLLBACK_UNKNOWN"
			_, err = nativeScriptletRollbackWithClient(t.Context(), f, path, plan.ReviewSHA256, true)
		} else {
			_, err = nativeScriptletApplyWithClient(t.Context(), f, target, "host", path, plan.ReviewSHA256, true)
		}
		if err == nil {
			t.Fatal("missing native ACK fault")
		}
		count := f.updateCount
		wrapped := &s4B2JournalWriteFaultServer{fakeScriptletRoot: f, journal: path}
		result, err := nativeScriptletReconcileWithClient(t.Context(), wrapped, "host", path, plan.ReviewSHA256, true)
		if err == nil || result.Status != expected || result.ReadyForPeer || f.updateCount != count {
			t.Fatal("failed journal durability was reported as settled", rollback, result, err)
		}
		if err := os.Chmod(path, 0600); err != nil {
			t.Fatal(err)
		}
		stored, err := nativeScriptletLoadJournal(path)
		if err != nil || stored.Status != expected {
			t.Fatal("durable UNKNOWN must remain authoritative", stored.Status, err)
		}
	}
}
