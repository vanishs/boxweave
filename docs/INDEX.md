# Knowledge index

Baseline date: 2026-10-28, America/Los_Angeles. Legacy ref: `refs/heads/main` at `4318da6ea4e4e45c55cf43b64acbdd2ede65fc43`.

Read in order:

1. [Agent rules](../AGENTS.md), [master plan](process/MASTER_PLAN.md), [phase status](process/PHASE_STATUS.md).
2. [Overview](knowledge/PROJECT_OVERVIEW.md), [features](knowledge/FEATURE_INVENTORY.md), [behavior](knowledge/BEHAVIOR_BASELINE.md).
3. [Source register](knowledge/SOURCE_REGISTER.md), [Agent audit](knowledge/AGENT_NOTES_AUDIT.md), [open questions](knowledge/OPEN_QUESTIONS.md).
4. [Product proposal](product/PRODUCT.md), [requirements](product/REQUIREMENTS.md), [acceptance](product/ACCEPTANCE.md), [scope and five decisions](product/SCOPE.md).
5. [Legacy synchronization](knowledge/LEGACY_SYNC.md), [current handoff](process/HANDOFF.md), [archived phase 1 handoff](process/PHASE_01_HANDOFF.md), [phase 2 instructions](process/PHASE_02_INSTRUCTIONS.md).

Authority: PHASE_STATUS owns phase/gate state; SOURCE_REGISTER owns source identities; OPEN_QUESTIONS owns unresolved risks. REQUIREMENTS + ACCEPTANCE own proposed product behavior; SCOPE owns feature dispositions and pending DEC choices; PRODUCT summarizes user outcomes. Historical evidence is not a new product requirement. IDs F-xxx/B-xxx identify legacy features/behaviors; SRC-xxx sources; RISK-xxx uncertainties; REQ/AC proposed requirements/criteria; DEC pending product questions. ADR/TASK remain for later phases.

Phase 1 covers core repository contracts, representative implementation/test paths, meaningful Git changes, and selected ignored research notes. It is not a line-by-line audit of all code or every copied upstream artifact. Phase 2 now provides a reviewable proposal: 18 REQ, 18 AC, dispositions for all 15 legacy features and five pending decisions. All product requirements remain proposed; no new product code, architecture or application test result exists. Phase 2 document work is complete subject to developer review; its exit gate remains unsatisfied. See the current handoff for actual documentation/Git checks.
