package state

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestH2BExactOriginalObservation(t *testing.T) {
	ctx := context.Background()
	s, b, _ := n3RetiredFixture(t)
	b, e := s.OwnedBox(ctx, b.Project, "11111111-1111-4111-8111-000000000002")
	if e != nil {
		t.Fatal(e)
	}
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(struct {
		BoxID string `json:"box_id"`
	}{b.UUID})
	rec, e := s.Admit(ctx, Admission{Layer: b.Project, ActorID: tok.ID, TokenID: tok.ID, Key: "h2b-power-original", RequestID: "h2b-power", Action: "box_stop", ResourceKey: "box:" + b.Name, BoxUUID: b.UUID, Incarnation: b.Incarnation, SafeIntent: raw})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, rec.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, rec.ID, "native-power-original"); e != nil {
		t.Fatal(e)
	}
	if e = s.Unknown(ctx, rec.ID, "VERIFYING", "H2B_FIXTURE_UNKNOWN"); e != nil {
		t.Fatal(e)
	}
	snap := DiagnosticSnapshot{store: s}
	for i := 0; i < 2; i++ {
		l, e := snap.ReadMaintenanceLedger(ctx, b.Project)
		if e != nil {
			t.Fatal(e)
		}
		if len(l.Originals) != 1 {
			t.Fatal(l.Originals)
		}
		o := l.Originals[0]
		if o.ID != rec.ID || o.State != "UNKNOWN" || o.RetirementScope != "EXACT_OWNED_POWER_ORIGINAL" || o.ClaimProtection != "RETAINED" || o.Claims != 1 {
			t.Fatal(o)
		}
	}
}

func TestP2ScopedOfflineArchiveLedger(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ name, sql string }{
		{"exact", ""},
		{"wrong-action", "UPDATE operations SET action='box_start' WHERE state='UNKNOWN'"},
		{"missing-native", "UPDATE operations SET native_operation_id='' WHERE state='UNKNOWN'"},
		{"wrong-resource", "UPDATE operations SET resource_key='box:foreign' WHERE state='UNKNOWN'"},
		{"missing-claim", "DELETE FROM operation_resource_claims WHERE operation_id=(SELECT id FROM operations WHERE state='UNKNOWN')"},
		{"foreign-claim", "UPDATE operation_resource_claims SET layer='foreign' WHERE operation_id=(SELECT id FROM operations WHERE state='UNKNOWN')"},
		{"inflight", "UPDATE operations SET state='RUNNING' WHERE state='UNKNOWN'"},
		{"binding-drift", "UPDATE network_bindings SET acl='bw-foreign' WHERE state='PINNED'"},
		{"foreign-owner", "UPDATE owned_boxes SET project='user-9999' WHERE state='DELETED'"},
		{"missing-history", "DELETE FROM operation_events WHERE safe_detail='SCOPED_DELETE_ACL_ABSENT_ATOMIC'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, b, _ := n3RetiredFixture(t)
			b, e := s.OwnedBox(ctx, b.Project, "11111111-1111-4111-8111-000000000002")
			if e != nil {
				t.Fatal(e)
			}
			_, tok, e := s.IssueToken(ctx, time.Hour)
			if e != nil {
				t.Fatal(e)
			}
			raw, _ := json.Marshal(struct {
				BoxID string `json:"box_id"`
			}{b.UUID})
			r, e := s.AdmitScoped(ctx, Admission{Layer: b.Project, ActorID: tok.ID, TokenID: tok.ID, Key: "p2-offline-power", RequestID: "p2-offline", Action: "box_stop", ResourceKey: "box:" + b.Name, BoxUUID: b.UUID, Incarnation: b.Incarnation, SafeIntent: raw})
			if e != nil {
				t.Fatal(e)
			}
			if e = s.Claim(ctx, r.ID); e != nil {
				t.Fatal(e)
			}
			if e = s.NativeAck(ctx, r.ID, "native-p2-power-original"); e != nil {
				t.Fatal(e)
			}
			if e = s.Unknown(ctx, r.ID, "VERIFYING", "P2_FIXTURE_UNKNOWN"); e != nil {
				t.Fatal(e)
			}
			if tc.sql != "" {
				if _, e = s.db.Exec(tc.sql); e != nil {
					t.Fatal(e)
				}
			}
			snap := DiagnosticSnapshot{store: s}
			before, e := snap.ReadMaintenanceLedger(ctx, b.Project)
			if e != nil {
				t.Fatal(e)
			}
			proof, e := snap.ReadScopedOfflineArchiveLedger(ctx, b.Project, "incusbr-1001")
			if tc.sql != "" {
				if e == nil {
					t.Fatal("invalid archival scope admitted", proof)
				}
			} else {
				if e != nil || len(proof.Inventory.Boxes) != 2 || len(proof.Inventory.Retired) != 1 ||
					len(proof.Originals) != 1 || proof.Originals[0].ID != r.ID || proof.Originals[0].State != "UNKNOWN" ||
					proof.Originals[0].ClaimProtection != "RETAINED" || proof.LedgerSHA256 != before.ObservationDigest() {
					t.Fatal(proof, e)
				}
				if _, e = s.ReadScopedDrainWitness(ctx, b.Project); e == nil {
					t.Fatal("ordinary drain accepted unknown")
				}
			}
			after, e := snap.ReadMaintenanceLedger(ctx, b.Project)
			if e != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("archive observation changed source", e)
			}
		})
	}
}
