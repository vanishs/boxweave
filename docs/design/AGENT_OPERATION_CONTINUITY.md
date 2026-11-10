# Operation attribution and continuity — one domain contract, no workflow engine

**Status: proposed S2 implementation contract / owner-requested planning; not yet accepted code.** Purpose: support ordinary single-Agent use, **multiple Agents on the same controller**, and a new Agent continuing a work item safely after a Tool/client crash. Preserve the original 20 MCP Tools, SDK-native stateless `/mcp`, SQLite/Incus owner boundary and narrow 2-role authentication. Evidence and constraints: [Go SDK review](../research/MCP_GO_SDK_CONTINUITY_REVIEW_2026-11-10.md), [workflow frameworks](../research/AGENT_CONTINUITY_FRAMEWORKS_2026-11-10.md), [original Operations](OPERATIONS.md), [Tracking](TRACKING_CONTEXT.md), [future MCP Tasks adapter](../research/MCP_TASKS_EXTENSION_SEAM_2026-11-10.md).

## Identity and data ownership (immutable boundary)

| ID | Owner / meaning | Security and lifespan |
|---|---|---|
| `actor_id` and `token_id` | **Server-authenticated principal** from v1 `req.Extra.TokenInfo` (currently credential/token ID; verify actual mapping). | Only trusted per-request authorization source, distinct from any caller label. Rotated Token has new ID unless v1 later explicitly models a stable owner. Do not silently equate clients with human identities. |
| `work_ref` | **Optional client-supplied bounded opaque label**, e.g. `project-x/test-cycle`. | Groups related operations across multiple Agents, controller-local only. Not unique, secret, access control, scheduled job or guarantee of common ownership. |
| `agent_ref` | **Optional client-supplied bounded opaque execution label**, e.g. `agent-007`. | Descriptive/audit grouping only; callers can spoof it. Do not present it as a verified identity. Separate token per Agent if dependable attribution is needed. |
| `request_id` | One MCP transport attempt. | New on retries, log safe method/status/duration; never user authority. |
| `tracking_id` | One logical request and original operation evidence. | Reused when idempotent duplicate returns original record. |
| `operation_id` | One durable BoxWeave mutation receipt. | Stable authoritative handle across restarts and cross-client reconnects within authorization; may be UNKNOWN. |
| `native_operation_id` | Upstream Incus Operation handle. | Stored as evidence when available, may expire; always verify actual exact UUID/native postcondition. |
| `taskId` | **Future** MCP Tasks extension protocol handle. | Not implemented; separate TTL/access control from durable original Operation. |

**No new identity service.** Explicitly don't infer a reliable Agent identity from MCP `clientInfo` (identifies the software), Session ID, IP, HTTP header or self-declared `_meta`. Multi-Agent attribution can use distinct existing revocable tokens without introducing Agent registration.

## Safe correlation contract

