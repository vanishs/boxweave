# Public MCP tool catalog — v1 P0

**Status:** owner-approved design, not implementation. Go with the official MCP and Incus SDKs. Every authenticated tool invocation carries a server-issued `tracking_id`. Mutations are durable asynchronous operations and return `operation_id`, `tracking_id`, initial phase/state, and a safe result-query hint. Reads may respond synchronously, but use the same Tracking/context chain. Do not confuse these IDs with the native Incus operation ID or an idempotency key.

## P0 — 26 tools

| Tool | Kind | Contract |
|---|---|---|
| `box_create` | Mutation | Create an agentless reviewed Ubuntu LTS Box with configurable defaults; **start by default**, `start=false` opt-out and opt-in nesting; distinguish RUNNING from guest Exec readiness. |
| `box_list` | Read | List only this layer's owned Boxes, with native identity and scoped pagination. |
| `box_get` | Read | Inspect an exact owned Box, observed state, readiness, config and bounded capability summary. |
| `box_update` | Mutation | Allowlisted CPU/memory/disk/network/nesting; reject disk shrink before dispatch, capability-gate growth; no raw Incus settings or host mounts. |
| `box_start` | Mutation | Start exact Box UUID and verify native state/readiness as separate observations. |
| `box_stop` | Mutation | Stop exact Box UUID; never confuse a transport acknowledgment with verified stop. |
| `box_delete` | Mutation | Delete exact owned Box with normal confirmation; no special nested-controller discovery or recursive sub-tree protection. |
| `box_exec` | Mutation | Async Incus Exec **inside owned guest** as UID 0 by default, optional explicit guest identity; bounded command and working-directory policy, observed exit/output; no guest Agent, SSH or host Exec. |
| `box_file_list` | Read | Paginated listing of an allowed directory **inside** an exact owned guest, bounded metadata and safe path handling. |
| `box_file_read` | Read | Bounded guest file bytes/encoding/offset/EOF/hash, no host files, device reads or arbitrary resource capture. |
| `box_file_write` | Mutation | Create-only or revision-conditional guest file write; verify target identity/type/size/path and native resulting bytes; never blind overwrite. |
| `box_file_patch` | Mutation | Optimistic guest replacement/patch with expected original SHA-256 or revision and verified post-hash; stale/symlink/unsafe targets denied. |
| `operation_get` | Read | Original scoped durable operation receipt, state, target, Tracking ID, native evidence and observed guest exit where available. |
| `operation_list` | Read | Paginated actor/owned-resource scoped operation history, including UNKNOWN. |
| `operation_output_get` | Read | Bounded, paginated stdout/stderr of an authorized original Exec operation, with unavailable/expired/truncated markers. |
| `diagnostic_get` | Read | Read-only redacted operation/phase/source/omission evidence linked to authorized Tracking ID; never arbitrary host logs. |
| `http_route_list` | Read | List only BoxWeave-owned public HTTP route records. |
| `http_route_create` | Mutation | Bind domain only to verified owned Box UUID + guest port through dedicated managed Caddy. |
| `http_route_delete` | Mutation | Remove exact owned route identity, never arbitrary Caddy config. |
| `port_forward_list` | Read | List only this layer's TCP/UDP forward records. |
| `port_forward_create` | Mutation | Bind host protocol/port only to owned Box UUID + guest port, with collision and capability checks. |
| `port_forward_delete` | Mutation | Remove only identity-verified owned native forwarding rule. |
| `acl_rule_list` | Read | List scoped directional Box grants. |
| `acl_rule_add` | Mutation | Add directional private access between verified managed Box identities with protocol/port constraints. |
| `acl_rule_remove` | Mutation | Remove one exact owned ACL grant without disturbing foreign ACLs. |
| `host_status` | Read | Report this layer's state, observed dependency versions/capabilities, Incus/Caddy readiness, nested prerequisites and safe diagnostic coverage; no foreign inventory. |

## Universal request and response rules

1. Authenticate first and independently establish this layer's ownership scope. A token's permission category and optional exact Box UUID restrictions apply to **reads as well as writes**.
2. Issue a new `request_id` for each transport attempt and a server-generated `tracking_id` for each logical task. On a valid duplicate idempotency request, return the **original** `tracking_id` and `operation_id` without redispatch.
3. Journal an accepted mutating intent before sending a native side effect. A mutation is **asynchronous by default**, not a blocking MCP request waiting for completion. Output and final states are queried through read tools.
4. Pass a typed Go `context.Context` through transport, domain, journal/worker and adapters; worker context is reconstructed from a safe durable projection. A disconnected client does not automatically cancel native work.
5. Do not return `SUCCEEDED` for a lifecycle/config/route/file operation without verified postconditions. A guest command with `exit_code != 0` is a **completed Exec with failed command outcome**, not an unknown transport effect.
6. Enforce output/metadata limits, retention, pagination, honest missing-evidence states and secrets redaction. Unauthenticated clients get no operation/diagnostic inventory.
7. For Box file operations, resolve native guest identity and safely validate file type, traversal/symlink behavior and point-of-use race. Use the native Incus file API where it can meet the required safety/atomicity. Unsupported safe replacements fail closed rather than attempting unsafe host filesystem access.

## P1 / future candidate tools (not P0)

- `box_file_search`: bounded guest text search if a durable need emerges.
- `box_snapshot_list`, `box_snapshot_create`, `box_snapshot_restore`, `box_snapshot_delete`: optional native storage-specific snapshots.
- Background process management (launch/attach/read logs/stop), idle auto-sleep and HTTP wake, OAuth 2.1 browser authorization: independent later features; no tool names promised yet.

No mandatory in-Box management client. Do not add `run_host_command`, `read_host_file`, `incus_request`, `caddy_request`, `network_apply_raw`, arbitrary upstream host, arbitrary Incus device/host path or a generic shell running on the host.

Related: [development loop](../product/DEVELOPMENT_LOOP.md), [Tracking/context](TRACKING_CONTEXT.md), [operations](OPERATIONS.md), [security](SECURITY.md).
