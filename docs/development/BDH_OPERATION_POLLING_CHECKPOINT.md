# BDH bounded Operation observation candidate (PR #191)

Owner-authorized narrow improvement, 2026-11-26. No new product tool,
scheduler, capability, or change to operation semantics. Base
f1b848436eff00ca80301cb239690d5fddd31aa4. Canonical control
9d07665dcfc5ae6345d6bfcd0d71de7b6510b51d. Existing L014 transfer work
remains deferred/review-required; this is a distinct status-observation
cost gap, not a new transfer implementation.

## Actual observation and intervention

Previous P1 work reached the Code Mode tool-call ceiling before the
validated d20 publication. Exact total/remaining platform quota is not
observable. Two trace samples: original 92fe03b2b3180f668809154f4ffd09bc
(state race/build) and 628754cf8d9b9972df60b0228a720481 (75 Python tests)
each used four operation_get calls with wait_ms=1000. Both exited zero.
These are selected examples, not a measured whole-task baseline.

Actual rc.14 MCP discovery allows operation_get wait_ms 0..20000; this is
a read-only wait, not a cancellation or retry. Prefer independent work
before a needed read, then one bounded server-side wait. No early output
access, no duplicate dispatch, no relaxed UNKNOWN, native or CI proof.

## Implementation / evidence

Extend existing verification Skill and task-resume pointer only; no new
executor. Dedicated worktree /root/pr191-bdh-poll on existing authorized
Box 9d374e18-196a-493d-bf1f-2d582589f04d. Preflight original
3293568f15af8705806fa040855440d7 exit0 preserved the d20 staged tree
8152454966480f2b68e1fd60e2715fb7e8d85177 in its separate worktree.
Verification original b7b0c92c360e0b539ae1ad8394c68128 exit0: 14 existing
deterministic Eval cases and format gate PASS. Each of these two originals
used one operation_get (wait_ms=20000) and two terminal output reads;
no premature output read. Workloads differ; no measured percentage or
causal efficiency gain. The 14 cases do not test Agent polling behavior.

## Learning disposition

Candidate, ready for ordinary-task observation, not promoted. Continue
P1 using the policy and record actual selected counts, timeouts and any
loss of evidence or progress. Reconsider only if repeated observations
show residual worth a small improvement. No separate BDH project or
new product MCP Tool. Independent review and exact-head CI remain
required. No merge, release, or P1 closure.
