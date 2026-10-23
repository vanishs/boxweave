# Client support expansion research

Research date: 2026-10-22 (America/Los_Angeles).

This is a proposed implementation contract, not a claim of implemented or tested
client support. The requested initial matrix is Windows/macOS with Codex,
Cursor and Claude Code. No client enrollment or remote experiment was performed
during this research. The existing Platform MCP successfully returned container
inventory. The working tree was clean before this document was added.

## User-facing command

Run `boxweave client` on the installed Ubuntu server. With a terminal attached,
start a numbered, line-oriented wizard. Avoid a terminal UI dependency: accept a
number and Enter, repeat invalid answers, and support EOF/Ctrl-C cancellation.
Select the destination client OS, agent, and supported execution mode, then print
a summary and a copyable command. Generation is read-only; copying or executing
the command on the client is a separate action.

Proposed parameter equivalents:

```bash
boxweave client --help
boxweave client --os windows --agent codex --runtime native --transport native
boxweave client --os windows --agent cursor --runtime native --transport wsl
boxweave client --os windows --agent claude-code --runtime wsl
boxweave client --os macos --agent codex
boxweave client --os macos --agent cursor
boxweave client --os macos --agent claude-code
```

Explicit options preselect wizard answers; prompt only for missing required
choices. Fully specified invocations must not prompt. With non-terminal stdin,
incomplete selections fail with exit 2 and actionable usage; do not hang, infer
the client OS from the Ubuntu server, or consume a pipeline as wizard answers.
`--help` must work without installation configuration or privileges. For JSON
output, require a complete selection and keep stdout machine-readable. For text
output, identify the shell in which the generated command must be pasted.

Proposed optional arguments include `--json`, `--config`, `--ssh-host` and
`--ssh-port`. Endpoint overrides affect the rendered command only. Never silently
persist them into server configuration. Client paths and installed WSL
distributions must be discovered on the client, not guessed by the server.

## Windows: distinguish runtime, transport and paste shell

PowerShell is an installer entry shell, not an SSH transport or agent runtime.
The existing `-Mode WSL` means a native Windows agent starts `wsl.exe` to run SSH;
it does not install or run the agent inside WSL.

| Profile | Agent/configuration location | MCP process | Paste generated command in |
| --- | --- | --- | --- |
| Native + native SSH | Windows user profile | Windows `ssh.exe` | PowerShell |
| Native + WSL SSH | Windows user profile | `wsl.exe -d DISTRO -e ssh ...` | PowerShell |
| CLI in WSL | Selected Linux user's WSL home | Linux `ssh` | WSL shell |

Propose the first two profiles for all three native clients. Offer the third for
Codex CLI and Claude Code CLI; do not label it a Cursor desktop runtime. Cursor
Remote WSL workspace behavior is a separate compatibility question and needs a
real client test before being advertised. Do not switch a desktop application's
runtime preference as a side effect of MCP setup.

For the native-to-WSL bridge, the client installer should list actual installed
distributions, require a selection when ambiguous, verify that distribution can
run SSH, and retain its exact name in the MCP arguments. Interactive local
selection can be bypassed by an explicit distribution argument. This option
belongs to the client installer; server-side command generation cannot discover
a laptop's WSL installations. Preserve the legacy `-Mode Native/WSL` interface as
aliases for the first two profiles.

Create a key only in the selected transport environment. The current Windows
script creates a Windows key even in WSL transport mode; that is unnecessary.
Use the existing-login bootstrap identity only for enrollment, separately from
the restricted ongoing MCP identity. Keep Windows and WSL private keys in their
respective environments and never copy one to the other.

## macOS profile

Use a native client and system OpenSSH. Generate a command usable from Terminal
and invoke an explicit script interpreter, independent of the user's interactive
shell. Do not change shell startup files, require Docker/WSL, install Homebrew,
or assume a Python/Node runtime is already available.

