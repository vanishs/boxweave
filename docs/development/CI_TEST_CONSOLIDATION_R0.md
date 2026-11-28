# R0: exact RC15 CI consolidation audit

Source `c0ad27a11fd1644e7e9755900c2c9b5421f84c9f`; PR #201 starts at this source. All 46 workflow texts were read at that immutable ref. The complete, non-truncated Git tree contains 213 files under scripts (2,388,092 bytes), 54 script test-named files and 296 repository-wide Go test files. These are file inventory counts, not case counts. Live ruleset 26842628 readback: active, strict required `quality-gate` from integration 15368, PR, no force-push, no deletion, thread resolution. Do not change protection.

## 1. Workflow route inventory

- CURRENT: 8 files, 1065 lines.
- HISTORICAL: 36 files, 9114 lines.
- OPTIONAL: 2 files, 62 lines.
- Total: 10241 workflow lines (including comments and blanks).
- Historical-only classification accounts for 89.0% of workflow lines and 36 files; this is a conditional cleanup opportunity, NOT accepted deletion or runtime savings.

CURRENT = current PR, RC or stable workflow; preserve interface until migrated and verified. HISTORICAL = trigger and/or job guard confined to named historical branches; not automatically redundant assertions. OPTIONAL = historical push plus manually dispatchable on current refs; retain pending unique coverage review. Counts are source text newline counts. P4's 47-workflow ledger is historical: stable candidate was folded into `v1-stable-release.yml`, which now has manual source/build/package and separately conditional publish jobs. The current S5 entries are dispatch-only with old-ref guards, rather than the old ledger's push+dispatch classification.

| Workflow under .github/workflows | Lines | Route |
| --- | ---: | --- |
| `quality-gate.yml` | 114 | CURRENT |
| `v1-forward-install-policy.yml` | 65 | CURRENT |
| `v1-n2-installed-lifecycle.yml` | 123 | CURRENT |
| `v1-pr191-s25-scoped.yml` | 83 | CURRENT |
| `v1-release.yml` | 358 | CURRENT |
| `v1-s1-check.yml` | 65 | CURRENT |
| `v1-s2-native.yml` | 84 | CURRENT |
| `v1-stable-release.yml` | 173 | CURRENT |
| `v1-n4-capacity-budgets.yml` | 50 | HISTORICAL |
| `v1-s2-format-snapshot.yml` | 29 | HISTORICAL |
| `v1-s2-list-boundary.yml` | 38 | HISTORICAL |
| `v1-s25-bh018-installed-candidate.yml` | 77 | HISTORICAL |
| `v1-s25-bh018-native.yml` | 135 | HISTORICAL |
| `v1-s25-bridgehead.yml` | 92 | HISTORICAL |
| `v1-s25-native.yml` | 42 | HISTORICAL |
| `v1-s3-file-installed.yml` | 80 | HISTORICAL |
| `v1-s3-file-native.yml` | 138 | HISTORICAL |
| `v1-s4-a-installed.yml` | 136 | HISTORICAL |
| `v1-s4-b1-native.yml` | 328 | HISTORICAL |
| `v1-s4-b2-bridge-ownership-native.yml` | 101 | HISTORICAL |
| `v1-s4-b2-controller-root-installed.yml` | 475 | HISTORICAL |
| `v1-s4-b2-dedicated-native.yml` | 174 | HISTORICAL |
| `v1-s4-b2-fanout-fault-native.yml` | 164 | HISTORICAL |
| `v1-s4-b2-grant-fault-native.yml` | 147 | HISTORICAL |
| `v1-s4-b2-native.yml` | 292 | HISTORICAL |
| `v1-s4-b2-ndp-native.yml` | 169 | HISTORICAL |
| `v1-s4-b2-quiescent-native.yml` | 645 | HISTORICAL |
| `v1-s4-b2-scriptlet-native.yml` | 184 | HISTORICAL |
| `v1-s4-b2-scriptlet-parity-native.yml` | 410 | HISTORICAL |
| `v1-s4-b2-static-address-native.yml` | 109 | HISTORICAL |
| `v1-s4-b3-contention-upgrade-native.yml` | 684 | HISTORICAL |
| `v1-s4-b3-final-closure-native.yml` | 733 | HISTORICAL |
| `v1-s4-b3-final-gates-native.yml` | 782 | HISTORICAL |
| `v1-s4-b3-foreign-revoke-native.yml` | 785 | HISTORICAL |
| `v1-s4-b3-grant-native.yml` | 727 | HISTORICAL |
| `v1-s4-b3-stale-etag-native.yml` | 640 | HISTORICAL |
| `v1-s4-b4-forward-native.yml` | 230 | HISTORICAL |
| `v1-s5a1-installed.yml` | 52 | HISTORICAL |
| `v1-s5a2-installed.yml` | 40 | HISTORICAL |
| `v1-s5b-integrated-native.yml` | 244 | HISTORICAL |
| `v1-s5b-peer-production.yml` | 39 | HISTORICAL |
| `v1-s5b-rc10-residual-native.yml` | 40 | HISTORICAL |
| `v1-s5b-rc10-specialized-recovery.yml` | 35 | HISTORICAL |
| `v1-s5b1-frozen-installed.yml` | 68 | HISTORICAL |
| `v1-s1-a071-block-disk.yml` | 25 | OPTIONAL |
| `v1-unknown-create-repair.yml` | 37 | OPTIONAL |

