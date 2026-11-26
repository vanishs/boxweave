package state

import (
	"context"
	"database/sql"
)

// Availability is only an in-transaction preflight. The caller retains
// the same transaction, credential check, and exclusive claim insertion.
func legacyCreateBindingAvailableTx(ctx context.Context, tx *sql.Tx, project string, b VerifiedCreateBinding) (bool, error) {
	if err := verifyIPv4RetirementHistory(ctx, tx, project, b.Bridge, b.ACL); err != nil {
		return false, err
	}
	var n int
	if err := tx.QueryRowContext(ctx, legacyCreateBindingCollisionQuery, b.Bridge, b.IPv4, b.IPv6, project, b.Bridge, b.MAC, b.IPv4, project, b.IPv6, project).Scan(&n); err != nil {
		return false, err
	}
	if n != 0 {
		return false, nil
	}
	for i, key := range b.reservationKeys()[:3] {
		query := "SELECT COUNT(*) FROM operation_resource_claims WHERE resource_key=?"
		args := []any{key}
		if i == 0 {
			query = legacyCreateClaimCollisionQuery
			args = append(args, project, b.Bridge, b.IPv4)
		}
		if err := tx.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
			return false, err
		}
		if n != 0 {
			return false, nil
		}
	}
	return true, nil
}
