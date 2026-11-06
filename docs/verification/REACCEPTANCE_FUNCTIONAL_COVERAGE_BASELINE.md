# Functional coverage baseline for full BoxWeave reacceptance

**Status: acceptance-plan seed, not a newly executed test or current-version PASS.** This document merges the earlier [39-feature evidence-depth view](EVIDENCE_LAYERS_0449.md), [runtime entrypoint/call-chain and R-02/R-04 plan](../design/RUNTIME_CALL_CHAINS_0449.md), [test strategy and four client profiles](TEST_STRATEGY.md), [FEATURE_MATRIX](FEATURE_MATRIX.md) and [current outstanding risks](OPEN_RISKS.md). [The canonical full reacceptance runbook](../process/FULL_PROJECT_REACCEPTANCE.md) controls execution and [the run ledger template](REACCEPTANCE_RUN_TEMPLATE.md) controls outputs.

## Current selection rule — workflows first, action mapping second

**The purpose of this functional baseline is to locate the BoxWeave system behaviors most likely to break**, not to turn each MCP or CLI name into an isolated green test. Read [the current system-level scenario and invariant matrix](REACCEPTANCE_SYSTEM_SCENARIOS.md) and [canonical runbook](../process/FULL_PROJECT_REACCEPTANCE.md) before selecting tests. Map every feature to (a) an actual user workflow, (b) potential cross-interface bypass/race/degraded-state boundaries, (c) a real independent Provider/guest/receipt oracle and (d) a post-fix independent Verifier. **L1 full current MCP/CLI leaf coverage remains required**, but a leaf result never implies whole-feature PASS.

Use recent BoxWeave source diffs, old incidents and **observed running Containarium** (not a compiled compatibility minimum) to rank L2/L3 scenarios. The v0.128.0 upstream references in this document are **historical examples**; future runs fetch source/docs at the actual version's pinned GitHub release tag/commit into a private test workspace. Normal customer releases do not need those downloaded reference files or a test harness.

## How to use this seed

At every new run, **reconcile the complete current discovered MCP and CLI leaf-command catalogs against these feature rows**, plus Web/REST exposed actions and new version changes. Every MCP/CLI leaf gets its **own** black-box test-case record and a mapping to one or more functional IDs. Every functional ID maps back to all current implementing entrypoints and cross-entrypoint/sequence cases. Many-to-many is deliberate: 68 outer tools are not 68 feature groups; 39 historical feature groups are not a substitute for discovering today's tools.

The F01–F39 IDs are the **historical v0.44.9 classification** and provide test-selection ideas only. X01–X10 now include the v0.46.1 third-run extensions; extend beyond these IDs whenever the actual installed surface changes. Do not cap the inventory at 49 rows, hardcode historical MCP counts or promote past release PASS. Every historical case remains scoped to its original version. Unsupported/unavailable operations still need a distinct status, exact reason and next safe gate.

For each mapped entry, test a meaningful positive path, a deliberately malformed/boundary path and an unauthorized/target-mismatch/denial path (or justify why a particular dimension is inapplicable), then independently observe actual state, identity, receipts and cleanup. For each feature, also test stateful combinations and historical defect triggers. The owner-facing final report must list **all feature names**, actual interfaces/commands, intended behavior, concrete tests and observations, status and evidence links, plus what remained untested—never just a summary of passed/failed totals.

### Upstream-aware coverage extensions for existing feature IDs

Read the [pinned Containarium upstream evidence guide](UPSTREAM_CONTAINARIUM_REACCEPTANCE_GUIDE.md) and [execution architecture plan](../design/REACCEPTANCE_EXECUTION_ORCHESTRATION_PLAN.md) before selecting updated cases. Containarium documents are **source and technique references**, not inherited BoxWeave PASS labels. Track the upstream installed daemon/CLI/MCP/REST source and ref, the exact route/handler supported by the runtime, evidence state (implemented / opt-in / scoped / design-only / unverified), and how an upstream version/diff changes the test case and its owner.

