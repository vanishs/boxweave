package state

import (
	"context"
	"time"
)

// A new durable local stage marker is not proof of absent native effects
// from historical binaries or another authority. UNKNOWN is never released
// by these observations. Each exact original may enter each stage once.
func (s *Store) recordPreparedRootStage(ctx context.Context, id, stage string) error {
	if s == nil || s.db == nil || s.auditReadOnly || !validHexID(id) {
		return ErrInvalid
	}
	required := ""
	switch stage {
	case "PEER_ROOT_H3B_REVIEW_STARTED":
	case "PEER_ROOT_H3B_EFFECT_DISPATCH":
		required = "PEER_ROOT_H3B_REVIEW_STARTED"
	default:
		return ErrInvalid
	}
	res, e := s.db.ExecContext(ctx, `INSERT INTO operation_events(operation_id,at,state,safe_detail)
        SELECT id,?,'DISPATCHING',? FROM operations
        WHERE id=? AND state='DISPATCHING' AND action='box_peer_set'
        AND NOT EXISTS(SELECT 1 FROM operation_events WHERE operation_id=? AND safe_detail=?)
        AND (?='' OR EXISTS(SELECT 1 FROM operation_events WHERE operation_id=? AND safe_detail=?))`,
		time.Now().UnixNano(), stage, id, id, stage, required, id, required)
	if e != nil {
		return e
	}
	if n, e := res.RowsAffected(); e != nil || n != 1 {
		return ErrState
	}
	return nil
}
