# Current source and release status

Observed 2026-11-27 PT (2026-11-28 UTC), at v1
[b6f7ae0](https://github.com/vanishs/boxweave/commit/b6f7ae09edb927d204fbcaaf9b2ecd6f98e7f211).
Source integration is not release acceptance.

| Identity | Observed status |
| --- | --- |
| Stable v1 | UNPUBLISHED; [selector](../V1_PUBLISHED_RELEASE). |
| Published prerelease (metadata verified) | [RC14](https://github.com/vanishs/boxweave/releases/tag/v1.0.0-rc.14), tag commit [3fb71a6](https://github.com/vanishs/boxweave/commit/3fb71a6ddf47d13d165e5c039144aa66fad0eb05), schema 6. GitHub release metadata confirms six assets and reported digests; bytes were not independently verified. |
| Merged v1 | Schema 7, Scoped v2 only; [#191](https://github.com/vanishs/boxweave/pull/191) merged at [b842dc0](https://github.com/vanishs/boxweave/commit/b842dc0830ee8bc40a76aca9d4cfa8d440455f95). Only BOOTSTRAP.md changed between that merge and the observed HEAD. |
| Final #191 PR source | [cb67f99](https://github.com/vanishs/boxweave/commit/cb67f99c5bf320afba3b1e1211110dbff2c15dfe); same product tree as merge, with evidence bounded to the original source/build. |
| Installed Host | Independently query exact version, commit, schema and discovery. No current Host acceptance inferred. |

## Release readiness: NO-GO

Actual final #191 Actions evidence (PR head cb67f99, not a stable build):

| Gate | Result |
| --- | --- |
| [Quality](https://github.com/vanishs/boxweave/actions/runs/46098278881) | FAIL, full Go race regression. |
| [S1](https://github.com/vanishs/boxweave/actions/runs/46098278814) | FAIL, unit/race regression. |
| [Candidate](https://github.com/vanishs/boxweave/actions/runs/46098278827) | FAIL, source tooling and both architecture jobs; manifest skipped. |
| [N2](https://github.com/vanishs/boxweave/actions/runs/46098278841) | PASS, 29/29 installed lanes, bounded native evidence. |
| [S2](https://github.com/vanishs/boxweave/actions/runs/46098278865) | PASS, PR exact-CI crosswalk marker observed. |
| [S2.5](https://github.com/vanishs/boxweave/actions/runs/46098278831) | PASS, scoped independent nested lifecycle; not hostile recursive isolation. |
| [Forward](https://github.com/vanishs/boxweave/actions/runs/46098278810) | PASS, three policy lanes; not public WAN reachability. |

The same two Install tests fail across five jobs: the retired shared-ACL
reader returns ErrResourceLocked before the old positive and interleaving
oracles run. See [#194](https://github.com/vanishs/boxweave/issues/194),
[current source](https://github.com/vanishs/boxweave/blob/b6f7ae09edb927d204fbcaaf9b2ecd6f98e7f211/internal/state/network_audit_ledger.go)
and the [post-merge review](https://github.com/vanishs/boxweave/pull/191#issuecomment-7232991141).
This does not establish a new live Scoped defect or pass the retained
composed real protected SQLite/native interleaving obligation. The original
refused test-write must not be replayed or delegated; preserve all coverage
and seek legitimate review of the original denial where necessary.

The [owner's source-integration decision](https://github.com/vanishs/boxweave/pull/191#issuecomment-7232633581)
reports removing required quality-gate protection for the merge. It did not
waive acceptance. Live API readback confirms v1 protected=false, ruleset
[26842628](https://github.com/vanishs/boxweave/rules/26842628) disabled
and required status checks empty. Restore and independently verify the
intended protection under owner authority; stable candidate and signed
publication require protected v1. The
[#197 quality run](https://github.com/vanishs/boxweave/actions/runs/46103061090)
passed only the documentation fast path, skipping Go race/build.
Actual current v1 push [quality](https://github.com/vanishs/boxweave/actions/runs/46103179949)
and [candidate](https://github.com/vanishs/boxweave/actions/runs/46103179985)
runs both FAIL at b6f7ae0; all four failed jobs reproduce the same two
Install tests, and the manifest job is skipped. The previous draft status
is historical, not an instruction to repeat P1.

Before stable: resolve H/C01 and remaining C04/C11/C12/P5 evidence,
reconcile all supported acceptance IDs and [#170](https://github.com/vanishs/boxweave/pull/170),
verify protection, build an exact stable candidate without publication,
reconcile artifact/native evidence, obtain independent trusted signatures
and protected dry-run. GitHub returns zero v1 workflow_dispatch runs and no v1.0.0 tag or
Release; no stable candidate or signed release evidence was verified. Do not alter the selector or publish.

See the [acceptance ledger](development/CONSOLIDATION_ACCEPTANCE.md),
[workflow dispositions](development/P4_WORKFLOW_DISPOSITION.md), and
[versioning](process/VERSIONING.md). The older phase records are retained
with their original scope.

## Historical merged changes beyond RC13

The following entries preserve the 2026-11-24 audit at 06d2743;
they do not establish the present installed contract.

| PR | Scope |
| --- | --- |
| [#178](https://github.com/vanishs/boxweave/pull/178) | Scoped Peer v2; default autostart, independent ACLs, durable Root guard and unrelated-flow acceptance. [N5 evidence](history/development/V1_PR178_N5_DELIVERY_2026-11-23.md). |
| [#177](https://github.com/vanishs/boxweave/pull/177) | Local read-only Doctor and reviewed Recover. [D5 delivery](history/development/V1_PR177_D5_DELIVERY_2026-11-24.md). |
| [#180](https://github.com/vanishs/boxweave/pull/180) | Scoped fresh preview and mixed-mode refusal. |
| [#181](https://github.com/vanishs/boxweave/pull/181) | Guided legacy preview, phase markers, native drift refusal, original retirement resume. [Incident closure](history/development/V1_INSTALL_UNINSTALL_INCIDENT_CLOSURE_2026-11-24.md). |
| [#182](https://github.com/vanishs/boxweave/pull/182) | Scoped full retirement/resume; [native CI](https://github.com/vanishs/boxweave/actions/runs/45376800724). |
| [#183](https://github.com/vanishs/boxweave/pull/183) | Portable README SVG. |
| [#184](https://github.com/vanishs/boxweave/pull/184) | Completed-data purge and same-Host Legacy-to-Scoped fresh install. [Source](https://github.com/vanishs/boxweave/actions/runs/45381844478), [quality](https://github.com/vanishs/boxweave/actions/runs/45381844529), [dual arch](https://github.com/vanishs/boxweave/actions/runs/45381844630), [native](https://github.com/vanishs/boxweave/actions/runs/45381844479), all passed at fb6785f. |

Source, publication, installed version and physical acceptance are distinct. See [risks](verification/OPEN_RISKS.md), [versioning](process/VERSIONING.md), [installation](getting-started/MANUAL_INSTALL.md) and [historical evidence](history/README.md).
