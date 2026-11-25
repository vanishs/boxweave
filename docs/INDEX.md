# Developer documentation

Start with [README](../README.md), [installation](getting-started/MANUAL_INSTALL.md), [usage](getting-started/USE_BOXWEAVE.md) and [current status](STATUS.md). Dated reports are evidence for their original source, not present-day instructions.

| Topic | Current reference |
| --- | --- |
| Product | [Product](product/PRODUCT.md), [scope](product/SCOPE.md), [requirements](product/REQUIREMENTS.md), [boundary](decisions/PRODUCT_BOUNDARY.md) |
| Decisions | [Defaults](decisions/DEFAULTS.md), [register](decisions/DECISIONS.md), [operational policies](decisions/P0_OPERATIONAL_POLICIES.md), [glossary](GLOSSARY.md) |
| Runtime | [Architecture](design/ARCHITECTURE.md), [Incus integration](design/INCUS_INTEGRATION.md), [technology](design/TECH_STACK.md), [guest management](design/GUEST_MANAGEMENT.md), [security](design/SECURITY.md), [nesting](design/NESTING.md) |
| API | [MCP tools](design/MCP_TOOLS.md), [contracts](design/MCP_TOOL_CONTRACTS.md), [binding](design/MCP_BINDING.md), [network](design/NETWORK.md) |
| Durable work | [Operations](design/OPERATIONS.md), [tracking](design/TRACKING_CONTEXT.md), [continuity](design/AGENT_OPERATION_CONTINUITY.md), [development loop](product/DEVELOPMENT_LOOP.md) |
| Host maintenance | [Install/uninstall/reinstall](getting-started/MANUAL_INSTALL.md), [Doctor/Recover](process/DOCTOR_RECOVER.md), [UNKNOWN](process/UNKNOWN_MANUAL_RECOVERY.md), [dependencies](process/DEPENDENCY_LIFECYCLE.md) |
| Development | [Agent rules](../AGENTS.md), [contributing](../CONTRIBUTING.md), [workflow](development/AGENT_WORKFLOW.md), [phase protocol](development/PHASE_EXECUTION_PROTOCOL.md), [active plan](process/IMPLEMENTATION_PLAN.md), [execution log](development/EXECUTION_LOG.md) |
| Development Harness | [Start here](../dev/README.md), [design](development/DEVELOPMENT_HARNESS.md), [asset index](../dev/INDEX.md), [bwdev design](development/BWDEV_DESIGN.md), [v0.5 acceptance](development/BWDEV_V05_ACCEPTANCE.md), [universal activation delivery plan](development/BDH_DUAL_LOOP_DELIVERY_PLAN.md), [U1-U4 acceptance](development/BDH_DUAL_LOOP_ACCEPTANCE.md) |
| Verification | [Quality](process/QUALITY_GATES.md), [acceptance](verification/ACCEPTANCE.md), [continuity](verification/CONTINUITY_ACCEPTANCE.md), [risks](verification/OPEN_RISKS.md), [versioning](process/VERSIONING.md) |
| History | [Chronological archive](history/README.md), [upstream references](research/REFERENCES.md) |

Keep current contracts separate from dated experiments, superseded plans and original test results. Preserve historical observations and exact source/CI identity. Update STATUS only from verified source or publication. Link current documents from this index and maintain relative paths. A docs-only commit is not a runtime release.
