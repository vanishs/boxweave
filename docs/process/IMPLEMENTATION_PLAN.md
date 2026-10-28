# Implementation plan — authorized design, future work only

Phase 3 originally defined the tasks below. Phase 4 was authorized with its concrete scope (SRC-030); current progress is in the table at the end, not an assertion that all original completion conditions passed. After TASK-005 the developer must accept the minimum system before phase 5. Included release features are not dropped because the first slice is small.

## Ordered tasks

Each task must update its named documents, current PHASE_STATUS and HANDOFF in the same implementation change; attach actual command, environment, stdout/exit and evidence location. Planned tests below are obligations, not existing files or executed commands. Priority P0 = minimum-system dependency, P1 = selected release scope after the phase-4 gate. Dependencies are a DAG, not a promise of parallel agents.

| Task | Phase / priority / dependencies | REQ / AC | Input and output | Repeatable verification and completion | Documentation |
| --- | --- | --- | --- | --- | --- |
| TASK-001 | 4 / P0 / none | REQ-004, REQ-010, REQ-017; AC-004, AC-010, AC-017 | Approved contracts + fresh project LXC inventory; deliver minimal package, typed target/error/evidence contracts and fake provider, no generic plugin framework | Non-root Linux import/contract suite; malformed target/unknown identity rejected. Lock compatible dependencies and record tested versions. Ready only when fake boundary cannot hit real provider accidentally | ARCHITECTURE, TEST_STRATEGY, dependency/provenance notes |
| TASK-002 | 4 / P0 / TASK-001 | REQ-005, REQ-006, REQ-007, REQ-010; AC-005, AC-006, AC-007, AC-010 | Local schema/revisions + single worker intent/receipt repository; deliver create/confirm/observe/reconcile contracts and crash recovery | Real temporary SQLite transactions, stale CAS, two worker claim race, DB-write failure, clock rollback and crash-window tests; no replay after dispatch marker; receipt survives process restart | ADR-002, schema/recovery notes, TEST_STRATEGY |
| TASK-003 | 4 / P0 / TASK-001 | REQ-001, REQ-002, REQ-003, REQ-016, REQ-018; AC-001, AC-002, AC-003, AC-016, AC-018 | Trusted descriptor + Windows Codex/native profile; deliver minimal safe enrollment/diagnostic and reviewed provider identity/connect parser | Config preservation/replacement, pin mismatch, payload tamper and malformed MCP fixtures; no top-level privileged setup during syntax checks. Real enrollment only on a reviewed client/target; no credential export | ADR-003, profile/trust docs, TEST_STRATEGY |
| TASK-004 | 4 / P0 / TASK-001, TASK-002 | REQ-010, REQ-011, REQ-018; AC-010, AC-011, AC-018 | Minimal Web/CLI adapters over same services; local admin, one project/binding page and receipt view | Browser + real SQLite/fake provider: login/expiry/CSRF/Origin, stale binding edits, fixed harmless action only, reload receipt without replay. No arbitrary shell endpoint; unauthenticated target data denied | ARCHITECTURE, API/schema/session docs, TEST_STRATEGY |
| TASK-005 | 4 / P0 / TASK-002, TASK-003, TASK-004 | REQ-002, REQ-003, REQ-004, REQ-006, REQ-007, REQ-010, REQ-011, REQ-017; AC-002, AC-003, AC-004, AC-006, AC-007, AC-010, AC-011, AC-017 | Reviewed existing workload, actual Windows Codex/native client and independently authorized minimal Web listener; deliver first real vertical-slice evidence | Web login->select->persist binding->real MCP identity/fixed hello->durable receipt->restart observation. Pin/identity failure prevents execution. Actual stdout/exit and versions recorded. If unavailable, fixture pass stays fixture-only and phase 4 remains partial | VERTICAL_SLICE evidence, ADR dispositions, PHASE_STATUS with mandatory developer gate |
| TASK-006 | 5 / P1 / TASK-005 plus developer acceptance | REQ-004, REQ-005, REQ-006, REQ-007, REQ-018; AC-004, AC-005, AC-006, AC-007, AC-018 | Exact action plans and worker; deliver new-box creation, identity-bound readiness, confirmed start/stop/delete, receipt discovery/reconciliation | Delayed create/crash/repeated confirmation call-count fixtures, stale target rejects; reviewed dedicated real create and query before retry, no duplicate. Destructive actions only within explicit scope; preserve created workload unless cleanup approved | operation/state docs, risk/evidence matrix |
| TASK-007 | 5 / P1 / TASK-003, TASK-005 plus developer acceptance | REQ-001, REQ-002, REQ-003, REQ-017; AC-001, AC-002, AC-003, AC-017 | Selected native/explicit WSL transport + Codex/ZCode config contracts; deliver four profile enroll/diagnose paths | PowerShell AST/config fixtures, explicit distro/path/non-ASCII/null-device/EOF/deadline/process cleanup cases; actual four-profile MCP + registration/UI observations. No WSL install/distro fallback; no claims from shims alone | client profile matrix, ADR-003, risk/evidence matrix |
| TASK-008 | 5 / P1 / TASK-007 | REQ-001, REQ-015, REQ-016, REQ-017; AC-001, AC-015, AC-016, AC-017 | Version-pinned ZCode public config contract; deliver nonsecret GLM wizard + native sign-in/settings handoff or verified direct setup | Synthetic-secret redaction, invalid provider/model, preservation and idempotent selection tests; real client completion separate from MCP, tiny inference only if explicitly authorized. Unsupported automation remains pending, never Claude fallback | ADR-004, model wizard contract, profile evidence |
| TASK-009 | 5 / P1 / TASK-006, TASK-004 | REQ-005, REQ-011, REQ-012, REQ-018; AC-005, AC-011, AC-012, AC-018 | Authenticated capability + fresh Guard + actual upstream atomic policy profile; deliver Tenant metadata, shared new-box membership, directional grants/revokes | Fixtures for unsupported/stale/remote placement, account/shared identity, malformed lists, revision conflicts and readback; real separately approved capability/policy test including TCP if claimed. No provider upgrade workaround; if unsupported, record blocker for release acceptance | ADR-004, provider compatibility, Tenant/TCP evidence and OPEN_QUESTIONS |
| TASK-010 | 5 / P1 / TASK-006, TASK-004 | REQ-011, REQ-013, REQ-018; AC-011, AC-013, AC-018 | Exact target + reviewed ports/audience; deliver TCP/UDP passthrough list/add/remove | Collision/IP-reuse/ambiguous response fixtures; separately authorized real route + intended-client check, exact temporary listener/cleanup plan. No generic HTTP proxy editor/TLS or LAN-filter claim | port/security contract, exposure evidence, risk matrix |
| TASK-011 | 5 / P1 / TASK-004 | REQ-011, REQ-014, REQ-016; AC-011, AC-014, AC-016 | Graph schema/repository + basic canvas; deliver cards/comments/undirected links/copy/Markdown export | Browser persistence/reload/concurrent edit, invalid reference, escaped Markdown and clipboard failure tests; delete graph leaves environments intact; zero infrastructure calls from graph relations | graph schema/API, ADR-004, browser evidence |
| TASK-012 | 5 / P1 / TASK-006, TASK-007, TASK-008, TASK-009, TASK-010, TASK-011 | REQ-002, REQ-008, REQ-009, REQ-016, REQ-018; AC-002, AC-008, AC-009, AC-016, AC-018 | Locked app/UI artifacts, public trust manifest, independent credentials/services; deliver staged install/update/repair package | Non-root Linux installer sandbox + parse-only shell/PowerShell checks, followed by separately reviewed isolated live install/rerun/failure recovery. Settings/data preserved, tampered payload refused, Web failure leaves MCP working. No installed provider/Incus upgrade | install/recovery/provenance docs, ADR-001/003, real vs mocked evidence |
| TASK-013 | 5 / P1 / TASK-008, TASK-009, TASK-010, TASK-011, TASK-012 | REQ-001, REQ-003, REQ-009, REQ-011, REQ-012, REQ-013, REQ-014, REQ-015, REQ-017; AC-001, AC-003, AC-009, AC-011, AC-012, AC-013, AC-014, AC-015, AC-017 | Complete selected feature contracts; deliver release candidate evidence for all four profiles and basic Web features | Real client E2E + failure paths; declared provider compatibility; UI recognition and model completion distinguished. Every selected AC gets observed pass or explicit unresolved blocker. No release acceptance if mandatory evidence is missing | FEATURE_MATRIX, VERTICAL_SLICE, OPEN_RISKS, HANDOFF |
| TASK-014 | 6 / P1 / TASK-013 plus phase 6 authorization | REQ-017; AC-017 | Selected new behavior and curated legacy baseline; deliver difference/acceptance report | Compare only approved behavior using reproducible cases; classify deliberate changes/defects/unverified; don't demand legacy implementation parity or run dangerous legacy workflows. Phase 7 optimization remains separately authorized | COMPARISON_REPORT, FEATURE_MATRIX, OPEN_RISKS |

