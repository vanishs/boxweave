# BoxWeave v1 — single development entrypoint, five autonomous phases

**Status (2026-11-08): DESIGN ONLY. No v1 Go controller, installer or shipped release yet.** SDK P0 capability research was completed and recorded in [the last native/real-HTTP SDK tests](../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md). It is **not** installed v1 acceptance. This file is the **only top-level development plan**. Execute each phase end to end; [the short execution protocol](../development/PHASE_EXECUTION_PROTOCOL.md) defines the repeated loop and continuity rules.

## Next command and how to start

**Start at S1.** An executing Agent can be told: **“Execute BoxWeave v1 development phase S1 end-to-end according to docs/process/IMPLEMENTATION_PLAN.md. Keep the execution log and proceed autonomously through implementation, tests, failed-case repair and acceptance; record blocked/decision-required items while continuing independent work.”**

At every new context/session: read **[AGENTS.md](../../AGENTS.md) → this short plan → [execution protocol](../development/PHASE_EXECUTION_PROTOCOL.md) → the current [phase card](../development/phases/README.md) → [current checkpoint](../development/EXECUTION_LOG.md#current-checkpoint)**. Fetch one relevant design/SDK/acceptance section **only when its subtask begins**; do not preload the complete repository research, the full acceptance matrix or all phase cards.

Verify actual branch HEAD, uncommitted edits, active tasks, installed version and original native resource identities before continuing; **a prior Agent's last message is not a native readback**. Always update [the action/decision log](../development/EXECUTION_LOG.md) before context compaction, a risky retry, handoff or closing a phase.

## Five major development phases

| Stage | Full in-one-pass deliverable | New MCP tools | Document to execute | Current actual state |
|---|---|---:|---|---|
| **S1 Foundation** | Official SDK Go service, typed authenticated loopback MCP, safe host-local install/reuse/status, Incus restriction, SQLite ledger, Tracking/async admission/UNKNOWN, real security tests | **1: host_status** | [S1 card](../development/phases/01_FOUNDATION.md) | **NOT_STARTED** |
| **S2 Box and operations** | 7 native Box lifecycle tools; async guest Exec; 4 durable operation tools; native identity/ready/restart/readback, all positive and negative cases | **12** | [S2 card](../development/phases/02_BOX_LIFECYCLE_OPERATIONS.md) | **NOT_STARTED** |
| **S3 Files and Agent loop** | 3 guest file tools; SFTP exclusive create, safe unsupported conditional cases; a real Agent guest Git/edit/failing test/fix/passing test cycle | **3** | [S3 card](../development/phases/03_FILES_AGENT_LOOP.md) | **NOT_STARTED** |
| **S4 Networking and nesting** | 3 native TCP/UDP forward tools and box_peer_set; one installer-precreated ACL + exact TLS/Scriptlet, real peer/isolation/LAN, nested install, dependent Box delete | **4** | [S4 card](../development/phases/04_NETWORK_NESTING.md) | **NOT_STARTED** |
| **S5 Independent full acceptance & release** | 20/20 installed MCP + all real local CLI leaves; adversarial Finder/Fixer/independent Verifier, real native/LAN/nesting, exact binary digest/provenance, gated formal release | **0** | [S5 card](../development/phases/05_RELEASE_ACCEPTANCE.md) | **NOT_STARTED** |

**Coverage invariant: 1 + 12 + 3 + 4 = exactly 20 public tools.** [Authoritative 20-tool catalog](../design/MCP_TOOLS.md) and [tool contracts](../design/MCP_TOOL_CONTRACTS.md) govern behavior; every tool is covered again in [ACCEPTANCE](../verification/ACCEPTANCE.md), not just the phase in which it first appears. No deleted v0.x functionality can sneak back in as a future phase.

**Dependency continuity:** S1 owns one identity/journal/worker protocol used by every later tool; S2 owns Box lifecycle/operation IDs and ships a **safe deletion integration seam**, but **S4 must reaccept box_delete** with live forward/peer dependencies and faults. S3 reuses S2 Exec for the full Agent loop rather than introducing patch tooling. S4 extends the same S1 installer for global Scriptlet/ACL bootstrap. S5 verifies the **same exact source/build** across a separate installed candidate, not disconnected stage-specific prototypes.

## The substage loop is the same inside EVERY stage

**1. Locate required source → 2. Investigate only gaps → 3. Implement → 4. Build/negative-test → 5. Independent/native acceptance → 6. Fix code/docs and rerun → 7. Commit, log, hand off.** Read the [execution protocol](../development/PHASE_EXECUTION_PROTOCOL.md) for evidence, document editing, owner permissions and context checkpoint requirements. Do not terminate an otherwise executable phase at “wrote a design” or “opened a PR”.

Each phase card has its *small* entry reading set, task-time links and exact acceptance IDs. Full historical studies ([source/history research](../research/REFERENCES.md)), huge [ACCEPTANCE](../verification/ACCEPTANCE.md) and [OPEN_RISKS](../verification/OPEN_RISKS.md) must be loaded **by relevant section or ID**, not copied into every Agent prompt.

### Failure, external decisions and honest partial work

If a subcase fails twice or a fix cannot be established after bounded reproduction/attempts, retain the original failure/native identities, record tested corrections and linked evidence in [EXECUTION_LOG](../development/EXECUTION_LOG.md), update [OPEN_RISKS](../verification/OPEN_RISKS.md) or a narrowly scoped issue, mark the named test FAIL/BLOCKED/UNKNOWN, and **continue all safe independent subtasks**. An indispensable owner decision is written as **DECISION_REQUIRED** with options and affected safety gate; other work continues. Do not silently choose additional privilege/exposure, replay UNKNOWN native calls, or mark a dependent unsafe action PASS. A stage can be PARTIAL while unrelated later stage work advances; **S5 formal release remains blocked until every mandatory real acceptance gate is met**.

**Docs are first-class live artifacts:** edit them whenever code/native facts disagree, a newer decision supersedes an old claim, test coverage grows, or an Agent's context footprint is excessive. Update cross-links, mark historical passages explicitly superseded, and note the decision, old/new contract and acceptance impact in the immutable log. Preserve original evidence and Git history, never rewrite prior FAIL/UNKNOWN into PASS. Phase entry cards own task ordering; design/verification/decision docs remain the single source for their facts.

## Scope locks and SDK facts no Agent may silently override

- Go + **official MCP Go SDK and Incus Go SDK**, native Incus restricted OS identity, typed operations, real Box UUID and Project ownership, SQLite persistence. No Containarium, Caddy, generic host shell/REST management, distributed coordinator, guest resident Agent, generic ACL DSL or patch engine.
- Incus one shared **installer-created ACL** for owned NICs, isolated dedicated restricted TLS+Scriptlet GET/PUT, daemon-global authorization.client.tls-restricted=scriptlet **explicitly accepted by owner with README risk warning**, tested compatibility/rollback before a shared host switch. Runtime never receives Incus-admin or arbitrary ACL creation. [Network contract](../design/NETWORK.md), [real three-NIC proof](../verification/SINGLE_SHARED_ACL_SCRIPTLET_ACCEPTANCE_2026-11-08.md).
- Native SFTP exclusive create **proved**, but native file conditional replace/delete atomic CAS **not provided**. Keep one box_file_apply API with create/replace/delete modes; return honest **UNSUPPORTED** on unprovable safe replace/delete, do not hash-check then overwrite and pretend conditional. [SDK final results](../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md). A required positive contract unsatisfied at release remains a recorded gate, not a hidden “PASS”.
- Native async Exec recorded stdout/stderr **recoverable after SDK reconnect**; detached Operation.Cancel() cannot simply cancel the guest; live native Exec control WebSocket SIGTERM can work while still connected. Persist original operation ID/log paths, verify final nonzero exit; only claim cancellation if supported and proven. HTTP client disconnect never cancels admitted durable work.
- Nested Incus Cowsql leader loss happened in prior disposable Box labs. **Owner explicitly ruled out BoxWeave v1 Incus/Cowsql DB patches or separate repair services.** Treat as potential test environment issue with ordinary UNKNOWN safety, not expanding product responsibilities.
- Nested L1/L2 parity is supported **when native environment permits it**, not parent-controller orchestration. Installer may reuse compatible existing Incus with separate operator approval for disruptive maintenance; never auto reboot or change unrelated Incus resources.
- Initial root VERSION=1.0.0 and docs-only branch are **not** release evidence. [Version/release](VERSIONING.md) requires exact real binary/provenance, independent verifier and authenticated, full native black-box acceptance.

## Where to find facts, not preload them

| Need during a phase | Load on demand |
|---|---|
| Permanent scope and approved host defaults | [Boundary](../decisions/PRODUCT_BOUNDARY.md), [P0 policies](../decisions/P0_OPERATIONAL_POLICIES.md), [Defaults](../decisions/DEFAULTS.md) |
| Current action, prior failures, owner choices, safely resumable IDs | [Execution log checkpoint and entries](../development/EXECUTION_LOG.md) |
| Phase inputs/outputs / minimal targeted subtask docs | [S1–S5 cards](../development/phases/README.md) |
| Native method exact typed contract and proved SDK caveats | [20/20 SDK mapping](../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md), [P0 final probe](../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md) |
| Durable operation and missing-reply safety | [Operations](../design/OPERATIONS.md), [Tracking](../design/TRACKING_CONTEXT.md), [UNKNOWN local review](UNKNOWN_MANUAL_RECOVERY.md) |
| Networking and shared host authorization | [Network](../design/NETWORK.md), [Security](../design/SECURITY.md), [real single-ACL test](../verification/SINGLE_SHARED_ACL_SCRIPTLET_ACCEPTANCE_2026-11-08.md) |
| Final native/CLI/LAN/adversarial test oracle | [Acceptance IDs](../verification/ACCEPTANCE.md), [failure-driven scenario](../verification/FAILURE_DRIVEN_RELEASE_SCENARIOS.md), [Agent workflow](../development/AGENT_WORKFLOW.md) |

Historical original **0–7 slice order** maps to S1 (former 0–2), S2 (former 3), S3 (former 4), S4 (former 5–6), S5 (former 7). Previous SDK and design findings remain accessible through linked source reports and Git history; the old source-review stages are **already completed research**, not a command to rerun them at every build.

**If only one sentence is provided to the next Agent:** “Execute the current S# of [this plan](IMPLEMENTATION_PLAN.md) end-to-end, read the current phase docs only when needed, preserve original UNKNOWN, keep [the log](../development/EXECUTION_LOG.md) current, fix/retest/accept, and continue safe unfinished work even if one case is blocked.”
