# v0.50.0 isolated installed REST/async-create and R-04 concurrent acceptance — 2026-11-07 UTC

**Evidence:** actual official Release installed on an isolated Box, actual authenticated local Gateway/Provider REST requests, independent nested Incus and private durable SQLite readback, executed by the **same Coordinator Agent**. **NOT** independent multi-Agent acceptance; **NOT** external HTTP MCP/OAuth/TLS acceptance; **NOT** a source fixture. **Overall: FAIL / PARTIAL, not installed PASS.**

## Immutable release, target and preparation

- Exact tested official version: **BoxWeave 0.50.0**, source tag `29f75410fe523586b7fb27373081e084a36ac9ea`; GitHub Release non-draft, 32 assets.
- Isolation: existing test-owned parent `bwracc-mvp-iv3-container`, which previously had BoxWeave 0.49.0 and working nested Incus with existing child `bwracc-iv3-life-a-container`. Existing parent Incus UUID `06a0922a-4ef9-450e-9561-7bd15c9422db` (createdAt `1793939346`); parent was not recreated. No owner production-host upgrade, stable-pointer movement, or changes to unrelated old UNKNOWN #88/#92.
- Official `install.sh --version 0.50.0` pulled from the immutable tag into the isolated guest; downloaded installer SHA-256 `9945075e0958c93aa91bc0acfd5de31ba8fe09dccce77bd26ccf774b5fd1bcd5`. Guest installer exit marker `0`; `boxweave version` then reported `0.50.0`; compatible Containarium **0.128.0** was retained.
- `sudo boxweave doctor --mcp` post-install: **10/10 passed**. Local authenticated Gateway `tools/list` had **23** visible tools, including the eight fixed REST lifecycle/read capabilities and both receipt-status helpers.
- The installed private app was in `mode=mcp` with exact local Provider REST URL, private token-file reference and private app DB. The published Provider/backend/project scope matched app config. `sudo boxweave task provision` initialized an empty app-user-owned R-04 claims database; initial claims operations/resources/audits were zero.
- **Test-only opt-in** on this isolated guest: a backup of app config was saved as `/home/boxweave/boxweave/config.v0500-precoord.json`; app config enabled `coordination.enabled=true`; app worker and Web user services restarted; root-owned policies `coordination-policy.json` and `lifecycle-rest.json` installed in guest's `/var/lib/boxweave-default/public/`, file mode 0640 with group `boxweave`. The REST file references private paths only; no token body was copied. The subsequent doctor passed again.
- Installed Gateway calls were made through `bwmini.remote_mcp.relay` as the actual `boxweave` Unix principal, invoking the deployed protected `mcp_gate`. This proves **local real Gateway execution**, not the external HTTPS/OAuth request journey.

## A. Async REST create — **FAIL at pre-dispatch preflight**

