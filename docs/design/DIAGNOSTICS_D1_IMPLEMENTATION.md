# D1 implementation specification — durable, privacy-safe diagnostic evidence

Status: **implementation-ready architecture specification, not implemented**. Source review baseline `main@817273d98d8823fcd8489eb0d3f329a67929b91d` / BoxWeave 0.21.4. Depends on [D0](DIAGNOSTICS_D0_IMPLEMENTATION.md). Enables [D2](DIAGNOSTICS_D2_IMPLEMENTATION.md). Parent [proposal](DIAGNOSTICS_PROPOSAL.md), [source audit](../verification/DIAGNOSTICS_SOURCE_AUDIT.md). Implementation needs a new owner decision; no deployment or privileged host action is implied.

## 0. Objective and decision boundaries

When BoxWeave itself fails or behaves unexpectedly, a maintainer should be able to answer:
- **what** accepted operation was attempted; its expected target/revision and result;
- **which** app component and phase observed success, rejection, timeout or uncertain outcome;
- **when** each observation was durably recorded and where capture stopped;
- **which** installed source/configuration generation and trust boundary was actually observed;
- **what cannot be concluded** because evidence was absent, delayed, dropped, or from a different layer.

D1 is an internal evidence pipeline, **not** an MCP diagnostic tool or support-bundle exporter. No raw stdout/stderr, arbitrary user-project logs, full exception messages or command arguments may be collected. Do not change API/Provider semantics, credential access, Web authorization, or existing once-only dispatch/recovery safeguards.

The application-side MCP client and direct SSH MCP gate are different implementations and often run on different hosts. D1 can deliver a complete app-side vertical slice before a gate-side capture mechanism exists. Never claim a distributed trace spans an uninstrumented Provider or gate.

## 1. Source inventory and ownership

| Module | Current observed contract | D1 insertion point |
| --- | --- | --- |
| `store.py` | private SQLite, FULL sync, DB `user_version` accepts 0–8, synchronous single-writer, separate schemas/migrations | additive version **9**, trace linkage and event rows |
| `service.py` | `confirm` inserts `operations` in one transaction; `work_one` claims before external effect; `recover` marks lost worker unknown; `reconcile` observes but never replays | trace link at confirmed intent; classified worker/reconcile events |
| `lifecycle.py` | `environment_operations`/`environment_events`, receipt phases, explicit `acknowledged`; `work_one` invokes Provider once | link to exact environment-operation receipt; phase, submission and unknown observations |
| `access.py` | `access_operations`/`access_events`, network/port intent, Guard observation | record authenticated write/ack/uncertainty only; no tenant policy payload |
| `jobs.py` | `job_runs` stores `argv` (potentially sensitive) and output hash; not full output | classified job result only, never copy argv or stdout to diagnostic table |
| `selftest.py` | confirmation hash, live-only create/hello/stop/delete; steps in memory | D2 adds step persistence; D1 provides event primitives only |
| `web.py` | typed Flask routes and safe error responses, generic INTERNAL_ERROR | request-scoped trace after authentication/validation, safe handler event |
| `provider.py` / `access_provider.py` | selected upstream MCP tool contracts and trust checks | boundaries for classified upstream attempt/outcome |
| `mcp.py` | bounded JSON-RPC stdio, strippable errors, `MCP_STREAM_CLOSED` / `MCP_INVALID` / `OUTPUT_LIMIT` / `PROVIDER_REJECTED` | classify private error *shape* without retaining remote text |
| `mcp_gate.py` / `online.py` | root-installed restricted SSH gate; upstream stdio; launcher stderr discarded; gate outside app-owned SQLite | optional D1-gate sub-slice only after host/identity/ownership review |
| `distribution.py` / `online.py` | separate install/upgrade and published public provenance | optional version reference; no recursive install-log collection |

Important discrepancy: `store.py` currently stores `user_version=8`. A new event schema version must be **9** (not an additional copy of an old schema), with downgrade rejection by older binaries unchanged.

## 2. Minimal event contract (diagnostic schema 1)

Use **allowlisted typed data**, not generic key/value or exceptions as JSON. Suggested dataclass / TypedDict in new `bwmini/diagnostics.py`:

