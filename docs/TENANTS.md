# Tenant model and operation

A Tenant is the infrastructure membership and network boundary for Boxes. It is
separate from Web accounts, Incus Project, Unix users, encryption tenant_id and
Prompt Graph. The console retains one local administrator.

## Authority, identifiers and compatibility

The Provider owns actual membership, policies and NIC ACLs. SQLite schema 2 owns
display metadata, plans, jobs, audit and graphs. Display edits do not change
network membership. Native Provider keys take precedence; labels.tenant is never
interpreted as user.containarium.tenant.

Discovered IDs are tenant_ plus the full SHA256 of the native key. Empty local
records use tenant_local_ plus UUID hex. At first shared creation preview, a local
record is bound once, with optimistic revision control, to bw- plus its UUID.
Its stable local ID survives discovery of created Boxes. A declared key is not
proof of a Provider entity: empty records remain metadata only until membership
is observed. Name changes, duplicate display names and IP reuse never merge
networks. Existing Boxes are never moved, relabeled or recreated.

On the patched Provider, networkTenant/networkTenantSource are authoritative
membership observations. Fresh Guard must independently agree before ACL status
is verified. Old v0.114.2 falls back to fresh Guard subjects or explicitly inferred
cloud/name resolution. Inferred membership never establishes isolation. Unknown
backends and unresolved resources remain visible.

A/B controls require authenticated contract boxweave.tenant-network/v1, native
membership support and backendScope local-lxc. Old/unknown Providers retain
read-only Tenant views and Automatic creation; shared creation/policy writes show
a specific unavailable reason. No optimistic version or generic label workaround
enables writes. See [the compatibility matrix](WEB-CAPABILITIES.md).

## A: create Boxes with shared native membership

In Boxes, choose unified settings or individual rows. Select an existing Tenant,
create new Tenant metadata in the dialog, or explicitly choose Automatic. New
Tenant selection preserves other batch settings. The server computes the native
key from the selected stable ID; raw networkTenant, native labels, encryption
tenant_id and cloud spoofing are not browser input.

Review each username, resources, image, expected membership and pending Guard
state before confirming. The patched Provider persists native membership and
requires the same NIC Guard attachment before the first Start. Capability/Guard
readiness is checked again at confirmation. Automatic keeps the independent
admin name-fallback workflow.

Receipts distinguish created, provisioning, membership verified, Guard verified
and TCP not tested. Running alone is insufficient. Unknown, submitted and
verification-pending receipts reserve names; reconciliation reads the original
username and never repeats writes automatically. A mismatch is degraded. SSH,
application readiness and TCP connectivity are separate observations.

Shared membership normally permits internal access, subject to listeners and
other controls. It does not share files or Unix rights, imply cross-host layer 2,
or prove public/gateway/proxy paths are isolated.

## B: reviewed cross-Tenant connectivity

Open Network -> Tenant Connectivity. Choose source/destination and forward,
reverse or both directions, then grant or revoke. A -> B changes only B's
allow_from_tenants; both directions are one atomic Provider transaction. This
grants Tenant-wide initiation, not one Box/port. Source egress and destination
listeners/other controls still apply; established-flow replies do not mean a
reverse initiation grant.

Preview shows exact destination policies, before/after ingress sources, full
policy revisions and preserved fields. Confirmation consumes the short-lived
plan once. Revision conflicts abort the whole batch and require a new preview.
Egress CIDRs/domains, mode, metadata, preset and independent deny rules are not
reset. After commit, readback verifies revisions and preserved policy fields.
Unknown responses remain uncertain and must not be replayed automatically.

Policy configured, Guard verified/degraded/pending and TCP not tested are
separate states. Revoking a stored edge while Guard is unavailable does not prove
immediate enforcement. All current policies and recent receipts are visible.
Tenant detail has Internal Network and External Access; Monitoring shows policy
state without inventing traffic metrics.

Metadata deletion checks Boxes, routes, both ends of graph references and policy
references. Incomplete inventories block deletion; there is no resource cascade.

## Evidence rules

Verified Guard requires ENFORCE, supported nftables, no relevant error, a
timezone-bearing pass within 180 seconds (30 seconds future tolerance), exact
instance/current IP/subject/expected ACL match, attached:true and complete
inventory. Remote-backend evidence cannot be certified by the local Guard.
Empty metadata, Unknown, Pending, Unsupported and Degraded remain distinct.

ACL attachment and policy storage are not real TCP tests. The current isolated
test suite does not establish live A/B reachability. See [validation](WEB-TESTING.md)
and [Provider source patch and live gate](TENANT-UPSTREAM.md).

## Web API and authentication

Existing administrator session, strict Origin/CSRF, bounded JSON and exact input
validation apply. There is no arbitrary Provider proxy or new account/RBAC model.

| Endpoint | Behavior |
| --- | --- |
| GET /api/overview, /api/tenants | Discovery, metadata, Guard and live capability flags |
| POST /api/tenants | Empty metadata: name, optional description |
| GET/PATCH/DELETE /api/tenants/{id} | Detail; revision-protected display edits; confirmed empty-only deletion |
| GET /api/tenants/{id}/boxes | Provider-authoritative inventory |
| GET /api/tenants/{id}/network | Scoped Boxes/routes and Tenant access context |
| GET /api/tenants/{id}/monitoring | State/configuration, no fabricated utilization |
| POST /api/tenant-selection | Persist validated scope |
| GET /api/boxes, /api/metrics | Optional tenant stable ID or unresolved filter |
| GET /api/network/guard | Sanitized fresh Guard evidence |
| GET /api/tenant-access | Policies, directional edges, receipts; optional validated tenant filter |
| POST /api/tenant-access/plans | source/destination stable IDs, direction forward/reverse/both, operation grant/revoke |
| POST /api/tenant-access/plans/{id}/confirm | confirm:true only; one-time reviewed operation |

Create items accept selected tenantId; expectedTenant/native key are computed by
the server. Unknown/protected fields are rejected. Metadata revision conflicts
return 409; Provider policy conflicts are explicit receipts. Scope filters affect
views, not the administrator's underlying authority.

## Prompt context and persistence

Graphs remain independent with undirected commented relationships. Tenant/Box
cards retain stable IDs and optional native key context. Empty metadata never
claims a real network. Tenant-access cards include both ends, source-to-destination
direction, configured/revoked/unknown policy state and TCP not tested. Adding,
editing or exporting these cards never executes a network operation.

Legacy graphs export normally. Single-Tenant context gets a Tenant line; mixed
graphs keep per-card context. Duplicate cards, comments, autosave, manual copy
fallback and secret-pattern rejection remain intact.

Schema stays 2. Binding uses the existing provider_key column and optimistic CAS;
no destructive migration or administrator reset occurs. Schema 1 -> 2 remains
atomic/idempotent and preserves the original tables. The installer stages all
assets and a private consistent SQLite backup before migration. APT lock handling,
MCP forced commands, HTTPS/CA storage and existing service lifecycle stay intact.

Install the reviewed checkout through the existing --source-dir workflow.
Provider upgrade is separate and requires operator approval. Keep matching
software/configuration/private SQLite backups; never restore a stale database
over new writes. Schema-1 software cannot open schema-2 data. Reverting ingress
uses a fresh reviewed source-only revoke, not full-policy replacement.

The current native Tenant candidate is pinned in [the Provider source profile](../patches/containarium/README.md). The historical v0.114.2 patch is not applied to newer forks. Candidate binaries require separate reviewed deployment; the normal installer selects official MCP releases.
