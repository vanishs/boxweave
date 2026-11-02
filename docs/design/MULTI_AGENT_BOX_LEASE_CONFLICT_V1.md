# Multi-Agent Box Lease and Operation Conflict Protection v1

**Global plan:** [BoxWeave feature roadmap](FEATURE_ROADMAP_MASTER.md) — R-04 multi-Agent resource coordination. This link tracks the overall selection/status; this document retains detailed scope and test obligations.

**Status:** DESIGN ONLY / DEVELOPMENT HANDOFF — NOT IMPLEMENTED, NOT TESTED, NOT DEPLOYED.
**Owner decision:** Multiple independent projects and Agents may use the same existing BoxWeave controller. Prefer **multiple Boxes** for concurrent development; no Git worktree management and no Agent runner/delegation feature. On an actual resource conflict, **reject immediately (no automatic queue)**. Keep the feature small because simultaneous conflicting Agents are uncommon.
**Source inspection:** BoxWeave `main` v0.27.2 at `3fd31e19844ae88ff0a7f73218261584d5729415`, 2026-11-01. Runtime behavior and all proposed changes below require later implementation and exact-version acceptance.
**Guiding rule:** An explicit Box lease is optional. Normal single-Agent operations should not require manual lease setup or heartbeat. BoxWeave's existing Box lifecycle, Provider/Incus boundary, app Service, direct SSH/HTTP MCP, authentication and Web UX must remain recognizable. No software is installed inside a Box for this feature.

See [current architecture](ARCHITECTURE.md), [intent/reconciliation ADR](ADR/0002-intent-and-reconciliation.md), [MCP task reliability R1](MCP_TASK_RELIABILITY_R1.md), [Box operations v1](BOX_OPERATION_INTERFACE_V1.md), [MCP security boundaries](MCP_INTEGRATION_SECURITY.md), [Agent test workflow](../process/AGENT_TEST_WORKFLOW.md) and [verification ledger](../verification/REMEDIATION_TEST_QUEUE.md).

## 1. What problem this solves

Examples:

- Project A's Codex works in `box-a`; Project B's Claude works in `box-b`. Independent Box operations should proceed in parallel.
- Project A and Project C both want to modify `box-a`. Project A explicitly reserves `box-a` for a development session; Project C must receive a precise conflict **before dispatch**.
- An Agent changes a shared Tenant network or host port while another Agent wants to change the same shared resource. A Box-only lock cannot protect that shared resource.
- Agent A's process crashes without releasing its Box reservation. After the reservation expires, Box B may not remain blocked forever by an abandoned Agent.
- If Agent A had an operation whose remote outcome is uncertain or which may still be active, mere reservation expiry must **not** authorize a conflicting write.

**This is resource-operation coordination, not a per-Agent security or authorization system.** It prevents accidental overlapping changes made through integrated BoxWeave surfaces; it cannot stop an external administrator from directly using Incus, the Provider API or guest SSH outside BoxWeave.

## 2. Existing code and precise gap

| Current component | Existing behavior | Gap for this feature |
| --- | --- | --- |
| `bwmini/lifecycle.py` | Durable environment plans/receipts and a same-logical-name pending/unknown-operation guard | Protects its own application operations, not every direct gateway `create_container`, `delete_container`, `connect(exec)` or other calls. |
| `bwmini/access.py` | `access_claims(resource PRIMARY KEY, operation)` rejects certain duplicate resource writes. It retains claim/reconciliation semantics around uncertain outcomes | Claims are app-side and operation-type-specific; not a cross-SSH/HTTP/Web Box lease. Do not remove or weaken them. |
| `bwmini/box_actions.py` | One-use plans, durable receipts, unknown on crash, no replay | No unified Box occupation across independent Agent projects and direct gateway execution. |
| `bwmini/service.py` / `bwmini/store.py` | App SQLite, worker lock and receipt history | Worker serialization is not resource-specific concurrency safety for calls outside that worker. |
| `bwmini/mcp_gate.py` | Forced-command direct MCP gateway; exact-target trust for `connect`; optional host-wide tool allowlist; direct forwarding to Provider | Most forwarded mutators are not claimed in a controller-wide resource registry. Merely adding an MCP lease tool would leave bypass paths. |
| `bwmini/remote_mcp.py` | HTTP MCP invokes the same gateway in a separate short-lived process per request | In-process mutexes cannot coordinate different HTTP requests, SSH gateway processes or Web worker. |
| D3/D4 diagnostics | Classified bounded evidence and per-Box read diagnostics policy | Not a lock, operation ownership guarantee or complete guest process inventory. |

