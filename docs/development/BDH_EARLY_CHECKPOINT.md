# BDH early checkpoint candidate (PR #191)

Owner-authorized narrow refinement, 2026-11-26. Base
9e9691043baa5d3842a71cd4a6853f30b5aef646. Previous P1-06d20 publication
was interrupted by actual tool-call exhaustion after staged verification;
the next task recovered exact staged tree 8152454966480f2b68e1fd60e2715fb7e8d85177
and published d46d5b9 without repeating the 75 tests. See
BDH_OPERATION_POLLING_CHECKPOINT.md and CONSOLIDATION_P1_06D20_CHECKPOINT.md.
Quota and whole-task count are unavailable. No fixed platform limit inferred.

Refine existing task-resume Skill and checkpoint template. Advisory
freshness review after about 20 observed nested tool attempts since durable
save, or before expensive publication / repeated failures. Earlier save
only for new material state; do not interrupt productive work for unchanged
checkpoints. No new runtime, scheduler, command, or data schema. No
additional metering calls. PR record is not a backup of dirty source bytes.

Box preflight 5b927813fea36097e61dfa7700a23f05 exit0: synchronized original
P1 worktree to verified head, isolated /root/pr191-bdh-budget. Original
2ef6b8b7907ef323098661a1f737013c exit0: 17 existing checkpoint/workflow
tests, 14 Eval fixtures, format PASS. Both one bounded status read.
Existing tests verify checkpoint mechanics, not model budget behavior.
Candidate, not promoted. Observe actual PR checkpoint persistence before
next P1 export; verify later recovery separately. L014 review remains open.
No merge, release, P1 closure or additional BDH development authorized.
