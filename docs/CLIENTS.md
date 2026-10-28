# Client setup (BoxWeave 0.4.0)

BoxWeave 0.4.0 replaces the one-shot base64 setup command with a **plain,
editable enrollment script**. The script's header contains the connection
variables in clear text; the enrollment logic is downloaded from the pinned
source commit at run time. There is no embedded payload and no asset checksum:
the script is meant to be read and edited before it runs.

## Generate a script on the server

    boxweave client                          (interactive profile selection)
    boxweave client --output /tmp/boxweave-setup.sh
    boxweave client --help

Choose destination OS, agent and Windows runtime/SSH mode exactly as before.
The generator writes or prints a self-contained script (`boxweave-setup.sh`
for macOS/WSL terminals, `boxweave-setup.ps1` for Windows PowerShell). Copy it
to the client machine by any trusted channel. Exit 0 means script generation,
2 means invalid/missing input or descriptor, and 130 means cancellation.
With `--json` the generator returns the script text as JSON instead.

Explicit selection examples (same profiles as 0.3.0):

    boxweave client --os windows --agent codex --runtime native --transport native
    boxweave client --os windows --agent cursor --runtime native --transport wsl
    boxweave client --os windows --agent claude-code --runtime wsl
    boxweave client --os macos --agent codex

| Profile | Agent/configuration location | SSH | Run the script in |
| --- | --- | --- | --- |
| Windows native | Windows user profile | Windows OpenSSH | PowerShell |
| Windows bridge | Windows user profile | Selected WSL distribution | PowerShell |
| WSL CLI | WSL user home; Codex or Claude Code CLI | WSL OpenSSH | WSL shell |
| macOS native | Mac user home; any supported agent | Native OpenSSH | Terminal |

## The EDIT ME block

The generated script starts with the connection variables; edit them before
running when the detected address is not reachable from the client machine
(NAT, WSL, VPN and multi-homed hosts are the common cases):

```bash
# ===== EDIT ME: BoxWeave server connection =====
BOXWEAVE_SSH_HOST=192.168.2.233
BOXWEAVE_SSH_PORT=22
BOXWEAVE_SSH_USER=luka
# Pinned Ubuntu SSH host key (verify the fingerprint out-of-band on first use):
BOXWEAVE_SERVER_HOST_KEY=ssh-ed25519 AAAA...
# ================================================
```

`BOXWEAVE_SSH_HOST`/`PORT` must be the address **reachable from the client**;
the generated values come from the server installer and may be internal. The
script also exports `BOXWEAVE_CONTAINER_PUBLIC_KEY` — the managed
Ubuntu-to-container key that the agent later passes verbatim to
`create_container.ssh_keys`. All variables are plain text; nothing is packed
or hidden.

## Changing the server address later

Enrollment writes a managed block into the client's SSH config:

```
# BEGIN BOXWEAVE MANAGED
Host boxweave
    HostName 192.168.2.233
    Port 22
    User luka
    IdentityFile /Users/me/.ssh/boxweave
    ...
# END BOXWEAVE MANAGED
```

Every agent registration is just `ssh boxweave /usr/local/bin/boxweave-mcp`
(native Windows) or `wsl.exe -d DISTRO -e ssh boxweave ...` (bridge). **To move
or readdress the server, edit the `Host boxweave` block (HostName/Port/User)
and rerun the generated script** — no agent re-registration is needed, and the
MCP entries keep working. Rerunning the script with a changed EDIT ME host
repins the known-hosts file for the new address automatically.

## Enrollment and agent configuration

Run the script in the target runtime. It downloads the enrollment logic from
the server installation's pinned source commit, creates/reuses a dedicated
keypair only in the active SSH environment, pins the server ED25519 host key,
and enrolls the public key under the restricted MCP forced command using the
existing login (password/keyboard-interactive by default). It rejects
incomplete pairs, changed pins for the same address and unsafe paths. The
restricted transport key is not a host shell login.

Before agent configuration publication, setup verifies MCP initialize,
initialized notification, tools/list and a read-only list_containers tools/call.
No workload container, port route or background service is created. Reopen the
agent and confirm MCP recognition and guidance loading in a new session.

| Agent | MCP registration | Guidance |
| --- | --- | --- |
| Codex | Codex home config.toml; `mcp_servers.boxweave` → `ssh boxweave` (startup_timeout_sec = 30, tool_timeout_sec = 180) | Managed block in effective AGENTS.md or existing nonempty AGENTS.override.md |
| Cursor | Cursor home mcp.json; `mcpServers.boxweave` → `ssh boxweave` | Generated boxweave-user-rules.md or optional project rule |
| Claude Code | Official CLI with explicit user scope; `ssh boxweave` | Dedicated user configuration home rules/boxweave.md |
| ZCode | ZCode home cli/config.json; `mcp.servers.boxweave` → `ssh boxweave` | Managed block in ZCode home AGENTS.md |

