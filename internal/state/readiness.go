package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
)

const ReadinessAction = "box_wait_ready"

// An OBSERVING job has dispatched no native effect. Only its short final
// verification/probe takes the ordinary mutation fence (DISPATCHING/VERIFYING).
// Keep the original schema and store the bounded, typed samples in the journal.
type ReadinessIntent struct {
	BoxID          string   `json:"box_id"`
	Checks         []string `json:"checks"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

func NormalizeReadinessIntent(i ReadinessIntent) (ReadinessIntent, error) {
	if i.BoxID == "" || i.TimeoutSeconds < 0 || i.TimeoutSeconds > 600 {
		return i, ErrInvalid
	}
	if i.TimeoutSeconds == 0 {
		i.TimeoutSeconds = 300
	}
	if len(i.Checks) == 0 {
		i.Checks = []string{"exec", "running"}
	}
	if len(i.Checks) > 4 {
		return i, ErrInvalid
	}
	i.Checks = slices.Clone(i.Checks)
	slices.Sort(i.Checks)
	for n, c := range i.Checks {
		if c != "running" && c != "exec" && c != "ipv4" && c != "ipv6" || n > 0 && i.Checks[n-1] == c {
			return i, ErrInvalid
		}
	}
	return i, nil
}

type ReadinessSample struct {
	Running      string `json:"running"`
	ExecReady    string `json:"exec_ready"`
	IPv4Assigned string `json:"ipv4_assigned"`
	IPv6Assigned string `json:"ipv6_assigned"`
	CheckedAt    int64  `json:"checked_at_ns"`
	GuestExit    *int   `json:"guest_exit_code,omitempty"`
}

type ReadinessResult struct {
	ReadinessSample
	RequestedChecks []string `json:"checks"`
	StartedAt       int64    `json:"started_at_ns"`
	Deadline        int64    `json:"deadline_ns"`
	Ready           bool     `json:"ready"`
	LastError       string   `json:"last_error_class,omitempty"`
}

func NewReadinessSample(i ReadinessIntent) ReadinessSample {
	field := func(c string) string {
		if slices.Contains(i.Checks, c) {
			return "unknown"
		}
		return "not_checked"
	}
	return ReadinessSample{Running: field("running"), ExecReady: field("exec"), IPv4Assigned: field("ipv4"), IPv6Assigned: field("ipv6")}
}
func (s ReadinessSample) AllReady() bool {
	for _, v := range []string{s.Running, s.ExecReady, s.IPv4Assigned, s.IPv6Assigned} {
		if v != "ready" && v != "not_checked" {
			return false
		}
	}
	return true
}

const readinessPrefix = "READINESS:"

func (s *Store) admitReadinessTx(ctx context.Context, tx *sql.Tx, a Admission, keyHash, intentHash string) (Receipt, error) {
	var i ReadinessIntent
	if json.Unmarshal(a.SafeIntent, &i) != nil || i.BoxID != a.BoxUUID || a.BoxUUID == "" || a.Incarnation == "" || a.ResourceKey == "" || len(a.ResourceKeys) != 0 || a.CleanupDependencies || a.ForwardCreate != nil || a.ForwardDeleteID != "" {
		return Receipt{}, ErrInvalid
	}
	normalized, e := NormalizeReadinessIntent(i)
	if e != nil || !slices.Equal(i.Checks, normalized.Checks) || i.TimeoutSeconds != normalized.TimeoutSeconds {
		return Receipt{}, ErrInvalid
	}
	var n int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM owned_boxes WHERE project=? AND uuid=? AND incarnation=? AND state='OWNED' AND ('box:'||name)=?", a.Layer, a.BoxUUID, a.Incarnation, a.ResourceKey).Scan(&n); e != nil {
		return Receipt{}, e
	}
	if n != 1 {
		return Receipt{}, ErrNotFound
	}
	// A bounded number of observation jobs per layer; never dispatch a second
	// probe for an unresolved original or run concurrent waits for one Box.
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM operations o
 WHERE layer=? AND action=? AND state IN ('OBSERVING','DISPATCHING','VERIFYING','UNKNOWN')
 AND NOT EXISTS(SELECT 1 FROM recovery_audit r WHERE r.operation_id=o.id AND r.decision='RELEASED')`, a.Layer, ReadinessAction).Scan(&n); e != nil {
		return Receipt{}, e
	}
	if n >= 8 {
		return Receipt{}, ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM operations o
 WHERE layer=? AND box_uuid=? AND incarnation=? AND action=? AND state IN ('OBSERVING','DISPATCHING','VERIFYING','UNKNOWN')
 AND NOT EXISTS(SELECT 1 FROM recovery_audit r WHERE r.operation_id=o.id AND r.decision='RELEASED')`, a.Layer, a.BoxUUID, a.Incarnation, ReadinessAction).Scan(&n); e != nil {
		return Receipt{}, e
	}
	if n != 0 {
		return Receipt{}, ErrResourceLocked
	}
	id, e := newID()
	if e != nil {
		return Receipt{}, e
	}
	tracking, e := newID()
	if e != nil {
		return Receipt{}, e
	}
	now := time.Now().UnixNano()
	_, e = tx.ExecContext(ctx, `INSERT INTO operations(id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,intent_digest,safe_intent,work_ref,agent_ref,resource_key,box_uuid,incarnation,state,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, tracking, a.RequestID, a.Layer, a.ActorID, a.TokenID, keyHash, a.Action, intentHash, a.SafeIntent, a.WorkRef, a.AgentRef, a.ResourceKey, a.BoxUUID, a.Incarnation, "OBSERVING", now, now)
	if e != nil {
		return Receipt{}, e
	}
	raw, e := json.Marshal(NewReadinessSample(i))
	if e != nil {
		return Receipt{}, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,?,?)", id, now, "OBSERVING", readinessPrefix+string(raw)); e != nil {
		return Receipt{}, e
	}
	if e = tx.Commit(); e != nil {
		return Receipt{}, e
	}
	return Receipt{ID: id, TrackingID: tracking, State: "OBSERVING", Action: a.Action, BoxUUID: a.BoxUUID, Incarnation: a.Incarnation, WorkRef: a.WorkRef, AgentRef: a.AgentRef, CreatedAt: now, UpdatedAt: now, NextAction: nextAction("OBSERVING", a.Action, id)}, nil
}

