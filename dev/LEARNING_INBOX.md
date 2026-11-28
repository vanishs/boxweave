# Learning inbox

> **Reference only:** Historical candidate and observation registry. It is not an active task queue. Reconsideration and improvement require an explicitly requested scope.

## Deferred opportunity review (current #193 plan)

See the [active BDH plan](../docs/development/BDH_DEVELOPMENT_PLAN.md#find-deferred-opportunities-again)
for the owner-authorized improvement policy. This is the existing navigation
entry for later review of observations, linked evidence and deferred work;
canonical candidate histories remain under [learning](learning/), with
original work and PR evidence linked from those records. Review current
revisions rather than relying on historical entries below. For example,
[L014 revision 39](learning/BDH-L014/00000039.json) retains the latest
recorded transfer residual; further transfer-specific development is
currently deferred, with REOPEN preserved.

During normal work, capture material facts and defer nonblocking BDH
improvements. On request, review the current evidence and propose reuse,
creation, optimization, adjustment, repair or continued deferral. An
opportunity or review is not authority to select or implement a new BDH
change. Use the existing commands and durable histories. On request, use:

    python3 dev/bwdev --json learn next dev/learning --all --brief

For a relevant ordinary task, use the existing filters rather than --all.
The brief view retains audit findings (including nonzero exit), candidate
identity, original evidence, latest matching observation, recorded cost
when present and review/reconsideration information. Read the full view
and learn opportunity before deciding; historical reconsideration is not
a current decision. Without a checkout, read the linked current records
through authorized GitHub tools. No separate queue or background scheduler.

Manual bootstrap queue and conventions. An active PR/issue may hold the canonical candidate instead of editing this file; add a link here only when useful for discovery after the task. Do not duplicate the full observation. At each material failure, correction, handoff or completion, check for reusable value. Record one disposition even if no asset is needed.

Candidate fields: stable ID, work ref and source/evidence link, bounded observation, cause (or hypothesis), affected versions/environment, proposed asset or existing asset to update, validation plan, risk/change class, status and next step. Dispositions: candidate, verified, active, deferred, stale, deprecated, rejected, no-new-asset. A verified historical fact is not a verified new procedure. Larger unrelated improvements are separate tasks; no need to block product delivery for them. Use [learning skill](skills/continuous-learning/SKILL.md).

### Platform safety refusal observation (deferred; #191)

- Original evidence: [checkpoint](https://github.com/vanishs/boxweave/pull/191#issuecomment-7209278701),
  [investigation](https://github.com/vanishs/boxweave/pull/191#issuecomment-7209413770).
  An Install regression test-write request was explicitly refused by the tool
  safety layer. The internal rationale is unavailable; false positive is a
  hypothesis, not a confirmed diagnosis. Distinguish this from ordinary
  invalid admission, resource locks, transport errors and guest test failures.
- Disposition: deferred workflow observation, no new tool or permission.
  For future work prefer deterministic pure fixtures, read-only evidence
  consumers and already reviewed native lab boundaries. Preserve an explicit
  safety refusal; do not retry the denied action through alternate tools,
  encoding or transport. Seek platform review if necessary. Continue
  independent authorized product tasks without bypassing or weakening gates.
- Reconsider after a supported review or further independent real-work
  observations; measure actual refusal type, scope and cost before any
  improvement or promotion. The two Install conversions remain blocked.

## Initial seed from actual prior work

| ID | Observation / source | Disposition and next step |
| --- | --- | --- |
| BDH-L001 | [#188 H5](../docs/development/PR188_H5_DELIVERY.md): missing Git caused 13 Git-dependent errors; missing origin/v1 ref caused an earlier attempt to fail. | Candidate: environment preflight distinguishes Git/base-ref needs per test. Initial environment skill includes these checks; automation and fresh regression not yet verified. |
| BDH-L002 | Same H5 evidence: populated Box caused clean-state fixture failures; two external installer tests required a clean host. | Candidate: fixture capability tags and separate clean CI lane. Initial verification skill prevents treating fixture limits as PASS. Do not alter production admission. |
| BDH-L003 | Same H5 evidence: prior CI was bound to H4B parent; H5 required exact-head candidate and independent CI, with merge-result distinct. | Candidate: deterministic SHA/evidence binding check. Initial verification skill records the distinction; automation/eval pending. |

The original historical report owns actual commands, IDs and results. These observations are not claims about today's installed environment or final CI. Initial seed is intentionally small; mine additional history when a current task needs it.

## First bwdev implementation learning (PR #189)

| ID | Observation and evidence | Disposition |
| --- | --- | --- |
| BDH-L004 | Dedicated Box create dab2374c02b77d1d2e75deab58581a05 failed SAFE_NATIVE_PREFLIGHT_DENIED; the retained authorized lab supported a new isolated clone (a3859c60a98595358707d0d2ca517d3f). | Preserve refusal; recover/reuse a separate worktree or conventional fallback. No safety bypass. |
| BDH-L005 | A large argv source transfer was rejected at admission; exclusive per-file creation in the isolated clone worked. | Candidate: bound transfer size, read original receipt after uncertainty, prefer verified exclusive file transfer for new files. Do not infer an exact size limit from one refusal. |
| BDH-L006 | make invocation 5aaed2e8a6e98b6089779ebf1366bd9e exited 127. The follow-up 46197b71acc1db5a5e3b50b550ddcc49 observed absent Make, Go and origin/v1 while Python/Git were available. | Verified for that Box: optional capability-specific doctor blockers; use direct Python tests without installing tools. Base-ref regression included. |
| BDH-L007 | Earlier installed-lifecycle CI run 45547041460, Legacy job 138979037296: connection refused after test-induced service restart despite PID/UNKNOWN checks. | Candidate fixture timing diagnosis. Add bounded listener readiness observation, preserve original state/identity/authentication gates, rerun independent native CI. Offline regression is not installed acceptance. |

## v0.2 dogfooding

- BDH-L008: original Box staged tree matched the remote v0.1 tree exactly;
  verify both before fast-forwarding, preserve other worktree changes.
  Resume operation 07246fc614004a7a08384d2134c5b98e and sync operation
  2408c75359b5f2d28950f1f53c1a3479. Candidate for future resume helper.
- BDH-L009: task history uses an immutable local revision rather than
  overwriting shared state. Local concurrency regression is not distributed
  coordination proof. Use Git/PR for explicit cross-environment reconciliation.

## v0.3 dogfooding

- BDH-L010: staged and unstaged change sets must be considered separately;
  comparing only the working tree against HEAD can conceal staged changes
  that have been undone in the worktree. Regression in test_bwdev_verification.
- BDH-L011: actual verify run records PASS on the dirty development tree
  as UNBOUND, and missing gofmt as BLOCKED. Evidence operation
  efc3d3a720706260d9a48ee95f9e3957. Preserve this distinction in future
  acceptance automation; independent clean-source results remain separate.

## v0.4 structured learning

- [BDH-L012](learning/BDH-L012/00000001.json): first durable structured candidate from actual CI status; keep as candidate until reviewed. Captured and audited by operation 3e071e56e76d684500fa0c62cf553a62.

## v0.5 self-evolution

- [BDH-L012 revision 2](learning/BDH-L012/00000002.json): implementation and validation attached; candidate remains pending independent review. CI helper, Skill, regression and Eval now implement this learning. Start surfaces it on relevant tasks; audit checks asset drift. Do not implement it twice.

- [BDH-L013 revision 2](learning/BDH-L013/00000002.json): nonempty test discovery contract implemented and exercised; candidate pending review. This Box's builtin unittest already failed for zero tests, so no universal old-runner defect is asserted.

## Universal activation / dual-loop learning

- [BDH-L014 revision 5](learning/BDH-L014/00000005.json) structures historical BDH-L005 (do not duplicate), links real U2 friction, records deferral and rediscovery, and attaches the bounded Box source-transfer Skill improvement. [Revision 6](learning/BDH-L014/00000006.json) reconciles documented #191 reuse; [revision 7](learning/BDH-L014/00000007.json) records a distinct actual #192 transfer reuse. Candidate remains pending review; inspect implementation instead of duplicating it. Neither observation establishes independent Agent efficacy or promotion.

- [BDH-L014 revision 8](learning/BDH-L014/00000008.json): actual independent ordinary transfer investigation and manual pending handoff reconciled against the original Box evidence. [Feedback](../docs/development/BDH_V06_INDEPENDENT_TRANSFER_FEEDBACK.json) and [witness disposition](../docs/development/BDH_V06_RELIABLE_EVOLVE_PLAN.md#e4-independent-ordinary-task-witness-and-reconciliation-2026-11-25). Existing candidate remains open; assess a separate bounded manifest/resume helper after v0.6 acceptance, not an automatic promotion or duplicate capture. The prior independent Agent did not itself commit the structured record.

- [BDH-L014 revision 9](learning/BDH-L014/00000009.json): a second ordinary Agent autonomously began work, rediscovered and used the bounded transfer procedure, appended an exact observation in a separate inbox, and recorded a reconciled local HANDOFF. [Original report](https://github.com/vanishs/boxweave/pull/192#issuecomment-7157664834); [original work history](../docs/development/BDH_V06_RESUME_WORK_HISTORY/00000002.json) imported byte-for-byte. No promotion or complete helper. Actual transport failure and crash durability remain follow-up validation.

- [BDH-L012 revision 3](learning/BDH-L012/00000003.json): P0 closeout after #192 merge recorded actual exact-source CI and native gate separation; [work history](../docs/development/CONSOLIDATION_P0_CLOSE_WORK_HISTORY/00000002.json) reconciled by original local feedback operation. Reuse only, candidate remains open; no new implementation or promotion.

- [BDH-L014 revision 10](learning/BDH-L014/00000010.json): actual P1a edit admission refusal followed by verified exclusive script and short Exec; [work feedback](../docs/development/CONSOLIDATION_P1A_WORK_HISTORY/00000002.json) reconciled. Reuse, no new implementation or promotion.

- [BDH-L014 revision 11](learning/BDH-L014/00000011.json): P1b proactive exclusive transfer and exact readback; [append-only corrected work feedback](../docs/development/CONSOLIDATION_P1B_SOURCE_WORK_HISTORY/00000003.json) reconciled. Reuse, no promotion.

- [Bounded Operation observation candidate](../docs/development/BDH_OPERATION_POLLING_CHECKPOINT.md): actual P1 short-wait repetition and budget exhaustion; owner-authorized verification Skill refinement. Candidate, ordinary-task observation pending; no automatic promotion or proven savings.

- [BDH-L012 revision 5](learning/BDH-L012/00000005.json): [consolidated CI observation](../docs/development/BDH_CI_OBSERVATION_CHECKPOINT.md), candidate refinement; independent benefit pending.

- [Early checkpoint candidate](../docs/development/BDH_EARLY_CHECKPOINT.md): actual P1 interruption and reuse; candidate refinement of existing resume Skill and template. Observe overhead and actual recovery; no proven benefit yet.

- [Finish Gate and compact resume candidate](../docs/development/BDH_FINISH_GATE_CHECKPOINT.md): owner-authorized bounded workflow refinement; regression and Eval evidence, independent review and real-task benefit pending.

- [BDH-L015](learning/BDH-L015/00000001.json): [staged batch candidate](../docs/development/BDH_STAGED_BATCH_CHECKPOINT.md); actual D30 byte comparison, real-task reuse and independent review pending.


## Improvement cycle continuity

- Cycle 2026-11-26-01 (retrospective starting checkpoint): PR #191,
  evidence D24-D31; prior candidate [BDH-L015](learning/BDH-L015/00000001.json).
  [Implementation](../docs/development/BDH_STAGED_BATCH_CHECKPOINT.md)
  and [actual reuse](https://github.com/vanishs/boxweave/pull/191#issuecomment-7206324169).
  Product head 93774b943be882fb15fddd93164a38f7231d1ed0. Local tests
  passed and actual staged delivery used the candidate; export reduction
  85.5% on D31, not a whole-task speed claim. Independent CI/review open.
  Historical plan and research review remains due in the next full cycle.
  Forward candidates: inspect shared fixture callers before conversion;
  reduce repeated remote publication orchestration if permitted and
  measured; reassess V4/V5 residuals from the development plan when
  new evidence supports reopening. Resume P1 from verified current
  head; retain all native and final exact-head gates. No promotion.

- Cycle 2026-11-26-02 (candidate; owner invoked): #191 remote
  1ce5799aa874f61cf6ef95719bd097fd618f7f6c, local 9c76b3394f85d8b4545b9db82b2dd2b7328fd504.
  Window since cycle 01: D32-D36 staged reuse (D36 export -89.6%),
  R0/R1 replan, R2/R3 schema cutover and [local checkpoint](https://github.com/vanishs/boxweave/pull/191#issuecomment-7208165980).
  Staged helper not applicable to deletion/dirty candidate; prior private
  patch exactly recovered in 16c998bcc473b07b8a1cffb0f9ca46cf. CI/polling
  total counts and independent utility not established. Two long argv
  refusals previously observed; transfer candidate V4/V5 remains deferred
  without changed caller-side authorization/integration evidence.
  Reviewed R0 research, BDH plan and dual-loop plan: reuse Git/Go/Python
  and KCS/SRE bounded toil pattern; no external framework or new MCP.
  Selected [failure navigation](go_failure_index.py): repeated manual
  log counting/mapping and absent-test ambiguity; first failure count
  correction 71 (not 67), intermediate 52, latest prior 44.
  Independent local comparison 4cbdef3b48a792d53da78d71bfbd8b7d:
  71/44/43 name counts agree with sed; 44/44 file paths with git grep,
  26 files, 6 missing old names. Compact projection 6278 B from two
  raw logs totaling 48050 B (navigation only, not equivalent full logs);
  new product recheck projection 3935 B from 43422 B raw inputs.
  No measured whole-task speed or nested call savings: prior manual
  map already batched; setup, regex correction and validation add cost.
  Full harness 133/133 and Eval 20/20, cd10d7db75463a1e9fd5a0002101a62e.
  Actual use selected Create pin collision test, converted retained
  assertions with synthetic cross-project bridge (not native authorization),
  fa79d954af9b7de01ec5070d5cff6a67 exit0. Selected 44-test recheck
  1273ed77a06e0154676cb2bad30f4da2: explicit test PASS, regression exit1,
  43 remaining failures; raw log SHA256 507dccb1a4644bbc0ce44c7eb7404fad80273e9dbd46f3b14f596200ea548ffa.
  Forward candidates ranked: (1) reuse index on next R2/R3 batch, verify
  remaining assertions and all gates; (2) shared fixture caller inventory
  only if repeated navigation persists; (3) publication composition and
  V4/V5 only when actual integration prerequisites change. No promotion.
  Bounded cycle complete; original P1 remains open, no merge/release or
  publication of knowingly failing candidate.

- Cycle 2026-11-26-03 (owner-authorized execution remediation, #191):
  [audit](https://github.com/vanishs/boxweave/pull/191#issuecomment-7208593686)
  and original full-regression operation 7397d63915e73c8c20d6931bc589168a
  exposed premature stopping with ready work, repeated test/log orchestration
  and old/new workflow overlap. Prior nested-call total was an estimate,
  not a verified complete trace; no verified quota exhaustion.
  Preserve the existing product worktree; isolate the non-product change
  on the published head. Extend existing go_failure_index.py rather than
  add a scheduler, MCP tool or new task registry. Explicit caller-selected
  test group runs once with private log, original exit, source observations
  and bounded projection. Fake failure, source drift, no-tests, path,
  overwrite and named positive checks passed in
  c65272c4401af72f40d33c2fb150273d. Real old-head selected Go test
  TestS4B3ReadExactPeerCompilerSnapshot reported run/pass 1/1, exit 0,
  source drift false, operation a11ef91c9c666902d03b3b9bcd618ef1,
  log SHA256 6ad5b8b854ab1741362062532637fa308e8f31a5f35cf81cbe23c3cf40c5ca5d.
  One Exec + original state and two streams; includes initial edits/staging,
  not an equivalent-workload comparative benchmark. Record setup cost;
  no measured whole-task utility or automatic finish enforcement claim.
  Active plan/ledger and existing Finish Gate clarify continuation;
  unchanged safety, required CI and native evidence. No product change.
  Next real P1 work is paused by owner until explicitly resumed; measure
  actual nested attempts per closed group, verified C-ID exits, and
  handoff continuity, not document/commit counts. Do not reopen deferred
  transfer integration absent new evidence; no promotion.

- Cycle 2026-11-26-04 (owner invoked): #191 remote c258c67870a8e8249330b41bc10bdd1092968bfb.
  Since prior correction, actual R2/R3 revision10 remained active when
  owner started R4; two premature endings with independent work
  (7211474650, 7211564275). No observed quota boundary. Existing DAG
  mechanism worked for its supplied scope; wrong active scope was the
  missing invariant. Reuse existing task identity and shared CLI, not
  another tracker/client hook. Consistent with agent-agnostic direction,
  existing Beads-like ready query and checkpoint guidance. No V4/V5
  prerequisite change; defer transfer and further tooling.
  Add explicit expected work_ref to queries, transitions, replan and
  workflow status/resume/finish; retain old CLI compatibility.
  Red then green 65a8565ad57967aa076fdc7a78b03331; full 137 BDH
  regressions, Eval24, format1408 PASS 8f13a0b6e9d4c387f007a8ded3d4a662.
  Actual stale R2/R3 history refused with expected pr191-r4; new
  active R4 graph revision3 CONTINUE, two ready groups, inherited
  blocker and waiting dependencies preserved (5ca7313d9a6dd99baa86a73bd739fb56).
  No stop hook or automatic acceptance. Setup/activation costs not
  recovered; 1 actual scoped comparison, not a productivity trial.
  Product reuse: release old S2 and legacy N2 gates still active
  (87c2d13750f46ddbfe9dcfe7fdd13c95). Continue applicable local native
  and source work, retain exact candidate CI and C04. No promotion.
  Remote c258 changes docs; do not overwrite concurrent history.

- Cycle 2026-11-26-05 (owner invoked): #191 parent e233220b3c3a9fa214e79d31e468fc43fcde3832.
  Cycle04 scope guard actually reused in R5; nevertheless R5 response
  ended while Race original pending and three phase review groups still
  ready/working (7212289648). Two wait expiries were not cancellation or
  an observed client quota limit. Existing Finish Gate was not invoked;
  primary failure is execution discipline, not established causal code bug.
  One bounded gap: supplied checkpoint original IDs did not project
  pending output into workflow status/finish without repeated manual flags.
  Reuse client-owned checkpoint and existing Finish Gate; no tracker,
  scheduler, connector adapter or transfer changes. Research/deferrals
  revisited: Beads-like external state, Agentless bounded validation,
  no changed V4/V5 prerequisite; defer new infrastructure.
  Optional output_review PENDING/CONSUMED/NOT_REQUIRED; omitted
  conservatively unreviewed. Union checkpoint and explicit pending IDs,
  retain independent remote review and existing acceptance.
  Red/green 98d0c9f961b750a16e65a8ae034ed9e6; 137 regression tests,
  24 Eval, format1416 PASS 1a38a06ce1c1ecf3b4171a09d5ee7504.
  Actual original Race 302139a752474f6d82309ce837a87c3c terminal guest
  exit1, both streams consumed, unchanged source and same two failures
  (577 PASS/66 SKIP/2 FAIL, zero reported DATA RACE warnings).
  Historical pending checkpoint reconstructed for calibration, not an
  actual saved prior checkpoint; projected original without duplicate
  CLI flag (4fa8a677a7c2d2524577a576b0ba0be9). Real consumed checkpoint
  and active phase revision10 reviewed with Finish Gate
  (07bd766ed5c52af8de1fd5fc027a2a84): pending_output=[], no ready or
  working, REVIEW_REQUIRED/REVIEW_BLOCKERS. Four review groups recorded,
  original Install blocker and dependent exact candidate/P1 exit retained.
  No automatic termination enforcement, remote proof or comparative
  productivity claim; setup costs include new tests and two checkpoints.
  Next product action: legitimate resolution of retained Install evidence
  obligation, then coherent exact candidate gates. No replay or waiver.

## Phase-wide continuation correction (2026-11-26, #191)

Observed repeat premature end after candidate CLI checks:
[prior record](https://github.com/vanishs/boxweave/pull/191#issuecomment-7210337994).
No actual tool quota boundary established. The existing Finish Gate
already forbids early stop but depends on invocation; material work
used ad hoc prose, not a phase-wide actionable task history. Full
finish needs clean delivery evidence, and is unsuitable as the cheap
next-action query. Do not replace it or add a scheduler.

Extend existing task next/transition/status with a continuation
projection, also exposed in workflow summary. CONTINUE includes
WORKING and independent ready PENDING; REVIEW_BLOCKERS shows exact
waiting dependencies and blocked notes; REVIEW_EXIT requires real
acceptance review. Reuse transition output; no new call solely to
repeat it. No permission, platform stop hook or gate relaxation.

Red regression 80a1c222d5ea39bce912aefd540cabc3 exit1 (missing projection);
green f7184ecb6716e0c9c9cb467f5f55c6d4 exit0 (9 task tests, Eval24);
797983b2cfaf81a7c13731374309725c exit0 (136 BDH tests, Eval24, format1413,
YAML48). Actual activation aae7c05dd6320b586bf18a03c2a2595e exit0 in
original P1 Box: history /root/pr191-r23-phase-history revision4,
latest SHA256 815b3ef92a3a89f9f13f9fe32bbc20c9c347f438426ca41ef97532563b5fe9eb.
CONTINUE: startup-native and consumer-source-parity; Install core
regressions BLOCKED, candidate-local-exit waits on those three.
Local runtime/schema evidence recorded, not full native acceptance.
Owner requested correction before P1 continuation; no product change.
Candidate behavior regression and one real navigation observed;
future ordinary development adherence and net time savings unproven.

## Tool orchestration boundary observation (2026-11-26, #191)

Prior pass reported "Code Mode exceeded the maximum number of tool calls"
while reading historical N2 CI jobs in a large batch. Confirmed 16 job
comparisons returned and a subsequent user turn could access GitHub in
the same conversation. This is not evidence of a BoxWeave server quota,
a GitHub rate-limit response, account exhaustion, or a permanently blocked
conversation. Per-script versus per-turn limit and exact threshold remain
unknown; earlier 20/70 estimates are hypotheses, not platform contracts.
Do not probe by exhausting quota or attempt to circumvent it. Preserve
observed results before costly reads, count nested calls and avoid repeated
history reads. Smaller necessary independent batches and existing local
source checks reduce waste; an actual limit must not be interpreted as
permission to retry uncertain effects. Defer further BDH implementation
until authorized; current owner request resumes P1.

### Cycle 2026-11-27 — P2 delivery framing (owner invoked)

Evidence window: P2 R0-R5; exact signed product head 09152ff14b0fb75e73b0492d2a1d171bcc87499d,
[R5](https://github.com/vanishs/boxweave/pull/191#issuecomment-7228099703).
Prior cycle scope/Finish Gate guidance supported explicit R3/R5 signoff
and preservation of H/C04/RC obligations; no comparative intervention
or whole-task efficiency measurement. Staged export reused throughout
P2. R4 original e042a393a3e87f183beac920131e0436 had 13 rows and one
observed batch tool-call limit, then two full rereads for subsets.
Existing [L015 revision 2](learning/BDH-L015/00000002.json) records the
distinct recurrence. No claim of exact platform quota. L012/L014
asset-drift audit findings remain open, not accepted or silently fixed.

Revisited BDH plan, R0 research, dual-loop and agent-agnostic direction.
Select one bounded existing-asset change: indexed delivery frames,
plus append-only inbox delta. Prefer deterministic Git/Python and
original read-only output, not client adapters, runtime bridges,
schedulers, new MCP, or automatic dispatch. Existing frame-free export
and exact tree/parent lease unchanged. Reject oversize rows, duplicate
paths and invalid bounds. Use for multi-batch delivery only.
Historical exact R4 export reconstructed 193197 B; four frames plus
854 B index total 194537 B (+0.7%). Three full reads previously
579591 B; framed once would reduce bytes 66.4% before range overlap,
not a measured whole-task saving. 4d42c6ae6dd8abc21bb597536f2580af
exit0 comparison and Eval 24/24; actual nonempty focused 9/9
c0eaa3baee18e9c246cc06e72f9afcbc exit0. Final full harness and
exact-tree actual delivery to be recorded in PR. Bound learn attach
refused dirty source fc9b57ff9dd70dbd4e2c631459da6fed; preserve rev2,
attach reviewed clean-source evidence at next authorized checkpoint,
do not fake clean binding or create a second source SHA only for status.

Forward priorities: (1) observe actual framing on multi-file P3
checkpoints and compare all calls/bytes/attention; (2) use consolidated
CI read and failure index when applicable, review L012 asset drift
before any promotion; (3) L014/V4/V5 remain deferred pending legitimate
changed capability and independent utility evidence, no quota probing
or safety refusal retry. No new product or harness architecture.
P2 signed in its bounded scope; P3 R0 ready, await explicit stage
instruction. Inherited H/C04/C01/final C11 and actual RC proof open.
