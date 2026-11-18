package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"strings"
)

// PeerGrantBinding is a strict, read-only compiler input from durable SQLite.
// It is NOT a native permission or authorization to update Incus ACLs.
type PeerGrantBinding struct {
	SourceUUID, SourceIncarnation, TargetUUID, TargetIncarnation string
	SourceIPv4, TargetIPv4, SourceIPv6, TargetIPv6               string
}

// ReadPeerGrantSnapshot resolves all confirmed directional peers plus at most
// one exact staged original. Unknown/stale peers fail closed rather than being
// omitted from an ACL rebuild. Root must still recheck actual host NIC/ACL
// ownership and the kernel effect before acknowledging this plan.
func (s *Store) ReadPeerGrantSnapshot(ctx context.Context, project, bridge, acl, stagedOriginal string) ([]PeerGrantBinding, error) {
	if s == nil || s.db == nil || (stagedOriginal != "" && !validHexID(stagedOriginal)) {
		return nil, ErrInvalid
	}
	ledger, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil {
		return nil, ErrResourceLocked
	}
	owners := make(map[string]NetworkAuditBinding, len(ledger.Boxes))
	for _, b := range ledger.Boxes {
		owners[b.UUID] = b
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT source_uuid,source_incarnation,target_uuid,target_incarnation,
 source_ipv4,target_ipv4,source_ipv6,target_ipv6,desired,status,operation_id
 FROM peer_grants WHERE project=? ORDER BY source_uuid,target_uuid`, project)
	if err != nil {
		return nil, err
	}
	var out []PeerGrantBinding
	var staged int
	for rows.Next() {
		var p PeerGrantBinding
		var desired int
		var status, operationID string
		if err = rows.Scan(&p.SourceUUID, &p.SourceIncarnation, &p.TargetUUID, &p.TargetIncarnation,
			&p.SourceIPv4, &p.TargetIPv4, &p.SourceIPv6, &p.TargetIPv6, &desired, &status, &operationID); err != nil {
			break
		}
		if status == "REVOKED" && desired == 0 {
			continue
		}
		if len(out) >= 32 || p.SourceUUID == p.TargetUUID || p.SourceUUID == "" {
			err = ErrResourceLocked
			break
		}
		src, ok1 := owners[p.SourceUUID]
		dst, ok2 := owners[p.TargetUUID]
		if !ok1 || !ok2 || p.SourceIncarnation != src.Incarnation || p.TargetIncarnation != dst.Incarnation ||
			p.SourceIPv4 != src.IPv4 || p.TargetIPv4 != dst.IPv4 ||
			p.SourceIPv6 != src.IPv6 || p.TargetIPv6 != dst.IPv6 {
			err = ErrResourceLocked
			break
		}
		if status == "CONFIRMED" && desired == 1 {
			out = append(out, p)
			continue
		}
		if status != "PENDING" || desired != 1 || stagedOriginal == "" || operationID != stagedOriginal || staged > 0 {
			err = ErrResourceLocked
			break
		}
		staged++
		out = append(out, p)
	}
	if e := rows.Err(); err == nil {
		err = e
	}
	if e := rows.Close(); err == nil {
		err = e
	}
	if err != nil {
		return nil, err
	}
	if stagedOriginal != "" {
		if staged != 1 {
			return nil, ErrResourceLocked
		}
		var status, actor, token, rawLayer, srcUUID, srcInc string
		var raw []byte
		err = tx.QueryRowContext(ctx, `SELECT state,actor_id,token_id,layer,box_uuid,incarnation,safe_intent
   FROM operations WHERE id=? AND action='box_peer_set'`, stagedOriginal).
			Scan(&status, &actor, &token, &rawLayer, &srcUUID, &srcInc, &raw)
		if err != nil || (status != "ADMITTED" && status != "DISPATCHING") ||
			actor == "" || token == "" || rawLayer != project {
			return nil, ErrResourceLocked
		}
		var intent RootPeerRevokeSafeIntent
		if json.Unmarshal(raw, &intent) != nil {
			return nil, ErrResourceLocked
		}
		canonical, e := json.Marshal(intent)
		if e != nil || !bytes.Equal(raw, canonical) || !intent.Allow || intent.Project != project ||
			intent.Bridge != bridge || intent.ACL != acl || srcUUID != intent.SourceUUID ||
			srcInc != intent.SourceIncarnation {
			return nil, ErrResourceLocked
		}
		found := false
		for _, p := range out {
			if p.SourceUUID == intent.SourceUUID && p.TargetUUID == intent.TargetUUID &&
				p.SourceIncarnation == intent.SourceIncarnation && p.TargetIncarnation == intent.TargetIncarnation {
				found = true
				break
			}
		}
		if !found {
			return nil, ErrResourceLocked
		}
		claimsRows, e := tx.QueryContext(ctx, "SELECT resource_key FROM operation_resource_claims WHERE operation_id=? ORDER BY resource_key", stagedOriginal)
		if e != nil {
			return nil, e
		}
		var claims []string
		for claimsRows.Next() {
			var key string
			if e = claimsRows.Scan(&key); e != nil {
				claimsRows.Close()
				return nil, e
			}
			claims = append(claims, key)
		}
		if e = claimsRows.Err(); e != nil {
			claimsRows.Close()
			return nil, e
		}
		if e = claimsRows.Close(); e != nil {
			return nil, e
		}
		expected := []string{"acl:default/" + acl, "box:" + intent.SourceUUID, "box:" + intent.TargetUUID}
		slices.Sort(expected)
		if !slices.Equal(claims, expected) {
			return nil, ErrResourceLocked
		}
		var role string
		if tx.QueryRowContext(ctx, "SELECT role FROM credentials WHERE id=?", token).Scan(&role) != nil || role != "admin" {
			return nil, ErrRevoked
		}
		active, e := activeCredentialForActor(ctx, tx, actor, token)
		if e != nil || !active {
			return nil, ErrRevoked
		}
		// A grant snapshot must not be compiled if the owned Project has another
		// conflicting in-flight action, even one admitted in a legacy database.
		var busy int
		if tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations o WHERE o.layer=? AND o.id<>?
   AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')
   AND (o.action='box_peer_set' OR NOT EXISTS (SELECT 1 FROM recovery_audit r WHERE r.operation_id=o.id AND r.decision='RELEASED'))`,
			project, stagedOriginal).Scan(&busy) != nil || busy != 0 {
			return nil, ErrResourceLocked
		}
	}
	slices.SortFunc(out, func(a, b PeerGrantBinding) int {
		return strings.Compare(a.SourceUUID+"/"+a.TargetUUID, b.SourceUUID+"/"+b.TargetUUID)
	})
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}
