# R1: retire proven historical workflow entry points

## Scope and source

PR #201, parent `7227c7c75d0a1c1e01358b7117da64baa06051f2`, based on published RC15 source `c0ad27a11fd1644e7e9755900c2c9b5421f84c9f`. Live ruleset 26842628 is active and still requires strict `quality-gate`. No ruleset, runtime, test script, Makefile, installer, release workflow, or current job/marker changes.

Local checkout was blocked by DNS. Created a small ordinary (non-nested, no autostart) RC15 development Box `73d2ecae-3932-46e6-b4c6-3feecf7fba15` (pr201-ci-audit). Create operation `796cc72aab6716561b0cef200481e651` succeeded. Git was initially absent (Exec `2429a2b8a0f40b0912481d382d8827f4`, exit 127); guest-only Git setup and exact checkout succeeded (Exec `af83a5ea4ec1e59571e3dbcb3e591785`, exit 0). Keep this development workspace for the next round.

## Per-workflow disposition

| Removed workflow | Lines | Original route and retained evidence |
| --- | ---: | --- |
| v1-s2-format-snapshot.yml | 29 | Temporary old S2 branch gofmt snapshot printer, not an acceptance gate. Current required quality uses `check_format.py` on every tracked Go file (including ignored probes), rejects differences and checks changed whitespace; old base64 snapshot output is historical tooling, not a product assertion. C01/C11. |
| v1-n4-capacity-budgets.yml | 50 | Old N4 branch only. Current required quality runs identical `TestN4ProjectCapacityAndNetworkAudit1024LocalBudget`, with the identical fixture environment, and both `TestN3NativeProject1024ObjectsAnd1025FailureNoPartial` / `TestN3NativeSnapshotFullNetworkLeaseRaceIsFailClosed`; exact source, formatting, diff and whole-repository vet remain. Old N4 echo labels are historical; no active consumers. C01/C04/C05. |
| v1-s2-list-boundary.yml | 38 | Old RC11 branch only. Current N2 `scoped-auth-races` uses original compiled response-boundary probe and shell on the installed fixed Scoped ledger. It requires distinct revoke/expire cuts, two original IDs, no leaks and further native/private-read/poll checks. The historical wrapper repeated the entire isolated diagnostic twice; that repeat-on-old-fixture is retained in Git and not a current contract, and we do not claim four cuts were rerun. The original CI shell is retained. C03/C05/C11. |
| v1-s25-bridgehead.yml | 92 | Two historical branches only. Current Scoped S2.5 PR and RC use the same `s25_bridgehead_lab.sh`, independent verifier and evidence checker. 30 of 38 literal old PASS markers directly named in the current checker; seven old maintenance/rollback markers are replaced by active/offline legacy refusal, byte-identical snapshots, exact original post-restart readback and explicit out-of-scope successful legacy rollback/upgrade. The remaining foreign packet/socket marker is still checked inside the same lab through the independent r12-foreign verifier; both-family positive/denied packet and socket checks, cleanup, and `P1B_S25_SCOPED_FOREIGN_PACKET` remain. No change to nested or release evidence. C03/C04/C06/C09/C11. |

The eight unmatched S25 markers were examined against actual current script branches, not removed on the basis of naming: legacy wrong-digest/unbacked refusal is replaced by stronger active/offline no-effect snapshots, successful old upgrade and rollback are obsolete in the current fresh-only contract, old offline fallback and original readback are preserved via the stopped-controller refusal and restart/readback path. The old formal signoff NOT_YET marker still appears in the scoped script; it is not acceptance.

