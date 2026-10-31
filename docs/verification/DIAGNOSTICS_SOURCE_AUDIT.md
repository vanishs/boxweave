# Diagnostics source audit — BoxWeave 0.21.4

Status: **research / proposal only (S/D); NOT an approved implementation decision**.
Source: `main@0fcfa6a3ea0f3d86cc929a5d7646bbc06c1b952b`, `VERSION=0.21.4`.
Review date: 2026-10-30 America/Los_Angeles.
Request authority: SRC-104 (research, write and publish documentation only).
Companion: [diagnostics implementation proposal](../design/DIAGNOSTICS_PROPOSAL.md).

## 1. Question and conclusion

Question: can BoxWeave make failures and unexpected behavior easier to investigate during
rapid nested-BoxWeave development, while enhancing the **existing** MCP connection rather
than installing a separate Debug MCP?

**Finding:** yes, but MCP tool additions are not the first dependency. The primary missing
capability is durable, low-disclosure, cross-component diagnostic evidence. A host CLI that
still works when MCP is down must remain the primary recovery interface. The existing MCP
can eventually expose **narrow, authorized, read-only diagnostic summaries** over that
same evidence store. Do not create a parallel Debug MCP or a generic host-debug shell.

This is a **source audit**, not executed tests, a deployed-host observation, a security
assessment, or proof that all suspected bugs reproduce. The earlier live evidence below
is historically reported for other revisions and is not rerun here.

## 2. Actual topology: there is more than one MCP path

```text
Agent client
  -> client-native SSH (possibly WSL) -> forced-command boxweave account
  -> bwmini/mcp_gate.py -> installed upstream launcher -> Containarium MCP
  -> Provider / Incus / target Box

BoxWeave Web -> Flask -> Service -> SQLite / worker
                                   -> Provider adapter -> bwmini/mcp.py
                                   -> configured application MCP launcher
                                   -> upstream MCP -> target Box

operator -> /usr/local/bin/boxweave -> root-owned host_cli.py
         -> public installation metadata / local systemd / optional app-user probe
```

The gate and application adapter have **different process, storage and authority
boundaries**. A host-level trace cannot be fabricated by joining events merely because
they have similar Box names or timestamps. Direct Agent MCP has broader upstream
authority than the Web allowlisted plans; an enrolled transport key is not permission
to read unrelated host/application secrets.

Important source anchors:

| Boundary / evidence | Current source | S finding and limitation |
| --- | --- | --- |
| Forced-command MCP entry and key enrollment | `bwmini/online.py` `enroll` / `client_config` | SSH entry runs `mcp_gate.py` as dedicated account; no independent Debug MCP |
| Gate / upstream forwarding | `bwmini/mcp_gate.py` `catalog` / `main` | Already injects `boxweave_nesting` into discovery; most other tools are forwarded; gate has no persistent diagnostic journal |
| Application-side MCP | `bwmini/mcp.py` `MCP.call` / `MCP.tool` | Error translation discards most upstream detail; child stderr goes to DEVNULL |
| Provider and exec | `bwmini/provider.py` | Performs observed identity/`hello`/lifecycle via MCP; maps underlying tool outcomes |
| Core work receipts | `bwmini/store.py` / `bwmini/service.py` | Durable operations, audit, observations; unexpected process/remote exception detail intentionally omitted |
| Lifecycle / access | `bwmini/lifecycle.py` / `bwmini/access.py` | Durable state/phase/error and event tables; not a full trace across launcher/Provider |
| Job and selftest | `bwmini/jobs.py` / `bwmini/selftest.py` | Jobs persist metadata/hash rather than output; selftest keeps intermediate steps in memory |
| Web error contract | `bwmini/web.py` | Sends safe error codes; generic exception -> INTERNAL_ERROR without durable request correlation |
| Host CLI | `bwmini/host_cli.py` | Installed public doctor/version/inspect, opt-in app-user MCP read, scoped nesting diagnosis |
| Package health | `bwmini/distribution.py` | Source/package health checks installed release, user services, Web, optional MCP |
| Client diagnose | `bwmini/client_profiles.py` | Generated Python script checks key/pin, SSH, initialize/list/read and registration |
| Nested diagnosis | `bwmini/nesting.py` / `docs/NESTING.md` | Local LXC detection and proc-mount signature; parent nesting/effective execution explicitly unverified |

## 3. Specific diagnostic gaps (source-observed, not live reproduced)

### F-01: host doctor can report exit zero despite incomplete health

`bwmini/host_cli.py` `main` sets `code=0` for `doctor` after `status` and
`mcp_probe`, including when the returned MCP state is `unavailable` or
`not_checked`. `status` checks loopback HTTP reachability only, with
`app_health=not_checked`; its `doctor` service list covers credential/trust timers
and the user manager but does **not** independently check the app user
`serve` / `worker` units. HTTP 401 is reachability, not authenticated health.
Do not claim the installed host doctor currently proves application readiness.
The separate source/package `distribution.health` performs broader user-service
checks, but is not the same installed entry path.

