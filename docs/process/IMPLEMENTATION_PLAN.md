# Implementation plan — authorized design and iteration status

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
| TASK-002 | Complete approved fixed-command storage/worker scope0.3.0 | TASK-002a history plus actual SQLite FULL/claim/result failure, process crash/concurrency/deadline/exit tests and explicit read-only target observations. Unknown never auto-resolves/replays; lifecycle/create writes remain TASK-006 |
| TASK-003 | Complete selected minimal native Codex scope0.4.0 | Local private-ACL key preparation, authorized public-key-only restricted launcher enrollment with preservation/idempotence, trusted registration and independent actual connection/registration diagnosis; prior native TUI evidence retained. Other profiles remain TASK-007, packaging/release TASK-012/013 |
| TASK-004 | Implemented minimum fixture UI/service slice | Actual Chromium login/binding/plan/receipt/restart; production HTTPS/independent service identity and real-provider integration not claimed |
| TASK-005 | Complete selected minimal native/current0.4.0 chain | SRC-065 native public-key enrollment/registration/diagnosis; SRC-057 separate human TUI/read; SRC-068 current HTTP/Chromium real hello/receipt/history/restart/no replay and trust/identity failures. Gate SRC-051 accepted; other profiles/release remain TASK-007/013 |
| TASK-006 | Implemented scoped lifecycle0.5.0 | SRC-069:82 tests/fixture browser pass; first real five-action cycle passed. Final hardening repeat stops on real TRUST_FAILED; reconciler remains an environment limitation. Exact approved cleanup verified, original hello preserved. See TASK_006_LIFECYCLE for distinct evidence and recovery limits |
| TASK-011 | Implemented basic scope, 0.2.0 | SRC-051 developer gate accepted; SRC-052 graph/service/browser evidence; see PHASE_05_TASK_011 |
| TASK-006..010, TASK-012..014 | Not started | Keep per-task prerequisites and real-client/release evidence obligations |

The [feature matrix](../verification/FEATURE_MATRIX.md) and [vertical-slice report](../verification/VERTICAL_SLICE.md) map evidence to REQ/AC and distinguish fixtures from actual target execution. No status above is a first-release acceptance claim.


## Readiness-only assessment after SRC-041

No phase 5 task was started. Technically, TASK-011 (basic Prompt Graph) can be selected as a first credential-independent iteration because its sole technical dependency TASK-004 has an implemented, fixture-tested minimum interface. TASK-006 and other live-MCP/provider-dependent work remain unsuitable until their prerequisites are satisfied. The phase 4 developer architecture/slice acceptance is not recorded as fully passed; a future phase 5 execution request must explicitly accept the demonstrated minimum architecture/slice with the deferred live test as an exception. A deferred test is not completed acceptance or blanket permission to ignore dependencies.

## Current result — real chain verified (SRC-049/050)

Version 0.1.1 fixes a real queue-claim timing race. Thirty Linux tests pass, including a new regression that failed before the fix. An explicitly approved nesting-enabled backend now supports the real application -> independent managed MCP -> inner workload path. HTTP and Chromium binding/confirm/hello/receipt/restart/no-replay plus live trust/identity rejection passed. See [LIVE_MCP_CHAIN](../verification/LIVE_MCP_CHAIN.md) for exact identities, operation IDs, versions, commands and limits.

The previous missing-launcher/nested-startup blocker is resolved for this new lab only, not the old box or every LXC. TASK-005 selected Web path has live evidence; real Windows-native enrollment/Agent GUI acceptance remains incomplete. Developer required this real chain before phase 5 (SRC-048); no phase 5 feature was started and architecture/slice acceptance must still be recorded before TASK-011 or other new features. Original published demo remains fixture; isolated test servers are stopped, resources retained, no new public port or host-security change.

## Phase 5 opened — TASK-011 (SRC-051)

Developer explicitly accepted the demonstrated minimum architecture and slice after real-chain verification, and authorized TASK-011. This records the phase-4 developer gate; remaining Windows/WSL Agent registration and release evidence are not waived. Execute one slice only: REQ-011/014/016, AC-011/014/016, basic Prompt Graph. Assigned development LXC remains boxweave-tenant-dev. No deployment, public exposure, model or infrastructure mutation is part of the graph slice.

Before implementation: verify real SQLite additive migration and revision conflicts; authenticated/Origin/CSRF graph API; bounded cards/comments/undirected links/context; reject orphan/self/duplicate links; escaped and recognized-secret-redacted Markdown; record copy only after clipboard success (client report, not server proof); graph deletion preserves environments; browser save/reload/concurrent edit/clipboard failure/layout; assert graph services make zero provider calls. Explicit saves preserve local unsaved edits on conflict. Use architecture-selected XYFlow with an exact lock. Schema-1 data must survive migration; older binaries must refuse schema 2.

## Phase 5 iteration 1 complete — TASK-011 (SRC-051/052)

Developer acceptance of the minimum architecture/slice is explicitly recorded in SRC-051, after verified real Web/MCP evidence. TASK-011 basic Prompt Graph is implemented in 0.2.0: bounded cards, editable comments/context, undirected links, revision-checked persistence, conflict-preserved drafts, Markdown/clipboard handling and graph-only deletion. REQ-011/014/016 and AC-011/014/016 graph subset are evidenced; other selected-feature/client obligations remain incomplete.

