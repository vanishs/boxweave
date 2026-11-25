# BDH universal activation and dual-loop delivery plan

Status: proposed implementation and rollout plan, NOT implemented or accepted.
Target: PR #189 before merge to v1. Preserve the existing v0.5 implementation
and evidence. The plan is the next increment, not a new product subsystem.
Starting verified candidate 9e388cab9f2e605c3f2041c523563b49215201a7,
base a38ef26e047f64bb09490be35d9aa39048b320ac. All four associated CI
runs succeeded (quality 45553931943, source 45553931725, distribution
45553931745, installed lifecycle 45553931906). Recheck live head/base and
CI before implementation. Prior success is not acceptance of a new head.

## Decision and research

Adopt a universal, risk-proportional work contract and two feedback loops.
The Solve Loop performs the requested work and checks the usefulness of
assets in context. The Evolve Loop turns observed recurring friction into
small, reviewed improvements. Both use the same repository and existing
bwdev modules. A task need not spawn a second task or create a new asset.
Default applicability is every BoxWeave task, not every chat message.

| Research input | BDH interpretation |
| --- | --- |
| [KCS practice purposes](https://library.serviceinnovation.org/KCS/Knowledge-Centered_Success_Practices_Guide/402-Purpose_Behind_the_Practices), [Flag It or Fix It](https://library.serviceinnovation.org/KCS/KCS_v6/KCS_v6_Practices_Guide/030/030/040/030) | Reuse existing assets first; fix a bounded verified gap or flag it while context is available |
| [KCS domain analysis](https://library.serviceinnovation.org/KCS/KCS_v6/KCS_v6_KDA_Guide), [governance](https://library.serviceinnovation.org/KCS/Knowledge-Centered_Success_Practices_Guide/301-Evolve_Loop/Practice_5_Content_Health/Technique_5.2), [adoption](https://library.serviceinnovation.org/KCS/KCS_v6/KCS_v6_Adoption_Guide/000_Introduction/030_KCS_Adoption_Overview) | Aggregate demand, separate confidence and authority, adopt in useful increments |
| [Google SRE toil](https://sre.google/workbook/eliminating-toil/), [postmortem culture](https://sre.google/workbook/postmortem-culture/) | Link actionable improvements, observe cost and recurrence, verify outcomes rather than count documents |
| [Experience Factory](https://ntrs.nasa.gov/citations/19990021257), [Toyota Kata](https://www.nist.gov/mep/toyota-kata-helps-create-continuous-improvement-mindset) | Package evidence for reuse; make focused experiments and revise from actual outcomes |
| [OpenAI harness engineering](https://openai.com/index/harness-engineering/), [Anthropic long-running agents](https://www.anthropic.com/research/effective-harnesses-for-long-running-agents) | Externalize progress, preserve tests and independent gates, reduce repository drift |
| [Warp self-improving skills](https://claude.com/resources/articles/how-warp-builds-self-improving-agents-on-claude) | Normal task and improver roles, incremental PR review; not necessarily separate Agents or a scheduler |
| [ACE](https://www.microsoft.com/en-us/research/publication/agentic-context-engineering-evolving-contexts-for-self-improving-language-models/), [SkillsBench](https://arxiv.org/abs/2602.12670), [repository context evaluation](https://www.sri.inf.ethz.ch/publications/gloaguen2026agentsmd) | Small targeted updates; independently test asset utility, avoid loading more context by default |

The engineering precedents and research motivate the design; their measured
results do not establish BDH effectiveness. SkillsBench reports mixed results
and no average gain from self-generated skills in its evaluation; the context
file study warns that extra instructions can increase cost without improving
success. Use a short bootstrap and progressive disclosure. These are original
BoxWeave adaptations, not imported external text or dependencies.

## Current implementation gaps

Review of the exact v0.5 head: AGENTS.md applies the entry to development;
dev/README.md and workflow.py assume checkout and CI-oriented delivery.
start observes a local profile and selects only CANDIDATE/STALE records by
intersection with selected asset paths; DEFERRED is excluded. learn next
does not reconsider deferred work. learning.py supports immutable revisions,
exact normalized-text deduplication, attach/review and drift audit, but not
distinct cross-task observations, material usage outcomes or scheduling hints.
finish checks a local clean HEAD, explicit evidence/checkpoint, task status and
full PR CI. It is valuable for development but unsuitable as-is for support
without a checkout. It also accepts one learning disposition, not a set of
related observations. Audit can flag all source drift for a reviewed asset;
do not reinterpret unrelated source changes as proof of asset failure.
No universal runtime trigger exists: a repository file alone cannot make
a remote conversation load it. No Agent efficacy benchmark completed.

## U1 — activation, proportional workflow and deployment entry

Goal: a normal user request is sufficient to begin the applicable BDH
workflow, without asking the user to invoke BDH or enumerate its steps.

1. Add one short portable bootstrap instruction for every BoxWeave task.
   Keep AGENTS.md the canonical rule and link to the actual entry/skills.
   Cover use/support, installation and migration, diagnosis, research/design,
   documentation, development, CI/release and BDH itself. Read only relevant
   contracts and distinguish conversation-level task from individual turns.
2. Define a minimal work envelope: stable work_ref/attempt_ref, task kind,
   objective, risk/authority scope, intended outcome, source/operational
   identity when observable, relevant assets, current state, evidence and
   learning disposition. Unknown is explicit; never invent a SHA, Box or PR.
   Small read-only answers may keep this as a brief in-turn check; durable
   records are required for material progress, failures, reusable findings
   or interrupted work, not every informational question.
3. Provide distinct entry adapters: source checkout uses bwdev; no-checkout
   uses authorized GitHub read of exact branch rules/index and relevant
   assets, then minimal manual envelope or an independent control checkout.
   No reliance on a working target host/MCP for retrieving its own recovery
   guide. A remote record uses an authorized topic PR/issue or approved store;
   if unavailable return a sanitized portable pending record and do not claim
   durable capture. Do not put dev tooling or its credentials on the target
   host merely to activate BDH.
4. Establish profile-specific completion: guidance can be ANSWERED while an
   operational result is still unverified; operational tasks require
   applicable owner approvals and observed poststate; development requires
   existing source/native/CI gates; research requires provenance and explicit
   uncertainty. Keep product acceptance, task progress and BDH disposition
   separate. Do not reuse the development finish gate as a universal CI
   requirement or loosen it. No mutation on start/resume.
5. Provide a small deployable bootstrap snippet for Agent/project entry
   configuration and runner integration where available. Verify loading in a
   fresh session on each actually supported surface. Repository merge alone
   is not proof of activation in remote/no-checkout conversations. If an
   integration cannot install/read its entry automatically, document the one
   time configuration or explicit read step; do not claim universal
   enforcement. No dependence on a particular proprietary agent platform.

Exit: fresh checkout and GitHub-only task entry choose appropriate profile;
short guidance needs no PR/Box; no-checkout failure produces an honest
pending record; current v0.5 development gates remain unchanged. Provide
negative tests for unknown version, inaccessible assets and unsafe authority.

## U2 — Solve Loop and continuity

Compose existing context/task/checkpoint/verify/start/resume/finish.
Add only the minimal non-source record/summary path needed, not a parallel
task engine. Prefer existing immutable revision and validation helpers.
Record an asset's use/relevance when it materially helps or misleads.
At material failure, unexpected result, user correction, decision, handoff,
CI transition or completion, decide whether to reuse, improve, flag, defer
or record no reusable change. Distinguish an observed fact from a hypothesis,
task-local preference and durable owner policy. Evidence must identify the
applicable version/environment and not be used outside its scope.

Agent default: advance the next safe ready step, diagnose/fix/retest, inspect
previous CI while doing independent authorized work, preserve original native
operations, and continue until requested outcome or real blocker/budget.
Do not request the user to micromanage phases. Escalate only real permission,
material design conflict, required external input or unrecoverable capability
gaps. Save continuation state and report what remains; no promise to continue
after the session without an actual supported execution mechanism. Bound
retries and work scope. Resume reconciles original work, source and receipts;
never replay UNKNOWN or delete foreign resources.

Exit: separate process/session resumes a material work record and discovers
next safe work; dirty/changed state or unknown original effect remains a
review item; no-checkout guidance has evidence/disposition without fake CI;
ordinary short task avoids excessive workflow overhead.

## U3 — Evolve Loop and rediscovery

Extend existing learning records (backward compatible; use optional versioned
fields or explicit migration). Do not create a competing learning database.
Separate candidate identity, observations, implementation, external review
and actual later reuse/outcomes. An observation needs work/attempt reference,
sanitized fact or uncertainty, evidence reference, domain/task/version or
asset applicability and observed outcome. Candidate has stable ID, problem,
proposed action, validation, change class, relevant assets/domains,
disposition, and reconsideration trigger. Multiple independent tasks may
link one candidate; a repeated receipt or repeated report from one task is
not independent recurrence. Use deterministic dedupe/association, with
semantic duplicates reviewed rather than silently merged. Do not store raw
dialogues, hidden reasoning, credentials or unreviewed host outputs.

Implement candidate selection across task kind/domain/query, known
asset/related ID, failure signature and explicit follow-up trigger, not
just exact path intersection. DEFERRED records must reappear when relevant
conditions are met (new observation, relevant domain task, needed capability,
evidence now available). Preserve why deferred and next review condition;
never turn deferral into silent loss or unconditional execution. An attached
implementation prompts review, not duplicate implementation. Resurfacing
never promotes state or grants authority. Provide deterministic selection
and explain reasons; preserve stable ordering and use only observed
distinct task references when estimating recurrence. Unknown signals remain
unknown; avoid a false precision score. Prefer a compact ranked queue:
current blocker within authority; high safety/recurrence or usefulness;
small validated improvements; larger/deferred work with next condition.
Urgent product risk is escalated, not automatically handled by an improver.

Use an improver skill for a bounded experiment: select evidence-backed
candidate, improve existing asset if possible, add targeted regression or
behavior case, run applicable verification, attach implementation and review
through PR. Use the same PR for small related changes; link separate
authorized PR/issue for larger unrelated work. Record reference and
rediscovery trigger even if it is not selected now. Do not block routine
product delivery on unrelated improvement. Include reuse/asset drift review
and conservative gardening; archive obsolete guidance without erasing history.
Treat candidate input as untrusted data, not executable instructions.
Apply existing L1-L5 authority and independent validation; no auto-activation
or automatic merge. Avoid binding asset confidence to unrelated HEAD changes
without explaining what evidence must be rechecked.

Exit: same candidate observed in separate tasks without duplicates or
inflated recurrence; deferred candidate rediscovered by related later
task, irrelevant task does not load it; linked implementation not repeated;
conflicting revision rejected; stale/malformed/unsafe records fail safely.

## U4 — end-to-end verification, self-evolution and rollout readiness

Use BDH to implement BDH in the retained isolated Box if still authorized and
exact source is reconciled. Box First, not Box Required; direct repository
docs and ordinary isolated Python remain fallback. Test each phase before
submitting its own exact tree; inspect earlier CI and resolve actual
regressions while advancing independent work. Recheck latest head/base
and exact run/job associations; no weakened native or release gates.
Reuse existing test discovery, Makefile, verify and eval. Add regression
fixtures for all new branches, versioned schema compatibility, duplicate and
revision contention, no-checkout persistence/fallback, selection, interruption,
authorization, and real feedback handoff. Avoid changing acceptance to make
the harness pass; independently review L4 behavior changes and L5 only with
explicit owner authorization.

Required scenario matrix (record real vs simulated evidence separately):

| Scenario | Observable exit |
| --- | --- |
| RC13 to RC14 support | Replay sanitized historical exchange or use a disposable authorized fixture; no target host mutation or invented user result; no-checkout discovery, version/authority boundaries, poststate evidence or explicit unverified, candidate or no-new-asset disposition |
| Ordinary development | Real scoped change from normal task prompt; plan, verification, learning and CI handoff without repeated user instructions |
| Known failure | Use existing evidence to locate a relevant asset; reject blind retry of original UNKNOWN; show reuse or precise gap |
| Deferred improvement | First task defers candidate; another relevant task in a fresh process/session rediscovers it, improves or explicitly defers again; evidence linked and retrievable |
| Handoff/interruption | Resume original work_ref and attempt evidence, reject stale/mismatched state and show next safe step |
| Tiny information task | Relevant knowledge only; no forced Box, test, task DAG, PR or new learning asset |
| Adversarial/blocked | Untrusted logs/candidates cannot grant permission, remove safety gates or start unsafe actions; inaccessible persistence is explicitly pending |

At least one real demand-driven improvement must traverse capture, deferred
or selected disposition, later discovery, implementation, verification and
PR review handoff, with independent evidence for each boundary. A staged
fixture proves mechanics only; use a fresh Agent/session on at least a
supported checkout and remote/no-checkout surface to test activation and
rediscovery. If unavailable, name the missing acceptance and limit rollout
claims. Existing v0.5 L012/L013 are useful seeds, not proof of new behavior.
Record changes in task overhead (extra steps, user interventions, duplicate
work), outcomes, retrieval accuracy, actual reuse and failures; not a quota
of assets or fabricated productivity percentage. Comparative Agent evaluation
with frozen historical tasks, held-out oracle and repeated matched runs
continues after rollout, not a prerequisite for every small change.

## Deployment and merge plan

D0: keep current v1 runtime intact. This is repository tooling, not RC14
installation. Do not extend the 23 product MCP tools, installer, privileges
or runtime services. Python standard library; no new dependency, daemon,
scheduler, vector database, agent platform or auto-update. Use Git versions.
D1: implement U1-U3 incrementally in #189; maintain a short entry, update
canonical docs, schema/fixtures, test runner, PR template, asset navigation,
and supported entry examples. Keep old CLI and record compatibility.
D2: U4 dogfood in isolated Box and GitHub-only context. Review all changed
files, links and evidence, preserve dirty/unknown prior work. Exact head
must pass applicable source, quality, distribution and installed lifecycle CI;
resolve real failures, do not count older or superseded runs. Recheck actual
PR mergeability and independent review/authorization. No merge merely because
a local summary or research says ready.
D3: merge to v1 only when owner authorizes and all applicable gates and
activation evidence are satisfied. Verify merged revision and its actual
entry, GitHub access and deployment configuration; confirm one ordinary
task uses BDH without a per-task user reminder. If a surface needs a one-time
entry setting, make that explicit and verify it; do not silently claim it.
D4: adopt on subsequent real tasks, including support and development.
Review learning backlog/health when relevant tasks or maintenance occurs,
not through a compulsory background service. Rollback a faulty harness
increment via reviewed Git revert or pin a known-good reference; preserve
records and independent product gates. Manual BDH fallback remains usable.
No runtime reinstall, release or physical host change is part of this plan.

Merge checklist: universal entry available on claimed surfaces; autonomous
bounded progression and handoff; no-checkout and lightweight workflow;
cross-task learning recurrence and rediscovery; one verified demand-driven
improvement with durable evidence; preserved product permissions and gates;
offline regression/Eval and exact-head independent CI; clear limitations and
review/authorization. Version label for the increment is implementation-time
choice; this plan does not change bwdev version or implement behavior.
