package state

// Historical schema-v6 DDL, test fixture only. Never included in the
// production schema or used to open/adopt a real old installation.
const (
	legacyRetirementTableDDL      = "CREATE TABLE network_ipv4_retirements (project TEXT NOT NULL, box_uuid TEXT NOT NULL, incarnation TEXT NOT NULL, bridge TEXT NOT NULL, acl TEXT NOT NULL, mac TEXT NOT NULL, ipv4 TEXT NOT NULL, ipv6 TEXT NOT NULL, create_operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id), delete_operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id), native_proof_sha256 TEXT NOT NULL CHECK(length(native_proof_sha256)=64), peer_compaction_sha256 TEXT NOT NULL DEFAULT '' CHECK(length(peer_compaction_sha256) IN (0,64)), retired_at INTEGER NOT NULL, PRIMARY KEY(project,box_uuid,incarnation))"
	legacyRetirementQueueTableDDL = "CREATE TABLE network_ipv4_retirement_queue (delete_operation_id TEXT PRIMARY KEY REFERENCES operations(id), project TEXT NOT NULL, attempt_order INTEGER NOT NULL CHECK(attempt_order>0), last_reason TEXT NOT NULL DEFAULT '', last_attempt_at INTEGER NOT NULL DEFAULT 0 CHECK(last_attempt_at>=0))"
	legacyCompactionTableDDL      = "CREATE TABLE network_ipv4_compactions (project TEXT PRIMARY KEY, delete_operation_id TEXT NOT NULL UNIQUE REFERENCES operations(id), plan_sha256 TEXT NOT NULL CHECK(length(plan_sha256)=64), prepared_at INTEGER NOT NULL)"
	legacyRetirementIndexDDL      = "CREATE INDEX network_ipv4_retirements_bridge_ip ON network_ipv4_retirements(bridge,ipv4,project)"
	legacyRetirementQueueIndexDDL = "CREATE INDEX network_ipv4_retirement_queue_order ON network_ipv4_retirement_queue(project,attempt_order)"
)
