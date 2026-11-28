package state

import (
	"context"
	"database/sql"
)

// Retain the old exact-forward history oracle without the retired IPv4
// allocation tables. A revoked bit alone is not proof of settlement.
// The caller separately verifies complete native retirement.
const scopedRetiredForwardInvalid = `(
 f.incarnation!=b.incarnation OR f.state!='REVOKED' OR f.guest_ip!='127.0.0.1'
 OR length(f.forward_id)!=32 OR f.forward_id GLOB '*[^0-9a-f]*'
 OR f.native_device!='bwpf-'||substr(f.forward_id,1,16)
 OR NOT EXISTS (SELECT 1 FROM operations c WHERE c.id=f.forward_id AND c.layer=f.project AND c.box_uuid=f.box_uuid AND c.incarnation=f.incarnation
  AND c.action='port_forward_create' AND c.resource_key='box:'||b.name
  AND json_valid(c.safe_intent) AND json_extract(c.safe_intent,'$.box_id')=f.box_uuid
  AND json_extract(c.safe_intent,'$.protocol')=f.protocol AND json_extract(c.safe_intent,'$.host_bind')=f.host_bind
  AND json_extract(c.safe_intent,'$.host_port')=f.host_port AND json_extract(c.safe_intent,'$.guest_port')=f.guest_port
  AND ((c.state IN ('FAILED','CANCELLED') AND c.native_operation_id='' AND f.operation_id=c.id)
       OR (c.state='SUCCEEDED' AND c.native_operation_id!=''
          AND EXISTS (SELECT 1 FROM operations d WHERE d.id=f.operation_id AND d.layer=f.project AND d.box_uuid=f.box_uuid AND d.incarnation=f.incarnation
           AND d.action='port_forward_delete' AND d.state='SUCCEEDED' AND d.native_operation_id!=''
           AND d.resource_key='box:'||b.name AND json_valid(d.safe_intent) AND json_extract(d.safe_intent,'$.forward_id')=f.forward_id)))
 ))`

// Check all retained claims within the same read-only retirement snapshot.
// An orphan or foreign incarnation is never accepted as historical proof.
func checkScopedRetiredForwardHistory(ctx context.Context, tx *sql.Tx, project string) error {
	return checkScopedForwardHistory(ctx, tx, project, false)
}

// Exceptional archive observation also retains exact revoked Forward
// history of still-owned Boxes. The ordinary retired-only gate is unchanged.
func checkScopedOfflineForwardHistory(ctx context.Context, tx *sql.Tx, project string) error {
	return checkScopedForwardHistory(ctx, tx, project, true)
}

func checkScopedForwardHistory(ctx context.Context, tx *sql.Tx, project string, allowOwned bool) error {
	ownerState := "b.state!='DELETED'"
	if allowOwned {
		ownerState = "b.state NOT IN ('DELETED','OWNED')"
	}
	var n int
	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM forward_claims f
 LEFT JOIN owned_boxes b ON b.project=f.project AND b.uuid=f.box_uuid AND b.incarnation=f.incarnation
 WHERE f.project=? AND (b.uuid IS NULL OR `+ownerState+` OR `+scopedRetiredForwardInvalid+`)`, project).Scan(&n)
	if err != nil {
		return err
	}
	if n != 0 {
		return ErrResourceLocked
	}
	rows, err := tx.QueryContext(ctx, "SELECT "+forwardColumns+" FROM forward_claims WHERE project=?", project)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		f, e := scanForward(rows)
		if e != nil {
			return e
		}
		if !validHexID(f.OperationID) || !validForwardCreate(ForwardCreateAdmission{f.Protocol, f.HostBind, f.HostPort, f.GuestPort}) {
			return ErrResourceLocked
		}
	}
	return rows.Err()
}
