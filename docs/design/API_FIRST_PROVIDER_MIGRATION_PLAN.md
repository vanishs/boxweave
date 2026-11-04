# API-first Provider migration — code inventory, feature parity, Agent work plan

> **Status: future implementation design, not approved or executed.** Parent: [API-first evolution master](API_FIRST_CONTROL_PLANE_EVOLUTION_MASTER.md). Companion: [trace/diagnostics](UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md), [candidate ADR](ADR/0010-api-first-control-plane-candidate.md).
> Source-only review: BoxWeave `main`/`VERSION 0.42.11` at `4f5ffe7fae586c79e6d75ed90560ee196e618ec3`; Containarium `main` at `e5deb167c973033d9a847abdcb2420e917095546`. The BoxWeave-managed installer still pins **Containarium 0.120.1**, and the runtime is not certified by these source observations.

> **2026-11-03 implementation checkpoint (not part of the historical estimates):** App-owned REST lifecycle basics were implemented in v0.42.12 and independently live-checked as part of the v0.43.0 stable promotion. The upstream MCP Agent gateway, specialist Tenant/Guard and nesting features, Box SSH work, and cross-entry mutation mediation are **not** completely migrated or globally accepted. The existing `v0.42.11` analysis below is retained for rationale; use [actual REST evidence](../verification/PROVIDER_REST_A1_04212_2026-11-03.md), [v0.43.0 acceptance](../verification/STABLE_PROMOTION_DC01_0430_2026-11-03.md), and the current [feature ledger](../verification/FEATURE_MATRIX.md) for delivery status. The Containarium installer pin remains `v0.120.1`.

## 1. Exact question and answer

How hard would it be for a coding Agent to replace **BoxWeave's own internal Containarium MCP calls** with the **Containarium REST API**?

- **Application/Web-side Provider only: medium (rough engineering risk 5/10)**. Provider call sites are relatively localized and existing app services already separate intent, receipts and worker logic. It is still not a search-and-replace: text-vs-JSON contracts, target incarnation, network Guard, partial operations, auth/token rotation and negative tests must be translated.
- **All upstream MCP dependencies removed while preserving ~99% of behavior: high (rough engineering risk 9/10)**. The current Agent gateway forwards much of an upstream catalogue, applies tool-specific rewrites and policy, and its `connect(exec)`, task/file and agent-box paths are not equivalent to a REST lifecycle endpoint. A new independently implemented Box Executor and Agent-facing tool router would be required.
- These are **qualitative risk ratings**, not measured effort, deadlines, precise LOC estimates or assurances of a particular Agent's completion rate.
- The recommended staged route is **REST on the app/provider side first**, while keeping upstream platform MCP and existing client setup alive until a testable replacement exists.

### 1.1 Target channel map — *not* 100% REST

Containarium's published implementation has parallel `containarium` CLI and platform `mcp-server` clients; the REST gateway lives in **`containariumd`**. The platform MCP mostly calls REST but also performs **client-local** credential/configuration work, direct SSH (`connect(exec)`), local Git/SSH transfer (`push/sync`), and SSH-to-`agent-box` coding sessions (`code_*`). Its README's planned Q4 2026 OSS v1.0 API milestone is **not** evidence those SSH paths will be retired.

| Existing user-visible operation | Intended BoxWeave internal path | Migration/acceptance condition |
| --- | --- | --- |
| List/get/create/start/stop/delete/resize | `Operation Core → Provider REST → containariumd` | Current app-owned subset exists; recheck Agent gateway, identity, tenancy, negative results and postconditions |
| Network tenants, Guard, routing, metrics, auto-sleep, backup/snapshot | Same typed REST adapter **where supported** | Check installed daemon proto/HTTP route, scopes, Guard or storage eligibility, and real-Box effect; never infer from upstream `main` |
| Box command, shell session, bounded file read/write/patch | `Operation Core → exact-Box SSH Executor → Box` | Server-owned identities, host-key pin, incarnation, safe workspace, output/timeouts, uncertainty; no required in-Box helper |
| `push` / `sync` | Explicit transfer adapter over SSH/local source operations | Decide source locality/permissions, atomicity, size caps, workspace and user authorization; not a REST CRUD alias |
| `code_run` / `code_attach` / `code_status` | Optional SSH → `agent-box` MCP, or independently specified equivalent | Capability probe and install policy; offset/reconnect/cancel tests; never assume `agent-box` exists |
| Client `sync_ssh_config`, local keys/certificates, encrypted restore identity | Explicit client/service-local effect **plus** API | Separate secret/path privileges, operator consent and audit; REST success does not verify a usable local file |
| GitHub Actions runner provisioning | Orchestrated API + SSH + GitHub API | Per-stage receipts/authority, remote failure and partial-state reconciliation |

