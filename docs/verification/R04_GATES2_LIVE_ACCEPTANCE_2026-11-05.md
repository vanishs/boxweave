# R-04 follow-up live acceptance: natural lease expiry, immutable replacement, private HTTPS OAuth and Web-first claim

**Executed:** 2026-11-04 Pacific / 2026-11-05 UTC. **Installed source under test:** official BoxWeave `v0.44.9`, SHA `0ec0dc81d8cdc69d4fb9a8abaae0615cf41e4f24`. **Overall: PARTIAL; no global R-04 production activation.** This report records only executed proof and separates instrumentation, private-client TLS and normal-credential limitations.

## Authorized test topology and isolation

- Reused *previously task-owned* test parent `bwr04-1104-matrix-container` only for a **natural unrenewed 600-second Alpha lease**. Its separate original Beta Gateway UNKNOWN task `33e97ea5e16bb753e053644d535740fc` and two resource claims were preserved.
- Created a genuinely new disposable outer-managed Ubuntu 24.04 Box `bwr04-1104-gates2-container`, outer Provider createdAt `1793862085`, outer UUID `589b6e7b-8374-4710-b853-2b9a81b3dd59`, 2 CPU/4GB/35GB, nested-capable after exact-Box stop/start. Its actual outer `connect(exec)` returned `hello world`, exit 0. Fresh official v0.44.9 installer exited 0; provenance matched immutable SHA, `sudo boxweave doctor --mcp` 10/10 passed. Containarium v0.128.0 remained the inner Provider baseline. `sudo boxweave task provision` prepared private metadata/coordinator stores but did **not** activate global R-04.
- Only inside the new disposable parent: the app runtime was backed up; `coordination.enabled=true`, a two-target lifecycle allowlist (`r04-webfirst` and `r04-replace`), and a root-owned opt-in Gateway policy were established while test app services were stopped. App `serve`, `worker`, `remote-mcp` restarted and were observed active. Real app lifecycle workers created both nested workloads, reaching `succeeded/connection_ready`. Their Provider and Incus identity/state were independently observed.
- An **independent nested client container** `r04-oauth-client` on a distinct network namespace used the private host bridge. An Nginx TLS reverse proxy listened **only** on `10.203.0.1:18443` and forwarded to the unchanged loopback-only HTTP MCP listener `127.0.0.1:18844`; private lab DNS `mcp.r04.internal` and a lab-only CA public certificate were configured in the client. The lab CA private key never left the test parent. Neither a public DNS name nor external Internet ingress/tunnel was configured. The installed remote MCP `public_url` was pinned to `https://mcp.r04.internal:18443`. The owner host `dev`, stable release selector, historical incidents, and public networking remained unchanged.

## Executed gate L-EXP: natural full-length lease expiry — PASS

- In the existing dedicated `bwr04-1104-matrix` lab, an actual authenticated loopback HTTP MCP client acquired an Alpha Box lease with **600 seconds** from real server time, without artificial clocks, manual expiration edits or renewal.
- After actual expiry, the original lease release returned `LEASE_EXPIRED`. A different logical acquisition obtained `leased` with **generation 2** (previous generation 1). The newly active lease rejected a real application worker Alpha resize with `failed/RESOURCE_BUSY`, and independent Incus CPU read remained 1. New lease release returned `released`. No active lease remained.
- The special temporary HTTP API credential was revoked and its private token file deleted. The original unrelated Beta `dispatch_intent` UNKNOWN and exactly two resource keys stayed retained.
- This proves **lazy lease expiration and post-expiry re-acquisition** within the sampled installed path. It does not infer that expired leases can clear any unresolved operation claim; the prior separate crash-scene test proves the opposite fail-closed boundary.

## Executed gate I-REPLACE: exact-incarnation and same-name replacement — scoped PASS

- Genuine application lifecycle created nested `r04-replace-container`. Its original observed immutable incarnation was created `2026-11-05T07:13:00.870262897Z`, UUID `54d6dab4-4015-495f-aa27-ead8fd61cd8c`.
- A real tracked SSH MCP guest task was dispatched, then only its identified **test-only** Gateway process was SIGKILLed. The original task `344152969e5be069329bfcde312d93bf` remained a `dispatch_intent` coordination claim with **two resource keys**, and an independent old-guest observation found a **one-byte** side effect.
- Before replacement, consistent private SQLite backups of task receipts, app DB and claims were made inside `/home/boxweave/.boxweave/r04-replace-evidence` of the disposable parent. Exact original incarnation metadata was retained in private test files.
- **Explicit bypass simulation**: only this disposable guest was then deleted through the *uncoordinated* upstream Provider adapter; the actual instance absence was independently observed. The same logical name was recreated via that upstream adapter. New Incus UUID `77c16a3a-6a2d-428d-bb52-6b4143b8648f`, created `2026-11-05T07:18:11.577786327Z`. Name unchanged, UUID and created timestamp different.
- A newly authenticated real SSH MCP tracked write toward the replacement returned **`RECOVERY_REQUIRED`**. The replacement guest did **not** contain the denied marker. The app's old managed lifecycle identity rejected a new resize plan with **`TARGET_CHANGED`**. The original `dispatch_intent` and its two resource keys were still retained.
- **Scope:** the coordinated Gateway and application never silently re-used the old incarnation or old unresolved name claim. Direct Provider/Incus bypass was deliberately performed **only inside a disposable lab**; that bypass itself was not prevented and is outside BoxWeave's global guarantee. The replaced guest and original UNKNOWN evidence remain for investigation. Do not automatically reconcile, replay or delete this claim.

## Executed gate W-FIRST: app-worker-first deterministic conflict — PARTIAL

