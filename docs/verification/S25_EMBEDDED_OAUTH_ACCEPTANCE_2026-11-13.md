# S2.5 embedded OAuth — scoped implementation and Box acceptance (2026-11-13)

**Code-bearing source:** `f2af899125eda7e6acd0b28a2b36e65eea2ac73d` (PR #133, Draft).
**Exact installed binary:** `1.0.25-hostgo-oauth-f2af899`; SHA-256 `8371560029213919d5d4885afb003f9101515150d5913bcc5735bbda7f20fa06`.
**Verdict:** LOCAL_OAUTH_LAB_PASS (bounded), `OA-008/009=BLOCKED_LIVE_CLIENT`, `HOST_GO=PENDING_LIVE_AGENT`, physical `HOST_POSTFLIGHT=NOT_AUTHORIZED`. No release or merge.

## Implementation verified

- One canonical `/mcp` using official MCP Go SDK `v1.8.0` Streamable HTTP (stateless JSON). No second MCP service, REST management, new MCP Tool, or permanent public listener.
- Optional in-process OAuth authorization-code + PKCE S256, metadata RFC 9728/8414, bounded DCR, RFC 8707 resource binding, **RFC 9207 `iss` callback and metadata**, and explicit browser password/consent. Go OAuth request/PKCE validation uses reviewed `github.com/go-oauth2/oauth2/v4 v4.6.0`; its default Manager is NOT used because its code consumption and refresh are non-atomic.
- Single protected SQLite schema v2: hashed password (Argon2id), registered clients and exact redirect URIs, hashed single-use pending/code/refresh secrets, atomic code redemption and refresh rotation; existing manual Bearer and OAuth access share the unchanged admin/viewer and original owned UUID/Project checks.
- Host-local `oauth-setup`, `oauth-set-origin`, `oauth-disable`, `oauth-status`, `oauth-list-grants`, `oauth-revoke-grant`. Public address is a locally assigned HTTPS origin, never synthesized from `X-Forwarded-*` or caller Host. New `RevokeToken` transaction also invalidates the OAuth refresh grant when given an OAuth access ID.

## Independent validation and failures fixed

1. The first code commit `b1f7a68` passed local source tests but GitHub required quality/S1 format jobs FAILED because the GitHub tree upload lost final newlines in 8 Go files. Corrected in `00d429c`; both exact-head Actions SUCCESS. No product defect was hidden.
2. Local official Go SDK + browser-form simulation covers DCR, authorization GET/POST, exact callback, first-click success, replay refusal, wrong/null Origin and CSP/Referrer, S256-only PKCE, token exchange, refresh, `initialize`, 13-tool `tools/list`, `host_status`, viewer `box_exec` denial, optional omitted `tools/list.params`, and JSON (no SSE) response.
3. State tests cover wrong client/redirect/resource, wrong PKCE without consuming code, expired access/refresh, 12-way concurrent refresh (one winner), failed SQLite write without lost code/refresh, SQLite reopen, OAuth password/origin reset, unrelated manual token preservation, and OAuth access-ID revoke -> refresh-family denial. Revocation failure injection rolls back the complete transaction.
4. Exact clean checkout on main-managed `boxweave-test-ser2` independently rebuilt `f2af899` with Go 1.26.9/CGO1 and passed full `make verify`: format/YAML, tidy diff, vet, module verify and all five Go race packages. [Required quality](https://github.com/vanishs/boxweave/actions/runs/43408420432) and [S1 source checks](https://github.com/vanishs/boxweave/actions/runs/43408420414) on the **same code-bearing commit** both SUCCESS.

## Real isolated Guest and installed black-box

- Within existing nesting-enabled **`boxweave-test-ser2` only**, created fresh, separate Ubuntu 24.04 unprivileged inner Guest `hg-v1-oauth-1113`, native UUID `3cc315ad-7e02-4249-a784-6b36570d08e5`, 2 CPUs / 2GiB. Existing `demo-1-container` retained. Inner Incus 7.6 pinned signed package, fresh `user-1001` restricted nonroot project and dedicated ACL `bw-oauth-lab-acl-1113` were reviewed/applied; native prereq review **8/8 PASS**, including denied foreign default project. **Dir quotas remain UNVERIFIED**.
- Fresh `install-stage` + nonroot `doctor` + `install-activate` passed schema v2 and loopback MCP 401. After intermediate `00d429c` was upgraded, its installer-created rollback snapshot correctly prevented a second upgrade. Read-only inspection confirmed the service/binary had not changed. Per verified installer API, `install-commit` finalized that settled earlier upgrade; only then did the one reviewed upgrade to `f2af899` succeed with exact installed SHA and active service. The new rollback snapshot is deliberately retained pending further acceptance.
- The **installed earlier `00d429c`** exercised PTY-only local approval setup (no password displayed), real local HTTP RFC metadata, DCR, PKCE, consent redirect and token issuance; subsequent ad-hoc `initialize` request FAILED solely because the test client omitted the required `Accept: application/json, text/event-stream` header. A follow-up automated credential-bearing installed script was blocked by the tool guard; it was not bypassed. Consequently **no installed authenticated `tools/list` or actual MCP Box mutation is claimed**. Source-level official SDK authenticated flows passed separately.
- With **installed `f2af899`**, independently verified both RFC9728 metadata routes, AS metadata and RFC9207 flag, unauthenticated RFC9728 Bearer challenge, and **seven** fail-closed HTTP negatives (null/foreign Origin, forged Host, wrong path/method, invalid DCR, etc.). Installed service remained active and the restricted native Project empty.
- Native-only L2 smoke (before `f2af899` code change) used original `hg-oauth-l2-1113` UUID `dbcaad28-d353-4ac1-a60a-428be44fa8a6`: restricted Incus Init+Start, Python `print(42)` exit 0, STOP/DELETE verified by original UUID and final empty inventory. **This is not a v1 MCP Agent write I+N+V pass**.
- After local tests, the **test-only fictitious public origin was disabled** by the nonroot local CLI. Root metadata is now 404, wrong Host 403, the service remains loopback-only, and **no cloudflared process/public Tunnel was started**. The new installer rollback backup remains retained; old `boxweave-test-cli`, `bw-s25-l1a`, physical host, two original gateway UNKNOWNs and existing main services were untouched.

## OA acceptance matrix (scope matters)

| ID | State | Evidence / remaining condition |
| --- | --- | --- |
| OA-001 | LOCAL PASS | Single authenticated SDK `/mcp`, manual and OAuth bearer; 13 tools; omitted params and JSON transport |
| OA-002 | LOCAL + INSTALLED PASS | Two RFC9728 paths, 401 challenge, RFC8414/9207 issuer, hostile Host/forwarded headers denied |
| OA-003 | LOCAL PASS | Bounded DCR, exact redirect, code replay, S256-only PKCE, client/audience failure cases |
| OA-004 | BROWSER-SIMULATED PASS | Consent/CSP/Referrer/Origin/replay; **real ChatGPT browser not tested** |
| OA-005 | LOCAL PASS | SQLite atomic exchange/refresh, 12-way concurrency, DB failure, reopen, expiry, grant revocation |
| OA-006 | LOCAL PASS / NATIVE COMPOSITE PENDING | OAuth role mapping/viewer denial and unchanged native ownership checks; authenticated MCP write -> actual Incus not executed |
| OA-007 | LOCAL PASS | Local origin change/password reset revokes OAuth only; invalid config refused, installed disable postflight |
| OA-008 | BLOCKED | External Quick Tunnel HTTPS plus actual ChatGPT authorization/reconnect/refresh not tested; owner connection needed |
| OA-009 | BLOCKED | No authenticated remote Agent original-key MCP mutation + independent native I+N+V; native-only smoke is distinct |
| OA-010 | CODE-HEAD PASS; REPORT-HEAD RECHECK | `f2af899` independent build, race and both Actions SUCCESS; documentation-only follow-up must have its own required gate |

**Resumption:** do not re-run old gateway Creates or replay any UNKNOWN. Keep PR Draft. For real live acceptance, operator must deliberately enable a temporary Cloudflare Quick Tunnel inside a controlled Guest, locally configure that actual random HTTPS origin, privately set fresh approval credentials, connect the actual ChatGPT OAuth client and explicitly approve any mutating Agent test. Until then only bounded local OAuth acceptance can be signed.
