# D0 implementation specification — truthful diagnostics and exit contracts

Status: **implementation-ready proposal; documentation only, not deployed**. Research baseline: `main@817273d98d8823fcd8489eb0d3f329a67929b91d` (BoxWeave 0.21.4). Proposal origin: [DIAGNOSTICS_PROPOSAL](DIAGNOSTICS_PROPOSAL.md) and [source audit](../verification/DIAGNOSTICS_SOURCE_AUDIT.md). Precedes [D1](DIAGNOSTICS_D1_IMPLEMENTATION.md) and [D2](DIAGNOSTICS_D2_IMPLEMENTATION.md). The maintainer must approve code changes before implementation. Use current code and `AGENTS.md` if main advances.

**Implementation status:** Implemented in 0.21.5 (f57598a, c9fc1cd, 1aeafe1) and live-checkpointed on boxweave-test-ser; see REMEDIATION_TEST_QUEUE.

## 0. Acceptance objective and non-goals

**Objective:** no command says that its selected diagnosis **passed** when a required check was skipped, unavailable, or inconclusive. Define and test the exact difference between *reachable*, *healthy*, *authenticated-read-passed*, *workload-ready* and *unverified*. Preserve the existing restricted SSH, upstream MCP and installed CLI interfaces.

In scope:
- Installed host `boxweave doctor` and `boxweave doctor --mcp`, read-only `status`;
- app package health and deploy_check probe result normalization;
- generated Python 3.11+ client's `--diagnose` in all configured App/runtime/transport profiles;
- tests and operator-facing accurate statuses.

Out of scope: diagnostic DB tables, Trace ID, report command, MCP tools, journal collection, selftest mutation, parent-host privilege discovery, credential re-enrollment and any host change. Do not alter selected supported App catalog, URL publishing, resource creation or existing operation receipt semantics.

## 1. Source-to-path review before writing code

| Layer | Current behavior / source | D0 action |
| --- | --- | --- |
| Host executable | `online.py` publishes root-owned `/usr/local/bin/boxweave` shell wrapper calling `/usr/bin/python3 -I /usr/local/lib/boxweave/default/host_cli.py` | Keep isolated root-owned entry; no app-owned imports at root |
| Host descriptor | `host_cli.public_state` validates bounded root-owned `client-public.json`; does **not** contain a unit list | Keep compatibility; don't invent units from arbitrary user input |
| Host status | `host_cli.status` reports HTTP loopback reachability, `authenticated=false`, `app_health=not_checked` | Do not transform HTTP 401 into authentication or whole-app readiness |
| Host doctor | `host_cli.main` checks credential/trust timers and `user@UID.service`, calls optional `mcp_probe`, then always sets code 0 | Replace unconditional success and specify scoped checks |
| App-user health | `distribution.health` checks installed manifest, declared systemd user units, loopback HTTP, optional `deploy_check --mcp`; status is starting/failed/incomplete/ready | Reuse this trusted **app-user** validation via fixed invocation, not duplicated root parsing of private state |
| App MCP probe | `deploy_check.read_provider` performs initialize, tool discovery and `list_containers`, no execution | Expose typed success/failure separately from administrator initialization |
| Generated client | `client_profiles.run_diagnose` counts only failed/mismatch, not missing/absent/differs/manual; may print last 200 bytes of raw SSH stderr | Gate required checks, classify transport failures, do not disclose raw stderr |
| Source-only management | `manage.py` has a separate `doctor` for source/package workflows | Keep separate compatibility; share terminology/exit contract, not privilege implementation |

Scope note: `boxweave box diagnose NAME` currently classifies narrow Incus/nesting log signatures; it does not confirm parent policy. D0 may improve explicit unknown states but must not start using unreviewed parent access.

## 2. Diagnostic vocabulary, result shape and priority

Use an internal typed normalization function (for example `bwmini/diagnostic_status.py` **only if it can be safely packaged into both trusted origins**; otherwise two small pure functions with shared golden-vector tests). Do not make the installed root CLI import untrusted app modules. No new external dependency.

Each check:

```json
{
  "id": "app_worker",
  "state": "not_verified",
  "code": "APP_UNIT_VISIBILITY_UNAVAILABLE",
  "scope": "app_user_systemd",
  "required": true
}
```

