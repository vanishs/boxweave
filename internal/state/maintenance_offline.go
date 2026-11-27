package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
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

// ScopedOfflineArchiveLedger is an observation-only witness for exceptional
// full retirement, never a business-operation admission or native permit.
type ScopedOfflineArchiveLedger struct {
	Inventory    ScopedNetworkAuditLedger
	Current      []NativeNetworkBox
	Originals    []MaintenanceOriginal
	Policy       ScopedTerminalPeerPolicy
	LedgerSHA256 string
}

// ReadScopedOfflineArchiveLedger does not relax the settled retirement/drain
// readers. Native ownership, quiescence, archive and approval are separate.
func (s *DiagnosticSnapshot) ReadScopedOfflineArchiveLedger(ctx context.Context, project, bridge string) (ScopedOfflineArchiveLedger, error) {
	var zero ScopedOfflineArchiveLedger
	if s == nil || s.store == nil || s.store.db == nil || !SafeDiagnosticProject(project) || !auditName(bridge) {
		return zero, ErrInvalid
	}
	tx, err := s.store.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	l, err := readMaintenanceLedger(ctx, tx, project)
	if err != nil {
		return zero, err
	}
	if len(l.Originals) == 0 || l.ForeignRows != 0 || l.OrphanBindings != 0 || l.OrphanFences != 0 ||
		len(l.LegacyPending) != 0 || len(l.LegacyCompactions) != 0 || l.Network.Unsettled != len(l.Originals) ||
		l.NonterminalRecordedClaims != len(l.Originals) {
		return zero, ErrResourceLocked
	}
	for _, q := range []string{
		"SELECT count(*) FROM owned_boxes WHERE project=? AND state NOT IN ('OWNED','DELETED')",
		"SELECT count(*) FROM operations WHERE layer=? AND state NOT IN ('SUCCEEDED','FAILED','CANCELLED','TIMED_OUT','UNKNOWN')",
		"SELECT count(*) FROM operation_resource_claims c LEFT JOIN operations o ON o.id=c.operation_id WHERE c.layer=? AND (o.id IS NULL OR o.layer!=c.layer)",
	} {
		var n int
		if err = tx.QueryRowContext(ctx, q, project).Scan(&n); err != nil {
			return zero, err
		}
		if n != 0 {
			return zero, ErrResourceLocked
		}
	}
	if err = checkOfflinePowerOriginals(ctx, tx, project, l); err != nil {
		return zero, err
	}
	inv, err := readCompleteScopedNetworkAuditLedgerModeTx(ctx, tx, project, bridge, false)
	if err != nil {
		return zero, err
	}
	if err = checkOfflineCurrentInventory(inv, l, bridge); err != nil {
		return zero, err
	}
	policy, err := readScopedTerminalPeerPolicyTx(ctx, tx, project, bridge)
	if err != nil {
		return zero, err
	}
	if len(policy.Active) != 0 {
		return zero, ErrResourceLocked
	}
	if err = checkScopedOfflineForwardHistory(ctx, tx, project); err != nil {
		return zero, err
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return ScopedOfflineArchiveLedger{inv, l.Network.Boxes, l.Originals, policy, l.ObservationDigest()}, nil
}

func checkOfflinePowerOriginals(ctx context.Context, tx *sql.Tx, project string, l MaintenanceLedger) error {
	seen := map[string]bool{}
	for _, o := range l.Originals {
		if o.State != "UNKNOWN" || o.Action != "box_stop" || o.RetirementScope != "EXACT_OWNED_POWER_ORIGINAL" ||
			o.ClaimProtection != "RETAINED" || o.Claims != 1 || !o.NativeReceipt || o.BoxUUID == "" ||
			o.BoxUUID != o.Incarnation || seen[o.BoxUUID] {
			return ErrResourceLocked
		}
		var nativeID, resource string
		var duplicate int
		if err := tx.QueryRowContext(ctx, "SELECT native_operation_id,resource_key FROM operations WHERE id=? AND layer=?", o.ID, project).Scan(&nativeID, &resource); err != nil {
			return err
		}
		if strings.TrimSpace(nativeID) != nativeID || nativeID == "" || len(nativeID) > 128 {
			return ErrResourceLocked
		}
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM operations WHERE native_operation_id=?", nativeID).Scan(&duplicate); err != nil {
			return err
		}
		if duplicate != 1 {
			return ErrResourceLocked
		}
		matched := false
		for _, b := range l.Boxes {
			if b.UUID == o.BoxUUID && resource == "box:"+b.Name {
				matched = true
			}
		}
		if !matched {
			return ErrResourceLocked
		}
		seen[o.BoxUUID] = true
	}
	return nil
}

func checkOfflineCurrentInventory(inv ScopedNetworkAuditLedger, l MaintenanceLedger, bridge string) error {
	if len(inv.Boxes) != len(l.Network.Boxes) {
		return ErrResourceLocked
	}
	for i, b := range inv.Boxes {
		n := l.Network.Boxes[i]
		if n.UUID != b.UUID || n.Incarnation != b.Incarnation || n.Name != b.Name || n.OriginalResource != "box:"+b.Name ||
			!n.OriginalProof || n.OriginalCount != 1 || n.BindingCount != 1 || n.BindingState != "PINNED" ||
			n.Bridge != bridge || n.ACL != b.ACL || n.MAC != b.MAC || n.IPv4 != b.IPv4 || n.IPv6 != b.IPv6 ||
			n.FenceCount != 2 || n.FenceIPv4 != 1 || n.FenceIPv6 != 1 {
			return ErrResourceLocked
		}
	}
	return nil
}
