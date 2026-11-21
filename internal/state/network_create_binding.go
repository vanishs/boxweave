package state

import (
	"context"
	"database/sql"
	"time"
)

// VerifiedCreateBinding contains only an exact independently verified native
// static NIC/lease readback. Callers must bind that readback to the original
// native Create and exact UUID. It is never sourced from MCP input.
type VerifiedCreateBinding struct {
	Bridge, ACL, MAC, IPv4, IPv6 string
}

func (b VerifiedCreateBinding) valid() bool {
	return auditName(b.Bridge) && auditName(b.ACL) &&
		auditMAC(b.MAC) && auditIP(b.IPv4, false) && auditIP(b.IPv6, true)
}

// Create address reservations are private original-Operation resource claims.
// They precede the sole native Create; a lost ACK or released UNKNOWN never
// makes an address reusable. The exact UUID is learned only after native
// Create, so no placeholder UUID is written to the network ledger.
func (b VerifiedCreateBinding) reservationKeys() []string {
	prefix := "create-pin:" + b.Bridge + ":"
	return []string{prefix + "v4:" + b.IPv4, prefix + "v6:" + b.IPv6, prefix + "mac:" + b.MAC, prefix + "acl:" + b.ACL}
}

// StageCreateBinding reserves a prospective native allocation, not a PINNED
// witness. False means a collision and no change; errors forbid dispatch.
func (s *Store) StageCreateBinding(ctx context.Context, id, project, name string, b VerifiedCreateBinding) (bool, error) {
	if !b.valid() || !auditName(project) || project == "default" || !auditName(name) || !validHexID(id) {
		return false, ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var phase, action, layer, resource, uuid, nativeID, token, actor string
	if err = tx.QueryRowContext(ctx, `SELECT state,action,layer,resource_key,box_uuid,native_operation_id,token_id,actor_id FROM operations WHERE id=?`, id).Scan(&phase, &action, &layer, &resource, &uuid, &nativeID, &token, &actor); err != nil {
		return false, err
	}
	if phase != "DISPATCHING" || action != "box_create" || layer != project || resource != "box:"+name || uuid != "" || nativeID != "" {
		return false, ErrState
	}
	active, err := activeCredentialForActor(ctx, tx, actor, token)
	if err != nil {
		return false, err
	}
	if !active {
		return false, ErrRevoked
	}
	var n int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", id).Scan(&n); err != nil {
		return false, err
	}
	if n != 0 {
		return false, ErrState
	}
	if err = tx.QueryRowContext(ctx, `SELECT
  (SELECT COUNT(*) FROM network_address_fences WHERE bridge=? AND ((family='ipv4' AND address=?) OR (family='ipv6' AND address=?) OR (project!=? AND state!='RELEASED')))+
  (SELECT COUNT(*) FROM network_bindings WHERE bridge=? AND (mac=? OR ipv4=? OR ipv6=? OR (project!=? AND state!='RELEASED')))`, b.Bridge, b.IPv4, b.IPv6, project, b.Bridge, b.MAC, b.IPv4, b.IPv6, project).Scan(&n); err != nil {
		return false, err
	}
	if n != 0 {
		return false, nil
	}
	keys := b.reservationKeys()
	for _, key := range keys[:3] {
		if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE resource_key=?", key).Scan(&n); err != nil {
			return false, err
		}
		if n != 0 {
			return false, nil
		}
	}
	for _, key := range keys {
		if _, err = tx.ExecContext(ctx, "INSERT INTO operation_resource_claims(operation_id,layer,resource_key) VALUES(?,?,?)", id, project, key); err != nil {
			return false, err
		}
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'DISPATCHING','CREATE_STATIC_ADDRESSES_RESERVED')", id, time.Now().UnixNano()); err != nil {
		return false, err
	}
	return true, tx.Commit()
}
func requireCreateBindingReservation(ctx context.Context, tx *sql.Tx, id, project string, b VerifiedCreateBinding) error {
	var n int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", id).Scan(&n); err != nil {
		return err
	}
	if n != 4 {
		return ErrResourceLocked
	}
	for _, k := range b.reservationKeys() {
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND layer=? AND resource_key=?", id, project, k).Scan(&n); err != nil {
			return err
		}
		if n != 1 {
			return ErrResourceLocked
		}
	}
	return nil
}

// insertVerifiedCreateBinding is part of the atomic original Create
// finalization. All inserts are exclusive, never upserts; even RELEASED
// address fences are not reassigned. Any failure rolls back ownership and
// terminal success together. The worker preserves UNKNOWN on uncertainty.
func insertVerifiedCreateBinding(ctx context.Context, tx *sql.Tx, opID, project, uuid string, b VerifiedCreateBinding, now int64) error {
	if !b.valid() || !auditName(project) || project == "default" || uuid == "" {
		return ErrInvalid
	}
	if err := requireCreateBindingReservation(ctx, tx, opID, project, b); err != nil {
		return err
	}
	var overlap int
	if err := tx.QueryRowContext(ctx, `SELECT
        (SELECT COUNT(*) FROM network_bindings WHERE bridge=? AND project!=? AND state!='RELEASED')+
        (SELECT COUNT(*) FROM network_address_fences WHERE bridge=? AND project!=? AND state!='RELEASED')`,
		b.Bridge, project, b.Bridge, project).Scan(&overlap); err != nil {
		return err
	}
	if overlap != 0 {
		return ErrResourceLocked
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO network_bindings
        (project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at)
        VALUES(?,?,?,'eth0',?,?,?,?,?,'PINNED',?)`,
		project, uuid, uuid, b.Bridge, b.ACL, b.MAC, b.IPv4, b.IPv6, now); err != nil {
		return err
	}
	for _, p := range []struct{ family, address string }{{"ipv4", b.IPv4}, {"ipv6", b.IPv6}} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO network_address_fences
            (bridge,family,address,project,box_uuid,incarnation,state,operation_id,observed_at)
            VALUES(?,?,?,?,?,?,'RESERVED',?,?)`,
			b.Bridge, p.family, p.address, project, uuid, uuid, opID, now); err != nil {
			return err
		}
	}
	return nil
}
