package state

import (
	"context"
	"strings"
	"testing"
)

func TestActivityIssuesSeparatesLiveReviewReleasedHistoryAndCoverage(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	released := activityAdmit(t, s, tok, "user-1001", "issues-clarity-released-01", "box_exec", strings.Repeat("a", 32), "box:old-release")
	if err := s.Claim(ctx, released.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, released.ID, "DISPATCHING", "UNCERTAIN_NATIVE"); err != nil {
		t.Fatal(err)
	}
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,1,'tester','RELEASED','test')", released.ID)
	if err != nil {
		t.Fatal(err)
	}
	needsReview := activityAdmit(t, s, tok, "user-1001", "issues-clarity-active-02", "box_exec", strings.Repeat("b", 32), "box:still-unknown")
	if err := s.Claim(ctx, needsReview.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, needsReview.ID, "DISPATCHING", "UNCERTAIN_NATIVE"); err != nil {
		t.Fatal(err)
	}
	failed := activityAdmit(t, s, tok, "user-1001", "issues-clarity-failed-03", "box_exec", strings.Repeat("c", 32), "box:failed-pre-dispatch")
	if err := s.Claim(ctx, failed.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.FailPreDispatch(ctx, failed.ID, "PROVED_NO_NATIVE_EFFECT"); err != nil {
		t.Fatal(err)
	}
	view, err := s.ReadActivity(ctx, ActivityQuery{Layer: "user-1001", Requester: tok.ID, View: "issues", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Issues) != 1 || view.Issues[0].OperationID != needsReview.ID ||
		view.Issues[0].Disposition != "HOST_REVIEW_REQUIRED" {
		t.Fatalf("missing current review item: %+v", view)
	}
	if len(view.History) != 2 {
		t.Fatalf("historical sample did not include failed and released UNKNOWN: %+v", view)
	}
	got := map[string]string{}
	for _, item := range view.History {
		got[item.OperationID] = item.Disposition
	}
	if got[released.ID] != "CLAIM_RELEASED_NATIVE_OUTCOME_UNKNOWN" ||
		got[failed.ID] != "TERMINAL_FAILED_NOT_ACTIVE" {
		t.Fatalf("historical outcome accidentally marked resolved/safe: %+v", view.History)
	}
	c := view.CoverageDetails
	if c == nil || !c.AuthorizedProjectOnly || !c.SQLiteSnapshotOnly ||
		c.NativeLiveVerified || c.AttentionComplete || c.CrossPageSnapshotStable ||
		!c.HistorySampleBounded || view.AttentionComplete {
		t.Fatalf("bounded SQLite scope incorrectly claimed complete/native: %+v", view)
	}
	_, err = s.db.ExecContext(ctx,
		"INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,2,'tester','RELEASED','test')", needsReview.ID)
	if err != nil {
		t.Fatal(err)
	}
	none, err := s.ReadActivity(ctx, ActivityQuery{Layer: "user-1001", Requester: tok.ID, View: "issues"})
	if err != nil || len(none.Issues) != 0 || len(none.History) != 3 || none.AttentionComplete {
		t.Fatalf("released claim must no longer block, without erasing UNKNOWN history: %+v %v", none, err)
	}
}