A narrow, shared and **durable controller-side coordination boundary** is the only new primitive needed. Reuse existing claims/receipts where possible; do not build a task scheduler, agent marketplace, worktree manager or full distributed lock service.

## 3. Minimal approved behavior

### 3.1 Rules

1. **Default: no explicit lease required.** An unreserved Box is operated exactly as today, except an integrated mutating operation temporarily obtains an automatic **operation claim** for its resource(s).
2. **Optional Box lease:** A project/Agent can acquire a Box reservation spanning several commands. It is an exclusive *write-coordination* lease, not exclusive read access or an authorization grant.
3. **Read-only known-safe tools:** Multiple reads may proceed with a Box reserved, unless a particular read has separate safety constraints. Do not classify arbitrary shell commands as read-only.
4. **Mutations with competing reservations/claims:** Immediately return a typed `RESOURCE_BUSY` / `LEASE_REQUIRED` conflict. **No wait, no FIFO queue, no automatic retry.**
5. **Lease expiry:** The holder must renew explicitly. If expired and no unresolved operation claim remains, the lease is lazily removed/invalidated and a different requester may acquire the Box **without administrator approval**.
6. **Uncertain operation:** If a remote operation was dispatched or may have been dispatched and its completion cannot be established, keep a separate `RECOVERY_REQUIRED` operation claim, independent of the lease timeout. New conflicting modifications are denied. Allow safe observation and explicit operator resolution.
7. **No hard assumption about Agent identity:** User-facing labels (`project`, `agent`) are diagnostic labels; they do not create authorization or distinguish security principals when multiple clients share trusted-operator credentials.
8. **No automatic replay, no forced takeover:** A timeout never proves a remote command stopped. Explicit recovery action may clear an uncertain coordination claim only after reviewing the original operation/target; clearing the claim is **not** a rollback or process termination.

### 3.2 Resource granularity, kept intentionally small

Use typed **canonical resource keys**, scoped by existing backend/project/tenant identity, to avoid a single global host mutex:

| Resource kind | Key concept | Example conflicting operations |
| --- | --- | --- |
| Exact Box | Logical name **and immutable instance incarnation** plus management scope | `connect(exec)`, Box resize/start/stop/delete, guest file write or process mutation |
| Box name before creation | Management scope + requested logical name | Two creates of the same name; concurrent create and delete/claim of that logical identity |
| Shared Tenant/network | Authorized Tenant and network-policy identity | Competing route, membership, Tenant access or network-policy writes |
| Published port | Host/network scope + protocol + external port | Conflicting publication/reassignment/removal |
| Other host-shared change | A small explicitly enumerated domain key if needed | Host-wide policy or trust changes supported through BoxWeave |

An operation may need more than one key: for example, a Box's host port publish can affect both a port and its target Box. Acquire **all necessary keys atomically**, in canonical order, or acquire none. Do not serialize unrelated Boxes or unrelated ports unnecessarily.

**Risk of classification:** A tool that changes shared state but cannot be classified to a safe key must not silently bypass the coordinator. Mark it `UNCLASSIFIED_MUTATION` / refuse for this protected coverage until mapped to the correct scope. Do not block safe reads just because they query a related Box.

An explicit Box reservation should not automatically reserve all Tenants or every host port. The operation itself must claim the actual affected shared resources.

### 3.3 Exec is conservatively a write

Existing `connect(exec)` executes an opaque shell string, and `shell_exec`, `process_start`, etc. can modify the guest even when an Agent says the intention is “read-only.” Therefore treat all generic guest execution as **potential mutation** requiring a Box claim and honoring any explicit lease.

A completed foreground shell response does not prove the guest has no detached child processes. For supported durable process-start/run handles, keep the operation claim until actual completion is confirmed; for opaque detached work whose lifetime cannot be observed, document the safety limitation and prefer an explicit Box lease across the overall development session. Do **not** advertise this as complete OS process isolation.

## 4. Lease lifecycle and crash handling

### 4.1 Defaults (proposed, easy to change after real validation)

- **Duration:** 10 minutes per lease.
- **Renewal guidance:** about every 2 minutes while actively using the Box, with each valid renewal extending to 10 minutes from server time.
- **No background server heartbeat:** the owning Agent/client explicitly calls renew. No lease-renewal daemon inside the Box.
- **Early release:** explicit release when work is finished; release is idempotent for an already released lease but refuses an unrelated/expired/replaced token.
- **After 10 minutes without renewal:** automatically eligible for re-acquisition **only if no outstanding operation claim needs reconciliation**.
- **No timer needed:** evaluate expiration in the same short database transaction as acquisition, renewal, claim or status query. Automatic expiry does not require a scheduler.

