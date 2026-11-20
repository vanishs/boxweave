package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Use the existing durable resource-claim ledger for internal lineage;
// no schema migration, extra public mutation, or caller-controlled parent ID.
func deleteProjectKey(project string) string { return "delete-project:" + project }
func deleteParentKey(original string) string { return "delete-parent:" + original }

type DeleteChild struct {
	Receipt Receipt
	Forward ForwardClaim
}

type deleteParent struct {
	ID, Project, Actor, Token, Work, Agent, UUID, Incarnation, Name, State string
}

func readCascadeParent(ctx context.Context, tx *sql.Tx, id string) (deleteParent, error) {
	var p deleteParent
	p.ID = id
	err := tx.QueryRowContext(ctx, "SELECT o.layer,o.actor_id,o.token_id,o.work_ref,o.agent_ref,o.box_uuid,o.incarnation,b.name,o.state FROM operations o JOIN owned_boxes b ON b.project=o.layer AND b.uuid=o.box_uuid AND b.incarnation=o.incarnation JOIN operation_resource_claims c ON c.operation_id=o.id AND c.layer=o.layer AND c.resource_key='delete-project:'||o.layer WHERE o.id=? AND o.action='box_delete' AND o.resource_key='box:'||b.name AND b.state='OWNED'", id).Scan(&p.Project, &p.Actor, &p.Token, &p.Work, &p.Agent, &p.UUID, &p.Incarnation, &p.Name, &p.State)
	return p, err
}

func countDeleteDependencies(ctx context.Context, tx *sql.Tx, project, uuid string) (peer, forward, pinned, fences int, err error) {
	err = tx.QueryRowContext(ctx, "SELECT (SELECT COUNT(*) FROM peer_grants WHERE project=? AND status!='REVOKED' AND (source_uuid=? OR target_uuid=?)),(SELECT COUNT(*) FROM forward_claims WHERE project=? AND box_uuid=? AND state!='REVOKED'),(SELECT COUNT(*) FROM network_bindings WHERE project=? AND box_uuid=? AND state!='RELEASED'),(SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=? AND state!='RELEASED')", project, uuid, uuid, project, uuid, project, uuid, project, uuid).Scan(&peer, &forward, &pinned, &fences)
	return
}

// Exact pinned identity: two RESERVED fences with matching address, family,
// bridge and incarnation, or no binding/fences at all. Unknown/quarantined
// claims can never be swept up into a delete.
func verifyDeletePinned(ctx context.Context, tx *sql.Tx, project, uuid, inc string) (bool, error) {
	var bindings, fences int
	if err := tx.QueryRowContext(ctx, "SELECT (SELECT COUNT(*) FROM network_bindings WHERE project=? AND box_uuid=? AND state!='RELEASED'),(SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=? AND state!='RELEASED')", project, uuid, project, uuid).Scan(&bindings, &fences); err != nil {
		return false, err
	}
	if bindings == 0 && fences == 0 {
		return false, nil
	}
	if bindings != 1 || fences != 2 {
		return false, ErrResourceLocked
	}
	var bridge, acl, mac, v4, v6 string
	if err := tx.QueryRowContext(ctx, "SELECT bridge,acl,mac,ipv4,ipv6 FROM network_bindings WHERE project=? AND box_uuid=? AND incarnation=? AND nic='eth0' AND state='PINNED'", project, uuid, inc).Scan(&bridge, &acl, &mac, &v4, &v6); err != nil {
		return false, ErrResourceLocked
	}
	if !auditName(bridge) || !auditName(acl) || !auditMAC(mac) || !auditIP(v4, false) || !auditIP(v6, true) {
		return false, ErrResourceLocked
	}
	var exact int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=? AND incarnation=? AND bridge=? AND state='RESERVED' AND ((family='ipv4' AND address=?) OR (family='ipv6' AND address=?))", project, uuid, inc, bridge, v4, v6).Scan(&exact); err != nil || exact != 2 {
		return false, ErrResourceLocked
	}
	return true, nil
}

