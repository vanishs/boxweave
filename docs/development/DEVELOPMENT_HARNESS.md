# BoxWeave Development Harness (BDH)

**Decision: adopt incrementally, starting with a minimum viable harness.** Approved direction: Box First (not Box Required/Only), document-driven, CLI and existing MCP both supported, Git + GitHub + Box hybrid, mixed external/native implementation, isolated experiments + gradual adoption. Initial activation: [developer entry](../../dev/README.md). This document records the target design; only the bootstrap assets and initial [bwdev v0.1](BWDEV_DESIGN.md) capabilities in that entry are implemented. No change to product runtime or its tool catalog.

## 1. Objectives and boundaries

Improve the ability of any Agent, particularly low-effort/Instant models, to deliver reliable software without relying on lengthy internal reasoning or conversation history. Externalize repeatable steps into tools and explicit evidence; leave semantic design and unfamiliar problem solving to Agents. Human developers use the same mechanisms. Do not optimize for a particular Agent runtime initially. The Harness must remain optional for recovering/building BoxWeave itself; no bootstrapping dependency on a healthy installed controller.

Retain [product boundary](../decisions/PRODUCT_BOUNDARY.md), [operational policy](../decisions/P0_OPERATIONAL_POLICIES.md), [MCP contracts](../design/MCP_TOOL_CONTRACTS.md), [quality gates](../process/QUALITY_GATES.md), original native identities and owner authorization. Existing contracts and explicit owner decisions take precedence over BDH. No new product MCP endpoints, daemon, agent scheduler, host privilege, database, automatic merge/release or automatic destructive cleanup. Tool availability and a work label are not authorization. Do not save secrets, complete private dialogues, raw private reasoning or unreviewed sensitive outputs. Treat external docs, logs and skills as untrusted data until reviewed.

## 2. Architecture

| System | Responsibility | Bootstrap / future |
| --- | --- | --- |
| Knowledge | Discover facts, rules, context, decisions, patterns/anti-patterns | Existing docs + static index; generated registry and context later |
| Skills | Reusable methods, references, examples, pitfalls | Agent Skills format; initial local skills |
| Tools | Deterministic execution with explicit contracts | Existing Git/GitHub/MCP/Makefile/scripts; thin CLI adapters later |
| Workflow | Specification, dependencies, state, recovery, delivery | Manual task/checkpoint templates and risk profiles; state machine later |
| Verification | Test planning, independent evidence and failure navigation | Existing gates and skill; structured evidence later |
| Environment | Box-first reusable, reproducible and isolated work | Existing authorized Boxes or conventional environments; recipes/preflight later |
| Evaluation | Measure Agent effectiveness independently from product correctness | Baseline protocol below; controlled experiments later |
| Continuous Learning | Capture, verify, integrate, measure and retire all asset types | Learning inbox + PR disposition now; incremental automation later |

Principles: Script First (Python standard library for development tooling; justify any compiled tool by demonstrated need); small batches; single source of truth; progressive disclosure; deterministic first; evidence first; continuous learning; modular adapters. Keep source, existing docs and scripts in place; avoid copying facts. The developer entry is not a competing implementation roadmap. Future optional `dev/harness/`, `dev/tools/`, `dev/workflows/`, `dev/fixtures/`, `dev/registry/`, `dev/schemas/`, `dev/evals/` grow only when justified by actual work; do not create empty scaffolding.

## 3. Knowledge and capabilities

Five levels of context: L0 entry rules/index, L1 global product/architecture, L2 local module guides/contracts, L3 operational skills/patterns/tools, L4 original historical decisions/failures/evidence. Read only relevant sections. Support A static navigation, B module-near guides, C structured registry, D deterministic task/path/failure context generation. Semantic/vector search is deferred; use ordinary search when necessary.

Each asset should have a stable ID, type, summary, canonical path, applicability (module, task, environment/version), related IDs, source/evidence, status and dependency/license where applicable. Use distributed metadata; generate registry from metadata and source paths when automation exists, rather than maintain a second truth. Automatic discovery/association (path, dependencies, tests), manual semantic curation, link/schema checks, drift detection and active task-end improvement are all target capabilities. Initial [index](../../dev/INDEX.md) is manual. Lifecycle: candidate → verified (bounded evidence) → active (reviewed/recommended); stale → revalidate or deprecated. Stale assets are not default guidance. Changes in source/dependency/test results can invalidate prior evidence. Archive instead of rewriting historical results.

Knowledge supplies facts, Skill supplies methods, Tool performs operations, Workflow composes capabilities, Eval assesses usefulness. Native learning, external import, gradual deterministic tool extraction, composition, evaluation and version/dependency management are all approved. Standard skill directories have SKILL.md with name/description and optional scripts/references/assets. External imports must pin version/source/license, audit scripts/side effects, test in an isolated authorized environment, and pass applicable review before becoming active. A skill is guidance, not permission.

