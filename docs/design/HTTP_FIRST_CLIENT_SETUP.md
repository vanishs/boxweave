# HTTP-first unified MCP client setup (v0.28.0)

Status: source v0.28.0 implemented; PR fixture CI passed and isolated Cursor/Codex/OpenCode generated-script checks passed. Real Agent App OAuth and installed-release acceptance remain UNVERIFIED. Parent: [HTTP MCP quick-start](HTTP_MCP_QUICKSTART.md); [client compatibility](../../clients/README.md); [HTTP MCP verification](../verification/REMOTE_MCP_TEST_PLAN.md).

## Intent and boundaries

- HFC-01: `boxweave client` on a full interactive terminal prompts **HTTP first**, then SSH (advanced). `--method http|ssh` selects explicitly. In noninteractive execution, an explicit `--method` is required for a new HTTP request; any legacy SSH transport or admin-user flags retain their original SSH behavior. Ambiguous conflicting HTTP+SSH flags fail before output. All existing `--app ... --os ... --transport ... --admin-user ... --replace|--no-replace` workflows must emit the same SSH Python source, modulo the source release version. SSH key enrollment and host pinning are unchanged.
- HFC-02: Every client generator output remains a standalone **Python 3.11+ source program**, starting on byte 0 with `#!/usr/bin/env python3`, with prompts/status on stderr as `#` comments and no user input concatenated into code. The setup script runs on the chosen CLIENT, never on the BoxWeave host; no installer tunnel or root privileges. Keep `--output` new-file-only and stdout code-only.
- HFC-03: HTTP transport accepts one canonical `https://host[:port]/mcp` endpoint with no credentials, query, fragment or unusual Unicode/whitespace. On the server, use operator-supplied `--url` or a bounded **localhost-only** OAuth protected-resource metadata read to find its public resource. Never infer a public URL from Host/X-Forwarded-* or accept local HTTP as a hosted external URL. Missing public origin must fail with an actionable `HTTP_PUBLIC_URL_REQUIRED` and guide `sudo boxweave http configure --public-url`. Never verify external reachability by virtue of metadata discovery.
- HFC-04: Generated client script configures **only the named `boxweave` MCP entry** in the selected App’s user config. Existing unrelated services, settings and supported comments survive. If the entry differs, default refuse; `--replace` explicitly authorizes a scoped backup and change. Exact matching is idempotent. Refuse symlinked/ambiguous files, malformed JSON/JSONC/TOML, invalid nesting and unsupported client layout. Backups must be private and read/write atomic.
- HFC-05: The client **Agent App**, not the installer, performs OAuth login, consent and final MCP initialize/tools/read verification. Report `configured / awaiting_authorization`, never claim a live client connection from a generated config. Print clear App-specific steps (OAuth/verify) and a warning that all BoxWeave tools have administrator-level authority.
- HFC-06: Supported configuration catalog in alphabetical order and in sync with source. HTTP support is current documented schema, with manual imports where safe file ownership/path are unavailable. Unsupported OAuth in a client must not be represented as verified.
- HFC-07: Include runtime URL validation, negative injection/symlink/replace tests, seven JSON/TOML profile fixtures, manual UI import and SSH regression. Packaging of root-owned public client helper code remains byte-verified and root-only. Do not touch live user client files or private credentials in repository CI.

## Client transport catalog and source documentation (2026-11-01)

| App | HTTP registration | Automation scope | App-owned OAuth |
| --- | --- | --- | --- |
| Antigravity | `~/.gemini/config/mcp_config.json` → `mcpServers.boxweave.serverUrl` | Automatic scoped JSON merge | DCR reported; live not verified |
| Claude (Claude Code) | `~/.claude.json` → `mcpServers.boxweave = {type:'http',url:...}` | Automatic scoped JSON merge; *not* Claude Desktop | Client verification pending |
| Codex | `~/.codex/config.toml` → `[mcp_servers.boxweave] url` | Automatic scoped TOML section rewrite; no global AGENTS edits | `codex mcp login boxweave`; live pending |
| Copilot (Copilot CLI) | `~/.copilot/mcp-config.json` → `mcpServers.boxweave` | Automatic scoped JSON merge | Client verification pending |
| Cursor | `~/.cursor/mcp.json` → `mcpServers.boxweave.url` | Automatic scoped JSON merge | Client verification pending |
| OpenCode | `~/.config/opencode/opencode.json` → existing `mcp.servers` (v2) or `mcp` (v1) | Automatic scoped JSON merge without rewriting other format | `opencode mcp auth boxweave`; live pending |
| Qoder | IDE MCP Settings → JSON import | Emit JSON for explicit user import; do NOT guess a global config file | Client verification pending |
| TRAE | IDE MCP Settings → JSON import | Emit JSON for explicit user import; do NOT guess a global config file | Client verification pending |
| ZCode | `~/.zcode/cli/config.json` → `mcp.servers.boxweave` | Automatic scoped JSON merge | Client verification pending |

Official reference: https://antigravity.google/docs/mcp ; https://code.claude.com/docs/en/mcp ; https://developers.openai.com/learn/docs-mcp ; https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers ; https://cursor.com/docs/mcp ; https://opencode.ai/v2/docs/mcp-servers ; https://docs.qoder.com/user-guide/chat/model-context-protocol ; https://docs.trae.cn/ide_add-mcp-servers ; https://zcode.z.ai/en/docs/mcp-services .

## Tasks / acceptance

1. **HFC-A**: source documentation and catalog; CLI `--method` selector and unchanged legacy SSH cases.
2. **HFC-B**: URL discovery and validation; portable HTTP Python generator and App-specific safe registration.
3. **HFC-C**: tests for each App, three OSs, negative replace, symlink, malformed input, and no implicit OAuth success; update README, client guide and verification.
4. **HFC-D**: complete CI (knowledge, Python, browser fixtures, packaging and offline smoke), immutable release and checked default installer pointer; installed user-client OAuth tests separately labeled **UNVERIFIED** unless executed on a specific authorized client.

Return to [HTTP MCP quick-start](HTTP_MCP_QUICKSTART.md) and [verification](../verification/REMOTE_MCP_TEST_PLAN.md).
