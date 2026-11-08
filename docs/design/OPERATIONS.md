# Durable operations and recovery — P0

Incus has native operations; **BoxWeave operations are separate durable business records**, tracking authorization, stable Box identity, ownership, idempotency, original **Tracking ID**, Go execution context metadata, dispatch evidence, and postcondition verification across native Incus changes and local SQLite. All mutations, including guest Exec and file writes, are **asynchronous by default**. Reads may be synchronous but still carry a Tracking ID.

## Lifecycle

```text
REQUEST -> ADMITTED (durable before side effect)
                 |
            DISPATCHED -> RUNNING -> VERIFYING -> SUCCEEDED
                 |           |          |
                 +-----------+----------+--> FAILED (proven)
                 |           |          |
                 +-----------+----------+--> UNKNOWN (ambiguous)
```

An operation may later move from UNKNOWN to SUCCEEDED or FAILED **only** after sufficient independent evidence; a transient GET returning not found is not by itself proof that a possibly dispatched action never ran. `operation_cancel` is a **separately journaled mutation request**, not an automatic stop or rollback. `CANCEL_REQUESTED` is not `CANCELLED_BEFORE_DISPATCH`, a verified stopped guest Exec, or `CANCELLED`. Only proven no-dispatch can claim no native effects; stopping an already running guest command cannot undo effects it previously made.

## Minimum journal fields

`operation_id`, original `tracking_id`, inbound `request_id` and safe diagnostic `trace_id` linkage, owner/layer identity, authenticated actor/token ID (not token value), operation type, requested `box_uuid` and stable incarnation, normalized **redacted** immutable intent/fingerprint, scoped idempotency key digest, safe persisted `OperationContext` projection, queued/dispatch timestamps, native Incus operation ID where available, native resource revision, state transitions, bounded error, observed guest command exit, verification evidence, and private bounded output reference.

Require a nonempty, caller-generated, reusable `idempotency_key` for **every MCP mutation**; reject missing keys before admission/native effects. Persist a safe pre-dispatch intent transactionally. Enforce unique (layer, authenticated stable actor, key), with separate canonical action/intent fingerprint (including existing target UUID/incarnation where applicable). For `box_create`, UUID does not exist at admission. Exact duplicates return original Tracking/Operation receipts; same-key/different-action or payload conflicts fail with a typed error. Do not rely on a short-lived in-process map as the only operation registry.

## Worker and reconciliation

- Acquire per-Box/per-port/owned peer-pair resource locks transactionally. Do not block unrelated resources unnecessarily.
- Recheck Token revocation and Box ownership immediately **before dispatch**. If revoked and provably undispatched, record `AUTH_REVOKED_BEFORE_DISPATCH` (or equivalent), refuse effect and keep audit evidence. An already-dispatched or uncertain operation must remain independently reconciled and never be labeled canceled solely because its Token was revoked.
- Persist native operation identity immediately after obtaining it. A crash in the interval between effect and persisted receipt remains UNKNOWN; never blindly send the command again.
- Categorize transport timeout separately from native operation failure, application deadline and failed verification. Incus may finish after a local timeout.
- Check native state and original operation whenever possible; reconcile native port-forward and simple peer desired-versus-actual state under strict ownership and revision checks.
- Permit exact-operation cancellation **only with proof**: atomically cancel a queued-undispatched intent or request native termination of a supported, observed running Exec. Record cancellation's own Tracking ID/receipt and authorization; never free UNKNOWN claims based on a cancel request alone.
- Implement `operation_get(wait_ms)` as read-only long polling, default 0 and maximum 20000 ms. Canceling a waiting HTTP request does not cancel the underlying durable native operation.
- Verify actual postconditions (CPU/memory config, Box state, guest command exit, native peer isolation and owned forward identity). A successful HTTP response from Incus alone is not sufficient.
- Bounded stdout/stderr, paginated `operation_output_get`, expiration and retention; redact tokens, file bytes and private data. Preserve UNKNOWN original identities until reviewed resolution. An observed guest exit `!= 0` is **completed execution** with `command_success=false`, not an MCP timeout or UNKNOWN.
- Define safe, case-specific compensation only for proven owned partial effects. No catch-all retry, force-delete or reset of a suspicious Box.
- SQLite durability and crash recovery must be demonstrated under service kill, host reboot simulation and nested parent restart.