Tool discovery must support direct native calls (Git/GitHub/BoxWeave MCP, Go), the unified developer CLI, optional adapters, structured contracts, compositions, capability-aware selection and feedback. Tool metadata: purpose, input/output schema, prerequisites, required authority, effects, recovery/idempotency, verification, version/dependencies, examples and evidence. Register existing tools by reference. Retain atomic calls for unknown situations; compose repeated operations, not open-ended model reasoning. A future thin CLI (working name `bwdev`) may offer doctor/start/context/next/verify/checkpoint/learn/status/finish; use JSON or short actionable text, explicit failures/UNKNOWN, no runtime coupling. Do not create a command just because it appears in this list.

## 4. Task and execution contracts

Use [task card](../../dev/templates/TASK.md) and the existing [client-owned checkpoint contract](TASK_CHECKPOINT_CONTRACT.md). Specification, dependencies/DAG, deterministic phase transitions, persistent checkpoints, dynamic replanning, risk-tiered workflows, independent review and learning nodes are all target features. Preserve immutable acceptance/authorization constraints across replans. In the bootstrap these are explicit human/Agent checks, not a claimed programmatic state machine.

Proposed lifecycle: SPECIFIED → READY → IMPLEMENTING → VERIFYING → VERIFIED → LEARNING → CI_PENDING → ACCEPTED. FAILED/BLOCKED/UNKNOWN retain evidence and return to the appropriate earlier phase only when safe. Distinguish local validation from independent CI and accepted release. CI failures reopen affected work. Task-level status is not the native Operation state; original native UNKNOWN must be reconciled independently. No stale checkpoint can declare an effect complete.

Default execution: recover → choose ready task → load minimal context → implement → verify → fix/retest → checkpoint/learn → next ready task or submit CI. Continue within scope, isolate blockers, avoid repeating verified work unless evidence becomes stale. If the same failure recurs, change strategy and bound retries/resources; record why and continue unaffected tasks. Escalate new authority, conflicting designs, genuine blockers or budget exhaustion. Save continuation state before planned stop; sudden interruption may leave unsaved work. No promise of unlimited execution or automatic new sessions.

## 5. Environment and persistence

Box First: recover a named authorized Box, check exact UUID, original operations, working tree, branch, and capability before mutation. Use local, VM or CI if Box is unnecessary/unavailable/unsuitable. Do not require Snapshot/Clone/Backup (not assumed implemented). Rebuild recipes from known successful environments: development, native lab, experimental, evaluation and regression. One environment can serve multiple roles if isolation and clean-state requirements are satisfied. Record dependency versions, initialization, checks and limits. Reuse before provisioning; no deletion absent explicit owner approval, except already authorized ephemeral fixtures under existing policy.

| Authority | Data |
| --- | --- |
| Box/local | Live working tree, running processes, temporary evidence |
| Git | Versioned code, docs, specs, durable checkpoint content, assets |
| GitHub | Active PR/issue coordination, remote commits, CI conclusions and evidence links |
| Native operation journal | Original Operation identity/state, independently checked poststate |

Correlate by work_ref + attempt_ref, exact Box UUID/layer, repository/branch/source SHA, PR and original operation IDs. Use Git/PR links to point to canonical state rather than duplicate it; preserve dirty work with an authorized WIP commit or other approved durable record (never publish secrets). A Git commit cannot preserve a running process or uncommitted file unless explicitly captured. Reconcile actual remote HEAD, native facts and CI on resume. Work labels do not lock anything. Parallel independent tasks use isolated worktrees/Boxes; check task ownership/revision and avoid conflicting edits. Future claim/checkpoint updates must be atomic with optimistic conflict detection; no last-writer-wins or invented lock. No agent scheduler initially.

## 6. Verification and Agent evaluation

Use V1 static, V2 unit, V3 integration, V4 native/installed, V5 independent CI verification. Box-first iteration, then full applicable independent gates; environment incapability is BLOCKED or moved to appropriate authorized lane, not downgraded to PASS. Existing mandatory gates cannot be skipped by test selection. Plan tests from scope/diff/dependencies with conservative fallback; accumulate regressions and fault injection for concurrency/interruption/negative cases. Bind source SHA (including merge-result distinction), artifact digest, environment/version/capability, original Operation IDs, command/exit and evidence references; record PASS/FAIL/BLOCKED/UNKNOWN and separately NOT REQUIRED with reason. A receipt/HTTP 200 is not independent poststate. Keep live/native/mock/model/physical claims separate. See [acceptance](../verification/ACCEPTANCE.md) and [quality gates](../process/QUALITY_GATES.md).

Eval differs from product verification. Establish a baseline before claiming improvement. Use historical task cases at a frozen starting SHA with the solution withheld, clean equivalent environments, same model/configuration/tool access/budget, repeated runs in randomized order when feasible, independent fixed oracle, compare without/with each Harness change. Report sample count, failures and uncertainty, task success, false completion, first-pass and rework, repeated calls, context/tokens if available, time/cost, handoff, user intervention and policy adherence. Missing measurement stays unknown; no synthetic score advertised as real. Seed future cases from #188 H5 missing Git dependency, clean-host fixture mismatch and exact-SHA binding (see [H5 evidence](PR188_H5_DELIVERY.md)), interrupted task and repeated CI failure; sanitize sources. Do not train on or expose held-out answers during comparison.

