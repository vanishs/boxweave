package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
)

// RootPeerGrant is a read-only authorization proof for one already-claimed
// allow=true original and a complete, exact directional allowlist. It is NOT
// native ACL permission, a packet witness or an authority to change host root.
type RootPeerGrant struct {
	OperationID string
	Intent RootPeerRevokeSafeIntent
	Confirmed []PeerGrantBinding
	Target []PeerGrantBinding
}

// ReadRootPeerGrant is the B3 analogue of ReadRootPeerRevocation. It refuses
// ADMITTED/UNKNOWN originals and does NOT loosen B2's revoke-only checker.
// The root process must separately reprove NIC/native scope before dispatch.
func (s *Store) ReadRootPeerGrant(ctx context.Context, original, project, bridge, acl string) (RootPeerGrant, error) {
	var zero RootPeerGrant
	if s == nil || s.db == nil || !s.IsNetworkAuditReadOnly() ||
		!validHexID(original) || !auditName(project) || project == "default" ||
		!auditName(bridge) || !auditName(acl) {
		return zero, ErrInvalid
	}
	var action, status, actor, token, layer, box, incarnation string
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT action,state,actor_id,token_id,layer,box_uuid,incarnation,safe_intent
 FROM operations WHERE id=?`, original).Scan(&action, &status, &actor, &token, &layer, &box, &incarnation, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return zero, ErrNotFound
	}
	if err != nil {
		return zero, err
	}
	var intent RootPeerRevokeSafeIntent
	if json.Unmarshal(raw, &intent) != nil {
		return zero, ErrResourceLocked
	}
	canonical, err := json.Marshal(intent)
	if err != nil || !bytes.Equal(canonical, raw) || action != "box_peer_set" ||
		status != "DISPATCHING" || actor == "" || token == "" ||
		!intent.Allow || layer != project || intent.Project != project ||
		intent.Bridge != bridge || intent.ACL != acl ||
		intent.SourceUUID == "" || intent.SourceUUID == intent.TargetUUID ||
		intent.SourceIncarnation == "" || intent.TargetIncarnation == "" ||
		box != intent.SourceUUID || incarnation != intent.SourceIncarnation {
		return zero, ErrResourceLocked
	}
	// Snapshot independently enforces complete SQLite NIC/IPv4/v6/MAC/fence
	// ledger, full-project sibling barrier, active actor/admin token, exact
	// ACL+two Box resource claims, pending/UNKNOWN uniqueness and all rows.
	target, err := s.ReadPeerGrantSnapshot(ctx, project, bridge, acl, original)
	if err != nil {
		return zero, err
	}
	var confirmed []PeerGrantBinding
	found := 0
	for _, row := range target {
		if row.SourceUUID == intent.SourceUUID && row.SourceIncarnation == intent.SourceIncarnation &&
			row.TargetUUID == intent.TargetUUID && row.TargetIncarnation == intent.TargetIncarnation {
			found++
		} else {
			confirmed = append(confirmed, row)
		}
	}
	if found != 1 {
		return zero, ErrResourceLocked
	}
	// Snapshot is a separate read transaction. Repeat the original identity
	// after it: no caller- or IPC-supplied ACL may become the native target.
	var stateAfter string
	var rawAfter []byte
	err = s.db.QueryRowContext(ctx, "SELECT state,safe_intent FROM operations WHERE id=?", original).
		Scan(&stateAfter, &rawAfter)
	if err != nil || stateAfter != "DISPATCHING" || !bytes.Equal(raw, rawAfter) {
		return zero, ErrResourceLocked
	}
	if err := s.RequireActiveTokenID(ctx, token); err != nil {
		return zero, ErrRevoked
	}
	if len(target) != len(confirmed)+1 || reflect.DeepEqual(target, confirmed) {
		return zero, ErrResourceLocked
	}
	return RootPeerGrant{OperationID: original, Intent: intent, Confirmed: confirmed, Target: target}, nil
}
