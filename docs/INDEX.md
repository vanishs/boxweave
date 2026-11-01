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
| REQ/AC → code/tests/evidence | [FEATURE_MATRIX](verification/FEATURE_MATRIX.md), [code-first acceptance](verification/CODE_FIRST_ACCEPTANCE_2026-11-01.md) |
| Agent-driven rapid development/test workflow | [AGENT_TEST_WORKFLOW](process/AGENT_TEST_WORKFLOW.md), [Agent deployment v1.0](process/AGENT_TEST_DEPLOYMENT.md) and [local source bundle helper](../scripts/agent_source_bundle.py) (developer practice) |
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

## Diagnostics implementation (D0–D2 in 0.22.0; D3/D4 source in 0.25.0)

- [Current source/behavior audit](verification/DIAGNOSTICS_SOURCE_AUDIT.md)
- [Evidence-first CLI and existing-MCP implementation proposal](design/DIAGNOSTICS_PROPOSAL.md)
- [D0 implementation specification — diagnostic correctness](design/DIAGNOSTICS_D0_IMPLEMENTATION.md)
- [D1 implementation specification — trace and evidence](design/DIAGNOSTICS_D1_IMPLEMENTATION.md)
- [D2 implementation specification — reports and nested diagnosis](design/DIAGNOSTICS_D2_IMPLEMENTATION.md)
- [D3+D4 owner-approved master](design/DIAGNOSTICS_D3_D4_MASTER.md)
- [D3+D4 implementation and tests](design/DIAGNOSTICS_D3_D4_IMPLEMENTATION.md)
- [D3/D4 operator guide](DIAGNOSTICS.md)

## Box environment management research (proposal; NOT IMPLEMENTED)

- [Architecture — Box checkpoints, cloning and environment fingerprints](design/BOX_ENVIRONMENT_PROPOSAL.md)
- [E0 — Provider capability / security boundary](design/BOX_ENV_E0_CAPABILITIES.md)
- [E1 — Environment fingerprint capture and difference](design/BOX_ENV_E1_FINGERPRINTS.md)
- [E2 — Snapshot create, list and delete](design/BOX_ENV_E2_SNAPSHOTS.md)
- [E3 — Snapshot restore](design/BOX_ENV_E3_RESTORE.md)
- [E4 — Independent Box cloning](design/BOX_ENV_E4_CLONE.md)

These environment features are **design-only**; D0–D2 shipped in 0.22.0, while D3/D4 are source-implemented in 0.25.0 with live acceptance separate.


## MCP capability integration (source inventory and staged implementation)

- [Master design and updated main status](design/MCP_CAPABILITY_INTEGRATION_MASTER.md)
- [Existing feature gap audit](verification/MCP_CAPABILITY_SOURCE_AUDIT.md)
- [Platform tool matrix](design/MCP_PLATFORM_CAPABILITY_MATRIX.md)
- [agent-box tool/resource matrix](design/MCP_AGENT_BOX_CAPABILITY_MATRIX.md)
- [Trust boundaries](design/MCP_INTEGRATION_SECURITY.md)
- [M0–M5 delivery handoff](design/MCP_INTEGRATION_DELIVERY_PLAN.md) and [acceptance](verification/MCP_INTEGRATION_TEST_PLAN.md)

MCP integration is distinct from D0–D2 diagnostics and E0–E4 environment proposals. PR #6 is research/design and must not be reported as runtime feature completion.

## MCP integration implementation checkpoint

- [Actual source completion, CI evidence, open decisions and next developer handoff](process/MCP_REUSE_IMPLEMENTATION_HANDOFF.md) (historical checkpoint; see later 0.24.0 live M0–M5 evidence and current code-first register)

## M0–M5 MCP source implementation (live Box acceptance deferred)

- [M0–M5 source state, operator configuration, Web API and remaining live acceptance](process/MCP_M0_M5_SOURCE_PROGRESS.md). Direct-gateway tool policy is optional host-wide enforcement, **not per-user RBAC**; agent-box requires a separately reviewed fixed launcher and live Box testing.

## Remote MCP HTTP(S) and OAuth (implemented; public deployment acceptance is separate)

- [Master design, eight owner decisions and phase map](design/REMOTE_MCP_MASTER.md)
- [ADR-009: keep SSH, add remote Streamable HTTP with no vendor tunnel dependency](design/ADR/0009-remote-mcp-transports-and-auth.md)
- Phased implementation: [R0 shared dispatcher](design/remote-mcp/R0_PROTOCOL_AND_DISPATCH.md), [R1 HTTP](design/remote-mcp/R1_STREAMABLE_HTTP.md), [R2 OAuth/API Token/credential URL](design/remote-mcp/R2_AUTH_AND_CREDENTIALS.md), [R3 HTTPS/proxies](design/remote-mcp/R3_HTTPS_AND_PROXY.md), [R4 Web onboarding](design/remote-mcp/R4_WEB_CLIENT_EXPERIENCE.md)
- [Negative and positive acceptance matrix](verification/REMOTE_MCP_TEST_PLAN.md), [Agent implementation handoff](process/REMOTE_MCP_HANDOFF.md)

**Code implemented:** v0.24.0 introduced remote HTTP MCP/OAuth and the capability remains in v0.25.0. Historical loopback HTTP/OAuth and local TLS-proxy tests are recorded; real public TLS, all external clients and current installed v0.25.0 require distinct acceptance. SSH/stdio remains supported, Web cookies are not OAuth MCP credentials, and the host-wide allowlist is not per-Agent RBAC.

- [Remote HTTP MCP source setup, CLI and safety limits](REMOTE_MCP.md) (v0.24.0 source only; real public/TLS acceptance pending)
