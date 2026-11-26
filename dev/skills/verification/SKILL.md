---
name: verification
description: Plan and execute applicable BoxWeave tests, diagnose CI failures, and bind independent native acceptance evidence to the exact candidate.
---

# Verification and CI recovery

1. Read relevant task acceptance, [quality gates](../../../docs/process/QUALITY_GATES.md) and [acceptance](../../../docs/verification/ACCEPTANCE.md). Identify changed surfaces and mandatory checks. Use docs-only path only when policy actually allows it.
2. Check environment with [box-development](../box-development/SKILL.md). Run focused tests first, then all applicable local gates (see [Makefile](../../../Makefile)). Record command, exit, test scope and candidate SHA/artifact digest. Keep real/native/installed, source/mock/model and physical evidence distinct.
3. For a failure, inspect original failing case/job and smallest relevant logs. Reproduce in suitable authorized Box or other environment, identify root cause (or label hypothesis), change one bounded thing and rerun the original plus affected regressions. Capture new failure signatures, do not repeatedly rerun identical failures without new evidence.
4. For native effects, read original operation result and independent actual poststate including negative/prohibited effects. Receipt alone, HTTP 200, mock success or stale evidence is insufficient. Preserve UNKNOWN. Do not change production constraints to accommodate a fixture.
5. Submit the exact candidate to applicable CI; inspect actual run/job conclusion and head SHA. If workflow tests a merge result rather than head, record both identities. Record PASS/FAIL/BLOCKED/UNKNOWN or NOT REQUIRED with reason separately for each mandatory gate. Pending is not PASS. CI success does not imply physical or release acceptance.
6. If correction changes source, invalidate affected evidence and rerun gates. Keep blockers and limitations visible; don't merge until required conditions and authorization are satisfied.

Historical example: [#188 H5](../../../docs/development/PR188_H5_DELIVERY.md) shows missing Git/base ref, populated fixture limits and source binding. Use it as bounded learning, not current success evidence. Update [learning inbox](../../LEARNING_INBOX.md) if there is reusable value.

When CI overlaps across revisions, separate cancelled/superseded runs from actual failures. Use bwdev ci with the exact PR head, PR number and base SHA, from an authorized GitHub export or gh read. Inspect the specific current run, job and artifact before declaring acceptance. A missing, ambiguous, cancelled or pending current run is not PASS. See [BDH-L012](../../learning/BDH-L012/00000001.json) and the [CLI contract](../../../docs/development/BWDEV_DESIGN.md).


## Consolidated CI observation (BDH-L012 candidate refinement)

At a meaningful decision boundary (resume, failure triage, ready-to-submit,
or closeout), resolve the actual PR head/base once. Request one workflow-run
list for that exact SHA using the authorized repository-level CI interface.
Reuse that snapshot for all relevant gates in the current decision; do not
issue separate status requests per workflow, reread unchanged terminal runs,
or query every intermediate/superseded commit. Refresh on a new head, when
a pending result gates work, or when evidence is stale for the next decision.
Continue independent ready work between observations.

Compare returned rows with the applicable workflow inventory and exact
candidate. Keep run ID, workflow identity, status/conclusion and observation
time. Only request job metadata/logs or artifacts when investigating a
failure, ambiguity, skipped gate, or verifying required acceptance evidence.
Do not skip mandatory job/artifact proof to save calls.

Prefer the existing bwdev ci adapter with complete raw GitHub run data.
A connector list scoped by requested SHA may expose only run ID, workflow
name/ID, status and conclusion (first page). Treat it as navigation/status
observation, not a full PR/base/attempt-bound snapshot. Do not manufacture
missing event/path/head/base/association fields or claim completeness; resolve
missing/duplicate/ambiguous expected rows and exact identity using authorized
reads before acceptance. The existing bwdev ci full scope is four named
workflows; include additional task-specific gates independently. No current
gate is PASS merely because it is absent or an earlier SHA succeeded.

Record sampled list, job, log and retry calls separately; compare only
equivalent scopes. One list replaces per-workflow status lookups where
supported, but does not establish whole-task cost savings. No new product
tool, CI exemption or automatic scheduler.

## Bounded Operation observation (candidate)

For asynchronous Box work, retain the original receipt and ID. Do not poll
immediately after every admission or repeat short status reads while a long
test is running. Do independent ready work first (for example source or CI
inspection). When a dependent result is needed, use one bounded server-side
wait via operation_get, up to the maximum advertised by actual tool discovery
(rc.14: wait_ms=20000). A quick read is appropriate for a result likely already
terminal, or a specific immediate diagnostic need. If waiting is unsupported,
use spaced observations at meaningful work boundaries.

A wait-budget expiry is not a failure, cancellation or permission to retry.
If still nonterminal, continue independent work or use another bounded wait
only when its result is required; never spin with repeated zero/short waits.
When no independent work remains, preserve the original and pending checkpoint
if the client budget cannot support another wait. Do not dispatch duplicate
work. UNKNOWN requires original read-only reconciliation, not replay.

Read output only when the original reports it ready or terminal; verify actual
guest exit and required stdout/stderr pages. SUCCEEDED alone is insufficient.
Keep all existing native, authorization, evidence and CI checks.

During ordinary development record per sampled Operation: number of
operation_get calls, wait durations, premature output reads, final outcome
and dependent-work delay if observable. Count nested calls, including failures,
not just outer script calls. Compare against prior actual samples without
claiming equivalent workload or measured savings unless controlled evidence
exists. Candidate guidance, not a new product tool or automatic scheduler.
