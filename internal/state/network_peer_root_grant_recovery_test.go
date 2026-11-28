package state

import (
	"context"
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

	scopedUnknownPeerLineage(t, true)
	t.Log("S4_B3_COMPLETED_GRANT_ACK_LOSS_READONLY_SETTLEMENT_NO_REPLAY=PASS")

}

func TestS4B3IncompleteGrantRecoveryRetainsUNKNOWN(t *testing.T) {

	ctx := context.Background()
	for _, bad := range []string{"status", "digest", "steps", "plan", "final", "phase"} {
		t.Run(bad, func(t *testing.T) {
			s, r, id := pr178ScopedReadyToStage(t, true)
			if e := s.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
				t.Fatal(e)
			}
			if e := s.Unknown(ctx, id, "DISPATCHING", "LOST_ROOT_ACK"); e != nil {
				t.Fatal(e)
			}
			p := scopedTestReceipt(t, r, id, true)
			switch bad {
			case "status":
				p.Status = "INCOMPLETE"
			case "digest":
				p.ProofSHA256 = strings.Repeat("f", 64)
			case "steps":
				p.VerifiedSteps = 0
			case "plan":
				p.PlanSHA256 = "bad"
			case "final":
				p.FinalSHA256 = "bad"
			case "phase":
				p.RecoveryOnly = false
			}
			if e := s.FinalizeScopedPeerOriginal(ctx, p); e == nil {
				t.Fatal("incomplete proof")
			}
			rec, e := s.Get(ctx, id)
			if e != nil || rec.State != "UNKNOWN" {
				t.Fatal(rec, e)
			}
			var status string
			if e = s.db.QueryRow("SELECT status FROM peer_grants WHERE operation_id=?", id).Scan(&status); e != nil || status != "PENDING" {
				t.Fatal(status, e)
			}
		})
	}
	t.Log("S4_B3_UNVERIFIED_GRANT_RECOVERY_UNKNOWN_AND_LOCKS_RETAINED=PASS")

}