Only fixed, allowlisted `id`/`code`/`scope`. State enum: `passed`, `failed`, `not_verified`, `not_checked`. `not_checked` is valid only when the check is outside the invoked command's promised scope, or `--mcp` was not requested. `not_verified` means in-scope but impossible to establish. `failed` requires a definite negative finding (e.g. observed inactive/failed selected unit or confirmed mismatch), not merely command launch failure.

Aggregate:
- `passed` iff every required check is `passed`;
- `failed` iff at least one required check is `failed`, with remaining statuses still reported;
- otherwise `incomplete` iff some required check is `not_verified` or `not_checked`;
- `not_installed`/`invalid_install` are separately typed metadata errors; neither can return 0.

Exit codes for **doctor and generated client diagnose**:
- `0`: all promised required checks passed;
- `1`: at least one definite failed required check;
- `2`: incomplete, unavailable prerequisites, invalid/unsafe invocation or unknown.
Classification precedence: definite failure (1) beats incomplete (2), but list both findings so information is not hidden. For a failed trust/pin check, do not run network probes; they are marked `not_verified` with `blocked_by`. Existing API/application error codes are unchanged. `status` remains an informational, explicitly limited snapshot; preserve existing standard successful invocation behavior unless its trusted descriptor is missing/invalid.

For machine compatibility keep existing top-level fields (`web`, `mcp`, `services`, `user_manager` and, where present, `state`) and introduce `diagnostic_schema:1`, `checks`, `overall`, `coverage` without deleting valid previous keys. Where legacy `state` has a distinct meaning (e.g. `installed_public_descriptor`), keep it and use `overall` for the new health result. Do not silently rename an already published response field.

**Coverage is mandatory:** record selected scope, which checks were attempted, which were omitted and why, and that `agent_ui`, model inference, reboot/autostart, effective nesting and runtime workload execution were **not verified**. A successful MCP `list_containers` read does not prove target execute/SSH readiness.

Example, read-only host with HTTP 401 and stopped worker:

```json
{
  "diagnostic_schema": 1,
  "overall": "failed",
  "web": {"reachable": true, "authenticated": false, "http_status": 401},
  "mcp": {"state": "not_checked"},
  "checks": [
    {"id":"web_loopback","state":"passed","code":"HTTP_REACHABLE","required":true},
    {"id":"app_worker","state":"failed","code":"APP_UNIT_INACTIVE","required":true},
    {"id":"authenticated_mcp","state":"not_checked","code":"NOT_REQUESTED","required":false}
  ],
  "coverage": {"workload_execute":"not_verified","effective_nesting":"not_verified"}
}
```

Illustration only; tests must freeze the final exact output contract before shipping.

## 3. Host doctor implementation plan

### D0-1: prepare a dedicated read-only app-user probe

Host-safe code must **never** execute application-owned code with effective UID root. Add a fixed internal function in `host_cli.py` that:
1. obtains `info` only from validated root-owned public descriptor, with fixed application user `boxweave` and known absolute prefix;
2. checks `pwd.getpwnam` and that the runtime prefix/current executable is in the expected installation boundary; rejection is an incomplete check, never an attempt to repair;
3. if operator has no permission to become the application user, returns `not_verified` and a safe next action (not `passed`); non-root `doctor` must remain useful for public/systemd checks;
4. under explicitly approved privilege (`sudo boxweave doctor`), invokes `runuser -u boxweave -- env XDG_RUNTIME_DIR=/run/user/<trusted UID> <expected app python> -m bwmini.distribution health --prefix <fixed prefix>` via argument-array subprocess, with bounded stdout (e.g. 16 KiB), stderr discarded, no shell, fixed deadline, and working directory inside the installed app release;
5. parses only a strict, versioned, bounded result; verifies expected release, service list and health status, no arbitrary paths or dynamic command from emitted data;
6. if the isolated app helper cannot run, preserves public host findings but marks app health `not_verified`. Never execute the same helper as root as a fallback.

`distribution.py` CLI entry accepts `health --prefix ... [--mcp]`. For `doctor --mcp`, prefer **one app-user health/probe transaction** if equivalent to the existing `host_cli.mcp_probe`, or keep the latter but make their scopes explicit. Avoid accidental double MCP probes that could be slow and create conflicting snapshots.

