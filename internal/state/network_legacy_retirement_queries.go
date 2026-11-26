package state

// Legacy-only query fragments. Scoped callers never grant retirement credit.

const legacyHeldBindingsSuffix = " AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.project=n.project AND r.box_uuid=n.box_uuid AND r.incarnation=n.incarnation)"

const legacyHeldClaimsSuffix = " AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.bridge=? AND r.create_operation_id=c.operation_id AND r.project=c.layer)"

const legacyCreateBindingCollisionQuery = `SELECT
	  (SELECT COUNT(*) FROM network_address_fences WHERE bridge=? AND ((family='ipv4' AND address=?) OR (family='ipv6' AND address=?) OR (project!=? AND state!='RELEASED')))+
	  (SELECT COUNT(*) FROM network_bindings n WHERE bridge=? AND (mac=? OR (ipv4=? AND NOT EXISTS (SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.project=n.project AND r.box_uuid=n.box_uuid AND r.incarnation=n.incarnation)) OR ipv6=? OR (project!=? AND state!='RELEASED')))`

const legacyCreateClaimCollisionQuery = "SELECT COUNT(*) FROM operation_resource_claims c WHERE resource_key=? AND NOT EXISTS (SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.bridge=? AND r.ipv4=? AND r.create_operation_id=c.operation_id AND c.layer=r.project)"

const legacyPinRepairClaimSuffix = " AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=? AND r.bridge=? AND r.ipv4=? AND r.create_operation_id=c.operation_id AND r.project=c.layer)"

const legacyPinRepairIPv4Collision = "(ipv4=? AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=n.project AND r.box_uuid=n.box_uuid AND r.incarnation=n.incarnation))"
