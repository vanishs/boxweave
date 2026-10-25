# Implemented upstream Tenant networking patch

Baseline: the current fork branch `feat/native-network-tenant-core`, pinned to
`c13582b5016e7471d83d50d1e09a49a2fe661395`.
The reviewed source profile and build instructions
are in [patches/containarium](../patches/containarium/README.md).
This is a reviewable source candidate, not an installed or published release.

## A: native shared membership before first Start

The dedicated `network_tenant` protobuf/MCP field, REST `networkTenant` and CLI
`--network-tenant` reach the same authorized server and core creation options.
Encryption `tenant_id` remains independent. Generic tenant/native labels cannot
override membership; conflicting cloud attribution is rejected. Explicit sharing
requires administrator authority. Non-admin local LXC creation uses the reserved
`account:<username>` namespace to prevent joining administrator-owned shared
Tenants through username selection. Admin callers cannot allocate this namespace.

The native `user.containarium.tenant` config is written during instance creation.
The required NIC Guard consumes that identical key before the first Start;
unsupported/off/failing Guard blocks Start. Requested boot autostart is deferred
until attachment succeeds. Failures attempt cleanup; callers must still inspect
ambiguous outcomes by original logical name before retrying. Native membership
and its source are returned through list/get across REST, MCP and CLI.

The contract covers local Linux LXC only. Explicit VM, Windows, K8s and remote
placement are rejected. Existing Box reassignment remains unsupported. Admin
omitted-field creation preserves the legacy resolver; non-admin namespace
hardening is an intentional compatibility change.

## B: atomic directional ingress authorization

A -> B means B.allow_from_tenants contains A. Both directions update two
destination policies in one transaction; revoke removes only the chosen sources.
The new endpoint never replaces full policies or independently managed deny rules.
Egress CIDRs/domains, mode, egressMode, metadata permission, source and preset are preserved.

Each destination requires a SHA256 revision of its entire protobuf policy, or
`absent`. Any conflict aborts all destinations. PostgreSQL locks rows in sorted
order, detects competing inserts, and updates only allow_from_tenants/updated_at.
The in-memory store has the same all-or-none contract under a mutex. A missing
policy gets the Provider's unspecified/log-only defaults; the ingress list is
consumed separately by the NIC Guard. It does not silently enable an egress mode.

Admin plus `network-policy:write` is required for ingress edits and existing
policy write methods. Nil legacy scopes retain the Provider's existing convention.
Grants require fresh supported ENFORCE Guard and a durable store. Revokes may be
recorded while Guard is degraded; their enforcement remains pending/degraded until
verified. Store success, Guard reconciliation and TCP connectivity are separate.
The response always reports connectivityTested:false.

| Surface | Capability/read | Reviewed write |
| --- | --- | --- |
| CLI | network-tenant capabilities; existing list/get | network-policy access preview SOURCE DEST [--both] [--revoke]; access apply FILE --confirm |
| Host MCP | network_tenant_capabilities; list_network_policies; get_network_policy | patch_tenant_access with exact revisions |
| REST | GET /v1/network-tenants/capabilities; GET /v1/network-policies[/TENANT] | POST /v1/network-policies/tenant-access |
| gRPC | GetNetworkTenantCapabilities; existing list/get | PatchTenantAccess |

The authenticated capability contract is `boxweave.tenant-network/v1` with
backendScope `local-lxc`. A version string alone never enables Web writes.

## Release, compatibility and live acceptance gate

The normal installer selects official MCP releases and does not upgrade the
Provider daemon. Build the pinned fork commit in an isolated checkout first;
verify the source profile and review the changes. Candidate binaries and checksums are
development outputs, not signed upstream releases. Updating the daemon, CLI and
MCP backend is a separate operator-controlled deployment. Reconnect MCP afterward.
Web token rotation probes the exact loopback capability endpoint: only recognized
atomic-capable backends receive the additional network-policy read/write scopes.
Old Providers retain the original container/route scope set and disabled A/B UI.

Isolated validation includes pre-Start ordering/fail-closed unit tests, policy
preservation/conflicts, real PostgreSQL transactions, generated REST/gRPC, actual
MCP stdio initialize/initialized/tools/list/tools/call and browser tests.
See [actual evidence](WEB-TESTING.md). No live Provider was upgraded.

Live acceptance still needs an explicitly approved plan: deploy the reviewed
candidate in a dedicated test environment, create same-Tenant and different-Tenant
workloads using that host's verified full public key, verify pre-start evidence,
and test real TCP in both directions before/after forward/bidirectional grants and
revocation. Use 600-second readiness deadlines and temporary listeners; inspect
original names after timeout. Check management access and unchanged unrelated
policies. No public port route is needed. Agree cleanup before creation.

Rollback is not a full-policy snapshot overwrite. Review current revisions and
revoke only the introduced ingress edges. Preserve newly created native membership
and metadata; older Providers lack this new contract and must not be assumed safe
for new shared creates. Back up software/configuration/SQLite before deployment,
preserve subsequent writes, and restore compatible components together. No
automatic host rollback, relabeling, credential change or destructive downgrade
is implemented.


## Current consumer compatibility

The Web adapter accepts only the typed Provider application-state enum listed
in the source profile. Unknown states produce an unknown receipt with no automatic
replay. Policy readback compares egressMode as well as the other preserved fields.
An authenticated unique backend record of type local may establish a nonempty
local backend ID; tunnel, unknown, duplicate or unavailable metadata cannot.

Revoking an absent destination is skipped and an entirely absent revocation is a
no-write validation error. Grants require fresh enforced Guard evidence; revokes
can be recorded while degraded. ACL attachment is still not a TCP test.

The normal BoxWeave installer downloads an official MCP release. It does not
install this source candidate or upgrade the Provider daemon. Rerunning it after
a separately approved candidate installation can replace the candidate MCP;
coordinate daemon, CLI and MCP versions explicitly. This development task did
not install binaries, rotate real credentials or restart services.


## Suggested upstream PR descriptions

A �� **feat(network): expose native Tenant selection across creation interfaces**

Allow an administrator to select a real local Linux LXC network Tenant through
CLI, REST, gRPC and Host MCP. Write the native Incus identity at creation and
attach its required ACL before first Start; fail closed and report cleanup
failures. Return Provider-derived membership and source through list/get.
Keep encryption tenant_id separate, reject unsupported placements, and use the
reserved account namespace for non-admin new local workloads. Existing Boxes
are unchanged.

Validation: generated protocol compatibility, permission/label/cloud bypass
regressions, composed first-Start and cleanup failure tests, CLI/REST/gRPC/MCP,
and authenticated independent capability reporting. Real deployment and TCP
acceptance remain separate.

B �� **feat(network): add scoped atomic Tenant ingress updates**

Add one narrow PatchTenantAccess operation over the existing
NetworkPolicy.allow_from_tenants field. Require admin and network-policy:write,
whole-policy revisions and all-or-none destination updates. Preserve independent
egressMode, deny rules and other policy fields. Keep grant readiness, persisted
revocation, Guard reconciliation and actual connectivity distinct. Expose this
operation via CLI reviewed preview/apply, REST/gRPC and Host MCP; report B
independently from shared creation A.

Validation: in-memory and actual PostgreSQL transaction/conflict tests,
authenticated REST scope enforcement, generated REST/gRPC, MCP protocol calls,
CLI and lint/protocol checks. No new Tenant CRUD, schema migration, SaaS system
or replacement network stack.
