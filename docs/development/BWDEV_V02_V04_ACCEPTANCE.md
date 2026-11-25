# bwdev v0.2-v0.4 incremental delivery

PR #189; work_ref boxweave/pr189-bwdev. Reuse isolated clone
/tmp/bdh-pr189-v01 in owned Box 9d374e18-196a-493d-bf1f-2d582589f04d.
No changes to other worktrees, installed controller, or product security.

## v0.2

Start at 8dcea8890e2d0dd747b0150433b74026cda8f99e; original staged
tree exactly matches remote v0.1. Resume/sync operations recorded in
Learning Inbox. v0.1 source, quality and distribution CI succeeded;
installed lifecycle 45549596239 was still running at start.

Initial v0.2 focused tests plus CLI dependency walkthrough:
511aba1def5504002ca38de6820678bd, exit 0, 12 tests.
The walkthrough uses the example task specification, not an independently
executed multi-Agent workflow. Separate-process revision contention is
covered by the final focused regression. A task RECORDED state is only
a caller assertion. No automatic acceptance, cross-host lock or replay.

Final v0.2 focused regression and staged format: d6961083db13fc668a9a3269da1526f5,
13 tests and 10 changed files, exit 0. Exact submitted v0.2 commit
39e4c49329170e088ad7b3c405463eda8e8caf0e, tree
95964590ff7af55791070e929f077680d5d3e9dc. All CI runs started;
v0.1 installed lifecycle still pending when v0.3 development began.

## v0.3

Fast-forwarded only after exact tree equality and clean unstaged state.
Operation e7db528044c0ccd3c5f0dc75226635e9.
Initial regression, actual verify plan/run/inspect and missing-gofmt
negative check: efc3d3a720706260d9a48ee95f9e3957, exit 0, 18 tests.
The real PR required full scope; local dirty-tree PASS was UNBOUND,
and the format gate was BLOCKED. A local output digest is not independent
CI evidence. Full Go and native tests remain assigned to CI.

Final v0.3 regression and staged format: 0cb6488bd4913d7d8b41b10aec76c351,
21 tests and seven changed paths, exit 0. Exact commit
c85da136280d268e2548b06bde7c07a5a35d3d40, tree
7899073c9b920e1bbcf8afaa5cd234128134ef27. Clean-head dogfooding
0b89afff734085a0534848a328069714 ran verify run bwdev and verify inspect
on that exact commit, exit 0, CLEAN_HEAD_OBSERVED and LOCAL_RECORD_MATCH.

## v0.4

At v0.4 start, v0.1 CI all four runs succeeded (quality 45549596188,
source 45549596184, distribution 45549596149, installed 45549596239).
Earlier Legacy fixture issue passed the new independent installed run.
v0.2 quality run 45550362915 was cancelled after a newer head; source
45550362949 succeeded. Remaining v0.2 and v0.3 runs were pending;
do not treat superseded checks as final-head acceptance.

Initial tests and live candidate capture/audit: 3e071e56e76d684500fa0c62cf553a62,
26 tests, exit 0. BDH-L012 is persisted in dev/learning; no promotion.
Combined regression and actual CLI checks: e872c6152d21b222d142f22597045d0a,
28 tests plus scope and recovery-fixture registered checks, exit 0.
Dirty worktree results remained UNBOUND; learning audit no findings.
Early output retrieval was unavailable while the original operation ran;
re-read the same operation and output, no replay.

Actual delivery-task history and checkpoint:
fe3d19735fe3be5b915e226bdb853c21, exit 0. Local directory
/tmp/bdh-pr189-v04-task-history and /tmp/bdh-pr189-v04-handoff.json
are retained in the same owned Box. Work_ref boxweave/pr189-bwdev;
prior stages recorded with evidence, v0.4 working, final CI not accepted.
The task history is an advisory local record, not independent approval.
The PR, staged-tree evidence, original operations and this document
are the durable handoff. Verify actual remote head before further effects.

Observed value: the harness found missing toolchain capability, surfaced
relevant assets, preserved revision conflicts, bound clean-source local
results and captured a real learning candidate during its own development.
No controlled baseline, human/Agent efficacy or distributed coordination
proof is claimed. Full CI must validate the final exact head; no merge or
release authorization assumed. No Go or product runtime changes in these
increments. Existing installed/native gates remain authoritative.
