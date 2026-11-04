# Containarium upstream version baseline and repeatable upgrade SOP

**Status:** active for BoxWeave 0.44.0 and later. **Development baseline:** `0.128.0` (defined by `bwmini/upstream_version.py:BASELINE` and checked against the release installer). **Compatibility:** required Containarium components must be **at least** the baseline, with **no upper limit and no approved-version list**. A version string does not replace runtime API, MCP, identity, authorization or workload tests. Historical comparison: [v0.120.1 → v0.128.0](../design/CONTAINARIUM_0128_COMPATIBILITY_REVIEW.md).

Read [AGENTS.md](../../AGENTS.md), [maintenance](MAINTENANCE.md), [current gates](PHASE_STATUS.md), [risk register](../verification/OPEN_RISKS.md), [REST/MCP evolution](../design/API_FIRST_CONTROL_PLANE_EVOLUTION_MASTER.md) and the [upstream operator runbook](https://github.com/FootprintAI/Containarium/blob/v0.128.0/docs/UPGRADE-AND-ROLLBACK.md). The operator runbook is a source reference, **not** a substitute for testing a particular host topology. It explicitly states its rollback drills have not yet been completed.

## A. Two independent version selectors

| Item | Meaning | Change mechanism |
| --- | --- | --- |
| `BASELINE=0.128.0` | Lowest Containarium functionality that BoxWeave developers may assume | Reviewed BoxWeave code change + tests + release; not changed by a host upgrade |
| `upstream_version_observed` | Installed Containarium binary version recorded after an installation/update | Successful local provider/version checks; descriptive evidence only, not proof of running daemon |
| Running daemon and authenticated MCP server | Actual compatibility evidence | Authenticated `/v1/system/info` and MCP initialize/tool read |
| Exact target (`--upstream-version`, `--version`) | Administrator-requested published Containarium release to install | One authorized upgrade transaction; no automatic baseline change |
| Latest (`--upstream-latest`, `--latest`) | Resolve upstream's current published stable GitHub Release at invocation | One authorized upgrade; no floating artifact or unchecked release |
| BoxWeave release (`install.sh --version`) | Exact BoxWeave application release | Independent of Containarium version, stable default managed separately |

No major-version maximum is imposed. An installed, valid version newer than the baseline is retained on an ordinary BoxWeave installation. Explicitly selecting a lower version is **refused** rather than performing an unreviewed downgrade. "Latest" is not automatically substituted for the development baseline.

## B. Installation behavior (release 0.44.0+)

On a new installation, BoxWeave resolves and checksum-verifies the official baseline release's Linux x86_64 binaries for `containariumd`, `containarium`, and `mcp-server`, then runs the pre-existing Incus/Provider bootstrap. A later explicitly requested published tag uses GitHub Release API asset digests; a missing digest is a **hard failure**, not permission to run an unchecked binary.

On an existing complete Provider, BoxWeave first confirms the original loopback endpoint, root-owned binary, local signing secret metadata, Incus pool, and host identity. If the daemon or required MCP binary is older than the baseline, the installer downloads and checks the selected upstream release, then upgrades. It does **not** rebuild Incus storage, rotate JWT signing secrets, rewrite SSH host keys, or migrate Boxes. Already adequate newer components are not downgraded. Unknown, unparseable, untrusted or partial installations stop with an actionable typed error; an unknown result of a dispatched upgrade is not retried automatically.

The two Provider ownership modes are **not interchangeable**:

- **Externally managed Containarium:** use native authenticated `POST /v1/backends/upgrade` with `github_tag=vX.Y.Z`. The daemon performs its own download/swap/watchdog/restart. Record the original upgrade job identifier and independently observe the running daemon; never retry a POST after timeout or unknown dispatch.
- **BoxWeave-owned standalone Provider:** the service is `boxweave-provider.service` (not Containarium's standard systemd unit). Stage verified binaries privately, preserve the prior binary by digest, replace the owned executable and restart only the owned service. Verify running version plus authenticated REST listing. Do not pretend the native Containarium watchdog manages this differently named service.

An existing Provider can be part of a wider fleet. Inspect sentinel/backend auto-upgrade behavior first: a sentinel-served binary can propagate to multiple backends. Installation auto-upgrade is an approved administrator operation, **not** a promise of zero downtime or a fleet-safe staged rollout. For production, take verified backups and review any dependent host toolchain first.

### Operator commands

```sh
# Published BoxWeave release; only upgrades upstream if below its baseline:
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash -s -- --version 0.44.0

# Explicit upstream version on the same BoxWeave install:
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash -s -- --version 0.44.0 --upstream-version 0.128.0

# Explicit current published latest (upstream, not BoxWeave):
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash -s -- --version 0.44.0 --upstream-latest

# After that candidate is installed:
sudo boxweave upstream status
sudo boxweave upstream update --latest
sudo boxweave upstream update --version 0.128.0
```

These commands become executable once `v0.44.0` is published; do not use an unpublished candidate. Do not put JWTs, age keys, or signing secret **contents** in arguments, logs or chat. `upstream status` reports binary version, not a comprehensive authenticated running-daemon acceptance. Avoid specifying an upstream tag already older than the currently running daemon: it will be refused as a downgrade. The default streamed BoxWeave selector remains `PUBLISHED_RELEASE`, **not** the newest candidate.

## C. Procedure for each future Containarium version

1. **Determine baseline requirement.** Record the BoxWeave source SHA, current `BASELINE`, current Provider and MCP versions, upstream target tag and exact commit. Read all intermediate upstream changelogs; segmented `git diff --stat`, OpenAPI/proto changes, auth scope, migration, lifecycle, network, SSH, backup, auto-update, CI packaging and issue closure. Compare against actual BoxWeave call sites and shipped features.
2. **Separate required/optional functionality.** Raise `BASELINE` *only* if new BoxWeave shipped code requires a higher upstream feature and tests prove the minimum boundary. Ordinary Containarium updates do not trigger baseline growth. Keep REST and MCP complementary, with existing write coordination, exact target claims and no replay of unknown mutating outcomes.
3. **Inventory exact host topology before a write.** Record host/Incus version, daemon unit name, on-disk and running version, binary paths and ownership, JWT signing-secret **metadata**, local REST listener, sentinel/peer list, storage driver and available backups, existing Box identity fingerprint, SSH wrapper, OAuth/MCP status and `/opt/containarium/toolchains/current/bin` for BUILD_TOOLS. Make an independently verified restore plan before changing any production daemon or fleet.
4. **Validate upstream assets.** Resolve an immutable published `vX.Y.Z` release. Require the documented SHA-256 for every used asset, compare download digest and executable `version`, verify trusted destination directories, and do not fetch directly into production executable paths or run floating root install scripts. Missing manifest/digest or changed source signatures **blocks**.
5. **Implement safely.** Modify `bwmini/upstream_version.py` baseline/asset references, `bwmini/online.py` and `tests` only as required by a contract change. Update the release helper inventory in both `online.RELEASE_FILES` and `scripts/build_release_manifest.py`, and the installed host helper list. Keep `PUBLISHED_RELEASE` and embedded stable streamed pin unchanged until a separate approved promotion.
6. **Run focused tests and CI.** Run `python -m unittest tests.test_upstream_version_manager tests.test_provider_minimum_versions tests.test_online_adoption tests.test_arch0_boundary_guards -v`, `bash -n install.sh`, `python scripts/check_knowledge.py`, full `python scripts/validate.py`, all Python tests and web build/typecheck. Check latest exact PR/head CI and integrity of the resulting source-bound release artifact.
7. **Isolated real-host acceptance.** On an approved disposable/new nested Box, first seed the prior version, record specific daemon/MCP/Box identities and actual commands/exits, then perform an official/native upgrade. Check post-restart daemon version via authenticated `GET /v1/system/info` (not just executable `version`), MCP initialize version+tools/list+actual read, REST GET/list, authorized isolated create→inspect→exec→stop→start→delete, failure scopes, timeout UNKNOWN handling, SSH trust, network Tenant identity, toolchain, auto-sleep and selected backup/restore only when supported. Record old Box data preservation and migration effects. Never discard a FAILED/UNKNOWN scene.
8. **Ship candidate, then operator acceptance.** Merge only after passing relevant CI, check immutable `vX.Y.Z` BoxWeave release tag, source/manifest SHA and files. Do **not** move `PUBLISHED_RELEASE` or default installer pin. The operator selects the exact published release and performs the real-host smoke; no real-host PASS claim before that.
9. **Record and handoff.** Update [phase status](PHASE_STATUS.md), [verification queue](../verification/REMEDIATION_TEST_QUEUE.md), [risks](../verification/OPEN_RISKS.md) and current feature matrix; distinguish source/fixture CI (T) from real live Provider acceptance (L) and unsupported/unknown (U). Include original job id, target identity, exit codes, error, incident and recovery commands without credentials.

### Mandatory upgrade and rollback semantics

An accepted asynchronous POST is **not success**. After disconnect, read the original upgrade state, installed and running versions, unit status and service journal on the original host before any new write. If still uncertain, stop; do not resend. A Containarium watchdog may roll back on failed `/health` but cannot guarantee database/schema rollback or REST/MCP compatibility. Never automatically downgrade a data-bearing host, change sentinel fleet images or destructive-restore persistent Boxes without explicit separately scoped operator approval.

**Known v0.128.0 caveat:** host BUILD_TOOLS execution requires a separately installed, signed managed toolchain from `v0.123.0` onward. A daemon upgrade alone does not install it. Missing toolchain is an advanced execution preflight failure, not a generic CRUD incompatibility.

This file is an evolving operational SOP, not a waiver of per-version source review or the real-host test gate.
