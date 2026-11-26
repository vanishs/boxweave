# R0 — reuse research and scope recommendation

2026-11-25. PR #193 only. R0 research authorized; R1 baseline and
R2-R4 implementation not authorized by this instruction. Canonical v1
9d07665dcfc5ae6345d6bfcd0d71de7b6510b51d; inspected source
431e8662ec3e726949134779c084bf4c919fe7df; parent #191 remains
3494ae8a35c50c386686c9d9d675c1ae37e1b1b7. This is a design recommendation,
not independent utility acceptance. No executable changes, new episode,
Box mutation, dependency installation, or historical revision changes.

## What the current evidence actually says

- Read the canonical bootstrap/rules, active plan, toolbox, learning
  inbox, earlier [F1-F5 research](BDH_V07_IMPROVEMENT_RESEARCH.md),
  [V4d trial](BDH_V07_V4D_TRIAL.md) and [V4e adaptation](BDH_V07_V4E_ADAPTATION.md).
  The plan reports 73/250/81 MCP calls in lanes A/B/C; C found but did
  not execute the delivered planner/template. Functional correctness
  is not proof of reuse or utility. Preserve both REOPENs and L014 rev39.
- F5 is directly visible in [P1b work revision 3](CONSOLIDATION_P1B_SOURCE_WORK_HISTORY/00000003.json)
  and [L014 revision 11](../../dev/learning/BDH-L014/00000011.json):
  the same work/attempt/evidence/observation/outcome is manually repeated.
  The earlier failed association and append-only correction are recorded
  in F5. The corrected pair has matching literal facts, not two independent
  observations. This is evidence of redundant construction and repair,
  not a measurement of all BDH overhead.
- [feedback.py](../../dev/harness/feedback.py) matching_observation
  already verifies those five keys; preserve() prevents historical fact
  loss/rewrite and requires explicit correction evidence. It is a validator,
  not an authoring helper. [work.py](../../dev/harness/work.py) validates
  and publishes a supplied envelope, with summary/plan/resume already
  present. There is no need for a second feedback schema.
- [cli.py](../../dev/harness/cli.py) already has an argparse dispatch
  table and common emit; work.plan combines context and learning audit/
  selection (inspect actual output before adding a new entry). Existing
  [verification.py](../../dev/harness/verification.py) execute/inspect
  uses allowlisted checks, exclusive evidence directories, source identity,
  exit code and output hash; [ci.py](../../dev/harness/ci.py) has
  identity/evaluate/inspect; [check_runner.py](../../dev/harness/check_runner.py)
  refuses an empty test suite. Reuse these first.
- [learning.py](../../dev/harness/learning.py) all_records/audit and
  learn next/opportunity/decide/defer, [evolution.py](../../dev/harness/evolution.py)
  opportunity/select, and [records.py](../../dev/harness/records.py)
  already implement review signals and exclusive local revision publication.
  No need for a new issue tracker, dispatcher, or state machine.
- V4e showed an authorized read-only connector-to-Git evidence bridge,
  not a persistent caller-side executor. A guest cannot inherit connector
  authority. This boundary is not fixed by a CLI or a prompt. Local Git
  and SQLite results are not distributed dispatch authority. Exact-head
  PR workflow query for starting HEAD returned no runs; preceding CI
  results are historical. No new native or CI PASS claimed.

## Focused external references and decisions

References inspected 2026-11-25; moving documentation is identified by
retrieval date, not an invented source commit. Pinned paper versions and
the depot_tools metrics revision are listed where observed. These are
mechanism-level adaptations in existing Python, not source imports.
No external code, packages, service, license grant or telemetry endpoint
is adopted; source copying would require a separate pinned license,
dependency and security review. Expected benefit is an unmeasured hypothesis.

