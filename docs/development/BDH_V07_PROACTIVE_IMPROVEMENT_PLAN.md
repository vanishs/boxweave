# BDH v0.7 — proactive improvement execution

> **Reference only:** Historical candidate evaluation. #193 is merged; continuation commands and candidate stages are not active.

**Historical V0-V5 plan and evidence.** The [active BDH development plan](BDH_DEVELOPMENT_PLAN.md) governs the remaining work in #193. Preserve original V4 utility gates, both REOPEN decisions and V5; historical next actions are not a parallel active plan.

Status: V0/V1a CI PASS; V1b and V2a/V2b implemented and Box-tested.
V2c CI PASS; V3c independent rereview PASS at 5a98abb, V3d authorized delivery recorded (episode rev15). See [V3c repair](BDH_V07_V3C_REPAIR.md) and [V3d/V4 trial](BDH_V07_V3D_V4_TRIAL.md). V4 real paired functional PASS but utility gate NOT MET; witness outcomes recorded, [V4c start](BDH_V07_V4C_START.md) feasibility probe complete; independent REOPEN imported, second selected episode implemented and validated, [V4c implementation](BDH_V07_V4C_IMPLEMENTATION.md) independent review pending. V4c-01 independent PASS on a77e49e and technical delivery recorded; [repair evidence](BDH_V07_V4C_REPAIR.md), [V4d pre-registration](BDH_V07_V4D_TRIAL.md), L014 rev38 records independent REOPEN, second episode rev17 outcomes recorded; [V4e investigation](BDH_V07_V4E_REUSE_INVESTIGATION.md) proposed; REVIEW_DUE, no third selection yet. V5 pending. See [V3 progress](BDH_V07_V3_PROGRESS.md). See [V2c progress](BDH_V07_V2C_PROGRESS.md), [V2 progress](BDH_V07_V2_PROGRESS.md) and
[first real selected episode](BDH_V07_FIRST_EPISODE.md).
See [V0 evidence and checkpoint](BDH_V07_V0_BASELINE.md).
Stacked PR based on paused #191 at 3494ae8a35c50c386686c9d9d675c1ae37e1b1b7.
Owner authorized this branch to prepare and develop BDH v0.7 before later
integration back into #191. Parent P1b source checkpoint and its unresolved
native ACK/WAL and nesting gates remain intact. Do not merge or publish
without separate owner authorization. No changes to #191 or v1 in this PR.

Read [investigation and precedents](BDH_V07_IMPROVEMENT_RESEARCH.md),
[v0.6 contract](BWDEV_DESIGN.md), [dual-loop plan](BDH_DUAL_LOOP_DELIVERY_PLAN.md),
[bootstrap](../../dev/BOOTSTRAP.md), [improver](../../dev/skills/harness-improver/SKILL.md)
and [phase protocol](PHASE_EXECUTION_PROTOCOL.md). Reconcile live PR head/base,
CI, source and original Box receipts on each resume. This document is a
design proposal, not an authorization to weaken AGENTS or native gates.

## Objective and architecture

UA remains the universal entry. Solve Loop does normal user work and records
material asset use. Evolve Loop additionally aggregates observed residual
friction, produces a reviewed improvement decision, turns selected work
into a durable bounded task, executes it within authority, tests its result,
requests independent review where required, and links later real usage.
The existing shared capability library remains the source of truth.

Logical flow: ordinary task -> use/feedback -> domain review -> opportunity
-> select/defer/decline -> ready task -> implementation -> independent
verification -> authorized delivery -> later outcome evaluation -> reuse or
reopen. A satisfied review with no remaining useful work is a valid exit.
Separate task success, FEEDBACK_RECORDED, improvement delivery, verified
benefit and promotion. Neither CI nor a reference string proves all five.

Design constraints:

- Python standard library and existing bwdev modules, immutable revisions,
  task DAG, work histories, Git and PR. Prefer extending the existing records
  and task runner; do not implement a competing scheduler, tracker or
  learning store. Reuse existing Skills first, keep bootstrap concise.
- Existing L1-L5 change classes, product boundaries, exact-head gates,
  source/installed-target separation, fail-closed authority and UNKNOWN
  handling are unchanged. No new product code or MCP Tool, daemon, vector
  search, auto-approval, permission escalation, automatic merge or release.
- Selection never executes untrusted candidate text. Work starts only with
  actual scoped authority, observed ready dependencies and bounded effort.
  No separate Agent required to perform small improvements; independent
  review and independent witness still require genuine separation.
- No mandatory work for every observation, every task or every repeated
  helpful outcome. Do not prioritize an unrelated optimization above the
  requested work or invalidate honest HANDOFF/finish for lack of authority.
