# BoxWeave

**Your agents. Your machines. Connected.**

BoxWeave is a self-hosted control plane for AI coding, remote development environments, and an independent Web console. Containarium and Incus manage the underlying containers.

**Current source version: 0.23.0.** This is a preview, not a certified clean-host deployment.

## Install (Ubuntu 24.04+ recommended)

The desired default command is:

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash
```

**Ubuntu 24.04 or newer is the recommended host**, not a required distribution. The installer checks Linux x86_64, CPython >= 3.12, systemd, Provider/Incus availability and needed package tooling directly. On hosts without APT, prepare reviewed dependencies and Provider yourself; BoxWeave does not invoke an alternative package manager. Other distributions require separate live acceptance. It verifies source-bound release assets, provisions missing infrastructure on a clean host, **preserves an already configured Incus storage pool**, and **reuses an existing complete Containarium** without installing or upgrading it. Existing Provider identity, configuration, network and storage remain external. Partial installations are refused. See [existing Provider instructions](docs/verification/EXISTING_PROVIDER_INSTALL.md) and [two-box MCP test](docs/verification/STREAMED_INSTALL_TEST.md). The released offline wheelhouse includes CPython 3.12–3.15 x86_64 wheels; Ubuntu 26.04 requires live host acceptance. The supported client catalog covers Linux/macOS native SSH and Windows native/WSL SSH; real App acceptance remains pending.

A successful streamed install prints the verified installed BoxWeave version and
prominently shows `boxweave client`. For a fresh host without Incus, the
managed-Provider path runs an APT dependency simulation and selects a coherent
Incus package set without the old partial `7.5.1` pins; it never upgrades
an already-installed Incus. This does not yet mean that Incus/PostgreSQL
provisioning has been moved into an independently owned Containarium installer.

Persistent MCP-over-SSH client registrations use `ServerAliveInterval=15`
and `ServerAliveCountMax=4` to mitigate silent idle connection loss.
Keepalive detects an unresponsive connection but does not automatically
restart a terminated SSH/MCP session.

The Provider API and Web application stay on loopback; the MCP client uses the existing SSH service with an explicitly enrolled client public key and a verified server pin. No public route, firewall change or SSH host-key replacement is performed. The initial Web administrator password is saved only in a private file whose location is printed; it is never printed by the installer.
The explicit `bootstrap-plan` / `bootstrap` workflow can now prepare a dedicated application account, systemd user bus, independent scoped MCP launcher and rotating credential on an already reviewed Provider. It requires independently reviewed artifact digests and explicit root-change acknowledgment; see [application bootstrap](docs/HOST_BOOTSTRAP.md#independent-application-bootstrap).

On a prepared Linux x86_64 host with Python 3.12+, reviewed Containarium/Incus, a non-root application user with a functioning systemd user bus, an authorized MCP launcher and a trusted release archive, use the existing installer:

```bash
bash install.sh plan --app-user boxweave \
  --archive /reviewed/boxweave.zip --sha256 ARCHIVE_SHA256 \
  --distribution-sha256 INSTALLER_SHA256 --config /reviewed/application.json

bash install.sh apply --app-user boxweave \
  --archive /reviewed/boxweave.zip --sha256 ARCHIVE_SHA256 \
  --distribution-sha256 INSTALLER_SHA256 --config /reviewed/application.json
```

The paths and SHA256 values are placeholders, not ready-made downloads. The first installation may require local administrator initialization and a rerun. See [installation and repair](docs/INSTALL.md) and [host prerequisites](docs/HOST_BOOTSTRAP.md).

### Common installer parameters

| Option | Purpose |
| --- | --- |
| `plan` | Read-only prerequisites and artifact verification |
| `apply` | Install, update or repair from the reviewed package |
| `--app-user NAME` | Existing non-root application account |
| `--archive FILE` / `--sha256 HEX` | Application ZIP archive and trusted digest; `--package` is a compatible alias |
| `--distribution-sha256 HEX` | Trusted digest of the application installer code |
| `--config FILE` | Approved application/MCP configuration |
| `--prefix PATH` | Optional installation location in the application user's home |
| `--provision-provider` | Explicit empty-host provisioning only; requires a separately verified upstream installer, SHA256 and root-change acknowledgment |

Run `bash install.sh --help` for all parameters.

Streamed install/update/repair automatically preserves an already-installed complete Containarium and Incus. It does **not** adopt an unrelated BoxWeave installation or silently repair/upgrade a third-party Provider. For a preinstalled `dir` storage pool on bcachefs, first verify its source and mount dependencies; the installer will not reconfigure the storage pool.

## After installation: finish the setup

The installer prints a `next_steps` record and prominently shows the **first
action: configure your client**. As your ordinary Linux administrator login
(**no sudo**), run:

```sh
boxweave client
boxweave client --help  # More options
```

**Optional commands** for viewing the Web address and diagnosing the host (not
required before generating client configuration):

```sh
boxweave                # Host overview and connection instructions
boxweave --help         # Full host command reference
boxweave status         # Public Web reachability, not an authenticated MCP proof
boxweave doctor         # Safe host/service observations
```

The Web server is bound to **127.0.0.1** for safety, so its URL is
accessible on the server host only. To open it in your own browser, make an
authenticated tunnel from your local computer (use the actual port printed by
`boxweave`):

```sh
ssh -N -L 18640:127.0.0.1:18640 YOUR_HOST_ADMIN@YOUR_LINUX_HOST
```

Then browse to `http://127.0.0.1:18640` locally (replace 18640 when
the installed port differs). The initial Web administrator password is stored
privately at `/home/boxweave/boxweave/initial-admin-password`: an authorized
administrator can read it locally; the installer never prints the password.
The service account `boxweave` is deliberately different from your SSH
administrator and must **not** be used as an everyday login.

