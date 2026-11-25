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
feedback_protocol 1. Preserve actual feedback facts at each boundary.
Record only materially used assets in used_assets, not selected or merely
discovered assets; every listed asset needs feedback. A missing asset can
be recorded as an explicit gap, not falsely counted as helpful use.
Each material asset_feedback item
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
boundary cannot assert closure with unresolved feedback. New work begin
requires protocol 1. Legacy histories remain readable and may be handed off,
but completion requires explicit reconciliation to protocol 1; no automatic
migration. Reconcile known assets, evidence, candidate observations, current
learning links and inbox availability. Corrected candidate decisions may
change current links only with a new correction_ref in the item and new
review evidence in the envelope; earlier links and facts remain in prior
revisions. An exact recorded observation must remain linked, even when
the proposal is declared not-applicable; missing observations from current/represented attempts
also require review. Conflicting recorded associations require explicit
review/handoff, not an automatic unlink or rewritten observation.

A historical implementation attachment substantiates reuse, not a new
improvement. For an improved disposition, attach a changed asset on a
clean source with learn attach --work-ref WORK --attempt-ref ATTEMPT
(in addition to expected revision, implementation-ref and validation-evidence).
Record an exact linked observation for that originating attempt. Add
implementation_claim to the work envelope with candidate_revision
(the attachment event revision, not necessarily latest observation
revision), implementation_ref, validation_evidence and handoff_ref
(null for the originating attempt). Include the two evidence references
in evidence; target.source_sha must equal the attachment's observed clean
commit. A later attempt of the same work requires an additional explicit
handoff review reference in evidence. Unchanged asset/digest or old
validation evidence is not a new bound improvement. Legacy unbound
attachments remain readable and support reuse only. Use an authorized
separate inbox/control record location while validating clean source;
persist the reviewed records through Git/PR afterwards, distinguish
tested code SHA from a later evidence-only commit, and run applicable
exact-head checks for delivery. These are consistency checks of supplied
references, not authentication of external evidence or automatic promotion.

Use bwdev finish with --work-history DIRECTORY for structured delivery review,
rather than legacy reason-only learning flags. The complete work history,
objective, checkpoint attempt and exact repository/source must agree. This is an advisory
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
recurrence need review. The v0.5.1 legacy --related-candidate finish
flag is historical; use structured learning_links and exact observations
in the protocol-1 envelope. Missing or invalid inbox is not no-learning.

### Residual opportunity review (v0.7 V1a candidate)

For a new observation with explicit residual friction, optionally pass
`learn observe ... --residual-file FILE` with a reviewed local JSON object:

```json
{
  "signature": "manual-assembly",
  "impact": "friction",
  "environment": null,
  "cost": null,
  "uncertainty": "No measured cost yet"
}
```

Use only observed cost; when known, cost contains value (nonnegative
integer), unit (manual-actions, tool-calls, bytes, milliseconds or tokens)
and provenance (explicit evidence reference). Null means unknown.
Impact is blocker, risk, friction or unknown. Signature is a bounded
failure/friction label; no executable content. Historical observations
without residual metadata remain readable and are not backfilled.

The advisory opportunity in learn next, work plan, start and related
material feedback identifies review signals and exact evidence beyond
a history-derived attachment/lifecycle-review boundary. It does not
prove earlier issues were resolved, authorize work, establish independent
recurrence, or mutate candidate state. Review can be due even when
existing Skill use helped and the candidate is VERIFIED or ACTIVE.
V1b records explicit review decisions; V2a/V2b bind a bounded selection
to the existing local task history. Terminal resolution remains pending.

### Durable decisions and improvement episodes (v0.7 V1b/V2 candidate)

Inspect an exact candidate using:

    python3 dev/bwdev --json learn opportunity dev/learning BDH-L014

Use the returned revision and basis for an explicit decision:

    python3 dev/bwdev --json learn decide INBOX ID --expect N --basis HASH --decision preserve --reason REASON --review-ref EVIDENCE --when CONDITION

The choices preserve, defer, reject and select are opportunity dispositions,
not lifecycle promotion. An explicit distinct trigger reference is needed
when no new review signal exists. Select also needs --proposal FILE with
objective, assets (canonical bounded harness paths including candidate asset),
authority_ref, risk, exact clean source_sha, baseline, expected_outcome,
max_attempts (5..50), acceptance (implement, validate, review, deliver,
benefit), and next_safe_action.

Keep mutable learning and task histories in an explicitly authorized
directory outside the source worktree while checking clean source. Persist
sanitized reviewed records through Git/PR. Inspect/materialize the selected
episode with:

    python3 dev/bwdev --json learn episode INBOX ID --expect N --history-root DIR
    python3 dev/bwdev --json learn episode INBOX ID --expect N --history-root DIR --materialize

The existing task DAG has one ready implement step. Explicit transitions use
--event FILE --task-expect TASK_REVISION with step, to, attempt_ref,
actor_ref, authority_ref, checkpoint_ref, evidence_ref, reason,
reconcile_ref and outcome. WORKING and RECORDED use null outcome/reconcile;
BLOCKED uses blocked or unknown and null reconcile; returning BLOCKED to
PENDING requires resolved and distinct new reconcile_ref. No automatic
dispatch; do not replay uncertain effects. Complete implement only with a
new exact work-bound learning attachment. Later task records use the exact
implemented clean source (use a distinct checkout for evidence publication).
Separate reviewer/beneficiary references are necessary but not proof of
independence. Final recorded outcomes remain review required until explicit terminal
review with exact matching later learning evidence. Use the canonical
history root (repository docs/development/BDH_V07_EPISODES or an external
inbox sibling episodes directory). See V2c below.

