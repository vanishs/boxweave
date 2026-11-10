# Lightweight Agent Task Checkpoint — client-owned handoff contract

**Purpose:** help any Agent/workflow client resume long-running work that used BoxWeave; this is **not** a BoxWeave database record, MCP Tool, authentication role, private model thought log or new product boundary. [Design owner](../design/AGENT_OPERATION_CONTINUITY.md). Agents in projects other than BoxWeave may implement the same checkpoint format in their own storage.

## Example format (illustrative, safe placeholder IDs only)

```yaml
format_version: 1
work_ref: project-x/test-cycle
attempt_ref: agent-run-007
updated_at_utc: "2026-11-10T00:00:00Z"
objective: "Validate project X in a disposable Box"
stage: "verify"
source:
  repository: "project-x"
  commit: "example-commit-sha"
  worktree: "dirty"   # clean | dirty | unknown
verified_completed:
  - "Prepared disposable environment; native Box readback recorded"
current_uncertainty:
  - "Asynchronous test result not yet independently observed"
original_operations:
  - operation_id: "example-op-1"
    tracking_id: "example-track-1"
    verified_state: "UNKNOWN"
    evidence_ref: "private-evidence-record"
do_not_repeat:
  - "Do not create a replacement Box before exact original postread"
next_safe_action: "With authorized client, operation_get(example-op-1), then inspect exact native UUID"
links:
  - "repo-relative-evidence-or-issue"
```

## Minimal rules

- **No content duplication:** stable `work_ref` is shared across the workflow/client, not a secret or a permissions grant; `attempt_ref` can change on handoff. Use separate authenticated BoxWeave credentials if the user wants trustworthy per-Agent attribution. Preserve original BoxWeave `operation_id` and native ID, never rewrite them to make a checkpoint look complete.
- **Write at meaningful boundaries:** before context compaction, after admitting an irreversible mutation (persist the returned original receipt), after a verified result, before handing work to another Agent, and before user-requested stop. **Sudden crashes can occur before a checkpoint is written**; missing information must be rediscovered from original indexed journal, Git and native postread. Do not claim zero data loss.
- **Checkpoint state isn't native truth:** a declared `completed` step must carry evidence ref/source SHA; `UNKNOWN` remains UNKNOWN even if a summary says “probably finished”. Reconnect with `operation_list(work_ref)`, then `operation_get(original_id)`, native readback and Git/CI checks before further mutations.
- **Concurrency and privacy:** the Agent/workflow's Git/revisioned store owns checkpoint merge/optimistic concurrency; preserve conflicts rather than last-writer-wins lost information. No tokens/keys/host secrets/guest stdout/raw prompt chains/private env in project-shared checkpoint; use scoped evidence references and sanitized summaries. For private artifacts, follow local organization controls.
- **Compatibility:** plain file/Git is an acceptable first implementation; optional external workflow systems may maintain their own checkpoints. BoxWeave never needs to parse this format or implement `checkpoint_save`/`checkpoint_get`; a later MCP Tasks adapter is unrelated to this broader work checkpoint.

## Continuation prompt contract (for clients/operators)

> Resume the original task from the most recent authorized checkpoint. Verify the repository HEAD and dirty tree and query original BoxWeave operations using the authorized `work_ref` and exact IDs. Never auto-repeat a possible native effect or assume a stale checkpoint proves completion. If an original operation is UNKNOWN or permission is missing, preserve it and continue independent safe work; record any required decision and the next exact read-only probe.

**Tests:** S2.5 TC-011–014 (new Agent + independent operation verification), S3 TC-015–016 (Git/guest edit/failed test fix), S5 TC-019–020 (real client replay and independent verification). This client-owned artifact is **not** a new S2 BoxWeave schema table or a precondition for unrelated clients.