- Exact fresh name: `bw50qa-async-a`. One original `create_container` with 1 CPU, 1GB, 10GB, Podman false, nesting true and an explicit unique request key returned JSON-RPC `-32602 TRUST_OR_WORKLOAD_KEY_GATE` in ~0.753 s. No accepted `REST_CREATE_ACCEPTED` or `REST_GUEST_READY` receipt.
- Original-name read-only status and Provider get did not yield a valid target. Private app `remote_create_tasks` and `remote_create_options` tables both had **zero** rows; independent native nested `incus list` had no matching target, so this original attempt has no observed creation or durable dispatch.
- Installed code's reviewed root policy, protected selection, managed public key readability, normalized POST route and application journal validation all passed as the app user. Actual Provider REST `GET /v1/containers/bw50qa-async-a` returned **HTTP 500** with JSON content, classified by the installed `rest_wire` as `API_HTTP_REJECTED`. `gateway_create.dispatch` requires `API_TARGET_NOT_FOUND` (HTTP 404) to establish verified absence and rejects all other codes with `REST_CREATE_PREFLIGHT_UNVERIFIED`; the outer gateway error was generic. Authenticated `list_containers` and `get_container` for an actual existing nested child succeeded.
- This is a real **unhandled Provider 0.128.0 absent-target status incompatibility** on this installation; root cause of the upstream HTTP 500 itself remains unconfirmed. **Do not** generally translate HTTP 500 to absent or bypass the preflight: those would break duplicate-write safety.
- Tracking: [Issue #104](https://github.com/vanishs/boxweave/issues/104).

## B. R-04 concurrent claim — **denial no-side-effect PASS, lifecycle FAIL**

- Exact test-owned inner target: `bwracc-iv3-life-a-container` (logical `bwracc-iv3-life-a`), running, native Incus CPU **1**, createdAt `1793940007`. Initial shared claims and `remote_rest_tasks` zero.
- Actual protected Gateway process **A** acquired a 120-second exact-identity lease; `state=leased`. The original token was held only in private process memory and never printed.
- While held, an independent installed Gateway process **B** called the **REST-selected** `resize_container` with CPU **2** and a new 32-hex key. Actual response: generic JSON-RPC `-32602 TRUST_OR_WORKLOAD_KEY_GATE`, not a typed resource-busy rejection. Lease `status` remained `leased` until **A** explicitly released it; release returned `state=released`.
- Native `incus config get bwracc-iv3-life-a-container limits.cpu` was **1 before and after**. Native Box stayed RUNNING. This establishes no forbidden CPU side-effect under the lease.
- **Critical installed bug:** private `remote_rest_tasks` now contained a single row for the denied write, task `c65df49940771e147cbf7e0b18a2386d`, logical `bwracc-iv3-life-a`, action `resize`, `state=unknown`, `code=REST_OUTCOME_UNKNOWN`. The installed `boxweave_rest_status` independently read the same `unknown` and `retry=not_authorized`. Private claims store had **0** operations/resources/leases/audits. A read-only `MutationJournal.active` check for a different action then raised `REST_OPERATION_PENDING`; thus a known pre-dispatch lease denial has poisoned subsequent normal admission on the target.
- Source candidate: `bwmini/rest_mutations.py` `dispatch()` persists `MutationJournal.reserve` **before** `SharedClaims.acquire`; a pre-dispatch claim denial executes `ledger.unknown` and raises `REST_PREDISPATCH_UNVERIFIED` (lines ~257-270). Preserve true dispatched UNKNOWNs, but do not treat a known refused claim as uncertain upstream work.
- **No** repeat of the denied request, no privileged force-release, no manual deletion of the status row, no mutation of older historical UNKNOWN cases. Preserve this single new UNKNOWN as original issue evidence.
- Tracking: [Issue #105](https://github.com/vanishs/boxweave/issues/105).

## C. Separate outer-host legacy MCP async control (not v0.50 REST)

The controlling outer BoxWeave MCP accepted exactly one `create_container` for uniquely owned `bw0500-1106-qa`, task `263e80d5977579bfb6728fddb1b3c275`. Its durable receipt moved `submitted -> succeeded` (`UPSTREAM_CREATE_ACKNOWLEDGED`), while `guest_user_and_ssh_verified=false` remained an honest upstream-only receipt flag. The native Box progressed STOPPED -> RUNNING (createdAt `1794023897`), nesting was configured true, and subsequent independent actual guest `connect(exec)` succeeded after transient `TRUST_NOT_READY`. This checks **legacy outer wrapper behavior only**, not the newly installed v0.50 REST create worker.

## Named installed verification matrix

| Feature | Verdict | Evidence / required follow-up |
| --- | --- | --- |
| Exact official v0.50 install / preserved Provider | **PASS** | Installer exit 0, `boxweave version`, Provider 0.128.0 |
| Post-install app, HTTP service and local MCP diagnostic | **PASS** | Doctor 10/10 |
| Root-selected coordinated Gateway catalogue | **PASS** | Actual 23-tool installed local `tools/list`; eight REST capabilities visible |
| REST authenticated list and existing GET | **PASS** | Actual installed Gateway -> Provider reads |
| REST missing-target GET strict absence contract | **FAIL** | Observed HTTP 500 not 404, Issue #104 |
| REST async POST `CREATING` / receipt / guest readiness / nesting | **BLOCKED** | Preflight fails before POST, Issue #104; no false PASS |
| Same-name duplicate REST create / restart / >120s | **NOT TESTED** | No accepted original task to deduplicate |
| Lease acquisition/status/release for actual nested Box | **PASS** | Distinct Gateway calls, no retained lease |
| Competing REST resize no-effect while leased | **PASS (scoped negative)** | Native CPU stays 1; distinct Gateway process |
| Correct typed R-04 denial and subsequent clean admission | **FAIL** | Permanent denied-write UNKNOWN and `REST_OPERATION_PENDING`, Issue #105 |
| Web vs SSH vs HTTP multi-ingress F30-F32 with fault injection | **NOT TESTED** | Cannot claim multi-Agent / independent Verifier PASS |
| True dispatched UNKNOWN recovery and audited manual claim release | **NOT TESTED** | Preserve test UNKNOWN; no unreviewed recovery |

## Disposition / safety

- Scope remains isolated `bwracc-mvp-iv3`; no owner production-host deployment, no default stable pin changes, no source release.
- Keep the v0.50 test lab and its single task-owned new R-04 UNKNOWN row for independent debugging. The preexisting inner test child was neither stopped, deleted nor resized.
- The separate outer control test Box `bw0500-1106-qa` is running; retained for explicit test-owner cleanup rather than conflating it with the failed v0.50 REST create.
- No credentials or lease token exposed; no UNKNOWN replay or host reboot.
- **Final gate: v0.50.0 installed independent REST async/R-04 acceptance is NOT PASS.** Fix #104 and #105, then rerun on a separate installed Verifier environment with distinct actors and the exact preflight/lease race original repros.
