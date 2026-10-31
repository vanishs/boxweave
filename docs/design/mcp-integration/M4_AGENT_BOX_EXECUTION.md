# M4 — Box file writes, commands and process execution

**2026-10-31 main reconciliation:** This original inventory was compiled at BoxWeave v0.21.4; latest reviewed main is v0.22.0 (`aaa8362decd55110eadf8c5caa7ff1b3d1d80ac5`). D0–D2 diagnostics were implemented in main (SRC-108), with remaining live/coverage limits; optional D3 direct-gateway diagnostic lookup remains unimplemented. E0–E4 environment features remain unimplemented. All M0–M5 MCP capability reuse stages are proposal-only here (separately tracked implementation must cite exact source/CI evidence). Previously written statements that D0–D2 are unimplemented are superseded by this notice. Source inventory does not establish installed tool availability.


**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.

**Parent:** [capability-led MCP master](../MCP_CAPABILITY_INTEGRATION_MASTER.md). **Related:** [delivery](../MCP_INTEGRATION_DELIVERY_PLAN.md), [security](../MCP_INTEGRATION_SECURITY.md), [acceptance](../../verification/MCP_INTEGRATION_TEST_PLAN.md), [diagnostics D0–D2 — NOT IMPLEMENTED](../DIAGNOSTICS_PROPOSAL.md), [Box environments E0–E4 — NOT IMPLEMENTED](../BOX_ENVIRONMENT_PROPOSAL.md).

## Goal and non-goals

Extend the proven read-only Box adapter to tightly scoped `write_file`, `move_file`, `delete_file`, `shell_exec`, `process_start`, `process_kill`, and only selected Compose autostart changes. These are **not** implicit consequences of a read grant or an Agent's own shell access. This phase requires separate product and real-environment authorization.

## Candidate source touchpoints (no edits authorized)

- `bwmini/agent_box.py` from accepted M3, `service.py`, `store.py`, existing lifecycle/receipts and app actor policy.
- `mcp_gate.py` only if explicit direct Agent tool exposure is approved; otherwise scope first to app-side workflows.
- Source tests for execution policy, bounded stdout/stderr and concurrent process state.

## Developer sequence (conditional on future approval)

1. Define separate execution and file-write capabilities bound to immutable Box identity, project workspace, caller and policy revision. Avoid raw generic Shell from Web by default.
2. Preflight exact command/schema/path/working directory, resource limit and output/timeout; run as nonprivileged Box user within the LXC boundary. Treat shell commands as able to access the **whole OS permission domain**, not just `AGENTBOX_ROOT`.
3. Persist a single-dispatch receipt before mutation; `unknown` is not permission to replay. `process_start` ACK is a running-process reference, not verified completion.
4. Obtain deterministic process ID/run correlation, bounded `tail_log` and safe readback; explicit authorization for `process_kill`.
5. Preserve atomically written files only within approved paths; reject symlinks/traversal, directories for delete, unauthorized existing file replacements, and mismatched revision where applicable.
6. If test workflow is selected later, link Agent-selected test plan and observed steps to receipts; don't conflate it with existing fixed `selftest.py`.

## Required negative/boundary tests

- G4-01: unauthorized command does not cross the Box adapter; no option for arbitrary SSH command field.
- G4-02: timeout/disconnect during write/process start leaves `unknown`, no automatic duplicate process.
- G4-03: symlink race, path traversal, wrong target and concurrent workspace changes.
- G4-04: process lifecycle under Box restart; PID reuse and stale log references.
- G4-05: root/host break-out attempts, environment secret spill, output size, execution exhaustion.

## Exit criteria and handoff

Pass only with both positive and denial results in a separately approved non-root disposable Box, isolated from Provider/host credentials. Preserve original test Box for unknown outcomes; never delete by name alone.

**Separate approvals:** A signed upstream run-level `approve` response is not a generalized BoxWeave command execution grant.

**Next step:** document accepted maintainer decisions in [open questions](../../knowledge/OPEN_QUESTIONS.md) and normative requirements/ADR only if selected. A documentation proposal is never a feature release.
