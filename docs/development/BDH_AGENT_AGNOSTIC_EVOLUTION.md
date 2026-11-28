# BDH evolution — agent-agnostic capability compensation

**Owner direction (2026-11-26), PR #191.** Preserve the existing BDH
architecture and evolve it incrementally through the explicit owner request
"BDH优化" (BDH optimization). This is a forward design and selection policy,
not a new mandatory workflow, feature quota, runtime integration project,
standalone roadmap or proof of utility. Do not build per-Agent adapters.

## Motivation and evidence boundary

BoxWeave dogfooding exposed distinct infrastructure and external Agent
limitations. BoxWeave supplies the execution environment; the external
Agent owns its model and runtime; BDH supplies repository-local engineering
capabilities and verifiable contracts. A full runtime harness may already
provide capabilities that an ordinary remote MCP session lacks. Client or
model identity must not be a routing input. Verify actual authorized tools,
not assumed shell, hooks, credentials or session recovery.

Baseline: canonical v1 9d07665dcfc5ae6345d6bfcd0d71de7b6510b51d;
task-source #191 d6143595ed4184b800723996905ec9cab9a46a62.
The latter contains short task routes, stage-wide continuation, Finish Gate,
batch/failure helpers and an owner-invoked improvement cycle. The
[BDH plan](BDH_DEVELOPMENT_PLAN.md) records V4d transfer trials (73/250/81
MCP calls), two REOPEN decisions and unmet utility conditions. Those counts
describe specific lanes, not a general Agent comparison. Preserve the
limited #193 integration's open R3/R4 fresh-use, R6 utility, L014 REOPEN
and V5 obligations. This investigation observed multiple repository reads
during remote startup, but did not measure total overhead or net benefit.

## Research and reusable precedents

Primary sources reviewed on 2026-11-26. Paper results apply to their own
benchmarks, not BoxWeave. No external code was imported or executed.

