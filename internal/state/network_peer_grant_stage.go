package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
)

// StagePeerGrant records an inert, directional B3 grant intent. It does not
// perform an ACL PUT or claim native success. The future native grant worker
// must independently revalidate its protected original and every NIC.
// The project's all-sibling admission barrier remains held throughout.
func (s *Store) StagePeerGrant(ctx context.Context, original string) error {
	if s == nil || s.db == nil || s.auditReadOnly || !validHexID(original) {
		return ErrInvalid
	}
	var action, status, actor, token, layer, boxUUID, incarnation string
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT action,state,actor_id,token_id,layer,box_uuid,incarnation,safe_intent
  FROM operations WHERE id=?`, original).Scan(&action, &status, &actor, &token, &layer, &boxUUID, &incarnation, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	var intent RootPeerRevokeSafeIntent
	if json.Unmarshal(raw, &intent) != nil {
		return ErrInvalid
	}
	canonical, e := json.Marshal(intent)
	if e != nil || !bytes.Equal(canonical, raw) || !intent.Allow ||
		action != "box_peer_set" || status != "ADMITTED" || actor == "" || token == "" ||
		intent.Project == "default" || intent.Project != layer || !auditName(layer) ||
		!auditName(intent.Bridge) || !auditName(intent.ACL) ||
		intent.SourceUUID == "" || intent.SourceUUID == intent.TargetUUID ||
		boxUUID != intent.SourceUUID || incarnation != intent.SourceIncarnation ||
		intent.SourceIncarnation == "" || intent.TargetIncarnation == "" {
		return ErrInvalid
	}

	// This demands a complete private-project inventory, pinned IP/MAC/NIC and
	// exclusive address fences, never merely an observed guest DHCP address.
	ledger, err := s.ReadCompleteNetworkAuditLedger(ctx, intent.Project, intent.Bridge, intent.ACL)
	if err != nil {
		return ErrResourceLocked
	}
	var src, dst NetworkAuditBinding
	for _, b := range ledger.Boxes {
		if b.UUID == intent.SourceUUID {
			src = b
		}
		if b.UUID == intent.TargetUUID {
			dst = b
		}
	}
	if src.UUID == "" || dst.UUID == "" || src.Incarnation != intent.SourceIncarnation ||
		dst.Incarnation != intent.TargetIncarnation {
		return ErrResourceLocked
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Recheck exact original under the same transaction as the durable grant.
	var still int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE id=? AND action='box_peer_set'
  AND state='ADMITTED' AND actor_id=? AND token_id=? AND layer=? AND
  box_uuid=? AND incarnation=? AND safe_intent=?`,
		original, actor, token, layer, boxUUID, incarnation, raw).Scan(&still); err != nil || still != 1 {
		return ErrResourceLocked
	}
	var role string
	if err := tx.QueryRowContext(ctx, "SELECT role FROM credentials WHERE id=?", token).Scan(&role); err != nil || role != "admin" {
		return ErrRevoked
	}
	active, err := activeCredentialForActor(ctx, tx, actor, token)
	if err != nil || !active {
		return ErrRevoked
	}

	var busy int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations o WHERE o.layer=? AND o.id<>?
 AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')
 AND (o.action='box_peer_set' OR NOT EXISTS (SELECT 1 FROM recovery_audit r
 WHERE r.operation_id=o.id AND r.decision='RELEASED'))`,
		intent.Project, original).Scan(&busy); err != nil || busy != 0 {
		return ErrResourceLocked
	}
	rows, err := tx.QueryContext(ctx, "SELECT resource_key FROM operation_resource_claims WHERE operation_id=? ORDER BY resource_key", original)
	if err != nil {
		return err
	}
	var claims []string
	for rows.Next() {
		var k string
		if e := rows.Scan(&k); e != nil {
			rows.Close()
			return e
		}
		claims = append(claims, k)
	}
	if e := rows.Err(); e != nil {
		rows.Close()
		return e
	}
	if e := rows.Close(); e != nil {
		return e
	}
	expected := []string{"acl:default/" + intent.ACL, "box:" + src.UUID, "box:" + dst.UUID}
	slices.Sort(expected)
	if !slices.Equal(expected, claims) {
		return ErrResourceLocked
	}

	// Refuse a stale or withdrawn address lease even if it changed after the
	// preliminary complete-inventory read. Root must repeat native checks.
	for _, b := range []NetworkAuditBinding{src, dst} {
		var n int
		if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM owned_boxes ob JOIN network_bindings nb
   ON nb.project=ob.project AND nb.box_uuid=ob.uuid AND nb.incarnation=ob.incarnation
   WHERE ob.project=? AND ob.uuid=? AND ob.incarnation=? AND ob.state='OWNED'
   AND nb.state='PINNED' AND nb.nic='eth0' AND nb.bridge=? AND nb.acl=? AND
   nb.ipv4=? AND nb.ipv6=? AND nb.mac=?`,
			layer, b.UUID, b.Incarnation, intent.Bridge, intent.ACL, b.IPv4, b.IPv6, b.MAC).Scan(&n); e != nil || n != 1 {
			return ErrResourceLocked
		}
		for _, f := range []struct{ family, ip string }{{"ipv4", b.IPv4}, {"ipv6", b.IPv6}} {
			if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_address_fences WHERE bridge=?
    AND family=? AND address=? AND project=? AND box_uuid=? AND incarnation=? AND state='RESERVED'`,
				intent.Bridge, f.family, f.ip, layer, b.UUID, b.Incarnation).Scan(&n); e != nil || n != 1 {
				return ErrResourceLocked
			}
		}
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM peer_grants WHERE project=?
 AND status IN ('PENDING','UNKNOWN')`, layer).Scan(&busy); err != nil || busy != 0 {
		return ErrResourceLocked
	}
	// Re-grant of an EXACT previously revoked incarnation is permitted; old
	// unknown/stale rows never inherit a new UUID or address.
	changed, err := tx.ExecContext(ctx, `UPDATE peer_grants SET desired=1,status='PENDING',operation_id=?
 WHERE project=? AND source_uuid=? AND source_incarnation=? AND
 target_uuid=? AND target_incarnation=? AND status='REVOKED' AND desired=0 AND
 source_ipv4=? AND target_ipv4=? AND source_ipv6=? AND target_ipv6=?`,
		original, layer, src.UUID, src.Incarnation, dst.UUID, dst.Incarnation, src.IPv4, dst.IPv4, src.IPv6, dst.IPv6)
	if err != nil {
		return err
	}
	n, err := changed.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		_, err = tx.ExecContext(ctx, `INSERT INTO peer_grants
   (project,source_uuid,source_incarnation,target_uuid,target_incarnation,
    desired,status,source_ipv4,target_ipv4,source_ipv6,target_ipv6,operation_id)
   VALUES(?,?,?,?,?,1,'PENDING',?,?,?,?,?)`,
			layer, src.UUID, src.Incarnation, dst.UUID, dst.Incarnation, src.IPv4, dst.IPv4, src.IPv6, dst.IPv6, original)
		if err != nil {
			return ErrResourceLocked
		}
	} else if n != 1 {
		return ErrResourceLocked
	}
	return tx.Commit()
}