```python
@dataclass(frozen=True)
class DiagnosticEvent:
    event_id: str             # random 128-bit opaque hex
    trace_id: str             # independent opaque 128-bit hex
    span_id: str              # random 64/128-bit opaque hex
    parent_span_id: str | None
    operation_kind: str | None  # one of: hello/environment/access/job/selftest
    operation_id: str | None
    actor_scope: str           # recognized operator scope, not arbitrary input
    component: str             # enumerated
    phase: str                 # enumerated
    event_type: str            # accepted/claimed/submitted/acknowledged/observed/rejected/unknown
    outcome: str               # passed/failed/unknown/not_verified
    code: str                  # fixed allowlist from classified errors
    wall_time_utc: str         # ISO8601 UTC for ordering in local report
    elapsed_ms: int | None    # monotonic elapsed for this process only
    capture: str              # captured/truncated/dropped/unavailable
    target_ref: str | None    # bounded safe identity digest, not raw instance contents
```

This is a proposed API shape; prefer strict enums and validation with no freeform `details` string. `event_id`/`trace_id` are unpredictable, not authority tokens; merely knowing one never grants access to reports. Do not reuse JSON-RPC request integer IDs, guessed IP addresses or user-entered project names as cross-host correlation.

**Time:** record wall clock plus local monotonic elapsed only; local event insertion sequence (SQLite rowid) is authoritative for an individual process/store. Separate hosts can have clock skew. No absolute ordering across hosts without verified NTP/clock evidence.

**Codes:** use existing public Fault code where safe (`PROVIDER_REJECTED`, `MCP_INVALID`, `MCP_STREAM_CLOSED`, `TRUST_FAILED`, `DEADLINE_EXCEEDED`, `OUTCOME_UNKNOWN`, `WORKER_LOST`). Optional private `error_class` enum `upstream_jsonrpc_rejection` / `tool_is_error` / `transport_eof` / `timeout` / `malformed_protocol` / `local_state_unavailable` can disambiguate. Do not store unreviewed upstream string, status payload, stack or client process stderr as diagnostic content. E.g. an upstream 404 is not safe to interpret as `POLICY_ABSENT` unless it passes the existing exact contract validation in `mcp.py`.

**Nested identity:** `environment_ref` may record installed source commit SHA and locally observed UUID/creation fingerprint only when trusted evidence already exists. Gate-host/Provider-host/parent-child relationships are a separate explicit evidence classification; do not make a shared trace ID assert a verified layer relationship.

## 3. Proposed SQLite migration and boundaries

Create **two** tables and indexes without altering existing operation table layouts. Recommended schema, to be reconciled with tests before code:

```sql
CREATE TABLE diagnostic_trace_links (
  actor TEXT NOT NULL,
  operation_kind TEXT NOT NULL,
  operation_id TEXT NOT NULL,
  trace_id TEXT NOT NULL,
  created_utc TEXT NOT NULL,
  PRIMARY KEY (actor, operation_kind, operation_id),
  UNIQUE (trace_id)
);
CREATE TABLE diagnostic_events (
  seq INTEGER PRIMARY KEY,
  event_id TEXT NOT NULL UNIQUE,
  trace_id TEXT NOT NULL,
  span_id TEXT NOT NULL,
  parent_span_id TEXT,
  actor_scope TEXT NOT NULL,
  operation_kind TEXT,
  operation_id TEXT,
  time_utc TEXT NOT NULL,
  elapsed_ms INTEGER,
  component TEXT NOT NULL,
  phase TEXT NOT NULL,
  event_type TEXT NOT NULL,
  outcome TEXT NOT NULL,
  code TEXT NOT NULL,
  capture TEXT NOT NULL,
  target_ref TEXT
);
CREATE INDEX diagnostic_events_trace_order
  ON diagnostic_events(trace_id, seq);
CREATE INDEX diagnostic_events_time_order
  ON diagnostic_events(time_utc, seq);
```

