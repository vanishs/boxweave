package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ExecRetention is measured from the original durable native output witness,
// not a mutable guest timestamp or the time a reader last requested bytes.
const ExecRetention = 7 * 24 * time.Hour

// ExecGCIntent is an exact, journaled original stream DELETE intent. It is not
// a caller-chosen Incus URL or a path to be opened directly on the host.
type ExecGCIntent struct {
	OperationID       string
	Stream            string
	Project           string
	Name              string
	BoxUUID           string
	Incarnation       string
	NativeOperationID string
	Path              string
}

// ExecGCCandidates is bounded. UNKNOWN and in-flight effects never qualify.
// Any stream with a prior DELETE_SENT/REVIEW_REQUIRED/EXPIRED receipt is omitted
// forever from automatic dispatch; only a separately reviewed local recovery
// can resolve an uncertain native deletion.
func (s *Store) ExecGCCandidates(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if limit < 1 || limit > 32 {
		return nil, ErrInvalid
	}
	rows, err := s.db.QueryContext(ctx, "SELECT o.id FROM operations o "+
		"JOIN exec_outputs e ON e.operation_id=o.id "+
		"JOIN owned_boxes b ON b.uuid=o.box_uuid AND b.incarnation=o.incarnation AND b.project=o.layer "+
		"WHERE o.action='box_exec' AND o.state='SUCCEEDED' AND o.native_operation_id<>'' "+
		"AND e.exit_code IS NOT NULL AND e.observed_at>0 AND e.observed_at<=? AND b.state='OWNED' "+
		"AND NOT EXISTS (SELECT 1 FROM exec_output_gc blocked WHERE blocked.project=o.layer AND blocked.box_uuid=o.box_uuid "+
		"AND blocked.state IN ('DELETE_SENT','REVIEW_REQUIRED','REVIEWED_RELEASED')) "+
		"AND NOT EXISTS (SELECT 1 FROM operations pending WHERE pending.layer=o.layer "+
		"AND pending.box_uuid=o.box_uuid AND pending.incarnation=o.incarnation "+
		"AND pending.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN') "+
		"AND NOT EXISTS (SELECT 1 FROM recovery_audit a WHERE a.operation_id=pending.id AND a.decision='RELEASED')) "+
		"AND ((e.stdout_path<>'' AND NOT EXISTS (SELECT 1 FROM exec_output_gc j WHERE j.operation_id=o.id AND j.stream='stdout')) "+
		"OR (e.stderr_path<>'' AND NOT EXISTS (SELECT 1 FROM exec_output_gc j WHERE j.operation_id=o.id AND j.stream='stderr'))) "+
		"ORDER BY e.observed_at ASC,o.id ASC LIMIT ?", now.Add(-ExecRetention).UnixNano(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func exactExecGCPath(name, nativeID, stream, path string) bool {
	if !validRef(name) || strings.Contains(name, "/") || len(nativeID) != 36 ||
		(stream != "stdout" && stream != "stderr") {
		return false
	}
	for i, b := range nativeID {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if b != '-' {
				return false
			}
		} else if (b < '0' || b > '9') && (b < 'a' || b > 'f') {
			return false
		}
	}
	return path == fmt.Sprintf("/1.0/instances/%s/logs/exec-output/exec_%s.%s", name, nativeID, stream)
}

// PrepareExecGC atomically claims one original stream and appends a durable
// BEFORE_DELETE audit event before any external native effect may begin.
// The same DB transaction excludes other pending Box mutations, and Admit
// refuses new mutations on this instance while DELETE is uncertain.
func (s *Store) PrepareExecGC(ctx context.Context, opID, stream string, now time.Time) (ExecGCIntent, error) {
	var out ExecGCIntent
	if !validHexID(opID) || (stream != "stdout" && stream != "stderr") {
		return out, ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	var opState, ownedState, stdout, stderr string
	var observedAt int64
	var exit sql.NullInt64
	err = tx.QueryRowContext(ctx, "SELECT o.layer,o.native_operation_id,o.box_uuid,o.incarnation,b.name,b.state,"+
		"o.state,e.stdout_path,e.stderr_path,e.observed_at,e.exit_code "+
		"FROM operations o JOIN exec_outputs e ON e.operation_id=o.id "+
		"JOIN owned_boxes b ON b.uuid=o.box_uuid AND b.incarnation=o.incarnation AND b.project=o.layer "+
		"WHERE o.id=? AND o.action='box_exec'", opID).Scan(
		&out.Project, &out.NativeOperationID, &out.BoxUUID, &out.Incarnation, &out.Name, &ownedState,
		&opState, &stdout, &stderr, &observedAt, &exit)
	if errors.Is(err, sql.ErrNoRows) {
		return out, ErrNotFound
	}
	if err != nil {
		return out, err
	}
	out.OperationID, out.Stream = opID, stream
	out.Path = stdout
	if stream == "stderr" {
		out.Path = stderr
	}
	if opState != "SUCCEEDED" || ownedState != "OWNED" || !exit.Valid ||
		observedAt <= 0 || observedAt > now.Add(-ExecRetention).UnixNano() ||
		out.BoxUUID == "" || out.Incarnation == "" || out.Project == "" ||
		!exactExecGCPath(out.Name, out.NativeOperationID, stream, out.Path) {
		return ExecGCIntent{}, ErrState
	}
	var active int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations o WHERE o.layer=? "+
		"AND o.box_uuid=? AND o.incarnation=? "+
		"AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN') "+
		"AND NOT EXISTS (SELECT 1 FROM recovery_audit a WHERE a.operation_id=o.id AND a.decision='RELEASED')",
		out.Project, out.BoxUUID, out.Incarnation).Scan(&active)
	if err != nil {
		return ExecGCIntent{}, err
	}
	if active != 0 {
		return ExecGCIntent{}, ErrResourceLocked
	}
	var present int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM exec_output_gc WHERE operation_id=? AND stream=?", opID, stream).Scan(&present); err != nil {
		return ExecGCIntent{}, err
	}
	if present != 0 {
		return ExecGCIntent{}, ErrState
	}
	// A partial/ambiguous prior deletion on ANY stream of this Box holds
	// the entire native incarnation: never silently delete its other stream.
	var gcBusy int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM exec_output_gc WHERE project=? "+
		"AND box_uuid=? AND state IN ('DELETE_SENT','REVIEW_REQUIRED','REVIEWED_RELEASED')",
		out.Project, out.BoxUUID).Scan(&gcBusy)
	if err != nil {
		return ExecGCIntent{}, err
	}
	if gcBusy != 0 {
		return ExecGCIntent{}, ErrResourceLocked
	}
	at := now.UnixNano()
	_, err = tx.ExecContext(ctx, "INSERT INTO exec_output_gc(operation_id,stream,project,box_uuid,incarnation,name,native_operation_id,path,state,claimed_at) "+
		"VALUES(?,?,?,?,?,?,?,?,?,?)", opID, stream, out.Project, out.BoxUUID, out.Incarnation, out.Name, out.NativeOperationID, out.Path, "DELETE_SENT", at)
	if err != nil {
		return ExecGCIntent{}, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO exec_output_gc_audit(operation_id,stream,at,event) VALUES(?,?,?,?)",
		opID, stream, at, "BEFORE_DELETE"); err != nil {
		return ExecGCIntent{}, err
	}
	if err = tx.Commit(); err != nil {
		return ExecGCIntent{}, err
	}
	return out, nil
}

