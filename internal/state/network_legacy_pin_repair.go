package state

import (
	"context"
	"database/sql"
)

// Only Legacy may exclude archived IPv4 collisions, after the original
// history and compaction checks in the caller's same transaction.
func checkLegacyPinRepairCollisionsTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget, pin VerifiedCreateBinding) error {
	if e := checkPinRepairBindingCollisionsTx(ctx, tx, t, pin, legacyPinRepairIPv4Collision); e != nil {
		return e
	}
	for i, key := range pin.reservationKeys()[:3] {
		sqlText := pinRepairClaimQuery
		args := []any{key, t.ID}
		if i == 0 {
			sqlText += legacyPinRepairClaimSuffix
			args = append(args, t.Layer, pin.Bridge, pin.IPv4)
		}
		var n int
		if e := tx.QueryRowContext(ctx, sqlText, args...).Scan(&n); e != nil {
			return e
		}
		if n != 0 {
			return ErrResourceLocked
		}
	}
	return nil
}