- `diagnostic_trace_links` ties exactly **one local durable receipt** (by actor+kind+ID) to a trace. No foreign key to every distinct domain table; reader first checks the authoritative typed operation row and its actor to authorize lookup. A trace link cannot authorize a sibling actor or make a missing operation successful.
- `diagnostic_events` has a total order `seq` within this DB. Use a hard cap on row count and **per-trace** cap, bounded field lengths and strictly whitelisted enums. The event table is **not** a second state machine; authoritative receipt and existing `environment_events`/`access_events` remain truth.
- Migrate `version<9` to these tables and set `PRAGMA user_version=9` in the same migration transaction. All previous 0–8 upgrades still work. Reject version >9 and unexpected existing-table names; do not silently repair malformed state. D1 should not change `SCHEMA` for existing receipt rows. Preserve manifest-backed pre-upgrade backup and tested rollbacks. Old binaries fail on v9 instead of corrupting an unfamiliar database.
- For new *confirmed* operations, generate a trace ID in process and insert its link into the **same** existing transaction that marks the plan used and writes intent. Keep this a minimal deterministic insert validated before any Provider effect. If storage fails, the entire intent transaction fails; **no external dispatch**. This matches the existing atomic safety contract.
- For unbound requests/health checks, emit a standalone short-lived trace; it may never link to an operation and has narrower retention. Never persist an unauthenticated Web request body or login credential.
- For older pre-D1 receipts, report `trace_unavailable_pre_migration` rather than fabricating a past trace. D2 may still export their existing receipts/events with an explicit coverage gap.

## 4. Transaction, execution and failure semantics

### D1-1: transaction-safe operation creation

At `Service.confirm`, `Lifecycle.confirm`, `Access.confirm` (and any other accepted durable operation), in one transaction:
1. verify current plan/revision/actor and use it once;
2. allocate durable operation ID and opaque trace ID;
3. insert existing operation intent and diagnostic trace-link row;
4. commit before any remote request; **only then** allow worker to act.

No dependency on diagnostics should make an already committed operation run a second time. Never insert a diagnostic trace that implies dispatch happened before dispatch actually occurred.

### D1-2: transition recording with **noninterference**

Two categories of evidence:

- **Authoritative business transitions** (`queued`, `dispatching`, `acknowledged`, `unknown`, `failed`, etc) already persisted. They always remain correct even if optional diagnostics fail; derive those facts in reports directly from original tables.
- **Diagnostic observations** (`upstream error class`, local duration, message-shape category) are optional. Write them through `safe_emit` **outside** the business transaction and outside remote I/O; a failure to insert them must not block or retry a user operation.

Critically, avoid catching a SQLite error *inside* a transaction then committing a half-failed transaction. `safe_emit` should open/commit its own bounded transaction; failure yields a local capture gap only where reliably writable. During DB-full, no second DB write can reliably record `dropped`; the report must infer `capture_unknown` or missing data from sequence, not invent evidence. Do not log event contents to stdout/stderr as fallback (those streams may be MCP protocol streams).

Required sequence for externally mutating worker action:

```text
transaction: validate + mark dispatching -> commit
(optional) emit "dispatch_intent_observed"
perform exactly one Provider action (no DB transaction held)
on known successful response: persist acknowledged/observing/succeeded
on confirmed rejection: persist failed where existing semantics allow
on ambiguous transport/timeout/crash: persist or recover unknown
(optional) emit classified elapsed/result observation
```

**Never let event recording change error classification**. In particular, an uncertain remote effect cannot be downgraded into failed or retried. Do not automatically reconstruct missing "completed" events on startup; existing receipt/state is observed truth.

### D1-3: cross-process context

Use `contextvars.ContextVar` for Web synchronous request-level trace context, but **do not rely on it** for the separately running worker or after restart. Resolve worker trace from `diagnostic_trace_links` by (actor, operation kind, operation ID). For a directly invoked `Provider` call without receipt, create a new local trace and mark as `unlinked_app_mcp`. Propagate trace/spans only through trusted in-process explicit parameters; do not add nonstandard keys to upstream `tools/call` arguments or SSH argv.

The source-created trace IDs are opaque identifiers for correlation, not a promise of server-to-Provider distributed tracing. A later D3 design must separately handle direct Agent gate visibility and access control. The gate must never read app-owned SQLite by assumption.

## 5. Classification at critical code paths

