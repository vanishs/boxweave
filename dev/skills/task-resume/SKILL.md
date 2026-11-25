---
name: task-resume
description: Resume BoxWeave development after interruption, lost replies, agent handoff, dirty work, or partial CI without repeating uncertain effects.
---

# Task resume

1. Locate active task/PR and latest canonical checkpoint. If absent, reconstruct from authorized Git/PR/Box records; do not guess.
2. Read current branch/head, remote base, working tree and prior edits before writing. Preserve dirty work; never reset, force-push, or overwrite another Agent's changes.
3. Resolve exact owned Box UUID/layer; check current operation journal via available MCP. Use original work_ref and operation IDs. Check original receipt, guest exit and native poststate; UNKNOWN is not success or permission to repeat. If unavailable, record blocker; do not create replacement merely because a response was lost.
4. Compare current source/artifact and CI runs to the checkpoint. Reuse only evidence applicable to the same candidate and scope; rerun stale or affected checks. Distinguish prior head, merge-result and final candidate.
5. Continue the next ready task, or isolate blocker and work on unrelated safe tasks. At the next meaningful boundary update the durable checkpoint, preserve original evidence and write exact next safe action.

References: [checkpoint contract](../../../docs/development/TASK_CHECKPOINT_CONTRACT.md), [template](../../templates/CHECKPOINT.md), [operations](../../../docs/design/OPERATIONS.md), [manual recovery](../../../docs/process/UNKNOWN_MANUAL_RECOVERY.md). This skill grants no authority and does not implement automatic resume or a task lock.

Optional v0.5: use bwdev resume with original task history and explicit checkpoint; supply existing evidence and CI snapshot as available. Review REVIEW_REQUIRED items, preserve original operations, and export a sanitized handoff summary to the active PR. No replay or remote state authority.
