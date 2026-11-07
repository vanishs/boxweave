# Planned public MCP tools

**Status:** tool contract proposal (not implemented). Every listed mutation returns a durable BoxWeave `operation_id` and its state after intent recording; read-only queries return synchronously. Tool arguments, permissions, response schemas and detailed error codes must be frozen in implementation milestones.

## P0 — 20 tools

| Tool | Kind | Contract |
|---|---|---|
| `box_create` | Mutation | Provision one Box with approved image, resources and optional nesting capability. |
| `box_list` | Read | List only Boxes owned by this layer, including identity and lifecycle state. |
| `box_get` | Read | Return native identity, configuration, state, health and bounded diagnostics. |
| `box_update` | Mutation | Whitelisted CPU, memory, disk and network/nesting options; no arbitrary Incus config or host mounts. |
| `box_start` | Mutation | Start exact Box UUID and verify running state. |
| `box_stop` | Mutation | Stop exact Box UUID and verify stopped state. |
| `box_delete` | Mutation | Delete exact owned Box UUID, with standard server-enforced confirmation and owned-attachment cleanup; no special discovery or protection of a nested controller inside the Box. |
| `box_exec` | Mutation | Execute inside verified owned guest via Incus Exec, capture bounded output/exit; not a host shell. |
| `operation_get` | Read | Original durable operation receipt, current phase, target and evidence. |
| `operation_list` | Read | Scoped, paginated operation history with explicit unknowns. |
| `http_route_list` | Read | List only BoxWeave-owned public HTTP route records. |
| `http_route_create` | Mutation | Bind domain to verified owned Box UUID + guest port through managed Caddy. |
| `http_route_delete` | Mutation | Remove owned route by identity, not arbitrary Caddy configuration. |
| `port_forward_list` | Read | List this layer's owned TCP/UDP forwards. |
| `port_forward_create` | Mutation | Bind host protocol/port to verified owned Box UUID + guest port. |
| `port_forward_delete` | Mutation | Remove only owned forward with exact native identity check. |
| `acl_rule_list` | Read | List scoped Box peer/network grants. |
| `acl_rule_add` | Mutation | Add explicit directional permission between managed Box identities, using protocol/port constraints. |
| `acl_rule_remove` | Mutation | Remove exact owned ACL grant without replacing foreign ACLs. |
| `host_status` | Read | Report this layer's controller health, Incus/Caddy capability, storage and nesting preflight; no foreign host inventory. |

All P0 mutating tools use a scoped idempotency key (or a documented equivalent exactly-once admission mechanism), expected target/incarnation identity, typed error states, authorization checks and durable evidence.

Only Box-owned guest commands and guest file accesses are legitimate; the host-local maintenance CLI is separate from this catalog.

## P1 — optional planned tools

| Tool | Kind | Contract |
|---|---|---|
| `box_file_read` | Read | Bounded access within an owned guest, not host file read. |
| `box_file_write` | Mutation | Scoped, validated guest file write. |
| `box_snapshot_list` | Read | List native snapshots for an owned Box. |
| `box_snapshot_create` | Mutation | Snapshot an owned Box when native storage supports it. |
| `box_snapshot_restore` | Mutation | Exact-target restore with explicit confirmation and verification. |
| `box_snapshot_delete` | Mutation | Delete exact owned snapshot identity. |

Snapshots are **not** equivalent to off-host backups. Whole-Box backup is deferred. No specific storage driver is mandated.

## Constraints

- No generic `run_host_command`, `read_host_file`, `incus_request`, `caddy_request`, `network_apply_raw`, wildcard shell, or arbitrary URL proxy tool.
- Do not expose internal Incus REST endpoints to clients. Public administration is MCP even when the internal SDK uses HTTP.
- One operation ID is not an assertion of completion. Consumers must inspect `operation_get` for final, independently verified outcomes.