| Existing feature group | Additional upstream-derived reacceptance obligation |
| --- | --- |
| F01-F03 / release and installer | Upstream `RELEASE-PROCESS.md` rehearsal patterns plus `DRILL-RECORD-TEMPLATE.md`: build-only candidate, installed prepublish check, exact provenance and rollback recovery **only in disposable test lanes** |
| F05/F24-F28 / Provider, MCP, REST and clients | Proto RPC/HTTP annotation + installed handler + CLI/forwarded MCP transport diff, source vs runtime route availability and typed errors; upstream CLI-first is **not** a BoxWeave architectural migration |
| F06-F12 / Box/SSH/nesting | Upstream source and SSH retry characterization: exact immutable identity, actual Provider effect/dial counts, stale target no-effect and guest readback |
| F18-F23 / Web/session/worker | Claim evidence and degradation tests: real authenticated positive, unauthorized no-effect, store readiness vs health, post-recovery fresh action |
| F29-F33 / R-02/R-04/UNKNOWN | Two-agent barrier, crash-after-intent uncertainty, exact single remote effect, source-level process and installed-client contention; retain original UNKNOWN without retry |
| F36-F38 / diagnosis | Upstream security CLAIMS and seeded docs-claims failures: distinguish implemented, enabled, design-only and unverified; intentionally invalid proof must be rejected |
| F39 / backup/restore | Real-host drill template as an **isolated planning reference only**; no direct owner-host restore, reboot or storage action |
| X01/X02/X05/X06 / CLI, selftest, error mapping, Compose | Compare real installed CLI/help/dispatched commands; capture unique selftest actor outputs; compare declared Compose protobuf routes/native handler vs forwarded MCP 404; preserve upstream attribution uncertainty |

Optional **upstream Proof Pack** supports execution provenance for reviewed source tests, not MCP action coverage or independent Provider side-effect proof; it requires a separately authorized nested Provider host and an owned child, since `containariumd proof-pack generate` executes child commands as root. There is **no direct BoxWeave MCP Proof Pack generator** established by the v0.128.0 documentation/runtime check. The [run template](REACCEPTANCE_RUN_TEMPLATE.md) records proof mode and all missing gates explicitly.

## Feature-by-feature adversarial acceptance baseline

### Delivery

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| F01 | **Immutable release and asset verification** | Pinned tag/manifest/digest, tampered asset, missing asset, mismatch refusal; compare release bytes and provenance. |
| F02 | **Clean install and exact-version upgrade** | Fresh test Box vs seeded prior release; package/worker/runtime match, preserved state, install and upgrade exits. |
| F03 | **Interrupted install, schema migration and repair** | Bounded 5xx/timeout/partial install, migrations, busy lock, rollback/repair; no duplicate side effects or lost data. |
| F04 | **Doctor, provenance, authenticated MCP/REST health** | Actual installed binary and HTTP/MCP read; bad token, missing service and conflicting version; distinguish ready vs merely reachable. |
| F05 | **Upstream version compatibility and managed upgrade** | Old/minimum/newer/unknown Provider; fail-closed on untrusted mode; separate host/root upgrade authorization. |

### Box lifecycle

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| F06 | **List/get/exact target identity** | Valid/absent/stale/reused logical names; immutable UUID/createdAt and target-bound errors. |
| F07 | **Create/start/stop/delete** | Fresh real Box, invalid resources, duplicate name, stop/start persistence and confirmed deletion with independent Provider readback; two-actor create/delete name/identity collision with no second unintended effect. |
| F08 | **CPU/memory resize** | Live growth and restore, invalid units/limits, concurrent change, guest cgroup and Provider state. |
| F09 | **Disk no-shrink protection** | Known shrink rejected before dispatch with zero upstream write; distinguish BoxWeave guard from native Provider behavior. |
| F10 | **Nesting and child workload** | Actual nested authenticated MCP create, guest Python stdout/exit, cleanup and absence; test two concurrent confirmed selftests with first visibly in-flight, second rejected with zero extra child, then post-contention success; not native Incus alone. |
| F11 | **SSH connect(exec) and trust/identity** | Exact guest/user/host, failing command exit, stale key/host override, restart readiness without trust bypass. |
| F12 | **Bounded guest file and directory reads** | Regular file/page/limit, symlink/traversal/outside-home refusal, source bytes and metadata match. |
| F13 | **Nesting inspect/plan/confirm** | Current setting vs proposed change, invalid identity, new-child proof only where authorized; do not equate configuration with capability. |