The server uses its own clock and a bounded expiry; never trust a caller-supplied expiration time. Keep the duration conservative and adjustable only through reviewed operator config (not arbitrary client overrides).

### 4.2 Minimal state transitions

```text
Unreserved --acquire--> LEASE_ACTIVE --release--> Unreserved
      ^                       |
      |                       +--time passes/no renew--> LEASE_EXPIRED
      |                                                 |
      +------------------ no uncertain claims ----------+
                                                        |
                                          if uncertain active claim
                                                        v
                                                RECOVERY_REQUIRED
                                                (no conflicting writes)
                                                        |
                                      read/reconcile or explicit reviewed
                                      operator override, then release
```

**Important:** `RECOVERY_REQUIRED` is an **operation-claim safety state**, not an immortal lease. The expired lease does not renew itself or keep an old Agent's privilege alive.

- After a crash **before dispatch was possible**, remove the preflight operation claim once the durable record proves no effect could have been sent.
- After durable **dispatch intent**, loss of the gateway/worker/request yields `UNKNOWN`; retain the claim. A delayed Provider response cannot safely be reinterpreted as a definite failure.
- After observed command exit/definite terminal operation success or failure, release the **operation** claim. The explicit project Box lease stays active until release/expiry.
- A stale Agent resuming after another holder acquired the Box must be rejected. Use a fresh generation/epoch and unguessable token; never accept an expired lease token as a current owner.

### 4.3 Fencing and release correctness

Acquire/renew/release and conflicting operation-claim checks are **atomic** and performed in a short SQLite transaction. The record binds to exact Box incarnation and current lease generation. The gateway must check the current lease generation just before its protected dispatch; after expiry/reacquisition, a stale caller cannot submit another protected write.

**No false fencing claim:** A controller-side generation check cannot cancel a command that was already dispatched to the guest. Its unresolved operation claim is what blocks a subsequent conflicting command. Without a Provider-level fencing primitive, the design does not promise complete prevention of late externally dispatched writes; never release a claim merely because an Agent disconnected.

## 5. API — keep the surface tiny

Candidate **MCP** operations (names subject to implementation review):

| Tool | Input | Purpose |
| --- | --- | --- |
| `boxweave_box_lease` | `action=acquire|renew|release|status`, exact `box`, `lease_token` when needed, optional non-authorizing `project_label` | Single small typed tool rather than four separate tools. |
| Existing mutating tools | Existing original args plus optional gateway-owned `lease_token` field, **consumed/stripped by gateway** before forwarding when needed | No new generic execution tool; maintain Provider schema compatibility. Existing calls with no lease remain valid if no conflicting lease exists. |

If schema extension cannot be performed safely for an upstream tool, use a verified equivalent scoped envelope instead. Do not call Provider with an unknown `lease_token` argument. The optional global tool allowlist must account for the new lease tool and remain enforced before forwarding.

Example response for conflicting call:

```json
{
  "state": "conflict",
  "code": "RESOURCE_BUSY",
  "resource": "box:dev-a",
  "blocking_kind": "lease",
  "retry": "manual_after_recheck"
}
```

Do **not** leak the other Agent's command, project contents, secret lease token, session credentials or private arguments. A bounded non-secret owner label may appear only in operator-authorized diagnostics. The error should distinguish `RESOURCE_BUSY`, `LEASE_EXPIRED`, `LEASE_TOKEN_INVALID`, `TARGET_CHANGED`, `RECOVERY_REQUIRED` and `COORDINATION_UNAVAILABLE`.

### 5.1 Direct MCP and Web/app parity

A hard product gate: **all protected entry paths must consult the same operation coordinator before dispatch**, including:

- Direct SSH/stdio `mcp_gate.py` and authenticated HTTP MCP (which invokes the same gateway in separate processes).
- Web/app lifecycle actions, `box_actions`, Access/Tenant/port actions and any other selected app-side mutator.
- Nested/host-mutating BoxWeave helper operations, after their actual resource impact is classified.
- Read-only tools should remain available without a lease.

A raw forwarded Provider write cannot bypass a reservation merely because the new `boxweave_box_lease` tool is optional.

The existing app SQLite and gateway D3 journal have **separate privilege and trust boundaries**. The simplest acceptable implementation is a **minimal controller-owned coordination SQLite store** with restricted, explicitly reviewed access for only existing gateway/app principals (e.g. owned controlled group/ACL and no general file access), or an equivalently safe existing host mediation boundary. Do not grant gateway write access to the entire app DB or expose app credentials. Do not add a new long-running lock service merely to implement leases. If safe shared access cannot be established, mark cross-entry parity **BLOCKED**, not done.

