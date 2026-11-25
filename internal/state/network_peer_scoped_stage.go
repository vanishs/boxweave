package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/vanishs/boxweave/internal/networklimits"
)

// StageScopedPeerOriginal stages exactly one v2 grant/revoke in the SAME
// SQLite write transaction that audits every owned Box, deterministic ACL
// and dual-stack lease, native grant provenance, actor and resource claims.
// It performs NO Incus/nft/conntrack/boot mutation. The installed worker
// uses StageAndClaimScopedPeerOriginal before calling Root.
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
	if err = stageScopedPeerOriginalTx(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// StageAndClaimScopedPeerOriginal closes the crash window between the
// pending policy row and dispatch claim. No native effect is sent here.
func (s *Store) StageAndClaimScopedPeerOriginal(ctx context.Context, id string) error {
	if s == nil || s.db == nil || s.auditReadOnly || !validHexID(id) {
		return ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = stageScopedPeerOriginalTx(ctx, tx, id); e != nil {
		return e
	}
	now := time.Now().UnixNano()
	r, e := tx.ExecContext(ctx, "UPDATE operations SET state='DISPATCHING',updated_at=? WHERE id=? AND state='ADMITTED' AND action='box_peer_set'", now, id)
	if e != nil {
		return e
	}
	if n, e := r.RowsAffected(); e != nil || n != 1 {
		return ErrState
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'DISPATCHING','SCOPED_PEER_V2_CLAIMED')", id, now); e != nil {
		return e
	}
	return tx.Commit()
}

var ErrScopedPeerHistoryFull = errors.New("scoped Peer policy history capacity reached")
var ErrScopedPeerDirectionFull = fmt.Errorf("scoped Peer retained direction capacity reached: %w", ErrResourceLocked)

func stageScopedPeerOriginalTx(ctx context.Context, tx *sql.Tx, id string) error {
	var err error
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
	var used, activeCount int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events e JOIN operations o ON o.id=e.operation_id
 WHERE o.layer=? AND o.action='box_peer_set' AND o.state='SUCCEEDED' AND e.state='SUCCEEDED'
 AND e.safe_detail IN ('SCOPED_PEER_V2_GRANT_KERNEL_VERIFIED','SCOPED_PEER_V2_REVOCATION_KERNEL_VERIFIED')`, i.Project).Scan(&used); e != nil {
		return e
	}
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM peer_grants WHERE project=? AND status='CONFIRMED'", i.Project).Scan(&activeCount); e != nil {
		return e
	}
	if !networklimits.ScopedPeerHistoryCanAppend(used, activeCount, i.Allow) {
		return ErrScopedPeerHistoryFull
	}
	inventory, e := readCompleteScopedNetworkAuditLedgerTx(ctx, tx, i.Project, i.Bridge)
	if e != nil || len(inventory.Boxes) < 2 {
		return ErrResourceLocked
	}
	owners := map[string]ScopedNetworkAuditBinding{}
	for _, b := range inventory.Boxes {
		owners[b.UUID] = b
	}
	retiredOwners := map[string]bool{}
	for _, b := range inventory.Retired {
		owners[b.UUID] = b.ScopedNetworkAuditBinding
		retiredOwners[b.UUID] = true
	}

	src, srcOK := owners[i.SourceUUID]
	dst, dstOK := owners[i.TargetUUID]
	if !srcOK || !dstOK || retiredOwners[i.SourceUUID] || retiredOwners[i.TargetUUID] || src.Incarnation != i.SourceIncarnation ||
		dst.Incarnation != i.TargetIncarnation {
		return ErrResourceLocked
	}
	expectedClaims, e := scopedPeerResourceClaims(i)
	if e != nil {
		return ErrResourceLocked
	}
	rows, e := tx.QueryContext(ctx, `SELECT resource_key,layer FROM operation_resource_claims
  WHERE operation_id=? ORDER BY resource_key`, id)
	if e != nil {
		return e
	}
	var claims []string
	for rows.Next() {
		var k, l string
		if e = rows.Scan(&k, &l); e != nil {
			break
		}
		if l != i.Project {
			e = ErrResourceLocked
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
		if total > networklimits.ScopedDirectionLimit || !okA || !okB || p.SourceUUID == p.TargetUUID || (retiredOwners[p.SourceUUID] || retiredOwners[p.TargetUUID]) && (desired != 0 || state != "REVOKED") ||
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
	if i.Allow && !found && total >= networklimits.ScopedDirectionLimit {
		return ErrScopedPeerDirectionFull
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
	return nil
}
