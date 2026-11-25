# BoxWeave task entry (portable)

This is the canonical startup entry for BDH on any Agent surface.
Apply BDH to every BoxWeave task, not every message: usage, support,
installation, diagnosis, design/research, docs, development, release and
BDH itself. Use the smallest workflow that fits the goal and risk.

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

The repository owns the workflow; an Agent/project setting only locates
this entry. Do not copy evolving rules into platform configuration. Repo
merge alone cannot install an external trigger; configure once per Agent
surface and verify actual activation. See [remote entry configuration](examples/REMOTE_ENTRY.md)
and [delivery plan](../docs/development/BDH_DUAL_LOOP_DELIVERY_PLAN.md).
