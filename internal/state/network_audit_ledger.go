package state

import "context"

// Removed schema-6 execution. Historical caller signatures refuse before any SQL or native witness.

type NetworkAuditLedger struct {
	Project, Bridge, ACL string
	Boxes                []NetworkAuditBinding
}

func (s *Store) ReadCompleteNetworkAuditLedger(ctx context.Context, project, bridge, acl string) (NetworkAuditLedger, error) {
	if s == nil || s.db == nil || !auditName(project) || project == "default" || !auditName(bridge) || !auditName(acl) {
		return NetworkAuditLedger{}, ErrInvalid
	}
	return NetworkAuditLedger{}, ErrResourceLocked
}

func (s *Store) readCompleteNetworkAuditLedger(ctx context.Context, project, bridge, acl string, allowEmpty bool) (NetworkAuditLedger, error) {
	if s == nil || s.db == nil || !auditName(project) || project == "default" || !auditName(bridge) || !auditName(acl) {
		return NetworkAuditLedger{}, ErrInvalid
	}
	return NetworkAuditLedger{}, ErrResourceLocked
}
