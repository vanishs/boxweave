package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// RecoveryTarget is private evidence, never serialized to MCP. The original
// receipt and claim are separate. SafeIntent may include a guest path.
type RecoveryTarget struct {
	ID, TrackingID, Action, State, NativeID, UUID, Incarnation, ErrorClass string
	Layer, Resource, SafeIntent, Name, OwnedState                          string
	UpdatedAt                                                              int64
	Released, DependentDelete                                              bool
}

func readRecoveryTarget(ctx context.Context, q credentialRowQuery, layer, id string) (RecoveryTarget, error) {
	var t RecoveryTarget
	if layer == "" || !validHexID(id) {
		return t, ErrInvalid
	}
	e := q.QueryRowContext(ctx, `SELECT o.id,o.tracking_id,o.action,o.state,o.native_operation_id,o.box_uuid,o.incarnation,o.error_class,
 o.layer,o.resource_key,o.safe_intent,o.updated_at,COALESCE(b.name,''),COALESCE(b.state,''),
 EXISTS(SELECT 1 FROM recovery_audit a WHERE a.operation_id=o.id AND a.decision='RELEASED'),
 EXISTS(SELECT 1 FROM operation_resource_claims c WHERE c.operation_id=o.id AND c.layer=o.layer AND c.resource_key='delete-project:'||o.layer)
 FROM operations o LEFT JOIN owned_boxes b ON b.uuid=o.box_uuid AND b.incarnation=o.incarnation AND b.project=o.layer
 WHERE o.id=? AND o.layer=?`, id, layer).Scan(&t.ID, &t.TrackingID, &t.Action, &t.State, &t.NativeID, &t.UUID, &t.Incarnation, &t.ErrorClass, &t.Layer, &t.Resource, &t.SafeIntent, &t.UpdatedAt, &t.Name, &t.OwnedState, &t.Released, &t.DependentDelete)
	if errors.Is(e, sql.ErrNoRows) {
		return t, ErrNotFound
	}
	if e != nil {
		return t, e
	}
	if t.Action == "box_create" && t.UUID == "" && t.Incarnation == "" {
		var in struct {
			Name string `json:"name"`
		}
		if json.Unmarshal([]byte(t.SafeIntent), &in) == nil && in.Name != "" && t.Resource == "box:"+in.Name {
			t.Name = in.Name
		}
	}
	return t, nil
}

func (s *Store) ReadRecoveryTarget(ctx context.Context, layer, id string) (RecoveryTarget, error) {
	return readRecoveryTarget(ctx, s.db, layer, id)
}

// RecoveryRequest is built by an authenticated handler or the existing
// kernel-verified host-local root worker. LocalReviewer is never a wire field.
type RecoveryRequest struct {
	Layer, ActorID, TokenID, LocalReviewer, OriginalID, Stream, Key, WorkRef, AgentRef string
}
type RecoveryProof struct {
	Kind, Reason, NativeID, UUID, Fingerprint string
	Exit                                      *int
	StdoutPath, StderrPath                    string
}

const (
	RecoverySettle    = "SETTLE"
	RecoveryBind      = "BIND_IDENTITY"
	RecoveryApproval  = "APPROVAL_REQUIRED"
	RecoveryBlocked   = "BLOCKED"
	RecoveryNotNeeded = "NOT_REQUIRED"
)

type RecoveryRecord struct {
	Receipt     Receipt `json:"operation"`
	OriginalID  string  `json:"original_operation_id"`
	Disposition string  `json:"disposition"`
	Reason      string  `json:"reason"`
}

