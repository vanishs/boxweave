# BDH development plan — simplify, reuse, verify

Status: owner-directed planning checkpoint, 2026-11-25. Implementation is
NOT_STARTED under this plan; wait for owner instruction to begin.
Continue in [PR #193](https://github.com/vanishs/boxweave/pull/193), on
feat/v1-bdh-proactive-improvement-20261125. Do not open another PR.
Parent #191 stays paused at 3494ae8a35c50c386686c9d9d675c1ae37e1b1b7.
Retain the BDH (BoxWeave Development Harness) name and bwdev CLI.

## 1. Goal and boundaries

The user tells the Agent what to do, not how to use BDH. Help even a
low-effort Agent deliver reliable engineering work with less reasoning,
context, repetitive coordination and user intervention. Move deterministic
complexity into engineering tools; keep semantic decisions, authority,
novel problem solving and independent judgment with the Agent.
Routine observation and recording should be deterministic wherever possible.
New or modified BDH capabilities require explicit owner authorization for
implementation; ordinary product work does not authorize a side project.

Validate this increment in the existing ChatGPT environment with authorized
GitHub and BoxWeave MCP access and Box-first execution where useful. Defer
other Agent integrations. Do not create a general execution platform,
credential bridge, new database, daemon, scheduler or product MCP Tool.
Retain the 23-tool product boundary, canonical bootstrap, manual fallback,
original UNKNOWN, permission, exact-source, independent review and CI gates.

### Small effort, substantial effect

Use the owner's 10% effort / 60% effect principle as a prioritization
heuristic, not a measured percentage, code quota or safety exemption.
Reuse existing tools, schemas and evidence before adding anything.
Target the few observed recurrent actions with disproportionate cost.
Prefer short inputs, compact outputs, one small composition over new
primitives, and one canonical fact over repeated transcription.
Stop when marginal benefits no longer justify implementation, setup,
maintenance and risk. Do not finish every historical feature for its own sake.

| Design input | Minimal adaptation | Defer |
| --- | --- | --- |
| [Mozilla Mach](https://firefox-source-docs.mozilla.org/mach/index.html), [Chromium depot_tools](https://chromium.googlesource.com/chromium/tools/depot_tools/) | Existing thin bwdev dispatcher and pinned scripts; simple command entry and observable results | Installing frameworks or a general environment platform |
| [KCS v6](https://library.serviceinnovation.org/KCS/KCS_v6/KCS_v6_Practices_Guide), [Google SRE toil](https://sre.google/sre-book/eliminating-toil/) | Flag while working, aggregate recurrence, selectively improve | Mandatory second task or new asset on every job |
| [DevEx](https://doi.org/10.1145/3610285), [process mining](https://tf-pm.org/resources/manifesto) | Measure observable friction and identify repeated operations from existing events | Telemetry service or full event analytics platform |
| [SWE-agent](https://arxiv.org/abs/2405.15793), [Agentless](https://arxiv.org/abs/2407.01489) | Bounded deterministic interfaces and simple workflow | New Agent framework |
| [SkillsBench](https://arxiv.org/abs/2602.12670), [repository context evaluation](https://arxiv.org/abs/2602.11988) | Minimal relevant context and independent benefit evaluation | Asset quotas or imported benchmark claims |

These are design inputs, not proof of BDH utility. Earlier detailed
research remains in the [original design](DEVELOPMENT_HARNESS.md),
[toolbox design](BWDEV_DESIGN.md), [dual-loop plan](BDH_DUAL_LOOP_DELIVERY_PLAN.md)
and [v0.7 research](BDH_V07_IMPROVEMENT_RESEARCH.md).

## 2. Reconciled baseline and open gates

Starting HEAD 1cd8017eb83775ef01f42f2455eda53d78441159:
[quality 45716607486](https://github.com/vanishs/boxweave/actions/runs/45716607486)
and [candidate 45716607072](https://github.com/vanishs/boxweave/actions/runs/45716607072)
completed successfully. Prior-head CI is not current-head CI, native/release
acceptance or proof of Agent utility.

| Prior plan or stage | Disposition |
| --- | --- |
| #189 bwdev v0.1-v0.5.1, UA U1-U4 | Implemented and merged; preserve scoped acceptance, broader actual-use efficacy still open |
| #192 v0.6 E1-E4 | Reviewed and merged into #191; actual ordinary local feedback and durable handoff demonstrated; no universal automatic persistence claim |
| #193 V0-V3 | Completed bounded phase acceptance; no repeat |
| V4a/b | Functional PASS, benefit not met, first independent REOPEN |
| V4c-01, integration and technical delivery | Independent scoped PASS at a77e49e; no benefit claim |
| V4d | Functional PASS in A/B/C, total MCP calls 73/250/81; C discovered but did not execute the corrected template/planner; second independent REOPEN |
| V4e-01/02 | Read-only evidence bridging and local feasibility; reusable callable and automatic caller-private MCP journal not proven on current connector surface |
| V4e-03/04 | Third episode not selected; defer further transfer-specific development, actual-use benefit remains open |
| V5 | Final independent review and exact-head CI open |

Preserve [first REOPEN](https://github.com/vanishs/boxweave/pull/193#pullrequestreview-6063576130),
[V4c review](https://github.com/vanishs/boxweave/pull/193#pullrequestreview-6064325816),
[second REOPEN](https://github.com/vanishs/boxweave/pull/193#pullrequestreview-6065876762),
[L014 rev39](../../dev/learning/BDH-L014/00000039.json) and second episode rev17.
No retroactive changes, promotion or third selection. Retain
[ZCode local feasibility](https://github.com/vanishs/boxweave/pull/193#issuecomment-7177584066)
as deferred research, not production code or evidence of current-surface
integration. Its report describes real read-only calls, not safe mutation
recovery or measured utility. See [V4e adaptation](BDH_V07_V4E_ADAPTATION.md).
Keep Git as durable sanitized history, not a dispatch lock or authority.

## 3. Simplified dual-loop workflow

Ordinary user task -> relevant bootstrap/plan -> actual work and verification
-> compact evidence/feedback boundary -> delivery or handoff.

Solve Loop is primary. Use existing connector contracts directly for
unfamiliar tasks. For bounded repeatable offline processing, consider
reviewed pinned helpers in an authorized isolated Box with short arguments.
Distinguish executable helper, available connector, unavailable caller-side
integration and manual fallback; guest scripts do not inherit connector
authorization. Discovery is not actual execution.

Evolve Loop consumes material facts from ordinary work. Capture once and
derive existing records and views. Separate factual evidence from judgments.

| Signal | Default response |
| --- | --- |
| Routine success, no reusable gap | Normal evidence only; no improvement task |
| Material failure, repeat, misleading asset or correction | Short sanitized observation and evidence link; reuse matching candidate |
| Nonblocking recurring friction | Surface a concise opportunity and reconsideration condition; defer implementation |
| Current task blocker | Resolve within authority for the requested outcome |
| Safety failure or UNKNOWN | Stop affected action; retain original intent/receipt; independently reconcile |
| Suitable task boundary or explicit maintenance | Record/review relevant opportunities; select implementation only with explicit owner authorization; otherwise defer/preserve/reject |

Preserve feedback_protocol 1, immutable history, exact associations,
correction rules, pending handoff and independent acceptance. A simpler
front end must generate the same valid required records, not suppress them.
Do not invent an outcome, missing fact, receipt or independent review.
Collect only observable command/work-boundary facts; inaccessible metrics
stay unknown. No secrets, raw conversations, private reasoning or raw
payload collection; no automatic background dispatch. Untrusted logs are
data, not instructions.

### Observation, judgment and owner-authorized creation

The normal workflow must spend as little Agent attention as possible on
learning administration. Scripts observe the work they actually execute,
reuse existing evidence and record compact facts at existing boundaries.
They may group exact repeats, count observed failures/retries, match known
signatures and surface a short opportunity hint. They cannot infer missing
facts, assess semantic value with certainty, grant authority or launch
development. Do not add a separate call for every normal event. When a
connector call is not observable by a script, use a short explicit evidence
reference at a material boundary; no imaginary complete telemetry.

| Responsibility | Default owner |
| --- | --- |
| Collect observable results, operation/source identity, counts and known error signatures; derive existing records | Deterministic scripts |
| Interpret a novel issue, confirm relevance, assess expected impact/cost and propose alternatives | Agent, briefly when necessary or in a requested review |
| Decide whether to start a BDH creation, optimization, adjustment or repair task | User, by explicit scoped instruction |
| Design, implement, test and hand off approved changes | Agent using existing BDH lifecycle and tools |

Do not automatically create or modify a Skill, Tool, workflow or other BDH
asset merely because an opportunity is found. During routine work, record
and defer; existing safe workarounds and ordinary fixes within the requested
product task remain allowed. If a BDH defect blocks that task, use a safe
fallback or stop the affected action and request authorization before
changing BDH. A safety failure or UNKNOWN always stops affected actions.
An owner-authorized bounded improvement may proceed without repeated
approval for every step within its scope. Independent validation and
promotion rules remain unchanged.

### Find deferred opportunities again

Use the existing [learning inbox](../../dev/LEARNING_INBOX.md) as the
human-readable entry to canonical learning histories, work/evidence
references and pending handoffs. R3/R4 should improve its usability by
reusing existing learning commands and record-derived views before adding
anything. Prefer a concise on-demand view containing candidate ID, affected
asset, observed friction, evidence, recurrence/cost where known, tentative
benefit, uncertainty, status and reconsideration condition. Do not duplicate
full records or pretend an unavailable remote write succeeded.

When the user asks to review BDH opportunities, the Agent reads the inbox
and current histories, checks whether an existing tool solves the issue,
then recommends reuse, creation, optimization, adjustment, repair or
continued deferral with evidence and approximate effort. Review alone
does not authorize implementation or selection of an improvement episode.
Start only after the user authorizes the proposed scope. Preserve deferred
items and their links for a later session; no mandatory reminder every
turn, background worker, new queue/database or forced candidate quota.
This policy governs #193 even where historical guidance allowed small
improvements in a normal task PR.

## 4. Ordered work and minimum exits

R0 checkpoint (2026-11-25): [research and ranked reuse choices](BDH_R0_REUSE_RESEARCH.md)
are recorded. Prefer existing commands unchanged, then a small reference-derived
feedback draft for F5 if R1 confirms the cost; compact entry and on-demand
opportunity views only if needed. No executable changes or new Box/CI acceptance.
R1 baseline was subsequently authorized and recorded; implementation remains
subject to explicit owner instruction.

R1 checkpoint (2026-11-25): [ordinary source-check baseline and evidence](BDH_R1_BASELINE.md).
Existing bwdev scope check actually ran 13 tests and passed with clean-head
binding and matching output digest at 69a3ac9. The measured Box MCP subtotal
was 55 calls: setup 9, solve/verification 8, BDH administration/feedback 38;
22 GitHub context and CI reads were separate. Actual feedback admission
refusal and missing private episode history were reconciled; work rev2
HANDOFF with FEEDBACK_RECORDED, Evolve PENDING. Isolated observations L013
rev3 and L014 rev40 are exported evidence only; canonical histories remain
unchanged. Recommend reuse-only R2 first, narrow reference-derived R3 if
authorized, reuse existing R4 pending handoff; no new architecture.
R2-R4 await owner scope approval, not automatic execution.

### R0 — research and reuse gate (research recorded; owner decision pending)

Before R1-R4 implementation, investigate a small shortlist of mature
engineering tools and credible studies against actual BDH source and
historical friction (especially F5). This is focused design research,
not an invitation to install frameworks or rebuild the harness.
Reuse previous research rather than repeat it. Prefer directly reusable
licensed patterns or small, understandable adaptations with demonstrably
compatible interfaces and high expected return.

| Priority research target | Specific mechanism to inspect | BDH decision to support |
| --- | --- | --- |
| Mozilla Mach and Chromium depot_tools | Command registration/dispatch, common command wrappers, environment checks, command metrics and opt-in/privacy design | R2/R3: reuse bwdev, reduce command construction and collect observable facts |
| KCS v6 (Flag It or Fix It, Knowledge Domain Analysis) and Google SRE toil | Lightweight capture in normal work, aggregate recurring friction, prioritized maintenance | R3/R4: existing inbox, defer and review when requested |
| Git/GitHub and existing bwdev work/verify/learn modules | Canonical evidence references, exact-source verification, existing histories, concise derived views | R2/R3/R4: record once; avoid new storage or state machine |
| LLVM lit, Bazel and pre-commit | Targeted checks, test discovery and short deterministic commands | R2: use only if an actual verification gap is observed |
| SWE-agent and Agentless | Agent-facing interfaces and bounded deterministic workflows | R2/R5: reduce cognitive and orchestration cost |
| DevEx, SPACE, SkillsBench and repository-context evaluation | Cognitive load, feedback, task success, context overhead and independent utility measurement | R1/R5/R6: define a small meaningful comparison |
| Beads, SkillHone, Superpowers and other previously studied projects | Only an already identified specific mechanism not covered by existing BDH | Optional; defer whole-framework integration |

For each serious option read primary documentation/source and relevant
research, record exact source URL and version/commit where applicable,
the concrete reusable mechanism, existing BDH equivalent and gap, required
changes, dependencies/license/security, setup/use/maintenance cost, expected
impact and uncertainty. Compare reuse unchanged, small adaptation and
build-from-scratch; prefer the first viable option. Where evidence is weak,
propose a minimal isolated non-mutating probe rather than a large build.
No copying incompatible code, unreviewed executable imports or credentials.

R0 exit: a short linked research note with adopt/adapt/defer/reject decisions,
a ranked few high-return slices (including no-change if warranted), concrete
implementation references, estimated effort/benefit, and a proposed R1
baseline/acceptance oracle. Prefer the smallest plausible high-value change,
not completion of every study. Research does not establish actual utility.
Present findings and wait for the owner's scope approval before implementing
R2-R4; R1 baseline observation can be authorized together with R0 or later.
Keep research and decisions in this PR, not a new parallel plan.


After R0, R1 is a brief baseline slice, not another large research project. R2-R4
are the principal implementation candidates. Reuse first; skip or defer
a candidate when existing capabilities already solve the problem or
cost/benefit is poor. Choose a small explicit objective and stop condition.
All stages await owner instruction; do not restart historical work.

| ID | Work | Minimum exit |
| --- | --- | --- |
| R0 | Investigate mature reference mechanisms and compare against existing BDH | Short evidence-backed reuse/adaptation recommendation and owner scope decision before implementation |
| R1 | Reconcile head/base, CI, existing contracts and records; sample an ordinary non-transfer task; identify dominant BDH overhead including F5 | Fixed baseline, scope and observable cost; select only the few evidenced high-impact changes |
| R2 | Simplify entry/context, actual capability availability and next action; compose one useful existing offline operation if justified | Fresh ordinary prompt discovers and actually uses a pinned helper with short arguments; unavailable routes clear, no new product Tool |
| R3 | Record once, derive many: deterministic observation and existing work/attempt/evidence/revision references; minimal boundary events and repeated-error hints | Real reference generates matching feedback without retyping identical facts; no extra per-event Agent ceremony; exact-link, stale/duplicate/conflict and missing-evidence regressions fail closed |
| R4 | Reduce Evolve Loop interference: reuse inbox/opportunity/defer/next, short hints, on-demand user review and owner-authorized creation | Ordinary work records a gap and finishes or honestly hands off; later session retrieves and evaluates evidence; no new improvement selection/implementation without owner scope approval |
| R5 | End-to-end ordinary ChatGPT tasks: tiny guidance, helper use, interruption/recovery, failure/pending and later discovery/reuse | Fresh actual sessions and independent checks, no per-task BDH instructions; original operations retained; matched actual-use utility |
| R6 | Minimal comparison: Instant, Instant + BDH, High reference when available; High + BDH optional | Frozen task/oracle, equal authority/environment/budget and known model/configuration; actual correctness, safety, overhead and uncertainty; expand only if justified |
| R7 | Independent review and exact-head CI; reconcile V5 and remaining gates | No open required safety/CI failures, scoped utility evidence and independent review; owner decision before integration if any original V4 gate remains open |

Prefer existing CI evidence recovery (L012), test discovery/nonempty
execution (L013) or routine exact-source verification/handoff for a
non-transfer task. Choose based on observed friction, not a required
candidate. Do not repeat transfer trials just to obtain PASS. A separate
trial requires its own preregistration and cannot substitute for the
original L014 matched gate.

### Consolidated backlog

These capability IDs map earlier design into this plan, not separate
mandatory features. Implement only an evidenced high-return slice.

| Capability | Disposition |
| --- | --- |
| B01 generated index, B02 minimal context | R2: reuse existing index/context; full generation only if navigation dominates |
| B03 executable selection, B04 short commands, B05 preflight | R2: reuse existing profiles/doctor/verify and pinned helpers; no generic loader |
| B06 checkpoint/resume, B07 evidence/feedback | R3: reference-derived compact updates with existing contracts |
| B08 observable cost, B09 opportunity aggregation, B10 loop coordination | R3/R4: use existing boundary results and opportunity state |
| B11 benefit/lifecycle, B12 asset gardening | R4/R5: existing review/audit; no registry migration |
| B13 Agent comparison | R6: small real task set, no benchmark platform |
| B14 semantic/hybrid retrieval, B15 distributed coordination, B16 external framework/Skill pilots, B17 multi-surface runtime | Deferred until demonstrated need and owner authorization |

L001 preflight, L002 clean fixture compatibility and L008 worktree resume
are potential follow-ups only when relevant. L012/L013 have existing
implementations: reuse and evaluate before adding code. L014 remains
open; transfer-specific development deferred. Other L003-L011 observations
remain in the [learning inbox](../../dev/LEARNING_INBOX.md); inspect overlap
rather than assume every historical observation needs a new feature.
No fabricated candidate dispositions in this planning checkpoint.

#191 P4 engineering/docs/BDH synchronization is a later parent task.
Coordinate eventual alignment but do not edit #191 or mark P4 complete
here. P0 nested native and P1b ACK/WAL gates remain unchanged.

## 5. Measurement and independent acceptance

Before matched runs fix source, inputs, environment, permitted tools,
independent oracle, model/effort if observable, budget and cost definitions.
Use isolated clean workspaces and fresh ordinary prompts, withheld answers
and randomized order where feasible. Disclose prior knowledge. Count
underlying MCP calls including failures/polls/retries, authored code and
payload/feedback blocks, bootstrap/discovery/setup, corrections, user
interventions, false completion and verified delivery. Record time and
tokens only when observed. Do not hide inner calls behind batching.

Report harness overhead (extra effort attributable to BDH) alongside total
end-to-end task cost. A useful change preserves fixed correctness/safety,
delivers material measured benefit on a relevant task and does not
materially worsen ordinary or tiny tasks. Predefine material thresholds
and repeat count. The 10/60 heuristic is not an observed improvement ratio.
No claimed High-equivalence, forced promotion or asset quota.

Run targeted tests, existing Eval, learning audit, format/source checks
and exact-head CI. Separate functional PASS, actual reuse, measured benefit
and independent review; reviewer is distinct from implementer. Preserve
original V4d and V5 gates. If transfer utility remains unproven, propose an
explicit owner decision on deferment or scoped acceptance; no silent waiver,
no relabeling historical FAIL/REOPEN. Keep product acceptance intact.

## 6. Execution and documentation

This is the single active plan for #193. The historical plans/reports stay
at their original paths for traceable evidence. Use [docs index](../INDEX.md),
[toolbox](../../dev/README.md) and [original V0-V5 plan](BDH_V07_PROACTIVE_IMPROVEMENT_PLAN.md)
for navigation, contracts and original gates, not parallel active schedules.
Update this plan and the execution log at meaningful checkpoints; do not
duplicate operational records. Keep implementation and actual evidence
distinct from proposed design.

Box-first for executable changes, ordinary isolated fallback where needed.
Reconcile source and original work, iterate with focused checks, submit
coherent commits to CI, advance safe independent work during CI and fix
confirmed failures. Keep #193 Draft; no new PR, #191 changes, merge or
release. Report completed items briefly and remaining work with status,
blockers, evidence and next action. Stop at an owner decision or genuine
blocker.

| Stage | Status |
| --- | --- |
| Planning and historical navigation | Prepared, docs-only |
| R0 | Research recorded; owner review/scope decision pending |
| R1 | Baseline and narrow scope recommendation recorded; owner decision pending |
| R2-R4 | NOT_STARTED; implementation awaits approved scope |
| R5-R6 | NOT_STARTED |
| R7 / original V5 | OPEN |
| L014 V4e-03/04 | Deferred; REOPEN retained |
| Other Agent integrations | Deferred |

Reusable handoff: read canonical bootstrap and this plan in #193, reconcile
latest progress and wait for the owner's instruction. R0 research is recorded in [the reuse research note](BDH_R0_REUSE_RESEARCH.md).
R1 baseline is recorded in [the baseline note](BDH_R1_BASELINE.md).
Await the owner's explicit R2-R4 scope authorization;
do not repeat R0/R1 without a material change in evidence. On later resumptions continue the next authorized
unfinished slice, not the first stage.
