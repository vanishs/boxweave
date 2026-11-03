# Unified operation trace and diagnostics — API-first evolution design

> **Status: candidate future design only. Not implemented, deployed, or authorized by this document.**
> Parent: [API-first architecture](API_FIRST_CONTROL_PLANE_EVOLUTION_MASTER.md). Migration phases: [API Provider plan](API_FIRST_PROVIDER_MIGRATION_PLAN.md).
> Current decisions and safeguards remain authoritative: [D3/D4 master](DIAGNOSTICS_D3_D4_MASTER.md), [D1](DIAGNOSTICS_D1_IMPLEMENTATION.md), [D2](DIAGNOSTICS_D2_IMPLEMENTATION.md), [R-02](MCP_TASK_RELIABILITY_R1.md), [R-04](MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md).

## 1. Primary outcome

Move from partly independent **app trace/receipts** and **gateway MCP event journal** to one **protocol-neutral operational evidence vocabulary**, with correlated records at every BoxWeave-owned boundary:

```text
Agent MCP / Web / CLI entry
    -> authentication + policy
    -> typed Operation Core (intent, claim, actor, target incarnation)
    -> Containarium REST API and/or Box SSH Executor
    -> independent observed Provider / Box state
    -> authoritative operation receipt and bounded diagnostic report
```

**What API-first improves:** request method/route, HTTP status, structured response/code, time to first response, transport failure, API acknowledgement and follow-up observation can be classified *separately* instead of being obscured behind MCP JSON-RPC and upstream tool text. It also avoids a separate platform MCP start/initialize/tools/list on *normal app-side* calls.

**What it does not improve automatically:** Containarium daemon-internal Incus/ZFS details, post-acceptance convergence, SSH correctness, source-of-truth authorization, and whether the daemon received bytes before the connection was lost. Those need explicit upstream observability and reconciliation. An HTTP 200 is **not** proof that the Box is ready.

Do not increase verbosity indiscriminately. The goal is a shorter reliable error chain, clear **coverage/omissions**, correct next step, and less accidental disclosure.

## 2. Existing source and honest limits

| Layer | Existing code/evidence | Preserve / change |
| --- | --- | --- |
| D1 typed diagnostics | `bwmini/diagnostics.py`: trace, span, event, actor scope, operation ref, component, phase, result, capture status; bounded SQLite persistence | Reuse concepts and business receipt linkage; add independent `provider_api`, `box_executor` components and transport-neutral codes |
| D2 read-only report | `diagnostics.py` selects allowlisted receipt fields and refuses free-form payload, target, secret or document content | Preserve private reader, actor scope, omission manifest and bounded read |
| D3 direct-Agent gateway | `bwmini/gate_diagnostics.py`: private SQLite journal with `trace/tool/outcome/code/elapsed_ms`, default 7 days/50 MiB; `mcp_gate.py` records tool outcomes | Retain inbound Agent transport evidence; instrument common core instead of losing visibility when upstream MCP relay is removed |
| D3 minimum upstream MCP compat | `bwmini/mcp_compat.py` and `boxweave_diagnose(source=compat)` compare selected `tools/list` signatures on demand | Replace *outbound platform MCP* part with REST/proto/runtime capability reports; keep inbound BoxWeave MCP protocol/schema tests and fallback-lane checks |
| D4 exact Box diagnosis | `boxweave_project_diagnose`, fixed process/port/service/log checks; root-owned exact-Box allowlist, summary-only and default deny | Keep authorization and fixed probes; use the validated Box Executor instead of direct `connect(exec)` |
| Host fallback | `host_cli.py` private, bounded host/service report | Must remain available when API, Agent MCP or Web are unavailable; never require working MCP to diagnose dead MCP |

The D3 existing `classify()` reports some upstream tool error envelopes as `failed`, while an ambiguous dispatched write can remain `UNKNOWN`. A new evidence model must not treat a successful response envelope or non-2xx HTTP response as proof of *application effect*. Receipt truth overrides speculative classifications.

