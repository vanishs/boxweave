package state

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestActivityIssuesCoverPeerForwardFenceAndRealBoxUUID(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	box := "36e79888-4f9e-4afd-a22d-058f95f071c9"
	target := "09df6231-a3cb-4ccb-83e7-b735f074846d"
	if _, err := s.db.ExecContext(ctx, "INSERT INTO peer_grants(project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status) VALUES(?,?,?,?,?,1,'CONFIRMED')",
		"user-1001", target, target, box, box); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv4','10.99.2.3',?,?,?,'RESERVED',42)",
		"user-1001", box, box); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO forward_claims(forward_id,project,box_uuid,incarnation,native_device,protocol,host_bind,host_port,guest_port,guest_ip,state) VALUES('fwd1',?,?,?,'px1','tcp','127.0.0.1',31000,8080,'10.99.2.3','PENDING')",
		"user-1001", box, box); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO forward_claims(forward_id,project,box_uuid,incarnation,native_device,protocol,host_bind,host_port,guest_port,guest_ip,state) VALUES('other1',?,?,?,'px2','tcp','127.0.0.1',31001,8080,'10.99.2.4','UNKNOWN')",
		"foreign", box, box); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	cursor := ""
	for i := 0; i < 4; i++ {
		page, err := s.ReadActivity(ctx, ActivityQuery{
			Layer: "user-1001", Requester: tok.ID, View: "issues",
			BoxUUID: box, Limit: 1, Cursor: cursor,
		})
		if err != nil || len(page.Issues) != 1 {
			t.Fatalf("page %d: %+v %v", i, page, err)
		}
		reason := page.Issues[0].Reason
		if seen[reason] {
			t.Fatalf("duplicate across signed pages: %s", reason)
		}
		seen[reason] = true
		cursor = page.NextCursor
		if cursor == "" {
			if page.Truncated {
				t.Fatal("truncated result without next cursor")
			}
			break
		}
		if !page.Truncated {
			t.Fatal("next cursor without lookahead evidence")
		}
	}
	if len(seen) != 3 || !seen["PEER_ACTIVE_DEPENDENCY"] ||
		!seen["ADDRESS_FENCE_DEPENDENCY"] || !seen["FORWARD_DEPENDENCY"] {
		t.Fatalf("network blockers lost: %+v", seen)
	}
	foreign, err := s.ReadActivity(ctx, ActivityQuery{
		Layer: "foreign-empty", Requester: tok.ID, View: "issues", BoxUUID: box,
	})
	if err != nil || len(foreign.Issues) != 0 {
		t.Fatalf("foreign Layer leaked network ledger: %+v %v", foreign, err)
	}
}

func TestActivityOverviewCursorAdmissionWatermarkAndTamper(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	first := activityAdmit(t, s, tok, "user-1001", "activity-page-original-0001", "box_exec", strings.Repeat("a", 32), "box:first")
	second := activityAdmit(t, s, tok, "user-1001", "activity-page-original-0002", "box_exec", strings.Repeat("b", 32), "box:second")
	query := ActivityQuery{Layer: "user-1001", Requester: tok.ID, View: "overview", Limit: 1}
	page, err := s.ReadActivity(ctx, query)
	if err != nil || len(page.Operations) != 1 || !page.Truncated || page.NextCursor == "" {
		t.Fatalf("overview first page incomplete: %+v %v", page, err)
	}
	third := activityAdmit(t, s, tok, "user-1001", "activity-page-original-0003", "box_exec", strings.Repeat("c", 32), "box:third")
	query.Cursor = page.NextCursor
	secondPage, err := s.ReadActivity(ctx, query)
	if err != nil || len(secondPage.Operations) != 1 || secondPage.Truncated {
		t.Fatalf("overview second page failed: %+v %v", secondPage, err)
	}
	ids := map[string]bool{page.Operations[0].ID: true, secondPage.Operations[0].ID: true}
	if !ids[first.ID] || !ids[second.ID] || ids[third.ID] {
		t.Fatalf("new admission crossed signed watermark: %+v", ids)
	}
	query.WorkRef = "different"
	if _, err := s.ReadActivity(ctx, query); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cursor changed filter accepted: %v", err)
	}
	query.WorkRef = ""
	query.Layer = "foreign"
	if _, err := s.ReadActivity(ctx, query); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cursor crossed Layer: %v", err)
	}
	query.Layer = "user-1001"
	query.Cursor = query.Cursor[:len(query.Cursor)-1] + "!"
	if _, err := s.ReadActivity(ctx, query); !errors.Is(err, ErrInvalid) {
		t.Fatalf("tampered cursor accepted: %v", err)
	}
}

func TestActivityIssuesSameTimestampKeysetAndScope(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	for _, addr := range []string{"10.7.0.1", "10.7.0.2", "10.7.0.3"} {
		if _, err := s.db.ExecContext(ctx, "INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES('bwbr0','ipv4',?,'user-1001','same-box','same-box','RESERVED',1)", addr); err != nil {
			t.Fatal(err)
		}
	}
	cursor := ""
	seen := map[string]bool{}
	for i := 0; i < 4; i++ {
		result, err := s.ReadActivity(ctx, ActivityQuery{
			Layer: "user-1001", Requester: tok.ID, View: "issues", Limit: 1, Cursor: cursor,
		})
		if err != nil || len(result.Issues) != 1 {
			t.Fatalf("issue cursor: %+v %v", result, err)
		}
		key := result.Issues[0].ResourceRef
		if seen[key] {
			t.Fatalf("duplicate across keyset pages: %s", key)
		}
		seen[key] = true
		if i == 0 {
			if _, err := s.ReadActivity(ctx, ActivityQuery{Layer: "user-1001", Requester: tok.ID,
				View: "issues", Limit: 1, WorkRef: "different", Cursor: result.NextCursor}); !errors.Is(err, ErrInvalid) {
				t.Fatalf("cross-filter cursor replay accepted: %v", err)
			}
		}
		cursor = result.NextCursor
		if cursor == "" {
			break
		}
	}
	if len(seen) != 3 {
		t.Fatalf("lost issue in keyset walk: %+v", seen)
	}
}
