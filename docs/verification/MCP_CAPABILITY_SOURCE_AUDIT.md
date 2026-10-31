# Current BoxWeave functionality vs upstream MCP — static gap audit

**2026-10-31 main reconciliation:** This original inventory was compiled at BoxWeave v0.21.4; latest reviewed main is v0.22.0 (`aaa8362decd55110eadf8c5caa7ff1b3d1d80ac5`). D0–D2 diagnostics were implemented in main (SRC-108), with remaining live/coverage limits; optional D3 direct-gateway diagnostic lookup remains unimplemented. E0–E4 environment features remain unimplemented. All M0–M5 MCP capability reuse stages are proposal-only here (separately tracked implementation must cite exact source/CI evidence). Previously written statements that D0–D2 are unimplemented are superseded by this notice. Source inventory does not establish installed tool availability.


**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.


## Audit method / what "unused" means

Static source comparison of existing BoxWeave modules against upstream Containarium Platform tool registration, agent-box tool registration and approval-only MCP. "Not directly called by the BoxWeave application adapter" does **not** mean "not visible to Agent clients": the direct SSH gateway forwards ordinary Platform calls largely unchanged. Nor does source registration prove the pinned **installed** tool is advertised: Platform `tools/list` can filter by JWT scope, release/backend and permissions.

Observed source total at upstream pinned commit: **109 Platform tool definitions** in the registration files identified by [Platform matrix](../design/MCP_PLATFORM_CAPABILITY_MATRIX.md); **14 agent-box tools and 2 resources** from `internal/agentbox`. One internal permission-prompt server exposes `approve` only for eligible runs. Totals are *source inventory*, not deployed `tools/list` counts.

| Evidence | State | Limit |
| --- | --- | --- |
| Current Python source, upstream source | S: static inspected | No installed binary or authenticated session tested |
| Existing BoxWeave test/verification records | D: historical | Different commits, installations, versions and scopes |
| Live Containarium `tools/list`, Box `agent-box` probe | U: not executed | Cannot assert current support, number, scope or connectivity |
| Proposed mapping / adapters / new UI | Proposal only | No implementation, release, migration or product approval |

## Existing BoxWeave capabilities — source ownership

| Area | Current source and supported behavior | What is *not* established | Possible reuse |
| --- | --- | --- | --- |
| Client onboarding | `online.py`, `client_profiles.py`, `mcp_gate.py`: enrolled SSH identity, pinned server host key, managed SSH workload public key, native/WSL public client setup | Fine-grained tool permissions per enrolled Agent, current deployed acceptance of all combinations | No new MCP required; improve identity/scope metadata first |
| Direct Agent gateway | `mcp_gate.py`: stdio relay, `tools/list` extension, `create_container` key/nesting augmentation, `connect` target trust gate | Policy interception of **every** arbitrary upstream tool, durable cross-app event correlation | M1 policy gap; avoid suddenly advertising admin tools as safe |
| App MCP transport | `mcp.py`: bounded process, initialize, notification, paginated tool discovery, tools/call, text result validation | Generic resource read support, binary/image contents, server-sent sampling/roots, automatic retries or multi-Box sessions | M0 compatibility investigation; extend only as needed |
| Target and execution | `provider.py`: `get_container` identity; fixed `connect` identity/hello; create/inspect/start/stop/delete/resize | General box file/process/shell gateway or agent-box transport; adaptive test runner product | M3/M4 candidate; no generic Web shell in first slice |
| Lifecycle | `lifecycle.py`, `store.py`: exact managed identity, plan/confirm, one pending action, unknown result reconciliation | Auto-sleep state, actual instance snapshot/clone/restore support | M2 auto-sleep study; E0–E4 separate |
| Tenant/access | `access.py`, `access_provider.py`: native network capability/Guard/policy, scoped access and port route operations | General network policy admin surface or unrestricted `list_routes` | Keep typed app interface; no extra MCP for core access |
| Web | `web.py`, `service.py`, graphs/models/client setup | Dynamic 109-tool Web editor, fine-grained team RBAC, Agent orchestration scheduler | Only useful domain-specific screens after service contract |
| Diagnostics | `host_cli.py`, `deploy_check.py`, `selftest.py`, `jobs.py`, `diagnostics.py`: D0–D2 source-implemented in 0.22.0 with bounded reports, typed events and journal; D3 existing-gateway diagnostic read remains deferred | Full live report/selftest evidence and direct-gateway diagnostic read not established by this source audit | `debug_container`, `tail_log` may add evidence after independent scope checks |
| Nesting | `nesting.py`, `mcp_gate.py`, `NESTING.md` | Effective nesting without host/child verification; split helper reliability | Preserve current tested contract, do not clone/change hosts |
| Secrets/security | Upstream forwarded tools exist, BoxWeave own SSH/credentials protected | Product secrets/KMS admin contract, permission UI or safe sensitive return policy | Deny/defer by default pending explicit scope |

