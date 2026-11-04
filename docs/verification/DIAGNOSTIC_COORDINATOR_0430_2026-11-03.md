# DC-01 v0.43.0 — agent-safe diagnostic coordinator acceptance ledger

**Date:** 2026-11-03 (America/Los_Angeles). **Scope:** standalone Agent Gateway tool + application-owned read-only subprocess. **Release:** v0.43.0 candidate; default `PUBLISHED_RELEASE` remains independently accepted v0.42.1 until new installed-host gate.

## Implemented source contract

- `boxweave_diagnostic_run(profile=preflight)`: fixed installed app Python, cwd, persistent/runtime REST binding, mode; no remote mutation.
- `profile=rest_read`: independent real read-only REST inventory and legacy MCP `list_containers` from the app identity; both must pass to report PASS. No container list is returned.
- `profile=operation_trace`: actor-scoped existing D2 app operation ID/trace and at most 25 classified events, including API spans for environment actions.
- Strict selector (no caller-supplied commands, paths, tokens, URLs or Box identities), host-wide tool policy and optional protected gateway both enforced. The app probe runs without host-root authority. Result schema and error codes are allowlisted; no stderr, raw error response, JSON request/response, token or full endpoint.
- Absent/unsupported input, missing app interpreter or configuration, pre-migration/missing trace and probe timeout return typed denial or `not_verified`. A remote write is never attempted or blindly retried.

## Evidence by grade

| Grade | Environment and exact action | Result |
|---|---|---|
| S | Reviewed `bwmini/mcp_gate.py`, `gate_diagnostics.py`, `diagnostic_probe.py`, `gateway_protection.py` and `online.RELEASE_FILES` trust/packaging boundaries | Source inspected; no new root helper required |
| T | Fresh task-owned `bwdiag-0430-ci` Ubuntu 24.04 Box, `python -m unittest discover -s tests -q` | PASS, **699 tests**, exit 0 before protected-gateway extension; rerun required for final revision |
| T | Same isolated Box, `python scripts/check_knowledge.py` | PASS, exit 0 |
| T | Same isolated Box, `python scripts/validate.py` with prepared locked Python + npm dependencies | PASS, 3 steps: knowledge, unit, Web build |
| T | Same Box `npm --prefix web ci --ignore-scripts --no-audit --no-fund` and `npm --prefix web run build` | PASS, TypeScript/Vite build |
| T | Built `bwmini.distribution build` with locked offline wheels; extracted application ZIP; imported `bwmini.diagnostic_probe` using the **bundled** Python environment | PASS; version 0.43.0 |
| T | Executed bundled `gate_diagnostics.query_coordinator` using real subprocess, synthetic authenticated loopback HTTP service, synthetic MCP process and private test token | PASS: preflight, REST and MCP read, exact GET count 1, no POST, no secret in responses, MCP-only fallback `not_verified` |
| T | Invalid paths/selectors, injection, crash/timeout, secret canary, D2 trace linkage, protected gateway policy | Unit test profiles; no real Provider operations inferred |
| CI | PR GitHub Actions and final commit provenance | PENDING until PR created and check observed |
| L | Installation into isolated **real Containarium** host and authenticated `boxweave_diagnostic_run` from external Agent transport | **NOT VERIFIED**; packaged synthetic test is not installed Provider acceptance |
| L | Owner's running `dev` host still using BoxWeave 0.42.12 | **UNCHANGED**; do not claim v0.43.0 live acceptance |

## Defects caught and corrected

1. Legacy MCP-only config initially classified as binding drift; corrected to report REST `not_verified` while independently checking MCP.
2. The existing Agent catalog fixture assumed nine helper tools; updated to account for the new tenth tool.
3. Optional protected gateway restricted tool catalog; added the read-only coordinator to its fixed allowed set, retaining denial of unknown/dynamic tools.

## Release/rollback

Candidate source changes are additive: no schema migrations, no upstream Containarium upgrade, no new privilege grant, no host service config changes, no Gateway rewrite, no automatic mutation. Existing `boxweave_diagnose`, D2/D3/D4 and MCP legacy paths retain their contracts. Reinstall the prior immutable source tag to roll back, with the ordinary installer-owned backup/ownership checks; do not manually replace running helper files.

**Gates:** final-source 699+ tests, knowledge, package/manifest, protected gateway, PR CI, immutable release, and separately a live installed read-only probe are all distinct. Do not promote `PUBLISHED_RELEASE` merely because an isolated synthetic probe or GitHub release succeeded.

## Next-agent read-only installed smoke after candidate release

On a specifically authorized isolated BoxWeave host with v0.43.0 installed, invoke through a newly reconnected Agent MCP:
1. `boxweave_diagnostic_run({"profile":"preflight"})` and expect explicit installed runtime path and REST binding PASS.
2. `boxweave_diagnostic_run({"profile":"rest_read"})` and independently verify REST + MCP authenticated read PASS.
3. `boxweave_diagnostic_run({"profile":"operation_trace","operation_kind":"environment","operation_id":"EXACT-OWNED-ID"})` and verify D2 trace correlation; do not replay a write to manufacture a trace.
4. Send rejected extra args and ensure `COORDINATOR_SELECTOR_INVALID` before any upstream dispatch; repeat with optional protected Gateway enabled if deployed.
5. Verify no secrets in Agent response and no changes to Boxes/token/config. Compare installed source/running version and check stable pin separately.

## References

- [Design and contract](../design/DIAGNOSTIC_COORDINATOR_V1.md)
- [Unified tracing evolution](../design/UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md)
- [Remediation validation queue](REMEDIATION_TEST_QUEUE.md)
- [0.42.12 REST source/host gate](PROVIDER_REST_A1_04212_2026-11-03.md)
