# Proposal: evidence-first diagnostics for the existing BoxWeave MCP and CLI

**Proposal, not accepted ADR or implementation authorization.** SRC-104;
reviewed baseline `main@0fcfa6a` / BoxWeave 0.21.4.
Source findings: [diagnostics source audit](../verification/DIAGNOSTICS_SOURCE_AUDIT.md).
Related accepted requirements (do not silently extend): REQ-002/003/004/005/006/007/
008/009/016/017/018/019/020 and their corresponding AC. Existing
[ADR-003](ADR/0003-transport-and-trust.md),
[ADR-007](ADR/0007-host-install-boundary.md) and
[ADR-008](ADR/0008-independent-nesting.md) remain authoritative.

## Developer implementation handoff — D0, D1 and D2 (research-only)

This proposal has three **separate, source-reviewed implementation specifications** for programmers. They give explicit file touchpoints, interfaces, negative tests, packaging/rollback constraints and acceptance gates. All are proposals, not shipped capabilities or permission to perform live resource changes.

1. **[D0 — diagnostic correctness and truthful exits](DIAGNOSTICS_D0_IMPLEMENTATION.md)**: fix installed CLI/app health/client diagnose false-success paths; zero new MCP tools or DB schema changes. Tests D0-T01–D0-T16.
2. **[D1 — structured events and safe trace correlation](DIAGNOSTICS_D1_IMPLEMENTATION.md)**: private schema-v9 evidence, atomic link to existing durable operations, classified error shapes, strict no-replay/no-secret behavior. Tests D1-T01–D1-T17. Gate-host capture is separately gated.
3. **[D2 — private support reports, nested attribution and durable selftest](DIAGNOSTICS_D2_IMPLEMENTATION.md)**: host-safe read-only export and missing-evidence handling (D2-a), optionally persistent mutating selftest steps (D2-b, schema-v10 only if needed). Tests D2-T01–D2-T20.

**Implementation order:** D0 -> D1 app-side -> D2-a -> D2-b only after approval. The optional existing-MCP diagnostic lookup (D3) and generic user-project debugging (D4) remain deferred. Before any coding, reconcile the specs with the then-current main HEAD and maintainer decisions in [OPEN_QUESTIONS](../knowledge/OPEN_QUESTIONS.md). Source-only documentation is S/D evidence, not CI, deployment or live health proof.

## Related environment-management proposal (independent; NOT IMPLEMENTED)

The proposed [Box checkpoints, clone and fingerprint architecture](BOX_ENVIRONMENT_PROPOSAL.md) and its [E0 capability](BOX_ENV_E0_CAPABILITIES.md), [E1 fingerprint](BOX_ENV_E1_FINGERPRINTS.md), [E2 snapshots](BOX_ENV_E2_SNAPSHOTS.md), [E3 restore](BOX_ENV_E3_RESTORE.md), [E4 clone](BOX_ENV_E4_CLONE.md) coding handoffs complement D0-D2 but **are NOT IMPLEMENTED**. Their primary use is a dual-mode development workflow (reuse an existing nested Box, or create a new test Box); the capabilities are designed as general BoxWeave product features. D0/D1/D2 diagnostics are also proposals, not already shipped. Environment capture/checkpoint features must not require working diagnostics, and diagnostics must not imply that an Incus native operation is authenticated via deployed Containarium MCP. No additional MCP, host privilege, DB schema or runtime capability was added by this document link.

## 1. Design choice

Implement **one evidence model** behind separate, appropriately authorized
read-only surfaces:

1. Install-side and app-side emit bounded **structured events** as part of
   their respective trusted processes.
2. The **installed host CLI** diagnoses from public metadata and explicitly
   permitted app-user observations; report export remains available when
   Web/API/SSH MCP is unavailable. It never executes app-owned Python as root.
3. The existing direct-Agent MCP **may later** add a narrowly scoped
   `boxweave_diagnose` tool or response reference, with no second service,
   but only after an explicit cross-boundary access design is approved.

The initial product is a **diagnostic evidence system**, not an autonomous
repair agent. A failed diagnostic must not alter the meaning of a user action,
promise cancellation, grant more authority or cause an automatic retry.

## 2. Competing options and explicit rejection criteria