Check application-managed user services using `installed.json.units` **only after** the app-owned reader validated installation provenance. `distribution.health` already knows these names. An empty unit list must not be reported ready. Do not guess a systemd unit name from the app version or infer a unit's health from user-manager `active`. A unit could exit/crash while the timer and user manager remain active.

For `doctor` without MCP, required checks: trusted public descriptor, declared installed application release/provenance (when app-user view available), selected serve/worker units, loopback Web reachability, and installed host-managed timer/user-manager observations. An inaccessible app-user view implies `incomplete`, not `passed`. In `doctor --mcp` add the authenticated list-read as required, plus initialized-admin proof as a distinct check; do not classify live workload or TLS.

### D0-2: robust host command observations

`service_observation` currently reads `LoadState,ActiveState,UnitFileState`; make timeout and missing unit explicit. Differentiate "observed inactive/failed" from `systemctl` unavailable/nonzero. For timers verify expected `active/waiting/enabled` and distinguish oneshot trust service from timer. A parent user manager running is not equivalent to BoxWeave app services running.

System control commands remain read-only; no `systemctl start/restart`, no install, no `incus config set` or automatic parent access. Do not read raw app config as root. Handle `pwd.getpwnam` failures, malicious/stale descriptors, subprocess EOF, excessive output, malformed JSON, timeout, process nonzero and incomplete installation with fixed codes.

### D0-3: app-side health and deploy-check hygiene

`distribution.health` currently treats HTTPError 401 as reachable, and `web.authenticated=false`; preserve that. Ensure requested `--mcp` failure doesn't accidentally leave a `ready` status or a success exit. For malformed probe output, missing `admin_initialized`, missing `mcp.state` or an inaccessible SQLite DB, return a typed incomplete/failure with omission reason; never print secret-bearing `subprocess` details. The root probe observes app-side state **as the app user** rather than parsing private database state.

`bwmini/manage.py` is a different source-distribution management entry. Confirm its prior exit behavior in tests and document any changed response. Do not wire `boxweave.sh` to bypass the installed host entry. Both surfaces can share a testable status vocabulary but retain their own authority.

## 4. Generated client diagnose plan

### D0-4: explicit stage graph

Generate one canonical array of stages: `python`, `client_tools`, `identity`, `ssh_transport`, `mcp_initialize`, `mcp_tools`, `mcp_read`, `registration`. On missing local key/pin, all downstream stages are `not_verified` with `blocked_by=identity`; do not leave them absent and count them as success. Do not create keys during `--diagnose`.

The required stage set depends on profile semantics:
- auto-managed registration profiles: every stage above required;
- manual integration profiles such as Qoder/TRAE: connection stages required, `registration` marked `manual/not_verified`; overall `incomplete` until user-mediated application registration is independently checked, with `connection_overall` separately `passed` if the MCP probe passed.
- user interface recognition and inference are always `not_verified` unless independently observed; never mark end-to-end passed from transport alone.

If a script legitimately needs an application-specific parser (e.g. Codex TOML), use already reviewed existing profile path; do not equate JSON missing/invalid to a registered success. Preserve unmodified unrelated client config and the generated Python shebang-first output (`#!/usr/bin/env python3`).

### D0-5: bounded MCP handshake verification

Maintain strict pinned host-key verification, SSH keepalive options, public-key-only transport, explicit WSL distro selection. Probe `initialize`, `notifications/initialized`, `tools/list` (including pagination where required), and `list_containers`. Validate JSON-RPC structure and IDs; require an actual tool success shape, not merely any nonempty `content` or exit 0. Limit total bytes, line length, messages and wall-clock timeout; avoid buffered process deadlock (stdout/stderr must be drained concurrently or stderr diverted to DEVNULL).

Never put arbitrary SSH stderr into JSON or human-readable status (even a 200-character suffix may contain secrets). Replace with `SSH_TRANSPORT_FAILED`, `MCP_HANDSHAKE_INVALID`, `MCP_READ_FAILED` etc, including safe exit class/timeout category. Keep underlying full output out of automatic report scope. Detect stale MCP stream separately from nonexistent server where possible, but do not claim exact root cause without evidence.

