package state

import (
	"context"
	"database/sql"
)

// An exact SUCCEEDED native delete tombstone can anchor historical DROP
// rules; a RELEASED bit alone cannot. Address claims remain non-reassignable.
func verifiedDeletedNetworkBindings(ctx context.Context, tx *sql.Tx, project, bridge, acl string) (map[string]NetworkAuditBinding, error) {
	rows, e := tx.QueryContext(ctx, "SELECT n.box_uuid,n.incarnation,n.mac,n.ipv4,n.ipv6,n.nic,n.bridge,n.acl,b.name,b.deleted_by_operation_id FROM network_bindings n JOIN owned_boxes b ON b.project=n.project AND b.uuid=n.box_uuid AND b.incarnation=n.incarnation JOIN operations o ON o.id=b.deleted_by_operation_id AND o.layer=b.project AND o.box_uuid=b.uuid AND o.incarnation=b.incarnation AND o.action='box_delete' AND o.state='SUCCEEDED' AND o.native_operation_id!='' WHERE n.project=? AND n.state='RELEASED' AND b.state='DELETED'", project)
	if e != nil {
		return nil, e
	}
	type history struct {
		b   NetworkAuditBinding
		del string
	}
	var all []history
	for rows.Next() {
		var h history
		var nic, nw, ac string
		if e = rows.Scan(&h.b.UUID, &h.b.Incarnation, &h.b.MAC, &h.b.IPv4, &h.b.IPv6, &nic, &nw, &ac, &h.b.Name, &h.del); e != nil {
			rows.Close()
			return nil, e
		}
		if nic != "eth0" || nw != bridge || ac != acl || !auditMAC(h.b.MAC) || !auditIP(h.b.IPv4, false) || !auditIP(h.b.IPv6, true) || !auditName(h.b.Name) || !validHexID(h.del) {
			rows.Close()
			return nil, ErrResourceLocked
		}
		all = append(all, h)
		if len(all) > 256 {
			rows.Close()
			return nil, ErrResourceLocked
		}
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return nil, e
	}
	if e = rows.Close(); e != nil {
		return nil, e
	}
	out := map[string]NetworkAuditBinding{}
	for _, h := range all {
		b := h.b
		if _, dup := out[b.UUID]; dup {
			return nil, ErrResourceLocked
		}
		var total, matching int
		if e = tx.QueryRowContext(ctx, "SELECT (SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=? AND incarnation=?),(SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=? AND incarnation=? AND bridge=? AND operation_id=? AND state='RELEASED' AND ((family='ipv4' AND address=?) OR (family='ipv6' AND address=?)))", project, b.UUID, b.Incarnation, project, b.UUID, b.Incarnation, bridge, h.del, b.IPv4, b.IPv6).Scan(&total, &matching); e != nil || total != 2 || matching != 2 {
			return nil, ErrResourceLocked
		}
		peer, fwd, bind, fences, e := countDeleteDependencies(ctx, tx, project, b.UUID)
		if e != nil || peer+fwd+bind+fences != 0 {
			return nil, ErrResourceLocked
		}
		out[b.UUID] = b
	}
	var total int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM network_bindings WHERE project=? AND state='RELEASED'", project).Scan(&total); e != nil || total != len(out) {
		return nil, ErrResourceLocked
	}
	return out, nil
}
