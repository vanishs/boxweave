# v0.46.2 candidate — selftest atomic admission and CLI discovery

**Run:** ST-0462-20261105. Compose upstream #79 remains open and deferred. Combine selftest repair with prior CLI help change and publish a new patch version only after verification; keep stable PUBLISHED_RELEASE unchanged.

## Finder: installed official v0.46.1

The isolated retained Box bw-reaccept-1105-b runs Containarium 0.128.0 / Incus 7.5.1. Historical issue #83 observed conflicting immediate unreconciled and later passed receipts from the same selftest result file, with the child eventually absent. That ambiguous incident did not alone prove product failure.

Three sequential, uniquely planned host CLI selftests ran on unmodified installed v0.46.1, each collecting a separate stdout, stderr, exact confirmation, original exit and independent Incus inventory under the retained test lab folder work/selftest-repeat-20261105:

| Run | Original planned Box | CLI exit | Steps | Independently observed residue |
| --- | --- | --- | --- | --- |
| 1 | boxweave-selftest-154b85 | 0 / passed | create/hello/stop/delete all succeeded | absent |
| 2 | boxweave-selftest-3aaa57 | 0 / passed | create/hello/stop/delete all succeeded | absent |
| 3 | boxweave-selftest-d4b9b8 | 0 / passed | create/hello/stop/delete all succeeded | absent |

Serial path PASS does not guarantee concurrent-actor safety.

## Confirmed race and Fixer repro

Original selftest.run executed a separate Journal.unreconciled SELECT before Journal.start INSERT. Two agents could both see no running selftest and each dispatch a create. Two new isolated SQLite tests failed on original code: the second direct Journal.start was accepted despite a running row, and two simultaneous Journal.start invocations both returned accepted. These are true bug-finding negative controls with no Provider writes.

The candidate uses one SQLite BEGIN IMMEDIATE transaction to check existing running rows and persist the new run. Second concurrent caller receives SELFTEST_RUN_UNRECONCILED without dispatch. The redundant preflight SELECT in run() is removed, leaving the atomic Journal.start as the mandatory gate. Historical running rows still block new work and require separate exact operator reconciliation.

Post-fix both concurrency regressions and existing crash-durability tests passed. CLI #80 candidate already correctly advertises upstream, http, task, job, and selftest in global help without changing privileged dispatch. Full source/CI and installed candidate remain separate gates.

## Restrictions

Compose #79 is OPEN, WAITING UPSTREAM, with no workarounds in this patch. No owner host reboot, R-04 enablement, network/credentials/SSH changes, historical UNKNOWN recovery or stable pin movement. This document is candidate evidence and does not itself assert a published tag or installed-candidate acceptance.

**Intended version:** v0.46.2. **Stable selector stays:** 0.43.1.

## Final shipped and installed acceptance (2026-11-05)

**Published:** [GitHub Release v0.46.2](https://github.com/vanishs/boxweave/releases/tag/v0.46.2), draft=false, prerelease=false, 26 release assets. The immutable tag and release.json source both point to `b790a7f64d3b765f71b2f78d0a91c8fc69cef2f1`. [Main verify/publish workflow](https://github.com/vanishs/boxweave/actions/runs/42097937250) completed successfully. SHA256SUMS verified official install.sh (`9945075e0958c93aa91bc0acfd5de31ba8fe09dccce77bd26ccf774b5fd1bcd5`) and release.json (`3fd7e4e17339375302aa86832d8cd1e8362f95bb6f496b0c710c50434df619cf`). PUBLISHED_RELEASE remains 0.43.1; no stable selection change.

**Independent isolation, same-Agent sequential execution:** Created test-only outer Box `bw-st0462-verify-container`, original Provider identity UUID `6bcbf50d-3920-4aee-93cc-966f92d73a8a`, createdAt `1793908148`, backend dev. Nested mode was enabled with an explicit stop/start of **only this disposable Box**; no owner host restart or policy edit. Fetched official tagged v0.46.1 installer, verified SHA256 and performed exact fresh installation: exit 0, provenance source `2ec04fa1553e4c23a71b5a7fb4f2eab3b305a0b1`, installed 0.46.1 doctor 10/10 PASS, zero child residue. Then used the officially published, SHA256-verified v0.46.2 installer to upgrade the same Box: exit 0, installed 0.46.2 provenance source `b790a7f64d3b765f71b2f78d0a91c8fc69cef2f1`, doctor 10/10 PASS. Global help now actually lists the formerly hidden upstream/http/task/job/selftest command families.

**Installed adversarial concurrency:** In installed v0.46.2, launched a confirmed selftest for exact child `boxweave-selftest-ec3376`; Incus independently observed the original child while in flight. A **different** confirmed selftest for `boxweave-selftest-abd301` returned exit **2** with `SELFTEST_RUN_UNRECONCILED`. A second Incus inventory proved no competing child was created. The original selftest then returned **exit 0 / passed** with create, hello, stop and delete all successful, and independent child absence. After completion, a fresh selftest for `boxweave-selftest-46480a` returned **exit 0 / passed** with four successful steps and independent absence. Final installed acceptance runner exit **0**; final Incus inventory contained **zero** nested selftest children. All results were captured under test-owned `~/release0462-validation/installed-acceptance` in the isolated Box, with the summary copied to durable verification records. No attempted mutation was automatically retried on an UNKNOWN result.

**Additional concurrency oracle:** Five pairs of separately forked Python processes tested the same candidate SQLite journal; **5/5** admitted exactly one run and rejected the other, with one durable running row. This is source-candidate process evidence, not a substitute for the installed black-box denial above.

**Acceptance decision:** F10/X02 selftest lifecycle, verified **installed I PASS for this isolation and concurrency scenario**; X01 CLI help **installed I PASS**; no claim of full-feature BoxWeave acceptance or independent separate-Agent verification. The original #83 contradictory observer output was not reliably reproduced in serialized tests; the confirmed concurrent-admission defect is repaired and installed-verified. Compose #79 remains **OPEN, WAITING UPSTREAM**, and other historic issues are unaffected.
