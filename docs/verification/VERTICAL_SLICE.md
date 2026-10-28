# Phase 4 vertical-slice evidence

Date: 2026-10-28 America/Los_Angeles. **Partial completion, not a verified live end-to-end system.** Developer approved the concrete execution plan, then confirmed there is no independent application launcher and requested preservation of current results (SRC-030, SRC-032).

## Environment and isolation

All product implementation, dependency installation and Linux tests were performed through Platform MCP inside the approved boxweave-tenant-dev LXC, as non-root user boxweave-tenant-dev, hostname boxweave-tenant-dev-container. `whoami && hostname` returned those exact two lines with exit 0. New source root: `/home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a`. Ubuntu 24.04 / Python 3.12.3 / Linux x86_64 / Node 18.19.1 / npm 9.2.0. Original remote project `/home/boxweave-tenant-dev/work/boxweave` remains at 7818668ea1225ebbd0763b8215db6fa666dd775e with its preexisting 32 status entries, untouched.

Local implementation files were synchronized back only after remote authoring/testing. All 28 source/lock/readme files matched remote SHA256; see [manifest](phase4-source-hashes.json). No credentials, raw old project files or databases were copied into Git. The new version is 0.1.0, an unreleased minimum-system implementation on the orphan branch.

## What actually runs

Real Flask/Waitress HTTP + React/TypeScript interface + SQLite + separate API/worker processes: local administrator sign-in -> select explicitly labeled fixture environment -> save project/workspace binding -> preview exact fixed hello plan -> one-use confirmation -> durable operation receipt -> API and worker restart -> query the same receipt without a new hello call. The fixture supplies infrastructure behavior; browser, HTTP, application, DB and restart are real.

The UI prominently states fixture mode and receipts say evidence=fixture. A missing live launcher fails as MCP_LAUNCHER_UNAVAILABLE before new state creation; live mode never falls back to a fixture. Authentication, Origin/CSRF checks, private state paths, revisions, one-use plans, actor-bound receipt lookup, worker lock and unknown/no-replay recovery are implemented for this slice. No arbitrary Web command entry exists.

## Actual commands and outcomes

Commands below ran from the remote source root unless marked Windows. Dependency/install phases were bounded by `timeout 600`; browser runs by `timeout 120`. Background exit files were inspected; no transport timeout was treated as an exit code. Readiness succeeded immediately, so no retry/deadline reset occurred.

| Command/check | Actual result | Evidence class and limit |
| --- | --- | --- |
| `.venv/bin/python -m unittest discover -s tests -v` | 29 tests, OK, exit 0, final run 6.648 seconds | Real Linux unit/integration tests; infrastructure synthetic |
| Fresh `.runtime/repro-venv` install with `--no-index --find-links .runtime/wheels --require-hashes -r requirements-dev.lock` | exit 0; pip check: No broken requirements found | Clean virtual environment, exact downloaded wheel hashes; Ubuntu/CPython 3.12/x86_64 only |
| `.runtime/repro-venv/bin/python -m unittest discover -s tests -v` | 29 tests, OK, exit 0, 6.671 seconds | Same suite reproduced under the locked test environment |
| `(cd web && npm run build)` | TypeScript + Vite 6.4.5 production build, exit 0 | Actual build; frontend dependencies in package-lock |
| `PYTHONPATH=. timeout 120 .venv/bin/python scripts/browser_check.py` | exit 0, Chromium passed; one hello fixture call before/after restart; no mobile horizontal overflow | Real browser/application/SQLite with fixture infrastructure; not live Provider acceptance |
| `.venv/bin/python -m compileall -q bwmini tests scripts` | exit 0 | Syntax/compilation only |
| `.venv/bin/python -m pip check` | exit 0, no broken requirements | Dependency consistency, not vulnerability certification |
| Windows `powershell.exe -NoProfile -ExecutionPolicy Bypass -File tests/windows_connection.ps1` | exit 0; AST + 3 no-network rejection cases passed | Actual Windows PowerShell 5.1.26100.5651; identity missing, pin mismatch, invalid input. No real SSH or Agent recognition |
| Platform MCP connect: `whoami && hostname && python3 -c "print('hello world')"` | exit 0, stdout below | Real direct Platform MCP target command; separate from application worker |
| Live application prerequisite | MCP_LAUNCHER_UNAVAILABLE; checked managed launcher/backend paths absent | No reviewed independent application launcher/credential; developer confirmed none available |
| `ss -ltn '( sport = :18640 )'` after tests | Header only; no listener | Owned API/worker processes stopped; no route/public/private-address publication |

