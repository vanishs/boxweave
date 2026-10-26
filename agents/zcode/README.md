# ZCode adapter

BoxWeave 0.3.0 supports native Windows and macOS ZCode, including Windows
with WSL SSH transport. Select `boxweave client --agent zcode` and the
appropriate OS/runtime/transport flags.

The adapter merges a stdio registration into `~/.zcode/cli/config.json`
at `mcp.servers.boxweave`, and manages a policy block in
`~/.zcode/AGENTS.md`. `--agent-home` selects a different `.zcode` root.
It preserves unrelated entries and backs up changed files. Model providers
and sign-in remain managed in ZCode.

See [setup, fallback precedence and validation limits](../../docs/CLIENTS.md#zcode-030).
