# MCP Task Reliability R1 — Recoverable Operation Records and Results

**Global plan:** [BoxWeave feature roadmap](FEATURE_ROADMAP_MASTER.md) — R-02 MCP task reliability. This link tracks the overall selection/status; this document retains detailed scope and test obligations.

**Status:** DESIGN / IMPLEMENTATION HANDOFF ONLY. This document does **not** authorize source implementation, host/Box modification, release or deployment.
**Approved objective (2026-11-01):** Prioritize discovering a previously submitted MCP operation and its known result after timeout, disconnect or reconnect. Do **not** require every remote command to survive transport loss, gateway death, Box restart or host restart.
**Source baseline:** BoxWeave main at `96f0094139de5bc21b91f0b1ab29d65b474a434d` (v0.26.9) immediately before this documentation change.
**Design constraints:** Keep existing BoxWeave architecture, SSH/stdio + authenticated HTTP MCP, single trusted-operator trust model, Provider/Incus boundaries, existing Box lifecycle and clean agentless Boxes. External-first weaving only.
**Related:** [Box operation interface design](BOX_OPERATION_INTERFACE_V1.md), [MCP capability master](MCP_CAPABILITY_INTEGRATION_MASTER.md), [M0–M5 progress and live evidence](../process/MCP_M0_M5_SOURCE_PROGRESS.md), [D3/D4 diagnostics](DIAGNOSTICS_D3_D4_MASTER.md), [remote HTTP MCP](REMOTE_MCP_MASTER.md), [Agent test workflow](../process/AGENT_TEST_WORKFLOW.md).

## 1. Purpose and acceptance definition

For a **supported explicitly tracked task invocation**, the Agent should be able to:

1. Supply a stable caller-generated `request_key` **before dispatch** (or search recent tasks if the key is lost).
2. Ask `boxweave_task_get` or `boxweave_task_list` for the original task after establishing a new MCP connection.
3. Recover the persisted target identity, operation classification, known execution state, observed exit status, and **any retained result**.
4. Learn definitively when dispatch was prevented; distinguish this from an operation whose remote side effect **may** have happened.
5. Avoid double-dispatch of the same accepted request key, including when the first response was lost.

**Best-effort recoverability, not guaranteed execution continuity.** If the gateway or remote process dies before returning an output, report `unknown` and missing/unverified output. A persisted task record is not evidence that a command is still alive or finished. A client-side timeout is not proof that no command ran.

This phase does not build a generic task orchestrator, multi-Agent planner, worker scheduler, durable process supervisor or replay engine.

## 2. Source reality and gap

| Existing feature | Actual ownership | Reuse and limits |
| --- | --- | --- |
| Direct MCP gateway `bwmini/mcp_gate.py` | SSH/stdio gateway; exact Box `connect(exec)` target checks; D3 classified event logging | The classified diagnostic journal does **not** retain a durable user-queryable operation record, request identity, exit status or original output. `connect(exec)` has no stable reconnect key today. |
| Remote HTTP MCP `bwmini/remote_mcp.py` | Authenticated, stateless HTTP request invokes isolated gateway for each request (bounded subprocess) | Reconnection does not reattach to that subprocess. A 45-second HTTP relay timeout can leave upstream outcome uncertain. Never transparently replay. |
| App Service `bwmini/box_actions.py` | SQLite v11 one-time plan/confirm, receipts, `unknown` recovery on uncertain dispatch | Existing application/Web actor and direct Agent gateway are different principals. Do not simply give the gateway access to the app database, worker or more privileged credentials. Receipts omit raw outputs by design. |
| App MCP client `bwmini/mcp.py` | Bounded MCP transport/protocol and deadline | An MCP timeout/EOF cannot establish whether an already sent mutation was applied. |
| D3/D4 `bwmini/gate_diagnostics.py` | 7-day/default 50 MiB classified gateway diagnostics, strict redaction; D4 separately preauthorized guest inspection | Continue using trace links for troubleshooting. A diagnostic event is not an authoritative task record; do not weaken diagnostic privacy to implement task lookup. |
| Upstream `code_runs` and `code_logs` | Real Box execution products in the installed Provider tool catalog where authorized | These observe code-run products, **not** an arbitrary `connect(exec)` invocation. Do not invent a process/run ID for ordinary `connect(exec)`. |