### Network/access

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| F14 | **Tenant, guardrails, atomic policy/ACL** | Allowed and rejected Tenant pairs, stale revision, cross-tenant negative packet probe and rollback of test-owned policy. |
| F15 | **Port exposure, route and revoke** | Publish/duplicate/conflict/wrong target/revoke; independent listener/traffic oracle, clean exact route. |

### Runtime extras

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| F16 | **Auto-sleep, idle stop and wake** | Real idle timer transition, HTTP wake first response, unauthorized wake denial and exact release version. |
| F17 | **Monitoring/OTel and metrics export** | Absent collector refusal vs configured positive receipt; no metrics export claimed from a configuration toggle alone. |

### Web/app

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| F18 | **Authentication, sessions, Origin and CSRF** | Normal installed login, expiry/logout, forbidden actor/wrong Origin, no unauthorized mutation. |
| F19 | **Plan→confirm→worker→receipt lifecycle** | Authenticated installed Web positive and negative, REST/MCP path choice, Provider identity and receipt consistency; stale/parallel confirm and worker interruption after dispatch intent must not duplicate remote effects. |
| F20 | **Projects, binding, receipts and history** | Create/bind/update/stale revision, duplicate actor/restart, historical receipt integrity and pagination. |
| F21 | **Prompt graph editor/copy/export** | Installed persistence and readback, invalid graph/permission/copy failure, not merely Chromium fixture. |
| F22 | **Model wizard and native client handoff** | Selection/confirmation/cancel, ownership split, generated client runtime; inference only if separately authorized. |
| F23 | **Tenant/port management in authenticated Web** | Plan/confirm/observe, denied revision/actor, actual network/policy state and no hidden out-of-band mutation. |

### MCP transports

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| F24 | **SSH/stdio MCP Gateway** | Real initialize/initialized/tools/list, every exposed tool exercised separately; forwarded vs native and allowlist drift. |
| F25 | **HTTP MCP and OAuth credentials** | Real initialized client, bearer/link flow, expiry/revocation, wrong scope and no side effects on denial. |
| F26 | **HTTPS/public-origin external OAuth** | Real TLS chain/Origin, OAuth consent+PKCE+token and independent client; public exposure needs separate approval. |
| F27 | **Native/WSL external client profiles** | Actual chosen Windows/WSL client registration, SSH argv/keepalive/trust, open tool and read; source generator alone insufficient. |
| F28 | **MCP-first/REST-assisted Provider parity** | Actual app REST lifecycle and specialist MCP behaviors, identity/error/receipt semantics; do not assume all REST parity. |

### Coordination

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| F29 | **R-02 durable tasks/dedup** | submit/get/list, same key/body, changed body conflict and two simultaneous writers; one original remote effect and stable receipt; reboot persistence only when owner-authorized. |
| F30 | **R-04 shared claims/lease/protected Gateway** | Explicit test-only enablement, lease conflicts, invalid/expired tokens, unrelated Box forward progress, zero competing mutation. |
| F31 | **R-04 Web-first/SSH-first/HTTP-first races** | All three ingress orderings, exact incarnation, two-Agent interleavings, independent Provider/guest/claim oracles. |
| F32 | **UNKNOWN after dispatched work/server interruption** | Kill test-only worker near intent and after effect, retain original receipt/claim, no blind replay; controlled recovery separately. |
| F33 | **Trust readiness after create/restart** | Real first failure→pending→ready or recorded absence of reproduction; never infer from a successful later connect. |

