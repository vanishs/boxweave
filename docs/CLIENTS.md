# Client setup (BoxWeave 0.3.0)

The Ubuntu installer retains its existing workflow. Its final message now points
to `boxweave client`; it does not enroll a client or start a wizard.

## Generate a command on the server

    boxweave client
    boxweave client --help

The default command starts a numbered wizard. Choose destination OS, agent and
Windows runtime/SSH mode, then whether to replace an existing
BoxWeave MCP entry. Choose No to retain conflict protection or Yes to replace
the entry with backups. Paste the output into the indicated client terminal.
Trailing guidance uses shell comments (`#`) so it can be copied with the command.
Explicit arguments skip corresponding questions. EOF/Ctrl-C exits 130; invalid
answers can be retried. With redirected stdin or JSON output, missing choices
fail with exit 2. Noninteractive replacement defaults to off unless
`--replace` is supplied. JSON notes remain plain strings. Exit 0 means command generation, not successful enrollment.

    boxweave client --os windows --agent codex --runtime native --transport native
    boxweave client --os windows --agent cursor --runtime native --transport wsl
    boxweave client --os windows --agent claude-code --runtime wsl
    boxweave client --os macos --agent codex
    boxweave client --os macos --agent cursor
    boxweave client --os macos --agent claude-code --json

| Profile | Agent/configuration location | SSH | Paste into |
| --- | --- | --- | --- |
| Windows native | Windows user profile | Windows OpenSSH | PowerShell |
| Windows bridge | Windows user profile | Selected WSL distribution | PowerShell |
| WSL CLI | WSL user home; Codex or Claude Code CLI | WSL OpenSSH | WSL shell |
| macOS native | Mac user home; any supported agent | Native OpenSSH | Terminal |

macOS defaults to native runtime/SSH. WSL CLI defaults to WSL SSH; native Windows
requires an explicit transport outside the wizard. Cursor Remote WSL workspaces
and hosted/background runtimes are outside this initial contract. WSL bridge
mode does not change the agent runtime or Codex desktop runtime preference.

## Prerequisites and options

Install the selected agent and OpenSSH first. Windows native/bridge setup uses
PowerShell 5.1+ syntax and Windows OpenSSH; the bridge also needs SSH/ssh-keygen
inside its selected WSL distribution. macOS and WSL CLI setup require **Python
3.11+**, Bash, curl and OpenSSH. Python is checked before downloading setup;
dependencies, agents, WSL distributions and package managers are never installed
automatically. Codex/Cursor configuration does not require their CLI binaries.
Claude Code registration requires its CLI in the selected runtime.

Enrollment uses an existing authorized SSH login. The default is password or
keyboard-interactive authentication. For key-only servers pass an existing
identity path on the CLIENT, not the Ubuntu server:

    boxweave client --os macos --agent codex --bootstrap-identity /Users/me/.ssh/existing_login

The server only renders that path and never opens the identity. SSH uses it to
enroll the restricted public key; BoxWeave does not copy it into WSL.

| Generator option | Purpose |
| --- | --- |
| `--config PATH` | Read another public client descriptor |
| `--ssh-host`, `--ssh-port` | Render endpoint overrides without saving them |
| `--bootstrap-identity PATH` | Existing authorized client SSH identity |
| `--wsl-distro NAME` | Preselect the distribution for a Windows to WSL bridge |
| `--agent-home PATH` | Client Codex/Cursor/ZCode configuration directory |
| `--project-path PATH` | Cursor rule in an explicitly selected local workspace |
| `--replace` | Replace a reviewed BoxWeave MCP entry; skip the replacement question |
| `--json` | Machine-readable profile, paste shell, command and notes |

Honor `CODEX_HOME` or `CLAUDE_CONFIG_DIR` in the actual client runtime.
For Claude Code set its environment variable instead of `--agent-home`.
WSL distribution discovery happens on Windows; a WSL CLI command must be pasted
inside the Linux user/distribution that runs that CLI.

## Enrollment and agent configuration

The generated command downloads from the server installation's resolved source
commit and verifies SHA256 recorded during installation. It contains only public
connection metadata and guidance. Obtain it from a trusted server session:
initial host-key pinning depends on that trusted delivery.