## Gap list and prioritized findings

| Gap ID | Finding | Direct effect | Priority | Relevant handoff |
| --- | --- | --- | --- | --- |
| GAP-MCP-01 | Direct-Agent proxy and typed app Provider have different authority and logging; no shared per-tool policy | Broad upstream authority cannot be marketed as fine-grained BoxWeave protection | P0 | M0/M1 |
| GAP-MCP-02 | Upstream source registrations are not measured installed capabilities | "All 109" promise unreliable; schema/name/permission drift | P0 | M0 |
| GAP-MCP-03 | No reviewed general agent-box adapter or per-Box target selector | Cannot promise file/log/process APIs over BoxWeave | P2 | M3/M4 |
| GAP-MCP-04 | BoxWeave MCP client only implements tools discovery/call and text replies | agent-box Resources and future message forms require protocol verification | P0 before M3 | M0/M3 |
| GAP-MCP-05 | No managed auto-sleep integration/state contract | Idle-stop semantics may conflict with lifecycle receipts/readiness | P1 candidate | M2 |
| GAP-MCP-06 | D0–D2 diagnostics now source-implemented; upstream `debug_container`/`tail_log` enrichment and direct Agent D3 lookup remain unverified | Avoid duplicate diagnostic engine; first reconcile running versions and safe disclosure | P1 candidate | M2, D0–D2 implementation |
| GAP-MCP-07 | Existing selftest is not a general Agent-chosen testing/approval product | Cannot conflate `code_answer` with approval for arbitrary new workflow | P3 | M5 |
| GAP-MCP-08 | Native snapshot/clone capabilities are unverified at the app MCP boundary | A database backup or an Incus CLI existence is not instance checkpoint | Separate E0–E4 | Existing environment design |
| GAP-MCP-09 | Platform and agent-box Compose operations overlap under different scopes | Duplicated controls can race or act on different subjects | P3 | M5 |
| GAP-MCP-10 | No reason to expose raw secret/admin tools | Secret disclosure, host upgrades and RBAC gaps | Defer/explicit opt-in | M1 security |

## Negative source findings

- **No evidence** in current BoxWeave `bwmini` of `agent-box` protocol registration/transport/verification. Fixed `connect` commands **are not** the same as in-box MCP tool support.
- **No** general application-side `create_backup`/snapshot/clone Provider adapter. Upstream `create_backup` is a tenant **database** backup, not an instance/storage snapshot.
- **No** automatic implementation of a comprehensive user-facing tool permission manager because the direct gate forwards most tool calls. Platform JWT scopes remain a separate upstream enforcement mechanism.
- **No** claim that source-only 109 definitions are discoverable on any installed Containarium version, or that agent-box is installed on all Box images.
- **No** separate general-purpose debugging MCP is warranted. The internal approval MCP is for signed tool approvals for particular coding runs, not a general authorization backend.

## Code and reference anchors

**BoxWeave pinned source:**
- [MCP gate](https://github.com/vanishs/boxweave/blob/a55926fcea55fdd3e2219900c787b90e49cb5e19/bwmini/mcp_gate.py)
- [MCP app client](https://github.com/vanishs/boxweave/blob/a55926fcea55fdd3e2219900c787b90e49cb5e19/bwmini/mcp.py)
- [Provider](https://github.com/vanishs/boxweave/blob/a55926fcea55fdd3e2219900c787b90e49cb5e19/bwmini/provider.py)
- [Access Provider](https://github.com/vanishs/boxweave/blob/a55926fcea55fdd3e2219900c787b90e49cb5e19/bwmini/access_provider.py)
- [Lifecycle](https://github.com/vanishs/boxweave/blob/a55926fcea55fdd3e2219900c787b90e49cb5e19/bwmini/lifecycle.py)
- [Web](https://github.com/vanishs/boxweave/blob/a55926fcea55fdd3e2219900c787b90e49cb5e19/bwmini/web.py)
- [Existing architecture](../design/ARCHITECTURE.md); [feature matrix](FEATURE_MATRIX.md).

**Upstream pinned source:** [tool registration](https://github.com/FootprintAI/Containarium/blob/f37358e631282ad90bd534796667af8d0a56b72c/internal/mcp/tools.go), [agent-box tool registry](https://github.com/FootprintAI/Containarium/blob/f37358e631282ad90bd534796667af8d0a56b72c/internal/agentbox/server.go), [approval-only MCP](https://github.com/FootprintAI/Containarium/blob/f37358e631282ad90bd534796667af8d0a56b72c/internal/agentbox/approval.go).

**Related plans (not implemented):** [diagnostics D0–D2](../design/DIAGNOSTICS_PROPOSAL.md), [environment E0–E4](../design/BOX_ENVIRONMENT_PROPOSAL.md). [Master and decision](../design/MCP_CAPABILITY_INTEGRATION_MASTER.md).