### Native Provider

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| F34 | **Create after error and same-name outcome** | First request error/side-effect correlation; compare duplicate/invalid/missing-resource errors across Gateway and native Provider without treating upstream REST 500 as a proven BoxWeave root cause; test original UUID unchanged. |
| F35 | **Native disk shrink contract** | Distinguish upstream direct shrink and BoxWeave prevention; risky native shrink requires an expressly authorized disposable target. |

### Diagnostics

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| F36 | **D0/D2 doctor/trace/receipt** | Known-good and known-bad diagnostics, exact trace binding, redaction and stable typed error. |
| F37 | **D3/D4 classified and scoped project diagnostics** | Allowed exact target positive and default-deny negative; no unauthorized raw log or secret exposure. |
| F38 | **DC-01 preflight/rest_read/operation_trace** | Independently authenticated REST and MCP reads, missing config/token/trace negatives and no false health claim. |

### Recovery

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| F39 | **Backup/restore guard, snapshot and clone scope** | Test default denial and approved exact target only; distinguish shipped capability from upstream proposal/unsupported operation. |

### Current extensions

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| X01 | **Actual installed CLI command discovery** | Compare source dispatch (including pre-parser branches), installed real callable leaves and printed help: upstream/http/task/job/selftest plus new commands; deliberate hidden-command negative control; verify each safe leaf by actual invocation. |
| X02 | **Live selftest plan/run** | Private plan/confirmation mismatch, create→hello→stop→delete and Incus absence; two distinct in-flight actors with second correctly refused, zero extra children and next run succeeding; immutable per-actor exit/stdout/stderr; separately investigate initial/later receipt contradictions; never repeat unknown writes. |
| X03 | **debug_container REST/MCP projection parity** | Real installed authenticated REST debug and legacy MCP path, identity pin, redaction, bad token/unknown Box. |
| X04 | **Release CDN fallback** | Bounded CDN 502/503/504 fallback with exact asset ID/hash; tampered payload must not launch. |
| X05 | **Missing-target error contract** | Known absent Box through native Provider, Gateway and CLI; compare NotFound/500 translation without accidental create. |

| X06 | **Compose discovery, auto-start and status** | On test-owned Box create real valid compose.yaml; call discover/status/enable/disable through the *actual* installed forwarded MCP, compare native upstream support at pinned version, user systemd units and clear 404/unsupported semantics; preserve issue #79 while awaiting verified upstream fix. |
| X07 | **GitHub ephemeral runner management** | Provision/list/remove a disposable scoped runner only with separately approved repository credential, bounded budget and teardown; no PAT means BLOCKED, not no-functionality PASS. |
| X08 | **Security sentry, finding store and process policy** | Positive test-owned finding, safe negative pattern, rule receipt and explicit disabled/HTTP503 cases; no production monitoring/policy changes. |
| X09 | **Code execution, logs and permission journal** | Start a bounded test-owned run, verify stdout/exit and approved/denied permission boundaries, list/logs/pending and no orphan job after cleanup. |
| X10 | **GPU/backend capability and recipe deployment** | Read and validate backend capabilities; an actual GPU deployment requires a compatible authorized isolated target; unsupported/no-GPU is documented BLOCKED. |

### Mandatory changed-boundary scenario selection

Use this matrix as **test-selection input**, not a release-independent PASS registry. A source diff should increase risk weight for directly changed modules and all their callers, routes, side-effect adapters, shared state, previous defects and concurrency neighbors.

