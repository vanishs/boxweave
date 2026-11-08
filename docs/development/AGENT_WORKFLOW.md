# Agent development, verification and handoff workflow

This document elaborates [AGENTS.md](../../AGENTS.md). It does **not** authorize any real host or production resource modification. It applies at every nested BoxWeave layer.

## Start-of-task checklist

1. Confirm requested outcome, current repository and branch, HEAD SHA, existing edits, related decisions, one observable acceptance criterion and current open risks.
2. Read [product requirements](../product/REQUIREMENTS.md), [architecture](../design/ARCHITECTURE.md), [security](../design/SECURITY.md) and [open risks](../verification/OPEN_RISKS.md) before altering architecture or privileged code.
3. Separate repo-only edits, local unit tests, approved disposable Box tests, privileged host activity and production deployment. A tool granting technical access is not operator authorization.
4. Plan briefly in the requester's preferred/current language for multi-step work. Proceed autonomously within already authorized scope. Ask for new permission only when a new target, public exposure, privilege, credential or destructive effect is required.
5. Preserve original failures and concurrent Agent edits. Never use force-push, arbitrary cleanup, restart or a newly created replacement Box to erase uncertain evidence.

## Engineering standards

- Implement runtime code in Go with the official MCP Go SDK and official Incus Go client; use typed Go structs/enums for tool and native API payloads. Do not pass raw Incus/native Incus config from MCP clients or model outcomes as arbitrary strings.
- Keep MCP transport/auth, domain policy, durable operation journal and native adapters independent. Test the domain service without an MCP transport where possible.
- Check real native identity and local ownership before each sensitive read or mutation. Guest Exec is not host Exec.
- Validate the original result, negative/refusal behavior, and postcondition on the actual Incus system. Do not equate API success with applied state.
- Require caller-generated reusable `idempotency_key` on **every** mutating MCP call. Refuse missing/conflicting keys before effects; persist mutation intent, safe caller scope and original Tracking ID *before* dispatch; carry typed Go `context.Context` through MCP/domain/worker/Incus/Caddy/SQLite and reconstruct safe metadata when asynchronous. Keep distinct idempotency, BoxWeave operation and native operation IDs; maintain recoverable UNKNOWN without blind replay.
- A revoked Token blocks new requests and provably undispatched queued tasks; do not infer cancel/rollback of a running or UNKNOWN operation. Never reboot a host automatically from installer/upgrade; report accurate `reboot_required` and leave reboot to the operator.
- Favor focused dependencies; explain license, provenance, transitive dependencies and security risk for additions. Pin required/observed dependency versions in the install ledger; perform automatic managed-dependency reconciliation in **explicit local** install/upgrade workflows, with a separate authorization gate for shared Incus/foreign services. Treat CI workflow, installer, privileges, permissions and deployment scripts as high-review areas.
- Use generic names and example domains in public docs. Escape untrusted text in console and log output; avoid recording private IPs or real service names.
- Maintain user-visible documentation and machine-readable MCP schemas in English. Reply to the requester in their explicitly requested language or this conversation's language by default.

## Small development loop

1. Reproduce a bug before modifying code; save original request and reusable idempotency key, Tracking/Operation/native IDs, controller version/build SHA, sanitized failure phase, bounded stdout/stderr/exit, owned forward/peer state and independent native state. Never create a replacement Box to erase the original UNKNOWN evidence.
2. Write one bounded change, its acceptance criterion and positive/negative regression tests. Avoid unrelated refactors or sweeping generated changes.
3. Run targeted checks first (Go: `go test`, `go vet`, lint/security tools when present). Do not mark absent checks as passed.
4. For MCP behavior, test authenticated Streamable HTTP initialization, discovery and call; invalid arguments, admin/viewer role enforcement and owned Box scope, consistent Tracking IDs, Go context/deadline propagation, and ownership failures; real side-effect/absence-of-effect oracles.
5. For startup/upgrade/nesting/privileged behavior, use an explicitly approved disposable environment. Prefer a clean test Box to prove clean installation. An existing Box with an old release can be seeded in a clean lab to reproduce upgrades.
6. Test the **complete Agent file-apply/edit/build/test/short-wait/cancel/diagnose/retest loop**, including guest-root Exec editing within its Box despite the absence of a patch engine; inspect exact candidate artifact, dependency versions, Tracking ID, service version, Box UUID and actual layer. For a failed delete, compare `operation_get`, `box_get` and `port_forward_list` at every cleanup phase and preserve partial effects. L0/L1/L2 parity is required for nesting-sensitive slices where technically supported.
7. Preserve FAIL/UNKNOWN evidence and its original resource; clean only task-owned successful artifacts and verify native removal.
8. Record exact PASS/FAIL/PARTIAL/BLOCKED/SKIPPED/UNKNOWN with evidence tier and remaining tasks.