Setup creates/reuses a dedicated key only in the active SSH environment, pins
the server ED25519 host key, and enrolls the public key under the restricted MCP
forced command using the existing login. It rejects incomplete pairs, changed
pins and unsafe paths. The restricted transport key is not a host shell login.

Before agent configuration publication, setup verifies MCP initialize,
initialized notification, tools/list and a read-only list_containers tools/call.
No workload container, port route or background service is created. Reopen the
agent and confirm MCP recognition and guidance loading in a new session.

| Agent | MCP registration | Guidance |
| --- | --- | --- |
| Codex | Codex home config.toml; mcp_servers.boxweave | Managed block in effective AGENTS.md or existing nonempty AGENTS.override.md |
| Cursor | Cursor home mcp.json; mcpServers.boxweave | Generated boxweave-user-rules.md or optional project rule |
| Claude Code | Official CLI with explicit user scope | Dedicated user configuration home rules/boxweave.md |
| ZCode | ZCode home cli/config.json; mcp.servers.boxweave | Managed block in ZCode home AGENTS.md |

**Cursor global rules require a manual step:** paste the generated guidance into
Settings > Rules > User Rules. BoxWeave does not edit Cursor's private database
or invent a global rule directory. With `--project-path`, it writes
`.cursor/rules/boxweave.mdc` with `alwaysApply: true` for that workspace.

Changed files are backed up and replaced atomically. Preserve unrelated
configuration and instructions. A conflicting MCP entry requires explicit
replacement. Inline, quoted or nested Codex MCP spellings may require manual
migration instead of a potentially destructive rewrite. Claude Code's existing
entry requires `--replace` on rerun because its CLI owns the registry.

Failures may leave completed enrollment/file changes; there is no automatic
cross-machine rollback or public-key removal. Review the reported stage before
retrying. Never disable host-key verification to repair a failure.

## Installation and validation boundary

Installation stages the generator and client source hashes before publishing
the root-owned public `/etc/boxweave/client.json` descriptor (0644).
Privileged `config.json` remains 0600. Public fields are whitelisted:
endpoint/user, full public keys, launcher, source commit and asset hashes. The
selected source ref must contain the new files before deploying this installer.
Editing a checkout does not update an installed server.

Unit/installer sandbox tests cover command generation, merges, conflicts,
idempotency and mock MCP negotiation. PowerShell parsing is a syntax check.
These do not prove actual Windows/macOS enrollment, GUI recognition or guidance
loading; record real per-profile client results separately.

## PowerShell to WSL quoting regression

The Windows bridge encodes each shell script as UTF-8 Base64 and normalizes its
line endings before invoking WSL. The native command receives one single-line shell
wrapper with validated ASCII arguments, avoiding embedded double quotes and
multiline source in legacy PowerShell argument serialization. Key preparation,
home lookup and host-key pinning use the same helper. Host-key mismatch checks
remain enabled; no existing trust entry is removed to repair setup.

`tests/test_windows_wsl.py` runs the actual PowerShell helper functions against
an isolated Linux WSL shim. With PowerShell 7.3+ available as `pwsh` (or through
`BOXWEAVE_TEST_PWSH`), it exercises Legacy and Standard native argument modes,
reproduces the original Base64 error, and checks pin preservation, key reuse,
CRLF, nested quotes and paths with spaces. It creates only temporary fixtures;
it is not a real Windows/WSL enrollment result. This group is skipped when the
required test runtime is absent.

After updating the server with the revised installer, rerun `boxweave client`
and use the newly generated command. An earlier command pins an earlier source
commit and checksum, so it continues downloading the old script even after a
fix is published. Do not bypass the checksum check or remove known_hosts entries.

The MCP probe also builds a native process command line. Do not quote every
argument: WSL can treat quoted switch tokens such as `"-d"` and `"-e"` as shell
text and exit 127 before SSH starts. The shared process argument encoder leaves
ordinary tokens bare and quotes only empty values or values containing whitespace
or quote characters. Empty key-generation passphrases remain explicit arguments.
Early MCP stream closure now includes the observed process exit status without
printing raw stderr or credentials.

On 2026-10-22, a read-only Windows PowerShell 5.1 to WSL/SSH diagnostic reproduced
exit 127 with every argument quoted. Conditional quoting passed initialize,
tools/list and a read-only provider call with exit 0. That diagnostic did not
reenroll a client, publish agent configuration or verify GUI recognition.
PowerShell regression fixtures additionally cover the probe's complete protocol
sequence, switch quoting, empty arguments and sanitized early-exit reporting.

