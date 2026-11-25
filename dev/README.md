# BoxWeave Development Harness (BDH) — toolbox and workflow

**Status: bootstrap plus bwdev v0.6.0-dev.1 (stacked candidate).** This is a task-side development and operational learning layer, not a product runtime feature. Use it on the next ordinary task; do not wait for future automation. The initial [script toolbox](../docs/development/BWDEV_DESIGN.md) is optional and ready for use. [Design and decisions](../docs/development/DEVELOPMENT_HARNESS.md) | [asset index](INDEX.md) | [learning inbox](LEARNING_INBOX.md).

Universal activation and dual-loop design and rollout gates: [universal activation and dual-loop delivery plan](../docs/development/BDH_DUAL_LOOP_DELIVERY_PLAN.md).

## Universal entry

Start every BoxWeave task with the canonical [bootstrap](BOOTSTRAP.md). External Agent/project configuration needs only the [minimal startup instruction](examples/REMOTE_ENTRY.md); the repository owns evolving rules. A tiny question can use only relevant knowledge and a learning judgment. For material work without a target checkout, use the portable [work envelope](examples/WORK_ENVELOPE.json) via GitHub or an independent control checkout. Unknown target fields stay null; the control repository SHA is not the installed version. A local record is not remote persistence. Use existing start/resume/finish for source development.

```sh
python3 dev/bwdev --json work plan --kind operations --query 'install migration'
python3 dev/bwdev --json work check dev/examples/WORK_ENVELOPE.json
```

Use work begin DIRECTORY SPEC to publish a material work envelope.
At material progress, failure, correction, handoff or completion, create an
updated JSON envelope (without revision), then work boundary DIRECTORY FILE
--expect N. Preserve identity, original evidence and permission scope; include
a learning disposition and next safe action. work resume reads original
references and review requirements; work export writes a fresh local copy.
The target does not need a checkout. Follow [GitHub-only entry](examples/REMOTE_ENTRY.md)
for remote tasks. UNKNOWN remains UNKNOWN; use linked reconciled work only
after independent review. Version changes on the same target need new evidence.
A material outcome record is a caller assertion, not independent PASS.

## Material feedback protocol (v0.6 candidate)

For material tasks use the current task checkout's envelope template with
feedback_protocol 1. Preserve actual feedback facts at each boundary;
selected assets are not evidence of use. Each material asset_feedback item
records asset (registry ID or canonical path), evidence, outcome, attempt_ref,
observation, decision, candidate (ID or null), and reason. Decisions: linked,
no-change, not-applicable (explain why related candidates do not apply),
pending. Link known relevant candidates with learn observe and the expected
revision; include their IDs in learning_links. Never invent an observation
or force a new asset. Use the same sanitized evidence reference and fact in
the work record and learning revision.

Run bwdev work feedback FILE to review exact local links; work boundary and
resume also surface review. Pending feedback can be durably handed off
without blocking independent product work; a final RECORDED/ANSWERED
boundary cannot assert closure with unresolved protocol-1 feedback.
Use bwdev finish with --work-history DIRECTORY for structured delivery review,
rather than legacy reason-only learning flags. Legacy records remain readable
but require explicit review; no automatic migration. This is an advisory
record check, not proof of remote persistence, completeness of observations,
native acceptance, or independent learning efficacy.

For GitHub-only tasks, record the same fields in an authorized PR/issue or
a portable pending record and reconcile in a control checkout when available.
Tiny read-only questions need no record. If the current task ref implements
a newer authorized harness, use that checkout's instructions and tools while
retaining the canonical v1 safety and acceptance requirements.

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

Run focused tests with `python3 dev/harness/check_runner.py 'test_bwdev*.py'` (or make test-bwdev).
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

Plan uses actual merge-base, committed, staged, unstaged and untracked paths. A missing base means conservative full scope. Only explicit registered source checks run; no native mutations or CI calls. Local PASS on a dirty tree remains UNBOUND. Inspect compares the observed clean HEAD, not evidence authenticity. Use the normal CI and installed/native contract for acceptance. The output digest is recorded; v0.5 can retain a private local diagnostic log with --retain-log. Never publish raw logs blindly.

## Learning toolbox (v0.4)

Create a fresh explicit inbox directory with mkdir. Capture sanitized observations and references.

```sh
mkdir /tmp/my-learning-inbox
python3 dev/bwdev --json learn capture /tmp/my-learning-inbox L001 --work-ref example/task --observation 'Record exact test scope' --evidence 'PR evidence link' --asset dev/skills/verification/SKILL.md --validation 'Review the scope regression' --change-class L2
python3 dev/bwdev --json learn audit /tmp/my-learning-inbox
```

Use learn review with expected revision, explicit review/evidence references and applicable L4 independent-review or L5 owner references. A recorded VERIFIED or ACTIVE is a reviewer assertion, not authority, a registry update or independent proof. Audit detects exact duplicates and reviewed asset/source drift. Keep live histories outside the source worktree; persist sanitized copies in Git/PR and reconcile copies. Markdown inbox remains valid.

## Demand-driven evolution (v0.5.1)

Use [the improver skill](skills/harness-improver/SKILL.md) for material gaps.
Reuse existing assets first.

```sh
python3 dev/bwdev --json learn next dev/learning --domain box-exec --signal admission
python3 dev/bwdev --json work plan --kind harness --domain box-exec
```

