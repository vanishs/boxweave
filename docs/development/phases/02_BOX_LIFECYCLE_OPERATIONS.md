# S2 — Exact owned Box lifecycle and persistent asynchronous work

**State: NOT_STARTED (2026-11-08).** Execute end to end in one pass after reading the S1 handoff. Deliver all named tools and their focused native and negative tests; fix and rerun without leaving an unverified implementation stub.

**Entry context:** [Phase protocol](../PHASE_EXECUTION_PROTOCOL.md), [S1 checkpoint](../EXECUTION_LOG.md#current-checkpoint), [catalog of 20 tools](../../design/MCP_TOOLS.md), [owned operations](../../design/OPERATIONS.md). Confirm S1's actual installed digest/SQLite schema/roles; don't reload all S1 design files or recreate its admission engine.

**Fetch at affected subtask only:** [Incus integration](../../design/INCUS_INTEGRATION.md) for native Project, effective config and UpdateInstance; [guest management](../../design/GUEST_MANAGEMENT.md) for Exec/ready; [specific tool contract](../../design/MCP_TOOL_CONTRACTS.md#box_exec-and-operation-tools) for cancel/output; [Tracking ID](../../design/TRACKING_CONTEXT.md) for split contexts; [last SDK P0 results](../../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md) for recorded output and control WebSocket; [SDK mapping](../../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md) per method. Testing: [acceptance](../../verification/ACCEPTANCE.md) **A-004–010, A-016–022, A-034, A-039, A-043–047 (partial dependency), A-054–055, A-057–058, A-060–064, A-066–068, A-074–078, A-080–082** and any later test for the actual changed behavior.

## Public tool inventory owned here — exactly 12

**Seven Box lifecycle tools:** box_create, box_list, box_get, box_update, box_start, box_stop, box_delete.

**Exec:** box_exec.

**Four Operation tools:** operation_get, operation_list, operation_output_get, operation_cancel.

**All these must use the S1 identity, auth, durable admission and typed Tool boundary**, not parallel operation registries or a second transport.

## Internal work in this single phase

1. **Native capability and identity:** exact instance UUID/incarnation, owned Project/profile/root pool/nesting and Ubuntu default, before-first-start network fail-closed. Check effective config, not request echo. Reject all foreign/stale names, unsupported image/capabilities, raw Incus configuration, disk shrink and unsafe quantity overflow. Detect absent/no-root-disk and name/IP reuse. Respect first-start vs stop/start readiness.
2. **Lifecycle:** async create and optional start (start=true default), stop/start idempotent *with distinct caller keys*, targeted CPU/memory/disk-growth typed update (omitted fields preserved), correct nesting/internet change, deletion of **exact owned instance** only with explicit scope. Verify post-state and truthful native running vs guest ready; preserve original partial/UNKNOWN if native state is unreadable or create completes after timeout.
3. **Journal and worker integration:** every admitted native mutation persists original intent/actor/Tracking/BoxWeave op/native op ID, with bounded independent status polling and idempotency. operation_get uses non-canceling bounded observation; operation_list paginates scoped records; backend restart rehydrates original receipts, no duplicate effects. Revoke an undispatched token safely; do not retroactively undo dispatched work.
4. **Guest Exec:** typed argv/UID/GID/cwd/env, guest root by default, worker-owned SDK **RecordOutput=true without output writers** for durable async, preserve original operation ID and recorded stdout/stderr log URLs, native return 0/nonzero/missing as distinct, bounded paginated retrieval after separate MCP/SDK connection or restart, and no host shell. Verify quiet and noisy output limits.
5. **Cancel:** queued-undispatched cancellation through journal; live SDK Exec control-WebSocket signal **only while the owning worker holds that socket**, verified guest termination; a detached Exec Operation.Cancel() was actually unsupported. Report CANCEL_REQUESTED/unsupported/UNKNOWN honestly; caller disconnect is not task cancellation. No custom guest process supervisor.
6. **Delete integration seam:** phase-local delete must safely delete only a Box **without active owned network dependents**; for any potential future forwarding/peer record, refuse/hold unless claims are verified. **S4 explicitly finishes and reaccepts** box_delete with owned proxy/peer dependency removal and mid-delete failures. Do not falsely mark A-043–046 fully PASS here.

**Load on demand for slow-call/Agent-stall cases only:** [legacy Incus I/O + MCP outage incident](../../research/LEGACY_MAIN_INCUS_MCP_INCIDENT_2026-11-10.md) (O-027/O-028). Distinguish HTTP timeout from durable native state, test bounded reads/backpressure and never replay an original UNKNOWN. A separate heartbeat subsystem is not approved.

**MCP future compatibility, no new S2 feature:** At the Operation receipt/output subtask, read the [Tasks extension future-adapter seam](../../research/MCP_TASKS_EXTENSION_SEAM_2026-11-10.md) **only to avoid welding transport sessions, ad-hoc task IDs or client polling to business durability**. Keep stable typed result/operation ID and native evidence usable by all current clients. Do **not** implement or advertise `tasks/*` during S2 or add a speculative schema field, task worker or 21st MCP Tool. S2's original security, crash and UNKNOWN acceptance remains mandatory.

## S2 additional implementation pack — 4 MCP + 4 domain

Use [tagged SDK study](../../research/MCP_GO_SDK_CONTINUITY_REVIEW_2026-11-10.md), [continuity design](../../design/AGENT_OPERATION_CONTINUITY.md) and **mandatory [TC-001–TC-010](../../verification/CONTINUITY_ACCEPTANCE.md)** only at the affected subtask.

1. **MCP `_meta`** via official SDK: optional bounded work/Agent labels, tested Tool-argument fallback for clients without custom metadata, never authorization.
2. **MCP middleware:** one redacted request/method/actor/timing hook, no raw guest data or tracing server.
3. **MCP typed output:** original IDs, last verified evidence and safe read-next hint; legacy text fallback.
4. **MCP progress:** optional notification only for a still-live request when client/token supports it; does **not** delay immediate durable async admission.
5. **Domain durability:** finish original worker/Incus postread, UNKNOWN and keyed replay prevention; reviewed safe migration/backups from exact S1 SQLite schema v1 before changing DDL.
6. **Domain identity:** persist untrusted `work_ref`/`agent_ref` **once**, with trusted authenticated actor and immutable original receipt.
7. **Domain search:** extend **existing** `operation_list` with bounded indexed, authorized actor/work/agent/state/time filters and private cursors.
8. **Domain recovery:** deterministic structured `READ_OPERATION/READ_OUTPUT/INSPECT_OWNED_BOX/HOST_LOCAL_REVIEW/NONE` only; no automatic redo.

**Extra small SDK wins:** correct Tool read-only/destructive/idempotency annotations, authenticated private cache, SDK-native old/new negotiated protocol errors. [MCP Tasks](../../research/MCP_TASKS_EXTENSION_SEAM_2026-11-10.md) remains future adapter only, no task database or new Tool. All original S2 native and security tests still apply.

## Test and negative matrix

- Real native create/read/list/start/stop/update/delete against the **same exact binary/instance UUID**; new image, storage pool, start=false, restart, nesting availability, name reuse, foreign Box, forged stale IDs, and omission-vs-zero/clear fields.
- Two-actor admin/viewer, duplicate same idempotency key, conflict same-key different intent, distinct keys for stop→start→stop, Token revocation while queued, canceled polling cannot cancel native work, delayed read-side 404 isn't proof of delete.
- Worker HTTP disconnect, quiet/delayed/noisy Exec, exact recorded stdout/stderr and metadata via **fresh connection**, nonzero exit 7 and canceled guest 143 without reporting Incus op success as guest success. Test unsupported cancellation after control socket disappearance; never cancel other native operations.
- Fault-inject lost Incus ACK **after effect**, lost SQLite commit, duplicate create/delete reply, worker killed before/after persist and native readback. Store original IDs and UNKNOWN locks, then independently reconcile; no re-dispatch. Include known-bad negative control where feasible.
- Local recovery procedure for original UNKNOWN must require explicit host-local operator confirmation and audit; no remote MCP unlock. Do not invent an Incus/Cowsql repair path or re-create a Box just because nested lab daemon lost leader.

## Exit, cross-phase continuity and blockers

**PASS for S2-owned subset** requires all 12 Tool schemas and operations implemented with native and negative acceptance evidence and a complete Box/Exec/operation lifecycle against the installed candidate. **The whole v1 20/20 and network-dependent deletion are NOT yet PASS.**

Log schema migrations, owned UUID ledger, native ID/execution log retention path, command cancellation limits, and any residual originally UNKNOWN resource. Test and hand off **reusable resource locks + delete-dependency interface**, but leave network object rules to S4. If a specific native feature is impossible, classify and document exact unsupported behavior while finishing independent work; a mandatory unsatisfied acceptance remains BLOCKED and later publication cannot silently pass.

**Next:** [S2.5 BRIDGEHEAD](025_BRIDGEHEAD.md) performs a **separate, stronger real-host/native L0→L1→L2 development-base gate** before S3. Finish **all S2 tools and native acceptance first**; do not omit mutation semantics, bounded operation reads, secure first-start networking, worker crash/UNKNOWN or guest Exec to produce a superficial early deployment. BRIDGEHEAD cannot assert S4 full network/peer/LAN or S5 release PASS; if host pilot is unsafe or unauthorized, S3 may proceed only in existing isolated test environments with the original release gates intact.
