package state

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestN3ModernDeleteAtomicRecoveryFenceFault(t *testing.T) {
	ctx := context.Background()
	s, a, _, pin, target := n3OwnedCreateForRepair(t)
	dr := n3AdmitDeleteForPin(t, s, a, target.UUID)
	n3DeleteOriginalUnknown(t, s, dr.ID)
	req := RecoveryRequest{Layer: a.Layer, ActorID: a.ActorID, TokenID: a.TokenID, OriginalID: dr.ID, Key: "n3-modern-delete-reconcile"}
	dt, e := s.ReadRecoveryTarget(ctx, a.Layer, dr.ID)
	if e != nil || dt.State != "UNKNOWN" {
		t.Fatal(dt, e)
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	original, e := readScopedDeleteOriginalTxPhase(ctx, tx, dr.ID, dt.Layer, pin.Bridge, "UNKNOWN")
	tx.Rollback()
	if e != nil {
		t.Fatal(e)
	}
	proof := p1CompletedProof(t, dt, "DELETE", pin, ScopedLifecycleOriginalSHA256(original))
	proof.UUID = dt.UUID
	proof.ScopedLifecycle.Record.UUID = dt.UUID

	if _, e = s.db.Exec("DELETE FROM network_bindings WHERE box_uuid=?", dt.UUID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitRecovery(ctx, req, dt, proof); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("missing modern pin accepted", e)
	}
	if _, e = s.db.Exec("INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES(?,?,?,'eth0',?,?,?,?,?,'PINNED',1)", dt.Layer, dt.UUID, dt.Incarnation, pin.Bridge, pin.ACL, pin.MAC, pin.IPv4, pin.IPv6); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("UPDATE network_address_fences SET operation_id='foreign' WHERE box_uuid=?", dt.UUID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitRecovery(ctx, req, dt, proof); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("wrong original fence accepted", e)
	}
	var createID string
	if e = s.db.QueryRow("SELECT id FROM operations WHERE layer=? AND action='box_create' AND box_uuid=?", dt.Layer, dt.UUID).Scan(&createID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("UPDATE network_address_fences SET operation_id=? WHERE box_uuid=?", createID, dt.UUID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("CREATE TRIGGER n3_del_fault BEFORE UPDATE ON network_bindings WHEN OLD.state='PINNED' BEGIN SELECT RAISE(ABORT,'delete partial fault'); END"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitRecovery(ctx, req, dt, proof); e == nil {
		t.Fatal("partial delete committed")
	}
	if _, e = s.db.Exec("DROP TRIGGER n3_del_fault"); e != nil {
		t.Fatal(e)
	}
	var n int
	for _, table := range []string{"network_bindings", "network_address_fences"} {
		if e = s.db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE box_uuid=? AND state='RELEASED'", dt.UUID).Scan(&n); e != nil || n != 0 {
			t.Fatal(table, n, e)
		}
	}
	var phase string
	if e = s.db.QueryRow("SELECT state FROM owned_boxes WHERE uuid=?", dt.UUID).Scan(&phase); e != nil || phase != "OWNED" {
		t.Fatal(phase, e)
	}
	res, e := s.CommitRecovery(ctx, req, dt, proof)
	if e != nil || res.Disposition != "SETTLED" {
		t.Fatal(res, e)
	}
	replay, e := s.CommitRecovery(ctx, req, dt, proof)
	if e != nil || !replay.Receipt.Duplicate || replay.Receipt.ID != res.Receipt.ID {
		t.Fatal(replay, e)
	}
	if e = s.db.QueryRow("SELECT state FROM owned_boxes WHERE uuid=?", dt.UUID).Scan(&phase); e != nil || phase != "DELETED" {
		t.Fatal(phase, e)
	}
	for _, table := range []string{"network_bindings", "network_address_fences"} {
		want := 1
		if table == "network_address_fences" {
			want = 2
		}
		if e = s.db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE box_uuid=? AND state='RELEASED'", dt.UUID).Scan(&n); e != nil || n != want {
			t.Fatal(table, n, e)
		}
	}
	ro, e := OpenNetworkAuditReadOnly(strings.TrimSuffix(s.ExecArchiveDir(), ".exec-archive"))
	if e != nil {
		t.Fatal(e)
	}
	defer ro.Close()
	history, e := ro.ReadScopedRetirementLedger(ctx, a.Layer, pin.Bridge)
	if e != nil || len(history.Retired) != 1 || history.Retired[0].DeleteID != dr.ID {
		t.Fatal(history, e)
	}
}
