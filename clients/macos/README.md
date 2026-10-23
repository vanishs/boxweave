# macOS client

BoxWeave 0.2.0 generates native macOS setup with `boxweave client --os macos --agent codex` (or cursor/claude-code).

Implementation: `clients/posix/setup.py`. Requirements: Python 3.11+, Bash, curl and OpenSSH. It uses local keys and a pinned host identity without installing dependencies or changing shell startup files. Codex/Cursor do not require their CLI; Claude Code requires its CLI. Keychain is optional. See [client setup and validation limits](../../docs/CLIENTS.md). Sandbox coverage is not a live Mac enrollment result.