| Option | Benefit | Cost / risk | Proposed disposition |
| --- | --- | --- | --- |
| CLI fixes only | Small, robust without MCP | Historical root causes still lost | Immediate prerequisite |
| CLI + shared event recorder/report | Can investigate from real evidence | Additive schema, retention/security work | Preferred next slice |
| Existing MCP read-only lookup on same store | Agent can query without extra registration | Gate/app trust boundaries; tool-list/client compatibility | Conditional follow-up |
| Separate Debug MCP | Independent namespace | New enrollment/service/permissions; can fail with same host | Reject for current scope |
| Universal project debugger | Potentially broad value | Unbounded privileged file/process visibility | Defer; demand explicit evidence and opt-in |

If user research later finds terminal/report sufficient, **do not** ship
an MCP diagnostic tool merely to complete a roadmap.

## 3. Proposed interfaces (not implemented)

### CLI

- `boxweave doctor [--json]`: explicit depth and coverage for shallow
  host/timer/Web checks; each check has result `passed / failed / not_verified`.
  `boxweave doctor --mcp`: privileged opt-in authenticated read as app user;
  report `not_checked` without privileges, never success.
- `boxweave report --operation ID [--since UTC --until UTC] --output PATH`:
  bounded, read-only, private exported evidence for a named operation.
- `boxweave report --trace TRACE_ID --output PATH`: same, subject to verified
  trace provenance and available scope. A time-window-only report is useful
  for crashes without an operation ID, with stricter limits.
- `boxweave box diagnose EXACT_NAME --project PROJECT`: keep exact identity
  and Incus privilege checks. Provide classified nesting evidence, not
  raw unreviewed instance log output.
- `boxweave selftest plan/run` stays separately confirmed and **mutating**.
  Add durable per-step evidence only in a later slice; never run it in doctor.

Never copy the entire app DB or `journalctl` output to stdout. Exports must
be named, non-symlink private new files/directories with restrictive modes,
bounded bytes and an explicit "coverage/omissions" manifest. Invalid access
and timeout are `not_verified`, not a successful empty report.

### MCP (deferred, not a separate server)

Once diagnostic scopes are independently tested, a possible exposed tool:

```json
{
  "name": "boxweave_diagnose",
  "description": "Read an authorized, bounded BoxWeave operation diagnostic summary.",
  "inputSchema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {
      "trace_id": {"type": "string"}
    },
    "required": ["trace_id"]
  }
}
```

Illustrative **tool-level** result, not a modification to upstream response
schemas or a statement that the feature exists:

```json
{
  "trace_id": "bw-example-opaque-id",
  "state": "incomplete",
  "scope": "authorized_operation_summary",
  "observed": [
    {"component": "mcp_gate", "phase": "upstream_call", "code": "PROVIDER_REJECTED"}
  ],
  "cause": "not_verified",
  "next": "Collect authorized host report for the same trace"
}
```

Requirements before enabling:

- Exact caller identity/scope **must** be established on the forced SSH
  boundary and remain bound to stored evidence. Current `mcp_gate.py` does
  not itself implement a per-client persistent diagnostic authorization
  registry. Possession of any enrolled key must not expose all host traces.
- Fail closed for a trace outside the caller's verified scope. Do not
  introduce an arbitrary `path`, `command`, `tenant` or `log_bytes` parameter.
- Keep responses short, allowlisted, rate/size limited and redacted; use
  controlled tool `isError` / standard JSON-RPC semantics.
- A new tool must coexist with `boxweave_nesting` and upstream pagination,
  tool-name conflicts, existing `tools/list` behavior, selected client
  compatibility, and pinned `2024-11-05` protocol negotiation.
- Optional diagnostic references in existing tool responses require their
  own **backward-compatibility proof**. Do not blindly wrap upstream tool
  content or replace a Provider response: clients may parse it strictly.
- Do not assume that `_meta` is surfaced to every Agent UI. Choose only a
  negotiated/validated MCP mechanism; a safe error code plus CLI lookup
  remains the fallback.

## 4. Event and correlation contract (proposal schema v1)

Candidate fields (allowlist; omit unknown, do not fabricate data):

