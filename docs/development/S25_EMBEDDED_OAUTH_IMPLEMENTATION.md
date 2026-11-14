# S2.5 — Embedded OAuth for ChatGPT HTTP MCP (implementation contract)

**2026-11-13 subsequent RC.3 pilot addendum:** Embedded OAuth is shipped in [RC.3](https://github.com/vanishs/boxweave/releases/tag/v1.0.0-rc.3). The current [physical-host record](../verification/RC3_PHYSICAL_HOST_PILOT_2026-11-13.md) includes real ChatGPT OAuth authorization and three authenticated **read** MCP calls. The historical OA-008/009 lab-blocked entries below remain dated lab facts, but no physical **native write** or final HOST_GO is inferred. Generalized operator steps are in the [manual guide](../getting-started/MANUAL_INSTALL.md).


**Status: GO IMPLEMENTATION + BOUNDED ISOLATED BOX LAB PASS (2026-11-13); OA-008/009 BLOCKED on real ChatGPT and remote Agent I+N+V.**
**Owner decision:** preserve one canonical `https://<public-origin>/mcp` for
all MCP requests, support OAuth in the existing Go controller, and retain
optional direct revocable Bearer tokens. This supersedes **only** the old
v1 exclusion of embedded OAuth; it does **not** restore the old `main` web
console, Containarium, Caddy, HTTP REST management, or a general identity
platform. No backwards DB/API compatibility for pre-release v1 is required.
**No new public MCP tools.**

**Original implementation trigger:** the owner said "开始 OAuth 开发";
this bounded work has now been implemented and local Box-tested. The next
unmet gate is an explicitly approved live ChatGPT connection / original-key
MCP-to-Incus readback, not another unscoped OAuth implementation run. Do **not** treat this planning document as permission to execute
formal HOST_GO Round 3, merge v1, expose the physical host, or issue a
release. Round 3 is triggered separately under
[HOST_GO runbook](HOST_GO_THREE_STAGE_RUNBOOK.md).

## Latest code + acceptance checkpoint (2026-11-13)

- Exact latest code-bearing PR #133 commit: `a8dd90c7bd0c88150e96fba954b3468887c963af`; installed in *new* isolated `boxweave-test-ser2` inner Guest `hg-v1-oauth-1113` with binary SHA-256 `9307011adf0587d07ce8c81ed2d293c8be11715e7dea41ead690d03d27e00a73`.
- The planned Go SDK v1.8.0 + go-oauth2/v4 v4.6.0 integration is implemented; its manager was explicitly **not** used for code/refresh transactions. SQLite v2 owns all atomic code, grant and refresh state. RFC 9207 issuer callback and host-local OAuth access-ID grant revocation were added after code review. A further RFC 9700 security audit exposed the need for **spent refresh-token replay detection**; exact `a8dd90` retains digest-only consumed refresh tombstones and revokes the entire grant on reuse, with concurrency, reopen and SQLite-fault regression tests. No schema migration was required.
- All Go race/quality/S1 required Actions at the **code-bearing commit** are SUCCESS; original installed/native lab evidence is distinct from fake HTTPS origin simulation. The test-only public origin is disabled and there is **no running Tunnel**. Real ChatGPT DCR/OAuth over remote HTTPS and actual authenticated MCP native mutation remain BLOCKED, not PASS.
- [Exact evidence, failures, fixes and OA-001–010 matrix](../verification/S25_EMBEDDED_OAUTH_ACCEPTANCE_2026-11-13.md) supersede older preparatory topology/test-location assumptions below. Existing original UNKNOWNs and physical host remain untouched.
## 1. Goal, operator experience and immutable boundaries

The prior `main` workflow is the intended UX:

1. BoxWeave v1 MCP already listens on `127.0.0.1:18650/mcp`; it stays
   loopback-only on all layers. The single existing Go HTTP server now
   also serves bounded OAuth discovery, authorization and token endpoints.
2. An operator separately starts **Cloudflare Quick Tunnel**
   (`cloudflared tunnel --url http://127.0.0.1:18650`) *in the same
   host/Guest network namespace as that v1 listener*. No Cloudflare
   account, named Tunnel, domain, Cloudflare Portal or embedded tunnel
   management is required for this **temporary development path**.
   Quick Tunnel is publicly reachable, short-lived, random-hostname and
   **does not support Server-Sent Events**; real SDK/ChatGPT transport
   negotiation MUST prove JSON-compatible operation before acceptance.
   A fixed named Tunnel or trusted proxy is optional for later stable use.
3. Operator sets the **exact externally returned HTTPS origin** locally
   (not a URL supplied by an HTTP caller). The root-local CLI must
   reject non-HTTPS, paths, queries, fragments, user-info, ambiguity,
   invalid ports/hostnames, loopback/private public origins, and
   unsupported multi-host configurations. Review/update persisted config
   transactionally and restart/reload under the existing installer
   maintenance/serve locks; failed reload restores previous config.
4. ChatGPT registers `https://<random>.trycloudflare.com/mcp`. It
   discovers OAuth metadata, opens the single-operator browser approval
   page, receives resource-bound credentials and then calls the **same**
   `/mcp` with `Authorization: Bearer ...`. It does not need a second
   `/mcp/oauth`, `/mcp/token` or REST-management surface.
5. A host-local administrator initializes/rotates a **separate OAuth
   consent password** (entered through a non-echoing local prompt;
   Argon2id/safely configured memory-hard hash in protected SQLite),
   or an equally small, independently validated local-only approval
   mechanism if the password bootstrap proves unsuitable. v1 has **no
   existing Web administrator password to reuse from `main`**. Never
   interpret an existing raw admin MCP Bearer token as a browser login
   password or paste a password/token into GitHub/ChatGPT/logs. The
   browser approval page displays the exact requesting client and
   complete canonical redirect destination, the role to grant, and
   explicit consent.
6. The existing independently revocable `admin` / `viewer` caller
   roles remain authoritative. OAuth is an **additional credential
   acquisition method**, not an additional privilege tier. No general
   Web UI, users, custom provider APIs or new Box ownership grants.

**Important:** localhost being available does not authorize WAN exposure;
the owner must explicitly opt in for an individual test. A Quick Tunnel
URL alone does not prove authenticated MCP or actual Agent acceptance.

## 2. URL contract — stable resource, separate authentication

For configured `public_origin=https://example.trycloudflare.com`:

| HTTP method / route | Responsibility |
| --- | --- |
| `POST/GET /mcp` (and other MCP-supported methods) | **Only** SDK-owned Streamable HTTP MCP resource, stateless transport, authenticated on every request; not a custom JSON-RPC router |
| `GET /.well-known/oauth-protected-resource/mcp` | MCP-specific RFC 9728 discovery; `resource=https://example.trycloudflare.com/mcp` |
| `GET /.well-known/oauth-protected-resource` | Root RFC 9728 compatible discovery for clients that probe root |
| `GET /.well-known/oauth-authorization-server` | RFC 8414 AS metadata; exact `issuer=public_origin`; accurately advertise grant types, PKCE S256, registration capabilities |
| `POST /register` | **Bounded** RFC 7591 DCR public-client registration for actual ChatGPT compatibility; server-side exact registered redirect URI checks, durable client records |
| `GET /authorize` | Validate resource, public client, requested role/scope, PKCE and CSRF state, then render the single-operator consent page |
| `POST /authorize` | Verify local operator credential, rate/CSRF/Origin policy and one-time pending intent; send exactly one authorization code via the approved callback |
| `POST /token` | Only `authorization_code` with PKCE S256 and `refresh_token`; exact resource/client/redirect binding; short-lived access and rotated refresh |
| Other paths | 404; method mismatch uses 405/Allow when appropriate; never redirect `POST /` or silently alias `/` to MCP |

Exactly one canonical MCP URL is used **regardless of** authentication scheme.
A future XAuth/other supported authentication mechanism can provide separate
discovery/login endpoints while its resulting verified actor enters the
same MCP role/owner boundary. Do not claim a future client supports a
particular scheme without client-side evidence. Multiple OAuth authorization
servers may be listed in the resource metadata only once truly supported;
**initial release advertises exactly one local AS**.

The OAuth `resource` is the **full HTTPS /mcp URL**; the `issuer` is
the public HTTPS *origin*. Access tokens are audience/resource-bound,
and every authorization/token request validates the explicit
`resource` parameter (RFC 8707). The auth scope naming should be
minimal and unambiguous: advertise `mcp:viewer` (read-only) and
`mcp:admin` (explicit operator-approved writes), map them solely to
internal `viewer` and `admin` roles. Never infer an administrator
grant from a default, absent scope, clientInfo, work_ref or agent_ref.
Determine whether an actual ChatGPT OAuth request uses a scope-absent
flow and define/verify the safest compatible explicit-consent behavior;
do not silently escalate absent scope.

**Client registration policy:** DCR is an actual supported compatibility
path, not unrestricted public registration. Validate redirect URI
scheme, exact URI equality, cardinality, bounded length and storage
quotas, rate-limit registration, and prevent open redirect/loopback
impersonation. Public clients use `token_endpoint_auth_method=none`
with mandatory PKCE S256. DCR capability is advertised only after it
passes tests. CIMD may be investigated later but is **not**
advertised as supported in this slice; do not partially implement it
or allow server-initiated unchecked SSRF metadata fetches.

## 3. Implementation choice — SDK first, bounded OAuth library

**Source baseline:** pinned official
`github.com/modelcontextprotocol/go-sdk v1.8.0`, Go 1.26.9 and existing
SQLite. Keep `mcp.NewStreamableHTTPHandler` and the same typed handlers,
including request-scoped `req.Extra.TokenInfo`.
Use official `auth.RequireBearerToken` for the common bearer gate and
`auth.ProtectedResourceMetadataHandler` for RFC 9728 discovery.
Set the SDK middleware's `ResourceMetadataURL` to the exact configured
public URL when external OAuth is enabled. The Go MCP SDK **does not
provide a complete AS with password consent, grants and refresh**.

**Preferred AS engine to evaluate and pin in go.mod:**
[go-oauth2/oauth2/v4](https://github.com/go-oauth2/oauth2).
It offers OAuth authorization-code server functions and PKCE/S256 support.
Before choosing a concrete version or writing HTTP adapters, **inspect the
current tag's full interfaces and transitive dependencies** and prove with
real Go tests that it can: (a) prohibit implicit/password/client-credentials
grants; (b) force only S256 public PKCE; (c) bind exact redirect, client and
resource; (d) use one durable SQLite transaction for code redemption +
token issuance, and another for refresh rotation; (e) respect disabled
clients, revocation and expiration. **Do not use its in-memory token/client
manager** in the installed service, and do not assume library defaults
satisfy MCP 2025-11-25 requirements. A wrapper with extensive custom
protocol code defeats the simplicity objective; if any critical property
cannot be met cleanly, record the concrete gap and compare a narrowly
scoped Fosite adapter before substituting a home-grown OAuth server.
No OIDC provider, external database, JWT/JWKS rotation service or
separate daemon is required for opaque local credentials.

**Internal layout guideline** (adjust to actual code, not a frozen API):
- `internal/control/server.go`: single HTTP mux, public Host/Origin/CSP
  policy at the correct per-route boundary, SDK `/mcp` untouched.
- `internal/control/oauth_*.go`: minimal consent/metadata/token handlers,
  relying on the reviewed AS library; no duplicate MCP handler.
- `internal/state/oauth_*.go`: protected SQLite credential, client,
  pending, code, access, refresh, grant, token-family and auth-attempt
  persistence/recovery; auditable transaction boundaries.
- `cmd/boxweave` / existing `internal/install`: strictly host-local
  `oauth setup` / `oauth public-url set` / `oauth status` /
  `oauth disable` (exact names may be simplified after tests). No MCP
  tool for changing OAuth policy or exposing a credential.

**Unifying authentication:** existing manual v1 API Bearer tokens and
new OAuth access tokens remain independently created/revoked/typed.
Refactor the store verifier only as needed so both yield one verified
`TokenInfo` with immutable token ID, role and (for OAuth) exact issuer/
resource/client binding; do not reuse a token ID or conflate access
tokens, refresh tokens, authorization codes, local approval passwords
and Incus credentials. A valid OAuth refresh token is **not** valid
as a direct MCP Bearer. Viewer cannot write/Exec. No host Incus admin
privilege is added to the MCP runtime.

## 4. Security and transactional rules (mandatory, fail closed)

1. **No cross-scheme authorization shortcut.** Every `/mcp` HTTP
   request must validate its access token, role, expiry/revocation,
   audience and Box UUID/Project ownership; deny unauthenticated,
   wrong-role, wrong-resource, unknown-origin, expired/revoked tokens.
   Do not log/tokenize full secrets, authorization codes, callback query
   strings or guest output.
2. **Authorization code:** crypto-random opaque secret; only digest
   stored; one-time consume; <=120 s lifetime; exact client ID,
   resource, redirect URI and S256 challenge; do not reuse on
   callback/CSP failure. The browser's `state` is preserved in the
   redirect but never an authorization grant.
3. **Refresh family:** crypto-random opaque refresh token; rotate
   atomically in SQLite together with new access/refresh issuance.
   Restart, concurrent refresh and injected SQLite I/O failures
   must not consume a usable old refresh token without producing a
   committed replacement, nor yield two distinct valid successors.
   Revoked grants/client/password reset and public-origin changes
   invalidate affected OAuth families, not unrelated local API tokens.
4. **Credential persistence:** no plaintext password or tokens in
   SQLite, logs, command-line arguments, Query Strings or
   `/root/*.secret` checked into source. Hash using domain-separated
   crypto digests; memory-hard operator password hash with explicit
   parameters. Store consent attempts in a separately committed,
   rate-limited table so denied password attempts are **not rolled
   back** with the rejected authorization transaction. Cap pending
   requests/DCR clients/codes, body size and concurrent authorization.
   Avoid cookie dependence on third-party browser contexts.
5. **Host / reverse-proxy trust:** MCP currently restricts
   `Host=localhost/127.0.0.1` and rejects supplied `Origin`; replace
   that with exact per-route policy for a **locally configured**
   public origin, plus loopback for local probes. Never trust arbitrary
   `Host`, `Forwarded`, `X-Forwarded-Host` or
   `X-Forwarded-Proto` to construct issuer, callback, CSRF checks
   or authority. Public OAuth redirects MUST be HTTPS. DNS
   rebinding, spoofed Host, foreign Origin and `Origin: null`
   remain denied. Do not just disable Go `http.NewCrossOriginProtection`.
6. **Browser callback fix from main:** on the verified, valid
   `GET /authorize` consent page only, use
   `Referrer-Policy: same-origin` and narrow CSP
   `form-action 'self' <exact registered callback origin>`;
   do not globally weaken Origin/CSP. The consent POST must be
   from the exact public origin, with a valid one-time pending
   session/CSRF binding. Other pages keep a strict no-referrer
   and no-frame policy; no `Origin: null` bypass.
7. **Public origin lifecycle:** default local-only OAuth disabled
   until configured. Updating/removing a Quick Tunnel hostname
   changes issuer/resource binding; explicitly invalidate old
   OAuth access/refresh authorization, clear pending flows, and
   require fresh authorization. An already accepted original
   native Operation/UNKNOWN is never replayed/deleted because of
   OAuth reconnect/revocation; do not restart Incus or clear ledger.
   Config/DB update failure leaves the previous safe state.
8. **HTTP response contract:** `401` must advertise the
   RFC 9728 resource-metadata URL (with a safe role/scope only
   when actually known), no secret in redirects/logs; `403` for
   policy/origin denial, `404` for nonexistent paths,
   `405` for unsupported methods. Preserve `Cache-Control:
   private, no-store` on MCP and `no-store` on tokens/forms,
   `Vary: Authorization` where appropriate, normal TLS proxy
   methods/bodies, and the official SDK protocol-version selection.
9. **No automatic public Tunnel** in the BoxWeave installer. A
   separately run Quick Tunnel is allowed only for an owner-approved
   bounded LAB. Existing `cloudflared-hostgo.service` (named-token
   variant) remains inactive; it is not required for this path.

## 5. Main branch: required knowledge / anti-regression crosswalk

Study the **actual patches and regressions**, not just current
`main` descriptions. Re-implement the *behavior in Go*, not the
Flask/old Provider/REST architecture.

| Prior main evidence | Failure mode v1 must prevent |
| --- | --- |
| [v0.24.2 browser callback patch](https://github.com/vanishs/boxweave/commit/d047a3cf384e2f214f0ccc663cadbd33371c50c0) | ChatGPT browser sends `Origin: null` with wrong Referrer-Policy; valid OAuth consent 302 blocked by `form-action 'self'`; second click reuses consumed consent and fails |
| [v0.26.9 atomic OAuth state patch](https://github.com/vanishs/boxweave/commit/008f659593376c7df1608599bd523885197a6362) | Failed code exchange/refresh leaves consumed one-time secret or orphan token; preserve atomic transaction and negative restart/fault cases |
| [v0.49.5 ChatGPT `tools/list` fix](https://github.com/vanishs/boxweave/commit/043f1daf1856d97f35167ca61630ff5a71840e6e) | ChatGPT sends `tools/list` with omitted optional `params`; legacy hand-written protocol rejected it; verify **official SDK** correctly handles it |
| [main remote MCP code](https://github.com/vanishs/boxweave/blob/main/bwmini/remote_mcp.py) and [original auth](https://github.com/vanishs/boxweave/blob/main/bwmini/remote_auth.py) | Metadata/PKCE/Host/Origin/Bearer semantics, DCR, revocation, cache/CSP, and consent bind; reuse lessons only |
| [main remote MCP runbook](https://github.com/vanishs/boxweave/blob/main/docs/REMOTE_MCP.md) | Exact `/mcp` rather than `/`; root `POST /` 405 is a *wrong URL*, metadata success does not prove Tool readiness; temporary hostname change requires reconfigure/reauthorize |
| [MCP 2025-11-25 HTTP auth spec](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2025-11-25/basic/authorization.mdx) | RFC 9728/8414/8707, resource-bound access tokens, discovery order, DCR fallback, exact scope, RFC 7591 registration |
| [official MCP Go SDK auth](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk@v1.8.0/auth) | Prefer existing SDK bearer middleware and protected-resource metadata helper; no duplicate MCP transport |
| [official Cloudflare Quick Tunnel](https://developers.cloudflare.com/tunnel/get-started/quick-tunnels/) | Random account-free HTTPS is test-only and **does not support SSE**; JSON response path and real hosted ChatGPT must be proved |

## 6. Development order and acceptance matrix

Develop this as **one bounded S2.5 pre-Round-3 slice**; checkpoints
below are substeps, *not* additional user-triggered HOST_GO rounds.
On failure, fix and re-run; write original failures to log. Use the
existing locally isolated Guest before Actions, never a physical host.

1. **Protocol/library spike (no live writes):** verify the exact
   pinned Go SDK v1.8.0 HTTP responses and selected go-oauth2 version,
   S256-only code issuance/refresh semantics, DCR extension, required
   SQLite atomicity, structured 401 challenge and SDK JSON vs SSE.
   Record decision/version and any unsupported compatibility before
   committing a library.
2. **Implement one Go service:** routes/metadata/registration/consent/
   code/refresh; local password bootstrap; exact public origin
   configuration; existing per-request role verification; no additional
   public MCP Tool or local Web UI. Add foreign client/host/Origin
   negatives before happy path. Keep canonical `/mcp`.
3. **Unit and browser/protocol integration tests** using a real SDK
   Streamable HTTP client plus browser-like cross-origin callback
   (not only a custom JSON-RPC mock).
4. **Black-box installed/native test:** install the **same new source
   binary SHA** into disposable nested `hg-v1-cft-1113`
   (outer main-managed `boxweave-test-cli`; inner own
   `user-1001`), with its own SQLite and token namespace.
   Respect original Incus UUID
   `c45c08e3-cb41-4354-8bf7-f1eb5651408d`, old outer
   `bw-s25-l1a`, all retained UNKNOWN, and any live original
   operations. Inspect and conservatively upgrade or choose a fresh
   separately reviewed disposable install; never reset SQLite or
   overwrite unrelated v1 service.
5. **Quick Tunnel and real ChatGPT:** owner runs/authorizes
   `cloudflared tunnel --url http://127.0.0.1:18650` from inside
   the new Guest or an equivalently owner-approved namespace.
   Configure returned public URL via the safe host-local CLI.
   Capture **redacted** remote HTTPS resource/AS discovery,
   actual consent first-click callback, authorization token,
   `initialize`, `tools/list`, `host_status`, reconnection,
   refresh/revocation and a permitted original-key S2 mutation
   with native readback + independent Verifier. Owner must connect
   the intended ChatGPT client; Agent writes require its separate
   explicit approval. A connected/green OAuth button alone is
   **not** an Agent PASS.
6. **Final only after real Box tests:** `make verify` +
   `go build`, exact new binary commit/digest, independent
   verifier, same-head required `quality-gate` and applicable
   Actions. Update
   [HOST_GO stage checkpoint](../verification/HOST_GO_STAGE_STATUS.md),
   [execution log](EXECUTION_LOG.md), this acceptance record,
   and PR #133. Keep Draft if live tests or safety negatives
   remain blocked; no premature `HOST_GO=PASS`.

**Acceptance IDs (each must have evidence, not assertion):**

| ID | Required result |
| --- | --- |
| OA-001 | `/mcp` is the single canonical endpoint for manual Bearer and OAuth access; SDK still handles MCP protocol including omitted optional `tools/list.params` |
| OA-002 | RFC 9728 resource metadata (both paths), RFC 8414 AS metadata, exact public issuer/resource, `401 WWW-Authenticate`; rejects fake Host and forwarded authority |
| OA-003 | Public-client registration bounded and exact redirect; wrong clients/URIs, PKCE plain/absent, code replay, wrong resource/redirect/client rejected |
| OA-004 | Browser first-attempt consent + 302 callback succeeds with narrow CSP/Referrer; wrong/empty/null Origin, missing approval, bad password, CSRF, click replay denied |
| OA-005 | Code redeem + refresh rotation atomic under concurrent requests, restart, SQLite failure injection; old/revoked/expired/foreign-audience token denied |
| OA-006 | Unified request actor and role; viewer cannot Exec/write, admin remains confined to exact owned Box UUID/Project; unrelated local API token/revocation unchanged |
| OA-007 | Local URL configuration change/rollback/restart and Quick Tunnel hostname change revoke only affected OAuth families, preserve existing ledger/UNKNOWN |
| OA-008 | External HTTPS discovery, real ChatGPT OAuth, MCP Tool refresh, authenticated read, reconnect and refresh PASS without SSE dependency |
| OA-009 | Same pinned candidate actual MCP original-key write -> independent native L2 readback and separate verifier, no blind retry or foreign native change |
| OA-010 | Exact-head Go race/format/CI and negative security matrix PASS; original installed/native identities and incident evidence preserved |

**PASS semantics:** OA-001–OA-010 only after each actually passes
its scoped proof. If Cloudflare/ChatGPT/host credentials are absent,
record `OA-008/009=BLOCKED` and retain `HOST_GO=PENDING_LIVE_AGENT`.
Pure local OAuth development does not sign formal HOST_GO Round 3,
physical HOST_POSTFLIGHT, release or merger into v1.

## 7. Handoff and current lab state (read before development)

- Draft [PR #133](https://github.com/vanishs/boxweave/pull/133),
  branch `feat/v1-s25-native-readiness-20261111`. Base Go code
  before OAuth: exact installed source `df2ba2a6125378044b4d6afb5389cb23ed2c36c0`,
  v1 binary SHA256
  `59dc559bb771e8dca0e16ee4d899d6162eecc735e4cc2e89cd00d0234ac6a511`.
  Subsequent **docs-only** commits changed PR HEAD; never confuse
  their Git SHA with the code-bearing binary's build identity.
- New disposable inner test Guest `hg-v1-cft-1113`, exact UUID
  `c45c08e3-cb41-4354-8bf7-f1eb5651408d`, on the existing
  top-level `boxweave-test-cli`. Guest running Incus 7.6,
  native Project `user-1001`, v1 service on loopback 18650,
  `cloudflared 2026.11.0` installed; a **named tunnel**
  `cloudflared-hostgo.service` is **disabled/inactive** with
  no token file; Quick Tunnel will be an independent temporary
  process. No OAuth credentials have been provisioned.
  Existing parent service, `bw-s25-l1a` and original two gateway
  UNKNOWN Creates are untouchable.
- Actual documented native-only L2 create/exec/cleanup passed, but
  there is **no** authenticated remote MCP or ChatGPT Agent proof
  yet. See [current checkpoint](../verification/HOST_GO_STAGE_STATUS.md).
- Before writing source, re-fetch PR HEAD, verify retained
  original operations/current Guest inventory and read the relevant
  source/SDK interfaces; do not silently treat this doc's SHA as
  latest after new commits.

**Documentation hierarchy:** this file is the S2.5 OAuth development
contract; [phase card](phases/025_BRIDGEHEAD.md) defines milestone,
[MCP binding](../design/MCP_BINDING.md) and
[security](../design/SECURITY.md) remain authoritative for Box ownership,
and [HOST_GO runbook](HOST_GO_THREE_STAGE_RUNBOOK.md) controls explicit
stages and acceptance. This bounded OAuth decision overrides only
older "OAuth out of scope" statements in those older documents.
