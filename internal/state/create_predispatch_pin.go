package state

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// A FAILED local pre-dispatch Create has a proven zero native dispatch.
// Release ONLY its exact four prospective static address reservations in the
// SAME transaction as the terminal FAILED receipt. Any malformed, partial,
// already native-bound or independently-owned claim remains locked.
func releasePreDispatchCreatePinsTx(ctx context.Context, tx *sql.Tx, id string) error {
	var action, layer, resource, uuid, inc, nativeID string
	if e := tx.QueryRowContext(ctx, "SELECT action,layer,resource_key,box_uuid,incarnation,native_operation_id FROM operations WHERE id=?", id).Scan(&action, &layer, &resource, &uuid, &inc, &nativeID); e != nil {
		return e
	}
	if action != "box_create" {
		return nil
	}
	if uuid != "" || inc != "" || nativeID != "" || !strings.HasPrefix(resource, "box:") {
		return ErrResourceLocked
	}
	rows, e := tx.QueryContext(ctx, "SELECT resource_key FROM operation_resource_claims WHERE operation_id=? AND layer=? AND resource_key LIKE 'create-pin:%'", id, layer)
	if e != nil {
		return e
	}
	var keys []string
	for rows.Next() {
		var key string
		if e = rows.Scan(&key); e != nil {
			break
		}
		keys = append(keys, key)
	}
	if e == nil {
		e = rows.Err()
	}
	if closeErr := rows.Close(); e == nil {
		e = closeErr
	}
	if e != nil {
		return e
	}
	if len(keys) == 0 {
		return nil
	}
	if len(keys) != 4 {
		return ErrResourceLocked
	}
	pin := VerifiedCreateBinding{}
	seen := map[string]bool{}
	for _, key := range keys {
		p := strings.SplitN(key, ":", 4)
		if len(p) != 4 || p[0] != "create-pin" || p[3] == "" || seen[p[2]] {
			return ErrResourceLocked
		}
		seen[p[2]] = true
		if pin.Bridge != "" && pin.Bridge != p[1] {
			return ErrResourceLocked
		}
		pin.Bridge = p[1]
		switch p[2] {
		case "v4":
			pin.IPv4 = p[3]
		case "v6":
			pin.IPv6 = p[3]
		case "mac":
			pin.MAC = p[3]
		case "acl":
			pin.ACL = p[3]
		default:
			return ErrResourceLocked
		}
	}
	if !pin.valid() {
		return ErrResourceLocked
	}
	wanted := map[string]bool{}
	for _, k := range pin.reservationKeys() {
		wanted[k] = true
	}
	for _, k := range keys {
		if !wanted[k] {
			return ErrResourceLocked
		}
	}
	var n int
	e = tx.QueryRowContext(ctx, `SELECT
  (SELECT COUNT(*) FROM owned_boxes WHERE project=? AND 'box:'||name=? AND state='OWNED')+
  (SELECT COUNT(*) FROM network_address_fences WHERE operation_id=?)`, layer, resource, id).Scan(&n)
	if e != nil {
		return e
	}
	if n != 0 {
		return ErrResourceLocked
	}
	r, e := tx.ExecContext(ctx, "DELETE FROM operation_resource_claims WHERE operation_id=? AND layer=? AND resource_key LIKE 'create-pin:%'", id, layer)
	if e != nil {
		return e
	}
	if released, re := r.RowsAffected(); re != nil || released != 4 {
		return ErrResourceLocked
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'FAILED','PREDISPATCH_STATIC_RESERVATIONS_RELEASED_NO_NATIVE')", id, time.Now().UnixNano())
	return e
}
