**2026-11-15 S3 verification refinement:** the disposable supervised Git harness now checkpoints first/second Git commit IDs, `final_git_clean`, the original four BoxWeave IDs **and their native Incus Operation IDs**, plus source SHA-256. Following installed-service restart, a fresh authenticated MCP process compares the same original action/UUID/native IDs and persisted exits; the outer Incus verifier separately checks Git HEAD, file bytes and an empty porcelain worktree. This is bounded **harness evidence**, not a replacement for a real autonomous Agent run, not an authenticated `agent_ref`, and not independent raw Incus daemon terminal-operation capture. [Exact scope](S3_FILE_TOOLS_PHASE3_GIT_ACCEPTANCE_2026-11-15.md).

# Lightweight Agent Task Checkpoint — client-owned handoff contract

**2026-11-15 revised timing:** S3 only captures the minimal Guest Git SHA/dirty state and original Operation IDs in a supervised **single-Agent** evidence row. **Full second-Agent recovery and two-writer revision conflict TC-015/016 are mandatory in S5**; do not ship without their release proof. [Current phase contract](S3_S4_S5_MINIMAL_REBALANCE_2026-11-15.md).


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

**Tests:** S2.5 TC-011–014 (new Agent + independent operation verification), S3 limited one-Agent Git/guest edit/failing test fix smoke (not TC-015/016 PASS), S5 **full TC-015–016** two-Agent conflict/recovery plus TC-019–020 (real client replay and independent verification). This client-owned artifact is **not** a new S2 BoxWeave schema table or a precondition for unrelated clients.