## 3. Three related identifiers: never collapse them

1. **`trace_id`**: opaque, server-generated correlation identifier for one *logical request or operation*. Shared across app-visible events if supported by the same trusted BoxWeave installation. A caller may provide a previously issued lookup reference for *authorized read*, not mint privileged trace authority.
2. **`operation_id`**: durable immutable identifier for the accepted intent (and any separate transport request key). Can outlive connection, service restart and log-retention window. A successful read-only call may have a trace but no durable operation.
3. **`span_id`/`parent_span_id`**: optional diagnostic substeps within the trace; not a task ID or privilege proof. Where app and gateway process principals cannot write one store, use a trust-checked trace linkage record instead of copying raw events into a shared writable file.

A Box identity reference is a **non-secret digest of provider/backend/project/logical/incarnation**, never merely a Box name or reusable IP. Avoid exposing the full target registry in public Agent-readable diagnostics.

The trace identity is *not* an upstream distributed trace by default. Propagate `X-Request-ID`/W3C trace headers to the Containarium API only after verifying upstream behavior and rejecting user-controlled untrusted tracing IDs. If upstream ignores the header, report `upstream_trace_correlation: unavailable`. Never claim end-to-end daemon trace coverage from a BoxWeave-generated ID alone.

## 4. Proposed minimal event contract

Conceptual example (no live system currently emits this schema):

```json
{
  "schema_version": 2,
  "trace_id": "a8f32c91000000000000000000000000",
  "span_id": "cd438921ff112233",
  "parent_span_id": null,
  "operation_id": "opaque-operation-id",
  "time_utc": "2026-11-03T00:00:00.000Z",
  "actor_scope": "operator",
  "entry_transport": "mcp_http",
  "operation_kind": "create_box",
  "component": "provider_api",
  "phase": "api_response",
  "event_type": "acknowledged",
  "outcome": "not_verified",
  "code": "API_ACKNOWLEDGED_PENDING_OBSERVATION",
  "http_status": 200,
  "elapsed_ms": 81,
  "target_ref": "nonsecret-identity-digest",
  "capture": "captured",
  "evidence_level": "response_only",
  "upstream_request_id": null
}
```

Allowed values and fields should be versioned and **allowlisted**. This is a schema sketch, **not** a request to start logging every JSON response. The actual deployment can keep a narrow SQLite table with optional fields rather than adopting a full OpenTelemetry storage stack.

Suggested component vocabulary: `mcp_inbound`, `web`, `cli`, `auth`, `operation_core`, `provider_api`, `box_executor`, `guest_probe`, `worker`, `observer`, `diagnostic_reader`. Suggested evidence levels: `local_validation`, `request_sent_unconfirmed`, `api_response`, `provider_observed`, `guest_observed`, `independent_live_check`, `not_available`.

Do not expose HTTP request body, response body, URL query with secrets, Authorization, JWT claims, SSH key material, age identities, guest stdout/stderr, filenames, user prompts or raw exception strings in this default event model.

## 5. Operation state and error taxonomy

### 5.1 Operation truth versus network response

| Observed point | Durable operation state | Example public classification | Do not claim |
| --- | --- | --- | --- |
| Refused by local validator/policy before dispatch | `failed_before_dispatch` | `VALIDATION_DENIED`/`POLICY_DENIED` | That daemon refused it |
| No request bytes possibly left and local connection failed | `failed_before_dispatch` *only when provable* | `API_CONNECT_FAILED_PRE_SEND` | That a previous request did not execute |
| Request potentially sent, connection dropped/timeout | **`unknown`** | `API_DELIVERY_OR_OUTCOME_UNKNOWN` | Safe to replay mutation |
| API responded with explicit authenticated typed rejection | `rejected` for that request; operation effect may still need scrutiny | `API_AUTH_DENIED`, `API_VALIDATION_REJECTED` | No side effects unless API contract guarantees it |
| API returned success/accepted | `acknowledged` or `observing` | `API_ACKNOWLEDGED_PENDING_OBSERVATION` | Guest ready, route reachable, backup verified |
| Subsequent verified read matches exact intended state | `succeeded` | `PROVIDER_STATE_CONFIRMED` | SSH/packet/function works unless separately tested |
| Verified state contradicts intended effect | `failed` or `partial` with evidence | `POSTCONDITION_MISMATCH` | No other side effects |
| Guest command exit nonzero observed from authorized exact Box | `failed` (for the exec action) | `GUEST_EXIT_NONZERO` | Daemon mutation failed |

