# Documentation index

This branch is a **v1 design baseline**, not implemented product functionality.

| Question | Authoritative document |
|---|---|
| What is BoxWeave? | [Product](product/PRODUCT.md) |
| Which permanent boundaries override superseded v1 plans? | [Permanent product boundary](decisions/PRODUCT_BOUNDARY.md) |
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
| How are requests correlated across services? | [Tracking ID and Go context](design/TRACKING_CONTEXT.md) |
| How are dependency versions reconciled? | [Dependency lifecycle](process/DEPENDENCY_LIFECYCLE.md) |
| How do networking and exposure work? | [Networking](design/NETWORK.md) |
| How is Incus integrated? | [Incus integration](design/INCUS_INTEGRATION.md) |
| Which official SDK behaviors constrain the v1 architecture? | [Official Incus + MCP SDK fit review](research/OFFICIAL_SDK_FIT_2026-11-07.md) |
| How does each of the 20 public MCP tools map to native SDK methods? | [Native SDK mapping: 20/20 tools](research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md) |
| Which further SDK call semantics still affect v1's design? | [Second-pass SDK gap closure: Tool auth, Bridge ACL, files, Exec, wait and Project bootstrap](research/SDK_GAP_CLOSURE_2026-11-07.md) |
| What did the disposable Box SDK validation actually establish? | [2026-11-07 live Box probe and remaining blockers](verification/SDK_NATIVE_BOX_PROBE_2026-11-07.md) |
| What is the smallest practical batched native SDK verification, and which tests have actually passed? | [Original minimal test plan and preliminary smoke](verification/SDK_MINIMAL_BATCH_2026-11-08.md) |
| What were the actual native B/C SDK tests, database recovery, packet findings and cleanup states? | [2026-11-08 executed Incus 7.6 nested B/C batch, native evidence and ACL permission blocker](verification/SDK_NATIVE_BATCH_RESULTS_2026-11-08.md) |
| Can Incus native restricted TLS/Scriptlet safely edit one ACL, and what still blocks dynamic creation? | [Executed Incus Go SDK TLS, Scriptlet, ETag and ACL creation-granularity probe](verification/INCUS_ACL_AUTHZ_TLS_SCRIPTLET_2026-11-08.md) |
| What are the approved runtime defaults? | [Owner-approved defaults](decisions/DEFAULTS.md) |
| What are the finalized P0 operational rules? | [P0 operational policies](decisions/P0_OPERATIONAL_POLICIES.md) and [permanent scope](decisions/PRODUCT_BOUNDARY.md) |
| What remains undecided? | [Open decisions and risks](verification/OPEN_RISKS.md) |
| How is behavior accepted? | [Acceptance matrix](verification/ACCEPTANCE.md) |
| How should agents develop? | [Agent workflow](development/AGENT_WORKFLOW.md) |
| How is v1 versioned? | [Versioning](process/VERSIONING.md) |
| What are implementation slices? | [Implementation plan](process/IMPLEMENTATION_PLAN.md) |
| What external lessons were reviewed? | [Research](research/REFERENCES.md) |

Repository-wide hard rules live in [AGENTS.md](../AGENTS.md). Historical v0.x material is **not** part of this branch.
