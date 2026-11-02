# Remote MCP HTTP(S) — source integration guide

**Published stable v0.27.1 fixes a standalone installer import failure in the v0.27.0 default-managed loopback HTTP MCP rollout.** The default streamed installer targets the published stable release; publishing does not deploy it to existing hosts. Existing SSH/stdio MCP remains supported. Remote MCP is opt-in, uses a separate loopback listener, and does not install or administer Cloudflare/OpenAI Tunnel.

## Configure

The authenticated HTTP MCP server now starts automatically at `127.0.0.1:18844`
in the standard **activated** managed install. No private configuration or
Cloudflare account is required to use its local endpoint. The landing page is
`http://127.0.0.1:18844/`, while the MCP endpoint is `/mcp`.
Existing `remote_mcp` settings are preserved and an explicit `enabled:false`
prevents auto-start. The configured HTTPS `public_url` remains optional and
is never discovered from untrusted proxy headers.

For an operator-managed public HTTPS endpoint, the following is an example
of the private configuration this tool will write. **Do not edit
`runtime.json` manually while the managed service is running**:



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

The managed installer creates and enables a separate `remote-mcp` user
systemd service with restart-on-failure. Do not also run `python -m bwmini
remote-mcp` on the same port. The manual command above remains an explicit
diagnostic/developer option after stopping the managed service. On installed
systems use `boxweave http status`, `sudo boxweave http doctor` and
`sudo boxweave http restart` instead.

## Installation completion screen (v0.27.2 source)

As of published stable v0.27.2, the installer prints a human-readable HTTP MCP section **only after** a successful managed application installation. It shows the actual effective port and loopback `/mcp` URL, `boxweave http status`, `sudo boxweave http doctor`, `sudo boxweave http restart`, and Cloudflare Quick Tunnel + configure + hosted ChatGPT OAuth sequence. If a public HTTPS origin is already configured, its `/mcp` client URL is shown, explicitly without claiming external reachability. If HTTP MCP is explicitly disabled, the text explains that state and omits activation/tunnel commands. The default installed service remains loopback-only. JSON `next_steps` and SSH client onboarding are preserved.

## Fastest ChatGPT / Cloudflare Quick Tunnel setup

```sh
boxweave http status
# Install cloudflared separately using its official packages.
cloudflared tunnel --url http://127.0.0.1:18844
# Copy the HTTPS origin printed by cloudflared (omit /mcp):
sudo boxweave http configure --public-url https://YOUR-HOST.trycloudflare.com
sudo boxweave http doctor
```

Then register `https://YOUR-HOST.trycloudflare.com/mcp` with ChatGPT and
complete BoxWeave administrator OAuth approval. This is a temporary external
tunnel, not a permanent host service or a stable URL. Restarting Quick Tunnel
may change its hostname; update `--public-url` and reauthorize. To return
to local mode use `sudo boxweave http configure --local-only`. Never
copy a token, password or full OAuth redirect query into diagnostics.

**Security:** The listener always remains on `127.0.0.1`, Origin and
DNS-rebinding checks stay active, and all MCP tools require valid credentials.
Public networking is strictly opt-in, with no automatic firewall, DNS or
Cloudflare configuration.

## HTTPS choices

**Existing proxy or external Tunnel:** Map the entire HTTPS origin to the local HTTP endpoint. Preserve authorization, OAuth metadata paths, methods, bodies and streaming. Paths include `/mcp`, `/mcp/link/*`, `/.well-known/oauth-protected-resource/mcp`, `/.well-known/oauth-authorization-server`, `/register`, `/authorize` and `/token`. Do not expose the local HTTP port; do not log secret capability URL paths. A Cloudflare Tunnel can forward the same HTTPS origin but is configured outside BoxWeave.

**Opt-in Caddy:** The source provides an optional generator and runner for an already installed `/usr/bin/caddy`:

~~~sh
python -m bwmini remote-caddyfile --config /path/to/private/application.json
python -m bwmini remote-https --config /path/to/private/application.json
~~~

The generated Caddyfile preserves the application-owned `Referrer-Policy` response header rather than forcing `no-referrer`: doing so would break OAuth browser POST and callback redirects. Caddy requires DNS control, permission to bind 80/443, and an absence of port conflicts. These commands do not install Caddy, alter foreign services, apply firewall rules or enroll public TLS automatically during the BoxWeave installer. Do not run BoxWeave Python code as root to acquire low-port privileges. Separate operator service/supervision and recovery are required.

