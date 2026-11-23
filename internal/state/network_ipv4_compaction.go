package state

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func requireNoIPv4Compaction(ctx context.Context, tx *sql.Tx, project string) error {
	var n int
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM network_ipv4_compactions WHERE project=?", project).Scan(&n); e != nil {
		return e
	}
	if n != 0 {
		return ErrResourceLocked
	}
	return nil
}

// IPv4CompactionPreview performs ONLY independent root observations. Its
// returned digest is a fixed-scope ACL plan, never a caller-supplied rule.
type IPv4CompactionPreview func(context.Context, IPv4RetirementCandidate) (string, error)

// PrepareIPv4Compaction commits a persistent Project fence only after an
// independently verified read-only preview. It never calls a root mutation.
// Exact retries read the original intent. A lost root write acknowledgement
// cannot bypass the fence because archive and fence release commit together.
func (s *Store) PrepareIPv4Compaction(ctx context.Context, project, original string, preview IPv4CompactionPreview) (bool, error) {
	if s == nil || s.db == nil || s.auditReadOnly || preview == nil {
		return false, ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return false, e
	}
	defer tx.Rollback()
	r, e := tx.ExecContext(ctx, "UPDATE schema_version SET version=version WHERE version=?", SchemaVersion)
	if e != nil {
		return false, e
	}
	if n, e := r.RowsAffected(); e != nil || n != 1 {
		return false, ErrResourceLocked
	}
	c, e := readIPv4RetirementCandidate(ctx, tx, project, original)
	if e != nil {
		return false, e
	}
	if !c.RequiresPeerCompaction {
		return false, nil
	}
	var id, sha string
	e = tx.QueryRowContext(ctx, "SELECT delete_operation_id,plan_sha256 FROM network_ipv4_compactions WHERE project=?", project).Scan(&id, &sha)
	if e == nil {
		if id != original || !validRetirementProofSHA(sha) {
			return false, ErrResourceLocked
		}
		return true, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return false, e
	}
	sha, e = preview(ctx, c)
	if e != nil {
		return false, e
	}
	if !validRetirementProofSHA(sha) || ctx.Err() != nil {
		return false, ErrResourceLocked
	}
	checked, e := readIPv4RetirementCandidate(ctx, tx, project, original)
	if e != nil || checked != c {
		return false, ErrResourceLocked
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO network_ipv4_compactions(project,delete_operation_id,plan_sha256,prepared_at) VALUES(?,?,?,?)", project, original, sha, time.Now().UnixNano()); e != nil {
		return false, e
	}
	if e = tx.Commit(); e != nil {
		return false, e
	}
	return true, nil
}

func (s *Store) ReadIPv4CompactionPlan(ctx context.Context, project, original string) (string, error) {
	if s == nil || s.db == nil || !auditName(project) || !validHexID(original) {
		return "", ErrInvalid
	}
	var plan string
	e := s.db.QueryRowContext(ctx, "SELECT plan_sha256 FROM network_ipv4_compactions WHERE project=? AND delete_operation_id=?", project, original).Scan(&plan)
	if errors.Is(e, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if e != nil || !validRetirementProofSHA(plan) {
		return "", ErrResourceLocked
	}
	return plan, nil
}
