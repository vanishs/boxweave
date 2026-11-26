package state

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
)

// ScopedDrainActor has no issued bearer token. The exact private root
// retirement receipt chooses its identity and expiration. It is usable
// only by the restricted in-process cleanup worker.
type ScopedDrainActor struct {
	ID      string `json:"id"`
	Expires int64  `json:"expires"`
}

func (s *Store) EnsureScopedDrainActor(ctx context.Context, a ScopedDrainActor) error {
	if !validHexID(a.ID) || a.Expires <= time.Now().Unix() || a.Expires > time.Now().Add(365*24*time.Hour).Unix() {
		return ErrInvalid
	}
	h := sha256.Sum256([]byte("boxweave-scoped-drain-no-issued-bearer-v1|" + a.ID))
	proof := hex.EncodeToString(h[:])
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var existing string
	var expiry int64
	var revoked int
	e = tx.QueryRowContext(ctx, "SELECT token_digest,expires_at,revoked FROM credentials WHERE id=?", a.ID).Scan(&existing, &expiry, &revoked)
	if errors.Is(e, sql.ErrNoRows) {
		_, e = tx.ExecContext(ctx, "INSERT INTO credentials(id,token_digest,expires_at,created_at) VALUES(?,?,?,?)", a.ID, proof, a.Expires, time.Now().Unix())
	} else if e == nil {
		if existing != proof || expiry != a.Expires || revoked != 0 {
			return ErrConflict
		}
		var n int
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM oauth_access WHERE token_id=?", a.ID).Scan(&n); e != nil || n != 0 {
			return ErrConflict
		}
	}
	if e != nil {
		return e
	}
	return tx.Commit()
}

// Exact original lookup, including UNKNOWN: never create a second
// operation to substitute for an uncertain first dispatch.
func (s *Store) ScopedDrainOriginal(ctx context.Context, project, actor, key string) (Receipt, error) {
	if !SafeDiagnosticProject(project) || !validHexID(actor) || len(key) < 16 || len(key) > 128 {
		return Receipt{}, ErrInvalid
	}
	var id string
	e := s.db.QueryRowContext(ctx, "SELECT id FROM operations WHERE layer=? AND actor_id=? AND token_id=? AND key_digest=?", project, actor, actor, digest(key)).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		return Receipt{}, ErrNotFound
	}
	if e != nil {
		return Receipt{}, e
	}
	return s.Get(ctx, id)
}

func (s *Store) ReadScopedDrainWitness(ctx context.Context, project string) (MaintenanceLedger, error) {
	if !SafeDiagnosticProject(project) {
		return MaintenanceLedger{}, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return MaintenanceLedger{}, e
	}
	defer tx.Rollback()
	l, e := readMaintenanceLedger(ctx, tx, project, true)
	if e != nil {
		return MaintenanceLedger{}, e
	}
	if l.ForeignRows != 0 || l.OrphanBindings != 0 || l.OrphanFences != 0 || len(l.Originals) != 0 || len(l.LegacyPending) != 0 || len(l.LegacyCompactions) != 0 {
		return MaintenanceLedger{}, ErrResourceLocked
	}
	return l, tx.Commit()
}
