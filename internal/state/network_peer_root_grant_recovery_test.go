package state

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func s4b3UnknownGrantRecoveryFixture(t *testing.T) (*Store, *Store, string) {
	t.Helper()
	s, ro, id := s4b3RootGrantFixture(t)
	if err := s.Claim(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(context.Background(), id, "DISPATCHING", "SIMULATED_ROOT_COMPLETED_ACK_LOST"); err != nil {
		t.Fatal(err)
	}
	return s, ro, id
}

func TestS4B3CompletedGrantLostACKReadOnlyRecoveryNoReplay(t *testing.T) {
	s, ro, id := s4b3UnknownGrantRecoveryFixture(t)
	ctx := context.Background()
	if _, err := ro.ReadRootPeerGrant(ctx, id, "user-1001", "bwbr0", "bw-s4b2"); err == nil {
		t.Fatal("UNKNOWN eligible for ordinary root dispatch")
	}
	grant, err := ro.ReadRootPeerUnknownGrant(ctx, id, "user-1001", "bwbr0", "bw-s4b2")
	if err != nil || !grant.Intent.Allow || len(grant.Target) != 1 {
		t.Fatalf("read-only grant unavailable: %+v %v", grant, err)
	}
	sha := strings.Repeat("a", 64)
	calls := 0
	result := RootPeerResult{Status: "RECONCILE_VERIFIED", PlanSHA256: sha, ProbedScopes: 4,
		MutationComplete: true, KernelVerified: true, VerifiedNICs: 3, PostNativeDigestSHA256: strings.Repeat("b", 64)}
	worker := Worker{Store: s}
	if err := worker.ReconcileCompletedUnknownRootPeer(id, sha, func(_ context.Context, original, hash string) (RootPeerResult, error) {
		calls++
		if original != id || hash != sha {
			t.Fatal("wrong original or plan")
		}
		return result, nil
	}); err != nil {
		t.Fatal(err)
	}
	rec, err := s.Get(ctx, id)
	if err != nil || rec.State != "SUCCEEDED" || calls != 1 {
		t.Fatalf("original not reconciled: %+v %v calls=%d", rec, err, calls)
	}
	var status string
	var desired int
	if err = s.db.QueryRowContext(ctx, "SELECT status,desired FROM peer_grants").Scan(&status, &desired); err != nil ||
		status != "CONFIRMED" || desired != 1 {
		t.Fatalf("grant not confirmed: %s/%d %v", status, desired, err)
	}
	if err = worker.ReconcileCompletedUnknownRootPeer(id, sha, func(context.Context, string, string) (RootPeerResult, error) {
		t.Fatal("recovered original re-dispatched")
		return result, nil
	}); !errors.Is(err, ErrState) {
		t.Fatalf("recovered original reused: %v", err)
	}
	t.Log("S4_B3_COMPLETED_GRANT_ACK_LOSS_READONLY_SETTLEMENT_NO_REPLAY=PASS")
}

func TestS4B3IncompleteGrantRecoveryRetainsUNKNOWN(t *testing.T) {
	for _, bad := range []string{"incomplete", "wrong-sha", "no-kernel", "no-nics", "missing-journal"} {
		t.Run(bad, func(t *testing.T) {
			s, _, id := s4b3UnknownGrantRecoveryFixture(t)
			sha := strings.Repeat("c", 64)
			proof := RootPeerResult{Status: "RECONCILE_VERIFIED", PlanSHA256: sha,
				ProbedScopes: 4, MutationComplete: true, KernelVerified: true, VerifiedNICs: 3,
				PostNativeDigestSHA256: strings.Repeat("d", 64)}
			switch bad {
			case "incomplete":
				proof.MutationComplete = false
			case "wrong-sha":
				proof.PlanSHA256 = strings.Repeat("e", 64)
			case "no-kernel":
				proof.KernelVerified = false
			case "no-nics":
				proof.VerifiedNICs = 0
			}
			err := (Worker{Store: s}).ReconcileCompletedUnknownRootPeer(id, sha,
				func(context.Context, string, string) (RootPeerResult, error) {
					if bad == "missing-journal" {
						return RootPeerResult{}, errors.New("missing original root journal")
					}
					return proof, nil
				})
			if err == nil {
				t.Fatal("unverified root grant proof settled")
			}
			rec, e := s.Get(context.Background(), id)
			if e != nil || rec.State != "UNKNOWN" {
				t.Fatalf("original moved: %+v %v", rec, e)
			}
			var status string
			if e = s.db.QueryRow("SELECT status FROM peer_grants").Scan(&status); e != nil || status != "PENDING" {
				t.Fatalf("pending grant released: %s %v", status, e)
			}
		})
	}
	t.Log("S4_B3_UNVERIFIED_GRANT_RECOVERY_UNKNOWN_AND_LOCKS_RETAINED=PASS")
}
