# Codex adapter

**Status: implemented for Windows Codex Desktop (Native or WSL SSH transport).**

Configuration artifacts:

- `<Codex home>/config.toml`: native desktop agent runtime and `[mcp_servers.boxweave]` registration.
- `<Codex home>/AGENTS.md`: managed `BOXWEAVE` policy block.

The `clients/windows/setup.ps1` script is the current adapter entry point. It uses the current Windows user's `%USERPROFILE%\.codex` directory unless `-AgentHome` (or `CODEX_HOME`) is provided. Do not rename the two Codex-defined filenames to `boxweave.*`.

The server's Containarium MCP tool names are unchanged; only their MCP **registration namespace** is `boxweave`.
