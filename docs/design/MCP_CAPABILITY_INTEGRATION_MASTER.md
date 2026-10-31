# BoxWeave capability-led MCP reuse — master plan

**Implementation checkpoint 2026-10-31:** partial M0 app-side tool discovery and M2 bounded debug summary added in implementation PR #7 with fixture CI evidence; **M1/M3/M4 and most M5 NOT IMPLEMENTED**. See [implementation handoff](../process/MCP_REUSE_IMPLEMENTATION_HANDOFF.md) for actual status, tests, release limitations and remaining decisions.

**2026-10-31 main reconciliation:** This original inventory was compiled at BoxWeave v0.21.4; latest reviewed main is v0.22.0 (`aaa8362decd55110eadf8c5caa7ff1b3d1d80ac5`). D0–D2 diagnostics were implemented in main (SRC-108), with remaining live/coverage limits; optional D3 direct-gateway diagnostic lookup remains unimplemented. E0–E4 environment features remain unimplemented. All M0–M5 MCP capability reuse stages are proposal-only here (separately tracked implementation must cite exact source/CI evidence). Previously written statements that D0–D2 are unimplemented are superseded by this notice. Source inventory does not establish installed tool availability.


**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.


## Decision and scope

**Recommendation:** start from existing BoxWeave product features and verified gaps, **not** a project to repackage every upstream MCP tool. Preserve the current direct-Agent SSH/MCP gateway, independent app-side MCP Provider adapter, Web/worker/SQLite lifecycle and host CLI. Reuse specific Containarium Platform MCP and agent-box MCP capabilities where a gap is real and the installed contract, scope and trust boundary can be verified.

This document requests a **decision on integration direction**, not authorization to change code, services, hosts, token grants, box images or network routes. The current single trusted operator is not full per-client, multi-user RBAC. The upstream Platform MCP may already be forwarded by the Agent gateway; that is not equivalent to application-level managed integration or least-privilege authorization.

### What remains unchanged

- Clients use existing BoxWeave public SSH endpoint and enrollment. Web is independent and remains optional for basic Agent transport.
- `bwmini/mcp_gate.py` launches and relays upstream Platform MCP; it currently overlays managed SSH keys on `create_container`, checks trust on `connect`, and adds `boxweave_nesting`.
- `bwmini/provider.py` and `access_provider.py` call a separately configured application MCP launcher, behind typed lifecycle/network preflight and durable receipts.
- `bwmini/host_cli.py` remains the separate, local recovery/diagnostic surface; it must work when the Agent MCP or Web cannot.
- Do not bypass Provider controls using direct host Incus, a generic shell, an unreviewed REST adapter, or a second state store.
- Existing diagnostic D0–D2 and environment E0–E4 proposals remain **NOT IMPLEMENTED**; this proposal is their potential integration substrate, not a substitute for approval.

## Current topology vs suggested reuse

```text
Agent client -> enrolled SSH -> boxweave forced-command gate -> Platform MCP -> Provider/Incus
Browser -> Web -> typed Service / durable worker -> app Provider -> Platform MCP
Operator -> host CLI -> safe local observations (independent of MCP)

Proposed, conditional only:
  Web/app Service -> authorized named Box selector -> Box adapter
      -> verified per-Box SSH -> agent-box MCP within that Box
  Existing direct-Agent gate -> policy-checked optional Box-facing tools
      -> named Box adapter (NOT raw caller-selected SSH/hostname)
```

The first two existing flows do **not** collapse into a single security process. A common catalogue/policy model must have explicit separately validated caller identities and scope for each entry. A catalogue is a metadata inventory, not a credential or authorization grant.

## Key findings and candidate product value

| Current feature | Source-established state | Useful upstream candidate | Assessment / decision |
| --- | --- | --- | --- |
| Box lifecycle and exact identity | App already implements create/start/stop/delete/resize, identity-bound plan/confirm, observation and unknown-result preservation | `toggle_auto_sleep` | Candidate **P1**; define asleep vs stopped and reconcile observed state, never mark sleep as deletion |
| SSH/connect and diagnostics | Gate validates target/host-key on `connect`; Web fixed `hello world` and host `doctor` exist, but diagnostics have evidence gaps | `debug_container`; agent-box `tail_log` | Candidate **P1** classified read-only summaries; D0 diagnostic correctness precedes new debug claims |
| Box-internal execution | App uses upstream `connect` for *fixed* commands, not general project shell | agent-box `shell_exec`, `process_*` | Candidate **P2**; explicit target and execution authorization, no automatic arbitrary Web shell |
| Project/workspace | Typed project->target binding and fixed hello; no general file gateway | agent-box `read_file`, `list_directory`, then writes | Candidate **P2**; protect tenant/workspace, confirm agent-box installation and roots; don't duplicate local Agent tools without real need |
| Agent jobs and approvals | Existing host jobs, selftest and receipts; no claimed BoxWeave agent task approval product | Platform `code_runs/logs/pending/answer`, in-run approval MCP | Candidate **P3** scoped task approval; cannot treat separate signatures/permissions as interchangeable |
| Backups and environment snapshots | E0–E4 plans only, no verified snapshot/clone Provider adapter | Platform `create_backup/list_backups/verify_backup` | **Different feature:** database backup is *not* Incus instance snapshot; no E2/E3/E4 shortcut |
| Security and secrets | SSH keys/trust and upstream boundaries exist; no general secret management product | `security_findings`, `security_scan`, `get_secret`, KMS tools | Read-only exposure **P3** after policy; secret retrieval and host admin **defer/deny** pending explicit product authorization |
| Container Compose | No reviewed BoxWeave Compose UI/adapter | Platform `compose_*` and in-box `compose_autostart_*` | Candidate **P3**; names and locations overlap, choose one authoritative path |
| Web | Typed project, Tenant, route, lifecycle, graph, profile features already exist | no need to mirror all upstream tools | Show only domain-relevant capabilities, state and receipts, not 100+ generic forms |