The implementation must verify its precise installed `tools/list` and transport results. Source registrations, historic v0.24.0 tests and fixture tests are not current-version live acceptance.

## 3. Scope, explicit coverage and backward compatibility

### Supported in R1

- Introduce one **opt-in, gateway-owned tracked operation route**, initially for bounded exact-Box command execution where existing trust grants allow it. Candidate name: `boxweave_task_submit`.
- New **read-only** tools: `boxweave_task_get` and `boxweave_task_list`, both bound to the same authorized trusted-operator context and implemented in the existing gateway. Candidate `boxweave_task_capabilities` may report coverage and retention if needed, but do not proliferate tools unnecessarily.
- Keep the record durable across individual HTTP requests, reconnects and gateway-process restarts using a protected gateway-owned store **outside every Box**.
- Detect a duplicate `request_key` before any new side effect. Return the original receipt/state for the same request; reject reuse with different operation, target identity or arguments.
- Persist **result metadata** after confirmed completion, plus an intentionally bounded output extract only when permitted by the operator's capture policy.
- Search recent records when the Agent lost its task identifier.
- Provide a clear coverage indicator: `tracked`, `legacy_untracked`, `provider_run` or `unavailable`.

### Not implicitly covered

- Existing raw `connect(exec)`, Box create/delete, network policy, direct forwarded Platform tools, Web jobs and external tools remain supported **without a new idempotency guarantee** until individually integrated. The gateway must never claim every MCP tool is safely deduplicated.
- `boxweave_task_submit` must **not** gain authority beyond the already permitted `connect(exec)`: no caller-controlled host/user/key, no root/host shell, no implicit new Box access, no bypass of the existing host-wide MCP tool allowlist.
- Explicitly separate the app-side M4 BoxActions receipts from new direct-MCP tracked records. Linking by a safe correlation reference is optional; do not duplicate their business logic or storage privileges.
- No long-running process reattachment, guaranteed streaming output, persistent process ID, automatic retries or new service inside the Box.
- Do not interpret a `request_key` as a permission token or replace OAuth, SSH identity, explicit approval or upstream authorization.

## 4. Proposed API contract (candidate names; final schemas require implementation review)

```json
{
  "tool": "boxweave_task_submit",
  "arguments": {
    "request_key": "<caller-generated stable UUID or random opaque ID>",
    "box": "dev-box",
    "operation": "exec",
    "exec": "pytest -q"
  }
}
```

The submitted `exec` string is still an ordinary powerful shell command run under the existing guest user; it is not automatically read-only or root-confined. The initial task route must not expand the set of allowed command operations or add a new file-write grant. The caller must intentionally reuse the **same** key when locating the same operation; a new key is a new operation.

Successful response schema outline:

```json
{
  "task_id": "<server-assigned opaque ID>",
  "request_key": "<same caller key>",
  "box": "dev-box",
  "state": "completed",
  "execution": {"observed": true, "exit_code": 0},
  "result": {"available": true, "output_truncated": false},
  "coverage": "tracked",
  "next_action": "Use boxweave_task_get for a fresh server-side observation"
}
```

`boxweave_task_get({ "task_id": "..." })` and `boxweave_task_list({ "box": "dev-box", "limit": 20, "cursor": "..." })` return bounded nonsecret metadata, state, revision/time, and permitted retained result. Support lookup by `request_key` through an explicit selector on `get` where possible to handle a first-response loss. The server must reject ambiguous selectors and unknown actor scopes.

**Important:** `completed` means a command's termination/exit code was observed; `exit_code != 0` is an observed failing command, *not* a transport `unknown`. `result.available=false` can coexist with a known exit code if output retention was disabled.

### 4.1 Minimal state machine

