# R3 bounded closeout and R4 candidate handoff

Exact accepted source 6cf33939ff8c84c05b1469c5582d091a71037c8c. All nine observed PR/push workflows succeeded. N2 46132116083: 29/29; S2 46132116085: exact crosswalk PASS in job 140939347698; S25 46132116105, Forward 46132116099, S1 46132116135, quality 46132116195, candidate 46132116116 all successful. Push quality 46132111985 and candidate 46132112018 successful. Read original SDK job 140939348163 and File job 140939348291 logs: 126/127 original receipt/restart readback, OAuth refresh/revoke/one-effect and fixed ledger, File independent Guest readback and fixed ledger/native absence all PASS. Earlier exact R3A/B/C N2 runs 46130816381/46131230741/46131781056 each 29/29.

## Disposition

Retire BH018 installed-candidate wrapper (77 lines), old exact source/blob recorded in parallel checkpoint and PR comment 7237631881. R3D retired obsolete provisional shared-ACL diagnostic. Remaining 18 historical wrappers are intentionally retained, not claimed equivalent:
- Exec (2): BH018 direct native provider matrix/BH019 and S5A1 repeat/raw diagnostics.
- File/Git (2): direct SFTP 12-writer and wire budget, phase3 Git HEAD/original and stale writer.
- Identity/recovery (2): S4 raw event timing and remaining S5A2 power/update/archive composition; obsolete local CLI diagnose/reconcile not restored.
- Peer/network (10): per-wrapper HOLD mapping in R3D report, especially composed internet=false traffic, foreign flows, native ACL ETag, kernel/ledger interleaving and linked Delete.
- Forward/lifecycle (2): actual collision/removal, dependent cleanup and combined pending Exec restart.

Retain all scripts and independent native/current release gates. These are explicit hold decisions, not new required migration tasks or waived coverage. Bounded R3 implementation/disposition complete; final candidate CI still required. Avoid expanding tests solely to increase retirement count.

## Candidate and R4 handoff

Candidate code tree adac39b7694c0581efc7bdea3d4d7880b6e91046 passed full Python discovery (495 tests in 63.165s), 28-file YAML and diff checks in Box operation 12b28a50f1269264c0d3cca5dc606f5f exit 0. SDK/File/OAuth scenario code unchanged from accepted 6cf3393. R4 per-lane pending coalescing and S1 cache included from previously requested parallel work. Stale PR association mismatch diagnostic fails closed, not evidence substitution. Prior R3B S2 timeout documented in PR comment 7237796577; current exact S2 succeeded. No need to rerun superseded consumers.

28 workflows / 7328 lines versus RC15 46 / 10241; 18 historical, 8 core and 2 optional. Scripts 215 versus 213. No measured runner-minute saving yet. R4 scheduling and cache effects require actual CI; no claim that current proof transfers to a different head. Push/PR independent classification retained. No product change, protection change, host mutation, merge or release.
