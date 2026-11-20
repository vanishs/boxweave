package state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// IsDependentDelete is a private-ledger evidence read, not an unlock.
func (s *Store) IsDependentDelete(ctx context.Context, id string) (bool, error) {
	var n int
	e := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims c JOIN operations o ON o.id=c.operation_id AND o.layer=c.layer WHERE o.id=? AND o.action='box_delete' AND c.resource_key='delete-project:'||o.layer", id).Scan(&n)
	return n == 1, e
}

// FinalizeReviewedUnknownDelete is called only by the privileged local
// recovery worker after native terminal SUCCESS and independent exact absence.
// A missing/expired native ID cannot take this path. The challenge is bound
// to a distinct success observation namespace. All settlement and audit
// writes occur in a single transaction, with no native mutation.
func (s *Store) FinalizeReviewedUnknownDelete(ctx context.Context, r RecoverySnapshot, reviewer, challenge, confirmUUID, reason, observed string, acknowledge bool) error {
	if r.Action != "box_delete" || r.State != "UNKNOWN" || r.NativeOperationID == "" || r.Released || !acknowledge || confirmUUID != r.BoxUUID || !safeReviewer.MatchString(reviewer) || len(challenge) != 64 || len(reason) < 16 || len(reason) > 240 || strings.ContainsAny(reason, "\x00\n\r") || !strings.HasPrefix(observed, "DELETE_SUCCESS:") {
		return ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var got RecoverySnapshot
	e = tx.QueryRowContext(ctx, "SELECT id,tracking_id,state,action,intent_digest,native_operation_id,box_uuid,incarnation,layer,updated_at FROM operations WHERE id=?", r.OperationID).Scan(&got.OperationID, &got.TrackingID, &got.State, &got.Action, &got.IntentDigest, &got.NativeOperationID, &got.BoxUUID, &got.Incarnation, &got.Project, &got.UpdatedAt)
	if e != nil {
		return e
	}
	got.Name = r.Name
	if got != r {
		return ErrConflict
	}
	var released int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_audit WHERE operation_id=? AND decision='RELEASED'", r.OperationID).Scan(&released); e != nil || released != 0 {
		return ErrResourceLocked
	}
	var at int64
	var who, digest string
	if e = tx.QueryRowContext(ctx, "SELECT at,reviewer,evidence_digest FROM recovery_audit WHERE operation_id=? AND decision='CHALLENGE_ISSUED' ORDER BY seq DESC LIMIT 1", r.OperationID).Scan(&at, &who, &digest); e != nil {
		return e
	}
	if who != reviewer || digest != recoveryChallengeHash(challenge, reviewer, r, observed) || time.Since(time.Unix(0, at)) < 0 || time.Since(time.Unix(0, at)) > 5*time.Minute {
		return ErrConflict
	}
	b := OwnedBox{UUID: r.BoxUUID, Project: r.Project, Name: r.Name, Incarnation: r.Incarnation, State: "OWNED"}
	if e = finishVerifiedDeleteTx(ctx, tx, r.OperationID, b, "UNKNOWN", "DELETE_REVIEWED_TERMINAL_NO_REPLAY"); e != nil {
		return e
	}
	hash := sha256.Sum256([]byte(reason + "|" + observed))
	if _, e = tx.ExecContext(ctx, "INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,?,?,'DELETE_CONFIRMED',?)", r.OperationID, time.Now().UnixNano(), reviewer, hex.EncodeToString(hash[:])); e != nil {
		return e
	}
	return tx.Commit()
}
