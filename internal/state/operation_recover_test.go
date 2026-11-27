package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/peerscope"
)

const rNative = "12345678-1234-4abc-8def-123456789abc"
const rUUID = "87654321-4321-4abc-8def-abcdef012345"

func recoveryOriginal(t *testing.T, s *Store, action, uuid string) (RecoveryRequest, RecoveryTarget) {
	t.Helper()
	ctx := context.Background()
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	if uuid != "" {
		if e = s.RegisterOwnedBox(ctx, uuid, "recovery-test", "owned", uuid); e != nil {
			t.Fatal(e)
		}
	}
	orig, e := s.Admit(ctx, Admission{Layer: "recovery-test", ActorID: tok.ID, TokenID: tok.ID, Key: "original-recovery-fixture-key", Action: action, RequestID: "original", ResourceKey: "box:owned", BoxUUID: uuid, Incarnation: uuid, SafeIntent: []byte(`{"name":"owned"}`)})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, orig.ID); e != nil {
		t.Fatal(e)
	}
	from := "DISPATCHING"
	if uuid != "" {
		if e = s.NativeAck(ctx, orig.ID, rNative); e != nil {
			t.Fatal(e)
		}
		from = "VERIFYING"
	}
	if e = s.Unknown(ctx, orig.ID, from, "TEST_ORIGINAL_UNKNOWN"); e != nil {
		t.Fatal(e)
	}
	r := RecoveryRequest{Layer: "recovery-test", ActorID: tok.ID, TokenID: tok.ID, OriginalID: orig.ID, Key: "new-recovery-request-key-001", WorkRef: "s5a2/test"}
	target, e := s.ReadRecoveryTarget(ctx, r.Layer, r.OriginalID)
	if e != nil {
		t.Fatal(e)
	}
	return r, target
}

// The legacy fixture remains available to explicit transitional witnesses.
func recoveryOriginalMode(t *testing.T, s *Store, action string, scoped bool) (RecoveryRequest, RecoveryTarget) {
	t.Helper()
	if !scoped {
		return recoveryOriginal(t, s, action, rUUID)
	}
	ctx := context.Background()
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	in := ScopedCreateIntent{Schema: 2, Project: "user-1001", Bridge: "incusbr-1001", Name: "owned", CPU: 1, MemoryMiB: 1024, DiskGiB: 10, Internet: true, BootAutostart: true, Options: json.RawMessage("{}")}
	raw, e := json.Marshal(in)
	if e != nil {
		t.Fatal(e)
	}
	a := Admission{Layer: in.Project, ActorID: tok.ID, TokenID: tok.ID, Key: "generic-scoped-create-fixture", Action: "box_create", RequestID: "create", ResourceKey: "box:owned", SafeIntent: raw}
	c, e := s.AdmitScoped(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, c.ID); e != nil {
		t.Fatal(e)
	}
	pin := VerifiedCreateBinding{in.Bridge, peerscope.QuarantineACL, "02:16:3e:11:22:33", "10.243.18.11", "fd43:18::11"}
	if ok, e := s.StageCreateBinding(ctx, c.ID, in.Project, in.Name, pin); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = s.NativeAck(ctx, c.ID, "synthetic-create-receipt"); e != nil {
		t.Fatal(e)
	}
	pin.ACL, e = peerscope.ACLName(in.Project, in.Bridge, rUUID)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedScopedCreate(ctx, c.ID, in.Project, in.Name, rUUID, pin); e != nil {
		t.Fatal(e)
	}
	a.Key, a.Action, a.RequestID = "original-recovery-fixture-key", action, "original"
	a.BoxUUID, a.Incarnation, a.SafeIntent = rUUID, rUUID, []byte(`{"box_id":"`+rUUID+`"}`)
	orig, e := s.AdmitScoped(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, orig.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, orig.ID, rNative); e != nil {
		t.Fatal(e)
	}
	if e = s.Unknown(ctx, orig.ID, "VERIFYING", "TEST_ORIGINAL_UNKNOWN"); e != nil {
		t.Fatal(e)
	}
	r := RecoveryRequest{Layer: in.Project, ActorID: tok.ID, TokenID: tok.ID, OriginalID: orig.ID, Key: "new-recovery-request-key-001", WorkRef: "s5a2/test"}
	target, e := s.ReadRecoveryTarget(ctx, r.Layer, orig.ID)
	if e != nil || target.ScopedLifecycle || target.ScopedPeer {
		t.Fatal(target, e)
	}
	if (action == "box_start" || action == "box_stop") && (target.LifecyclePinStatus != "PINNED" || target.LifecycleBinding != pin) {
		t.Fatal(target)
	}
	return r, target
}

