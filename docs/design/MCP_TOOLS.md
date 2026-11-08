# Public MCP tool catalog — v1 P0

**Status:** accepted API design, not implemented. Go + official MCP Go SDK + official Incus Go SDK. Every authenticated tool call uses a server-owned Tracking ID; accepted mutations return distinct durable BoxWeave operation IDs and run asynchronously. Read tools may return synchronously.

## P0 — 26 tools

| Tool | Kind | Contract |
|---|---|---|
| `box_create` | Mutation | Create an owned, agentless Linux Box with reviewed Ubuntu LTS default, configurable resources, start=true by default, optional start=false and nesting=true; report native running and verified guest readiness separately. |
| `box_list` | Read | Paginated discovery of only this controller's owned Boxes. |
| `box_get` | Read | Exact owned Box native identity, state, readiness, safe configuration and capabilities. |
| `box_update` | Mutation | Explicit allowlist of CPU, memory, disk growth, nesting and internet setting; no arbitrary Incus/NIC/device/network configuration; disk shrink always denied. |
| `box_start` | Mutation | Start exact owned Box UUID and independently verify state and guest readiness. |
| `box_stop` | Mutation | Stop exact owned Box UUID and independently verify stopped state. |
| `box_delete` | Mutation | Delete exact owned Box UUID under standard explicit confirmation; no special nested-controller discovery. |
| `box_exec` | Mutation | Async native Exec inside exact owned guest, UID 0 by default; typed argv, optional safe cwd/guest UID/GID/bounded env, timeout and output limits, durable exit/output receipt; no host Exec. |
| `box_file_list` | Read | Bounded, paginated exact guest directory listing with safe metadata. |
| `box_file_read` | Read | Bounded guest file byte/page read with size, encoding, offset/EOF, SHA-256 and verification status for safe optimistic writes. |
| `box_file_apply` | Mutation | Single identity-checked guest file change with `mode=create|replace|patch|delete`, expected SHA/revision for existing files, exact path, bounded payload, post-read verification; no recursive delete or host file access. |
| `operation_get` | Read | Exact authorized durable operation receipt with optional bounded `wait_ms` long-poll (default 0, maximum 20000), latest observed state/exit and Tracking ID. |
| `operation_list` | Read | Paginated, ownership-scoped operation history, including UNKNOWN and cancellation outcomes. |
| `operation_output_get` | Read | Paginated, bounded original Exec stdout/stderr, cursor and expired/truncated/unknown markers. |
| `operation_cancel` | Mutation | Authenticated, auditable cancellation *request* against an exact owned operation; prove queued cancellation or attempt supported active Exec termination. Never promise native cancellation or undo effects without evidence. |
| `diagnostic_get` | Read | Ownership-scoped redacted phases/errors/coverage by authorized Tracking ID; no arbitrary host logs. |
| `http_route_list` | Read | Read this layer's BoxWeave-owned HTTP routes. |
| `http_route_create` | Mutation | HTTP-only application domain -> verified owned Box UUID + guest port via managed Caddy; no P0 TLS, certificate issuance, forced HTTPS redirects or arbitrary upstream. |
| `http_route_delete` | Mutation | Delete exact owned route; no arbitrary Caddy editing. |
| `port_forward_list` | Read | List this layer's owned TCP/UDP forwarding records. |
| `port_forward_create` | Mutation | Bind host protocol/port -> verified owned Box UUID + guest port; no arbitrary upstream IP. |
| `port_forward_delete` | Mutation | Remove exact owned, independently verified native forward. |
| `acl_rule_list` | Read | Read identity-scoped private access grants. |
| `acl_rule_add` | Mutation | Add directional protocol/port-scoped access between verified managed Box identities. |
| `acl_rule_remove` | Mutation | Remove exact owned ACL rule without disturbing foreign settings. |
| `host_status` | Read | Safe this-layer health, observed dependency versions, Incus/Caddy capability, and nested prerequisites; no foreign host inventory. |

The catalog keeps **26 tools** by merging two guest-file mutations into one and adding an operation-cancellation request. Keep separate lifecycle verbs (not generic `box_action`), read/list tools, Caddy routes vs TCP/UDP forwards, and status vs large output/diagnostic queries.

## Shared MCP behavior

1. Authenticate, enforce both BoxWeave resource ownership and the authenticated principal's operation-class + optional exact Box UUID scopes on reads, writes, diagnostics and cancellation.
2. Create a fresh `request_id` for each transport attempt. A newly admitted logical mutation receives server-owned `tracking_id` and persistent `operation_id`; duplicate idempotency requests return the **original** IDs without dispatch.
3. Require a caller-supplied reusable `idempotency_key` on **every** mutation before admission (including create, Exec, file edits and cancel). Persist accepted mutating intent before native effect; return a quick durable receipt and allow polling. Never silently generate a non-reusable server-only substitute. Distinguish caller timeout/cancel from operation cancellation.
4. Propagate typed Go `context.Context` to domain services, worker and Incus/Caddy/SQLite adapters. Async workers rebuild a safe durable context; client disconnect alone does not cancel work.
5. Return only independently verified post-state as success. Nonzero guest command exit is an observed completed command with `command_success=false`, not a transport failure.
6. Bound and scope all output, file access and diagnostic evidence. Refuse stale native identity, unsafe guest file paths, unverified safe replacement, foreign services, and unsupported native cancellation.
7. Preserve distinct Tracking, Request, Idempotency, BoxWeave Operation and native Incus Operation IDs; tracking and cancellation IDs never authorize access.

For precise arguments, cancellation status and file safety, see [tool contract details](MCP_TOOL_CONTRACTS.md). For [Agent development](../product/DEVELOPMENT_LOOP.md), [Tracking/context](TRACKING_CONTEXT.md), [operations](OPERATIONS.md), and [security](SECURITY.md), see the linked contracts.

## Later (not P0)

Native snapshots, bounded guest file search, persistent background process supervision, idle sleep/HTTP wake, and OAuth 2.1 browser login. No mandatory guest client, SSH or host shell. Never expose raw Incus/Caddy APIs, arbitrary host file operations, host devices, mounts or unmanaged proxy targets.
