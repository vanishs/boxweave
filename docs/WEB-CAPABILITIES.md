# Web capability gate: Containarium v0.114.2

Source tag `v0.114.2`, commit `8107ea2724dd12786bbd73ff5371e8314f3d35d9`.
This is source-contract verification, not a live provider integration result.
Evidence: upstream `internal/cmd/`, `internal/mcp/tools.go`,
`internal/mcp/client.go`, `proto/containarium/v1/{service,network,container}.proto`,
`api/swagger/containarium.swagger.json`, `internal/auth/scopes.go`, and
`internal/server/{container_server,network_server}.go` at this commit.

| Resource | Operation | CLI | Host MCP | REST | Web v1 |
| --- | --- | --- | --- | --- | --- |
| Box | list | list | list_containers | GET /v1/containers | yes |
| Box | detail | get | get_container | GET /v1/containers/{username} | yes |
| Box | create | create | create_container | POST /v1/containers | yes; explicit sshKeys, resources, image |
| Box | resize | resize | resize_container | PUT /v1/containers/{username}/resize | yes; CPU/memory/disk only |
| Box | start | start | start_container | POST /v1/containers/{username}/start | yes |
| Box | stop | stop | stop_container | POST /v1/containers/{username}/stop | yes; graceful |
| Box | delete | delete | delete_container | DELETE /v1/containers/{username} | yes; no force by default |
| Box | rename / arbitrary edit | no corresponding rename | no corresponding rename | no corresponding rename | no |
| Proxy route | list | route list | list_routes | GET /v1/network/routes | yes |
| Proxy route | expose | expose-port / route add | expose_port | GET box then POST /v1/network/routes | yes; resolve current box IP |
| Proxy route | delete | route delete | delete_route | DELETE /v1/network/routes/{domain} | yes |
| Proxy route | edit / toggle | no edit verb | no edit tool | PUT exists | excluded |
| Passthrough | list | passthrough-route list | list_passthrough_routes | GET /v1/network/passthrough | yes |
| Passthrough | add | passthrough-route add | add_passthrough_route | POST /v1/network/passthrough | yes; TCP/UDP |
| Passthrough | delete | passthrough-route remove | delete_passthrough_route | DELETE /v1/network/passthrough/{externalPort}?protocol=ROUTE_PROTOCOL_TCP | yes |
| Passthrough | edit / toggle | no edit verb | no edit tool | PUT exists | excluded |
| Monitoring | resource state/config | list / get | list_containers / get_container | container GET operations | yes |
| Monitoring | runtime usage metrics | info --metrics explicitly says not implemented | get_metrics | GET /v1/metrics | excluded by three-surface rule |

CLI names alone are insufficient evidence: `internal/cmd/info.go` prints
"Detailed metrics are not yet implemented." Web deliberately omits this call.
The checked-in Swagger uses camelCase URL placeholders (`externalPort`) and body
properties; the proto HTTP annotation spells the source field `external_port`.
Do not confuse these descriptions with a different API endpoint.

## Authorization and limits

Web credential: distinct subject `boxweave-web`, admin role with explicit
`containers:read,containers:write,routes:read,routes:write` scopes and two-hour
expiry, rotated hourly by a root oneshot. These scopes are broad by resource
class: they are not per-box or per-action least privilege. Admin remains required
for fleet operations. Web adds local session authentication, strict operation
validation, exact confirmation plans and protected-name checks. It is not a
replacement for provider-side authorization. No token, signing secret or MCP
credential is delivered to the browser.

Create is asynchronous. `submitted` means accepted, not SSH-ready. Reconciliation
queries the original username and records observed state. Unknown/submitted
receipts reserve names; there is no automatic retry or rollback. An absent box
after an ambiguous create is insufficient to prove the earlier create cannot
complete. Operators must diagnose such receipts rather than bypassing them.

## Architecture and implementation plan

Keep the existing forced-command launcher, MCP binary and MCP token timer intact.
Add `server/webui` (Flask/Waitress, Argon2id, SQLite), `web` (React/TypeScript,
XYFlow), independent credential rotation and dedicated Caddy/systemd services.
Extend the existing installer with mandatory web staging before publication,
separate web verification after MCP timers resume, CLI commands and diagnostics.
No agent runtime, projects, workflow engine, external graph database or automatic
prompt execution is introduced.

Risks: provider-version drift; broad provider scopes; local CA trust distribution;
per-file rather than whole-host atomic update; filesystem/database backup needs;
ambiguous infrastructure writes; browser clipboard restrictions; concurrent graph
edits. Dedicated-box tests and fake-provider tests do not certify production
host security, bare-metal installation or external LAN reachability.

### Missing-instance response compatibility

On the verified v0.114.2 host, GET /v1/containers/{username} returns HTTP 500 for a
missing default-project Incus instance. Web recognizes only the exact observed
gateway error for the requested instance and normalizes it to absence (404).
Generic 500s and all write failures retain their error/unknown semantics. See
[regression evidence](WEB-TESTING.md#missing-instance-compatibility-regression).

## Tenant contract

| Capability | CLI | Host MCP | REST | BoxWeave behavior |
| --- | --- | --- | --- | --- |
| Query network Tenant | generic labels + Guard | list/get + network_guard_status | containers + GET /v1/network/guard | Fresh Guard subject authoritative; official cloud/name fallback explicitly inferred |
| Explicit shared Tenant at creation | generic --labels; not native guarantee | no labels/Tenant parameter | labels accepted under generic namespace | disabled: birth guard/native namespace mismatch |
| Existing Box reassignment | no verified atomic contract | no verified atomic contract | no verified atomic contract | disabled |
| Guard status | network-guard status | network_guard_status | GET /v1/network/guard | read-only Core/Tenant evidence and freshness |
| Network policy | network-policy | no corresponding tool | policy endpoints | writes disabled; Guard evidence only |
| Tenant display metadata | no Provider CRUD | no Provider CRUD | no Provider CRUD | local authenticated SQLite CRUD, not Provider entities |

The native key is user.containarium.tenant; labels.tenant becomes
user.containarium.label.tenant and is not interchangeable. OSS tenant_id is
restricted encryption scope, not arbitrary network Tenant assignment. See
[model/API/operation](TENANTS.md) and the [proposed upstream extension](TENANT-UPSTREAM.md).
Create reconciliation now also requires matching fresh attached ACL evidence;
Running alone does not imply Success. Provisioning/verification-pending/degraded
receipts retain name reservations. Existing auth scopes remain unchanged.
