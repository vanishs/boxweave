package state

import (
	"context"
	"testing"
)

func TestS4B3GrantAtomicFinalizeOnlyVerifiedOriginal(t *testing.T) {

	ctx := context.Background()
	s, r, id := pr178ScopedReadyToStage(t, true)
	if e := s.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
		t.Fatal(e)
	}
	p := scopedTestReceipt(t, r, id, false)
	bad := p
	bad.ProofSHA256 = "bad"
	if e := s.FinalizeScopedPeerOriginal(ctx, bad); e == nil {
		t.Fatal("unverified staged grant")
	}
	if e := s.FinalizeScopedPeerOriginal(ctx, p); e != nil {
		t.Fatal(e)
	}
	rec, e := s.Get(ctx, id)
	if e != nil || rec.State != "SUCCEEDED" {
		t.Fatal(rec, e)
	}
	var status, verified string
	var desired int
	if e = s.db.QueryRow("SELECT desired,status,last_verified_operation_id FROM peer_grants WHERE source_uuid=? AND target_uuid=?", purePeerUUID(1), purePeerUUID(2)).Scan(&desired, &status, &verified); e != nil || desired != 1 || status != "CONFIRMED" || verified != id {
		t.Fatal(desired, status, verified, e)
	}
	if e = s.FinalizeScopedPeerOriginal(ctx, p); e == nil {
		t.Fatal("replayed final")
	}
	t.Log("S4_B3_GRANT_SQLITE_ONE_ORIGINAL_ATOMIC_POST_KERNEL_FINALIZE=PASS")

}

func TestS4B3GrantFinalizeRefusesAmbiguousLedger(t *testing.T) {

	ctx := context.Background()
	for _, sql := range []string{"UPDATE peer_grants SET status='UNKNOWN' WHERE operation_id=?", "UPDATE peer_grants SET desired=0 WHERE operation_id=?", "DELETE FROM peer_grants WHERE operation_id=?"} {
		t.Run(sql, func(t *testing.T) {
			s, r, id := pr178ScopedReadyToStage(t, true)
			if e := s.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
				t.Fatal(e)
			}
			proof := scopedTestReceipt(t, r, id, false)
			res, e := s.db.Exec(sql, id)
			if e != nil {
				t.Fatal(e)
			}
			if n, e := res.RowsAffected(); e != nil || n == 0 {
				t.Fatal(n, e)
			}
			if e = s.FinalizeScopedPeerOriginal(ctx, proof); e == nil {
				t.Fatal("untrusted ledger")
			}
			rec, e := s.Get(ctx, id)
			if e != nil || rec.State == "SUCCEEDED" {
				t.Fatal(rec, e)
			}
		})
	}
	t.Log("S4_B3_GRANT_FINALIZE_UNKNOWN_LEDGER_FAIL_CLOSED=PASS")

}
