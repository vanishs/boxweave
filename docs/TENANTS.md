# Tenant model and operation

A Tenant is the infrastructure ownership and network boundary for a group of
Boxes. It is the only resource grouping concept in BoxWeave. A Tenant is not a
Web account, Incus Project, Unix username, encryption key scope, or Prompt Graph.
The console retains its single local administrator and independent MCP transport.

## Authority and identifiers

Containarium owns existence, state, membership and NIC ACL attachment. SQLite
schema 2 stores display metadata, current selection, graphs and existing durable
plans, jobs and audit records. Editing a name never changes a network boundary.

Discovered IDs are `tenant_` followed by the full SHA256 of the native Provider
key. Local empty records use `tenant_local_` followed by a random UUID hex value.
Neither uses an IP. `providerKey` is immutable through the metadata API; `name`
and `description` are editable using optimistic revisions. Two Tenants can have
the same display name without being merged. Local empty records have no Provider
network, even when their display name matches an existing native key.

At the pinned Containarium v0.114.2 source commit, the official resolver uses:

1. Native `user.containarium.tenant`, trimmed.
2. Generic `user.containarium.label.cloud_org_id`, trimmed.
3. Instance name with `-container` removed.

The Box REST representation exposes generic labels, **not** the native Tenant
key. Generic `labels.tenant` is a different namespace and is never treated as the
native network Tenant. A fresh Tenant Guard entry's `subject` supplies the
resolved native key, including its explicit-key precedence. Without this evidence,
cloud attribution or the official name fallback is shown as inferred, with
`confirmed:false`; missing identifiers, unresolved/unmanaged resources and
unverified remote-backend attribution remain Unknown. Inferred membership never
establishes isolation. Read-only discovery does not insert or mutate metadata.
No existing Box is renamed, recreated, relabeled or moved during migration.

## Guard evidence

`GET /v1/network/guard` is read through the existing server-side Web credential.
No new credentials are delivered to browsers. Membership includes its source,
confirmation flag, and sanitized Guard evidence. The console requires all of:

- ENFORCE mode, supported `nftables` driver and no unsupported indication;
- a timezone-bearing last pass no more than 180 seconds old (30 seconds of future
  clock tolerance), no stale marker and no relevant reported error;
- exactly one matching current instance entry, matching Tenant subject and ACL
  `containarium-tenant-<first 12 SHA256 hex characters of native key>`;
- `attached:true`, no entry error and the Box's current IP matching the entry;
- complete inventory and no unresolved/unmanaged ambiguity for aggregate status.

Other states are Unknown, Unsupported, Pending or Degraded. Empty metadata has
state Empty. No route-to-Tenant association is inferred from IP reuse: routes
must identify a current Box by exact instance name or username; orphans remain
Unresolved. Remote-backend resources are conservatively unresolved until the
adapter can bind Guard evidence to the correct backend. A local-host Guard must
never certify a peer's similarly named Box.

Verified ACL attachment is not a bidirectional TCP connectivity result. Same
Tenant normally permits internal access, subject to services and other controls.
Direct bridge traffic between different Tenants is guarded; gateway, proxy,
published-port and internet-egress paths differ. Tenant membership shares neither
files nor Unix users/admin rights, and does not promise cross-host layer-2 access.

## Console workflow

1. Sign in and open Overview or Tenants. Existing native Tenants are discovered.
2. Select a Tenant for its Overview, Boxes, Network, Monitoring and Prompt Context.
3. Edit display metadata; the stable ID and Provider key stay unchanged.
4. Use Open scoped Boxes or the Tenant scope selector. All Tenants and Unknown /
   Unresolved remain available. Group by Tenant preserves search, sort and batch
   selection. Scope changes only the view and saved selection.
5. Tenant detail supports selected start/stop/delete using the same exact-target
   confirmation plans. Open Boxes for selected-field resize and batch creation.
6. Create empty Tenant metadata if desired; this does not create a Provider entity.
7. Delete metadata only after fresh checks for Boxes, routes, graph references and
   incomplete/uncertain inventory. No implicit cascade deletes any resource.
8. Add Tenant or Box cards to any graph, including a mixed-Tenant graph.

### Batch creation limitation

Containarium v0.114.2 cannot guarantee explicit shared network Tenant creation
across CLI, MCP, REST and the pre-start NIC guard. Selecting a target Tenant in
unified or individual mode retains the selection but disables review/submission
with the reason visible. The server rejects the same request regardless of UI.
No generic-label injection, `cloud_org_id` spoofing, arbitrary `tenant_id`, or
post-start relabeling workaround exists in BoxWeave.