Keep the bootstrap compatible with the shipped shell tools rather than GNU-only
flags. In particular, audit `base64`, `sed`, `stat`, temporary files and shell
features instead of copying Linux commands unchanged. Discover the user's home,
restrict `.ssh` and private-key permissions, use absolute executable/key paths,
and test launching MCP from a GUI app, where PATH and ssh-agent availability may
differ from Terminal. Handle Intel and Apple Silicon without a hardcoded
Homebrew prefix. Keychain integration is optional, not a prerequisite.

The configuration-writer dependency is still an implementation decision. Prefer
official agent CLI registration for Codex/Claude Code when available, and require
a documented native fallback for desktop-only installations before claiming
those installations are supported. Cursor needs a JSON writer. Evaluate native
macOS facilities versus a clearly declared helper dependency; do not hide an
automatic runtime installation in client enrollment. This can be resolved with
local sandbox fixtures before any live enrollment.

## Agent adapters and instruction delivery

| Agent | MCP configuration | Proposed BoxWeave guidance |
| --- | --- | --- |
| Codex | `[mcp_servers.boxweave]` in Codex home `config.toml` | Managed block in effective global `AGENTS.md` or existing `AGENTS.override.md` |
| Cursor | `mcpServers.boxweave` in `~/.cursor/mcp.json` | Generate User Rules text; optionally install a project rule at an explicitly selected local workspace |
| Claude Code | Official `claude mcp add` with stdio and user scope | Dedicated `~/.claude/rules/boxweave.md` |

Respect `CODEX_HOME` and supported configuration-directory overrides. Resolve
paths in the chosen runtime environment. Do not assume that a Windows profile
and its WSL Linux user share settings.

For Codex, detect global `AGENTS.override.md`: it takes precedence over
`AGENTS.md`, so writing only the latter can leave guidance inactive. Preserve
unmanaged instructions. The existing script also unconditionally writes
`runCodexInWindowsSubsystemForLinux = false`; remove that side effect from the
general adapter design.

For Cursor, official documentation establishes global MCP JSON, project
`.cursor/rules` files and global User Rules in Settings. This research did not
establish a supported file-writing interface for global User Rules. Do not
guess `~/.cursor/rules` is globally loaded or edit Cursor's internal databases.
An optional project rule must target a local workspace explicitly selected by
the user, not a remote container path. Report global rule installation as
pending manual action when appropriate; MCP registration alone is not evidence
that the operational guidance was loaded.

For Claude Code, prefer the official CLI with explicit `--scope user`; its
default scope is local to a project. Avoid treating `settings.json` as its MCP
registry or implementing a new writer for the credential-bearing user registry.
Check existing registration before replacement, preserve unrelated entries, and
handle a conflicting BoxWeave name explicitly. The dedicated rules file avoids
rewriting a user's main `CLAUDE.md`. Honor supported directory overrides through
the CLI and verify the corresponding rule location.

Keep one shared provider policy template, with adapter-specific wrappers.
Operational guidance must include ambiguous-create reconciliation, bounded
readiness, exact managed public identity, dynamic connect, remote execution,
restricted host credential use, and explicit network exposure. Guidance is not
an enforced sandbox and must not be described as one.

## Service-side metadata and command generation

Today `install.sh` prints two Windows/Codex commands at the end. Move rendering
into a reusable CLI module and have installation reference the same generator.
The saved configuration already contains endpoint, Linux user, source repo/ref
and the managed container public-key path. The installation manifest records a
resolved source commit. Read only public host/container keys; never open the
private-key, token or JWT signing-secret files.

`/etc/boxweave/config.json` is mode 0600 and includes privileged configuration.
Do not make it world-readable for command generation. Prefer an explicitly
whitelisted, non-secret `/etc/boxweave/client.json` descriptor published atomically
by installation. It can include schema version, endpoint/user, full public keys,
launcher path and source provenance. It is root-owned; its read permissions must
deliberately permit the intended local CLI users. Do not copy the whole server
configuration into it. Old installations without the descriptor can report that
an update or privileged read-only invocation is needed rather than invent data.

