package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"

	"github.com/vanishs/boxweave/internal/peerscope"
)

type ScopedDeleteOriginalProof struct {
	OriginalID, NativeID, CreateID string
	Box                            OwnedBox
	Binding                        VerifiedCreateBinding
}

func readScopedDeleteOriginalTx(ctx context.Context, tx *sql.Tx, id, project, bridge string) (ScopedDeleteOriginalProof, error) {
	var zero ScopedDeleteOriginalProof
	if !validHexID(id) || !auditName(project) || !auditName(bridge) {
		return zero, ErrInvalid
	}
	var p ScopedDeleteOriginalProof
	var raw []byte
	var action, phase, resource, actor, token string
	p.OriginalID = id
	err := tx.QueryRowContext(ctx, `SELECT o.native_operation_id,o.action,o.state,o.resource_key,o.actor_id,o.token_id,o.safe_intent,b.uuid,b.project,b.name,b.incarnation,b.state
 FROM operations o JOIN owned_boxes b ON b.project=o.layer AND b.uuid=o.box_uuid AND b.incarnation=o.incarnation WHERE o.id=?`, id).Scan(&p.NativeID, &action, &phase, &resource, &actor, &token, &raw, &p.Box.UUID, &p.Box.Project, &p.Box.Name, &p.Box.Incarnation, &p.Box.State)
	if err != nil {
		return zero, err
	}
	canonical, err := json.Marshal(struct {
		BoxID string `json:"box_id"`
	}{p.Box.UUID})
	if err != nil || !bytes.Equal(raw, canonical) || action != "box_delete" || phase != "VERIFYING" || resource != "box:"+p.Box.Name || p.Box.State != "OWNED" || p.Box.Project != project || p.Box.Incarnation != p.Box.UUID || p.NativeID == "" {
		return zero, ErrResourceLocked
	}
	active, err := activeCredentialForActor(ctx, tx, actor, token)
	if err != nil || !active {
		return zero, ErrRevoked
	}
	p.Binding, phase, err = readLifecycleRecoveryPin(ctx, tx, RecoveryTarget{Layer: project, UUID: p.Box.UUID, Incarnation: p.Box.Incarnation, Resource: resource})
	if err != nil || phase != "PINNED" || p.Binding.Bridge != bridge {
		return zero, ErrResourceLocked
	}
	acl, err := peerscope.ACLName(project, bridge, p.Box.UUID)
	if err != nil || p.Binding.ACL != acl {
		return zero, ErrResourceLocked
	}
	if err = tx.QueryRowContext(ctx, "SELECT id FROM operations WHERE layer=? AND box_uuid=? AND incarnation=? AND action='box_create' AND state='SUCCEEDED'", project, p.Box.UUID, p.Box.Incarnation).Scan(&p.CreateID); err != nil {
		return zero, err
	}
	schema, err := scopedCreateIntentSchema(ctx, tx, p.CreateID)
	if err != nil || schema != 2 {
		return zero, ErrResourceLocked
	}
	var n int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND layer=? AND resource_key=?", id, project, deleteProjectKey(project)).Scan(&n); err != nil || n != 1 {
		return zero, ErrResourceLocked
	}
	peer, forward, _, _, err := countDeleteDependencies(ctx, tx, project, p.Box.UUID)
	if err != nil || peer+forward != 0 {
		return zero, ErrResourceLocked
	}
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM peer_grants WHERE project=?", project).Scan(&n); err != nil || n != 0 {
		return zero, ErrResourceLocked
	}
	return p, nil
}
func (s *Store) ReadRootScopedDeleteOriginal(ctx context.Context, id, project, bridge string) (ScopedDeleteOriginalProof, error) {
	var zero ScopedDeleteOriginalProof
	if s == nil || s.db == nil || !s.IsNetworkAuditReadOnly() {
		return zero, ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	p, err := readScopedDeleteOriginalTx(ctx, tx, id, project, bridge)
	if err != nil {
		return zero, err
	}
	return p, tx.Commit()
}
