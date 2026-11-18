package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// PeerOperationDirection reads ONLY the immutable, canonical direction of
// one exact original. This is a dispatcher selector, NOT authorization to
// stage, approve or replay a native effect; the grant/revoke preflights still
// independently validate every actor, claim, native NIC and kernel witness.
func (s *Store) PeerOperationDirection(ctx context.Context, original, expectedState string) (bool, error) {
	if s == nil || s.db == nil || !validHexID(original) ||
		(expectedState != "ADMITTED" && expectedState != "DISPATCHING" && expectedState != "UNKNOWN") {
		return false, ErrInvalid
	}
	var action, status, layer, src, incarnation, actor, token string
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT action,state,layer,box_uuid,incarnation,actor_id,token_id,safe_intent
		FROM operations WHERE id=?`, original).
		Scan(&action, &status, &layer, &src, &incarnation, &actor, &token, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, err
	}
	if action != "box_peer_set" || status != expectedState || actor == "" || token == "" {
		return false, ErrResourceLocked
	}
	var intent RootPeerRevokeSafeIntent
	if err := json.Unmarshal(raw, &intent); err != nil {
		return false, ErrResourceLocked
	}
	canonical, err := json.Marshal(intent)
	if err != nil || !bytes.Equal(canonical, raw) ||
		!auditName(intent.Project) || intent.Project == "default" ||
		!auditName(intent.Bridge) || !auditName(intent.ACL) ||
		layer != intent.Project || src != intent.SourceUUID ||
		incarnation != intent.SourceIncarnation ||
		intent.SourceUUID == "" || intent.TargetUUID == "" ||
		intent.SourceUUID == intent.TargetUUID ||
		intent.SourceIncarnation == "" || intent.TargetIncarnation == "" {
		return false, ErrResourceLocked
	}
	return intent.Allow, nil
}
