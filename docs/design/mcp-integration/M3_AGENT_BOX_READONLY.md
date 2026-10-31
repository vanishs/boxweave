# M3 — Bounded read-only agent-box MCP adapter

**2026-10-31 main reconciliation:** This original inventory was compiled at BoxWeave v0.21.4; latest reviewed main is v0.22.0 (`aaa8362decd55110eadf8c5caa7ff1b3d1d80ac5`). D0–D2 diagnostics were implemented in main (SRC-108), with remaining live/coverage limits; optional D3 direct-gateway diagnostic lookup remains unimplemented. E0–E4 environment features remain unimplemented. All M0–M5 MCP capability reuse stages are proposal-only here (separately tracked implementation must cite exact source/CI evidence). Previously written statements that D0–D2 are unimplemented are superseded by this notice. Source inventory does not establish installed tool availability.


**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.

**Parent:** [capability-led MCP master](../MCP_CAPABILITY_INTEGRATION_MASTER.md). **Related:** [delivery](../MCP_INTEGRATION_DELIVERY_PLAN.md), [security](../MCP_INTEGRATION_SECURITY.md), [acceptance](../../verification/MCP_INTEGRATION_TEST_PLAN.md), [diagnostics D0–D2 — NOT IMPLEMENTED](../DIAGNOSTICS_PROPOSAL.md), [Box environments E0–E4 — NOT IMPLEMENTED](../BOX_ENVIRONMENT_PROPOSAL.md).

## Goal and non-goals

Make a first useful Box-scoped surface: `read_file`, `list_directory`, `tail_log`, `process_list`, `compose_discover`, `compose_autostart_status` and two CI resources **only where installed and permitted**. Keep this read-only; no arbitrary shell, auto-install or host mutation.

## Candidate source touchpoints (no edits authorized)

- Potential `bwmini/agent_box.py` Box-only stdio/SSH adapter; reuse `mcp.py` transport parts **after** protocol verification, not by disabling validation.
- `provider.py` to revalidate exact target, `domain.py` for immutable identity, `service.py` for actor/workspace binding; no new standalone store unless justified.
- `mcp_gate.py` optional external tool exposure only after M1 and explicit direct-Agent caller authorization review.

## Developer sequence (conditional on future approval)

1. For a named authorized managed Box, observe provider/backend/project/logical/created identity; check running state and trusted host key, then negotiate in-box MCP.
2. Map commands to fixed vetted remote `agent-box` executable path/launch policy, never accept caller-provided SSH destinations/arguments or keys.
3. Support `resources/list` and `resources/read` where proven, with per-Box URI handling and explicit cache invalidation, alongside tools/list pagination.
4. Constrain file reads to reviewed workspace and Box filesystem permissions; ensure symlink/canonicalization, size bounds, and denial of foreign project access. For logs/process arguments use stricter disclosure policy than ordinary file listing.
5. Detect missing/uninstalled agent-box, no upstream scopes, changed contract, or unsupported client media types; preserve a useful diagnosis without replacing with `incus exec`.

## Required negative/boundary tests

- G3-01: identical resource URI from two different Boxes cannot collide.
- G3-02: Box replaced under same logical name, wrong host pin or wrong user.
- G3-03: symlink traversal, path outside root, client roots missing, unbounded logs/outputs.
- G3-04: unsupported `resources/read`, binary/image content, non-text result, invalid JSON IDs.
- G3-05: malformed/missing agent-box must return not_verified/unavailable; never install it silently.

## Exit criteria and handoff

Passes with trusted exact Box identity, bounded read-only tools/resources, failed negative cases, and actual valid file/log read from an explicitly authorized test Box. A healthy Platform MCP cannot stand in for this evidence.

**Upstream caveat:** `AGENTBOX_ROOT` protects file-tool paths only; it does not create an OS-level Shell sandbox.

**Next step:** document accepted maintainer decisions in [open questions](../../knowledge/OPEN_QUESTIONS.md) and normative requirements/ADR only if selected. A documentation proposal is never a feature release.
