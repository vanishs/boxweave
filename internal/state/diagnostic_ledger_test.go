package state

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/networklimits"
)

func TestPR177D2ScopedRetiredAndLegacyRoutes(t *testing.T) {
	ctx := context.Background()
	s, b, _ := n3RetiredFixture(t)
	// This is a stationary synthetic fixture, not a live source capture.
	snap := DiagnosticSnapshot{store: s}
	l, e := snap.ReadLedger(ctx, b.Project, "SCOPED_V2", "incusbr-1001")
	if e != nil || l.LegacyRetirement != nil || l.PeerCapacity == nil || l.RetiredIdentities != 1 || !l.HoldsObserved || l.HeldIPv4 != 3 || l.HeldIPv6 != 3 || l.HeldMAC != 3 {
		t.Fatal(l, e)
	}
	if l.Network.OrphanBindings != 0 || l.Network.OrphanFences != 0 {
		t.Fatal(l.Network)
	}
	legacy, e := snap.ReadLedger(ctx, b.Project, "LEGACY", "incusbr-1001")
	if e != nil || legacy.LegacyRetirement != nil || legacy.PeerCapacity != nil {
		t.Fatal(legacy, e)
	}
	unknown, e := snap.ReadLedger(ctx, b.Project, "UNVERIFIED", "incusbr-1001")
	if e != nil || unknown.LegacyRetirement != nil || unknown.PeerCapacity != nil {
		t.Fatal(unknown, e)
	}
	t.Log("D2_RETIRED_MODE_ROUTING_PASS")
}
func TestPR177D2ScopedUnknownAndFailedRecoveryHistory(t *testing.T) {
	ctx := context.Background()
	s, _, id := pr178ScopedReadyToStage(t, true)
	if e := s.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
		t.Fatal(e)
	}
	if e := s.Unknown(ctx, id, "DISPATCHING", "SCOPED_PEER_V2_ROOT_OUTCOME_UNVERIFIED"); e != nil {
		t.Fatal(e)
	}
	target, e := s.ReadRecoveryTarget(ctx, "user-1001", id)
	if e != nil {
		t.Fatal(e)
	}
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	req := RecoveryRequest{Layer: target.Layer, ActorID: tok.ID, TokenID: tok.ID, OriginalID: id, Key: "d2-blocked-original-recovery"}
	blocked, e := s.CommitRecovery(ctx, req, target, RecoveryProof{Kind: RecoveryBlocked, Reason: "SCOPED_PEER_ROOT_PROOF_REQUIRED"})
	if e != nil {
		t.Fatal(e)
	}
	snap := DiagnosticSnapshot{store: s}
	l, e := snap.ReadLedger(ctx, target.Layer, "SCOPED_V2", purePeerBridge)
	if e != nil {
		t.Fatal(e)
	}
	if len(l.Active) != 1 || l.Active[0].ID != id || l.Active[0].RecoveryRoute != "MCP_RECOVERY_INSPECTION_REQUIRED" || l.Active[0].LiveClaims != 5 || l.Active[0].NativeReceipt || l.Active[0].Cause != "ROOT_OUTCOME_UNVERIFIED" {
		t.Fatal(l.Active)
	}
	if l.HistoricalRecoveryAttempts != 1 || len(l.HistorySample) != 1 || l.HistorySample[0].ID != blocked.Receipt.ID || l.HistorySample[0].RecoveryOriginalID != id {
		t.Fatal(l.HistorySample)
	}
	if l.Activity.NextCursor != "" || l.Activity.AttentionComplete {
		t.Fatal("local diagnostic forged MCP cursor")
	}
	other, e := snap.ReadLedger(ctx, "user-9999", "UNVERIFIED", "")
	if e != nil || other.ActiveCount != 0 || other.HistoricalFailures != 0 {
		t.Fatal(other, e)
	}
	t.Log("D2_UNKNOWN_RETAINED_BLOCKED_ATTEMPT_HISTORY_PASS")
}
func TestPR177D2CapacityAndRedactedOAuth(t *testing.T) {
	ctx := context.Background()
	s, _, id := pr178ScopedReadyToStage(t, true)
	pr178SeedPolicyHistory(t, s, id, networklimits.ScopedPeerHistoryLimit-2)
	p, e := s.ReadDiagnosticPeerCapacity(ctx, "user-1001")
	if e != nil || p.NewGrantHistoryAvailable || p.CompletedLedgerEvents != networklimits.ScopedPeerHistoryLimit-2 || p.DirectionLimit != 32 {
		t.Fatal(p, e)
	}
	if _, e = s.db.Exec("UPDATE oauth_config SET public_origin='https://private.example',password_hash='secret-password-hash' WHERE id=1"); e != nil {
		t.Fatal(e)
	}
	snap := DiagnosticSnapshot{store: s}
	l, e := snap.ReadLedger(ctx, "user-1001", "SCOPED_V2", purePeerBridge)
	if e != nil || !l.OAuth.Enabled {
		t.Fatal(l, e)
	}
	raw, e := json.Marshal(l)
	if e != nil || strings.Contains(string(raw), "private.example") || strings.Contains(string(raw), "secret-password-hash") || strings.Contains(string(raw), purePeerBridge) || strings.Contains(string(raw), "safe_intent") {
		t.Fatal("redaction failed", e)
	}
	t.Log("D2_CAPACITY_RESERVE_AND_OAUTH_REDACTION_PASS")
}

func TestPR177D2ReleasedHistoryAndGuestExitSeparate(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	released := activityAdmit(t, s, tok, "user-1001", "d2-released-fixture-0001", "box_exec", strings.Repeat("a", 32), "box:old")
	if e := s.Claim(ctx, released.ID); e != nil {
		t.Fatal(e)
	}
	if e := s.Unknown(ctx, released.ID, "DISPATCHING", "UNCERTAIN_NATIVE"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec("INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,1,'test','RELEASED','test')", released.ID); e != nil {
		t.Fatal(e)
	}
	completed := activityAdmit(t, s, tok, "user-1001", "d2-completed-fixture-0002", "box_exec", strings.Repeat("b", 32), "box:done")
	// Synthetic durable receipt only, no guest effect or output availability claim.
	if _, e := s.db.Exec("UPDATE operations SET state='SUCCEEDED',native_operation_id='fixture' WHERE id=?", completed.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec("INSERT INTO exec_outputs(operation_id,stdout_path,exit_code,observed_at) VALUES(?,'private-output-path',7,1)", completed.ID); e != nil {
		t.Fatal(e)
	}
	snap := DiagnosticSnapshot{store: s}
	l, e := snap.ReadLedger(ctx, "user-1001", "UNVERIFIED", "")
	if e != nil || l.ActiveCount != 0 || l.ReleasedUnknown != 1 || len(l.Active) != 0 || l.GuestExecNonzeroCount != 1 || len(l.GuestExecNonzeroSample) != 1 {
		t.Fatal(l, e)
	}
	d := l.GuestExecNonzeroSample[0]
	if d.State != "SUCCEEDED" || d.GuestOutcome != "EXIT_NONZERO" || d.GuestExitCode == nil || *d.GuestExitCode != 7 {
		t.Fatal(d)
	}
	b, e := json.Marshal(l)
	if e != nil || strings.Contains(string(b), "private-output-path") {
		t.Fatal("output path leaked", e)
	}
}
