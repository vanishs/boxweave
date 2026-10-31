# BoxWeave MCP client setup

The installed `boxweave client` generator supports the following Apps.
**Always list Apps in alphabetical order (case-insensitive)** when adding
support; this also applies to CLI selectors, Web UI and README matrices.

| App | Registration | Target |
| --- | --- | --- |
| Antigravity | JSON merge | User `~/.gemini/config/mcp_config.json` |
| Claude | JSON merge | **Claude Code** `~/.claude.json` (not Claude Desktop) |
| Codex | TOML merge | User `~/.codex/config.toml`, or `CODEX_HOME` |
| Copilot | JSON merge | **GitHub Copilot CLI** `~/.copilot/mcp-config.json`, or `COPILOT_HOME` |
| Cursor | JSON merge | User `~/.cursor/mcp.json` |
| OpenCode | JSON merge | User `~/.config/opencode/opencode.json`; supports existing v2 `mcp.servers` |
| Qoder | JSON import | Qoder IDE > Settings > MCP > My Servers > Add |
| TRAE | JSON import | TRAE IDE > Settings > MCP > Add > Manual; or project `.trae/mcp.json` |
| ZCode | JSON merge | User `~/.zcode/cli/config.json` > `mcp.servers` |

Supported client OS: **Linux**, **macOS**, **Windows**. SSH transports:
**native** on every OS, plus **WSL** only for Windows-hosted Apps.
Windows-to-WSL SSH is not a Linux-hosted Agent. The BoxWeave **server**
still requires Linux and is not installed on macOS or Windows.

## Generate on the BoxWeave Linux host

```sh
boxweave client
boxweave client --help
boxweave client --app antigravity --os linux --transport native --admin-user YOUR_HOST_ADMIN > boxweave-setup.py
boxweave client --app cursor --os macos --transport native --admin-user YOUR_HOST_ADMIN > boxweave-setup.py
boxweave client --app copilot --os windows --transport native --admin-user YOUR_HOST_ADMIN > boxweave-setup.py
boxweave client --app zcode --os windows --transport wsl --admin-user YOUR_HOST_ADMIN > boxweave-setup.py
boxweave client --app codex --os windows --transport wsl --admin-user YOUR_HOST_ADMIN > boxweave-setup.py
```

`--agent` remains an alias for `--app`. By default, the generated script
prints on stdout and prompts appear on stderr. `--output NEW_FILE` is
optional and refuses overwrites; ordinary `>` shell redirection does not.
Use `--no-replace` (default for noninteractive use) to preserve a
conflicting client registration, or `--replace` to permit a scoped
change **after** a real MCP read, with a backup.

## Run on the selected client

**One runtime for every profile:** the installed generator emits a
self-contained **Python 3.11+** client program for every App, Linux,
macOS and Windows, including **Windows Codex + WSL**. PowerShell is
no longer the output format of `boxweave client`. Run the reviewed
file with `python3 boxweave-setup.py` on Linux/macOS, or
`py -3 boxweave-setup.py` on Windows (after installing Python 3.11+
and Windows OpenSSH). Windows WSL transport additionally requires
an installed WSL distribution with Python3/OpenSSH; specify
`--distro Ubuntu` when there is more than one distro.

The generated `.py` starts **exactly** with
`#!/usr/bin/env python3` on the first line. The `# =====`
banner and all documentation comments come **after** that line, so
direct POSIX execution with executable permission still recognizes
the shebang. The wizard writes `#`-prefixed prompts and status to
stderr; stdout contains only Python source. Use shell redirection,
or `--output NEW_FILE` to safely write a new script without overwriting
an existing file. Avoid copying your own terminal input into the script.

Review the generated `HOST`/`PORT` EDIT ME values for the actual
client-reachable IP/port. Independently verify `SERVER_KEY` before
first use; never silently change the pinned host key. The setup
generates an Ed25519 transport key only on the selected client,
uses a dedicated `known_hosts` entry with strict checking, requests
authorized Linux SSH/sudo enrollment of **only the public key**, and
tests real initialize/tools/list/list_containers before App registration.
For split administration use `--no-enroll` to print the server-side
approval command, then `--already-enrolled` after approval.
All profiles now use the Python `--no-enroll`/`--already-enrolled` options.

**Long-running MCP SSH keepalive:** Every generated and legacy client
registration uses `-o ServerAliveInterval=15 -o ServerAliveCountMax=4`,
matching the archived client implementation. This reduces idle network
connection loss and detects an unresponsive SSH peer, but it does not
automatically reconnect a terminated MCP session. A server upgrade does
not silently change existing client MCP settings. To adopt the flags,
regenerate the registration with explicit `--replace`, then restart
the App.

Client-only setup never changes host Incus/Provider/sshd or Codex
`AGENTS.md`. Existing unrelated MCP servers survive scoped JSON/TOML
updates; replaced configs are backed up. Any ambiguous file/symlink or
unsupported JSONC/TOML form refuses automatic modification. Qoder/TRAE
receive verified JSON for their documented IDE import and no direct edits
to undocumented global files.

**Boundaries:** New platforms/Apps have fixture/source coverage, **not
real-client acceptance**. On Windows native, the private key must satisfy
Windows OpenSSH ACL rules. JSONC comments in OpenCode configurations need
manual handling. Real desktop/CLI MCP discovery and permissions must be
verified separately; a successful standalone SSH/MCP read is not proof
of GUI Agent tool injection.

## Reference documents

Official MCP configuration references:
[Antigravity](https://antigravity.google/docs/mcp),
[Claude Code](https://code.claude.com/docs/en/mcp),
[Codex](https://developers.openai.com/codex/mcp),
[Copilot CLI](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers),
[Cursor](https://cursor.com/docs/mcp),
[OpenCode](https://opencode.ai/docs/mcp-servers/),
[Qoder IDE](https://docs.qoder.com/user-guide/chat/model-context-protocol),
[TRAE](https://docs.trae.cn/ide_add-mcp-servers),
[ZCode](https://zcode.z.ai/en/docs/mcp-services).

Older standalone Windows enrollment utilities remain in
`clients/profile_client.py` and `clients/windows/` for compatibility,
but the installed `boxweave client` command does not emit PowerShell.
They are not automatically run or migrated by the unified generator.
