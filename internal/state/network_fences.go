package state

import (
	"context"
	"database/sql"
	"errors"
	"net/netip"
	"strings"
	"time"
)

// NetworkAddressFence represents a durable *local* address hold, not an
// Incus DHCP lease, an exclusive bridge IP, or a proven live native NIC.
// Until native UUID/MAC/lease/ACL readback and packet tests pass, no peer
// grants may consume this record as their security evidence.
type NetworkAddressFence struct {
	Bridge, Family, Address       string
	Project, BoxUUID, Incarnation string
	OperationID                   string
}

func validNetworkAddressFence(f NetworkAddressFence) (string, error) {
	for _, v := range []string{f.Bridge, f.Project, f.BoxUUID, f.Incarnation} {
		if v == "" || len(v) > 100 || strings.ContainsAny(v, " \t\n\r\x00/") {
			return "", ErrInvalid
		}
	}
	if len(f.OperationID) > 128 || strings.ContainsAny(f.OperationID, "\x00\n\r") {
		return "", ErrInvalid
	}
	ip, err := netip.ParseAddr(f.Address)
	if err != nil || !ip.IsValid() || !ip.IsGlobalUnicast() || ip.Is4In6() {
		return "", ErrInvalid
	}
	if (f.Family == "ipv4" && !ip.Is4()) || (f.Family == "ipv6" && !ip.Is6()) ||
		(f.Family != "ipv4" && f.Family != "ipv6") {
		return "", ErrInvalid
	}
	return ip.String(), nil
}

// ReserveNetworkAddress can only create a previously unknown local claim.
// It never releases, recycles or reassigns a quarantined address, even if the
// caller claims a Box was deleted. Native lease exclusivity is a separate P0.
func (s *Store) ReserveNetworkAddress(ctx context.Context, f NetworkAddressFence) error {
	address, err := validNetworkAddressFence(f)
	if err != nil {
		return err
	}
	f.Address = address
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var project, uuid, incarnation, state string
	err = tx.QueryRowContext(ctx,
		"SELECT project,box_uuid,incarnation,state FROM network_address_fences "+
			"WHERE bridge=? AND family=? AND address=?",
		f.Bridge, f.Family, f.Address).Scan(&project, &uuid, &incarnation, &state)
	if err == nil {
		if project != f.Project || uuid != f.BoxUUID || incarnation != f.Incarnation || state != "RESERVED" {
			return ErrResourceLocked
		}
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = tx.ExecContext(ctx,
		"INSERT INTO network_address_fences "+
			"(bridge,family,address,project,box_uuid,incarnation,state,operation_id,observed_at) "+
			"VALUES (?,?,?,?,?,?,'RESERVED',?,?)",
		f.Bridge, f.Family, f.Address, f.Project, f.BoxUUID, f.Incarnation,
		f.OperationID, time.Now().UnixNano())
	if err != nil {
		return err
	}
	return tx.Commit()
}

// QuarantineNetworkAddress retains ownership through crash, lost ACK,
// partial ACL fanout or ambiguous native identity. There is intentionally
// NO TTL release and NO automatic same-IP new-incarnation reassignment.
func (s *Store) QuarantineNetworkAddress(ctx context.Context, f NetworkAddressFence) error {
	address, err := validNetworkAddressFence(f)
	if err != nil {
		return err
	}
	f.Address = address
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx,
		"UPDATE network_address_fences SET state='QUARANTINED',observed_at=? "+
			"WHERE bridge=? AND family=? AND address=? AND project=? "+
			"AND box_uuid=? AND incarnation=? AND state IN ('RESERVED','QUARANTINED')",
		time.Now().UnixNano(), f.Bridge, f.Family, f.Address,
		f.Project, f.BoxUUID, f.Incarnation)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrResourceLocked
	}
	return tx.Commit()
}