## Incident-derived result and readiness invariants

- **Native state vs business readiness:** `box_get` and `box_list` must consistently expose native stopped/running state separately from actual guest Exec readiness and any still-unverified create requirements. Incus `RUNNING`, a healthy daemon or a successful create API response is not proof that the admitted Box is fully ready. A create interrupted after native instance creation retains the exact UUID/ownership and an honest partial/UNKNOWN receipt; its cleanup is not assumed complete.
- **Waiting, state probing and replay are different operations:** Native `Operation.WaitContext` reaching its deadline means *this waiter stopped waiting*, not that Incus stopped. If native dispatch may have occurred and final effect is unproven, retain `UNKNOWN` and reconcile the original native operation. Transient read-side NotFound may be rechecked with a bounded deadline and exact UUID; it cannot prove deletion and must never trigger repeat dispatch.
- **Exec result truth:** An observed guest exit 0 is distinct from an observed nonzero exit and from *no trustworthy exit value*. Missing/malformed exit metadata, a broken websocket, or an incomplete native operation cannot default to zero; preserve the original Exec operation and mark evidence missing/UNKNOWN as appropriate. A nonzero **observed** exit remains a completed guest command, not an RPC failure invented from stderr.
- **Disconnected client, noisy output and caps:** Durable Exec ownership/output/exit observation must not depend on a live MCP HTTP request; test both quiet and continuously-writing commands. Enforce byte limits while streaming, keep a visible truncation/coverage indicator, and never suppress a concurrent transport/native error merely because the output exceeded its cap. A client long-poll timeout affects the poll only.
- **Mutation preflight and partial effects:** Validate all requested native capabilities, input bounds and exact ownership *before* side effects. A failed cleanup, rollback or creation after a native instance exists cannot be described as "nothing happened"; preserve observed partial effects and the original IDs. Existing resource claims lock only the affected Box/forward/pair; no general lease coordinator is required.

Evidence and exact incident IDs: [Containarium study](../research/CONTAINARIUM_MAIN_HISTORY_2026-11-07.md).

## Box deletion as a multi-resource operation

`box_delete` is **not** one Incus API call once BoxWeave has created native TCP/UDP forwards and directional peer grants. Its standard confirmation covers removal of **all verified BoxWeave-owned** dependent forward/peer claims for the exact Box UUID/incarnation; it must not imply deletion of foreign host/network resources.

Before dispatch, snapshot and lock the exact owned Box **and its dependent forward/peer claims**, and ensure concurrent create-forward, peer-set, update, stop and delete cannot race into an orphan exposure. The journal must record individual step intent, native result, postcondition observation and any missed/uncertain evidence. Backend differences matter: an Incus instance proxy device may disappear with the instance, while an independent Incus network-forward entry may survive. Use a verified provider-specific dependency graph, not guessed cascading behavior.

Preferred **reconcilable** phases: `DELETE_PREFLIGHT` -> `FORWARD_CLEANUP` -> `PEER_CLEANUP` -> `INSTANCE_DELETE` -> `POST_VERIFY` -> `TOMBSTONE_COMMIT`. Include a native stop/substep only when the chosen supported deletion policy requires it. Mark `SUCCEEDED` only after the exact instance is gone, all managed dependents are absent, and the tombstone/ownership transition is durably committed. Verify both actual Incus state and local ownership records, not just a successful API response.

