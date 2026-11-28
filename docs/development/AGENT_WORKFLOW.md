# Agent development, verification and handoff workflow

This document elaborates [AGENTS.md](../../AGENTS.md). It does **not** authorize any real host or production resource modification. It applies at every nested BoxWeave layer.

## Development Harness

Start every BoxWeave task with the [canonical BDH bootstrap](../../dev/BOOTSTRAP.md). Use the [toolbox](../../dev/README.md) proportionately; BDH wraps this existing workflow without replacing product phase contracts. Discover only relevant assets, capture learning at meaningful boundaries, and save a canonical checkpoint for handoff. For small documentation tasks the lightweight profile is sufficient under the existing quality policy. Future automation is not a prerequisite to use these rules.

## One-pass development phases and durable Agent handoff

Start with the active task plan and its current ledger. For #191 use the
[consolidation plan](CONSOLIDATION_PLAN.md) and
[live ledger](CONSOLIDATION_PROGRESS.md); the original
[S1–S5 cards](../history/development/phases/README.md) are historical evidence,
not a parallel execution queue. For other tasks use their applicable plan
and [common protocol](PHASE_EXECUTION_PROTOCOL.md). Complete the authorized
stage in one pass, including verification and correction; internal
checkpoints are not user handoffs.

If a subtask remains unresolved, document the original error, native/Tracking/UUID identity, attempts, permission/decision needs and next safe probe. Continue independent work rather than abandon the whole phase; **mandatory safety/release conditions still block their own acceptance/publication**. Update docs whenever code, native facts or context constraints demand it; log cross-phase changes and use narrow links rather than loading the entire archive. The final S5 must reaccept **all 23 approved** tools and every installed local CLI leaf on the exact tested candidate, even if a previous SDK-only or early-stage probe succeeded.

## Commit / PR gates (required at every stage)

Install hooks once per clone. Before every local commit, review the
exact staged blob, pass the applicable staged format/YAML gate and focused
tests for changed behavior; never commit unverified Go. Before publishing a
coherent mixed/source candidate, run full applicable local verification
(normally `make verify`); docs-only uses the existing quality policy.
Required exact-head `quality-gate`, independent native gates and final review
remain mandatory. A local slice is not CI acceptance. No bypass/force-push.
See [quality policy](../process/QUALITY_GATES.md).

## Start-of-task checklist

1. Confirm requested outcome, current repository and branch, HEAD SHA, existing edits, related decisions, one observable acceptance criterion and current open risks.
2. Read **only** the phase card's required entry links at first; consult the [product requirements](../product/REQUIREMENTS.md), [architecture](../design/ARCHITECTURE.md), [security](../design/SECURITY.md) and relevant [open risk IDs](../verification/OPEN_RISKS.md) **when** touching that design/privilege boundary. Don't preload all research/acceptance documents.
3. Separate repo-only edits, local unit tests, approved disposable Box tests, privileged host activity and production deployment. A tool granting technical access is not operator authorization.
4. Plan briefly in the requester's preferred/current language for multi-step work. Proceed autonomously within already authorized scope. Ask for new permission only when a new target, public exposure, privilege, credential or destructive effect is required.
5. Preserve original failures and concurrent Agent edits. Never use force-push, arbitrary cleanup, restart or a newly created replacement Box to erase uncertain evidence.

## Engineering standards