func validateDeleteAdmission(ctx context.Context, tx *sql.Tx, a Admission) (bool, error) {
	peer, fwd, bind, fence, e := countDeleteDependencies(ctx, tx, a.Layer, a.BoxUUID)
	if e != nil {
		return false, e
	}
	if peer+fwd+bind+fence == 0 {
		return false, nil
	}
	if peer > 32 || fwd > 50 || a.BoxUUID == "" || a.Incarnation == "" {
		return false, ErrResourceLocked
	}
	var owned int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM owned_boxes WHERE project=? AND uuid=? AND incarnation=? AND state='OWNED' AND 'box:'||name=?", a.Layer, a.BoxUUID, a.Incarnation, a.ResourceKey).Scan(&owned); e != nil || owned != 1 {
		return false, ErrResourceLocked
	}
	var bad int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM peer_grants WHERE project=? AND status!='REVOKED' AND ((source_uuid=? AND source_incarnation!=?) OR (target_uuid=? AND target_incarnation!=?) OR status!='CONFIRMED' OR desired!=1)", a.Layer, a.BoxUUID, a.Incarnation, a.BoxUUID, a.Incarnation).Scan(&bad); e != nil || bad != 0 {
		return false, ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM forward_claims f WHERE f.project=? AND f.box_uuid=? AND f.state!='REVOKED' AND (f.state!='CONFIRMED' OR f.incarnation!=? OR NOT EXISTS (SELECT 1 FROM operations o WHERE o.id=f.forward_id AND o.action='port_forward_create' AND o.state='SUCCEEDED' AND o.layer=f.project AND o.box_uuid=f.box_uuid AND o.incarnation=f.incarnation))", a.Layer, a.BoxUUID, a.Incarnation).Scan(&bad); e != nil || bad != 0 {
		return false, ErrResourceLocked
	}
	pinned, e := verifyDeletePinned(ctx, tx, a.Layer, a.BoxUUID, a.Incarnation)
	if e != nil || peer > 0 && !pinned {
		return false, ErrResourceLocked
	}
	return true, nil
}

func (s *Store) DeleteCascadePlan(ctx context.Context, id string) (bool, []ForwardClaim, error) {
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return false, nil, e
	}
	defer tx.Rollback()
	p, e := readCascadeParent(ctx, tx, id)
	if errors.Is(e, sql.ErrNoRows) {
		return false, nil, nil
	}
	if e != nil || p.State != "DISPATCHING" {
		return false, nil, ErrResourceLocked
	}
	a := Admission{Layer: p.Project, BoxUUID: p.UUID, Incarnation: p.Incarnation, ResourceKey: "box:" + p.Name}
	if _, e = validateDeleteAdmission(ctx, tx, a); e != nil {
		return false, nil, e
	}
	rows, e := tx.QueryContext(ctx, "SELECT "+forwardColumns+" FROM forward_claims WHERE project=? AND box_uuid=? AND incarnation=? AND state='CONFIRMED' ORDER BY forward_id", p.Project, p.UUID, p.Incarnation)
	if e != nil {
		return false, nil, e
	}
	var forwards []ForwardClaim
	for rows.Next() {
		f, e := scanForward(rows)
		if e != nil {
			rows.Close()
			return false, nil, e
		}
		forwards = append(forwards, f)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return false, nil, e
	}
	if e = rows.Close(); e != nil {
		return false, nil, e
	}
	return true, forwards, tx.Commit()
}

// A root peer child is the only operation that may coexist with its parent
// during whole-Project isolation. Revalidate the link and exact ownership in
// BOTH controller staging and independent root handoff, never merely by action.
func linkedDeletePeerParent(ctx context.Context, tx *sql.Tx, child string, intent RootPeerRevokeSafeIntent, allowUnknown bool) (string, error) {
	rows, e := tx.QueryContext(ctx, "SELECT resource_key FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'delete-parent:%'", child)
	if e != nil {
		return "", e
	}
	var links []string
	for rows.Next() {
		var k string
		if e = rows.Scan(&k); e != nil {
			rows.Close()
			return "", e
		}
		links = append(links, k)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return "", e
	}
	if e = rows.Close(); e != nil {
		return "", e
	}
	if len(links) == 0 {
		return "", nil
	}
	if len(links) != 1 || intent.Allow {
		return "", ErrResourceLocked
	}
	id := strings.TrimPrefix(links[0], "delete-parent:")
	if !validHexID(id) || id == child {
		return "", ErrResourceLocked
	}
	p, e := readCascadeParent(ctx, tx, id)
	if e != nil || p.Project != intent.Project || p.State != "DISPATCHING" && !(allowUnknown && p.State == "UNKNOWN") {
		return "", ErrResourceLocked
	}
	if !((p.UUID == intent.SourceUUID && p.Incarnation == intent.SourceIncarnation) || (p.UUID == intent.TargetUUID && p.Incarnation == intent.TargetIncarnation)) {
		return "", ErrResourceLocked
	}
	var n int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations o JOIN operation_resource_claims c ON c.operation_id=o.id AND c.layer=o.layer WHERE o.id=? AND o.layer=? AND o.actor_id=? AND o.token_id=? AND o.action='box_peer_set' AND c.resource_key=?", child, p.Project, p.Actor, p.Token, deleteParentKey(p.ID)).Scan(&n); e != nil || n != 1 {
		return "", ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM recovery_audit WHERE operation_id=? AND decision='RELEASED'", p.ID).Scan(&n); e != nil || n != 0 {
		return "", ErrResourceLocked
	}
	return id, nil
}

