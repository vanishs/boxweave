package state

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func scopedTestReceipt(t *testing.T, r *Store, id string, unknown bool) ScopedPeerRootReceipt {
	t.Helper()
	ctx := context.Background()
	var p ScopedPeerOriginalProof
	var e error
	if unknown {
		p, e = r.ReadRootScopedPeerUnknownOriginal(ctx, id, "user-1001", purePeerBridge)
	} else {
		p, e = r.ReadRootScopedPeerOriginal(ctx, id, "user-1001", purePeerBridge)
	}
	if e != nil {
		t.Fatal(e)
	}
	return ScopedPeerRootReceipt{OriginalID: id, Project: p.Intent.Project, Bridge: p.Intent.Bridge, Status: "COMPLETE_VERIFIED", ProofSHA256: ScopedPeerProofSHA256(p), PlanSHA256: strings.Repeat("a", 64), FinalSHA256: strings.Repeat("b", 64), VerifiedSteps: 4, RecoveryOnly: unknown}
}

func TestPR178N3AtomicScopedPeerFinalization(t *testing.T) {
	for _, allow := range []bool{true, false} {
		for _, fault := range []string{"success", "complete-unknown", "lost-root-ack", "invalid-receipt", "changed-actor", "wrong-claim-layer", "commit-fault", "stage-fault"} {
			t.Run(map[bool]string{true: "grant-", false: "revoke-"}[allow]+fault, func(t *testing.T) {
				s, r, id := pr178ScopedReadyToStage(t, allow)
				ctx := context.Background()
				if fault == "stage-fault" {
					if _, e := s.db.Exec("CREATE TRIGGER n3_stage_abort BEFORE UPDATE ON operations WHEN NEW.state='DISPATCHING' BEGIN SELECT RAISE(ABORT,'stage fault'); END"); e != nil {
						t.Fatal(e)
					}
					calls := 0
					e := (Worker{Store: s}).ExecuteScopedPeerOriginal(id, func(context.Context, string) (ScopedPeerRootReceipt, error) {
						calls++
						return ScopedPeerRootReceipt{}, nil
					})
					var n int
					if q := s.db.QueryRow("SELECT COUNT(*) FROM peer_grants WHERE operation_id=? AND status='PENDING'", id).Scan(&n); q != nil || e == nil || n != 0 || calls != 0 {
						t.Fatal("stage rollback failed", e, q, n, calls)
					}
					return
				}
				calls := 0
				var original ScopedPeerRootReceipt
				e := (Worker{Store: s}).ExecuteScopedPeerOriginal(id, func(_ context.Context, got string) (ScopedPeerRootReceipt, error) {
					calls++
					if got != id {
						t.Fatal("wrong original")
					}
					original = scopedTestReceipt(t, r, id, false)
					out := original
					switch fault {
					case "complete-unknown", "lost-root-ack":
						return out, errors.New("lost ACK")
					case "invalid-receipt":
						out.ProofSHA256 = strings.Repeat("c", 64)
					case "changed-actor":
						if _, e := s.db.Exec("UPDATE operations SET actor_id='other' WHERE id=?", id); e != nil {
							t.Fatal(e)
						}
					case "wrong-claim-layer":
						if _, e := s.db.Exec("UPDATE operation_resource_claims SET layer='foreign' WHERE operation_id=?", id); e != nil {
							t.Fatal(e)
						}
					case "commit-fault":
						if _, e := s.db.Exec("CREATE TRIGGER n3_finalize_abort BEFORE UPDATE ON operations WHEN NEW.state='SUCCEEDED' BEGIN SELECT RAISE(ABORT,'final fault'); END"); e != nil {
							t.Fatal(e)
						}
					}
					return out, nil
				})
				if fault == "complete-unknown" {
					proof := scopedTestReceipt(t, r, id, true)
					if proof.ProofSHA256 != original.ProofSHA256 {
						t.Fatal("phase changed proof digest")
					}
					e = s.FinalizeScopedPeerOriginal(ctx, proof)
				}
				success := fault == "success" || fault == "complete-unknown"
				got, q := s.Get(ctx, id)
				if q != nil || calls != 1 || (e == nil) != success {
					t.Fatal(got, e, q, calls)
				}
				want := "UNKNOWN"
				status := "PENDING"
				if success {
					want = "SUCCEEDED"
					status = "CONFIRMED"
					if !allow {
						status = "REVOKED"
					}
				}
				var actual string
				if q = s.db.QueryRow("SELECT status FROM peer_grants WHERE source_uuid=? AND target_uuid=?", s4b2RootSrc, s4b2RootDst).Scan(&actual); q != nil || actual != status || got.State != want {
					t.Fatal(got, actual, want, status, q)
				}
				if e = (Worker{Store: s}).ExecuteScopedPeerOriginal(id, func(context.Context, string) (ScopedPeerRootReceipt, error) { calls++; return original, nil }); e == nil || calls != 1 {
					t.Fatal("duplicate dispatch", e, calls)
				}
			})
		}
	}
}