| Reference / mechanism | Existing BDH equivalent and disposition | Cost / risk / expected return |
| --- | --- | --- |
| Mozilla [Mach](https://firefox-source-docs.mozilla.org/mach/index.html), [command registration](https://firefox-source-docs.mozilla.org/mach/commands.html), [metrics handle](https://firefox-source-docs.mozilla.org/mach/telemetry.html): one dispatcher, command-specific context, instrumentation at command boundary | ADOPT pattern only; existing argparse COMMANDS and registered checks already provide the main mechanism. Prefer a compact result/next-action view and existing evidence fields. No decorators/framework migration. | Small if output projection only; medium if altering dispatch. No new dependency. Avoid recording argv values or hidden data. Expected fewer navigation decisions, unmeasured. |
| Chromium [depot_tools](https://www.chromium.org/developers/how-tos/depottools/), [pinned metrics policy](https://chromium.googlesource.com/chromium/tools/depot_tools/+/071d5b9d91e06cb2a9c9ce926d6ee666df185b49/metrics.README.md): command wrappers and bounded measurements such as command name, duration and exit | ADAPT allowlisted observable facts, not their auto-update or remote metrics collection. Reuse verify.execute and existing work/attempt identity. No extra event call. | Small for already recorded fields; unknown for inaccessible connector events. Never collect arbitrary strings, credentials, payloads or private reasoning. No default external telemetry. |
| KCS v6 [flag or fix](https://library.serviceinnovation.org/KCS/KCS_v6/KCS_v6_Practices_Guide/030/030/040/030), [knowledge domain analysis](https://library.serviceinnovation.org/KCS/KCS_v6/KCS_v6_Practices_Guide/030/040/010/060): usage-linked review and trends, authority determines who may edit | ADAPT flag-by-default for nonblocking BDH gaps, existing inbox and learn next/opportunity/defer. Agent assesses and owner explicitly authorizes implementation. Reuse/no-change valid. | Small read-only projection or documentation if existing output suffices. No new workflow service. Expected less interruption; unmeasured. |
| Google SRE [Eliminating Toil workbook](https://sre.google/workbook/eliminating-toil/): measure repeated work, avoid measurement burden, automate high-priority slices | ADOPT prioritization and before/after measurement; F5 is strongest current candidate. Keep partial automation and manual fallback. | Small initial baseline; opportunity cost of writing wrappers must be counted. Do not import Google staffing or savings targets. |
| Git/GitHub and existing bwdev work/verify/learn: exact immutable source, links and derived views | REUSE unchanged identity, record and validation machinery. ADAPT one reference-derived envelope draft only if R1 confirms repeated transcription. Git remains sanitized handoff, not private journal or lock. | Medium for safe composition/regressions; no storage migration. Best evidenced likely return: avoid retyping and association repair. |
| [LLVM lit](https://llvm.org/docs/CommandGuide/lit.html), [Bazel test contract 9.2](https://bazel.build/versions/9.2.0/reference/test-encyclopedia), [pre-commit](https://pre-commit.com/): discovery, bounded checks, exit/fixture discipline and selected hooks | REUSE check_runner, verify plan/run/inspect, existing hooks and CI classifier; no new gap demonstrated. DEFER importing these runners. | Existing setup cost zero; import introduces toolchain/config/download/maintenance complexity. No extra verification system. |
| [SWE-agent, NeurIPS 2024](https://proceedings.nips.cc/paper_files/paper/2024/hash/5a7c947568c1b1328ccc5230172e1e7c-Abstract-Conference.html), [Agentless arXiv:2407.01489](https://arxiv.org/abs/2407.01489): task-facing interface, hierarchical localization and bounded repair/validation | ADAPT concise action/evidence interfaces and relevant context; existing work.plan and registered offline checks. DEFER new agent framework. | Small if projection; large if framework. Published benchmark results are not BDH utility evidence. |
| [DevEx DOI:10.1145/3610285](https://doi.org/10.1145/3610285), [SPACE overview](https://github.blog/enterprise-software/devops/measuring-enterprise-developer-productivity/), [SkillsBench v4](https://arxiv.org/abs/2602.12670v4), [AGENTS.md evaluation v3](https://arxiv.org/abs/2602.11988v3) | ADOPT multidimensional evaluation, fixed task and oracle, smallest relevant context, independent comparison. Repository-context paper reports cost overhead and no general success improvement; do not remove required safety rules. | Small matched task set; no benchmark infrastructure, self-reported savings, skill quotas or general model-equivalence claims. |
| Beads, SkillHone, Superpowers and previous research | DEFER: no uncovered specific mechanism justifies integration after source review. Existing V0-V5 research remains linked; no repeated broad market survey. | Avoid additional dependency, agent orchestration, and duplicated lifecycle. |

## Ranked minimum slices (scope recommendation, not authorization)

1. **R3/F5: record once, derive feedback.** First reuse existing command
   outputs; then, only if needed, add an explicit preview/draft operation
   within work/feedback that resolves an exact candidate/revision and
   observation from the existing history, checks attempt/work/source
   applicability and copies only verified literal fact fields into a
   protocol-1 envelope. Caller still supplies asset/use and semantic
   judgment, reason, authority, next action and any unavailable facts.
   Validate through existing matching/preserve/boundary logic; use existing
   separate writes and pending-handoff on partial persistence. No new
   transaction promise. Reject stale revision, ambiguous/missing observation,
   mismatched asset/work/attempt/source, duplicate/conflicting facts and
   unknown outcome claims. Never fabricate a learning observation or
   silently relink. Candidate code surface: feedback.py/work.py and focused
   test_bwdev_feedback regressions; medium effort/risk, no dependency.
2. **R2: use existing commands unchanged first.** Test whether a fresh
   ordinary prompt discovers the relevant existing helper in an isolated
   exact-source Box and actually uses it with concise arguments. Only if
   R1 shows material navigation overhead, add a compact read-only
   work.plan/summary view exposing relevant next action, evidence and
   observed availability; existing JSON detail remains accessible.
   Candidate surface: work.py/workflow.py and focused tests, small effort.
   Never say a local script is a callable remote MCP adapter. Do not
   replace canonical bootstrap or add a general loader.
3. **R4: review on demand.** Try existing learn next/opportunity/audit and
   inbox links first. If needed, add a concise record-derived read-only
   view over existing validated histories: candidate, affected asset,
   friction, evidence, observed recurrence/cost, uncertainty, status
   and reconsideration condition. Group only exact signatures; semantic
   duplicates and value require Agent judgment. Do not automatically
   select or implement. Candidate surface: learning.py/evolution.py and
   focused tests, small effort. No new queue.
4. **No change** is the fallback for any slice with no observed gap or
   no accessible execution route. Prefer documented manual Git/MCP
   fallback for current remote-only work. Defer L014 transport automation,
   further framework integrations, distributed journal, background jobs,
   generalized metrics collection, new product tools and new schemas.

Rank is expected return, not permission to reorder plan stages. R1 comes
before implementation; R2 can be reuse-only and R3/R4 can be skipped if
existing output suffices. Small = localized existing output/docs and
tests; medium = narrow composition and safety regressions. These are
relative engineering estimates, not timed or measured savings.

## Proposed R1 baseline and acceptance oracle (owner approval required)

Start with one bounded ordinary non-transfer task: recover exact-source CI
and report required job results (L012), or run one existing nonempty
registered verification check (L013) in an authorized clean isolated Box
if CI data is unavailable. Choose the single case using available exact
source and evidence. Include a small material work/handoff boundary and
one real linked observation only if it actually occurs. Do not invent a
gap merely to exercise learning; use the immutable P1b pair as a separate
offline F5 fixture. No new native mutation needed for the initial baseline.

Freeze task prompt, source, existing histories, environment, available
tools/authority, model/effort if known, budget and independent correctness
oracle. Use existing implementation before changing code. Count actual
underlying tool calls including setup, failures, polls, retries, discovery,
manual authored/retyped blocks, correction steps and feedback overhead
separately from task execution. Time/tokens only if observable. For CI:
exact candidate/base/run/job association, required conclusions and
missing/failed/skipped gates must be reported accurately. For verification:
nonzero discovered tests, real exit/output identity, clean-vs-unbound
source and exact evidence. For material feedback: protocol-1 matching,
preserved immutable facts and original receipts, pending truthfully
reported and no unauthorized selection. A tiny guidance task is a
no-overhead control. Independent reviewer must distinguish local evidence
from native, release and utility acceptance.

Proposed pragmatic decision gate (not pre-registered yet): after R1 fix
the actual comparison denominator and two independent fresh matched
ordinary-task pairs; require identical correctness/safety, at least 30%
fewer manually constructed BDH evidence/feedback blocks on the relevant
case with at least one block removed, no increase in total underlying
calls or manual actions on the tiny control, and report full end-to-end
cost/setup and tradeoffs. If baseline has no repeatable overhead, skip
implementation. Counts do not prove cognitive load; if actual use or
independence cannot be observed, keep utility open. R5/R6 should use fresh
sessions with neutral prompts and disclose prior project knowledge.
Threshold and repeat count are proposed choices for the owner/R1, not
claims of current gains or a substitute for the original V4 gate.

## R0 checkpoint and limitations

Research note is ready for owner review. No new execution probe: static
source and preserved records resolve the reuse/no-rebuild question;
runtime feasibility was previously tested in V4e, and actual ordinary
task measurement is R1. No Box-first executable change or fresh Box/CI
acceptance in R0. External references are primarily official documentation
and paper abstracts/available text, not reproduced full experimental
results or installed framework tests. Further license/source pinning
would be needed only if code reuse were proposed; none is proposed.
Learning judgment: reuse/defer existing candidates, no-new-asset for
a duplicate candidate or implementation; this note records scope choices.
Owner must authorize R1 and subsequent implementation scope. Preserve
V4e-03/04 deferred, V5 open, both REOPENs and #191 unchanged.
