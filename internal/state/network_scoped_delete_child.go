package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"strings"
)

// Recheck the private child link in staging, independent Root read and
// historical policy audit. No caller-supplied parent ID grants authority.
// Terminal history survives credential revocation and parent tombstoning.
func scopedPeerParentClaims(ctx context.Context, tx *sql.Tx, id string, i ScopedPeerOriginalIntent, phase string) ([]string, string, error) {
	keys, e := scopedPeerResourceClaims(i)
	if e != nil {
		return nil, "", e
	}
	rows, e := tx.QueryContext(ctx, "SELECT resource_key,layer FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'delete-parent:%'", id)
	if e != nil {
		return nil, "", e
	}
	var links []string
	for rows.Next() {
		var k, l string
		if e = rows.Scan(&k, &l); e != nil || l != i.Project {
			rows.Close()
			return nil, "", ErrResourceLocked
		}
		links = append(links, k)
	}
	e = rows.Err()
	ce := rows.Close()
	if e != nil || ce != nil {
		return nil, "", ErrResourceLocked
	}
	var events int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM operation_events WHERE operation_id=? AND safe_detail LIKE 'DELETE_PARENT:%'", id).Scan(&events); e != nil {
		return nil, "", e
	}
	var parentEvents int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM operation_events WHERE safe_detail=?", "DELETE_CHILD:"+id+":box_peer_set").Scan(&parentEvents); e != nil {
		return nil, "", e
	}
	if len(links) == 0 && events == 0 && parentEvents == 0 {
		return keys, "", nil
	}
	if len(links) != 1 || events != 1 || parentEvents != 1 || i.Allow {
		return nil, "", ErrResourceLocked
	}
	parent := strings.TrimPrefix(links[0], "delete-parent:")
	if !validHexID(parent) || parent == id {
		return nil, "", ErrResourceLocked
	}
	var project, actor, token, uuid, inc, name, resource, st, nativeID, owned string
	var raw []byte
	e = tx.QueryRowContext(ctx, `SELECT o.layer,o.actor_id,o.token_id,o.box_uuid,o.incarnation,b.name,o.resource_key,o.state,o.native_operation_id,b.state,o.safe_intent FROM operations o JOIN owned_boxes b ON b.project=o.layer AND b.uuid=o.box_uuid AND b.incarnation=o.incarnation WHERE o.id=? AND o.action='box_delete'`, parent).Scan(&project, &actor, &token, &uuid, &inc, &name, &resource, &st, &nativeID, &owned, &raw)
	canonical, _ := json.Marshal(struct {
		BoxID string `json:"box_id"`
	}{uuid})
	if e != nil || project != i.Project || actor == "" || token == "" || resource != "box:"+name || !bytes.Equal(raw, canonical) || !((uuid == i.SourceUUID && inc == i.SourceIncarnation) || (uuid == i.TargetUUID && inc == i.TargetIncarnation)) {
		return nil, "", ErrResourceLocked
	}
	if phase == "SUCCEEDED" {
		if !slices.Contains([]string{"DISPATCHING", "VERIFYING", "UNKNOWN", "SUCCEEDED", "FAILED", "CANCELLED", "TIMED_OUT"}, st) || (owned != "OWNED" && owned != "DELETED") {
			return nil, "", ErrResourceLocked
		}
	} else {
		want := "DISPATCHING"
		if phase == "UNKNOWN" {
			want = "UNKNOWN"
		}
		if st != want || owned != "OWNED" || nativeID != "" {
			return nil, "", ErrResourceLocked
		}
	}
	// Exact parent claims and bidirectional events, not only a child label.
	for _, q := range []struct {
		sql  string
		args []any
		want int
	}{
		{"SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=?", []any{parent}, 2},
		{"SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND layer=? AND resource_key IN (?,?)", []any{parent, project, resource, deleteProjectKey(project)}, 2},
		{"SELECT COUNT(*) FROM operations WHERE id=? AND layer=? AND actor_id=? AND token_id=?", []any{id, project, actor, token}, 1},
		{"SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='ADMITTED' AND safe_detail=?", []any{id, "DELETE_PARENT:" + parent}, 1},
		{"SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='DISPATCHING' AND safe_detail=?", []any{parent, "DELETE_CHILD:" + id + ":box_peer_set"}, 1},
		{"SELECT COUNT(*) FROM recovery_audit WHERE operation_id=? AND decision='RELEASED'", []any{parent}, 0},
	} {
		var n int
		if e = tx.QueryRowContext(ctx, q.sql, q.args...).Scan(&n); e != nil || n != q.want {
			return nil, "", ErrResourceLocked
		}
	}
	keys = append(keys, links[0])
	slices.Sort(keys)
	return keys, parent, nil
}
