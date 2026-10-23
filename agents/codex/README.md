# Codex adapter

BoxWeave 0.2.0 configures Codex on native Windows, macOS and WSL CLI. Native Windows supports native or WSL SSH without changing desktop runtime preferences.

Artifacts: Codex home `config.toml` with `[mcp_servers.boxweave]`, and a managed block in effective global `AGENTS.md` or existing nonempty `AGENTS.override.md`. Honor `CODEX_HOME` and explicit client agent-home options. Preserve unrelated content; unusual TOML spellings fail rather than being guessed at. POSIX setup requires Python 3.11+, but not the Codex CLI. See [client setup](../../docs/CLIENTS.md).
