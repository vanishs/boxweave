package state

import (
	"context"
	"errors"
	"fmt"
	"github.com/vanishs/boxweave/internal/networklimits"
	"testing"
	"time"
)

func TestH4BScopedIdentityReserveAndReplay(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	project := "user-1001"
	// Synthetic history only: no claim of native lifecycle/lineage proof.
	_, e = s.db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<?)
 INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at)
 SELECT printf('fixture-%d',i),?,printf('old-%d',i),printf('fixture-%d',i),'DELETED',1 FROM n`, networklimits.ScopedRetiredIdentityLimit-2, project)
	if e != nil {
		t.Fatal(e)
	}
	seedCapacityOwned(t, s, project, 1)
	a := Admission{Layer: project, ActorID: tok.ActorID, TokenID: tok.ID, Key: "h4b-final-identity-create-01", Action: "box_create", SafeIntent: []byte(`{"schema":2,"name":"last"}`), RequestID: "h4b-last", ResourceKey: "box:last"}
	c, e := s.ReadScopedIdentityCapacity(ctx, project)
	if e != nil || c.RemainingNewIdentities != 1 || c.Level != "LOW" {
		t.Fatal(c, e)
	}
	first, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	c, e = s.ReadScopedIdentityCapacity(ctx, project)
	if e != nil || c.RemainingNewIdentities != 0 || c.PendingCreates != 1 || c.Level != "EXHAUSTED" {
		t.Fatal(c, e)
	}
	again, e := s.Admit(ctx, a)
	if e != nil || !again.Duplicate || again.ID != first.ID {
		t.Fatal(again, e)
	}
	// Simulate the last admitted create settling into an owned identity.
	// Use direct fixture transitions only, not production recovery/release.
	if _, e = s.db.Exec("UPDATE operations SET state='SUCCEEDED' WHERE id=?", first.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES('last',?,'last','last','OWNED',1)", project); e != nil {
		t.Fatal(e)
	}
	a.Key = "h4b-refused-new-identity-01"
	a.ResourceKey = "box:next"
	a.SafeIntent = []byte(`{"schema":2,"name":"next"}`)
	if _, e = s.Admit(ctx, a); !errors.Is(e, ErrScopedIdentityCapacity) {
		t.Fatal(e)
	}
	// Retirement of an existing identity must remain possible; occupancy moves
	// from owned to retired, so it does not create headroom for another Create.
	if _, e = s.db.Exec("UPDATE owned_boxes SET state='DELETED' WHERE uuid='last'"); e != nil {
		t.Fatal(e)
	}
	c, e = s.ReadScopedIdentityCapacity(ctx, project)
	if e != nil || c.RemainingNewIdentities != 0 || c.Retired != networklimits.ScopedRetiredIdentityLimit-1 || c.Owned != 1 {
		t.Fatal(c, e)
	}
	if _, e = s.Admit(ctx, a); !errors.Is(e, ErrScopedIdentityCapacity) {
		t.Fatal(e)
	}
	other, e := s.ReadScopedIdentityCapacity(ctx, "user-9999")
	if e != nil || other.RemainingNewIdentities != other.Limit {
		t.Fatal(other, e)
	}
}
func TestH4BPeerCapacityReportsRetainedAndUnknown(t *testing.T) {
	s, _, id := pr178ScopedReadyToStage(t, true)
	ctx := context.Background()
	before, e := s.ReadDiagnosticPeerCapacity(ctx, "user-1001")
	if e != nil || before.DirectionLevel != "AVAILABLE" || before.RemainingDirections != before.DirectionLimit-before.RetainedDirections || before.ReservedRevokeEvents != before.ActiveDirections {
		t.Fatal(before, e)
	}
	pr178SeedPolicyHistory(t, s, id, networklimits.ScopedPeerHistoryLimit-2)
	c, e := s.ReadDiagnosticPeerCapacity(ctx, "user-1001")
	if e != nil || c.NewGrantHistoryAvailable || c.RemainingGrantEventsUpperBound != 0 || c.GrantHistoryLevel != "EXHAUSTED" {
		t.Fatal(c, e)
	}
	// Pending/unknown is deliberately not permission to allocate.
	_, e = s.db.Exec("UPDATE peer_grants SET status='UNKNOWN' WHERE project='user-1001'")
	if e != nil {
		t.Fatal(e)
	}
	c, e = s.ReadDiagnosticPeerCapacity(ctx, "user-1001")
	if e != nil || c.LedgerCoverage != "UNSETTLED_OR_INCONSISTENT" || c.DirectionLevel != "UNVERIFIED" || c.GrantHistoryLevel != "UNVERIFIED" {
		t.Fatal(c, e)
	}
	t.Log(fmt.Sprintf("H4B history used=%d unknown=%d", c.CompletedLedgerEvents, c.PendingOrUnknown))
}
