package state

import (
	"context"
	"errors"
	"github.com/vanishs/boxweave/internal/networklimits"
	"strings"
	"testing"
)

func TestN3NativeLedgerFullScopeAndCorruption(t *testing.T) {
	ctx := context.Background()
	s, a, c, p, tgt := n3OwnedCreateForRepair(t)
	snap, e := s.ReadNativeNetworkLedger(ctx, a.Layer)
	if e != nil || len(snap.Boxes) != 1 || snap.OrphanBindings != 0 || snap.OrphanFences != 0 || snap.Unsettled != 0 {
		t.Fatal(snap, e)
	}
	b := snap.Boxes[0]
	if b.UUID != tgt.UUID || b.Name != "alpha" || b.OriginalID != c.ID || b.OriginalCount != 1 ||
		b.OriginalState != "SUCCEEDED" || b.OriginalResource != "box:alpha" || b.BindingCount != 1 ||
		b.BindingState != "PINNED" || b.FenceCount != 2 || b.FenceIPv4 != 1 || b.FenceIPv6 != 1 ||
		b.IPv4 != p.IPv4 || b.IPv6 != p.IPv6 {
		t.Fatal("lost full exact audit tuple", b)
	}
	if _, e = s.db.Exec("DELETE FROM network_address_fences WHERE box_uuid=? AND family='ipv6'", tgt.UUID); e != nil {
		t.Fatal(e)
	}
	snap, e = s.ReadNativeNetworkLedger(ctx, a.Layer)
	if e != nil || snap.Boxes[0].FenceCount != 1 || snap.Boxes[0].FenceIPv6 != 0 {
		t.Fatal("partial address fence hidden", snap, e)
	}
	if _, e = s.db.Exec("UPDATE owned_boxes SET state='DELETED' WHERE uuid=?", tgt.UUID); e != nil {
		t.Fatal(e)
	}
	snap, e = s.ReadNativeNetworkLedger(ctx, a.Layer)
	if e != nil || len(snap.Boxes) != 0 || snap.OrphanBindings != 1 || snap.OrphanFences != 1 {
		t.Fatal("orphan drift invisible", snap, e)
	}
	other, e := s.ReadNativeNetworkLedger(ctx, "other-project")
	if e != nil || len(other.Boxes) != 0 || other.OrphanBindings != 0 || other.OrphanFences != 0 {
		t.Fatal("foreign scope leaked", other, e)
	}
}
func TestN3NativeAuditCursorAuthenticatedAndChangeRefusal(t *testing.T) {
	ctx := context.Background()
	s, a, _, _, tgt := n3OwnedCreateForRepair(t)
	input := []ActivityIssue{
		{BoxUUID: tgt.UUID, Reason: "NATIVE_LEASE_MISMATCH"},
		{BoxUUID: tgt.UUID, Reason: "NETWORK_FENCE_MISSING"},
		{BoxUUID: tgt.UUID, Reason: "NETWORK_BINDING_MISSING"},
	}
	digest := strings.Repeat("a", 64)
	info := NativeNetworkAuditInfo{Owned: 1, Native: 1, InventoryScanned: true}
	p, e := s.PageNativeNetworkIssues(ctx, a.Layer, a.TokenID, "", 1, "", digest, input, info)
	if e != nil || len(p.Issues) != 1 || p.NextCursor == "" || !p.Truncated || p.AttentionComplete || p.CoverageDetails.NativeLiveVerified != true {
		t.Fatal(p, e)
	}
	next, e := s.PageNativeNetworkIssues(ctx, a.Layer, a.TokenID, "", 1, p.NextCursor, digest, input, info)
	if e != nil || len(next.Issues) != 1 || next.Issues[0].Reason != input[1].Reason {
		t.Fatal("lost signed page", next, e)
	}
	for _, tc := range []struct{ scope, actor, box, cursor, digest string }{
		{"another-project", a.TokenID, "", p.NextCursor, digest},
		{a.Layer, a.TokenID, tgt.UUID, p.NextCursor, digest},
		{a.Layer, a.TokenID, "", p.NextCursor, strings.Repeat("b", 64)},
		{a.Layer, a.TokenID, "", p.NextCursor + "!", digest},
	} {
		if _, e = s.PageNativeNetworkIssues(ctx, tc.scope, tc.actor, tc.box, 1, tc.cursor, tc.digest, input, info); e == nil {
			t.Fatal("untrusted native page admitted", tc)
		}
	}
}

