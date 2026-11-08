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
| What lessons were extracted from all 814 main commits? | [Main history audit, research only](research/MAIN_HISTORY_LESSONS_2026-11-07.md) |
| How are requests correlated across services? | [Tracking ID and Go context](design/TRACKING_CONTEXT.md) |
| How are dependency versions reconciled? | [Dependency lifecycle](process/DEPENDENCY_LIFECYCLE.md) |
| How do networking and exposure work? | [Networking](design/NETWORK.md) |
| How is Incus integrated? | [Incus integration](design/INCUS_INTEGRATION.md) |
| What are the approved runtime defaults? | [Owner-approved defaults](decisions/DEFAULTS.md) |
| What are the finalized P0 operational rules? | [P0 operational policies](decisions/P0_OPERATIONAL_POLICIES.md) and [permanent scope](decisions/PRODUCT_BOUNDARY.md) |
| What remains undecided? | [Open decisions and risks](verification/OPEN_RISKS.md) |
| How is behavior accepted? | [Acceptance matrix](verification/ACCEPTANCE.md) |
| How should agents develop? | [Agent workflow](development/AGENT_WORKFLOW.md) |
| How is v1 versioned? | [Versioning](process/VERSIONING.md) |
| What are implementation slices? | [Implementation plan](process/IMPLEMENTATION_PLAN.md) |
| What external lessons were reviewed? | [Research](research/REFERENCES.md) |

Repository-wide hard rules live in [AGENTS.md](../AGENTS.md). Historical v0.x material is **not** part of this branch.
