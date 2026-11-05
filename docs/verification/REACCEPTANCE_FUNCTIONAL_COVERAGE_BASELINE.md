# Functional coverage baseline for full BoxWeave reacceptance

**Status: acceptance-plan seed, not a newly executed test or current-version PASS.** This document merges the earlier [39-feature evidence-depth view](EVIDENCE_LAYERS_0449.md), [runtime entrypoint/call-chain and R-02/R-04 plan](../design/RUNTIME_CALL_CHAINS_0449.md), [test strategy and four client profiles](TEST_STRATEGY.md), [FEATURE_MATRIX](FEATURE_MATRIX.md) and [current outstanding risks](OPEN_RISKS.md). [The canonical full reacceptance runbook](../process/FULL_PROJECT_REACCEPTANCE.md) controls execution and [the run ledger template](REACCEPTANCE_RUN_TEMPLATE.md) controls outputs.

## How to use this seed

At every new run, **reconcile the complete current discovered MCP and CLI leaf-command catalogs against these feature rows**, plus Web/REST exposed actions and new version changes. Every MCP/CLI leaf gets its **own** black-box test-case record and a mapping to one or more functional IDs. Every functional ID maps back to all current implementing entrypoints and cross-entrypoint/sequence cases. Many-to-many is deliberate: 68 outer tools are not 68 feature groups; 39 historical feature groups are not a substitute for discovering today's tools.

The F01–F39 IDs are the **historical v0.44.9 classification** and provide test-selection ideas only. X01–X05 cover more recent needs and can be expanded with dynamically discovered feature IDs; do not cap the new inventory at 44 rows, hardcode historical MCP counts or promote past release PASS. Unsupported/unavailable operations still need a distinct status, exact reason and next safe gate.

For each mapped entry, test a meaningful positive path, a deliberately malformed/boundary path and an unauthorized/target-mismatch/denial path (or justify why a particular dimension is inapplicable), then independently observe actual state, identity, receipts and cleanup. For each feature, also test stateful combinations and historical defect triggers. The owner-facing final report must list **all feature names**, actual interfaces/commands, intended behavior, concrete tests and observations, status and evidence links, plus what remained untested—never just a summary of passed/failed totals.

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
| F07 | **Create/start/stop/delete** | Fresh real Box, invalid resources, duplicate name, stop/start persistence and confirmed deletion with independent Provider readback. |
| F08 | **CPU/memory resize** | Live growth and restore, invalid units/limits, concurrent change, guest cgroup and Provider state. |
| F09 | **Disk no-shrink protection** | Known shrink rejected before dispatch with zero upstream write; distinguish BoxWeave guard from native Provider behavior. |
| F10 | **Nesting and child workload** | Actual nested authenticated MCP create, guest Python stdout/exit, cleanup and absence; not native Incus alone. |
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
| F19 | **Plan→confirm→worker→receipt lifecycle** | Authenticated installed Web positive and negative, REST/MCP path choice, Provider identity and receipt consistency. |
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
| F29 | **R-02 durable tasks/dedup** | submit/get/list, same key/body, changed body conflict, reboot persistence only when owner-authorized; exact one guest effect. |
| F30 | **R-04 shared claims/lease/protected Gateway** | Explicit test-only enablement, lease conflicts, invalid/expired tokens, unrelated Box forward progress, zero competing mutation. |
| F31 | **R-04 Web-first/SSH-first/HTTP-first races** | All three ingress orderings, exact incarnation, two-Agent interleavings, independent Provider/guest/claim oracles. |
| F32 | **UNKNOWN after dispatched work/server interruption** | Kill test-only worker near intent and after effect, retain original receipt/claim, no blind replay; controlled recovery separately. |
| F33 | **Trust readiness after create/restart** | Real first failure→pending→ready or recorded absence of reproduction; never infer from a successful later connect. |

### Native Provider

| ID | Feature / actual user goal | Minimum adversarial black-box scenario and independent evidence |
| --- | --- | --- |
| F34 | **Create after error and same-name outcome** | First request error/side-effect correlation; old incident is historical, new independent native run on disposable Box only. |
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
| X01 | **Actual installed CLI command discovery** | Recursive CLI/help/parser enumeration including special dispatch: http/task/selftest; verify each leaf by real invocation. |
| X02 | **Live selftest plan/run** | Private plan/confirmation mismatch, create→hello→stop→delete and Incus absence; avoid repeating unknown writes. |
| X03 | **debug_container REST/MCP projection parity** | Real installed authenticated REST debug and legacy MCP path, identity pin, redaction, bad token/unknown Box. |
| X04 | **Release CDN fallback** | Bounded CDN 502/503/504 fallback with exact asset ID/hash; tampered payload must not launch. |
| X05 | **Missing-target error contract** | Known absent Box through native Provider, Gateway and CLI; compare NotFound/500 translation without accidental create. |

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
