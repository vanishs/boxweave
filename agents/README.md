# Agent adapters

BoxWeave 0.3.0 provides Codex, Cursor, Claude Code and ZCode adapters through `clients/windows/setup.ps1` and `clients/posix/setup.py`. The `boxweave client` wizard selects OS, agent, runtime and transport. All adapters reuse the Ubuntu restricted launcher and Containarium provider.

Codex writes MCP TOML and effective global instructions. Cursor merges global MCP JSON and emits User Rules text or an explicitly selected project rule; global rules still require the manual Settings step. Claude Code uses its official user-scope CLI and a dedicated user rule. See [profiles, prerequisites and validation limits](../docs/CLIENTS.md).

Generated commands share the provider policy in `server/clients.py`, including ambiguous-create reconciliation, bounded readiness, exact managed public identity, remote execution and network constraints. Legacy Windows parameter calls retain their inline policy. Never embed test-specific endpoints or keys in generic adapters. Actual UI recognition and instruction loading require per-client checks.

ZCode uses native mcp.servers in .zcode/cli/config.json and a managed block in .zcode/AGENTS.md; see [the ZCode adapter](zcode/README.md).
