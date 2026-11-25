package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// AdmissionBlock is evidence of an actual existing claim in the admission
// transaction, not a current liveness guarantee or authorization to retry.
type AdmissionBlock struct {
	OriginalID string
	State      string
}

func (e AdmissionBlock) Error() string { return ErrResourceLocked.Error() }
func (e AdmissionBlock) Unwrap() error { return ErrResourceLocked }

var (
	ErrDependencyReview = fmt.Errorf("resource dependencies require review: %w", ErrResourceLocked)
	ErrOwnedName        = fmt.Errorf("owned name already in use: %w", ErrResourceLocked)
)

func newAdmissionBlock(id, status string) error {
	if !validHexID(id) || !validOperationState(status) {
		return ErrResourceLocked
	}
	return AdmissionBlock{OriginalID: id, State: status}
}

// Only fixed SQL with an exact layer predicate is supplied by Admit.
// Prioritize UNKNOWN if multiple claims match; never sample unrelated rows.
func admissionBlock(ctx context.Context, tx *sql.Tx, query string, args ...any) error {
	var id, status string
	e := tx.QueryRowContext(ctx, query+" ORDER BY (o.state='UNKNOWN') DESC,o.created_at,o.id LIMIT 1", args...).Scan(&id, &status)
	if errors.Is(e, sql.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	return newAdmissionBlock(id, status)
}
