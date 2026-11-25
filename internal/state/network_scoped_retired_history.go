package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"slices"

	"github.com/vanishs/boxweave/internal/networklimits"
	"github.com/vanishs/boxweave/internal/peerscope"
)

// Released is a historical address hold, NOT an available address.
// Root must separately verify both original journal receipts and absence
// of the retired native UUID and ACL before enabling Peer writes.
type ScopedNetworkAuditRetiredBinding struct {
	ScopedNetworkAuditBinding
	CreateID, DeleteID string
}

const scopedRetiredHistoryLimit = networklimits.ScopedRetiredIdentityLimit

func scopedRetiredOwners(ctx context.Context, tx *sql.Tx, project, bridge string) (map[string]ScopedNetworkAuditRetiredBinding, error) {
	out := map[string]ScopedNetworkAuditRetiredBinding{}
	rows, e := tx.QueryContext(ctx, `SELECT uuid,name,incarnation,COALESCE(deleted_by_operation_id,'') FROM owned_boxes WHERE project=? AND state='DELETED' ORDER BY uuid LIMIT ?`, project, scopedRetiredHistoryLimit+1)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var h ScopedNetworkAuditRetiredBinding
		if e = rows.Scan(&h.UUID, &h.Name, &h.Incarnation, &h.DeleteID); e != nil {
			return nil, e
		}
		acl, e := peerscope.ACLName(project, bridge, h.UUID)
		if e != nil || h.UUID != h.Incarnation || !auditName(h.Name) || !validHexID(h.DeleteID) || len(out) >= scopedRetiredHistoryLimit {
			return nil, ErrResourceLocked
		}
		h.ACL = acl
		out[h.UUID] = h
	}
	return out, rows.Err()
}
func scopedRetiredOriginals(ctx context.Context, tx *sql.Tx, project, bridge string, h ScopedNetworkAuditRetiredBinding) (ScopedNetworkAuditRetiredBinding, error) {
	var zero ScopedNetworkAuditRetiredBinding
	var raw []byte
	var resource, actor, token, nativeID string
	err := tx.QueryRowContext(ctx, `SELECT id,safe_intent,resource_key,actor_id,token_id,native_operation_id FROM operations
 WHERE layer=? AND box_uuid=? AND incarnation=? AND action='box_create' AND state='SUCCEEDED'`, project, h.UUID, h.Incarnation).Scan(&h.CreateID, &raw, &resource, &actor, &token, &nativeID)
	if err != nil || !validHexID(h.CreateID) || actor == "" || token == "" || nativeID == "" || resource != "box:"+h.Name {
		return zero, ErrResourceLocked
	}
	var intent ScopedCreateIntent
	if json.Unmarshal(raw, &intent) != nil {
		return zero, ErrResourceLocked
	}
	canonical, e := json.Marshal(intent)
	if e != nil || !bytes.Equal(canonical, raw) || intent.Schema != 2 || intent.Project != project || intent.Bridge != bridge || intent.Name != h.Name {
		return zero, ErrResourceLocked
	}
	var n int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE layer=? AND box_uuid=? AND incarnation=? AND action='box_create'`, project, h.UUID, h.Incarnation).Scan(&n); e != nil || n != 1 {
		return zero, ErrResourceLocked
	}
	b := VerifiedCreateBinding{bridge, h.ACL, h.MAC, h.IPv4, h.IPv6}
	if !b.valid() {
		return zero, ErrResourceLocked
	}
	keys := append([]string{"box:" + h.Name}, b.reservationKeys()...)
	if e = scopedRetiredClaims(ctx, tx, h.CreateID, project, keys, "ORIGINAL_CREATE_UUID_ATOMIC"); e != nil {
		return zero, e
	}
	err = tx.QueryRowContext(ctx, `SELECT safe_intent,resource_key,actor_id,token_id,native_operation_id FROM operations
 WHERE id=? AND layer=? AND box_uuid=? AND incarnation=? AND action='box_delete' AND state='SUCCEEDED'`, h.DeleteID, project, h.UUID, h.Incarnation).Scan(&raw, &resource, &actor, &token, &nativeID)
	want, _ := json.Marshal(struct {
		BoxID string `json:"box_id"`
	}{h.UUID})
	if err != nil || !bytes.Equal(raw, want) || actor == "" || token == "" || nativeID == "" || resource != "box:"+h.Name {
		return zero, ErrResourceLocked
	}
	if e = scopedRetiredClaims(ctx, tx, h.DeleteID, project, []string{"box:" + h.Name, deleteProjectKey(project)}, "SCOPED_DELETE_ACL_ABSENT_ATOMIC"); e != nil {
		return zero, e
	}
	return h, nil
}
func scopedRetiredClaims(ctx context.Context, tx *sql.Tx, id, project string, expected []string, event string) error {
	rows, e := tx.QueryContext(ctx, "SELECT resource_key,layer FROM operation_resource_claims WHERE operation_id=? ORDER BY resource_key", id)
	if e != nil {
		return e
	}
	var keys []string
	for rows.Next() {
		var key, p string
		if e = rows.Scan(&key, &p); e != nil || p != project {
			rows.Close()
			return ErrResourceLocked
		}
		keys = append(keys, key)
	}
	e = rows.Err()
	ce := rows.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	slices.Sort(expected)
	if !slices.Equal(keys, expected) {
		return ErrResourceLocked
	}
	var n int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED' AND safe_detail=?", id, event).Scan(&n); e != nil || n != 1 {
		return ErrResourceLocked
	}
	return nil
}
