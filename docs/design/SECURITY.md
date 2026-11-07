# Security and resource ownership boundary

**Primary invariant (P0):** No request arriving through this BoxWeave layer's MCP can observe private state of, mutate, proxy, delete, or otherwise administer resources outside this layer's verified BoxWeave ownership domain. Even the BoxWeave super-administrator cannot override this *product* boundary through MCP.

## Ownership model

- Own each managed Box by the observed Incus native instance UUID (not name or IP), this BoxWeave installation/layer identity, and durable local records. Native `volatile.uuid` is the preferred source, subject to runtime verification and Incus version-specific behavior.
- Incus Project may be used internally for isolation, but is not a user-facing Tenant. Project membership alone is not sufficient proof of ownership.
- A deleted/recreated instance with the same name/IP is a **new identity**; old ACLs, routes, forwards and operations must not silently attach.
- Each managed Caddy route, ACL rule, forward and operation has an owner record and a native/config identity. Write operations compare current ownership and revision before mutation.
- If ownership records are lost, conflicting or unverifiable, deny actions and offer safe local recovery. Never adopt arbitrary pre-existing Incus or Caddy resources automatically.

## MCP threat model and admission

- Authenticate MCP over TLS with centrally verifiable bearer credentials; bootstrap the single super-admin locally, issue/revoke tokens through a reviewed local workflow. Never send administrator passwords on every MCP call.
- Require authorization, scoped parameters, idempotency and exact native target identity for every mutation; deny unknown fields or raw native config that could bypass policy.
- Serve a bounded, typed public schema and redact secrets. Protect HTTP MCP against Origin/CORS abuse, request floods, oversized parameters and output leakage.
- No direct MCP access to Incus's unrestricted Unix socket or Caddy's Admin API. Minimize privileges of the HTTP process; put privileged Incus/host setup behind a narrow local boundary with an allowlist of operations.
- Do not confuse authentication with resource authority. Host files/services and foreign Instances, Incus networks, projects, devices, mounts, disks, sockets, and Caddy routes remain prohibited.

## Allowed / denied examples

| Request | Disposition |
|---|---|
| Create or stop a Box whose UUID is owned by this layer | Allow when authorized and safe |
| Exec inside an owned guest, within native guest confinement | Allow (subject to guest execution policy) |
| Read or write files inside an owned Box guest | Planned P1; never arbitrary host paths |
| Forward an owned Box's HTTP/TCP/UDP service | Allow only via Box UUID + verified port |
| Send Caddy to proxy an arbitrary host daemon, unrelated system or user-supplied URL | Deny |
| Modify unrelated Incus instances, storage or security config | Deny |
| Mount parent host directories/Incus socket into a Box using raw config | Deny |
| Remove arbitrary host files or run host shell commands | Deny |
| Control a parent/sibling layer from a nested BoxWeave | Deny |
| Access an owned Box's public Caddy site from another Box | Allowed by agreed network policy |

**Important limitation:** A guest Exec can modify the guest's own files and processes; the isolation boundary is the enclosing guest and native security policy. This design does not claim that arbitrary commands inside a permitted guest are harmless.

## Caddy and Incus authority

Prefer a dedicated BoxWeave-owned Caddy instance. Fail safely on host port conflicts; never overwrite or adopt a pre-existing Caddy process without a separately approved installation migration. Restrict Admin API to local protected access. Persist expected routes and reconcile only owned Caddy entries.

For Incus, prefer restricted project-scoped identity when technically adequate. Where installation or low-level operations require elevated Incus permission, use a narrow locally callable privileged helper that validates ownership and approved fields; deny arbitrary API calls. Do not mistake a Python/Go policy wrapper around a root-equivalent socket for OS-level privilege separation.

## Nested layers

A nested BoxWeave is independently authenticated with new keys, DB and local Incus. It receives no parent credentials, socket, file bind or unmediated route-management rights. Provisioning a nested-capable Box does not authorize a child to change the parent's firewall, Caddy or Incus.

## Required negative acceptance

Test foreign Box, fabricated/stale UUID, host path, host mount, raw Incus config, arbitrary upstream, foreign Caddy route, parent/sibling control, concurrent ownership updates, expired token, and unknown state; assert native state is unchanged. Preserve actual evidence, not just HTTP/MCP error text.