- Implement runtime code in Go with the official MCP Go SDK and official Incus Go client; use typed Go structs/enums for tool and native API payloads. Do not pass raw Incus/native Incus config from MCP clients or model outcomes as arbitrary strings.
- Keep MCP transport/auth, domain policy, durable operation journal and native adapters independent. Test the domain service without an MCP transport where possible.
- Check real native identity and local ownership before each sensitive read or mutation. Guest Exec is not host Exec.
- Validate the original result, negative/refusal behavior, and postcondition on the actual Incus system. Do not equate API success with applied state.
- Require caller-generated reusable `idempotency_key` on **every** mutating MCP call. Refuse missing/conflicting keys before effects; persist mutation intent, safe caller scope and original Tracking ID *before* dispatch; carry typed Go `context.Context` through MCP/domain/worker/Incus/SQLite and reconstruct safe metadata when asynchronous. Keep distinct idempotency, BoxWeave operation and native operation IDs; maintain recoverable UNKNOWN without blind replay.
- A revoked Token blocks new requests and provably undispatched queued tasks; do not infer cancel/rollback of a running or UNKNOWN operation. Never reboot a host automatically from installer/upgrade; report accurate `reboot_required` and leave reboot to the operator.
- Favor focused dependencies; explain license, provenance, transitive dependencies and security risk for additions. Pin required/observed dependency versions in the install ledger; perform automatic managed-dependency reconciliation in **explicit local** install/upgrade workflows, with a separate authorization gate for shared Incus/foreign services. Treat CI workflow, installer, privileges, permissions and deployment scripts as high-review areas.
- Use generic names and example domains in public docs. Escape untrusted text in console and log output; avoid recording private IPs or real service names.
- Maintain user-visible documentation and machine-readable MCP schemas in English. Reply to the requester in their explicitly requested language or this conversation's language by default.

## Fast development, strict acceptance

Default to the existing authorized Box, worktree, local Git objects, toolchain,
caches and verified fixture. Observe identity, ownership, dirty/staged work and
original Operations at resume and meaningful changes, not before every edit.
Reuse immutable context. Consult GitHub for changed or missing facts; fetch
only when a required revision is absent or remote reconciliation is due.
A cached ref is not proof of the live remote head.

Group related inspection, edits and focused tests into bounded local operations.
Avoid another Exec against a known busy resource while the original is pending;
do independent reads or work on a genuinely independent authorized resource.
Review and commit coherent, locally verified slices inside the Box. Accumulate
compatible commits; run affected/adjacent regression on the combined exact
candidate. Publish at a meaningful CI boundary: a native result needed for a
decision, coherent stage, high-risk change needing independent evidence,
durable handoff/backup requirement, or final acceptance. Do not submit every
small edit or create a fresh Box/worktree for every checkpoint. No fixed batch size.

Reuse populated labs for suitable iterative tests; use a clean independent
fixture for fresh install, schema/absence, isolation, reproducibility or other
gates whose oracle requires it. Preserve required native and exact-head CI.
Cache hits, earlier SHA results and reused labs are not independent clean
acceptance. Invalidate affected evidence on relevant change; reconcile drift
or conflicting ownership. Never bypass admission, replay UNKNOWN, reset dirty
work, or weaken a test.

Local commits are not independent backup. Record local branch, base, HEAD/tree,
original receipts, evidence and next action outside the Box; ensure actual bytes
are durably stored when required. Prefer normal fast-forward publication of
local commits. If only the authorized GitHub tree API works, a recorded
aggregate commit may represent their exact final tree; retain local commits
and mapping, verify remote tree and expected parent before advancing the ref.
Do not pretend local commit IDs were published.

## Outcome-first adaptive stages

For a multi-stage task, define a finite remaining exit ledger in the
active plan and at most three user-facing instructions per stage.
Investigate decisive dependencies early, then implement against actual
exit outcomes. At each boundary compare the remaining acceptance and
critical path, invalidate affected evidence and adjust the next stage.
Do not mistake commits, new tests, refactoring or record counts for
completion. Preserve the approved product and mandatory safety gates.

When blocked, classify the affected exit condition and resolve it or
advance only necessary independent work. Defer optional improvements;
do not manufacture work to avoid reporting a blocker. Repeated
non-converging slices require replanning rather than another slice.
Where replacing an architecture, disposition each retained behavior
and remove obsolete live code/test consumers after equivalent evidence
or as a reversible, independently verified cutover candidate. Retain
historical evidence. Do not add compatibility merely for old tests.
Use ordinary deletion-capable Git or reviewed exact-tree publication
when a source export helper cannot represent a removal.

