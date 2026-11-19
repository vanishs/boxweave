package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
)

// StageRootPeerRevocation changes ONLY an already-confirmed exact peer grant
// into a durable pending revoke. It takes an admitted operation ID, NEVER a
// caller-controlled root tuple, and does not touch Incus or conntrack.
// A crash after staging retains the SQLite multi-resource operation claim.
func (s *Store) StageRootPeerRevocation(ctx context.Context, id string) error {
	if s == nil || s.db == nil || s.auditReadOnly || !validHexID(id) {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var action, state, actor, token, layer, boxUUID, incarnation string
	var raw []byte
	if err := tx.QueryRowContext(ctx, `SELECT action,state,actor_id,token_id,layer,box_uuid,incarnation,safe_intent
		FROM operations WHERE id=?`, id).
		Scan(&action, &state, &actor, &token, &layer, &boxUUID, &incarnation, &raw); err != nil {
		return err
	}
	if action != "box_peer_set" || state != "ADMITTED" || actor == "" || token == "" {
		return ErrState
	}
	var intent RootPeerRevokeSafeIntent
	if err := json.Unmarshal(raw, &intent); err != nil {
		return ErrInvalid
	}
	canonical, err := json.Marshal(intent)
	if err != nil || !bytes.Equal(raw, canonical) || intent.Allow ||
		intent.Project == "default" || intent.Project != layer ||
		!auditName(intent.Project) || !auditName(intent.Bridge) || !auditName(intent.ACL) ||
		intent.SourceUUID == "" || intent.TargetUUID == "" ||
		intent.SourceUUID == intent.TargetUUID ||
		boxUUID != intent.SourceUUID || incarnation != intent.SourceIncarnation {
		return ErrInvalid
	}
	active, err := activeCredentialForActor(ctx, tx, actor, token)
	if err != nil || !active {
		return ErrRevoked
	}
	// Independently enforce the all-NIC Project barrier at staging, including
	// old databases containing an in-flight sibling admitted before this fix.
	var siblingBusy int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations o
		WHERE o.layer=? AND o.id<>?
		AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')
		AND (o.action='box_peer_set' OR NOT EXISTS (SELECT 1 FROM recovery_audit r
		  WHERE r.operation_id=o.id AND r.decision='RELEASED'))`,
		intent.Project, id).Scan(&siblingBusy); err != nil || siblingBusy != 0 {
		return ErrResourceLocked
	}
	// All three exact claims must have been acquired atomically at Admit.
	rows, err := tx.QueryContext(ctx,
		"SELECT resource_key FROM operation_resource_claims WHERE operation_id=? ORDER BY resource_key", id)
	if err != nil {
		return err
	}
	var claims []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return err
		}
		claims = append(claims, key)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	expected := []string{
		"acl:default/" + intent.ACL,
		"box:" + intent.SourceUUID,
		"box:" + intent.TargetUUID,
	}
	slices.Sort(expected)
	if !slices.Equal(claims, expected) {
		return ErrResourceLocked
	}
	// Pending or UNKNOWN sibling directions must never be dropped
	// during an otherwise valid whole-policy ACL rewrite.
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM peer_grants WHERE project=? AND status IN ('PENDING','UNKNOWN')",
		intent.Project).Scan(&siblingBusy); err != nil || siblingBusy != 0 {
		return ErrResourceLocked
	}
	// The row cannot be staged from a prior UNKNOWN or guessed foreign
	// grant, and its native identity must be exactly pinned in SQLite.
	var srcInc, dstInc, src4, dst4, src6, dst6 string
	err = tx.QueryRowContext(ctx, `SELECT source_incarnation,target_incarnation,
		source_ipv4,target_ipv4,source_ipv6,target_ipv6 FROM peer_grants
		WHERE project=? AND source_uuid=? AND target_uuid=? AND desired=1 AND status='CONFIRMED'`,
		intent.Project, intent.SourceUUID, intent.TargetUUID).
		Scan(&srcInc, &dstInc, &src4, &dst4, &src6, &dst6)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrResourceLocked
	}
	if err != nil {
		return err
	}
	if srcInc != intent.SourceIncarnation || dstInc != intent.TargetIncarnation ||
		!auditIP(src4, false) || !auditIP(dst4, false) ||
		!auditIP(src6, true) || !auditIP(dst6, true) {
		return ErrResourceLocked
	}
	for _, peer := range []struct{ uuid, incarnation, v4, v6 string }{
		{intent.SourceUUID, srcInc, src4, src6},
		{intent.TargetUUID, dstInc, dst4, dst6},
	} {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*)
			FROM owned_boxes b JOIN network_bindings n
			  ON b.uuid=n.box_uuid AND b.project=n.project AND b.incarnation=n.incarnation
			WHERE b.project=? AND b.uuid=? AND b.incarnation=? AND b.state='OWNED'
			  AND n.state='PINNED' AND n.nic='eth0' AND n.bridge=? AND n.acl=?
			  AND n.ipv4=? AND n.ipv6=?`,
			intent.Project, peer.uuid, peer.incarnation,
			intent.Bridge, intent.ACL, peer.v4, peer.v6).Scan(&count); err != nil || count != 1 {
			return ErrResourceLocked
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE peer_grants SET desired=0,status='PENDING',operation_id=?
		WHERE project=? AND source_uuid=? AND target_uuid=? AND
		      source_incarnation=? AND target_incarnation=? AND desired=1 AND status='CONFIRMED'`,
		id, intent.Project, intent.SourceUUID, intent.TargetUUID, srcInc, dstInc)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return ErrResourceLocked
	}
	return tx.Commit()
}
