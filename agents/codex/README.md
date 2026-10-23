# Codex adapter

**Status: implemented for Windows Codex Desktop (Native or WSL SSH transport).**

Configuration artifacts:

- `<Codex home>/config.toml`: native desktop agent runtime and `[mcp_servers.boxweave]` registration.
- `<Codex home>/AGENTS.md`: managed `BOXWEAVE` policy block.

The `clients/windows/setup.ps1` script is the current adapter entry point. It uses the current Windows user's `%USERPROFILE%\.codex` directory unless `-AgentHome` (or `CODEX_HOME`) is provided. Do not rename the two Codex-defined filenames to `boxweave.*`.

The server's Containarium MCP tool names are unchanged; only their MCP **registration namespace** is `boxweave`.

The generated managed policy includes bounded startup/provisioning retries,
ambiguous-create reconciliation, service/oneshot interpretation, per-layer public
key identities and the minimum requirements for a nested MCP smoke test. See the
[shared execution instructions](../../AGENTS.md) and
[verified nested test](../../docs/TEST-BOXWEAVE-LXC.md).

Changing the generator does not immediately rewrite an existing user's global
AGENTS.md. The updated rules are emitted when the client setup is next run through
its normal enrollment workflow. Do not run enrollment solely to refresh prose
without authorization, or overwrite the user's unmanaged instructions. Markdown
backticks in the expandable PowerShell policy here-string must be escaped so
generated rules do not contain unintended control characters.