func TestN3NativeLedgerOriginalCreateJournalAndClaimsTamper(t *testing.T) {
	for _, which := range []string{"success event", "staged original reservation"} {
		t.Run(which, func(t *testing.T) {
			ctx := context.Background()
			s, a, c, p, _ := n3OwnedCreateForRepair(t)
			before, e := s.ReadNativeNetworkLedger(ctx, a.Layer)
			if e != nil || !before.Boxes[0].OriginalProof {
				t.Fatal("healthy original was not proven", before, e)
			}
			switch which {
			case "success event":
				_, e = s.db.Exec("DELETE FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'", c.ID)
			case "staged original reservation":
				_, e = s.db.Exec("DELETE FROM operation_resource_claims WHERE operation_id=? AND resource_key=?", c.ID, p.reservationKeys()[0])
			}
			if e != nil {
				t.Fatal(e)
			}
			after, e := s.ReadNativeNetworkLedger(ctx, a.Layer)
			if e != nil || len(after.Boxes) != 1 || after.Boxes[0].OriginalProof {
				t.Fatal("forged original accepted as healthy", after, e)
			}
		})
	}
}

func TestN3NativeAuditManyIssuesRemainPageable(t *testing.T) {
	ctx := context.Background()
	s, a, _, _, _ := n3OwnedCreateForRepair(t)
	issues := make([]ActivityIssue, 3000)
	for i := range issues {
		issues[i] = ActivityIssue{BoxUUID: "11111111-1111-4111-8111-000000000001", Reason: "NATIVE_STATIC_LEASE_DRIFT"}
	}
	digest := strings.Repeat("e", 64)
	p, e := s.PageNativeNetworkIssues(ctx, a.Layer, a.TokenID, "", 50, "", digest, issues, NativeNetworkAuditInfo{Owned: 1024, Native: 1024, InventoryScanned: true})
	if e != nil || len(p.Issues) != 50 || !p.Truncated || p.NextCursor == "" {
		t.Fatal("large bounded scope silently dropped", p, e)
	}
	next, e := s.PageNativeNetworkIssues(ctx, a.Layer, a.TokenID, "", 50, p.NextCursor, digest, issues, NativeNetworkAuditInfo{})
	if e != nil || len(next.Issues) != 50 || next.NextCursor == "" {
		t.Fatal("large scope pagination broken", next, e)
	}
}

func TestN3NativeNetworkLedger1024ProjectLimitAndNoPartialReport(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	const project = "user-1001"
	seedCapacityOwned(t, st, project, networklimits.MaxProjectBoxes)
	audit, e := st.ReadNativeNetworkLedger(ctx, project)
	if e != nil || len(audit.Boxes) != networklimits.MaxProjectBoxes || audit.Project != project ||
		audit.OrphanBindings != 0 || audit.OrphanFences != 0 || audit.Unsettled != 0 {
		t.Fatal("lost complete bounded 1024-Box SQLite inventory", len(audit.Boxes), e)
	}
	if audit.Boxes[0].OriginalCount != 0 || audit.Boxes[0].OriginalProof {
		t.Fatal("synthetic Box was adopted as proven Create", audit.Boxes[0])
	}
	if _, e = st.db.Exec("INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES('overflow','user-1001','overflow','overflow','OWNED',1)"); e != nil {
		t.Fatal(e)
	}
	audit, e = st.ReadNativeNetworkLedger(ctx, project)
	if !errors.Is(e, ErrResourceLocked) || len(audit.Boxes) != 0 {
		t.Fatal("1025 owned entries produced a partial trusted report", len(audit.Boxes), e)
	}
}