**Cursor global rules require a manual step:** paste the generated guidance into
Settings > Rules > User Rules. BoxWeave does not edit Cursor's private database.
With `-ProjectPath` (Windows), it writes `.cursor/rules/boxweave.mdc` with
`alwaysApply: true` for that workspace.

Changed files are backed up and replaced atomically. Preserve unrelated
configuration and instructions. A conflicting MCP entry requires rerunning with
`--replace` / `-Replace`. Inline, quoted or nested Codex MCP spellings may
require manual migration instead of a potentially destructive rewrite.
Claude Code's existing entry always requires `--replace` on rerun because its
CLI owns the registry.

Failures may leave completed enrollment/file changes; there is no automatic
cross-machine rollback or public-key removal. Review the reported stage before
retrying. Never disable host-key verification to repair a failure.

## Prerequisites

Install the selected agent and OpenSSH first. Windows native/bridge setup uses
PowerShell 5.1+ and Windows OpenSSH; the bridge also needs SSH/ssh-keygen inside
its selected WSL distribution. macOS and WSL CLI setup require **Python 3.11+**,
Bash, curl and OpenSSH. Dependencies, agents, WSL distributions and package
managers are never installed automatically. Codex/Cursor configuration does not
require their CLI binaries; Claude Code registration requires its CLI in the
selected runtime.

The generator accepts no enrollment-time address overrides (`--ssh-host` and
the base64 payload are gone). The address is edited in the script itself — that
is the supported customization point, plus the managed SSH config block after
enrollment. `--bootstrap-identity PATH` (POSIX) / `-BootstrapIdentityFile PATH`
(Windows) remains available for key-only servers; the server never opens the
identity.

## Independent client diagnostics

Run the generated script with `--diagnose` (POSIX) or `-Diagnose` (Windows):

    bash boxweave-setup.sh --diagnose --timeout 30
    powershell -NoProfile -ExecutionPolicy Bypass -File boxweave-setup.ps1 -Diagnose -Timeout 30

The same env-vars input is used. The diagnostic reads the managed SSH config
block, the transport key and the host pin, then executes a controlled MCP probe.
It never creates keys, enrolls a public key, repairs configuration or changes
known_hosts. Optional `--report NEW_PATH` / `-ReportPath NEW_PATH` exports a
JSON report to a new file; existing reports are never overwritten.

Diagnosis separates connection, registration, guidance file presence and agent
GUI verification. Exit 0 means required automated checks passed, 1 means an
observed failure (including a registration mismatch), and 2 means invalid
input, a missing prerequisite or an unsupported/incomplete check. GUI
recognition and actual instruction loading remain unverified. Claude Code
registry inspection is not implemented: connection testing is available, but
registration reports unsupported and exits 2.

Connection testing executes a controlled OpenSSH profile with config hooks
disabled, batch authentication, existing pinned trust and a single selected
identity. It never executes a command obtained from an agent configuration.
SSH/network access can produce ordinary server access logs. The MCP probe
supports the 2024-11-05 revision, bounded pagination, response validation and a
read-only list_containers call; reports omit inventory, raw provider errors,
SSH stderr and home paths.

The `--timeout` / `-Timeout` option bounds protocol pipe reads and writes
(default 30 seconds, range 1-300). Cleanup can add up to four seconds; Windows
WSL preflight has a separate ten-second read budget plus bounded cleanup.

## GLM through Claude Code

Generate with an explicit provider:

    boxweave client --os macos --agent claude-code --claude-provider zhipu
    boxweave client --os windows --agent claude-code --runtime native --transport native --claude-provider zai

The provider/model names are written into the generated script's
`BOXWEAVE_CLAUDE_PROVIDER` / `BOXWEAVE_GLM_MODEL` variables — edit them there if
needed. The key itself is never embedded: set `GLM_API_KEY` in the client
runtime environment before running the script (PowerShell for native Windows,
or the Mac/WSL terminal for POSIX setup):

```powershell
$secret = Read-Host 'GLM API key' -AsSecureString
$env:GLM_API_KEY = [System.Net.NetworkCredential]::new('', $secret).Password
# Run the generated BoxWeave setup script here.
Remove-Item Env:GLM_API_KEY
```

