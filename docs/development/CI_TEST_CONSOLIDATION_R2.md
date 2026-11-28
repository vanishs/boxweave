# R2: shared candidate build and SQLite smoke

Parent 041c70f42fa5ecf475d8795bb5aa2c0e0ad10846, PR #201. Reused development Box 73d2ecae-3932-46e6-b4c6-3feecf7fba15. Reviewed N2, S25, RC, stable, Forward workflow setup and existing shared helpers. R2 first coherent slice targets duplicate candidate build and smoke code.

## Implementation

RC and stable both call scripts/ci_candidate_build.py. Exact native target, checked-out source, CGO, version and commit, reproducible compiler flags, actual binary build-info, private 0700 SQLite directory and read-only database validation remain required. Schema 7, required tables, retired table exclusion, empty operations and not-ready status are unchanged; explicit failures replace assertion-only checks in the shared helper. Both source jobs run new offline calibration; both native build matrices still independently run all Go tests and hash their own artifacts. No change to jobs, events, stable authorization, publication, N2/S25, native receipts or release provenance. Stable static regression now checks the shared oracle and its invocation before Go tests rather than requiring inline SQL text. Other assertions remain.

The candidate workflows lose 49 lines (358 -> 331, 173 -> 151). New shared helper 70 lines, calibration tests 87 lines, adjusted stable test +5 net. Net source/test/workflow LOC +113 in this slice: maintenance benefit is one build/smoke implementation, not total LOC reduction. Two new script-directory files (213 -> 215). Workflow count 30 unchanged, workflow lines 7736 -> 7687. No runtime savings claimed.

## Actual verification

- Existing isolated Box had no Go compiler. No native build or stable dispatch performed there.
- Two initial oversized Exec input requests were rejected at admission; scoped Operation list and clean-tree precondition checked before smaller exact writes. No uncertain mutation replay.
- Initial focused run dea0a79c285e4dc122de4e8b1cbef50e exited 1: stable static test still required SQL literal in YAML. Updated reference without dropping its SQL checks.
- Exact rerun 9c9d474ddebb74be96ed76c48548ee38 exited 0: 88 existing focused tests plus 4 new calibration tests PASS. The new tests use mocked commands and real temporary SQLite fixtures, including wrong source, architecture, CGO, version/source inputs, identity, status, schema, missing/retired table and nonempty operations. They are not a real binary or installed native PASS.
- Exact workflow delta assertions passed: only inline build/smoke extraction and explicit test call changed. Five-file format and diff checks passed. Staged code-only tree 98995363967afe83a14ba368dab17d7b1504f486. Export operation b0b4053fb17995bd2d4446670c003b08 completed exit 0.

## Remaining R2 decisions

Existing S25 transcript verifier and APT source selection already shared. N2/S25 package preparation differs in sequence, package lists, source-file format and distinct bootstrap/guard work. Do not unify privileged preparation or change native fixture behavior merely to reduce lines. Further audit can isolate identical read-only package/identity operations and reusable setup steps, retaining distinct prerequisites and exact runner effects. R4 handles repeated event/source execution. R3 coverage queue unchanged. R2 partial pending further helper review and exact-head CI. No stable dispatch, merge, publication or host mutation.
