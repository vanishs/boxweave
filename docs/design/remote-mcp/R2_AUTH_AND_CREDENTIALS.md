# R2 — OAuth, API Token and credential-bearing URL

Parent: [remote MCP master](../REMOTE_MCP_MASTER.md). Status: **DESIGN ONLY / NOT IMPLEMENTED**. Dependencies: [R0](R0_PROTOCOL_AND_DISPATCH.md), [R1](R1_STREAMABLE_HTTP.md). Next: [R3](R3_HTTPS_AND_PROXY.md).

## Decision boundary

One existing local administrator remains the only human account. SSH authorization is unchanged; no per-Agent/per-tool/per-Box RBAC and no new expiry tiers for API tokens or credential URLs. **Transport authentication is required in protected modes** and cannot be silently skipped. No implicit "None" mode with an anonymous management endpoint. An opaque capability URL is a bearer **credential**, not lack of verification.

## Credential contracts

| Mode | Registration experience | Server-side proof | Revocation |
| --- | --- | --- | --- |
| OAuth | Public `https://host/mcp`; browser login/consent | OAuth 2.1 code + PKCE S256, resource/audience-bound access token; validated each request | Disconnect/grant invalidation (plus normal OAuth token lifetime) |
| API Token | URL plus out-of-band Bearer token | Hash-verified random token in `Authorization: Bearer`; never persisted in plaintext | Revoke/rotate token |
| Credential URL | Paste one secret-bearing URL | Exact high-entropy opaque **path** capability to MCP dispatch; validated each request; no OAuth token query param | Revoke/rotate link |

Credential URL is explicitly a **non-OAuth convenience mechanism**. A sample shape for design only is `https://mcp.example.com/mcp/links/OPAQUE_SECRET`. This path is an alternative endpoint mapping to the same logical MCP handler, not a claim that every MCP client supports it. OAuth endpoints and metadata must publish the canonical resource correctly. Never place OAuth access tokens or API Bearer tokens in URI **query** strings; the MCP authorization standard prohibits OAuth access tokens there. If a client ignores custom paths, fails discovery, or rewrites the URL, report it unsupported and use OAuth/API Token. Never swap to an unguarded endpoint.

## OAuth implementation contract

1. Reuse `bwmini/web.py` administrator password validation only for the **interactive human sign-in**. Provide a separate consent screen naming the actual MCP client, target resource and broad trusted-operator authority; never silently approve based only on possession of a cookie or malicious `redirect_uri`.
2. Serve RFC 9728 protected resource metadata (including authorization server location) and RFC 8414 authorization server metadata at canonical `.well-known` locations. Include correct `WWW-Authenticate` challenges and stable public issuer/resource URLs.
3. Support Authorization Code + PKCE S256 and `state`; exact registered redirect matching, one-use short-lived codes, exact `client_id` binding, and access token issuance/verification with issuer, audience/resource, expiry and revocation.
4. Implement interoperable client registration: prefer Client ID Metadata Documents where safely supported; offer a reviewed preregistration/DCR fallback only as required by actual target clients. For remote client metadata retrieval, reject SSRF/private-network/unsafe redirects and validate document origin, identity and URI matches. Support public clients safely.
5. Scope is a **single current trusted-operator MCP access grant**, not a false user/team RBAC guarantee. Never exchange the Web session cookie for an unrestricted upstream Provider JWT; no token passthrough. If refresh tokens are offered, rotate and revoke them with bound client/grant.
6. Never confuse credential stores: Web cookie session, MCP OAuth access token, API Token, URL capability and SSH key are separately verified; none is treated as the upstream Provider launcher's credential.
7. Store only required credential/verifier digests/metadata in private owned storage. Use cryptographic randomness, constant-time comparisons where appropriate, safe database migrations and private backups. No credentials in exception traces, ordinary audit, reverse-proxy logs, referrer headers or exported setup profiles.
8. On password reset or explicit revoke, define and test intended active-grant/token invalidation; do not claim SSH keys were revoked by a Web password reset.
9. A hard failure in OAuth discovery or token validation **never** falls back to URL mode, anonymous HTTP or SSH authority.

## Important compatibility/security caveat

Credential URLs are intrinsically easier to leak through clipboard sharing, access logs, HTTP traces and history than SSH private keys. This plan does **not** add configurable credential expiry or client-specific permission profiles (owner's decision), but **must** use strong unpredictable secrets, HTTPS for public exposure, hash-at-rest, logging/referrer redaction, explicit owner creation and immediate manual revocation. These are entry-level safety properties, not a separately authorized RBAC expansion.

## Acceptance

- OAuth client discovers metadata, performs interactive login/consent and S256 code exchange; valid Bearer can initialize/list/call; wrong audience/redirect/state/code/PKCE/expired/revoked token fails.
- API Token works through standard header, cannot be retrieved again after generation, revoked/altered tokens fail.
- Credential URL works with at least one verified real client that accepts it; no other URL is a grant, no token is embedded in query, full path is redacted in normal application and proxy records.
- Cross-mode confusion fails closed. One actor's session/cookie or MCP session ID alone is not authorization. Existing SSH remains unchanged.
- Real ChatGPT Plus-specific behavior is labeled tested/untested per actual account, client version and UI; a standards fixture does not certify a service plan or feature entitlement.

See [acceptance plan](../../verification/REMOTE_MCP_TEST_PLAN.md), R2/N2; [MCP Authorization](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization).