- Prefer a **namespaced optional request `_meta`** on modern SDK clients, but treat it as untrusted. On clients that cannot supply custom `_meta`, *allow* optional `work_ref` and `agent_ref` as arguments on existing mutation tools. Use exactly **one normalized correlation** for the request; if both are present and disagree, **reject pre-admission** with typed invalid input and no effect, rather than guess. Absent is valid and backward compatible. Confirm the tagged SDK accessor/API shape in the S2 implementation, not custom JSON-RPC parsing.
- Proposed validation: UTF-8 ASCII-safe `[A-Za-z0-9._/-]`, 1–96 bytes per field, no `..` path meaning, colon/protocol keywords, whitespace, control chars, raw URLs, secrets or tokens. Use as **opaque identifiers**, not filesystem names, path join, access tokens or SQL fragments. Never store the whole client `_meta` blob. Cap/canonicalize once at ingress, identical across tool types. The syntax/size is a **proposed implementation default**, adjust in S2 only with an explicit migration-compatible test.
- The original **accepted mutation** persists those two labels (or NULL) once at admission with authenticated actor, layer, original idempotency key digest, immutable intent, target and timestamps. A retry of the original key with identical intent returns **original IDs and original labels**, even if later caller-supplied labels differ. For predictability, **reject contradictory later attribution as conflict** rather than silently relabel history; never re-dispatch. A second caller with a *different actor* cannot inherit the first caller's idempotency scope by copying a label or key.
- Keep **read-only** Tool calls lightweight: request ID and correlation in protected local redacted audit if useful, but do **not** invent `operation_id` or persist every `operation_get` poll as a new Operation. Protect SQLITE under I/O stress; optional bounded metrics rather than unbounded read-event writes.
- **Schema migration:** S1 uses `SchemaVersion=1`, validates exact DDL/quick_check/SQLite history and refuses unknown tables/migrations. S2 must introduce a reviewed increment, transactional migration with backup/rollback/partial-failure recovery and documented schema fingerprint; add two indexed nullable columns only when necessary. Never edit v1 database by deleting/recreating it. Test from exact S1 schema with existing ADMITTED/UNKNOWN and recovery audit records, malformed schema, interrupted migration, concurrent openers, no foreign DB adoption.
- Privacy/retention: only safe short identifiers, the existing bounded operation phase/summary, no raw arguments/env/guest output/prompt/system instructions/private host info. Redaction before local audit and before Tool result. Use cursor pages/limits; no unbounded full history scan as a remote Tool.

## Query/visibility semantics: reuse existing `operation_list`

- Extend the planned **one** `operation_list` (read) with optional filters `work_ref`, `agent_ref`, `actor_id` (authorized administrative use), action, state, time range and existing opaque cursor/page limit. **No `agent_list` or `task_list` Tool.** Implement efficient layer+actor/status/time/work indexes when supported by measured queries; stable `created_at,operation_id` ordering and cursor scopes binding filter/actor/layer prevents pagination leaks when data changes.
- **Authorization order:** verify token, role and owned controller/layer first; choose permitted record set, then apply self-declared filters. Existing `admin` may see **only this layer's owned records** under the approved 2-role model; `viewer` stays within its existing read-visibility contract, and sensitive Exec output remains protected by existing output scope and retention. An `agent_ref` or shared `work_ref` never authorizes sharing across actors/layers/projects. If product policy does not allow an actor to see another's output, return denial/redaction even if `operation_list` shows safe metadata.
- **Two truth types:** `operation_list` is a *search index of relevant records*, not a complete Agent task progress report; `operation_get(original_id)` and native exact readback are current operation truth. A no-results response may mean tag absent/old records/retention/visibility, **never** “Agent finished” or “native create did not happen”. Viewer/admin refusal must not disclose hidden IDs through counts or cursors.

## Typed response and safe next-step hints

An S2 admission response should convey a **small SDK-typed structured Tool result**, for example **illustrative only**:

```json
{
  "tracking_id": "opaque-original-tracking",
  "operation_id": "opaque-original-operation",
  "state": "RUNNING",
  "work_ref": "project-x/test-cycle",
  "agent_ref": "agent-007",
  "evidence": { "native_operation_recorded": true, "postcondition_verified": false },
  "next_action": {
    "kind": "READ_OPERATION",
    "tool": "operation_get",
    "operation_id": "opaque-original-operation",
    "wait_ms": 0
  }
}
```

Do not invent progress percentages or auto-link to guest logs. A failed/denied *pre-admission* call gets safe `tracking_id`, typed reason and **no** fake Operation. Terminal receipt should distinguish transport/request success, BoxWeave verified state and guest command `exit_code`/`command_success`. `UNKNOWN` next action is **read original / host-local manual review** only; no “retry box_create/delete/Exec” hint. Keep optional `next_action` deterministic finite vocabulary (`READ_OPERATION`, `READ_OUTPUT`, `INSPECT_OWNED_BOX`, `HOST_LOCAL_REVIEW`, `NONE`), no arbitrary call descriptors/URLs/text commands. Responses should be small, bounded, status-scoped; protocol clients without rich UI still receive clear text fallback, with no secret or 20 Tool contract change.