func recoveryAdmission(r RecoveryRequest) (Admission, error) {
	if r.Layer == "" || !validHexID(r.OriginalID) || len(r.Key) < 16 || len(r.Key) > 128 || !validRef(r.WorkRef) || !validRef(r.AgentRef) || (r.Stream != "" && r.Stream != "stdout" && r.Stream != "stderr") {
		return Admission{}, ErrInvalid
	}
	actor, token := r.ActorID, r.TokenID
	if r.LocalReviewer != "" {
		if !safeReviewer.MatchString(r.LocalReviewer) || actor != "" || token != "" {
			return Admission{}, ErrInvalid
		}
		actor, token = "host-local:"+r.LocalReviewer, "host-local"
	} else if !validHexID(actor) || !validHexID(token) {
		return Admission{}, ErrInvalid
	}
	intent, _ := json.Marshal(struct {
		Original string `json:"original_operation_id"`
		Stream   string `json:"stream,omitempty"`
	}{r.OriginalID, r.Stream})
	return Admission{Layer: r.Layer, ActorID: actor, TokenID: token, Key: r.Key, Action: "operation_recover", SafeIntent: intent, WorkRef: r.WorkRef, AgentRef: r.AgentRef}, nil
}

func recoveryAuthorized(ctx context.Context, q credentialRowQuery, r RecoveryRequest, a Admission) error {
	if r.LocalReviewer != "" {
		return nil
	}
	ok, e := activeCredentialForActor(ctx, q, a.ActorID, a.TokenID)
	if e != nil {
		return e
	}
	if !ok {
		return ErrRevoked
	}
	return nil
}