Capture can include --domain, --signal, --task-kind, --impact and --effort.
Use learn observe with candidate ID, expected revision, work/attempt ref,
new evidence, observation and outcome to link cross-task feedback.
learn defer records a reason and reconsideration condition. Relevant deferred
items reappear without auto-promotion or execution; reopen via review to
CANDIDATE before attach. start and work plan show reasons and next actions.
Exact evidence duplicates are refused; semantic duplication and independent
recurrence need review. Finish accepts --related-candidate for additional
candidates linked to the work or observations. Missing inbox is not no-learning.

## Integrated workflow (v0.5)

For a nontrivial task with a checkout, use the compositional commands rather
than rediscovering the sequence. Existing individual commands remain usable.

```sh
python3 dev/bwdev --json start --profile development --path internal/install
python3 dev/bwdev --json start --profile development --spec dev/examples/TASK_SPEC.json --history /tmp/my-history
python3 dev/bwdev --json status /tmp/my-history
python3 dev/bwdev --json resume /tmp/my-history --checkpoint /tmp/my-checkpoint.json
python3 dev/bwdev --json eval
```

Start reports selected assets, relevant learning candidates, environment
profile and verification plan. Initialization is explicit. Resume reads
original local records; it does not replay native effects. Status and finish
can include checkpoint, repeatable --evidence, --inbox and exact CI snapshot.
Use --export with a fresh local filename for a JSON handoff and copy the
pr_summary field into the active PR after sanitizing. Finish requires a
learning disposition (candidate:ID from the same work ref and supplied
inbox, or no-new-asset:REASON), recorded tasks, local evidence, checkpoint
and exact repo/PR/base/CI metadata; unresolved items remain REVIEW_REQUIRED.
A handoff summary is never permission to merge.

See [v0.5 usage and contracts](../docs/development/BWDEV_DESIGN.md#v05-integrated-workflow).
The [environment profiles](profiles.json) are declarative prerequisite recipes,
not installers. Native lab readiness still needs independent authority and
capability checks. Use [Eval cases](evals/README.md) when changing the harness.

## Start, continue, finish

1. Read [AGENTS.md](../AGENTS.md), confirm owner intent and active PR/branch/HEAD. Select the smallest workflow that fits the risk. For an ongoing task, locate its checkpoint and original workspace/operations before creating or changing anything.
2. For nontrivial work, use bwdev start when a Python checkout is available, or fill [task card](templates/TASK.md) in the active PR or linked file. For small work, a few equivalent PR fields suffice. List observable acceptance, scope, dependencies and required knowledge/skills from the [index](INDEX.md).
3. Prefer an authorized existing Box; use [environment skill](skills/box-development/SKILL.md). Local/VM/CI or direct GitHub docs edits are valid when sufficient; record why. Work in a topic branch.
4. Implement, run focused tests, fix failures, and continue independent ready work. Use [verification skill](skills/verification/SKILL.md); final required gates remain governed by [quality policy](../docs/process/QUALITY_GATES.md) and product acceptance contracts. Never silently reduce scope or repeat a possible UNKNOWN effect.
5. At material boundaries, before handoff, and when stopping, save [checkpoint](templates/CHECKPOINT.md) outside the Box using a branch/PR or approved durable store; preserve original IDs and dirty work. Use [resume skill](skills/task-resume/SKILL.md).
6. At failures, material decisions, corrections and completion, capture a learning candidate or disposition in the task record. Apply [learning skill](skills/continuous-learning/SKILL.md). No new asset is required if nothing reusable was found. Small verified improvements can ship with the task; larger ones go to the inbox.
7. Use bwdev finish for nontrivial work when records are available (or equivalent manual checks). Complete the PR template, including evidence, remaining gates and learning disposition. Keep working within scope until acceptance or a genuine blocker/budget boundary. Never claim CI/native PASS without exact evidence. Do not merge or release without applicable authorization.

## Workflow profiles

| Profile | Typical use | Minimum path |
| --- | --- | --- |
| Lightweight | Small low-risk bug/docs | Scope → change → applicable checks → learning → PR |
| Standard | Ordinary feature/refactor | Spec → tasks → implementation → verification → learning → CI |
| Advanced | Network, security, host lifecycle, architecture | Research → design/authority review → phased work → native independent proof → learning → CI |
| Experimental | External tools or uncertain approach | Isolated trial → comparison → evidence → adopt/reject decision |

Risk, not diff size, determines profile. All profiles retain the existing safety/approval boundaries; product-specific gates always win. A task can be locally verified and submitted while CI remains pending. No background agent, scheduler, lock service or automatic status transition is installed by this bootstrap.

## First use and next improvement

The initial skills and templates are ready for the next task. Record actual friction and candidate improvements rather than build every planned feature upfront. The first CLI now covers environment observation, context discovery and basic checkpoint capture/check. Task continuity is now available via task init/next/status/transition. Verification planning and bounded local evidence are now available. The integrated start/resume/status/finish, profiles, CI navigation and behavior Eval are available. Future automation is driven by real product work and controlled evaluation; see the [bwdev roadmap](../docs/development/BWDEV_DESIGN.md). See the phased roadmap in the design. Use existing Makefile, Git, GitHub and MCP directly today.
