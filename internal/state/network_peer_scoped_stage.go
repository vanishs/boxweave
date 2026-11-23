package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"time"
)

// StageScopedPeerOriginal stages exactly one v2 grant/revoke in the SAME
// SQLite write transaction that audits every owned Box, deterministic ACL
// and dual-stack lease, native grant provenance, actor and resource claims.
// It performs NO Incus/nft/conntrack/boot mutation; no MCP/Root caller exists
// for this internal pathway until the full migration+recovery is accepted.
func (s *Store) StageScopedPeerOriginal(ctx context.Context, id string) error {
	if s == nil || s.db == nil || s.auditReadOnly || !validHexID(id) {
		return ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var action, status, actor, token, layer, source, incarnation, resource string
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT action,state,actor_id,token_id,layer,box_uuid,
  incarnation,resource_key,safe_intent FROM operations WHERE id=?`, id).
		Scan(&action, &status, &actor, &token, &layer, &source, &incarnation, &resource, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	var i ScopedPeerOriginalIntent
	if json.Unmarshal(raw, &i) != nil {
		return ErrResourceLocked
	}
	canonical, e := json.Marshal(i)
	if e != nil || !bytes.Equal(canonical, raw) || i.Schema != 2 ||
		action != "box_peer_set" || status != "ADMITTED" || actor == "" || token == "" ||
		i.Project != layer || i.Bridge == "" || source != i.SourceUUID ||
		incarnation != i.SourceIncarnation || i.SourceUUID == i.TargetUUID ||
		i.SourceUUID == "" || i.TargetUUID == "" || i.SourceIncarnation == "" ||
		i.TargetIncarnation == "" || resource != "peer-guard:"+i.Bridge {
		return ErrResourceLocked
	}
	inventory, e := readCompleteScopedNetworkAuditLedgerTx(ctx, tx, i.Project, i.Bridge)
	if e != nil || len(inventory.Boxes) < 2 {
		return ErrResourceLocked
	}
	owners := map[string]ScopedNetworkAuditBinding{}
	for _, b := range inventory.Boxes {
		owners[b.UUID] = b
	}
	src, srcOK := owners[i.SourceUUID]
	dst, dstOK := owners[i.TargetUUID]
	if !srcOK || !dstOK || src.Incarnation != i.SourceIncarnation ||
		dst.Incarnation != i.TargetIncarnation {
		return ErrResourceLocked
	}
	expectedClaims, e := scopedPeerResourceClaims(i)
	if e != nil {
		return ErrResourceLocked
	}
	rows, e := tx.QueryContext(ctx, `SELECT resource_key FROM operation_resource_claims
  WHERE operation_id=? ORDER BY resource_key`, id)
	if e != nil {
		return e
	}
	var claims []string
	for rows.Next() {
		var k string
		if e = rows.Scan(&k); e != nil {
			break
		}
		claims = append(claims, k)
	}
	if x := rows.Err(); e == nil {
		e = x
	}
	if x := rows.Close(); e == nil {
		e = x
	}
	if e != nil {
		return e
	}
	if !slices.Equal(expectedClaims, claims) {
		return ErrResourceLocked
	}
	active, e := activeCredentialForActor(ctx, tx, actor, token)
	if e != nil || !active {
		return ErrRevoked
	}
	var busy int
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations o WHERE o.layer=? AND o.id<>?
  AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')
  AND (o.action='box_peer_set' OR NOT EXISTS (SELECT 1 FROM recovery_audit r
  WHERE r.operation_id=o.id AND r.decision='RELEASED'))`, i.Project, id).Scan(&busy)
	if e != nil || busy != 0 {
		return ErrResourceLocked
	}
	rows, e = tx.QueryContext(ctx, `SELECT source_uuid,source_incarnation,target_uuid,
  target_incarnation,source_ipv4,target_ipv4,source_ipv6,target_ipv6,
  desired,status,operation_id,COALESCE(last_verified_operation_id,'')
  FROM peer_grants WHERE project=? ORDER BY source_uuid,target_uuid`, i.Project)
	if e != nil {
		return e
	}
	found := false
	total := 0
	for rows.Next() {
		var p PeerGrantBinding
		var desired int
		var state, original, verified string
		if e = rows.Scan(&p.SourceUUID, &p.SourceIncarnation, &p.TargetUUID, &p.TargetIncarnation,
			&p.SourceIPv4, &p.TargetIPv4, &p.SourceIPv6, &p.TargetIPv6,
			&desired, &state, &original, &verified); e != nil {
			break
		}
		a, okA := owners[p.SourceUUID]
		b, okB := owners[p.TargetUUID]
		total++
		if total > 32 || !okA || !okB || p.SourceUUID == p.TargetUUID ||
			p.SourceIncarnation != a.Incarnation || p.TargetIncarnation != b.Incarnation ||
			p.SourceIPv4 != a.IPv4 || p.TargetIPv4 != b.IPv4 ||
			p.SourceIPv6 != a.IPv6 || p.TargetIPv6 != b.IPv6 ||
			!(desired == 1 && state == "CONFIRMED" || desired == 0 && state == "REVOKED") {
			e = ErrResourceLocked
			break
		}
		e = scopedPeerVerifyTerminalLineageTx(ctx, tx, i.Project, i.Bridge,
			p, desired, state, original, verified)
		if e != nil {
			break
		}
		if p.SourceUUID == i.SourceUUID && p.TargetUUID == i.TargetUUID {
			if found || i.Allow && state != "REVOKED" || !i.Allow && state != "CONFIRMED" {
				e = ErrResourceLocked
				break
			}
			found = true
		}
	}
	if x := rows.Err(); e == nil {
		e = x
	}
	if x := rows.Close(); e == nil {
		e = x
	}
	if e != nil {
		return e
	}
	if i.Allow && !found && total >= 32 {
		return ErrResourceLocked
	}
	var affected int64
	if i.Allow {
		if found {
			res, e := tx.ExecContext(ctx, `UPDATE peer_grants SET desired=1,status='PENDING',
    operation_id=? WHERE project=? AND source_uuid=? AND target_uuid=?
    AND source_incarnation=? AND target_incarnation=? AND desired=0
    AND status='REVOKED' AND operation_id=last_verified_operation_id
    AND last_verified_operation_id IS NOT NULL
    AND source_ipv4=? AND target_ipv4=?
    AND source_ipv6=? AND target_ipv6=?`,
				id, i.Project, src.UUID, dst.UUID, src.Incarnation, dst.Incarnation,
				src.IPv4, dst.IPv4, src.IPv6, dst.IPv6)
			if e != nil {
				return e
			}
			affected, e = res.RowsAffected()
			if e != nil {
				return e
			}
		} else {
			res, e := tx.ExecContext(ctx, `INSERT INTO peer_grants
    (project,source_uuid,source_incarnation,target_uuid,target_incarnation,
    desired,status,source_ipv4,target_ipv4,source_ipv6,target_ipv6,operation_id)
    VALUES(?,?,?,?,?,1,'PENDING',?,?,?,?,?)`,
				i.Project, src.UUID, src.Incarnation, dst.UUID, dst.Incarnation,
				src.IPv4, dst.IPv4, src.IPv6, dst.IPv6, id)
			if e != nil {
				return ErrResourceLocked
			}
			affected, e = res.RowsAffected()
			if e != nil {
				return e
			}
		}
	} else {
		if !found {
			return ErrResourceLocked
		}
		res, e := tx.ExecContext(ctx, `UPDATE peer_grants SET desired=0,status='PENDING',
   operation_id=? WHERE project=? AND source_uuid=? AND target_uuid=?
   AND source_incarnation=? AND target_incarnation=? AND desired=1
   AND status='CONFIRMED' AND operation_id='' AND source_ipv4=? AND target_ipv4=?
   AND source_ipv6=? AND target_ipv6=?`,
			id, i.Project, src.UUID, dst.UUID, src.Incarnation, dst.Incarnation,
			src.IPv4, dst.IPv4, src.IPv6, dst.IPv6)
		if e != nil {
			return e
		}
		affected, e = res.RowsAffected()
		if e != nil {
			return e
		}
	}
	if affected != 1 {
		return ErrResourceLocked
	}
	if i.Allow && found {
		if _, e = tx.ExecContext(ctx, `INSERT INTO operation_events
		 (operation_id,at,state,safe_detail)
		 VALUES(?,?,'ADMITTED','SCOPED_PEER_V2_REGRANT_FROM_REVOKED')`,
			id, time.Now().UnixNano()); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO operation_events(operation_id,at,state,safe_detail)
  VALUES(?,?,'ADMITTED','SCOPED_PEER_V2_STAGED_ONLY_NO_NATIVE_EFFECT')`,
		id, time.Now().UnixNano()); e != nil {
		return e
	}
	return tx.Commit()
}
