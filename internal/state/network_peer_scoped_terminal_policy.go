package state

import (
	"context"
	"database/sql"
)

// ScopedTerminalPeerPolicy is the durable terminal active and revoked
// policy, including retained historical addresses. Native Root must still
// independently check every current ACL and the published Guard.
type ScopedTerminalPeerPolicy struct {
	Active, Revoked []PeerGrantBinding
}

func (s *Store) ReadScopedTerminalPeerPolicy(ctx context.Context, project, bridge string) (ScopedTerminalPeerPolicy, error) {
	var zero ScopedTerminalPeerPolicy
	if s == nil || s.db == nil {
		return zero, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return zero, e
	}
	defer tx.Rollback()
	p, e := readScopedTerminalPeerPolicyTx(ctx, tx, project, bridge)
	if e != nil {
		return zero, e
	}
	return p, tx.Commit()
}
func readScopedTerminalPeerPolicyTx(ctx context.Context, tx *sql.Tx, project, bridge string) (ScopedTerminalPeerPolicy, error) {
	var zero ScopedTerminalPeerPolicy
	inventory, e := readCompleteScopedNetworkAuditLedgerModeTx(ctx, tx, project, bridge, true)
	if e != nil {
		return zero, e
	}
	owners := map[string]ScopedNetworkAuditBinding{}
	retired := map[string]bool{}
	for _, b := range inventory.Boxes {
		owners[b.UUID] = b
	}
	for _, b := range inventory.Retired {
		owners[b.UUID] = b.ScopedNetworkAuditBinding
		retired[b.UUID] = true
	}
	rows, e := tx.QueryContext(ctx, `SELECT source_uuid,source_incarnation,target_uuid,target_incarnation,source_ipv4,target_ipv4,source_ipv6,target_ipv6,
 desired,status,operation_id,COALESCE(last_verified_operation_id,'') FROM peer_grants WHERE project=? ORDER BY source_uuid,target_uuid`, project)
	if e != nil {
		return zero, e
	}
	defer rows.Close()
	var out ScopedTerminalPeerPolicy
	for rows.Next() {
		var p PeerGrantBinding
		var desired int
		var st, op, verified string
		if e = rows.Scan(&p.SourceUUID, &p.SourceIncarnation, &p.TargetUUID, &p.TargetIncarnation, &p.SourceIPv4, &p.TargetIPv4, &p.SourceIPv6, &p.TargetIPv6, &desired, &st, &op, &verified); e != nil {
			return zero, e
		}
		a, okA := owners[p.SourceUUID]
		b, okB := owners[p.TargetUUID]
		if !okA || !okB || p.SourceUUID == p.TargetUUID || p.SourceIncarnation != a.Incarnation || p.TargetIncarnation != b.Incarnation || p.SourceIPv4 != a.IPv4 || p.TargetIPv4 != b.IPv4 || p.SourceIPv6 != a.IPv6 || p.TargetIPv6 != b.IPv6 || len(out.Active)+len(out.Revoked) >= 32 {
			return zero, ErrResourceLocked
		}
		if e = scopedPeerVerifyTerminalLineageTx(ctx, tx, project, bridge, p, desired, st, op, verified); e != nil {
			return zero, e
		}
		switch st {
		case "CONFIRMED":
			if retired[p.SourceUUID] || retired[p.TargetUUID] {
				return zero, ErrResourceLocked
			}
			out.Active = append(out.Active, p)
		case "REVOKED":
			out.Revoked = append(out.Revoked, p)
		default:
			return zero, ErrResourceLocked
		}
	}
	if e = rows.Err(); e != nil {
		return zero, e
	}
	return out, nil
}
