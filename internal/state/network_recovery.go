package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// A4 uses only the exact protected claim linked to an original. No
// listener allocation, device mutation or original replay is permitted.
func readRecoveryForward(ctx context.Context, q credentialRowQuery, t RecoveryTarget) (ForwardClaim, bool, error) {
	var zero ForwardClaim
	f, e := scanForward(q.QueryRowContext(ctx, "SELECT "+forwardColumns+" FROM forward_claims WHERE project=? AND operation_id=?", t.Layer, t.ID))
	if errors.Is(e, sql.ErrNoRows) {
		return zero, false, nil
	}
	if e != nil {
		return zero, false, e
	}
	if f.BoxID != t.UUID || f.Incarnation != t.Incarnation || f.State != "PENDING" || f.OperationID != t.ID || !validHexID(f.ForwardID) || f.NativeDevice != "bwpf-"+f.ForwardID[:16] || f.GuestIP != "127.0.0.1" || !validForwardCreate(ForwardCreateAdmission{Protocol: f.Protocol, HostBind: f.HostBind, HostPort: f.HostPort, GuestPort: f.GuestPort}) {
		return zero, false, nil
	}
	var n int
	if e = q.QueryRowContext(ctx, "SELECT count(*) FROM forward_claims WHERE operation_id=?", t.ID).Scan(&n); e != nil {
		return zero, false, e
	}
	if n != 1 {
		return zero, false, nil
	}
	var createIntent string
	if t.Action == "port_forward_create" {
		if f.ForwardID != t.ID {
			return zero, false, nil
		}
		createIntent = t.SafeIntent
	} else {
		var in struct {
			ForwardID string `json:"forward_id"`
		}
		if json.Unmarshal([]byte(t.SafeIntent), &in) != nil || in.ForwardID != f.ForwardID {
			return zero, false, nil
		}
		e = q.QueryRowContext(ctx, "SELECT safe_intent FROM operations WHERE id=? AND layer=? AND box_uuid=? AND incarnation=? AND action='port_forward_create' AND state='SUCCEEDED'", f.ForwardID, t.Layer, t.UUID, t.Incarnation).Scan(&createIntent)
		if errors.Is(e, sql.ErrNoRows) {
			return zero, false, nil
		}
		if e != nil {
			return zero, false, e
		}
	}
	var in struct {
		BoxID     string `json:"box_id"`
		Protocol  string `json:"protocol"`
		HostBind  string `json:"host_bind"`
		HostPort  int    `json:"host_port"`
		GuestPort int    `json:"guest_port"`
		External  bool   `json:"confirm_external_exposure"`
	}
	if json.Unmarshal([]byte(createIntent), &in) != nil || in.BoxID != t.UUID || in.Protocol != f.Protocol || in.HostBind != f.HostBind || in.HostPort != f.HostPort || in.GuestPort != f.GuestPort {
		return zero, false, nil
	}
	// Nonloopback listener was originally explicitly reviewed. The recovery
	// additionally rechecks current host exposure allowlist for Create.
	if f.HostBind != "127.0.0.1" && f.HostBind != "::1" && !in.External {
		return zero, false, nil
	}
	return f, true, nil
}

