package state

import (
	"context"
	"database/sql"
)

// During schema consolidation, the Scoped policy refuses any old retirement
// history. Legacy retains its original Project compaction fence. No path
// treats an old reservation as free or ignores a pending compaction.
func requireNetworkAdmissionTx(ctx context.Context, tx *sql.Tx, project string, scoped bool) error {
	if scoped {
		return requireScopedNoLegacyRetirementTx(ctx, tx)
	}
	return requireNoIPv4Compaction(ctx, tx, project)
}

// The original must be re-read and compared inside this transaction.
// A durable Scoped Peer or lifecycle original cannot use the Legacy fence.
func requireOriginalRecoveryAdmissionTx(ctx context.Context, tx *sql.Tx, original RecoveryTarget, scoped bool) error {
	return requireNetworkAdmissionTx(ctx, tx, original.Layer, scoped || original.ScopedPeer || original.ScopedLifecycle)
}