// FinishExecGC marks EXPIRED only when the scoped DELETE returned successfully
// AND a separate exact-path read verified native 404. Any other result is an
// auditable REVIEW_REQUIRED, never an invitation to replay native DELETE.
func (s *Store) FinishExecGC(ctx context.Context, opID, stream, state string) error {
	if !validHexID(opID) || (stream != "stdout" && stream != "stderr") ||
		(state != "EXPIRED" && state != "REVIEW_REQUIRED") {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "UPDATE exec_output_gc SET state=?,decided_at=? "+
		"WHERE operation_id=? AND stream=? AND state='DELETE_SENT'", state, time.Now().UnixNano(), opID, stream)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrState
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO exec_output_gc_audit(operation_id,stream,at,event) VALUES(?,?,?,?)",
		opID, stream, time.Now().UnixNano(), state); err != nil {
		return err
	}
	return tx.Commit()
}

// ExecGCStreamState returns only a controller-authenticated original's per-
// stream retention outcome. A missing file without EXPIRED is not expiry.
func (s *Store) ExecGCStreamState(ctx context.Context, opID, stream string) (string, error) {
	if !validHexID(opID) || (stream != "stdout" && stream != "stderr") {
		return "", ErrInvalid
	}
	var st string
	err := s.db.QueryRowContext(ctx, "SELECT state FROM exec_output_gc WHERE operation_id=? AND stream=?", opID, stream).Scan(&st)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return st, err
}

// RecoverExecGC never resends a DELETE; it converts an interrupted durable
// intent to REVIEW_REQUIRED with an explicit read-only recovery audit.
func (s *Store) RecoverExecGC(ctx context.Context) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT operation_id,stream FROM exec_output_gc WHERE state='DELETE_SENT'")
	if err != nil {
		return 0, err
	}
	type item struct{ op, stream string }
	var pending []item
	for rows.Next() {
		var v item
		if err = rows.Scan(&v.op, &v.stream); err != nil {
			break
		}
		pending = append(pending, v)
	}
	if closeErr := rows.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, err
	}
	now := time.Now().UnixNano()
	for _, v := range pending {
		if _, err = tx.ExecContext(ctx, "UPDATE exec_output_gc SET state='REVIEW_REQUIRED',decided_at=? "+
			"WHERE operation_id=? AND stream=? AND state='DELETE_SENT'", now, v.op, v.stream); err != nil {
			return 0, err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO exec_output_gc_audit(operation_id,stream,at,event) VALUES(?,?,?,?)",
			v.op, v.stream, now, "RECOVERED_UNCERTAIN"); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return len(pending), nil
}
