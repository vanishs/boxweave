package state

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestS5A5RecoveryNavigationExactLinksAndBoundedLegacy(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	r, target := recoveryOriginal(t, s, "box_start", rUUID)
	before, e := s.ReadRecoveryNavigation(ctx, r.Layer, target.ID)
	if e != nil || before.ClaimProtection != "RETAINED" || before.NextAction.Tool != "operation_recover" || before.NextAction.Mode != "inspect" || before.NativeLiveVerified {
		t.Fatalf("before %+v %v", before, e)
	}
	blocked, e := s.CommitRecovery(ctx, r, target, RecoveryProof{Kind: RecoveryBlocked, Reason: "TEST_MISSING_EVIDENCE"})
	if e != nil {
		t.Fatal(e)
	}
	// A pre-A5 receipt is still navigable via its canonical stored intent.
	if _, e = s.db.Exec("UPDATE operations SET resource_key='' WHERE id=?", blocked.Receipt.ID); e != nil {
		t.Fatal(e)
	}
	v, e := s.ReadRecoveryNavigation(ctx, r.Layer, target.ID)
	if e != nil || len(v.Recent) != 1 || v.Recent[0].OperationID != blocked.Receipt.ID || v.Recent[0].Disposition != "BLOCKED" || v.ClaimProtection != "RETAINED" {
		t.Fatalf("legacy link %+v %v", v, e)
	}
	back, e := s.ReadRecoveryNavigation(ctx, r.Layer, blocked.Receipt.ID)
	if e != nil || back.Original == nil || back.Original.OriginalID != target.ID || back.NextAction.OperationID != target.ID {
		t.Fatalf("reverse link %+v %v", back, e)
	}
	// Future receipts use the exact existing index without new DDL.
	r.Key = "a5-recovery-settled-key"
	settled, e := s.CommitRecovery(ctx, r, target, recoverProof(target))
	if e != nil {
		t.Fatal(e)
	}
	v, e = s.ReadRecoveryNavigation(ctx, r.Layer, target.ID)
	if e != nil || v.Disposition != "RECOVERY_SETTLED" || v.OriginalState != "SUCCEEDED" || len(v.Recent) != 2 || v.Recent[0].OperationID != settled.Receipt.ID {
		t.Fatalf("settled link %+v %v", v, e)
	}
	for i := 0; i < recoveryLegacyWindow+10; i++ {
		k := fmt.Sprintf("a5-newer-fixture-%04d", i)
		if _, e = s.Admit(ctx, Admission{Layer: r.Layer, ActorID: r.ActorID, TokenID: r.TokenID, Key: k, Action: "operation_cancel", RequestID: k, SafeIntent: []byte("{}")}); e != nil {
			t.Fatal(e)
		}
	}
	v, e = s.ReadRecoveryNavigation(ctx, r.Layer, target.ID)
	if e != nil || !v.LegacyWindowBounded || len(v.Recent) != 1 || v.Recent[0].OperationID != settled.Receipt.ID || v.Disposition != "RECOVERY_SETTLED" {
		t.Fatalf("lost indexed link or missing coverage %+v %v", v, e)
	}
	evidence, e := s.ReadActivity(ctx, ActivityQuery{Layer: r.Layer, Requester: r.TokenID, View: "evidence", OperationID: target.ID})
	if e != nil || evidence.Evidence == nil || !evidence.Truncated || len(evidence.Evidence.Recovery.Recent) != 1 || evidence.Evidence.Recovery.Recent[0].OperationID != settled.Receipt.ID {
		t.Fatalf("evidence %+v %v", evidence, e)
	}
	if _, e = s.ReadRecoveryNavigation(ctx, "foreign", settled.Receipt.ID); !errors.Is(e, ErrNotFound) {
		t.Fatal("cross-layer link", e)
	}
}
func TestS5A5RecoveryNavigationLimitsAndRedaction(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	r, target := recoveryOriginal(t, s, "box_stop", rUUID)
	for i := 0; i < 8; i++ {
		r.Key = fmt.Sprintf("a5-blocked-attempt-%04d", i)
		if _, e := s.CommitRecovery(ctx, r, target, RecoveryProof{Kind: RecoveryBlocked, Reason: "TEST_MISSING_WITNESS"}); e != nil {
			t.Fatal(e)
		}
	}
	v, e := s.ReadRecoveryNavigation(ctx, r.Layer, target.ID)
	if e != nil || len(v.Recent) != 5 || !v.LinksTruncated || v.LegacyWindowBounded {
		t.Fatalf("limit %+v %v", v, e)
	}
	// The recovery-link lookup must not expose raw safe_intent or foreign IDs,
	// even if a malformed historical fixture row is present.
	id := v.Recent[0].OperationID
	if _, e = s.db.Exec("UPDATE operations SET safe_intent=? WHERE id=?", []byte(`{"original_operation_id":"`+target.ID+`","secret":"SENTINEL_PRIVATE"}`), id); e != nil {
		t.Fatal(e)
	}
	back, e := s.ReadRecoveryNavigation(ctx, r.Layer, id)
	if e != nil || back.Original != nil || back.Disposition != "RECOVERY_LINK_UNVERIFIED" {
		t.Fatalf("unsafe parsed link %+v %v", back, e)
	}
}
func TestS5A5BoxIssuesIncludeSiblingProjectBarrier(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	own := activityAdmit(t, s, tok, "user-1001", "a5-peer-unknown-001", "box_peer_set", strings.Repeat("a", 32), "acl:default/owned", "box:"+strings.Repeat("a", 32))
	a4Unknown(t, s, own.ID, false)
	foreign := activityAdmit(t, s, tok, "foreign", "a5-foreign-unknown-001", "box_exec", strings.Repeat("b", 32), "box:foreign")
	a4Unknown(t, s, foreign.ID, false)
	if _, e := s.db.Exec("INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,1,'test','RELEASED','test')", own.ID); e != nil {
		t.Fatal(e)
	}
	q := ActivityQuery{Layer: "user-1001", Requester: tok.ID, View: "issues", BoxUUID: strings.Repeat("b", 32), Limit: 2}
	for _, only := range []bool{false, true} {
		q.BlockersOnly = only
		v, e := s.ReadActivity(ctx, q)
		if e != nil || len(v.Issues) != 1 || v.Issues[0].OperationID != own.ID || v.Issues[0].Reason != "PROJECT_PEER_BARRIER" || v.Issues[0].NextAction == nil || v.Issues[0].NextAction.Mode != "inspect" || v.AttentionComplete {
			t.Fatalf("sibling issue %+v %v", v, e)
		}
	}
	q.WorkRef = "different"
	v, e := s.ReadActivity(ctx, q)
	if e != nil || len(v.Issues) != 0 || v.AttentionComplete {
		t.Fatalf("label scope %+v %v", v, e)
	}
	nav, e := s.ReadRecoveryNavigation(ctx, "user-1001", own.ID)
	if e != nil || nav.ClaimProtection != "RETAINED" || nav.Disposition != "NETWORK_LEGACY_RELEASE_REVIEW_REQUIRED" {
		t.Fatalf("inconsistent protection %+v %v", nav, e)
	}
}