| Source and evidence type | Observation | Minimal BDH implication and limit |
| --- | --- | --- |
| [Adobe Repository Harness guide](https://opensource.adobe.com/ai-repo-harness-guide/03-Five-Control-Layers/), [build guide](https://opensource.adobe.com/ai-repo-harness-guide/07-Build-Your-Harness/) and [source](https://github.com/adobe/ai-repo-harness-guide) — engineering guidance | Repository-local guides, sensors, context/state, tools/permissions and lifecycle observability; progressive discovery and before/after probes. | Reuse versioned shared rules and executable checks. Not a controlled cross-Agent equality benchmark; clients still need a supported entry. |
| [Beads](https://github.com/gastownhall/beads/blob/main/docs/core-concepts/index.md) — implemented system | Dependency-aware external work graph and deterministic ready queries. | Refine existing task next and checkpoints, not install a second tracker or database. |
| [AgentSpec](https://github.com/yimwoo/agent-spec) — project documentation | Repository-local operating contract, lifecycle and handoff for multiple coding Agents. | Compare portable contracts; no independent utility established here and no dependency adoption. |
| [Agentless](https://arxiv.org/abs/2407.01489v2) — empirical software-repair research | A simple localization, repair and validation pipeline can compete with more elaborate autonomous approaches on its repair benchmark. | Extract bounded deterministic steps, not a universal fixed workflow. |
| [SWE-agent](https://papers.nips.cc/paper_files/paper/2024/hash/5a7c947568c1b1328ccc5230172e1e7c-Abstract-Conference.html) — published interface study | Purpose-built Agent-computer interfaces improve repository operations. | Compact actionable results, clear errors and bounded operations; runtime experiments do not establish arbitrary external Agent utility. |
| [SkillsBench v4](https://arxiv.org/abs/2602.12670v4) — matched benchmark, revised 2026-06-14 | 87 tasks, 18 model/harness configurations; average pass rate 33.9% to 50.5% with curated Skills (+16.6 percentage points). Focused small bundles outperform exhaustive bundles; smaller models sometimes match larger models without Skills. | Small relevant Skills and paired testing; do not mix earlier version figures with v4 or infer BoxWeave gains. |
| [Evaluating AGENTS.md](https://arxiv.org/abs/2602.11988) — empirical study | In the reported settings repository context files tend to reduce success while increasing inference cost over 20%. | Remove redundant procedural context; retain essential owner and safety requirements. Measure fixed startup cost. |
| [An Empirical Study of Harness Design for Coding Agents](https://arxiv.org/abs/2609.20804) — preprint | 176 matched settings across four models; planning helps weaker models' accuracy, predefined tools help weaker shell users, and context management matters with tight budgets. | Offer optional planning and tools with compact context; native shell remains valid. This study controls the runtime loop, unlike BDH. |
| [Beyond the Model / NanoHarness](https://arxiv.org/abs/2609.32459) — preprint | Ten models, three benchmarks; gains vary with model and task, and some compression or generic subagent mechanisms hurt performance. | Evaluate small components separately and remove harmful ones. No new runtime loop, compression or subagent system. |

Continue to use the [R0 reuse research](BDH_R0_REUSE_RESEARCH.md) and
[original design](DEVELOPMENT_HARNESS.md). Codex and other runtime harnesses
are mechanism references, not targets to clone. Before importing anything,
inspect upstream revision, license, dependencies, security, side effects
and maintenance cost under existing L1-L5 review rules. Precedents show
feasibility of patterns, not universal capability parity.

## Stable architecture

Keep Universal Activation, Solve/Evolve Loops, all eight existing systems,
CLI, stable asset identifiers, records and layout. The following five
groups are a responsibility view, not new subsystems or directories.

| Shared capability group | Existing systems and assets | Direction |
| --- | --- | --- |
| Knowledge and methods | Knowledge + Skills; AGENTS.md, INDEX.md, docs, SKILL.md | Short entry, relevant discovery, remove duplication, concrete recipes. |
| Task and durable state | Workflow; work envelopes, task next, checkpoints, stage exits | Outcome/dependency queries and compact portable handoff, no parallel ledgers or compulsory per-slice ceremony. |
| Tools and execution environment | Tools + Environment; bwdev, Make, scripts, Box, Git/MCP | Optional bounded compositions through authorized paths, equivalent native methods allowed. |
| Verification and feedback | Verification; tests, CI, evidence, Finish Gate | Executable invariants and actionable failures; exact-source independent acceptance. |
| Learning and evaluation | Continuous Learning + Evaluation; inbox, revision histories, Eval | Observe during ordinary work, select on authorization, measure real outcomes and full overhead; retire harmful or unused assets. |

Use a small shared contract: task identity/objective, authorized scope,
required observable result, relevant constraints, evidence, next action
or blocker. Optional tools supply missing intermediate work; a capable
Agent can use native methods and deliver equivalent evidence. Choose by
task, risk, actual authorized capabilities and cost, not Agent brand.
Do not introduce per-client profiles, duplicated policy, compatibility
matrices as runtime routing or a capability detection service. A minimal
entry locator may be needed if a client cannot discover the repository.

Mandatory owner, product, security, original Operation, exact-source and
independent verification/CI constraints apply to all routes. Do not
repeat an equivalent operation solely for BDH. Missing capabilities
justify a safe manual route or explicit blocker, not invented access.
BDH cannot guarantee equal reasoning, creativity, tool access or
automatic continuation; it cannot wake a stopped external session.
Persistent next actions help an authorized later caller recover.
Keep repository-local assets and sanitized durable history. No new
repository, database, daemon, scheduler, credential bridge, product MCP
endpoint or product runtime dependency solely for this direction.

## Incremental opportunities, not a prescribed backlog

Eliminate waste before automating it. These hypotheses are investigation
seeds, not selected episodes or new candidate IDs. Reuse existing records
for matching issues.

| Opportunity | Reuse first | Evidence required before selection |
| --- | --- | --- |
| Reduce compulsory startup and repeat reads | BOOTSTRAP, INDEX, short routes, context | Repeated irrelevant reads or latency; unchanged required safety/context. |
| Improve planning and continuation | task next, exit ledger, checkpoints | Reproduced premature stop or stale next action; whole-stage outcomes, not slices. |
| Reduce remote operation toil | batch/failure/transfer helpers, native Git | Recurrent calls/manual work at equivalent scope; actual authorization and availability; preserve L014 gates. |
| Improve error feedback | verification helpers, CI, evidence | Repeated failures or missed checks; actionable localization with unchanged oracle. |
| Distill knowledge and Skills | current docs and Skills | Recurring omission or context cost; compare focused guidance against ordinary route. |
| Simplify or retire redundant assets | learning review and Eval | Native duplication or measured overhead without benefit; preserve history and required gates. |

## Owner-driven evolution

Use the existing [owner-invoked cycle](../../dev/skills/harness-improver/SKILL.md#owner-invoked-improvement-cycle-candidate).
An explicit "BDH优化" starts one bounded cycle within the authorized PR
and ordinary product scope, with unchanged L1-L5 review authority.
A mention or ordinary task does not authorize implementation. This
document does not select an episode.

During ordinary work record material observations once and defer
nonblocking improvements. On each explicit cycle:

1. Review the latest cycle, real reuse, open gates and recurrent friction.
   Distinguish product defects, external runtime limits and BDH gaps.
   Missing evidence stays unknown.
2. Investigate relevant mature mechanisms. Select one small delta,
   including simplification, removal or no change. Define baseline,
   capability hypothesis, acceptance oracle, costs and risk first.
   Prefer existing assets to a new subsystem.
3. Implement, regress, run applicable Eval and independent review/CI;
   reuse in real authorized work where possible. Do not switch clients
   or force unrelated product work merely to manufacture comparisons.
4. Record the result once in existing learning/cycle history, including
   overhead, actual use, limitations, disposition and next safe action.
   Promote only with applicable evidence; otherwise defer, reopen,
   revert or retire. No recursive maintenance cycle.

Retain the owner's small effort / substantial effect heuristic. Never
treat it as a measured benefit, safety exemption or completion quota.

## Measure capability first, efficiency second

Follow the [Eval contract](../../dev/evals/README.md). Deterministic
regressions prove helper behavior, not actual Agent improvement.
Freeze tasks, held-out acceptance tests, starting SHA, equivalent
fixtures, permissions, configuration, budgets and time limits. Compare
ordinary routes with mandatory project constraints unchanged against
optional BDH assistance; compare old versus revised BDH where feasible.
Use the same optional modules across clients. Configuration is experimental
metadata, never a routing rule. Randomize order and repeat isolated runs
when possible; disclose sample size, contamination and missing metrics.

Primary outcomes: independently accepted completion, quality, false
finish, policy violations, interventions, recovery and autonomous
progress. Secondary: all nested calls, I/O, elapsed time, observable
tokens, setup, manual coordination, rework and maintenance, including
failed and unused routes. Predefine meaningful improvement for lower-
performing configurations and tolerable non-regression/cost limits for
higher-performing ones. Compare within configurations first and compare
gaps only on matched feasible tasks with equivalent gates and budgets.
Report unavailable tools separately. Never lower the better configuration
to manufacture convergence, or treat a single success as general proof.

Aim to raise the demonstrated lower-end delivery capability without
material harm to capable Agents. Slower but reliably completed work may
be valuable; needless overhead is a removal candidate. Set thresholds
before each trial, not universal percentages. Cross-Agent capability
compensation remains a hypothesis until measured.

## Current disposition

Documentation and research only, not implementation or promotion.
Existing P1 checkpoint, source work, prior REOPENs and acceptance gates
are preserved. No Box or product change, new PR, merge or release.
The next explicit optimization should select the highest-value
evidence-backed small delta, or record no justified change.
