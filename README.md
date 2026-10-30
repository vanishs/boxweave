# BoxWeave

**Current source version: 0.13.5.** BoxWeave is a self-hosted, single-operator workspace for remote development with project/environment bindings, deliberate lifecycle actions, durable operation receipts, Tenant membership and network access, TCP/UDP publication, Prompt Graph, and an assisted ZCode model configuration handoff.

The current application was independently reconstructed from the archived 0.5.1 implementation; it is **not** a drop-in replacement for the old host deployment. The historical source is at [archive/0.5.1](https://github.com/vanishs/boxweave/tree/archive/0.5.1). Do not execute the archived installer against a new installation. Phase 7 maintenance has completed in its documented test scope; host bootstrap and operational parity are a new remediation effort, not a previously accepted release. See [current status](docs/process/PHASE_STATUS.md) and [open risks](docs/verification/OPEN_RISKS.md).

## Support and boundaries

| Area | Current supported scope |
| --- | --- |
| Agent and transport | Windows-hosted Codex or ZCode via native SSH or explicit Ubuntu WSL SSH (four profiles) |
| Server | Non-root Ubuntu 24.04, Linux x86_64, CPython 3.12, systemd user manager for the packaged application |
| Infrastructure | A reviewed Containarium Provider, Incus and an independently authorized application MCP launcher |
| Web | Local administrator, bindings, environments, Tenant/port plans, receipt history, Graph and client setup guidance |

macOS, Cursor, Claude Code, WSL-hosted Agent binaries, automatic inference, recursive deployment and an automatically trusted public HTTPS endpoint are **not** included in the selected current support matrix. Provider/Incus bootstrap, host-key rotation, SSH hardening and token ownership are not supplied by the existing v2 application package: [host-installation work and operator boundaries](docs/HOST_BOOTSTRAP.md). A new [read-only management CLI](docs/CLI.md) is included in source and release packages. The new [host installer](install.sh) supports reviewed local release preflight, non-root app installation/update/repair and optionally a separately pinned upstream bootstrap on an empty host. It **does not yet create an MCP launcher, app account, login bus, SSH authority or real TLS**; these remain explicit operator prerequisites and a tracked end-to-end gap. Do not treat this source as verified fresh-host deployment.

## Install, update, or repair (real Provider)

**Start here for a real BoxWeave application, not with the fixture.** The supported operator-facing entry is [`install.sh`](install.sh), which checks reviewed local artifacts and delegates application installation, updates, and repairs to the existing verified-package mechanism. This is **not yet a complete one-command setup from a blank Ubuntu host**: creating the application account and user systemd bus, preparing an independent MCP launcher and credentials, SSH trust, and TLS still require separate operator work and acceptance.

Before running the installer, prepare Ubuntu 24.04 x86_64 with Python 3.12, an existing non-root app user with a working systemd user bus, reviewed Incus/Containarium dependencies, an independently authorized working MCP launcher, a private [live-mode configuration](examples/live.json), and a trusted offline application archive. Build the archive using the exact locked wheels and built Web UI as described in [installation and release packaging](docs/INSTALL.md). Obtain **independently reviewed SHA256 digests** for both the archive and the local `bwmini/distribution.py` file; do not treat a digest downloaded alongside untrusted code as proof of authenticity.

From a reviewed checkout, with the prerequisites in place:

```sh
# Read-only preflight; replace placeholders with actual independently reviewed digests.
bash install.sh plan --app-user boxweave \
  --archive /reviewed/boxweave-v2.zip --sha256 ARCHIVE_SHA256 \
  --distribution-sha256 DISTRIBUTION_PY_SHA256 \
  --config /reviewed/application.json

# Explicit installation, update, or repair after reviewing the preflight result.
bash install.sh apply --app-user boxweave \
  --archive /reviewed/boxweave-v2.zip --sha256 ARCHIVE_SHA256 \
  --distribution-sha256 DISTRIBUTION_PY_SHA256 \
  --config /reviewed/application.json
```

The placeholders above are intentionally non-operational. `plan` performs local checks without installing anything; `apply` may fail with `ADMIN_SETUP_REQUIRED` until local administrator initialization and a subsequent rerun. It does not declare the application ready without real authenticated MCP checks. For first administrator setup, upgrades, backups, failure markers, and recovery, use [INSTALL](docs/INSTALL.md) and the [host bootstrap prerequisites](docs/HOST_BOOTSTRAP.md). The optional `--provision-provider` path is a separately approved, digest-pinned, privileged operation for an otherwise empty host; it is **not** an automatic Provider upgrade.

**Do not run the archived 0.5.1 installer or pipe an unreviewed remote script to sudo.** An application source commit is not proof of clean-host installation, boot persistence, TLS, effective confinement, Agent GUI, or model inference. See [current operational gaps](docs/verification/OPEN_RISKS.md).

## Operator and contributor navigation

- [Documentation map](docs/INDEX.md), [host/bootstrap boundaries](docs/HOST_BOOTSTRAP.md), [application installation](docs/INSTALL.md)
- [Management CLI](docs/CLI.md), [client profiles](clients/README.md) and [scope](docs/product/SCOPE.md)
- [R-00 cross-branch audit](docs/verification/ARCHIVE_051_TO_MAIN_AUDIT.md), [semantic parity F-001–F-015](docs/verification/FEATURE_PARITY_ARCHIVE_051.md), [Web workflow mapping](docs/verification/WEB_WORKFLOW_PARITY.md), [feature/evidence matrix](docs/verification/FEATURE_MATRIX.md), [remaining risks](docs/verification/OPEN_RISKS.md)
- [Manual migration from 0.5.1](docs/MIGRATING_FROM_051.md), [troubleshooting](docs/TROUBLESHOOTING.md), [upstream contract](docs/UPSTREAM_PROVIDER.md)
- [Contributing agent instructions](AGENTS.md), [handoff](docs/process/FINAL_HANDOFF.md), [change procedure](docs/process/MAINTENANCE.md)

Historical version-by-version reports are indexed under [verification](docs/INDEX.md); they describe their original environment and source, not today's deployment.

## Development & testing

### Isolated fixture (synthetic)

This is the local developer/demo and regression-test mode, **not a real installation**. It exercises the Web UI, project binding, and operation receipts using synthetic infrastructure without contacting or changing a real Incus/Containarium Provider.

On a prepared non-root Ubuntu 24.04 x86_64 machine with Python 3.12, Node/npm, browser test dependencies if needed, and no occupied port 18640:

```sh
python3 -m venv .venv
.venv/bin/pip install --require-hashes -r requirements-dev.lock
(cd web && npm ci --ignore-scripts --no-audit --no-fund && npm run build)
.venv/bin/python -m bwmini init-admin --config examples/fixture.json
.venv/bin/python -m bwmini worker --config examples/fixture.json
# In another terminal:
.venv/bin/python -m bwmini serve --config examples/fixture.json
```

Open `http://127.0.0.1:18640`, sign in using the administrator password entered locally, save a project binding and inspect a receipt. This is a **synthetic fixture**: working UI and simulated activity do not prove real Provider access or isolation. Processes are foreground-only and bind loopback. See [verification strategy](docs/verification/TEST_STRATEGY.md).