Run `boxweave client` to choose the MCP App, client OS (Windows, Linux or
macOS) and SSH transport (native or Windows WSL). It prints a reviewed
self-contained Python 3.11+ client-side source on stdout for **every**
App, operating system and SSH transport, including Windows Codex/WSL.
An authorized administrator must enroll only the client public SSH key.
After a real authenticated MCP read, the script updates only the selected
App's MCP configuration (or prints official import JSON for Qoder/TRAE).
It never rewrites the client's AGENTS files. See
[alphabetically sorted client App support](clients/README.md).

If an installation fails, inspect its original log/phase and repair that
specific issue before retrying. Do not relax `/home/boxweave` from mode 0700
or grant the human login broad Incus privileges just to run the CLI.
[Host CLI reference](docs/CLI.md).


## Connect an AI coding agent

Use `boxweave client` (interactive) or select a supported profile:

```bash
boxweave client --app cursor --os linux --transport native --admin-user YOUR_HOST_ADMIN > boxweave-setup.py
boxweave client --app codex --os windows --transport wsl --admin-user YOUR_HOST_ADMIN > boxweave-setup.py
```

Supported Apps (**alphabetical**): **Antigravity, Claude (Code), Codex,
Copilot (CLI), Cursor, OpenCode, Qoder (IDE), TRAE (IDE), ZCode**.
Linux and macOS use native OpenSSH. Windows supports native OpenSSH or a
Windows-hosted App using WSL OpenSSH. Qoder and TRAE use official IDE JSON
import rather than edits to undocumented user configuration paths.

| Client platform | Agent | Transport | Support |
| --- | --- | --- | --- |
| Linux | All nine supported Apps | Native SSH | Generator/fixture; live App verification pending |
| macOS | All nine supported Apps | Native SSH | Generator/fixture; live App verification pending |
| Windows | All nine supported Apps | Native OpenSSH | Generator/fixture; Python 3.11+ required |
| Windows | All nine supported Apps | WSL SSH | Unified Python generator/fixture; WSL distro required |
| WSL-hosted Agent | Any | Independent Linux runtime | Not implicitly configured by Windows generator |

The Web Client Setup form retains its four earlier approved local
descriptor profiles; the installed `boxweave client` CLI supports the
expanded catalog. Generating a script or passing its isolated MCP probe
does not prove the App has actually loaded or granted MCP tools.
The generated stdout starts exactly with `#!/usr/bin/env python3`;
all decorative separators are Python comments **after** the shebang.
Interactive prompts/summary messages are comment-prefixed on stderr, so
redirect stdout or use `--output NEW_FILE` for a clean, executable `.py`.
See [client setup](clients/README.md) for the individual App destinations
and prerequisites.

## Web console

The Web UI includes project and environment binding, reviewed lifecycle plans and durable receipts, Tenant membership and controlled network access, TCP/UDP port publication, Prompt Graph, model setup guidance for ZCode, and supported client setup downloads. Graphs do not execute commands and an HTTP response is not proof of authenticated MCP or effective network isolation.

## Common management commands

From a checked-out source directory or its matching installed release:

| Command | Purpose |
| --- | --- |
| `bash boxweave.sh version` | Read the local BoxWeave version |
| `bash boxweave.sh version --components` | Also observe Incus/Containarium CLI versions |
| `bash boxweave.sh status` | Observe application and user service state |
| `bash boxweave.sh doctor` | Read-only host and service diagnostics |
| `bash boxweave.sh doctor --mcp` | Additionally run an explicitly authorized MCP read |
| `bash boxweave.sh box inspect NAME --project default` | Inspect one exact Incus instance |
| `bash boxweave.sh client` | Display supported client profile guidance; no enrollment |

These commands do not automatically install missing components or modify the host. The command set is smaller than the historical implementation. See the [CLI reference](docs/CLI.md) for exit codes.

## Documentation

[Install and update](docs/INSTALL.md) · [Host prerequisites](docs/HOST_BOOTSTRAP.md) · [Client setup](clients/README.md) · [CLI reference](docs/CLI.md) · [Troubleshooting](docs/TROUBLESHOOTING.md) · [Supported scope](docs/product/SCOPE.md) · [Development and tests](docs/DEVELOPMENT.md) · [Open risks](docs/verification/OPEN_RISKS.md)

Container-host setup: see [nesting configuration and restart guidance](docs/NESTING.md).