Pin all downloaded client components to the same installed resolved source
commit. Validate metadata and quote PowerShell and POSIX shell arguments
independently. Download to a temporary location before execution, fail on fetch
errors, and retain access to stdin for bootstrap SSH authentication. A public-key
snapshot supplied through the trusted server session is the initial trust anchor;
do not replace it with unauthenticated key scanning.

## Enrollment, updates and verification

Preflight all local requirements before enrollment or configuration writes.
Reuse valid dedicated keys; reject incomplete pairs and changed pinned host
identities. Register only the selected public key with the existing restricted
forced command. Preserve existing configuration entries and back up files before
atomic replacement. Repeated setup must not duplicate entries or managed policy
blocks. Do not remove a public key after a later failure unless this run can
prove it added that key and rollback is explicitly part of the operation.

Separate results into prerequisites, enrollment, MCP handshake/tool discovery,
agent registration and guidance installation. A forced-command key cannot be
tested with ordinary `whoami`; initialize MCP over SSH, send initialized, call
tools/list, and use a read-only provider call for authenticated operation. Do
not create a container to verify basic client setup. Agent UI recognition and
instruction loading require an additional actual-client check.

The current `server/mcp_client.py` is POSIX-specific (`selectors`, nonblocking
pipes, process groups). It cannot be reused unchanged as a native Windows probe.
Implement bounded pipe handling and process cleanup per platform, keep protocol
stdout separate from diagnostics, and report transport timeouts separately from
process exit codes. Do not log arbitrary provider responses or configuration.

## Implementation and validation sequence

1. Add a declarative compatibility matrix and pure command renderer, then wire
   the wizard and parameter validation into `server/cli.py`.
2. Publish the public descriptor and new CLI module through the existing staged,
   atomic installer/provenance workflow. Preserve update/repair behavior.
3. Extract the Windows transport/enrollment layer and add three adapters, keeping
   existing parameter aliases and preserving agent runtime choices.
4. Add a shared POSIX bootstrap for macOS and WSL-hosted CLIs with platform-aware
   dependency/path handling; finish the desktop-only configuration fallback.
5. Add bounded MCP probing, granular results, documentation and fixture coverage.

Meaningful tests include terminal/non-terminal behavior, invalid combinations,
cancel/EOF, Windows and POSIX quoting, missing metadata, source pinning,
configuration preservation, repeat enrollment, incomplete keypairs, host-key
changes, WSL home/path separation, custom configuration homes, Codex override
precedence, malformed configuration, JSON/TOML semantics and probe cleanup.

Run Linux installer sandbox regressions as a non-root user in an authorized
project LXC. Parse PowerShell and shell changes without privileged setup.
Linux mocks cannot establish real macOS support. Real Windows native/bridge,
WSL CLI and Mac GUI/CLI checks require the corresponding machines and an agreed
live enrollment plan. No such live test was authorized or performed in this
research phase. Do not claim all matrix cells passed from shared unit tests.

## Primary sources

- [Codex MCP](https://developers.openai.com/codex/mcp)
- [Codex Windows](https://developers.openai.com/codex/windows)
- [Codex instruction discovery](https://developers.openai.com/codex/guides/agents-md)
- [Cursor MCP](https://cursor.com/docs/mcp)
- [Cursor rules](https://cursor.com/docs/rules)
- [Claude Code MCP and scopes](https://code.claude.com/docs/en/mcp)
- [Claude Code Windows setup](https://code.claude.com/docs/en/setup)
- [Claude Code user rules](https://code.claude.com/docs/en/memory)
- [Apple Terminal SSH guidance](https://support.apple.com/guide/terminal/connect-to-servers-trml1018/mac)

Repository evidence: `README.md`, `docs/CLI.md`, `docs/INSTALL-UPDATE.md`,
`docs/ARCHITECTURE.md`, `clients/windows/setup.ps1`, `clients/macos/README.md`,
`agents/README.md`, `agents/codex/README.md`, `server/cli.py`,
`server/mcp_client.py`, `install.sh` and existing tests.
