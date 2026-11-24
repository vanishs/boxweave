# Owner-approved target behavior and implementation dependency — PR #178 precedes #177

**Decision recorded: 2026-11-23. Status: authoritative product requirements for future implementation, NOT a claim that RC13 already behaves this way.** This document **supersedes** earlier suggestions in this Draft PR to retain `boot.autostart=false` after successful Peer changes, to require Host terminal approval for every routine Peer operation, or to accept disruptive whole-Project restarts as normal successful Peer behavior. The earlier [RCA](V1_RC13_PEER_ROOT_RCA_2026-11-23.md) remains valid **historical RC13 evidence**.

## Superseding Owner decision — fresh installation only (2026-11-23)

The final #178 acceptance target is a **fresh BoxWeave installation**, not online migration of an existing shared ACL database/network. Do not implement or accept shared→per-Box live migration as a requirement. An existing installation or incompatible/shared ACL environment must be refused explicitly before any unsupported upgrade, overwrite, service stop, schema write or native policy change. Never silently convert old environments; retain the released baseline and its protection. Fresh installation must establish independently owned per-Box ACLs and guarded policy from initial provisioning, including Create/Delete and boot/restart lifecycle, before enabling unattended Peer. Existing Incus resources must not be taken over or modified by inference. If safe reuse cannot be proven, refuse it; no migration fallback.

This scope decision does not waive fail-closed behavior, the autostart contract, authenticated routine MCP operation, unrelated C/D continuity, or real native fault acceptance. Historical migration references elsewhere in #178 describe earlier proposals and are superseded as final scope. #177 remains deferred.

## Product requirements / acceptance

1. **Default `boot.autostart=true` for new managed Boxes.** A normal Box must start after an eligible Host/Incus reboot by default. Respect an *explicit* user-requested autostart-disable setting when supported; do not silently override opt-outs. Successful Peer grant/revoke and normal network recovery must neither permanently change a Box's original autostart choice nor turn an unrelated Box's original choice off. This includes exact unset/true/false semantics, creation/update/migration, service/Incus/Host restart and state reconciliation. Existing shared-ACL BoxWeave environments are outside this PR's supported upgrade scope; refuse unsupported upgrade before effects. Do not silently alter Host inventory or unknown/protected claims. Verified persisted state, not an instance running right now, is the acceptance witness.

2. **Routine Peer management through authenticated MCP without per-operation Host shell steps.** Once the owner installs and grants the minimum necessary authority at a **trusted initial setup**, authorized `box_peer_set` grant/revoke and ordinary evidence-supported recovery should complete using MCP and a bounded, auditable Host-local mediation mechanism. Prohibit an unrestricted reusable root daemon, generic Host shell, arbitrary ACL writes, cross-Project access, elevation by untrusted arguments, and bypassing per-actor authorization. Prefer **least-privilege, scoped and continuously checked** service capabilities; enforce native UUID/incarnation, exact Project/network, ETag, idempotency, per-Operation identity, original journal and fail-closed authorization. A single initial operator setup/approval may be unavoidable; the goal is **no repeated manual Host commands for routine valid operations**. High-risk exceptional Host/Root recovery with incomplete evidence must remain explicitly blocked or require separately trusted local authorization rather than insecure automation.

3. **Unrelated Box services and networking must remain uninterrupted on normal Peer changes.** Grant/revoke A→B must **not stop/restart C, D or other unaffected workloads**, must not kill their listeners, must not change their boot flags, and must preserve unrelated active network sessions and Forward reachability. Ideally A and B should also remain running without application disruption; if a narrowly scoped endpoint interruption is fundamentally unavoidable, show its precise cause and explicit impact before execution and do not represent it as full no-downtime acceptance. No transient fail-open for unauthorized A→C, C→B, IPv4 or IPv6, no stale conntrack access after revoke, no privilege/address-reuse transfer. Native ACL propagation currently affects multiple NICs asynchronously; **a candidate must demonstrate real packet safety under partial native fanout and ACK loss**. If the selected design cannot satisfy safety and availability together, **fail closed and report unsupported/blocked**. Do not quietly fall back to whole-Project shutdown while claiming success.