**P0** is accurate capability/audit and policy boundary, **P1** safe differentiated reads and auto-sleep assessment, **P2** scoped Box operations, **P3** optional Web/workflow adoption. These are proposal priorities, **not** accepted REQ or implementation status.

## Product ownership and integration rules

1. **Native BoxWeave domain first:** keep existing typed lifecycle, access and receipt services authoritative for app-owned plans; never let direct tools silently bypass a promise the Web made.
2. **Upstream name preservation:** existing Platform tool names remain externally stable; do not unilaterally rename `create_container`, `connect` or `boxweave_nesting`.
3. **Explicit Box naming:** agent-box tools are a separate domain. Candidate `boxweave_box_<operation>` tools carry an exact authorized target identity selected by BoxWeave, never an arbitrary SSH host or raw credential.
4. **No "all tools enabled" switch:** distinguish `catalogued`, `advertised_by_installed_upstream`, `authorized`, `compatible`, `tested` and `implemented_in_boxweave`. New names must not automatically gain execution permission.
5. **Preserve transport correctness:** the current app MCP client speaks a pinned 2024-11-05 protocol and accepts only text tool content. Confirm agent-box actual initialize/resources/tools behavior before reuse; never assume the current client fully supports it.
6. **No silent deployment:** agent-box availability may differ per Box. Absence is `unavailable`, not permission to install a binary, log in as root or switch host credentials.

## Deliverable map (this proposal)

| Document | Role |
| --- | --- |
| [Source and gap audit](../verification/MCP_CAPABILITY_SOURCE_AUDIT.md) | Current functionality vs opportunities; source/evidence precision |
| [Platform tool matrix](MCP_PLATFORM_CAPABILITY_MATRIX.md) | Complete *source-defined* upstream inventory and BoxWeave app usage |
| [agent-box mapping](MCP_AGENT_BOX_CAPABILITY_MATRIX.md) | 14 tools, 2 resources, distinct approval MCP |
| [Security/authorization](MCP_INTEGRATION_SECURITY.md) | Caller, target, credentials, permission boundary, compatibility migration |
| [Delivery/batch dependencies](MCP_INTEGRATION_DELIVERY_PLAN.md) | Priority, ownership, gates and stop conditions |
| [Acceptance specification](../verification/MCP_INTEGRATION_TEST_PLAN.md) | Negative cases, evidence classes and real-test requirements |
| [M0 capability/source audit](mcp-integration/M0_SOURCE_AND_RUNTIME_CAPABILITY.md) | Discovery/evidence before changes |
| [M1 routing/policy design](mcp-integration/M1_ROUTING_AND_POLICY.md) | Policy enforcement with compatibility |
| [M2 selected Platform reuse](mcp-integration/M2_PLATFORM_REUSE.md) | Diagnostic read and idle-state candidate |
| [M3 Box read-only access](mcp-integration/M3_AGENT_BOX_READONLY.md) | Authenticated per-Box reads/resources |
| [M4 Box execution](mcp-integration/M4_AGENT_BOX_EXECUTION.md) | Files/processes/commands |
| [M5 optional Web/workflows](mcp-integration/M5_WEB_AND_WORKFLOWS.md) | Safe product-level features only |

### Existing design links — independent, still unimplemented

- [Diagnostics research and D0–D2](DIAGNOSTICS_PROPOSAL.md), [diagnostics source audit](../verification/DIAGNOSTICS_SOURCE_AUDIT.md).
- [Box environment E0–E4](BOX_ENVIRONMENT_PROPOSAL.md) and [E0 capability contract](BOX_ENV_E0_CAPABILITIES.md).
- [Actual implementation boundaries](ARCHITECTURE.md), [upstream ownership](../UPSTREAM_PROVIDER.md), [nesting behavior](../NESTING.md), [Agent test practice](../process/AGENT_TEST_WORKFLOW.md).

## Implementation order and acceptance discipline

**M0 -> M1 -> M2 -> M3 -> M4 -> M5**, with M2 and M3 separable after M1. All changes require an explicit later coding authorization. Before any runtime claim: pin deployed versions, list negotiated tool/resource schemas, prove effective scopes, run targeted negative tests, and record original identity + operation receipt. The high-risk hosted tool set must not be widened just to gain a convenient UI.

**Design-only handoff:** The maintainer must decide (a) whether any direct-Agent gateway tool filtering/restriction is approved despite compatibility effects, (b) how caller identity/authorization maps to app project/Box identity, and (c) which selected reuse slice, if any, is approved first. Document decisions in [open questions](../knowledge/OPEN_QUESTIONS.md), then update SCOPE/REQ/AC/ADR only if product scope changes.

**Backlink convention:** Every child spec links to this master and to the relevant existing diagnostics/environment documents. Any completed code slice must mark tests and evidence independently; no checkboxes are marked complete here.