Actual non-root project-LXC checks: 41 tests OK (11.173s), TypeScript/Vite build, final Chromium graph workflow including mobile fit and clipboard failure, original project hello/restart fixture regression; all exit0. Graph API forbids provider access in tests; browser has zero queued/executed operations. Thirteen changed source/lock files match remote SHA256. See PHASE_05_TASK_011.md for commands and precise evidence. No new box, infrastructure mutation, deployment, persistent listener, public route or push. Stop this iteration.

Next suggestion: finish TASK-003/005 minimum actual Windows-native enrollment/recognition evidence, then dependent TASK-006. Developer gate acceptance does not fabricate those missing client checks or waive per-task prerequisites. Phase 5 overall is not complete.

## Iteration 2 scope — TASK-003a (SRC-053)

Developer requested continuation and box-based or mapped cooperative testing. Current built-in MCP handles return Transport closed, but a fresh Windows native SSH session using the existing registered restricted launcher passed initialize/initialized, tools-list109 and actual read-only list_containers against MCP0.120.1. No key content/configuration was read/exported/changed. Ephemeral ignored native-client management helper now performs MCP tools/call connect; assigned LXC identity verified exit0. This is the same authorized MCP, not a host shell or new server registration.

Split the remaining TASK-003: TASK-003a delivers local operator-trusted payload verification and Windows-native Codex registration using an already authorized SSH identity; test in an isolated Agent home with actual installed Codex CLI and native MCP probe. Preserve unrelated configuration, explicit conflict/replacement intent, strict trust and idempotence; tampering must prevent execution. No new key enrollment, global Codex/SSH changes, model invocation, public mapping or GUI-loading claim. Full bootstrap identity provisioning and GUI/remaining profiles stay incomplete. This one bounded slice covers REQ-001/002/003/016/017 and matching AC subsets; it does not complete TASK-003/005 wholesale.

## Iteration 2 complete — TASK-003a, version 0.2.1 (SRC-053/054)

Scoped existing-identity Windows-native Codex registration now has trusted pre-execution payload verification, strict live connection gating, targeted config preservation, explicit replacement, byte-idempotence and backups. Actual isolated Windows/CLI/native MCP and failure-path tests passed; 41 project Linux tests and the three prior native preflight checks passed. No global/personal registration, new key, model request, mapping, box or deployment changed.

TASK-003a is complete in its stated existing-identity scope; parent TASK-003/005 remain partial. Actual GUI/TUI loading, bootstrap identity provisioning and other client/runtime profiles are unverified. Named-mutex/hash/backup recovery does not replace quiescing non-cooperating editors. See TASK_003A_NATIVE_REGISTRATION for commands, limitations and cooperative next steps. Stop this iteration; suggest TASK-003b, do not waive dependencies for TASK-006.

TASK-003 split: TASK-003a = existing-identity native Codex trusted registration/verifier (implemented0.2.1); TASK-003b = remaining reviewed bootstrap identity provisioning and actual selected-client loading/recognition evidence (not started). Preserve parent TASK-003 requirements/dependencies and TASK-005 full acceptance; child completion is not a release claim.

## Current iteration 3 — TASK-003b1, version 0.2.2 (SRC-055..058)

TASK-003b is split into TASK-003b1 (explicit Windows-native executable selection plus recorded cooperative TUI/read-only acceptance, complete) and TASK-003b2 (product bootstrap identity provisioning, incomplete). Parent TASK-003 and full TASK-005 remain partial; do not waive TASK-006 dependencies. Developer architecture gate SRC-051 remains accepted.

The native bootstrap/registration accepts an operator-selected absolute -CodexExecutable and retains exact-version/payload/trust checks. Real Windows run-06 passes with Codex absent from PATH, plus missing/relative runtime rejection and existing preservation/integrity regressions; personal config digest unchanged. Project-LXC 41 tests pass (11.586s); native AST/3 rejection checks pass. SRC-057 records the user's actual TUI connected(2 tools) and list_containers result v2-live-hello-container against the dedicated backend. This is separate from desktop GUI, instruction/model readiness, WSL/ZCode and release evidence. See [TASK-003b1 evidence](../verification/TASK_003B1_NATIVE_CLIENT.md).

The dedicated test key/18447 route were separately approved in SRC-055/056, and retained; no new route/resource/policy/model operation is part of this implementation iteration. Existing public Web demo remains fixture. No automatic key provisioning or whole-phase completion claim. Next task is TASK-003b2; stop this one slice after its code/test/docs commit, no push.

## Phase5 entry disposition (SRC-059/060)

All four original startup checks are satisfied; see [entry audit](../verification/PHASE_05_READINESS.md). TASK-003b2 and full client/release obligations stay incomplete but are phase5 implementation work, not new phase5 entry conditions. The prior parent-task partial statuses and per-task dependencies remain; this audit does not authorize TASK-006 automatically or mark automatic enrollment implemented. Select the next bounded task after the readiness-only handoff.

## Phase5 iteration4 — TASK-002a (SRC-061/062)

Split TASK-002's remaining scope: TASK-002a receipt discovery (complete0.2.3, REQ-007/011 and AC-007/011 read subset), TASK-002b state-write failure/crash-window coverage (planned), other provider reconciliation remains pending. TASK-002a needs existing storage/HTTP foundations; it adds no provider write and does not waive TASK-006 dependencies. [Actual evidence](../verification/TASK_002A_RECEIPT_HISTORY.md):46 tests, TypeScript/Vite, Chromium history plus original hello/Graph regressions passed. Stop this slice; suggest the bounded TASK-002b before expanding mutations.