func recoveryCommitMode(s *Store, scoped bool) func(context.Context, RecoveryRequest, RecoveryTarget, RecoveryProof) (RecoveryRecord, error) {
	if scoped {
		return s.CommitScopedRecovery
	}
	return s.CommitRecovery
}

func recoverProof(t RecoveryTarget) RecoveryProof {
	return RecoveryProof{Kind: RecoverySettle, Reason: "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED", NativeID: t.NativeID, UUID: t.UUID, Fingerprint: "test-proof"}
}
func recoverProofWithPin(t RecoveryTarget) RecoveryProof {
	p := recoverProof(t)
	p.LifecycleBinding = t.LifecycleBinding
	return p
}

func TestS5A2RecoveryAtomicReceiptAndReplay(t *testing.T) {
	for _, scoped := range []bool{false, true} {
		t.Run(fmt.Sprint(scoped), func(t *testing.T) { recoveryAtomicReceiptAndReplay(t, scoped) })
	}
}

func recoveryAtomicReceiptAndReplay(t *testing.T, scoped bool) {
	ctx := context.Background()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "recovery.db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	r, target := recoveryOriginalMode(t, s, "box_start", scoped)
	// Simulated lost COMMIT response: discard return before reopening DB.
	first, e := recoveryCommitMode(s, scoped)(ctx, r, target, recoverProofWithPin(target))
	if e != nil {
		t.Fatal(e)
	}
	if first.Receipt.ID == target.ID || first.Receipt.State != "SUCCEEDED" || first.Receipt.NativeOperationID != "" || first.Disposition != "SETTLED" {
		t.Fatalf("wrong recovery receipt %+v", first)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	duplicate, e := s.ReplayRecovery(ctx, r)
	if e != nil || !duplicate.Receipt.Duplicate || duplicate.Receipt.ID != first.Receipt.ID || duplicate.Disposition != "SETTLED" {
		t.Fatalf("lost receipt %+v %v", duplicate, e)
	}
	current, e := s.ReadRecoveryTarget(ctx, r.Layer, r.OriginalID)
	if e != nil || current.State != "SUCCEEDED" || current.NativeID != rNative || current.Released {
		t.Fatalf("wrong original %+v %v", current, e)
	}
	var n int
	if e = s.db.QueryRow("SELECT count(*) FROM operation_events WHERE operation_id=? AND state='UNKNOWN'", target.ID).Scan(&n); e != nil || n != 1 {
		t.Fatalf("lost unknown history: %d %v", n, e)
	}
	if e = s.db.QueryRow("SELECT count(*) FROM recovery_audit WHERE operation_id=? AND decision='RECOVERY_SETTLED'", target.ID).Scan(&n); e != nil || n != 1 {
		t.Fatalf("lost audit: %d %v", n, e)
	}
	if _, e = s.ReadRecoveryTarget(ctx, "foreign", target.ID); !errors.Is(e, ErrNotFound) {
		t.Fatalf("cross-layer allowed: %v", e)
	}
	wrong := r
	wrong.Stream = "stdout"
	if _, e = s.ReplayRecovery(ctx, wrong); !errors.Is(e, ErrConflict) {
		t.Fatalf("same key changed intent: %v", e)
	}
	wrong = r
	wrong.WorkRef = "different"
	if _, e = s.ReplayRecovery(ctx, wrong); !errors.Is(e, ErrConflict) {
		t.Fatalf("same key changed attribution: %v", e)
	}
	if e = s.RevokeToken(ctx, r.TokenID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ReplayRecovery(ctx, r); !errors.Is(e, ErrRevoked) {
		t.Fatalf("revoked replay: %v", e)
	}
}
func TestS5A2RecoveryCommitFaultRollsBackBoth(t *testing.T) {
	for _, scoped := range []bool{false, true} {
		t.Run(fmt.Sprint(scoped), func(t *testing.T) { recoveryCommitFaultRollsBackBoth(t, scoped) })
	}
}

func recoveryCommitFaultRollsBackBoth(t *testing.T, scoped bool) {
	s := newStore(t)
	ctx := context.Background()
	r, target := recoveryOriginalMode(t, s, "box_stop", scoped)
	// Test-only fault inside the same SQLite transaction, after the original
	// terminal UPDATE but before its recovery receipt event can commit.
	if _, e := s.db.Exec(`CREATE TRIGGER s5a2_fail BEFORE INSERT ON operation_events WHEN NEW.safe_detail LIKE 'SETTLED:%' BEGIN SELECT RAISE(ABORT,'injected recovery event fault'); END`); e != nil {
		t.Fatal(e)
	}
	if _, e := recoveryCommitMode(s, scoped)(ctx, r, target, recoverProofWithPin(target)); e == nil {
		t.Fatal("fault not observed")
	}
	if got, e := s.ReadRecoveryTarget(ctx, r.Layer, target.ID); e != nil || got != target {
		t.Fatalf("partial original commit %+v %v", got, e)
	}
	if _, e := s.ReplayRecovery(ctx, r); !errors.Is(e, ErrNotFound) {
		t.Fatalf("partial recovery receipt: %v", e)
	}
	if _, e := s.db.Exec("DROP TRIGGER s5a2_fail"); e != nil {
		t.Fatal(e)
	}
	if _, e := recoveryCommitMode(s, scoped)(ctx, r, target, recoverProofWithPin(target)); e != nil {
		t.Fatal(e)
	}
}
func TestS5A2RecoveryConcurrentSameKeyAndDifferentKey(t *testing.T) {
	for _, scoped := range []bool{false, true} {
		t.Run(fmt.Sprint(scoped), func(t *testing.T) { recoveryConcurrentSameKeyAndDifferentKey(t, scoped) })
	}
}

func recoveryConcurrentSameKeyAndDifferentKey(t *testing.T, scoped bool) {
	for _, same := range []bool{true, false} {
		t.Run(fmt.Sprint(same), func(t *testing.T) {
			s := newStore(t)
			ctx := context.Background()
			r, target := recoveryOriginalMode(t, s, "box_start", scoped)
			type result struct {
				v RecoveryRecord
				e error
			}
			results := make(chan result, 2)
			var wg sync.WaitGroup
			for i := range 2 {
				req := r
				if !same {
					req.Key = fmt.Sprintf("distinct-recovery-key-%03d", i)
				}
				wg.Go(func() {
					v, e := recoveryCommitMode(s, scoped)(ctx, req, target, recoverProofWithPin(target))
					results <- result{v, e}
				})
			}
			wg.Wait()
			close(results)
			var ids []string
			conflicts := 0
			for x := range results {
				if errors.Is(x.e, ErrConflict) {
					conflicts++
					continue
				}
				if x.e != nil {
					t.Fatal(x.e)
				}
				ids = append(ids, x.v.Receipt.ID)
			}
			if same && (len(ids) != 2 || ids[0] != ids[1] || conflicts != 0) {
				t.Fatalf("same key %v conflicts=%d", ids, conflicts)
			}
			if !same && (len(ids) != 1 || conflicts != 1) {
				t.Fatalf("distinct keys %v conflicts=%d", ids, conflicts)
			}
		})
	}
}
func TestS5A2RecoveryBindDoesNotUnlockOrInventNativeAck(t *testing.T) {
	ctx := context.Background()
	s, original, in, _ := p1CreateFixture(t, false)
	if e := s.Unknown(ctx, original.ID, "DISPATCHING", "ACK_LOST"); e != nil {
		t.Fatal(e)
	}
	r, target := a4Request(t, s, original.ID)
	var originalSeq, originalAt int64
	var originalDetail string
	if e := s.db.QueryRow("SELECT seq,at,safe_detail FROM operation_events WHERE operation_id=? AND state='UNKNOWN'", target.ID).Scan(&originalSeq, &originalAt, &originalDetail); e != nil {
		t.Fatal(e)
	}
	var claims int
	if e := s.db.QueryRow("SELECT count(*) FROM operation_resource_claims WHERE operation_id=?", target.ID).Scan(&claims); e != nil || claims == 0 {
		t.Fatal(claims, e)
	}
	p := RecoveryProof{Kind: RecoveryBind, Reason: "CREATE_MARKER_EXACT_IDENTITY", UUID: p1UUID, Fingerprint: "test-marker"}
	if _, e := s.CommitScopedRecovery(ctx, r, target, p); !errors.Is(e, ErrResourceLocked) {
		t.Fatalf("retired bind accepted: %v", e)
	}
	current, e := s.ReadRecoveryTarget(ctx, r.Layer, target.ID)
	if e != nil || current != target || current.State != "UNKNOWN" || current.NativeID != "" || current.Released {
		t.Fatalf("unsafe binding %+v %v", current, e)
	}
	if _, e = s.ReplayRecovery(ctx, r); !errors.Is(e, ErrNotFound) {
		t.Fatalf("refusal created receipt: %v", e)
	}
	var at int64
	var detail, phase string
	if e = s.db.QueryRow("SELECT at,state,safe_detail FROM operation_events WHERE operation_id=? AND seq=?", target.ID, originalSeq).Scan(&at, &phase, &detail); e != nil || at != originalAt || phase != "UNKNOWN" || detail != originalDetail {
		t.Fatalf("original event changed %v", e)
	}
	var n int
	if e = s.db.QueryRow("SELECT count(*) FROM operation_events WHERE operation_id=? AND state='UNKNOWN'", target.ID).Scan(&n); e != nil || n != 1 {
		t.Fatalf("expected unchanged UNKNOWN history, got %d %v", n, e)
	}
	if e = s.db.QueryRow("SELECT count(*) FROM recovery_audit WHERE operation_id=? AND decision='RECOVERY_IDENTITY_BOUND'", target.ID).Scan(&n); e != nil || n != 0 {
		t.Fatalf("refusal wrote binding audit %d %v", n, e)
	}
	if _, e = s.OwnedBox(ctx, in.Project, p1UUID); !errors.Is(e, ErrNotFound) {
		t.Fatalf("refusal registered identity: %v", e)
	}
	if e = s.db.QueryRow("SELECT count(*) FROM operation_resource_claims WHERE operation_id=?", target.ID).Scan(&n); e != nil || n != claims {
		t.Fatalf("original lost claims: %d %v", n, e)
	}
}
func TestS5A2RecoveryExecEvidenceAndDeleteAtomic(t *testing.T) {
	// Scoped Delete atomicity has its dedicated witness TestP1ScopedDeleteCompletedRecoveryAtomic.
	ctx := context.Background()
	s := newStore(t)
	r, target := recoveryOriginalMode(t, s, "box_exec", true)
	p := recoverProof(target)
	exit := 127
	p.Exit = &exit
	p.StdoutPath = "/1.0/instances/owned/logs/exec-output/exec_" + rNative + ".stdout"
	p.StderrPath = "/1.0/instances/owned/logs/exec-output/exec_" + rNative + ".stderr"
	bad := p
	bad.StdoutPath = "/1.0/instances/foreign/logs/exec-output/exec_" + rNative + ".stdout"
	if _, e := s.CommitScopedRecovery(ctx, r, target, bad); !errors.Is(e, ErrInvalid) {
		t.Fatalf("foreign path accepted %v", e)
	}
	got, e := s.CommitScopedRecovery(ctx, r, target, p)
	if e != nil || got.Disposition != "SETTLED" {
		t.Fatalf("%+v %v", got, e)
	}
	out, e := s.ExecOutput(ctx, target.ID)
	if e != nil || out.ExitCode == nil || *out.ExitCode != 127 || out.StdoutPath != p.StdoutPath {
		t.Fatalf("lost guest exit %+v %v", out, e)
	}
}
func TestS5A2RecoveryRefusalAndLocalParity(t *testing.T) {
	for _, scoped := range []bool{false, true} {
		t.Run(fmt.Sprint(scoped), func(t *testing.T) { recoveryRefusalAndLocalParity(t, scoped) })
	}
}

func recoveryRefusalAndLocalParity(t *testing.T, scoped bool) {
	ctx := context.Background()
	s := newStore(t)
	r, target := recoveryOriginalMode(t, s, "box_exec", scoped)
	p := RecoveryProof{Kind: RecoveryApproval, Reason: "ORIGINAL_NATIVE_RECEIPT_UNAVAILABLE"}
	got, e := recoveryCommitMode(s, scoped)(ctx, r, target, p)
	if e != nil || got.Disposition != "APPROVAL_REQUIRED" || got.Receipt.State != "FAILED" {
		t.Fatalf("refusal %+v %v", got, e)
	}
	current, e := s.ReadRecoveryTarget(ctx, r.Layer, target.ID)
	if e != nil || current != target {
		t.Fatalf("refusal modified original %+v %v", current, e)
	}
	local := RecoveryRequest{Layer: r.Layer, LocalReviewer: "local-operator", OriginalID: target.ID, Key: r.Key}
	reviewed, e := recoveryCommitMode(s, scoped)(ctx, local, target, p)
	if e != nil || reviewed.Receipt.ID == got.Receipt.ID {
		t.Fatalf("local parity / key isolation %+v %v", reviewed, e)
	}
	if e = s.RevokeToken(ctx, r.TokenID); e != nil {
		t.Fatal(e)
	}
	r.Key = "revoked-recovery-request-new"
	if _, e = recoveryCommitMode(s, scoped)(ctx, r, target, p); !errors.Is(e, ErrRevoked) {
		t.Fatalf("revoked before commit: %v", e)
	}
}
