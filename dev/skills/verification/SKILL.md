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
