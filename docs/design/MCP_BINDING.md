# Local-only HTTP MCP and operator-controlled external access

**2026-11-13 accepted S2.5 OAuth design amendment (not yet coded):** keep one canonical `https://<public-origin>/mcp` resource for both direct Bearer and OAuth-issued Bearer access. OAuth discovery/authorization/token routes live in the *same* Go HTTP server, reuse the MCP SDK middleware and unchanged `admin`/`viewer` authorization. The operator alone provides a temporary Quick Tunnel and exact local `public_origin` setting; no new external listener or extra MCP route. **No `/mcp/oauth` route.** [Implementation contract](../development/S25_EMBEDDED_OAUTH_IMPLEMENTATION.md).

**Approved default:** BoxWeave's Streamable HTTP MCP listener binds **`127.0.0.1`** by default. It is not published on `0.0.0.0`, a container bridge address, the public interface, or a public service port by the installer.

## Separate control and application traffic

- **MCP control plane:** a Go service using the official MCP Go SDK on loopback HTTP. Example `http://127.0.0.1:<configured-port>/mcp`. A locally authorized MCP client can connect without TLS on loopback; authentication and ownership restrictions still apply.
- **Public control-plane access, only if operator configures it:** an operator may separately run `cloudflared`/Cloudflare Tunnel or a trusted reverse proxy mapping external HTTPS to the local MCP loopback endpoint. BoxWeave does not provision, expose, start, control or require the tunnel. Do not share parent-layer credentials or change parent networking for a child.
- **Guest service traffic:** native Incus TCP/UDP forwarding maps authorized host bind/port to owned Box UUID + guest port. Operator-managed TLS proxy/Tunnel or DNS may use that port, but BoxWeave does not install or configure Caddy, website routes, certificates or a reverse proxy.

## Trust and deployment constraints

- Every authenticated tool request receives safe request/Tracking correlation, including errors and read calls where protocol-appropriate. MCP request cancellation does not automatically cancel an already admitted durable native operation.
- Even when the transport is HTTP on loopback, MCP requires a caller credential. Authentication, authorization, token revocation, resource ownership, request size controls and structured denials are enforced by BoxWeave, not assumed from Cloudflare.
- Treat proxy-exposed MCP as an Internet-reachable administrator surface, even if no local interface listens publicly. Enforce secure bearer token storage and meaningful rate limits; never log tokens, allow anonymous public tools or bypass the ownership boundary.
- Honor supported MCP transport and Origin validation requirements without trusting arbitrary `Host`, `Forwarded`, `X-Forwarded-*` or Origin values as authorization. Any trusted-proxy configuration must be explicit and tested with the chosen client/tunnel.
- Do not require a production/public DNS name or auto-generate certificates for the local MCP server.
- Public Cloudflare Tunnel usage and other optional proxy operations are separate operator-owned setup, not BoxWeave-owned routes/forwards and not an MCP-exposed host administration action.

## SDK-native protocol selection

Use `mcp.NewStreamableHTTPHandler` and typed `mcp.AddTool` rather than a custom JSON-RPC dispatcher, Schema engine or persistent MCP session layer. For protocol 2026-07-28, the SDK HTTP transport requires `StreamableHTTPOptions.Stateless=true`; treat that as the preferred design candidate while verifying actual ChatGPT and legacy-client negotiation. Long-running BoxWeave operations survive independently in the durable journal and are queried through `operation_get` after reconnect; do not rely on MCP transport session IDs or `Last-Event-ID` to implement business durability.

Use native `auth.RequireBearerToken` with BoxWeave's own revocation-aware `TokenVerifier`; **typed MCP tool handlers read current actor/token from `req.Extra.TokenInfo`**, not `auth.TokenInfoFromContext` (the latter is for HTTP handlers). The MCP Streamable HTTP SDK **already provides a default 4 MiB request body cap and localhost Host/DNS-rebinding checks**. Do not duplicate those primitives. Explicit Origin policy (e.g. Go `http.NewCrossOriginProtection`), rate policy and trusted reverse-proxy handling remain to be configured. Ensure the file-apply limit accounts for JSON/base64 overhead. If an operator proxy forwards external `Host` to loopback, test compatibility before any explicit protection change. See [upstream SDK review](../research/OFFICIAL_SDK_FIT_2026-11-07.md).

For the full 20-tool SDK and middleware interface mapping, see [native SDK mapping](../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md); this is source research, not deployed validation.

## Small SDK-native continuity and privacy improvements

