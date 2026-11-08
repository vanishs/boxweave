# Architecture — narrow single-controller runtime

**Status:** design only; no Go v1 runtime has yet been implemented.

```text
Authenticated localhost Streamable HTTP MCP (operator tunnel optional)
     |
 Typed API, admin/viewer auth, Tracking ID, native ownership checks
     |
 Domain policy + durable SQLite operation/idempotency journal
     |
 Async worker + safe Go context + native post-state verifier
     |
 Official Incus Go SDK --> compatible local Incus daemon
     |                         |
     |             owned Box(es) and native ports/peer policy
     |                         |
     +--> agentless guest Exec and guest file APIs
```

A separate host-local CLI manages bootstrap, token rotation/recovery, verified Incus dependency installation/upgrades and health. The local Incus Unix socket grants full Incus administrator authority: a dedicated Project plus `UseProject` is **not** privilege separation. The MCP-facing runtime must use a verified restricted native identity or a narrowly enforcing privileged boundary; the operator-only installer may use the admin socket. No secondary management service, Caddy instance, custom proxy daemon, SSH hop or public REST/Web management exists.

## SDK-first boundary

The official MCP Go SDK owns Streamable HTTP negotiation and typed tool handling; the official Incus Go SDK owns native Instances, Operations, Exec, files and device APIs. BoxWeave retains only its specific product policy, durable idempotency receipts and independently observed postconditions. Prefer stateless Streamable HTTP for MCP 2026-07-28 and one instance-owned proxy-device backend for P0 host-port publishing, with client/native tests before freezing. See [upstream SDK review](../research/OFFICIAL_SDK_FIT_2026-11-07.md).

## Ownership and operations

- One installation owns a restricted Incus Project and an identity-verified set of its own Boxes, native port mappings and minimal directional private peer grants. No foreign container/project/network adoption by default.
- Every authenticated mutation requires a client-reusable idempotency key and returns original Tracking and async Operation ID. For `box_delete`, journal a provider-aware sequence that includes owned forward/peer cleanup and exact instance removal; retain partial effects and safe structured error evidence. SQLite stores intent before dispatch; workers reconstruct safe Go context and verify actual native post-state. UNKNOWN results are not blindly retried.
- Native Guest Exec defaults to guest root (not host root); files use native list/read/create/conditional replace/single delete, no patch engine. Operation status includes minimal safe diagnostics.
- Two revocable MCP token roles only: `admin` and `viewer`. Viewer can read guest metadata/files and bounded output (potentially sensitive), but cannot mutate or Exec.
- Networking is a **fixed safety policy plus two narrow capabilities**: native TCP/UDP forward management for owned Box ports and a directional `box_peer_set` allow/revoke for owned peer tests. The controller is not a general Incus networking manager.
- An inner BoxWeave has its own copy of the same stack. Its host is its containing Box; there is no cross-layer controller discovery or route coordination.

## Libraries and dependencies

Go + official `github.com/modelcontextprotocol/go-sdk`, official `github.com/lxc/incus/v7/client` and matching typed API; local SQLite and operating-system service support. Incus daemon version/extensions and native enforcement are checked. An explicit local installer may take responsibility for an already-installed compatible Incus daemon and reconcile owned dependencies, but shared disruptive upgrades require scoped approval; whole-host reboots remain manual.

See [permanent product boundary](../decisions/PRODUCT_BOUNDARY.md), [Incus integration](INCUS_INTEGRATION.md), [operations](OPERATIONS.md), [security](SECURITY.md), [networking](NETWORK.md).
