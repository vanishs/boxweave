# Developer documentation

For installation and normal agent use, begin with the [product README](../README.md), [installation](getting-started/MANUAL_INSTALL.md) and [usage](getting-started/USE_BOXWEAVE.md). This index is the developer entry point, not an accumulating chronological status log.

## Current implementation and release status

Baseline verified on 2026-11-24: v1 commit [3165f8e](https://github.com/vanishs/boxweave/commit/3165f8eedfad514ee4e5c733e7807d19ddbec2ee), including merged PRs #178 and #177. Latest published prerelease is [RC13](https://github.com/vanishs/boxweave/releases/tag/v1.0.0-rc.13), source 6c995687; stable v1 is unpublished. Do not infer deployment or release from merged source.

- Current source: 23 MCP Tools, SQLite schema 6, one independently revocable management capability per Actor; live installed discovery is authoritative.
- [PR #178 scoped Peer delivery](development/V1_PR178_N5_DELIVERY_2026-11-23.md): N1–N5 accepted and merged at ba29c742. Fresh-only opt-in, no legacy migration/upgrade/rollback/uninstall; normal MCP Peer with scoped Root validation. Capacity limits and physical acceptance boundaries remain.
- [PR #177 Doctor/Recover delivery](development/V1_PR177_D5_DELIVERY_2026-11-24.md): D0–D5 accepted and merged at 3165f8e. Exact candidate e754d8f passed [source](https://github.com/vanishs/boxweave/actions/runs/45358890236), [quality](https://github.com/vanishs/boxweave/actions/runs/45358890320), [dual architecture](https://github.com/vanishs/boxweave/actions/runs/45358890183) and [installed legacy/scoped lifecycle](https://github.com/vanishs/boxweave/actions/runs/45358890250). This is not a release or physical Host signoff.
- Stable release acceptance and residual physical, external network, nested isolation and full-scale limits are separate from these merged feature gates. Verify the actual current release and phase evidence before making claims.

## Architecture and contracts

| Topic | Reference |
| --- | --- |
| Product and exclusions | [Product](product/PRODUCT.md), [scope](product/SCOPE.md), [permanent boundary](decisions/PRODUCT_BOUNDARY.md), [requirements](product/REQUIREMENTS.md) |
| Architecture and ownership | [Architecture](design/ARCHITECTURE.md), [Incus integration](design/INCUS_INTEGRATION.md), [security](design/SECURITY.md), [nesting](design/NESTING.md) |
| Public API | [MCP tool catalog](design/MCP_TOOLS.md), [detailed contracts](design/MCP_TOOL_CONTRACTS.md), [MCP binding](design/MCP_BINDING.md) |
| Native networking | [Network design](design/NETWORK.md), [owner requirements](development/V1_PEER_MCP_AUTOSTART_NONDISRUPTIVE_OWNER_DECISION_2026-11-23.md), [N5 architecture and limits](development/V1_PR178_N5_DELIVERY_2026-11-23.md) |
| Durable operations | [Operations](design/OPERATIONS.md), [continuity](design/AGENT_OPERATION_CONTINUITY.md), [agent development loop](product/DEVELOPMENT_LOOP.md) |
| Diagnostics and recovery | [Operator guide](process/DOCTOR_RECOVER.md), [manual recovery](process/UNKNOWN_MANUAL_RECOVERY.md), [D4 contract](development/V1_PR177_D4_REVIEWED_RECOVERY_2026-11-23.md), [D5 evidence](development/V1_PR177_D5_DELIVERY_2026-11-24.md) |
| Terms and defaults | [Glossary](GLOSSARY.md), [defaults](decisions/DEFAULTS.md), [operational policies](decisions/P0_OPERATIONAL_POLICIES.md) |

## Installation maintenance

[Install/uninstall incident closure](development/V1_INSTALL_UNINSTALL_INCIDENT_CLOSURE_2026-11-24.md) records #177 follow-up work and verification. [Scoped retirement design](development/V1_SCOPED_RETIREMENT_DESIGN_2026-11-24.md) is the original review contract. [PR #182](https://github.com/vanishs/boxweave/pull/182) tracks its full-only implementation and acceptance (not a release).

## Contribute and verify

- [AGENTS.md](../AGENTS.md), [Contributing](../CONTRIBUTING.md), [Agent workflow](development/AGENT_WORKFLOW.md) and [phase execution protocol](development/PHASE_EXECUTION_PROTOCOL.md).
- [Quality gates](process/QUALITY_GATES.md), [implementation plan](process/IMPLEMENTATION_PLAN.md), [phase records](development/phases/README.md), [execution log](development/EXECUTION_LOG.md).
- [Versioning](process/VERSIONING.md), [pilot/release contract](development/V1_PILOT_DEPLOY_AND_RELEASE_CONTRACT.md), [acceptance](verification/ACCEPTANCE.md), [open risks](verification/OPEN_RISKS.md).
- [Python MCP testkit](development/V1_PYTHON_LOOPBACK_MCP_TESTKIT_2026-11-19.md), [legacy retirement](development/V1_UNINSTALL_SH_OPERATOR_CONTRACT_2026-11-19.md), [guided upgrade](development/V1_GUIDED_UPGRADE_2026-11-21.md).

## Historical evidence

Dated design, research, development and verification files retain their original checkpoint wording. A candidate/Draft or PASS/PENDING statement in a dated file is scoped to its original commit, not the current branch. Preserve those records; use the latest owner decisions, exact source and installed discovery for current behavior.

- [Complete pre-refresh historical index](https://github.com/vanishs/boxweave/blob/3165f8eedfad514ee4e5c733e7807d19ddbec2ee/docs/INDEX.md) retains the old detailed links and chronological evidence.
- Browse [development](development/), [verification](verification/), [research](research/) and [decisions](decisions/).
- [D0–D5 plan](development/V1_PR177_D0_D5_DEVELOPMENT_PLAN_2026-11-23.md) and [N5 delivery](development/V1_PR178_N5_DELIVERY_2026-11-23.md) are the entry points for the two latest completed feature tracks.

## Documentation maintenance

Keep README focused on users, the getting-started guides version-aware, and this index concise. On release, update the pinned examples and source-versus-release status. On API changes, update the catalog and contracts against actual MCP discovery. Record historical evidence separately; do not turn README into a PR timeline or rewrite old test outcomes.
