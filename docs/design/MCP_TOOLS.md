# Public MCP tool catalog — v1

Current v1 source registers 23 public Tools, including Activity and Recovery. This count is checked by the MCP discovery test in internal/control/server_test.go. The installed server's tools/list and build identity are authoritative. See [current source and release status](../INDEX.md). All admitted mutations require reusable idempotency keys and durable original receipts. No new remote Host repair Tool was added by #177 or #178.

**Continuity enhancement to existing Tools, not extra Tools:** [single design](AGENT_OPERATION_CONTINUITY.md) and [TC tests](../verification/CONTINUITY_ACCEPTANCE.md) extend S2 receipts and `operation_list/get/output_get` with authorized optional work/Agent correlation, structured evidence and safe next reads. Authenticated principal and untrusted label must never be confused.

**Git / Task Checkpoint scope:** MCP `tools/list` can expose `box_exec` and the generic `operation_*` Tools. An Agent must choose and call guest `git` commands through `box_exec` (or use its own external terminal), then interpret their output itself. BoxWeave does **not** advertise a Git-specific Tool, auto-detect Git commands, persist parsed Git HEAD/branch/dirty state, or write a Task Checkpoint. [The external client-owned Checkpoint contract](../development/TASK_CHECKPOINT_CONTRACT.md) records derived Git state only when an Agent/workflow explicitly collects and saves it; merely connecting to MCP does not restore such a file.

## Public catalog — 23 tools

| Tool | Kind | Contract |
|---|---|---|
| `box_create` | Mutation | Create owned Ubuntu-LTS-default agentless Linux Box, configurable CPU/RAM/disk, start=true by default, start=false opt-out, opt-in nesting, readiness check; current scoped mode defaults boot.autostart=true and preserves explicit false. |
| `box_list` | Read | Paginated owned Box identities and current states. |
| `box_get` | Read | One exact owned Box's state/readiness/resources and effective directional private-peer grants; optional `detail=true` for a live authenticated management Actor returns native `InstanceFull` (State/Snapshots/Backups, potentially sensitive config) and `observed_at`; requires single-request native extension. |
| `box_update` | Mutation | Allowlisted CPU, RAM, safe disk growth, opt-in nesting and per-Box internet toggle only. Never raw Incus configs or disk shrink. |
| `box_start` | Mutation | Start exact owned Box, verify actual running/readiness. |
| `box_stop` | Mutation | Stop exact owned Box, verify native state. |
| `box_delete` | Mutation | Delete exact stopped owned Box with normal confirmation **covering owned dependent port forwards/peer grants**; journal per-step cleanup, preserve PARTIAL/UNKNOWN and verify no orphan exposure; no inner-controller discovery. |
| `box_exec` | Mutation | Durable async guest Exec, guest root by default, typed argv/cwd/bounded env, timeout and output; no host shell. |
| `box_wait_ready` | Observation (authenticated management Actor) | Asynchronous bounded wait for verified Running, fixed native Guest Exec and optional owned-NIC IPv4/IPv6; durable result through existing `operation_get`, not a guest-ready guess. |
| `box_file_list` | Read | Bounded/paginated list of regular guest directory entries. |
| `box_file_read` | Read | Bounded guest file bytes/pages with truthful metadata/observed hash. |
| `box_file_apply` | Mutation | Single owned guest regular file `create|replace|delete`; **native SFTP `O_EXCL` exclusive create PASSED**, but **conditional `replace/delete` must return `UNSUPPORTED` until atomic point-of-use hash/inode safeguards are actually established**. Never silently substitute ordinary overwrite/delete; no patch engine or recursive delete. |
| `operation_get` | Read | Original scoped async receipt/status with optional bounded wait_ms and structured **failing phase, sanitized native error, last observed Box/dependent resources, evidence gaps, safe recovery hint and controller build identity**. |
| `operation_list` | Read | Paginated owned operations, including UNKNOWN. |
| `box_activity_get` | Read | `overview/issues/evidence`: Layer-authenticated bounded SQLite original/evidence readback; issues distinguish blockers and history; native_network adds bounded read-only same-Project native comparison; signed cursors; no native effects or task checkpoint. |
| `operation_output_get` | Read | Bounded/paginated guest Exec output and observed exit details after reconnect. |
| `operation_cancel` | Mutation | Cancel only a proven queued-undispatched original; detached running Exec/native work is not cancelable through the verified SDK path. No speculative undo or UNKNOWN resolution. |
| `operation_recover` | Read diagnose / durable reconcile | Exact original evidence; deterministic settlement or identity binding; no re-dispatch or unproven claim release. |
| `port_forward_list` | Read | List only native BoxWeave-owned TCP/UDP forwards and host binds. |
| `port_forward_create` | Mutation | Native Incus host listener protocol/address/port -> verified owned Box UUID/guest port; default loopback is not LAN-reachable, explicit LAN/public bind authorized, native device verification and external reachability are separate. |
| `port_forward_delete` | Mutation | Remove verified owned native forwarding only. |
| `box_peer_set` | Mutation | Allow or revoke directional private access from exact owned source Box UUID to target Box UUID; no CIDR/IP/port/protocol policy DSL. Scoped mode supports routine authenticated MCP grant/revoke; legacy may require local review. |
| `host_status` | Read | Read-only this-layer controller **VERSION/build SHA**, Incus version/API capabilities and health with no foreign host inventory. |

