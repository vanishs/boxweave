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
	// and verified accounting of every relevant residual effect. A retained
	// exact owned Box remains protected by its OWNED registration.
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
		WHERE layer=? AND id=? AND (state='ADMITTED' OR (action='box_wait_ready' AND state='OBSERVING' AND native_operation_id=''))`, now, layer, id)
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
	if err := settleForwardNoEffectTx(ctx, tx, id); err != nil {
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

// FinalizeVerifiedExec durably commits the original Exec terminal receipt,
// recorded output paths and the SUCCEEDED operation event as ONE transaction.
// The worker must use this only after verifying the exact original native
// terminal outcome, never based on command text, output-file existence alone,
// an expired operation ID, or a guessed absence of guest side effects.
func (s *Store) FinalizeVerifiedExec(ctx context.Context, operationID string, box OwnedBox,
	nativeID, stdoutPath, stderrPath string, exit int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = finishVerifiedExecTx(ctx, tx, operationID, box, nativeID, stdoutPath, stderrPath, exit, "VERIFYING", "VERIFIED_NATIVE_EXEC_TERMINAL"); err != nil {
		return err
	}
	return tx.Commit()
}

func finishVerifiedExecTx(ctx context.Context, tx *sql.Tx, operationID string, box OwnedBox,
	nativeID, stdoutPath, stderrPath string, exit int, expectedPhase, detail string) error {
	if operationID == "" || box.UUID == "" || box.Project == "" ||
		box.Name == "" || box.Incarnation == "" || box.State != "OWNED" ||
		len(box.Name) > 63 || strings.ContainsAny(box.Name, "/\\?%#\x00\r\n") ||
		len(nativeID) != 36 || (stdoutPath == "" && stderrPath == "") ||
		len(stdoutPath) > 512 || len(stderrPath) > 512 ||
		int64(exit) < -2147483648 || int64(exit) > 2147483647 {
		return ErrInvalid
	}
	for i, c := range nativeID {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return ErrInvalid
			}
		} else if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return ErrInvalid
		}
	}
	prefix := "/1.0/instances/" + box.Name + "/logs/exec-output/exec_" + nativeID
	if stdoutPath != "" && stdoutPath != prefix+".stdout" ||
		stderrPath != "" && stderrPath != prefix+".stderr" {
		return ErrInvalid
	}

	// A terminal ACK for a foreign operation or prior incarnation cannot
	// release this resource claim, even if the output filename looks valid.
	now := time.Now().UnixNano()
	res, err := tx.ExecContext(ctx, `UPDATE operations
		SET state='SUCCEEDED',error_class='',updated_at=?
		WHERE id=? AND action='box_exec' AND state=?
		AND layer=? AND box_uuid=? AND incarnation=? AND resource_key=?
		AND native_operation_id=?
		AND EXISTS (SELECT 1 FROM owned_boxes
			WHERE uuid=? AND project=? AND name=? AND incarnation=? AND state='OWNED')`,
		now, operationID, expectedPhase, box.Project, box.UUID, box.Incarnation,
		"box:"+box.Name, nativeID,
		box.UUID, box.Project, box.Name, box.Incarnation)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrState
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO exec_outputs
		(operation_id,stdout_path,stderr_path,exit_code,observed_at)
		VALUES (?,?,?,?,?)`,
		operationID, stdoutPath, stderrPath, exit, now); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO operation_events
		(operation_id,at,state,safe_detail) VALUES (?,?,?,?)`,
		operationID, now, "SUCCEEDED", detail); err != nil {
		return err
	}
	return nil
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

// RegisterCreatedBox commits the native UUID/incarnation together with the
// original operation. A native create is never inferred from name alone.
func (s *Store) RegisterCreatedBox(ctx context.Context, operationID, project, name, uuid string) error {
	return s.registerCreatedBox(ctx, operationID, project, name, uuid, nil)
}
func (s *Store) RegisterCreatedBoxWithBinding(ctx context.Context, operationID, project, name, uuid string, binding VerifiedCreateBinding) error {
	if !binding.valid() {
		return ErrInvalid
	}
	return s.registerCreatedBox(ctx, operationID, project, name, uuid, &binding)
}
func (s *Store) registerCreatedBox(ctx context.Context, operationID, project, name, uuid string, binding *VerifiedCreateBinding) error {
	if operationID == "" || project == "" || name == "" || uuid == "" ||
		len(name) > 63 || len(uuid) > 128 {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var action, state, resource, existingUUID, layer, nativeID string
	err = tx.QueryRowContext(ctx, `SELECT action,state,resource_key,box_uuid,layer,
		native_operation_id FROM operations WHERE id=?`, operationID).
		Scan(&action, &state, &resource, &existingUUID, &layer, &nativeID)
	if err != nil {
		return err
	}
	if action != "box_create" || state != "VERIFYING" || layer != project ||
		resource != "box:"+name || existingUUID != "" || nativeID == "" {
		return ErrState
	}
	schema, err := scopedCreateIntentSchema(ctx, tx, operationID)
	if err != nil {
		return err
	}
	if schema != 0 {
		return ErrResourceLocked
	}
	now := time.Now().UnixNano()
	if _, err := tx.ExecContext(ctx, `INSERT INTO owned_boxes
		(uuid,project,name,incarnation,state,created_at)
		VALUES (?,?,?,?,'OWNED',?)`, uuid, project, name, uuid, now); err != nil {
		return err
	}
	if binding != nil {
		if err := insertVerifiedCreateBinding(ctx, tx, operationID, project, uuid, *binding, now); err != nil {
			return err
		}
	} else {
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", operationID).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			return ErrResourceLocked
		}
	}
	res, err := tx.ExecContext(ctx, `UPDATE operations
		SET box_uuid=?,incarnation=?,updated_at=? WHERE id=? AND box_uuid=''`,
		uuid, uuid, now, operationID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return ErrState
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO operation_events
		(operation_id,at,state,safe_detail) VALUES (?,?,?,?)`,
		operationID, now, "VERIFYING", "NATIVE_UUID_CONFIRMED"); err != nil {
		return err
	}
	return tx.Commit()
}

