package state

import (
	"context"
	"database/sql"
)

// Legacy-only maintenance reads. The labels remain in the schema-6 digest
// for Scoped until the exact fresh-schema transition invalidates old plans.
const legacyMaintenanceArchiveHistory = `SELECT * FROM network_ipv4_retirements WHERE project=? ORDER BY box_uuid,incarnation`
const legacyMaintenanceQueueHistory = `SELECT * FROM network_ipv4_retirement_queue WHERE project=? ORDER BY delete_operation_id`
const legacyMaintenanceCompactionHistory = `SELECT * FROM network_ipv4_compactions WHERE project=? ORDER BY delete_operation_id`

func legacyMaintenanceForeignQueries() []string {
	return []string{
		"SELECT count(*) FROM network_ipv4_retirements WHERE project!=?",
		"SELECT count(*) FROM network_ipv4_retirement_queue WHERE project!=?",
		"SELECT count(*) FROM network_ipv4_compactions WHERE project!=?",
	}
}

func readLegacyMaintenancePending(ctx context.Context, tx *sql.Tx, project string, o *MaintenanceLedger) error {
	for _, q := range []struct {
		query string
		dst   *[]string
	}{
		{`SELECT delete_operation_id FROM network_ipv4_retirement_queue WHERE project=? ORDER BY delete_operation_id LIMIT ?`, &o.LegacyPending},
		{`SELECT delete_operation_id FROM network_ipv4_compactions WHERE project=? ORDER BY delete_operation_id LIMIT ?`, &o.LegacyCompactions},
	} {
		rows, e := tx.QueryContext(ctx, q.query, project, maintenanceRowLimit+1)
		if e != nil {
			return e
		}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				break
			}
			if len(*q.dst) >= maintenanceRowLimit || !validHexID(id) {
				e = ErrDiagnosticInvalid
				break
			}
			*q.dst = append(*q.dst, id)
		}
		if e == nil {
			e = rows.Err()
		}
		if ce := rows.Close(); e == nil {
			e = ce
		}
		if e != nil {
			return e
		}
	}
	return nil
}