func readRecoveryReplay(ctx context.Context, q credentialRowQuery, r RecoveryRequest, a Admission) (RecoveryRecord, error) {
	var out RecoveryRecord
	var hash, detail string
	v := &out.Receipt
	e := q.QueryRowContext(ctx, `SELECT id,tracking_id,state,native_operation_id,work_ref,agent_ref,action,box_uuid,incarnation,error_class,created_at,updated_at,intent_digest,
 COALESCE((SELECT safe_detail FROM operation_events e WHERE e.operation_id=o.id ORDER BY seq DESC LIMIT 1),'')
 FROM operations o WHERE layer=? AND actor_id=? AND key_digest=?`, a.Layer, a.ActorID, digest(a.Key)).
		Scan(&v.ID, &v.TrackingID, &v.State, &v.NativeOperationID, &v.WorkRef, &v.AgentRef, &v.Action, &v.BoxUUID, &v.Incarnation, &v.ErrorClass, &v.CreatedAt, &v.UpdatedAt, &hash, &detail)
	if errors.Is(e, sql.ErrNoRows) {
		return out, ErrNotFound
	}
	if e != nil {
		return out, e
	}
	if v.Action != a.Action || hash != admissionIntentDigest(a, nil) || (a.WorkRef != "" && a.WorkRef != v.WorkRef) || (a.AgentRef != "" && a.AgentRef != v.AgentRef) {
		return RecoveryRecord{}, ErrConflict
	}
	d, reason, ok := strings.Cut(detail, ":")
	if !ok || !validRecoveryDetail(d, reason) {
		return RecoveryRecord{}, ErrState
	}
	if (v.State == "SUCCEEDED") != (d == "SETTLED" || d == "IDENTITY_BOUND") || (v.State != "SUCCEEDED" && v.State != "FAILED") {
		return RecoveryRecord{}, ErrState
	}
	v.Duplicate = true
	v.NextAction = nextAction(v.State, v.Action, v.ID)
	out.OriginalID = r.OriginalID
	out.Disposition = d
	out.Reason = reason
	return out, nil
}
func (s *Store) ReplayRecovery(ctx context.Context, r RecoveryRequest) (RecoveryRecord, error) {
	a, e := recoveryAdmission(r)
	if e != nil {
		return RecoveryRecord{}, e
	}
	if e = recoveryAuthorized(ctx, s.db, r, a); e != nil {
		return RecoveryRecord{}, e
	}
	return readRecoveryReplay(ctx, s.db, r, a)
}
func validRecoveryDetail(d, reason string) bool {
	switch d {
	case "SETTLED", "IDENTITY_BOUND", RecoveryApproval, RecoveryBlocked, RecoveryNotNeeded:
	default:
		return false
	}
	if len(reason) == 0 || len(reason) > 80 {
		return false
	}
	for _, c := range reason {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

// Recovery is a narrowly scoped, read-only-native operation. The receipt and
// any original settlement commit together. No ordinary admission barrier is
// bypassed: no native dispatch, no resource mutation, no RELEASED audit.
// An interrupted observation before commit has no effect; after commit the
// same key returns exactly the durable result, even if the original changed.
func (s *Store) CommitRecovery(ctx context.Context, r RecoveryRequest, t RecoveryTarget, p RecoveryProof) (RecoveryRecord, error) {
	a, e := recoveryAdmission(r)
	if e != nil {
		return RecoveryRecord{}, e
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return RecoveryRecord{}, e
	}
	defer tx.Rollback()
	if e = recoveryAuthorized(ctx, tx, r, a); e != nil {
		return RecoveryRecord{}, e
	}
	if prior, err := readRecoveryReplay(ctx, tx, r, a); err == nil {
		return prior, tx.Commit()
	} else if !errors.Is(err, ErrNotFound) {
		return RecoveryRecord{}, err
	}
	nowTarget, e := readRecoveryTarget(ctx, tx, r.Layer, r.OriginalID)
	if e != nil {
		return RecoveryRecord{}, e
	}
	if nowTarget != t {
		return RecoveryRecord{}, ErrConflict
	}
	id, e := newID()
	if e != nil {
		return RecoveryRecord{}, e
	}
	tracking, e := newID()
	if e != nil {
		return RecoveryRecord{}, e
	}
	now := time.Now().UnixNano()
	outcome := p.Kind
	success := false
	if p.Kind == RecoverySettle || p.Kind == RecoveryBind {
		if r.Stream != "" || t.State != "UNKNOWN" || t.Released || t.DependentDelete || t.Name == "" || t.Resource != "box:"+t.Name {
			return RecoveryRecord{}, ErrResourceLocked
		}
		var busy int
		e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations o WHERE o.layer=? AND o.id<>? AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN','OBSERVING') AND
   (o.action='box_peer_set' OR NOT EXISTS(SELECT 1 FROM recovery_audit a WHERE a.operation_id=o.id AND a.decision='RELEASED')) AND
   (o.action='box_peer_set' OR o.resource_key=? OR (?<>'' AND o.box_uuid=? AND o.incarnation=?) OR EXISTS(SELECT 1 FROM operation_resource_claims c WHERE c.operation_id=o.id AND c.resource_key='delete-project:'||o.layer))`,
			t.Layer, t.ID, t.Resource, t.UUID, t.UUID, t.Incarnation).Scan(&busy)
		if e != nil {
			return RecoveryRecord{}, e
		}
		if busy != 0 {
			return RecoveryRecord{}, ErrResourceLocked
		}
		e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM exec_output_gc WHERE project=? AND state IN ('DELETE_SENT','REVIEW_REQUIRED') AND ((?<>'' AND box_uuid=?) OR name=?)`, t.Layer, t.UUID, t.UUID, t.Name).Scan(&busy)
		if e != nil {
			return RecoveryRecord{}, e
		}
		if busy != 0 {
			return RecoveryRecord{}, ErrResourceLocked
		}
		if p.Kind == RecoveryBind {
			if t.Action != "box_create" || t.UUID != "" || t.Incarnation != "" || p.UUID == "" || p.Fingerprint == "" || p.Reason != "CREATE_MARKER_EXACT_IDENTITY" {
				return RecoveryRecord{}, ErrInvalid
			}
			if e = bindUnknownCreateTx(ctx, tx, t.ID, t.Layer, t.Name, t.UpdatedAt, p.UUID, "EXACT_NATIVE_UUID_BOUND_RECOVERY_NO_REPLAY"); e != nil {
				return RecoveryRecord{}, e
			}
			outcome = "IDENTITY_BOUND"
		} else {
			if t.OwnedState != "OWNED" || t.UUID == "" || t.Incarnation == "" || p.UUID != t.UUID || t.NativeID == "" || p.NativeID != t.NativeID || p.Fingerprint == "" || p.Reason != "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED" {
				return RecoveryRecord{}, ErrInvalid
			}
			box := OwnedBox{UUID: t.UUID, Project: t.Layer, Name: t.Name, Incarnation: t.Incarnation, State: "OWNED"}
			switch t.Action {
			case "box_delete":
				if !strings.HasPrefix(p.Fingerprint, "DELETE_SUCCESS:") {
					return RecoveryRecord{}, ErrInvalid
				}
				e = finishVerifiedDeleteTx(ctx, tx, t.ID, box, "UNKNOWN", "RECOVERY_DELETE_TERMINAL_NO_REPLAY")
			case "box_exec":
				if p.Exit == nil {
					return RecoveryRecord{}, ErrInvalid
				}
				e = finishVerifiedExecTx(ctx, tx, t.ID, box, t.NativeID, p.StdoutPath, p.StderrPath, *p.Exit, "UNKNOWN", "RECOVERY_EXEC_TERMINAL_NO_REPLAY")
			case "box_start", "box_stop", "box_update":
				var res sql.Result
				res, e = tx.ExecContext(ctx, `UPDATE operations SET state='SUCCEEDED',error_class='',updated_at=? WHERE id=? AND state='UNKNOWN' AND updated_at=?`, now, t.ID, t.UpdatedAt)
				if e == nil {
					n, x := res.RowsAffected()
					if x != nil || n != 1 {
						return RecoveryRecord{}, ErrConflict
					}
					_, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'SUCCEEDED','RECOVERY_NATIVE_TERMINAL_NO_REPLAY')", t.ID, now)
				}
			default:
				return RecoveryRecord{}, ErrInvalid
			}
			if e != nil {
				return RecoveryRecord{}, e
			}
			outcome = "SETTLED"
		}
		success = true
	} else if p.Kind != RecoveryApproval && p.Kind != RecoveryBlocked && p.Kind != RecoveryNotNeeded {
		return RecoveryRecord{}, ErrInvalid
	}
	if !validRecoveryDetail(outcome, p.Reason) {
		return RecoveryRecord{}, ErrInvalid
	}
	phase, errClass := "FAILED", "RECOVERY_"+outcome
	if success {
		phase, errClass = "SUCCEEDED", ""
	}
	uuid, inc := t.UUID, t.Incarnation
	if outcome == "IDENTITY_BOUND" {
		uuid, inc = p.UUID, p.UUID
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO operations(id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,intent_digest,safe_intent,work_ref,agent_ref,resource_key,box_uuid,incarnation,state,error_class,created_at,updated_at)
 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, tracking, id, a.Layer, a.ActorID, a.TokenID, digest(a.Key), a.Action, admissionIntentDigest(a, nil), a.SafeIntent, a.WorkRef, a.AgentRef, "", uuid, inc, phase, errClass, now, now)
	if e != nil {
		return RecoveryRecord{}, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'ADMITTED','RECOVERY_READ_ONLY_NATIVE')", id, now); e != nil {
		return RecoveryRecord{}, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,?,?)", id, now, phase, outcome+":"+p.Reason); e != nil {
		return RecoveryRecord{}, e
	}
	// Link audit to immutable original; digest includes exact recovery receipt,
	// original version and the bounded observation. No native or guest bytes.
	if success {
		raw, _ := json.Marshal([]any{id, t.ID, t.UpdatedAt, t.NativeID, t.UUID, outcome, p.Reason, p.Fingerprint})
		if _, e = tx.ExecContext(ctx, "INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,?,?,?,?)", t.ID, now, a.ActorID, "RECOVERY_"+outcome, digest(string(raw))); e != nil {
			return RecoveryRecord{}, e
		}
	}
	if e = tx.Commit(); e != nil {
		return RecoveryRecord{}, e
	}
	return RecoveryRecord{Receipt: Receipt{ID: id, TrackingID: tracking, State: phase, Action: a.Action, BoxUUID: uuid, Incarnation: inc, WorkRef: a.WorkRef, AgentRef: a.AgentRef, ErrorClass: errClass, CreatedAt: now, UpdatedAt: now, NextAction: nextAction(phase, a.Action, id)}, OriginalID: t.ID, Disposition: outcome, Reason: p.Reason}, nil
}