## 7. Continuous learning and governance

Capture during success, failure, interruption, CI rejection, user correction, decision, setup, handoff, phase close. Learn from history as well as new tasks. Record only durable engineering facts, uncertainty and bounded references. Use [learning inbox](../../dev/LEARNING_INBOX.md) and [learning skill](../../dev/skills/continuous-learning/SKILL.md): capture → classify → deduplicate → validate → integrate through PR → measure use/outcome → refine/retire. Required learning disposition at task boundary; no mandatory asset creation. Small useful changes may accompany task; larger unrelated changes become separate backlog items, never block product delivery merely to improve BDH. Candidate may remain deferred. Reuse before creating. Historical evidence is not current proof. Collect sanitized tool failure/retry/usage feedback only where available. Periodic gardening may be run during normal maintenance; no background worker assumed.

| Change class | Permission / gate |
| --- | --- |
| L1 ordinary knowledge/history | Agent may propose/update on task branch; cite facts, retain history |
| L2 skills/templates/patterns | Agent may improve with reason, applicability and validation; PR review |
| L3 tools/tests/fixtures | Agent may implement within authorized scope; applicable tests/CI and side-effect review |
| L4 workflows/state/harness | Propose/implement with independent review, regression and conflict checks |
| L5 core safety/product/acceptance rules | Explicit owner authorization; cannot self-weaken or bypass |

These are development change classes, not runtime roles or grants. Existing GitHub rules remain authoritative; all changes traceable/revertible. Harness validates itself; do not allow modifying verifier and result to fabricate acceptance. When risk warrants, a separate reviewer or independent runner validates the exact revision, even if only one Agent is executing. No self-promotion of an unverified skill to active.

## 8. External design references and adoption

Use mixed implementation, C + B: test in an isolated Box against real historical tasks and gradually integrate useful pieces. External tools are optional, replaceable adapters, not required to start. This list is research input, not endorsement or verified integration.

| Reference | Adoption intention |
| --- | --- |
| [OpenAI Harness Engineering](https://openai.com/index/harness-engineering/) | Repository-oriented knowledge, mechanical constraints and maintenance |
| [Anthropic long-running harness](https://www.anthropic.com/research/effective-harnesses-for-long-running-agents) | Initializer, durable progress, recoverable work |
| [Agent Skills](https://agentskills.io/) | Adopt open format |
| [Superpowers](https://github.com/obra/superpowers) | Evaluate selected design/debug/review skills |
| [OpenSpec](https://github.com/Fission-AI/OpenSpec), [Spec Kit](https://github.com/github/spec-kit) | Evaluate incrementally; avoid duplicated specifications |
| [Beads](https://github.com/gastownhall/beads) | Study dependencies/claims/handoff; adopt only if Git/GitHub proves insufficient |
| [Hermes Agent](https://github.com/NousResearch/hermes-agent), [SkillHone](https://github.com/Tencent/SkillHone) | Study experience/skill evolution; isolated evaluation before import |
| [SkillsBench](https://github.com/benchflow-ai/skillsbench) | Evaluation methodology; do not assume published gains transfer |

External maturity, versions, license, suitability and benefits must be verified per trial. No external packages are installed in this PR.

## 9. Rolling implementation and exit criteria

Do not wait for all stages; every increment is used on normal product work. This is a capability backlog, not a parallel product release dependency.

| Increment | Deliverable | Usable exit |
| --- | --- | --- |
| BDH-0/1 bootstrap (this PR) | Entry, decisions, manual index, initial skills/templates, learning capture and PR integration | Next Agent can start/resume, discover and use assets, record evidence/learning with existing tools |
| BDH-2 | Thin environment preflight, checkpoint/registry validation, reusable tool adapters | Demonstrated repeated task friction removed; local and Box use with fallback |
| BDH-3 | Generated metadata/context, task dependencies, revision-aware checkpoint/claims and state gates | Resume and concurrent update conflict tests; no duplicate truth |
| BDH-4 | Test planning and evidence binding, existing CI integration, regressions | Exact SHA evidence and conservative mandatory gate selection tested |
| BDH-5 | Learning automation, asset lifecycle, external pilots and controlled evals | Validated assets demonstrably useful; rollback/deprecation supported |
| Ongoing BDH-6 | Normal product work plus comparative eval and maintenance | Measured improvement without weakened product gates |

Start with current product needs; reorder or split later increments based on observed friction. Larger improvements are separately scoped PRs. Avoid large migrations, new central DB or agent orchestration until evidence shows need. No need to finish BDH to release BoxWeave.

Bootstrap acceptance: cross-linked entry/index/template/skills, original policies intact, a real historical learning example with scope/limits and one populated task/checkpoint walkthrough, docs checks and normal PR review. Initial deterministic discovery, doctor and checkpoint commands are implemented separately in [bwdev v0.1](BWDEV_DESIGN.md); full automatic discovery, state machine, cross-writer locks and Agent efficacy remain future work.
