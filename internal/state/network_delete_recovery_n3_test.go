package state

import (
	"context"
	"errors"
	"github.com/vanishs/boxweave/internal/peerscope"
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
	a.SafeIntent = []byte(`{"box_id":"` + uuid + `"}`)
	r, e := s.AdmitScoped(context.Background(), a)
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
	for _, tc := range []struct{ name, query string }{
		{"complete", ""},
		{"damaged recovery audit", "UPDATE recovery_audit SET decision='RELEASED' WHERE operation_id=?"},
		{"damaged original event", "UPDATE operation_events SET safe_detail='DAMAGED' WHERE operation_id=? AND safe_detail='RECOVERY_SCOPED_CREATE_ATOMIC_NO_REPLAY'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, orig, in, pin := p1CreateFixture(t, true)
			if e := s.Unknown(ctx, orig.ID, "VERIFYING", "FINAL_COMMIT_LOST"); e != nil {
				t.Fatal(e)
			}
			req, target := a4Request(t, s, orig.ID)
			tx, e := s.db.BeginTx(ctx, nil)
			if e != nil {
				t.Fatal(e)
			}
			proof, e := readScopedCreateOriginalTxPhase(ctx, tx, orig.ID, in.Project, in.Bridge, "UNKNOWN")
			tx.Rollback()
			if e != nil {
				t.Fatal(e)
			}
			pin.ACL, e = peerscope.ACLName(in.Project, in.Bridge, p1UUID)
			if e != nil {
				t.Fatal(e)
			}
			a4SettleMode(t, s, req, target, p1CompletedProof(t, target, "CREATE", pin, ScopedLifecycleOriginalSHA256(proof)), true)
			a := b5DeleteAdmission(t, s, p1UUID, in.Name)
			a.Layer = in.Project
			a.SafeIntent = []byte(`{"box_id":"` + p1UUID + `"}`)
			if tc.query != "" {
				if _, e = s.db.Exec(tc.query, orig.ID); e != nil {
					t.Fatal(e)
				}
			}
			_, e = s.AdmitScoped(ctx, a)
			if tc.query != "" {
				if !errors.Is(e, ErrResourceLocked) {
					t.Fatal("damaged recovery provenance admitted Delete", e)
				}
			} else if e != nil {
				t.Fatal("valid recovery refused", e)
			}
		})
	}
}
