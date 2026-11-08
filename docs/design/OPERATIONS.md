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

## Tool response

An accepted mutation responds with persistent `tracking_id`, `operation_id`, and current state. The `operation_cancel` response distinguishes its new `cancel_operation_id` from the `target_operation_id`. `operation_get`, `operation_list`, and `operation_output_get` expose scoped progress, bounded output/exit and evidence; `diagnostic_get` provides a redacted trace/phase explanation by Tracking ID. Read-only tools may return immediately but carry correlation. A valid duplicate request returns original IDs. Use standard MCP error categories and meaningful domain error codes; do not invent fake HTTP/gRPC statuses or misclassify UNKNOWN as not found. See [Tracking/context](TRACKING_CONTEXT.md).

## High-risk cases

Repeated create after lost reply; start competing with delete; concurrent ACL additions; native port forward add interruption; Exec streaming interruption; deletion with reused name; partial forward apply; parent restart with child operation still running; native success but postcondition mismatch.
