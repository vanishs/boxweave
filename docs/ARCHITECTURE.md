# Architecture and extension points

BoxWeave is a vendor-neutral product interface around provider-specific execution services. The initial integration remains intentionally small: Containarium as the infrastructure provider and Codex Desktop as the Windows coding agent.

```text
Human request
  -> Agent adapter (Codex now; others later)
  -> Client transport (Windows Native SSH / WSL SSH now)
  -> BoxWeave restricted SSH launcher (Ubuntu)
  -> Provider MCP server (Containarium now)
  -> Provider API (loopback) -> Incus/LXC instances
              ^
              | trusted host keys from Incus + independent network check
              +-- BoxWeave systemd reconciler
```

## Stable concepts

- **Agent adapter**: maps a coding agent's configuration and policy file to BoxWeave's MCP entry point. The current `Codex` adapter edits `config.toml` and `AGENTS.md`. Additional agents should not be implemented by editing server scripts.
- **Client transport**: starts an MCP stdio process on the server. Today, an OpenSSH forced command prevents normal shell access for the dedicated client key. Windows Native and Windows WSL modes are separate transports for one client OS.
- **Infrastructure provider**: supplies create/list/connect actions. The Containarium backend is not renamed, and its own MCP tool names, metadata and Incus SSH behavior are provider-specific.
- **Host identity reconciliation**: reconciles temporary project-container addresses with the trusted Incus identity of a running container. It is not a general-purpose SSH `known_hosts` cleanup service.
- **Server identity**: the client's dedicated `known_hosts` pins the Ubuntu SSH service independently of container host keys.

## Current implementation boundaries

The source tree reserves these integration locations:

```text
clients/windows/      implemented: PowerShell with Native / WSL transports
clients/macos/        planned: macOS installer and ssh/Keychain handling
clients/linux/        planned: POSIX SSH installer
agents/codex/        implemented by clients/windows/setup.ps1
agents/<agent>/      planned: agent-specific MCP and instruction adapters
providers/containarium/ current upstream integration
providers/<backend>/  planned: alternate infrastructure providers
```

Use `-Agent` and `-Provider` selectors as explicit compatibility gates. They currently only accept `Codex` and `Containarium`: an unknown integration must fail clearly rather than silently writing a Codex-specific configuration for another client.

Suggested future provider interface (design placeholder, **not yet implemented**): `discover`, `provision`, `get`, `connect`, `destroy`, `expose`, `health`. Future agent adapter inputs: `mcp_name`, `mcp_command`, `mcp_args`, `public_box_key`, `policy_template`. Providers and agents can each evolve without renaming the BoxWeave binary, timers or configuration root.

## Configuration conventions

- Server product root: `/etc/boxweave/` with provider environment files in `/etc/boxweave/providers/`; sensitive runtime credential root: `$HOME/.config/boxweave/`.
- System launcher: `/usr/local/bin/boxweave-mcp`.
- Agent MCP registration: `boxweave`.
- User SSH transport key: `~/.ssh/boxweave` on the selected transport system.
- Existing upstream locations (`/etc/containarium/`, `containarium.service`, `user.containarium.*` metadata) belong to the provider and must retain their names.
- Windows Codex files stay at the Codex-defined paths; other agents get their own adapters and paths.

No `host`, account name, WSL distribution, LXC subnet or client home path is embedded as an installer constant. The upstream repository default is configurable via `BOXWEAVE_REPO` / `BOXWEAVE_REF`.