// ClaimReadiness acquires a short native fence only after observation checks
// pass. Conservatively defer while any ordinary Project operation is active.
// All other admission paths already respect DISPATCHING/VERIFYING/UNKNOWN.
func (s *Store) ClaimReadiness(ctx context.Context, id string) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var layer, uuid, inc, resource, actor, token, status string
	e = tx.QueryRowContext(ctx, "SELECT layer,box_uuid,incarnation,resource_key,actor_id,token_id,state FROM operations WHERE id=? AND action=?", id, ReadinessAction).Scan(&layer, &uuid, &inc, &resource, &actor, &token, &status)
	if e != nil {
		return e
	}
	if status != "OBSERVING" {
		return ErrState
	}
	active, e := activeCredentialForActor(ctx, tx, actor, token)
	if e != nil {
		return e
	}
	if !active {
		return ErrRevoked
	}
	var n int
	e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM owned_boxes WHERE project=? AND uuid=? AND incarnation=? AND ('box:'||name)=? AND state='OWNED'", layer, uuid, inc, resource).Scan(&n)
	if e != nil {
		return e
	}
	if n != 1 {
		return ErrNotFound
	}
	e = tx.QueryRowContext(ctx, `SELECT count(*) FROM operations o WHERE o.layer=? AND o.id<>? AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN') AND (o.action='box_peer_set' OR NOT EXISTS(SELECT 1 FROM recovery_audit r WHERE r.operation_id=o.id AND r.decision='RELEASED'))`, layer, id).Scan(&n)
	if e != nil {
		return e
	}
	if n != 0 {
		return ErrResourceLocked
	}
	e = tx.QueryRowContext(ctx, "SELECT count(*) FROM exec_output_gc WHERE project=? AND box_uuid=? AND state IN ('DELETE_SENT','REVIEW_REQUIRED')", layer, uuid).Scan(&n)
	if e != nil {
		return e
	}
	if n != 0 {
		return ErrResourceLocked
	}
	now := time.Now().UnixNano()
	if _, e = tx.ExecContext(ctx, "UPDATE operations SET state='DISPATCHING',updated_at=? WHERE id=? AND state='OBSERVING'", now, id); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,?,?)", id, now, "DISPATCHING", "READINESS_POINT_OF_USE_FENCE"); e != nil {
		return e
	}
	return tx.Commit()
}

// ReadinessAddressAllowed refuses quarantined/foreign reserved addresses.
// The caller already independently observed the actual native private NIC.
func (s *Store) ReadinessAddressAllowed(ctx context.Context, project, uuid, inc, bridge, family, address string) (bool, error) {
	var n int
	e := s.db.QueryRowContext(ctx, `SELECT count(*) FROM network_address_fences WHERE bridge=? AND family=? AND address=? AND state!='RELEASED' AND (project!=? OR box_uuid!=? OR incarnation!=? OR state='QUARANTINED')`, bridge, family, address, project, uuid, inc).Scan(&n)
	if e != nil {
		return false, e
	}
	if n != 0 {
		return false, nil
	}
	e = s.db.QueryRowContext(ctx, `SELECT count(*) FROM network_bindings WHERE project=? AND box_uuid=? AND incarnation=? AND (state='QUARANTINED' OR (state='PINNED' AND (bridge!=? OR CASE WHEN ?='ipv4' THEN ipv4 ELSE ipv6 END !=?)))`, project, uuid, inc, bridge, family, address).Scan(&n)
	return n == 0, e
}

