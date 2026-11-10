package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestS2CorrelationOriginalReceiptAndPrivateCursor(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, a, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, b, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	_, viewer, err := s.IssueToken(ctx, "viewer", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	admit := func(actor Token, key, agent string) Receipt {
		t.Helper()
		r, err := s.Admit(ctx, Admission{
			Layer: "L0", ActorID: actor.ID, TokenID: actor.ID,
			Key: key, RequestID: "original", Action: "box_exec",
			SafeIntent: []byte(`{"sanitized":"guest-exec"}`),
			WorkRef: "project-x/test-cycle", AgentRef: agent,
		})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	first := admit(a, "s2-original-intent-0001", "agent-a")
	second := admit(a, "s2-original-intent-0002", "agent-b")
	foreign := admit(b, "s2-original-intent-0001", "agent-b")

	duplicate, err := s.Admit(ctx, Admission{
		Layer: "L0", ActorID: a.ID, TokenID: a.ID, Key: "s2-original-intent-0001",
		RequestID: "transport-retry", Action: "box_exec",
		SafeIntent: []byte(`{"sanitized":"guest-exec"}`),
		WorkRef: "project-x/test-cycle", AgentRef: "agent-a",
	})
	if err != nil || !duplicate.Duplicate || duplicate.ID != first.ID ||
		duplicate.TrackingID != first.TrackingID || duplicate.AgentRef != "agent-a" {
		t.Fatalf("original correlation receipt lost on retry: %+v %v", duplicate, err)
	}
	_, err = s.Admit(ctx, Admission{
		Layer: "L0", ActorID: a.ID, TokenID: a.ID, Key: "s2-original-intent-0001",
		RequestID: "relabel-attempt", Action: "box_exec",
		SafeIntent: []byte(`{"sanitized":"guest-exec"}`),
		WorkRef: "project-x/test-cycle", AgentRef: "spoofed",
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("relabel original key must conflict: %v", err)
	}
	for _, bad := range []string{"hello world", "../etc", "https://example", "a:b", "secret\n", "a\\b", "x/../y"} {
		if ValidCorrelationRef(bad) {
			t.Fatalf("unsafe work/agent label accepted: %q", bad)
		}
	}
	if _, err := s.Admit(ctx, Admission{
		Layer: "L0", ActorID: a.ID, TokenID: a.ID, Key: "s2-bad-label-0000001",
		RequestID: "invalid", Action: "test", SafeIntent: []byte("{}"),
		WorkRef: "hidden\nsecret",
	}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid correlation should refuse before mutation: %v", err)
	}

	q := OperationQuery{Layer: "L0", Requester: a.ID, Role: "viewer",
		WorkRef: "project-x/test-cycle", Limit: 1}
	p1, err := s.ListOperations(ctx, q)
	if err != nil || len(p1.Operations) != 1 || p1.NextCursor == "" {
		t.Fatalf("first page: %+v %v", p1, err)
	}
	q.Cursor = p1.NextCursor
	p2, err := s.ListOperations(ctx, q)
	if err != nil || len(p2.Operations) != 1 || p2.NextCursor != "" {
		t.Fatalf("second page: %+v %v", p2, err)
	}
	if p1.Operations[0].ID == p2.Operations[0].ID {
		t.Fatal("pagination repeated original record")
	}
	q.AgentRef = "agent-a"
	if _, err := s.ListOperations(ctx, q); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-filter cursor reused: %v", err)
	}
	q.AgentRef = ""
	q.Requester = b.ID
	if _, err := s.ListOperations(ctx, q); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-actor cursor reused: %v", err)
	}
	q.Cursor = "invalid-or-tampered"
	if _, err := s.ListOperations(ctx, q); !errors.Is(err, ErrInvalid) {
		t.Fatalf("forged cursor: %v", err)
	}
	all, err := s.ListOperations(ctx, OperationQuery{Layer: "L0", Requester: a.ID, Role: "admin", Limit: 20})
	if err != nil || len(all.Operations) != 3 {
		t.Fatalf("admin own-controller catalog: %+v %v", all, err)
	}
	own, err := s.ListOperations(ctx, OperationQuery{Layer: "L0", Requester: a.ID, Role: "viewer"})
	if err != nil || len(own.Operations) != 2 {
		t.Fatalf("viewer cross-actor leak: %+v %v", own, err)
	}
	if _, err := s.GetScopedOperation(ctx, "L0", a.ID, "viewer", foreign.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("guessed foreign ID leaked: %v", err)
	}
	if _, err := s.GetScopedOperation(ctx, "L1", a.ID, "admin", first.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-layer operation leaked: %v", err)
	}
	if _, err := s.GetScopedOperation(ctx, "L0", viewer.ID, "viewer", second.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("different viewer accessed output owner: %v", err)
	}
}

func TestS2FreshSchemaCrashUnknownAndLabelsPersist(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "fresh-s2.sqlite")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, actor, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	a := Admission{Layer: "L1", ActorID: actor.ID, TokenID: actor.ID,
		Key: "s2-persistent-original-key", Action: "box_create",
		SafeIntent: []byte(`{"sanitized":"create"}`), RequestID: "first",
		WorkRef: "lab/crash-recovery", AgentRef: "runner-1"}
	r, err := s.Admit(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.NativeAck(ctx, r.ID, "original-native-op-id"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if n, err := reopened.Recover(ctx); err != nil || n != 1 {
		t.Fatalf("crash recovery: count=%d err=%v", n, err)
	}
	got, err := reopened.GetScopedOperation(ctx, "L1", actor.ID, "viewer", r.ID)
	if err != nil || got.State != "UNKNOWN" || got.NativeOperationID != "original-native-op-id" ||
		got.WorkRef != a.WorkRef || got.AgentRef != a.AgentRef ||
		got.NextAction.Kind != "HOST_LOCAL_REVIEW" {
		t.Fatalf("failed original evidence recovery: %+v err=%v", got, err)
	}
	a.RequestID = "client-after-crash"
	again, err := reopened.Admit(ctx, a)
	if err != nil || !again.Duplicate || again.ID != r.ID ||
		again.TrackingID != r.TrackingID {
		t.Fatalf("retransmitted native create risk: %+v %v", again, err)
	}
	page, err := reopened.ListOperations(ctx, OperationQuery{
		Layer: "L1", Role: "viewer", Requester: actor.ID, WorkRef: a.WorkRef,
	})
	if err != nil || len(page.Operations) != 1 || page.Operations[0].State != "UNKNOWN" {
		t.Fatalf("recovered indexed history lost: %+v %v", page, err)
	}
}
