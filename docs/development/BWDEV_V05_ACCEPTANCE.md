# bwdev v0.5 delivery and self-evolution evidence

PR #189; work_ref boxweave/pr189-bwdev; attempt bwdev-v05.
Starting exact v0.4 head 5ecf1e8bb655e1d6d66a5ec7566bbee14cd54407,
v1 base a38ef26e047f64bb09490be35d9aa39048b320ac.
Reuse owned Box 9d374e18-196a-493d-bf1f-2d582589f04d, isolated checkout
/tmp/bdh-pr189-v01. Original resume 3c3d957b423919230c4969190e794719
confirmed clean starting HEAD and Python/Git; no Go, Make or gh on PATH.
No product runtime, other worktree, installed controller or host change.

## Prior CI and risk

v0.4 exact-head quality/source/distribution/installed runs
45550962075, 45550962222, 45550962192, 45550962105 all SUCCESS.
v0.2 source/distribution/installed passed; quality was superseded.
v0.3 source/distribution passed; quality superseded. Legacy installed
job 138990476280 (run 45550562080) failed on reviewed recovery apply
after guided decline, with state.ErrConflict's error message. Three other
lanes passed. The same installed gate passed at v0.4. The generic
ErrConflict is also returned on changed review/ledger/native evidence;
the log alone does not establish the specific cause. No new native
mutation, retry, or weakened guard to conceal this intermittent prior
failure. If it recurs, reproduce in a clean authorized fixture and retain
the exact reviewed plan, original operation, evidence and poststate.
Independent final-head installed CI remains mandatory.

## Functional acceptance

| Original Operation | Observed result |
| --- | --- |
| 771fdeb71579982da4f275a5de552e7d | First suite 35 tests and four Eval cases, start plan exit 0 |
| 9d48121f3e86ca73748ef8cbce21f1ee | 39 tests/4 cases and actual CI snapshot worked; wrapper exit 5 because Python 3.12.3 already refused zero discovery |
| 616764fdbba335c30245927811d103b6 | Explicit runner zero-case exit 2, built-in exit 5; registered checks and retained-log witness, wrapper exit 0 |
| c7cd287388928addb723188c66d0708c | 40 tests, four Eval cases, actual v0.4 CI snapshot with one old cancellation excluded, exit 0 |
| 0560abc69b63864a68c161772af70d85 | BDH-L012 revision 2 attached; relevant candidate surfaced by start; 40 tests passed |
| 8ac9c9c850bf898ef7036fc332b61fdf | BDH-L013 captured/attached, both audited, 40 tests and four Eval cases passed |
| c5b0424b0891eef19b797a7f8492f45e | Actual task-history creation and retrospective progress, checkpoint, finish/export correctly REVIEW_REQUIRED; wrapper exit 0 |

## Self-evolution witness

BDH-L012 originated from actual superseded CI in v0.4. It now has
a CI read adapter, updated verification Skill, exact-head/PR/base and
cancelled/missing/ambiguous/pending regression coverage, a deterministic
Eval case and linked implementation evidence in its immutable revision 2.
BDH-L013 records nonempty discovery and the observed Python-specific
exit status, not a universal old-runner defect. Start discovers related
candidates; learn next distinguishes already attached implementation;
finish requires a concrete learning disposition. Both remain CANDIDATE
until applicable review, not automatically promoted. Future source
or asset drift is flagged for review.

## Handoff and limits

Original task history /tmp/bdh-pr189-v05-task-history, checkpoint
/tmp/bdh-pr189-v05-handoff.json, export /tmp/bdh-pr189-v05-handoff-export.json
and prior CI snapshot /tmp/bdh-pr189-v05-prior-ci.json retained in the Box.
Task revision 7 has ci ready, not accepted. Export and PR are advisory;
reconcile source and original operations before continuing. Persist
approved summaries outside the Box. Existing Markdown workflow remains
available without the CLI. No automatic scheduling, distributed claim,
promotion or permission. The gh live adapter requires separately
authorized gh; only the real connector snapshot path was exercised here.
Full Go, dual architecture, installed acceptance and independent review
must validate the final exact head. Four Eval cases are deterministic
behavior fixtures, not a comparative Agent productivity study.

Final offline gate: original operation 0b209eca6e896e651dbfa79c7630a6e1, exit 0; 40 bwdev tests and 109 existing tests (16 MCP testkit, 57 uninstall core, 10 stable gate, 7 raw operation witness, 6 recovery fixture, 13 scope), four Eval cases, learning audit and diff check. All streams read through terminal page. This is offline source/fixture evidence, not installed/native acceptance.
