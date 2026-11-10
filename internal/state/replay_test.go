package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestS2ReplayAfterNativeDeleteOrUnknownWithoutReadback(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, actor, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, other, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	in := Admission{
		Layer: "s2-lab", ActorID: actor.ID, TokenID: actor.ID,
		Key: "original-after-deleted-box-01", Action: "box_delete",
		RequestID: "original", BoxUUID: "fixed-box-uuid",
		Incarnation: "fixed-box-uuid", ResourceKey: "box:old-name",
		SafeIntent: []byte(`{"box_id":"fixed-box-uuid"}`),
		WorkRef:    "s2/work", AgentRef: "agent-first",
	}
	receipt, err := s.Admit(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, receipt.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, receipt.ID, "DISPATCHING", "LOST_ACK"); err != nil {
		t.Fatal(err)
	}
	// No exact native Box is even in this fixture: replay is strictly read-only
	// and returns the locked original UNKNOWN instead of retrying a delete.
	legacy := in
	legacy.RequestID = "different-client"
	legacy.WorkRef, legacy.AgentRef = "", ""
	again, err := s.Replay(ctx, legacy)
	if err != nil || !again.Duplicate || again.ID != receipt.ID ||
		again.TrackingID != receipt.TrackingID || again.State != "UNKNOWN" ||
		again.WorkRef != in.WorkRef || again.AgentRef != in.AgentRef ||
		again.NextAction.Kind != "HOST_LOCAL_REVIEW" {
		t.Fatalf("native-deleted/UNKNOWN replay lost original: %+v %v", again, err)
	}
	legacy.WorkRef = "conflicting/relabel"
	if _, err := s.Replay(ctx, legacy); !errors.Is(err, ErrConflict) {
		t.Fatalf("explicit relabel accepted: %v", err)
	}
	legacy = in
	legacy.ActorID, legacy.TokenID = other.ID, other.ID
	if _, err := s.Replay(ctx, legacy); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign actor inherited original receipt: %v", err)
	}
	legacy = in
	legacy.SafeIntent = []byte(`{"box_id":"another-box"}`)
	if _, err := s.Replay(ctx, legacy); !errors.Is(err, ErrConflict) {
		t.Fatalf("different intent replayed original: %v", err)
	}
	if err := s.RevokeToken(ctx, actor.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Replay(ctx, in); !errors.Is(err, ErrRevoked) {
		t.Fatalf("revoked actor could replay original receipt: %v", err)
	}
}
