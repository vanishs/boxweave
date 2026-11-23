package state

import (
	"context"
	"testing"
)

func n3NetworkIssuesForBox(t *testing.T, s *Store, a Admission, uuid string) []ActivityIssue {
	t.Helper()
	ctx := context.Background()
	q := ActivityQuery{Layer: a.Layer, Requester: a.ActorID, View: "issues", BoxUUID: uuid, BlockersOnly: true, Limit: 1}
	var out []ActivityIssue
	for page := 0; page < 16; page++ {
		result, e := s.ReadActivity(ctx, q)
		if e != nil {
			t.Fatal(e)
		}
		out = append(out, result.Issues...)
		if result.NextCursor == "" {
			return out
		}
		q.Cursor = result.NextCursor
	}
	t.Fatal("bounded pagination did not terminate")
	return nil
}
func TestN3ActivityModernCreateLineageAndScopedOrphan(t *testing.T) {
	ctx := context.Background()
	s, a, create, _, target := n3OwnedCreateForRepair(t)
	if got := n3NetworkIssuesForBox(t, s, a, target.UUID); len(got) != 0 {
		t.Fatal("healthy Box has blockers", got)
	}
	if _, e := s.db.Exec("UPDATE operations SET resource_key='box:tampered' WHERE id=?", create.ID); e != nil {
		t.Fatal(e)
	}
	got := n3NetworkIssuesForBox(t, s, a, target.UUID)
	if len(got) != 1 || got[0].Reason != "NETWORK_CREATE_LINEAGE_MISMATCH" || got[0].OperationID != create.ID ||
		got[0].NextAction == nil || got[0].NextAction.Tool != "box_activity_get" {
		t.Fatal("bad original not diagnosed", got)
	}
	if _, e := s.db.Exec("UPDATE operations SET resource_key='box:alpha' WHERE id=?", create.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec("UPDATE owned_boxes SET state='DELETED' WHERE uuid=?", target.UUID); e != nil {
		t.Fatal(e)
	}
	got = n3NetworkIssuesForBox(t, s, a, target.UUID)
	seen := map[string]int{}
	for _, v := range got {
		seen[v.Reason]++
		if v.BoxUUID != target.UUID || v.Disposition != "INSPECT_EVIDENCE" {
			t.Fatal("unsafe orphan diagnosis", v)
		}
	}
	if len(got) != 3 || seen["NETWORK_ORPHAN_BINDING"] != 1 || seen["NETWORK_ORPHAN_ADDRESS_FENCE"] != 2 {
		t.Fatal("lost orphan rows", got)
	}
	foreign, e := s.ReadActivity(ctx, ActivityQuery{Layer: "foreign", Requester: a.ActorID, View: "issues", BoxUUID: target.UUID, BlockersOnly: true, Limit: 10})
	if e != nil || len(foreign.Issues) != 0 {
		t.Fatal("foreign project exposure", foreign, e)
	}
	filtered, e := s.ReadActivity(ctx, ActivityQuery{Layer: a.Layer, Requester: a.ActorID, View: "issues", BoxUUID: target.UUID, WorkRef: "different", BlockersOnly: true, Limit: 10})
	if e != nil || len(filtered.Issues) != 0 {
		t.Fatal("orphan falsely attributed to unrelated work", filtered, e)
	}
}

func TestN3ActivityNativeBridgeUniquenessRejectsCrossProjectCollision(t *testing.T) {
	ctx := context.Background()
	for _, family := range []string{"mac", "ipv4", "ipv6"} {
		t.Run(family, func(t *testing.T) {
			s, a, _, pin, target := n3OwnedCreateForRepair(t)
			otherMAC, otherV4, otherV6 := "02:16:3e:cc:dd:ee", "10.24.18.254", "fd43:18::fe"
			switch family {
			case "mac":
				otherMAC = pin.MAC
			case "ipv4":
				otherV4 = pin.IPv4
			case "ipv6":
				otherV6 = pin.IPv6
			}
			_, e := s.db.Exec("INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES('foreign','ffffffff-ffff-4fff-8fff-ffffffffffff','ffffffff-ffff-4fff-8fff-ffffffffffff','eth0',?,?,?,?,?,'PINNED',42)", pin.Bridge, pin.ACL, otherMAC, otherV4, otherV6)
			if e == nil {
				t.Fatal("global bridge uniqueness constraint bypassed")
			}
			got := n3NetworkIssuesForBox(t, s, a, target.UUID)
			if len(got) != 0 {
				t.Fatal("rejected duplicate altered ledger", got)
			}
			f, e := s.ReadActivity(ctx, ActivityQuery{Layer: "foreign-empty", Requester: a.ActorID, View: "issues", BoxUUID: target.UUID, BlockersOnly: true, Limit: 10})
			if e != nil || len(f.Issues) != 0 {
				t.Fatal("foreign scope exposed", f, e)
			}
		})
	}
}
