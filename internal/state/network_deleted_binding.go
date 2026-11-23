package state

import (
	"context"
	"database/sql"
)

// An exact SUCCEEDED native delete tombstone can anchor historical DROP
// rules; a RELEASED bit alone cannot. Unarchived address claims remain non-reassignable.
// Validate all retained rows in one set-based snapshot, with no lifetime
// delete-count ceiling. Released IPv4 fences are unique on the bridge, so
// valid retained bindings are naturally bounded by the supported IPv4 pool.
// A mismatch must fail closed, never silently omit a tombstone.
func verifiedDeletedNetworkBindings(ctx context.Context, tx *sql.Tx, project, bridge, acl string) (map[string]NetworkAuditBinding, error) {
	if e := verifyIPv4RetirementHistory(ctx, tx, project, bridge, acl); e != nil {
		return nil, e
	}
	var total int
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM network_bindings n WHERE project=? AND state='RELEASED' AND NOT EXISTS (SELECT 1 FROM network_ipv4_retirements r WHERE r.project=n.project AND r.box_uuid=n.box_uuid AND r.incarnation=n.incarnation)", project).Scan(&total); e != nil {
		return nil, e
	}
	// Aggregate dependencies/fence counts once. In particular, do not scan all
	// forward claims again for each historical Box. The two exact family
	// joins use the bridge/family/address primary key on the fence table.
	rows, e := tx.QueryContext(ctx, `WITH
 fence_counts AS (
   SELECT box_uuid,incarnation,COUNT(*) AS n
   FROM network_address_fences WHERE project=? GROUP BY box_uuid,incarnation
 ),
 blocked AS (
   SELECT source_uuid AS uuid FROM peer_grants WHERE project=? AND status!='REVOKED'
   UNION SELECT target_uuid FROM peer_grants WHERE project=? AND status!='REVOKED'
   UNION SELECT box_uuid FROM forward_claims WHERE project=? AND state!='REVOKED'
   UNION SELECT box_uuid FROM network_bindings WHERE project=? AND state!='RELEASED'
   UNION SELECT box_uuid FROM network_address_fences WHERE project=? AND state!='RELEASED'
 )
 SELECT n.box_uuid,n.incarnation,n.mac,n.ipv4,n.ipv6,n.nic,n.bridge,n.acl,b.name,b.deleted_by_operation_id
 FROM network_bindings n
 JOIN owned_boxes b ON b.project=n.project AND b.uuid=n.box_uuid AND b.incarnation=n.incarnation
 JOIN operations o ON o.id=b.deleted_by_operation_id AND o.layer=b.project AND o.box_uuid=b.uuid
   AND o.incarnation=b.incarnation AND o.action='box_delete' AND o.state='SUCCEEDED' AND o.native_operation_id!=''
 JOIN fence_counts fc ON fc.box_uuid=n.box_uuid AND fc.incarnation=n.incarnation AND fc.n=2
 JOIN network_address_fences f4 ON f4.bridge=n.bridge AND f4.family='ipv4' AND f4.address=n.ipv4
   AND f4.project=n.project AND f4.box_uuid=n.box_uuid AND f4.incarnation=n.incarnation
   AND f4.state='RELEASED' AND f4.operation_id=b.deleted_by_operation_id
 JOIN network_address_fences f6 ON f6.bridge=n.bridge AND f6.family='ipv6' AND f6.address=n.ipv6
   AND f6.project=n.project AND f6.box_uuid=n.box_uuid AND f6.incarnation=n.incarnation
   AND f6.state='RELEASED' AND f6.operation_id=b.deleted_by_operation_id
 LEFT JOIN blocked d ON d.uuid=n.box_uuid
 WHERE n.project=? AND n.state='RELEASED' AND b.state='DELETED' AND d.uuid IS NULL AND NOT EXISTS (SELECT 1 FROM network_ipv4_retirements r WHERE r.project=n.project AND r.box_uuid=n.box_uuid AND r.incarnation=n.incarnation)`,
		project, project, project, project, project, project, project)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := map[string]NetworkAuditBinding{}
	for rows.Next() {
		var b NetworkAuditBinding
		var nic, nw, ac, del string
		if e = rows.Scan(&b.UUID, &b.Incarnation, &b.MAC, &b.IPv4, &b.IPv6, &nic, &nw, &ac, &b.Name, &del); e != nil {
			return nil, e
		}
		if nic != "eth0" || nw != bridge || ac != acl || !auditMAC(b.MAC) || !auditIP(b.IPv4, false) || !auditIP(b.IPv6, true) || !auditName(b.Name) || !validHexID(del) {
			return nil, ErrResourceLocked
		}
		if _, dup := out[b.UUID]; dup {
			return nil, ErrResourceLocked
		}
		out[b.UUID] = b
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	if e = rows.Close(); e != nil {
		return nil, e
	}
	if total != len(out) {
		return nil, ErrResourceLocked
	}
	return out, nil
}
