package state

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/peerscope"
)

func TestP1ScopedAdmissionOriginalReplay(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, actor, err := s.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	admission := func(project, bridge, name, key string) Admission {
		raw, err := json.Marshal(ScopedCreateIntent{Schema: 2, Project: project, Bridge: bridge, Name: name, CPU: 1, MemoryMiB: 1024, DiskGiB: 10, Options: json.RawMessage("{}")})
		if err != nil {
			t.Fatal(err)
		}
		return Admission{Layer: project, ActorID: actor.ID, TokenID: actor.ID, Key: key, RequestID: key, Action: "box_create", ResourceKey: "box:" + name, SafeIntent: raw}
	}
	original := admission("user-2002", "incusbr-2002", "scoped-original", "p1-scoped-admit-original")
	rec, err := s.AdmitScoped(ctx, original)
	if err != nil || rec.Duplicate {
		t.Fatal(rec, err)
	}
	if err = s.Claim(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.Unknown(ctx, rec.ID, "DISPATCHING", "NATIVE_OUTCOME_AMBIGUOUS"); err != nil {
		t.Fatal(err)
	}
	var n int
	if got, err := s.AdmitScoped(ctx, original); err != nil || !got.Duplicate || got.ID != rec.ID || got.TrackingID != rec.TrackingID {
		t.Fatal(got, err)
	}
	fresh := admission(original.Layer, "incusbr-2002", "scoped-fresh", "p1-scoped-admit-fresh")
	if _, err = s.AdmitScoped(ctx, fresh); !errors.Is(err, ErrResourceLocked) {
		t.Fatal("unsettled original admitted sibling", err)
	}
	for _, q := range []string{"SELECT count(*) FROM operations WHERE layer=? AND resource_key='box:scoped-fresh'", "SELECT count(*) FROM operation_resource_claims WHERE layer=? AND resource_key='box:scoped-fresh'"} {
		if err = s.db.QueryRow(q, fresh.Layer).Scan(&n); err != nil || n != 0 {
			t.Fatal("partial admission", q, n, err)
		}
	}
}

func TestP1ScopedOriginalRecoveryReplayAndConflictingRepair(t *testing.T) {
	ctx := context.Background()
	s, orig, in, pin := p1CreateFixture(t, true)
	pin.ACL, _ = peerscope.ACLName(in.Project, in.Bridge, p1UUID)
	if e := s.FinalizeVerifiedScopedCreate(ctx, orig.ID, in.Project, in.Name, p1UUID, pin); e != nil {
		t.Fatal(e)
	}
	req, target := a4Request(t, s, orig.ID)
	if !target.ScopedLifecycle {
		t.Fatal(target)
	}
	priorProof := RecoveryProof{Kind: RecoveryNotNeeded, Reason: "ALREADY_TERMINAL"}
	prior, e := s.CommitRecovery(ctx, req, target, priorProof)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("DELETE FROM network_address_fences WHERE box_uuid=? AND family='ipv4'", p1UUID); e != nil {
		t.Fatal(e)
	}
	repair := RecoveryProof{Kind: RecoveryRepairPin, Reason: "CREATE_PIN_LEDGER_REPAIRED", NativeID: target.NativeID, UUID: p1UUID, Fingerprint: "synthetic-exact-pin", CreateBinding: pin}
	snap := DiagnosticSnapshot{store: s}
	if e = snap.CheckPlanRecoveryProof(ctx, target, repair); e != nil {
		t.Fatal(e)
	}
	if got, e := s.CommitRecovery(ctx, req, target, priorProof); e != nil || !got.Receipt.Duplicate || got.Receipt.ID != prior.Receipt.ID {
		t.Fatal(got, e)
	}
	req.Key = "p1-fresh-recovery-refused"
	repair.CreateBinding.IPv4 = "10.243.18.77"
	if e = snap.CheckPlanRecoveryProof(ctx, target, repair); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("conflicting plan", e)
	}
	if _, e = s.CommitRecovery(ctx, req, target, repair); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("conflicting repair", e)
	}
	if _, e = s.ReplayRecovery(ctx, req); !errors.Is(e, ErrNotFound) {
		t.Fatal("partial receipt", e)
	}
	var n int
	if e = s.db.QueryRow("SELECT count(*) FROM recovery_audit WHERE operation_id=?", orig.ID).Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	if e = s.db.QueryRow("SELECT count(*) FROM network_address_fences WHERE box_uuid=?", p1UUID).Scan(&n); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	p1AssertOriginal(t, s, orig.ID, "SUCCEEDED", target.NativeID)
}

func TestP1ScopedModeGenericRecoveryPlanWriterParity(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	req, target := recoveryOriginal(t, s, "box_start", rUUID)
	if target.ScopedPeer || target.ScopedLifecycle {
		t.Fatal("not generic", target)
	}
	proof := recoverProof(target)
	snap := DiagnosticSnapshot{store: s}
	if e := snap.CheckScopedPlanRecoveryProof(ctx, target, proof); e != nil {
		t.Fatal(e)
	}
	blocked := RecoveryProof{Kind: RecoveryBlocked, Reason: "NATIVE_UNVERIFIED"}
	prior, e := s.CommitScopedRecovery(ctx, req, target, blocked)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("UPDATE operations SET updated_at=updated_at+1 WHERE id=?", target.ID); e != nil {
		t.Fatal(e)
	}
	if got, e := s.CommitScopedRecovery(ctx, req, target, blocked); e != nil || !got.Receipt.Duplicate || got.Receipt.ID != prior.Receipt.ID {
		t.Fatal(got, e)
	}
	req.Key = "p1-scoped-generic-recovery-fresh"
	if e = snap.CheckScopedPlanRecoveryProof(ctx, target, proof); !errors.Is(e, ErrConflict) {
		t.Fatal("stale plan", e)
	}
	if _, e = s.CommitScopedRecovery(ctx, req, target, proof); !errors.Is(e, ErrConflict) {
		t.Fatal("stale writer", e)
	}
	if _, e = s.ReplayRecovery(ctx, req); !errors.Is(e, ErrNotFound) {
		t.Fatal("partial receipt", e)
	}
	if got, e := s.ReadRecoveryTarget(ctx, req.Layer, req.OriginalID); e != nil || got.ID != target.ID || got.NativeID != target.NativeID || got.State != target.State {
		t.Fatal("original changed", got, e)
	}
	var n int
	if e = s.db.QueryRow("SELECT count(*) FROM recovery_audit WHERE operation_id=?", target.ID).Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
}
