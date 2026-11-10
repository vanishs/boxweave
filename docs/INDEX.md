# Documentation index

The **v1 base** started as a design-only baseline but now contains a real installed **S1 development candidate** accepted with independent native evidence; [PR #115](https://github.com/vanishs/boxweave/pull/115) merged into protected `v1`. [S1 formal exit](verification/S1_FINAL_EXIT_AUDIT_2026-11-09.md) and [GitHub quality protection](process/QUALITY_GATES.md) are accepted, **not a released or automatically deployed v1**. S2 has not begun; the newly approved S2.5 BRIDGEHEAD plan remains unexecuted.

**Start development without loading the whole repository:** [five feature phases + S2.5 BRIDGEHEAD checkpoint entry](process/IMPLEMENTATION_PLAN.md) → [phase execution and context protocol](development/PHASE_EXECUTION_PROTOCOL.md) → [current checkpoint / append-only operations + decision log](development/EXECUTION_LOG.md#current-checkpoint) → **only the current** [S1–S5 card or S2.5 BRIDGEHEAD gate](development/phases/README.md). Phase cards reference the **smallest relevant sections** of the detailed documents below. Do not treat upstream SDK probes as installed v1 acceptance.

| Development question | One entrypoint |
|---|---|
| What source formatting, staged commit hooks, PR checks and required merge rules apply? | [Contributing](../CONTRIBUTING.md) · [quality / GitHub gate](process/QUALITY_GATES.md) |
| Which old Incus migration, rollback and crash windows were really accepted? | [S1 Issue #116 native acceptance](verification/S1_ISSUE116_NATIVE_MIGRATION_2026-11-09.md) |
| What big stages and which MCP tool ownership? | [Five feature phases + S2.5 acceptance/deployment checkpoint](process/IMPLEMENTATION_PLAN.md) |
| Must S2 migrate S1 SQLite test databases, or may it redesign the first supported schema? | [Pre-deployment schema freedom and first-retained-install boundary](process/VERSIONING.md#initial-schema-freeze--no-user-data-migration-before-first-deployment) · [decision D-061](decisions/DECISIONS.md) · [TC-003 fresh DB test](verification/CONTINUITY_ACCEPTANCE.md) |
| When can the accepted S1+S2 candidate become an early real-host development base without weakening S4/S5? | [S2.5 BRIDGEHEAD phase card](development/phases/025_BRIDGEHEAD.md) · [BH-001–BH-017 exact native acceptance](verification/BRIDGEHEAD_ACCEPTANCE.md) |
| How to execute continuously, handle repeated failures and avoid context loss? | [Execution protocol](development/PHASE_EXECUTION_PROTOCOL.md) |
| What did earlier Agents do, decide, leave UNKNOWN or require the owner to decide? | [Execution log and live checkpoint](development/EXECUTION_LOG.md) |
| Which focused inputs and exit gates belong to the current phase? | [Phase cards index](development/phases/README.md) |

**Reference index — load one topic below when the current phase needs it:**

| Question | Authoritative document |
|---|---|
| What is BoxWeave? | [Product](product/PRODUCT.md) |
| Which permanent boundaries override superseded v1 plans? | [Permanent product boundary](decisions/PRODUCT_BOUNDARY.md) |
| What supports multiple Agents, attributed native operations and safe restart? | [Agent Operation Continuity contract](design/AGENT_OPERATION_CONTINUITY.md) · [TC-001–TC-020 acceptance](verification/CONTINUITY_ACCEPTANCE.md) |
| How is an Agent's broader goal/Git state checkpointed without new BoxWeave Tools? | [Lightweight client-owned Task Checkpoint](development/TASK_CHECKPOINT_CONTRACT.md) |
| What is the primary P0 Agent workflow? | [Sustainable development loop](product/DEVELOPMENT_LOOP.md) |
| How to simulate a real LAN Hello World, failed delete, nested candidate and release? | [Fault-driven release scenarios](verification/FAILURE_DRIVEN_RELEASE_SCENARIOS.md) |
| What is in/out? | [Scope](product/SCOPE.md) |
| What must work? | [Requirements](product/REQUIREMENTS.md) |
| What is the architecture? | [Architecture](design/ARCHITECTURE.md) |
| Which runtime and SDKs are required? | [Go technology stack](design/TECH_STACK.md) |
| Which MCP tools are planned? | [MCP catalog](design/MCP_TOOLS.md) |
| What are the detailed file, Exec, wait and cancel contracts? | [MCP tool contracts](design/MCP_TOOL_CONTRACTS.md) |
| Must a Box run a client? | [Agentless guest management](design/GUEST_MANAGEMENT.md) |
| How does local MCP become reachable? | [MCP loopback transport](design/MCP_BINDING.md) |
| What may an MCP client touch? | [Security and ownership](design/SECURITY.md) |
| How does nesting work? | [Nesting](design/NESTING.md) |
| What happens to interrupted mutations? | [Operations](design/OPERATIONS.md) |
| How does an operator safely unlock an UNKNOWN claim? | [Approved local UNKNOWN recovery](process/UNKNOWN_MANUAL_RECOVERY.md) |
| What was learned from Containarium current main and 2,577 commit subjects? | [Containarium study — source and historical counterexamples](research/CONTAINARIUM_MAIN_HISTORY_2026-11-07.md) |
| What did two smaller Incus MCP implementations teach? | [Incus MCP source review, Python and Go](research/INCUS_MCP_IMPLEMENTATIONS_2026-11-07.md) |
| Why did BoxWeave `main` become troublesome around MCP→REST changes, and what must v1 avoid? | [Pinned main source/814-commit history and owner retrospective — primarily counterexamples](research/MAIN_HISTORY_LESSONS_2026-11-07.md) |
| What can the legacy host I/O/Cowsql incident, wrong MCP root and Agent stalls teach v1? | [Observed incident and bounded architecture discussion](research/LEGACY_MAIN_INCUS_MCP_INCIDENT_2026-11-10.md) |
| Can future MCP Tasks extension be used without changing S2 Operation ownership or adding a task engine? | [Protocol-neutral future Tasks adapter seam — official spec/SDK readiness](research/MCP_TASKS_EXTENSION_SEAM_2026-11-10.md) |
| How are requests correlated across services? | [Tracking ID and Go context](design/TRACKING_CONTEXT.md) |
| How are dependency versions reconciled? | [Dependency lifecycle](process/DEPENDENCY_LIFECYCLE.md) |
| How do networking and exposure work? | [Networking](design/NETWORK.md) |
| How is Incus integrated? | [Incus integration](design/INCUS_INTEGRATION.md) |
| What small tagged Go SDK capabilities could improve Agent safety without expanding Tool count? | [SDK v1.8.0 continuity and Tool annotation/cache research](research/MCP_GO_SDK_CONTINUITY_REVIEW_2026-11-10.md) |
| What is reusable from Temporal, LangGraph and CrewAI? | [Framework durability/Agent identity/visibility comparison](research/AGENT_CONTINUITY_FRAMEWORKS_2026-11-10.md) |
| Which official SDK behaviors constrain the v1 architecture? | [Official Incus + MCP SDK fit review](research/OFFICIAL_SDK_FIT_2026-11-07.md) |
| How does each of the 20 public MCP tools map to native SDK methods? | [Native SDK mapping: 20/20 tools](research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md) |
| Which further SDK call semantics still affect v1's design? | [Second-pass SDK gap closure: Tool auth, Bridge ACL, files, Exec, wait and Project bootstrap](research/SDK_GAP_CLOSURE_2026-11-07.md) |
| What did the disposable Box SDK validation actually establish? | [2026-11-07 live Box probe and remaining blockers](verification/SDK_NATIVE_BOX_PROBE_2026-11-07.md) |
| What is the smallest practical batched native SDK verification, and which tests have actually passed? | [Original minimal test plan and preliminary smoke](verification/SDK_MINIMAL_BATCH_2026-11-08.md) |
| What were the actual native B/C SDK tests, database recovery, packet findings and cleanup states? | [2026-11-08 executed Incus 7.6 nested B/C batch, native evidence and ACL permission blocker](verification/SDK_NATIVE_BATCH_RESULTS_2026-11-08.md) |
| Can Incus native restricted TLS/Scriptlet safely edit one ACL, and what still blocks dynamic creation? | [Executed Incus Go SDK TLS, Scriptlet, ETag and ACL creation-granularity probe](verification/INCUS_ACL_AUTHZ_TLS_SCRIPTLET_2026-11-08.md) |
| Which v1 networking design was selected and what actually passed across three Boxes? | [Single shared ACL + native Scriptlet: three-Box IPv4/IPv6, peer revoke, proxy, legacy TLS compatibility and cleanup](verification/SINGLE_SHARED_ACL_SCRIPTLET_ACCEPTANCE_2026-11-08.md) |
| Which official SDK P0 behaviors actually passed, and which Incus-native limits must v1 respect? | [Last pre-development P0 plan](verification/P0_SDK_LAST_GATE_PLAN_2026-11-08.md) and [executed Incus + MCP SDK test results/cleanup](verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md) |
| What are the approved runtime defaults? | [Owner-approved defaults](decisions/DEFAULTS.md) |
| What are the finalized P0 operational rules? | [P0 operational policies](decisions/P0_OPERATIONAL_POLICIES.md) and [permanent scope](decisions/PRODUCT_BOUNDARY.md) |
| What remains undecided? | [Open decisions and risks](verification/OPEN_RISKS.md) |
| How is behavior accepted? | [Acceptance matrix](verification/ACCEPTANCE.md) |
| How should agents develop? | [Agent workflow](development/AGENT_WORKFLOW.md) |
| How is v1 versioned? | [Versioning](process/VERSIONING.md) |
| What are implementation slices? | [Implementation plan](process/IMPLEMENTATION_PLAN.md) |
| What external lessons were reviewed? | [Research](research/REFERENCES.md) |

Repository-wide hard rules live in [AGENTS.md](../AGENTS.md). Historical v0.x material is **not** part of this branch.