- A real released application lifecycle `resize` operation was queued for existing `r04-webfirst`. Only the disposable parent app worker service was temporarily stopped. A one-use **test-only runtime barrier** wrapped `provider.lifecycle_mutate` in an isolated separate process; all deployed BoxWeave source bytes stayed unchanged. The wrapper signalled **after** the genuine app `SharedClaims.dispatch_intent` and **before** the real Provider call. It resumed on an explicit bounded test signal.
- Original app operation `a853c79743c04090bd60c649c92cc5e0` was independently read in `dispatch_intent` while the barrier was active. A completely independent pinned-host-key SSH MCP Gateway process attempted a same-Box guest write and returned `RECOVERY_REQUIRED`; independent actual guest check confirmed **no denied file**. Releasing the barrier allowed the original app worker receipt to succeed with `OK` and real Provider CPU 2; subsequently normal installed app worker resumed and CPU was restored to 1 with a successful receipt and Provider readback.
- **Honest limit:** this is deterministic *real app-worker ownership vs real SSH MCP*, using an injected pause. It was **not submitted through an authenticated installed Web HTTP client** and did not simultaneously exercise authenticated HTTP MCP. It is not the completed `Web → SSH MCP + HTTP MCP` three-entry installed acceptance gate. Full W-FIRST remains **U** until a normal authenticated Web plan/confirm actor owns the original receipt and both independent MCP clients race while that receipt remains in-flight.

## Executed gate TLS-OAUTH: private client HTTPS/protocol — PARTIAL

From the actual separately isolated nested LXC client, with the test CA installed and hostname checked:

| Check | Observed result | Scope |
| --- | --- | --- |
| TLS certificate-chain and hostname verification | HTTPS request succeeded, HTTP 200 | Private network and private CA only |
| `/.well-known/oauth-protected-resource/mcp` | 200, correct HTTPS resource URL | Live installed remote MCP through TLS proxy |
| `/.well-known/oauth-authorization-server` | 200, issuer and S256 PKCE advertised | Live protocol metadata |
| Dynamic client registration `POST /register` | 201 | Real HTTPS DCR |
| PKCE authorization `GET /authorize` with registered redirect | 200, consent page and restrictive callback CSP | Real authorization page, **no approval** |
| Unauthenticated HTTP MCP `POST /mcp` | 401 | Denied before tool dispatch |
| Wrong request Origin / invalid token exchange | 403 / 400 | Security negative paths |
| Wrong TLS hostname | Certificate verification rejected | TLS trust negative |
| Installed Web `POST /api/login` with intentionally invalid credentials | 401 | Negative login only |

**Unverified, not PASS:** valid first-admin/operator password Web login and session/CSRF; user consent `POST /authorize`; real authorization-code callback, PKCE exchange, bearer access, refresh/revocation from an independent HTTPS client; public Internet reachability, public DNS, trusted-public-CA chain and actual Cloudflare/reverse-proxy deployment. In-tool access to privileged administrator credentials was unavailable/blocked by the safety boundary; no private admin password was read, exported, reset or bypassed. A deliberately injected Web session or bearer token would not be evidence of the normal login/OAuth approval flow. Do not mislabel live HTTPS metadata success as completed external OAuth sign-in.

## Safety disposition and executable next matrix

**Test credentials:** the separate SSH MCP test public key was unenrolled and its test-only private client key deleted; the old lab's temporary lease API credential was revoked. TLS server CA private key remains root-owned in the private, disposable test parent for optional additional acceptance; no public listener was added. Both labs remain under their explicitly test-owned resources. The new same-name `r04-replace` unknown operation `344152969e5be069329bfcde312d93bf` plus its two keys were deliberately retained and backed up; old `bwr04-1104-matrix` Beta UNKNOWN remains unchanged. Do not reset either database or blindly delete/replay operations.

Remaining acceptance plan, in this order:
1. **Normal Web and OAuth credential flow:** operator provides a legitimate one-use test administrator login **only through the private Web or OAuth login UI**, without disclosing the credential to agents or logs. From the isolated external-client Box, execute OAuth DCR → PKCE S256 → explicit user approval → callback → code exchange → authenticated `tools/list` → refresh/revoke; independently read session and expired/revoked failures. Keep TLS private; public Cloudflare/DNS needs *separate exposure approval*.
2. **Deterministic true Web-first three-way:** using a fresh disposable Box name and a normal authenticated Web HTTP plan/confirm, attach a test-only synchronization barrier immediately after the original Web worker's shared `dispatch_intent` commit, before Provider mutation. Launch independent pinned-SSH MCP and independently authenticated HTTP MCP concurrent requests to the **same immutable Box**, plus a distinct-Box control. Assert `RECOVERY_REQUIRED`/fail-before-dispatch, zero denied guest side effect and unaltered Provider CPU; release barrier and independently verify original Web receipt then a clean follow-up. Clearly distinguish instrumented synchronization from unmodified scheduling and preserve any UNKNOWN.
3. **External Internet TLS is a separate opt-in gate:** if requested, use an owner-approved disposable public DNS/TLS/tunnel origin pointing only at the isolated environment, complete OAuth with an actual off-lab client, and then disable the exposure. Private CA HTTPS evidence cannot satisfy this gate.
4. **Original uncertain-claim reconciliation** is not part of these tasks: requires original immutable Provider-request correlation and a separately reviewed recovery mechanism. Keep both retained scenes untouched.

**Status:** natural full-length lease **L PASS**; same-name live replacement guarded surface **L PASS scoped**; test-instrumented app-worker-first vs SSH MCP **L PASS scoped / full W-FIRST U**; independent private TLS and OAuth discovery/DCR **L PASS scoped / full AUTH U**. Global R-04 remains **PARTIAL**, not operator-host enabled and not production certified.
