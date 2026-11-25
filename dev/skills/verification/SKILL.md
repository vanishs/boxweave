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
