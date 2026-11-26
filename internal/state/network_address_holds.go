package state

import (
	"context"
	"database/sql"
	"net/netip"
	"strings"

	"github.com/vanishs/boxweave/internal/networklimits"
)

// ReadCreateAddressHolds loads the complete durable exclusion set once per
// Create rather than rescanning all historical rows for each IPv4 candidate.
// In particular, all RELEASED/UNKNOWN reservations remain blocked. This
// snapshot only skips candidates; it never authorizes a new reservation.
func (s *Store) ReadCreateAddressHolds(ctx context.Context, project, bridge string) (networklimits.AddressHolds, error) {
	return s.readCreateAddressHolds(ctx, project, bridge, false)
}

// ReadScopedCreateAddressHolds never credits any Legacy retirement as free
// capacity. During consolidation the old tables still exist; refuse mixed
// history before a Scoped reservation or capacity observation.
func (s *Store) ReadScopedCreateAddressHolds(ctx context.Context, project, bridge string) (networklimits.AddressHolds, error) {
	return s.readCreateAddressHolds(ctx, project, bridge, true)
}

func (s *Store) readCreateAddressHolds(ctx context.Context, project, bridge string, scoped bool) (networklimits.AddressHolds, error) {
	zero := networklimits.AddressHolds{}
	if s == nil || s.db == nil || !auditName(project) || project == "default" || !auditName(bridge) {
		return zero, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return zero, e
	}
	defer tx.Rollback()
	out, e := readCreateAddressHoldsModeTx(ctx, tx, project, bridge, scoped)
	if e != nil {
		return zero, e
	}
	return out, tx.Commit()
}

// Transitional refusal only. These legacy tables are not a source of
// Scoped allocations or capacity. Remove this check only together with
// the exact schema-version transition and old-state refusal.
func requireScopedNoLegacyRetirementTx(ctx context.Context, tx *sql.Tx) error {
	for _, q := range []string{
		"SELECT EXISTS(SELECT 1 FROM network_ipv4_retirements)",
		"SELECT EXISTS(SELECT 1 FROM network_ipv4_retirement_queue)",
		"SELECT EXISTS(SELECT 1 FROM network_ipv4_compactions)",
	} {
		var present bool
		if e := tx.QueryRowContext(ctx, q).Scan(&present); e != nil {
			return e
		}
		if present {
			return ErrResourceLocked
		}
	}
	return nil
}

func readCreateAddressHoldsModeTx(ctx context.Context, tx *sql.Tx, project, bridge string, scoped bool) (networklimits.AddressHolds, error) {
	zero := networklimits.AddressHolds{}
	var e error
	if scoped {
		e = requireScopedNoLegacyRetirementTx(ctx, tx)
	} else {
		e = verifyIPv4RetirementHistory(ctx, tx, project, bridge, "")
	}
	if e != nil {
		return zero, e
	}
	holds := networklimits.NewAddressHolds()
	addIP := func(raw string, v6 bool) error {
		if !auditIP(raw, v6) {
			return ErrResourceLocked
		}
		a, _ := netip.ParseAddr(raw)
		holds.IPs[a] = true
		return nil
	}
	addMAC := func(mac string) error {
		if !auditMAC(mac) {
			return ErrResourceLocked
		}
		holds.MACs[mac] = true
		return nil
	}
	// The supported native IPv4 pool is at most /16. 131072 distinct
	// addresses and 65536 MACs bound the usual retained dual-stack history.
	// A corrupted or unsupported ledger is never silently truncated.
	bound := func() error {
		if len(holds.IPs) > 131072 || len(holds.MACs) > 65536 {
			return ErrResourceLocked
		}
		return nil
	}
	bindingsQuery := "SELECT project,ipv4,ipv6,mac,state FROM network_bindings n WHERE bridge=?"
	bindingsArgs := []any{bridge}
	if !scoped {
		bindingsQuery += " AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.project=n.project AND r.box_uuid=n.box_uuid AND r.incarnation=n.incarnation)"
		bindingsArgs = append(bindingsArgs, project)
	}
	rows, e := tx.QueryContext(ctx, bindingsQuery, bindingsArgs...)
	if e != nil {
		return zero, e
	}
	for rows.Next() {
		var p, v4, v6, mac, state string
		if e = rows.Scan(&p, &v4, &v6, &mac, &state); e != nil {
			break
		}
		if p != project && state != "RELEASED" {
			e = ErrResourceLocked
			break
		}
		// Malformed legacy/OBSERVED rows are not a free address.
		if e = addIP(v4, false); e != nil {
			break
		}
		if e = addIP(v6, true); e != nil {
			break
		}
		if e = addMAC(mac); e != nil {
			break
		}
		if e = bound(); e != nil {
			break
		}
	}
	if er := rows.Err(); e == nil {
		e = er
	}
	if er := rows.Close(); e == nil {
		e = er
	}
	if e != nil {
		return zero, e
	}
	rows, e = tx.QueryContext(ctx, "SELECT project,family,address,state FROM network_address_fences WHERE bridge=?", bridge)
	if e != nil {
		return zero, e
	}
	for rows.Next() {
		var p, fam, ip, state string
		if e = rows.Scan(&p, &fam, &ip, &state); e != nil {
			break
		}
		if p != project && state != "RELEASED" || (fam != "ipv4" && fam != "ipv6") {
			e = ErrResourceLocked
			break
		}
		if e = addIP(ip, fam == "ipv6"); e != nil {
			break
		}
		if e = bound(); e != nil {
			break
		}
	}
	if er := rows.Err(); e == nil {
		e = er
	}
	if er := rows.Close(); e == nil {
		e = er
	}
	if e != nil {
		return zero, e
	}
	prefix := "create-pin:" + bridge + ":"
	claimsQuery := "SELECT resource_key FROM operation_resource_claims c WHERE substr(resource_key,1,?)=?"
	claimsArgs := []any{len(prefix), prefix}
	if !scoped {
		claimsQuery += " AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.bridge=? AND r.create_operation_id=c.operation_id AND r.project=c.layer)"
		claimsArgs = append(claimsArgs, project, bridge)
	}
	rows, e = tx.QueryContext(ctx, claimsQuery, claimsArgs...)
	if e != nil {
		return zero, e
	}
	for rows.Next() {
		var key string
		if e = rows.Scan(&key); e != nil {
			break
		}
		kind, raw, ok := strings.Cut(strings.TrimPrefix(key, prefix), ":")
		if !ok {
			e = ErrResourceLocked
			break
		}
		switch kind {
		case "v4":
			e = addIP(raw, false)
		case "v6":
			e = addIP(raw, true)
		case "mac":
			e = addMAC(raw)
		case "acl":
			if !auditName(raw) {
				e = ErrResourceLocked
			}
		default:
			e = ErrResourceLocked
		}
		if e != nil {
			break
		}
		if e = bound(); e != nil {
			break
		}
	}
	if er := rows.Err(); e == nil {
		e = er
	}
	if er := rows.Close(); e == nil {
		e = er
	}
	if e != nil {
		return zero, e
	}
	return holds, nil
}
