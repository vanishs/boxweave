# Knowledge index

Baseline date: 2026-10-28, America/Los_Angeles. Legacy ref: `refs/heads/main` at `4318da6ea4e4e45c55cf43b64acbdd2ede65fc43`.

Read in order:

1. [Agent rules](../AGENTS.md), [master plan](process/MASTER_PLAN.md), [phase status](process/PHASE_STATUS.md).
2. [Overview](knowledge/PROJECT_OVERVIEW.md), [features](knowledge/FEATURE_INVENTORY.md), [behavior](knowledge/BEHAVIOR_BASELINE.md).
3. [Source register](knowledge/SOURCE_REGISTER.md), [Agent audit](knowledge/AGENT_NOTES_AUDIT.md), [open questions](knowledge/OPEN_QUESTIONS.md).
4. [Legacy synchronization](knowledge/LEGACY_SYNC.md), [handoff](process/HANDOFF.md), [phase 1 instructions](process/PHASE_01_INSTRUCTIONS.md).

Authority: PHASE_STATUS owns phase/gate state; SOURCE_REGISTER owns source identities; OPEN_QUESTIONS owns unresolved risks. Historical evidence is not a new product requirement. IDs F-xxx/B-xxx identify legacy features/behaviors; SRC-xxx sources; RISK-xxx uncertainties. REQ/AC/ADR/TASK IDs are reserved for later phases, not invented approvals.

Phase 1 covers core repository contracts, representative implementation/test paths, meaningful Git changes, and selected ignored research notes. It is not a line-by-line audit of all code or every copied upstream artifact. No old application tests were run. No product, design or verification architecture documents are created yet: phases 2 onward own those decisions. See the handoff for actual documentation/Git checks.
