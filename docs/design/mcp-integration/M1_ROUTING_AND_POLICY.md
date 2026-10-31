# M1 — Routing, policy and direct-gateway compatibility

**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.

**Parent:** [capability-led MCP master](../MCP_CAPABILITY_INTEGRATION_MASTER.md). **Related:** [delivery](../MCP_INTEGRATION_DELIVERY_PLAN.md), [security](../MCP_INTEGRATION_SECURITY.md), [acceptance](../../verification/MCP_INTEGRATION_TEST_PLAN.md), [diagnostics D0–D2 — NOT IMPLEMENTED](../DIAGNOSTICS_PROPOSAL.md), [Box environments E0–E4 — NOT IMPLEMENTED](../BOX_ENVIRONMENT_PROPOSAL.md).

## Goal and non-goals

Design a common metadata registry for capabilities while preserving **different actual principals** for direct Agent SSH and Web/worker. Do not promise that an app-side policy wrapper protects calls that bypass it through existing direct-Agent passthrough. Tool authorization must be real server-side enforcement, not prompt instructions or a Web-only UI.

## Candidate source touchpoints (no edits authorized)

- `bwmini/mcp_gate.py` (direct gateway), `mcp.py` (bounded app stdio), `service.py`, `lifecycle.py`, `access.py`, `store.py` (existing app authorities).
- Candidate new modules `tool_registry.py` and `tool_policy.py` only if review proves separation avoids duplication; names are provisional.
- `client_profiles.py` and tests for old `tools/list`/client registration compatibility.

## Developer sequence (conditional on future approval)

1. Define canonical key `(upstream_source, upstream_tool, schema_digest, caller_boundary, selected_box_identity?)`; keep user-facing Platform names unchanged.
2. Resolve naming collisions including Platform and in-Box `compose_discover`; decide explicit prefix for Box-scoped tools and resource URIs.
3. Decide and implement, **only after approval**, how actor scopes bind to direct Agent transport and separate Web session. No implicit actor derivation from SSH key name or agent prompt.
4. Gate sensitive/unknown tools in **the path that could reach upstream**; negative tests prove rejection before request dispatch. Evaluate replacing broad Platform credentials with scoped credentials instead of adding a decorative policy class.
5. Provide legacy operator migration reports; maintain exact create/connect/nesting semantics, protocol-version compatibility and bounded error responses.
6. Keep a read-only capability/status view separate from execution permission. No new open server socket, auto-enrollment or raw secret logging.

## Required negative/boundary tests

- G1-01: direct Agent can bypass Web-only policy: test and reject false protection claim.
- G1-02: old clients with tool name/cursor/Schema expectations.
- G1-03: unknown tool, conflicting name, forbidden admin/secret request, expired caller scope.
- G1-04: stale tool schema or exact target identity; no upstream side effect.
- G1-05: gate unavailable but host CLI doctor still responds independently.

## Exit criteria and handoff

Accept only with maintainer-approved compatibility policy, tests of **both** Agent and Web callers, a documented default deny/legacy exception migration strategy, and explicit evidence of enforcement. Otherwise remain a design and do not claim least privilege.

**Hard stop:** The current broad direct gateway is a product/security constraint. Do not silently change trusted operator access under the guise of internal refactoring.

**Next step:** document accepted maintainer decisions in [open questions](../../knowledge/OPEN_QUESTIONS.md) and normative requirements/ADR only if selected. A documentation proposal is never a feature release.
