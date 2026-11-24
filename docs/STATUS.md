# Current source and release status

Reviewed baseline: v1 [06d2743](https://github.com/vanishs/boxweave/commit/06d27434b11997e9254822c122483c9dacba12a3), 2026-11-24. This is not a claim about any installed Host.

| Item | Observed baseline |
| --- | --- |
| Stable | Unpublished; see [stable selector](../V1_PUBLISHED_RELEASE). |
| Prerelease | [RC13](https://github.com/vanishs/boxweave/releases/tag/v1.0.0-rc.13), source 6c995687, latest published tag verified at audit. |
| RC14 | [Release branch](https://github.com/vanishs/boxweave/tree/release/v1.0.0-rc.14) at 06d2743; publication not yet verified. Check [release CI](https://github.com/vanishs/boxweave/actions?query=branch%3Arelease%2Fv1.0.0-rc.14) and [actual Releases](https://github.com/vanishs/boxweave/releases) before installing. |
| Runtime | 23 public MCP Tools, SQLite schema 6, independently revocable management Actors. |
| Modes | Legacy; explicit fresh-only Scoped Peer v2 (no migration, upgrade or rollback). |
| Maintenance | Local Doctor/Recover, full-only scoped retirement with original preimage, separate completed-data purge. These newer features are not in RC13. |

## Merged changes beyond RC13

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