[Tagged Go SDK v1.8.0 research](../research/MCP_GO_SDK_CONTINUITY_REVIEW_2026-11-10.md): receive bounded optional namespaced `_meta` through SDK types, attach one redacted `AddReceivingMiddleware` observation, return typed structured original Operation receipts and bounded next-read hints, classify Tools truthfully with `ToolAnnotations`, and set authenticated `cacheScope=private` via SDK `SetCacheable` where applicable (default may be public). `work_ref`/`agent_ref` are **untrusted**; authorization still comes from verified TokenInfo, never raw client metadata. SDK Progress Notifications are optional **live** UX with progress token, not a persistent heartbeat. Protocol `2026-07-28` removed ping; the SDK's legacy KeepAlive is not an operation liveness signal. Avoid MRTR reinvocation of mutating handlers without original key, remote approval, or a new messaging service. [TC negative tests](../verification/CONTINUITY_ACCEPTANCE.md).

## MCP Tasks future extension: capability-gated, not yet implemented

The [2026-07-28 official `io.modelcontextprotocol/tasks` extension](../research/MCP_TASKS_EXTENSION_SEAM_2026-11-10.md) is **not** the 2025-11-25 experimental Tasks wire format. The pinned official Go SDK v1.8.0 does not yet expose a complete server-side Tasks implementation. **V1 must not advertise this extension, fabricate `resultType:"task"`, or register raw `tasks/*` RPCs now**. Keep existing ordinary authenticated, stateless `/mcp` `tools/call` and S2 immediate durable Operation receipts for all supported/legacy clients.

If an official Go SDK later supplies Tasks, use request-scoped client capability detection and SDK-native protocol handlers, returning a Task result **only** to an explicitly declaring client. The adapter should read the same authorized domain journal rather than inventing its own queue or MCP-session persistence. Future `tasks/get/update/cancel` require authorization on every request and correct handle expiry; do not use task IDs or client-provided work tags as capabilities. Note that `tasks/update` means replying to pending client-input requests, not updating progress, and `tasks/list` does not exist. Preserve optional external proxy routing headers only when real SDK/native tests justify it. Existing clients must keep working without the extension.

## Exact authenticated Tool request boundary

The tagged MCP Go SDK v1.8.0 [official protocol guide](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/docs/protocol.md#authorization) explicitly distinguishes HTTP middleware context from MCP tool-request metadata. The Bearer verifier runs for **every HTTP request** and attaches `TokenInfo` to the MCP `CallToolRequest.Extra`. A single shared server/typed-handler registration must never cache a prior HTTP request's actor in global state. Each handler must reject absent token metadata, authorize the **current** role and owned UUID, and derive the durable operation actor from that request. Set a stable non-secret `TokenInfo.UserID` to enable SDK legacy-session principal binding; don't put the raw token there. Test separate admin/viewer tokens, revocation, parallel requests and legacy/new protocol negotiation. See [second-pass gap analysis](../research/SDK_GAP_CLOSURE_2026-11-07.md).

## Observed legacy client path and protocol failure class

[Legacy host/MCP incident](../research/LEGACY_MAIN_INCUS_MCP_INCIDENT_2026-11-10.md): the `main` OAuth UI reported successful authorization, and test-token `/mcp` catalog requests returned HTTP 200, but ChatGPT Tool refresh failed while captured requests included **`POST /` → 405**. After checking the complete client endpoint ending in `/mcp`, the owner reported recovery and a connected MCP read returned real Boxes. A root-path 405 and an optional OIDC metadata 404 must be interpreted per request path; authorization/metadata alone is not Tool readiness. The **old main OAuth implementation remains excluded as code**; the owner's new [S2.5 Go-native embedded OAuth authorization plan](../development/S25_EMBEDDED_OAUTH_IMPLEMENTATION.md) supersedes the former exclusion, preserving `https://<host>/mcp` and these regression obligations.

For v1's operator-managed external proxy/Tunnel, verify the **full** public `https://<host>/mcp` path, method, Host/Origin and SDK-native initialize/negotiation, authenticated Tool discovery and a harmless Tool read. The legacy main version allowlist was separately fixed in v0.50.5 but was **not** the observed recovery. Rely on the official MCP Go SDK; test supported legacy revisions and malformed input without duplicating an MCP parser. Use operator-local redacted method/path/status diagnostics only, never credentials or request bodies.

## Verification

Verify default listener is only `127.0.0.1`; there is no unsolicited local-network/public management access. Validate authenticated negotiated MCP `server/discover` where supported (2026-07-28) and legacy `initialize`, then `tools/list` and `tools/call`. Record the negotiated protocol and verify both allowed and denied requests. Test via operator-configured HTTPS proxy/tunnel separately, including Streamable HTTP behavior, invalid credentials, Origin handling, disconnects and read-only state polling. If a proxy is not configured, remote MCP access is intentionally unavailable.

Cloudflare local tunnel origin example: [official Cloudflare Tunnel docs](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/create-local-tunnel/).
