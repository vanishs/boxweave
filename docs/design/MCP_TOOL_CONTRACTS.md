# v1 MCP tool contracts: agent development and operation control

**Status:** accepted behavioral design, not an implemented or fully frozen JSON Schema. Implementation Agents must test official MCP Go SDK behavior and real Incus operations before freezing detailed schemas. Do not add a generic native API pass-through.

## Guest file changes: `box_file_apply`

Required: exact owned `box_id` (native UUID binding), safe guest `path`, and `mode`. All mutating requests **MUST provide a caller-generated, reusable and nonempty `idempotency_key`**. No implicit server-generated substitute is accepted; reject missing/invalid keys before any native effect. Same actor/layer/key plus identical normalized action/intent returns the original receipt; reuse with different intent is a conflict. The native Box UUID may not yet exist at `box_create` admission, so it cannot be required to construct the idempotency uniqueness scope. `mode` is one of:

| Mode | Inputs | Safety/result |
|---|---|---|
| `create` | New regular-file content; `must_not_exist=true` | Refuse an existing path, symlink, special file or unsafe parent; verify new bytes/hash. |
| `replace` | Complete new content; required `expected_sha256` or equivalent stable revision | Compare original regular-file bytes immediately before native write; refuse stale revisions; verify resulting bytes/hash. |
| `patch` | Bounded text patch and required original SHA-256/revision | Apply to the observed original file only; malformed/context-mismatch patches fail; verify resulting bytes/hash. |
| `delete` | Required original SHA-256/revision | Remove **only one verified existing regular guest file**; never recursive deletion, directories, host paths or device files. |

**No bypassing native limitations:** A pre-read hash check followed by a write is *not automatically atomic*. Test symlink/hardlink/race and atomic replace semantics in the actual native Incus file interface. Where correct expected-version enforcement cannot be guaranteed under concurrency, use verified per-file exclusive locks plus point-of-use native checks; if the guarantee is still unavailable, return `UNSUPPORTED` rather than claiming compare-and-swap. Never use the parent host filesystem as an alternate mutation path.

The `box.file.write` scope covers `create`, `replace` and `patch`; `delete` also requires explicit `box.file.delete` permission (or a stated administrator-equivalent grant). Applying a patch or deleting a file is separately journaled and independently verified, with an operation and Tracking ID. Read results include file size, observed SHA-256 and `hash_verified`; pagination must not claim a whole-file hash from a partial or non-snapshot-consistent read.

Guest-root Exec can itself create, replace and delete guest files. Therefore `box.file.delete` is a **tool-specific control, not a sandbox restriction on a token that also has `box.exec`**.

## `box_exec`: typed, bounded, async

- Accept `box_id`, `argv` (non-empty array of argument strings; executable first), optional safe `cwd` inside the guest, explicit guest UID/GID overrides (default guest UID 0), allowlisted bounded environment values, `timeout_ms`, and output budgets. No raw arbitrary Incus config or caller-chosen host executable.
- A shell is not implicitly added or used to interpolate arguments. A caller may explicitly run a guest shell if authorized, e.g. `["/bin/sh","-lc","go test ./..."]` **inside the owned guest**. This is powerful guest-root authority: do not misrepresent it as read-only.
- A successfully accepted Exec produces an operation ID and Tracking ID immediately. Its output is observed through `operation_output_get`, and the exact exit/terminal evidence through `operation_get`; bounded output may be truncated or expire.
- Observed exit code `!= 0` is a completed invocation whose command failed, not `UNKNOWN`. A lost exit, interrupted native Exec stream, guest stop, canceled client or worker crash may yield `UNKNOWN` until evidence is found.
- Exec may initiate guest filesystem modifications or outbound network traffic. Isolate the guest with Incus-native controls; no Token permission combination can claim guest write protection while unrestricted guest-root Exec remains available.

## `operation_get`: bounded wait

- `operation_id` is required. `wait_ms` defaults to 0 and must be in the range 0–20000 milliseconds. A positive wait is an authorized, cancellable **read-only long-poll**, not a mutation.
- Return on a relevant state/phase change or when the wait ends, with the newest verified state and the same original Tracking ID and operation ID. A timed-out long poll is **not** an operation timeout or a cancellation.
- Client disconnect cancels only this waiting HTTP/MCP handler, not an admitted durable operation. Limit concurrent waiters; test MCP Streamable HTTP and optional Tunnel/proxy timeouts.