```text
recorded -> preflight -> dispatch_intent -> completed (exit code observed)
                    |             |
                    |             +----> unknown (lost response / died / not observed)
                    +------------------> failed_before_dispatch (proven no side effect)
```

- `recorded`: durable task record exists, remote dispatch not yet attempted.
- `preflight`: validating Box identity/permission; no side effect yet.
- `dispatch_intent`: **write-ahead persisted before** any possible remote request transmission. A crash at this stage is conservatively `unknown`, even if the request never reached the Box.
- `completed`: observed a valid result/exit code and committed the result before replying when feasible.
- `failed_before_dispatch`: validation/authorization failure with proven non-dispatch.
- `unknown`: might have executed; never automatically resubmit.

If a task is still being handled synchronously, a query may return `in_progress` with the last durable stage and no claim about actual guest process liveness. On gateway-process restart, stale `dispatch_intent` becomes `unknown`; do **not** convert it to failed or completed based on time alone.

### 4.2 Key-based deduplication

- Validate a bounded, high-entropy caller-supplied `request_key`; enforce unique key in the authorized operator scope and a configured retention window.
- On submit, atomically insert the receipt before remote dispatch. Enforce a unique constraint in durable SQLite; simultaneous SSH/HTTP calls with the same key cannot race into two dispatches.
- Bind the key to canonical task type, exact immutable Box incarnation, and a **private** fingerprint of the full normalized arguments. Same key + same operation returns **the original** record, without a second dispatch. Same key + different payload/Box fails `REQUEST_KEY_CONFLICT` with no dispatch.
- The normal JSON-RPC `id` is **not** a stable deduplication key; HTTP and SSH transports may reuse it.
- A missing or expired key is not proof that a command did not execute. Provide explicit `not_found` / `expired` / `coverage_unverified` semantics; never automatically submit a new execution when evidence is missing.

## 5. Persistence, security and result handling

