package state

import (
	"context"
	"database/sql"
	"errors"
)

// ResolvePeerAuditScope maps two currently owned UUIDs to their one exact
// operator-pinned ACL. A caller cannot supply native addresses or ACL names.
func (s *Store) ResolvePeerAuditScope(ctx context.Context, project, sourceUUID, targetUUID string) (string, string, error) {
	if s == nil || s.db == nil || s.auditReadOnly || !auditName(project) || project == "default" ||
		sourceUUID == "" || sourceUUID == targetUUID || targetUUID == "" {
		return "", "", ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback()
	bridge, acl := "", ""
	for _, uuid := range []string{sourceUUID, targetUUID} {
		var nw, ac string
		query := "SELECT nb.bridge,nb.acl FROM network_bindings nb " +
			"JOIN owned_boxes ob ON ob.project=nb.project AND ob.uuid=nb.box_uuid " +
			"AND ob.incarnation=nb.incarnation " +
			"WHERE ob.project=? AND ob.uuid=? AND ob.state='OWNED' " +
			"AND nb.state='PINNED' AND nb.nic='eth0'"
		err := tx.QueryRowContext(ctx, query, project, uuid).Scan(&nw, &ac)
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrResourceLocked
		}
		if err != nil {
			return "", "", err
		}
		if !auditName(nw) || !auditName(ac) || (bridge != "" && (bridge != nw || acl != ac)) {
			return "", "", ErrResourceLocked
		}
		bridge, acl = nw, ac
	}
	if err := tx.Commit(); err != nil {
		return "", "", err
	}
	ledger, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil {
		return "", "", ErrResourceLocked
	}
	src, dst := false, false
	for _, b := range ledger.Boxes {
		if b.UUID == sourceUUID {
			src = true
		}
		if b.UUID == targetUUID {
			dst = true
		}
	}
	if !src || !dst {
		return "", "", ErrResourceLocked
	}
	return bridge, acl, nil
}
