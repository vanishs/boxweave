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