4. **Do not fossilize RC13-specific implementation in diagnostics.** Existing one-use root unit, `ADMITTED` Host wait, all-consumer STOP, retained `boot.autostart=false`, Guest listener loss and CLI/daemon policy differences are current/historical implementation observations. The new design can change or eliminate these phenomena. Diagnosis must be written *against the verified finished architecture*, not hard-coded to historical states or current implementation details.

## Why this cannot be treated as an install.sh-only bug

RC13 uses a one-original/one-SHA Root Helper and a **shared ACL with full-bridge stop barrier** because native rule propagation is not atomic. Preinstalling a generic unit without a security redesign could expose overly broad Host privilege; merely restoring autostart without revising the current fail-closed resume proof is unsafe; changing only `install.sh` does not prevent unrelated Box downtime. All three requirements need source-level design, threat modelling and real native packet/rollback tests. Compare bounded installed helper models and safe per-NIC/per-Box policy approaches before selecting a change; avoid prescribing an unverified nftables backend.

## Required #178 implementation sequence

A. Audit current `v1` architecture: Box creation/start and autostart defaults; shared ACL root barrier and full fan-out; conntrack/nft/NetworkACL readback, one-original root unit; MCP/authorization and recovery; installer prerequisites. Document exact privilege and durable-state ownership.

B. In a disposable Box with independent nested Incus and candidate daemon, reproduce existing affected/unaffected topology and security baseline. Propose at least two viable least-privilege MCP→Host execution and minimal-network-impact options, with a threat/complexity/rollback matrix. **Prioritize no unrelated-Box interruption;** reject unproven candidates.

C. Implement safe autostart default and state preservation with tests. Implement the chosen bounded MCP Root mediation with exact original authorization and comprehensive negative cases. Implement non-disruptive network policy update **only after** native fail-open/partial convergence/conntrack proof. Prefer small tested commits and keep normal successful operations fully MCP-driven.

D. Real native acceptance: `A→B allow/revoke` IPv4+IPv6/TCP+UDP where supported; `B→A`, `C→B`, `A→C` denial; active C/D HTTP/Forward/connection remains alive throughout; no stop/restart of unrelated Box (SDK events/instance uptime/live process); native ACL/conntrack/nft and durable ledger verified independently, including stale ETag, ACK loss, crash/restart, stop in mid-flight, simultaneous grants, IP/MAC reincarnation and address reuse, default autostart through explicitly authorized reboot. Unknowns hold locks and refuse speculative replay. Test alternative endpoint interruption cases separately and record limits honestly.

E. Run affected Go/race, Python MCP/installed API, Native CI and digest-exact final HEAD checks; document any unavailable physical/IPv6/Host-reboot proof as `NOT_TESTED`. No physical L0 changes, no v1 merge, no release until independent Owner sign-off. End with the actual operation/diagnostic semantics/schema and migration notes that #177 will consume.

## Hard dependency: #177 waits for #178

**Freeze the new unified Doctor/Recover implementation in Draft PR #177 until #178's network/Root/autostart implementation is finished, independently verified, and integrated into the chosen target branch by separately authorized review.** #177's *already-completed historic guided-uninstall work and evidence are preserved* and do not need re-implementation or teardown. Historical RC13 reports remain readable as test evidence, never as a specification of final diagnostic codes.

At handoff, #178 Agent must publish:
- final effective network model, permission/helper lifecycle and whether normal Peer still needs any approval;
- `boot.autostart` creation and persistence/migration behavior;
- operation state transitions, original claim/lock and Root/native journal event contract;
- affected versus unrelated Box impact, Forward device vs Guest app vs real packet semantics;
- signed exact source/test/CI references, any UNSUPPORTED/BLOCKED boundary and remaining privileges/coverage gaps;
- versioned schema/issue taxonomy delta, explicit unsupported-upgrade rejection semantics and precise follow-up to #177.

Only then should #177 Agent reread **actual merged candidate code**, replace outdated RC13-only findings, update reference data/source maps, then implement read-only Doctor and scoped Recover. Do not interpret this scheduling as “#178 finished” or “#177 feature implemented” today.
