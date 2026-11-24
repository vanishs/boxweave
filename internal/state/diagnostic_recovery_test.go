package state

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestPR177D3ScopedSnapshotProofAndRevision(t *testing.T) {
	ctx := context.Background()
	s, ro, id := pr178ScopedReadyToStage(t, true)
	if e := s.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
		t.Fatal(e)
	}
	if e := s.Unknown(ctx, id, "DISPATCHING", "SCOPED_PEER_V2_ROOT_OUTCOME_UNVERIFIED"); e != nil {
		t.Fatal(e)
	}
	// Same read predicates on synthetic isolated fixture; real private capture
	// and CLI source-no-effect checks are separate.
	snap := DiagnosticSnapshot{store: s}
	a, e := snap.ReadOriginal(ctx, "user-1001", id)
	if e != nil || len(a.Claims) != 5 || !a.CredentialActive {
		t.Fatal(a, e)
	}
	if _, e = snap.ReadOriginal(ctx, "user-9999", id); !errors.Is(e, ErrNotFound) {
		t.Fatal("foreign original", e)
	}
	rec := scopedTestReceipt(t, ro, id, true)
	p := RecoveryProof{Kind: RecoverySettle, Reason: "SCOPED_PEER_COMPLETED_NO_REPLAY", UUID: a.Target.UUID, Fingerprint: "synthetic-observation", ScopedPeer: rec}
	if e = snap.CheckPlanRecoveryProof(ctx, a.Target, p); e != nil {
		t.Fatal(e)
	}
	p.ScopedPeer.ProofSHA256 = strings.Repeat("f", 64)
	if e = snap.CheckPlanRecoveryProof(ctx, a.Target, p); e == nil {
		t.Fatal("forged digest accepted")
	}
	p.ScopedPeer = rec
	p.ScopedPeer.RecoveryOnly = false
	if e = snap.CheckPlanRecoveryProof(ctx, a.Target, p); e == nil {
		t.Fatal("dispatch receipt accepted")
	}
	b, e := snap.ReadOriginal(ctx, "user-1001", id)
	if e != nil || a.Digest != b.Digest || b.Target.State != "UNKNOWN" {
		t.Fatal(b, e)
	}
	raw, e := json.Marshal(b)
	if e != nil || strings.Contains(string(raw), "peer-guard:") || strings.Contains(string(raw), "safe_intent") {
		t.Fatal("private evidence leaked", e)
	}
	if _, e = s.db.Exec("UPDATE operation_events SET safe_detail=safe_detail||'-changed' WHERE operation_id=? AND seq=(SELECT min(seq) FROM operation_events WHERE operation_id=?)", id, id); e != nil {
		t.Fatal(e)
	}
	b, e = snap.ReadOriginal(ctx, "user-1001", id)
	if e != nil || a.Digest == b.Digest {
		t.Fatal("same-count history drift ignored", e)
	}
	var token string
	if e = s.db.QueryRow("SELECT token_id FROM operations WHERE id=?", id).Scan(&token); e != nil {
		t.Fatal(e)
	}
	if e = s.RevokeToken(ctx, token); e != nil {
		t.Fatal(e)
	}
	b, e = snap.ReadOriginal(ctx, "user-1001", id)
	if e != nil || b.CredentialActive {
		t.Fatal(b, e)
	}
	p.ScopedPeer = rec
	if e = snap.CheckPlanRecoveryProof(ctx, b.Target, p); e == nil {
		t.Fatal("revoked original accepted")
	}
	t.Log("D3_SCOPED_READ_ONLY_ORIGINAL_HISTORY_AND_PROOF_PASS")
}
