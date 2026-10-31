# MCP reuse security, authority and compatibility boundaries

**2026-10-31 main reconciliation:** This original inventory was compiled at BoxWeave v0.21.4; latest reviewed main is v0.22.0 (`aaa8362decd55110eadf8c5caa7ff1b3d1d80ac5`). D0–D2 diagnostics were implemented in main (SRC-108), with remaining live/coverage limits; optional D3 direct-gateway diagnostic lookup remains unimplemented. E0–E4 environment features remain unimplemented. All M0–M5 MCP capability reuse stages are proposal-only here (separately tracked implementation must cite exact source/CI evidence). Previously written statements that D0–D2 are unimplemented are superseded by this notice. Source inventory does not establish installed tool availability.


**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.


## Trust zones

| Zone | Current principal/evidence | Allowed role | Prohibited inference |
| --- | --- | --- | --- |
| Native/WSL Agent client | Enrolled SSH public key + strict server host pin | Opens restricted BoxWeave gateway | Enrolled key is *not* a scoped Web user/actor nor privilege to all boxes |
| Gate/Platform launcher | BoxWeave forced-command account, upstream launcher credential | Existing Platform tool relay and specific create/connect/nesting checks | Credential breadth does *not* equal BoxWeave per-tool policy |
| Web/worker | Authenticated local admin, CSRF/Origin, app-owned SQLite | Typed plan/confirm, Worker and scoped app Provider | Web login is not automatically passed into Agent SSH |
| App Provider | Independently configured MCP launcher | Observes approved managed targets; lifecycle/access contract | No direct host Incus or new REST admin bridge |
| Selected Box | Verified running target + scoped SSH identity + inside-Box agent-box (proposed) | Bounded File/Process/Exec actions | Box access must not grant parent Incus, sibling Box, or Provider admin |
| Run-local approval MCP | Run-bound signed approval key, upstream task IDs | Answer a specific coding-run permission prompt | Not a reusable human consent or generic grant |
| Host CLI/root-owned utilities | Local operator/admin with explicit privilege | Safe diagnostics and bootstrap, separately authorized | Never run app-controlled user code as root |

## Proposed operation classification

| Class | Examples | Policy proposal |
| --- | --- | --- |
| Metadata read | `list_containers`, `get_container`, `list_runners` | Allowed *only* for authenticated scope and exact actor/target; prevent unrelated inventory disclosure |
| Potentially sensitive read | `debug_container`, `tail_log`, process arguments, `security_findings` | Explicit target and bounded redaction; logs may contain secrets |
| Stateful write | `toggle_auto_sleep`, Compose service writes, start/stop, route changes | Typed authorization; relevant plan/confirm and re-observation |
| Unbounded execution | agent-box `shell_exec`, `process_start`, Platform `code_run` | Separate execution grant; file-root checks do not contain shell behavior; cost/timeout/output caps |
| Destructive/irreversible | `delete_container`, `restore_backup`, `set_ssh_keys`, `revoke_token` | Default deny until specifically approved; exact identity, irreversible-effect warning, one-time dispatch |
| Credentials, host/tenant admin | `get_secret`, KMS, backend upgrade, broad policy modification | Default deny/defer, never hidden in routine app UI |

An advertised tool or accepted `tools/call` packet is not independently proof of a successful authorized side effect. The upstream Platform uses per-tool JWT scopes; BoxWeave's gateway currently does **not** implement a complete per-tool end-user authorization layer. Any new "policy" claim must first close that gap or explicitly say the broad, trusted operator launcher remains.

## Target policy decision inputs

Proposed *typed* context: entry point (`agent_gate`, `app_service`), caller identity established at that entry (possibly `unbound`), selected app project, Provider, backend, immutable instance identity, tool source, upstream tool name, discovered schema digest/version, requested operation class, current approved policy revision, optional plan confirmation and caller scope.

Never accept those values solely from untrusted tool arguments. Server observations (and their provenance) establish the subject. Bind operation and target to an immutable identity; stale or absent evidence fails closed. Do not store credential values, SSH private keys, arbitrary shell outputs or full caller-supplied payloads in a general audit ledger.

## Compatibility migration — **decision required**

A suddenly enforced deny-all filter at the present direct Agent gateway could break existing enrolled trusted-operator integrations: ordinary upstream tools are currently forwarded. Conversely, retaining that relay while claiming a new blanket least-privilege policy is false.

Proposed path: (1) M0 read-only discovery/impact matrix, (2) explicit operator-approved tool policy and scoped launcher identity, (3) test **legacy operator** and **restricted pilot** connections separately, (4) stage policy diagnostics/compatibility reporting without a silent default privilege change, (5) enable reviewed enforcement only after acceptance. An approved stronger credential and/or independent gateway authorization boundary may be required to prevent policy bypass. No policy migration or credential re-enrollment is authorized by this document.

## Failure, approvals and auditable outcomes

- Preserve existing `unknown` state: a timed-out remote write may have succeeded. Query exact target state before any new authorized action; never auto-retry create/delete/restore/route/secret operations.
- Do not share approval tokens or claim upstream `code_answer` is permission to change BoxWeave Provider/storage/network outside that run.
- For read-only diagnostics, prefer allowlisted state/error categories. Gate/app-host evidence crossing boundaries needs an explicit authorized storage design (see D0–D2); do not expose raw launcher's stderr, JWT, SSH command or environment to Agent.
- Prefer an independent host CLI diagnostic route when Web/Agent gate fail. Host-owned inspections are separate from app-owned storage and must never load app code with root privileges.
- Unknown upstream tool names, unsafe schema drift, missing agent-box, missing rights, bad host pin and mismatched target identity are fail-closed, *subject to the operator's explicit migration decision for the currently open relay*.

**Backlinks:** [master](MCP_CAPABILITY_INTEGRATION_MASTER.md), [source audit](../verification/MCP_CAPABILITY_SOURCE_AUDIT.md), [Platform](MCP_PLATFORM_CAPABILITY_MATRIX.md), [agent-box](MCP_AGENT_BOX_CAPABILITY_MATRIX.md), [M1](mcp-integration/M1_ROUTING_AND_POLICY.md), [test plan](../verification/MCP_INTEGRATION_TEST_PLAN.md), [diagnostic proposal](DIAGNOSTICS_PROPOSAL.md).
