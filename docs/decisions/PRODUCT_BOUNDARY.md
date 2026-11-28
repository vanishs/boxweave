# Permanent v1 product boundary

BoxWeave is a deliberately small agentless MCP control plane for owned Linux containers. The [23-tool catalog](../design/MCP_TOOLS.md) and local operator maintenance are the current surface; read [status](../STATUS.md).

- One Go controller per layer, official MCP/Incus SDKs, restricted nonroot runtime, SQLite ownership/operations. No foreign adoption, general Host shell/files or administrator passthrough.
- One explicitly authorized management capability per independently revocable Actor. No admin/viewer roles, application permission scopes, generic RBAC or per-Box delegation. Local HTTP, operator-owned HTTPS/Tunnel.
- Typed guest lifecycle, Exec, readiness, bounded file list/read/exclusive create, durable operations, Activity and guarded original Recovery. Unsafe conditional replace/delete unsupported. Git/build/test and Task Checkpoint are client/guest work.
- Default deny private peer, one directional owned pair, native TCP/UDP forwards. The current #191 Scoped fresh candidate uses bounded Root helpers; historical Legacy differs. No arbitrary network editor, custom proxy, Caddy, DNS or website routing.
- Host-only maintenance and evidence-bound Doctor/Recover. Conversation consent is not backend-verifiable trusted approval. MCP Elicitation deferred; unsupported uncertain repair remains refused or locally reviewed. No generic unlock or replay.
- Independent nesting is conditional native compatibility, not recursive orchestration or guaranteed hostile isolation. Do not weaken policy for test success.
- Historical compatible Legacy upgrade belongs only to matching older builds. Current #191 is Scoped fresh-only, with reviewed full retirement and separate completed-data purge; no product upgrade or rollback. No silent migration/reset. Stable release requires distinct approval/evidence.

Snapshots, Clone, Backup and Import are not implemented v1 features; separately review their scope before development. Native metadata visibility does not authorize creation/restoration. See [scope](../product/SCOPE.md), [defaults](DEFAULTS.md) and [historical decisions](../history/README.md).
