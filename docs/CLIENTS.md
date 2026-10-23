# Client setup (BoxWeave 0.2.0)

The Ubuntu installer retains its existing workflow. Its final message now points
to `boxweave client`; it does not enroll a client or start a wizard.

## Generate a command on the server

    boxweave client
    boxweave client --help

The default command starts a numbered wizard. Choose destination OS, agent and
Windows runtime/SSH mode; paste its output into the indicated client terminal.
Explicit arguments skip corresponding questions. EOF/Ctrl-C exits 130; invalid
answers can be retried. With redirected stdin or JSON output, missing choices
fail with exit 2. Exit 0 means command generation, not successful enrollment.

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
| macOS native | Mac user home; any of the three agents | Native OpenSSH | Terminal |

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
| `--agent-home PATH` | Client Codex/Cursor configuration directory |
| `--project-path PATH` | Cursor rule in an explicitly selected local workspace |
| `--replace` | Replace a reviewed BoxWeave MCP entry |
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
