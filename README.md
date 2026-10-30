# BoxWeave

**Current source version: 0.10.0.** BoxWeave is a self-hosted, single-operator workspace for remote development with project/environment bindings, deliberate lifecycle actions, durable operation receipts, Tenant membership and network access, TCP/UDP publication, Prompt Graph, and an assisted ZCode model configuration handoff.

The current application was independently reconstructed from the archived 0.5.1 implementation; it is **not** a drop-in replacement for the old host deployment. The historical source is at [archive/0.5.1](https://github.com/vanishs/boxweave/tree/archive/0.5.1). Do not execute the archived installer against a new installation. Phase 7 maintenance has completed in its documented test scope; host bootstrap and operational parity are a new remediation effort, not a previously accepted release. See [current status](docs/process/PHASE_STATUS.md) and [open risks](docs/verification/OPEN_RISKS.md).

## Support and boundaries

| Area | Current supported scope |
| --- | --- |
| Agent and transport | Windows-hosted Codex or ZCode via native SSH or explicit Ubuntu WSL SSH (four profiles) |
| Server | Non-root Ubuntu 24.04, Linux x86_64, CPython 3.12, systemd user manager for the packaged application |
| Infrastructure | A reviewed Containarium Provider, Incus and an independently authorized application MCP launcher |
| Web | Local administrator, bindings, environments, Tenant/port plans, receipt history, Graph and client setup guidance |

macOS, Cursor, Claude Code, WSL-hosted Agent binaries, automatic inference, recursive deployment and an automatically trusted public HTTPS endpoint are **not** included in the selected current support matrix. Provider/Incus bootstrap, host-key rotation, SSH hardening and token ownership are not supplied by the existing v2 application package: [host-installation work and operator boundaries](docs/HOST_BOOTSTRAP.md). A new [read-only management CLI](docs/CLI.md) is included in source and release packages. The new [host installer](install.sh) supports reviewed local release preflight, non-root app installation/update/repair and optionally a separately pinned upstream bootstrap on an empty host. It **does not yet create an MCP launcher, app account, login bus, SSH authority or real TLS**; these remain explicit operator prerequisites and a tracked end-to-end gap. Do not treat this source as verified fresh-host deployment.

## Try the isolated fixture

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

## Install/update the real application

**Do not use the removed 0.5.1 `curl .../main/install.sh | sudo bash` command.** The existing package installer manages only the independently deployed app; it does not install Incus, Containarium, SSH trust, TLS or a usable MCP launcher. The integrated host-install flow is a separately gated part of the 0.10.x remediation; verify its completion and tests before operating it.

Prerequisites: independently approved and reachable Provider/MCP; reviewed executable launcher; private application config derived from [live example](examples/live.json), non-root systemd user bus, trusted release archive and independently supplied SHA256. With the exact locked wheels and UI already built, create the package in the prepared Linux environment:

```sh
python3 bwmini/distribution.py build --root . --wheels /private/build/wheels \
  --output /private/build/boxweave-v2.zip \
  --source-commit REVIEWED_40_CHARACTER_COMMIT --source-state clean
python3 /trusted/distribution.py install --archive /trusted/boxweave-v2.zip \
  --sha256 REVIEWED_ARCHIVE_SHA256 --prefix "$HOME/boxweave-v2" \
  --config /private/application.json --activate
```

Generate the wheels first with the hash-locked requirements as described in [installation, first administrator, update and repair](docs/INSTALL.md). Expected result: immutable verified release, private configuration and app user services. First install can report `ADMIN_SETUP_REQUIRED` until the administrator is explicitly initialized and the install rerun. The installer will not claim ready until it completes a real MCP read; a network or credential failure stays incomplete. **No fresh Ubuntu host, boot persistence, public TLS, real Windows GUI or model inference is claimed verified by this README.**

## Operator and contributor navigation

- [Documentation map](docs/INDEX.md), [host/bootstrap boundaries](docs/HOST_BOOTSTRAP.md), [application installation](docs/INSTALL.md)
- [Management CLI](docs/CLI.md), [client profiles](clients/README.md) and [scope](docs/product/SCOPE.md)
- [R-00 cross-branch audit](docs/verification/ARCHIVE_051_TO_MAIN_AUDIT.md), [feature/evidence matrix](docs/verification/FEATURE_MATRIX.md), [remaining risks](docs/verification/OPEN_RISKS.md)
- [Contributing agent instructions](AGENTS.md), [handoff](docs/process/FINAL_HANDOFF.md), [change procedure](docs/process/MAINTENANCE.md)

Historical version-by-version reports are indexed under [verification](docs/INDEX.md); they describe their original environment and source, not today's deployment.