## Requirement coverage

| Requirement | Components | Tasks | Acceptance |
| --- | --- | --- | --- |
| REQ-001 | Client adapters | TASK-003, TASK-007, TASK-008 | AC-001 |
| REQ-002 | Trust/client/install | TASK-003, TASK-005, TASK-007, TASK-012 | AC-002 |
| REQ-003 | Independent diagnosis | TASK-003, TASK-005, TASK-007, TASK-013 | AC-003 |
| REQ-004 | Project/target/provider | TASK-001, TASK-005, TASK-006 | AC-004 |
| REQ-005 | Operation/repository/provider | TASK-002, TASK-006, TASK-009 | AC-005 |
| REQ-006 | Operation/clock/runner | TASK-002, TASK-005, TASK-006 | AC-006 |
| REQ-007 | Repository/worker/receipt query | TASK-002, TASK-005, TASK-006 | AC-007 |
| REQ-008 | Installer/state recovery | TASK-012 | AC-008 |
| REQ-009 | Diagnostics/install | TASK-012, TASK-013 | AC-009 |
| REQ-010 | Project/SQLite/Web/CLI | TASK-001, TASK-002, TASK-004, TASK-005 | AC-010 |
| REQ-011 | Web/shared services | TASK-004, TASK-005, TASK-009, TASK-010, TASK-011, TASK-013 | AC-011 |
| REQ-012 | Tenant/provider/revisions | TASK-009, TASK-013 | AC-012 |
| REQ-013 | Port/provider/plans | TASK-010, TASK-013 | AC-013 |
| REQ-014 | Graph/repository/canvas | TASK-011, TASK-013 | AC-014 |
| REQ-015 | Client model wizard | TASK-008, TASK-013 | AC-015 |
| REQ-016 | Packaging/client/redaction | TASK-003, TASK-008, TASK-011, TASK-012 | AC-016 |
| REQ-017 | Verification/profile evidence | TASK-001, TASK-005, TASK-007, TASK-008, TASK-013, TASK-014 | AC-017 |
| REQ-018 | Target/plan/credential boundary | TASK-003, TASK-004, TASK-006, TASK-009, TASK-010, TASK-012 | AC-018 |