Proposed correction: explicit per-check tri-state `passed / failed / not_verified`,
`required_for` scope, and documented 0/1/2 exit semantics. A diagnostic command
may return zero only when **all checks it explicitly promised** completed
successfully; a plain, intentionally shallow `status` remains distinct.

### F-02: generated client diagnose can label partial results as pass

In `bwmini/client_profiles.py` `run_diagnose`, the final exit 2 checks only
`failed` and `mismatch`. For `identity=missing`, `registration=absent` or
`registration=differs`, checks can remain incomplete without making the overall
state fail. The code does distinguish many stages; it should instead report
`incomplete` unless the required selected-client checks passed. Separate
`agent_ui=not_verified` from SSH/MCP registration and never infer model inference.

The generator currently exposes some of the last SSH stderr on failure, up to
200 characters (`run_diagnose`). That text is **not** a safe general-purpose
redaction policy. Replace it with classified codes and store any further
approved detail in a private, bounded local diagnostic channel.

### F-03: upper-layer errors lose useful lower-layer evidence

`bwmini/mcp.py` maps almost all upstream JSON-RPC errors and tool `isError`
to `PROVIDER_REJECTED` (with limited special handling for trust and an exact
network-policy-absent case), while discarding process stderr. `mcp_gate.py`
also discards launcher stderr and forwards upstream responses without a
durable correlation record. `service.py` ignores unexpected exceptions in
work execution and limits receipt fields by design. This is appropriate for
public secrecy, but loses provenance unless a **separate safe event channel**
exists. Never solve it by exposing raw error text, token-bearing subprocess
output, `connect` command payloads, or full instance logs to Agent tools.

### F-04: useful receipts do not equal trace evidence

Operations have IDs, state, code, deadlines and in some paths persisted
phase/events. `environment_events` and `access_events` already offer sound
building blocks. However, an operation does not currently carry a shared
trace/span identifier across host CLI -> gate -> upstream MCP -> workload,
or reliably retain a classified upstream rejection detail and diagnostic
capture coverage. `jobs.py` stores output size/hash and metadata, not the
actual output; that is a deliberate confidentiality boundary.

### F-05: selftest is live, but crash-time forensic steps are weak

`bwmini/selftest.py` confirms create, identity/Python `hello world`, stop,
delete and cleanup. It returns per-step duration/results **after** the
sequence. Steps are accumulated in memory; a killed process can lose the
timeline before returning. It can perform mutating work, so it must never be
part of routine read-only `doctor`. Cleanup can be `best_effort_failed`;
the original test instance must then be reconciled by exact identity,
not automatically duplicated or deleted by a subsequent run.

### F-06: nested environments require explicitly bounded attribution

`bwmini/nesting.py` can report the local container type and classify
a specific proc-mount denial, but cannot directly attest the parent host's
`security.nesting` or effective mount policy. The existing
[NESTING_VERIFICATION](NESTING_VERIFICATION.md) reports effective nested
behavior changing only after outer restart in its test configuration.
[OPEN_RISKS](OPEN_RISKS.md) RISK-620 documents a real split gate-host
vs Provider-host helper availability problem. Therefore a diagnostic report
must never turn `parent_nesting=not_verified` into a confirmed cause, or
confuse a missing gate-side helper with a workload-create failure.

### F-07: installed-source and live-runtime identities differ

`boxweave provenance` exposes reviewed installed source identity; Git
`main` and live installed code are not necessarily the same version.
`docs/verification/REMEDIATION_TEST_QUEUE.md` reports historical 0.21.4
update/key-preservation, a changed worker config requiring restart and a
stale long-lived client SSH transport. A report must carry installed release,
effective runtime configuration digest/restart state when safely observable,
and connection/probe evidence independently, rather than assuming the current
Git commit is running.

### F-08: no verified general-purpose support bundle path

No existing repository command was identified that atomically combines
an operation's typed receipt, separate app/host/Provider observations,
bounded journald slices, source identity and nested-environment provenance
into one privacy-reviewed export. Application install logs exist in private
install paths (`bootstrap.py` / `online.py`), but are not a universal runtime
trace. A `debug_logs`-style unrestricted file reader is not an acceptable
replacement.

## 4. Failure examples to preserve in acceptance

| Symptom | Distinguish with evidence | Must not assert / do |
| --- | --- | --- |
| MCP initialize fails | transport exit/EOF/timeout, pin, forced-command and gate/launcher availability | Do not call it a Provider API fault without a successful MCP handshake |
| Create returns a Provider rejection | upstream category, Guard/policy storage if observed, pre/post exact target inventory and submission certainty | Do not re-create under a new name or publish raw rejected payload |
| Create succeeded but nesting apply failed | original create acknowledgment, target identity, helper host identity, nested helper code | Do not pretend creation failed; do not repeat create |
| Box Running but Python hello fails | provisioning completion, user/host identity, managed key/trust, exec exit and availability | Do not equate Running with command-ready |
| Inner proc-mount denied | child Incus log signature, independently supplied parent policy/effective test | Do not infer parent config solely from child denial |
| Web 500 after upgrade | release SHA, app/worker unit state, request/case IDs, effective config digest | Do not assume Web route reachability implies worker success |
| User reports unexpected behavior without error | expected state and actual state, request time, operation/revision, changed config, explicit gaps | Do not invent a traceback or causal diagnosis |

