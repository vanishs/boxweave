package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// RootPeerRevokeSafeIntent is the ONLY canonical controller-side projection
// the host-root peer worker recognizes. Neither MCP JSON nor a socket caller
// can supply native IP addresses, ACL rules or a shell command.
type RootPeerRevokeSafeIntent struct {
	Project           string `json:"project"`
	Bridge            string `json:"bridge"`
	ACL               string `json:"acl"`
	SourceUUID        string `json:"source_uuid"`
	SourceIncarnation string `json:"source_incarnation"`
	TargetUUID        string `json:"target_uuid"`
	TargetIncarnation string `json:"target_incarnation"`
	Allow             bool   `json:"allow"`
}

// RootPeerRevocation is a protected, read-only, original-operation proof.
// It says that SQLite has a pending revoke; it DOES NOT approve native
// mutation, release a network fence or establish any root ACL readiness.
type RootPeerRevocation struct {
	OperationID string
	Intent      RootPeerRevokeSafeIntent
}

// ReadRootPeerRevocation independently verifies the original, claimed
// DISPATCHING operation and its exact owned peer_grants, address lease,
// canonical safe intent, credential and multi-resource claims. No arbitrary
// unclaimed or UNKNOWN operation is ever eligible for host-root dispatch.
func (s *Store) ReadRootPeerRevocation(ctx context.Context, id, project, bridge, acl string) (RootPeerRevocation, error) {
	return s.readRootPeerRevocationState(ctx, id, project, bridge, acl, "DISPATCHING")
}

// ReadRootPeerUnknownRevocation is for read-only host-root post-mutation
// reconciliation. It preserves every claim, token, ledger and UUID check,
// and NEVER authorizes a second native dispatch of the old operation.
func (s *Store) ReadRootPeerUnknownRevocation(ctx context.Context, id, project, bridge, acl string) (RootPeerRevocation, error) {
	return s.readRootPeerRevocationState(ctx, id, project, bridge, acl, "UNKNOWN")
}

func (s *Store) readRootPeerRevocationState(ctx context.Context, id, project, bridge, acl, expectedState string) (RootPeerRevocation, error) {
	if expectedState != "DISPATCHING" && expectedState != "UNKNOWN" {
		return RootPeerRevocation{}, ErrInvalid
	}
	var zero RootPeerRevocation
	if !s.IsNetworkAuditReadOnly() || !validHexID(id) || !auditName(project) ||
		project == "default" || !auditName(bridge) || !auditName(acl) {
		return zero, ErrInvalid
	}
	ledger, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil {
		return zero, fmt.Errorf("incomplete protected root network ledger: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	var action, state, actor, token, boxUUID, incarnation, layer string
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT action,state,actor_id,token_id,box_uuid,incarnation,layer,safe_intent
		FROM operations WHERE id=?`, id).Scan(&action, &state, &actor, &token, &boxUUID, &incarnation, &layer, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return zero, ErrNotFound
	}
	if err != nil {
		return zero, err
	}
	if action != "box_peer_set" || state != expectedState || actor == "" ||
		token == "" || layer != project {
		return zero, ErrResourceLocked
	}
	var intent RootPeerRevokeSafeIntent
	if err := json.Unmarshal(raw, &intent); err != nil {
		return zero, ErrInvalid
	}
	canonical, err := json.Marshal(intent)
	if err != nil || !bytes.Equal(raw, canonical) || intent.Allow ||
		intent.Project != project || intent.Bridge != bridge || intent.ACL != acl ||
		intent.SourceUUID == "" || intent.TargetUUID == "" ||
		intent.SourceUUID == intent.TargetUUID ||
		boxUUID != intent.SourceUUID || incarnation != intent.SourceIncarnation {
		return zero, ErrResourceLocked
	}
	// Host root repeats the independent barrier proof before any native effect.
	// An unclaimed sibling may be operating in a pre-upgrade database: refuse.
	parent, e := linkedDeletePeerParent(ctx, tx, id, intent, expectedState == "UNKNOWN")
	if e != nil {
		return zero, e
	}
	var concurrent int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations o
		WHERE o.layer=? AND o.id<>? AND o.id<>?
		AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')
		AND (o.action='box_peer_set' OR NOT EXISTS (SELECT 1 FROM recovery_audit r
		  WHERE r.operation_id=o.id AND r.decision='RELEASED'))`,
		project, id, parent).Scan(&concurrent); err != nil || concurrent != 0 {
		return zero, ErrResourceLocked
	}
	owner := map[string]NetworkAuditBinding{}
	for _, b := range ledger.Boxes {
		owner[b.UUID] = b
	}
	src, srcOK := owner[intent.SourceUUID]
	dst, dstOK := owner[intent.TargetUUID]
	if !srcOK || !dstOK || src.Incarnation != intent.SourceIncarnation ||
		dst.Incarnation != intent.TargetIncarnation {
		return zero, ErrResourceLocked
	}
	var n int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM peer_grants WHERE operation_id=?", id).Scan(&n); err != nil {
		return zero, err
	}
	if n != 1 {
		return zero, ErrResourceLocked
	}
	var desired int
	var status, grantSrcInc, grantDstInc, src4, dst4, src6, dst6 string
	err = tx.QueryRowContext(ctx, `SELECT desired,status,source_incarnation,target_incarnation,
			source_ipv4,target_ipv4,source_ipv6,target_ipv6
		FROM peer_grants WHERE project=? AND source_uuid=? AND target_uuid=? AND operation_id=?`,
		project, intent.SourceUUID, intent.TargetUUID, id).
		Scan(&desired, &status, &grantSrcInc, &grantDstInc, &src4, &dst4, &src6, &dst6)
	if err != nil || desired != 0 || status != "PENDING" ||
		grantSrcInc != src.Incarnation || grantDstInc != dst.Incarnation ||
		src4 != src.IPv4 || dst4 != dst.IPv4 ||
		src6 != src.IPv6 || dst6 != dst.IPv6 {
		return zero, ErrResourceLocked
	}
	rows, err := tx.QueryContext(ctx, "SELECT resource_key FROM operation_resource_claims WHERE operation_id=? ORDER BY resource_key", id)
	if err != nil {
		return zero, err
	}
	var claims []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return zero, err
		}
		claims = append(claims, key)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return zero, err
	}
	if err := rows.Close(); err != nil {
		return zero, err
	}
	expected := []string{"acl:default/" + acl, "box:" + src.UUID, "box:" + dst.UUID}
	if parent != "" {
		expected = append(expected, deleteParentKey(parent))
	}
	slices.Sort(expected)
	if !slices.Equal(claims, expected) {
		return zero, ErrResourceLocked
	}
	active, e := activeCredentialForActor(ctx, tx, actor, token)
	if e != nil || !active {
		return zero, ErrResourceLocked
	}
	if err := tx.Commit(); err != nil {
		return zero, err
	}
	if err := s.RequireActiveTokenID(ctx, token); err != nil {
		return zero, ErrResourceLocked
	}
	return RootPeerRevocation{OperationID: id, Intent: intent}, nil
}
