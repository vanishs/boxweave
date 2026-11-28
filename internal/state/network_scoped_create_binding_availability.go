package state

import (
	"context"
	"database/sql"
)

// Availability is only an in-transaction preflight. The caller retains
// the same transaction, credential check, and exclusive claim insertion.
func scopedCreateBindingAvailableTx(ctx context.Context, tx *sql.Tx, project string, b VerifiedCreateBinding) (bool, error) {
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT
  (SELECT COUNT(*) FROM network_address_fences WHERE bridge=? AND ((family='ipv4' AND address=?) OR (family='ipv6' AND address=?) OR (project!=? AND state!='RELEASED')))+
  (SELECT COUNT(*) FROM network_bindings WHERE bridge=? AND (mac=? OR ipv4=? OR ipv6=? OR (project!=? AND state!='RELEASED')))`, b.Bridge, b.IPv4, b.IPv6, project, b.Bridge, b.MAC, b.IPv4, b.IPv6, project).Scan(&n); err != nil {
		return false, err
	}
	if n != 0 {
		return false, nil
	}
	for _, key := range b.reservationKeys()[:3] {
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE resource_key=?", key).Scan(&n); err != nil {
			return false, err
		}
		if n != 0 {
			return false, nil
		}
	}
	return true, nil
}