| Hook | Safe fields | Explicitly forbidden fields |
| --- | --- | --- |
| Web accepted typed request and safe error | controlled route ID, HTTP class, resulting receipt ID if committed | URL query values, body, cookies, sessions, password, raw traceback |
| Worker `work_one` claim | operation kind/ID, state transition, local time | stored project path, full spec, credentials |
| Application MCP `initialize` / `tools/list` / `tools/call` | method enum, tool name from allowlist, timeout/exit class, valid/invalid protocol classification | JSON-RPC arguments, response text, token/SSH argv |
| `Provider` lifecycle/hello | resource reference digest, phase, existing public error code, outcome certainty | hostname/IP alone as identity, command or stdout |
| Access/Guard | phase, revision observed (bounded int), state/verification class | policy document, entire Tenant network map |
| Jobs | operation ID, exit class, output **hash/byte count already in authoritative receipt**, not copied | `job_runs.argv`, stdout/stderr contents |
| `selftest` | not until D2 step persistence | auto cleanup inference |

`bwmini/mcp.py` has separate JSON-RPC `error` and tool-result `isError` paths, which both currently map to generic `PROVIDER_REJECTED`. D1 should classify those two internal shapes distinctly while retaining the same public Fault code. If malformed/error object is non-dict, mark `malformed_protocol` per existing behavior instead of doing unchecked `.get`. Do not include error message fragments in an event.

For unexpected `except Exception` in `service.py`, `lifecycle.py` and `web.py`, record a fixed class such as `unclassified_exception` (or a bounded, approved exception-type enum), not `str(error)` or repr/stack. Even exception *types* should be allowlisted to avoid dynamically including attacker-controlled names.

## 6. Gate-side event capture: conditional D1-gate

The direct Agent path uses `mcp_gate.py` running under dedicated restricted account; it forwards upstream stdout and drops stderr. Instrumenting it to log into the application database is **incorrect**. Before enabling a persistent gate journal:

- inspect the installed file/UID ownership and actual split gate-host versus Provider-host topology in a disposable approved lab; RISK-620 shows they can differ;
- use one fixed, allowlisted per-host sink owned by the gate identity, bounded byte count, private permissions and no stdout writes (stdout is JSON-RPC);
- do not log caller public keys, SSH environment, raw tool results, arbitrary client command args or gate process exceptions;
- distinguish locally confirmed `forwarded` from upstream acknowledged, and never forge a remote completion;
- retain unlinked gate traces with `source=gate_only` until a *trusted* mapping exists.

Gate logging can be **deferred** while D1 app-side evidence ships; D2 will explicitly show `gate_capture=not_installed` rather than treating it as a missing bug. No new independent daemon or Debug MCP is required.

## 7. Capacity, retention, privacy and quotas

Candidate initial **review values**, not release claims: 7-day retention, max 20,000 events / installation, 64 events / trace, 512 bytes maximum per encoded event field group, and 10 MiB total diagnostic SQLite content budget. The maintainer must approve actual limits and decide behavior of long-lived operations before implementation.

Implementation guidance:
- enforce hard caps **before** SQLite insert; no arbitrary truncated user bytes stored;
- prune only owned diagnostics in bounded batches; never delete business receipts or pending operation trace links required for reconciliation;
- avoid global log sampling that silently loses unknown-phase evidence; if pressure requires dropped entries, expose aggregate `capture_limited` with evidence of quota decisions;
- never rotate or vacuum synchronously in a mutating remote action; prune on bounded maintenance/read path with careful SQLite lock timing;
- maintain 0600 DB, 0700 directory and test symlink/hardlink refusal; do not turn on network exposure or remote telemetry;
- use no external logging service or subscription. Avoid changing SQLite journal mode solely for diagnostics without measured contention.

**Security gate:** synthetic secret/cookie/JWT/public and private key strings, raw JSON payloads, `ToolExecutionError` text, shell control bytes, and prompt injection text must never be findable anywhere in the diagnostic table or its debug result. A generic regex scrubber is insufficient; use field-only generation. No diagnostic event should include executable instructions derived from arbitrary logs.

## 8. Developer implementation order