// SaveReadiness atomically persists the bounded sample and state; an old
// worker cannot overwrite a cancellation, restart verdict or newer original.
func (s *Store) SaveReadiness(ctx context.Context, id, from, to, reason string, sample ReadinessSample) error {
	switch from {
	case "OBSERVING", "DISPATCHING", "VERIFYING":
	default:
		return ErrInvalid
	}
	switch to {
	case "OBSERVING", "FAILED", "TIMED_OUT", "UNKNOWN", "SUCCEEDED":
	default:
		return ErrInvalid
	}
	if to == "SUCCEEDED" && (from == "OBSERVING" || !sample.AllReady() || sample.CheckedAt == 0) {
		return ErrInvalid
	}
	if len(reason) > 120 {
		return ErrInvalid
	}
	raw, e := json.Marshal(sample)
	if e != nil {
		return e
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	now := time.Now().UnixNano()
	if to == "SUCCEEDED" {
		var actor, token, layer, uuid, inc, resource string
		var intentBytes []byte
		var started int64
		if e = tx.QueryRowContext(ctx, "SELECT actor_id,token_id,layer,box_uuid,incarnation,resource_key,safe_intent,created_at FROM operations WHERE id=? AND action=? AND state=?", id, ReadinessAction, from).Scan(&actor, &token, &layer, &uuid, &inc, &resource, &intentBytes, &started); e != nil {
			return e
		}
		var intent ReadinessIntent
		if json.Unmarshal(intentBytes, &intent) != nil {
			return ErrState
		}
		active, err := activeCredentialForActor(ctx, tx, actor, token)
		if err != nil {
			return err
		}
		var owned int
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM owned_boxes WHERE project=? AND uuid=? AND incarnation=? AND ('box:'||name)=? AND state='OWNED'", layer, uuid, inc, resource).Scan(&owned); e != nil {
			return e
		}
		if !active {
			to, reason = "FAILED", "READINESS_AUTH_REVOKED"
		}
		if owned != 1 {
			to, reason = "FAILED", "EXACT_BOX_IDENTITY_LOST"
		}
		if now >= started+int64(time.Duration(intent.TimeoutSeconds)*time.Second) {
			to, reason = "TIMED_OUT", "READINESS_DEADLINE_ELAPSED"
		}
		expected := NewReadinessSample(intent)
		for _, pair := range [][2]string{{expected.Running, sample.Running}, {expected.ExecReady, sample.ExecReady}, {expected.IPv4Assigned, sample.IPv4Assigned}, {expected.IPv6Assigned, sample.IPv6Assigned}} {
			if pair[0] == "not_checked" && pair[1] != "not_checked" || pair[0] != "not_checked" && pair[1] != "ready" {
				return ErrInvalid
			}
		}
		if expected.ExecReady != "not_checked" && (from != "VERIFYING" || sample.GuestExit == nil || *sample.GuestExit != 0) {
			return ErrInvalid
		}
	}
	r, e := tx.ExecContext(ctx, "UPDATE operations SET state=?,error_class=?,updated_at=? WHERE id=? AND action=? AND state=?", to, reason, now, id, ReadinessAction, from)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if n != 1 {
		return ErrState
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,?,?)", id, now, to, readinessPrefix+string(raw)); e != nil {
		return e
	}
	return tx.Commit()
}

func (s *Store) Readiness(ctx context.Context, receipt Receipt) (*ReadinessResult, error) {
	if receipt.Action != ReadinessAction {
		return nil, nil
	}
	var raw []byte
	var detail string
	e := s.db.QueryRowContext(ctx, `SELECT safe_intent, COALESCE((SELECT safe_detail FROM operation_events e WHERE e.operation_id=o.id AND e.safe_detail LIKE 'READINESS:%' AND e.at<=? ORDER BY seq DESC LIMIT 1),'') FROM operations o WHERE id=? AND action=?`, receipt.UpdatedAt, receipt.ID, ReadinessAction).Scan(&raw, &detail)
	if e != nil {
		return nil, e
	}
	var i ReadinessIntent
	if json.Unmarshal(raw, &i) != nil {
		return nil, ErrState
	}
	sample := NewReadinessSample(i)
	if detail != "" {
		if !strings.HasPrefix(detail, readinessPrefix) || json.Unmarshal([]byte(strings.TrimPrefix(detail, readinessPrefix)), &sample) != nil {
			return nil, ErrState
		}
	}
	// A crash after the last observation does not retroactively prove readiness.
	if receipt.State == "UNKNOWN" && sample.ExecReady != "not_checked" {
		sample.ExecReady = "unknown"
	}
	return &ReadinessResult{ReadinessSample: sample, RequestedChecks: i.Checks, StartedAt: receipt.CreatedAt, Deadline: receipt.CreatedAt + int64(time.Duration(i.TimeoutSeconds)*time.Second), Ready: receipt.State == "SUCCEEDED" && sample.AllReady(), LastError: receipt.ErrorClass}, nil
}

func (s *Store) ReadinessActorActive(ctx context.Context, id string) error {
	var actor, token string
	e := s.db.QueryRowContext(ctx, "SELECT actor_id,token_id FROM operations WHERE id=? AND action=? AND state IN ('OBSERVING','DISPATCHING','VERIFYING')", id, ReadinessAction).Scan(&actor, &token)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrState
	}
	if e != nil {
		return e
	}
	ok, e := activeCredentialForActor(ctx, s.db, actor, token)
	if e != nil {
		return e
	}
	if !ok {
		return ErrRevoked
	}
	return nil
}