A separate process holding a SQLite `BEGIN IMMEDIATE` transaction must **never** keep the DB transaction open during an SSH/Provider call. Persist a short operation claim, commit, then execute; release/mark `unknown` in another short transaction.

## 6. Minimum durable data and operation semantics

The coordinator needs only two logical record types; implement with the smallest practical schema:

1. **Lease:** canonical resource, exact Box target digest/identity, opaque lease ID, token hash (never plain token), generation, optional safe project label, created/renewed/expires timestamps, status.
2. **Operation claim:** opaque operation ID, claimed resource key(s), exact target identity, accepted/dispatch-intent/terminal/unknown stage, linked existing receipt or safe correlation ID, timestamps. No raw command, file data, OAuth token or private environment.

A single transaction guarantees uniqueness of active conflicting ownership across processes. Reuse existing BoxWeave M4/lifecycle/access receipts to determine *what is actually known*, not duplicate the whole business-operation state machine. Maintain the current `UNKNOWN` contract (see ADR-0002). The minimal coordinator keeps enough durable evidence to refuse a conflicting modification after a process crash.

If shared coordination storage is corrupt, full, inaccessible, or on a version-mismatched schema, **fail closed for protected mutations** and keep read-only observation available. Do not silently fall back to an unlocked Provider call.

## 7. Concrete example flows

### A. One Agent, one Box (expected common case)

- Agent calls `connect(exec)` without an explicit Box lease.
- Gateway acquires a short Box operation claim, runs the reviewed existing trust checks and forwards the call once.
- If a terminal result is observed, release the claim. No setup, renewal prompt or new user workflow.
- If another mutator attempts the same resource during the call, immediately refuse with `RESOURCE_BUSY`.

### B. Two projects, one Box

- Project A chooses `boxweave_box_lease(action=acquire, box=dev-a)`; receives a private token (shown only to that client).
- During its development session it renews as needed and includes the lease token for mutations.
- Project C reads Box metadata normally; its `connect(exec)` without the token is refused.
- Project A releases. Project C can then acquire its own lease.

### C. Agent disappears

- Project A acquired the Box, issued no incomplete operations, then crashed.
- 10 minutes without renewal: the reservation expires. On the next acquisition, the coordinator checks unresolved operation claims.
- With none: Project C acquires immediately with a **new generation**. No administrator needed; A's old token cannot be used to mutate.

### D. Unknown remote mutation

- Project A issued a potentially mutating command, and the MCP connection died after dispatch intent.
- The lease expires, but the durable operation claim remains `unknown`.
- Project C's conflicting mutation is refused with `RECOVERY_REQUIRED`. Read-only investigation is still available.
- Operator reviews the original target and original operation receipt. Only when completion/noninterference is supported by evidence may the claim be cleared. A narrowly audited explicit administrative override is possible, but must not claim the guest command has stopped or automatically kill it.

### E. Shared host resources

- Project A publishes TCP port 8080 or modifies Tenant X policy.
- Project B's simultaneous change to the **same** port or Tenant is refused; changes to independent port/Tenant/Box resources may proceed.
- For an operation affecting multiple keys, acquisition is all-or-nothing. No implicit global lock for all Boxes.

## 8. Development batches (keep it deliberately small)

| Batch | What is implemented | Exit criteria |
| --- | --- | --- |
| **MAL-0: inventory/contract** | Audit direct gateway forwarded mutators, app lifecycle/access/box-actions, existing claims, precise resource ownership and source/installed Provider variants. Document the protected tool coverage map and safe-read allowlist. | All relevant entry paths classified. Unknown or unprotected mutators explicitly listed as BLOCKED; no false global-coordination claim. |
| **MAL-1: automatic short operation claims** | Tiny shared coordinator, atomic conflict detection, known Box and host-resource keys, immediate errors, direct SSH/HTTP + app/Web integration for selected verified operations. No explicit lease UX yet. | Same-Box and same-host-resource concurrent writes conflict; independent Boxes parallel; crashes preserve `unknown` claims; no background daemon. |
| **MAL-2: optional expiring Box leases** | One MCP lease tool; 10-minute timeout, explicit renew/release/status, lazy expiry, unguessable token/generation, no queues or forced takeover; short docs. | Agent disappears => clean Box available after expiry; stale token refused; unknown operation continues blocking only conflicting mutations. |
| **MAL-3: regression and installed acceptance** | Negative tests for all protected transports, optional app integration, old clients and Provider tools, restart/migration/cleanup/security bounds; record source/fixture/live separately. | Exact-version source and installed-client/Box evidence; existing single-Agent `connect`, Web and auto-sleep behavior preserved. |

