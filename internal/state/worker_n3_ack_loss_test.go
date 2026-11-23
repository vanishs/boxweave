package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A DB cut that rejects BOTH the ordinary ACK and the emergency trusted
// receipt write still allows a state-only UNKNOWN; no reboot needed.
func TestN3WorkerBothNativeACKWritesRejectedProtectsUnknown(t *testing.T) {
	ctx := context.Background()
	s, a, _, _, target := n3OwnedCreateForRepair(t)
	deleteOp := n3AdmitDeleteForPin(t, s, a, target.UUID)
	if _, e := s.db.Exec(`CREATE TRIGGER n3_double_ack_fault BEFORE UPDATE OF native_operation_id ON operations
 WHEN OLD.action='box_delete' AND OLD.native_operation_id='' AND NEW.native_operation_id!=''
 BEGIN SELECT RAISE(ABORT,'n3 ACK durability fault'); END`); e != nil {
		t.Fatal(e)
	}
	dispatched, verified := 0, 0
	Worker{Store: s, Budget: 3 * time.Second}.Execute(deleteOp.ID,
		func(context.Context) (string, error) { dispatched++; return "original-native-delete-id", nil },
		func(context.Context) error { verified++; return nil })
	if dispatched != 1 || verified != 0 {
		t.Fatal("native effect retried or erroneously verified", dispatched, verified)
	}
	row, e := s.Get(ctx, deleteOp.ID)
	if e != nil || row.State != "UNKNOWN" || row.NativeOperationID != "" || row.ErrorClass != "NATIVE_ACK_DURABILITY_UNVERIFIED" {
		t.Fatal("live Worker did not preserve UNKNOWN", row, e)
	}
	var n int
	if e = s.db.QueryRow("SELECT COUNT(*) FROM owned_boxes WHERE uuid=? AND state='OWNED'", target.UUID).Scan(&n); e != nil || n != 1 {
		t.Fatal("false delete tombstone", n, e)
	}
	for _, tbl := range []string{"network_bindings", "network_address_fences"} {
		if e = s.db.QueryRow("SELECT COUNT(*) FROM "+tbl+" WHERE box_uuid=? AND state='RELEASED'", target.UUID).Scan(&n); e != nil || n != 0 {
			t.Fatal("lost network holds", tbl, n, e)
		}
	}
	another := a
	another.Action = "box_delete"
	another.Key = "n3-double-ack-second-delete-original"
	another.RequestID = "second"
	another.BoxUUID = target.UUID
	another.Incarnation = target.Incarnation
	another.CleanupDependencies = true
	if _, e = s.Admit(ctx, another); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("UNKNOWN did not protect native replay", e)
	}
	if _, e = s.db.Exec("DROP TRIGGER n3_double_ack_fault"); e != nil {
		t.Fatal(e)
	}
	if recovered, e := s.Recover(ctx); e != nil || recovered != 0 {
		t.Fatal("restart changed safe UNKNOWN", recovered, e)
	}
	if _, e = s.Get(ctx, deleteOp.ID); e != nil {
		t.Fatal(e)
	}
}
