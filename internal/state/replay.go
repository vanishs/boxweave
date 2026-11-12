package state

import (
	"context"
	"database/sql"
	"errors"
)

// Replay returns a verified original receipt without reading a possibly
// missing/tombstoned native Box. It is used BEFORE live native preflight for
// mutating Tool retries; it never dispatches or changes any operation state.
// The caller must already be authenticated on the exact original actor/layer.
func (s *Store) Replay(ctx context.Context, a Admission) (Receipt, error) {
	if a.Layer == "" || a.ActorID == "" || a.TokenID != a.ActorID ||
		len(a.Key) < 16 || len(a.Key) > 128 || a.Action == "" ||
		len(a.SafeIntent) == 0 || len(a.SafeIntent) > 65536 ||
		!validRef(a.WorkRef) || !validRef(a.AgentRef) {
		return Receipt{}, ErrInvalid
	}
	active, err := s.tokenActive(ctx, a.TokenID)
	if err != nil {
		return Receipt{}, err
	}
	if !active {
		return Receipt{}, ErrRevoked
	}
	var r Receipt
	var hash, token string
	err = s.db.QueryRowContext(ctx, `SELECT id,tracking_id,state,native_operation_id,
		work_ref,agent_ref,action,box_uuid,incarnation,error_class,created_at,
		updated_at,intent_digest,token_id FROM operations
		WHERE layer=? AND actor_id=? AND key_digest=?`,
		a.Layer, a.ActorID, digest(a.Key)).Scan(
		&r.ID, &r.TrackingID, &r.State, &r.NativeOperationID,
		&r.WorkRef, &r.AgentRef, &r.Action, &r.BoxUUID, &r.Incarnation,
		&r.ErrorClass, &r.CreatedAt, &r.UpdatedAt, &hash, &token)
	if errors.Is(err, sql.ErrNoRows) {
		return Receipt{}, ErrNotFound
	}
	if err != nil {
		return Receipt{}, err
	}
	expected := digest(a.Action + "\x00" + string(a.SafeIntent) + "\x00" +
		a.BoxUUID + "\x00" + a.Incarnation)
	if hash != expected || token != a.TokenID ||
		(a.WorkRef != "" && a.WorkRef != r.WorkRef) ||
		(a.AgentRef != "" && a.AgentRef != r.AgentRef) {
		return Receipt{}, ErrConflict
	}
	r.Duplicate = true
	r.NextAction = nextAction(r.State, r.Action, r.ID)
	return r, nil
}
