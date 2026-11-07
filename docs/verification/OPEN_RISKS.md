# Open decisions, compatibility risks and deliberate exclusions

**Status:** active design questions, not permission to omit agreed P0 invariants.

| ID | Status | Item / resolution gate |
|---|---|---|
| O-001 | **DECISION NEEDED** | Implementation language: Python was initially selected; Go + official Incus Go SDK + official MCP Go SDK was later recommended but not explicitly ratified. Select before implementation, then record ADR. |
| O-002 | **VALIDATE** | Incus native `volatile.uuid` semantics and availability for create/delete/re-create/cloning across supported versions; refuse identity-sensitive writes if unavailable. |
| O-003 | **VALIDATE** | Native bridge/NIC ACL semantics for actual same-bridge Box-to-Box isolation, IPv4/IPv6, inherited NIC profile edits, nested routes and IP spoofing. |
| O-004 | **VALIDATE** | Restricting Incus privileges without giving the network-facing MCP service root-equivalent socket power; explicit local helper and project restrictions. |
| O-005 | **VALIDATE** | Dedicated Caddy instance isolation, port 80/443 ownership, TLS/DNS behavior, Admin API confinement, and shared-host conflict refusal. |
| O-006 | **VALIDATE** | Incus network forwards vs proxy devices for TCP/UDP, host address collision rules, and nested networking. |
| O-007 | **VALIDATE** | Nested Incus kernel/user namespace/idmap/storage/network support and what functions can honestly be guaranteed at each layer. |
| O-008 | **DESIGN** | Secure parent-authorized ingress delegation across nested layers without allowing a child to administer its parent. |
| O-009 | **DESIGN** | Operation persistence format, idempotency scope, journal recovery limits, original Incus operation disappearance and UNKNOWN escalation procedures. |
| O-010 | **DESIGN** | Initial administrator credential bootstrap, MCP token lifecycle/revocation and secure TLS/Origin controls; keep the single-user scope. |
| O-011 | **DEFERRED** | Full route/forward-based public hairpin isolation bypass protection; the owner chose private-direct isolation only and permits public-site access from other Boxes. |
| O-012 | **DEFERRED** | Whole-Box backup, remote backup and restore; snapshot handling may be considered independently as P1. |
| O-013 | **UNCONSTRAINED** | Storage driver choice; support is capability-dependent. Do not silently require ZFS/Btrfs. |

No unresolved choice permits MCP operation against unowned resources or bypass of the parental layer boundary.

Maintain exact test evidence and the reason a risk was closed. Proposals remain unimplemented until accepted and verified.
