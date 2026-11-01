# Remote MCP HTTP(S) / OAuth verification and acceptance plan

**Status: PLANNED TESTS; ZERO NEW TESTS EXECUTED BY THIS DOCUMENT.** Owner decisions: [master](../design/REMOTE_MCP_MASTER.md); [ADR-009](../design/ADR/0009-remote-mcp-transports-and-auth.md). This document defines the acceptance matrix for REMOTE-R0..R4 and AC-021..024. Real public hostname/TLS/Provider/Box/SSH/client actions require separately granted environment authority.

## Evidence labels

**S** reviewed source, **T** executed local fixture/CI, **I** actual installed standalone process, **E** independently observed external HTTPS/client, **L** actual authorized Provider/Box behavior, **U** unverified. No status may be upgraded by assuming connectivity, reading docs or seeing a successful HTTP 200 alone. Always record source SHA, installed VERSION, actual runtime, client/version/account, origin, target, exact command, exit/result and secret-safe proof location.

## Gate matrix

| Gate / AC | Normal positive path | Required negative/failure path | Evidence to claim done |
| --- | --- | --- | --- |
| R0 / AC-021 | Same existing stdio initialize/list/read and managed create rewrite on shared dispatcher; optional policy stable | Malformed JSON; fake upstream; modified launcher/identity; opt-in allowlist denies before Provider | T + approved I for installed change; SSH parity regression |
| R1 / AC-021 | MCP Inspector initialize/initialized, tools/list pagination, tools/call safe read, supported resources over loopback HTTP | Bad Accept/content type/version, JSON-RPC id, Origin, DNS rebinding, unauthorized/non-loopback bind, GET/DELETE, 202/405, session hijack, streaming disconnect | T plus independent real installed inspector; E only when remotely exposed |
| R2 OAuth / AC-022 | Metadata discovery, interactive local admin sign-in/consent, client registration, auth code+PKCE S256, Bearer request for exact resource | No approval, wrong state, fake client ID metadata, SSRF, redirect URI mismatch, PKCE verifier mismatch, replayed code, invalid issuer/audience/resource, expired/revoked token, csrf/origin abuse, login rate-limit | T + installed browser/auth + E with real OAuth client |
| R2 token / AC-022 | Generate/reveal-once API Bearer, valid init/list, revoke and deny | Wrong/partial token, replay after revoke, credential leakage, malformed Authorization, Web cookie or MCP session alone | T + I; E when published |
| R2 URL / AC-023 | High-entropy URL **path** validated on every request, copy-once, standard protocol calls from at least one compatible client, revoke and deny | No secret, guessed/changed link, leaked logs/referrer, proxy path rewrite, client rejects custom endpoint, OAuth token query denied, wrong path discovery | T + **real client** compatible evidence; otherwise U/UNSUPPORTED per client |
| R3 / AC-024 | Explicit optional managed TLS (when approved), independent existing HTTPS proxy, external OAuth metadata and streaming, optional user-managed Tunnel mapping | Occupied 443, missing hostname, invalid/expired cert, buffer/strip Authorization, hostile forwarded headers, 302 to HTTP, partial failure, stale DNS, no external reachability | T + E for each claimed deployment; host-change approval |
| R4 / AC-024 | Web creates/displays/revokes correct registration instructions and public-ready status; SSH clients still work | Missing external origin, misleading green status, unsupported client write, missing button auth/CSRF, secret exported, older configuration overwritten | T + real Web UI + supported-client runtime |
| Release / all | Exact release build/install/repair + version/provenance; installed app+MCP+SSH health and rollback | DB v11 migration failure, restart after token revocation, proxy outage, upstream timeout creates UNKNOWN, legacy-only mode persists | I + E and L only for separately authorized real Box; no blanket claims |

## Required cross-mode negative fixtures (N0–N4)

