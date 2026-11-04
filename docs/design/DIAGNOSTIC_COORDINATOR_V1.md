# Diagnostic Coordinator v1 — agent-safe host evidence (DC-01)

**Status:** v0.43.0 shipped; source/CI and isolated real installed-host acceptance passed. Stable default selection is documented separately. **Upstream:** Containarium 0.120.1 unchanged. Original production host not upgraded.

## Problem and evidence

The 0.42.12 REST migration had authenticated host inventory success and six real app-owned lifecycle operations correlated with D2/D3 evidence. The L2 and L3 operator-run acceptance scripts initially failed before dispatch because Python did not resolve `bwmini` from the executing directory. The later corrected scripts validated real REST writes, API acknowledgements versus state observations, D2/D3 operation traces, 401 rejection, token rotation without process restart, and legacy MCP reads. These observations must not be interpreted as proof that the Agent has privileged host execution.

A read-only Agent should be able to diagnose the installed application and obtain typed, attributable evidence **without** pasting arbitrary shell commands into the host or receiving secrets.

## Requirements and acceptance (DC-01)

| ID | Requirement | Acceptance |
|---|---|---|
| DC-01-A | A new Agent tool, `boxweave_diagnostic_run`, has three strictly selected read-only profiles | MCP `tools/list` schema and rejection fixtures |
| DC-01-B | `preflight` checks fixed app interpreter, working directory, installed config and REST selection | Correct typed states and no remote mutation |
| DC-01-C | `rest_read` independently performs authenticated REST inventory and existing MCP read from the **application** runtime | Both must succeed independently to report PASS; no container details or tokens exported |
| DC-01-D | `operation_trace` joins an actor-scoped D2 receipt and bounded API/lifecycle timeline | Exact operation kind/ID required; missing trace is NOT_VERIFIED |
| DC-01-E | Agent-controlled paths, commands, arbitrary environment variables, root escalation, write retries and privileged repair are impossible | Invalid selector tests; source-bound fixed subprocess argv and private ownership model |
| DC-01-F | Fixed diagnostic codes, bounded output, limited timeout, no raw stderr/body/token/argument capture | Secret-canary, import-failure and timeout fixtures |
| DC-01-G | Separate proof grades for source, fixture, CI, installed app, real host; never label missing evidence as PASS | Verification record and honest release gate |

## Contract

```json
{"name":"boxweave_diagnostic_run","arguments":{"profile":"preflight"}}
{"name":"boxweave_diagnostic_run","arguments":{"profile":"rest_read"}}
{"name":"boxweave_diagnostic_run","arguments":{"profile":"operation_trace","operation_kind":"environment","operation_id":"opaque-id"}}
```

Result schema v1: `source=coordinator`, `profile`, `state=passed|failed|not_verified`, `coverage`, bounded `checks=[{id,state,code}]`, `omissions`, and, for operation traces, optional `trace_id`, `event_count`, and a bounded sanitized `events` array. Every event is actor-scoped through the existing D2 reader. No generic details, request bodies, raw paths, JWT, log text or host error strings. No writable journal is required for v1; preflight is ephemeral read-only evidence.

The gateway has **fixed** application paths from its installed root-controlled deployment; a caller supplies only the enumerated profile or an actor-scoped existing operation ID. Gateway verifies execution prerequisites and invokes the installed app interpreter via a fixed argv, fixed cwd, a bounded stdout budget and a deadline. The application probe does not import the CLI facade or execute arbitrary shell commands. Results are validated against a tight schema before crossing the MCP boundary. Application REST and MCP reads have separate outcomes; output truncation and probe crashes are NOT_VERIFIED, not negative proof of an upstream failure.

`operation_trace` reuses the current `gate_diagnostics.query_app` actor-scoped D2 reader. There is no new cross-account report access and no user-chosen receipt path. An absent receipt, pre-migration trace, or captured-but-unobserved write never becomes PASS.

## Boundaries

- **No host shell:** `connect(exec)` remains guest-only; `boxweave_diagnostic_run` is not a general host exec interface.
- **No mutations:** v1 does not create/delete/resize Boxes, change service configuration, rotate credentials, upgrade binaries, run installs or issue arbitrary REST endpoints. L2 and L3 execution remain explicitly authorized operator acceptance workflows. A future write-runner requires a separate ADR, exact disposable-target policy, durable dispatch/outcome receipts, independent observation, and no blind retry.
- **No claimed distributed trace:** D3 inbound gateway and D2 app receipts remain separate stores. Shared timing alone cannot establish a cross-layer parent span.
- **No root app imports:** The app probe runs as the installed non-root service principal. Root-owned host CLI remains independent when MCP is down.
- **No disclosure:** Only allowlisted codes/counts/trace metadata; endpoint addresses, host paths, token bytes and raw error text are never returned. Retain existing D4 exact-Box authorization.
- **Single-operator gateway:** Existing host-wide MCP tool allowlist is honored. Not per-Agent RBAC.

## Architecture

1. `bwmini/mcp_gate.py`: advertises one read-only tool, rejects nonconforming selectors and dispatches locally; the optional host-wide policy still filters it.
2. `bwmini/gate_diagnostics.py`: fixed safe subprocess, bounded result parser, and operation receipt presentation through existing D2.
3. `bwmini/diagnostic_probe.py`: app-user-only `preflight` and `rest_read`; imports `app_runtime`, `provider` and `deploy_check` without changing state.
4. `tests/test_diagnostic_coordinator.py`: tool-schema, fake app results, no-host-write, unexpected error, secret and failure isolation.
5. Package manifest automatically includes the app probe; the root standalone list already contains updated `mcp_gate.py` and `gate_diagnostics.py`.

## Implementation and verification

**D**: this document and current D1/D2/D3/REST contracts. **S**: source inspection of executable path, input schemas, output projection, root-owned release list. **T**: syntax, focused unit/arch guards, all tests and knowledge check; CI check on the exact PR commit. **L**: after isolated build/install, invoke the new MCP tool against that *installed instance* and check a successful read-only PASS and a deliberately refused/incomplete selection. Do not use pre-existing host 0.42.12 as proof of 0.43.0.

Release workflow: branch -> source implementation + tests -> CI -> verified isolated install -> PR merge -> immutable release artifact check -> separate stable pin promotion after installation acceptance. A code merge or successful fixture alone does not satisfy L. Record any blocker in the verification queue. Do not modify Containarium or a live host as part of this slice.

## Next slices (not promised in v1)

1. Operator-approved disposable-Box write acceptance with explicit target lease and durable operations.
2. Safe REST method/route-template/status, upstream request ID (if authenticated contract supports it), and parent-child span joins; never log raw URLs.
3. Typed error root-cause recommendations with evidence provenance and missing-capture markers.
4. Versioned acceptance profiles for install/upgrade/token rotation/SSH/agent conflict, with risk-specific grants.

## Related documents

- [Unified Operation Trace evolution](UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md)
- [D3/D4 authority](DIAGNOSTICS_D3_D4_MASTER.md)
- [D1 implementation](DIAGNOSTICS_D1_IMPLEMENTATION.md)
- [D2 implementation](DIAGNOSTICS_D2_IMPLEMENTATION.md)
- [0.42.12 REST acceptance gates](../verification/PROVIDER_REST_A1_04212_2026-11-03.md)
- [Verification queue](../verification/REMEDIATION_TEST_QUEUE.md)
