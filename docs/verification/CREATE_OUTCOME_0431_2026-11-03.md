# v0.43.1 post-dispatch error classification and Agent create ambiguity

**Scope:** BoxWeave's existing application lifecycle and outward SSH/HTTP Agent MCP Gateway. This change does **not** patch Containarium's underlying create implementation, enable the optional shared coordinator, change upstream binaries, or automatically retry a failed create. The user authorized an independently tested patch release before updating their own host.

## Reproduced defect

The preceding [current-main audit](REACCEPTANCE_0430_2026-11-03.md) observed an actual `create_container` reply of HTTP 500 / `Instance already exists`, followed by an exact-original-name read showing a running, newly created Box. The precise upstream cause is still unknown and must not be guessed.

Code review found a **separate concrete BoxWeave defect** in `Lifecycle.work_one`: `PROVIDER_REJECTED` was classified as a definite pre-dispatch failure in the known-error set. The MCP transport generates this code *after* a rejected tool result or JSON-RPC response, which may arrive after a remote write committed. The old classification wrote `failed / not_submitted`, incorrectly let the application treat the original attempt as terminal, and could mislead callers into treating an uncertain write as safely finished. Existing shared-claim code conservatively held the claim, but the application receipt was still incorrect. Existing `UNKNOWN` reconciliation was bypassed.

Two new side-effect-after-error tests were deliberately run against the original source and failed:
- `test_provider_rejected_after_create_side_effect_stays_unknown`: actual code returned `failed / PROVIDER_REJECTED / not_submitted` instead of `unknown / OUTCOME_UNKNOWN / submitted`.
- `test_upstream_rejected_after_create_keeps_shared_claim_unknown`: actual code returned `failed / PROVIDER_REJECTED` and did not retain the correct unknown-result classification.

## Repair

1. **App lifecycle:** Remove `PROVIDER_REJECTED` from the proven-pre-dispatch failure set. Return a durable `UNKNOWN` receipt when a completed Provider call rejects, preserving the already-recorded resource claim and original request ID. Subsequent inspection reports only the observed candidate and `completion=unverified`, without auto-adoption or replay.
2. **Outward Agent Gateway:** On a forwarded `create_container` error, retain the original JSON-RPC error code/message and add `boxweave_create = {state:unknown, code:CREATE_OUTCOME_UNVERIFIED, logical:<validated exact name>, inspect:get_container, retry:not_authorized}` in typed error data. For `result.isError`, add a bounded textual first content item while retaining `isError=true` and the original content. Successful creates are unchanged. The gateway **never** silently performs a second create, reads private credentials, treats an existing resource as proof this particular request created it, or alters Containarium.
3. **Version boundary:** Update `VERSION`, Python, Web and gateway-server version to `0.43.1`. Preserve immutable v0.43.0 assets and the separately selected stable default until the new tagged release and isolated installed acceptance pass. This also resolves the preceding source-only `scripts/browser_check.py` main publish refusal: the publisher correctly requires a new version for executable source changes.

## Acceptance matrix

| Gate | Grade/status | Exact proof or remaining check |
| --- | --- | --- |
| App's post-commit MCP response error | T PASS | Added lifecycle regression reproduces old failure then asserts `UNKNOWN`, observed candidate, no guest replay, no automatic adoption |
| Shared-claim retention after upstream rejection | T PASS | Added claim-store regression asserts `unknown`, another Agent's claim refused until explicit recovery |
| Agent-facing create error contract | T PASS | Added gateway transport tests assert preserved upstream error and typed unknown hint, one forwarded create, no hidden readback |
| Other Provider REST/SSH and fixture paths | **T + CI PASS** | On prepared audit Box, `scripts/validate.py --browser` returned exit **0**, persisted **10/10 PASS** (knowledge, **703/703 Python tests**, TypeScript/Vite build, seven Chromium browser fixtures). Evidence: `.runtime/validation/9d6d8aad7c744b1387f1623c7bd9b8c2/result.json`; npm locked install passed. This is fixtures/source, not installed Provider acceptance |
| Real installed service and new release | **L PASS in isolated Box** | [Main CI 41776913234](https://github.com/vanishs/boxweave/actions/runs/41776913234) verify/publish PASS, immutable v0.43.1 tag matches `eadcb27db5fec7e77bcb905c0452e58c9c87a3a0` and 23 digested assets. Authentic tagged installer upgrade on task-owned Box `bwaudit-0430-1103` exit 0; `boxweave version` 0.43.1, `doctor --mcp` 10/10 PASS, REST/MCP authenticated reads PASS, 68-tool installed Gateway, DC-01 preflight/rest_read PASS, unauthenticated local HTTP MCP 401. Installed gateway SHA-256 `e495207a8379446e375aa87637da9d4da081f6491b6a638d3878c58710821603` equals published release asset. Installed safe-helper error injection preserved upstream code and yielded UNKNOWN/no retry. No forced real Provider 500 fault injection was done |
| Original user's host and external clients | U / deferred | Do not update the operator's host; they will update separately and request recheck. Public OAuth/TLS and desktop clients were not enrolled |
| Root upstream create 500 | **OPEN** | Containarium v0.120.1 root cause unverified; this release repairs **unsafe BoxWeave error interpretation**, not upstream creation itself |

**Rollback:** Immutable prior v0.43.0 remains installable by its exact release tag; follow the installer-managed backup/upgrade guidance and never overwrite an active DB with a live copy. New reply metadata is additive; older clients continue to receive the underlying error. After any UNKNOWN, inspect the **original** name/incarnation before deciding whether a write can be safely retried. No automatic host/provider upgrade or private token movement is included.

## Published source and isolated host acceptance

[Stable release promotion evidence](STABLE_PROMOTION_0431_2026-11-03.md) holds separate exact source/tag, baseline/upgraded versions, live MCP, SHA-256 and the default-installer-promotion gate. The installed host is a disposable project test Box, **not** the user's operator host.
