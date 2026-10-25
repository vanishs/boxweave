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

## Tenant compatibility profiles

The baseline table above describes unchanged v0.114.2. The complete optional
[reviewed source branch](../patches/containarium/README.md) extends all three surfaces:

| Capability | Unchanged v0.114.2 | Patched contract boxweave.tenant-network/v1 |
| --- | --- | --- |
| Membership query | Guard authoritative; cloud/name inferred | Native networkTenant + source returned; independent fresh Guard evidence |
| Shared creation A | blocked | CLI --network-tenant; MCP network_tenant; REST networkTenant; same pre-Start required Guard |
| Cross-Tenant B | Web writes blocked; CLI/REST legacy policy support | CLI access preview/apply; MCP patch_tenant_access; REST atomic tenant-access |
| Policy query | no Host MCP counterpart | CLI list/get; MCP list_network_policies/get_network_policy; REST existing endpoints with full revision |
| Capability | no recognized contract | CLI network-tenant capabilities; MCP network_tenant_capabilities; GET /v1/network-tenants/capabilities |
| Existing Box reassignment | blocked | blocked |
| Metadata | local SQLite only | local SQLite; one-time declared key binding is not membership proof |
| Connectivity evidence | not tested | configured / Guard verified / TCP not tested remain separate |

The handshake is authenticated, requires exact contract/backendScope local-lxc
and nativeMembershipRead, and is reevaluated for writes. Shared creation requires
fresh ENFORCE supported Guard. Atomic policy grants also require durable storage;
admin plus network-policy:write is enforced on the Provider. Revoke can be stored
while enforcement is degraded, but is never reported as applied without evidence.

The independent Web token retains its original four container/route scopes on
old Providers. Token rotation probes only the fixed loopback capability endpoint;
recognized atomic-capable Providers receive network-policy:read and
network-policy:write in a newly minted Web-subject token. Existing MCP tokens,
launcher credentials and timers are not rewritten. A newly upgraded Provider may
need the next normal Web token rotation before writes are available.

Native network Tenant is independent of encryption tenant_id and generic labels.
Explicit shared native keys are admin-only and local Linux LXC only. Non-admin
local creation is hardened to reserved account:<username> membership; admin
omitted-field behavior remains the existing resolver. See [operation](TENANTS.md),
[upstream changes and deployment gate](TENANT-UPSTREAM.md), and
[test evidence](WEB-TESTING.md). No version-string check, label injection or
post-start relabeling workaround enables A/B.

The current native Tenant candidate is pinned in [the Provider source profile](../patches/containarium/README.md). The historical v0.114.2 patch is not applied to newer forks. Candidate binaries require separate reviewed deployment; the normal installer selects official MCP releases.