- Use local-only work state if Git is inaccessible, mark persistence pending,
  preserve evidence and reconcile safely. Do not infer autonomous Git push
  or background execution.

## V0 — reconcile and reproduce before implementation

Use an isolated Box checkout or worktree at the exact current child head;
do not mutate the parent's existing /root/pr191-consolidation checkout,
running controller, partial fixtures or other Agent work. Reuse a safe
existing owned Box only after verifying exact identity and authority.
Compare parent head and CI to current snapshots; record any moving parent
as integration blocker rather than silently rebasing.

Read actual L012/L013/L014 histories and relevant original Operation outputs.
Run current bwdev regression, Eval, audit, and read-only selection of L014
against real domains/signals and a copied fixture with new residual evidence.
Record the actual old selector result, not a hypothetical test PASS.
Reproduce F1 with attached, deferred and stale candidates, and F4 with valid
reuse feedback but an unresolved improvement opportunity. Check whether
current useful behavior is caused by the Skill rather than new routing.
Baseline old helper/manual procedure with real authorized transfer fixtures;
define metrics and negative controls before choosing a new implementation.
Keep product native gates separate and preserve prior history. Deliver
precise reproduction evidence and adjust V1-V5 if a hypothesis fails.

## V1 — distinguish usefulness, residual need and review readiness

Extend existing advisory selection and compatible structured data. Separate
the learning candidate (problem identity and history) from a bounded
improvement episode (specific delta and desired outcome). An attached
implementation can be useful while residual toil or a new failure merits a
different follow-up. Prefer a typed reference to the implemented asset and
specific observation range/revisions over free-form inference. Do not replace
an old implementation with a new claim or count old evidence twice.

Add optional or versioned fields for residual friction/impact, observed
cost with units and provenance, failure signature, applicable environment,
and explicit evaluation uncertainty. Existing observations must remain
readable, with unavailable values unknown. No synthetic backfill.
Aggregate actual distinct evidence; work/attempt labels are not proof of
independence. Distinguish positive reuse with no residual gap from positive
reuse with observed remaining manual toil. Review due signals may include
new meaningful evidence since last decision/implementation, material
misled/missing/regression, applicable stale asset, fulfilled reconsideration
condition or explicit owner review. Avoid automatic promotion, numeric
confidence scores without calibration, hard-coded L014 rules, fixed time
quotas or a repeated-use threshold that forces unnecessary work.

Make the reason and next action observable at plan/next and material
boundary. Do not suppress actionable residual gaps merely because an
implementation exists. Review can select an improvement, preserve reuse,
defer with concrete reason/condition, or reject with evidence. New evidence
must cause relevant reconsideration rather than silent permanent deferral;
a review may still legitimately defer. Keep discovery bounded and
context-specific; tiny read-only guidance stays lightweight. Document
behavior for ACTIVE/VERIFIED assets with fresh issues; do not lose them
merely because they are excluded from the old open-candidate queue.

Exit: deterministic tests for old records, unknown metrics, exact deduplication,
helped vs residual friction, new evidence vs already reviewed evidence,
attached+deferred+stale+active cases, relevance and no false positives;
current v0.6 feedback and safety tests remain green.

## V2 — durable improvement selection, assignment and continuity

Use the existing task/checkpoint system and learning revisions to represent
selected improvement work with stable ID, candidate/revision, exact motivating
evidence, bounded objective, affected asset(s), authority/risk, dependencies,
acceptance criteria, baseline and expected outcome, estimate/budget, state,
and next safe action. Make one logical episode idempotent across processes;
do not create another record on each task or duplicate existing implementations.
Preserve append-only decision/assignment/evidence history; use expected
revisions and explicit Git reconciliation. Concurrent work, changed source,
lost receipts, stale claims, unavailable persistence and interrupted review
must fail safely rather than replay uncertain effects.

Proposed lifecycle (names may be adapted to existing task types): review
due -> selected/ready -> active -> validation/review pending -> delivered
-> benefit verified or reopened; deferred/rejected/blocked/unknown have
explicit reason and next condition. Distinguish logical episode closure
from candidate VERIFIED/ACTIVE and product finish. Bind implementation
and evidence to work/attempt/revision and exact source; old attachments
cannot satisfy new implementation claims. Read-only status/plan is never
a mutation. No implicit dispatch, new Agent or owner permissions.

