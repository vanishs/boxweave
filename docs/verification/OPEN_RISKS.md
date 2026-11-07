# Open decisions, compatibility risks and deliberate exclusions

**Status:** active design questions, not permission to omit agreed P0 invariants.

| ID | Status | Item / resolution gate |
|---|---|---|
| O-001 | **RESOLVED** | The owner explicitly selected Go + official MCP Go SDK + official Incus Go client/module. Actual dependency versions and required Incus API extensions remain to be verified in slice 0. |
| O-002 | **VALIDATE** | Incus native `volatile.uuid` semantics and availability for create/delete/re-create/cloning across supported versions; refuse identity-sensitive writes if unavailable. |
| O-003 | **VALIDATE** | Native bridge/NIC ACL semantics for actual same-bridge Box-to-Box isolation, IPv4/IPv6, inherited NIC profile edits, nested routes and IP spoofing. |
| O-004 | **VALIDATE** | Restricting Incus privileges without giving the network-facing MCP service root-equivalent socket power; explicit local helper and project restrictions. |
| O-005 | **VALIDATE** | Dedicated Caddy instance isolation, port 80/443 ownership, TLS/DNS behavior, Admin API confinement, and shared-host conflict refusal. |
| O-006 | **VALIDATE** | Incus network forwards vs proxy devices for TCP/UDP, host address collision rules, and nested networking. |
| O-007 | **VALIDATE** | Nested Incus kernel/user namespace/idmap/storage/network support and what functions can honestly be guaranteed at each layer. |
| O-008 | **OUT OF SCOPE** | No special nested ingress delegation protocol is required. Operators configure ordinary owned-Box routes/forwards separately at each needed layer. |
| O-009 | **DESIGN** | Operation persistence format, idempotency scope, journal recovery limits, original Incus operation disappearance and UNKNOWN escalation procedures. |
| O-010 | **DESIGN** | Initial administrator credential bootstrap, MCP token lifecycle/revocation and secure TLS/Origin controls; keep the single-user scope. |
| O-011 | **DEFERRED** | Full route/forward-based public hairpin isolation bypass protection; the owner chose private-direct isolation only and permits public-site access from other Boxes. |
| O-012 | **DEFERRED** | Whole-Box backup, remote backup and restore; snapshot handling may be considered independently as P1. |
| O-013 | **UNCONSTRAINED** | Storage driver choice; support is capability-dependent. Do not silently require ZFS/Btrfs. |
| O-014 | **DECISION NEEDED** | Default guest Exec authority: non-root by default with separately permitted guest elevation vs root guest execution by default. Neither may expose host shell or host mounts. |
| O-015 | **RESOLVED** | No automated parent delegation workflow. Ordinary route/forward actions belong to the respective layer and its authorized operator; nesting adds no cross-layer API. |
| O-016 | **DECISION NEEDED** | New installs on hosts already running Incus and Caddy: isolated managed resources with conflict refusal vs operator-reviewed adoption/migration. Never silently seize foreign resources. |
| O-017 | **DECISION NEEDED** | MCP bootstrap and token lifecycle: exact secure local token issuance/rotation/revocation and initial password ceremony, including headless nested installs. HTTP MCP must never accept a password as a long-lived substitute for scoped credentials. |
| O-018 | **DESIGN** | Public MCP HTTPS endpoint and application-domain DNS/certificate strategy, including nested access and port 80/443 ownership without exposing local Incus/Caddy administration. |
| O-019 | **VALIDATE** | SDK/module compatibility, required Incus API Extensions and tested minimum supported Incus capabilities; official Go client dependency must not be confused with a hard maximum daemon version. |
| O-020 | **DESIGN** | Resource ownership registry and local state recovery when SQLite is missing/corrupted; preserve fail-closed behavior and decide how a human can recover without automatic adoption. |
| O-021 | **RESOLVED** | Ordinary `box_delete` semantics apply to any parent-owned Box, even when it contains an inner BoxWeave. No subtree detection, child handshake, special recursive protection or extra confirmation; operator assumes normal Box data loss. |

No unresolved choice permits MCP operation against unowned resources or bypass of the parental layer boundary.

Maintain exact test evidence and the reason a risk was closed. Proposals remain unimplemented until accepted and verified.