## Full reacceptance: independent bug hunt

The owner instruction **Run full BoxWeave reacceptance** invokes an adversarial, black-box-first campaign, not merely unit-test execution.

- **Coordinator:** enumerate currently installed MCP tools and host-local CLI leaves, plus actual Incus/runtime versions. Build a named per-tool/function matrix including tests that cannot safely run.
- **Finder:** attempt realistic end-to-end create/read/edit/async Exec/reconnect/fix/retest workflows, invalid/refusal paths, two-actor concurrency, service interruption, nested failures and independently observed side effects. Preserve first failure and known-bad calibration.
- **Fixer:** isolate root cause, retain reproduction and produce an exact candidate with focused tests.
- **Independent Verifier:** first establish the known-bad repro (where safe), then run the unchanged original black-box failure against the **exact candidate binary digest** in a different authorized nested/native test installation; inspect actual owned Box/port/peer state and negative effects. A build-only mock or same-Agent self-declaration is not independent installed acceptance.
- **Reporter:** list every tool by name, Tracking ID, original operation identity, exact installed SHA, affected Box and native dependent forwards, external LAN probe witness, tested layer, scenario, result and evidence tier; retain suspected bugs, upstream limitations, harness defects and UNKNOWN receipts separately. Nested-controller localhost MCP tests must run from a guest-local authorized client; external HTTP reachability requires an independent LAN witness. If independent agents are unavailable, label same-Agent, not independent.

The full test harness belongs in repository/CI or disposable test Boxes, **not** inside the standard customer runtime unless a small diagnostic is a deliberate product feature.

## Privileged-boundary adversarial cases

Test missing/reused/conflicting caller idempotency keys, revoked Token queued/active operation races, unsupported IPv6 fail-closed and **absent Caddy process/website management**, foreign Incus instances/project, stale/reused Box names/IPs, raw Incus/Caddy API input, all `box_file_apply(create|replace|delete)` modes with guest file traversal/symlink/hardlink/device/stale hash, operation cancel without permission/unsupported cancellation, short wait after disconnect, host paths/mounts, hostile logs, operation output cross-token disclosure, native port collisions, simple peer allow/revoke, nested access to parent/sibling, expired credentials, HTTP MCP origin policy, lost worker replies and restart with original UNKNOWN identity. Denial must leave native foreign state unchanged.

The owner-approved private-network contract does not promise to block a deliberately published native TCP/UDP port if underlying routing allows access. Do not silently expand P0 into global hairpin/firewall management.

## Version, publication and handoff

- Version is `1.X.Y`: X = feature, Y = patch. Read [versioning](../process/VERSIONING.md) and [approved defaults](../decisions/DEFAULTS.md). Documentation-only edits do not create functional releases.
- Source-CI success, immutable GitHub Release and a production-host upgrade are distinct events. **The owner's CI auto-publication policy applies only after the exact candidate passes an independent real native/nested acceptance gate**, with reproducible hashes and original regression/negative evidence. Candidate bytes must be installable for acceptance **before** a formal release exists; CI may publish matching assets afterward. Never upgrade the old L0 host, change its failed operation, or move an installer stable selector as an implicit side effect.
- Handoff in one short note, with exact source SHA/dirty state, target layer/Box UUID, changes, commands run and exits, proof level, retained resources, unresolved risks/UNKNOWN and smallest next executable task.

For full requirements see [acceptance](../verification/ACCEPTANCE.md).