## 5. Research disposition

- **Reuse:** bounded error codes, durable operation IDs/receipts/events, honest
  `unknown` and single-dispatch semantics, current restricted SSH gate,
  exact instance inspection, live selftest plan/confirmation.
- **Repair first:** success/incomplete classifications and missing check coverage.
- **Add next:** shared private diagnostic event recorder and independent CLI report.
- **Optional later:** one narrowly scoped, read-only diagnostic lookup on the
  **existing** MCP gate only after authorization and runtime compatibility tests.
- **Not justified yet:** separate Debug MCP, general host shell, universal project
  log ingest, remote telemetry/SaaS, auto-repair or an AI root-cause oracle.

See [implementation proposal](../design/DIAGNOSTICS_PROPOSAL.md) for interfaces,
sequencing, security, source touchpoints and executable acceptance design.

## 6. Evidence limits and references

Evidence class **S** = current source inspected; **D** = existing historical
documentation; **T** = actually executed at named revision; **U** = not tested.
This report is **S/D**. No local checkout, Linux host, child Box, database,
authenticated Provider, browser or generated client was executed or changed.

Primary code refs at reviewed commit:
- [host_cli.py](https://github.com/vanishs/boxweave/blob/0fcfa6a3ea0f3d86cc929a5d7646bbc06c1b952b/bwmini/host_cli.py)
- [mcp.py](https://github.com/vanishs/boxweave/blob/0fcfa6a3ea0f3d86cc929a5d7646bbc06c1b952b/bwmini/mcp.py)
- [mcp_gate.py](https://github.com/vanishs/boxweave/blob/0fcfa6a3ea0f3d86cc929a5d7646bbc06c1b952b/bwmini/mcp_gate.py)
- [client_profiles.py](https://github.com/vanishs/boxweave/blob/0fcfa6a3ea0f3d86cc929a5d7646bbc06c1b952b/bwmini/client_profiles.py)
- [service.py](https://github.com/vanishs/boxweave/blob/0fcfa6a3ea0f3d86cc929a5d7646bbc06c1b952b/bwmini/service.py)
- [lifecycle.py](https://github.com/vanishs/boxweave/blob/0fcfa6a3ea0f3d86cc929a5d7646bbc06c1b952b/bwmini/lifecycle.py)
- [selftest.py](https://github.com/vanishs/boxweave/blob/0fcfa6a3ea0f3d86cc929a5d7646bbc06c1b952b/bwmini/selftest.py)
- [nested verification](NESTING_VERIFICATION.md) and [remediation evidence](REMEDIATION_TEST_QUEUE.md).
- MCP protocol compatibility must be evaluated against the pinned client/server
  `2024-11-05` negotiation and a reviewed current specification, not presumed:
  [MCP tools (2025-11-25)](https://modelcontextprotocol.io/specification/2025-11-25/server/tools);
  [base JSON-RPC contract](https://modelcontextprotocol.io/specification/2025-11-25/basic).

## Related later proposal — not part of this historical source audit

A distinct [Box checkpoints, clone and environment fingerprint proposal](../design/BOX_ENVIRONMENT_PROPOSAL.md) now documents phased [E0-E4 implementation handoffs](../design/BOX_ENVIRONMENT_PROPOSAL.md#12-development-batches--source-of-implementation-handoffs) for the owner's reused-box and fresh-box development workflow. **NOT IMPLEMENTED**. These proposed features do not change the reviewed `main@0fcfa6a` behavior or the evidentiary classification of this audit. Similarly, [D0-D2 diagnostics](../design/DIAGNOSTICS_PROPOSAL.md) remain design-only; snapshot or cloning support is not established for the deployed Containarium MCP by native Incus capabilities alone.

## Related capability-led MCP reuse audit — research only (2026-10-31)

The independently proposed [BoxWeave MCP capability integration master](../design/MCP_CAPABILITY_INTEGRATION_MASTER.md) and [source gap audit](MCP_CAPABILITY_SOURCE_AUDIT.md) compare **current BoxWeave functions** with Containarium Platform and agent-box MCP tools, including proposed security boundaries and M0–M5 handoffs. This is **S/D RESEARCH ONLY; NOT IMPLEMENTED**, does not change or approve D0–D2 diagnostics or E0–E4 Box environment plans, and does not establish authenticated runtime MCP capabilities. Current existing diagnostic/environment specifications retain their independent implementation gates.
