package state

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
			WorkRef:    "project-x/test-cycle", AgentRef: agent,
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
		WorkRef:    "project-x/test-cycle", AgentRef: "agent-a",
	})
	if err != nil || !duplicate.Duplicate || duplicate.ID != first.ID ||
		duplicate.TrackingID != first.TrackingID || duplicate.AgentRef != "agent-a" {
		t.Fatalf("original correlation receipt lost on retry: %+v %v", duplicate, err)
	}
	// A legacy client may retry without optional work/Agent metadata.
	// It must receive the original immutable attribution and safe next read.
	legacy, err := s.Admit(ctx, Admission{
		Layer: "L0", ActorID: a.ID, TokenID: a.ID, Key: "s2-original-intent-0001",
		RequestID: "legacy-transport-retry", Action: "box_exec",
		SafeIntent: []byte(`{"sanitized":"guest-exec"}`),
	})
	if err != nil || !legacy.Duplicate || legacy.ID != first.ID ||
		legacy.WorkRef != first.WorkRef || legacy.AgentRef != first.AgentRef ||
		legacy.Action != "box_exec" || legacy.NextAction.Kind != "READ_OPERATION" {
		t.Fatalf("legacy retry erased original evidence: %+v %v", legacy, err)
	}
	_, err = s.Admit(ctx, Admission{
		Layer: "L0", ActorID: a.ID, TokenID: a.ID, Key: "s2-original-intent-0001",
		RequestID: "relabel-attempt", Action: "box_exec",
		SafeIntent: []byte(`{"sanitized":"guest-exec"}`),
		WorkRef:    "project-x/test-cycle", AgentRef: "spoofed",
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

func TestS2OperationCursorPreservesFirstPageAdmissionWatermarkAcrossWriter(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	raw, actor, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil || raw == "" {
		t.Fatal("missing durable test identity", err)
	}
	admit := func(i int) string {
		t.Helper()
		r, err := s.Admit(ctx, Admission{
			Layer: "L0", ActorID: actor.ID, TokenID: actor.ID,
			Key:    fmt.Sprintf("history-stable-original-%03d", i),
			Action: "box_exec", RequestID: fmt.Sprintf("stable-%03d", i),
			SafeIntent: []byte(fmt.Sprintf("{\"i\":%d}", i)),
			WorkRef:    "lab/stable-cursor", AgentRef: "writer",
		})
		if err != nil || r.Duplicate {
			t.Fatalf("durable original #%d: %+v %v", i, r, err)
		}
		return r.ID
	}
	originals := make(map[string]bool, 5)
	for i := 0; i < 5; i++ {
		originals[admit(i)] = true
	}
	q := OperationQuery{
		Layer: "L0", Requester: actor.ID, Role: "viewer",
		WorkRef: "lab/stable-cursor", Limit: 2,
	}
	first, err := s.ListOperations(ctx, q)
	if err != nil || len(first.Operations) != 2 || first.NextCursor == "" {
		t.Fatalf("first snapshot page failed: %+v %v", first, err)
	}
	// A second writer inserts after the original cursor was issued.
	// A page's signed bound must not include these later rows.
	added := make(map[string]bool, 3)
	for i := 5; i < 8; i++ {
		added[admit(i)] = true
	}
	seen := make(map[string]bool)
	for _, r := range first.Operations {
		if !originals[r.ID] {
			t.Fatalf("new row unexpectedly in first page: %s", r.ID)
		}
		seen[r.ID] = true
	}
	q.Cursor = first.NextCursor
	var cursorID string
	for page := 0; page < 2; page++ {
		next, err := s.ListOperations(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		want := 2
		if page == 1 {
			want = 1
		}
		if len(next.Operations) != want {
			t.Fatalf("snapshot page %d changed size: %+v", page+2, next)
		}
		for _, r := range next.Operations {
			if !originals[r.ID] || seen[r.ID] || added[r.ID] {
				t.Fatalf("concurrent admission entered old cursor walk: %+v", r)
			}
			seen[r.ID] = true
		}
		cursorID = q.Cursor
		q.Cursor = next.NextCursor
	}
	if len(seen) != 5 || q.Cursor != "" {
		t.Fatalf("old cursor lost/duplicated rows, len=%d cursor=%q", len(seen), q.Cursor)
	}
	// A fresh cursor now traverses all eight committed rows, including
	// three that could not appear in the original five-row cursor.
	q.Cursor = ""
	freshSeen := make(map[string]bool)
	newCount := 0
	for page := 0; page < 4; page++ {
		fresh, err := s.ListOperations(ctx, q)
		if err != nil || len(fresh.Operations) != 2 {
			t.Fatalf("fresh pagination page %d: %+v %v", page, fresh, err)
		}
		for _, r := range fresh.Operations {
			if freshSeen[r.ID] {
				t.Fatalf("fresh cursor duplicated operation %s", r.ID)
			}
			freshSeen[r.ID] = true
			if added[r.ID] {
				newCount++
			}
		}
		q.Cursor = fresh.NextCursor
	}
	if len(freshSeen) != 8 || newCount != 3 || q.Cursor != "" {
		t.Fatalf("fresh cursor failed to see all 8 (3 new) rows: count=%d new=%d cursor=%q",
			len(freshSeen), newCount, q.Cursor)
	}
	// HMAC also binds the first-page upper rowid so a client cannot
	// force a later snapshot using a foreign/mutated anchor.
	data, err := base64.RawURLEncoding.DecodeString(cursorID)
	if err != nil {
		t.Fatal(err)
	}
	var tampered operationCursor
	if err := json.Unmarshal(data, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered.MaxRowID++
	data, err = json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	q.Cursor = base64.RawURLEncoding.EncodeToString(data)
	if _, err := s.ListOperations(ctx, q); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unsigned rowid ceiling escalation accepted: %v", err)
	}
}

func TestS2CursorStatusChangeDuringPagingKeepsIdentityAndFilter(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, actor, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		rec, err := s.Admit(ctx, Admission{
			Layer: "L0", ActorID: actor.ID, TokenID: actor.ID,
			Key:    fmt.Sprintf("s2-live-state-cursor-%02d", i),
			Action: "box_exec", RequestID: fmt.Sprintf("req-%02d", i),
			SafeIntent: []byte(fmt.Sprintf(`{"seq":%d}`, i)),
			WorkRef:    "lab/live-state-pagination", AgentRef: "probe",
		})
		if err != nil || rec.State != "ADMITTED" {
			t.Fatalf("failed original admission %d: %+v %v", i, rec, err)
		}
		ids = append(ids, rec.ID)
	}
	q := OperationQuery{
		Layer: "L0", Requester: actor.ID, Role: "viewer",
		WorkRef: "lab/live-state-pagination", State: "ADMITTED", Limit: 2,
	}
	first, err := s.ListOperations(ctx, q)
	if err != nil || len(first.Operations) != 2 || first.NextCursor == "" {
		t.Fatalf("original in-flight page: %+v %v", first, err)
	}
	seen := map[string]bool{}
	for _, rec := range first.Operations {
		if rec.State != "ADMITTED" || rec.ID == ids[4] {
			t.Fatalf("first page unexpectedly included the oldest later-changed row: %+v", rec)
		}
		seen[rec.ID] = true
	}
	// Independently change a not-yet-visited originally ADMITTED row to a
	// durable terminal state. The old signed cursor may not duplicate rows
	// or return an out-of-filter receipt.
	if err := s.CancelQueued(ctx, "L0", ids[4]); err != nil {
		t.Fatal(err)
	}
	q.Cursor = first.NextCursor
	second, err := s.ListOperations(ctx, q)
	if err != nil || len(second.Operations) != 2 || second.NextCursor != "" {
		t.Fatalf("continuation across status transition: %+v %v", second, err)
	}
	for _, rec := range second.Operations {
		if rec.State != "ADMITTED" || rec.ID == ids[4] || seen[rec.ID] {
			t.Fatalf("status-changing cursor duplicated/leaked receipt: %+v", rec)
		}
		seen[rec.ID] = true
	}
	if len(seen) != 4 {
		t.Fatalf("stable original cursor lost rows: %d", len(seen))
	}
	q.Cursor = ""
	q.State = "CANCELLED"
	cancelled, err := s.ListOperations(ctx, q)
	if err != nil || len(cancelled.Operations) != 1 || cancelled.Operations[0].ID != ids[4] {
		t.Fatalf("fresh state-filtered cursor omitted newly terminal original: %+v %v", cancelled, err)
	}
	// An unrelated actor may have an identical untrusted work_ref, but must
	// never borrow the original row or signed cursor.
	_, other, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	q.Requester = other.ID
	foreign, err := s.ListOperations(ctx, q)
	if err != nil || len(foreign.Operations) != 0 {
		t.Fatalf("unrelated actor borrowed a status-changing original: %+v %v", foreign, err)
	}
}
