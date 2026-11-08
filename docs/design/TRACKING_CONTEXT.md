# End-to-end Tracking ID, Go context, and asynchronous execution — P0

**Owner decision:** Async-first operations and durable Tracking IDs, with safe execution context propagated as far as practical throughout each BoxWeave layer. This is a central product contract, not a logging-only feature. No v1 implementation exists yet.

## Identity vocabulary: do not conflate identifiers

| Identifier | Created by | Lifetime / meaning |
|---|---|---|
| `tracking_id` | BoxWeave at the first accepted logical request | Stable cross-component correlation for one logical task, including reads, denials, mutations and their recovery. A deduplicated retry returns the original tracking ID. Never an authentication credential. |
| `request_id` | BoxWeave HTTP/MCP ingress | One inbound transport attempt. Reconnects/retries receive new request IDs even if the logical task is the same. |
| `operation_id` | Durable BoxWeave journal | One accepted mutating business operation, including Exec and guest file writes. Returned with tracking ID; stable across duplicate submissions. |
| `native_operation_id` | Incus, where available | An upstream execution handle. May be absent or expire; never replace the BoxWeave operation ID. |
| `trace_id` / `span_id` | Internal bounded diagnostic instrumentation | Correlate checked component transitions; a separate trace/span may be associated with one tracking ID. Do not expose a generic tracing backend or secret-bearing logs to MCP. |
| `idempotency_key` | Caller, when provided, or a documented durable admission mechanism | Prevent a logically identical mutation from being dispatched twice. Key is scoped to actor + action + owned target identity + canonical intent. Not the tracking or operation ID. |

The tracking ID can appear in structured MCP tool results, safe diagnostic errors, local CLI output and internal logs. Protocol-level MCP errors that cannot use a tool result include a safe correlation identifier where the protocol permits. An unauthenticated request may receive an opaque request ID but never a private operation trace. `operation_get`, `operation_list`, `operation_output_get` and `diagnostic_get` remain authorization- and ownership-scoped.

Do not reuse BoxWeave's tracking IDs as native Incus UUIDs, file-path selectors, access tokens, or permission claims. Never treat a caller-provided `tracking_id` or HTTP tracing header as proof of authority or a unique task identity.

## Go `context.Context` propagation

```text
MCP transport
  -> authentication/permission checks
  -> validated request context (tracking_id, request_id, actor identity, scope)
  -> BoxWeave domain service and durable admission journal
  -> worker (reconstructed safe durable metadata, original tracking_id)
  -> native Incus SDK / Caddy adapter / SQLite transaction
  -> independent post-state verification
  -> operation receipt, diagnostics and bounded outputs
```

- Pass `context.Context` as the first argument through handlers, domain services, repositories and adapters, including request deadlines and cancellation **signals**. Use typed private context keys; do not use package globals or an unbounded `map[string]any` baggage channel.
- Persist only an explicit, versioned `OperationContext` projection: tracking ID, authenticated actor/token **identifier** (never token secret), tenantless ownership scope/layer ID, exact Box UUID/incarnation, action, idempotency hash, admitted permission decision, timestamps, deadlines/budget and trace link. A Go `context.Context` object is **not serializable** and must not be stored directly.
- Async workers reconstruct a fresh Go context from the durable projection and recheck current authorization policy and native resource identity before a privileged effect. A prior permission decision is audit evidence, not an unlimited future capability.
- A caller disconnect, canceled HTTP request or request deadline **does not automatically cancel a previously admitted or dispatched native operation**. It must remain discoverable and proceed under bounded worker deadlines; if the outcome is uncertain, mark UNKNOWN and reconcile the original work.
- `box_exec` is asynchronous by default for potentially long builds/tests, independent of MCP request lifetime. Record start, running, observed exit code, bounded stdout/stderr and completion state. Do not claim command survival across a host crash unless native reattachment/process supervision was actually verified.
- `operation_get(wait_ms)` is an authorized short read-only wait (0–20000 ms); its timeout or caller disconnect ends **only the wait**, not the original operation. `operation_cancel` is a new, idempotent and separately tracked mutation requesting cancellation of an exact original operation. Native termination is capability-dependent and must be confirmed; original UNKNOWN claims are preserved.
- Read calls may complete synchronously, but still carry request/tracking context and produce safe classified evidence. Small local preflight rejections may be synchronous, with no false operation receipt if dispatch was definitively prevented.
- Handle deadlines as distinct classes: inbound MCP deadline, journal admission deadline, worker execution budget, native Incus API deadline, guest command timeout and verification deadline. Never infer native cancellation from a timed-out caller.
- Nested BoxWeave installations generate independent local tracking IDs; no implicit cross-layer propagation, shared database, parent token inheritance or orchestration. A separately authorized operator may store an external *reference* between layer receipts, but no automatic correlation is required.

## Data and output boundaries

- Every emitted event includes stable `tracking_id` when the work is authenticated and admitted, component, phase, safe error category, timestamp and evidence/coverage state. Use `trace_id`/span when available. Never log raw tokens, full environment variables, command arguments containing secrets, arbitrary guest stdout, guest files or Caddy/Incus API bodies.
- Persist **bounded**, retention-limited command output separately from immutable operation metadata. `operation_output_get` must paginate stdout/stderr by opaque cursor and report `truncated`, `expired`, `not_captured` or `unknown` honestly. A large output cannot exhaust SQLite, memory or an MCP response.
- `diagnostic_get` exposes approved read-only classification/coverage linked to a tracking ID, not arbitrary log files or host process inspection. Authorized Box-level diagnostics are scoped to owned identities; sensitive details remain local private evidence.
- Persist redaction, retention, and limits as documented versioned settings; protect operation IDs, cursors, and trace selectors against cross-token leaks. Tracking IDs are not secret but the details behind them may be.
- Define consistent structured `COMPLETED` semantics for guest Exec: an observed nonzero exit is a completed failed command, not a transport UNKNOWN. `SUCCEEDED` for BoxWeave Exec means its transport and exit result were observed and durably recorded; report actual `exit_code` and an explicit `command_success` boolean separately.
- For a mutating operation, successful Incus/Caddy transport receipt is insufficient: independently verify requested postcondition, with UNKNOWN if native outcome cannot be established.

## Test requirements

1. A single logical create, Exec or file change retains the **same tracking ID** through MCP response, durable record, worker, native adapter events, redacted diagnostic lookup and read-back; repeated identical submission returns the original tracking and operation IDs without a second effect.
2. Two distinct transport retries have different request IDs, one logical tracking ID; a mismatched idempotency payload is denied before dispatch.
3. Kill a client while Exec/build runs. Original operation remains queryable; if lost worker/native continuity cannot be proven, return UNKNOWN rather than invent a successful exit.
4. Kill the worker between intent and native receipt, then restart. Preserve UNKNOWN and prevent duplicate create/Exec/file write; independently observe original target.
5. Test parent/child nested controllers: IDs and credentials do not cross automatically.
6. Test token rotation, scoped trace lookup rejection, output retention expiration, oversized output, secret canaries and truncated diagnostic evidence.
7. Test native command exit 0 and nonzero (both have observed `COMPLETED` state), guest process timeout, missing exit status and transport failure as separate cases.

Related: [operation journal](OPERATIONS.md), [MCP tools](MCP_TOOLS.md), [development loop](../product/DEVELOPMENT_LOOP.md), [security](SECURITY.md).
