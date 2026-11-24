package state

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPR178N3InstalledAdmissionReplayAndRecovery(t *testing.T) {
	ctx := context.Background()
	s, _, _ := n3RetiredFixture(t)
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	a := Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID, Key: "n3-installed-admit-test", RequestID: "n3-admit"}
	src, dst := "11111111-1111-4111-8111-000000000002", "11111111-1111-4111-8111-000000000003"
	r, e := s.AdmitScopedPeerSet(ctx, a, src, dst, true)
	if e != nil {
		t.Fatal(e)
	}
	prior, ok, e := s.ReplayScopedPeerSet(ctx, a.Layer, a.ActorID, a.TokenID, a.Key, src, dst, true, "", "")
	if e != nil || !ok || prior.ID != r.ID {
		t.Fatal(prior, ok, e)
	}
	if _, ok, e = s.ReplayScopedPeerSet(ctx, a.Layer, a.ActorID, a.TokenID, a.Key, src, dst, false, "", ""); ok || e == nil {
		t.Fatal("different direction replay", ok, e)
	}
	second := a
	second.Key = "n3-installed-second-claim"
	if _, e = s.AdmitScopedPeerSet(ctx, second, dst, src, true); e == nil {
		t.Fatal("concurrent peer admitted")
	}
	if e = s.StageAndClaimScopedPeerOriginal(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.Unknown(ctx, r.ID, "DISPATCHING", "TEST_LOST_ACK"); e != nil {
		t.Fatal(e)
	}
	target, e := s.ReadRecoveryTarget(ctx, a.Layer, r.ID)
	if e != nil || !target.ScopedPeer {
		t.Fatal(target, e)
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	proof, e := readScopedPeerOriginalTx(ctx, tx, r.ID, a.Layer, "incusbr-1001", "UNKNOWN")
	tx.Rollback()
	if e != nil {
		t.Fatal(e)
	}
	// Synthetic receipt deliberately proves only the SQLite commit boundary.
	rr := ScopedPeerRootReceipt{OriginalID: r.ID, Project: a.Layer, Bridge: "incusbr-1001", Status: "COMPLETE_VERIFIED", ProofSHA256: ScopedPeerProofSHA256(proof), PlanSHA256: strings.Repeat("a", 64), FinalSHA256: strings.Repeat("b", 64), VerifiedSteps: 4, RecoveryOnly: true}
	request := RecoveryRequest{Layer: a.Layer, ActorID: a.ActorID, TokenID: a.TokenID, OriginalID: r.ID, Key: "n3-installed-recover-key"}
	p := RecoveryProof{Kind: RecoverySettle, Reason: "SCOPED_PEER_COMPLETED_NO_REPLAY", Fingerprint: "fixture", ScopedPeer: rr}
	result, e := s.CommitRecovery(ctx, request, target, p)
	if e != nil || result.Disposition != "SETTLED" {
		t.Fatal(result, e)
	}
	got, e := s.Get(ctx, r.ID)
	if e != nil || got.State != "SUCCEEDED" {
		t.Fatal(got, e)
	}
	old, e := s.ReplayRecovery(ctx, request)
	if e != nil || !old.Receipt.Duplicate || old.Receipt.ID != result.Receipt.ID {
		t.Fatal(old, e)
	}
}
