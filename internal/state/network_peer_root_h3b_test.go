package state

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestH3BPreparedRootDurableStageAndNoReplay(t *testing.T) {
	for _, lost := range []bool{false, true} {
		s, _, id := s4b2RootOperationFixture(t)
		s4b2PendingExistingGrantToConfirmed(t, s)
		ctx := context.Background()
		if e := s.StageRootPeerRevocation(ctx, id); e != nil {
			t.Fatal(e)
		}
		if e := s.Claim(ctx, id); e != nil {
			t.Fatal(e)
		}
		w := Worker{Store: s}
		calls := 0
		count := func(detail string) int {
			var n int
			if e := s.db.QueryRow("SELECT count(*) FROM operation_events WHERE operation_id=? AND safe_detail=?", id, detail).Scan(&n); e != nil {
				t.Fatal(e)
			}
			return n
		}
		e := w.ExecutePreparedRootPeerRevocation(id, func(_ context.Context, _ string, sha string) (RootPeerResult, error) {
			calls++
			if count("PEER_ROOT_H3B_REVIEW_STARTED") != 1 {
				t.Fatal("missing review reservation")
			}
			if sha == "" {
				// A competing local worker must not reach the Root socket.
				if e := w.ExecutePreparedRootPeerRevocation(id, func(context.Context, string, string) (RootPeerResult, error) {
					t.Fatal("duplicate dispatch")
					return RootPeerResult{}, nil
				}); e == nil {
					t.Fatal("duplicate review accepted")
				}
				if !lost {
					return RootPeerResult{}, errors.New("review transport failed")
				}
				return RootPeerResult{Status: "REVIEW_ONLY", PlanSHA256: strings.Repeat("a", 64)}, nil
			}
			if count("PEER_ROOT_H3B_EFFECT_DISPATCH") != 1 {
				t.Fatal("missing effect barrier")
			}
			return RootPeerResult{}, errors.New("effect ACK lost")
		})
		if e == nil {
			t.Fatal("unexpected success")
		}
		r, e := s.Get(ctx, id)
		if e != nil || r.State != "UNKNOWN" {
			t.Fatal(r, e)
		}
		if lost {
			if calls != 2 || OperationCause(r) != "ROOT_OUTCOME_UNVERIFIED" {
				t.Fatal(r, calls)
			}
		} else if calls != 1 || count("PEER_ROOT_H3B_EFFECT_DISPATCH") != 0 || OperationCause(r) != "ROOT_REVIEW_UNVERIFIED" {
			t.Fatal(r, calls)
		}
		if e = w.ExecutePreparedRootPeerRevocation(id, func(context.Context, string, string) (RootPeerResult, error) {
			t.Fatal("UNKNOWN replayed")
			return RootPeerResult{}, nil
		}); e == nil {
			t.Fatal("UNKNOWN accepted")
		}
	}
}
