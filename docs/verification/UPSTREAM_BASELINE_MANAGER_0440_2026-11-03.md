# Containarium baseline and automatic upstream upgrade: BoxWeave 0.44.0 candidate

**Date:** 2026-11-03. **Baseline:** Containarium `0.128.0` (minimum only; no upper bound). **Release:** candidate source branch `feat/upstream-version-manager-0440`; no stable-default promotion.

## Requirements and implementation

| Requirement | Implementation | Evidence grade |
| --- | --- | --- |
| Keep explicit upstream development baseline | `bwmini/upstream_version.py:BASELINE` plus installer consistency guard and state `upstream_development_baseline` | S, pending CI |
| Default installer provisions missing upstream or upgrades too-old daemon/MCP | `online.install` and verified download / native upgrade coordinator | S, pending CI / installed real-host gate |
| Preserve already newer upstream | `ensure` minimum-only semver compare; explicit downgrade refused | S, test fixture pending |
| Exact/latest target selection | Streamed `--upstream-version` / `--upstream-latest`; root-owned `sudo boxweave upstream update --version/--latest` | S, pending CI |
| Signed/verified source-bound installer and upstream asset SHA | Manifest includes `upstream_version.py`, pinned 0.128.0 baseline, GitHub published asset `digest` for other tags | S, manifest/test CI pending |
| Upstream failure safety | No retry of dispatched native upgrade; separate running version and authenticated GET checks; root-only and same install lock | S, synthetic tests pending |
| Operator SOP and usage | Standard [upgrade SOP](../process/UPSTREAM_PROVIDER_VERSION_UPGRADE_SOP.md) and README | S |

- The `mcp-server` binary has no `version` subcommand: the manager verifies the publisher digest of staged bytes, probes existing/new MCP through `initialize.serverInfo.version`, and requires subsequent authenticated MCP/REST checks.
## Scope and non-claims

- Existing provider upgrades **can be disruptive**; upstream's watchdog checks `/health` and does not prove workload compatibility.
- Stable `PUBLISHED_RELEASE` remains `0.43.1` and streamed `install.sh` default remains `0.43.1`. The new feature applies only when its `v0.44.0` release is selected.
- No Containarium daemon, Incus, JWT signing secret, SSH host key, Box or production database was changed by source edits.
- CI and fixture success, if achieved, are not proof of an installed real-provider migration.
- External-provider native upgrade and self-owned `boxweave-provider.service` replacement intentionally use different mechanisms.
- A published release is usable by the operator only after GitHub Actions builds immutable source-bound assets and the release metadata is verified.

## Tests and gaps

### Synthetic tests to run

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

Once exact `v0.44.0` is published and after an independent host backup/preflight, install with `--version 0.44.0`, then run `sudo boxweave upstream status`, `containariumd version`, `sudo boxweave doctor --mcp`, inspect authenticated running version, verify MCP and REST read, existing Box identity and bounded `connect(exec)`. Confirm no surprise sentinel/peer rollout. Retain results in the verification queue. An upgrade timeout is UNKNOWN and must be reconciled before re-invocation.
