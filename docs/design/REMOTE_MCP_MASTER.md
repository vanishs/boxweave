# Remote MCP over HTTP(S): master design and implementation decisions

**Status: APPROVED DESIGN ONLY — NOT IMPLEMENTED.** Owner decisions recorded in the eight-question design discussion on 2026-10-31. This document is the single design parent; implementation/acceptance evidence lives in the linked phase and test documents. No deployment, public listener, DNS, proxy, key, token, firewall or host change is authorized by these documents.

## Product decision

BoxWeave remains a **single trusted operator managing personal AI development environments**, not a multi-tenant cloud or team RBAC product. Add standards-based **Streamable HTTP MCP** alongside existing SSH/stdio. HTTP access may use local HTTP through a trusted proxy, public HTTPS with optional self-managed TLS, or an externally operated tunnel; no OpenAI/Cloudflare-specific tunnel integration is built into BoxWeave.

Decisions (Q1-Q8):
1. Both direct HTTPS and externally operated tunnels must work through the **same** HTTP MCP endpoint.
2. OAuth authorization reuses the existing local administrator password/session for the **human approval step**; it does not export the Web cookie to MCP clients.
3. **No new per-Agent/per-tool/per-Box authorization tiers** in this feature. Preserve the current single-trusted-operator authority and existing optional SSH gateway tool allowlist; do **not** claim least privilege. Future SSH+HTTP unified identity/policy is a separate initiative.
4. Support both an **opt-in automated HTTPS deployment** and user-operated reverse proxies, with no unapproved modifications to existing listeners, routing or certificates.
5. No bundled OpenAI Secure MCP Tunnel, Cloudflare Tunnel or tunnel API. Standard public URL configuration and proxy/tunnel interoperability only.
6. Three HTTP connection experiences: OAuth Authorization Code + PKCE, static API Bearer Token, and a **credential-bearing connection URL** for one-field client registration.
7. Do not impose network-location restrictions on the administrator's explicit deployment choice. No public exposure by default; deliberate public availability may be configured.
8. Do not design independent expiry/per-Agent permission tiers for credential-bearing URLs now. Preserve the operator's SSH-like trusted-client posture; their lifecycle/authorization hardening is postponed **together** with SSH. Basic credential validation, revocation/rotation, redaction and TLS remain mandatory to avoid an actually anonymous public endpoint.

## Current code reality (main reviewed 2026-10-31)

- `bwmini/mcp_gate.py`: SSH-forced-command/stdio gateway relays the upstream Platform MCP; operator-owned, optional host-wide allowlist in `/etc/boxweave/mcp-tool-policy.json`. Missing file retains broad legacy forwarding; not per-client RBAC.
- `bwmini/mcp.py`: bounded **client** of a configured stdio launcher, not a remotely addressable Streamable HTTP server.
- `bwmini/web.py`: Flask Web with one local admin password, cookie session, Origin and CSRF guards; the authenticated Web API is typed management, **not** a generic `tools/call` server.
- `bwmini/store.py`: SQLite schema v11, single admin and sessions without a user ID; retain existing records and migrations.
- `bwmini/client_profiles.py`: Python 3.11+ generated native/WSL SSH enrollment; do not replace it or break existing profiles.
- `bwmini/online.py` and `bwmini/bootstrap.py`: installation and loopback-service boundaries. Source `VERSION` is 0.23.0, while `PUBLISHED_RELEASE` is 0.22.0 at the reviewed baseline. A source change is not a shipped default.
- Existing M0-M5 merge is **source/fixture**, not a live-box safety or functionality certification: [M0-M5 checkpoint](../process/MCP_M0_M5_SOURCE_PROGRESS.md).

## Target topology and trust boundaries

```text
                   Optional public HTTPS entry
  Agent client ────> [Caddy managed opt-in | existing reverse proxy | external tunnel]
      |                                          |
      | OAuth / API Bearer / capability URL      v
      +--------------------------------> local Streamable HTTP MCP listener
                                                |
                                     authenticated trusted-operator dispatch
                                                |
                              shared reviewed MCP gateway behavior/Provider
                                                |
                                           Containarium -> Box

  Legacy Agent ── SSH with forced command ──> stdio MCP gateway (unchanged)
  Web browser ── cookie + CSRF ─────────────> typed management Web (unchanged)
```

**No privilege merging by assumption.** A common dispatcher can reuse safe MCP protocol/tool translation, but SSH account, Web session, OAuth token, API token, URL credential and upstream Provider credential are distinct trust principals. Never accept a caller-supplied identity, launcher path, Provider token, host, selected Box owner or upstream scope override as authority. No new arbitrary Web API tool executor.

## Interface contract

