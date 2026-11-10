package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// OwnedBox is an internal UUID/incarnation claim, not a user-supplied name
// authorization or evidence that the native instance still exists.
type OwnedBox struct {
	UUID        string `json:"box_id"`
	Project     string `json:"project"`
	Name        string `json:"name"`
	Incarnation string `json:"incarnation"`
	State       string `json:"state"`
}

func (s *Store) OwnedBox(ctx context.Context, project, uuid string) (OwnedBox, error) {
	var box OwnedBox
	if project == "" || uuid == "" {
		return box, ErrInvalid
	}
	err := s.db.QueryRowContext(ctx, `SELECT uuid,project,name,incarnation,state
		FROM owned_boxes WHERE project=? AND uuid=? AND state='OWNED'`,
		project, uuid).Scan(&box.UUID, &box.Project, &box.Name, &box.Incarnation, &box.State)
	if errors.Is(err, sql.ErrNoRows) {
		return OwnedBox{}, ErrNotFound
	}
	return box, err
}

func (s *Store) OwnedByName(ctx context.Context, project, name string) (OwnedBox, error) {
	var box OwnedBox
	if project == "" || name == "" {
		return box, ErrInvalid
	}
	err := s.db.QueryRowContext(ctx, `SELECT uuid,project,name,incarnation,state
		FROM owned_boxes WHERE project=? AND name=? AND state='OWNED'`,
		project, name).Scan(&box.UUID, &box.Project, &box.Name, &box.Incarnation, &box.State)
	if errors.Is(err, sql.ErrNoRows) {
		return OwnedBox{}, ErrNotFound
	}
	return box, err
}

// ListOwnedBoxes returns a finite, stable keyset page of this Project's own
// registered UUIDs; each must still be reverified against native Incus.
func (s *Store) ListOwnedBoxes(ctx context.Context, project, after string, limit int) ([]OwnedBox, string, error) {
	if project == "" || len(project) > 128 || len(after) > 128 || limit < 0 || limit > 50 {
		return nil, "", ErrInvalid
	}
	if limit == 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT uuid,project,name,incarnation,state FROM owned_boxes
		WHERE project=? AND state='OWNED' AND uuid>? ORDER BY uuid LIMIT ?`, project, after, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	out := make([]OwnedBox, 0, limit)
	cursor := ""
	for rows.Next() {
		var b OwnedBox
		if err := rows.Scan(&b.UUID, &b.Project, &b.Name, &b.Incarnation, &b.State); err != nil {
			return nil, "", err
		}
		if len(out) == limit {
			cursor = out[len(out)-1].UUID
			break
		}
		out = append(out, b)
	}
	return out, cursor, rows.Err()
}

// TombstoneOwnedBox is called only after independent exact native postread,
// never merely after DeleteInstance acknowledgement or transient 404.
func (s *Store) TombstoneOwnedBox(ctx context.Context, box OwnedBox) error {
	if box.UUID == "" || box.Project == "" || box.Incarnation == "" {
		return ErrInvalid
	}
	result, err := s.db.ExecContext(ctx, `UPDATE owned_boxes SET state='DELETED'
		WHERE uuid=? AND project=? AND incarnation=? AND name=? AND state='OWNED'`,
		box.UUID, box.Project, box.Incarnation, box.Name)
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
	return nil
}

func (s *Store) VerifiedFailed(ctx context.Context, id, errClass string) error {
	if errClass == "" || len(errClass) > 120 || strings.ContainsAny(errClass, "\n\r\x00") {
		return ErrInvalid
	}
	// This terminal failure must be backed by an actual native final state
	// and verified absence of ambiguous residual effects.
	return s.transition(ctx, id, "VERIFYING", "FAILED", "", errClass)
}

// CancelQueued atomically proves and persists that an original ADMITTED
// Operation cannot subsequently be claimed by any worker. It is never
// evidence that a DISPATCHING/RUNNING/UNKNOWN guest process was terminated.
func (s *Store) CancelQueued(ctx context.Context, layer, id string) error {
	if layer == "" || id == "" {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UnixNano()
	res, err := tx.ExecContext(ctx, `UPDATE operations SET state='CANCELLED',
		error_class='CANCELLED_BEFORE_DISPATCH',updated_at=?
		WHERE layer=? AND id=? AND state='ADMITTED'`, now, layer, id)
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
	if _, err := tx.ExecContext(ctx, `INSERT INTO operation_events
		(operation_id,at,state,safe_detail) VALUES (?,?,?,?)`,
		id, now, "CANCELLED", "CANCELLED_BEFORE_DISPATCH"); err != nil {
		return err
	}
	return tx.Commit()
}

// NativeReadSnapshot is a structured immutable reference to an original
// recorded Exec output. It is not the output bytes and it never authorizes
// a caller to supply a log URL or arbitrary host/native path.
type NativeReadSnapshot struct {
	OperationID string `json:"operation_id"`
	StdoutPath  string `json:"-"`
	StderrPath  string `json:"-"`
	ExitCode    *int   `json:"exit_code,omitempty"`
	ObservedAt  int64  `json:"observed_at"`
	Truncated   bool   `json:"truncated"`
}

func (s *Store) SaveExecOutput(ctx context.Context, opID, stdoutPath, stderrPath string, exit *int) error {
	if opID == "" || (stdoutPath == "" && stderrPath == "") ||
		len(stdoutPath) > 512 || len(stderrPath) > 512 ||
		!strings.HasPrefix(stdoutPath, "/1.0/instances/") && stdoutPath != "" ||
		!strings.HasPrefix(stderrPath, "/1.0/instances/") && stderrPath != "" {
		return ErrInvalid
	}
	var exitValue any
	if exit != nil {
		exitValue = *exit
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO exec_outputs
		(operation_id,stdout_path,stderr_path,exit_code,observed_at)
		VALUES (?,?,?,?,?)
		ON CONFLICT(operation_id) DO UPDATE SET stdout_path=excluded.stdout_path,
			stderr_path=excluded.stderr_path,exit_code=excluded.exit_code,observed_at=excluded.observed_at`,
		opID, stdoutPath, stderrPath, exitValue, time.Now().UnixNano())
	return err
}

func (s *Store) ExecOutput(ctx context.Context, originalID string) (NativeReadSnapshot, error) {
	var out NativeReadSnapshot
	var exit sql.NullInt64
	var truncated int
	err := s.db.QueryRowContext(ctx, `SELECT operation_id,stdout_path,stderr_path,
		exit_code,observed_at,truncation FROM exec_outputs WHERE operation_id=?`,
		originalID).Scan(&out.OperationID, &out.StdoutPath, &out.StderrPath,
		&exit, &out.ObservedAt, &truncated)
	if errors.Is(err, sql.ErrNoRows) {
		return out, ErrNotFound
	}
	if err != nil {
		return out, err
	}
	if exit.Valid {
		if exit.Int64 < -2147483648 || exit.Int64 > 2147483647 {
			return out, fmt.Errorf("unsafe recorded exit code")
		}
		v := int(exit.Int64)
		out.ExitCode = &v
	}
	out.Truncated = truncated != 0
	return out, nil
}
