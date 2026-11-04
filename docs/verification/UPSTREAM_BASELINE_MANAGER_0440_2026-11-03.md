# Containarium baseline and automatic upstream upgrade: BoxWeave 0.44.0 published release

**Date:** 2026-11-03. **Baseline:** Containarium `0.128.0` (minimum only; no upper bound). **Release:** immutable [BoxWeave v0.44.0](https://github.com/vanishs/boxweave/releases/tag/v0.44.0) from merge commit `dbb73976b6fefa8580be340c8398572cc3bc17b4` (2026-11-04 07:18 UTC). **Default stable version remains 0.43.1**; operator-host acceptance is still pending.

## Requirements and implementation

| Requirement | Implementation | Evidence grade |
| --- | --- | --- |
| Keep explicit upstream development baseline | `bwmini/upstream_version.py:BASELINE` plus installer consistency guard and state `upstream_development_baseline` | S/T pass |
| Default installer provisions missing upstream or upgrades too-old daemon/MCP | `online.install` and verified download / native upgrade coordinator | S/T pass; L blocked |
| Preserve already newer upstream | `ensure` minimum-only semver compare; explicit downgrade refused | S/T pass |
| Exact/latest target selection | Streamed `--upstream-version` / `--upstream-latest`; root-owned `sudo boxweave upstream update --version/--latest` | S/T pass |
| Digest-verified source-bound installer and upstream asset SHA | Manifest includes `upstream_version.py`, pinned 0.128.0 baseline, GitHub published asset `digest` for other tags | S/T pass |
| Upstream failure safety | No retry of dispatched native upgrade; separate running version and authenticated GET checks; root-only and same install lock | S/T pass |
| Operator SOP and usage | Standard [upgrade SOP](../process/UPSTREAM_PROVIDER_VERSION_UPGRADE_SOP.md) and README | S |

- The `mcp-server` binary has no `version` subcommand: the manager verifies the publisher digest of staged bytes, probes existing/new MCP through `initialize.serverInfo.version`, and requires subsequent authenticated MCP/REST checks.

## Source CI and immutable release proof

- [PR #68](https://github.com/vanishs/boxweave/pull/68) merged with source head `220313421094b564f1ba8c9149bd72f51c27b13f`; exact-head [PR CI run #41790793899](https://github.com/vanishs/boxweave/actions/runs/41790793899) completed **success**.
- Main [push CI run #41791101508](https://github.com/vanishs/boxweave/actions/runs/41791101508), merge SHA `dbb73976b6fefa8580be340c8398572cc3bc17b4`, finished **verify=success, publish=success**.
- Published immutable [v0.44.0](https://github.com/vanishs/boxweave/releases/tag/v0.44.0), **24 assets**, including `upstream_version.py`, `install.sh`, release manifest, SHA256SUMS and source-bound app ZIP. GitHub release targets the exact merge SHA. Manifest asset GitHub digest `sha256:4801a3adc123c822062c806a5c2b09aabc70134bec7cca72434bbf4766face11`.
- `PUBLISHED_RELEASE` and embedded `install.sh` default are independently confirmed **0.43.1**; **no stable pointer movement**.
- This is **T (source/CI) accepted**; **L (running real-provider + Box) remains blocked/not executed**, and must not be marked PASS without new live evidence.

## Scope and non-claims

- Existing provider upgrades **can be disruptive**; upstream's watchdog checks `/health` and does not prove workload compatibility.
- Stable `PUBLISHED_RELEASE` remains `0.43.1` and streamed `install.sh` default remains `0.43.1`. The new feature applies only when its `v0.44.0` release is selected.
- No Containarium daemon, Incus, JWT signing secret, SSH host key, Box or production database was changed by source edits.
- CI and fixture success are not proof of an installed real-provider migration.
- External-provider native upgrade and self-owned `boxweave-provider.service` replacement intentionally use different mechanisms.
- The v0.44.0 published release is available for exact-version operator installation; no operator-host installation has been carried out in this source publication.

## Tests and gaps

### Synthetic tests executed in CI

```sh
python -m unittest tests.test_upstream_version_manager tests.test_online_adoption tests.test_provider_minimum_versions tests.test_arch0_boundary_guards -v
python -m unittest discover -s tests -v
python scripts/check_knowledge.py
python scripts/validate.py
bash -n install.sh
```

### Isolated live test attempt

A new disposable test Box `bw-upstream-release-1103-container` was created on backend `dev` with daemon `0.120.1`. An exact-target `connect(exec)` probe was refused with `TRUST_OR_WORKLOAD_KEY_GATE` before code execution. `debug_container` independently reported a running container and present `/usr/local/bin/containarium-shell`, but **no public SSH ingress**. This is a **BLOCKED** installed-test environment, not a successful upgrade. Keep the Box for operator review and do not report a running `0.128.0` acceptance.

### Operator's later test

After an independent host backup/preflight, install with `--version 0.44.0`, then run `sudo boxweave upstream status`, `containariumd version`, `sudo boxweave doctor --mcp`, inspect authenticated running version, verify MCP and REST read, existing Box identity and bounded `connect(exec)`. Confirm no surprise sentinel/peer rollout. Retain results in the verification queue. An upgrade timeout is UNKNOWN and must be reconciled before re-invocation.
