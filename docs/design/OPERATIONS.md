# Durable operations and recovery — P0

Incus has native operations; **BoxWeave operations are separate durable business records**, tracking authorization, stable Box identity, ownership, idempotency, dispatch evidence, and postcondition verification across Incus, Caddy and local databases.

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

An operation may later move from UNKNOWN to SUCCEEDED or FAILED **only** after sufficient independent evidence; a transient GET returning not found is not by itself proof that a possibly dispatched action never ran. `CANCEL_REQUESTED` is not the same as `CANCELLED`.

## Minimum journal fields

`operation_id`, owner/layer identity, authenticated actor, operation type, requested target `box_uuid`, stable incarnation/creation identity, immutable normalized intent, scoped idempotency key, queued/dispatch timestamps, native Incus operation ID where available, Caddy/resource revision, state transitions, bounded error, verification evidence, and redacted diagnostic references.

Persist a safe pre-dispatch intent transactionally. Apply a unique constraint to the idempotency scope. Exact duplicate requests return the original receipt; conflicting or same-key/different-payload requests fail with a typed conflict. Do not rely on a short-lived in-process map as the only operation registry.

## Worker and reconciliation

- Acquire per-Box/per-route/port/ACL resource locks transactionally. Do not block unrelated resources unnecessarily.
- Persist native operation identity immediately after obtaining it. A crash in the interval between effect and persisted receipt remains UNKNOWN; never blindly send the command again.
- Categorize transport timeout separately from native operation failure, application deadline and failed verification. Incus may finish after a local timeout.
- Check native state and original operation whenever possible; reconcile Caddy and ACL desired-versus-actual state under strict ownership and revision checks.
- Verify actual postconditions (CPU/memory config, Box state, guest command exit, native ACL, Caddy routing, forward identity). A successful HTTP response from Incus/Caddy alone is not sufficient.
- Bounded stdout/stderr, pagination and expiring retention; redact tokens and private data. Preserve UNKNOWN original identities until a reviewed resolution.
- Define safe, case-specific compensation only for proven owned partial effects. No catch-all retry, force-delete or reset of a suspicious Box.
- SQLite durability and crash recovery must be demonstrated under service kill, host reboot simulation and nested parent restart.

## Tool response

An accepted mutation responds with the persistent `operation_id` and current state. `operation_get` and `operation_list` expose scoped progress, terminal result and evidence. Read-only tools may return immediately. Use standard MCP error categories and meaningful domain error codes; do not invent fake HTTP/gRPC statuses or misclassify UNKNOWN as not found.

## High-risk cases

Repeated create after lost reply; start competing with delete; concurrent ACL additions; Caddy admin outage after desired route persisted; Exec streaming interruption; deletion with reused name; partial forward apply; parent restart with child operation still running; native success but postcondition mismatch.
