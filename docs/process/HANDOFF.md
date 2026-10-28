# Phase 3 handoff

Date: 2026-10-28 America/Los_Angeles. Result: **design and plan complete; implementation not started**.

Process authority: SRC-025 (phase 3 request/prompt); product approval: SRC-026 (developer five choices). Phase 2 gate is now satisfied. Earlier draft recommendations deferring the requested features are superseded, not treated as approval. 'WSL SSL' is interpreted as WSL SSH from context.

Legacy: refs/heads/main@4318da6ea4e4e45c55cf43b64acbdd2ede65fc43; last knowledge sync same SHA, no new legacy commit. Worktree: C:/Users/vanis/.codex/visualizations/2026/10/28/01a18158-8e09-7373-b64c-800d96ecb72e/boxweave-rewrite-v2, branch rewrite/v2. Phase 3 parent: 130e88429b3aed3e4a8dc8f37d5cd34ef5cae236. Exact new commit is reported after committing; locate via `git log --format=oneline --grep='docs: design approved remote development architecture and plan' rewrite/v2`. Prior phase 2 draft handoff is archived separately.

## Approved scope and design

Single trusted operator, personal Agent remote work plus project binding; Windows-hosted Codex and ZCode(GLM) over native SSH or explicit WSL SSH. Basic Web, Tenant network writes, TCP/UDP port publication, Prompt Graph and client model wizard are included. Recursive deployment, shared-team/public service, automatic graph execution and other platforms/Agents are deferred. Basic boundaries are documented in SCOPE; all 18 REQ/AC selected for first-release design, none implemented/tested.

Architecture: Python modular application, Flask/Waitress API, local SQLite, one operation worker, CLI/Web shared services, React/TypeScript/XYFlow canvas; upstream restricted SSH/MCP retained, fixed Containarium REST adapter, independent credential planes and HTTPS frontend. This is a new design, not old implementation reuse. Alternatives Go/Node and distributed queue/database were compared; no single-operator need justified their additional scope. SQLite rollback journal/FULL synchronous initially; provider I/O outside transactions. Dependency versions will be locked/tested during implementation.

ADR-001: application/adapters/stack. ADR-002: persisted intent, unknown outcomes and no replay after crash. ADR-003: upstream transport, separate trust and operator-delivered payload digest manifest. ADR-004: narrow basic feature adapters and native ZCode model credential ownership. Official documentation references SRC-027/028 inform design only; real compatibility still unknown.

## First vertical slice and first five tasks

One actual Windows Codex/native client, one reviewed preexisting dedicated workload, one project: Web login -> select target -> real SQLite binding -> trusted MCP identity + fixed hello -> durable receipt -> restart/reload observation without replay. No create/policy/port/model call is needed for this slice. Later selected features remain mandatory release obligations.

1. TASK-001: contracts/minimal package and isolated fake boundary.
2. TASK-002: real SQLite revisions/intent/worker/receipt and crash/deadline cases.
3. TASK-003: minimal native client trust/enrollment/diagnosis and real provider identity parser.
4. TASK-004: minimal authenticated Web/CLI binding/receipt interface.
5. TASK-005: real end-to-end slice and failure evidence, followed by explicit developer gate.

Full plan: 14 tasks with dependencies, outputs, REQ/AC coverage, tests and same-task document updates. Six experiments are planned only. TASK-006 onward implements the other selected profiles/basic features after the phase-4 acceptance gate; TASK-014 requires phase 6 authorization.

## Highest risks and future validation

- RISK-006: current native Tenant capability/atomic policy contract unproven; gated writes must not be simulated with multiple non-atomic calls or enabled by an unapproved provider upgrade.
- RISK-014: actual ZCode GLM automation contract/model access unproven. Wizard uses client-owned settings/sign-in and explicitly pending completion, never a Claude fallback or MCP-only ready claim.
- RISK-015: WSL distro/argv/cleanup across Windows boundary; actual four-profile matrix is required, not shim tests.
- RISK-016: local transactions cannot make remote writes exactly-once; intent-before-send crash windows remain unknown, never auto-resubmitted.
- RISK-003/005/009/012 remain live concerns for bootstrap trust, real clients, update repair and remote cancellation. Security/confinement/autostart are not certified.

No live experiment ran: Platform MCP inventory only succeeded with four Running resources; no connect/provider mutation/route exposure/GLM request. Legacy ref/status/client hash were preserved at entry (SRC-029). Current source discovery is not evidence of workload readiness.

## Deliverables and gate

New architecture, four ADRs, IMPLEMENTATION_PLAN, TEST_STRATEGY, verbatim PHASE_03_INSTRUCTIONS and archived PHASE_02_HANDOFF. Updated all product authority documents, index, source register, risks, legacy sync, phase status and this handoff. No application code/scaffold, dependencies or VERSION bump.

Phase 4 entry: a new execution request and a concrete reviewed environment/experiment plan with exact identities, intended changes, deadlines, success and cleanup. Phase 4 must provide real slice evidence; developer acceptance of that minimum system is required before phase 5. This phase stops at design.

Documentation validation results are recorded below after checks. No product tests were executed; every task/experiment and AC runtime outcome remains not started/unverified.

## Documentation verification receipt

The read-only document validator completed with exit 0: 28 Markdown documents, 48 valid local links, 18 reciprocal REQ/AC pairs, all 18 requirements mapped to design/components/tasks, 14 tasks with ordered acyclic dependencies and REQ/AC links, and all 15 legacy feature dispositions. Referenced SRC/TASK/RISK/ADR/EXP IDs resolve. The phase 3 prompt matches the supplied file byte-for-byte; archived phase 2 handoff normalizes to the exact prior Git blob. Narrow private-key/JWT-pattern scan plus content review found no credential payload; this is not universal secret detection. `git diff --check` passed for tracked changes. Staged new files are checked separately before committing.

The phase 3 source review also inspected the legacy installer's upstream MCP token-file launch pattern, read-only. This is static evidence only; dedicated application-worker credentials/launch must be proven in TASK-003, without reading or reusing the existing SSH MCP token. No live provider capability, Agent configuration, model sign-in or workload execution was tested.

Final commit scope/clean-worktree and legacy state checks are performed around commit; exact commit SHA and any remaining validation failures are reported to the developer. Original main/WIP/archive are preserved. Phase 3 is complete as documentation, not runtime acceptance.
