# Proposed upstream network Tenant contract (not implemented)

Baseline: Containarium v0.114.2, commit
`8107ea2724dd12786bbd73ff5371e8314f3d35d9`. This document is a design proposal,
not a Provider patch, deployed fork or verification of shared-network creation.

## Source findings

- `proto/containarium/v1/container.proto`: CreateContainerRequest.labels exists;
  the existing OSS tenant_id is an encryption scope restricted to default/empty.
- `internal/cmd/create.go`: --labels accepted; `internal/mcp/tools.go` creation
  schema has no general labels input. Network-policy has CLI/REST but no Host
  MCP counterpart. Network-guard status has all three read surfaces.
- `pkg/core/incus/client.go`: LabelPrefix is `user.containarium.label.`;
  TenantLabelKey is **separately** `user.containarium.tenant`. ResolveTenant uses
  native key -> cloud_org_id -> name fallback.
- `pkg/core/container/manager.go`: opts.Labels are copied under LabelPrefix;
  a generic tenant label is not the native Tenant key. The initial NIC guard is
  invoked before first Start.
- `pkg/core/container/nic_guard.go`: guardNIC calls ResolveTenant with an empty
  explicit key and cloud_org_id/name inputs. Merely adding a label to an HTTP
  request does not establish a consistent birth-time ACL.
- `internal/tenantguard/` later reconciliation resolves native Tenant identity,
  creating an unacceptable split if creation uses another rule.
- `internal/server/container_server.go`: returned Box labels are generic labels;
  the explicit native field is not separately exposed. Guard entries contain the
  resolved subject and attachment evidence.
- `internal/server/network_server.go`: Guard status requires admin authorization;
  MCP additionally checks security:read. The present REST method does not invoke
  an extra RequireScope. BoxWeave retains its existing independent Web token.

## Minimal compatible extension

1. Add a dedicated optional `network_tenant` / JSON `networkTenant` field to
   CreateContainerRequest and the core creation options. Add a matching CLI flag
   and an explicitly validated Host MCP parameter. Do not repurpose encryption
   tenant_id or expose unconstrained native Incus config maps.
2. Validate once at the shared server boundary and again for direct core callers:
   bounded canonical key, reject empty-as-explicit ambiguity, controls, reserved
   core identities/ACL namespace, and unsupported cross-backend destinations.
   Only an appropriately authorized administrator may assign a non-default
   network Tenant. Never let generic labels override protected native fields.
   Define precedence/conflict behavior with cloud_org_id: reject conflicting
   explicit inputs or use one documented shared resolver; do not silently spoof
   cloud identity. Preserve omitted-field behavior exactly.
3. Persist the native `user.containarium.tenant` key during instance creation,
   before first Start. Feed the **same resolved key** into birth NIC guard and
   periodic reconciliation. Fail closed on required Guard attachment failures;
   never start first and relabel later. Return an explicit provisioning failure
   or retained stopped resource receipt, preserving ambiguous-outcome semantics.
4. Expose native resolved membership and its source in list/detail on CLI, REST
   and MCP, scoped to backend identity. Return the birth ACL evidence separately
   from later reconciliation, with timestamps/errors and no secrets.
5. Keep existing-Box reassignment disabled. A future move needs a separately
   reviewed quiesce/rollback/authorization contract. Keep network-policy writes
   disabled until their CLI/MCP/REST contracts and security checks agree.

An alternative dedicated validated `tenant` label parameter in MCP is possible,
but only if it maps explicitly to the native key and **all** CLI/REST/core paths
apply the same rule before first start. Silently reinterpreting an existing
arbitrary generic label is a backward-compatibility risk; a dedicated field is
preferable. A fingerprint, local Tenant UUID or display name must never substitute
for the requested native Provider key.

## Tests and release gate

Required upstream tests: omitted-field backward compatibility; reserved/native
label spoofing rejection; authorization failures on all transports; identical
resolver precedence; native key and correct NIC ACL present **before first
Start**; failure blocks Start; reconciliation preserves membership; native key
visible consistently through list/get. Test async timeout without duplicate
creation, conflicts, stopped resources, peers, clock freshness and IP reuse.

After an approved reviewed Provider release, add an explicit version/capability
handshake to BoxWeave rather than enabling writes from an optimistic version
string. Test CLI/MCP/REST end to end in dedicated resources. The live plan needs
separate user approval: create two same-Tenant Boxes and one different-Tenant
Box with the verified managed public key, 600-second readiness deadlines, then
serve a temporary TCP service and verify both A -> B and B -> A, separate-Tenant
restriction, Host management access and unchanged original Guard state. Record
birth-time evidence; later Guard convergence alone does not prove it. Agree
cleanup in that plan; do not mutate existing resources or infer isolation from
ping failure. No route/public exposure is necessary.

Independent upstream edits, publishing a Provider version, installing a fork,
or live resource/network tests require authorization. None is part of the
current BoxWeave-only implementation.
