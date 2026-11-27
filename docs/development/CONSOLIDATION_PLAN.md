# v1 pre-release consolidation — active PR plan

Baseline: `9d07665dcfc5ae6345d6bfcd0d71de7b6510b51d`, 2026-11-24. This is an authorized development target, not a statement that the baseline already implements it. One topic branch and one Draft PR into protected v1. Do not merge or publish without separate owner approval.

Start with [BDH](../../dev/BOOTSTRAP.md), then [progress](CONSOLIDATION_PROGRESS.md), [retirement inventory](RETIREMENT_INVENTORY.md) and [acceptance](CONSOLIDATION_ACCEPTANCE.md) as relevant. Use existing task/checkpoint and learning mechanisms. The [phase protocol](PHASE_EXECUTION_PROTOCOL.md) applies with the parallel-CI refinement below.

## Owner direction and scope

There are no supported production users or data to migrate before the first stable release. The owner permits rebuilding disposable development environments; obsolete pre-release compatibility is not a product requirement. This does not authorize deletion of foreign resources, destruction of unresolved original evidence, or arbitrary physical/shared-host effects. Record exact fixture UUID/layer and authority before cleanup; preserve Git, test evidence and checkpoint state.

- Deliver one Scoped network/runtime architecture: per-incarnation ACL, default deny, routine authenticated directional Peer, independently verified native poststate, unchanged unrelated flows and boot choices. No Legacy runtime, fallback or selection mode. Old installation evidence must be refused safely before effects; refer to exact old published artifacts for retirement or rebuild an explicitly authorized disposable fixture. Do not implement migration, adoption or automatic reset.
- Preserve the 23-tool MCP catalog and supported semantics, official SDKs, restricted nonroot controller, bounded Root mediation, Actor/Layer ownership, SQLite original receipts, UNKNOWN locks and recovery. External client protocol compatibility is not pre-release server-data compatibility.
- One coherent install/uninstall/diagnostic contract, one initial schema with explicit version and integrity checks. Preserve normal and exceptional cleanup, reviewed plans, resume, completed-data purge, foreign resource protection and conditional nesting. Separate user CLI entries from internal systemd/coordinator leaves; do not remove an internal entry until all callers and witnesses are converted.
- Reuse compatible pre-existing Incus; retain relevant native dependency, storage, quota, identity and isolation checks. Incus package lifecycle is not BoxWeave data migration. Review its surface separately rather than delete by name.
- Retain existing Scoped capacity limits, fail-closed policy and unsupported file CAS. No address reuse/compaction feature, new MCP tools, snapshot/clone/backup product, host automation, new auth scheme, or speculative replacement subsystem. Future stable-to-stable nondestructive Scoped upgrade/rollback is a separate design/release-readiness decision for #170; no claim that it is implemented here. Do not silently promise it.
- Consolidate redundant phase-specific workflows/fixtures only after mapping each retained behavioral assertion to a current equivalent. Do not change required quality-gate, branch protection, dual architecture support, signed stable release authorization or immutable publication guarantees. #170 owns stable candidate automation and final publication; update its acceptance baseline after this PR, not in parallel.
- Keep history and old immutable releases as evidence; current documentation and BDH must describe verified target behavior, not fossilize historical names.

## Execution: Box first, CI in parallel

