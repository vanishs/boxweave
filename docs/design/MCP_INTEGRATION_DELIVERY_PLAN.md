# MCP capability gap integration — phased delivery and handoff plan

**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.


## Scope and sequencing

This plan does **not** approve implementation. No generic framework, second root Shell, new standing MCP service, forced provider upgrade, new public port, hidden clone/snapshot support or blanket secret/tool grant is justified by the gap audit.

| Batch | Objective | Dependent on | Proposed result | Boundary |
| --- | --- | --- | --- | --- |
| **M0** | Source/live capability and caller-scope audit | Existing source only | Honest supported/unsupported/not_verified matrix, compatibility test baselines | Read-only; no runtime mutation |
| **M1** | Tool metadata, namespacing, security decision | M0, maintainer's gate-compatibility decision | Policy-controlled dispatcher, old Platform names preserved, no privilege claim without enforcement | Avoid silent breaking direct-Agent access |
| **M2** | Reuse selected Platform capabilities | M1; live tool schemas/scope | First: `debug_container` safe read. Then possible `toggle_auto_sleep` managed state path | No full trace and no automatic container state changes without approval |
| **M3** | Scoped agent-box read-only access | M1; exact target + trusted connection | File/list/log/process and resources on specifically authorized Box | No Shell, writes, install, or host bypass |
| **M4** | Scoped agent-box writes and execution | M3 accepted; dedicated execution authorization | Files, processes, bounded Shell, possible Compose actions | Strong execution isolation, review and unknown result |
| **M5** | Optional Web and workflows | Selected earlier slices | Selected status views, approved Agent/test flows, receipts, optional backup/security UI | No generic 109-tool screen or automatic E0–E4 |

**M2 and M3 can be developed in either order once M1 is accepted**. Avoid coupling D0–D2 diagnostics to M3, and E0–E4 environment work to Platform database backups. Missing upstream capability blocks only the affected feature.

## Document handoff list

- [M0 capability audit](mcp-integration/M0_SOURCE_AND_RUNTIME_CAPABILITY.md)
- [M1 routing and policy](mcp-integration/M1_ROUTING_AND_POLICY.md)
- [M2 selected Platform reuse](mcp-integration/M2_PLATFORM_REUSE.md)
- [M3 agent-box read-only](mcp-integration/M3_AGENT_BOX_READONLY.md)
- [M4 agent-box execution](mcp-integration/M4_AGENT_BOX_EXECUTION.md)
- [M5 selected Web/workflows](mcp-integration/M5_WEB_AND_WORKFLOWS.md)

For each approved future implementation slice, coding Agent must: refresh source HEAD; read [AGENTS.md](../../AGENTS.md), [accepted architecture](ARCHITECTURE.md), [security contract](MCP_INTEGRATION_SECURITY.md), [test plan](../verification/MCP_INTEGRATION_TEST_PLAN.md); name exact touched files, feature/REQ acceptance mapping, tool version and test lane; do small changes without changing the public tool catalog inadvertently; write actual S/D/T/U evidence and remaining limits.

## Go/no-go and rollback principles

- No claimed feature is complete on source-only discovery or fixture tests. Actual runnable target must be separately approved; record installed source vs running version.
- Break-glass failure: if access/trust/tool routing is broken, retain the original host CLI path; do not repair by disabling host pin checks or assuming root access.
- Reverse app additions by feature configuration/release rollback only after checking persistent SQLite migration compatibility, possible already dispatched work, and scoped actor/target. No whole-host rollback promise.
- Never automatically delete test boxes or rebuild a disputed target. Preserve original evidence when create/restore/stop returns unknown.
- PR/development code rollout is separate from an external Provider deployment; installation/CI green does not mean live MCP passes.

## Feature candidates and explicit deferrals

**First potential useful candidates:** `debug_container` error classification, cautious auto-sleep plan, `tail_log`/read-only agent-box; these directly improve existing BoxWeave features. **Later:** process/exec, project file ops, `code_pending/answer`, selective Compose, database backup views. **Deferred by default:** `get_secret`, KMS, broad admin, provider upgrade, JWT revocation, unrestricted Shell, unrestricted Web tool editor. **Independent proposals:** D0–D2 diagnostics and E0–E4 snapshots/clone; their status remains NOT IMPLEMENTED.

**Backlinks:** [master](MCP_CAPABILITY_INTEGRATION_MASTER.md), [audit](../verification/MCP_CAPABILITY_SOURCE_AUDIT.md), [existing diagnostic](DIAGNOSTICS_PROPOSAL.md), [existing environment](BOX_ENVIRONMENT_PROPOSAL.md).
