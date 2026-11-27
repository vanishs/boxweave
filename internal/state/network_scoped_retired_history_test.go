package state

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/peerscope"
)

func n3ScopedOwnedFixture(t *testing.T) (*Store, OwnedBox, string) {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	const project = "user-1001"
	const bridge = "incusbr-1001"
	var box OwnedBox
	for i := 1; i <= 3; i++ {
		name := fmt.Sprintf("n3-history-%d", i)
		uuid := fmt.Sprintf("11111111-1111-4111-8111-%012d", i)
		intent := ScopedCreateIntent{Schema: 2, Project: project, Bridge: bridge, Name: name, CPU: 1, MemoryMiB: 256, DiskGiB: 2, Options: json.RawMessage("{}")}
		raw, _ := json.Marshal(intent)
		a := Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: "n3-create-history-" + name, RequestID: name, Action: "box_create", ResourceKey: "box:" + name, SafeIntent: raw}
		rec, e := s.AdmitScoped(ctx, a)
		if e != nil {
			t.Fatal(e)
		}
		if e = s.Claim(ctx, rec.ID); e != nil {
			t.Fatal(e)
		}
		b := VerifiedCreateBinding{bridge, peerscope.QuarantineACL, fmt.Sprintf("02:16:3e:11:22:%02x", i), fmt.Sprintf("10.43.18.%d", i+10), fmt.Sprintf("fd43:18::%d", i+10)}
		ok, e := s.StageCreateBinding(ctx, rec.ID, project, name, b)
		if e != nil || !ok {
			t.Fatal(ok, e)
		}
		if e = s.NativeAck(ctx, rec.ID, "native-create-"+name); e != nil {
			t.Fatal(e)
		}
		b.ACL, e = peerscope.ACLName(project, bridge, uuid)
		if e != nil {
			t.Fatal(e)
		}
		if e = s.FinalizeVerifiedScopedCreate(ctx, rec.ID, project, name, uuid, b); e != nil {
			t.Fatal(e)
		}
		if i == 1 {
			box, e = s.OwnedBox(ctx, project, uuid)
			if e != nil {
				t.Fatal(e)
			}
		}
	}
	return s, box, tok.ID
}
func n3DeleteOwned(t *testing.T, s *Store, box OwnedBox, tokenID string) Receipt {
	t.Helper()
	ctx := context.Background()
	const project = "user-1001"
	raw, _ := json.Marshal(struct {
		BoxID string `json:"box_id"`
	}{box.UUID})
	del, e := s.AdmitScoped(ctx, Admission{Layer: project, ActorID: tokenID, TokenID: tokenID, Key: "n3-history-delete-" + box.Name, RequestID: "delete", Action: "box_delete", CleanupDependencies: true, ResourceKey: "box:" + box.Name, BoxUUID: box.UUID, Incarnation: box.Incarnation, SafeIntent: raw})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, del.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, del.ID, "native-delete-history"); e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedScopedDelete(ctx, del.ID, box); e != nil {
		t.Fatal(e)
	}
	return del
}
func n3RetiredFixture(t *testing.T) (*Store, OwnedBox, Receipt) {
	t.Helper()
	s, box, token := n3ScopedOwnedFixture(t)
	return s, box, n3DeleteOwned(t, s, box, token)
}
func TestPR178N3ScopedRetiredHistory(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ name, sql string }{
		{"valid", ""},
		{"missing-delete-event", "DELETE FROM operation_events WHERE safe_detail='SCOPED_DELETE_ACL_ABSENT_ATOMIC'"},
		{"legacy-delete-event", "UPDATE operation_events SET safe_detail='NATIVE_DELETE_ATOMIC' WHERE safe_detail='SCOPED_DELETE_ACL_ABSENT_ATOMIC'"},
		{"missing-fence", "DELETE FROM network_address_fences WHERE state='RELEASED' AND family='ipv6'"},
		{"foreign-fence-original", "UPDATE network_address_fences SET operation_id='forged' WHERE state='RELEASED'"},
		{"missing-binding", "DELETE FROM network_bindings WHERE state='RELEASED'"},
		{"shared-historical-acl", "UPDATE network_bindings SET acl='bw-legacy' WHERE state='RELEASED'"},
		{"wrong-historical-claim", "DELETE FROM operation_resource_claims WHERE operation_id=(SELECT deleted_by_operation_id FROM owned_boxes WHERE state='DELETED')"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, b, del := n3RetiredFixture(t)
			if tc.sql != "" {
				if _, e := s.db.Exec(tc.sql); e != nil {
					t.Fatal(e)
				}
			}
			ledger, e := s.ReadCompleteScopedNetworkAuditLedger(ctx, b.Project, "incusbr-1001")
			if tc.sql != "" {
				if e == nil {
					t.Fatal("unverified retired history accepted", ledger)
				}
				return
			}
			if e != nil || len(ledger.Boxes) != 2 || len(ledger.Retired) != 1 || ledger.Retired[0].UUID != b.UUID || ledger.Retired[0].DeleteID != del.ID {
				t.Fatal(ledger, e)
			}
		})
	}
}