| Field | Rule |
| --- | --- |
| `schema_version` | Fixed integer; strict forward compatibility |
| `event_id`, `trace_id`, `span_id`, `parent_span_id` | Random opaque identifiers; distinct from JSON-RPC request ID; parent only when actually propagated |
| `operation_id` | Existing durable operation receipt ID when present |
| `time_utc`, `elapsed_ms` | UTC observation timestamp and monotonic local duration; do not compare process monotonic clocks |
| `component` / `stage` / `event_type` | Controlled enums: host_cli, web, worker, adapter, mcp_gate, upstream, selftest, nesting |
| `action` / `outcome` / `code` | Controlled and bounded; distinguish pass, fail, unknown, not_verified |
| `target_ref` | Exact managed identity fingerprint or pseudonymous reference, not name/IP-only authority |
| `environment_ref` | Verified layer/install identity and source/release digest; no secret config content |
| `evidence` / `confidence` | Direct observation, documented inference, caller report or missing; never equate inference with proof |
| `scope` / `capture_status` | Which trust layer was observed, truncation/dropped events, clock and permission limitations |

Log events should capture phase transitions, tool name (allowlisted), status,
deadline, duration and **classified** errors. Never store full MCP arguments,
raw tool responses, public/secret keys, SSH arguments, passwords, JWTs,
private config, environment dumps, Web body, model prompts, user project
files or arbitrary stdout by default. Some arbitrary strings can themselves
contain secrets: redaction is defense in depth, **not** authorization.

Generate trace IDs at **owned ingress points**: Web accepted operation,
app adapter MCP call, direct gate MCP call, explicit host selftest, and CLI
diagnostic request. Bind to existing operation IDs when available. A
direct-gate trace and app-side trace may be **unlinked** until a vetted
propagation mechanism exists; do not infer shared identity from same Box
and time. Failed initialize may have no established remote trace; local
host/client probe identifiers are still useful.

Suggested example local record:

```json
{
  "schema_version": 1,
  "trace_id": "bw-example-opaque-id",
  "span_id": "span-example",
  "operation_id": "operation-example",
  "time_utc": "2026-10-30T20:00:00Z",
  "component": "app_mcp",
  "stage": "tools_call",
  "action": "create_container",
  "outcome": "failed",
  "code": "PROVIDER_REJECTED",
  "capture_status": "classified_only",
  "evidence": "observed"
}
```

This JSON is an **illustrative proposal**, not current BoxWeave output.

## 5. Storage, process and crash architecture

- Prefer additive, versioned private SQLite event tables within the
  **application-owned** store for Web/Worker/Service events and receipt
  correlation; preserve transaction boundaries and existing migration/
  backup semantics. A remote Provider call must not run inside a DB
  transaction. Logging may not alter single-dispatch/recovery behavior.
- The **root-owned installed host CLI** must not read app-writable source
  as root or bypass private DB permissions. Run narrow approved app-user
  readers via fixed arguments, or return `not_verified` without permission.
- The direct SSH **gate** is a separate long-lived-per-connection process
  using the restricted account. If persistent gate events are accepted,
  store them in a dedicated, bounded, owner-protected journal/sink **only
  for that boundary**. Do not assume it can open the app SQLite file or
  Provider-host root logs, especially when the gate and Provider are
  deployed in different boxes. A standalone local collector process is
  **not** required just for v1; confirm actual file ownership first.
- Support a bounded append/flush sequence for *critical transitions*
  (e.g. intent persisted, request submitted, known rejection, uncertain
  outcome), not a byte-by-byte copy of every MCP stream. If capture fails,
  operation behavior is unchanged and `capture_status=unavailable` is
  reported wherever independently possible.
- Keep a finite retention horizon and byte/event cap, secure file modes,
  explicit disk-full behavior, monotonic local durations and safe cleanup
  of **owned** expired diagnostics. Do not collect arbitrary persistent
  user project data or upload anything by default.
- In case of crash/kill, the last *durably written* event plus operation
  receipt is evidence; absence of a later event proves neither completion
  nor non-execution. Source/manifest version, worker restart/generation
  and config digest need separately observed attestations.

## 6. Nested chain: attach evidence, do not grant new authority

Layer labels L0/L1/L2/L3 are descriptive, not reliable identities. Each
report records its own verified instance UUID/creation/project/source where
authorized and uses a cryptographically opaque link label. The parent
relationship is either:

1. **verified** by an authorized outer-layer read against an exact target;
2. **operator_attested** and clearly attributed; or
3. **not_verified**.

Guest CLI must not infer the parent from hostname, IP or cgroup alone.
Do not auto-run `sudo incus` on a host or install a new privileged helper.
Outer-layer diagnosis may be attached **manually** as a separate
operator-approved redacted report. A failed nested child can yield
`nested_proc_mount_denied=true` while `parent_nesting=not_verified`.