Do not reduce `failed`, `partial`, `unknown`, `unsupported`, `not_verified` to a single status. Distinguish write acknowledgement and product-level validation. An API error may give a *safe structured* reason, but the authenticated daemon contract needs to be inspected before treating it as no-side-effect evidence.

### 5.2 Candidate typed error families

- **Inbound**: `MCP_INITIALIZE_FAILED`, `MCP_SCHEMA_UNSUPPORTED`, `MCP_TOOL_NOT_ALLOWED`, `OAUTH_TOKEN_EXPIRED`, `WEB_SESSION_DENIED`.
- **Policy/identity**: `BOX_IDENTITY_CHANGED`, `TARGET_NOT_AUTHORIZED`, `RESOURCE_CLAIM_CONFLICT`, `DIAGNOSTIC_SCOPE_DENIED`.
- **Provider API**: `API_UNREACHABLE`, `API_TLS_TRUST_FAILED`, `API_AUTH_EXPIRED`, `API_SCOPE_DENIED`, `API_ROUTE_UNSUPPORTED`, `API_CONTRACT_CHANGED`, `API_RATE_LIMITED`, `API_TIMEOUT_BEFORE_SEND`, `API_RESULT_UNKNOWN`, `API_RESPONSE_LIMIT`.
- **Guest transport**: `SSH_HOSTKEY_CHANGED`, `SSH_AUTH_DENIED`, `SSH_GATEWAY_UNREACHABLE`, `GUEST_NOT_READY`, `GUEST_EXIT_NONZERO`, `GUEST_TIMEOUT_UNKNOWN`.
- **Postconditions**: `BOX_CREATION_OBSERVED`, `BOX_EXISTS_BUT_SSH_NOT_READY`, `BACKUP_RECORD_CREATED_UNVERIFIED`, `RESTORE_TARGET_UNVERIFIED`, `NETWORK_RULE_CONFIGURED_NOT_PROVEN_REACHABLE`.
- **Diagnostic itself**: `TRACE_NOT_FOUND`, `TRACE_CAPTURE_DISABLED`, `TRACE_TRUNCATED`, `TRACE_UPSTREAM_UNCORRELATED`, `D4_UNAUTHORIZED`, `D4_PROBE_UNSUPPORTED`.

Exact names are proposed: a future implementation must publish a code registry and avoid reclassifying an existing public error without migration/acceptance. Typed response parsing may use HTTP status, protobuf error details and allowlisted fields; **never parse arbitrary daemon error prose as a trusted instruction**.

## 6. One concrete end-to-end example

An Agent invokes `create_box`. The Provider API returns success, but SSH is not yet ready:

```text
T=001 MCP HTTP authenticated                        observed
T=002 operation accepted and identity/name claimed  observed
T=003 dispatch intent durably committed             observed
T=004 POST /v1/containers sent                      attempt recorded
T=005 authenticated API response received           acknowledged only
T=006 GET exact target: created/running              Provider-observed
T=007 SSH readiness attempt: not yet ready           guest-not-verified
T=008 operation receipt: partial/pending             no duplicate create
```

Agent-facing diagnostic result should emphasize:

> The target Box was observed created. SSH readiness is unverified/unavailable. Do not submit another create request. Inspect readiness or the exact authorized SSH diagnostic.

