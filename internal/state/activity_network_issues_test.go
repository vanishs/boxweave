package state

import (
	"context"
	"strings"
	"testing"
)

func TestN3ActivityNetworkBindingIssues(t *testing.T) {
	const uuid = "11111111-1111-4111-8111-000000000001"
	cases := []struct{ name, change, want string }{
		{"healthy", "", ""},
		{"missing", "DELETE FROM network_bindings WHERE box_uuid='" + uuid + "'", "NETWORK_BINDING_MISSING"},
		{"incarnation", "UPDATE network_bindings SET incarnation='wrong' WHERE box_uuid='" + uuid + "'", "NETWORK_BINDING_IDENTITY_MISMATCH"},
		{"pin", "UPDATE network_bindings SET state='QUARANTINED' WHERE box_uuid='" + uuid + "'", "NETWORK_BINDING_NOT_PINNED"},
		{"missing fence", "DELETE FROM network_address_fences WHERE box_uuid='" + uuid + "' AND family='ipv4'", "NETWORK_ADDRESS_FENCE_MISSING"},
		{"conflict", "UPDATE network_address_fences SET project='foreign',box_uuid='foreign-secret' WHERE box_uuid='" + uuid + "' AND family='ipv4'", "NETWORK_ADDRESS_FENCE_MISMATCH"},
		{"released", "UPDATE network_address_fences SET state='RELEASED' WHERE box_uuid='" + uuid + "' AND family='ipv6'", "NETWORK_ADDRESS_FENCE_MISMATCH"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s := newStore(t)
			tok := activityTestToken(t, s)
			original, err := s.Admit(ctx, Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID, Key: "network-diagnosis-create-01", RequestID: "network-diagnosis-create-01", Action: "box_create", SafeIntent: []byte(`{"name":"a"}`), ResourceKey: "box:a", WorkRef: "activity-work", AgentRef: "activity-agent"})
			if err != nil {
				t.Fatal(err)
			}
			s4b2InsertAuditFixture(t, s)
			if _, err := s.db.Exec("UPDATE operations SET box_uuid=?,incarnation=?,state='SUCCEEDED' WHERE id=?", uuid, uuid, original.ID); err != nil {
				t.Fatal(err)
			}
			if tc.change != "" {
				if _, err := s.db.Exec(tc.change); err != nil {
					t.Fatal(err)
				}
			}
			q := ActivityQuery{Layer: "user-1001", Requester: tok.ID, View: "issues", BoxUUID: uuid, Limit: 1}
			var found []ActivityIssue
			for i := 0; i < 10; i++ {
				res, err := s.ReadActivity(ctx, q)
				if err != nil {
					t.Fatal(err)
				}
				for _, issue := range res.Issues {
					if strings.Contains(issue.ResourceRef, "foreign") {
						t.Fatalf("foreign data: %+v", issue)
					}
					if strings.HasPrefix(issue.Reason, "NETWORK_") {
						found = append(found, issue)
					}
				}
				if res.NextCursor == "" {
					break
				}
				q.Cursor = res.NextCursor
			}
			if tc.want == "" {
				if len(found) != 0 {
					t.Fatal(found)
				}
				return
			}
			if tc.name == "incarnation" {
				if len(found) != 2 || found[1].Reason != "NETWORK_ORPHAN_BINDING" || found[1].BoxUUID != uuid {
					t.Fatalf("orphan incarnation not reported: %+v", found)
				}
				found = found[:1]
			}
			if len(found) != 1 || found[0].Reason != tc.want || found[0].OperationID != original.ID || found[0].BoxUUID != uuid || found[0].ResourceRef != "network-binding:"+uuid+"/"+uuid || found[0].NextAction == nil || found[0].NextAction.Tool != "box_activity_get" {
				t.Fatalf("diagnosis: %+v", found)
			}
			q.Cursor = ""
			q.BlockersOnly = true
			res, err := s.ReadActivity(ctx, q)
			if err != nil || len(res.Issues) != 1 || res.Issues[0].Reason != tc.want {
				t.Fatalf("attention: %+v %v", res, err)
			}
		})
	}
}