1. **Authority:** one Web admin cookie does not authorize MCP; one OAuth token cannot be an upstream Provider JWT; one SSH key is not an OAuth access token; OAuth failure never falls back to API-token/capability URL or unauthenticated behavior.
2. **URL credentials:** store hash/digest only, no exposed raw tokens or full capability paths in default app/proxy access logs, diagnostics, trace spans, screenshots or issued profiles; test revocation and restoration after restart. If third-party proxy logs cannot be controlled, warn and fail any claim of guaranteed secrecy.
3. **OAuth:** discovery and metadata cannot be forged by Host/forwarded headers; validate exact issuer/resource/audience, registered redirects, S256, state, one-time codes, client binding, expiry and revocation; metadata URL fetch must not reach loopback/link-local/private cloud metadata.
4. **HTTP:** enforce Origin validation; wrong version, stale/missing session ID if stateful, JSON/body size limits, SSE cross-client data confusion and notification 202; distinguish 401 vs 403 vs 400; check arbitrary methods not accepted.
5. **Transport continuity:** a lost HTTP/SSH response cannot auto-repeat a mutation; record operation intent/unknown and reconcile exact target under existing domain rules. Never treat disconnect as cancellation.
6. **Network:** bind local by default; empty/malformed/unknown external URL must not accidentally publish; TLS, proxy, URL path, well-known discovery, public client callback and stream are tested end-to-end.
7. **Existing user workflows:** Python setup output shebang still first byte, native/WSL SSH argv/keepalive/host key pin preserved, conflict preservation and successful existing MCP direct gateway behavior retained.
8. **Independent Web:** break optional HTTP/HTTPS proxy and confirm SSH and existing Web/worker can recover and diagnose independently; break Web login and confirm existing SSH direct gateway still respects its existing trust constraints.

## Client compatibility trials

Minimum actual observations for each advertised supported client:
- ChatGPT custom MCP URL with OAuth, observable discovered tools, safe read call; write only if the account's actual workspace permits it, with separately authorized safe target.
- One local native MCP client via Bearer header; one via capability URL **if actually supported**.
- Independent MCP Inspector protocol session via both loopback and HTTPS proxy.
- Existing installed SSH native and WSL profiles through their exact supported application versions; older Source/fixture is only T, not installed end-to-end I.
- A third-party Cloudflare Tunnel remains **externally configured**; optional compatibility demo, not a dependency or requirement to have a Cloudflare account.

## Implementation and gates

Perform tests in order R0 then R1 then R2 then R3 then R4. At each phase:
1. Run targeted local fixtures; record actual exit status.
2. Run `python scripts/check_knowledge.py` and `.venv/bin/python scripts/validate.py` in the permitted prepared non-root environment if available. No unexecuted command is reported as passing.
3. Check migration/restart/revoke using an isolated test DB and a consistently backed-up old schema, never a production DB.
4. For installed host/helper behavior, obtain separately approved reused-Box checkpoint from [Agent workflow](../process/AGENT_TEST_WORKFLOW.md); verify failure paths before writes.
5. Report PASS/PARTIAL/FAIL/UNKNOWN by gate, evidence class and actual observed target; append to [remediation verification queue](REMEDIATION_TEST_QUEUE.md) and relevant [risk register](OPEN_RISKS.md) during implementation.
6. Only advance release status after reviewed artifact hashes and specifically approved deployment. The source baseline's `PUBLISHED_RELEASE` remains unchanged by this document.

## Requirement mapping

- [REQ-021 / AC-021](../product/ACCEPTANCE.md): dual transport without SSH regression.
- [REQ-022 / AC-022](../product/ACCEPTANCE.md): interoperable local-admin OAuth and API Bearer.
- [REQ-023 / AC-023](../product/ACCEPTANCE.md): explicitly created, non-OAuth URL credential, with minimum protection and confirmed client compatibility.
- [REQ-024 / AC-024](../product/ACCEPTANCE.md): direct/public/proxy modes, independent Web onboarding and truthful readiness.

Return to [master](../design/REMOTE_MCP_MASTER.md), [handoff](../process/REMOTE_MCP_HANDOFF.md), [index](../INDEX.md).
