package state

import (
	"context"
	"database/sql"
	"encoding/json"
)

// Classify only the original's bounded ownership, never its outcome.
// A power operation with one exact Box claim has no authorization to
// create a new instance, alter Peer policy, or change a host listener.
// UNKNOWN remains UNKNOWN. Native containment and quiescence are
// independently required by the offline retirement coordinator.
func maintenanceRetirementScope(ctx context.Context, tx *sql.Tx, project string, boxes []MaintenanceBox, o *MaintenanceOriginal) error {
	o.RetirementScope = "UNPROVEN"
	if o.State != "UNKNOWN" || (o.Action != "box_stop" && o.Action != "box_start") || o.BoxUUID == "" || o.BoxUUID != o.Incarnation || o.Claims != 1 || o.ClaimProtection != "RETAINED" {
		return nil
	}
	var name string
	for _, b := range boxes {
		if b.UUID == o.BoxUUID && b.Incarnation == o.Incarnation && b.CreateProof && b.BindingState == "PINNED" {
			name = b.Name
		}
	}
	if name == "" {
		return nil
	}
	var raw []byte
	var key string
	e := tx.QueryRowContext(ctx, `SELECT o.safe_intent,c.resource_key FROM operations o JOIN operation_resource_claims c ON c.operation_id=o.id AND c.layer=o.layer WHERE o.id=? AND o.layer=?`, o.ID, project).Scan(&raw, &key)
	if e != nil {
		return e
	}
	want, _ := json.Marshal(struct {
		BoxID string `json:"box_id"`
	}{o.BoxUUID})
	if string(raw) == string(want) && key == "box:"+name {
		o.RetirementScope = "EXACT_OWNED_POWER_ORIGINAL"
	}
	return nil
}

// An original probe must not open SQLite on the live source as root;
// a cold WAL reader could create source sidecars owned by root.
func (s *DiagnosticSnapshot) ScopedDrainOriginal(ctx context.Context, project, actor, key string) (Receipt, error) {
	return s.store.ScopedDrainOriginal(ctx, project, actor, key)
}
