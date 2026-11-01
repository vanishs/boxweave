# R3 — Public HTTPS, reverse proxies and external tunnels

Parent: [remote MCP master](../REMOTE_MCP_MASTER.md). Status: **PARTIAL SOURCE IMPLEMENTATION / NO HOST TLS TEST**. Dependencies: [R1](R1_STREAMABLE_HTTP.md), [R2](R2_AUTH_AND_CREDENTIALS.md). Next: [R4](R4_WEB_CLIENT_EXPERIENCE.md).

## Accepted deployment choices

1. BoxWeave provides a **loopback HTTP MCP origin** by default, not a publicly reachable service.
2. An operator may explicitly choose **managed HTTPS** with a reviewed, supported automatic certificate/proxy component (candidate Caddy, selection to be verified against the installed bootstrap contract). This is an optional installer feature, not a hidden package/proxy install.
3. An operator may instead connect an **existing HTTPS reverse proxy**, with a stable external base URL and documented `/mcp` path, discovery and OAuth routes.
4. Independently installed Cloudflare Tunnel, ngrok, Tailscale Funnel or other HTTPS ingress may forward to the same origin when they preserve required methods, response streaming, paths and headers. BoxWeave does **not** install or administer these products.
5. No location/region/LAN restriction is added to an explicitly chosen public endpoint; HTTPS and credential verification remain mandatory for public use.

## Installer and service contract

- Existing Web app, SSH/stdio gate and provider services continue to work independently of the new HTTP MCP listener. A failed HTTPS deployment must not disable SSH or mutate Provider/Incus.
- For managed HTTPS: preflight hostname/DNS control, listener/port ownership, installed proxy compatibility, TLS challenge ability, certificate storage and rollback. Require explicit confirmation before touching firewall, port 80/443, proxy service, host TLS material or routing. Refuse conflicting preexisting systems; no taking over another Caddy/Nginx.
- For manual proxy: publish the expected origin port, path/method matrix, accepted `Host`, `Forwarded`/`X-Forwarded-*` policy, known proxy IPs, stream buffering and timeout recommendations, discovery paths, and necessary redirects.
- Configure immutable-at-request externally visible origin (`https://mcp.example.com`), canonical MCP resource (`https://mcp.example.com/mcp`), OAuth issuer and canonical redirect targets; reconfigure with validation, never infer from arbitrary request headers or client-supplied URLs.
- Use a trusted-proxy allowlist and explicit external URL to prevent malicious forwarding-header influence. Require correct origin/TLS even if the proxy connects over private HTTP loopback.
- No cache of bearer-protected MCP responses, streaming bodies, authorization or token responses. No accidental redirect of authorization headers to off-origin targets.
- Credential-bearing URL paths must be stripped/redacted from proxy request logs. Reject unsafe proxy configurations where masking is not enforceable; documentation must warn that third-party ingress may retain path logs outside BoxWeave's control.
- Never claim Cloudflare Tunnel `Healthy` proves origin MCP or OAuth is working: only a real external initialize/authorization exchange demonstrates readiness.
- No automatic public endpoint by merely setting a URL; exposure has explicit service-ownership, listener and rollback steps.

## HTTP/TLS acceptance and rollback

- Valid TLS chain from an outside client; correct redirect, origin metadata, OAuth login, token and MCP handshake via **both** managed proxy and one manually configured proxy.
- Test a third-party tunnel as a **compatibility fixture and optional live operator-run**. Do not require Cloudflare credentials for BoxWeave CI; no tunnel-specific code.
- Negative cases: invalid `X-Forwarded-Host`, DNS mismatch, wrong certificate, proxy stripped Authorization, buffered SSE, missing discovery route, unwanted public bind, occupied port, certificate renewal failure and proxy stop/restart.
- Failed optional public setup leaves existing SSH/Web/private MCP intact; review restoration without making unknown host state appear reverted.
- Any real network/TLS/firewall/proxy mutation is a **separately approved environment operation**. Repository source implementation alone never authorizes it.

See [acceptance](../../verification/REMOTE_MCP_TEST_PLAN.md), R3/N3, and [Cloudflare routing background](https://developers.cloudflare.com/tunnel/get-started/).