When atomically replacing an existing Windows configuration file, the backup
argument to `File.Replace` must be a genuine .NET null string. Windows PowerShell
5.1 can coerce ordinary `$null` to an empty string for this API, producing an
invalid-path exception. The writer uses `NullString.Value`; caller-created
backups remain separate and replacement remains atomic.

`tests/windows_config.ps1` exercises the actual writer and Codex/rules helpers
with disposable files, including new/existing files, repeat writes, backup
preservation, BOM-free UTF-8 and temporary-file cleanup. Its optional `original`
case reproduces the legacy null binding error under Windows PowerShell 5.1.
No real client configuration or SSH credentials are used by these fixtures.

## Independent client diagnostics

After updating the server from a source ref containing this feature, generate a
diagnostic command instead of running enrollment again:

    boxweave client --diagnose --os windows --agent codex --runtime native --transport native
    boxweave client --diagnose --os windows --agent codex --runtime native --transport wsl --wsl-distro Ubuntu
    boxweave client --diagnose --os windows --agent codex --runtime wsl
    boxweave client --diagnose --os macos --agent cursor

The generator retains source-commit pinning and SHA256 validation. Exit 0 on the
server still means command generation only. Paste the command in the indicated
client runtime. It uses the already enrolled transport key and existing host pin;
it never creates keys, enrolls a public key, repairs configuration, changes
known_hosts, calls connect or creates workloads. Windows bridge diagnosis requires
an explicitly selected WSL distribution. Replacement/bootstrap-identity and project-overlay options
are rejected. The generated Windows command preserves the diagnostic script's
exit in $LASTEXITCODE without closing the interactive shell.

The downloaded client artifact is independently usable with a previously
trusted public payload:

    python3 setup.py --diagnose --data-base64 PAYLOAD --timeout 30
    powershell.exe -NoProfile -File setup.ps1 -Diagnose -DataBase64 PAYLOAD -Timeout 30

Use the payload from the trusted generator, not arbitrary network input. A minimal
diagnostic payload needs profile, endpoint/user/launcher and the server public key;
it does not need a container key or policy text. This uses the same Base64 JSON
encoding as setup. A stale command downloads its pinned old artifact; regenerate
after updating the server. Changing this checkout does not upgrade an installation.

Diagnosis emits one JSON report. It separates connection, registration, guidance
file presence and agent GUI verification. Exit 0 means required automated checks
passed, 1 means an observed failure, and 2 means invalid input, a missing
prerequisite or an unsupported/incomplete check. GUI recognition and actual
instruction loading remain unverified. A successful connection with a missing or
unsupported registration still exits 2; mismatched registration exits 1.

Codex and Cursor registration checks compare the supported generated connection.
Windows Codex inspection supports the generated TOML subset, not arbitrary TOML
spellings. Custom environment/settings are reported unsupported rather than
executed. Claude Code registry inspection is not implemented in this increment:
connection testing is available, but registration is incomplete and exits 2.
Project configuration overlays are not resolved. Cursor guidance remains manual.

Connection testing executes a controlled OpenSSH profile with config hooks
disabled, batch authentication, existing pinned trust and a single selected
identity. It never executes a command obtained from an agent configuration.
Custom SSH proxies/configuration hooks are outside this initial diagnostic
contract. SSH uses the existing identity through its normal credential interface;
the diagnostic never reads or exports private-key contents. SSH/network access
can produce ordinary server access logs.

The MCP probe supports the tested 2024-11-05 revision, bounded tool discovery
pagination, response validation and a read-only list_containers call. It omits
inventory, raw provider errors, SSH stderr, full commands and home paths from
reports. Generic SSH exit failures remain transport failures when a more precise
cause cannot be established.

The --timeout / -Timeout option bounds protocol pipe reads and writes (default
30 seconds, range 1-300). Cleanup can add up to four seconds; Windows WSL preflight
has a separate ten-second read budget plus bounded cleanup. This is not a timeout
for all local filesystem checks or downloads. Local subprocess cleanup does not
prove termination of every remote descendant.