This is **illustrative**, not evidence of a real run. If the response was lost after T=004 and no reliable subsequent read is possible, the action stays **UNKNOWN**, even though no error detail is available.

## 7. D1–D4 evolution and cross-layer diagnostic contract

### D1 — ordinary event capture
- At entry, service, Provider API and Executor boundaries record class/code/duration/capture status with no user content.
- Keep bounded best-effort ingestion independent of transaction completion. For each mutation, **durably record the intent/dispatch intent in the business store**; trace loss must not lose the receipt.
- Measure overhead in a task-owned lab; no unbounded synchronous logging on critical paths.

### D2 — private report and self-diagnosis
- Read by a limited, owner-authorized service; fixed selectors `operation_id`/`trace_id`, independent saved receipt, source coverage and missing spans.
- Return useful stages and remediation guidance; never claim a missing event proves no operation.
- Keep a minimal host-CLI read-only `doctor/report` route for broken API/MCP/Web; no host-root dynamically imported app code.

### D3 — Agent-facing incident evidence
- Retain `boxweave_diagnose` concept, with selectors for `inbound_mcp`, `app_operation`, `provider_api`, `box_executor`, `compatibility`, subject to allowed scopes.
- The current `source=compat` tests outbound platform MCP `tools/list`; after migration, this should become **API route/contract/capability** evidence. If a compatibility MCP lane survives, report its schema separately. The **inbound BoxWeave MCP** initialize/tools/resources and OAuth still require separate diagnostics.
- An Agent should be advised to inspect before attempting recovery, but tool instructions cannot force automatic Agent behavior. Default no automatic mutation or external publication.

### D4 — exact Box, operator-authorized bounded checks
- Reuse the existing single trusted operator's server-owned exact Box incarnation allowlist for process/port/service/log summaries.
- Move the fixed diagnostic command into the new Box Executor; never grant D4 raw SSH, unrestricted log/FS scan, access to a different Box, or host privilege.
- Confirm Box identity and authorization immediately before each read. Handle suspended/stopped/rotated SSH as `not_verified` or typed denial, not permission to alter the target.

## 8. Logging privacy, security and retention

Preserve D3/D4 approved defaults unless an explicit later decision changes them: **7-day** automatic retention, **50 MiB** default D3 gateway budget, bounded query/count/size, private root/owner policies and failure-isolated logging. A unified view **does not require** one writable DB across root, gateway and app identities; prefer private per-principal stores with verified read-time joins and stable opaque links.

Disallowed by default:
- Authorization/Cookie headers, JWT, API tokens, SSH private/public key contents, age private identities, environment variables, decrypted restore bodies.
- Raw prompts, command arguments, file paths/file bytes, full stdout/stderr, arbitrary provider journal/error messages, stack traces with secrets.
- Public URLs containing credential-bearing capabilities, exact cross-account host addresses, unreviewed Box logs or full model/inference requests.

Approved support reports disclose only fixed reviewed classes, non-sensitive counts, lengths/statuses, omitting fields with explicit `coverage` metadata. Redaction is **not** a guarantee against every unknown secret. External issue creation remains **two-stage**: private draft, human sensitive-data review, separate explicit upload/comment consent.

REST sends the age identity secret to the authenticated daemon for backup restore in the upstream API contract. The trace adapter must have a **request-body-free** logging path for `POST /v1/backups/{id}/restore-box`; audit only approved backup reference/target digest and result class. Never log outgoing JSON as a default debug shortcut.

## 9. API/daemon observability boundary

Three separate stages should be visible:

1. **BoxWeave-owned**: call attempted, local policy, DNS/connect/TLS, send/read, response status, structured validation, elapsed time.
2. **Containarium daemon-owned**: whether a particular request was accepted/applied, and any internal Incus/storage/capability problem. Only available if daemon explicitly provides trustworthy status IDs/events.
3. **Guest/user outcome**: SSH ready, file hash matches, network packet traverses, restored backup verifies. Requires independent check.

