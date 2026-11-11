package state

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func setupS2ExecGCTest(t *testing.T) (*Store, string, string, string, string) {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	_, actor, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	const boxUUID = "c0000000-0000-4000-8000-000000000102"
	const nativeID = "d0000000-0000-4000-8000-000000000103"
	const name = "bw-s2-retention-test"
	if err := s.RegisterOwnedBox(ctx, boxUUID, "L0", name, boxUUID); err != nil {
		t.Fatal(err)
	}
	r, err := s.Admit(ctx, Admission{
		Layer: "L0", ActorID: actor.ID, TokenID: actor.ID,
		Key: "gc-original-command-000000001", Action: "box_exec",
		SafeIntent: []byte("{}"), RequestID: "test-original", BoxUUID: boxUUID,
		Incarnation: boxUUID, ResourceKey: "box:" + name,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.NativeAck(ctx, r.ID, nativeID); err != nil {
		t.Fatal(err)
	}
	stdout := fmt.Sprintf("/1.0/instances/%s/logs/exec-output/exec_%s.stdout", name, nativeID)
	stderr := fmt.Sprintf("/1.0/instances/%s/logs/exec-output/exec_%s.stderr", name, nativeID)
	exit := 1
	if err := s.SaveExecOutput(ctx, r.ID, stdout, stderr, &exit); err != nil {
		t.Fatal(err)
	}
	if err := s.Verified(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	return s, r.ID, actor.ID, boxUUID, nativeID
}

func TestS2ExecGCRequiresSevenDaysAndExactTerminalOwnership(t *testing.T) {
	ctx := context.Background()
	s, op, actor, boxUUID, nativeID := setupS2ExecGCTest(t)
	now := time.Now()
	candidates, err := s.ExecGCCandidates(ctx, now, 16)
	if err != nil || len(candidates) != 0 {
		t.Fatalf("premature GC: %v %+v", err, candidates)
	}
	if _, err := s.PrepareExecGC(ctx, op, "stdout", now); !errors.Is(err, ErrState) {
		t.Fatalf("fresh original output allowed deletion: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE exec_outputs SET observed_at=? WHERE operation_id=?",
		now.Add(-8*24*time.Hour).UnixNano(), op); err != nil {
		t.Fatal(err)
	}
	candidates, err = s.ExecGCCandidates(ctx, now, 16)
	if err != nil || len(candidates) != 1 || candidates[0] != op {
		t.Fatalf("aged candidate: %+v %v", candidates, err)
	}
	intent, err := s.PrepareExecGC(ctx, op, "stdout", now)
	if err != nil || intent.OperationID != op || intent.NativeOperationID != nativeID ||
		intent.Project != "L0" || intent.BoxUUID != boxUUID || intent.Stream != "stdout" {
		t.Fatalf("incorrect durable exact-path GC claim: %+v %v", intent, err)
	}
	state, err := s.ExecGCStreamState(ctx, op, "stdout")
	if err != nil || state != "DELETE_SENT" {
		t.Fatalf("claim missing: %q %v", state, err)
	}
	// Concurrent new native actions must not bypass the GC interlock.
	_, err = s.Admit(ctx, Admission{
		Layer: "L0", ActorID: actor, TokenID: actor,
		Key: "gc-conflicting-action-00000001", Action: "box_stop",
		SafeIntent: []byte("{}"), RequestID: "conflict",
		BoxUUID: boxUUID, Incarnation: boxUUID, ResourceKey: "box:bw-s2-retention-test",
	})
	if !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("admission bypassed live native DELETE: %v", err)
	}
	if err := s.FinishExecGC(ctx, op, "stdout", "EXPIRED"); err != nil {
		t.Fatal(err)
	}
	st, err := s.ExecGCStreamState(ctx, op, "stdout")
	if err != nil || st != "EXPIRED" {
		t.Fatalf("verified expiry missing: %q %v", st, err)
	}
	if _, err := s.PrepareExecGC(ctx, op, "stdout", now); !errors.Is(err, ErrState) {
		t.Fatalf("expired stream can be redispatched: %v", err)
	}
	other, err := s.PrepareExecGC(ctx, op, "stderr", now)
	if err != nil || other.Stream != "stderr" {
		t.Fatalf("untouched stderr not eligible: %+v %v", other, err)
	}
	if err := s.FinishExecGC(ctx, op, "stderr", "REVIEW_REQUIRED"); err != nil {
		t.Fatal(err)
	}
	candidates, err = s.ExecGCCandidates(ctx, now, 16)
	if err != nil || len(candidates) != 0 {
		t.Fatalf("uncertain streams requeued: %+v %v", candidates, err)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM exec_output_gc_audit WHERE operation_id=?", op).Scan(&count); err != nil || count != 4 {
		t.Fatalf("GC audit is not append-only/complete: %d %v", count, err)
	}
}

func TestS2ExecGCInterruptedNativeDELETEBecomesReviewOnlyNoRetry(t *testing.T) {
	ctx := context.Background()
	s, op, actor, boxUUID, _ := setupS2ExecGCTest(t)
	now := time.Now()
	if _, err := s.db.ExecContext(ctx, "UPDATE exec_outputs SET observed_at=? WHERE operation_id=?",
		now.Add(-8*24*time.Hour).UnixNano(), op); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PrepareExecGC(ctx, op, "stdout", now); err != nil {
		t.Fatal(err)
	}
	n, err := s.RecoverExecGC(ctx)
	if err != nil || n != 1 {
		t.Fatalf("interrupted delete not fenced: %d %v", n, err)
	}
	st, err := s.ExecGCStreamState(ctx, op, "stdout")
	if err != nil || st != "REVIEW_REQUIRED" {
		t.Fatalf("crash was not review-only: %q %v", st, err)
	}
	n, err = s.RecoverExecGC(ctx)
	if err != nil || n != 0 {
		t.Fatalf("recovery retried original: %d %v", n, err)
	}
	if _, err := s.PrepareExecGC(ctx, op, "stdout", now); !errors.Is(err, ErrState) {
		t.Fatalf("ambiguous DELETE reissued: %v", err)
	}
	_, err = s.Admit(ctx, Admission{
		Layer: "L0", ActorID: actor, TokenID: actor,
		Key: "gc-blocked-after-crash-000001", Action: "box_delete",
		SafeIntent: []byte("{}"), RequestID: "blocked",
		BoxUUID: boxUUID, Incarnation: boxUUID, ResourceKey: "box:bw-s2-retention-test",
	})
	if !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("postcrash lock lost: %v", err)
	}
	var before, review int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM exec_output_gc_audit WHERE operation_id=? AND event='BEFORE_DELETE'", op).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM exec_output_gc_audit WHERE operation_id=? AND event='RECOVERED_UNCERTAIN'", op).Scan(&review); err != nil {
		t.Fatal(err)
	}
	if before != 1 || review != 1 {
		t.Fatalf("uncertain native write lost audit before=%d recovered=%d", before, review)
	}
}

func TestS2ExecGCRejectsTamperedNativePaths(t *testing.T) {
	ctx := context.Background()
	s, op, _, _, _ := setupS2ExecGCTest(t)
	now := time.Now()
	if _, err := s.db.ExecContext(ctx, "UPDATE exec_outputs SET observed_at=?,stdout_path=? WHERE operation_id=?",
		now.Add(-8*24*time.Hour).UnixNano(),
		"/1.0/instances/foreign/logs/exec-output/exec_d0000000-0000-4000-8000-000000000103.stdout", op); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PrepareExecGC(ctx, op, "stdout", now); !errors.Is(err, ErrState) {
		t.Fatalf("foreign native output path accepted for deletion: %v", err)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM exec_output_gc WHERE operation_id=?", op).Scan(&count); err != nil || count != 0 {
		t.Fatalf("untrusted path created deletion intent: count=%d err=%v", count, err)
	}
}