No `http_route_*`, `acl_rule_*`, `diagnostic_get`, Caddy API, generic host shell/file/Incus management or additional management transport. Read-only failures return Tracking ID + safe classified error; admitted Operation failures include diagnostic summary in `operation_get`. Detailed logs remain host-local.

## Exact native SDK mapping

Native-facing baseline tools map to tagged upstream APIs in [the original 20/20 mapping](../history/research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md). The Go MCP SDK already supplies typed `mcp.AddTool`, Bearer auth middleware, Schema validation and tool-error handling. Incus `GetInstancePortForwardConn` is **not** persistent port publishing; `CreateNetworkPeer` is **not** a Box-to-Box ACL. Use native instance `proxy` devices and NIC-level ACL instead. Unproven native file CAS must remain UNSUPPORTED rather than simulated.

**Last pre-development SDK P0 gate actually executed:** [2026-11-08 native Incus + real HTTP MCP results](../history/verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md). Real SDK v1.8.0 Streamable HTTP discovery (new and legacy), auth, concurrent Token isolation, body cap and request-cancel handling **PASSED**. Native Incus v7.6.0 Exec/recorded output retrieval after SDK reconnect and live control-socket SIGTERM **PASSED with limitations**; native API did **not** supply atomic file conditional replace/delete. P0 SDK research is complete, **not** a claim that the v1 runtime and all 20 tools passed.

## Shared semantics

- Each authorized Actor uses its own independently revocable management credential (manual Bearer or OAuth grant with stable Actor across refresh); **no application Scope or admin/viewer roles**. Valid credentials can read/mutate only exact Layer-owned resources, including guest-root Exec. MCP authentication still rejects anonymous, forged, expired and revoked Tokens; no per-Box Token delegation or tool-level RBAC. Operation evidence within the same Layer may be read by a different authenticated Actor for safe takeover, never to reuse another Actor's original idempotency identity.
- Every mutation requires nonempty client-generated reusable `idempotency_key`. Exact same key+intent+actor returns the same original Tracking/Operation IDs; conflicting reuse is denied before effect.
- Every native change is journaled, scoped by actual Incus Box UUID, independently checked, and accurately classified UNKNOWN if observation fails. Reader disconnect does not cancel original async work.
- Guest-root `box_exec` inherently permits modifying that guest's files; don't pretend restricting file tools restricts guest Exec. Safe guest file writes require native point-of-use guarantees; otherwise UNSUPPORTED.
- Native port forwards and peer grants must not modify host foreign Incus/network resources, expose arbitrary upstream or weaken mandatory default isolation.
- Repeat the minimal full Agent loop inside a supported nested Box; never build a parent-controller-management API.

See [permanent product boundary](../decisions/PRODUCT_BOUNDARY.md), [detailed tool contracts](MCP_TOOL_CONTRACTS.md), [operation model](OPERATIONS.md), [networking](NETWORK.md) and [Agent loop](../product/DEVELOPMENT_LOOP.md).
