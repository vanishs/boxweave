package state

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// RootPeerResult is a strictly limited local transport acknowledgement.
// It is NOT evidence of kernel nft convergence or re-authorized peer access.
type RootPeerResult struct {
	Status           string
	PlanSHA256       string
	ProbedScopes     int
	DeletedScopes    int
	MutationComplete bool
	ReadyForPeer     bool
}

type RootPeerCall func(context.Context, string, string) (RootPeerResult, error)
type RootPeerVerifier func(context.Context, RootPeerResult) error

// FinalizeRootPeerRevocation marks BOTH the original verified operation and
// the exact pending peer_grant revoked in ONE SQLite transaction. Native root
// completion alone is insufficient: controller verifier must independently
// have checked the per-NIC kernel deny state before calling this method.
func (s *Store) FinalizeRootPeerRevocation(ctx context.Context, original string) error {
	if s == nil || s.db == nil || s.auditReadOnly || !validHexID(original) {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var action, status string
	if err := tx.QueryRowContext(ctx, "SELECT action,state FROM operations WHERE id=?", original).
		Scan(&action, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if action != "box_peer_set" || status != "VERIFYING" {
		return ErrState
	}
	result, err := tx.ExecContext(ctx, `UPDATE peer_grants
		SET status='REVOKED',desired=0
		WHERE operation_id=? AND desired=0 AND status='PENDING'`, original)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return ErrResourceLocked
	}
	now := time.Now().UnixNano()
	result, err = tx.ExecContext(ctx, `UPDATE operations SET state='SUCCEEDED',error_class='',updated_at=?
		WHERE id=? AND state='VERIFYING' AND action='box_peer_set'`, now, original)
	if err != nil {
		return err
	}
	affected, err = result.RowsAffected()
	if err != nil || affected != 1 {
		return ErrState
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO operation_events(operation_id,at,state,safe_detail)
		VALUES(?,?,'SUCCEEDED','PEER_ROOT_SCOPED_REVOCATION_VERIFIED')`, original, now); err != nil {
		return err
	}
	return tx.Commit()
}

// ExecuteRootPeerRevocation is the installed controller's typed one-shot
// bridge. It cannot submit arbitrary native parameters: root sees only the
// immutable operation ID, then independently derives a reviewed plan SHA.
// If the socket reply is lost after effect dispatch, the SQLite original is
// marked UNKNOWN, retained under its ACL+two Box claims, and NEVER retried.
func (w Worker) ExecuteRootPeerRevocation(original string, isolate SynchronousACLIsolation,
	call RootPeerCall, verify RootPeerVerifier) {
	if w.Store == nil || isolate == nil || call == nil || verify == nil {
		return
	}
	rec, err := w.Store.Get(context.Background(), original)
	if err != nil || rec.Action != "box_peer_set" || rec.State != "ADMITTED" {
		return
	}
	if w.Budget <= 0 {
		w.Budget = 3 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), w.Budget)
	defer cancel()
	if err := w.Store.StageRootPeerRevocation(ctx, original); err != nil {
		return
	}
	if err := w.Store.Claim(ctx, original); err != nil {
		return
	}
	unknown := func(reason string) {
		_ = w.Store.Unknown(context.Background(), original, "DISPATCHING", reason)
		_ = w.Store.Unknown(context.Background(), original, "VERIFYING", reason)
	}
	if err := isolate(ctx); err != nil {
		unknown("PEER_ROOT_QUARANTINE_UNVERIFIED")
		return
	}
	plan, err := call(ctx, original, "")
	if err != nil || plan.Status != "REVIEW_ONLY" || len(plan.PlanSHA256) != 64 ||
		strings.Trim(plan.PlanSHA256, "0123456789abcdef") != "" {
		unknown("PEER_ROOT_REVIEW_UNVERIFIED")
		return
	}
	result, err := call(ctx, original, plan.PlanSHA256)
	if err != nil || result.Status != "COMPLETE_VERIFIED" ||
		result.PlanSHA256 != plan.PlanSHA256 || !result.MutationComplete ||
		result.ReadyForPeer || result.ProbedScopes != 4 ||
		result.DeletedScopes < 0 || result.DeletedScopes > 4 {
		unknown("PEER_ROOT_EFFECT_ACK_UNCERTAIN")
		return
	}
	if err := w.Store.AckSynchronousACL(context.Background(), original); err != nil {
		unknown("PEER_ROOT_SQLITE_ACK_UNVERIFIED")
		return
	}
	if err := verify(ctx, result); err != nil {
		unknown("PEER_ROOT_NFT_AND_FOREIGN_PACKET_READBACK_UNVERIFIED")
		return
	}
	if err := w.Store.FinalizeRootPeerRevocation(context.Background(), original); err != nil {
		unknown("PEER_ROOT_FINAL_REVOKE_DURABILITY_UNVERIFIED")
	}
}

func (w Worker) SubmitRootPeerRevocation(original string, isolate SynchronousACLIsolation,
	call RootPeerCall, verify RootPeerVerifier) error {
	if w.Store == nil || isolate == nil || call == nil || verify == nil {
		return errors.New("root peer controller missing isolation or native verifier")
	}
	rec, err := w.Store.Get(context.Background(), original)
	if err != nil {
		return err
	}
	if rec.Action != "box_peer_set" || rec.State != "ADMITTED" {
		return ErrState
	}
	go w.ExecuteRootPeerRevocation(original, isolate, call, verify)
	return nil
}
