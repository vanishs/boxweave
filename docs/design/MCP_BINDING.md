# Local-only HTTP MCP and operator-controlled external access

**Approved default:** BoxWeave's Streamable HTTP MCP listener binds **`127.0.0.1`** by default. It is not published on `0.0.0.0`, a container bridge address, the public interface, or a public Caddy site by the installer.

## Separate control and application traffic

- **MCP control plane:** a Go service using the official MCP Go SDK on loopback HTTP. Example `http://127.0.0.1:<configured-port>/mcp`. A locally authorized MCP client can connect without TLS on loopback; authentication and ownership restrictions still apply.
- **Public control-plane access, only if operator configures it:** an operator may separately run `cloudflared`/Cloudflare Tunnel or a trusted reverse proxy mapping external HTTPS to the local MCP loopback endpoint. BoxWeave does not provision, expose, start, control or require the tunnel. Do not share parent-layer credentials or change parent networking for a child.
- **Box application traffic:** dedicated BoxWeave-owned Caddy handles domain/TLS routing **to verified owned Boxes** and Incus-native port forwarding handles owned TCP/UDP. This is not the MCP transport unless an operator independently chooses a safe proxy arrangement outside BoxWeave's normal managed-site API.

## Trust and deployment constraints

- Every authenticated tool request receives safe request/Tracking correlation, including errors and read calls where protocol-appropriate. MCP request cancellation does not automatically cancel an already admitted durable native operation.
- Even when the transport is HTTP on loopback, MCP requires a caller credential. Authentication, authorization, token revocation, resource ownership, request size controls and structured denials are enforced by BoxWeave, not assumed from Cloudflare.
- Treat proxy-exposed MCP as an Internet-reachable administrator surface, even if no local interface listens publicly. Enforce secure bearer token storage and meaningful rate limits; never log tokens, allow anonymous public tools or bypass the ownership boundary.
- Honor supported MCP transport and Origin validation requirements without trusting arbitrary `Host`, `Forwarded`, `X-Forwarded-*` or Origin values as authorization. Any trusted-proxy configuration must be explicit and tested with the chosen client/tunnel.
- Do not require a production/public DNS name or auto-generate certificates for the local MCP server.
- Public Cloudflare Tunnel usage and other optional proxy operations are separate operator-owned setup, not BoxWeave-owned routes/forwards and not an MCP-exposed host administration action.

## Verification

Verify default listener is only `127.0.0.1`; there is no unsolicited local-network/public management access. Validate authenticated local MCP `initialize`, `tools/list` and `tools/call`. Test via operator-configured HTTPS proxy/tunnel separately, including Streamable HTTP behavior, invalid credentials, Origin handling, disconnects and read-only state polling. If a proxy is not configured, remote MCP access is intentionally unavailable.

Cloudflare local tunnel origin example: [official Cloudflare Tunnel docs](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/create-local-tunnel/).
