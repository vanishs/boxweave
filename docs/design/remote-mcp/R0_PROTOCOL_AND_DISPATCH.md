# R0 — Protocol inventory and shared gateway dispatch

Parent: [remote MCP master](../REMOTE_MCP_MASTER.md). Status: **SOURCE IMPLEMENTED WITH DEVIATION / LIVE NOT TESTED**. Dependencies: none. Next: [R1](R1_STREAMABLE_HTTP.md).

## Goal

Introduce a transport-neutral, testable MCP gateway core without weakening the existing SSH/stdio client path or granting Web/HTTP callers implicit upstream authority.

## Existing source touchpoints

- `bwmini/mcp_gate.py`: stdio message framing, Provider upstream launcher, managed workload-key injection, nesting check, optional protected allowlist.
- `bwmini/mcp.py`: outbound MCP stdio client used by the application; do not mistake this for the inbound server.
- `bwmini/provider.py` / `bwmini/service.py`: typed app-owned Provider and action boundaries.
- `bwmini/web.py` and `bwmini/store.py`: single-admin Web session/private SQLite, not shared SSH identity.
- `tests/test_mcp_gate_policy.py`, `tests/mcp_peer.py` and current client profile tests.

## Implementation tasks

1. Snapshot the exact `tools/list` pagination, tool schemas, `tools/call` effects and `resources/list` / `resources/read` behavior available at the selected installed Provider; source-only inventory must be labeled source, not live.
2. Extract one **gateway request dispatcher** with explicit server-assigned origin context, named upstream launcher and existing policy/managed identity checks. Accept only protocol request fields, not caller-supplied launcher/authority data.
3. Keep SSH stdio as one adapter. HTTP adapter will be another (R1), not a fork of tool definitions or a Web REST proxy to arbitrary Provider methods.
4. Preserve existing tool rewriting and managed SSH key injection for create; reject partial/invalid transform input before Provider dispatch. Allowlist policy remains opt-in and host-wide exactly as current behavior.
5. Preserve the one-trusted-operator security disclosure: no new per-Agent/per-Box policy, no new generic ability to reach host shell via Web.
6. Define explicit capability/version map for protocol negotiation and bounded requests; never list unsupported resources/prompts/subscriptions to clients.
7. Add a single test fixture for representative successful read, transformed create, existing nesting tool, forbidden tool when allowlist enabled, malformed requests and upstream transport loss.

## Acceptance / rollback

- Existing SSH/stdio packet fixture, forced-command enrollment and pinned-host behavior unchanged.
- New shared core rejects unauthorized caller-supplied upstream credentials, arbitrary OS commands/launcher paths and unknown protected transforms.
- Tool listings and call effects match the original SSH path under **the same configured operator authority**; no promise of same per-client permissions.
- On error, existing stdio remains available; extraction may be independently reverted without DB migration.

Evidence: [remote MCP acceptance plan](../../verification/REMOTE_MCP_TEST_PLAN.md), gates R0 and negative N0. Record exact test command, exit, source SHA and whether tests were fixture or live.
