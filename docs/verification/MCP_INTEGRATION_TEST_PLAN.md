# MCP reuse — acceptance and evidence matrix

**Update 2026-10-31 — source integrated (PR #9 merged):** M0–M5 selected source code is in `main` (merge `8038afc`), with fixture CI [41191096024](https://github.com/vanishs/boxweave/actions/runs/41191096024) passing. **Real Box/Provider/SSH/agent-box tests were explicitly deferred**; no full per-client RBAC, production authorization/deployment or release publication is claimed. For exact implemented scope, remaining decisions and release boundaries, see [M0–M5 merged-source handoff](../process/MCP_M0_M5_SOURCE_PROGRESS.md). Earlier “proposal-only/not implemented” status statements below refer to the original research baseline, not this merged source status.


**2026-10-31 main reconciliation:** This original inventory was compiled at BoxWeave v0.21.4; latest reviewed main is v0.22.0 (`aaa8362decd55110eadf8c5caa7ff1b3d1d80ac5`). D0–D2 diagnostics were implemented in main (SRC-108), with remaining live/coverage limits; optional D3 direct-gateway diagnostic lookup remains unimplemented. E0–E4 environment features remain unimplemented. All M0–M5 MCP capability reuse stages are proposal-only here (separately tracked implementation must cite exact source/CI evidence). Previously written statements that D0–D2 are unimplemented are superseded by this notice. Source inventory does not establish installed tool availability.


**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.


## Evidence and test lanes

**S:** code/schema inspected at pinned SHA. **D:** prior/historical report reviewed, not rerun. **T:** a real command/test was executed with recorded environment, exit and evidence. **U:** not tested. No new T evidence in this document.

- Knowledge/link checks: `python scripts/check_knowledge.py` (read-only when an actual checkout is available).
- Source/unit/fixture: `.venv/bin/python scripts/validate.py` in separately prepared non-root development environment. Browser flag only for approved fixture lane.
- Live: a separately authorized exact disposable Box with explicit outer/inner identity, installed Containarium/agent-box/BoxWeave versions and targeted scopes. Never use production private keys or run remote mutations under a repository-doc request.

## Suggested acceptance cases (NOT EXECUTED)

| ID | Case | Required positive evidence | Fail-closed or negative evidence |
| --- | --- | --- | --- |
| MCP-G0-01 | Platform `tools/list` at pinned real upstream | Negotiated protocol, full paginated list, name/Schema digest, scope and installed version | Missing launcher, stale JWT, invalid cursor, old release: `not_verified/unavailable`, not pass |
| MCP-G0-02 | agent-box read-only probe on exact allowed target | Trusted host pin, identity and MCP init/tools/resources supported | Missing binary, stopped Box, host mismatch, foreign Box, unsupported resource: no fallback |
| MCP-G0-03 | Runtime vs source inventory | Diff reason for 109-source/installed count, explicit uncovered tools | Do not equate a name with exact allowed schema or executed capability |
| MCP-G1-01 | Existing Platform client compatibility | Old `initialize/tools/list/tools/call` create/connect schema remains valid after policy-approved migration | No silent tool removal, hide breaking privileges or weaken trust |
| MCP-G1-02 | Two caller trust zones | Direct Agent and Web identities classified independently, separate scopes | SSH key alone cannot impersonate Web/Project, no cross-Box grant |
| MCP-G1-03 | Sensitive tool denial | Unknown/secret/admin name rejected **before dispatch** with safe code | Prove no upstream request on denied `get_secret`/`upgrade_backend`; honor explicit compatibility decision |
| MCP-G1-04 | Duplicate names | Platform `compose_discover` and Box `compose_discover` select different explicit routes | No unqualified or wrong-Box execution |
| MCP-G2-01 | `debug_container` | Authorized exact target, classified useful diagnostics | Foreign target and unexpected raw log/secret output refused/redacted |
| MCP-G2-02 | `toggle_auto_sleep` (if approved) | Plan/confirmation, observed state transition and durable reconciliation | Idle-stop != user-requested stop/delete; timeout unknown, no repeat write |
| MCP-G3-01 | File/list reads | Exact Box, allowed workspace path, bounded sizes and UTF-8/media contract | Symlink/path traversal, missing root, other Box/tenant, unsupported binary fail safely |
| MCP-G3-02 | Logs/process reads | Bounded records with provenance and identified process | No unbounded log retention or unredacted secret spill |
| MCP-G3-03 | CI resources | `resources/list/read` with selected Box and correct URI namespace | No resource URI collision across two Boxes; missing resource != fabricated CI |
| MCP-G4-01 | Authorized shell/process | Exact Box, verified command scope, timeout/exit/stdout/stderr semantics, receipt | Shell cannot be treated as file-root confined; denied or stale identity no execution |
| MCP-G4-02 | Process lifecycle | `process_start` acknowledgement, observe/list/log, authorized kill | Start ACK not completion; disconnect/timeout does not authorize restart |
| MCP-G4-03 | File mutation | Atomic write/rename/cleanup identity and bounded result | Denied directory/delete/traversal, concurrent state mismatch; no silent fallback |
| MCP-G5-01 | Upstream run approval | Match run, pending request and signed upstream answer; record disposition | A `code_answer` or internal `approve` cannot approve unrelated BoxWeave writes |
| MCP-G5-02 | Optional Web UI | Existing project/lifecycle/Tenant/route UI unaffected; only selected operations exposed | No raw 109-tool editor, secret input leakage or new host network exposure |
| MCP-G5-03 | Backup distinction | DB backup verification evidence labelled DB-only | Cannot call `create_backup` an instance snapshot/clone; E0–E4 remain unimplemented |
| MCP-GR-01 | Fault injection | EOF, invalid JSON, malformed IDs, wrong Schema, wrong actor, stale target reported accurately | No unchecked retry, duplicate create, hidden upgrade, trust bypass |
| MCP-GR-02 | Legacy regressions | Existing client registration, hosted `connect`, nesting, Web receipt/history and source validation retained | Earlier fixed failures not reintroduced; no claim about untested OS |

## Required evidence record per future implemented slice

```json
{
  "source_commit": "reviewed-git-sha",
  "installed_boxweave": "not_verified",
  "installed_containarium_mcp": "not_verified",
  "installed_agent_box": "not_verified",
  "provider_backend_project": "not_verified",
  "exact_box_identity": "not_verified",
  "test_case": "MCP-G0-01",
  "environment": "source_only",
  "command": "not_executed",
  "exit_code": null,
  "result": "not_tested",
  "evidence_class": "S",
  "scope_and_auth": "not_verified",
  "limits_and_next_actions": []
}
```

An automated Agent may choose test steps within an **already specifically authorized** disposable environment or draft a plan-first sequence. Neither mode authorizes privileged host mutation, uncontrolled new Box creation, Provider install or cleanup without scope approval; see [active developer test workflow](../process/AGENT_TEST_WORKFLOW.md).

**Backlinks:** [master](../design/MCP_CAPABILITY_INTEGRATION_MASTER.md), [audit](MCP_CAPABILITY_SOURCE_AUDIT.md), [security](../design/MCP_INTEGRATION_SECURITY.md), [delivery](../design/MCP_INTEGRATION_DELIVERY_PLAN.md), [diagnostic D0–D2](../design/DIAGNOSTICS_PROPOSAL.md), [environment E0–E4](../design/BOX_ENVIRONMENT_PROPOSAL.md).

## Implementation evidence checkpoint (2026-10-31)

See [handoff](../process/MCP_REUSE_IMPLEMENTATION_HANDOFF.md): PR #7 has source-only partial M0/M2 with fixture CI run [41188227613](https://github.com/vanishs/boxweave/actions/runs/41188227613) successful. None of the installed Provider/agent-box/live Box checks in this matrix were actually run. **M1/M3/M4/M5 acceptance is pending.**
