package state

import (
	"context"
	"database/sql"
	"reflect"
	"slices"
	"strings"
)

// ReadPeerRevocationSnapshot returns the entire exact allowlist before and
// after ONE already-claimed directional revoke. Its pending row contributes to
// before only: it must not remove any independently confirmed peer direction.
// It does no native mutation and cannot authorize an UNKNOWN replay.
func (s *Store) ReadPeerRevocationSnapshot(ctx context.Context, project, bridge, acl, original string) ([]PeerGrantBinding, []PeerGrantBinding, error) {
	if s == nil || s.db == nil || !validHexID(original) {
		return nil, nil, ErrInvalid
	}
	proof, err := s.ReadRootPeerRevocation(ctx, original, project, bridge, acl)
	if err != nil {
		return nil, nil, err
	}
	ledger, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil {
		return nil, nil, ErrResourceLocked
	}
	owners := make(map[string]NetworkAuditBinding, len(ledger.Boxes))
	for _, b := range ledger.Boxes {
		owners[b.UUID] = b
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT source_uuid,source_incarnation,target_uuid,target_incarnation,"+
		"source_ipv4,target_ipv4,source_ipv6,target_ipv6,desired,status,operation_id "+
		"FROM peer_grants WHERE project=? ORDER BY source_uuid,target_uuid", project)
	if err != nil {
		return nil, nil, err
	}
	var before, after []PeerGrantBinding
	var pending int
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
		src, ok1 := owners[p.SourceUUID]
		dst, ok2 := owners[p.TargetUUID]
		if len(before) >= 32 || !ok1 || !ok2 || p.SourceUUID == p.TargetUUID ||
			p.SourceIncarnation != src.Incarnation || p.TargetIncarnation != dst.Incarnation ||
			p.SourceIPv4 != src.IPv4 || p.TargetIPv4 != dst.IPv4 ||
			p.SourceIPv6 != src.IPv6 || p.TargetIPv6 != dst.IPv6 {
			err = ErrResourceLocked
			break
		}
		switch {
		case status == "CONFIRMED" && desired == 1:
			before = append(before, p)
			after = append(after, p)
		case status == "PENDING" && desired == 0 && operationID == original &&
			p.SourceUUID == proof.Intent.SourceUUID && p.TargetUUID == proof.Intent.TargetUUID &&
			pending == 0:
			pending++
			before = append(before, p)
		default:
			err = ErrResourceLocked
		}
		if err != nil {
			break
		}
	}
	if e := rows.Err(); err == nil {
		err = e
	}
	if e := rows.Close(); err == nil {
		err = e
	}
	if err != nil {
		return nil, nil, err
	}
	if pending != 1 {
		return nil, nil, ErrResourceLocked
	}
	sortBindings := func(items []PeerGrantBinding) {
		slices.SortFunc(items, func(a, b PeerGrantBinding) int {
			return strings.Compare(a.SourceUUID+"/"+a.TargetUUID, b.SourceUUID+"/"+b.TargetUUID)
		})
	}
	sortBindings(before)
	sortBindings(after)
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	again, err := s.ReadRootPeerRevocation(ctx, original, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(proof, again) {
		return nil, nil, ErrResourceLocked
	}
	return before, after, nil
}
