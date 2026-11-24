package state

import (
	"context"
	"database/sql"
	"time"
)

// CheckRecoveryCreateBinding is a read-only preflight for the exact staged
// original. The same checks run again under the recovery writer transaction.
func (s *Store) CheckRecoveryCreateBinding(ctx context.Context, t RecoveryTarget, p RecoveryProof) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := readRecoveryTarget(ctx, tx, t.Layer, t.ID)
	if err != nil {
		return err
	}
	if current != t {
		return ErrConflict
	}
	return checkRecoveryCreateTx(ctx, tx, t, p)
}

func checkRecoveryCreateTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget, p RecoveryProof) error {
	schema, err := scopedCreateIntentSchema(ctx, tx, t.ID)
	if err != nil {
		return err
	}
	// Scoped recovery must prove Root ACL and boot journal, never settle
	// a quarantined v2 Create via the legacy pinned-NIC verifier.
	if schema != 0 {
		return ErrResourceLocked
	}
	b := p.CreateBinding
	if t.Action != "box_create" || t.State != "UNKNOWN" || t.Released || !t.CreatePinStaged ||
		t.UUID != "" || t.Incarnation != "" || p.NativeID != t.NativeID || !validCreateRecoveryReason(t, p) ||
		!validActivityBoxID(p.UUID) || !b.valid() || t.Name == "" || t.Resource != "box:"+t.Name {
		return ErrResourceLocked
	}
	if err := requireNoIPv4Compaction(ctx, tx, t.Layer); err != nil {
		return err
	}
	if err := verifyIPv4RetirementHistory(ctx, tx, t.Layer, b.Bridge, b.ACL); err != nil {
		return err
	}
	if err := requireCreateBindingReservation(ctx, tx, t.ID, t.Layer, b); err != nil {
		return err
	}
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT
	 (SELECT COUNT(*) FROM owned_boxes WHERE uuid=? OR (project=? AND name=? AND state='OWNED'))+
	 (SELECT COUNT(*) FROM network_address_fences WHERE bridge=? AND ((family='ipv4' AND address=?) OR (family='ipv6' AND address=?) OR (project!=? AND state!='RELEASED')))+
	 (SELECT COUNT(*) FROM network_bindings n WHERE bridge=? AND (mac=? OR (ipv4=? AND NOT EXISTS (SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.project=n.project AND r.box_uuid=n.box_uuid AND r.incarnation=n.incarnation)) OR ipv6=? OR (project!=? AND state!='RELEASED')))`, p.UUID, t.Layer, t.Name, b.Bridge, b.IPv4, b.IPv6, t.Layer, b.Bridge, b.MAC, b.IPv4, t.Layer, b.IPv6, t.Layer).Scan(&n); err != nil {
		return err
	}
	if n != 0 {
		return ErrResourceLocked
	}
	for i, key := range b.reservationKeys()[:3] {
		query := "SELECT COUNT(*) FROM operation_resource_claims c WHERE resource_key=? AND operation_id!=?"
		args := []any{key, t.ID}
		if i == 0 {
			query += " AND NOT EXISTS (SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.bridge=? AND r.ipv4=? AND r.create_operation_id=c.operation_id AND c.layer=r.project)"
			args = append(args, t.Layer, b.Bridge, b.IPv4)
		}
		if err := tx.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			return ErrResourceLocked
		}
	}
	return nil
}

// Do not allow a known-receipt proof to represent an absent ACK, or
// fabricate a native receipt ID to make later historical audits pass.
func validCreateRecoveryReason(t RecoveryTarget, p RecoveryProof) bool {
	if t.NativeID == "" {
		return p.Reason == "CREATE_ACK_LOST_STATIC_POSTSTATE_VERIFIED"
	}
	return p.Reason == "CREATE_ORIGINAL_POSTSTATE_VERIFIED"
}

func finishRecoveryCreateTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget, p RecoveryProof) error {
	if err := checkRecoveryCreateTx(ctx, tx, t, p); err != nil {
		return err
	}
	now := time.Now().UnixNano()
	if _, err := tx.ExecContext(ctx, "INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,'OWNED',?)", p.UUID, t.Layer, t.Name, p.UUID, now); err != nil {
		return err
	}
	if err := insertVerifiedCreateBinding(ctx, tx, t.ID, t.Layer, p.UUID, p.CreateBinding, now); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, "UPDATE operations SET box_uuid=?,incarnation=?,state='SUCCEEDED',error_class='',updated_at=? WHERE id=? AND state='UNKNOWN' AND box_uuid='' AND incarnation='' AND updated_at=?", p.UUID, p.UUID, now, t.ID, t.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return ErrConflict
	}
	detail := "RECOVERY_CREATE_PIN_ATOMIC_NO_REPLAY"
	if t.NativeID == "" {
		detail = "RECOVERY_CREATE_ACK_LOST_PIN_ATOMIC_NO_REPLAY"
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'SUCCEEDED',?)", t.ID, now, detail)
	return err
}
