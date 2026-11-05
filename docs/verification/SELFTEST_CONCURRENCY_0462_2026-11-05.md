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