```bash
read -r -s -p 'GLM API key: ' GLM_API_KEY; printf '\n'
export GLM_API_KEY
# Run the generated BoxWeave setup script here.
unset GLM_API_KEY
```

Use `zhipu` for `https://open.bigmodel.cn/api/anthropic`, or `zai` for
`https://api.z.ai/api/anthropic`, with a Coding Plan key for the selected
platform. Setup merges `env` into Claude's `settings.json` under
`CLAUDE_CONFIG_DIR` or `~/.claude`, preserving other settings and backing up
existing files. The settings and backups contain the API key; keep them private.
Pass `--glm-model MODEL` at generation time (or edit the script variable) to
set `ANTHROPIC_MODEL` and all three Opus/Sonnet/Haiku mappings explicitly.
GLM settings are rejected for diagnostics and other agents. To revert, restore
the settings backup; the BoxWeave MCP registration remains independently managed.

Configuration references:
[Z.ai Claude Code guide](https://docs.z.ai/devpack/tool/claude),
[Zhipu Claude Code guide](https://docs.bigmodel.cn/cn/coding-plan/tool/claude).

## ZCode

Select ZCode in the generator or pass `--agent zcode`:

    boxweave client --os windows --agent zcode --runtime native --transport native
    boxweave client --os windows --agent zcode --runtime native --transport wsl
    boxweave client --os macos --agent zcode

The adapter targets the native ZCode desktop application. WSL is supported as an
SSH bridge for Windows; a ZCode runtime inside WSL is not supported. Install
ZCode and complete its model sign-in in ZCode itself. Claude Code's
`--claude-provider` and `--glm-model` options do not apply.

Setup merges the BoxWeave stdio server into `~/.zcode/cli/config.json` under
`mcp.servers` and adds the managed remote-development policy to
`~/.zcode/AGENTS.md`. Other settings, MCP servers and user instructions are
retained; existing files are backed up. Conflicting, disabled or customized
BoxWeave entries require explicit `--replace`; globally disabled MCP must be
enabled in ZCode before setup.

ZCode prefers native `.zcode` MCP configuration over `.agents/mcp.json` within
each scope. Adding this native configuration can therefore stop loading
fallback `.agents` servers. Review and import those servers in ZCode's MCP
settings if you rely on them; BoxWeave does not migrate unrelated registrations.

Reopen ZCode, check Settings > MCP Servers for an enabled BoxWeave entry, then
start a new task and request a read-only container listing. Check that the task
loads the global remote-development policy. The independent diagnostic checks
the native registration, guidance file and authenticated MCP protocol separately;
it does not prove that the desktop UI loaded the configuration or that model
authentication works.

Official contracts: [MCP configuration](https://zcode.z.ai/en/docs/mcp-services)
and [global instructions](https://zcode.z.ai/en/docs/agents).

## Installation and validation boundary

Installation publishes the root-owned public `/etc/boxweave/client.json`
descriptor (0644, schema 2) atomically; privileged `config.json` remains 0600.
Public fields are whitelisted: endpoint/user, full public keys, launcher, source
repository and commit, and the descriptor schema version. The generator
downloads enrollment logic from the pinned source commit at run time — an
older generated script keeps fetching that older commit's logic, so regenerate
scripts after updating the server. Editing a checkout does not update an
installed server.

Unit/installer sandbox tests cover script generation, merges, conflicts,
idempotency and mock MCP negotiation. PowerShell parsing is a syntax check.
These do not prove actual Windows/macOS enrollment, GUI recognition or guidance
loading; record real per-profile client results separately.

## PowerShell to WSL quoting regression

The Windows bridge encodes each shell script as UTF-8 Base64 and normalizes its
line endings before invoking WSL. The native command receives one single-line
shell wrapper with validated ASCII arguments, avoiding embedded double quotes
and multiline source in legacy PowerShell argument serialization. Key
preparation, home lookup, host-key pinning and the ssh_config managed block use
the same helper. Host-key mismatch checks remain enabled; no existing trust
entry is removed to repair setup.

The MCP probe also builds a native process command line. Do not quote every
argument: WSL can treat quoted switch tokens such as `"-d"` and `"-e"` as shell
text and exit 127 before SSH starts. The shared process argument encoder leaves
ordinary tokens bare and quotes only empty values or values containing
whitespace or quote characters.

When atomically replacing an existing Windows configuration file, the backup
argument to `File.Replace` must be a genuine .NET null string. Windows
PowerShell 5.1 can coerce ordinary `$null` to an empty string for this API,
producing an invalid-path exception. The writer uses `NullString.Value`;
caller-created backups remain separate and replacement remains atomic.
