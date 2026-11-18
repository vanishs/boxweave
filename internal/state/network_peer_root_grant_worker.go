package state

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// FinalizeRootPeerGrant commits the exact staged directional grant and the
// verified original atomically. Call ONLY after installed PRS has returned
// durable COMPLETE_VERIFIED with independently checked per-NIC kernel state;
// SQLite staging and SDK ACL GET alone are never authorization to finalize.
func (s *Store) FinalizeRootPeerGrant(ctx context.Context, original string) error {
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
	if action != "box_peer_set" || status != "VERIFYING" {
		return ErrState
	}
	var n int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM peer_grants WHERE operation_id=? AND desired=1 AND status='PENDING'", original).Scan(&n); err != nil || n != 1 {
		return ErrResourceLocked
	}
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM peer_grants WHERE operation_id=?", original).Scan(&n); err != nil || n != 1 {
		return ErrResourceLocked
	}
	result, err := tx.ExecContext(ctx, `UPDATE peer_grants SET status='CONFIRMED',desired=1,operation_id=''
 WHERE operation_id=? AND desired=1 AND status='PENDING'`, original)
	if err != nil {
		return err
	}
	n64, err := result.RowsAffected()
	if err != nil || n64 != 1 {
		return ErrResourceLocked
	}
	now := time.Now().UnixNano()
	result, err = tx.ExecContext(ctx, `UPDATE operations SET state='SUCCEEDED',error_class='',updated_at=?
 WHERE id=? AND state='VERIFYING' AND action='box_peer_set'`, now, original)
	if err != nil {
		return err
	}
	n64, err = result.RowsAffected()
	if err != nil || n64 != 1 {
		return ErrState
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO operation_events(operation_id,at,state,safe_detail)
 VALUES(?,?,'SUCCEEDED','PEER_ROOT_SCOPED_GRANT_KERNEL_VERIFIED')`, original, now); err != nil {
		return err
	}
	return tx.Commit()
}

// FinalizeCompletedUnknownRootPeerGrant only runs after independent root
// proof of a COMPLETE_VERIFIED original; no native effect may be replayed.
func (s *Store) FinalizeCompletedUnknownRootPeerGrant(ctx context.Context, original string) error {
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
	var n int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM peer_grants WHERE operation_id=? AND desired=1 AND status='PENDING'", original).Scan(&n); err != nil || n != 1 {
		return ErrResourceLocked
	}
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM peer_grants WHERE operation_id=?", original).Scan(&n); err != nil || n != 1 {
		return ErrResourceLocked
	}
	result, err := tx.ExecContext(ctx, `UPDATE peer_grants SET status='CONFIRMED',desired=1,operation_id=''
 WHERE operation_id=? AND desired=1 AND status='PENDING'`, original)
	if err != nil {
		return err
	}
	n64, err := result.RowsAffected()
	if err != nil || n64 != 1 {
		return ErrResourceLocked
	}
	now := time.Now().UnixNano()
	result, err = tx.ExecContext(ctx, `UPDATE operations SET state='SUCCEEDED',error_class='',updated_at=?
 WHERE id=? AND state='UNKNOWN' AND action='box_peer_set'`, now, original)
	if err != nil {
		return err
	}
	n64, err = result.RowsAffected()
	if err != nil || n64 != 1 {
		return ErrState
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO operation_events(operation_id,at,state,safe_detail)
 VALUES(?,?,'SUCCEEDED','PEER_ROOT_COMPLETED_UNKNOWN_GRANT_READONLY_NO_REPLAY')`, original, now); err != nil {
		return err
	}
	return tx.Commit()
}

// ExecutePreparedRootPeerGrant invokes only one already-claimed original
// through the installed operator-approved local PRS socket. It deliberately
// cannot create a peer from an unclaimed/request-selected native tuple.
// Any lost receipt/failed preview makes the original UNKNOWN and preserves
// ACL+Box locks; no automatic replay or release is possible.
func (w Worker) ExecutePreparedRootPeerGrant(original string, call RootPeerCall) error {
	if w.Store == nil || call == nil {
		return ErrInvalid
	}
	rec, err := w.Store.Get(context.Background(), original)
	if err != nil {
		return err
	}
	if rec.Action != "box_peer_set" || rec.State != "DISPATCHING" {
		return ErrState
	}
	if w.Budget <= 0 {
		w.Budget = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), w.Budget)
	defer cancel()
	unknown := func(reason string) {
		_ = w.Store.Unknown(context.Background(), original, "DISPATCHING", reason)
		_ = w.Store.Unknown(context.Background(), original, "VERIFYING", reason)
	}
	review, err := call(ctx, original, "")
	if err != nil || review.Status != "REVIEW_ONLY" || len(review.PlanSHA256) != 64 ||
		strings.Trim(review.PlanSHA256, "0123456789abcdef") != "" {
		unknown("PEER_ROOT_GRANT_REVIEW_UNVERIFIED")
		return ErrResourceLocked
	}
	result, err := call(ctx, original, review.PlanSHA256)
	if err != nil || result.Status != "COMPLETE_VERIFIED" ||
		result.PlanSHA256 != review.PlanSHA256 || !result.MutationComplete ||
		result.ReadyForPeer || result.ProbedScopes != 4 ||
		result.DeletedScopes < 0 || result.DeletedScopes > 4 ||
		!result.KernelVerified || result.VerifiedNICs < 2 ||
		len(result.PostNativeDigestSHA256) != 64 {
		unknown("PEER_ROOT_GRANT_NATIVE_ACK_UNCERTAIN")
		return ErrResourceLocked
	}
	if err := w.Store.AckSynchronousACL(context.Background(), original); err != nil {
		unknown("PEER_ROOT_GRANT_ACK_UNCERTAIN")
		return err
	}
	if err := w.Store.FinalizeRootPeerGrant(context.Background(), original); err != nil {
		unknown("PEER_ROOT_GRANT_SQLITE_COMMIT_UNCERTAIN")
		return err
	}
	return nil
}