**Do not** combine this feature with an Agent Runner, task dispatch system, process supervisor, snapshot/clone, per-Agent RBAC, permanent Box lock, Git worktree manager, or default project-to-Box allocation. Do not increase the number of MCP tools beyond the one lease-management tool until a measured gap justifies it.

## 9. Test and acceptance matrix

| ID | Case | Expected evidence |
| --- | --- | --- |
| MAL-T01 | One legacy Agent without lease | Existing `connect(exec)` succeeds on unclaimed Box; no extra setup. |
| MAL-T02 | Two Agents modify two distinct Boxes | Both execute without global serialization. |
| MAL-T03 | Two simultaneous writes to one Box from SSH/HTTP | Exactly one accepted; other `RESOURCE_BUSY`; no wait or retry. |
| MAL-T04 | Box A lease active; second client tries write | Rejected before upstream dispatch, safe read permitted. |
| MAL-T05 | Same Agent uses valid token and renews | Accepted while current generation valid; expiration extended using server clock. |
| MAL-T06 | Agent stops renewing with no operation in flight | Expired lease automatically re-acquirable after 10 minutes; no administrator. |
| MAL-T07 | Stale Agent resumes after lease acquired by another | Old token/generation rejected and does not dispatch a write. |
| MAL-T08 | Gateway dies after recorded dispatch intent | Operation claim persists `UNKNOWN`; no auto retry; conflicting write denied. |
| MAL-T09 | Existing Web M4/lifecycle operation races with direct MCP | Same coordinator prevents conflicting dispatch; existing receipts and app rules remain valid. |
| MAL-T10 | Two operations on same Tenant/port vs independent resources | Correct resource-level conflicts and safe parallelism. |
| MAL-T11 | One operation needs Box + port keys | All claimed atomically or none, no partial resource ownership. |
| MAL-T12 | Unknown tool/opaque shell classified incorrectly | Generic exec treated as mutation; unknown write capabilities fail closed under protected coverage. |
| MAL-T13 | Coordination DB locked, missing, corrupt, full or permissions unsafe | Typed refusal; no silent unlocked call; no leaked app DB access. |
| MAL-T14 | Box destroyed/recreated using same logical name | Old lease cannot write to new incarnation; target digest mismatch. |
| MAL-T15 | Read-only status/diagnostics during busy lease or recovery | Safe observation remains available with bounded metadata. |
| MAL-T16 | Operator reviews/overrides unknown claim | Explicit audit, no automatic retry, no assertion that remote process terminated. |
| MAL-T17 | Upgrade + legacy client integration and optional allowlist | Existing clients unchanged when no lease; new tool respects allowlist, schemas and disabled/unsupported cases. |
| MAL-T18 | No additional Box software or state | Source/installed evidence that leases exist only in BoxWeave management layer; no agent/daemon installed in guest. |

Tests must include real independent gateway processes (not only sequential mocks), live authorized target identity, interrupted commands and `unknown` acceptance; record exact revision, running version, original instance, evidence and remaining BLOCKED lanes. This document's creation is **S/D only**, not T or live acceptance.

## 10. Compatibility, risk and rollback

- **Default behavior remains familiar:** no extra commands or prompts when one Agent works normally; the only new visible response is an actual conflict or tracked uncertain operation.
- **Not a security sandbox:** A lease token coordinates calls **within authorized BoxWeave entrypoints**. Shared `operator` authentication is still shared; Agent labels are not independently verified, and no per-Agent permission/credential model is invented.
- **Uninstrumented external channels remain outside guarantees.** A third-party Provider client, guest SSH or Incus command can still change resources; this feature cannot make claims about those channels.
- **Background work caveat:** A shell can spawn processes that outlive the MCP command. No managed lease can prove all guest processes have stopped without genuine upstream observability. Avoid overclaiming.
- **Security of storage:** No secret operation arguments/results in coordination DB or diagnostic response. Keep explicit operator control over UNKNOWN override; never leak another project’s task content.
- **Rollback:** Feature-gated deployment must be planned so disabling the new coordinator never silently permits writes while durable `unknown` claims exist. Preserve existing DB, receipts and access claims; revert only after a reviewed recovery policy.
- **Authority:** Source implementation, normative requirements, release, host ACL/group changes, controlled crash tests and real Box mutations require separate authorization. No code, installs, services, versions or Box contents are changed by this documentation.

**Single-line product principle:** *Normally invisible coordination; on conflict, reject immediately; stale clean leases expire; uncertain remote work blocks only the resources it may still affect.*
