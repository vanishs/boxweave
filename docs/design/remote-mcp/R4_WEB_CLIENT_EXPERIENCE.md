# R4 — Web onboarding, connection management and compatibility

Parent: [remote MCP master](../REMOTE_MCP_MASTER.md). Status: **SOURCE IMPLEMENTED / REAL CLIENT AND BROWSER ACCEPTANCE PENDING**. Dependencies: [R1](R1_STREAMABLE_HTTP.md), [R2](R2_AUTH_AND_CREDENTIALS.md), [R3](R3_HTTPS_AND_PROXY.md).

## Goal

Make standard HTTP MCP connections meaningfully simpler than today's Python/SSH enrollment, without deleting or silently changing SSH clients or making BoxWeave a multi-user SaaS management plane.

## Proposed authenticated administrator experience

Add a **Remote MCP** section to the existing Web management area. Show:
1. **Listener**: enabled/disabled, loopback bind, current runtime/negotiated MCP transport, verified local connection state; never infer external reachability from listener status.
2. **Public endpoint**: explicitly configured external HTTPS URL and relevant OAuth metadata path, managed/manual ingress mode, independent external probe result and reason when unavailable.
3. **Connect using OAuth**: copy only canonical `https://host/mcp`, tell users to select OAuth in their client, complete BoxWeave admin login/consent.
4. **Connect using API Token**: create a high-entropy token on an authenticated, CSRF-checked request; reveal **once** and show the standard bearer-header setup for clients which support it. Existing token values remain hidden.
5. **Connect using URL credential**: create a one-field URL, reveal it once, plainly label that anyone holding the link may act as the current trusted operator; allow revoke/rotate; show compatibility warning and verified client list.
6. **Connection inventory**: safe labels/created/revoked metadata, test and revoke/rotate actions; never show secrets or assume label means a distinct permission scope.
7. **Troubleshoot**: local initialize/discovery/authorized read, public TLS/DNS/proxy health, OAuth discovery/browser/login, token/header/path routing and operator-visible failure codes. An internal successful test is not an external proof.
8. **SSH remains available**: link the original client generator unchanged; do not migrate existing profiles or touch SSH identities.

## Client compatibility rule

Do not make untested promises about individual client apps, subscription entitlements or full tool-write support. Test actual clients through **their real supported flows** and record exact version, mode and outcomes:

| Client / connection type | Expected route | Evidence requirement |
| --- | --- | --- |
| ChatGPT custom MCP URL | Streamable HTTPS + OAuth | Actual user-enabled ChatGPT test; account features, tool discovery, safe read then authorized write **if supported**, independently observed |
| Codex / Claude / Cursor / other standard clients | Streamable HTTPS + OAuth or Bearer when supported | Exact client config syntax/version and real initialize/discovery/call |
| One-field capability URL | Opaque path credential on Streamable HTTPS | Per-client observed compatibility; not assumed from generic HTTP support |
| Existing SSH client profiles | Existing Python 3.11+ / SSH stdio | All currently supported profile combinations remain unaffected |

Any “copy URL” UI must correspond to an actual tested ready endpoint. Without configured public URL, present local-only instructions and a clear “not externally reachable” state. Do not auto-generate a fake reachable public hostname.

## Update/migration constraints

- Additive SQLite migrations; preserve v11 private admin, session, plan, operation and existing data. Old binaries must reject newer schema cleanly where necessary; backup/rollback before any real migration.
- Keep user-owned services/private file permissions and reviewed publication manifest rules. No secrets in downloadable generated source, screenshot-friendly output or diagnostics.
- No implicit default credential, OAuth bypass, certificate overwrite, re-enrollment or mass disconnect. Record changes independently and make recovery visible.
- Mark docs/client setup “source complete” separately from release build, installed service and real external/browser use.

## Completion

One administrator can configure either proxy mode and connect one verified OAuth client with URL only, one Bearer client with URL + token, and one verified compatible URL-credential client with one field; can revoke them and confirm denied access. Existing SSH client setup still passes. Tests trace [AC-021–024](../../product/ACCEPTANCE.md) and [remote acceptance](../../verification/REMOTE_MCP_TEST_PLAN.md).
