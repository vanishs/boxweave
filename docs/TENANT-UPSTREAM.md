# Upstream Tenant networking contract

Native Tenant support is merged in official Containarium main at
`d4aad1a8a8cd748ab6211c67a3417f173316f9e1`
([PR #3165](https://github.com/FootprintAI/Containarium/pull/3165)).
The [source profile](../patches/containarium/README.md) pins that immutable
upstream commit for compatibility checks. Official
[v0.120.1](https://github.com/FootprintAI/Containarium/releases/tag/v0.120.1)
contains this commit. Deploy compatible Provider components separately; BoxWeave
still requires authenticated capabilities and does not automatically replace
the daemon.

## A: native shared membership before first Start

The dedicated `network_tenant` protobuf/MCP field, REST `networkTenant` and CLI
`--network-tenant` reach the same authorized server and core creation options.
Encryption `tenant_id` remains independent. Generic tenant/native labels cannot
override membership; conflicting cloud attribution is rejected. Explicit sharing
requires administrator authority. When the operator enables
`CONTAINARIUM_NATIVE_ACCOUNT_TENANCY` (default off), non-admin local LXC creation
uses the reserved `account:<username>` namespace to prevent joining
administrator-owned shared Tenants through username selection. Admin callers cannot allocate this namespace.

The native `user.containarium.tenant` config is written during instance creation.
The required NIC Guard consumes that identical key before the first Start;
unsupported/off/failing Guard blocks Start. Requested boot autostart is deferred
until attachment succeeds. Failures attempt cleanup; callers must still inspect
ambiguous outcomes by original logical name before retrying. Native membership
and its source are returned through list/get across REST, MCP and CLI.

The contract covers local Linux LXC only. Explicit VM, Windows, K8s and remote
placement are rejected. Existing Box reassignment remains unsupported.
Omitted-field creates retain the legacy path by default. Account and shared
keys remain distinct in Guard ACLs and eBPF tenant IDs; only absent account
egress configuration can inherit a legacy bare-user policy. Ingress policies
use exact keys and are never inherited across these domains.

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

The authenticated capability contract is `native-network-tenant/v1` with
backendScope `local-lxc`. A version string alone never enables Web writes.

## Release, compatibility and live acceptance gate

The normal installer selects official MCP releases and does not upgrade the
Provider daemon during a normal rerun. The explicit
[--upgrade-provider flow](INSTALL-UPDATE.md#explicit-official-provider-upgrade-050)
can align local daemon/CLI/MCP binaries to one official release and verify the
running authenticated contract. An official release containing the contract can be selected
without applying the historical downstream patch. Optional source builds use
the pinned upstream commit in an isolated checkout. Source-built binaries and
checksums are development outputs, not signed upstream releases. Updating the daemon, CLI and
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
