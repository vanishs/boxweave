# M2 — Reuse the Platform tools that improve existing features

**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.

**Parent:** [capability-led MCP master](../MCP_CAPABILITY_INTEGRATION_MASTER.md). **Related:** [delivery](../MCP_INTEGRATION_DELIVERY_PLAN.md), [security](../MCP_INTEGRATION_SECURITY.md), [acceptance](../../verification/MCP_INTEGRATION_TEST_PLAN.md), [diagnostics D0–D2 — NOT IMPLEMENTED](../DIAGNOSTICS_PROPOSAL.md), [Box environments E0–E4 — NOT IMPLEMENTED](../BOX_ENVIRONMENT_PROPOSAL.md).

## Goal and non-goals

Add **selected** upstream capability to **existing** BoxWeave domains; avoid general API/GUI pass-through. First candidate is safe classification of `debug_container` diagnostics. Second candidate `toggle_auto_sleep` only after reviewed idle-state, lifecycle ownership, and explicit plan/confirm semantics. Other candidates (`get_metrics_export`, `code_logs`) are deferred unless an actual product gap is demonstrated.

## Candidate source touchpoints (no edits authorized)

- `bwmini/provider.py` exact typed adapter and `mcp.py` transport.
- `bwmini/host_cli.py` and diagnostic proposals [D0–D2](../DIAGNOSTICS_PROPOSAL.md) (do not duplicate or assert implemented).
- `bwmini/lifecycle.py`, `store.py`, `service.py`, `web.py` only for an **approved** mutable auto-sleep design.

## Developer sequence (conditional on future approval)

1. Verify actual upstream `debug_container` tool Schema/scope/backend and target ownership; define a bounded allowlist of safe diagnostic categories (never raw unlimited log strings).
2. Keep host doctor failure handling independent of the upstream tool. If D0 is selected, implement its truthful health exits under its own approval.
3. For auto-sleep, document idle detection, status when stopped, wake/start reconciliation, operator stop/delete precedence, budget, pending operations, and failure scenarios. Do not assume `toggle_auto_sleep` alone implements automatic resume.
4. Route approved changes through typed plans and durable single dispatch; no optimistic state change on upstream ACK, no automatic retry after timeout.
5. Offer Web representation only if needed later in M5; do not create a new generic admin console for upstream settings.

## Required negative/boundary tests

- G2-01: missing tool, wrong Schema/scope, foreign/stale target, redaction leak.
- G2-02: auto-sleep is unsupported or disabled in current Provider/backend.
- G2-03: idle stop racing with user start/stop/delete, worker crash, transport timeout.
- G2-04: upstream mutation acknowledgement without actual state transition; mark observing/unknown.
- G2-05: diagnostics of failed client SSH must not expose host/tenant logs beyond authorization.

## Exit criteria and handoff

Diagnostics-only slice passes only if classification is truthful, secret-safe, and tied to exact target. Auto-sleep is separately shippable **only** if safe state transitions and revival contract are tested against an approved real Box. Each may be deferred independently.

**No equivalence claim:** Upstream `toggle_auto_sleep` is not a complete BoxWeave sleep/resume/resource recovery product by itself.

**Next step:** document accepted maintainer decisions in [open questions](../../knowledge/OPEN_QUESTIONS.md) and normative requirements/ADR only if selected. A documentation proposal is never a feature release.