**Boundary invariant:** The **inbound** Agent-facing BoxWeave MCP (SSH-stdio and HTTP/OAuth) remains a supported product entry; migration changes only **outbound** Provider calls. The default Box executor must not require installing `agent-box` inside a clean Box. The operator recovery CLI stays independently available without becoming a host-shell escape for Agents. All mutating channels require the same operation claim and UNKNOWN handling before any global coordination guarantee.

## 2. Current code owners and required changes

| Source path | v0.42.11 baseline role (historical) | Future disposition | Relative effort |
| --- | --- | --- | --- |
| `bwmini/provider.py` | Platform MCP session; get/list/create/start/stop/delete/resize; identity verification with `connect(exec)` | Typed `ContainariumAPI`; preserve lifecycle/identity interface while moving `hello`/identity exec to Box Executor | Medium |
| `bwmini/access_provider.py` | `network_tenant_capabilities`, Guard, tenant policy and passthrough route operations over tool names | Map to authenticated REST network/tenant endpoints; retain revisions, no partial two-Tenant writes, Guard verification | Medium–high |
| `bwmini/environment_observe.py` | E0/E1 capability and fingerprint probe via MCP catalogue and `get_container` | Separate declared REST contracts, runtime support and actual host storage eligibility | Medium |
| `bwmini/auto_sleep.py`, `box_debug.py` | MCP `toggle_auto_sleep` and `debug_container` | Typed REST route and structured summary, fail on missing/changed fields | Low–medium |
| `bwmini/capabilities.py`, `mcp_compat.py` | Platform MCP catalogue and minimum schema drift report | Provider API capability probes and contract fingerprints. Preserve *separate* outbound BoxWeave MCP transport checks | Medium |
| `bwmini/mcp.py` | Bounded stdio JSON-RPC MCP client | Keep for optional agent-box/compatibility adapters; not needed by standard REST Provider | Low conditional |
| `bwmini/mcp_gate.py` | 817-line upstream tool relay plus managed key, trust, nesting, D3/D4, restore guard and tracked operations | Eventually replace with native BoxWeave MCP tools calling Operation Core + Provider API + Executor; feature and policy parity first | **Very high** |
| `bwmini/remote_mcp.py` | Authenticated Agent-facing HTTP MCP, per-request gateway spawn | Preserve HTTP/OAuth behavior; route to new internal tool server only after tested handshake/cancel/error parity | High |
| `bwmini/agent_box.py` | Optional configured per-Box *separate* MCP for read/write/process/resource actions | Retain as a distinct Box Executor strategy until exact-equivalence proof; do not mistake for platform MCP | Medium–high |
| `bwmini/box_reads.py`, `box_patch.py`, `gateway_tasks.py` | Fixed commands and tracked metadata around platform `connect(exec)` | Execute through shared Box Executor with bounded results, CAS and crash-safe receipts | High |
| `bwmini/operation_coordination.py`, `coordination_bridge.py`, `coordination_runtime.py`, `gateway_protection.py` | Some claims/leases/mutator protections are optional and path-specific | Canonical shared pre-dispatch boundary for Web, SSH and HTTP Agent paths; no silent bypass | **High** |
| `bwmini/lifecycle.py`, `access.py`, `box_actions.py`, `service.py` | Durable intent/workers/revisions and business receipts | Preserve semantics while changing adapter; improve convergence/unknown errors, not rewrite automatically | Medium |
| `bwmini/bootstrap.py`, `online.py`, `host_cli.py`, `distribution.py` | Pinned MCP launcher, credentials, Containarium binaries, install/repair | New API service/principal checks, token rotation and cleanup; independent host rescue path, no provider upgrade | Medium–high |
| `bwmini/gate_diagnostics.py`, `diagnostics.py` | D3 gateway journal, D1/D2 trace/receipts | One correlation model and classified API/SSH/MCP spans; see [diagnostic contract](UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md) | High if bundled with full gateway refactor |