## MCP features mapping

- **Middleware:** SDK receiving middleware observes `method/tool`, actor, safe normalized labels, request ID, duration, error class and *existing* receipt ID. Handler/database carries authoritative admitted IDs; middleware cannot fabricate successful native results. No generic event bus or second tracing database.
- **`_meta`:** client-provided correlation namespaced from framework's own reserved keys; absence is normal, legacy clients use optional existing arguments; do not require custom metadata to complete a Box operation.
- **Progress:** SDK `NotifyProgress` only with supplied token for a live in-flight short read or a meaningful admission step, and only if tested client renders it; underlying execution persists separately. Default no perpetual HTTP call or duplicate heartbeat daemon.
- **Annotations + caching (extra low-cost leverage):** classify read-only vs potentially destructive truthfully, cache private authenticated views as `private` with minimal TTL, avoid `IdempotentHint` for mutations unless same-arguments repeat safety is genuinely established. Annotation is a hint, never authorization. New SDK protocol `2026-07-28` lacks old ping/KeepAlive; no fake liveness.
- **MCP Tasks:** stay with existing `CallToolResult` and one durable Operation journal; the [separate future adapter](../research/MCP_TASKS_EXTENSION_SEAM_2026-11-10.md) can eventually project a protocol task handle **only after pinned Go SDK and real client support**. Checkpoint `work_ref` is not MCP `taskId`.

## Task Checkpoint belongs to the Agent/workflow, not BoxWeave

**Lightweight, optional [checkpoint template](../development/TASK_CHECKPOINT_CONTRACT.md)** is a client-owned file/workflow artifact: objective, last Agent attempt, changed Git SHA/dirty files, evidence IDs, previous decisions, next safe action and do-not-repeat keys. The checkpoint must never contain credentials/secret guest content and has a format version and update time. Clients can use project Git or another trusted store; no MCP checkpoint-save/get Tool, inner L1 controller awareness, prompt/memory storage, or scheduler is required.

**Resume protocol:** (1) read checkpoint and original task instructions; (2) use trusted auth to query `operation_list(work_ref)` / exact Operation; (3) independently check Git/native Box/CI state; (4) classify each action as verified DONE, still RUNNING, UNKNOWN or NOT_ADMITTED; (5) only then decide next safe action with original keys and authorization. Missing checkpoint or absent Tool result means **investigate**, not retry. Two simultaneous Agent checkpoint writers must use the external store's revision/Git conflict protection, not BoxWeave's Operation journal as a distributed collaboration lock.

## S2–S5 execution responsibility

| Phase | Required contract completion | Evidence |
|---|---|---|
| **S2** | Single versioned journal migration, safe correlation/auth/query/typed receipt, SDK middleware/`_meta`, truthful tool annotations/cache policy, baseline progress capability if actually useful; original native idempotency/UNKNOWN remains mandatory | [TC-001–TC-010](../verification/CONTINUITY_ACCEPTANCE.md), no S2 PASS without regressions |
| **S2.5 BRIDGEHEAD** | Independent new Agent reconnect and multi-Agent same/different work, actual intended client compatibility, native readback of interrupted operation + checkpoint | TC-011–TC-014 **additional to** BH-001–BH-017; HOST_GO needs all mandatory owner/native gates |
| **S3** | Agent source-change checkpoint handoff, dirty tree/Git reconciliation, real guest file+Exec fail/fix/test in an isolated Box | TC-015–TC-016; S3 file safety unchanged |
| **S4** | Same correlation survives owned forward/peer/delete dependency operation and independent L0/L1 scopes, no cross-layer authority | TC-017–TC-018; all S4 network/IPv6/LAN gates unchanged |
| **S5** | Exact installed 20/20 Tools/CLI, old/new SDK clients and real intended Agent, two-actor churn, dropped request and independent verifier red/green test calibration | TC-019–TC-020 **plus** all S5 A-/SDK-G gates |

**Do not** publish this as if code exists yet. This is the planned narrow P0 implementation contract with its own separately tracked native and client failure evidence.
