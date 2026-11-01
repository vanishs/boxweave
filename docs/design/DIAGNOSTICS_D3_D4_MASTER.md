# D3 + D4: agent-oriented, evidence-first diagnosis

Status: owner-approved design for implementation; **NOT a live deployment attestation**.
Baseline: BoxWeave 0.24.5, main 11b8f58b. Supersedes the D3/D4 deferred scope in [DIAGNOSTICS_PROPOSAL](DIAGNOSTICS_PROPOSAL.md), while retaining D0-D2 truth and trust rules.
Companion [implementation batches](DIAGNOSTICS_D3_D4_IMPLEMENTATION.md), [source audit](../verification/DIAGNOSTICS_SOURCE_AUDIT.md), [D2 report](DIAGNOSTICS_D2_IMPLEMENTATION.md), [MCP security](MCP_INTEGRATION_SECURITY.md).

## Owner decisions (2026-11-01)

1. D3.1 app-operation and D3.2 direct-gateway diagnostics share one readonly `boxweave_diagnose` tool, inside the **existing** SSH/HTTP MCP gateway. All authenticated, trusted-operator clients share the same diagnostic depth; no fictional per-client RBAC.
2. Ordinary MCP tools emit **minimal classified metadata** at the owned gateway boundary without storing tool arguments, prompts, responses, SSH commands, or secrets. Prefer compatible trace metadata where proven; do not wrap or rewrite upstream business responses. No assumption that agents observe MCP `_meta`: fallback recent-event search.
3. Gateway records persist across process restarts: enabled by default, retention **7 days**, operator-configurable, with a default **50 MiB** total storage cap; bounded queries, writes, timestamps and output. Configuration is server-owned, validated and fail-closed when unsafe. Failures recording diagnostics never change upstream operation outcome, cause replay, or leak to stdout.
4. Agent instructions encourage automatic diagnosis on failures/timeouts/unknown outcomes. **Agent action is not guaranteed** by MCP tool text, and D3 must not execute repair, retry unknown writes, or grant permissions.
5. D4 is project debugging **inside an exact managed Box**, not a general host debugger. Light observations for already-performed MCP calls may reuse D3 data. Add a separate `boxweave_project_diagnose` read tool for bounded process/service/port/status checks and deliberately restricted log reads.
6. **Deep D4 defaults DENY for every Box** until a **server-admin-owned, specific-Box allowlist** is configured. The allowlist binds an exact logical owner and immutable observed Box identity (name and creation fingerprint), selected service names and inspection classes; name-only matching is insufficient. No cross-Box reads, raw arbitrary exec, whole-project file scans, or privilege escalation. If an agent client lacks cryptographically verifiable chat approval, preconfigured operator policy is the only gateway-enforced grant: prose approvals are not trusted as server proof.
7. Once a Box/scope has valid operator preauthorization, routine bounded readonly checks are allowed without repeated approval. Extending to secrets, unrestricted logs, another Box, higher privilege or any mutation is **not** silently allowed; the Agent must seek appropriate separate authorization, and a chat approval alone cannot override server allowlist.
8. D3/D4 publish **one coherent diagnostic evidence model** (source, trace, outcome, code, observations, coverage/omissions, evidence level and suggested investigation). Full redaction is not guaranteed; keep generated reports private.
9. **External disclosure is two-stage**: draft an issue/report, require the user to personally review all fields/attachments for sensitive data, and obtain a separate explicit final approval before submitting/uploading to GitHub or any other external service. Never auto-submit.
10. Developer versus ordinary-user next steps are **Agent workflow guidance**, not a D3 mode or a difference in read depth. Users receive safe explanation/issue draft; developer agents may investigate BoxWeave source and plan fixes under independently authorized tools. D3/D4 remain readonly.

## Trusted components and data ownership

SSH stdio/HTTP OAuth -> BoxWeave operator gateway -> owned Provider MCP -> selected Box. D3 gateway evidence belongs to the gateway OS account's private storage; app-side D1/D2 evidence stays in application-owned SQLite and is read only by the fixed app-user diagnostic reader. No root-owned helper may import app-controlled Python as root. Split gateway/app/provider hosts are explicit missing sources; never merge traces by Box name or timestamp. Per-gateway trace IDs do not assert authenticated upstream Provider execution.

HTTP relay currently launches a fresh gateway process for each request: in-memory storage cannot implement retention. Writes must be bounded, durable, concurrency-safe, owner-only; reader must reject unsafe path/symlink/hardlink and avoid migrations on read. Existing CLI report remains fallback when MCP itself fails. Stored events have a finite retention window and per-trace/total cardinality limits. Provide explicit omission when capture is disabled/unavailable/full.

## Operation and output contract

`boxweave_diagnose`: select one `source: gateway|app`; `gateway`: `trace_id` or `recent`; `app`: an authorized `trace_id` or typed `operation_kind`+`operation_id`. Server fixes `actor=operator`; client cannot supply arbitrary actor, file path, command, database, tenant or logs. No arbitrary time-window dump by default. All errors classified with stable allowlisted codes; `not_verified` is not `failed`.

`boxweave_project_diagnose`: named Box only; server policy validates exact current instance fingerprint from an authenticated Provider observation and existing trusted target registry before any guest read; inspection is a fixed command template for a declared check class, selected fixed service and capped output. If the Provider lacks required schema or execution permissions, fail closed without silently running something else. This is **not** a generic command execution substitute.

Use `readOnlyHint`, `idempotentHint` and MCP standard result and error formats. Keep upstream `tools/list` pagination, tool policy allowlist, protocol 2024-11-05, SSH/HTTP behavior and upstream result bodies compatible. Attach trace identifiers to upstream `_meta` only after client and Provider compatibility testing, otherwise use recent lookup.

## Threats and deliberate limitations

- Accidental secret capture via untrusted stdout, logs, exceptions, JSON-RPC payloads or project filenames: **deny raw output by default**, return controlled codes and counts.
- Prompt injection embedded in diagnostic data: treat all diagnostics as untrusted observations; never execute reported instructions.
- Operator key compromise is full operator scope, not per-Agent RBAC; D4 deep allowlist limits **new D4 interface**, not older `connect(exec)` rights.
- TOCTOU Box replacement: require fresh name+creation identity verification at each deep read; deny mismatch.
- Monitoring overhead or outages: bounded latency and IO; diagnostic exceptions swallowed on ordinary calls, but diagnostic queries report capture gaps.
- Failure after external write: never repeat a write; `unknown` and next inspection guidance survive.
- D4 does not continuously scan projects, read process environment, fetch arbitrary paths, execute debugger/breakpoints, or send logs off-box.

## Acceptance gates

Unit/negative tests cover schema and tool list, retention/limits, concurrent/restarted readers, disabled configuration, exact identity and allowlist, permission/path races, no secret capture, app other-actor blocking, no upstream replay, unchanged Provider results, SSH/HTTP contracts, synthetic injection and uncertain writes. GitHub CI is T/fixture only. Release of an installed-surface feature must be explicitly marked source-published vs authenticated real-host accepted; the latter requires separately authorized installed upgrade plus actual SSH and HTTP agent invocation, a failing path and a bounded D4 read on a specifically allowed test Box. Existing host resources must not be mutated merely to establish a checkpoint.

**Links back**: [D3/D4 batches](DIAGNOSTICS_D3_D4_IMPLEMENTATION.md) and [parent D0-D4 proposal](DIAGNOSTICS_PROPOSAL.md).