At material boundaries, reconcile relevant review-due items into a durable
decision or explicit pending/blocked handoff; a feedback-only PASS is not
improvement completion. Preserve a separate status for opportunity review,
not a global improvement requirement for every ordinary task. An unrelated
backlog must not block normal delivery; an unaccounted material decision
should not be reported as a fully reviewed Evolve Loop. Keep the boundary
workflow compact and offer reference-derived observation fields if useful
for reducing P1b transcription toil while retaining v0.6 exact association
and immutable corrections. No automatic rewrite of prior history.

Exit: deterministic tests for selection, no-change, deferral with a later
reconsideration, one selected task per evidence range, separate process
resume/claim reconciliation, budget and scope limits, untrusted input,
duplicate/conflicting revisions, unavailable Git, no false completion,
historical v0.6 compatibility and current task continuation.

## V3 — implement and validate a real demand-driven improvement

Use L014 as the first real domain investigation, not a hard-coded acceptance
shortcut. It already has seven observations and one historical Skill
attachment, including positive proactive reuse in P1b. The owner has now
authorized bounded v0.7 improvement work; do not assume permission for
product protocol changes. A separate complete production transfer system
was deferred in v0.6. Measure the residual manual work with existing
authorized RC14 tools before choosing the smallest effective delta.

Preferred candidate if evidence supports it: a client-side manifest/planner
or small reusable transfer helper that reduces manual source-transfer
assembly/verification and supports safe resumption across calls. It may
be an existing Skill refinement plus deterministic tooling. No new MCP
Tool, arbitrary host shell, bypass of admission or native exclusive-create
guarantees. Inspect actual tool discovery and original RC14 version; retain
measured limits (4096 argv bytes, 64 KiB raw create) only where verified,
check current capabilities. Persist intent, per-part length/digest,
idempotency keys and original receipts, whole digest, ordering, safe
exclusive final publication, explicit unknown/conflict recovery. Reuse
identical validated chunks; refuse mismatched/missing/short parts and
unproven overwrites. Do not replay unknown outcomes or claim transport,
controller restart or power-loss durability from a between-call simulation.
Consider a planner plus existing connector calls if the actual surface
cannot invoke a portable client. Avoid dependency on unavailable secrets
or Git push credentials. Retain manual fallback.

Let the actual V1/V2 mechanism surface, select and record the episode;
do not manually inject a preselected success or fabricate prior learning.
If L014's baseline shows no sufficient remaining benefit, record that
decision and choose another real justified gap (e.g. observed feedback
transcription or CI evidence retrieval) rather than inventing a tool to
pass V3. Record why the selected scope is worth its maintenance cost.
Run Box-first targeted tests, adversarial regression, Eval and applicable
format gates; exact implementation and validation evidence linked under
work-bound v0.6 rules. Persist the actual revision and code with ordinary
reviewable Git/PR process. Keep candidate status honest.

Exit: real problem -> evidence -> new episode -> selected ready task ->
implementation -> validated results -> durable authorized PR delivery.
New implementation must be distinguishable from the historical Skill
attachment. If only a recommendation or manual pending handoff is possible,
mark this stage pending; do not count a synthetic fixture as real execution.

## V4 — independently measure utility and close the loop

Use matched old-v0.6 versus new-asset runs on bounded comparable tasks with
the same target, input sizes, model/surface where observable, authority,
acceptance oracle and resource conditions. Define the pass oracle and
primary metric before testing. Count logical MCP/tool calls including
failed/admission calls, retry and manual orchestration steps, user/Agent
interventions, elapsed time and tokens only if observable. Report raw
counts, denominators, variance and unknowns, not invented savings.
A useful helper can reduce planning/coordination while not reducing raw
transport chunks; distinguish metrics. Retain byte integrity, refusal and
native safety checks, no regressions on irrelevant/tiny tasks. Compare
both normal and interruption/refusal cases. Use repeated matched trials
where feasible; distinguish synthetic mechanism tests from real task utility.

A fresh independent Agent with ordinary task prompt and the existing UA
locator should discover and actually reuse the new asset without being
told which candidate or tool to choose. Preserve original plan, work,
tool outputs, outcome and durable feedback; verify exact published records.
A second ordinary task or independent held-out relevant task should
confirm a material benefit relative to baseline or identify a remaining gap.
Do not claim full autonomous evolution from a prompt explicitly asking to
build the helper, a controlled fixture, Git-only reconciliation by another
Agent, or a green test. Report what the originating Agent itself persisted
and what needed authorized handoff. Scope benefit to tested tasks. If
benefit is absent or regresses, reopen, improve or reject with evidence;
no forced promotion. Any VERIFIED/ACTIVE transition must obey existing
review/authority requirements and exact asset identity.