## `operation_cancel`: proof-based cancellation request

- Input: target `operation_id` plus mandatory caller-generated reusable `idempotency_key`. The server validates the original actor/administrator authority, exact owned Box/resource, original operation identity and a dedicated `operation.cancel` class **plus** applicable target-operation permission. A token that may read status is not thereby authorized to cancel.
- Every accepted cancellation attempt records its own auditable cancellation intent, `tracking_id` and receipt; return `target_operation_id` and separate `cancel_operation_id`. Do **not** overwrite the target's original operation or replay it.
- Before dispatch: a transactionally proven queued/undispatched operation can be atomically marked `CANCELLED_BEFORE_DISPATCH`; verify no native effect was sent. A mere queue time estimate or missing native receipt is **not** proof of non-dispatch.
- After dispatch: try termination **only** for a native operation type with a verified cancellation mechanism (initially guest Exec, capability-gated). An accepted request means `CANCEL_REQUESTED`, not `CANCELLED`. Confirm termination separately; report partial effects, exit or lost evidence honestly. A killed test process may already have modified guest files.
- Non-cancellable operation classes (e.g. dispatched Box create/delete or file write without proven native cancellation) return a typed `CANCEL_UNSUPPORTED` or record a non-effective request; never imply rollback or safe retry. `UNKNOWN` originals retain their locks/claims until their actual outcome is resolved.
- Deduplicate cancellation requests and conflicts under resource locks. Concurrent completion, second cancel, worker restart, token revocation and parent Box stop must never yield a false cancellation-success claim.

## `box_update`: explicitly allowlisted fields only

- Supported P0 input fields are typed CPU, memory, **disk growth only**, opt-in nesting and per-Box internet access; include expected Box UUID/incarnation and revision/concurrency checks.
- Restrict any required native NIC or bridge changes to internally derived, BoxWeave-owned configurations. No arbitrary Incus config key, raw NIC/ACL map, host disk path, host mount, privileged device, Caddy payload, external upstream or host command.
- Report capability `UNSUPPORTED` rather than silently applying unverified network/storage driver changes; independently inspect resulting native state. A requested disk size lower than current observed size always fails before dispatch.

## `box_file_read` and diagnostics

- Return byte/size/encoding/offset/EOF and an observed SHA-256 with `hash_verified` or `not_verified` coverage. A full-file hash must be calculated from a consistent/native-verifiable file revision; do not attach a confident hash to unrelated page snapshots.
- Reads, long-polling, output pages and diagnostic lookup must recheck both token permission and exact owned target identity, not merely accept possession of a Tracking ID/operation ID.
- `diagnostic_get` remains separate from `operation_get`: reads and pre-dispatch refusals can be correlated without an admitted business operation. Sensitive log text and raw guest data stay private.

## Revocation and site transport constraints

- A revoked Token cannot start a new mutation or dispatch an admitted but **proven undispatched** queued mutation; the worker rechecks revocation and records an explicit blocked state before native dispatch. Already dispatched or UNKNOWN tasks retain the original outcome and are not automatically terminated by revocation.
- `http_route_create` in P0 publishes only **HTTP** applications. The managed Caddy must never automatically issue certificates, activate TLS/port 443 or redirect HTTP to HTTPS. An optional operator-managed external HTTPS proxy is not a BoxWeave MCP-controlled site resource.
- New Boxes use fail-closed private isolation for IPv4 and any enabled IPv6. If owned IPv6 cannot be safely denied, it may be disabled only for managed network resources; otherwise refuse creation/start instead of silently weakening private isolation.

## Required API/permission acceptance

- Independent native tests for each of the four file modes, racing hash changes, symlink/hardlink/host-path refusal, cancellation before dispatch and cancellation attempted during running Exec.
- Verify guest-root `box.exec` token really can modify **guest** files despite absence of file-write scope; ensure it still cannot alter parent/unowned resources.
- Verify `operation_get(wait_ms)` returns on state change, times out cleanly without canceling the operation, and remains proxy-compatible.
- Verify token with only `operation.read` cannot cancel; canceled worker vs original UNKNOWN and original native side effect are reported separately.
- Verify authorization on file `delete`, scoped operation output/diagnostic reads, and native post-state checks.

See [MCP catalog](MCP_TOOLS.md), [security](SECURITY.md), [operations](OPERATIONS.md), [Tracking/context](TRACKING_CONTEXT.md), and [open risks](../verification/OPEN_RISKS.md).