Historical files remain retrievable at [RC15 exact source](https://github.com/vanishs/boxweave/tree/c0ad27a11fd1644e7e9755900c2c9b5421f84c9f/.github/workflows). No archive duplication in the active tree. RC15 and historical refs unchanged.

## Caller and evidence audit

- Full tracked-tree text scan of 1,436 paths (skip non-text/symlinks), Exec `edd1194fd47968c576a0f3ebe6c1b4a1`, exit 0, no truncated output. Three initial names, old labels and CI wrapper references were confined to their own producer, historic inventories/docs, or the original diagnostic script. Dynamic workflow consumer scan identified the current BDH CI aliases, S2 exact producer paths, release provenance and existing static tests; none requires a deleted route.
- All 38 non-core workflow names scanned against tracked non-doc source, Exec `1c18d4882b712bc1b740ecf7e869a39b`, exit 0. No external literal filename callers found; this is not a proof of behavioral equivalence. Existing historical S5 wildcard test requires nonempty historical files; no S5 file removed.
- List-boundary current compiled cut and exact gate check: Exec `0b7449b0c2306a32719f5068862cf493`, exit 0. S25 marker comparison Exec `abb0bdb1260b3f7bd97ef59ab2b9e4fa` printed 30 direct matches and eight unmatched; final marker grep returned exit 1 (BH018 old names absent from the selected current scripts), so BH018 retained. Reviewed scoped maintenance/foreign verifier branch directly.
- Prior exact RC15 current witnesses (historical evidence, not new R1 CI): [required quality / capacity](https://github.com/vanishs/boxweave/actions/runs/46124870136/job/140917150484), [N2 auth races](https://github.com/vanishs/boxweave/actions/runs/46124999390/job/140917531225), [release S2.5](https://github.com/vanishs/boxweave/actions/runs/46124999468/job/140917530982), all success.

## Focused verification

Box Exec `4cee7ed5754dae1a15b932a369ea6a73`: exit 0, complete stdout/stderr. Staged only the four YAML deletions; git diff --cached --check passed. Verified 46 -> 42 workflow files, all eight current and two optional workflow blobs byte-identical, scripts/runtime unchanged.

| Existing regression suite | Tests | Result |
| --- | ---: | --- |
| test_ci_change_scope.py | 13 | PASS |
| test_local_hook_scope.py | 8 | PASS |
| test_pr191_s2_witness_map.py | 15 | PASS |
| test_pr191_s2_ci_gate.py | 12 | PASS |
| test_pr191_s25_evidence_check.py | 4 | PASS |
| test_v1_stable_gate.py | 14 | PASS |
| test_v1_release_manifest.py | 22 | PASS |
| Total | 88 | PASS |

No native scenario rerun locally. Full Go/race and dual-arch CI remain required. This workflow-only diff may not match N2/S25 native PR path filters; that is the existing policy, not proof of new native execution. No change to those filters or their crosswalk.

## Remaining disposition

R1 first slice: 4 historical wrappers / 209 lines removed; 42 workflows remain (32 historical, eight core, two optional). 213 script files and 296 Go test files unchanged. No current PR runtime saving claimed, because the removed workflows were historical-ref-only.

Retain the remaining 32 historical wrappers for their unresolved assertion-level applicability, grouped by next review:

- S25 BH018 installed/native and native prerequisite (3): 126/127, WAL kill, kernel prerequisites and independent dual-controller ACL semantics need explicit Scoped equivalents or contract retirement.
- S3 installed/native (2): SFTP exclusive-create race, hostile file types, bounded list and independent guest evidence need mapping.
- S4 A and B1 (2): installed raw terminal evidence, OAuth and packet isolation.
- S4 B2 (11): bridge ownership, dedicated/controller-root, fanout/grant faults, basic native, NDP/quiescent, Scriptlet/parity and static address lanes. Exact old names in R0; shared-ACL/Scriptlet positive mechanisms are obsolete, generic ownership, no-effect, packet and fault obligations are not.
- S4 B3 (6) and B4 (1): contention, ACK loss, foreign established flows, directional grant/revoke, ETag, Forward; require exact behavioral mapping before removal.
- S5 A/B historical (7): frozen product source, retained operator/CLI/recovery and independent evidence; existing nonempty-history contract test also needs review.
- Optional A071 and UNKNOWN Create (2) remain manually dispatchable; not candidates for historical-only deletion.

R0 full source caller gap resolved for workflow names, not all script transitive dependencies or all assertions. Remaining groups are candidates for R1 follow-up or R3 assertion consolidation, not accepted obsolete routes. No new release or merge. Rollback: revert this single change on #201 if applicable.
