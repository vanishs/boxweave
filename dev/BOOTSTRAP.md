# BoxWeave task entry (portable)

This is the canonical startup entry for BDH on any Agent surface.
Apply BDH to every BoxWeave task, not every message: usage, support,
installation, diagnosis, design/research, docs, development, release and
BDH itself. Use the smallest workflow that fits the goal and risk. For material work,
use the current authorized task checkout's feedback protocol and toolbox
when available; retain the canonical safety and acceptance requirements.

## Load the task context

1. Resolve the current canonical v1 revision at task start. Read
   [AGENTS.md](../AGENTS.md) (authority, safety and engineering rules) and
   [asset index](INDEX.md) (capability discovery) at that revision. Load
   only relevant contracts, including applicable rules at the actual task
   checkout/ref. Keep the canonical control revision distinct from the
   task source and installed target version; do not switch branches or
   overwrite work merely to load instructions. Reuse reads within the
   task; refresh when resuming after relevant source or policy changes.
2. Recover original task, workspace, target and Operations before changing
   anything. Read the [toolbox and workflow](README.md) as relevant. With
   a Python checkout, prefer bwdev work plan --kind KIND --query QUERY,
   then start/resume, task next, verify, learn and finish as applicable.
   Preserve checkpoints. Without a checkout, use authorized GitHub reads
   and the [remote entry](examples/REMOTE_ENTRY.md); a control checkout is
   optional. Do not install dev tooling on a target Host just for BDH.
   Use existing tools or a documented manual fallback when needed. If
   current canonical instructions are inaccessible, report the limitation
   and do not assume stale rules, permissions or acceptance.

For remote clients, group relevant capability discovery in one lookup where
supported. Reuse exact discovered schemas within the task; discover further
only when needed. Resolve the current checkpoint pointer first, then read
that checkpoint and relevant acceptance rules rather than reloading long
historical progress logs. Expand historical evidence only for a concrete
gap, conflict, or required witness.

For development follow [fast local reuse and strict acceptance](../docs/development/AGENT_WORKFLOW.md#fast-development-strict-acceptance):
reuse authorized Box/worktree and Git objects; batch locally verified commits
and publish at meaningful CI boundaries. Clean independent fixtures are
reserved for gates that require them. Preserve remote freshness, original
Operations and durable handoff.

For staged development, use the active task plan and finite exit ledger;
apply [adaptive stages](../docs/development/AGENT_WORKFLOW.md#outcome-first-adaptive-stages).
At stage boundaries replan the next actions from evidence, not checkpoint
counts. Do not expand scope merely because a dependency is blocked.

## Choose the shortest applicable route

Use the authorized task checkout's [short task routes](INDEX.md#short-task-routes)
and actual capabilities; a candidate checkout does not replace canonical
v1 authority. This is a choice, not an extra mandatory workflow.

- Tiny read-only question: read the relevant contract and answer. No Box,
  plan, work envelope or learning write solely for BDH.
- Existing source check in a Python checkout: start with
  `python3 dev/bwdev --json verify plan`; if a registered check matches,
  use `verify run` and `verify inspect` as described in the toolbox.
  A separate work plan is useful for material or ambiguous work, not a
  prerequisite for a focused check. Use `--brief` only where supported.
- Explicit owner request "BDH优化" (BDH optimization): run one
  [BDH improvement cycle](skills/harness-improver/SKILL.md#owner-invoked-improvement-cycle-candidate)
  on the active authorized task branch. A quotation or discussion of the
  phrase is not an execution request. Review-only requests remain read-only.
- Owner-requested improvement review: start at the learning inbox; use
  the compact existing review view where available, then exact current
  records and original evidence. No implementation without owner scope.
- No usable matching helper or local route: use the authorized manual
  route, record the reason and actual evidence. Do not install a new
  runner or invent a remote executor just for BDH.

Current procedures and historical results are separate: load applicable
rules and execution guidance before tests; consult historical evidence for
recovery, review or comparison, not as a new result. Do not omit required
safety or acceptance guidance to avoid historical exposure.

## Solve Loop and Evolve Loop

3. Discover and reuse relevant knowledge, Skills, tools and workflows.
   Advance safe ready work autonomously through execution, verification,
   correction and delivery until the requested outcome or a genuine
   blocker/budget boundary. Ask the user only for necessary input or
   authorization. Keep advice, recorded progress, native outcomes and CI
   acceptance distinct. Preserve original UNKNOWN Operations; never
   replay uncertain effects, delete resources or weaken safety to continue.
4. At material failures, corrections, decisions, handoff and completion,
   make a [learning judgment](skills/continuous-learning/SKILL.md).
   Reuse and review existing assets first; rediscover relevant deferred
   candidates and fix bounded useful gaps or flag/defer them with evidence.
   Use the [improver skill](skills/harness-improver/SKILL.md), applicable
   review, targeted regression and bwdev eval for harness improvements.
   Implementation evidence alone does not promote a candidate. Preserve
   improvements and sanitized material records through authorized Git/PR
   storage. If persistence is inaccessible, return an explicit pending
   record. Small read-only guidance needs no Box, PR or new asset.

## Before ending material work

Apply the [Finish Gate](skills/task-resume/SKILL.md#finish-gate-candidate)
before returning a final report. Use bwdev finish for structured delivery
where available, or the equivalent manual check. A pending CI run or
completed Exec receipt alone is not a reason to stop. Continue safe ready
work within the requested scope; record a concrete blocker or budget
boundary and the next safe action if continuation is impossible.

The repository owns the workflow; an Agent/project setting only locates
this entry. Do not copy evolving rules into platform configuration. Repo
merge alone cannot install an external trigger; configure once per Agent
surface and verify actual activation. See [remote entry configuration](examples/REMOTE_ENTRY.md)
and [delivery plan](../docs/development/BDH_DUAL_LOOP_DELIVERY_PLAN.md).