// Internal child admissions use existing original Operation/event/claim
// contracts. They can be created only by a claimed exact delete parent while
// the whole Project is reserved, never from an MCP argument.
func (s *Store) NextDeleteChild(ctx context.Context, parent string, needStop bool) (DeleteChild, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return DeleteChild{}, e
	}
	defer tx.Rollback()
	p, e := readCascadeParent(ctx, tx, parent)
	if e != nil || p.State != "DISPATCHING" {
		return DeleteChild{}, ErrResourceLocked
	}
	active, e := activeCredentialForActor(ctx, tx, p.Actor, p.Token)
	if e != nil || !active {
		return DeleteChild{}, ErrRevoked
	}
	var busy int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations o WHERE o.layer=? AND o.id<>? AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN') AND (o.action='box_peer_set' OR NOT EXISTS (SELECT 1 FROM recovery_audit r WHERE r.operation_id=o.id AND r.decision='RELEASED'))", p.Project, p.ID).Scan(&busy); e != nil || busy != 0 {
		return DeleteChild{}, ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims c JOIN operations o ON o.id=c.operation_id WHERE c.layer=? AND c.resource_key=? AND o.state!='SUCCEEDED'", p.Project, deleteParentKey(p.ID)).Scan(&busy); e != nil || busy != 0 {
		return DeleteChild{}, ErrResourceLocked
	}
	a := Admission{Layer: p.Project, ActorID: p.Actor, TokenID: p.Token, BoxUUID: p.UUID, Incarnation: p.Incarnation, ResourceKey: "box:" + p.Name, WorkRef: p.Work, AgentRef: p.Agent}
	if _, e = validateDeleteAdmission(ctx, tx, a); e != nil {
		return DeleteChild{}, e
	}
	var out DeleteChild
	f, e := scanForward(tx.QueryRowContext(ctx, "SELECT "+forwardColumns+" FROM forward_claims WHERE project=? AND box_uuid=? AND incarnation=? AND state='CONFIRMED' ORDER BY forward_id LIMIT 1", p.Project, p.UUID, p.Incarnation))
	if e == nil {
		out.Forward = f
		a.Action = "port_forward_delete"
		a.ForwardDeleteID = f.ForwardID
		a.ResourceKeys = []string{"box:" + p.UUID, fmt.Sprintf("listener:%s:%s:%d", f.Protocol, f.HostBind, f.HostPort)}
		a.SafeIntent, _ = json.Marshal(struct {
			ForwardID string `json:"forward_id"`
		}{f.ForwardID})
	} else if !errors.Is(e, sql.ErrNoRows) {
		return out, e
	} else {
		var intent RootPeerRevokeSafeIntent
		intent.Project = p.Project
		e = tx.QueryRowContext(ctx, "SELECT source_uuid,source_incarnation,target_uuid,target_incarnation FROM peer_grants WHERE project=? AND desired=1 AND status='CONFIRMED' AND ((source_uuid=? AND source_incarnation=?) OR (target_uuid=? AND target_incarnation=?)) ORDER BY source_uuid,target_uuid LIMIT 1", p.Project, p.UUID, p.Incarnation, p.UUID, p.Incarnation).Scan(&intent.SourceUUID, &intent.SourceIncarnation, &intent.TargetUUID, &intent.TargetIncarnation)
		if e == nil {
			for _, b := range []struct{ uuid, inc string }{{intent.SourceUUID, intent.SourceIncarnation}, {intent.TargetUUID, intent.TargetIncarnation}} {
				var bridge, acl string
				if e = tx.QueryRowContext(ctx, "SELECT n.bridge,n.acl FROM network_bindings n JOIN owned_boxes b ON b.project=n.project AND b.uuid=n.box_uuid AND b.incarnation=n.incarnation WHERE n.project=? AND n.box_uuid=? AND n.incarnation=? AND b.state='OWNED' AND n.nic='eth0' AND n.state='PINNED'", p.Project, b.uuid, b.inc).Scan(&bridge, &acl); e != nil || !auditName(bridge) || !auditName(acl) {
					return out, ErrResourceLocked
				}
				if intent.Bridge != "" && (intent.Bridge != bridge || intent.ACL != acl) {
					return out, ErrResourceLocked
				}
				intent.Bridge, intent.ACL = bridge, acl
			}
			a.Action = "box_peer_set"
			a.BoxUUID, a.Incarnation = intent.SourceUUID, intent.SourceIncarnation
			a.ResourceKey = "acl:default/" + intent.ACL
			a.ResourceKeys = []string{"box:" + intent.SourceUUID, "box:" + intent.TargetUUID}
			a.SafeIntent, _ = json.Marshal(intent)
		} else if !errors.Is(e, sql.ErrNoRows) {
			return out, e
		} else if needStop {
			a.Action = "box_stop"
			a.SafeIntent, _ = json.Marshal(struct {
				BoxID string `json:"box_id"`
			}{p.UUID})
		} else {
			return out, tx.Commit()
		}
	}
	id, e := newID()
	if e != nil {
		return out, e
	}
	tracking, e := newID()
	if e != nil {
		return out, e
	}
	key, e := newID()
	if e != nil {
		return out, e
	}
	a.Key = "b5-internal-" + key
	a.RequestID = tracking
	claims, e := normalizeAdmissionResourceKeys(a)
	if e != nil {
		return out, e
	}
	now := time.Now().UnixNano()
	_, e = tx.ExecContext(ctx, "INSERT INTO operations(id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,intent_digest,safe_intent,work_ref,agent_ref,resource_key,box_uuid,incarnation,state,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,'ADMITTED',?,?)", id, tracking, tracking, a.Layer, a.ActorID, a.TokenID, digest(a.Key), a.Action, admissionIntentDigest(a, claims), a.SafeIntent, a.WorkRef, a.AgentRef, a.ResourceKey, a.BoxUUID, a.Incarnation, now, now)
	if e != nil {
		return out, e
	}
	claims = append(claims, deleteParentKey(parent))
	for _, k := range claims {
		if _, e = tx.ExecContext(ctx, "INSERT INTO operation_resource_claims(operation_id,layer,resource_key) VALUES(?,?,?)", id, p.Project, k); e != nil {
			return out, e
		}
	}
	if e = stageForwardAdmission(ctx, tx, a, id); e != nil {
		return out, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'ADMITTED',?)", id, now, "DELETE_PARENT:"+parent); e != nil {
		return out, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES(?,?,'DISPATCHING',?)", parent, now, "DELETE_CHILD:"+id+":"+a.Action); e != nil {
		return out, e
	}
	if e = tx.Commit(); e != nil {
		return out, e
	}
	out.Receipt = Receipt{ID: id, TrackingID: tracking, State: "ADMITTED", Action: a.Action, BoxUUID: a.BoxUUID, Incarnation: a.Incarnation, WorkRef: a.WorkRef, AgentRef: a.AgentRef, CreatedAt: now, UpdatedAt: now, NextAction: nextAction("ADMITTED", a.Action, id)}
	return out, nil
}