## Client connection choices

1. **OAuth:** Fill in `https://mcp.example.com/mcp`. The client discovers OAuth metadata, registers a public client (DCR), starts Authorization Code + PKCE S256, and the operator signs in with the existing BoxWeave admin password and explicitly approves its broad authority.
2. **API Token:** Generate a token in the authenticated **Remote MCP** Web tab. Use the token in the `Authorization: Bearer` header when supported by the client.
3. **Single URL credential:** Generate an opaque path URL in the Web tab. It is a bearer credential (not anonymous), may not work in every client, and can leak from third-party proxy logs. It does not contain an OAuth access token in a query parameter.

All authorized clients have the existing trusted-operator MCP gateway authority, subject to the existing optional host-wide allowlist. No new per-Agent RBAC, Box policy or custom expiry tiers are included. Tokens and links can be revoked in the Web tab. OAuth access tokens retain protocol-required expiry.

## Web local listener status

The authenticated Remote MCP Web panel checks a bounded TCP connection to `127.0.0.1:<remote_mcp.port>` when its status API is read. A TCP connection only proves that something accepted a local socket, not that the service is BoxWeave or that the HTTP MCP handshake, access token, upstream gateway, TLS or external client works. Public reachability always remains `externally_verified: false` in this local-only status API. If the socket check fails, run the reviewed `remote-mcp` service and validate the local OAuth resource metadata endpoint separately. No third-party URL is fetched by this status check.

## Browser OAuth callback compatibility

The successful `GET /authorize` page uses `Referrer-Policy: same-origin` rather than `no-referrer`, because browsers may otherwise send `Origin: null` when the password form is submitted. Only that valid, already-registered client's callback **origin** is added to the authorization page's `form-action` CSP so browsers can follow the OAuth 302 back to the client. All other pages retain `no-referrer` and `form-action 'self'`. The `POST /authorize` Origin check is not weakened; do not add `null` to `allowed_origins`.

A browser may display `access_denied` after a second click if the first, successful authorization consumed the one-time request but the callback redirect was blocked by CSP. Restart OAuth from the client instead of replaying the old form. On a temporary Cloudflare Quick Tunnel, `public_url` must match the *current* HTTPS hostname; a URL change requires updating configuration and restarting the remote MCP process.

## Agent self-information (0.24.4 source)

Both SSH/stdio and authenticated HTTP MCP initialization now include a
BoxWeave identity, descriptive metadata and safe-use instructions. The
read-only **boxweave_info** tool supports topic-specific overview,
architecture, development, capabilities, security and documentation
guidance; it does not expose host secrets, live status or credentials.
Use tools/list and Provider read tools to discover actual authorized
tools and Box state. An enabled gateway allowlist must permit
**boxweave_info** explicitly. See [MCP Agent guide](MCP_AGENT_GUIDE.md).

## Verification and exclusions

Check authorized initialize/tools/list, wrong-Origin rejection, OAuth authorization, actual HTTPS/TLS and revocation with a standalone real client before claiming external readiness. A tunnel's Healthy label is not MCP proof. A 2026-11-01 owner-operated Cloudflare Quick Tunnel and ChatGPT OAuth workflow reached real `list_containers` results (four running boxes); this does not validate a production TLS deployment, a complete Box tool workflow, or all other client apps. The v0.24.2 browser header change was verified by fixture CI; v0.24.3 adds the managed-Caddy header and local-status regression tests. Both require a separately installed release for source-to-runtime parity. Network/host installation or public port changes require separate explicit approval. For a timed-out mutating MCP tool call, completion is UNKNOWN; never automatically retry.

See [architecture](design/REMOTE_MCP_MASTER.md), [development phases](process/REMOTE_MCP_HANDOFF.md), [verification](verification/REMOTE_MCP_TEST_PLAN.md).
\n\n## D3/D4 gateway diagnostics (0.25.0)\n\nBoth transport profiles share the same classified gateway evidence journal and default-deny D4 project policy. Allowlist the new tools explicitly if the optional host tool policy is enabled. See [exact-Box operator setup](DIAGNOSTICS.md). No client is guaranteed to use automatic error diagnosis; no upstream response body is rewritten or automatically uploaded.\n