For split gate-host/Provider-host setups (RISK-620), record the **actual
helper host and capability** independently. A post-create nesting helper
error is distinct from a create rejection; the original Box identity
must be inspected before considering any new mutation.

## 7. Report export format and privacy rules

Candidate manifest:

```text
report/
  manifest.json        # schema, trace, exact source/release, collection scope
  timeline.jsonl        # bounded allowlisted events
  checks.json          # passed / failed / not_verified by component
  operation.json       # redacted receipt/state/reconciliation summary
  layers.json          # verified and unverified parent-child evidence
  omissions.json       # missing permissions, truncated spans, clock/retention gaps
  README.txt           # safe review/handling instructions
```

Export locally with atomic finalization, bounded size/time and restrictive
permissions. By default **no** `.env`, full `runtime.json`, full SSH logs,
raw journal, full Incus config, full SQLite DB, user workspace, key material,
HTTP request body, arbitrary exec output, stack memory or external upload.
Use a conservative key allowlist; all unexpected/unrecognized fields are
dropped, not printed. A private local diagnostic view (if approved) may
retain classified upstream subtype/status only, not the upstream raw text.

Disclose redaction/coverage limits honestly: machine output may still
contain sensitive information if the allowed fields are wrong. Test for
synthetic credentials, prompt-injection strings and control characters.
Protect against symlink/hardlink/path traversal, other-actor report read,
oversized logs and unknown schema. An exported report is an artifact to
review before sharing, not automatically safe to publish.

## 8. Implementation slices and affected files

### Slice D0 — correctness, no added MCP tool (P0)

- `bwmini/host_cli.py`: derive doctor result/exit from explicit promised
  checks; inspect actual app units under supported ownership, and report
  absent/inaccessible units as `not_verified` rather than `ready`.
- `bwmini/client_profiles.py`: selected connection/identity/registration
  requirements gate overall `pass`; missing is not pass; remove raw SSH
  stderr from public diagnostic output.
- `bwmini/deploy_check.py`, `bwmini/distribution.py` and
  `bwmini/manage.py`: harmonize result vocabulary and exit status without
  collapsing different `status` versus authenticated `doctor` scopes.
- `tests/test_host_cli.py` / `tests/test_client_setup.py` /
  `tests/test_distribution.py`: negative and privilege-boundary tests.

### Slice D1 — local structured evidence (P0/P1)

- Add a small **typed** `bwmini/diagnostics.py` utility and additive
  `bwmini/store.py` schema for app-owned events; do not create a second
  business-state model.
- Instrument `web.py` / `service.py` / `lifecycle.py` / `access.py` /
  `provider.py` / `mcp.py` at ingress, submit, acknowledge,
  state-observation, rejection and unknown transitions.
- Keep classified underlying errors *privately* while returning existing
  public codes. Distinguish `MCP_STREAM_CLOSED`, deadline, malformed
  protocol, trust failures, tool rejection, unknown completion.
- Independently review gate-side event capture and ownership in
  `mcp_gate.py` / installed `online.py`; this may be deferred until
  an actual gate-host boundary is tested.
- New tests: deterministic clock, crash between event/receipt, bounded
  persistence, malformed Provider messages, no secret data, store
  migration/backups, no extra remote calls/replays.

### Slice D2 — support report / nested evidence (P1)

- `host_cli.py`: `report` with fixed typed source selectors and bounded
  app-user reader, never arbitrary shell/file/DB access.
- `nesting.py`: preserve existing classified proc denial; attach
  unverified/verified layer labels only with actual authority.
- `selftest.py`: proposed incremental durable step receipts and explicit
  preserved-target reconciliation; keep mutating plan/run confirmation.
- New export tests: limits, partial failure, inaccessible logs,
  redaction, sibling-user isolation, disk-full and restart.
- Add operator docs and examples only after command help is implemented.

### Slice D3 — existing MCP diagnostic read (conditional P2)

- Consider `boxweave_diagnose` in **existing** `mcp_gate.py`, not a
  separate MCP registration; implementation gated by proof of caller
  identity, scope, compatible tool-list pagination and storage locality.
- Supply only authorized summary data. Operator CLI remains the
  recovery route if MCP itself is unreachable.
- Test all selected generated client App/runtime/transport profiles,
  old-client compatibility and a disabled-feature baseline.
- Do not claim the optional tool exists or release it solely because
  D0-D2 completed.

