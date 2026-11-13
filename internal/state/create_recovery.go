package state

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CreateRecoverySnapshot is a strictly host-local review of an original
// UNKNOWN Create whose native UUID was not yet bound to the durable receipt.
// The native witness must separately prove the original operation marker.
type CreateRecoverySnapshot struct {
	OperationID       string `json:"operation_id"`
	TrackingID        string `json:"tracking_id"`
	NativeOperationID string `json:"native_operation_id,omitempty"`
	Project           string `json:"project"`
	Name              string `json:"name"`
	IntentDigest      string `json:"intent_digest"`
	UpdatedAt         int64  `json:"-"`
}

func (s *Store) CreateRecoveryInspect(ctx context.Context, id string) (CreateRecoverySnapshot, error) {
	var r CreateRecoverySnapshot
	var action, phase, resource, boxUUID, incarnation string
	var safe []byte
	err := s.db.QueryRowContext(ctx, `SELECT id,tracking_id,native_operation_id,layer,
        intent_digest,updated_at,action,state,resource_key,box_uuid,incarnation,safe_intent
        FROM operations WHERE id=?`, id).Scan(
		&r.OperationID, &r.TrackingID, &r.NativeOperationID, &r.Project,
		&r.IntentDigest, &r.UpdatedAt, &action, &phase, &resource, &boxUUID, &incarnation, &safe)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	if action != "box_create" || phase != "UNKNOWN" || boxUUID != "" || incarnation != "" {
		return r, ErrState
	}
	var intent struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(safe, &intent); err != nil {
		return r, ErrInvalid
	}
	if intent.Name == "" || "box:"+intent.Name != resource {
		return r, ErrState
	}
	r.Name = intent.Name
	return r, nil
}

func createBindingDigest(r CreateRecoverySnapshot, reviewer, observed, challenge string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("create-unknown-bind-v1|%s|%s|%s|%s|%s|%s|%s|%d|%s|%s",
		r.OperationID, r.TrackingID, r.NativeOperationID, r.Project, r.Name, r.IntentDigest,
		reviewer, r.UpdatedAt, observed, challenge)))
	return hex.EncodeToString(h[:])
}

// IssueCreateBindingChallenge never claims a native operation succeeded and
// never releases the original UNKNOWN resource lock.
func (s *Store) IssueCreateBindingChallenge(ctx context.Context, r CreateRecoverySnapshot, reviewer, observed string) (string, error) {
	if !safeReviewer.MatchString(reviewer) || observed == "" {
		return "", ErrInvalid
	}
	latest, err := s.CreateRecoveryInspect(ctx, r.OperationID)
	if err != nil {
		return "", err
	}
	if latest != r {
		return "", ErrConflict
	}
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	challenge := hex.EncodeToString(bytes)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var updated int64
	var phase string
	if err := tx.QueryRowContext(ctx, "SELECT updated_at,state FROM operations WHERE id=?", r.OperationID).Scan(&updated, &phase); err != nil {
		return "", err
	}
	if phase != "UNKNOWN" || updated != r.UpdatedAt {
		return "", ErrConflict
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,?,?,?,?)",
		r.OperationID, time.Now().UnixNano(), reviewer, "CREATE_BIND_CHALLENGE_ISSUED", createBindingDigest(r, reviewer, observed, challenge))
	if err != nil {
		return "", err
	}
	return challenge, tx.Commit()
}

// BindUnknownCreatedBox links ONLY an exact, independently verified native
// UUID to the original UNKNOWN Create. It does not infer native success, retry,
// unlock the resource, or change the original immutable receipt state.
func (s *Store) BindUnknownCreatedBox(ctx context.Context, r CreateRecoverySnapshot, reviewer, challenge, uuid, observed, reason string, acknowledge bool) error {
	if !acknowledge || !safeReviewer.MatchString(reviewer) ||
		len(challenge) != 64 || uuid == "" || len(uuid) > 128 || observed == "" ||
		len(reason) < 16 || len(reason) > 240 || strings.ContainsAny(reason, "\x00\r\n") {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var updated int64
	var phase, origUUID, inc, layer, resource, nativeID, intentDigest string
	err = tx.QueryRowContext(ctx, `SELECT updated_at,state,box_uuid,incarnation,layer,resource_key,native_operation_id,intent_digest
         FROM operations WHERE id=? AND action='box_create'`, r.OperationID).Scan(
		&updated, &phase, &origUUID, &inc, &layer, &resource, &nativeID, &intentDigest)
	if err != nil {
		return err
	}
	if phase != "UNKNOWN" || origUUID != "" || inc != "" || updated != r.UpdatedAt ||
		layer != r.Project || resource != "box:"+r.Name || nativeID != r.NativeOperationID ||
		intentDigest != r.IntentDigest {
		return ErrConflict
	}
	var at int64
	var issuedReviewer, evidence string
	err = tx.QueryRowContext(ctx, `SELECT at,reviewer,evidence_digest FROM recovery_audit
        WHERE operation_id=? AND decision='CREATE_BIND_CHALLENGE_ISSUED'
        ORDER BY seq DESC LIMIT 1`, r.OperationID).Scan(&at, &issuedReviewer, &evidence)
	if err != nil {
		return err
	}
	if issuedReviewer != reviewer || evidence != createBindingDigest(r, reviewer, observed, challenge) ||
		time.Since(time.Unix(0, at)) < 0 || time.Since(time.Unix(0, at)) > 5*time.Minute {
		return ErrConflict
	}
	now := time.Now().UnixNano()
	if _, err = tx.ExecContext(ctx, `INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at)
        VALUES(?,?,?,?,'OWNED',?)`, uuid, r.Project, r.Name, uuid, now); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE operations SET box_uuid=?,incarnation=?,updated_at=?
        WHERE id=? AND state='UNKNOWN' AND box_uuid='' AND updated_at=?`,
		uuid, uuid, now, r.OperationID, r.UpdatedAt)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return ErrConflict
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO operation_events(operation_id,at,state,safe_detail)
        VALUES(?,?,?,?)`, r.OperationID, now, "UNKNOWN", "EXACT_NATIVE_UUID_BOUND_LOCAL_REVIEW_NO_REPLAY"); err != nil {
		return err
	}
	reasonHash := sha256.Sum256([]byte(reason))
	if _, err = tx.ExecContext(ctx, `INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest)
        VALUES(?,?,?,?,?)`, r.OperationID, now, reviewer, "CREATE_BOUND", hex.EncodeToString(reasonHash[:])); err != nil {
		return err
	}
	return tx.Commit()
}
