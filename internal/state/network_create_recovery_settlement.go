package state

import (
	"context"
	"database/sql"
	"time"
)

// settleRecoveredCreateTx is shared by the independently proved Scoped and Legacy recovery paths.
// Keep the original transaction, ownership and durable event semantics together.
func settleRecoveredCreateTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget, p RecoveryProof) error {
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
	if t.ScopedLifecycle {
		detail = "RECOVERY_SCOPED_CREATE_ATOMIC_NO_REPLAY"
	}
	if t.NativeID == "" {
		detail = "RECOVERY_CREATE_ACK_LOST_PIN_ATOMIC_NO_REPLAY"
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'SUCCEEDED',?)", t.ID, now, detail)
	return err
}
