package state

import (
	"context"
	"database/sql"
	"errors"

	"github.com/vanishs/boxweave/internal/networklimits"
)

// ErrProjectCapacity is a refusal before admission, not a native failure.
var ErrProjectCapacity = errors.New("PROJECT_BOX_CAPACITY_REACHED: owned and unresolved Create slots exhausted")

// ProjectBoxCapacity describes the ledger limit, not available native
// addresses, storage, CPU or RAM. Unknown Creates occupy a slot even after
// a manual claim release, unless their exact identity is already counted.
type ProjectBoxCapacity struct {
	Limit          int `json:"limit"`
	Owned          int `json:"owned"`
	PendingCreates int `json:"pending_creates"`
	Remaining      int `json:"remaining"`
}

type capacityQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readProjectBoxCapacity(ctx context.Context, q capacityQuerier, project string) (ProjectBoxCapacity, error) {
	var out ProjectBoxCapacity
	if project == "" || project == "default" {
		return out, ErrInvalid
	}
	out.Limit = networklimits.MaxProjectBoxes
	err := q.QueryRowContext(ctx, `SELECT
 (SELECT COUNT(*) FROM owned_boxes WHERE project=? AND state='OWNED'),
 (SELECT COUNT(*) FROM operations o WHERE o.layer=? AND o.action='box_create'
 AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')
 AND NOT EXISTS (SELECT 1 FROM owned_boxes b WHERE b.project=o.layer AND b.uuid=o.box_uuid
 AND b.incarnation=o.incarnation AND b.state='OWNED'))`, project, project).Scan(&out.Owned, &out.PendingCreates)
	if err != nil {
		return ProjectBoxCapacity{}, err
	}
	out.Remaining = max(0, out.Limit-out.Owned-out.PendingCreates)
	return out, nil
}

// ReadProjectBoxCapacity provides a single statement ledger snapshot.
func (s *Store) ReadProjectBoxCapacity(ctx context.Context, project string) (ProjectBoxCapacity, error) {
	return readProjectBoxCapacity(ctx, s.db, project)
}