Optional --report NEW_PATH / -ReportPath NEW_PATH exports JSON to a new file in
an existing directory. Existing reports are never overwritten. POSIX publication
uses a private temporary file and a no-overwrite hard link; Windows publication
sets an owner-only ACL before writing and renames the temporary file. Unsupported
filesystem/security semantics fail with REPORT_WRITE_FAILED. Report export is the
only requested persistent client-side write; generated downloads use temporary
directories that are removed afterward.

Behavior tests cover pagination, malformed results, mismatched trust, output
redaction, non-mutating preflight, missing registration, private report export,
protocol failures and blocked pipes. Native PowerShell fixtures remain separate
from real SSH/WSL/macOS client acceptance.

## GLM through Claude Code (0.3.0)

Use the existing Claude Code adapter with an explicit model provider:

```bash
boxweave client --os macos --agent claude-code --claude-provider zhipu
boxweave client --os windows --agent claude-code --runtime native --transport native --claude-provider zai
boxweave client --os windows --agent claude-code --runtime wsl --claude-provider zai
```

Use `zhipu` for `https://open.bigmodel.cn/api/anthropic`, or `zai`
for `https://api.z.ai/api/anthropic`. Use a Coding Plan key for the
selected platform. The command generator never accepts or embeds the key.

Before running the generated setup command, set `GLM_API_KEY` in the
**client runtime** environment (PowerShell for native Windows, or the
Mac/WSL terminal for POSIX setup). For example, read it without terminal echo:

```powershell
$secret = Read-Host 'GLM API key' -AsSecureString
$env:GLM_API_KEY = [System.Net.NetworkCredential]::new('', $secret).Password
# Paste the generated BoxWeave setup command here.
Remove-Item Env:GLM_API_KEY
```

```bash
read -r -s -p 'GLM API key: ' GLM_API_KEY; printf '\n'
export GLM_API_KEY
# Paste the generated BoxWeave setup command here.
unset GLM_API_KEY
```

Setup merges `env` into Claude's `settings.json` under
`CLAUDE_CONFIG_DIR` or `~/.claude`, preserving other settings and backing
up existing files. It writes the endpoint, token and API timeout, and removes
a conflicting `ANTHROPIC_API_KEY` from that settings file.
The settings and backups contain the API key; keep them private.
Existing process-level authentication/model overrides should be cleared before
launching Claude. No model identifier is forced by default: provider defaults
or existing mappings apply. Pass `--glm-model MODEL` to set
`ANTHROPIC_MODEL` and all three Opus/Sonnet/Haiku mappings explicitly.

Restart Claude Code and inspect `/status` and `/mcp`, then request a
short model response and a read-only BoxWeave container listing.
MCP enrollment checks do not establish GLM authentication, quota or model
availability. GLM settings are opt-in and are rejected for diagnostics and
other agents. To revert, restore the settings backup (or remove the GLM env
entries); the BoxWeave MCP registration remains independently managed.

Configuration references:
[Z.ai Claude Code guide](https://docs.z.ai/devpack/tool/claude),
[Zhipu Claude Code guide](https://docs.bigmodel.cn/cn/coding-plan/tool/claude).

## ZCode (0.3.0)

Select ZCode in the client wizard or pass `--agent zcode`:

    boxweave client --os windows --agent zcode --runtime native --transport native
    boxweave client --os windows --agent zcode --runtime native --transport wsl
    boxweave client --os macos --agent zcode
    boxweave client --os macos --agent zcode --diagnose

The adapter targets the native ZCode desktop application. WSL is supported as an
SSH bridge for Windows; a ZCode runtime inside WSL is not supported by this
adapter. Install ZCode and complete its model sign-in in ZCode itself.
Claude Code's `--claude-provider` and `--glm-model` options do not apply.

Setup merges the BoxWeave stdio server into `~/.zcode/cli/config.json` under
`mcp.servers` and adds the managed remote-development policy to
`~/.zcode/AGENTS.md`. `--agent-home PATH` overrides the `.zcode` root
(not its `cli` subdirectory). Other settings, MCP servers and user instructions
are retained. Existing files are backed up. Conflicting, disabled or customized
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
authentication works. Adapter tests use isolated fixture commands, not a live
ZCode application.

Official contracts: [MCP configuration](https://zcode.z.ai/en/docs/mcp-services)
and [global instructions](https://zcode.z.ai/en/docs/agents).
