package state

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func s4b2RootFakeCall(t *testing.T, calls *int, resultMode string) RootPeerCall {
	t.Helper()
	return func(_ context.Context, id, sha string) (RootPeerResult, error) {
		*calls++
		if len(id) != 32 {
			t.Fatalf("root received non-original operation: %q", id)
		}
		if sha == "" {
			return RootPeerResult{Status: "REVIEW_ONLY", PlanSHA256: strings.Repeat("a", 64)}, nil
		}
		if sha != strings.Repeat("a", 64) {
			t.Fatalf("root received a stale or foreign plan SHA: %q", sha)
		}
		switch resultMode {
		case "lost-ack":
			return RootPeerResult{Status: "DISPATCH_UNKNOWN"}, errors.New("lost ACK after native conntrack removal")
		case "partial":
			return RootPeerResult{Status: "DISPATCH_UNKNOWN", PlanSHA256: sha, ProbedScopes: 3}, nil
		}
		return RootPeerResult{
			Status: "COMPLETE_VERIFIED", PlanSHA256: sha,
			ProbedScopes: 4, DeletedScopes: 2, MutationComplete: true, ReadyForPeer: false,
		}, nil
	}
}

func TestS4B2ControllerRootRevocationAtomicVerifiedFinish(t *testing.T) {
	ctx := context.Background()
	s, _, id := s4b2RootOperationFixture(t)
	s4b2PendingExistingGrantToConfirmed(t, s)
	w := Worker{Store: s}
	isolations, calls, verified := 0, 0, 0
	w.ExecuteRootPeerRevocation(id,
		func(context.Context) error {
			isolations++
			if calls != 0 {
				t.Fatal("root review/dispatch occurred before isolation")
			}
			return nil
		}, s4b2RootFakeCall(t, &calls, ""),
		func(_ context.Context, r RootPeerResult) error {
			verified++
			if r.Status != "COMPLETE_VERIFIED" || r.ReadyForPeer || r.ProbedScopes != 4 {
				t.Fatal("false root result in controller verifier")
			}
			state, err := s.Get(ctx, id)
			if err != nil || state.State != "VERIFYING" {
				t.Fatalf("premature succeeded before per-NIC packet verification: %+v %v", state, err)
			}
			return nil
		})
	r, err := s.Get(ctx, id)
	if err != nil || r.State != "SUCCEEDED" ||
		isolations != 1 || calls != 2 || verified != 1 {
		t.Fatalf("root controller did not atomically verify: %+v %v isolation=%d calls=%d verified=%d",
			r, err, isolations, calls, verified)
	}
	var status string
	var desired int
	if err := s.db.QueryRowContext(ctx, "SELECT status,desired FROM peer_grants WHERE operation_id=?", id).
		Scan(&status, &desired); err != nil || status != "REVOKED" || desired != 0 {
		t.Fatalf("operation succeeded without atomic revoked grant: %v %q desired=%d", err, status, desired)
	}
	w.ExecuteRootPeerRevocation(id, func(context.Context) error { t.Fatal("replayed"); return nil },
		s4b2RootFakeCall(t, &calls, ""), func(context.Context, RootPeerResult) error { return nil })
	if calls != 2 {
		t.Fatal("completed original replayed root")
	}
	t.Log("S4_B2_CONTROLLER_ROOT_REVOKE_ATOMIC_ORIGINAL_AND_GRANT_SUCCESS=PASS")
}

func TestS4B2ControllerRootRevocationUnknownFences(t *testing.T) {
	cases := []struct {
		name, mode          string
		isolation, verifier bool
	}{
		{"failed-native-isolation", "", true, false},
		{"lost-native-ACK", "lost-ack", false, false},
		{"partial-native-root", "partial", false, false},
		{"failed-kernel-postread", "", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, _, id := s4b2RootOperationFixture(t)
			s4b2PendingExistingGrantToConfirmed(t, s)
			calls, verifies := 0, 0
			isolate := func(context.Context) error {
				if tc.isolation {
					return errors.New("quiescence uncertain")
				}
				return nil
			}
			verify := func(context.Context, RootPeerResult) error {
				verifies++
				if tc.verifier {
					return errors.New("per-NIC nft packet readback uncertain")
				}
				return nil
			}
			w := Worker{Store: s}
			w.ExecuteRootPeerRevocation(id, isolate, s4b2RootFakeCall(t, &calls, tc.mode), verify)
			r, err := s.Get(ctx, id)
			if err != nil || r.State != "UNKNOWN" {
				t.Fatalf("failed root original did not stay UNKNOWN: %+v %v", r, err)
			}
			var status string
			if err := s.db.QueryRowContext(ctx, "SELECT status FROM peer_grants WHERE operation_id=?", id).Scan(&status); err != nil || status != "PENDING" {
				t.Fatalf("unknown operation released pending peer grant: %s %v", status, err)
			}
			prior := calls
			w.ExecuteRootPeerRevocation(id, isolate, s4b2RootFakeCall(t, &calls, tc.mode), verify)
			if calls != prior {
				t.Fatal("root UNKNOWN original replayed")
			}
			_, tok, err := s.IssueToken(ctx,time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.Admit(ctx, Admission{
				Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID,
				Key: "s4b2-root-revoke-conflicting-0001", RequestID: "conflict",
				Action: "network_acl_reconcile", SafeIntent: []byte("bypass"),
				ResourceKey: "acl:default/bw-s4b2",
			})
			if !errors.Is(err, ErrResourceLocked) {
				t.Fatalf("UNKNOWN root ACL claims released, bypass=%v", err)
			}
			if tc.isolation && calls != 0 {
				t.Fatal("unisolated root request sent")
			}
			if tc.mode == "lost-ack" && calls != 2 {
				t.Fatal("lost-ACK fixture did not dispatch exactly once")
			}
			if tc.verifier && verifies != 1 {
				t.Fatal("post-kernel readback was not exercised")
			}
		})
	}
	t.Log("S4_B2_CONTROLLER_ROOT_LOST_ACK_NFT_FAILURE_NO_REPLAY=PASS")
}
