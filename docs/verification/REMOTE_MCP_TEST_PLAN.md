# Remote MCP HTTP(S) / OAuth verification and acceptance plan

**Status: source fixtures added in tests/test_remote_mcp.py; external/public acceptance remains UNVERIFIED.** Owner decisions: [master](../design/REMOTE_MCP_MASTER.md); [ADR-009](../design/ADR/0009-remote-mcp-transports-and-auth.md). This document defines the acceptance matrix for REMOTE-R0..R4 and AC-021..024. Real public hostname/TLS/Provider/Box/SSH/client actions require separately granted environment authority.

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

## Executed source/fixture evidence, not real Box acceptance

GitHub Actions [run 41263506822](https://github.com/vanishs/boxweave/actions/runs/41263506822) on source commit `206a946e203ea59cf9819f668baee2107969d47f`: documentation/traceability checks, Python unit and contract tests (including remote OAuth/API/link/CSRF cases), frontend typecheck/build, seven existing fixture browser workflows and candidate build **all reported success**. The publish job was skipped. Follow-up commits (documentation and Host validation) require their own final CI evidence.

## Live loopback/proxy acceptance on the reused test Box (2026-11-01, owner-approved)

**T/I** on boxweave-test-ser at 0.24.0 (main 58ab5bc after live-found fixes f98c192 and 2de4920): loopback R1 positives/negatives (401/403 origin+host/405/406/400/202 with protocol 2025-11-25), R2 OAuth PKCE+DCR full flow with replay/verifier/redirect/state negatives, API Bearer and link credentials with hash-at-rest and revocation, R3 independent local TLS proxy with preserved public Host, R4 Web issuance/revocation persistence and 8/8 enrolled SSH-client diagnose parity. Two release blockers were found live and fixed: offline archives missing new modules (f98c192) and the relay's `python -I` spawn breaking the gateway import (2de4920). Full command/evidence record: [remediation queue](REMEDIATION_TEST_QUEUE.md).

No external TLS domain, login callback in a real hosted ChatGPT/Claude/Cursor account, or public Caddy service was exercised. These remain E/L=U (real public HTTPS and external clients); Caddy on a host without it fails closed (`CADDY_NOT_INSTALLED`); the R3 proxy evidence is loopback TLS, not a public origin. R0 gateway process overhead and third-party URL path handling remain open acceptance areas.


## Hosted ChatGPT OAuth incident and browser callback regression (2026-11-01)

**Operator-observed E/L, historical installed process (not the v0.24.2 package):** An owner-configured temporary Cloudflare Quick Tunnel returned HTTP/2 200 for both remote OAuth discovery endpoints. The hosted ChatGPT client opened `GET /authorize`; its browser reported a `POST /authorize` with `Origin: null`, followed by `invalid_origin`. With a temporary process-only server patch, the browser subsequently reported a Content Security Policy violation for `form-action 'self'` on the cross-origin OAuth callback. The initial successful consent consumed the one-time pending authorization; a second submission of that form returned `access_denied`. After changing only the successful authorization page's referrer policy to `same-origin` and adding the registered callback origin to its form-action CSP, the owner confirmed hosted ChatGPT OAuth connected, and an authenticated `list_containers` call through the connected BoxWeave MCP returned four running boxes on the `dev` backend. No passwords, codes, tokens, raw capability URLs or complete redirect URI query strings are retained here.

**v0.24.2 source correction:** `bwmini/remote_mcp.py` reproduces that scoped header behavior without changing the global Origin gate. `tests/test_remote_mcp.py` verifies OAuth callback CSP origin scoping, referrer behavior, denied `Origin: null` and hostile Origin, unknown registered-client callback denial, successful PKCE grant and one-time consent replay denial. This is fixture coverage (T once CI passes), not a replay of the exact hosted-browser flow on v0.24.2 (E=U for the new release).

**Pending real checks:** Following reviewed release/deployment, independently record installed/runtime version and retest hosted ChatGPT consent once, first-click 302 callback, valid access token tools/list and non-mutating read, wrong-Origin rejection, existing SSH MCP parity, and a restart/revocation case. Current temporary Tunnel and process-local hotfix are not permanent or automatically upgraded by committing this source.

## v0.24.3 managed-Caddy and Web status regression (2026-11-01)

**Source changes:** The R3 Caddyfile generator no longer sets a global `Referrer-Policy: no-referrer`; the upstream OAuth consent page's `same-origin` policy must survive the proxy unchanged. The R4 Web panel now performs a bounded 200 ms loopback TCP-connect observation to its configured remote port. The field `local_tcp_reachable` is explicitly not an MCP handshake or proof of public HTTPS/OAuth/real Box readiness; `externally_verified` remains false regardless of the TCP result. No live host was modified as part of this source update.

**Fixture tests:** `tests/test_remote_mcp.py` checks absence of proxy header override, original OAuth valid/hostile/null Origin cases and redirect CSP, Web port success/failure with patched loopback socket, separate browser admin CSRF and credential revocation. GitHub Actions verify and candidate ZIP/offline-wheel smoke must run against the final commit before source acceptance is reported as T. Release job verification and later stable promotion are distinct gates.

**Still unverified for 0.24.3:** Real Caddy TLS certificates/port ownership, public HTTPS origin via Caddy, source-to-installed upgrade, exact running process, newly installed ChatGPT browser consent and write access, independent Codex/Claude/Cursor OAuth clients, one-field capability URL compatibility, and a managed process auto-start/recovery. The 2026-11-01 temporary browser hotfix plus hosted ChatGPT list success is prior-process E/L evidence, not a 0.24.3 deployment result.

## v0.27.0 default local service and explicit public configuration

**Design**: [HTTP MCP quick-start](../design/HTTP_MCP_QUICKSTART.md), QH-01..QH-07. New tests cover default loopback resource, rejected anonymous/foreign Origin/Host, explicit disable, port conflict, user-systemd unit generation, private config+runtime transactional public URL update and restart rollback. Run the full GitHub PR/main check; record SHA and actual result. No tests involving Cloudflare credentials, real public TLS or production config are implicit.

**Real installation gate still required**: a separately authorized installed v0.26.x → v0.27.0 upgrade, a fresh install, both HTTP and SSH authenticated tools/list, OAuth browser callback after configure, a failed port-conflict path, opt-out, reboot auto-start and update/recovery. A fixture/local TCP check is not real installed evidence. Keep status U until actually observed.

## v0.27.2 installer human-readable HTTP MCP onboarding

**Fixture acceptance:** `tests/test_online.py` verifies the success message shows local endpoint, info page, status/doctor/restart, operator-started Quick Tunnel and ChatGPT OAuth URL, plus custom port propagation; when a valid public URL already exists it shows the exact `/mcp` URL without duplicate path and does not claim HTTPS reachability; explicitly disabled HTTP MCP omits enable/tunnel instructions. Existing SSH `boxweave client` and Web password-location text remain intact. This is source/fixture coverage, not real installed version validation.

**Live update evidence, separate:** On 2026-11-01 after operator reported host upgrade, the connected BoxWeave MCP described source_version `0.27.1` (static metadata, not installed version proof); `list_containers` returned six running Boxes; `boxweave_diagnose(source=gateway,recent=5)` reported successful read events; and `connect(box=boxweave-test-ser,exec=python3 -c 'print("hello world")')` returned exit_code 0 and the expected text. This verifies an authenticated read and command chain, not installed HTTP systemd restart behavior, public OAuth setup or automatic boot. Exact v0.27.2 installed text output remains U until observed.
