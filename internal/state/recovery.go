package state

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Local recovery is intentionally limited to exact registered owned Boxes.
// An UNKNOWN Box create with no proven UUID must NOT be auto-adopted or unlocked.
type RecoverySnapshot struct {
	OperationID       string `json:"operation_id"`
	TrackingID        string `json:"tracking_id"`
	State             string `json:"state"`
	Action            string `json:"action"`
	IntentDigest      string `json:"intent_digest"`
	NativeOperationID string `json:"native_operation_id"`
	BoxUUID           string `json:"box_uuid"`
	Incarnation       string `json:"incarnation"`
	Project           string `json:"project"`
	Name              string `json:"name"`
	UpdatedAt         int64  `json:"-"`
	Released          bool   `json:"claim_released"`
}

var safeReviewer = regexp.MustCompile(`^[a-zA-Z0-9_.-]{2,64}$`)

func (s *Store) RegisterOwnedBox(ctx context.Context, uuid, project, name, incarnation string) error {
	if uuid == "" || project == "" || name == "" || incarnation == "" ||
		len(uuid) > 128 || len(project) > 128 || len(name) > 128 || len(incarnation) > 128 {
		return ErrInvalid
	}
	// Domain callers must independently prove the native UUID/Project before
	// registration. This method is not exposed as a public MCP tool.
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO owned_boxes (uuid,project,name,incarnation,state,created_at) VALUES (?,?,?,?,?,?)",
		uuid, project, name, incarnation, "OWNED", time.Now().UnixNano())
	return err
}

func (s *Store) RecoveryInspect(ctx context.Context, operationID string) (RecoverySnapshot, error) {
	var r RecoverySnapshot
	var released int
	err := s.db.QueryRowContext(ctx, `SELECT o.id,o.tracking_id,o.state,o.action,o.intent_digest,
		o.native_operation_id,o.box_uuid,o.incarnation,b.project,b.name,o.updated_at,
		(SELECT COUNT(*) FROM recovery_audit a WHERE a.operation_id=o.id AND a.decision='RELEASED')
		FROM operations o JOIN owned_boxes b
		  ON b.uuid=o.box_uuid AND b.incarnation=o.incarnation
		WHERE o.id=?`,operationID).Scan(&r.OperationID,&r.TrackingID,&r.State,&r.Action,&r.IntentDigest,
		&r.NativeOperationID,&r.BoxUUID,&r.Incarnation,&r.Project,&r.Name,&r.UpdatedAt,&released)
	if errors.Is(err,sql.ErrNoRows) { return r,ErrNotFound }
	if err!=nil{return r,err}
	if r.State!="UNKNOWN" {return r,ErrState}
	r.Released=released>0
	return r,nil
}

func recoveryFingerprint(r RecoverySnapshot, observed string) string {
	// Bind to durable original identities and the latest native observation.
	// Never include raw authentication tokens or guest contents.
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%d|%s",
		r.OperationID,r.TrackingID,r.Action,r.IntentDigest,r.NativeOperationID,
		r.BoxUUID,r.Incarnation,r.Project,r.Name,r.UpdatedAt,observed)
}
func recoveryChallengeHash(challenge, reviewer string, r RecoverySnapshot, observed string) string {
	h:=sha256.Sum256([]byte("s1-local-recovery-v1|"+challenge+"|"+reviewer+"|"+recoveryFingerprint(r,observed)))
	return hex.EncodeToString(h[:])
}

// IssueRecoveryChallenge writes only a digest. It never releases a claim, nor
// decides whether the unknown native effect happened.
func (s *Store) IssueRecoveryChallenge(ctx context.Context, r RecoverySnapshot, reviewer, observed string) (string,error) {
	if !safeReviewer.MatchString(reviewer)||observed==""||r.State!="UNKNOWN"||r.Released {
		return "",ErrInvalid
	}
	b:=make([]byte,32)
	if _,err:=rand.Read(b);err!=nil{return "",err}
	challenge:=hex.EncodeToString(b)
	tx,err:=s.db.BeginTx(ctx,nil);if err!=nil{return "",err}
	defer tx.Rollback()
	var updated int64
	var state string
	if err=tx.QueryRowContext(ctx,"SELECT state,updated_at FROM operations WHERE id=?",r.OperationID).Scan(&state,&updated);err!=nil{return "",err}
	if state!="UNKNOWN"||updated!=r.UpdatedAt{return "",ErrState}
	var released int
	if err=tx.QueryRowContext(ctx,"SELECT COUNT(*) FROM recovery_audit WHERE operation_id=? AND decision='RELEASED'",r.OperationID).Scan(&released);err!=nil{return "",err}
	if released!=0{return "",ErrState}
	_,err=tx.ExecContext(ctx,"INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES (?,?,?,?,?)",
		r.OperationID,time.Now().UnixNano(),reviewer,"CHALLENGE_ISSUED",recoveryChallengeHash(challenge,reviewer,r,observed))
	if err!=nil{return "",err}
	return challenge,tx.Commit()
}

// ReleaseRecoveryClaim requires a fresh challenge, exact UUID confirmation,
// separate acknowledgement, and independently repeated native observation.
// The original UNKNOWN receipt and original idempotency key NEVER change.
func (s *Store) ReleaseRecoveryClaim(ctx context.Context,r RecoverySnapshot,reviewer,challenge,confirmUUID,reason,observed string,acknowledge bool) error {
	if !safeReviewer.MatchString(reviewer)||!acknowledge||confirmUUID!=r.BoxUUID||
		len(challenge)!=64||len(reason)<16||len(reason)>240||observed==""||r.Released{
		return ErrInvalid
	}
	tx,err:=s.db.BeginTx(ctx,nil);if err!=nil{return err}
	defer tx.Rollback()
	var state string
	var updated int64
	if err=tx.QueryRowContext(ctx,"SELECT state,updated_at FROM operations WHERE id=?",r.OperationID).Scan(&state,&updated);err!=nil{return err}
	if state!="UNKNOWN"||updated!=r.UpdatedAt{return ErrState}
	var released int
	if err=tx.QueryRowContext(ctx,"SELECT COUNT(*) FROM recovery_audit WHERE operation_id=? AND decision='RELEASED'",r.OperationID).Scan(&released);err!=nil{return err}
	if released!=0{return ErrState}
	var at int64
	var digest,issuedReviewer string
	err=tx.QueryRowContext(ctx,"SELECT at,reviewer,evidence_digest FROM recovery_audit WHERE operation_id=? AND decision='CHALLENGE_ISSUED' ORDER BY seq DESC LIMIT 1",r.OperationID).Scan(&at,&issuedReviewer,&digest)
	if err!=nil{return err}
	if issuedReviewer!=reviewer||digest!=recoveryChallengeHash(challenge,reviewer,r,observed)||
		time.Since(time.Unix(0,at))<0||time.Since(time.Unix(0,at))>5*time.Minute {
		return ErrConflict
	}
	now:=time.Now().UnixNano()
	reasonHash:=sha256.Sum256([]byte(reason))
	if _,err=tx.ExecContext(ctx,"INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES (?,?,?,?,?)",
		r.OperationID,now,reviewer,"RELEASED",hex.EncodeToString(reasonHash[:]));err!=nil{return err}
	if _,err=tx.ExecContext(ctx,"INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES (?,?,?,?)",
		r.OperationID,now,"UNKNOWN","CLAIM_RELEASED_LOCAL_REVIEW_NO_REPLAY");err!=nil{return err}
	return tx.Commit()
}