First five tasks build and prove a deliberately small path; later tasks cover all basic features the developer selected. No task can bypass the phase-4 developer gate or substitute fake results for required real evidence. Missing ZCode/native automation or provider capability must be reported as specific risk/blocker, not silently removed from scope.


## Current phase 4 task progress

| Task | Status | Actual evidence and remaining condition |
| --- | --- | --- |
| TASK-001 | Implemented minimum contracts/package | Typed target/error boundary, explicit fixture/live modes, source/runtime locks; real integration still belongs to TASK-005 |
| TASK-002 | Partial | Binding/CAS/intent/receipt/worker-lock/deadline and no-replay tests pass; arbitrary-command recovery, DB-full fault and provider observation/reconciliation interface not complete |
| TASK-003 | Partial | POSIX MCP synthetic peer tests and native connection probe with AST/three no-network rejects; full enrollment/payload trust distribution/real native registration not complete |
| TASK-004 | Implemented minimum fixture UI/service slice | Actual Chromium login/binding/plan/receipt/restart; production HTTPS/independent service identity and real-provider integration not claimed |
| TASK-005 | Real Web/MCP minimum path passed; client/developer gate incomplete | SRC-049/050: independent strict SSH MCP, actual inner create/identity/hello, HTTP and Chromium receipt/restart/no-replay passed with 0.1.1. Actual Windows-native enrollment/registration/UI and explicit developer architecture/slice acceptance remain unverified. See LIVE_MCP_CHAIN |
| TASK-006 through TASK-014 | Not started | Phase-4 developer gate has not passed; do not start full feature work |

The [feature matrix](../verification/FEATURE_MATRIX.md) and [vertical-slice report](../verification/VERTICAL_SLICE.md) map evidence to REQ/AC and distinguish fixtures from actual target execution. No status above is a first-release acceptance claim.


## Readiness-only assessment after SRC-041

No phase 5 task was started. Technically, TASK-011 (basic Prompt Graph) can be selected as a first credential-independent iteration because its sole technical dependency TASK-004 has an implemented, fixture-tested minimum interface. TASK-006 and other live-MCP/provider-dependent work remain unsuitable until their prerequisites are satisfied. The phase 4 developer architecture/slice acceptance is not recorded as fully passed; a future phase 5 execution request must explicitly accept the demonstrated minimum architecture/slice with the deferred live test as an exception. A deferred test is not completed acceptance or blanket permission to ignore dependencies.

## Current result — real chain verified (SRC-049/050)

Version 0.1.1 fixes a real queue-claim timing race. Thirty Linux tests pass, including a new regression that failed before the fix. An explicitly approved nesting-enabled backend now supports the real application -> independent managed MCP -> inner workload path. HTTP and Chromium binding/confirm/hello/receipt/restart/no-replay plus live trust/identity rejection passed. See [LIVE_MCP_CHAIN](../verification/LIVE_MCP_CHAIN.md) for exact identities, operation IDs, versions, commands and limits.

The previous missing-launcher/nested-startup blocker is resolved for this new lab only, not the old box or every LXC. TASK-005 selected Web path has live evidence; real Windows-native enrollment/Agent GUI acceptance remains incomplete. Developer required this real chain before phase 5 (SRC-048); no phase 5 feature was started and architecture/slice acceptance must still be recorded before TASK-011 or other new features. Original published demo remains fixture; isolated test servers are stopped, resources retained, no new public port or host-security change.
