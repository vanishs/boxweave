package state

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// CreatePinRepairStatus is local SQLite evidence, never native authority.
func (s *Store) CreatePinRepairStatus(ctx context.Context, t RecoveryTarget) (string, error) {
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	current, e := readRecoveryTarget(ctx, tx, t.Layer, t.ID)
	if e != nil {
		return "", e
	}
	if current != t {
		return "", ErrConflict
	}
	return createPinRepairStatusTx(ctx, tx, t)
}
func createPinRepairStatusTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget) (string, error) {
	if t.Action != "box_create" || t.State != "SUCCEEDED" || t.OwnedState != "OWNED" || !t.CreatePinRequired || !t.CreatePinStaged ||
		t.Released || t.UUID == "" || t.UUID != t.Incarnation || t.Name == "" || t.Resource != "box:"+t.Name {
		return "UNSUPPORTED", nil
	}
	var nb, nf int
	e := tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM network_bindings WHERE project=? AND box_uuid=?),(SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=?)`, t.Layer, t.UUID, t.Layer, t.UUID).Scan(&nb, &nf)
	if e != nil {
		return "", e
	}
	if nb > 1 || nf > 2 {
		return "CONFLICT", nil
	}
	if nb == 1 && nf == 2 {
		var exact int
		e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_bindings n
 JOIN network_address_fences f4 ON f4.bridge=n.bridge AND f4.family='ipv4' AND f4.address=n.ipv4
 AND f4.project=n.project AND f4.box_uuid=n.box_uuid AND f4.incarnation=n.incarnation AND f4.state='RESERVED' AND f4.operation_id=?
 JOIN network_address_fences f6 ON f6.bridge=n.bridge AND f6.family='ipv6' AND f6.address=n.ipv6
 AND f6.project=n.project AND f6.box_uuid=n.box_uuid AND f6.incarnation=n.incarnation AND f6.state='RESERVED' AND f6.operation_id=?
 WHERE n.project=? AND n.box_uuid=? AND n.incarnation=? AND n.nic='eth0' AND n.state='PINNED'`, t.ID, t.ID, t.Layer, t.UUID, t.Incarnation).Scan(&exact)
		if e != nil {
			return "", e
		}
		if exact == 1 {
			return "COMPLETE", nil
		}
		return "CONFLICT", nil
	}
	return "MISSING", nil
}

func (s *Store) CheckCreatePinRepair(ctx context.Context, t RecoveryTarget, p RecoveryProof) error {
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return e
	}
	defer tx.Rollback()
	current, e := readRecoveryTarget(ctx, tx, t.Layer, t.ID)
	if e != nil {
		return e
	}
	if current != t {
		return ErrConflict
	}
	return checkCreatePinRepairTx(ctx, tx, t, p)
}

func finishCreatePinRepairTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget, p RecoveryProof) error {
	if e := checkCreatePinRepairTx(ctx, tx, t, p); e != nil {
		return e
	}
	b := p.CreateBinding
	now := time.Now().UnixNano()
	var n int
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM network_bindings WHERE project=? AND box_uuid=?", t.Layer, t.UUID).Scan(&n); e != nil {
		return e
	}
	if n == 0 {
		_, e := tx.ExecContext(ctx, `INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at)
 VALUES(?,?,?,'eth0',?,?,?,?,?,'PINNED',?)`, t.Layer, t.UUID, t.Incarnation, b.Bridge, b.ACL, b.MAC, b.IPv4, b.IPv6, now)
		if e != nil {
			return e
		}
	}
	for _, f := range []struct{ family, ip string }{{"ipv4", b.IPv4}, {"ipv6", b.IPv6}} {
		var old string
		e := tx.QueryRowContext(ctx, "SELECT address FROM network_address_fences WHERE bridge=? AND family=? AND address=?", b.Bridge, f.family, f.ip).Scan(&old)
		if errors.Is(e, sql.ErrNoRows) {
			_, e = tx.ExecContext(ctx, `INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,operation_id,observed_at)
 VALUES(?,?,?,?,?,?,'RESERVED',?,?)`, b.Bridge, f.family, f.ip, t.Layer, t.UUID, t.Incarnation, t.ID, now)
		}
		if e != nil {
			return e
		}
	}
	return nil
}

// ReadOriginalCreatePin authenticates the current stored tuple against the
// exact original reservation; it is never a native observation.
func (s *Store) ReadOriginalCreatePin(ctx context.Context, t RecoveryTarget) (VerifiedCreateBinding, error) {
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return VerifiedCreateBinding{}, e
	}
	defer tx.Rollback()
	current, e := readRecoveryTarget(ctx, tx, t.Layer, t.ID)
	if e != nil {
		return VerifiedCreateBinding{}, e
	}
	if current != t || t.Action != "box_create" || t.State != "SUCCEEDED" {
		return VerifiedCreateBinding{}, ErrResourceLocked
	}
	pin, status, e := readLifecycleRecoveryPin(ctx, tx, t)
	if e != nil {
		return VerifiedCreateBinding{}, e
	}
	if status != "PINNED" {
		return VerifiedCreateBinding{}, ErrResourceLocked
	}
	if e = verifyIPv4RetirementHistory(ctx, tx, t.Layer, pin.Bridge, pin.ACL); e != nil {
		return VerifiedCreateBinding{}, e
	}
	return pin, nil
}