Exit: independent evidence of one genuine improvement selected from actual
feedback, actual implementation, and later beneficial discovery/reuse
(or honest open blocker if no valid witness); demonstrate durable review
and next action on no-benefit/blocked cases. No unsupported universal
efficacy or unattended execution claim.

## V4e — runtime adaptation checkpoint (2026-11-25)

See [investigation](BDH_V07_V4E_REUSE_INVESTIGATION.md) and
[actual adaptation evidence](BDH_V07_V4E_ADAPTATION.md).
The independent second REOPEN remains binding; no utility PASS.

| Slice | Current disposition |
| --- | --- |
| Feedback CI regression | Fixture drift fixed at 52cd4c2; pinned rev33 and new rev38 cases; Box 20/20 and Eval14 PASS, changed-file format PASS; exact-head CI required |
| V4e-01 | Actual authorized read-only MCP adapter works; repeated callable loading across fresh invocations BLOCKED on current surface |
| V4e-02 | Immutable Git evidence reread and original identity PASS; automatic private caller-state binding to real MCP and safe interruption remain BLOCKED |
| V4e-03 | Not selected; defer selection until supported runner, legitimate authorization and trusted state integration proven |
| V4e-04 | Not started; independent actual use and matched net benefit required |
| V5 | Not ready; exact-head CI and independent utility/review gates open |

L014 [revision 39](../../dev/learning/BDH-L014/00000039.json)
records the actual adapter/state evidence and blocker; candidate remains
CANDIDATE/REVIEW_DUE, second episode rev17 unchanged. Observation
publication and repeat feedback suite 20/20 PASS, audit clear, original
Box Operation d32cbb30a3edcf9cb8fc945cb88b8338 exit 0.
No third selection or promotion. Full Box Python run had one unrelated
clean-installer fixture failure in the populated lab and four skips;
gofmt unavailable, no Go changes. CI commit submitted; confirm actual
new head run and clean source gates, do not count prior success.

Next: on a supported caller surface, configure an independently
authorized revocable MCP client and persistent reviewed runner through
the normal platform authorization flow. Test the same pinned callable
twice with short arguments and real harmless MCP reads, then validate
caller-private CAS intent/receipt persistence, interruption, stale
writer, tampering and UNKNOWN refusal. If unavailable, preserve manual
fallback and immutable sanitized evidence; do not build a new transfer
tool or use Git as a dispatch lock. Reconsider V4e-03 only on that
demonstrated integration and a credible benefit hypothesis.

## V5 — integration readiness and independent L4 review

Run all preexisting regression/Eval, new targeted and adversarial tests,
learning audit, applicable format and exact-head CI. Verify current parent
and child heads, worktree, exact source/tree, both Operation streams and
guest exit codes. Independently review selection/reconsideration,
record integrity, scope/authority, idempotency, feedback protocol,
implementation evidence, comparative results, ordinary-task witness and
CI associations. Reviewer must not be implementer; failures remain open.
Stacked child push CI is not v1 parent PR or native/release acceptance.
Do not waive #191's original P1b native ACK/WAL, P0 nesting or later gates.

Report per-stage status (completed, CI pending, testing, next development,
not started or blocked), actual evidence, pending decisions and next
safe action. Request owner integration only when all scoped gates pass.
Never merge #191 or publish. Keep historical failures, plan updates,
learning disposition and execution log current; no duplicate observations
or promotion to satisfy a metric.

## Initial progress and reusable instruction

| Stage | Current checkpoint |
| --- | --- |
| V0 | Box baseline complete; CI PASS |
| V1 | V1a CI PASS; V1b durable decisions Box PASS, child CI pending |
| V2 | V2a bounded selection/ready DAG and V2b guarded lifecycle Box PASS; V2c terminal review, durable handoff and local contention CI PASS |
| V3 | Independent rereview and exact reviewed-head CI PASS; original FAIL preserved; actual delivery rev15, L014 rev24; ready benefit |
| V4 | Two real REOPEN decisions; V4d functional PASS but utility not proven. L014 rev39, second episode rev17; V4e-01/02 partial evidence PASS, executable loading and trusted state binding BLOCKED; V4e-03/04 not started |
| V5 | Not ready: independent utility and exact-head CI open |

Continue from the current PR checkpoint. Follow canonical BDH; reconcile
head/base, prior work, original Box receipts and CI. Work Box-first,
iterate and verify each ready bounded slice, submit coherent evidence and
code to CI, continue safe independent work during CI and fix confirmed
failures. Update plan/log/learning, report remaining stages and decisions.
Do not touch paused #191, merge or publish; stop at owner decision or a
genuine blocker. This instruction is reusable at any stage.
