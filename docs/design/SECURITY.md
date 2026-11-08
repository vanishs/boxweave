# Security and resource ownership boundary

**Primary invariant (P0):** No request arriving through this BoxWeave layer's MCP can observe private state of, mutate, proxy, delete, or otherwise administer resources outside this layer's verified BoxWeave ownership domain. Even the BoxWeave super-administrator cannot override this *product* boundary through MCP.

## Ownership model

- Own each managed Box by the observed Incus native instance UUID (not name or IP), this BoxWeave installation/layer identity, and durable local records. Native `volatile.uuid` is the preferred source, subject to runtime verification and Incus version-specific behavior.
- Incus Project may be used internally for isolation, but is not a user-facing Tenant. Project membership alone is not sufficient proof of ownership.
- A deleted/recreated instance with the same name/IP is a **new identity**; old ACLs, routes, forwards and operations must not silently attach.
- Each managed Caddy route, ACL rule, forward and operation has an owner record and a native/config identity. Write operations compare current ownership and revision before mutation.
- If ownership records are lost, conflicting or unverifiable, deny actions and offer safe local recovery. Never adopt arbitrary pre-existing Incus or Caddy resources automatically.

## MCP threat model and admission

- Authenticate MCP using verifiable, independently revocable scoped bearer credentials; bootstrap the single super-admin locally. The listener uses loopback HTTP by default; an operator-managed trusted proxy/Tunnel supplies external HTTPS when explicitly configured. Never send administrator passwords on every MCP call.
- Require authorization, operation-class and exact-Box UUID Token permissions, a **mandatory client-provided reusable idempotency key on every mutation**, scoped parameters, and exact native target identity; deny unknown fields or raw native config before any side effect. Revoked tokens are denied for new and provably undispatched work. **Guest-root `box.exec` can read, write and delete any guest file allowed by native guest confinement regardless of `box.file.write` denial**. Do not claim a read-only guest when unrestricted Exec is granted. Guest-root Exec is never permission to execute on or modify the parent host.
- Serve a bounded, typed public schema and redact secrets. Protect HTTP MCP against Origin/CORS abuse, request floods, oversized parameters and output leakage.
- No direct MCP access to Incus's unrestricted Unix socket or Caddy's Admin API. Minimize privileges of the HTTP process; put privileged Incus/host setup behind a narrow local boundary with an allowlist of operations.
- Do not confuse authentication with resource authority. Host files/services and foreign Instances, Incus networks, projects, devices, mounts, disks, sockets, and Caddy routes remain prohibited.

## Private network fail-closed rule

A restricted Incus Project is not proof of native bridge/OVN or IPv6 packet isolation. Validate real private traffic denial for managed IPv4 and enabled IPv6. If native IPv6 enforcement is unsupported, disable only the BoxWeave-owned IPv6 path and independently reverify. If that cannot prevent unintended private Box-to-Box access, refuse the unsafe Box network/start; never disable foreign/host-wide IPv6 as a workaround. Public HTTP sites remain reachable under the separately agreed public-route exception.

## Tracking and guest-file privacy

Assign server-owned Tracking IDs; treat caller-provided tracing headers as untrusted metadata, not authenticated identities. Go context propagates safe identifiers and deadlines to native adapters. Workers revalidate scoped policy, token revocation state and exact Box ownership **before native dispatch**. Revocation blocks queued work whose non-dispatch is proven, but never falsely reports a dispatched/possibly dispatched operation as canceled. Restrict `operation_get`, `operation_output_get`, `diagnostic_get` and `operation_cancel` per actor/token rights and relevant Box; cancellation requires separate `operation.cancel` authority and target-operation permission, not mere read access. Never publish secrets, full commands, guest content or foreign resource metadata through tracing. `box_file_apply` is the single P0 file mutation entry (`create|replace|patch|delete`), with distinct `box.file.delete` permission for deletion; it must refuse host paths, unsafe file types, traversal/symlink/hardlink escape and mismatched revisions. Do not pretend a hash preflight followed by a native write is atomic without evidence.