### Slice D4 — user-project debugging (deferred)

Only if real usage warrants it: explicit per-project opt-in and allowlisted
process/service/log descriptors **within that project's Box**, independent
target authorization and resource quotas. No cross-Tenant log search,
general `exec` proxy, breakpoint service, project filesystem crawling or
privilege escalation by default.

## 9. Acceptance matrix for an implementing Agent

| ID | Test scenario and required result | Evidence class before execution |
| --- | --- | --- |
| DG-01 | Doctor with HTTP 401, worker stopped, timer running: Web reachable but overall requested app readiness not passed | U |
| DG-02 | `doctor --mcp` without sudo and with failing launcher: not_checked/unavailable, nonzero if promised check incomplete | U |
| DG-03 | Generated client missing pin/identity, missing registration, mismatched registration, dead SSH: no false overall pass | U |
| DG-04 | Real MCP initialize/list/read success still reported correctly, without claiming Agent UI/inference | U |
| DG-05 | Provider rejects with synthetic credential in error: safe public error, private categorized evidence, zero secret leakage | U |
| DG-06 | Worker killed before submission / after remote effect: original receipt unknown, trace survives where flushed, **no replay** | U |
| DG-07 | Log storage unavailable/disk full: user operation and error contract unchanged; capture gap explicit | U |
| DG-08 | Report on absent/unreadable records, other identity/unknown trace: fail closed, no raw DB/secret output | U |
| DG-09 | Nested proc denied, parent unavailable: classify signature but parent/effective capability not verified | U |
| DG-10 | Split gate/Provider hosts, helper missing after create: original create preserved; diagnosis does not recommend duplicate | U |
| DG-11 | Installed update/restart: source/release/config-generation evidence is accurate; old records persist under migration | U |
| DG-12 | Tool compatibility: current clients and upstream MCP 2024-11-05 unchanged with diagnostic tool disabled/enabled | U |
| DG-13 | End-to-end confirmed create/identity/Python/stop/delete with no accidental mutations from doctor/report | U |
| DG-14 | Report contains injected tool text, fake paths, synthetic private key/JWT and huge output: no arbitrary command or leak | U |

Verification sequencing: run `python scripts/check_knowledge.py`, focused
unit tests and `scripts/validate.py` only in a prepared non-root checkout.
Run the actual client profiles and authorized nested live cases only under
a **new explicit** environment-specific approval; inspect/reuse original
test resources and preserve failures. Record source SHA, environment,
exact command, exit, redacted result, observed/not-verified classification
and residual risk. No CI/fixture pass establishes parent-host policy.

## 10. Go/no-go criteria and outstanding decisions

**Go for D0** after approval: fixes existing inaccurate diagnosis without
new privilege or public data collection. **Go for D1/D2** only once the
schema, ownership, storage quotas, default retention, private export and
no-replay acceptance are reviewed. **No-go for D3** until the SSH caller
is bound to a demonstrably limited diagnostic scope; no global trace
lookup under an arbitrary enrolled key. **No-go for D4** without a
separate product/privacy decision and real project usage evidence.

Questions for the owner (tracked in
[OPEN_QUESTIONS](../knowledge/OPEN_QUESTIONS.md)):

1. Approve D0 only, or D0-D2 as a bounded implementation outcome?
2. Should the optional MCP reader be supported at all, and what exact
   caller-to-trace authorization rule would make it safe?
3. What default retention/storage budget and report sharing policy is
   acceptable for a personal trusted-operator installation?
4. Should user-project diagnostics remain explicitly out of scope until
   BoxWeave self-diagnostics has live evidence?

No decisions are implied by publishing this proposal. The authoritative
scope/REQ/AC, ADR index and release version remain unchanged.

## Related capability-led MCP reuse audit — research only (2026-10-31)

The independently proposed [BoxWeave MCP capability integration master](MCP_CAPABILITY_INTEGRATION_MASTER.md) and [source gap audit](../verification/MCP_CAPABILITY_SOURCE_AUDIT.md) compare **current BoxWeave functions** with Containarium Platform and agent-box MCP tools, including proposed security boundaries and M0–M5 handoffs. This is **S/D RESEARCH ONLY; NOT IMPLEMENTED**, does not change or approve D0–D2 diagnostics or E0–E4 Box environment plans, and does not establish authenticated runtime MCP capabilities. Current existing diagnostic/environment specifications retain their independent implementation gates.
