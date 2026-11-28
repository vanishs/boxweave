package state

import (
	"context"
	"database/sql"
	"slices"
	"strings"

	"github.com/vanishs/boxweave/internal/networklimits"
	"github.com/vanishs/boxweave/internal/peerscope"
)

// ScopedNetworkAuditBinding is a read-only durable claim for ONE separately
// owned per-Box ACL. It is NOT a native ACL/lease proof or an approval to
// migrate existing shared-ACL Boxes. Install Root must independently prove
// exact ACL UsedBy, Incus UUID/NIC, sysfs ports, nft and host route scope.
type ScopedNetworkAuditBinding struct {
	NetworkAuditBinding
	ACL string
}

// ScopedNetworkAuditLedger is an atomic schema-7 snapshot including retained identities.
type ScopedNetworkAuditLedger struct {
	Project, Bridge string
	Boxes           []ScopedNetworkAuditBinding
	Retired         []ScopedNetworkAuditRetiredBinding
}

// ReadCompleteScopedNetworkAuditLedger requires a complete, nonempty Project
// with exact derived ACLs and retained Create/Delete lineage. No historical
// lease is discarded. Older schemas are rejected when opening the ledger.
func (s *Store) ReadCompleteScopedNetworkAuditLedger(ctx context.Context, project, bridge string) (ScopedNetworkAuditLedger, error) {
	var zero ScopedNetworkAuditLedger
	if s == nil || s.db == nil || !strings.HasPrefix(project, "user-") ||
		!auditName(project) || !auditName(bridge) {
		return zero, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	out, err := readCompleteScopedNetworkAuditLedgerTx(ctx, tx, project, bridge)
	if err != nil {
		return zero, err
	}
	if err := tx.Commit(); err != nil {
		return zero, err
	}
	return out, nil
}

// readCompleteScopedNetworkAuditLedgerTx preserves the entire inventory,
// including MAC and two address fences per Box, under ONE SQLite snapshot.
// A separate scoped Peer staging transaction uses this exact helper to
// atomically prove the entire Project before committing a pending direction.
// ReadScopedRetirementLedger permits an empty current inventory but retains
// all retired identities and their exact durable Create/Delete lineages.
func (s *Store) ReadScopedRetirementLedger(ctx context.Context, project, bridge string) (ScopedNetworkAuditLedger, error) {
	var zero ScopedNetworkAuditLedger
	if s == nil || s.db == nil || !s.IsNetworkAuditReadOnly() {
		return zero, ErrInvalid
	}
	return s.readScopedRetirementLedger(ctx, project, bridge)
}

// ReadScopedColdRetirementLedger uses the same strict settled inventory
// verifier on an exclusive private copy. No SQLite connection is made to
// the original stopped controller's DB/WAL/SHM.
func (s *DiagnosticSnapshot) ReadScopedColdRetirementLedger(ctx context.Context, project, bridge string) (ScopedNetworkAuditLedger, error) {
	if s == nil || s.store == nil || s.Capture != "SQLITE_EXCLUSIVE_COLD_COPY" {
		return ScopedNetworkAuditLedger{}, ErrDiagnosticBusy
	}
	return s.store.readScopedRetirementLedger(ctx, project, bridge)
}

func (s *Store) readScopedRetirementLedger(ctx context.Context, project, bridge string) (ScopedNetworkAuditLedger, error) {
	var zero ScopedNetworkAuditLedger
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	ledger, err := readCompleteScopedNetworkAuditLedgerModeTx(ctx, tx, project, bridge, true)
	if err != nil {
		return zero, err
	}
	if len(ledger.Boxes) != 0 {
		return zero, ErrResourceLocked
	}
	// A retirement is not an ordinary live network audit. Require all
	// resources and original operations to have settled. Historical retained
	// identity rows have already been independently validated above. Resource
	// claim rows remain as durable historical lineage even after settlement;
	// reject live, orphaned and foreign claims, not exact terminal history.
	for _, q := range []string{
		"SELECT count(*) FROM owned_boxes WHERE project!=? OR state!='DELETED'",
		"SELECT count(*) FROM operations WHERE layer!=? OR state NOT IN ('SUCCEEDED','FAILED','CANCELLED','TIMED_OUT')",
		"SELECT count(*) FROM network_bindings WHERE project!=? OR state!='RELEASED'",
		"SELECT count(*) FROM network_address_fences WHERE project!=? OR state!='RELEASED'",
		"SELECT count(*) FROM peer_grants WHERE project!=? OR status!='REVOKED'",
		"SELECT count(*) FROM forward_claims WHERE project!=? OR state!='REVOKED'",
		"SELECT count(*) FROM operation_resource_claims c LEFT JOIN operations o ON o.id=c.operation_id WHERE c.layer!=? OR o.id IS NULL OR o.layer!=c.layer OR o.state NOT IN ('SUCCEEDED','FAILED','CANCELLED','TIMED_OUT')",
	} {
		var n int
		if err = tx.QueryRowContext(ctx, q, project).Scan(&n); err != nil {
			return zero, err
		}
		if n != 0 {
			return zero, ErrResourceLocked
		}
	}
	if err = checkScopedRetiredForwardHistory(ctx, tx, project); err != nil {
		return zero, err
	}
	p, err := readScopedTerminalPeerPolicyTx(ctx, tx, project, bridge)
	if err != nil || len(p.Active) != 0 {
		return zero, ErrResourceLocked
	}
	return ledger, tx.Commit()
}

func readCompleteScopedNetworkAuditLedgerTx(ctx context.Context, tx *sql.Tx, project, bridge string) (ScopedNetworkAuditLedger, error) {
	return readCompleteScopedNetworkAuditLedgerModeTx(ctx, tx, project, bridge, false)
}

func readCompleteScopedNetworkAuditLedgerModeTx(ctx context.Context, tx *sql.Tx, project, bridge string, allowEmpty bool) (ScopedNetworkAuditLedger, error) {
	var zero ScopedNetworkAuditLedger
	if tx == nil || !strings.HasPrefix(project, "user-") || !auditName(project) || !auditName(bridge) {
		return zero, ErrInvalid
	}

	// A Scoped ledger may never borrow Legacy retirement or compaction
	// history. The transitional guard refuses any mixed state before
	// ordinary live audit as well as retirement.

	rows, err := tx.QueryContext(ctx,
		"SELECT uuid,name,incarnation FROM owned_boxes WHERE project=? AND state='OWNED' ORDER BY uuid", project)
	if err != nil {
		return zero, err
	}
	claimed := map[string]ScopedNetworkAuditBinding{}
	names := map[string]bool{}
	for rows.Next() {
		var b ScopedNetworkAuditBinding
		if err = rows.Scan(&b.UUID, &b.Name, &b.Incarnation); err != nil {
			break
		}
		if len(claimed) >= networklimits.MaxProjectBoxes ||
			!auditName(b.Name) || names[b.Name] ||
			b.UUID == "" || len(b.UUID) > 128 ||
			b.Incarnation == "" || len(b.Incarnation) > 128 ||
			claimed[b.UUID].UUID != "" {
			err = ErrResourceLocked
			break
		}
		var e error
		b.ACL, e = peerscope.ACLName(project, bridge, b.UUID)
		if e != nil {
			err = ErrResourceLocked
			break
		}
		claimed[b.UUID] = b
		names[b.Name] = true
	}
	if e := rows.Err(); err == nil {
		err = e
	}
	if e := rows.Close(); err == nil {
		err = e
	}
	if err != nil {
		return zero, err
	}
	if len(claimed) == 0 && !allowEmpty {
		return zero, ErrResourceLocked
	}

	historical, err := scopedRetiredOwners(ctx, tx, project, bridge)
	if err != nil {
		return zero, err
	}
	for uuid := range historical {
		if _, exists := claimed[uuid]; exists {
			return zero, ErrResourceLocked
		}
	}
	// Read every Project binding, including unexpected historical rows.
	// No disallowed row can disappear by filtering for the target ACL.
	rows, err = tx.QueryContext(ctx,
		"SELECT box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state FROM network_bindings WHERE project=?", project)
	if err != nil {
		return zero, err
	}
	found := map[string]bool{}
	allIPs := map[string]bool{}
	allMACs := map[string]bool{}
	for rows.Next() {
		var uuid, inc, nic, nw, acl, mac, v4, v6, status string
		if err = rows.Scan(&uuid, &inc, &nic, &nw, &acl, &mac, &v4, &v6, &status); err != nil {
			break
		}
		box, ok := claimed[uuid]
		old, retired := historical[uuid]
		expectedState := "PINNED"
		if retired {
			box = old.ScopedNetworkAuditBinding
			ok = true
			expectedState = "RELEASED"
		}
		if !ok || found[uuid] || inc != box.Incarnation ||
			nic != "eth0" || nw != bridge || acl != box.ACL ||
			status != expectedState || !auditMAC(mac) ||
			!auditIP(v4, false) || !auditIP(v6, true) ||
			allIPs[v4] || allIPs[v6] || allMACs[mac] {
			err = ErrResourceLocked
			break
		}
		found[uuid], allIPs[v4], allIPs[v6], allMACs[mac] = true, true, true, true
		box.MAC, box.IPv4, box.IPv6 = mac, v4, v6
		if retired {
			old.ScopedNetworkAuditBinding = box
			historical[uuid] = old
		} else {
			claimed[uuid] = box
		}
	}
	if e := rows.Err(); err == nil {
		err = e
	}
	if e := rows.Close(); err == nil {
		err = e
	}
	if err != nil {
		return zero, err
	}
	if len(found) != len(claimed)+len(historical) {
		return zero, ErrResourceLocked
	}

	var foreign int
	if err = tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM network_bindings WHERE bridge=? AND project!=?",
		bridge, project).Scan(&foreign); err != nil {
		return zero, err
	}
	if foreign != 0 {
		return zero, ErrResourceLocked
	}

	// Every active dual-stack lease must be reserved exactly once, and
	// no foreign/stale/released address may escape the full-bridge view.
	rows, err = tx.QueryContext(ctx,
		"SELECT family,address,project,box_uuid,incarnation,state,COALESCE(operation_id,'') FROM network_address_fences WHERE bridge=?", bridge)
	if err != nil {
		return zero, err
	}
	seen := map[string]bool{}
	for rows.Next() {
		var family, addr, p, uuid, inc, status, op string
		if err = rows.Scan(&family, &addr, &p, &uuid, &inc, &status, &op); err != nil {
			break
		}
		b, ok := claimed[uuid]
		wantState := "RESERVED"
		if old, retired := historical[uuid]; retired {
			b = old.ScopedNetworkAuditBinding
			ok = op == old.DeleteID
			wantState = "RELEASED"
		}
		key := uuid + "|" + family
		if !ok || p != project || inc != b.Incarnation ||
			status != wantState || seen[key] ||
			!((family == "ipv4" && addr == b.IPv4) ||
				(family == "ipv6" && addr == b.IPv6)) {
			err = ErrResourceLocked
			break
		}
		seen[key] = true
	}
	if e := rows.Err(); err == nil {
		err = e
	}
	if e := rows.Close(); err == nil {
		err = e
	}
	if err != nil {
		return zero, err
	}
	// Project rows on a different bridge are not covered by the bridge scan.
	// Refuse them rather than silently accepting an extra historical lease.
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM network_address_fences WHERE project=? AND bridge!=?", project, bridge).Scan(&foreign); err != nil {
		return zero, err
	}
	if foreign != 0 {
		return zero, ErrResourceLocked
	}
	if len(seen) != 2*(len(claimed)+len(historical)) {
		return zero, ErrResourceLocked
	}

	result := ScopedNetworkAuditLedger{
		Project: project, Bridge: bridge, Boxes: make([]ScopedNetworkAuditBinding, 0, len(claimed)),
	}
	for _, old := range historical {
		verified, e := scopedRetiredOriginals(ctx, tx, project, bridge, old)
		if e != nil {
			return zero, e
		}
		result.Retired = append(result.Retired, verified)
	}
	slices.SortFunc(result.Retired, func(a, b ScopedNetworkAuditRetiredBinding) int { return strings.Compare(a.UUID, b.UUID) })
	for _, box := range claimed {
		result.Boxes = append(result.Boxes, box)
	}
	slices.SortFunc(result.Boxes, func(a, b ScopedNetworkAuditBinding) int {
		return strings.Compare(a.UUID, b.UUID)
	})
	return result, nil
}
