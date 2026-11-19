package state

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// FinalizeCompletedUnknownRootPeerRevocation is the nonroot controller's
// single SQLite transaction AFTER a distinct READ-ONLY host-root replay-free
// witness. The original journal, unknown intent and operation ID are never
// removed, reset, or reexecuted. It is NOT an automatic recovery worker.
func (s *Store) FinalizeCompletedUnknownRootPeerRevocation(ctx context.Context, original string) error {
	if s == nil || s.db == nil || s.auditReadOnly || !validHexID(original) {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var action, status string
	if err := tx.QueryRowContext(ctx, "SELECT action,state FROM operations WHERE id=?", original).Scan(&action, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if action != "box_peer_set" || status != "UNKNOWN" {
		return ErrState
	}
	var count int
	if err := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM peer_grants WHERE operation_id=? AND desired=0 AND status='PENDING'", original).Scan(&count); err != nil || count != 1 {
		return ErrResourceLocked
	}
	effect, err := tx.ExecContext(ctx,
		"UPDATE peer_grants SET status='REVOKED',desired=0,last_verified_operation_id=? WHERE operation_id=? AND desired=0 AND status='PENDING'", original, original)
	if err != nil {
		return err
	}
	n, err := effect.RowsAffected()
	if err != nil || n != 1 {
		return ErrResourceLocked
	}
	now := time.Now().UnixNano()
	effect, err = tx.ExecContext(ctx,
		"UPDATE operations SET state='SUCCEEDED',error_class='',updated_at=? WHERE id=? AND state='UNKNOWN' AND action='box_peer_set'", now, original)
	if err != nil {
		return err
	}
	n, err = effect.RowsAffected()
	if err != nil || n != 1 {
		return ErrState
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'SUCCEEDED','ROOT_COMPLETED_UNKNOWN_RECONCILED_NO_REPLAY')", original, now); err != nil {
		return err
	}
	return tx.Commit()
}

// ReconcileCompletedUnknownRootPeer is a one-off, explicitly authorized
// recovery request for a root journal ALREADY COMPLETE, not for a partially
// mutated / DISPATCH_UNKNOWN journal. The callback must be an independently
// credential-gated native root read-only verifier in product code.
func (w Worker) ReconcileCompletedUnknownRootPeer(original, sha string, call RootPeerCall) error {
	if w.Store == nil || call == nil || len(sha) != 64 ||
		strings.Trim(sha, "0123456789abcdef") != "" {
		return ErrInvalid
	}
	rec, err := w.Store.Get(context.Background(), original)
	if err != nil {
		return err
	}
	if rec.Action != "box_peer_set" || rec.State != "UNKNOWN" {
		return ErrState
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	proof, err := call(ctx, original, sha)
	if err != nil || proof.Status != "RECONCILE_VERIFIED" || proof.PlanSHA256 != sha ||
		!proof.MutationComplete || proof.ReadyForPeer || proof.ProbedScopes != 4 ||
		proof.DeletedScopes < 0 || proof.DeletedScopes > 4 ||
		!proof.KernelVerified || proof.VerifiedNICs < 2 ||
		len(proof.PostNativeDigestSHA256) != 64 {
		return ErrResourceLocked
	}
	// Only the canonical pending ledger selects the settlement kind. The
	// original stays UNKNOWN if the root proof or transaction is incomplete.
	var desired int
	err = w.Store.db.QueryRowContext(context.Background(),
		"SELECT desired FROM peer_grants WHERE operation_id=? AND status='PENDING'", original).Scan(&desired)
	if err != nil {
		return ErrResourceLocked
	}
	if desired == 0 {
		return w.Store.FinalizeCompletedUnknownRootPeerRevocation(context.Background(), original)
	}
	if desired == 1 {
		return w.Store.FinalizeCompletedUnknownRootPeerGrant(context.Background(), original)
	}
	return ErrResourceLocked
}
