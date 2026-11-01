# ADR-009 — Add remote HTTP MCP without coupling to third-party tunnels

**Status: Accepted; source implemented in v0.24.0 (PR #10), external acceptance unverified** (2026-10-31 owner Q1–Q8). The [master specification](../REMOTE_MCP_MASTER.md) owns task decomposition; existing [ADR-003](0003-transport-and-trust.md) continues to describe the implemented SSH trust path.

## Context

The existing forced-command SSH/stdio MCP path needs platform-specific Python/OpenSSH/WSL/key enrollment. Personal agents and hosted chat clients may instead accept remote Streamable HTTP MCP URLs. The current local Web admin is not an MCP OAuth provider, and existing direct SSH forwards broad upstream tools unless a protected optional allowlist is installed. Adding HTTP without aligning trust boundaries risks a new anonymous administrative interface.

## Decision

- Keep SSH/stdio functional and supported; add a **separate** Streamable HTTP inbound transport with explicitly configured public HTTPS deployment.
- Use the existing single administrator for interactive OAuth sign-in/consent, with separate OAuth client/grant/token verification. Also offer API Bearer tokens and a revocable opaque credential URL for clients that support that method.
- Treat all approved remote credentials as reaching the existing single-operator MCP authority; no extra identity RBAC/tool/Box tiers now. Record the **deferred** eventual unified SSH+HTTP identity/policy project.
- Offer optional managed HTTPS and manual reverse-proxy deployment. **Do not bundle** OpenAI Secure MCP Tunnel, Cloudflare or other provider-specific connection mechanisms.
- Do not limit where an owner explicitly exposes HTTPS MCP, but keep the service loopback/private by default and require deliberate external-publication action.
- Do not add configurable expiry tiers for URL credentials as a separate feature. Standard OAuth expiration, explicit revocation, secret hygiene and URL credential validation still apply.
- Prohibit OAuth access tokens in URI query parameters; one-field URL is a distinct high-entropy **path capability**, subject to client interoperability verification and leakage warning.

## Consequences

Advantages: shared standards-based client endpoint; less local setup; independent tunnel/proxy choices; preserve existing SSH integrations. Costs: secure OAuth AS implementation, public TLS/proxy operations, additional transport tests and high-privilege link risk. A copied URL is *not* intrinsically safe like a private SSH key; risk must be visible, bounded by minimum credential hygiene, and deferred permissions cannot be misrepresented as delivered.

## Revisit triggers

- New requirement for individual Agent/Box/tool permissions, audits or restricted public-client grants: introduce **one** server-enforced authorization system for SSH and HTTP; do not fix just HTTP.
- Real client interoperability failures with capability-path URL or OAuth client-registration/discovery.
- Observed recurring certificate/reverse-proxy maintenance needs or policy changes in MCP/OAuth specs.
- Explicit operator approval for a changed trust/deployment/security posture.

## Regression/evidence

R0–R4 source/fixture gates and separate real HTTPS/OAuth/client/SSH evidence in [REMOTE_MCP_TEST_PLAN](../../verification/REMOTE_MCP_TEST_PLAN.md). A paper design or GitHub CI does not prove public exposure safety or live Box execution.

Return to [ADR index](INDEX.md), [scope](../../product/SCOPE.md) and [master](../REMOTE_MCP_MASTER.md).