1. Resume the actual PR HEAD, current phase, dirty work, original operations and pending CI; reconcile any divergence before editing. Use an authorized isolated Box/worktree first, with clean independent nested Incus fixture when feasible. Do not replace the active controller used to manage the development environment. Where native capability is genuinely unavailable, record its precise limitation and use an independent clean runner; source work need not stop.
2. Reproduce and test each coherent slice in the Box; format before committing, verify the applicable source/native regressions and preserve exact evidence. Transfer sanitized results to the PR. Push coherent checkpoints to the same topic branch so applicable CI runs independently; do not push broken intermediate states merely to generate CI.
   Apply [fast local reuse and strict acceptance](AGENT_WORKFLOW.md#fast-development-strict-acceptance): locally verify and commit compatible slices in the same worktree, then submit a coherent candidate at a meaningful CI boundary. All independent gates and durable handoff remain mandatory.
3. While CI is queued/running, continue Box-first development on the next safe ready slice or independent phase work. A pending run is not PASS. Do not advance a dependent irreversible/safety-critical action on unproven assumptions. Distinguish BOX_VERIFIED, CI_PENDING, CI_FAILED, CI_VERIFIED, BLOCKED and COMPLETE; record exact commit, run/attempt and job. Preserve prior runs even if newer commits make them stale.
4. When CI reports a concrete failure, inspect the actual job/log and exact SHA, reproduce in the Box where possible, determine product vs fixture vs environment vs infrastructure cause, fix the still-applicable defect on the current branch, rerun the failing and adjacent cases, and submit the correction. Do not weaken tests to make green; if an older failure is already fixed, link the fixing commit and fresh evidence. Continue unaffected work; stop dependent work on unresolved safety failures. Do not spin waiting for CI or discard failure evidence.
5. At each phase checkpoint update progress, inventory and acceptance mapping. Use ordinary BDH Solve/Evolve loops and record applicable learning dispositions with evidence; independently review reusable changes, regression and Eval before promotion. Continue until the requested scope is complete or genuine blocker/budget boundary. Final exact HEAD must pass all applicable independent gates; previous green runs are not final acceptance.

## P1 convergence recovery: R0-R5 (active)

Owner direction 2026-11-26: actual convergence, not more D checkpoints.
These R labels apply to P1, not BDH PR193 or later P2-P5. Start from the
[live exit ledger](CONSOLIDATION_PROGRESS.md#current-p1-recovery-ledger)
and [audit](https://github.com/vanishs/boxweave/pull/191#issuecomment-7207100365).

One user command ("P1 R0", etc.) starts the named stage; each has at most
three execution instructions. Complete work, checks and fixes autonomously.
At entry reconcile evidence. At exit compare actual deliverables with
acceptance, reassess assumptions/dependencies and adjust the next stage.
Record reason, affected C-IDs, blocks and next 1-3 actions in the live
ledger. Reorder or combine dependent work when justified; seek approval
for changed product scope, acceptance equivalence/waiver or new authority.
No extra owner command for routine fixes. No new stages or optional work
to fill a blocked stage; no PASS for unmet mandatory gates.

### R0 - Investigate and reset the critical path

1. Reconcile PR, local commits, CI/native proofs and original Operations;
   reuse audit and 72-marker map. Trace actual dispatch, SQL, installation,
   CLI, test and CI consumers. Identify true blocks and obsolete behavior.
2. Check shortest reversible cutover, deletion-capable publication,
   clean fixture feasibility and public WAN evidence/authority. Run only
   decisive bounded experiments; no speculative product/BDH development.
3. Freeze finite exit ledger, dependency order and witness dispositions;
   record decisions needed and R1 actions. Exit: executable cutover
   sequence and justified gate placement, not more research.

### R1 - Prepare minimum cutover dependencies

1. Convert only callers, fixtures and consumers preventing cutover;
   move necessary P2/P3/P4 dependencies forward. Reuse applicable proofs;
   disposition obsolete Legacy-only behavior.
2. Verify retained coverage and negative calibration; preserve historical
   evidence, not obsolete live dependencies. Exit: no unidentified cutover
   prerequisite; record focused local commits and reassess R2/R3.

### R2 - Switch to one runtime

1. Implement Scoped-only Create/Delete/Peer/admission/recovery/startup;
   remove Legacy selection, unreachable execution and dependent callers,
   with minimum installer/CLI changes. Combine R2/R3 atomically if needed.
2. Build and exercise actual binary startup, supported positive/negative
   behavior and source absence/reachability; repair failures. Exit: one
   runtime, no live fallback, preserved authority and UNKNOWN semantics.

### R3 - Switch to one fresh schema

1. Remove obsolete DDL and consumers; bump direct schema version.
   Retain state/claims/address holds and same-schema replay. Reject
   old/foreign/malformed/partial state before effects; no migration,
   adoption or reset. Coordinate R2 and minimum consumers.
2. Verify fresh DB, persistence, old-state no-write refusal and
   fault/recovery assertions. Exit: one schema, no live old-table
   dependency; aggregate R2/R3 local candidate verified.

### R4 - Converge affected acceptance

1. Finish cutover-required consumers, equivalent native mapping, old-mode
   retirement and gate plumbing. Preserve oracles and negative calibration;
   defer unrelated P2-P4 cleanup.
2. Verify combined candidate, submit at meaningful CI boundaries and fix
   concrete failures. Track WAN separately by affected C-ID; require
   exact current native evidence or explicit owner-approved equivalence.
   No assumed environment cause, fake PASS or waiver. Exit: all affected
   native/consumer gates satisfied, or explicitly BLOCKED (not complete).

### R5 - Independent integrated acceptance

1. Independently review exact candidate and C01-C05/C10/C11 dispositions,
   Legacy runtime/DDL absence and retained safety/behavior witnesses.
   Run applicable clean native and exact-head CI gates.
2. Correct affected defects and revalidate changed proof. Exit: P1 complete
   only when all mandatory P1 gates pass; save concise evidence and
   P2-P5 handoff. No merge or release.

## Overall consolidation stages (one PR)

P1 uses active R0-R5 above; the table retains overall scope. Necessary
dependencies may move into P1 without closing later phases.

| Stage | Work and exit checkpoint |
| --- | --- |
| P0 — preflight | Resolve exact source and CI baseline, existing authorized development Box and clean/nested fixture. Build call/reference graph across Go, SQL, CLI, systemd, shell/Python, workflow YAML, docs and BDH. Inventory every affected test/marker/required gate. In Box prove current Scoped install, authenticated positive/negative smoke, Create/Peer/Delete, restart and retirement/reinstall where supported; observe limitations. Rehearse how S2/S2.5 and installed lanes can start Scoped, identify mandatory test conversions and sequence commits. Update inventory with evidence and decisions before broad deletion. No new production behavior claimed. |
| P1 — runtime and schema | Extract shared guard/conntrack/native helpers first. Convert affected baseline/fixtures and callers in coherent slices alongside runtime removal; one supported controller and operation path for Create/Delete/Peer, one schema, preserve replay/recovery/ownership. Prove clean DB, real binary, negative old-state refusal, all affected native and fault tests; no Legacy dispatch in final target. Any temporary bridge must be tracked and eliminated by P5, never a new compatibility commitment. |
| P2 — install/retire | Make Scoped the sole install/serve contract; unify wrappers, native preparation, guard/helper startup, maintenance plans, full normal/offline retirement, resume/purge and reinstall. Preserve shared Incus and inherited policy; verify with clean, populated, nested and stale/foreign/unknown fixtures. Move release-installed startup tests to the same contract; archive old maintenance paths after replacement. |
| P3 — operator surface | Reconcile CLI dispatch, help, internal fixed leaves, Doctor/Recover, MCP descriptions, host_status, error/navigation, all auth and operation outputs. Remove obsolete command/flag paths and dependent taxonomy while preserving safe refusals and independently verified recovery. Reconcile service unit and wrapper invocations. |
| P4 — engineering | Consolidate remaining workflows and tests by durable capability rather than phase number; preserve exact quality status and coverage map, release dependencies and evidence contracts. Synchronize live docs, BDH index/profiles/skills/evals and archived references. Retire duplicate tests only with replacement witness; verify docs-only and mixed-change CI policy. |
| P5 — independent acceptance | Fresh checkout, independent review of complete diff and deletion proof; real clean installation and full MCP/CLI inventory; nested and normal lifecycle, fault/packet/recovery/purge, foreign preservation, source/race and dual architecture. Reconcile all mandatory acceptance IDs, scope limits and #170 handoff. No pending or failing required CI on exact final HEAD; keep Draft until independently reviewed and owner authorizes merge. |

Review [acceptance](CONSOLIDATION_ACCEPTANCE.md) for final constraints and [progress](CONSOLIDATION_PROGRESS.md) for resumable work. Do not set arbitrary deletion or workflow-count targets.
