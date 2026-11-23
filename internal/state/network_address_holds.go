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
	zero := networklimits.AddressHolds{}
	if s == nil || s.db == nil || !auditName(project) || project == "default" || !auditName(bridge) {
		return zero, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return zero, e
	}
	defer tx.Rollback()
	if e = verifyIPv4RetirementHistory(ctx, tx, project, bridge, ""); e != nil {
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
	rows, e := tx.QueryContext(ctx, "SELECT project,ipv4,ipv6,mac,state FROM network_bindings n WHERE bridge=? AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.project=n.project AND r.box_uuid=n.box_uuid AND r.incarnation=n.incarnation)", bridge, project)
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
	rows, e = tx.QueryContext(ctx, "SELECT resource_key FROM operation_resource_claims c WHERE substr(resource_key,1,?)=? AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.bridge=? AND r.create_operation_id=c.operation_id AND r.project=c.layer)", len(prefix), prefix, project, bridge)
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
	if e = tx.Commit(); e != nil {
		return zero, e
	}
	return holds, nil
}