Actual direct Platform MCP stdout:

```text
boxweave-tenant-dev
boxweave-tenant-dev-container
hello world
```

Do not combine that direct call with the fixture browser result into a fabricated live Web->worker->MCP pass. No independent authenticated application MCP initialize/tools/call session ran against real infrastructure. Synthetic stdio peers exercised those protocol messages, not the missing credential plane.

## Meaningful negative coverage and mappings

| REQ / AC / TASK | Executed evidence | Status |
| --- | --- | --- |
| REQ-004 / AC-004 / TASK-001, TASK-002 | `test_target_change_before_dispatch_has_zero_hello_calls`, scoped target contract, direct real target identity observation | Component/fixture verified; app-to-live target unverified |
| REQ-010 / AC-010 / TASK-002, TASK-004 | persisted binding, stale revision/rebind with existing plan, browser save/reload | Minimal create/rebind component verified; unbind/full workflows not implemented |
| REQ-005, REQ-006, REQ-007 / AC-005, AC-006, AC-007 / TASK-002 | duplicate confirm rejection, expired deadline/backward clock, dispatch-intent loss -> unknown, response loss not replayed, actor-bound receipts, worker lock | Slice state/receipt tests verified; real create absent and deliberately not implemented yet |
| REQ-011, REQ-018 / AC-011, AC-018 / TASK-004 | HTTP auth/Origin/CSRF/session expiry/login limits/body limits, reject extra command field, browser plan/confirm | Minimal fixture interface verified; production HTTPS/independent service identity not installed |
| REQ-002, REQ-003 / AC-002, AC-003 / TASK-003 | synthetic MCP identity mismatch, malformed/denied calls, wrong IDs/repeated pagination; Windows pin-mismatch rejection | Partial adapter verification only; no live native transport/registration or enrollment |
| REQ-016 / AC-016 / TASK-003 | hashed Python locks + exact source synchronization; invalid live config creates no state | Packaging groundwork; full enrollment payload verification/distribution remains unimplemented |
| REQ-017 / AC-017 / TASK-005 | explicitly separated fixture/direct-live/Windows-preflight reports | Evidence honesty verified; selected real-client/live acceptance NOT satisfied |

The 29-case suite includes malicious guest marker rejection, unsupported live-launcher rejection, unsafe state symlink/schema rejection, malformed output not stored, and no HTTP arbitrary command field. These are actual tests, not source-mirroring document tests. Failure states simulated at the dispatch boundary do not constitute physical power-loss testing or a real-provider exactly-once guarantee.

## Screenshots and retained artifacts

Inspected [desktop](assets/slice-desktop.png) and [mobile](assets/slice-mobile.png) screenshots are fixture browser results. The final remote browser evidence directory is `/home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a/.runtime/browser/a7935cc783d05e21` (private synthetic review state retained). Build/test logs and explicit exit files are in the remote source root. No real secret payload is included in committed summaries.

## Known failures, adjustments and next prerequisite

An initial prerequisite check loaded the placeholder live config before checking its launcher and failed with PermissionError on its placeholder database path. Load order was fixed to validate the launcher before any state creation; regression `test_invalid_live_config_does_not_create_state` passes. Browser isolation was improved to unique private state per run. Unexpected provider output is discarded rather than stored raw. These fixes are included in the final 29-test run.

The planned real phase-4 path is incomplete: no reviewed independent application launcher/credential, no real Windows native SSH/Agent recognition, no full enrollment or trusted setup-payload distribution, and no production HTTPS/service-user boundary. Current probe is connection-only and says registration/UI unverified. The application provider parser supports only explicit tested formats/protocol; installed-provider compatibility remains unverified.

No container lifecycle, provider upgrade, network policy, route publication, GLM inference, global trust or existing client registration was changed. No persistent services were created. The developer requested preservation rather than adding credentials. Next continuation must separately establish an independent protected application MCP launch path, then prove the real selected-client/application slice. Do not ask for token/private-key text. Phase 4 exit gate and developer architecture acceptance remain unsatisfied; phase 5 must not start.
