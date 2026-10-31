# M5 — Product-specific Web and Agent workflows

**2026-10-31 main reconciliation:** This original inventory was compiled at BoxWeave v0.21.4; latest reviewed main is v0.22.0 (`aaa8362decd55110eadf8c5caa7ff1b3d1d80ac5`). D0–D2 diagnostics were implemented in main (SRC-108), with remaining live/coverage limits; optional D3 direct-gateway diagnostic lookup remains unimplemented. E0–E4 environment features remain unimplemented. All M0–M5 MCP capability reuse stages are proposal-only here (separately tracked implementation must cite exact source/CI evidence). Previously written statements that D0–D2 are unimplemented are superseded by this notice. Source inventory does not establish installed tool availability.


**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.

**Parent:** [capability-led MCP master](../MCP_CAPABILITY_INTEGRATION_MASTER.md). **Related:** [delivery](../MCP_INTEGRATION_DELIVERY_PLAN.md), [security](../MCP_INTEGRATION_SECURITY.md), [acceptance](../../verification/MCP_INTEGRATION_TEST_PLAN.md), [diagnostics D0–D2 — NOT IMPLEMENTED](../DIAGNOSTICS_PROPOSAL.md), [Box environments E0–E4 — NOT IMPLEMENTED](../BOX_ENVIRONMENT_PROPOSAL.md).

## Goal and non-goals

Only after the earlier adapter/security slices, integrate **selected** valuable outcomes into existing BoxWeave Web, CLI or Agent workflows. There is no requirement to build a generic administration UI for every Platform tool. Do not turn the existing prompt graph into an automatic scheduler without its own scoped proposal/approval.

## Candidate source touchpoints (no edits authorized)

- `bwmini/web.py`, `service.py`, `store.py`, existing React UI; current typed APIs are the source of app actions.
- `bwmini/jobs.py`, `selftest.py` and accepted Agent development practices only as references, not an automatic test product.
- Existing [diagnostics](../DIAGNOSTICS_PROPOSAL.md) and [environment](../BOX_ENVIRONMENT_PROPOSAL.md) design owners stay separate.

## Developer sequence (conditional on future approval)

1. Display real ability/status per selected Box and current caller with `not_verified` where unavailable. No promotion of source names to usable tools.
2. Select candidate features after maintainer review: safe diagnostics, user-controlled auto-sleep, Box logs/process visibility, approved file management, limited Compose status.
3. For upstream coding runs, scope `code_runs/logs/pending/answer` to the actual run owner and signed upstream approval contract. Do not expose internal `containarium-approval.approve` to general callers.
4. For Agent-directed testing: accept explicit target/scope and execution mode (auto only within prior authorization or plan-first); have Agent choose steps; persist actual run identity, result, unknown/failed cleanup, source/version evidence.
5. If database backups are ever integrated, label them as tenant **database backups**. Never claim snapshot/clone/restore until separate E0–E4 capability and identity checks pass.
6. Reuse existing Web/CLI receipt and audit mechanisms. Avoid creating a second contradictory operation store or state model.

## Required negative/boundary tests

- G5-01: unauthenticated Web user cannot read Box logs/process data or approve a task.
- G5-02: `code_answer` cannot mutate an unrelated task, Box or Provider state.
- G5-03: submitted Agent test plan exceeds authorized Box/action; stop rather than auto-expand.
- G5-04: Web failures cannot take down direct MCP or independent host doctor.
- G5-05: historical output must distinguish already complete, running, unknown, failed and not_checked.

## Exit criteria and handoff

Each selected product feature receives its own REQ/AC/ADR decision, tests and runtime evidence before release. Unselected upstream abilities remain discoverable upstream but not claimed as BoxWeave feature parity.

**Not implemented by this document:** full diagnostics D0–D2, Box Environment E0–E4, generic Agent test scheduler, broad multi-user RBAC.

**Next step:** document accepted maintainer decisions in [open questions](../../knowledge/OPEN_QUESTIONS.md) and normative requirements/ADR only if selected. A documentation proposal is never a feature release.
