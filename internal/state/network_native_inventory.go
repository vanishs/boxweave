package state

import (
	"context"
	"database/sql"
	"github.com/vanishs/boxweave/internal/networklimits"
)

// NativeNetworkBox is a read-only, project-scoped SQLite observation.
// Inconsistent rows never authorize native mutation or adoption.
type NativeNetworkBox struct {
	UUID, Incarnation, Name, OriginalID, OriginalState, OriginalResource string
	OriginalCount, BindingCount                                          int
	OriginalProof                                                        bool
	NIC, Bridge, ACL, MAC, IPv4, IPv6, BindingState                      string
	FenceCount, FenceIPv4, FenceIPv6                                     int
}
type NativeNetworkLedger struct {
	Project                                 string
	Boxes                                   []NativeNetworkBox
	OrphanBindings, OrphanFences, Unsettled int
}

// ReadNativeNetworkLedger reads current owned Boxes, exact Create lineages,
// bindings and fences in one consistent bounded read-only transaction.
func (s *Store) ReadNativeNetworkLedger(ctx context.Context, project string) (NativeNetworkLedger, error) {
	var out NativeNetworkLedger
	if !auditName(project) || project == "default" {
		return out, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	out.Project = project
	rows, e := tx.QueryContext(ctx, `SELECT uuid,incarnation,name FROM owned_boxes
 WHERE project=? AND state='OWNED' ORDER BY uuid LIMIT ?`, project, networklimits.MaxProjectBoxes+1)
	if e != nil {
		return NativeNetworkLedger{}, e
	}
	for rows.Next() {
		var v NativeNetworkBox
		if e = rows.Scan(&v.UUID, &v.Incarnation, &v.Name); e != nil {
			break
		}
		if len(out.Boxes) >= networklimits.MaxProjectBoxes {
			e = ErrResourceLocked
			break
		}
		out.Boxes = append(out.Boxes, v)
	}
	if e == nil {
		e = rows.Err()
	}
	if ce := rows.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return NativeNetworkLedger{}, e
	}
	for i := range out.Boxes {
		v := &out.Boxes[i]
		rows, e = tx.QueryContext(ctx, `SELECT id,state,resource_key FROM operations
   WHERE layer=? AND action='box_create' AND box_uuid=? AND incarnation=? LIMIT 2`, project, v.UUID, v.Incarnation)
		if e != nil {
			return NativeNetworkLedger{}, e
		}
		for rows.Next() {
			var id, st, res string
			if e = rows.Scan(&id, &st, &res); e != nil {
				break
			}
			v.OriginalCount++
			if v.OriginalCount == 1 {
				v.OriginalID, v.OriginalState, v.OriginalResource = id, st, res
			}
		}
		if e == nil {
			e = rows.Err()
		}
		if ce := rows.Close(); e == nil {
			e = ce
		}
		if e != nil {
			return NativeNetworkLedger{}, e
		}
		rows, e = tx.QueryContext(ctx, `SELECT nic,bridge,acl,mac,ipv4,ipv6,state
    FROM network_bindings WHERE project=? AND box_uuid=? AND state!='RELEASED' LIMIT 2`, project, v.UUID)
		if e != nil {
			return NativeNetworkLedger{}, e
		}
		for rows.Next() {
			var nic, bridge, acl, mac, ipv4, ipv6, st string
			if e = rows.Scan(&nic, &bridge, &acl, &mac, &ipv4, &ipv6, &st); e != nil {
				break
			}
			v.BindingCount++
			if v.BindingCount == 1 {
				v.NIC, v.Bridge, v.ACL, v.MAC, v.IPv4, v.IPv6, v.BindingState = nic, bridge, acl, mac, ipv4, ipv6, st
			}
		}
		if e == nil {
			e = rows.Err()
		}
		if ce := rows.Close(); e == nil {
			e = ce
		}
		if e != nil {
			return NativeNetworkLedger{}, e
		}
		rows, e = tx.QueryContext(ctx, `SELECT family,address,bridge,state,operation_id FROM network_address_fences
   WHERE project=? AND box_uuid=? AND state!='RELEASED' LIMIT 4`, project, v.UUID)
		if e != nil {
			return NativeNetworkLedger{}, e
		}
		for rows.Next() {
			var family, address, bridge, st, op string
			if e = rows.Scan(&family, &address, &bridge, &st, &op); e != nil {
				break
			}
			v.FenceCount++
			if st == "RESERVED" && op == v.OriginalID && bridge == v.Bridge &&
				((family == "ipv4" && address == v.IPv4) || (family == "ipv6" && address == v.IPv6)) {
				if family == "ipv4" {
					v.FenceIPv4++
				} else {
					v.FenceIPv6++
				}
			}
		}
		if e == nil {
			e = rows.Err()
		}
		if ce := rows.Close(); e == nil {
			e = ce
		}
		if e != nil {
			return NativeNetworkLedger{}, e
		}
		// Never claim a healthy original solely from a mutable SUCCEEDED bit.
		if v.OriginalCount == 1 && v.OriginalState == "SUCCEEDED" && v.BindingCount == 1 {
			var valid int
			e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations c WHERE c.id=? AND c.layer=? AND c.action='box_create'
			AND c.state='SUCCEEDED' AND EXISTS(SELECT 1 FROM operation_events ev WHERE ev.operation_id=c.id AND ev.state='SUCCEEDED')
			AND `+retirementCreateEvidence, v.OriginalID, project).Scan(&valid)
			if e != nil {
				return NativeNetworkLedger{}, e
			}
			binding := VerifiedCreateBinding{v.Bridge, v.ACL, v.MAC, v.IPv4, v.IPv6}
			v.OriginalProof = valid == 1 && requireCreateBindingReservation(ctx, tx, v.OriginalID, project, binding) == nil
		}
	}
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_bindings n WHERE n.project=? AND n.state!='RELEASED' AND
 NOT EXISTS(SELECT 1 FROM owned_boxes b WHERE b.project=n.project AND b.uuid=n.box_uuid AND
 b.incarnation=n.incarnation AND b.state='OWNED')`, project).Scan(&out.OrphanBindings)
	if e != nil {
		return NativeNetworkLedger{}, e
	}
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_address_fences f WHERE f.project=? AND f.state!='RELEASED' AND
 NOT EXISTS(SELECT 1 FROM owned_boxes b WHERE b.project=f.project AND b.uuid=f.box_uuid AND
 f.incarnation=b.incarnation AND b.state='OWNED')`, project).Scan(&out.OrphanFences)
	if e != nil {
		return NativeNetworkLedger{}, e
	}
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE layer=? AND
 state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN','OBSERVING')`, project).Scan(&out.Unsettled)
	if e != nil {
		return NativeNetworkLedger{}, e
	}
	if e = tx.Commit(); e != nil {
		return NativeNetworkLedger{}, e
	}
	return out, nil
}
