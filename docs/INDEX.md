# Knowledge and design index

Legacy baseline: refs/heads/main at 4318da6ea4e4e45c55cf43b64acbdd2ede65fc43. Current design date: 2026-10-28 America/Los_Angeles.

Read in order:

1. [Agent rules](../AGENTS.md), [master plan](process/MASTER_PLAN.md), [phase status](process/PHASE_STATUS.md).
2. [Overview](knowledge/PROJECT_OVERVIEW.md), [features](knowledge/FEATURE_INVENTORY.md), [behavior](knowledge/BEHAVIOR_BASELINE.md), [Agent audit](knowledge/AGENT_NOTES_AUDIT.md).
3. [Sources](knowledge/SOURCE_REGISTER.md), [risks](knowledge/OPEN_QUESTIONS.md), [legacy sync](knowledge/LEGACY_SYNC.md).
4. [Approved product](product/PRODUCT.md), [requirements](product/REQUIREMENTS.md), [acceptance](product/ACCEPTANCE.md), [scope/decisions](product/SCOPE.md).
5. [Architecture](design/ARCHITECTURE.md), [ADR-001 application](design/ADR/0001-modular-application.md), [ADR-002 intent](design/ADR/0002-intent-and-reconciliation.md), [ADR-003 trust](design/ADR/0003-transport-and-trust.md), [ADR-004 features/models](design/ADR/0004-basic-features-and-models.md).
6. [Implementation tasks](process/IMPLEMENTATION_PLAN.md), [test strategy/experiments](verification/TEST_STRATEGY.md), [current handoff](process/HANDOFF.md).
7. Historical: [phase 1 handoff](process/PHASE_01_HANDOFF.md), [phase 2 draft handoff](process/PHASE_02_HANDOFF.md), [phase 1 instructions](process/PHASE_01_INSTRUCTIONS.md), [phase 2 instructions](process/PHASE_02_INSTRUCTIONS.md), [phase 3 instructions](process/PHASE_03_INSTRUCTIONS.md).

Authority: PHASE_STATUS owns gates; SOURCE_REGISTER evidence identity; OPEN_QUESTIONS risk state; REQUIREMENTS + ACCEPTANCE behavior; SCOPE approved feature boundary/DEC choices; ARCHITECTURE + ADR design; IMPLEMENTATION_PLAN task dependency/coverage. Historical drafts cannot override SRC-026 developer approval. SRC/REQ/AC/ADR/TASK/RISK/DEC/EXP IDs are stable references, not evidence that work ran.

Phase 1 was representative static/historical research, not complete old-code/archive audit. Phase 2 choices are now developer-confirmed with changes from its original recommendation. Phase 3 contains design only: 18 requirements, 18 criteria, 15 legacy dispositions, four ADRs, 14 tasks and six unexecuted experiments. No application code/tests exist yet in this document-only root lineage. Phase 4 execution remains separately requested and its real minimum system needs developer acceptance before phase 5.
