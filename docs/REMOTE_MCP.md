# Remote MCP HTTP(S) — source integration guide

**Source v0.24.2 (OAuth browser callback fix).** The current default published release is pinned separately in `PUBLISHED_RELEASE`; merging source does not deploy it. Existing SSH/stdio MCP remains supported. Remote MCP is opt-in, uses a separate loopback listener, and does not install or administer Cloudflare/OpenAI Tunnel.

## Configure

Add this fragment to the existing private application JSON while preserving all other existing keys:

~~~json
{
  "remote_mcp": {
    "enabled": true,
    "public_url": "https://mcp.example.com",
    "port": 18844,
    "allowed_origins": []
  }
}
~~~

This URL is illustrative. Public URL must be the correct canonical HTTPS origin; the remote and Web processes must use the same database and config. Remote listens **only** at 127.0.0.1:18844, not 0.0.0.0.

## Start the server

Under the existing non-root BoxWeave application user, in the installed Python environment:

~~~sh
python -m bwmini remote-mcp --config /path/to/private/application.json
~~~

Manage the process with a reviewed user service; the installer does not silently start it. Restart the existing Web process separately after changing its private config to display the new Remote MCP panel.

## HTTPS choices

**Existing proxy or external Tunnel:** Map the entire HTTPS origin to the local HTTP endpoint. Preserve authorization, OAuth metadata paths, methods, bodies and streaming. Paths include `/mcp`, `/mcp/link/*`, `/.well-known/oauth-protected-resource/mcp`, `/.well-known/oauth-authorization-server`, `/register`, `/authorize` and `/token`. Do not expose the local HTTP port; do not log secret capability URL paths. A Cloudflare Tunnel can forward the same HTTPS origin but is configured outside BoxWeave.

**Opt-in Caddy:** The source provides an optional generator and runner for an already installed `/usr/bin/caddy`:

~~~sh
python -m bwmini remote-caddyfile --config /path/to/private/application.json
python -m bwmini remote-https --config /path/to/private/application.json
~~~

Caddy requires DNS control, permission to bind 80/443, and an absence of port conflicts. These commands do not install Caddy, alter foreign services, apply firewall rules or enroll public TLS automatically during the BoxWeave installer. Do not run BoxWeave Python code as root to acquire low-port privileges. Separate operator service/supervision and recovery are required.

## Client connection choices

1. **OAuth:** Fill in `https://mcp.example.com/mcp`. The client discovers OAuth metadata, registers a public client (DCR), starts Authorization Code + PKCE S256, and the operator signs in with the existing BoxWeave admin password and explicitly approves its broad authority.
2. **API Token:** Generate a token in the authenticated **Remote MCP** Web tab. Use the token in the `Authorization: Bearer` header when supported by the client.
3. **Single URL credential:** Generate an opaque path URL in the Web tab. It is a bearer credential (not anonymous), may not work in every client, and can leak from third-party proxy logs. It does not contain an OAuth access token in a query parameter.

All authorized clients have the existing trusted-operator MCP gateway authority, subject to the existing optional host-wide allowlist. No new per-Agent RBAC, Box policy or custom expiry tiers are included. Tokens and links can be revoked in the Web tab. OAuth access tokens retain protocol-required expiry.

## Browser OAuth callback compatibility

The successful `GET /authorize` page uses `Referrer-Policy: same-origin` rather than `no-referrer`, because browsers may otherwise send `Origin: null` when the password form is submitted. Only that valid, already-registered client's callback **origin** is added to the authorization page's `form-action` CSP so browsers can follow the OAuth 302 back to the client. All other pages retain `no-referrer` and `form-action 'self'`. The `POST /authorize` Origin check is not weakened; do not add `null` to `allowed_origins`.

A browser may display `access_denied` after a second click if the first, successful authorization consumed the one-time request but the callback redirect was blocked by CSP. Restart OAuth from the client instead of replaying the old form. On a temporary Cloudflare Quick Tunnel, `public_url` must match the *current* HTTPS hostname; a URL change requires updating configuration and restarting the remote MCP process.

## Verification and exclusions

Check authorized initialize/tools/list, wrong-Origin rejection, OAuth authorization, actual HTTPS/TLS and revocation with a standalone real client before claiming external readiness. A tunnel's Healthy label is not MCP proof. A 2026-11-01 owner-operated Cloudflare Quick Tunnel and ChatGPT OAuth workflow reached real `list_containers` results (four running boxes); this does not validate a production TLS deployment, a complete Box tool workflow, or all other client apps. The v0.24.2 browser header change is subject to its own CI fixture tests and requires a separately installed release for source-to-runtime parity. Network/host installation or public port changes require separate explicit approval. For a timed-out mutating MCP tool call, completion is UNKNOWN; never automatically retry.

See [architecture](design/REMOTE_MCP_MASTER.md), [development phases](process/REMOTE_MCP_HANDOFF.md), [verification](verification/REMOTE_MCP_TEST_PLAN.md).
