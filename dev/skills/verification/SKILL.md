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

## Batch boundary

Follow the [fast development policy](../../../docs/development/AGENT_WORKFLOW.md#fast-development-strict-acceptance):
focused checks before each local commit, affected aggregate regression at
the combined candidate, independent CI/native evidence at meaningful
boundaries. Retain exact final-head and clean-fixture gates; do not repeat
unchanged valid evidence without an applicable reason.

## Staged source batch (candidate)

For a reviewed, staged-only source slice in an authorized development
checkout, use the candidate dev/staged_batch.py instead of reconstructing
test/vet/format/export commands each time. Pass exact parent and staged
tree, a fresh evidence directory outside checkout, explicit affected and
adjacent Go packages (--go-package ./internal/state etc.) and any matching
registered Python checks (--check bwdev/scope/recovery-fixture). The helper
runs diff check, selected Go test -count=1 and vet, selected existing checks
and format; stops on failure or source drift, retains private per-check
logs and digests, emits compact status and text export only on local PASS.
It does not stage, commit, dispatch native work or decide required checks.
Do not use for binary/deletion/symlink or unstaged changes; retain the
ordinary fallback and all applicable CI/native gates.

For export rows with base_blob and append, read that exact path at the
payload parent via authorized GitHub; require returned blob SHA equal
base_blob, reconstruct exact text and remove delta fields. All other rows
contain full content. Verify every page, original Operation exit and
identity; create a tree from the stated base only after reviewing all rows;
require returned remote tree equal the exact staged tree, then commit and
advance only with an expected-parent lease, never force. If reconstruction
or evidence fails, stop publication and preserve the original worktree.
For multi-batch delivery where rereading the entire original export would
be wasteful, add --framed-export to the same staged_batch.py command.
Choose explicit --frame-rows and --frame-bytes from the ordinary transport
budget (defaults 6 rows and 65536 bytes, not a platform quota contract).
On actual local exit zero, stdout begins with an ASCII JSON index and
one newline; each frame is a consecutive ASCII JSON object with the same
parent/base/tree and distinct export rows. The index has frame number,
offset relative to the byte after the newline, byte count, SHA256 and
row count, plus total body bytes, rows and canonical export hash. Retain
the original Operation and index. With authorized byte-range output reads,
read only the necessary exact ranges, verify frame length/digest/identity,
and account for every frame and distinct path before tree assembly.
If range reads or hash verification are unavailable, use the ordinary
verified export route; do not probe limits, retry uncertain mutations,
or claim a frame is a commit. Compare exact complete remote Git tree,
then use the existing non-force expected-parent lease. Preserve successful
checks and the original receipt on interruption; review ambiguous writes
rather than automatically repeat them. Single small exports need no frame
overhead. Private record.json remains the complete local check record.

Logs are private; do not publish raw logs. A staged-tree result is local,
not independent CI/native acceptance. Record measured nested calls, bytes
and rework; do not infer general productivity from one trial.


## Go failure navigation (candidate)

For iterative Go regression on an authorized checkout with existing plain
(non-JSON) go test logs, prefer one read-only projection over repeatedly
constructing grep and source-mapping snippets:

    python3 dev/go_failure_index.py CURRENT_LOG --previous PRIOR_LOG

The prior log is optional. Keep actual original operation exit, complete
logs, exact source, and applicable test command. The output groups reported
top-level failure names by current tracked declaration files, preserves
package failures (including build errors), flags missing/ambiguous names,
and marks previously failed names not reported now. It is a textual
navigation aid, not Go AST analysis or subtest coverage, and never a
PASS verdict. A missing name, zero failures, or exit zero of
the summary cannot establish successful or nonempty execution. Source
mapping reflects the current worktree, not the historical log's tree.
Use actual test output for each replacement, and do not retire behavior
solely because an old name is absent. Keep raw logs private and expand
only the relevant failure. No connector access or automatic selection.

For a developer-selected package or coherent test group, run one bounded
local Exec from the checkout root instead of separate test/log/index
operations:

    CGO_ENABLED=1 python3 dev/go_failure_index.py --run ./internal/state --pattern '^TestExample$' --evidence-dir /tmp/fresh-evidence

Use an unused evidence directory outside the checkout and the actual
approved packages/regex (omit pattern for all tests). Optionally pass
`--previous` for a prior log. The helper retains argv, original test
exit, full combined log/digest, source observations before/after and
compact top-level failure/run/pass/skip navigation. It returns nonzero for
test failure, observed source drift or zero reported passing tests.
Absence of result.json means no completed batch verdict; recover original
Operation rather than replay. Raw evidence remains private. Observe the
original Operation, guest exit and required streams separately. The local
source observation is not a Git tree, does not cover ignored inputs or
transient drift, and is not independent native proof. Reported unique
names are navigation, not complete per-package/subtest acceptance.
Use actual named assertions and full applicable gates before delivery.
Do not use the helper to run untrusted source or weaken a mandatory gate.
Batch related edits in the same authorized worktree; avoid concurrent
Exec on a busy Box. Reuse existing staged helper for verified publication,
ordinary exact-tree Git for removals. No new product tool or scheduler.