This inventory is **source reviewed**, not a complete runtime reachability assertion. Re-scan all import/call sites and generated install helper manifests immediately before any implementation.

## 3. API operation mapping

### 3.1 Straightforward typed management

| BoxWeave operation | Current platform MCP pattern | Candidate Containarium REST backing | Non-obvious acceptance check |
| --- | --- | --- | --- |
| Inventory, inspect, identity | `list_containers`, `get_container` | `ContainerService` list/get grpc-gateway routes | Pagination, returned logical/name/creation identity, state enum, empty inventory |
| Create / start / stop / delete / resize | Corresponding platform MCP tools | `ContainerService` typed routes | No implicit retry, readiness vs acknowledged, SSH-key setup, nesting constraints, CPU/memory/disk shape |
| Tenant capabilities & Guard | `network_tenant_capabilities`, `network_guard_status` | `GET /v1/network-tenants/capabilities`, `GET /v1/network/guard` | Positive/negative capability and enforcing Guard; do not confuse metadata with isolation |
| Tenant policy / ingress | `get_network_policy`, `patch_tenant_access` | Inspect actual network/tenant Protobuf REST routes first | Revision/atomicity, no silent partial cross-Tenant grant, `NotFound` normalization |
| TCP/UDP passthrough | `list/add/delete_passthrough_route` | `GET/POST /v1/network/passthrough`, `DELETE /v1/network/passthrough/{external_port}` | Owner/port/target and protocol, external reachability distinct from configured route |
| Auto-sleep | `toggle_auto_sleep` | Container auto-sleep route in deployed `container.proto` | Last-start/idle/new-born behavior, actual waking and HTTP route availability |
| Debug/metrics | `debug_container`, `get_metrics` | Typed `ContainerService` debug/metrics RPC routes | Source and observability classification, no raw journal/secret output |
| Whole-Box backup create | MCP `create_backup` cannot presently represent `scope=BOX` | `POST /v1/backups` with `scope=BACKUP_SCOPE_BOX` | Encrypted artifact manifest, identity custody, actual installed API support |
| Local snapshot list/create/delete/rollback | Not reliably exposed by installed platform MCP catalogue | `ContainerService` snapshot REST routes | ZFS-capable storage, key custody, destructive rollback and explicit stopped target |
| Box backup restore | MCP `restore_box_backup` uses host-local `age_identity_file` | `POST /v1/backups/{id}/restore-box` | Key material read only by trusted service; stopped target, separate copy identity and unknown outcomes |

**Important source distinctions:**
- Upstream `CLAUDE.md` says CLI-first at the *development* level, and Protobuf-first for API contracts. A new REST endpoint does **not** automatically register a corresponding MCP tool.
- The upstream `backup.proto` exposes `CreateBackupRequest.scope` and `POST /v1/backups`; whole-box create is **REST/CLI available in source, MCP parity gap**, not intrinsically CLI-only.
- `container-snapshots.md` identifies shipped list/create/delete/rollback API and a **proposed** clone method. Do not advertise snapshot-source clone merely because its description is documented.
- Restoring a whole Box backup as a new independent Box is **not** automatically a safe snapshot clone; rekey, SSH host identity, service collision, network/tenant placement and data/secret isolation require a separate acceptance gate.
- APIs may be defined in current source but missing, differently configured or storage-blocked in a deployed/pinned runtime. Record installed method capability explicitly.