Independent review can explicitly reject an exact implemented source
(event step review, to BLOCKED, outcome rejected). An already performed
external review may be imported from READY with a distinct reviewer/attempt
and original evidence; this is caller-recorded, not authenticated identity.
Record a separate review/PENDING/rework event against that exact rejected
source and attempt, with distinct reconciliation evidence. This resets the
current DAG to implement, retaining all immutable prior events/attempts
and the original budget. Corrected source must descend from the rejected
implementation, stay inside selected assets, bind a new attachment and
revalidate before independent review. Use --source-root for learn episode
and learn attach with an external canonical inbox and clean scoped checkout
when harness source differs from the scoped implementation. No new authority.
A separate reviewer may resolve-episode --outcome abandon for a known
pre-delivery blocker/rejection or exhausted budget with failure evidence;
active, UNKNOWN or delivered effects cannot be abandoned. This reopens
opportunity review without inventing benefit or promotion.

Resolve only after every episode step is recorded and matching later
ordinary-task learning evidence exists:

    python3 dev/bwdev --json learn resolve-episode INBOX ID --expect N --task-expect T --outcome benefit-confirmed --review-ref REF --actor-ref ACTOR --reason REASON

Use reopen for residual or non-helpful evidence. Resolution does not
promote assets; read audit replays terminal history. A material work
boundary persists evolve_handoff independently of feedback success.
Separate local histories must be reconciled in Git; no distributed lock.

See [V2 progress](../docs/development/BDH_V07_V2_PROGRESS.md) and
[V2c progress](../docs/development/BDH_V07_V2C_PROGRESS.md).

### Offline transfer preparation (candidate)

With a reviewed Python checkout, prepare a bounded manifest and exact
exclusive-create requests without network dispatch:

    python3 dev/transfer_plan.py plan --source SOURCE --output NEW_LOCAL_DIR --box BOX_UUID --target /tmp/NEW_FILE --work-ref WORK --transfer-ref UNIQUE_TRANSFER

Source is nonempty, up to 4 MiB, with 1..40960-byte parts (at most 128).
The output directory must not exist. Keep the original canonical bundle
in a trusted local durable filesystem; request JSON and readback pages
contain source bytes, not safe to commit indiscriminately. Each action
has a stable key and exact path. Read current canonical status, never
a saved NOT_SUBMITTED snapshot:

    python3 dev/transfer_plan.py assess BUNDLE

Before EVERY connector dispatch, including assembly, atomically append
and fsync its exact intent (expected revision from current assess):

    python3 dev/transfer_plan.py begin BUNDLE --expect N --action ACTION

Only a newly returned DISPATCH_INTENT_RECORDED permits the caller, with
existing authority, to send that exact request once. The helper never
dispatches. A concurrent claim, failed/uncertain begin, interrupted
caller or missing receipt must be reconciled using the original key
and actual original Operation, never begin again or use a stale output.
Use the returned request file only for this immediate dispatch; confirm
its SHA256 against the returned request_sha256. Preserve the original
receipt before independent readback. Append the actual receipt and/or
readback (not a replacement snapshot):

    python3 dev/transfer_plan.py record BUNDLE --expect N --action ACTION --receipt RECEIPT_JSON

Receipt fields: idempotency_key, operation_id, state, pages, exit_code.
Use UNKNOWN with no invented Operation ID, pages or exit code if the
original response is unavailable. Record observed success before or
with readback; only assembly uses actual guest exit code. Reconcile via
original Operation or manual owner review if no original receipt can
be recovered; no reset to NOT_SUBMITTED, no automatic retry. Missing or
changed history fails closed; legacy assess MANIFEST JOURNAL is refused.
A copied/relocated bundle is not a new dispatch authorization. This
local journal is not a distributed lock, tamper-proof storage or native
power-loss proof. Keep actual original bundle, do not restore an older
snapshot or copy it into a competing writer. No API credentials, network
client or new MCP Tool.

Only after all part and manifest receipts and bytes match, durably begin
the assemble action, then invoke the same reviewed helper in the owned
guest once via a short explicit argv with the returned exact key/path:

    python3 /REVIEWED/CHECKOUT/dev/transfer_plan.py assemble EXACT_GUEST_MANIFEST

The assembly uses verified ordered parts and exclusive same-directory
hard-link publication; it never replaces or removes the final target or
original parts. Existing files and unsupported publication fail closed.
Observe actual guest exit zero, independently read every final page, and
assess again. VERIFIED_READBACK means supplied bytes and receipts match,
not authenticated remote evidence or native crash/power-loss proof.
The helper cannot establish authority, remote persistence or independent
utility. Manual fallback: [Box skill](skills/box-development/SKILL.md).

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
pr_summary field into the active PR after sanitizing. Finish requires --work-history DIRECTORY with reconciled protocol-1
feedback and a recorded outcome, recorded tasks, local evidence, checkpoint
and exact repo/PR/base/CI metadata; unresolved items remain REVIEW_REQUIRED.
The v0.5 reason-only --learning-disposition flag cannot establish closure.
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
