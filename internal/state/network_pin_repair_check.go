package state

import (
	"context"
	"database/sql"
)

// The independently observed Create tuple must be the sole safe restoration
// candidate. Refuse mismatched rows, foreign occupancy, unresolved work or
// absent original reservation; never upsert or synthesize an address.
func checkCreatePinRepairTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget, p RecoveryProof) error {
	if p.Kind != RecoveryRepairPin || p.Reason != "CREATE_PIN_LEDGER_REPAIRED" ||
		p.UUID != t.UUID || p.NativeID != t.NativeID || p.Fingerprint == "" ||
		!p.CreateBinding.valid() || !validActivityBoxID(t.UUID) {
		return ErrInvalid
	}
	status, e := createPinRepairStatusTx(ctx, tx, t)
	if e != nil {
		return e
	}
	if status != "MISSING" {
		return ErrResourceLocked
	}
	pin := p.CreateBinding
	if e = requireNoIPv4Compaction(ctx, tx, t.Layer); e != nil {
		return e
	}
	if e = verifyIPv4RetirementHistory(ctx, tx, t.Layer, pin.Bridge, pin.ACL); e != nil {
		return e
	}
	if e = requireCreateBindingReservation(ctx, tx, t.ID, t.Layer, pin); e != nil {
		return e
	}
	var n int
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations c WHERE c.id=? AND c.layer=? AND c.action='box_create'
 AND c.state='SUCCEEDED' AND c.box_uuid=? AND c.incarnation=? AND c.resource_key=?
 AND EXISTS(SELECT 1 FROM operation_events e WHERE e.operation_id=c.id AND e.state='SUCCEEDED')
 AND `+retirementCreateEvidence, t.ID, t.Layer, t.UUID, t.Incarnation, t.Resource).Scan(&n)
	if e != nil {
		return e
	}
	if n != 1 {
		return ErrResourceLocked
	}
	e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM owned_boxes WHERE project=? AND name=? AND uuid=? AND incarnation=? AND state='OWNED'", t.Layer, t.Name, t.UUID, t.Incarnation).Scan(&n)
	if e != nil {
		return e
	}
	if n != 1 {
		return ErrResourceLocked
	}
	// Native effects may still be in flight even after manual legacy release.
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE layer=? AND id!=? AND state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN','OBSERVING')`, t.Layer, t.ID).Scan(&n)
	if e != nil {
		return e
	}
	if n != 0 {
		return ErrResourceLocked
	}
	e = tx.QueryRowContext(ctx, `SELECT
 (SELECT COUNT(*) FROM network_bindings WHERE project=? AND box_uuid=? AND
 (incarnation!=? OR nic!='eth0' OR state!='PINNED' OR bridge!=? OR acl!=? OR mac!=? OR ipv4!=? OR ipv6!=?))+
 (SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=? AND
 (incarnation!=? OR bridge!=? OR state!='RESERVED' OR operation_id!=? OR
 NOT ((family='ipv4' AND address=?) OR (family='ipv6' AND address=?))))+
 (SELECT COUNT(*) FROM network_bindings n WHERE bridge=? AND
 NOT (project=? AND box_uuid=? AND incarnation=?) AND
 (mac=? OR ipv6=? OR (ipv4=? AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=n.project AND r.box_uuid=n.box_uuid AND r.incarnation=n.incarnation))
 OR (project!=? AND state!='RELEASED')))+
 (SELECT COUNT(*) FROM network_address_fences WHERE bridge=? AND
 ((family='ipv4' AND address=?) OR (family='ipv6' AND address=?) OR (project!=? AND state!='RELEASED'))
 AND NOT (project=? AND box_uuid=? AND incarnation=?))+
 (SELECT COUNT(*) FROM owned_boxes WHERE uuid=? AND NOT (project=? AND incarnation=? AND name=? AND state='OWNED'))`,
		t.Layer, t.UUID, t.Incarnation, pin.Bridge, pin.ACL, pin.MAC, pin.IPv4, pin.IPv6,
		t.Layer, t.UUID, t.Incarnation, pin.Bridge, t.ID, pin.IPv4, pin.IPv6,
		pin.Bridge, t.Layer, t.UUID, t.Incarnation, pin.MAC, pin.IPv6, pin.IPv4, t.Layer,
		pin.Bridge, pin.IPv4, pin.IPv6, t.Layer, t.Layer, t.UUID, t.Incarnation,
		t.UUID, t.Layer, t.Incarnation, t.Name).Scan(&n)
	if e != nil {
		return e
	}
	if n != 0 {
		return ErrResourceLocked
	}
	for i, key := range pin.reservationKeys()[:3] {
		sqlText := "SELECT COUNT(*) FROM operation_resource_claims c WHERE c.resource_key=? AND c.operation_id!=?"
		args := []any{key, t.ID}
		if i == 0 {
			sqlText += " AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.bridge=? AND r.ipv4=? AND r.create_operation_id=c.operation_id AND r.project=c.layer)"
			args = append(args, t.Layer, pin.Bridge, pin.IPv4)
		}
		if e = tx.QueryRowContext(ctx, sqlText, args...).Scan(&n); e != nil {
			return e
		}
		if n != 0 {
			return ErrResourceLocked
		}
	}
	return nil
}