func recoveryClaimKeys(ctx context.Context, tx *sql.Tx, t RecoveryTarget, want []string) error {
	rows, e := tx.QueryContext(ctx, "SELECT layer,resource_key FROM operation_resource_claims WHERE operation_id=? ORDER BY resource_key", t.ID)
	if e != nil {
		return e
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var layer, key string
		if e = rows.Scan(&layer, &key); e != nil {
			return e
		}
		if layer != t.Layer {
			return ErrResourceLocked
		}
		keys = append(keys, key)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	slices.Sort(want)
	if !slices.Equal(keys, want) {
		return ErrResourceLocked
	}
	return nil
}

// A linked stopped parent may coexist only with its exact already-admitted
// child. It stays UNKNOWN; this exception never schedules the next child
// or the final Delete. All other active Project operations remain barred.
func recoveryDeleteChildParent(ctx context.Context, tx *sql.Tx, t RecoveryTarget) (string, error) {
	rows, e := tx.QueryContext(ctx, "SELECT resource_key FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'delete-parent:%'", t.ID)
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
	if len(links) != 1 || (t.Action != "port_forward_delete" && t.Action != "box_stop") {
		return "", ErrResourceLocked
	}
	id := strings.TrimPrefix(links[0], "delete-parent:")
	if !validHexID(id) || id == t.ID {
		return "", ErrResourceLocked
	}
	p, e := readCascadeParent(ctx, tx, id)
	if e != nil || p.Project != t.Layer || p.UUID != t.UUID || p.Incarnation != t.Incarnation || p.Name != t.Name || p.State != "UNKNOWN" || p.ID != id {
		return "", ErrResourceLocked
	}
	var n int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM operations WHERE id=? AND layer=? AND actor_id=? AND token_id=? AND box_uuid=? AND incarnation=?", t.ID, p.Project, p.Actor, p.Token, p.UUID, p.Incarnation).Scan(&n); e != nil || n != 1 {
		return "", ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM recovery_audit WHERE operation_id=? AND decision='RELEASED'", id).Scan(&n); e != nil || n != 0 {
		return "", ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM operation_resource_claims c JOIN operations o ON o.id=c.operation_id WHERE c.layer=? AND c.resource_key=? AND o.id<>? AND o.state!='SUCCEEDED'", t.Layer, deleteParentKey(id), t.ID).Scan(&n); e != nil || n != 0 {
		return "", ErrResourceLocked
	}
	return id, nil
}

func recoveryDependentDeleteReady(ctx context.Context, tx *sql.Tx, t RecoveryTarget) error {
	if !t.DependentDelete || t.Action != "box_delete" || t.State != "UNKNOWN" || t.Released || t.NativeID == "" {
		return ErrResourceLocked
	}
	p, e := readCascadeParent(ctx, tx, t.ID)
	if e != nil || p.State != "UNKNOWN" || p.Project != t.Layer || p.UUID != t.UUID || p.Incarnation != t.Incarnation || p.Name != t.Name {
		return ErrResourceLocked
	}
	if e = checkCascadeRelease(ctx, tx, t.ID); e != nil {
		return e
	}
	var n int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM operation_resource_claims c JOIN operations o ON o.id=c.operation_id WHERE c.resource_key=? AND (c.layer<>? OR o.layer<>? OR o.state!='SUCCEEDED')", deleteParentKey(t.ID), t.Layer, t.Layer).Scan(&n); e != nil || n != 0 {
		return ErrResourceLocked
	}
	peer, fwd, _, _, e := countDeleteDependencies(ctx, tx, t.Layer, t.UUID)
	if e != nil || peer+fwd != 0 {
		return ErrResourceLocked
	}
	if e = requireDeletePinLineage(ctx, tx, t.Layer, t.UUID, t.Incarnation, t.Resource); e != nil {
		return e
	}
	if _, e = verifyDeletePinned(ctx, tx, t.Layer, t.UUID, t.Incarnation); e != nil {
		return e
	}
	return nil
}

func (s *Store) CheckDependentDeleteRecovery(ctx context.Context, t RecoveryTarget) error {
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return e
	}
	defer tx.Rollback()
	current, e := readRecoveryTarget(ctx, tx, t.Layer, t.ID)
	if e != nil || current != t {
		return ErrConflict
	}
	if e = recoveryDependentDeleteReady(ctx, tx, t); e != nil {
		return e
	}
	return tx.Commit()
}

func finishRecoveryForwardTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget, p RecoveryProof, parent string) error {
	if !t.ForwardReady || p.Forward != t.Forward || p.Fingerprint == "" || p.Reason != "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED" {
		return ErrResourceLocked
	}
	f := t.Forward
	want := []string{"box:" + t.Name, "box:" + t.UUID, fmt.Sprintf("listener:%s:%s:%d", f.Protocol, f.HostBind, f.HostPort)}
	if parent != "" {
		want = append(want, deleteParentKey(parent))
	}
	if e := recoveryClaimKeys(ctx, tx, t, want); e != nil {
		return e
	}
	return finishVerifiedForwardTx(ctx, tx, t.ID, t.Action == "port_forward_create", "UNKNOWN", "RECOVERY_FORWARD_TERMINAL_NO_REPLAY")
}
