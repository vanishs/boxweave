# BoxWeave Development Harness (BDH) — start here

**Status: bootstrap / usable manually.** This is a development-only layer, not a product runtime feature. Use it on the next ordinary task; do not wait for the future CLI or registry generator. [Design and decisions](../docs/development/DEVELOPMENT_HARNESS.md) | [asset index](INDEX.md) | [learning inbox](LEARNING_INBOX.md).

## Start, continue, finish

1. Read [AGENTS.md](../AGENTS.md), confirm owner intent and active PR/branch/HEAD. Select the smallest workflow that fits the risk. For an ongoing task, locate its checkpoint and original workspace/operations before creating or changing anything.
2. For nontrivial work, fill [task card](templates/TASK.md) in the active PR or linked file. For small work, a few equivalent PR fields suffice. List observable acceptance, scope, dependencies and required knowledge/skills from the [index](INDEX.md).
3. Prefer an authorized existing Box; use [environment skill](skills/box-development/SKILL.md). Local/VM/CI or direct GitHub docs edits are valid when sufficient; record why. Work in a topic branch.
4. Implement, run focused tests, fix failures, and continue independent ready work. Use [verification skill](skills/verification/SKILL.md); final required gates remain governed by [quality policy](../docs/process/QUALITY_GATES.md) and product acceptance contracts. Never silently reduce scope or repeat a possible UNKNOWN effect.
5. At material boundaries, before handoff, and when stopping, save [checkpoint](templates/CHECKPOINT.md) outside the Box using a branch/PR or approved durable store; preserve original IDs and dirty work. Use [resume skill](skills/task-resume/SKILL.md).
6. At failures, material decisions, corrections and completion, capture a learning candidate or disposition in the task record. Apply [learning skill](skills/continuous-learning/SKILL.md). No new asset is required if nothing reusable was found. Small verified improvements can ship with the task; larger ones go to the inbox.
7. Complete the PR template, including evidence, remaining gates and learning disposition. Keep working within scope until acceptance or a genuine blocker/budget boundary. Never claim CI/native PASS without exact evidence. Do not merge or release without applicable authorization.

## Workflow profiles

| Profile | Typical use | Minimum path |
| --- | --- | --- |
| Lightweight | Small low-risk bug/docs | Scope → change → applicable checks → learning → PR |
| Standard | Ordinary feature/refactor | Spec → tasks → implementation → verification → learning → CI |
| Advanced | Network, security, host lifecycle, architecture | Research → design/authority review → phased work → native independent proof → learning → CI |
| Experimental | External tools or uncertain approach | Isolated trial → comparison → evidence → adopt/reject decision |

Risk, not diff size, determines profile. All profiles retain the existing safety/approval boundaries; product-specific gates always win. A task can be locally verified and submitted while CI remains pending. No background agent, scheduler, lock service or automatic status transition is installed by this bootstrap.

## First use and next improvement

The initial skills and templates are ready for the next task. Record actual friction and candidate improvements rather than build every planned feature upfront. First priority: automate the repeated, demonstrated steps in checkpoint validation and context discovery; then add environment preflight/test selection. See the phased roadmap in the design. Use existing Makefile, Git, GitHub and MCP directly today.