- HTTP endpoint: `/mcp` (single Streamable HTTP endpoint; correctly support POST, GET and optionally DELETE per negotiated protocol), plus standards-based OAuth metadata/authorize/token endpoints on an authenticated public origin.
- Loopback bind by default. External base URL, scheme, host, paths and trusted-proxy origin handling are explicit and validated; no deriving externally visible OAuth redirects from untrusted `Host` or forwarding headers.
- HTTP messages, initialization, tool/resource discovery and calls must preserve existing gateway semantics to the extent compatible; no unsupported capabilities advertised; full MCP transport headers, JSON/SSE negotiation, Origin validation, bounded reads, cancellation/disconnection semantics and well-defined session behavior.
- OAuth is standard HTTP MCP authorization (Authorization Code + PKCE S256, resource metadata discovery, authorization-server metadata, registered redirect validation, resource-bound access tokens). Support interoperable client registration routes verified against real clients; no implicit password-grant or public anonymous client approval.
- API Token uses `Authorization: Bearer` and is generated/rotated/revoked by the authenticated admin. URL credential is a **separate opaque capability-path credential**, **not an OAuth access token in a query parameter**. Availability to a named client is conditional on protocol-level interoperability testing; do not silently rewrite or embed Bearer tokens in query strings.
- An OAuth access token still follows required protocol validity/expiry checks. Q8 defers **additional operator-configurable lifetime tiers** for API tokens/URL credentials; it does not waive OAuth security requirements.
- One-field URL sharing means possession is authority. Do not expose the secret in metadata, redirects, status pages, profile exports, error bodies, application logs, referrers or health checks. Minimum secret handling is transport safety, **not** the deferred role/RBAC project.

## Non-goals / exclusions

No team accounts, shared-Box policy, OAuth identity-provider federation, automatic provider/Incus upgrades, embedded tunnel clients, Cloudflare Access control plane, new broad tool permissions, implicit HTTP fallback to unauthenticated operation, auto-conversion of SSH enrollment, auto-generated public DNS, blanket raw shell gateway, or unreviewed production changes. HTTP and SSH do not become equivalently authorized merely because both use the same upstream launcher.

## Design constraints, compatibility and unresolved checks

- Preserve existing SSH forced-command and public-key/host-pin workflow byte-for-byte unless a separately approved regression fix is required.
- Default public listener disabled; deploying public HTTPS is explicit owner action. Never silently start a tunnel, expose a firewall port, replace an existing reverse proxy or change the production certificate.
- Provider writes retain current unknown/one-use semantics where already implemented. A lost HTTP response **does not imply cancellation** and must never trigger automatic replay.
- API token and capability URL are operator-equivalent high-privilege credentials; present this risk in Web before creation. Standard OAuth still requires a visible human consent step.
- Credential-bearing path may be rejected, normalized or stripped by specific MCP clients or proxies. The phase test gate must prove real behavior; if unavailable, mark **UNSUPPORTED for that client**, not “MCP OAuth with no authentication.”
- Scope/deployment state is **design-approved, implementation pending**. Public listener and actual Box/provider testing require separately scoped environment authorization.

## Work packages (complete links and backlinks)

| Phase | Specification | Deliverable / evidence gate |
| --- | --- | --- |
| R0 | [Protocol/authority inventory](remote-mcp/R0_PROTOCOL_AND_DISPATCH.md) | Shared callable gateway contract + SSH baseline snapshots |
| R1 | [HTTP MCP transport](remote-mcp/R1_STREAMABLE_HTTP.md) | Loopback transport and protocol fixture/negative tests |
| R2 | [OAuth and other credentials](remote-mcp/R2_AUTH_AND_CREDENTIALS.md) | OAuth metadata/PKCE + API Bearer + capability URL with negative security tests |
| R3 | [HTTPS deployment and proxy](remote-mcp/R3_HTTPS_AND_PROXY.md) | Opt-in managed HTTPS, manual proxy/tunnel compatibility, no implicit publication |
| R4 | [Web configuration and migration](remote-mcp/R4_WEB_CLIENT_EXPERIENCE.md) | One-URL enrollment UX, credential management, SSH preservation and diagnostics |

Normative outcomes: [REQ-021–024](../product/REQUIREMENTS.md), [AC-021–024](../product/ACCEPTANCE.md). Design decision: [ADR-009](ADR/0009-remote-mcp-transports-and-auth.md). Consolidated [test/acceptance matrix](../verification/REMOTE_MCP_TEST_PLAN.md) and [Agent execution handoff](../process/REMOTE_MCP_HANDOFF.md). Existing [security trust inventory](MCP_INTEGRATION_SECURITY.md) and [architecture authority](ARCHITECTURE.md) remain authoritative for running code until these phases are verified.

## Standards / sources reviewed

- [MCP 2025-11-25 transport](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports)
- [MCP 2025-11-25 authorization](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)
- [OpenAI MCP public endpoint / HTTP testing guidance](https://developers.openai.com/plugins/build/mcp-server)
- [Cloudflare published-hostname-to-local-service mapping](https://developers.cloudflare.com/tunnel/get-started/)

Return to [knowledge index](../INDEX.md).
