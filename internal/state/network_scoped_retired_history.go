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
	if e = scopedRetiredClaims(ctx, tx, h.CreateID, project, keys, "ORIGINAL_CREATE_UUID_ATOMIC", "RECOVERY_SCOPED_CREATE_ATOMIC_NO_REPLAY"); e != nil {
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
	if e = scopedRetiredClaims(ctx, tx, h.DeleteID, project, []string{"box:" + h.Name, deleteProjectKey(project)}, "SCOPED_DELETE_ACL_ABSENT_ATOMIC", "RECOVERY_SCOPED_DELETE_ACL_ABSENT_NO_REPLAY"); e != nil {
		return zero, e
	}
	return h, nil
}
func scopedRetiredClaims(ctx context.Context, tx *sql.Tx, id, project string, expected []string, event, recovered string) error {
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
	var n, recovery int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*),COALESCE(SUM(CASE WHEN safe_detail=? THEN 1 ELSE 0 END),0) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED' AND safe_detail IN (?,?)", recovered, id, event, recovered).Scan(&n, &recovery); e != nil || n != 1 {
		return ErrResourceLocked
	}
	if recovery == 0 {
		return nil
	}
	// A recovered terminal event is not sufficient by itself. Require the
	// separate, exact read-only recovery receipt and its matching audit row.
	// Normal finalization still requires its original atomic event.
	intent, _ := json.Marshal(struct {
		Original string `json:"original_operation_id"`
	}{id})
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations o JOIN operations r
 ON r.layer=o.layer AND r.box_uuid=o.box_uuid AND r.incarnation=o.incarnation
 JOIN recovery_audit a ON a.operation_id=o.id AND a.at=r.created_at AND a.reviewer=r.actor_id
 WHERE o.id=? AND o.layer=? AND r.id!=o.id AND r.action='operation_recover'
 AND r.state='SUCCEEDED' AND r.native_operation_id='' AND r.resource_key='recovery:'||o.id
 AND CAST(r.safe_intent AS TEXT)=? AND r.actor_id!='' AND r.token_id!=''
 AND a.decision='RECOVERY_SETTLED' AND length(a.evidence_digest)=64 AND a.evidence_digest NOT GLOB '*[^0-9a-f]*'
 AND EXISTS (SELECT 1 FROM operation_events ev WHERE ev.operation_id=r.id AND ev.state='SUCCEEDED' AND ev.safe_detail='SETTLED:SCOPED_LIFECYCLE_COMPLETED_NO_REPLAY')`, id, project, string(intent)).Scan(&n); e != nil || n != 1 {
		return ErrResourceLocked
	}
	return nil
}