Primary upstream references:
- [Container Protobuf](https://github.com/FootprintAI/Containarium/blob/main/proto/containarium/v1/container.proto)
- [Network Protobuf](https://github.com/FootprintAI/Containarium/blob/main/proto/containarium/v1/network.proto)
- [Backup Protobuf](https://github.com/FootprintAI/Containarium/blob/main/proto/containarium/v1/backup.proto)
- [Snapshot design](https://github.com/FootprintAI/Containarium/blob/main/docs/architecture/container-snapshots.md)
- [Upstream code-run MCP](https://github.com/FootprintAI/Containarium/blob/main/internal/mcp/code_tools.go)
- [Upstream push/sync MCP](https://github.com/FootprintAI/Containarium/blob/main/internal/mcp/push.go)

### 3.2 Nontrivial exceptions — seven design workstreams

1. **Box command execution (`connect(exec)`) and trust.** The upstream platform MCP resolves a target and uses SSH, not generic lifecycle REST command execution. Replace by a tightly controlled SSH executor; observe Box identity first, never accept raw caller hostname/key/user, bind SSH host key, maintain managed key/certificate and capped stdout/stderr/timeouts. HTTP errors are no substitute for SSH error classification.
2. **Filesystem, `push`, `sync` and code transfer.** Upstream `push`/`sync` invoke transfer functions, not simple REST uploads; existing BoxWeave file read/patch use `connect(exec)` and optional `agent_box.py`. Design coherent file bytes, paths/roots/symlinks, hashes/CAS, large-file streams, output/size caps, atomic replacement, and clean-Box requirements. SFTP is one option, not a mandatory dependency or automatically authorized endpoint.
3. **Detached coding runs.** Upstream `code_run`, `code_attach`, `code_status` and related actions connect to `agent-box` over SSH and handle offset/reattach semantics. Track run vs caller request, process survival vs reply recovery, log-offset bounds, cancellation uncertainty, optional in-Box helper presence and separate approval trust; do not imply Agent approval-MCP is a universal approval API.
4. **Cross-transport authority and SSH identities.** The current gateway injects a server-owned public key on create, checks expected target before connect, and uses a separate app principal from direct-Agent authority. Splitting API and SSH must retain these distinctions, credential rotation and least privilege; do not merge Web cookies, model keys, outbound JWT and remote MCP OAuth tokens.
5. **Gateway-transparently-forwarded upstream MCP catalogue.** Current `mcp_gate.py` forwards upstream tool list and arbitrary permitted tools with special changes for create/resize/connect/restore/nesting. Native API-only tools must be explicitly registered and authorization audited. A temporary **restricted upstream MCP compatibility lane** may preserve untouched advanced tools, but **cannot** bypass common mutation claims or be described as API-only. Freeze/record every tool's disposition rather than silently dropping it.
6. **Encrypted backup restore.** The host-local `age_identity_file` is read on the MCP/client host; upstream client passes its *contents* as `age_identity` to the authenticated daemon API. A redesign should have operator-controlled key custody, exact backup/target allowlist, no client-selected secret path, redacted request logging, one-shot confirmation and honest partial/unknown receipts. Remote-key-reference design requires upstream work; do not claim nonexistent native support.
7. **Runtime capabilities, storage and API drift.** Replace MCP `tools/list` probing with explicitly allowlisted REST read/capability checks derived from protobuf and runtime evidence. Distinguish `defined`, `endpoint_available`, `authorized`, `storage_supported`, `operation_tested`. Older daemon version, missing Guard or non-ZFS backend must show `unsupported`, not a fabricated feature.

### 3.3 Unresolved design conflicts — ask owner, do not silently decide

1. **Accepted transport ADR vs long-term REST preference.** [ADR-003](ADR/0003-transport-and-trust.md) still records the accepted upstream SSH/MCP transport choice; [ADR-010](ADR/0010-api-first-control-plane-candidate.md) is only a candidate. The v0.42.12 app REST slice is limited implementation, **not** formal supersession. A full gateway transition requires a separately reviewed ADR decision and migration/rollback evidence.
2. **Raw forwarded upstream MCP writes vs common mutation coordination.** `mcp_gate.py` can forward permitted upstream writes even when app lifecycle uses REST; a shared `Operation Core` would be bypassed. Owner must choose between **(A)** mediation/allowlisting of every mutating forwarded tool and **(B)** refusing unmediated forwarded writes *in protected mode* until equivalent mediation is available. Tradeoff: (A) broader functionality/higher implementation cost; (B) temporarily reduces advanced Agent capabilities. Neither option is preselected here; do not claim full protection meanwhile.
3. **No mandatory guest agent vs upstream `code_*` dependence on `agent-box`.** The existing R-01 clean-Box rule takes precedence. Treat upstream `code_*` as optional and `unsupported/not configured` when its helper is absent; do **not** silently install a guest service to preserve tool parity. If code-run parity cannot be reached through approved SSH or optional helper, record a feature gap rather than reclassifying it as REST.
4. **Upstream latest code vs pinned deployed API.** `main` route existence and a published v1.0 roadmap do not prove `v0.120.1` supports, authorizes or can safely execute the feature. No automatic Provider upgrade; block unsupported operations and record a future upgrade gate.

This section records *conflicts and alternatives*, not a decision or new implementation authorization.

## 4. Agent-facing compatibility is a separate project

Preserve two different contracts:
- **Outbound internal Provider:** BoxWeave → Containarium REST (the proposed migration).
- **Inbound public controller:** user Agent → BoxWeave MCP, with SSH stdio or authenticated Streamable HTTP, discovery, pagination, resources, OAuth/login policies and observed errors.

Switching the first **does not imply removing the second**. A complete gateway rewrite must test:
- Registered tool names and schemas; agent-client discovery and access control; pagination and `resources/list/read`.
- Managed public key injection for newly created Boxes; exact target override denial on connect and file operations.
- Existing `boxweave_info`, `boxweave_diagnose`, `boxweave_project_diagnose`, Box reads/patches, tracked task lookup/submit and leases, `boxweave_nesting`.
- Root-owned policy and restore guard, unknown operation outcomes, protected-mode mutator blocking and single shared resource-claim boundary.
- HTTP/OAuth/stdio parity and transport-auth errors; no accidental exposure of private token/key content.
- Conditional features from upstream agent-box and forwarded platform catalogue (including code, compose, snapshots, collaboration, metrics, secrets and specialized upstream tools where actually exposed).

**Never silently offer all raw REST endpoints as Agent tools**. Tool registration is an authorization decision; read-only hints or upstream schema annotations alone are not sufficient.

## 5. Proposed implementation phases (small, independently accepted)

| Phase | Goal | Done when | Risk |
| --- | --- | --- | --- |
| **A0 — Audit only** | Inventory app calls + every gateway catalogue capability; verify deployed API/token/host ownership and Protobuf routes | Explicit per-feature row `MCP`/`REST`/`SSH`/`local-only`/`unsupported`, confidence and test ownership | Low; no mutation |
| **A1 — Read-only REST proof** | Implement bounded API read client and a few Get/List/metrics/capability probes behind an opt-in adapter | Same exact Box identity and state, direct 401/403/404/timeout classification, empty-list, pagination, no existing-host changes | Medium |
| **A2 — Application management REST** | Wire lifecycle, Tenant/Guard, routes, auto-sleep through existing service plans/receipts | Fixture + isolated live positive/negative/UNKNOWN, unchanged real authorization and no duplicated writes | Medium–high |
| **A3 — Box Executor** | Separate SSH/session/file/exec, optional existing agent-box reuse; keep fallback MCP for unmigrated routes | Known identity and trust; stdout/exit7/timeout/unknown; CAS patch/unsafe path/large file, no host-shell escape | High |
| **A4 — Unified operation gate + trace** | Merge Web and Agent dispatch, shared claims, receipts and D1–D4 metadata; preserve transport auth | All mutating entry points intercepted; crash/recovery and no replay, correlated spans, no secret capture | Very high |
| **A5 — Agent MCP native tool router** | Replace forwarding for proven typed tools; retain explicit fallback for unmapped/advanced tools | Inbound discovery, HTTP/SSH parity, capability denial and client-visible accepted tests | Very high |
| **A6 — Full parity/installed release gates** | Test pin matrix, upgrades, failure repair, auth, real Boxes, route/backup ZFS controls in disposable lab | Each feature row has evidence; no unverified high-risk claim; rollback/promotion independently authorized | Very high |

**Sequencing constraint:** A1 and A0 can proceed without any production host upgrade. A2 can ship independently of A5 **if app and gateway write boundaries remain honest**; it must not be marketed as globally coordinated while untracked Agent writes exist. A5 must not precede A3/A4 if its tool routing would bypass required trust/claims.

**Checkpoint against this phase map:** v0.43.0 provides a limited **application-owned** A1/A2 REST foundation and diagnostic evidence; it does **not** complete A0 catalogue parity, all A2 specialist paths, A3 SSH Executor, A4 cross-entry mediation, A5 native Agent gateway or A6 full installed parity. Work in that order as separately reviewed slices; do not couple gateway rewrite, OAuth changes and Provider host upgrades in one release. Special attention goes first to the already existing REST/Web-versus-forwarded-MCP mutation bypass and correct D1–D4 span coverage.

## 6. Concrete verification checklist for later Agent execution

Future testing must include at least:

1. Fresh instance list/get/identity, empty inventory, stopped/running/not-ready distinctions, expired credentials, 401 vs 403, valid/invalid token rotation, malformed JSON and response body limit.
2. Box create with independently assigned SSH key and optional nesting; verify exact instance name/creation stamp and no second create after lost response. Reject name reuse and unauthorized parameters.
3. Start/stop/delete/resize and wait/observe; never treat HTTP acknowledgement as readiness. Verify real instance state and resource limits, including partial failures.
4. Tenant create/Guard capability, revision conflicts and atomic cross-Tenant grant/revoke, source/destination ownership and degraded Guard; no unverified network enforcement claims.
5. TCP/UDP route config vs actual packet reachability in isolated lab, IP reuse and foreign route refusal; no accidental public exposure.
6. SSH user/hostname/host pin, wrong backend and changed instance, missing `containarium-shell`, missing `agent-box`, expired SSH cert, split host trust.
7. File read/list/search/patch, traversal/symlink/pseudo-files, binary/large files, concurrent editing, expected hash mismatch; no guest service/package installed unexpectedly.
8. Task request-key replay, crash before/after dispatch, lost HTTP/SSH reply, unknown retained, same Box protected conflict and independent Box parallelism.
9. Whole-box backup creation via REST and verification, restore with exact key custody into stopped new Box, wrong key/target and no secret in traces; distinguish local ZFS snapshot rollback and clone-like import.
10. Agent-facing MCP discovery/resources and 2+ real client auth paths (HTTP, SSH), revoked OAuth, tool allowlist and protected-mode denial, per-tool feature-parity diff.
11. D3 trace from inbound action through Provider API/SSH and observation, D4 fixed bounded read by exact server approval, private report and redaction.
12. Fresh install, installer pinned/provider external mode, daemon service restart, token rotation, app rollback and unchanged stable published release until explicit promotion.

Evidence rubric: **S = source**, **T = fixture/CI**, **L = live task-owned Box/host/client**, **U = unverified**. A complete source test cannot substitute for real installed authorization, timing and network behavior. Block tests requiring unavailable ZFS or a different provider version without substituting unsafe raw Incus/shell shortcuts.

## 7. 99% parity gate and migration acceptance model

Maintain a machine-readable feature ledger during A0/A6 with at minimum:

```yaml
feature_id: F-EXAMPLE
surface: web|agent_mcp|host_cli|upstream_mcp_relay|optional_agent_box
baseline_version: immutable-revision
baseline_observed: source|fixture|live|unverified
behavior: explicit-user-outcome
api_or_executor_mapping: reviewed-adapter-name
authority_boundary: named-owner
proposed_state: preserved|equivalent|retired|unsupported|unverified
required_negative_tests: [identity_drift, timeout_after_send]
actual_evidence: [test-or-report-ref]
```

Do not compute 99% across *unverified* features by treating them as passes; separately disclose catalogue breadth and actually exercised workflows. Set a special **zero-loss gate** for security and recovery invariants. A release blocked on an unsupported backend remains blocked rather than calling the hypothetical capability equivalent.

## 8. Reversibility and deployment

- No direct flip of the existing `v0.120.1` installer pin. Test the documented API on the **actual installed release**; if a needed method is unavailable, ask for a separately reviewed lab upgrade and tag exact daemon/build/feature capabilities.
- Use a branch and PR per phase. New REST reads must be explicitly opt-in until A1 acceptance. Changing one Provider adapter cannot silently change Agent gateway behavior.
- Do not mutate an existing Containarium or Incus installation as part of BoxWeave transport refactoring.
- Preserve independent non-MCP host CLI diagnosis and application backup/restore. For a full replacement, demonstrate new install / rollback / CLI recovery on disposable host before release promotion.
- Preserve prior D3/D4 security/retention defaults unless a separately reviewed ADR accepts a change.

## 9. Future decision checkpoint

The architecture direction should be promoted from *candidate* only after A0/A1 provide: deployed API and auth evidence, full gateway feature inventory, a real-Box and failure test plan, concrete testable containment of guest SSH, and explicit owner approval of any behavior intentionally retired. Until then, [ADR-003](ADR/0003-transport-and-trust.md) and the current implementation remain authoritative.