The local dependency installer is a **separate, privileged operator path**, never an MCP management tool. An explicitly initiated install/upgrade may automatically reconcile approved owned prerequisites, but shared Incus/foreign services require separate specific operator authorization before a disruptive upgrade. This preserves the MCP ownership invariant while maintaining versioned dependencies. See [Tracking/context](TRACKING_CONTEXT.md) and [dependency lifecycle](../process/DEPENDENCY_LIFECYCLE.md).

## Allowed / denied examples

| Request | Disposition |
|---|---|
| Create or stop a Box whose UUID is owned by this layer | Allow when authorized and safe |
| Exec as root (UID 0) **inside a verified owned guest**, within native guest confinement | Allow by default; never host root/host Exec |
| Read, list, create, replace, patch or delete one verified regular file inside an identity-verified owned Box guest | P0 via `box_file_apply` for mutation, with expected revision/hash, scoped rights and safe native guest filesystem handling; never host paths |
| Forward an owned Box's HTTP/TCP/UDP service | Allow only via Box UUID + verified port |
| Send Caddy to proxy an arbitrary host daemon, unrelated system or user-supplied URL | Deny |
| Modify unrelated Incus instances, storage or security config | Deny |
| Mount parent host directories/Incus socket into a Box using raw config | Deny |
| Remove arbitrary host files or run host shell commands | Deny |
| Control a parent/sibling layer from a nested BoxWeave | Deny |
| Access an owned Box's public Caddy site from another Box | Allowed by agreed network policy |

**Important limitation:** A guest Exec can modify the guest's own files and processes; the isolation boundary is the enclosing guest and native security policy. This design does not claim that arbitrary commands inside a permitted guest are harmless.

## Transport and guest control distinction

The public MCP listener is authenticated loopback HTTP. External HTTPS exposure is operator-owned, for example Cloudflare Tunnel, and must not weaken token checks, ownership enforcement or request/Origin validation. BoxWeave's dedicated Caddy publishes **HTTP-only P0 application** domains to owned Boxes, never its own MCP endpoint. It must not automatically issue certificates, expose 443 or redirect HTTP to HTTPS. Direct public HTTP traffic is unencrypted; advise operator-owned TLS termination for sensitive applications.

No guest-installed management Agent is required for Incus Linux containers. Guest-root Exec is explicitly allowed, but unprivileged container configuration, host-mount/device denial and native layer isolation remain mandatory. The target of every native Exec must be an identity-verified Box owned by this controller.

## Caddy and Incus authority

Prefer a dedicated BoxWeave-owned Caddy instance. Fail safely on host port conflicts; never overwrite or adopt a pre-existing Caddy process without a separately approved installation migration. Restrict Admin API to local protected access. Persist expected routes and reconcile only owned Caddy entries.

For Incus, prefer restricted project-scoped identity when technically adequate. Where installation or low-level operations require elevated Incus permission, use a narrow locally callable privileged helper that validates ownership and approved fields; deny arbitrary API calls. Do not mistake a Python/Go policy wrapper around a root-equivalent socket for OS-level privilege separation.

## Nested layers

A nested BoxWeave is independently authenticated with new keys, DB and local Incus. It receives no parent credentials, socket, file bind or unmediated route-management rights. Provisioning a nested-capable Box does not authorize a child to change the parent's firewall, Caddy or Incus. Each installation manages only its **directly owned Boxes**. The parent treats the Box containing a child installation like any other owned Box and does not discover or administer the child's internal Boxes. Ordinary parent-side deletion can therefore erase the child's entire environment; no cross-layer deletion protector or child-discovery capability is part of v1.

## Required negative acceptance

Test foreign Box, fabricated/stale UUID, host path, host mount, raw Incus config, arbitrary upstream, foreign Caddy route, parent/sibling control, concurrent ownership updates, guest-file stale hash/symlink/hardlink, expired token, unauthorized operation cancellation, and unknown state; assert unauthorized native state is unchanged. Preserve actual evidence, not just HTTP/MCP error text.
