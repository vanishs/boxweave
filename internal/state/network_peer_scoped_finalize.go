package state

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// ScopedPeerRootReceipt is a typed acknowledgment from the authenticated
// local Root worker, not a public request or permission to perform effects.
// Its proof digest binds the exact staged SQLite original. The worker must
// independently verify the complete native journal and live postimage.
type ScopedPeerRootReceipt struct {
	OriginalID, Project, Bridge, Status  string
	ProofSHA256, PlanSHA256, FinalSHA256 string
	VerifiedSteps                        int
	RecoveryOnly                         bool
}

func scopedPeerSHA(s string) bool { return len(s) == 64 && strings.Trim(s, "0123456789abcdef") == "" }

// FinalizeScopedPeerOriginal commits the exact policy row and its original
// operation together. A stale receipt, changed actor, changed sibling,
// or forged scope cannot settle a different original. No native effects.
func (s *Store) FinalizeScopedPeerOriginal(ctx context.Context, r ScopedPeerRootReceipt) error {
	if s == nil || s.db == nil || s.auditReadOnly || !validHexID(r.OriginalID) || r.Status != "COMPLETE_VERIFIED" ||
		r.VerifiedSteps != 4 || !scopedPeerSHA(r.ProofSHA256) || !scopedPeerSHA(r.PlanSHA256) || !scopedPeerSHA(r.FinalSHA256) {
		return ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = finalizeScopedPeerOriginalTx(ctx, tx, r); e != nil {
		return e
	}
	return tx.Commit()
}
func finalizeScopedPeerOriginalTx(ctx context.Context, tx *sql.Tx, r ScopedPeerRootReceipt) error {
	if !validHexID(r.OriginalID) || r.Status != "COMPLETE_VERIFIED" || r.VerifiedSteps != 4 || !scopedPeerSHA(r.ProofSHA256) || !scopedPeerSHA(r.PlanSHA256) || !scopedPeerSHA(r.FinalSHA256) {
		return ErrInvalid
	}
	phase := "DISPATCHING"
	if r.RecoveryOnly {
		phase = "UNKNOWN"
	}
	p, e := readScopedPeerOriginalTx(ctx, tx, r.OriginalID, r.Project, r.Bridge, phase)
	if e != nil || ScopedPeerProofSHA256(p) != r.ProofSHA256 {
		return ErrResourceLocked
	}
	i := p.Intent
	status, event, op, desired := "CONFIRMED", "SCOPED_PEER_V2_GRANT_KERNEL_VERIFIED", "", 1
	if !i.Allow {
		status, event, op, desired = "REVOKED", "SCOPED_PEER_V2_REVOCATION_KERNEL_VERIFIED", r.OriginalID, 0
	}
	res, e := tx.ExecContext(ctx, `UPDATE peer_grants SET status=?,operation_id=?,last_verified_operation_id=?
 WHERE project=? AND source_uuid=? AND source_incarnation=? AND target_uuid=? AND target_incarnation=?
 AND status='PENDING' AND desired=? AND operation_id=?`, status, op, r.OriginalID, i.Project, i.SourceUUID, i.SourceIncarnation, i.TargetUUID, i.TargetIncarnation, desired, r.OriginalID)
	if e != nil {
		return e
	}
	if n, e := res.RowsAffected(); e != nil || n != 1 {
		return ErrResourceLocked
	}
	now := time.Now().UnixNano()
	res, e = tx.ExecContext(ctx, "UPDATE operations SET state='SUCCEEDED',error_class='',updated_at=? WHERE id=? AND action='box_peer_set' AND state=? AND native_operation_id=''", now, r.OriginalID, phase)
	if e != nil {
		return e
	}
	if n, e := res.RowsAffected(); e != nil || n != 1 {
		return ErrState
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'SUCCEEDED',?)", r.OriginalID, now, event); e != nil {
		return e
	}
	if r.RecoveryOnly {
		if _, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'SUCCEEDED','SCOPED_PEER_V2_COMPLETED_UNKNOWN_NO_REPLAY')", r.OriginalID, now); e != nil {
			return e
		}
	}
	return nil
}

type ScopedPeerRootCall func(context.Context, string) (ScopedPeerRootReceipt, error)

// ExecuteScopedPeerOriginal is an internal worker boundary. No registered
// MCP caller exists until the installed Root executor is fully accepted.
func (w Worker) ExecuteScopedPeerOriginal(id string, call ScopedPeerRootCall) error {
	if w.Store == nil || call == nil {
		return ErrInvalid
	}
	budget := w.Budget
	if budget <= 0 {
		budget = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	if e := w.Store.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
		_ = w.Store.transition(context.Background(), id, "ADMITTED", "FAILED", "", "SCOPED_PEER_V2_PRE_DISPATCH_REFUSED")
		return e
	}
	r, e := call(ctx, id)
	if e == nil && r.OriginalID == id && !r.RecoveryOnly {
		e = w.Store.FinalizeScopedPeerOriginal(ctx, r)
	} else if e == nil {
		e = ErrResourceLocked
	}
	if e != nil {
		_ = w.Store.Unknown(context.Background(), id, "DISPATCHING", "SCOPED_PEER_V2_ROOT_OUTCOME_UNVERIFIED")
		return e
	}
	return nil
}