| Change or historical discovery | Minimum extra adversarial cases | Independent oracle |
| --- | --- | --- |
| Added/renamed/hidden CLI or MCP operation | Source dispatcher vs installed actual invocation vs global/subcommand help, per policy/transport | Real executable/tool list and typed exit, not only parser AST |
| Durable SQLite check/insert, plan/confirm, operation/lease state | Two separately scheduled processes/Agents, in-flight precondition, competing refusal/dedup, original finish, new operation after completion | Transaction/journal plus Provider exact side-effect count and guest/Incus target identity |
| MCP-to-REST or Provider upstream version/route change | For same supported action compare Gateway, selected REST/MCP, CLI/Web, missing/duplicate/invalid target and native error owner | Exact Provider route/version, typed status, remote object state; a 404 alone does not establish cause |
| Selftest failure or ambiguous shared result | Unique output paths/producer IDs, timestamped immutable exit, comparison to durable journal and final child absence; repeat serial and controlled concurrent cases separately | Original stdout/stderr and outer readback, no file overwrite or blind replay |
| Installer/release/version/package change | Freeze candidate digest, clean or seeded isolated install, original installed regression **before** public release, then release asset hash/source and official installer smoke | Installed package/runtime source, SHA256SUMS, tag, healthy services and lifecycle outcome |
| Authentication, Web, HTTP MCP, policy, Tenant boundary | Real positive authenticated permitted path *and* denied path in explicitly authorized isolated environment; packet/receipt where meaningful | Actual protected endpoint + independent state; fixture 401 is negative only |

For the v0.46.2 selftest incident, retain the distinction between an **installed-verified concurrency admission repair** and the historically ambiguous immediate-unreconciled/later-passed output observation. The latter must stay a separately tracked question unless the same symptom is explained by evidence. For Compose #79, waiting on upstream is an owner-approved deferral, not a positive acceptance or confirmed native root cause.


## Previous acceptance schemes that remain mandatory input

| Historical scheme | Carry forward into the current run, not as a PASS |
| --- | --- |
| [Five-layer evidence matrix](EVIDENCE_LAYERS_0449.md) | Separate source S, fixtures T, isolated installed I, authorized operator-host H, actual external client X; preserve scoped/historical statuses. |
| [Runtime call-chain view](../design/RUNTIME_CALL_CHAINS_0449.md) | Partition MCP Gateway, HTTP MCP, Web/worker, app CLI, native Provider/Incus; identify REST/MCP selection and uncoordinated bypass. Refresh against current version. |
| [R-04 UNKNOWN recovery](R04_UNKNOWN_RECOVERY_0450_2026-11-05.md) and [three-entry conflict matrix](REMEDIATION_TEST_QUEUE.md) | SSH/HTTP/Web writer ordering, original dispatch→UNKNOWN, claim/lease and operator audit; keep historical original tasks untouched. |
| [Test strategy](TEST_STRATEGY.md) | Four actual native/WSL client profiles, threat/failure-path obligations, install/repair, persistence and web/browser vs live boundary. |
| [Current v0.46.1 black-box run](REACCEPTANCE_0461_2026-11-05_BLACKBOX_RUN2.md) | Retest known CLI-help discoverability and missing-target error classification; old 15 sampled calls do **not** grant green status to remaining tools. |
| [Current source/feature matrix](FEATURE_MATRIX.md), [risks](OPEN_RISKS.md), [queue](REMEDIATION_TEST_QUEUE.md) | Map AC/requirements to current behavior, unverified or unsupported scope, original risk evidence and next-Agent packets. |

**No authorization expansion:** R-02 owner-host reboot remains deferred. Do not enable protected R-04 on the owner host, alter production credentials/SSH/Incus/Tenant/network policy, release historical UNKNOWN claims, force an upstream shrink or expose public TLS without specific authority. Instead write per-action BLOCKED/SKIPPED/UNKNOWN records and continue other safe checks.

## Required owner-facing detail level

An acceptable final summary includes (for **each** feature in the current expanded functional catalog) `Feature ID + plain-language behavior | MCP tools/transport and CLI leaf actions | attempted successful/invalid/denied cases | actual observed effects | status/evidence class/version | defects/open restrictions | linked case IDs`. Each per-action technical row adds `exact invocation/inputs or sanitized command | expected/actual output/exit | independent oracle | cleanup/release state`. If output is too large for the conversation, provide the full checked-in feature table and per-action ledger with direct links, **and** show a readable grouped feature table in the response. A totals-only response fails the reporting gate.
