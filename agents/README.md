# Agent adapters

BoxWeave does not assume that every AI coding agent has the same settings file or permissions model.

Currently supported:

- `codex/`: Codex Desktop on Windows, configured using `clients/windows/setup.ps1`.

Future adapters should implement the agent's MCP registration and instruction policy generation **without changing the Ubuntu transport or the Containarium provider**. Keep agent-specific paths and syntax in the adapter, not in shared server configuration. The Windows script exposes an `-Agent` selector that currently only accepts `Codex`.