If the daemon exposes no safe request-ID or operation lookup, the BoxWeave report must state `daemon_internal_trace: unavailable`. Future upstream contribution might add request/operation IDs to authenticated REST and daemon logs, plus narrowly scoped status lookup; **do not make BoxWeave correctness depend on such an upstream contribution**.

## 10. Decision-tree guidance for Agents

For a failed operation:
1. Look up existing receipt and trace for the *exact* operation/request key.
2. If **local rejected before dispatch**, report the specific authorized validation/scope failure; do not invoke Provider.
3. If **provider API request may have been sent** with no conclusive result, report **UNKNOWN**, inspect exact target/route/backup record, refuse non-idempotent replay.
4. If API acknowledged, inspect actual postconditions and separate guest readiness/packet/backup verification.
5. If failure is clearly at inbound MCP/OAuth, do not claim a Containarium error; if SSH trust fails, do not try a new untrusted host or repair keys silently.
6. If diagnosis remains missing, report `coverage: incomplete` and provide the next *read-only* probe. Do not manufacture a success from a missing span.
7. Before suggesting an upstream GitHub issue, verify it is still reproducible against an exact version, distinguish upstream vs BoxWeave ownership, privately draft and let a human approve disclosure.

This last rule is aimed at reducing repeat reports of already-fixed or unverified problems; it is not autonomous incident filing.

## 11. Acceptance tests and metrics

| Test ID | Required proof |
| --- | --- |
| TRACE-01 | Same `trace_id` across MCP/Web entry, Operation Core, outbound API, postcondition and D2 read; if split principals, correct authenticated linkage |
| TRACE-02 | 401/403/404/409/429/5xx, malformed structured error, route absent, JSON size bound each classified correctly without raw payload |
| TRACE-03 | Timeout before provable dispatch vs after possible send, no false `failed` and no automatic mutating retry |
| TRACE-04 | API acknowledgement followed by delayed Box readiness and SSH trust failure are distinguishable |
| TRACE-05 | Containarium request-ID missing: report gap, never invent upstream trace span |
| TRACE-06 | D3 survives gateway/process restart within retention; prune and size cap enforced; disabled/full capture reports omission |
| TRACE-07 | D4 default denial, exact Box/allowed service checks, wrong instance and unauthorized logs; no raw output or host-shell bypass |
| TRACE-08 | Network/backup/snapshot configured vs independently validated, including partial and unknown |
| TRACE-09 | Secret canaries in headers, restore body, tool args, stdout, URL query and malicious upstream error are absent from logs, reports and MCP results |
| TRACE-10 | Two Agent transports, Web and host CLI can inspect their permitted evidence; no unintended actor cross-read |
| TRACE-11 | Diagnostic storage failure does not alter original provider result, delay beyond bound or cause duplicate dispatch |
| TRACE-12 | Agent lookup after service restart finds durable receipt even when trace journal has expired; trace expiry clearly marked |

Suggested measures are *not performance promises*: percentage of requests with complete owned-boundary spans, percentage of UNKNOWN correctly preserved, number of false "Box create failed" diagnoses where Box was already created, log capture overhead and secret-leak tests. Record actual environments, revisions and CI/live evidence before quoting any numbers.

## 12. Minimum viable implementation boundary

The smallest useful delivery (during future migration phases A1/A2) is not a tracing system rewrite:

- Add an **API-specific classified span** around each new outbound REST call.
- Link it to existing app operation receipts and their trace IDs.
- Preserve D3 external MCP failure capture unchanged.
- Add `boxweave_diagnose` a *bounded read-only* API compatibility/health report without exposing tokens.
- Test UNKNOWN and redaction against a real disposable Box and a simulated post-send transport interruption.

Only after such a slice passes should Agent gateway and D3 internals be consolidated. Avoid simultaneous changes to outbound transport, gateway routing, OAuth, full task store and host installer: that produces untriageable regressions and defeats the project's diagnostic purpose.
