# Knowledge and design index

Legacy baseline: refs/heads/main at 4318da6ea4e4e45c55cf43b64acbdd2ede65fc43. Current implementation/evidence date: 2026-10-28 America/Los_Angeles.

Read in order:

1. [Agent rules](../AGENTS.md), [master plan](process/MASTER_PLAN.md), [phase status](process/PHASE_STATUS.md).
2. [Overview](knowledge/PROJECT_OVERVIEW.md), [features](knowledge/FEATURE_INVENTORY.md), [behavior](knowledge/BEHAVIOR_BASELINE.md), [Agent audit](knowledge/AGENT_NOTES_AUDIT.md).
3. [Sources](knowledge/SOURCE_REGISTER.md), [risks](knowledge/OPEN_QUESTIONS.md), [legacy sync](knowledge/LEGACY_SYNC.md).
4. [Approved product](product/PRODUCT.md), [requirements](product/REQUIREMENTS.md), [acceptance](product/ACCEPTANCE.md), [scope/decisions](product/SCOPE.md).
5. [Architecture](design/ARCHITECTURE.md), [ADR-001 application](design/ADR/0001-modular-application.md), [ADR-002 intent](design/ADR/0002-intent-and-reconciliation.md), [ADR-003 trust](design/ADR/0003-transport-and-trust.md), [ADR-004 features/models](design/ADR/0004-basic-features-and-models.md).
6. [Implementation tasks](process/IMPLEMENTATION_PLAN.md), [test strategy/experiments](verification/TEST_STRATEGY.md), [phase 4 evidence](verification/VERTICAL_SLICE.md), [dedicated test deployment](verification/TEST_DEPLOYMENT.md), [shared-Tenant lab result](verification/SHARED_TENANT_LAB.md), [feature matrix](verification/FEATURE_MATRIX.md), [current handoff](process/HANDOFF.md).
7. [Run the minimum system](../README.md), [ADR-005 phase 4 boundary](design/ADR/0005-phase-four-boundary.md), [approved execution scope](verification/PHASE_04_EXECUTION_PLAN.md).
8. Historical: [phase 1 handoff](process/PHASE_01_HANDOFF.md), [phase 2 draft handoff](process/PHASE_02_HANDOFF.md), [phase 1 instructions](process/PHASE_01_INSTRUCTIONS.md), [phase 2 instructions](process/PHASE_02_INSTRUCTIONS.md), [phase 3 instructions](process/PHASE_03_INSTRUCTIONS.md), [phase 3 handoff](process/PHASE_03_HANDOFF.md), [phase 4 instructions](process/PHASE_04_INSTRUCTIONS.md).

Authority: PHASE_STATUS owns gates; SOURCE_REGISTER evidence identity; OPEN_QUESTIONS risk state; REQUIREMENTS + ACCEPTANCE behavior; SCOPE approved feature boundary/DEC choices; ARCHITECTURE + ADR design; IMPLEMENTATION_PLAN task dependency/coverage. Historical drafts cannot override SRC-026 developer approval. SRC/REQ/AC/ADR/TASK/RISK/DEC/EXP IDs are stable references, not evidence that work ran.

Phase 1 was representative static/historical research, not complete old-code/archive audit. Phase 2 choices are now developer-confirmed with changes from its original recommendation. Phase 3 defined 18 requirements, 18 criteria, 15 legacy dispositions, four design ADRs, 14 tasks and six experiment plans. Phase 4 now has a partial runnable minimum implementation and ADR-005: real browser/application/storage/worker verification with explicit fixture infrastructure, synthetic MCP tests and Windows preflight; independent application live MCP and real native-client acceptance remain incomplete. User requested preservation, and phase 5 is not authorized. See VERTICAL_SLICE for actual results instead of treating historical plans as passes.

Current update SRC-049/050: [real Web-to-MCP chain](verification/LIVE_MCP_CHAIN.md) and [sanitized results/source hashes](verification/nested-live-results.json). New approved nesting-enabled backend and inner workload execute actual hello through v2 HTTP/Chromium/worker/MCP, with durable receipts/no replay. Version 0.1.1 fixes the observed queue-claim race; 30 tests pass. Earlier fixture-only/missing-launcher paragraphs are historical. Developer/client acceptance remains incomplete; phase5 feature work has not started and original public demo remains fixture.

Current phase5 iteration SRC-051/052: [TASK-011 delivery and tests](verification/PHASE_05_TASK_011.md), [ADR-006](design/ADR/0006-prompt-composition.md), [source hashes](verification/phase5-graph-source-hashes.json), [browser result](verification/phase5-graph-browser-result.json), [phase5 instructions](process/PHASE_05_INSTRUCTIONS.md). Developer gate accepted; 0.2.0 basic Prompt Graph complete, whole release/client matrix incomplete. No deployment or push.

Iteration2 SRC-053/054: [TASK-003a native registration](verification/TASK_003A_NATIVE_REGISTRATION.md), [results](verification/native-registration-results.json), [source hashes](verification/native-registration-source-hashes.json), [public descriptor example](../clients/windows/native-package.example.json). Existing-identity native CLI/connection scope verified, parent client/GUI/bootstrap obligations remain incomplete. No deployment or new mapping.

Current iteration3 SRC-055..058: [TASK-003b1 native executable and TUI evidence](verification/TASK_003B1_NATIVE_CLIENT.md). Version0.2.2 fixes actual ordinary-Windows-PATH registration failure with explicit executable selection. Dedicated native TUI/read-only acceptance is recorded; automated identity bootstrap and other profiles remain incomplete. Latest PHASE_STATUS supersedes earlier historical gate descriptions.

Current authority SRC-059/060: [phase5 entry audit — READY](verification/PHASE_05_READINESS.md). All four supplied startup checks are satisfied; SRC-051 acceptance remains recorded. Incomplete automatic identity provisioning and release/client features remain phase5 work, not additional entry conditions. This audit starts no new feature and does not waive task-level dependencies.

Iteration4 SRC-061/062: [TASK-002a receipt history](verification/TASK_002A_RECEIPT_HISTORY.md), [exact source hashes](verification/receipt-history-source-hashes.json). Version0.2.3 adds bounded actor-scoped local history/API/CLI/Web without execution;46 tests and actual Chromium history/original/graph regressions passed. Phase5 remains incomplete.

TASK-002 completed0.3.0: [recovery/observation evidence](verification/TASK_002_COMPLETION.md); user explicitly requested TASK-002 and TASK-003 consecutively. TASK-003 continues after this separate commit.

Requested TASK-002/003 run complete (SRC-063..065): [TASK-002 recovery/observation](verification/TASK_002_COMPLETION.md), [TASK-003 native identity/diagnosis](verification/TASK_003_COMPLETION.md), [native source hashes](verification/task003-source-hashes.json). Source0.4.0,64 Linux tests and actual isolated native Windows enrollment/diagnosis/regression passed. Full profile/release work remains pending; no push/deployment.

TASK-005 current minimal chain complete (SRC-068): [current-source actual evidence and harnesses](verification/TASK_005_CURRENT.md). TASK-006/007 continue under explicit SRC-066; dedicated lifecycle create/stop/start/delete approval SRC-067 does not include existing resources or new exposure.

- [TASK-006 lifecycle](verification/TASK_006_LIFECYCLE.md).