**Integration INT-01/INT-02:** This feature owns the gateway-private request-key, deduplication and optional retained result. The shared **operation claim/dispatch-intent** for protected mutations belongs to the minimal controller coordination boundary in [R-04](MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md), correlated by an opaque operation ID; it is not a second, independently interpreted remote-dispatch history. Do not give the Web/app actor access to gateway result storage or gateway access to the entire app SQLite DB. All tracked exec uses the same trusted `connect(exec)` adapter as [R-01](BOX_OPERATION_INTERFACE_V1.md); no second guest executor. See [integration decisions](FEATURE_ROADMAP_MASTER.md#5-integration-review-conflicts-reuse-and-corrected-dependencies-2026-11-01).


### 5.1 Store design

- Prefer a **gateway-private task-key/result SQLite table/database** under a protected gateway-owned directory; do not cross app-Service ownership. A separate **minimal shared coordinator** may hold canonical protected resource claims and dispatch-intent linked by `operation_id`; the gateway task store must not independently override that claim state. Reuse vetted code patterns from `box_actions.py` for one-dispatch receipts and `gate_diagnostics.py` for file-owner/mode/size validation.
- Restrict directory permissions to 0700 and database to 0600; no symlinks, unsafe hard links, group-writable parents or user-controlled storage paths. Define WAL/lock behavior for concurrent SSH/HTTP gateway processes and crash-safe transactions.
- Store only immutable target reference/digest, operator scope, request-key fingerprint or vetted bounded identifier, operation type, stage, timestamps, duration, safe status/error code, output presence/size/hash and optional bounded data.
- Do not persist SSH keys, tokens, OAuth codes, raw full commands, source file contents, prompts, environment secrets or arbitrary exception messages in diagnostics. Store only the private argument fingerprint necessary for deduplication; avoid printing it in ordinary logs.
- Explicitly define retention and quota. Candidate defaults: 7 days metadata; output capture disabled by default until separately configured, with <=64 KiB per task and a small host-wide budget. Never silently preserve sensitive stdout/stderr as general diagnostics.
- On missing/unreadable storage, return an explicit `tracking_unavailable` before attempting a **new tracked dispatch**. Do not silently fall back to untracked `connect(exec)` and falsely return a recoverable task.
- When output capture is enabled, document sensitivity and actor authorization, keep it private, bound and expiring. No automatic public issue/report disclosure.

### 5.2 Identity and transport

- Preserve the existing single trusted-operator model; **do not** claim true per-OAuth-client or per-SSH-client isolation. In particular, the HTTP bridge currently authenticates a request but does not forward a stable individual client principal into the isolated gateway process.
- Same operator's SSH and HTTP connections can look up shared records only where both are explicitly authorized by the tool policy. A request key grants neither new Box access nor new login.
- Verify the Box from the provider registry and trusted host-key/managed workload identity before dispatch. Do not accept caller-selected host, username or key; pin the exact Box incarnation into the receipt.
- New query tools are read-only; the existing host-wide tool allowlist must filter **both discovery and call**. Unlisted `task_submit` cannot be smuggled through a read route.
- No installation, agent process, task journal or management metadata inside the guest Box. All persistent tracking lives in BoxWeave's host/controller side.

## 6. Transport-loss and failure behavior

| Situation | Correct recovery behavior |
| --- | --- |
| Reply lost **after server recorded completion** | Lookup by `request_key` or recent list returns the committed result or bounded result metadata. No redispatch. |
| Reply lost **after dispatch intent** but before confirmed completion | Query returns `in_progress` while a live handler is known, otherwise `unknown`; no replay. Retained output may be absent. |
| Connection lost **before durable acceptance** | No known task may be found. This alone does not prove no remote side effect for legacy/untracked calls. The tracked endpoint must write before forwarding. |
| Agent retries same accepted key | Query/return same receipt; no second remote command, including across concurrent transports and gateway restarts. |
| Agent retries with same key but changed command/Box | Reject key conflict before dispatch, no leaked raw command. |
| Remote command exits 7 | `completed`, `exit_code=7`, failure is the command's observed result. |
| HTTP relay timeout / client cancel / gateway death | Conservative `unknown` if remote dispatch might have started. Do not claim running or killed without independent evidence. |
| Box replaced under same name | Old task remains bound to original immutable target. Never attach it to replacement Box. |
| Output truncated or disabled | Mark truncation/availability explicitly; do not invent full results or reconstruct inaccessible output. |
| Database corruption/full, permission failure | Refuse new tracked submits; preserve original record/media for diagnosis; no silent untracked fallback. |

## 7. Implementation batches and delivery boundaries

### MTR-0 — Audit and contract

- Inventory direct gateway, isolated HTTP relay, app receipts, D3 journal, current `connect(exec)` result shape and installed Provider permission/availability. Verify actual source/runtime versions.
- Decide the precise gateway-owned storage layout, transactional insert/order, coverage boundary, response shape and output retention policy; consider concurrent SSH/HTTP gateways and upgrades.
- Implement fixtures for state machine, duplicate keys, key collisions, stale target and pre-dispatch refusals **without executing commands**.

### MTR-1 — Gateway durable receipt lookup (first useful release slice)

- Coordinate transaction ordering with R-04's minimal operation claim authority before enabling any tracked **mutating** dispatch: persist accepted task key and required claims/dispatch intent before remote send, or refuse without dispatch. A task store alone does not make an operation collision-safe. If R-04 has not been activated, explicitly label the task as `tracked_only`, not globally coordinated.


- Add safe gateway-owned database and `boxweave_task_get` / `boxweave_task_list`; preserve existing D3 event journal separately and link by safe trace identifiers.
- Implement bounded pagination, actor scope, 0700/0600 storage, expiry/quotas, storage errors and tool allowlist enforcement.
- Add a protected tracked `exec` submission path with write-ahead `dispatch_intent`, exact Box trust and atomic idempotency keys. Do not expose an endpoint that can pass arbitrary commands to the *host*.

### MTR-2 — Known-result capture and uncertainty recovery

- Commit observed exit status and bounded result summary before returning the upstream success when possible.
- Treat duplicate accepted keys as read-only lookups. On interruption, persist `unknown` or recover stale dispatch intents after gateway restart; never execute them again.
- Offer optional bounded output retention under operator policy; verify no argument/output leakage through logs/diagnostics.
- Keep `connect(exec)` backward-compatible and explicitly document that legacy calls still lack deduplication.

### MTR-3 — Transport, packaging and real acceptance

- End-to-end test authenticated SSH MCP and HTTP MCP against the **same real managed Box**; simulate client disconnect before initial response and recover by request key.
- Exercise 45-second HTTP relay budget, worker/gateway crash, simultaneous duplicate calls and mixed-version Provider behavior. Record which interruptions yield confirmed results versus honest unknowns.
- Test reinstall/upgrade and gateway DB migration on a consistent test-system copy with unrelated app/diagnostics receipts intact; verify no more privileges than `connect(exec)`.
- Apply accepted documentation updates to normative requirements/scope/acceptance only after owner authorization; update agent guide, tool descriptors, packaging and test ledger. No version bump for design alone.

## 8. Acceptance matrix

In addition to the tests below, cross-entry protected mutation tests **INT-T01–INT-T03/INT-T06** in the [master roadmap](FEATURE_ROADMAP_MASTER.md#54-joint-acceptance-tests-in-addition-to-child-tests) are the gate for claiming coordination. Preserve the narrower claim of R-02 when only reconnect lookup is accepted.


| ID | Acceptance requirement | Evidence |
| --- | --- | --- |
| MTR-T01 | Submit tracked command, observe exit 0 and result | Actual Box command output + exit; record saved and queried |
| MTR-T02 | Exit 7 is `completed`, not `unknown` | Real command + persisted exit 7 |
| MTR-T03 | Reconnect and retrieve committed result from another MCP connection | Authenticated SSH/HTTP live test |
| MTR-T04 | Disconnect before first reply; lookup by caller-generated key | Original task observed, no new execution |
| MTR-T05 | Two simultaneous submissions with same key | Exactly one remote dispatch, same receipt |
| MTR-T06 | Same key + different payload or Box incarnation | Explicit conflict and zero new dispatch |
| MTR-T07 | SSH/HTTP gateway process crash after dispatch intent | `unknown`, no replay; restart preserves record |
| MTR-T08 | Refused trust/permission/path before dispatch | `failed_before_dispatch`, zero remote command |
| MTR-T09 | Backend/Box re-created with same logical name | Old record never retargeted; new submit rejected when stale |
| MTR-T10 | Task record storage unavailable / full / invalid permissions | Reject tracked submission, never silent untracked fallback |
| MTR-T11 | 64KiB capture, no-capture default, expiration and quota | Output limits, truthful truncation and no secrets in diagnostic journal |
| MTR-T12 | Host-wide tool allowlist, unauthorized client, OAuth errors | No authority escalation; query and submit appropriately denied |
| MTR-T13 | Legacy `connect(exec)` still works | Compatibility preserved, no false tracking claim |
| MTR-T14 | Clean Box before/after | No resident Agent/service, package install or new Box files from tracking itself |
| MTR-T15 | App/Web BoxActions and D3/D4 isolation | Receipts and policy not mixed; no app database privilege transfer |

Follow [Agent test workflow](../process/AGENT_TEST_WORKFLOW.md): focused fixtures first, then separately authorized clean disposable Box and mixed-version tests. Record exact source SHA, running/installed versions, commands, observed exits and evidence class S/T/L/U/BLOCKED. On a blocked test, continue independent verification and never mark the unexecuted criterion as passed.

## 9. Rollback and release rule

- New task tools must be separably disableable and rollback-safe; preserve old `connect`, existing app actions, D3/D4 journals and stored Box data.
- An uncertain write must **never** be auto-replayed as a recovery mechanism. A corrective attempt is a separate intentional operation after review of the original target.
- Retained records must survive compatible gateway service restart, not silently disappear on new connection; specify database schema migration and rollback compatibility before first release.
- This document records agreed **design objectives only**. No source code, package, runtime feature, host network, Box state or current installed version was changed by its creation.
