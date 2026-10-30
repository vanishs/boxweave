# BoxWeave

**Your agents. Your machines. Connected.**

BoxWeave is a self-hosted control plane for AI coding, remote development environments, and an independent Web console. Containarium and Incus manage the underlying containers.

**Current source version: 0.17.0.** This is a preview, not a certified clean-host deployment.

## Install on Ubuntu

The desired default command is:

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash
```

The streamed installer targets a fresh Ubuntu 24.04 x86_64 host with systemd. It verifies the matching GitHub release assets and source tag, installs pinned Incus/Containarium prerequisites only on an empty host, and prepares an independent application account and MCP endpoint. An existing Incus with a configured pool is preserved; an existing complete Containarium/Incus installation is reused rather than installed over or upgraded. BoxWeave still creates its own app account, scoped MCP launcher and managed units. An incomplete Provider or missing storage pool is refused. Existing-Provider defaults are loopback REST at `http://127.0.0.1:8080`, `/etc/containarium/jwt.secret`, and `containariumd` on PATH; use `sudo bash install.sh --provider-url URL --provider-secret-file FILE --provider-cli FILE` for nonstandard setups. The streamed release still requires Ubuntu 24.04 with Python 3.12; Ubuntu 26.04 needs a separately built/verified compatible release. Published source and fixture CI are not live-host acceptance; see [existing Provider acceptance](docs/verification/EXISTING_PROVIDER_INSTALL.md) and [two-box MCP test](docs/verification/STREAMED_INSTALL_TEST.md).

The Provider API and Web application stay on loopback; the MCP client uses the existing SSH service with an explicitly enrolled client public key and a verified server pin. No public route, firewall change or SSH host-key replacement is performed. The initial Web administrator password is saved only in a private file whose location is printed; it is never printed by the installer.
The explicit `bootstrap-plan` / `bootstrap` workflow can now prepare a dedicated application account, systemd user bus, independent scoped MCP launcher and rotating credential on an already reviewed Provider. It requires independently reviewed artifact digests and explicit root-change acknowledgment; see [application bootstrap](docs/HOST_BOOTSTRAP.md#independent-application-bootstrap).

On a prepared Ubuntu 24.04 x86_64 host with Python 3.12, reviewed Containarium/Incus, a non-root application user with a functioning systemd user bus, an authorized MCP launcher and a trusted release archive, use the existing installer:

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

## Connect an AI coding agent

The Web Client Setup page generates public descriptors and local client enrollment scripts. Actual SSH credentials, server trust and Agent registration must be reviewed on the client. Generating a script does not prove live registration or model inference.

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
