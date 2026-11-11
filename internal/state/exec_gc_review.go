package state

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// ExecGCReview is a durable, exact original stream claim awaiting LOCAL
// operator inspection. It is never authorization to repeat a lost DELETE.
type ExecGCReview struct {
	OperationID       string
	Stream            string
	Project           string
	BoxUUID           string
	Incarnation       string
	Name              string
	NativeOperationID string
	Path              string
	State             string
	ClaimedAt         int64
	DecidedAt         int64
}

func (s *Store) ExecGCReviewInspect(ctx context.Context, opID, stream string) (ExecGCReview, error) {
	var r ExecGCReview
	if !validHexID(opID) || (stream != "stdout" && stream != "stderr") {
		return r, ErrInvalid
	}
	var opState, ownedState, action string
	err := s.db.QueryRowContext(ctx, "SELECT j.operation_id,j.stream,j.project,j.box_uuid,j.incarnation,"+
		"j.name,j.native_operation_id,j.path,j.state,j.claimed_at,j.decided_at,o.state,b.state,o.action "+
		"FROM exec_output_gc j JOIN operations o ON o.id=j.operation_id AND o.box_uuid=j.box_uuid AND o.incarnation=j.incarnation AND o.layer=j.project AND o.native_operation_id=j.native_operation_id "+
		"JOIN owned_boxes b ON b.uuid=j.box_uuid AND b.incarnation=j.incarnation AND b.project=j.project AND b.name=j.name "+
		"WHERE j.operation_id=? AND j.stream=?", opID, stream).Scan(
		&r.OperationID, &r.Stream, &r.Project, &r.BoxUUID, &r.Incarnation, &r.Name,
		&r.NativeOperationID, &r.Path, &r.State, &r.ClaimedAt, &r.DecidedAt,
		&opState, &ownedState, &action)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err != nil {
		return r, err
	}
	if r.State != "REVIEW_REQUIRED" || opState != "SUCCEEDED" || ownedState != "OWNED" ||
		action != "box_exec" || r.Project == "" || r.BoxUUID == "" || r.Incarnation == "" ||
		!exactExecGCPath(r.Name, r.NativeOperationID, stream, r.Path) {
		return ExecGCReview{}, ErrState
	}
	return r, nil
}

func gcReviewHash(challenge, reviewer string, r ExecGCReview, observed string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("s2-exec-gc-review-v1|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%d|%d|%s",
		challenge, reviewer, r.OperationID, r.Stream, r.Project, r.BoxUUID, r.Incarnation,
		r.Name, r.NativeOperationID, r.Path, r.ClaimedAt, r.DecidedAt, observed)))
	return hex.EncodeToString(h[:])
}

func validGCObservation(o string) bool { return o == "NATIVE_PRESENT" || o == "NATIVE_MISSING" }

// IssueExecGCReviewChallenge only persists a hash of one short-lived local
// challenge, bound to original IDs, reviewer, exact native read observation
// and prior immutable GC evidence. The challenge is never an MCP token.
func (s *Store) IssueExecGCReviewChallenge(ctx context.Context, r ExecGCReview, reviewer, observation string) (string, error) {
	if !safeReviewer.MatchString(reviewer) || !validGCObservation(observation) || r.State != "REVIEW_REQUIRED" {
		return "", ErrInvalid
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	challenge := hex.EncodeToString(b)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var st string
	var decided int64
	err = tx.QueryRowContext(ctx, "SELECT state,decided_at FROM exec_output_gc WHERE operation_id=? AND stream=?",
		r.OperationID, r.Stream).Scan(&st, &decided)
	if err != nil {
		return "", err
	}
	if st != "REVIEW_REQUIRED" || decided != r.DecidedAt {
		return "", ErrState
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO exec_output_gc_audit(operation_id,stream,at,event,reviewer,evidence_digest) "+
		"VALUES(?,?,?,?,?,?)", r.OperationID, r.Stream, time.Now().UnixNano(), "CHALLENGE_ISSUED",
		reviewer, gcReviewHash(challenge, reviewer, r, observation))
	if err != nil {
		return "", err
	}
	return challenge, tx.Commit()
}

// ReleaseExecGCReview releases only a coordination hold after independent
// root-reviewed re-observation. It NEVER sends a second native DELETE, never
// claims EXPIRED without the first DELETE+404 evidence, and keeps all audit.
func (s *Store) ReleaseExecGCReview(ctx context.Context, r ExecGCReview, reviewer, challenge, confirmUUID, reason, observation string, ack bool) error {
	if !safeReviewer.MatchString(reviewer) || !ack || confirmUUID != r.BoxUUID ||
		len(challenge) != 64 || len(reason) < 16 || len(reason) > 240 ||
		!validGCObservation(observation) || r.State != "REVIEW_REQUIRED" {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var st string
	var decided int64
	err = tx.QueryRowContext(ctx, "SELECT state,decided_at FROM exec_output_gc WHERE operation_id=? AND stream=?",
		r.OperationID, r.Stream).Scan(&st, &decided)
	if err != nil {
		return err
	}
	if st != "REVIEW_REQUIRED" || decided != r.DecidedAt {
		return ErrState
	}
	var at int64
	var issuedReviewer, proof string
	err = tx.QueryRowContext(ctx, "SELECT at,reviewer,evidence_digest FROM exec_output_gc_audit "+
		"WHERE operation_id=? AND stream=? AND event='CHALLENGE_ISSUED' "+
		"ORDER BY seq DESC LIMIT 1", r.OperationID, r.Stream).Scan(&at, &issuedReviewer, &proof)
	if err != nil {
		return err
	}
	age := time.Since(time.Unix(0, at))
	if issuedReviewer != reviewer || proof != gcReviewHash(challenge, reviewer, r, observation) ||
		age < 0 || age > 5*time.Minute {
		return ErrConflict
	}
	now := time.Now().UnixNano()
	result, err := tx.ExecContext(ctx, "UPDATE exec_output_gc SET state='REVIEWED_RELEASED',decided_at=? "+
		"WHERE operation_id=? AND stream=? AND state='REVIEW_REQUIRED' AND decided_at=?",
		now, r.OperationID, r.Stream, r.DecidedAt)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrState
	}
	digest := sha256.Sum256([]byte(reason))
	_, err = tx.ExecContext(ctx, "INSERT INTO exec_output_gc_audit(operation_id,stream,at,event,reviewer,evidence_digest) "+
		"VALUES(?,?,?,?,?,?)", r.OperationID, r.Stream, now, "REVIEWED_RELEASED", reviewer, hex.EncodeToString(digest[:]))
	if err != nil {
		return err
	}
	return tx.Commit()
}
