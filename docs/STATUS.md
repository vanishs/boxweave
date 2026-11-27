# Current source and release status

Observed 2026-11-27. This page separates published artifacts, merged
source, the active draft candidate, and installed Host observations.

| Identity | Observed status |
| --- | --- |
| Stable v1 | UNPUBLISHED; [selector](../V1_PUBLISHED_RELEASE). |
| Published prerelease | [RC14](https://github.com/vanishs/boxweave/releases/tag/v1.0.0-rc.14), tag commit [3fb71a6](https://github.com/vanishs/boxweave/commit/3fb71a6ddf47d13d165e5c039144aa66fad0eb05), schema 6, published 2026-11-24 22:34 UTC. GitHub metadata lists both Linux architectures, installer, uninstaller, manifest and checksums. Asset bytes/signatures were not independently verified in this documentation audit. |
| Merged v1 | [9d07665](https://github.com/vanishs/boxweave/commit/9d07665dcfc5ae6345d6bfcd0d71de7b6510b51d), schema 6; do not assume it is the published tag. |
| Draft [PR #191](https://github.com/vanishs/boxweave/pull/191) | Reviewed R2 source [7ec2dc3](https://github.com/vanishs/boxweave/commit/7ec2dc31b13efff094c864ecf4fce302a4e52ba8), schema 7, 23 MCP Tools, Scoped v2 only, no Legacy product upgrade/rollback or automatic migration. Not merged or published. |
| Installed Host | Query its exact version, commit, schema and actual discovery; never infer from this page or a release tag. |

## Candidate progress

The R1 predecessor 17afed0f passed N2 (29/29), independent S25,
S2 crosswalk and Forward. R2 corrects the obsolete installer order
test and aligns current native PR producer/consumer routes; exact
R2 CI is not fully accepted. H remains frozen pending Support and
owner direction. C04 public WAN and C01/C11 final evidence remain
open. See [progress](development/CONSOLIDATION_PROGRESS.md),
[acceptance](development/CONSOLIDATION_ACCEPTANCE.md) and
[workflow dispositions](development/P4_WORKFLOW_DISPOSITION.md).
The candidate supports normal and reviewed exceptional offline
Scoped retirement with original evidence and separate completed-data
purge; the tested predecessor scope is not universal Host approval.

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