If a forward is verified removed but the instance delete fails, **the Box may still exist but its service is no longer published**. Record terminal `FAILED` with `partial_effects=true` only when the failure and all relevant actual state are **known**; use `UNKNOWN` when the dispatched outcome remains ambiguous. `PARTIAL` is an acceptance-report classification, **not** an additional operation-terminal state.  `operation_get` must identify the failing phase and successful earlier effects. Preserve original IDs and the journal; do not silently recreate the forward, discard the Box record or release UNKNOWN claims. An observed FAILED with verified stable native state may be repaired through a separately approved reconciliation, or by a **new** idempotency-keyed delete after safe identity/lock checks. Reusing the original key returns the original receipt and **never** dispatches a second delete. A merely missing Box after a lost reply does not prove dependent network cleanup; keep UNKNOWN until verified. A native **orphan forward** whose Box no longer exists may be inspected through `port_forward_list` and removed through the existing `port_forward_delete` **only if its exact original BoxWeave ownership, native forward identity and Box tombstone/operation history are independently verified**. Do not require that the deleted Box still exists for this narrowly scoped cleanup; never adopt or remove foreign forwards.

The remote Agent may query existing `operation_get`, `box_get` and `port_forward_list` to determine observed residuals. A failed `box_get` from a deleted Box is only a specific data point, not authority to clean arbitrary network state. Sensitive host stack traces, privileged diagnostics, socket access and raw journal inspection stay in the **host-local** CLI/OS; the MCP API does not grow a new host-log tool.

## Tool response

An accepted mutation responds with persistent `tracking_id`, `operation_id`, and current state. The `operation_cancel` response distinguishes its new `cancel_operation_id` from the `target_operation_id`. `operation_get`, `operation_list`, and `operation_output_get` expose scoped progress, bounded output/exit and evidence; `operation_get` includes a small redacted phase/error/evidence summary linked to the original Tracking ID; protected local CLI logs hold further diagnostics. No separate diagnostic MCP API exists. Read-only tools may return immediately but carry correlation. A valid duplicate request returns original IDs. Use standard MCP error categories and meaningful domain error codes; do not invent fake HTTP/gRPC statuses or misclassify UNKNOWN as not found. See [Tracking/context](TRACKING_CONTEXT.md).

## Safe structured failure evidence

`operation_get` must expose a **bounded, sanitized** diagnostic envelope sufficient for a remote Agent to distinguish which native stage failed and which side effects were independently observed: original Tracking/Operation IDs; controller VERSION/build SHA and layer; operation action/target UUID; stable phase/substep; typed `error_class` and native error category/status where safe; limited sanitized native message; `dispatch_possible`; last observed target state; owned forward/peer cleanup summary; `observed_at`, verification coverage and missing evidence; `reconciliation_state` and one actionable **safe** next step. Host paths, stack traces, raw Incus bodies, guest secrets and token values remain private. The full error may not be knowable from MCP; when missing evidence is material, report `local_operator_diagnostics_required` instead of fabricating root cause.

Pre-admission denials that never created an operation return a safe Tracking ID plus structured validation/authorization error in the tool response. Do not invent an `operation_id` for them. Support version/build identification in `host_status` so Agent can compare first and candidate installations without relying on release URLs.

## Privileged manual UNKNOWN recovery (owner approved)

When an exact mutating native outcome remains ambiguous, hold its Box/port/peer claim and preserve the original `UNKNOWN` receipt. A protected **host-local** recovery flow may inspect the original Operation and exact resource/identity snapshot, issue a short-lived one-use challenge, and require a second explicit operator confirmation/reason before **releasing only that local claim with a durable audit entry**. The original Operation outcome, Tracking ID and idempotency key never change solely because the lock was released. This is **not** retry, cancellation, postcondition success, automated rollback or native resource adoption. Abort on altered snapshots or audit-write failure. Do not expose force-unlock on MCP or introduce a general lease management API.

See [minimal local recovery procedure](../process/UNKNOWN_MANUAL_RECOVERY.md) for confirmed sequence, negative tests and reference to earlier main code/evidence.

## High-risk cases

Repeated create after lost reply; start/forward/peer-set competing with delete; Box delete after partially removed forward; instance gone but stale network forward survives; duplicate original delete idempotency key; native port forward add interruption; Exec streaming interruption; deletion with reused name; partial forward apply; parent restart with child operation still running; native success but postcondition mismatch. See [fault simulation](../verification/FAILURE_DRIVEN_RELEASE_SCENARIOS.md).
