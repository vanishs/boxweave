# P0 operational policy decisions — owner-approved

**Status:** The owner selected **5B** and accepted **1A–4A** from the final P0 questions. This is a specification, not an implemented or tested claim.

## 1A. Fail-closed Box private-network isolation

- Only create/start an owned Box when the actual Incus networking configuration can enforce private Box-to-Box isolation. Incus Project membership is not sufficient evidence.
- Prove private IPv4 denial and deny enabled IPv6 paths as well. If owned IPv6 cannot be safely filtered, BoxWeave may disable **only its own managed Box/NIC/network IPv6** and reverify. If that cannot stop private bypass, refuse creation/start and report a typed isolation-unsupported result.
- Never disable host-wide or unrelated/foreign IPv6, weaken ACLs, or change foreign networks merely to make a Box work. Native IPv4/IPv6 packet tests are required before claiming PASS.
- Private network isolation does not block a Box from accessing another Box's deliberately **published public HTTP website**; that public route exception remains.

## 2A. Client-provided reusable idempotency keys

- A **nonempty client-generated `idempotency_key` is mandatory** on every mutating MCP tool: create, update, start/stop/delete, Exec, guest file apply, route, forward, ACL, cancellation, etc. Reject a missing/invalid key **before native effects**. Read-only tools need no key.
- Client creates a key once per logical mutation and **reuses the same key on reconnect/retry**. Uniqueness is enforced on the stable layer + authenticated actor + client key; the stored canonical action, target identity (when available) and payload fingerprint are compared separately. **Same key/same intent** returns original `tracking_id`/`operation_id`; **same key/different action, target or payload** is a typed conflict, not a second mutation.
- For `box_create`, a native Box UUID does not yet exist when admitting intent. Never fabricate one to make key scoping work. Use the layer/actor/key uniqueness and canonical create intent comparison.
- Keys are not authentication credentials, permissions, Tracking IDs or native operation IDs. Returning an original receipt still requires authenticated authorization and scope checks. Keep a durable dedup record across crashes/retention policy boundaries appropriate to avoiding unsafe retry.

## 3A. Revocation: prevent new and provably undispatched work

- Revocation rejects a Token's subsequent MCP requests. An admitted queued operation that is **provably not dispatched** must be blocked at a worker pre-dispatch authorization checkpoint and durably marked `AUTH_REVOKED_BEFORE_DISPATCH` (or equivalent) without native effects.
- An already dispatched—or ambiguously dispatched—operation is **not automatically canceled, terminated or rolled back by Token revocation**. Continue tracking/reconciling original native evidence, preserving UNKNOWN where unresolved.
- A revoked credential loses permission to query operation history, but separately authorized administrators retain scoped, redacted audit access. `operation_cancel` is a distinct authorized, evidence-based action, not implied by revocation.

## 4A. Install/upgrade: no automatic whole-host reboot

- An explicitly invoked local install/upgrade may install or upgrade compatible *approved managed* dependencies, and restart **only authorized local services**. It must **never reboot the whole host automatically**, physical or nested.
- When a kernel/host reboot is necessary, preserve exact manifest, observed versions, operation/Tracking ID and safe installation journal; respond with `reboot_required=true`, affected component and next instructions. The installation remains `PARTIAL` or `BLOCKED` until the operator independently reboots and actual post-boot checks pass.
- Shared pre-existing Incus or other foreign service upgrades/restarts still require explicit target-specific local permission. Never reboot or modify an outer BoxWeave layer from its inner installer.

## 5B. HTTP-only P0 Box application publishing

- Dedicated BoxWeave Caddy only creates **HTTP application routes** for a verified owned Box UUID + guest port in P0. It does **not** issue/renew website TLS certificates, start ACME, bind managed HTTPS on port 443, or enable automatic HTTP-to-HTTPS redirects.
- Generate explicit HTTP-only Caddy site configuration so its usual automatic HTTPS behavior cannot activate accidentally. Do not disable or modify an independent foreign Caddy or unrelated site.
- **Direct public HTTP is unencrypted**, including application credentials and content. Do not present it as a secure public website transport. Operators may separately set up their own trusted HTTPS reverse proxy/CDN; BoxWeave does not manage its certificate or proxy identity. Box website HTTPS is deferred to P1 or later.
- The **MCP management** listener remains authenticated `127.0.0.1` HTTP, with optional operator-configured external HTTPS Tunnel/proxy. That independent MCP decision is unchanged.

## Evidence gates

Before marking P0 accepted, independently verify IPv4/IPv6 packet-level isolation or safe refusal, all tool key validation and duplicate/changed-intent concurrency, queued-vs-running Token revocation, local upgrade with reboot-required but **no** unattended reboot, and Caddy HTTP-only behavior with no 443/ACME/certificate/redirect.

Related: [MCP tools](../design/MCP_TOOLS.md), [operations](../design/OPERATIONS.md), [network](../design/NETWORK.md), [security](../design/SECURITY.md), [dependency lifecycle](../process/DEPENDENCY_LIFECYCLE.md), [acceptance](../verification/ACCEPTANCE.md).