Choose **Automatic — each Box has its own network Tenant** explicitly to retain
the existing independent-Box workflow. The confirmation preview shows each name,
resources, image, expected native name fallback and pending Guard status. This is
an expectation, not a successful membership assertion. Name conflicts and
unresolved create receipts still block duplicate creation.

Accepted creates are Submitted; Query actual creation state observes existence,
state, actual membership and ACL evidence. Creating/Provisioning and Verification
pending are distinct from Success. A confirmed membership mismatch is Degraded.
Success requires Running plus expected membership and verified ACL evidence;
SSH readiness and application/TCP readiness remain separate. No reconciliation
replays a write or retries an ambiguous create. Old receipts remain readable.

The requested `dev -> frontend/backend/database` shared-network creation and
same/different-Tenant TCP acceptance scenarios remain blocked pending the
[upstream contract change](TENANT-UPSTREAM.md). The current binary stays intact.

## API contract

All endpoints require the existing administrator session. Writes require exact
Origin, CSRF and bounded JSON bodies. No tenant account or RBAC system is added.

| Endpoint | Behavior |
| --- | --- |
| GET /api/overview, /api/tenants | Metadata + discovered membership + Guard + capability flags |
| POST /api/tenants | Create local metadata (`name`, optional `description`) |
| GET /api/tenants/{id} | Detail, capability flags and fresh deletion blockers |
| PATCH /api/tenants/{id} | Display fields and required integer `version` |
| DELETE /api/tenants/{id} | Empty metadata only; `confirm:true`, `version` required |
| GET /api/tenants/{id}/boxes | Provider-authoritative scoped inventory |
| GET /api/tenants/{id}/network | Scoped Boxes and associated routes/passthrough |
| GET /api/tenants/{id}/monitoring | Scoped state/configuration; no fabricated utilization |
| POST /api/tenant-selection | Persist validated `{id}` (empty means All) |
| GET /api/boxes, /api/metrics | Optional `?tenant=<stable ID>` or `unresolved`; old clients still receive containers |
| GET /api/network/{route,passthrough} | Same validated Tenant filter |
| GET /api/network/guard | Sanitized Core/Tenant Guard, capabilities |

Metadata writes audit `tenant-metadata-save` / `tenant-metadata-delete` in the same
transaction. Revision conflicts return 409. Unknown Tenant filters return 404;
unsupported filter fields return 400. Tenant filters restrict server responses,
not administrator authorization: this is still one privileged local administrator.
Unknown or protected write fields are rejected. An optional create-item `tenantId`
is accepted only when empty; nonempty intent is explicitly rejected until the
Provider contract is supported. `expectedTenant` in a returned plan is server
computed and never forwarded as a Provider label.

## Prompt context and persistence

Resource references retain old `{type,id}` compatibility and optionally carry
`tenant:{id,key?,name?,source?}`. `type:tenant` references use the same stable ID.
Tenant metadata is a context snapshot, not live authorization or ACL proof.
Graphs remain independent, many-to-many resource references with undirected
commented relations. No execution or nested node system is introduced.

A graph whose resource cards all identify the same Tenant exports one `Tenant:`
line after the fixed BoxWeave/MCP header. Mixed or partly unresolved context uses
per-card `[Tenant: ...]`. Native keys identify real networks even after display
renames. Empty local metadata is explicitly marked `local metadata only` with its
stable ID. Legacy graphs export normally, preserving comments, duplicates,
relationships and global prompts. Secret-pattern rejection remains in force.

Schema 1 -> 2 runs atomically, adding only `tenants` and updating user_version.
The old eight tables and their records are untouched. Reopening is idempotent;
failed migration rolls back. The existing installer stops only Web, creates a
consistent private `graphs.pre-migration-*.sqlite3` backup and migrates as the
Web identity. Releases, Caddy CA data and SQLite stay in their existing paths.
APT lock handling, MCP services/forced commands, provider token rotation and
Incus versions are unchanged.

Upgrade with the reviewed checkout using the existing install.sh --source-dir
workflow. Do not delete Boxes or reinitialize the administrator. Check Web status,
doctor, protected API discovery and retained graphs after upgrade. For rollback,
stop Web, preserve a private copy of current data, and restore matching old
software/configuration plus the pre-migration schema-1 backup. Schema-1 software
must not open schema-2 data; new writes after the backup require reconciliation.
There is no automatic whole-host rollback or destructive schema downgrade.

See [validation evidence](WEB-TESTING.md) and [capability matrix](WEB-CAPABILITIES.md).
