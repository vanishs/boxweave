# Architecture — narrow single-controller runtime

**Current architecture:** see [status](../STATUS.md), [network modes](NETWORK.md) and [installation](../getting-started/MANUAL_INSTALL.md). The runtime has one controller per layer, a confined Incus service identity and bounded host-local helpers. Source, installed readiness and physical acceptance are distinct.

```text
Authenticated localhost Streamable HTTP MCP (operator tunnel optional)
     |
 Typed API, verified Actor token and Layer ownership, Tracking ID, native safety checks
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

A separate host-local CLI manages bootstrap, token rotation/recovery, verified Incus dependency installation/upgrades and health. Root/`incus-admin` Unix socket clients are privileged, but a non-root user **only in `incus`** has native per-user Project confinement. Prefer a dedicated `incus`-only BoxWeave runtime, with local-admin-provisioned `restricted.*` capabilities and actual denial tests. `UseProject` alone is not authorization. Current v1 Schema 7 uses only Scoped Peer v2, independent UUID-derived ACLs per Box and bounded Root-local helpers. Historical Legacy artifacts require their matching version; no current Legacy fallback. No secondary management service, Caddy instance, custom proxy daemon, SSH hop or public REST/Web management exists.

## Human, Agent and source-code perspectives

The **human/operator** owns the Linux host, approves nonroot native preparation and public credentials; the **Agent** connects through MCP and receives only typed owned-Box capabilities; **BoxWeave** journals mutations and checks authorization; **Incus** enforces actual container/storage/network behavior. A Box runs guest workloads, **not** an obligatory resident Agent. Nested BoxWeave runs a separate controller in another layer; it does not inherit outer-layer management authority.

The existing v1 source places these boundaries in `cmd/boxweave` (single CLI/server entry), `internal/control` (MCP/auth/typed handlers), `internal/state` (SQLite and durable receipts), `internal/native` (Incus adapter and ownership observations), and `internal/install` (reviewed root-local installation and recovery). The root `install.sh` and `.github/workflows` are distribution/verification tooling, not runtime daemons. [Glossary](../GLOSSARY.md) defines each term.

## Readiness split

`native.service_ready` and `mcp_ready` allow installing a healthy service before preparing the first Box. `native.ready` and full CLI `doctor` are stronger native-write admission gates requiring prepared restricted Project/Profile, pool, NIC and the verified Scoped quarantine/guard and independent per-Box ACL contract. Neither gate requires a preloaded Guest image, proves packet-level isolation, or verifies Incus disk quotas. See [RC.3 decision](../history/decisions/V1_RC3_INCUS_NATIVE_INSTALL_AND_IMAGE_POLICY_2026-11-13.md).

## SDK-first boundary

The official MCP Go SDK owns Streamable HTTP negotiation, Bearer auth middleware, localhost Host/body guards and typed tool handling; the official Incus Go SDK owns native Instances, Operations, Exec, files and device APIs. BoxWeave retains only its specific product policy, durable idempotency receipts and independently observed postconditions. Prefer stateless Streamable HTTP for MCP 2026-07-28 and one instance-owned proxy-device backend for P0 host-port publishing, with client/native tests before freezing. See [upstream SDK review](../history/research/OFFICIAL_SDK_FIT_2026-11-07.md) and [20/20 native method mapping](../history/research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md).

## Ownership and operations

- One installation owns a restricted Incus Project and an identity-verified set of its own Boxes, native port mappings and minimal directional private peer grants. No foreign container/project/network adoption by default.
- Every authenticated mutation requires a client-reusable idempotency key and returns original Tracking and async Operation ID. For `box_delete`, journal a provider-aware sequence that includes owned forward/peer cleanup and exact instance removal; retain partial effects and safe structured error evidence. SQLite stores intent before dispatch; workers reconstruct safe Go context and verify actual native post-state. UNKNOWN results are not blindly retried.
- Native Guest Exec defaults to guest root (not host root); files use native list/read/create/conditional replace/single delete, no patch engine. Operation status includes minimal safe diagnostics.
- One authorized management capability for independently revocable Actor credentials; no `admin/viewer` roles or application permission Scope. Full-management access is limited to this controller's exact owned Box UUIDs and Layer, including sensitive guest-root Exec and bounded reads.
- Networking is a **fixed safety policy plus two narrow capabilities**: native TCP/UDP forward management for owned Box ports and a directional `box_peer_set` allow/revoke for owned peer tests. The controller is not a general Incus networking manager.
- An inner BoxWeave has its own copy of the same stack. Its host is its containing Box; there is no cross-layer controller discovery or route coordination.

## Libraries and dependencies

Go + official `github.com/modelcontextprotocol/go-sdk`, official `github.com/lxc/incus/v7/client` and matching typed API; local SQLite and operating-system service support. Incus daemon version/extensions and native enforcement are checked. An explicit local installer may take responsibility for an already-installed compatible Incus daemon and reconcile owned dependencies, but shared disruptive upgrades require scoped approval; whole-host reboots remain manual.

See [permanent product boundary](../decisions/PRODUCT_BOUNDARY.md), [Incus integration](INCUS_INTEGRATION.md), [operations](OPERATIONS.md), [security](SECURITY.md), [networking](NETWORK.md).