func (s *Store) AuthorizeCascadeFinal(ctx context.Context, parent string) error {
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return e
	}
	defer tx.Rollback()
	p, e := readCascadeParent(ctx, tx, parent)
	if e != nil || p.State != "DISPATCHING" {
		return ErrResourceLocked
	}
	active, e := activeCredentialForActor(ctx, tx, p.Actor, p.Token)
	if e != nil || !active {
		return ErrRevoked
	}
	if e = checkCascadeRelease(ctx, tx, parent); e != nil {
		return e
	}
	peer, fwd, _, _, e := countDeleteDependencies(ctx, tx, p.Project, p.UUID)
	if e != nil || peer+fwd != 0 {
		return ErrResourceLocked
	}
	return tx.Commit()
}

func checkCascadeRelease(ctx context.Context, tx *sql.Tx, id string) error {
	var n int
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims c JOIN operations o ON o.id=c.operation_id WHERE c.resource_key=? AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')", deleteParentKey(id)).Scan(&n); e != nil || n != 0 {
		return ErrResourceLocked
	}
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations o JOIN peer_grants p ON p.project=o.layer AND (p.source_uuid=o.box_uuid OR p.target_uuid=o.box_uuid) WHERE o.id=? AND p.status IN ('PENDING','UNKNOWN')", id).Scan(&n); e != nil || n != 0 {
		return ErrResourceLocked
	}
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations o JOIN forward_claims f ON f.project=o.layer AND f.box_uuid=o.box_uuid WHERE o.id=? AND f.state IN ('PENDING','UNKNOWN')", id).Scan(&n); e != nil || n != 0 {
		return ErrResourceLocked
	}
	return nil
}

