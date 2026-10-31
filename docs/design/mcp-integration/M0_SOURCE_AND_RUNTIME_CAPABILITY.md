# M0 — Source and installed MCP capability verification

**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.

**Parent:** [capability-led MCP master](../MCP_CAPABILITY_INTEGRATION_MASTER.md). **Related:** [delivery](../MCP_INTEGRATION_DELIVERY_PLAN.md), [security](../MCP_INTEGRATION_SECURITY.md), [acceptance](../../verification/MCP_INTEGRATION_TEST_PLAN.md), [diagnostics D0–D2 — NOT IMPLEMENTED](../DIAGNOSTICS_PROPOSAL.md), [Box environments E0–E4 — NOT IMPLEMENTED](../BOX_ENVIRONMENT_PROPOSAL.md).

## Goal and non-goals

Create an accurate source-vs-installed capability catalogue for both upstreams and for **each separately authorized caller entry**. Avoid any mutation. Do not claim the upstream Platform 109 tools or agent-box 14 tools are available on the current host until the actual authenticated protocol says so. Read-only source audits can proceed without a live host.

## Candidate source touchpoints (no edits authorized)

- `bwmini/mcp.py`, `mcp_gate.py`, `provider.py`, `access_provider.py`, `deploy_check.py` (review only).
- Containarium `internal/mcp/tools.go` + registration append modules; `internal/agentbox/server.go`, `cmd/agent-box/main.go`.
- Output evidence in [source gap audit](../../verification/MCP_CAPABILITY_SOURCE_AUDIT.md) and [Platform matrix](../MCP_PLATFORM_CAPABILITY_MATRIX.md), [Box matrix](../MCP_AGENT_BOX_CAPABILITY_MATRIX.md).

## Developer sequence (conditional on future approval)

1. Freeze BoxWeave and upstream source revisions. Enumerate names and resolve duplicate names **per source**, then hash exact advertised input schemas.
2. On a separately approved installed test environment, execute only `initialize`/`tools/list` for Platform (with pages), compare required JWT scopes, and observe a harmless authorized `list_containers`.
3. For agent-box, test the specifically allowed Box's host pin, installed binary and `initialize`/`tools/list`/`resources/list`; do not assume the app's Platform MCP client can handle agent-box's protocol behavior.
4. Record `available`, `unsupported`, `not_authorized`, `not_verified`, `contract_mismatch` and version evidence separately. No generic failure collapsed to success.
5. Identify whether any tool is actually used by app code versus merely passed through the direct gateway; produce a versioned review mapping, not a runtime grant.

## Required negative/boundary tests

- G0-01: old and new installed upstream version disagree with source catalog.
- G0-02: missing/invalid scope means hidden tools, not `unsupported`.
- G0-03: wrong Box identity, host pin, command path or no agent-box binary.
- G0-04: MCP pagination/cursor, wrong JSON-RPC ID, non-text content and resources unsupported by current client.
- G0-05: no live access: honestly mark all installed/runtime properties `U`, do not substitute a fixture.

## Exit criteria and handoff

Ready when all candidate mappings have a reviewed source owner, **actual runtime statuses only where measured**, a list of unresolved scopes and a compatibility-impact report. M1 must not begin enforcing new access policy without the maintainer's decision about existing direct gateway behavior.

**No mutation:** This batch must not create/stop/resize Boxes, open routes, enroll new credentials or install agent-box.

**Next step:** document accepted maintainer decisions in [open questions](../../knowledge/OPEN_QUESTIONS.md) and normative requirements/ADR only if selected. A documentation proposal is never a feature release.
