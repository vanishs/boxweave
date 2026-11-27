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

## Active delivery plan: P2-P5 (2026-11-27)

P1 development is closed by owner direction. Product checkpoint ff3c20e;
acceptance baseline 0e903a2a5d69ca35287109d4a1ef9c35bc2688c7.
[R5 review](https://github.com/vanishs/boxweave/pull/191#issuecomment-7216191395)
and [independent H review](https://github.com/vanishs/boxweave/pull/191#issuecomment-7216012723)
establish bounded evidence, not formal P1 PASS. See the
[handoff ledger](CONSOLIDATION_PROGRESS.md#active-p2-p5-handoff).

Execution contract:

- P2 R0 is READY, not started. Incoming Agent prepares read-only and
  waits for the owner. "P2 R0" starts that stage; "P2" starts the full
  phase. Each R has two or three outcome instructions. Complete routine
  fixes and verification autonomously; reassess dependencies at each exit,
  combine tightly dependent slices where justified and record why.
  Use phase-wide continuation and Finish Gate. No extra command per test.
- Use the exact published baseline in an isolated worktree of the existing
  owned Box; preserve old P1 worktree/history. Verify native fixture identity
  and authority before effects. Continue independent development while CI
  runs; record exact proof, actual blockers and next action. Do not repeat
  unchanged failures or manufacture work. Scope/equivalence decisions and
  real Host effects need separate authority.
- Carry P1 H (retained composed witness and test disposition), C04 (hosted
  WAN positive or owner-approved bounded equivalence), C11 (release S2 and
  old N2 gates) and C01/final CI. H remains a P1 acceptance obligation,
  not new P2 product scope. Historical refusal is not a product oracle;
  no repeat, delegation or bypass of refused operations. Closing #194
  is not proof. Reuse only legitimately obtained applicable evidence.
- P2 supplies one installed contract; P3 aligns operator/internal callers;
  P4 consolidates proven consumers; P5 independently validates all gates.
  Full Scoped offline retirement is P2 C07. Keep one Draft PR, no merge,
  tag, release or #170 changes without separate authority. BDH development
  only by explicit owner request. No new product features or gate waivers.

### P2 — single installed lifecycle and retirement contract

**R0 — Investigate critical path**
1. Reconcile exact source, N2 26/28, S2/S25 PASS and existing normal
   retirement evidence. Trace install.sh, uninstall_core.py, internal
   install/observer, CLI, units and packaged wrappers. Confirm actual
   offline dependencies, release S2 setup and authorized clean fixtures.
2. Freeze C06/C07/P2-C09/C10/C11 exits and shortest R1-R4 sequence:
   fresh, populated, stale/declined, foreign/mixed, nested, stopped,
   interrupted, purge/reinstall. Reuse existing proof; record scope
   decisions and inherited H/C04 separately.

**R1 — Fresh installation and serve**
1. Align reviewed Scoped fresh wrapper, stage/activate, guard/helper,
   readiness and packaged manifest. Preserve compatible Incus reuse,
   signed dependencies, exact preimage, stale/declined/old-state refusal,
   OAuth continuation and existing password; no adoption/reset.
2. Verify focused positive/negative checks and affected installed proof.
   Reuse unchanged schema/Forward/S25 evidence. Exit: single supported
   fresh contract, no upgrade feature or second installer.

**R2 — Scoped offline retirement**
1. Trace common observer/archive/purge and Legacy-only offline eligibility.
   Define minimum supported stopped-controller scope with exact identity,
   independently observed native scope and durable original/archive proof.
   Keep uncertain/unsupported cases fail-closed; bounded power original
   archival is not settlement.
2. Implement and verify coherent Scoped/common slices preserving UNKNOWN,
   claims, foreign resources, interruption/resume, mixed/stale refusal and
   archive integrity. No version-only fix or Legacy execution restoration.
   Exit: verified source candidate and retained negative tests.

**R3 — Installed retirement lifecycle**
1. Verify actual normal and supported offline plan/review, interruption/
   resume, exact original/archive and native poststate, foreign preservation,
   purge/reinstall and inherited LXC policy in authorized disposable labs.
   Reuse current normal proof where unaffected; exercise new offline path.
2. Disposition old N2 legacy/legacy-offline lanes only with explicit
   obsolete-oracle decisions and equivalent generic maintenance/UNKNOWN
   witnesses. Keep old gates until then. Exit: C07 evidence or named gaps.

**R4 — Installed and release consumers**
1. Convert release S2 to proven installed Scoped contract. PR-only
   crosswalk binds PR #191/pull_request and cannot run unchanged for
   release. Bind exact candidate/source/event/artifacts; preserve 72
   requirements, native S2/S25 gates, signatures and publish dependencies.
2. Verify negative calibration and applicable independent CI without
   tag/publish. Any release-only proof needing separate authority stays
   open. Exit: N2 and release S2 dispositions evidenced, no fake green.

**R5 — P2 acceptance**
1. Independently review C06/C07 and P2 parts of C09/C10/C11, artifact
   identity, supported lifecycle and required negatives.
2. Correct defects and revalidate. Distinguish P2 implementation exit
   from inherited formal gates; record exact P3 entry and open obligations.

### P3 — operator contract, diagnostics and recovery

**R0 — Inventory actual surface**
1. Enumerate actual CLI dispatch, help, flags, wrapper/unit callers and
   installed 23-tool MCP discovery. Classify public, fixed internal,
   retained Incus dependency lifecycle and obsolete Legacy paths.
2. Bind C08 and P3 parts of C03/C05/C10/C11 to P2 proof; identify finite
   R1-R4 work and targeted tests, not an arbitrary command-count goal.

**R1 — CLI and service entries**
1. Align fresh serve/maintenance, public command/flag contract and fixed
   systemd/wrapper calls; remove only obsolete dispatch after caller
   proof. Preserve Incus native prepare/rollback/reconciliation distinct
   from obsolete BoxWeave upgrade/rollback.
2. Verify built binary help, actual positive/negative invocations,
   old-path refusal/no effect and packaging callers. Exit: one contract.

**R2 — Doctor and maintenance**
1. Align root/restricted diagnostics to schema 7, exact identity, capacity,
   native state and original evidence. Keep read-only inspection, truthful
   unavailable/UNKNOWN and redaction.
2. Verify healthy, stale, unavailable, malformed/mixed and foreign fixtures.
   Exit: actionable evidence, no false health or mutation.

**R3 — Recover and MCP navigation**
1. Align reviewed recovery, typed errors and navigation with Scoped
   originals; preserve authorization, stale/declined review, claim safety,
   no redispatch and output privacy. Retain 23 tools and SDK contract.
2. Verify capacity parity (host_status/Doctor), selected actual recovery
   and refusal with original IDs, exit and independent poststate.
   Reuse unaffected ACK/WAL/SDK proof. No new MCP tool.

**R4 — Caller convergence**
1. Update affected tests, units, wrappers, examples and operator docs;
   map old leaves/oracles to retained behavior or obsolete requirement.
2. Run focused/adjacent and applicable independent installed checks;
   fix concrete issues. Exit: no unaccounted live old caller.

**R5 — P3 acceptance**
1. Independently review C08 and P3 parts of C03/C05/C10/C11, actual
   operator inventory, diagnostic/recovery and negatives.
2. Correct and revalidate; publish coherent evidence and P4 cleanup
   inventory with remaining obligations, no waiver.

### P4 — engineering, tests, docs and BDH

**R0 — Finite cleanup ledger**
1. Reconcile P2/P3 proofs and actual Go/SQL/CLI/systemd/embedded scripts,
   workflows/triggers, tests/markers and docs. Classify active, retained
   negative, historical, replacement pending and obsolete.
2. Prioritize smallest useful reductions and C10/C11/C12; retain all
   inherited gaps including H. Exit: finite caller/witness ledger.

**R1 — Retire residual bodies and duplicated tests**
1. Remove only proven unreachable obsolete bodies or redundant tests
   after caller/equivalent-witness review. Keep shared Scoped helpers,
   fail-closed refusal and historical evidence; no filename-based removal.
2. Verify build, affected tests, reachability and negative calibration.
   H test disposition/real composition stays required; no retry of
   refused action. Exit: accounted retirement, retained coverage.

**R2 — Workflow and release plumbing**
1. Reuse P2 installed contract and witness checks; align triggers and
   exact provenance. Preserve quality-gate, dual architecture, release
   S2/S25, signature/asset checks, stable inventory and #170 scope.
2. Verify docs-only/mixed/release classification, nonempty discovery,
   missing/wrong/duplicate proof refusal and applicable actual CI.
   Exit: no unaccounted live gate or weakened acceptance.

**R3 — Live docs and BDH synchronization**
1. Align install/retire, Doctor/Recover, MCP and examples to verified
   behavior; archive stale phase material without erasing failures.
2. Synchronize existing BDH entry/index/profiles/skills/Eval and resume
   paths, preserve eight systems and dual loops. Follow agent-agnostic
   evolution; no per-Agent adapter, new product tool or speculative feature.

**R4 — Engineering and observed use**
1. Run existing deterministic checks and applicable Eval; observe ordinary
   activation/resume and work-bound learning when available. Retain #193
   R3/R4 fresh-use, R6 utility NOT MET, L014 REOPENs and V5 unless proven.
2. Record actual costs, scope and deferrals; separate owner-triggered BDH
   optimization from synchronization. Exit: accurate use evidence,
   not assumed efficiency or equal Agent capabilities.

**R5 — P4 acceptance**
1. Independently review C10/C11 and engineering C12, caller/marker
   dispositions, live docs, actual workflow use and retained gates.
2. Fix and revalidate; freeze P5 candidate and unclosed ledger.

### P5 — independent final acceptance and #170 handoff

**R0 — Freeze final candidate**
1. Resolve exact remote head/base and independent clean checkout; reconcile
   C01-C12, P1 H/C04/C11, phase exits, original operations, authority and
   actual CI. Reuse only valid scope/identity evidence.
2. Freeze final matrix/artifact identities and request necessary owner
   equivalence/physical authority. Exit: executable matrix and explicit
   blockers, not assumed proof.

**R1 — Independent source and retirement audit**
1. Review full diff, shipped inputs, dispatch, schema, ownership, no
   Legacy fallback, packaging, public/internal contract and fail-closed
   invariants. Trace retired callers and shared helpers.
2. Correct genuine defects and invalidate affected evidence. Exit:
   independent review with no undisposed release-critical defect.

**R2 — Installed and native functional proof**
1. Verify exact clean installed candidate, 23 MCP tools, CLI, ownership,
   lifecycle, nested functional scope, retirement/purge/reinstall, actual
   packet/recovery and no-replay. Reuse exact unaffected proof.
2. Close C04 via legitimate independent WAN positive or owner-approved
   bounded equivalence, not assumed environment cause. Separate local
   HTTPS toggle from Hosted UNVERIFIED; keep physical/reboot/LAN/IPv6/
   storage and recursive-isolation limitations explicit for #170.

**R3 — Full exact-candidate regression and CI**
1. Run applicable format/vet/module/race/Python/build, CGO SQLite,
   amd64/arm64, required quality and independent CI; verify actual jobs
   and artifacts. H composed proof and test disposition must be valid,
   not old failure, skipped or vacuous PASS.
2. Correct issues, revalidate affected proof and require final exact
   head. No publish trigger. Exit: mandatory gates PASS or named
   FAIL/BLOCKED/UNVERIFIED with next legitimate action.

**R4 — Evidence and #170 handoff**
1. Reconcile every C01-C12 and old marker/leaf with exact evidence,
   retained behavior or explicit obsolete disposition. Reassess H,
   C04 and C11 on actual proof; review bounded BDH/utility status.
2. Prepare #170 handoff with exact source/artifacts, supported contract,
   limitations and remaining release-specific obligations. Keep stable
   upgrade/rollback and physical acceptance distinct, no silent transfer.

**R5 — Final independent verdict**
1. PASS only when all mandatory #191 gates and independent review are
   satisfied on final candidate; otherwise precise remaining obligations.
2. Persist final evidence/readiness and preserve workspaces/history.
   Keep Draft and await separate merge authorization; publication in #170.

## P1 convergence recovery: R0-R5 (active)

Historical entry retained for existing links. P1 development is closed;
use the active P2-P5 plan above. Original exits and formal P1 obligations
remain unchanged.

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

### Owner-approved P1/P2 scope boundary (2026-11-26)

P1 remains the single Scoped runtime and schema-7 cutover, including
necessary installer/consumer dependencies and complete retained core
regressions. The full offline retirement implementation migration
from the old Legacy leaf to Scoped belongs to P2, as originally planned.
Preserve UNKNOWN originals, archive integrity, foreign resources,
interrupted resume and purge obligations. Do not claim equivalence to
live drain. A dependency genuinely required for P1 stays in P1.
This is sequencing, not waiver: old gates remain until explicit
applicability/replacement disposition, R4/R5 require all mandatory P1
evidence, and final P5 cannot accept outstanding P2 obligations.
The two blocked Install regression conversions are not deferred by this
decision. No safety exception, product publication or release approval.

### Owner-directed development continuation (2026-11-27)

Owner decision (2026-11-27): P1 implementation is considered development-complete for scheduling; proceed with independent P2 development. This explicitly defers the unresolved Install composed witness in #194, not a PASS, waiver, or permission to repeat a refused operation. The existing 577 PASS / 66 SKIP / 2 FAIL aggregate and Race exit 1 remain recorded. Retain the original failed tests and all safety assertions. Exact-candidate regression, native and CI evidence, C04 hosted equivalence if needed, and old-gate applicability/replacement remain open before formal P1 acceptance, merge or release. This Schema 7 development checkpoint is not an accepted or release candidate. See [#194](https://github.com/vanishs/boxweave/issues/194) and [R5 review](https://github.com/vanishs/boxweave/pull/191#issuecomment-7212654941).

This supersedes the earlier no-deferral decision for development sequencing only. Formal R0-R5 exit standards remain unchanged.

### Joint R2/R3 execution (remaining work)

Use the [phase-wide continuation projection](../../dev/skills/task-resume/SKILL.md#phase-wide-continuation-candidate)
from the existing task history after each material checkpoint. The live
ledger must point to all remaining R2/R3 gates; a local blocker or one
completed slice cannot end the pass while another required action is ready.

Use the three outcome groups in the [live ledger](CONSOLIDATION_PROGRESS.md#current-p1-recovery-ledger),
not one task per failed test. These are internal work packages, not new
stages or owner commands.

1. Close history/retirement, Peer/Install and dependent Delete/recovery
   by shared Scoped fixtures and explicit behavioral dispositions. Batch
   related inspection, edits and focused regression; repair genuine defects.
2. Finish minimum schema-7 diagnostic, install/serve/recovery/drain and
   S2/S2.5/N2 consumer wiring required by the cutover; preserve all retained
   oracles and the 72-marker crosswalk. Defer unrelated cleanup to R4/P2-P4.
3. Run aggregate applicable checks on one coherent candidate; update the
   single ledger, publish only a verified tree, and compare actual R2/R3
   exit conditions. Use the Finish Gate; do not stop at a work package,
   failure, checkpoint or pending CI while safe necessary work remains.

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

The active P2-P5 plan above controls execution. The table retains overall
scope; P1 development is closed, formal acceptance open.

| Stage | Work and exit checkpoint |
| --- | --- |
| P0 — preflight | Resolve exact source and CI baseline, existing authorized development Box and clean/nested fixture. Build call/reference graph across Go, SQL, CLI, systemd, shell/Python, workflow YAML, docs and BDH. Inventory every affected test/marker/required gate. In Box prove current Scoped install, authenticated positive/negative smoke, Create/Peer/Delete, restart and retirement/reinstall where supported; observe limitations. Rehearse how S2/S2.5 and installed lanes can start Scoped, identify mandatory test conversions and sequence commits. Update inventory with evidence and decisions before broad deletion. No new production behavior claimed. |
| P1 — runtime and schema | Extract shared guard/conntrack/native helpers first. Convert affected baseline/fixtures and callers in coherent slices alongside runtime removal; one supported controller and operation path for Create/Delete/Peer, one schema, preserve replay/recovery/ownership. Prove clean DB, real binary, negative old-state refusal, all affected native and fault tests; no Legacy dispatch in final target. Any temporary bridge must be tracked and eliminated by P5, never a new compatibility commitment. |
| P2 — install/retire | Make Scoped the sole install/serve contract; unify wrappers, native preparation, guard/helper startup, maintenance plans, full normal/offline retirement, resume/purge and reinstall. Preserve shared Incus and inherited policy; verify with clean, populated, nested and stale/foreign/unknown fixtures. Move release-installed startup tests to the same contract; archive old maintenance paths after replacement. |
| P3 — operator surface | Reconcile CLI dispatch, help, internal fixed leaves, Doctor/Recover, MCP descriptions, host_status, error/navigation, all auth and operation outputs. Remove obsolete command/flag paths and dependent taxonomy while preserving safe refusals and independently verified recovery. Reconcile service unit and wrapper invocations. |
| P4 — engineering | Consolidate remaining workflows and tests by durable capability rather than phase number; preserve exact quality status and coverage map, release dependencies and evidence contracts. Synchronize live docs, BDH index/profiles/skills/evals and archived references. Retire duplicate tests only with replacement witness; verify docs-only and mixed-change CI policy. |
| P5 — independent acceptance | Fresh checkout, independent review of complete diff and deletion proof; real clean installation and full MCP/CLI inventory; nested and normal lifecycle, fault/packet/recovery/purge, foreign preservation, source/race and dual architecture. Reconcile all mandatory acceptance IDs, scope limits and #170 handoff. No pending or failing required CI on exact final HEAD; keep Draft until independently reviewed and owner authorizes merge. |

Review [acceptance](CONSOLIDATION_ACCEPTANCE.md) for final constraints and [progress](CONSOLIDATION_PROGRESS.md) for resumable work. Do not set arbitrary deletion or workflow-count targets.
