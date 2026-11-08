# Local-only HTTP MCP and operator-controlled external access

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

Authentication, token revocation, Origin/Host verification, body-size/rate limits and explicitly trusted-proxy configuration must be installed as **real HTTP middleware outside the SDK handler**; cross-origin checks cannot be assumed enabled by default. See [upstream SDK review](../research/OFFICIAL_SDK_FIT_2026-11-07.md).

## Verification

Verify default listener is only `127.0.0.1`; there is no unsolicited local-network/public management access. Validate authenticated negotiated MCP `server/discover` where supported (2026-07-28) and legacy `initialize`, then `tools/list` and `tools/call`. Record the negotiated protocol and verify both allowed and denied requests. Test via operator-configured HTTPS proxy/tunnel separately, including Streamable HTTP behavior, invalid credentials, Origin handling, disconnects and read-only state polling. If a proxy is not configured, remote MCP access is intentionally unavailable.

Cloudflare local tunnel origin example: [official Cloudflare Tunnel docs](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/create-local-tunnel/).