1. `diagnostics.py`: immutable event/trace types, generated opaque IDs, fixed enums, validation, minimal error classifier. Unit test before instrumentation.
2. `store.py`: schema v9 + migrations and query helpers `trace_for_operation` / `events_for_trace` with actor/kind scoping. Atomic intent-link integration, migration and rollback tests.
3. `service.py`: initial hello/project work and recovered unknown events; use authoritative receipt state, no remote replays.
4. `lifecycle.py` and `access.py`: map queue/dispatch/ack and observations to events; ensure existing event tables remain the operational authority.
5. `mcp.py` / `provider.py` / `access_provider.py`: classify error paths privately; public error unchanged; measure local monotonic duration.
6. `web.py`: safe ingress and error phase handling, trace association to accepted operations without changing existing JSON API contracts.
7. Optional D1-gate only after independent authority decision and approved host validation. No D2 report, D3 tool or general project log reader in this phase.

Touch `distribution.py` release packaging only if new module has to be included; current `distribution.build` has an explicit embedded module list. Packaging a Python module without updating this list would produce a development-only feature that is missing from installed releases. Similarly ensure `online.py` publishes any approved root-owned gate helper only after SHA/policy review.

## 9. Tests and fault injection

New `tests/test_diagnostics.py` and integration tests in `test_recovery.py`, `test_lifecycle.py`, `test_access.py`, `test_slice.py`, `test_distribution.py`.

| ID | Scenario | Acceptance |
| --- | --- | --- |
| D1-T01 | Fresh v0–8 DB upgrades | v9 created, all original receipts/deadlines/admin/graph preserved |
| D1-T02 | Pre-upgrade backup + old binary | old binary rejects unsupported v9; no downgrade corruption |
| D1-T03 | Confirm plan and trace-link transaction forced DB FULL | neither intent nor plan-use committed; zero remote effect |
| D1-T04 | Repeated confirm or same op ID across kinds | one-use preserved; no ambiguous trace association |
| D1-T05 | Worker terminated before dispatch | correct authoritative queued/unknown; missing event not fabricated |
| D1-T06 | Worker killed after remotely executing before response | unknown preserved; no second Provider mutation, trace partial |
| D1-T07 | Diagnostic writer fails/times out under lock | operation semantics unchanged, capture gap explicit when observable |
| D1-T08 | Upstream JSON-RPC rejection vs tool isError | different private categories; public provider code unchanged |
| D1-T09 | Malformed/oversized MCP response, stream close, timeout | existing Fault semantics and bounded data retained |
| D1-T10 | SSH/Provider stderr contains synthetic token, key, prompt | no diagnostics table leak, no stdout protocol leak |
| D1-T11 | Two actors and two operation kinds | exact actor/kind/ID query isolation |
| D1-T12 | Restart worker and update release/config generation | trace remains local; new process does not counterfeit old spans |
| D1-T13 | Retention limit reached while operation pending | business receipt intact; quota/capture gaps reported |
| D1-T14 | Pre-D1 old operation with no trace link | explicit `trace_unavailable_pre_migration`, still readable receipt |
| D1-T15 | Real package archive/source tests | diagnostics module included and installed code version correct |
| D1-T16 | Selftest/job outputs have synthetic secrets | no copied argv, stdout or stderr in diagnostic events |
| D1-T17 | Concurrent writer/SQLite lock contention | no user-operation deadlock, no unbounded busy waits |

Actual command lane in prepared non-root environment: `python scripts/check_knowledge.py`; `.venv/bin/python -m unittest -v tests.test_diagnostics tests.test_recovery tests.test_lifecycle tests.test_access`; `.venv/bin/python scripts/validate.py`. Build/package fixture must prove new modules are shipped. No actual host, private DB, account enrollment, Provider writes or nested Box test without separately approved environment scope. Record command, commit SHA, exit, fixture/live class and reproducible synthetic evidence.

## 10. Release gates, fallback and handoff to D2

**D1 is done** when the installed application can associate a fresh durable receipt with an opaque trace and classify its phase/error using strictly typed stored evidence, while all existing recovery, authorization and receipt tests pass. Source-only success never implies the installed gate has trace collection. Document exactly which components have `capture=not_installed` or `not_verified`.

Do not create or advertise `boxweave report` until [D2](DIAGNOSTICS_D2_IMPLEMENTATION.md) defines safe export and has passing tests. A runtime rollback requires the normal reviewed state backup/migration path; do not downgrade an upgraded v9 DB by running the old binary against it. Preserve existing unknown operations, original evidence and old release before repair. D3 direct MCP lookups require their own authorizations and are **not** a D1 acceptance shortcut.
