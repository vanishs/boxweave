# BoxWeave Development Harness (BDH) — start here

**Status: bootstrap plus bwdev v0.4.** This is a development-only layer, not a product runtime feature. Use it on the next ordinary task; do not wait for future automation. The initial [script toolbox](../docs/development/BWDEV_DESIGN.md) is optional and ready for use. [Design and decisions](../docs/development/DEVELOPMENT_HARNESS.md) | [asset index](INDEX.md) | [learning inbox](LEARNING_INBOX.md).

## Quick start (optional Python CLI)

Run directly from the source tree; no installation or compilation.

```sh
python3 dev/bwdev --help
python3 dev/bwdev --json doctor
python3 dev/bwdev --json context --path internal/install
python3 dev/bwdev --json context --query "CI failure"
```

Use `doctor --require go --require base` when a task needs the Go toolchain
and origin/v1. A missing optional dependency is not a global blocker.
Use `checkpoint capture` and `checkpoint check` with the
[template and example](templates/CHECKPOINT.md); persist the local checkpoint
through an authorized remote branch or PR. The commands do not claim native
readiness, automatically resume work, or replace existing gates.

Run focused tests with `python3 -m unittest discover -s scripts -p 'test_bwdev*.py' -v`.
Full applicable tests and CI remain mandatory.

## Task history (v0.2)

Create a JSON specification with work_ref, objective and tasks (id, acceptance, depends_on). See the [example](examples/TASK_SPEC.json). Use a fresh history directory outside the worktree.

```sh
python3 dev/bwdev --json task init /tmp/my-task-history dev/examples/TASK_SPEC.json
python3 dev/bwdev --json task next /tmp/my-task-history
python3 dev/bwdev --json task transition /tmp/my-task-history inspect --expect 1 --to WORKING --note 'Read the original evidence'
```

Task states are PENDING, WORKING, BLOCKED and RECORDED. RECORDED only records a claimed result and reference, never product acceptance. Publish each revision exclusively; reconcile divergent copies through Git/PR. No distributed claims, automatic native replay, remote sync or proof of evidence authenticity.

## Verification toolbox (v0.3)

```sh
python3 dev/bwdev --json verify plan
python3 dev/bwdev --json verify run bwdev /tmp/my-check-evidence
python3 dev/bwdev --json verify inspect /tmp/my-check-evidence
```

Plan uses actual merge-base, committed, staged, unstaged and untracked paths. A missing base means conservative full scope. Only explicit registered source checks run; no native mutations or CI calls. Local PASS on a dirty tree remains UNBOUND. Inspect compares the observed clean HEAD, not evidence authenticity. Use the normal CI and installed/native contract for acceptance. The output digest is recorded, not the log; run the underlying script directly when you need its diagnostic output.

## Learning toolbox (v0.4)

Create a fresh explicit inbox directory with mkdir. Capture sanitized observations and references.

```sh
mkdir /tmp/my-learning-inbox
python3 dev/bwdev --json learn capture /tmp/my-learning-inbox L001 --work-ref example/task --observation 'Record exact test scope' --evidence 'PR evidence link' --asset dev/skills/verification/SKILL.md --validation 'Review the scope regression' --change-class L2
python3 dev/bwdev --json learn audit /tmp/my-learning-inbox
```

Use learn review with expected revision, explicit review/evidence references and applicable L4 independent-review or L5 owner references. A recorded VERIFIED or ACTIVE is a reviewer assertion, not authority, a registry update or independent proof. Audit detects exact duplicates and reviewed asset/source drift. Keep live histories outside the source worktree; persist sanitized copies in Git/PR and reconcile copies. Markdown inbox remains valid.

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

The initial skills and templates are ready for the next task. Record actual friction and candidate improvements rather than build every planned feature upfront. The first CLI now covers environment observation, context discovery and basic checkpoint capture/check. Task continuity is now available via task init/next/status/transition. Verification planning and bounded local evidence are now available. Structured learning is now available; future automation is driven by real product work and controlled evaluation; see the [bwdev roadmap](../docs/development/BWDEV_DESIGN.md). See the phased roadmap in the design. Use existing Makefile, Git, GitHub and MCP directly today.
