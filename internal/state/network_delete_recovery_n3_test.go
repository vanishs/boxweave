package state

import (
	"context"
	"errors"
	"testing"
)

func n3AdmitDeleteForPin(t *testing.T, s *Store, a Admission, uuid string) Receipt {
	t.Helper()
	a.Action = "box_delete"
	a.Key = "n3-modern-delete-original"
	a.RequestID = "modern-delete"
	a.BoxUUID = uuid
	a.Incarnation = uuid
	a.CleanupDependencies = true
	r, e := s.Admit(context.Background(), a)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func n3DeleteOriginalUnknown(t *testing.T, s *Store, id string) {
	t.Helper()
	ctx := context.Background()
	if e := s.Claim(ctx, id); e != nil {
		t.Fatal(e)
	}
	if e := s.AuthorizeCascadeFinal(ctx, id); e != nil {
		t.Fatal(e)
	}
	if e := s.NativeAck(ctx, id, "n3-real-native-delete-receipt"); e != nil {
		t.Fatal(e)
	}
	if e := s.Unknown(ctx, id, "VERIFYING", "N3_TERMINAL_WRITE_LOST"); e != nil {
		t.Fatal(e)
	}
}
func TestN3ModernDeleteNeverDowngradesMissingPin(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ name, sql string }{
		{"both missing", "DELETE FROM network_bindings WHERE box_uuid=?"},
		{"binding and fences missing", "DELETE FROM network_address_fences WHERE box_uuid=?"},
		{"fence missing", "DELETE FROM network_address_fences WHERE box_uuid=? AND family='ipv6'"},
		{"wrong original fence", "UPDATE network_address_fences SET operation_id='foreign' WHERE box_uuid=?"},
		{"identity mismatch", "UPDATE network_bindings SET mac='02:16:3e:aa:bb:cc' WHERE box_uuid=?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, a, _, _, target := n3OwnedCreateForRepair(t)
			if tc.name == "both missing" {
				if _, e := s.db.Exec("DELETE FROM network_address_fences WHERE box_uuid=?", target.UUID); e != nil {
					t.Fatal(e)
				}
			}
			if _, e := s.db.Exec(tc.sql, target.UUID); e != nil {
				t.Fatal(e)
			}
			changed := a
			changed.Action = "box_delete"
			changed.Key = "n3-modern-delete-admission-attempt"
			changed.RequestID = "delete"
			changed.BoxUUID = target.UUID
			changed.Incarnation = target.Incarnation
			changed.CleanupDependencies = true
			if _, e := s.Admit(ctx, changed); !errors.Is(e, ErrResourceLocked) {
				t.Fatal("corrupted modern ledger admitted native delete", e)
			}
			var n int
			if e := s.db.QueryRow("SELECT COUNT(*) FROM operations WHERE action='box_delete'").Scan(&n); e != nil || n != 0 {
				t.Fatal(n, e)
			}
		})
	}
}

// Database restoration or partial repair must not treat a forged terminal
// Create bit as evidence for a modern pinned Delete.
func TestN3ModernDeleteRejectsDamagedCreateProvenance(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ name, query string }{
		{"failed original", "UPDATE operations SET state='FAILED' WHERE id=?"},
		{"missing success journal", "DELETE FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'"},
		{"wrong original resource", "UPDATE operations SET resource_key='box:other' WHERE id=?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, a, created, _, target := n3OwnedCreateForRepair(t)
			if _, e := s.db.Exec(tc.query, created.ID); e != nil {
				t.Fatal(e)
			}
			req := a
			req.Action = "box_delete"
			req.Key = "n3-damaged-provenance-delete"
			req.RequestID = "corrupt"
			req.BoxUUID = target.UUID
			req.Incarnation = target.Incarnation
			req.CleanupDependencies = true
			if _, e := s.Admit(ctx, req); !errors.Is(e, ErrResourceLocked) {
				t.Fatal("damaged Create lineage accepted", e)
			}
			var n int
			if e := s.db.QueryRow("SELECT COUNT(*) FROM owned_boxes WHERE uuid=? AND state='OWNED'", target.UUID).Scan(&n); e != nil || n != 1 {
				t.Fatal("original owned state changed", n, e)
			}
		})
	}
}

func TestN3DeleteRequiresLostCreateACKExactRecoveryLineage(t *testing.T) {
	ctx := context.Background()
	for _, corrupt := range []bool{false, true} {
		name := "complete"
		if corrupt {
			name = "damaged recovery audit"
		}
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			tok := activityTestToken(t, s)
			a := Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID, Key: "n3-delete-lost-create-ack-original", RequestID: "create", Action: "box_create", ResourceKey: "box:alpha",
				SafeIntent: []byte(`{"name":"alpha","cpu":1,"memory_mib":1024,"disk_gib":10,"start":false,"nesting":false,"internet":true,"instance_options":{}}`)}
			create, e := s.Admit(ctx, a)
			if e != nil {
				t.Fatal(e)
			}
			if e = s.Claim(ctx, create.ID); e != nil {
				t.Fatal(e)
			}
			pin := VerifiedCreateBinding{"bwbr0", "bw-peer", "02:16:3e:11:22:33", "10.24.18.11", "fd43:18::11"}
			if ok, e := s.StageCreateBinding(ctx, create.ID, a.Layer, "alpha", pin); e != nil || !ok {
				t.Fatal(ok, e)
			}
			if e = s.Unknown(ctx, create.ID, "DISPATCHING", "ACK_LOST"); e != nil {
				t.Fatal(e)
			}
			target, e := s.ReadRecoveryTarget(ctx, a.Layer, create.ID)
			if e != nil {
				t.Fatal(e)
			}
			const uuid = "11111111-1111-4111-8111-000000000001"
			proof := RecoveryProof{Kind: RecoverySettle, Reason: "CREATE_ACK_LOST_STATIC_POSTSTATE_VERIFIED", UUID: uuid, Fingerprint: "native-original-verified", CreateBinding: pin}
			if _, e = s.CommitRecovery(ctx, RecoveryRequest{Layer: a.Layer, ActorID: tok.ID, TokenID: tok.ID, OriginalID: create.ID, Key: "n3-delete-acklost-provenance-original-recover"}, target, proof); e != nil {
				t.Fatal(e)
			}
			if corrupt {
				if _, e = s.db.Exec("UPDATE recovery_audit SET decision='RELEASED' WHERE operation_id=?", create.ID); e != nil {
					t.Fatal(e)
				}
			}
			a.Action = "box_delete"
			a.Key = "n3-delete-after-lost-create-ack"
			a.RequestID = "delete"
			a.BoxUUID = uuid
			a.Incarnation = uuid
			a.CleanupDependencies = true
			_, e = s.Admit(ctx, a)
			if corrupt {
				if !errors.Is(e, ErrResourceLocked) {
					t.Fatal("corrupt Create noACK lineage admitted Delete", e)
				}
			} else if e != nil {
				t.Fatal("valid exact Create ACK recovery wrongly refused", e)
			}
		})
	}
}
