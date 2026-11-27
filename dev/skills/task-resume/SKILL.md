---
name: task-resume
description: Resume BoxWeave development after interruption, lost replies, agent handoff, dirty work, or partial CI without repeating uncertain effects.
---

# Task resume

1. Locate active task/PR and latest canonical checkpoint. If absent, reconstruct from authorized Git/PR/Box records; do not guess.
2. Read current branch/head, remote base, working tree and prior edits before writing. Preserve dirty work; never reset, force-push, or overwrite another Agent's changes.
3. Resolve exact owned Box UUID/layer; check current operation journal via available MCP. Use original work_ref and operation IDs. Check original receipt, guest exit and native poststate; UNKNOWN is not success or permission to repeat. If unavailable, record blocker; do not create replacement merely because a response was lost.
4. Compare current source/artifact and CI runs to the checkpoint. Reuse only evidence applicable to the same candidate and scope; rerun stale or affected checks. Distinguish prior head, merge-result and final candidate.
5. Continue the next ready task, or isolate blocker and work on unrelated safe tasks. For nonterminal original Operations use the bounded observation policy in [verification](../verification/SKILL.md#bounded-operation-observation-candidate), not repeated short polling. At the next meaningful boundary update the durable checkpoint, preserve original evidence and write exact next safe action.

References: [checkpoint contract](../../../docs/development/TASK_CHECKPOINT_CONTRACT.md), [template](../../templates/CHECKPOINT.md), [operations](../../../docs/design/OPERATIONS.md), [manual recovery](../../../docs/process/UNKNOWN_MANUAL_RECOVERY.md). This skill grants no authority and does not implement automatic resume or a task lock.

Optional v0.5: use bwdev resume with original task history and explicit checkpoint; supply existing evidence and CI snapshot as available. Review REVIEW_REQUIRED items, preserve original operations, and export a sanitized handoff summary to the active PR. No replay or remote state authority.


Prefer the same owned Box/worktree and preserved local commits.
Record both observed remote base and local HEAD/tree; verify before reuse.
Do not refetch known immutable objects or dispatch against a known busy
original. An external pointer to local commits is not independent backup.
See [fast development policy](../../../docs/development/AGENT_WORKFLOW.md#fast-development-strict-acceptance).

## Early checkpoint before call-budget pressure (candidate)

The client may exhaust its tool-call budget without warning. Count actual
observed nested tool attempts since the last durable checkpoint, including
failed calls, polling and output pages; do not count only outer scripts.
Use a cheap running tally or estimate marked as such, not additional tool
calls just to count. If the platform exposes no quota, remaining quota is
UNKNOWN. Do not infer a fixed platform limit or attempt to bypass it.

At a meaningful boundary, save verified progress before a costly export,
multi-page read, commit publication, long validation or another work slice
if pressure is plausible. As an initial adjustable advisory cadence, review
checkpoint freshness after about 20 observed tool attempts since the last
durable save (or earlier after repeated failures or a budget warning).
This is a review trigger, not a hard stop, quota estimate, or requirement
to save unchanged state. Reuse existing evidence and batch the save with
normal work. Reserve the next available calls for preserving new material
state when necessary; if pressure is not a blocker, continue ready work.

Use the existing checkpoint capture/template or a concise authorized PR
comment with exact parent, workspace, dirty/staged tree identity, verified
and pending results, original Operation IDs, and next safe action. Record
where actual bytes remain; a Git tree hash or comment is not a backup.
Persist the record outside the Box before starting an expensive dependent
sequence when feasible. Confirm the save returned success; a local-only
record, uncertain save or unavailable remote is not remotely durable.
On resume verify the original IDs and actual tree before any new effects.

Record observed or estimated call count and scope, checkpoint overhead,
and whether a later continuation actually reused the checkpoint. Do not
claim crash recovery or improved productivity from a saved record alone.
No automatic scheduler, quota probe, new permission or product tool.


At each phase boundary, use the active plan's finite exit ledger and
[adaptive stages](../../../docs/development/AGENT_WORKFLOW.md#outcome-first-adaptive-stages).
A ready task must advance a remaining exit condition or a necessary
dependency. Do not generate extra tasks just to avoid ending a blocked
phase. If repeated work does not reduce the remaining critical path,
replan the next stage; preserve all mandatory proof and owner decisions.

## Phase-wide continuation (candidate)

For a multi-slice phase, bind existing task history to the **whole
requested phase**, not today's edit or test. Reconcile its finite exit
ledger, exact candidate, mandatory obligations, dependencies and evidence
on resume. Missing history requires initialization from actual evidence
or an equivalent complete manual ledger, never an empty ready list.

On an owner-directed phase change, reconcile the full new phase exit
ledger before querying continuation. Use a distinct phase work_ref and
one active history; preserve prior revisions and explicitly carry any
unmet obligations into the new graph. Do not interpret an older phase's
empty ready list as a stop decision for the current phase. Supply
`--expect-work-ref ACTIVE_REF` on task next/status/transition/replan
and workflow status/resume/finish; a mismatch requires reconciliation,
not a handoff. Derive ACTIVE_REF from the authorized current task, not
from the history being checked. Equivalent manual routes must perform
the same identity comparison.

At each meaningful checkpoint, including a small success or local
failure, use the continuation projection already returned by task
transition, or `bwdev task next HISTORY` in the same save batch where
possible. CONTINUE means execute the next required safe action now;
do not report and wait. The projection includes WORKING, ready PENDING,
blocked and waiting dependencies. REVIEW_BLOCKERS requires examining
each remaining dependency and independent work; REVIEW_EXIT requires
comparison with the complete phase exit criteria. Neither authorizes
stopping or claims PASS. Only actual observed inability, missing
authorization or exact dependency failure justifies BLOCKED; never
retry a refused action. RECORDED is evidence navigation, not acceptance.

Before final response, save the latest phase-wide decision, exact next
action and explicit stop reason with the existing checkpoint/PR.
Reuse records, avoid redundant history reads, and do not invent optional
work. An actual platform boundary or owner stop remains valid.
This is advisory, not a client termination hook.

## Finish Gate (candidate)

For history discovery, prefer exact original Operation IDs and the current
work_ref. When searching history, bound by the last trusted checkpoint time
(with overlap where needed), work_ref and a small page; expand only for
missing evidence, conflicts or required recovery. Review older UNKNOWN and
unresolved originals without a recent-time filter. A bounded list is not
proof that no older Operation exists.

Before ending material work, reuse task status, the current CI snapshot
and the receipts already held; do not add repetitive list/poll calls.

1. Consume required original terminal output, including pagination and
   observed exit status; read required stderr or other evidence. If not
   available, retain the original ID and concrete blocker; never replay
   uncertain effects. A terminal receipt is not the result.
2. Inspect ready and WORKING tasks, including independent work while CI
   waits. Continue safe actionable work within scope. If none is actionable,
   distinguish requested completion, verified blockers and actual budget
   boundaries; record the exact next safe action. A checkpoint, local
   commit, progress report, test failure or pending CI is not a stop
   condition. A budget estimate triggers preservation, not termination.
   Record observed limit/error or actual inability to proceed; do not
   assert quota exhaustion without evidence. Honor explicit owner stop.
   If required work remains ready and no such boundary is present,
   continue the same authorized pass; never request a routine continue.
3. For structured delivery, use existing bwdev finish with
   --operations-reviewed after this review (including no Operations);
   repeat --pending-output ORIGINAL_ID for each known unread required
   output. CONTINUE_OR_RESOLVE directs the caller to resolve evidence or
   continue safe work, REVIEW_REQUIRED requires review, and HANDOFF_READY
   is only local advisory readiness. The flags are caller reports, not a
   remote scan, and do not establish completeness or authority.
4. Without structured records, apply the same checklist manually and save
   the result with the existing checkpoint/PR handoff and learning judgment.
   No invented task history, extra scheduler, or forced polling. Record
   scope, evidence, remaining actionable work, and explicit stop reason.

This candidate cannot intercept a client ending a response without invoking
it. Verify actual ordinary-task use and costs before claiming efficacy.
