package state

import "context"

// Removed schema-6 execution. Historical caller signatures refuse before any SQL or native witness.

func (s *Store) CheckRecoveryCreateBinding(ctx context.Context, t RecoveryTarget, p RecoveryProof) error {
	return ErrResourceLocked
}