func finalizeDeleteDependencies(ctx context.Context, tx *sql.Tx, id string, b OwnedBox) error {
	peer, fwd, _, _, e := countDeleteDependencies(ctx, tx, b.Project, b.UUID)
	if e != nil || peer+fwd != 0 {
		return ErrResourceLocked
	}
	if e = checkCascadeRelease(ctx, tx, id); e != nil {
		return e
	}
	pinned, e := verifyDeletePinned(ctx, tx, b.Project, b.UUID, b.Incarnation)
	if e != nil {
		return e
	}
	if !pinned {
		return nil
	}
	var marker int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND layer=? AND resource_key=?", id, b.Project, deleteProjectKey(b.Project)).Scan(&marker); e != nil || marker != 1 {
		return ErrResourceLocked
	}
	// Keep historical addresses non-reassignable (ReserveNetworkAddress refuses
	// RELEASED) and keep DROP tombstones for later whole-ACL recompilation.
	now := time.Now().UnixNano()
	r, e := tx.ExecContext(ctx, "UPDATE network_address_fences SET state='RELEASED',operation_id=?,observed_at=? WHERE project=? AND box_uuid=? AND incarnation=? AND state='RESERVED'", id, now, b.Project, b.UUID, b.Incarnation)
	if e != nil {
		return e
	}
	if n, e := r.RowsAffected(); e != nil || n != 2 {
		return ErrResourceLocked
	}
	r, e = tx.ExecContext(ctx, "UPDATE network_bindings SET state='RELEASED',observed_at=? WHERE project=? AND box_uuid=? AND incarnation=? AND state='PINNED'", now, b.Project, b.UUID, b.Incarnation)
	if e != nil {
		return e
	}
	if n, e := r.RowsAffected(); e != nil || n != 1 {
		return ErrResourceLocked
	}
	return nil
}

type DeletePinnedNIC struct{ Bridge, ACL, MAC, IPv4, IPv6 string }

// Read only the original parent scope, after exact admission and claim.
func (s *Store) DeletePinnedNIC(ctx context.Context, id string) (*DeletePinnedNIC, error) {
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	p, e := readCascadeParent(ctx, tx, id)
	if e != nil || p.State != "DISPATCHING" {
		return nil, ErrResourceLocked
	}
	pinned, e := verifyDeletePinned(ctx, tx, p.Project, p.UUID, p.Incarnation)
	if e != nil {
		return nil, e
	}
	if !pinned {
		return nil, tx.Commit()
	}
	var n DeletePinnedNIC
	if e = tx.QueryRowContext(ctx, "SELECT bridge,acl,mac,ipv4,ipv6 FROM network_bindings WHERE project=? AND box_uuid=? AND incarnation=? AND nic='eth0' AND state='PINNED'", p.Project, p.UUID, p.Incarnation).Scan(&n.Bridge, &n.ACL, &n.MAC, &n.IPv4, &n.IPv6); e != nil {
		return nil, e
	}
	return &n, tx.Commit()
}