Use the existing phase-wide task history and
[continuation projection](../../dev/skills/task-resume/SKILL.md#phase-wide-continuation-candidate)
at each meaningful boundary; select the next action from the whole
requested stage, not a newly narrowed slice. REVIEW_BLOCKERS and
REVIEW_EXIT require review, not automatic handoff.
Use the [Finish Gate](../../dev/skills/task-resume/SKILL.md#finish-gate-candidate)
before ending the authorized task. A checkpoint, report, local commit,
pending CI, test failure or estimated budget pressure is not a stop reason
while safe required work remains executable. Reuse the existing record,
not another status document. Only completion, a genuine blocker after
independent ready work, an observed platform boundary, or explicit owner
stop ends the pass. Do not claim a quota from an estimate.

One current ledger owns stage status; update it in place, link historical
proof and next 1-3 actions. No repeated full-history prompts or separate
progress registries. Ordinary work reuses BDH; a nonblocking BDH idea
is a deferred candidate, not a new prerequisite.

## Development environment and Box lifecycle

This is the default end-to-end developer workflow, not a new runtime feature or a requirement that BoxWeave already work to develop BoxWeave.

1. **Recover before provisioning.** Confirm the active PR/branch and remote HEAD. Find the existing authorized task workspace via the client [Task Checkpoint](TASK_CHECKPOINT_CONTRACT.md), `work_ref`, `box_activity_get`, `operation_list` and exact Box UUID. Inspect Git HEAD, dirty tree, outstanding Operations and running workloads; preserve other Agents' changes. Labels are not locks or authorization. Do not create a replacement merely because a previous response was lost.
2. **Select the least complex usable environment.** Prefer one task-owned nesting-enabled development Box when available, authorized and capacity permits. Clone/fetch the active branch without overwriting dirty work, set up the pinned Go/Python dependencies and hooks once, reuse caches and worktree. For source/unit/SQLite/HTTP MCP work, no inner Incus is needed. For native behavior, install a separate candidate controller and its own Incus inside the development Box and create child fixtures as needed; go deeper only for a concrete requirement. Use the [nesting contract](../design/NESTING.md), guest-local credentials and loopback client; never borrow parent sockets or broaden outer-host authority. Record capability limitations, not automatic functional failure.
3. **Conventional fallback.** If Box access is absent, unusable, unauthorized or unnecessary (e.g. documentation-only changes), use an isolated local clone/worktree, existing development sandbox, or authorized VM/CI. Preserve existing remote/local changes, follow the same code, test, review and evidence rules, and move only native cases that need unavailable capabilities to an appropriate authorized native environment. Record the reason and untested gates; do not wait for BoxWeave or claim native success from a mock. If the original Box is unreachable, preserve it and its pending Operations, do not assume it is empty or gone.
4. **Iterate locally before pushing.** Reuse focused Go tests, the existing [Python loopback MCP test kit](../history/development/V1_PYTHON_LOOPBACK_MCP_TESTKIT_2026-11-19.md) for real candidate HTTP MCP, and affected native scenarios. Install inner Incus only when needed for real native effects. Prefer the same scenario code in Box and CI with separate verified environment preflights, not duplicate test logic or pretending a Box is a GitHub-hosted runner. Keep test helpers out of the customer runtime. Inspect original Operation and guest exit separately; retain positive/negative and independent native poststate evidence. Repeat the original failing case after each fix.
5. **Finalize and verify.** Run applicable local gates (normally `make hooks` once and `make verify`), review the staged diff, save coherent code/docs and sanitized evidence to Git/approved durable storage, push to the active PR without force, and inspect the required exact-head GitHub Actions and applicable native/independent gates. CI is the final regression/independent fallback, not the default edit-test loop. Do not repeat already sufficient scoped native Box tests solely because they ran nested; do not waive required CI, physical-host, security or independent proof. If a gate remains pending or fails, record it and keep the workspace. Resume the same Box for the fix.
6. **Close or hand off without automatic deletion.** Save the client checkpoint, original receipts, exact tested source/binary and evidence references in durable storage accessible independently of the Box; confirm committed/pushed code (when applicable), docs, and any private evidence retention. On complete acceptance, tell the user what passed, where code/docs/evidence are saved, identify each retained development Box by name and UUID/layer, and ask explicitly whether to delete it. Silence, PASS, generic development authorization or a cleanup instruction in an older document is not approval. Leave the Box intact until the user decides. On partial/failed/blocked/interrupted work, retain its worktree, logs, pending/UNKNOWN Operations and checkpoint with the next safe step; continue unrelated work when authorized. A quiescent Box may be stopped only within authorized scope after checking no active work or volatile evidence will be lost, not as an implicit deletion.
7. **Deletion after approval.** Recheck the user's approval matches the exact name/UUID/layer, check concurrent work, active/UNKNOWN Operations, dependent resources and nested workspaces, and verify durable backups/evidence and authorized native deletion. Warn that deleting an outer Box destroys descendants stored inside it; include that scope in the approval. Use the normal owned stop/delete path, inspect original receipts and verify absence. On uncertainty do not retry blindly; report and retain what remains. No automatic snapshot purge, reset, or destruction of the source workspace. A retained PASS workspace may be reused for a later task only after verifying scope, state and ownership.

**Scope distinction:** this user-confirmed cleanup rule covers persistent development workspaces and retained acceptance labs at any layer (including a lab named disposable). Explicitly authorized short-lived child test fixtures whose creation/deletion is part of a bounded test may be cleaned up as that test requires, provided no development workspace or unresolved evidence is destroyed. Ordinary one-use CI runner teardown is unchanged. When uncertain whether a Box is a workspace or fixture, preserve it and ask. This is an Agent workflow policy, not a change to the public `box_delete` contract.

## Small development loop

1. Reproduce a bug before modifying code; save original request and reusable idempotency key, Tracking/Operation/native IDs, controller version/build SHA, sanitized failure phase, bounded stdout/stderr/exit, owned forward/peer state and independent native state. Never create a replacement Box to erase the original UNKNOWN evidence.
2. Write one bounded change, its acceptance criterion and positive/negative regression tests. Avoid unrelated refactors or sweeping generated changes.
3. Run targeted checks first (Go: `go test`, `go vet`, lint/security tools when present). Do not mark absent checks as passed.
4. For MCP behavior, test authenticated Streamable HTTP initialization, discovery and call; invalid arguments, admin/viewer role enforcement and owned Box scope, consistent Tracking IDs, Go context/deadline propagation, and ownership failures; real side-effect/absence-of-effect oracles.
5. For startup/upgrade/nesting/privileged behavior, use an explicitly approved disposable environment. Prefer a clean test Box to prove clean installation. An existing Box with an old release can be seeded in a clean lab to reproduce upgrades.
6. Test the **complete Agent file-apply/edit/build/test/short-wait/cancel/diagnose/retest loop**, including guest-root Exec editing within its Box despite the absence of a patch engine; inspect exact candidate artifact, dependency versions, Tracking ID, service version, Box UUID and actual layer. For a failed delete, compare `operation_get`, `box_get` and `port_forward_list` at every cleanup phase and preserve partial effects. L0/L1/L2 parity is required for nesting-sensitive slices where technically supported.
7. Preserve FAIL/UNKNOWN evidence and its original resource. Apply the [development Box lifecycle](#development-environment-and-box-lifecycle): clean only explicitly scoped ephemeral test fixtures; retain development Boxes even after PASS until the user explicitly approves deletion.
8. Record exact PASS/FAIL/PARTIAL/BLOCKED/SKIPPED/UNKNOWN with evidence tier and remaining tasks.

## Full reacceptance: independent bug hunt

The owner instruction **Run full BoxWeave reacceptance** invokes an adversarial, black-box-first campaign, not merely unit-test execution.

- **Coordinator:** enumerate currently installed MCP tools and host-local CLI leaves, plus actual Incus/runtime versions. Build a named per-tool/function matrix including tests that cannot safely run.
- **Finder:** attempt realistic end-to-end create/read/edit/async Exec/reconnect/fix/retest workflows, invalid/refusal paths, two-actor concurrency, service interruption, nested failures and independently observed side effects. Preserve first failure and known-bad calibration.
- **Fixer:** isolate root cause, retain reproduction and produce an exact candidate with focused tests.
- **Independent Verifier:** first establish the known-bad repro (where safe), then run the unchanged original black-box failure against the **exact candidate binary digest** in a different authorized nested/native test installation; inspect actual owned Box/port/peer state and negative effects. A build-only mock or same-Agent self-declaration is not independent installed acceptance.
- **Reporter:** list every tool by name, Tracking ID, original operation identity, exact installed SHA, affected Box and native dependent forwards, external LAN probe witness, tested layer, scenario, result and evidence tier; retain suspected bugs, upstream limitations, harness defects and UNKNOWN receipts separately. Nested-controller localhost MCP tests must run from a guest-local authorized client; external HTTP reachability requires an independent LAN witness. If independent agents are unavailable, label same-Agent, not independent.

**Record/replay and negative-control rule (upstream case study):** For each affected MCP/CLI/Incus adapter contract, capture a sanitized, version-pinned failing tool request/response and original Tracking/Operation/native IDs. Replay through the **actual built candidate's** MCP/CLI -> domain -> fake-native boundary with fault injection (never merely call a test helper); use it to reproduce timing/error patterns, not to assert real Incus PASS. Include a known-bad or intentionally injected fault that **must make the test fail**, then verify the fixed candidate in a separate authorized installed/native Box. A test that cannot fail is not evidence. Scrub sensitive transcripts and bind every fixture/result to the original source SHA, installed binary hash, target Box UUID/incarnation and layer. Review stdout truncation and child completion independently.

**Verifier authority and isolation:** A prompt saying “read-only,” a Git worktree or a local path allowlist does not enforce an OS security boundary, and delegated subagents can exceed the parent's requested permissions. Use actual viewer credentials where suitable, minimal-scoped independent test credentials and/or separate disposable Incus Boxes. Verify both direct and delegated forbidden writes with negative tests. Do not introduce per-Box RBAC or a provider sandbox inside the BoxWeave product.

The full test harness belongs in repository/CI or disposable test Boxes, **not** inside the standard customer runtime unless a small diagnostic is a deliberate product feature.

Source/evidence counterexamples: [DevSpace pinned current code and history review](../history/research/DEVSPACE_MAIN_HISTORY_2026-11-07.md).

## Privileged-boundary adversarial cases

Test missing/reused/conflicting caller idempotency keys, revoked Token queued/active operation races, unsupported IPv6 fail-closed and **absent Caddy process/website management**, foreign Incus instances/project, stale/reused Box names/IPs, raw Incus/host API input, all `box_file_apply(create|replace|delete)` modes with guest file traversal/symlink/hardlink/device/stale hash, operation cancel without permission/unsupported cancellation, short wait after disconnect, host paths/mounts, hostile logs, operation output cross-token disclosure, native port collisions, simple peer allow/revoke, nested access to parent/sibling, expired credentials, HTTP MCP origin policy, lost worker replies and restart with original UNKNOWN identity. Denial must leave native foreign state unchanged.

The owner-approved private-network contract does not promise to block a deliberately published native TCP/UDP port if underlying routing allows access. Do not silently expand P0 into global hairpin/firewall management.

## Version, publication and handoff

- Version is `1.X.Y`: X = feature, Y = patch. Read [versioning](../process/VERSIONING.md) and [approved defaults](../decisions/DEFAULTS.md). Documentation-only edits do not create functional releases.
- Source-CI success, immutable GitHub Release and a production-host upgrade are distinct events. **The owner's CI auto-publication policy applies only after the exact candidate passes an independent real native/nested acceptance gate**, with reproducible hashes and original regression/negative evidence. Candidate bytes must be installable for acceptance **before** a formal release exists; CI may publish matching assets afterward. Never upgrade the old L0 host, change its failed operation, or move an installer stable selector as an implicit side effect.
- Handoff in one short note, with exact source SHA/dirty state, target environment (Box name/UUID/layer or conventional worktree), changes, commands run and exits, proof level, durable code/docs/evidence locations, retained resources, unresolved risks/UNKNOWN and smallest next executable task. For accepted work with a retained development Box, ask the user whether to delete the named Box; do not infer approval from the acceptance verdict.

For full requirements see [acceptance](../verification/ACCEPTANCE.md).
