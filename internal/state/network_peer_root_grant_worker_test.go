package state

import (
	"context"
	"errors"
	"testing"
)

func TestS4B3GrantAtomicFinalizeOnlyVerifiedOriginal(t *testing.T) {
	s, id := s4b3GrantFixture(t)
	ctx := context.Background()
	if err := s.StagePeerGrant(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeRootPeerGrant(ctx, id); !errors.Is(err, ErrState) {
		t.Fatalf("unverified staged grant marked CONFIRMED: %v", err)
	}
	if err := s.Claim(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.AckSynchronousACL(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeRootPeerGrant(ctx, id); err != nil {
		t.Fatal(err)
	}
	rec, err := s.Get(ctx, id)
	if err != nil || rec.State != "SUCCEEDED" {
		t.Fatalf("grant receipt not succeeded %+v %v", rec, err)
	}
	var status string
	var desired int
	if err := s.db.QueryRowContext(ctx, "SELECT desired,status FROM peer_grants").Scan(&desired, &status); err != nil ||
		desired != 1 || status != "CONFIRMED" {
		t.Fatalf("grant row not atomically confirmed: %d %s %v", desired, status, err)
	}
	if err := s.FinalizeRootPeerGrant(ctx, id); !errors.Is(err, ErrState) {
		t.Fatalf("replayed final original: %v", err)
	}
	t.Log("S4_B3_GRANT_SQLITE_ONE_ORIGINAL_ATOMIC_POST_KERNEL_FINALIZE=PASS")
}

func TestS4B3GrantFinalizeRefusesAmbiguousLedger(t *testing.T) {
	for _, query := range []string{"UPDATE peer_grants SET status='UNKNOWN'", "UPDATE peer_grants SET desired=0",
		"DELETE FROM peer_grants"} {
		t.Run(query, func(t *testing.T) {
			s, id := s4b3GrantFixture(t)
			ctx := context.Background()
			if err := s.StagePeerGrant(ctx, id); err != nil {
				t.Fatal(err)
			}
			if err := s.Claim(ctx, id); err != nil {
				t.Fatal(err)
			}
			if err := s.AckSynchronousACL(ctx, id); err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.Exec(query); err != nil {
				t.Fatal(err)
			}
			if err := s.FinalizeRootPeerGrant(ctx, id); err == nil {
				t.Fatal("untrusted grant ledger was confirmed")
			}
			rec, err := s.Get(ctx, id)
			if err != nil || rec.State == "SUCCEEDED" {
				t.Fatalf("unsafe receipt %+v %v", rec, err)
			}
		})
	}
	t.Log("S4_B3_GRANT_FINALIZE_UNKNOWN_LEDGER_FAIL_CLOSED=PASS")
}
