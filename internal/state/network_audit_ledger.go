package state

import (
	"context"
	"database/sql"
	"net"
	"net/netip"
	"slices"
	"strings"
)

// NetworkAuditBinding is a local claim, not native authentication.
type NetworkAuditBinding struct{ Name, UUID, Incarnation, MAC, IPv4, IPv6 string }
type NetworkAuditLedger struct {
	Project, Bridge, ACL string
	Boxes                []NetworkAuditBinding
}

func auditName(s string) bool {
	if len(s) == 0 || len(s) > 100 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
func auditIP(s string, v6 bool) bool {
	ip, e := netip.ParseAddr(s)
	return e == nil && ip.IsGlobalUnicast() && !ip.Is4In6() && ip.Is6() == v6 && ip.String() == s
}
func auditMAC(s string) bool {
	m, e := net.ParseMAC(s)
	return e == nil && len(m) == 6 && m[0]&1 == 0 && m.String() == s && s == strings.ToLower(s)
}

// ReadCompleteNetworkAuditLedger provides a bounded, consistent SQLite
// snapshot, never authorization to change ACL, run root shell or release IPs.
// Existing S1 databases lacking pinned bindings deliberately return failure.
func (s *Store) ReadCompleteNetworkAuditLedger(ctx context.Context, project, bridge, acl string) (NetworkAuditLedger, error) {
	var zero NetworkAuditLedger
	if s == nil || s.db == nil || !auditName(project) || project == "default" || !auditName(bridge) || !auditName(acl) {
		return zero, ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT uuid,name,incarnation FROM owned_boxes WHERE project=? AND state='OWNED' ORDER BY uuid", project)
	if err != nil {
		return zero, err
	}
	bindings := map[string]NetworkAuditBinding{}
	for rows.Next() {
		var b NetworkAuditBinding
		if err = rows.Scan(&b.UUID, &b.Name, &b.Incarnation); err != nil {
			break
		}
		if len(bindings) >= 64 || !auditName(b.Name) || b.UUID == "" || len(b.UUID) > 128 || b.Incarnation == "" || len(b.Incarnation) > 128 || bindings[b.UUID].UUID != "" {
			err = ErrResourceLocked
			break
		}
		bindings[b.UUID] = b
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
	if len(bindings) == 0 {
		return zero, ErrResourceLocked
	}
	historical, err := verifiedDeletedNetworkBindings(ctx, tx, project, bridge, acl)
	if err != nil {
		return zero, ErrResourceLocked
	}
	rows, err = tx.QueryContext(ctx, "SELECT box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state FROM network_bindings WHERE project=?", project)
	if err != nil {
		return zero, err
	}
	found := map[string]bool{}
	macs, ips := map[string]bool{}, map[string]bool{}
	for rows.Next() {
		var uuid, inc, nic, nw, ac, mac, v4, v6, status string
		if err = rows.Scan(&uuid, &inc, &nic, &nw, &ac, &mac, &v4, &v6, &status); err != nil {
			break
		}
		if status == "RELEASED" {
			old, ok := historical[uuid]
			if !ok || old.Incarnation != inc || nic != "eth0" || nw != bridge || ac != acl || old.MAC != mac || old.IPv4 != v4 || old.IPv6 != v6 {
				err = ErrResourceLocked
				break
			}
			continue
		}
		b, ok := bindings[uuid]
		if !ok || found[uuid] || b.Incarnation != inc || nic != "eth0" || nw != bridge || ac != acl || status != "PINNED" || !auditMAC(mac) || !auditIP(v4, false) || !auditIP(v6, true) || macs[mac] || ips[v4] || ips[v6] {
			err = ErrResourceLocked
			break
		}
		found[uuid], macs[mac], ips[v4], ips[v6] = true, true, true, true
		b.MAC, b.IPv4, b.IPv6 = mac, v4, v6
		bindings[uuid] = b
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
	if len(found) != len(bindings) {
		return zero, ErrResourceLocked
	}
	var foreign int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM network_bindings WHERE bridge=? AND project!=? AND state!='RELEASED'", bridge, project).Scan(&foreign); err != nil {
		return zero, err
	}
	if foreign != 0 {
		return zero, ErrResourceLocked
	}
	rows, err = tx.QueryContext(ctx, "SELECT family,address,project,box_uuid,incarnation,state FROM network_address_fences WHERE bridge=?", bridge)
	if err != nil {
		return zero, err
	}
	seen := map[string]bool{}
	for rows.Next() {
		var family, address, p, uuid, inc, status string
		if err = rows.Scan(&family, &address, &p, &uuid, &inc, &status); err != nil {
			break
		}
		if status == "RELEASED" {
			old, ok := historical[uuid]
			if !ok || p != project || inc != old.Incarnation || !((family == "ipv4" && address == old.IPv4) || (family == "ipv6" && address == old.IPv6)) {
				err = ErrResourceLocked
				break
			}
			continue
		}
		b, ok := bindings[uuid]
		key := uuid + "|" + family
		match := (family == "ipv4" && address == b.IPv4) || (family == "ipv6" && address == b.IPv6)
		if !ok || p != project || inc != b.Incarnation || status != "RESERVED" || !match || seen[key] {
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
	if len(seen) != 2*len(bindings) {
		return zero, ErrResourceLocked
	}
	out := NetworkAuditLedger{Project: project, Bridge: bridge, ACL: acl, Boxes: make([]NetworkAuditBinding, 0, len(bindings))}
	for _, b := range bindings {
		out.Boxes = append(out.Boxes, b)
	}
	slices.SortFunc(out.Boxes, func(a, b NetworkAuditBinding) int { return strings.Compare(a.UUID, b.UUID) })
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return out, nil
}
