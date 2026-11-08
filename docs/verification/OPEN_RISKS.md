# Open decisions, compatibility risks and deliberate exclusions

**Status:** active design questions, not permission to omit agreed P0 invariants.

| ID | Status | Item / resolution gate |
|---|---|---|
| O-001 | **RESOLVED** | The owner explicitly selected Go + official MCP Go SDK + official Incus Go client/module. Actual dependency versions and required Incus API extensions remain to be verified in slice 0. |
| O-002 | **VALIDATE** | Incus native `volatile.uuid` semantics and availability for create/delete/re-create/cloning across supported versions; refuse identity-sensitive writes if unavailable. |
| O-003 | **VALIDATE** | Native bridge/NIC ACL semantics for actual same-bridge Box-to-Box isolation, IPv4/IPv6, inherited NIC profile edits, nested routes and IP spoofing. |
| O-004 | **VALIDATE** | Restricting Incus privileges without giving the network-facing MCP service root-equivalent socket power; explicit local helper and project restrictions. |
| O-005 | **VALIDATE** | Dedicated Caddy instance is an accepted product default. Validate actual port 80/443 conflicts, TLS/DNS behavior, Admin API confinement and safe refusal of foreign service takeover. |
| O-006 | **VALIDATE** | Incus network forwards vs proxy devices for TCP/UDP, host address collision rules, and nested networking. |
| O-007 | **VALIDATE** | Nested Incus kernel/user namespace/idmap/storage/network support and what functions can honestly be guaranteed at each layer. |
| O-008 | **OUT OF SCOPE** | No special nested ingress delegation protocol is required. Operators configure ordinary owned-Box routes/forwards separately at each needed layer. |
| O-009 | **DESIGN** | Operation persistence format, separate Tracking/Request/BoxWeave/Incus ID contracts, safe Go context projection, idempotency scope, async Exec result retention, journal recovery limits, original Incus operation disappearance and UNKNOWN escalation procedures. The cross-layer correlation boundary is intentionally independent. |
| O-010 | **DESIGN** | Administrator is locally bootstrapped and multiple revocable scoped MCP tokens are approved. Finalize exact storage/issuance/rotation/expiration, local HTTP Origin policy, trusted proxy handling and recovery implementation. No MCP public TLS auto-provisioning is needed. |
| O-011 | **DEFERRED** | Full route/forward-based public hairpin isolation bypass protection; the owner chose private-direct isolation only and permits public-site access from other Boxes. |
| O-012 | **DEFERRED** | Whole-Box backup, remote backup and restore; snapshot handling may be considered independently as P1. |
| O-013 | **UNCONSTRAINED** | Storage driver choice; support is capability-dependent. Do not silently require ZFS/Btrfs. |
| O-014 | **RESOLVED** | `box_exec` defaults to root (UID 0) **inside a verified owned guest**, optionally supports caller-selected guest UID/GID, never executes commands as host/parent root. No guest-installed client is needed for Linux containers. |
| O-015 | **RESOLVED** | No automated parent delegation workflow. Ordinary route/forward actions belong to the respective layer and its authorized operator; nesting adds no cross-layer API. |
| O-016 | **RESOLVED (refined)** | Reuse compatible existing Incus using a restricted BoxWeave Project and dedicated Caddy. Local install/upgrade now automatically reconciles **owned/approved** dependency versions. Shared Incus/foreign-service upgrade or restart requires a separate specific local impact authorization; no silent foreign takeover. |
| O-017 | **PARTIALLY RESOLVED** | Local super-admin bootstrap plus multiple scoped/revocable MCP tokens are accepted. The concrete credential format, rotation/recovery workflow and secure nested headless initial setup remain implementation-design tasks, not undecided product defaults. |
| O-018 | **PARTIALLY RESOLVED** | MCP is loopback HTTP only by default; operators optionally configure their own HTTPS proxy/Cloudflare Tunnel. Caddy owns Box application routes, not MCP TLS. Website DNS/certificate management, tunnel/client interoperability, and Caddy port 80/443 conflicts still require engineering validation. |
| O-019 | **VALIDATE** | SDK/module compatibility, required Incus API Extensions and tested minimum supported Incus capabilities; official Go client dependency must not be confused with a hard maximum daemon version. |
| O-020 | **DESIGN** | Resource ownership registry and local state recovery when SQLite is missing/corrupted; preserve fail-closed behavior and decide how a human can recover without automatic adoption. |
| O-021 | **RESOLVED** | Ordinary `box_delete` semantics apply to any parent-owned Box, even when it contains an inner BoxWeave. No subtree detection, child handshake, special recursive protection or extra confirmation; operator assumes normal Box data loss. |
| O-022 | **VALIDATE** | Native Incus guest-file list/read/write safety, symlink/hardlink/device handling, concurrent hash-checked replace and exact observed post-state without a Guest Agent or host filesystem fallback. |
| O-023 | **DESIGN** | Trace metadata schema, per-tool Tracking ID envelope compatibility with official MCP SDK, safe context propagation across worker crash, output pagination/retention and scoped diagnostic lookup. |
| O-024 | **VALIDATE** | Pin the actual compatible Incus/Caddy runtime versions, official Go SDK build revisions, Ubuntu LTS baseline and required API extensions; detect shared-service consumers and preflight safe version upgrades. No arbitrary guessed dependency versions. |
| O-025 | **VALIDATE** | Default create/start timing, guest Incus Exec readiness, startup false readiness, nested Linux capacity and the exact optional start=false semantics. |
| O-026 | **DESIGN** | Exact Box UUID/token operation-class authorization model, credential revocation propagation across durable workers, and denial of cross-token output/diagnostic access. |
| O-027 | **VALIDATE** | Long-lived guest Exec without resident Agent: typed argv/cwd/env, native cancel/termination proof, exit/output under disconnect, Box stop and worker restart. Cancellation requested is not confirmed termination. |
| O-028 | **VALIDATE** | Incus guest file API compare-and-swap under other guest writers. A pre-read SHA then write is not inherently atomic. Test point-of-use safety, symlink/hardlink, races and actual post-state before promising safe patch/replace/delete. |
| O-029 | **VALIDATE** | Incus Exec cancellation: waiter canceled versus guest process killed, active cancellation race, UNKNOWN native outcome, restart and partial guest effects. Queued-only cancellation must remain safe if native termination is unsupported. |
| O-030 | **DESIGN** | Final file hash coverage/content encoding and safe payload size, bounded Exec output retention, `operation_get(wait_ms)` waiter limits and Tunnel/proxy interoperability. Pick measured SDK-tested limits, not invented guarantees. |

No unresolved choice permits MCP operation against unowned resources or bypass of the parental layer boundary.

Maintain exact test evidence and the reason a risk was closed. Proposals remain unimplemented until accepted and verified.
