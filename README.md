# BoxWeave

**Your agents. Your machines. Connected.**

BoxWeave is a self-hosted control plane for AI coding, remote development environments, and an independent Web console. Containarium and Incus manage the underlying containers.

**Current source version: 0.18.2.** This is a preview, not a certified clean-host deployment.

## Install (Ubuntu 24.04+ recommended)

The desired default command is:

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash
```

**Ubuntu 24.04 or newer is the recommended host**, not a required distribution. The installer checks Linux x86_64, CPython >= 3.12, systemd, Provider/Incus availability and needed package tooling directly. On hosts without APT, prepare reviewed dependencies and Provider yourself; BoxWeave does not invoke an alternative package manager. Other distributions require separate live acceptance. It verifies source-bound release assets, provisions missing infrastructure on a clean host, **preserves an already configured Incus storage pool**, and **reuses an existing complete Containarium** without installing or upgrading it. Existing Provider identity, configuration, network and storage remain external. Partial installations are refused. See [existing Provider instructions](docs/verification/EXISTING_PROVIDER_INSTALL.md) and [two-box MCP test](docs/verification/STREAMED_INSTALL_TEST.md). The released offline wheelhouse includes CPython 3.12–3.15 x86_64 wheels; Ubuntu 26.04 requires live host acceptance. Selected Agent clients: Windows Codex and ZCode via native SSH or WSL SSH; macOS clients are not currently supported.

A successful streamed install prints the verified installed BoxWeave version and
prominently shows `boxweave client`. For a fresh host without Incus, the
managed-Provider path runs an APT dependency simulation and selects a coherent
Incus package set without the old partial `7.5.1` pins; it never upgrades
an already-installed Incus. This does not yet mean that Incus/PostgreSQL
provisioning has been moved into an independently owned Containarium installer.

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

Run `boxweave client` to print a copyable **Windows PowerShell** setup script;
the wizard's last question controls whether a conflicting *Windows BoxWeave MCP
registration* may be replaced after verified MCP communication (backup kept).
It does not manage Linux script files. You may redirect stdout to a new file,
for example `boxweave client > boxweave-setup.ps1` (shell redirection overwrites
an existing file). Review, transfer, and execute it on the Windows client,
**not in a Linux shell**. Only the actual client-key enrollment requires
administrative authorization. Finish by verifying MCP discovery and a real
read-only call; simply generating a script does not prove a connected Agent.

If an installation fails, inspect its original log/phase and repair that
specific issue before retrying. Do not relax `/home/boxweave` from mode 0700
or grant the human login broad Incus privileges just to run the CLI.
[Host CLI reference](docs/CLI.md).


## Connect an AI coding agent

For **Windows Codex using WSL SSH**, the installed server now offers a unified client generator with an interactive terminal wizard or explicit non-interactive flags:

```bash
boxweave client
boxweave client --agent codex --transport wsl --admin-user YOUR_HOST_ADMIN --no-replace > boxweave-setup.ps1
```

Transfer the generated script to Windows over an authenticated channel, review it, then run from PowerShell: `powershell.exe -NoProfile -File .\\boxweave-setup.ps1 -Distro Ubuntu`. It generates a dedicated key **inside WSL**, pins the server host key, obtains administrator approval through the existing SSH/sudo account, probes actual MCP initialize/tools/read, then safely updates the Windows ChatGPT/Codex desktop configuration at `%USERPROFILE%\\.codex\\config.toml` (or `CODEX_HOME`), without requiring any Codex CLI command on PATH. For separate administrator approval use `-NoEnroll`, perform the printed server-side enrollment command, and rerun with `-AlreadyEnrolled`. Existing Codex BoxWeave registrations require `-Replace`. The Linux MCP gateway automatically supplies its validated server-managed workload public key for new create_container requests. The client setup script no longer writes Codex AGENTS files; optional additional user-supplied public keys remain validated. The Windows-to-server SSH key is distinct and must not be reused as the workload key. No installed Provider, Incus or storage settings are changed by client generation.

The Web Client Setup page also generates public descriptors and local client enrollment scripts for selected profiles. Actual SSH credentials, server trust and Agent registration must be reviewed on the client. Generating a script does not prove live registration or model inference.

| Client platform | Agent | Transport | Support |
| --- | --- | --- | --- |
| Windows | Codex | Native OpenSSH | Selected |
| Windows | Codex | Explicit WSL SSH | Selected |
| Windows | ZCode | Native OpenSSH | Selected |
| Windows | ZCode | Explicit WSL SSH | Selected |
| macOS | Codex / ZCode | Native SSH | Not currently supported |
| Windows / macOS | Cursor / Claude Code | Any | Not currently supported |
| WSL-hosted Agent | Codex / ZCode | Agent runs inside WSL | Not currently supported |

WSL refers only to the SSH transport used by an Agent running on Windows. See [client setup](clients/README.md).

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
