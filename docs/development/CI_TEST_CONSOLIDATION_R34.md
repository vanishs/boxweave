# R3/R4 parallel checkpoint

Parent 6cf33939ff8c84c05b1469c5582d091a71037c8c. Candidate 20c90d4499a4864c7bf31fd47753edfe65651462 remains staged, not the PR head.

| N2 run | Successful lanes | Aggregate job seconds | First job delay (s) | End-to-end seconds |
| --- | ---: | ---: | ---: | ---: |
| R2 46130476383 | 29/29 | 6308 | 251 | 1107 |
| R3A 46130816381 | 29/29 | 6370 | 689 | 1526 |
| R3B 46131230741 | 29/29 | 6262 | 1062 | 1819 |

R3A SDK 140935324500 and WAL before/after 140935324550/140935324448 passed. R3B File 140936596838 passed. R3C OAuth SDK 140938317278 passed; N2 46131781056 had 20 success and 9 running. Latest N2 46132116083 had 7 running, 22 queued. S2 consumers 46131230795 and 46132116085 still running. Exact revisions only; no latest-head signoff. Preserve remaining direct SFTP/Git, raw event, repeat, recovery and Peer/Forward obligations. BH018 installed wrapper retirement remains in candidate.

Observed N2 effort stays near 104-106 runner-minutes while wall time rises from 18.45 to 30.32 minutes; not billing or causal savings. S2 waits 35 minutes from gate start. Whole-workflow serialization is held. Revised candidate coalesces pending same-repository first-attempt PR jobs per matrix.mode, never cancels running jobs; independent release/manual/fork/rerun routes remain. S1 enables Go cache with tests still -count=1. Actual scheduling and cache gain unmeasured.

Push/PR dedup deferred: QUALITY_GATES.md explicitly retains feature push validation. Previous-SHA push versus whole-PR merge-base classification differs; open PR alone cannot prove equivalent coverage. Candidate make verify has unique distribution checks; S1 uses merge-ref. No removal of these routes. Prior 762-second parallel push sample is cost, not proven avoidable work. Prefer no extra polling/API subsystem.

77 focused tests, YAML 28 files, format/exact-delta passed in operation 48fa53869daeaeddc28b6e2861e52f89; export 99ac0f8c2f92dccd630d6f7821537e74 matched tree 87d1cedb4ede2fe9c4966ce0625d6d4c7ebeb73e. Consumer audit 86008384ff6e95d3738757e2e6f8493c exit 0. Candidate 28 workflows/7328 lines, current PR 29/7400; 215 script files retained. No new CI, merge, release, host or protection change. Promote only after checking actual latest N2/S2 feedback; fix concrete failures first.