## 5. TDD / concrete regression matrix

Add or extend `tests/test_host_cli.py`, `tests/test_distribution.py`, `tests/test_manage.py`, `tests/test_client_setup.py`, `tests/test_public_profiles.py` and relevant generated-script/profile tests. Use synthetic fake processes, temporaries and patched path validators, not live SSH.

| Case | Setup | Required assertion |
| --- | --- | --- |
| D0-T01 | HTTP 401; worker stopped, timers healthy | web reachable, unauthenticated; doctor overall failed/exit 1 |
| D0-T02 | Healthy timers + no app-unit visibility | overall incomplete/exit 2; no root app import |
| D0-T03 | Missing public descriptor, wrong owner, writable parent, symlink | safe nonzero, no executable from untrusted path |
| D0-T04 | Non-root `doctor --mcp` | MCP not verified, exit 2; never runs app code as root |
| D0-T05 | Subprocess app health timeout/EOF/oversize/malformed JSON | explicit incomplete and bounded output, no crash |
| D0-T06 | Actual app-user health says unit failed, MCP passes | result failed/exit 1; never masked by successful MCP |
| D0-T07 | App health says ready but MCP requested and rejected | nonzero, no false authenticated success |
| D0-T08 | Client no key/pin or partial pair | missing + downstream blocked; never overall pass |
| D0-T09 | Pin mismatch | fail closed; zero SSH calls |
| D0-T10 | SSH exits 0 with malformed/empty JSON or protocol mismatch | failed/incomplete, not passed |
| D0-T11 | MCP response tools/list returns no read tool or isError | no false success |
| D0-T12 | Manual profile connects but registration unverified | `connection_overall=passed` while `overall=incomplete` |
| D0-T13 | All required stages valid, registered auto profile | exit 0; UI/inference not_verified |
| D0-T14 | Fake stderr embeds token/private key/control characters | no substring leaks to JSON/stderr |
| D0-T15 | Generated nine-App native/WSL variants | scripts compile and still have shebang byte zero; registrations unchanged |
| D0-T16 | `status`, `provenance`, `box inspect` and package management | existing supported behavior unchanged; no new mutations |

Treat D0-T15 as a matrix across the **actually supported** App/OS/transport combinations, not all imaginary cross products; preserve the alphabetical App catalog. Compare existing output and update compatibility snapshots only for intentional, documented changes.

## 6. Developer work items and delivery gates

- **Commit D0-a (independent):** status schema/aggregation + host doctor implementation and focused tests; document exit behavior.
- **Commit D0-b:** generated client diagnose stage graph and malicious-input tests; regenerated script parity and safe SSH behavior.
- **Commit D0-c (only if necessary):** reconcile `distribution.health`/`deploy_check`/`manage.py` paths, document actual CLI differences; avoid a broad refactor.

Gate per commit: run `python scripts/check_knowledge.py` and relevant unittest modules in a verified prepared non-root checkout; run `.venv/bin/python scripts/validate.py` when that environment is approved. No sudo live tests, host mutation, Provider create, existing-key enrollment or client overwrite without separate environment approval. Record exact source SHA, environment, commands, exit codes, secrets scan, and unresolved checks in `REMEDIATION_TEST_QUEUE.md`.

**Done** only when negative tests above pass, no compatibility break is undocumented, and a smoke of the installed wrapper (in an authorized disposable environment) distinguishes a stopped app worker from a live MCP read. Static or CI fixture success does not prove actual installed root/app-user behavior. Trace IDs and reports remain unimplemented until D1/D2.

## 7. Failure handling and rollback

No schema migration or new services in D0. Rollback uses the existing reviewed distribution upgrade/repair and manifest process; preserve original source, configuration, credentials, user units and receipts. If health collection fails, fail diagnostically (`incomplete`), **never** escalate scope or retry a user mutation. Privileged checks require explicit operator action. Record incomplete host privilege/test coverage honestly.

## 8. Cross-slice handoff

D0 publishes a stable diagnostic state/coverage vocabulary for D1 event classification and D2 reports. Keep check IDs and codes fixed and bounded. D1 may add `trace_id` without changing D0 pass criteria. D2 must not infer app health from the host-only `status` result. D3 existing-MCP tool is excluded.
