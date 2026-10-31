# BoxWeave v2 knowledge index

Start here, not in a chronological transcript. [PHASE_STATUS](process/PHASE_STATUS.md) owns current gates; [VERSION](../VERSION) owns the product version. This index owns navigation only.

| Find | Authoritative route |
| --- | --- |
| New Agent / developer startup | [FINAL_HANDOFF](process/FINAL_HANDOFF.md) |
| Current work / authorization | [PHASE_STATUS](process/PHASE_STATUS.md), [current handoff](process/HANDOFF.md) |
| Product goal and selected boundaries | [PRODUCT](product/PRODUCT.md), [SCOPE](product/SCOPE.md) |
| Required outcomes and acceptance | [REQUIREMENTS](product/REQUIREMENTS.md), [ACCEPTANCE](product/ACCEPTANCE.md) |
| Actual runtime / module design | [ARCHITECTURE](design/ARCHITECTURE.md), [ADR index and status](design/ADR/INDEX.md) |
| Task dependencies | [IMPLEMENTATION_PLAN](process/IMPLEMENTATION_PLAN.md) |
| REQ/AC → code/tests/evidence | [FEATURE_MATRIX](verification/FEATURE_MATRIX.md) |
| Agent-driven rapid development/test workflow | [AGENT_TEST_WORKFLOW](process/AGENT_TEST_WORKFLOW.md) (developer practice) |
| How to run and verify | [INSTALL](INSTALL.md), [test strategy](verification/TEST_STRATEGY.md), [validation runner](../scripts/validate.py) |
| Current unresolved risks | [OPEN_RISKS](verification/OPEN_RISKS.md) |
| New questions/proposals | [OPEN_QUESTIONS](knowledge/OPEN_QUESTIONS.md) |
| Evidence/source identities | [SOURCE_REGISTER](knowledge/SOURCE_REGISTER.md) |
| Legacy observations / review baseline | [BEHAVIOR_BASELINE](knowledge/BEHAVIOR_BASELINE.md), [LEGACY_SYNC](knowledge/LEGACY_SYNC.md) |
| Daily change and knowledge workflow | [MAINTENANCE](process/MAINTENANCE.md), [machine-readable authority map](process/DOCUMENT_AUTHORITY.json) |
| Latest maintenance/cold-start proof | [PHASE_07_REVIEW](verification/PHASE_07_REVIEW.md) |

Earlier evidence remains available: [phase6 comparison](verification/COMPARISON_REPORT.md), [phase5 release](verification/TASK_012_013_RELEASE.md), [native network/ports](verification/TASK_009_010_ACCESS.md), [historical index](INDEX_HISTORY.md). These are dated/revisioned records, not current gate owners. Archived status/handoff/matrix snapshots are linked from their current owners.

Use local Markdown links and stable REQ-/AC-/TASK-/ADR-/SRC-/RISK-/DEC-/EXP- IDs. Reports record evidence; the code, schema and installed manifest remain the truth for actual runtime state. Documentation does not deploy an application.

## 0.10+ operator remediation routes

- [Management CLI](CLI.md)
- [Host bootstrap and upstream ownership](HOST_BOOTSTRAP.md)
- [0.5.1 source parity audit](verification/ARCHIVE_051_TO_MAIN_AUDIT.md)
- [New change validation queue](verification/REMEDIATION_TEST_QUEUE.md)

- [New host/CLI architecture decision](design/ADR/0007-host-install-boundary.md)

- [F-001–F-015 semantic parity](verification/FEATURE_PARITY_ARCHIVE_051.md)
- [Old/new Web workflow navigation and browser validation](verification/WEB_WORKFLOW_PARITY.md)

- [Operator troubleshooting](TROUBLESHOOTING.md)
- [Manual migration from archived 0.5.1](MIGRATING_FROM_051.md)
- [Upstream Provider/Incus contract](UPSTREAM_PROVIDER.md)

- [Development and isolated fixture](DEVELOPMENT.md)

- [Historical installation-guide snapshot](INSTALL_HISTORY.md)

- [0.14.0 main completion and current acceptance evidence](verification/MAIN_014_REVIEW.md)

- [Streamed installer and two-box MCP acceptance](verification/STREAMED_INSTALL_TEST.md)

- [Container nesting](NESTING.md) and [measured verification](verification/NESTING_VERIFICATION.md)

## Diagnostics research (proposal; not yet approved for implementation)

- [Current source/behavior audit](verification/DIAGNOSTICS_SOURCE_AUDIT.md)
- [Evidence-first CLI and existing-MCP implementation proposal](design/DIAGNOSTICS_PROPOSAL.md)
- [D0 implementation specification — diagnostic correctness](design/DIAGNOSTICS_D0_IMPLEMENTATION.md)
- [D1 implementation specification — trace and evidence](design/DIAGNOSTICS_D1_IMPLEMENTATION.md)
- [D2 implementation specification — reports and nested diagnosis](design/DIAGNOSTICS_D2_IMPLEMENTATION.md)

## Box environment management research (proposal; NOT IMPLEMENTED)

- [Architecture — Box checkpoints, cloning and environment fingerprints](design/BOX_ENVIRONMENT_PROPOSAL.md)
- [E0 — Provider capability / security boundary](design/BOX_ENV_E0_CAPABILITIES.md)
- [E1 — Environment fingerprint capture and difference](design/BOX_ENV_E1_FINGERPRINTS.md)
- [E2 — Snapshot create, list and delete](design/BOX_ENV_E2_SNAPSHOTS.md)
- [E3 — Snapshot restore](design/BOX_ENV_E3_RESTORE.md)
- [E4 — Independent Box cloning](design/BOX_ENV_E4_CLONE.md)

These are **design-only** and complement, but do not implement, the D0-D2 diagnostic proposals.

## MCP capability reuse research (proposal; NOT IMPLEMENTED)

- [Capability-led MCP integration master](design/MCP_CAPABILITY_INTEGRATION_MASTER.md)
- [Current functionality and source gap audit](verification/MCP_CAPABILITY_SOURCE_AUDIT.md)
- [All registered Platform tools vs BoxWeave app use](design/MCP_PLATFORM_CAPABILITY_MATRIX.md)
- [agent-box tools/resources and separate internal approvals](design/MCP_AGENT_BOX_CAPABILITY_MATRIX.md)
- [MCP security and compatibility boundaries](design/MCP_INTEGRATION_SECURITY.md)
- [Delivery dependencies and batches](design/MCP_INTEGRATION_DELIVERY_PLAN.md)
- [MCP acceptance and evidence plan](verification/MCP_INTEGRATION_TEST_PLAN.md)
- Programmer handoffs: [M0 capability audit](design/mcp-integration/M0_SOURCE_AND_RUNTIME_CAPABILITY.md), [M1 policy](design/mcp-integration/M1_ROUTING_AND_POLICY.md), [M2 Platform reuse](design/mcp-integration/M2_PLATFORM_REUSE.md), [M3 Box reads](design/mcp-integration/M3_AGENT_BOX_READONLY.md), [M4 Box execution](design/mcp-integration/M4_AGENT_BOX_EXECUTION.md), [M5 Web/workflows](design/mcp-integration/M5_WEB_AND_WORKFLOWS.md).

This proposal complements, but does **not** implement or supersede, existing diagnostics D0–D2 or environment E0–E4 work. Source inventories do not prove installed or authorized MCP capabilities.