Historical source remains available at the [RC15 workflow tree](https://github.com/vanishs/boxweave/tree/c0ad27a11fd1644e7e9755900c2c9b5421f84c9f/.github/workflows). R1 must check actual live consumers and unique assertions before deleting any file. Historical refs remain untouched; do not copy all files into an active archive directory. Old workflow references in dated docs are provenance, not reasons to restore retired execution.

## 2. Measured runner time

Comparable source-changing PR #198 head `8f5938250177102a65f79fc3079bb31cb5d1a9d1`, attempt 1; all listed workflows concluded success. Sum of non-skipped job completed_at minus started_at from GitHub Actions REST jobs, not wall-clock duration or billable minutes. ARM billing multipliers and queue time excluded.

| Event / workflow | Run | Executed jobs | Runner seconds |
| --- | --- | ---: | ---: |
| PR quality | [46123873972](https://github.com/vanishs/boxweave/actions/runs/46123873972) | 1 | 193 |
| PR S1 | [46123873970](https://github.com/vanishs/boxweave/actions/runs/46123873970) | 1 | 252 |
| PR Forward | [46123873943](https://github.com/vanishs/boxweave/actions/runs/46123873943) | 3 | 323 |
| PR candidate | [46123874028](https://github.com/vanishs/boxweave/actions/runs/46123874028) | 5 | 481 |
| PR S2.5 | [46123873964](https://github.com/vanishs/boxweave/actions/runs/46123873964) | 1 | 463 |
| PR N2 | [46123873999](https://github.com/vanishs/boxweave/actions/runs/46123873999) | 29 | 6357 |
| PR S2 consumer | [46123874011](https://github.com/vanishs/boxweave/actions/runs/46123874011) | 1 | 613 |
| Same SHA feature push quality | [46123868863](https://github.com/vanishs/boxweave/actions/runs/46123868863) | 1 | 202 |
| Same SHA feature push candidate | [46123868828](https://github.com/vanishs/boxweave/actions/runs/46123868828) | 5 | 480 |
| Total | | 47 | 9364 |

PR-only: 41 executed jobs, 144.70 runner-minutes. N2: 105.95 minutes (73.2% of PR-only time), with 29 independent lanes. Feature push duplicates: 11.37 runner-minutes on this SHA, 7.3% of combined total, an upper bound on possible event deduplication benefit rather than achieved savings. PR and push scope bases differ; the PR quality context is mandatory. The S2 consumer spent 598/613 seconds in exact-provenance crosswalk step (including waiting, not another full Incus scenario). Keep independent receipts; consider replacing expensive waiting with verified orchestration only if it preserves the same proof and negative checks.

RC15 exact release source, attempt 1: N2 [46124999390](https://github.com/vanishs/boxweave/actions/runs/46124999390), 29 jobs, 6473 seconds; release [46124999468](https://github.com/vanishs/boxweave/actions/runs/46124999468), 8 jobs, 933 seconds. Total 37 jobs / 123.43 runner-minutes. Preserve this RC baseline and exact amd64 identity. One sample each, not a longitudinal estimate.

## 3. Active consumer and assertion boundaries

- quality-gate: exact HEAD, docs classifier, scope tests, repository format/YAML/whitespace/shell syntax, full Python discovery plus OAuth script, tagged install, explicit capacity witnesses, module verification, vet, race and build. Makefile verify has a narrower explicit Python selection and omits some of the quality workflow's special steps. Do not simply replace the latter with make verify.
- S1: Go build/race overlap with quality and candidate; unique no-CGO negative build, actual CLI/SQLite and selected protocol checks must be mapped before consolidation. Keep source identity and tags.
- Candidate: source make verify plus two ignored Go probe compiles, marker/shell/manifest checks; amd64 and arm64 each build and execute actual CGO/Schema 7 binary and run full Go suite. On PR and feature push this overlaps source checks; release needs exact native independent artifacts and manifest verification. Stable workflow independently builds candidate and gates conditional publication.
- N2: 29 fresh isolated jobs, `n2_installed_lifecycle.sh` (628 lines) -> lane-specific probes -> `pr191_n2_installed_gate.sh` (265 lines) -> `pr191_n2_evidence_identity.py` and crosswalk receipts. Preserve lifecycle, retirement, three offline, four ACK, six Create/Delete/Exec WAL, two Update WAL, Exec crash, SDK compat, S2 native, retention, transport, physical quota, auth races, noquota, malformed Exec, stop ACK, overlap, alternate pool. Seven non-S2 mapped lanes still require their own checks and release identity receipts.
- S2/S2.5: current PR S2 workflow is a consumer, not an independent repeat of all historical native scenarios. `pr191_s2_ci_gate.py` (399 lines) binds exact workflow paths, event/ref, run/attempt/job, producer artifact and candidate identity. `pr191_s2_witness_map.json` has 64 mapping rows (old marker groups; do not confuse with original marker count); `pr191_s2_witness_map.py` (254 lines) validates mapping and receipts. N2 and S25 producer paths and job names are hard-coded in consumer checks; renames require simultaneous schema/test updates. `s25_bridgehead_lab.sh` (775 lines) is a shared current entry and contains historical branches; do not retire it because its name is old.
- Forward matrix: fresh, legacy-unset and legacy-exact job labels. Names are not evidence that the assertions are obsolete; examine the current preservation/refusal behavior before retirement.
- Optional manual A071: real Incus dir pool on dedicated virtual block device via `accept_s1_a071_block_disk.sh`. No job ref guard; retain pending comparison to physical-quota/alternate-pool witnesses. Optional UNKNOWN Create: repository guard only, runs focused race and `accept_s2_native.sh`; preserve original recovery and no-replay evidence pending scoped applicability review. Neither is historical-only by trigger.
- Script overlap: N2 and S25 independently implement pinned Incus package install, candidate build and shared APT source helper `pr191_ci_apt_sources.py`. Retain isolated runner and environment-specific preflight; share only proven identical preparation. N2 uses many S2-era probes, including ack loss, SQLite fsync, guest and MCP quota, old SDK, overlap and Exec GC; old names cannot justify deletion. S25 invokes S3 file and S5 closeout helpers conditionally; inspect both paths before extracting obsolete modes. Current RC consumer compiles `s2_unknown_create_binding_probe.go` and `s2_p1_bound_create_receipt_fault_probe.go` directly.
- Offline Python: full discovery in quality and targeted invocations in Makefile and native lanes; `dev/harness/check_runner.py` verifies nonempty discovery. Compare test sets and special environment variables before deduplicating. Do not remove unique tests from discovery merely to reduce count.

## 4. Recommended smallest delivery order

1. R1: start with historical wrapper-only candidates (S2 format snapshot/list boundary, N4 capacity, S25 old wrapper family). Establish exact live callers, per-assertion current witnesses or obsolete product contract; where no proof exists, retain and record blocked. Then review the much larger S4/S5 wrapper families by behavioral contract. These old-ref routes are the highest file/line reduction opportunity, but generally not current PR runner savings. Preserve scripts until all callers are mapped.
2. R2: consolidate common CI setup, compile helpers, matrix lane routing and portable nonempty Python checks. Use existing helpers; no new framework. Compare hashes and env/fixture differences. Keep separate native job isolation.
3. R3: remove only obsolete positive legacy behavior and verified duplicate assertions; preserve architecture-independent isolation, native fault, durability, authorization, retirement and quota oracles. Reuse existing P0 inventory and 64-row S2 crosswalk instead of re-documenting every retained test.
4. R4: prioritize same-head feature-push/PR event redundancy and overlapping full Go/Python source checks. Validate docs-only, mixed, new branch, PR open/update, protected v1 push, manual and RC release cases. Preserve required PR check and release proof; no assumption that matching SHA alone implies matching scope. N2 dominates minutes; optimize setup and evidence transport before considering lane combination. Never collapse independent fault/fixture jobs without positive and negative equivalence.
5. R5: compare exact same event families before/after with actual job seconds, active workflow/script counts, LOC, and required evidence. No numerical acceptance target until unique coverage review. Keep rollback as normal revert of this PR, not changes to old release tags.

## 5. R0 status and limitations

Workflow trigger inventory and measured baseline complete; prioritization and principal producer/consumer chain documented. Detailed per-assertion and transitive script caller inventory is not complete: the local environment cannot resolve github.com for a Git checkout, and the connector limits per-call fetch count. No claim of complete repository-wide reference scan or assertion equivalence. Before each R1 deletion, run a full-tree caller/marker search (including dynamic entry points, hooks, build tags, current acceptance docs and crosswalks) in an authorized checkout, and keep unresolved cases. No actual CI/test implementation has changed. No test was newly executed in R0; historical CI metrics are observations, not new test results. No Host or release change. R0 is partially complete against the original exhaustive exit criteria; the wrapper investigation can proceed in R1 only with the above checks, not blanket deletion.
