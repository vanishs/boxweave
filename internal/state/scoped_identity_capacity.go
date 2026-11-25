package state

import (
	"context"
	"database/sql"
	"errors"
	"github.com/vanishs/boxweave/internal/networklimits"
)

var ErrScopedIdentityCapacity = errors.New("SCOPED_RETIRED_IDENTITY_CAPACITY: future retirement reserve exhausted")

// Reserve a retirement record for each owned or unresolved Create. Otherwise
// a large bridge could allow more live identities than the eventual retired
// audit can represent, stranding a valid Delete. Never refuse Delete here.
type ScopedIdentityCapacity struct {
	Limit                  int    `json:"limit"`
	Retired                int    `json:"retired"`
	Owned                  int    `json:"owned"`
	PendingCreates         int    `json:"pending_creates"`
	RemainingNewIdentities int    `json:"remaining_new_identities"`
	Level                  string `json:"level"`
}

func readScopedIdentityCapacity(ctx context.Context, q capacityQuerier, project string) (ScopedIdentityCapacity, error) {
	c := ScopedIdentityCapacity{Limit: networklimits.ScopedRetiredIdentityLimit}
	p, e := readProjectBoxCapacity(ctx, q, project)
	if e != nil {
		return c, e
	}
	c.Owned, c.PendingCreates = p.Owned, p.PendingCreates
	if e = q.QueryRowContext(ctx, "SELECT count(*) FROM owned_boxes WHERE project=? AND state='DELETED'", project).Scan(&c.Retired); e != nil {
		return c, e
	}
	c.RemainingNewIdentities = max(0, c.Limit-c.Retired-c.Owned-c.PendingCreates)
	c.Level = networklimits.CapacityLevel(c.Limit, c.RemainingNewIdentities)
	if c.Retired+c.Owned+c.PendingCreates > c.Limit {
		c.Level = "OVER_LIMIT"
	}
	return c, nil
}
func (s *Store) ReadScopedIdentityCapacity(ctx context.Context, project string) (ScopedIdentityCapacity, error) {
	if !auditName(project) || project == "default" {
		return ScopedIdentityCapacity{}, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return ScopedIdentityCapacity{}, e
	}
	defer tx.Rollback()
	c, e := readScopedIdentityCapacity(ctx, tx, project)
	if e != nil {
		return c, e
	}
	return c, tx.Commit()
}
