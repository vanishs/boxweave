# BDH development plan — simplify, reuse, verify

Status: owner authorized R7 on 2026-11-25. Implementation slices delivered;
R5 partial, R6 pilot complete with utility NOT MET. Final independent
review, current documentation-head CI and owner disposition remain open.
The checkpoint paragraphs below preserve dated evidence; the consolidated
status and R7 decision section govern current work.
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
R2 was subsequently authorized; R3-R4 remain subject to owner approval.

R2 checkpoint (2026-11-25): [technical delivery and original evidence](https://github.com/vanishs/boxweave/pull/193#issuecomment-7179312167).
Code b66d64eba21872e3139434cf32cc1f38e34b82de adds an opt-in brief projection
of existing work plan, retaining required context, gates, source, candidate
order/review statuses and audit findings; no new executor or selection.
Box 116 tests passed (4 unrelated Node skips), Eval14 PASS, audit clear;
fresh exact checkout 13/13 scope check with matching source/output evidence.
Same R1 query: 6720 -> 4513 output bytes (32.8% less), not measured Agent
productivity. Short plans may be longer; use full mode or no plan for tiny
guidance. Full mode unchanged. Exact code push quality job 139617173782 in
run 45732347009 SUCCESS; candidate 45732346982 SUCCESS (native and release
skipped). The previous empty PR-only CI list was not an absence of push CI:
stacked PR base is not v1. Public read-only exact-head push run lookup plus
existing connector job verification recovered the actual runs without
credentials; see original operation dd5a5f1fdb46b2bcdb9e0173d5090335 for
prior 69a3ac9 and 2fc49ee successful workflow pairs. Do not use push as
proof of PR/base association or release acceptance. No CI workflow change.
R2 technical delivery complete; independent fresh ordinary Agent discovery
and use is still pending because the implementer cannot initiate a separate
session in this surface. Do not label this smoke independent or waive the
minimum exit. R3 was subsequently authorized; R4-R7 not started.

Owner retirement (2026-11-25): [JS/Node retirement evidence](https://github.com/vanishs/boxweave/pull/193#issuecomment-7179571864), exact 7238f630292532610a21f88077bd633bdf4db5b6.
Remove experimental client JS template and four Node-dependent test methods;
retain Python planner, direct authorized MCP, original admission/readback
safety and historical evidence. Box 116/116 no skips, Eval14 PASS.
L014 implementation digest drift is real and remains REVIEW_REQUIRED;
fixture-only tests updated, canonical histories and both REOPENs unchanged.
No new transfer episode, no promotion. Python is the BDH script system.

R3 checkpoint (2026-11-25): [implementation and actual retained-record replay](https://github.com/vanishs/boxweave/pull/193#issuecomment-7179629656).
Opt-in repeatable work boundary --link-observation ID REVISION REASON
uses validated prior observation and current explicit work/attempt, target,
used asset and evidence refs. Agent supplies link and reason. Existing
advance/preserve/review/publication gates stay in force; reject stale or
ambiguous/conflicting/missing/unknown refs and changed target/source.
No extra per-boundary call, schema, learning revision or new executor.
Final Box 122/122 no skips, Eval14 PASS, format/diff PASS, audit expected
L014 drift only. Replay in isolated copies of actual R1 observations L013
rev3 and L014 rev40: same normalized five feedback items, HANDOFF,
FEEDBACK_RECORDED and Evolve PENDING. Same-serializer boundary 5055 to
3828 bytes plus 179 reference argv bytes, net 1048 fewer; 2 to 0 linked
fact retypes. Not fresh Agent performance; no demonstrated reduction in
number of manual submissions, overall tool calls, time or tokens.
Separate histories are not a distributed transaction. Local technical
implementation/replay complete; exact-head CI and actual fresh-use/benefit
validation pending. R3 exact code quality 45734840685 SUCCESS; candidate
45734840817 applicable source/build/manifest jobs SUCCESS, native and release
skipped. R4 subsequently authorized; R5-R7 not authorized.

R4 checkpoint (2026-11-25): [local technical evidence and retained-record review](https://github.com/vanishs/boxweave/pull/193#issuecomment-7179730081).
Reuse existing inbox, audit, next, opportunity, defer and decide. Opt-in
learn next INBOX --all --brief projects ranked candidates for owner-requested
review; ordinary task filters remain unchanged. Preserve global audit exit
and findings even with no matches, exact candidate/revision/basis, original
evidence, latest matching observation and recorded decision/reconsideration.
Full records remain available. Align active Skills with explicit owner
scope authorization; normal work flags/defers nonblocking gaps and continues.
No new scheduler, schema, queue, product Tool or automatic selection.
Box 125/125 no skips, Eval14 PASS, format/diff PASS; expected L014 drift only.
Actual canonical global output 47835 -> 5346 bytes, relevant 42973 -> 3197,
unrelated 1839 -> 726 with no matched candidate and preserved drift/exit 1.
R1 private original HANDOFF / FEEDBACK_RECORDED / Evolve PENDING retained;
private L014 rev40 -> rev41 explicit defer using existing decide, no proposal
or episode. Later separate process retrieved exact reason/condition, full
53821 -> brief 4625 bytes. All original records unchanged; only private
rev41 appended. This is knowledgeable same-Agent cross-process recovery,
not an independent fresh task/session or measured overall Agent benefit.
Canonical histories and both REOPENs unchanged. R4 local technical PASS;
exact-head quality 45735552060 and candidate 45735551966 SUCCESS on 55be132
(native and release skipped); fresh ordinary use remains pending. R5 now
owner-authorized.

### Reference mechanisms and actual composition (R0 reconciliation)

These are design adaptations of mature mechanisms, not six imported
frameworks. See [R0 research](BDH_R0_REUSE_RESEARCH.md) for the original
references and decisions. No new external package or service is required.

| Reference | Current BDH realization | Evidence / remaining limit |
| --- | --- | --- |
| Mozilla Mach | Existing Python bwdev dispatcher, context/profile selection, R2 opt-in work plan --brief | Unified entry and concise output implemented; fresh ordinary discovery/use pending |
| SWE-agent / Agentless | Relevant assets and short registered verification commands; R2 context projection | Existing Agent-facing interfaces, no Agent framework or reusable remote executor; R5 validates actual use |
| LLVM lit / Bazel (also pre-commit) | Existing check_runner, verify plan/run/inspect, exact-source CI and format gates | Reused nonempty, deterministic verification; not a new runner |
| Chromium depot_tools | Existing verification exit/output/source evidence, work/attempt and R3 exact observation links | Partial observation only; no automatic complete MCP call/failure/retry/time/token instrumentation or telemetry service |
| KCS v6 | Existing learning inbox, audit/next/opportunity/defer/decide; R4 brief review and flag/defer policy | Local/cross-process retrieval verified; owner authorizes improvement selection/implementation; fresh-session evidence pending |
| Google SRE toil | R1 baseline prioritization, R3 avoid repeated transcription, R4 concise on-demand review | Measured local payload/record improvements, not measured end-to-end toil reduction; automatic aggregation and all-call cost coverage not demonstrated |

Conceptual relationships (existing components, not separate services):

    Owner task -> Agent judgment -> bwdev/context (R2)
                                  -> Solve: authorized MCP/Python -> verify -> deliver
                                  -> observed facts / exact references (R3)
                                        -> immutable work/learning evidence
                                        -> Evolve: inbox / defer (R4)
                                             -> owner-requested Agent review
                                             -> owner scope authorization
                                             -> bounded improvement / independent validation
                                             -> Git/PR assets -> later discovery and reuse

Solve can continue or honestly hand off independently of nonblocking
Evolve work. Python records what it actually observes; the Agent supplies
semantic judgments and explicit references for otherwise inaccessible
connector events. Preserve UNKNOWN, pending and drift. Git/PR provides
sanitized evidence and knowledge persistence, not global dispatch or a
transaction. R2 reduced a material plan output 6720 to 4513 bytes (32.8%);
R3 removed two linked fact retypes and saved net 1048/5055 serialized
boundary bytes (20.7%); R4 global review 47835 to 5346 bytes (88.8%). These
are different local measures, not additive, token savings or a 10/60
productivity claim. Do not broaden automatic metrics collection ahead of
R5/R6; first measure actual accessible overhead and disclose missing data.

R5 checkpoint (2026-11-25): [witness protocol and preflight](BDH_R5_WITNESS.md).
Owner requested recording the reference composition and starting R5.
Freeze exact R4 code 55be132 and owned clean /root/pr193-r5-witness; Python
3.12.3, Git 2.43.0, no gh/Go/gofmt/Node. Preflight only, no ordinary test
executed. Actual fresh independent ChatGPT sessions are not launchable
through the implementer's available authorized tools. Neutral prompts and
oracle prepared; fresh witness execution requires a separate user-started
session. Do not repeat prior R2-R4 smoke/replays or claim them independent.
This was the original R5 preparation boundary, superseded by the actual
witness and R6 records below; do not repeat preparation or prior tests.

### R0 — historical research and reuse gate (completed; scope authorized)

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

### R5a — discovery and reuse navigation repair (owner authorized)

[Isolated-project retests](https://github.com/vanishs/boxweave/pull/193#issuecomment-7180405247)
showed actual source test success via direct unittest, not the registered
helper; prior coached helper use remains valid, not independent discovery.
Both opportunity reviews found L014 and deferred reasonably using remote
records; local brief review and fresh material link/recovery remain open.
The retest restriction on past reports and mixed historical quality guidance
are confounds, not proof that the helper was inaccessible. W2 retest 13/13,
9 Box + 11 GitHub reads + one comment; W4 retest 27 GitHub reads + four
Operation reads + one comment, plus seven schema discovery invocations.
No comparable end-to-end savings established.

Bounded R5a: prefer the matching existing registered check via short
routes at task-source AGENTS/bootstrap/index/toolbox; no mandatory full
plan for focused checks. Separate historical quality-gate witnesses from
current procedures while retaining all gates and immutable original links.
Do not change the verification Skill or attached historical asset bytes;
reuse its guidance and improve navigation instead. No executable change,
new schema, executor, telemetry, product Tool or asset promotion.

Candidate changes do not activate on canonical v1 until integrated. Fresh
witness should use the usual canonical project locator, then the exact
owner-authorized task checkout's rules/index/toolbox; keep control and
candidate versions distinct. If candidate entry is not reached, report
that rather than assume activation. Use neutral ordinary prompts without
helper commands, record incidental historical exposure and chosen route.
Tiny guidance must stay lightweight; a legitimate manual fallback remains
allowed. A focused registered check may skip the full plan; actual
registered helper use is the minimum R2 discovery/use result, not a quota
for every task. Do not repeat tests solely to get a preferred route.
R5a technical/docs validation and fresh registered-helper use passed.
W1/W3 and W5 were subsequently completed; R3 fresh linkage and R4 actual
lifecycle use remain open. See the consolidated dispositions below.

### R5-R7 actual-use and final decision checkpoint

R5 [independent W5 synthesis](https://github.com/vanishs/boxweave/pull/193#issuecomment-7180823730)
and [CI reconciliation](https://github.com/vanishs/boxweave/pull/193#issuecomment-7180870813):
PARTIAL PASS. W1 canonical tiny guidance, W2 actual registered check use,
W3 controlled handoff and W4 remote historical review passed in their
respective scopes. W4 fresh local learning lifecycle remains partial.
R5a at f4a63f6 has docs-only validation and exact-head applicable push CI,
[ordinary registered-helper use](https://github.com/vanishs/boxweave/pull/193#issuecomment-7180607474),
and lightweight canonical guidance. Candidate-specific tiny behavior,
universal activation and causal navigation benefit are not established.
[W3 recovery](https://github.com/vanishs/boxweave/pull/193#issuecomment-7180772925)
verified original ADMITTED -> SUCCEEDED guest exit 0, 125/125 tests,
clean exact source and matching retained log without reported replay.
It is not a crash, UNKNOWN settlement or formal protocol-1 resume proof.

R6 [preregistration](https://github.com/vanishs/boxweave/pull/193#issuecomment-7180910955),
[matched material case](https://github.com/vanishs/boxweave/pull/193#issuecomment-7181137986)
and [tiny case / closeout](https://github.com/vanishs/boxweave/pull/193#issuecomment-7181175533):
pilot complete, functional PASS, incremental locator utility NOT MET.
Both ordinary Instant lanes selected the same existing registered
recovery-fixture check; separate original Operations and identical
retained log/record hashes, six mock tests passed per lane. A (repository
locator only) reported 20 connector calls, B (canonical bootstrap locator)
30, including B's initial incomplete turn and one additional user
intervention. Both manual persistent record counts zero; primary
30% reduction criterion unmeasurable and not achieved. A did not resolve
current canonical v1, so the assurance scopes differ. Tiny answers both
correct, no Box effects or ceremony, A0/B6 remote reads; A reused prior
context, so the matched tiny-cost gate is not established. Actual hidden
model effort, tokens, total wall time and prior exposure are unknown.
No pure no-BDH control, no causal/general productivity or High-equivalence
claim. Do not repeat the pilot or add an optional High lane to rescue
the outcome. Preserve negative evidence and stop this evaluation.

R7 authorized: technical/evidence reconciliation on frozen source f4a63f6,
with code last changed at 55be132. Applicable exact push workflows and
jobs succeeded on both, and native/release jobs were skipped. Parent #191
still paused at its exact base. Compare to #191: 35 commits, 129 paths,
BDH Python code/tests, records and docs; no product Go runtime or CI
workflow changes. The two original L014 REOPEN resolutions are retained
in canonical revision 39, CANDIDATE, both episode histories revision 17.
JS retirement's historical implementation digest drift remains a visible
review requirement, not a new promotion or a clean audit. This checkpoint
does not claim a new independent source-wide review by the implementer.

| Open gate | Proposed disposition and trigger |
| --- | --- |
| R2 / R6 measured utility | Retain negative/inconclusive pilot; measure only a genuine future material task with comparable evidence and overhead; do not claim a percentage |
| R3 fresh linked feedback | Retain technical/replay PASS, observe first applicable real material boundary with exact existing observation; no fabricated learning write |
| R4 fresh lifecycle | Retain technical/cross-process and remote review evidence; observe actual gap, honest boundary and later retrieval when naturally applicable |
| L014 / V4 / V5 | Retain CANDIDATE, both REOPENs and expected drift; no third selection; reconsider when recurring actual transfer toil or supported authorized caller-side reuse changes |
| Product acceptance and integration | No waiver of #191 native ACK/WAL, nesting, P4 or required CI; independent R7 review plus explicit owner decision required before any scoped integration |

Recommend freezing feature work and asking a distinct reviewer to inspect
the complete #193 diff, evidence and exact final documentation-head CI.
Unconditional integration readiness is NOT established while required
utility and lifecycle exits remain open. Only the owner can decide to
accept a narrowly scoped integration with those non-safety gaps explicitly
carried forward; do not mark them PASS. Any required safety or CI failure
must remain blocking. No merge, release, promotion or parent edit follows
from R7 authorization. If the reviewer finds no safety/code blocker,
present the scoped acceptance decision to the owner, not another
unrequested implementation cycle.

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
| R0 | Complete; research and owner scope recorded |
| R1 | Complete; actual baseline and subsequent owner scope recorded |
| R2 | Technical and scoped registered-helper actual-use PASS; measured benefit open |
| JS/Node retirement | Complete, Box and exact retirement CI PASS; historical drift retained |
| R3 | Technical implementation/replay and applicable exact code CI PASS; fresh linked boundary and benefit open |
| R4 | Technical recovery and remote review PASS; fresh full local lifecycle and benefit open |
| R5 | W5 synthesis complete; PARTIAL PASS; W1/W2/W3 and W4 remote review scoped PASS; W4 lifecycle and utility open |
| R5a | Bounded navigation delivery PASS; docs CI and fresh helper use PASS; utility unproven |
| R6 | Pilot COMPLETE; functional PASS, incremental utility NOT MET, general utility inconclusive |
| R7 / original V5 | R7 authorized; consolidation and final review/CI in progress; V5 utility open, owner scope decision pending |
| L014 V4e-03/04 | Deferred; REOPEN retained |
| Other Agent integrations | Deferred |

Reusable handoff: read canonical bootstrap and this plan in #193, reconcile
the exact head/base, R7 review and current CI evidence. Do not repeat
accepted tests or manufacture observations. Complete the distinct final
review, report the actual blockers and seek explicit owner disposition of
remaining non-safety gates before integration. R0-R6 historical work
remains traceable in the linked evidence. Later ordinary work should
observe naturally applicable R3/R4 gaps and defer nonblocking improvement
unless the owner requests it. No merge or release without separate scope.