// FinalizeVerifiedCreate commits the first successful Create's exact native UUID,
// durable ownership claim, and original SUCCEEDED event in ONE SQLite transaction.
// The native verifier has already independently proved the native terminal state
// and current instance identity. A failed-start residue uses RegisterCreatedBox
// instead, deliberately retaining a FAILED owned claim for explicit deletion.
func (s *Store) FinalizeVerifiedCreate(ctx context.Context, opID, project, name, uuid string) error {
	return s.finalizeVerifiedCreate(ctx, opID, project, name, uuid, nil, false)
}

// FinalizeVerifiedCreateWithBinding couples the original Create receipt,
// exact UUID and independently native-verified static NIC binding. This is
// not a DHCP allocator or an authority to declare a dynamic lease pinned.
func (s *Store) FinalizeVerifiedCreateWithBinding(ctx context.Context, opID, project, name, uuid string, binding VerifiedCreateBinding) error {
	if !binding.valid() {
		return ErrInvalid
	}
	return s.finalizeVerifiedCreate(ctx, opID, project, name, uuid, &binding, false)
}

// FinalizeVerifiedScopedCreate requires verified native UUID/ACL/boot
// postconditions from the trusted controller. A legacy finalizer cannot
// accept a v2 intent or retain the provisional ACL as the final binding.
func (s *Store) FinalizeVerifiedScopedCreate(ctx context.Context, opID, project, name, uuid string, binding VerifiedCreateBinding) error {
	if !binding.valid() {
		return ErrInvalid
	}
	return s.finalizeVerifiedCreate(ctx, opID, project, name, uuid, &binding, true)
}
func (s *Store) finalizeVerifiedCreate(ctx context.Context, opID, project, name, uuid string, binding *VerifiedCreateBinding, scoped bool) error {
	if opID == "" || project == "" || name == "" || uuid == "" ||
		len(name) > 63 || len(uuid) > 128 {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var action, phase, resource, oldUUID, layer, nativeID string
	if err = tx.QueryRowContext(ctx, `SELECT action,state,resource_key,box_uuid,layer,
		native_operation_id FROM operations WHERE id=?`, opID).
		Scan(&action, &phase, &resource, &oldUUID, &layer, &nativeID); err != nil {
		return err
	}
	if action != "box_create" || phase != "VERIFYING" ||
		layer != project || resource != "box:"+name || oldUUID != "" || nativeID == "" {
		return ErrState
	}
	schema, err := scopedCreateIntentSchema(ctx, tx, opID)
	if err != nil {
		return err
	}
	if scoped {
		if schema != 2 || binding == nil {
			return ErrResourceLocked
		}
		if err = exchangeScopedCreateACL(ctx, tx, opID, project, name, uuid, *binding); err != nil {
			return err
		}
	} else if schema != 0 {
		return ErrResourceLocked
	}
	now := time.Now().UnixNano()
	if _, err = tx.ExecContext(ctx, `INSERT INTO owned_boxes
		(uuid,project,name,incarnation,state,created_at)
		VALUES (?,?,?,?,'OWNED',?)`, uuid, project, name, uuid, now); err != nil {
		return err
	}
	if binding != nil {
		if err := insertVerifiedCreateBinding(ctx, tx, opID, project, uuid, *binding, now); err != nil {
			return err
		}
	} else {
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", opID).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			return ErrResourceLocked
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE operations SET box_uuid=?,incarnation=?,
		state='SUCCEEDED',error_class='',updated_at=?
		WHERE id=? AND state='VERIFYING' AND box_uuid=''`,
		uuid, uuid, now, opID)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		return ErrState
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO operation_events
		(operation_id,at,state,safe_detail) VALUES (?,?,?,?)`,
		opID, now, "SUCCEEDED", "ORIGINAL_CREATE_UUID_ATOMIC"); err != nil {
		return err
	}
	return tx.Commit()
}

// FinalizeVerifiedDelete requires independent terminal native Delete and exact
// absence proof BEFORE invocation. Its tombstone and original receipt must
// never commit independently; any ambiguous ACK/crash stays UNKNOWN and locked.
func (s *Store) FinalizeVerifiedDelete(ctx context.Context, opID string, box OwnedBox) error {
	if opID == "" || box.UUID == "" || box.Project == "" || box.Name == "" ||
		box.Incarnation == "" || box.State != "OWNED" {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := finishVerifiedDeleteTx(ctx, tx, opID, box, "VERIFYING", "ORIGINAL_DELETE_TOMBSTONE_ATOMIC"); err != nil {
		return err
	}
	return tx.Commit()
}

// FinalizeVerifiedScopedDelete requires a trusted Root receipt proving
// the exact detached ACL no longer exists. Legacy recovery cannot settle
// this mode's original using instance absence alone.
func (s *Store) FinalizeVerifiedScopedDelete(ctx context.Context, opID string, box OwnedBox) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = finishVerifiedDeleteTxMode(ctx, tx, opID, box, "VERIFYING", "SCOPED_DELETE_ACL_ABSENT_ATOMIC", true); err != nil {
		return err
	}
	return tx.Commit()
}
func finishVerifiedDeleteTx(ctx context.Context, tx *sql.Tx, opID string, box OwnedBox, expectedPhase, detail string) error {
	return finishVerifiedDeleteTxMode(ctx, tx, opID, box, expectedPhase, detail, false)
}
func finishVerifiedDeleteTxMode(ctx context.Context, tx *sql.Tx, opID string, box OwnedBox, expectedPhase, detail string, scoped bool) error {
	var err error
	var action, phase, layer, uuid, incarnation, resource, nativeID string
	if err = tx.QueryRowContext(ctx, `SELECT action,state,layer,box_uuid,
		incarnation,resource_key,native_operation_id FROM operations WHERE id=?`,
		opID).Scan(&action, &phase, &layer, &uuid, &incarnation, &resource, &nativeID); err != nil {
		return err
	}
	if action != "box_delete" || phase != expectedPhase || nativeID == "" ||
		layer != box.Project || uuid != box.UUID ||
		incarnation != box.Incarnation || resource != "box:"+box.Name {
		return ErrState
	}
	var createID string
	e := tx.QueryRowContext(ctx, "SELECT id FROM operations WHERE action='box_create' AND layer=? AND box_uuid=? AND incarnation=?", box.Project, box.UUID, box.Incarnation).Scan(&createID)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return e
	}
	schema := 0
	if createID != "" {
		schema, e = scopedCreateIntentSchema(ctx, tx, createID)
		if e != nil {
			return e
		}
	}
	if scoped {
		if schema != 2 {
			return ErrResourceLocked
		}
		var bridge string
		if e = tx.QueryRowContext(ctx, "SELECT bridge FROM network_bindings WHERE project=? AND box_uuid=? AND incarnation=? AND state='PINNED'", box.Project, box.UUID, box.Incarnation).Scan(&bridge); e != nil {
			return e
		}
		p, e := readScopedDeleteOriginalTxPhase(ctx, tx, opID, box.Project, bridge, expectedPhase)
		if e != nil || p.Box != box {
			return ErrResourceLocked
		}
	} else if schema != 0 {
		return ErrResourceLocked
	}
	if err := finalizeDeleteDependencies(ctx, tx, opID, box); err != nil {
		return err
	}
	now := time.Now().UnixNano()
	result, err := tx.ExecContext(ctx, `UPDATE owned_boxes SET state='DELETED',
		deleted_by_operation_id=?,deleted_at=?
		WHERE uuid=? AND project=? AND name=? AND incarnation=? AND state='OWNED'`,
		opID, now, box.UUID, box.Project, box.Name, box.Incarnation)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		return ErrState
	}
	result, err = tx.ExecContext(ctx, `UPDATE operations SET state='SUCCEEDED',
		error_class='',updated_at=? WHERE id=? AND state=?`, now, opID, expectedPhase)
	if err != nil {
		return err
	}
	if n, err := result.RowsAffected(); err != nil || n != 1 {
		return ErrState
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO operation_events
		(operation_id,at,state,safe_detail) VALUES (?,?,?,?)`,
		opID, now, "SUCCEEDED", detail); err != nil {
		return err
	}
	return nil
}

// CompleteWithoutNative requires separate postcondition proof before an
// already-started or already-stopped Box is marked a successful no-op.
func (s *Store) CompleteWithoutNative(ctx context.Context, operationID string) error {
	return s.transition(ctx, operationID, "DISPATCHING", "SUCCEEDED", "",
		"VERIFIED_ALREADY_IN_DESIRED_STATE")
}

// FailPreDispatch is limited to provably local validation that occurred
// BEFORE any Incus mutation request. Never use it on HTTP errors/timeouts.
func (s *Store) FailPreDispatch(ctx context.Context, operationID, reason string) error {
	if reason == "" || len(reason) > 120 || strings.ContainsAny(reason, "\x00\n\r") {
		return ErrInvalid
	}
	return s.transition(ctx, operationID, "DISPATCHING", "FAILED", "", reason)
}
