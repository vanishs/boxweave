# D3 + D4 staged implementation and test handoff

**Authority:** [D3/D4 master](DIAGNOSTICS_D3_D4_MASTER.md); [original proposal](DIAGNOSTICS_PROPOSAL.md); [D1](DIAGNOSTICS_D1_IMPLEMENTATION.md), [D2](DIAGNOSTICS_D2_IMPLEMENTATION.md), [gateway trust](MCP_INTEGRATION_SECURITY.md).
**Baseline:** 0.24.5. A release is not an installed real-box verification. Each batch has independent T/U evidence.

## Batch 0 — source and design baseline

Freeze existing gateway relay and report-reader schemas. Record tool catalog compatibility, source version, installed helper allowlist, Python release versions, app-user ownership and current tests. Add these two approved documents and link them from index and proposal; do not imply untested features exist.

## Batch 1 — D3 gateway journal + MCP read

Create small standard-library-only root-owned installed helper for gateway diagnostic storage, secure owner-bound path, fixed server-owned configuration and 7-day/50-MiB defaults. Use SQLite private durable store with time/row/byte caps, no user text fields or full payloads. On all ordinary `tools/call` results and classified exceptions record selected tool name (bounded validated), opaque gateway trace ID, stage, outcome, code and elapsed duration; do not alter upstream response bodies or issue additional Provider writes. Errors cannot obscure original result. Expose read tool `boxweave_diagnose` for recent and trace lookups. Tool-policy filtering applies to it like any other tool. Document best-effort client automatic diagnosis.

Tests: write permission/retention/size limit, unknown/failed/success labels, disabled/write failure, malformed names and IDs, collisions, no raw tool args or returned output, concurrent calls, persistence across processes, recent/truncated queries, strict tool-catalog behavior.

## Batch 2 — D3 app-side fixed read

Reuse `bwmini.diagnostics.read_report` through fixed application-owned Python and fixed `/home/boxweave/boxweave/config.json`, not arbitrary caller paths. Enforce actor `operator`, typed operation or linked trace and bounded returned fields. No app DB migrations, no root/app privilege confusion. When App is absent or split, return explicit source coverage `not_verified`; no fabricated host correlation.

Tests: foreign actor blocked, trace/operation selectors validated, app reader absent/timeout/invalid JSON, only allowed keys returned, size cap and no raw DB/credential exposure.

## Batch 3 — D4 specifically preauthorized project diagnostics

Add server-owner-controlled **per-instance** policy file with exact logical Box, immutable observed real name+creation identity, allowlisted check types and optional service names. Default policy absent => **DENY**. Do not allow direct user-provided paths or commands. Reverify fresh Provider `get_container` plus managed target trust for every call; perform fixed bounded read over available `connect(exec)` only if permitted, then summarize classifications with no raw project text returned or persisted. No escalation/privileged reads or unapproved installs.

Tests: absent/untrusted policy, malicious filename/log data, exact Box replacement, caller override, stopped Box, upstream isError, timeout and missing capabilities, prohibited checks, large output. Existing `connect(exec)` privileges remain independent, no claim of enforced global operator isolation.

## Batch 4 — Agent interoperability and integration

Update gateway server instructions and `boxweave_info` guidance: explain auto-diagnose on failure, unknown-result no-retry, project-vs-platform diagnoses, developer-vs-user next steps, and mandatory personal review plus separate user confirmation prior to any external Issue submission. Test SSH stdio initialization/tool discovery and optional HTTP OAuth relay, including legacy clients and tool-policy-disabled baseline; check no upstream mutation and no duplicate calls.

## Batch 5 — release, checkpoint, records

Run `python scripts/check_knowledge.py`, focused tests and `python scripts/validate.py` in a prepared CI environment; browser fixtures where affected; package digest and Python version tests. Write S/T/U and exact commit/release IDs to [verification queue](../verification/REMEDIATION_TEST_QUEUE.md). Use new minor version for D3+D4 feature completion; publish GitHub source-bound assets only after CI. Promote `PUBLISHED_RELEASE` and embedded installer pin **after** verifying release. Installed host, SSH/HTTP real MCP, Provider and authorized D4 test Box remain **U** unless separately actually run.

## Scope and rollback

Maintain an independent disabled-feature/denied-policy baseline; D4 default deny on absent policy. The operator may disable automatic capture and deep D4 while preserving existing MCP business tools. Never auto-delete diagnostic evidence on release downgrade; keep previous release and configuration safe. Diagnostics are read-only and do not implement issue submission, repair, scheduler, arbitrary Shell or cross-Box log search.
