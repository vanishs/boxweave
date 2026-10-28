# Real Web-to-MCP chain — verified minimum slice

Date: 2026-10-28 America/Los_Angeles. Sources SRC-049/050. This is a real nested integration test of the v2 minimum application, not a fixture pass, a clean bare-metal installation, or complete phase-4/client-matrix acceptance. Phase 5 feature work has not started.

## Approved environment and identities

- Existing application box: boxweave-v2-test-container, user boxweave-v2-test, 10.206.30.6, outer backend dev/default, creation 1793176101. Its original source, services, fixture configuration/database and approved public route remain unchanged.
- New supporting backend: bw-v2-mcp-nested-container, user bw-v2-mcp-nested, 10.206.30.8, outer backend dev/default, creation 1793212095, 2 CPU / 4 GB / 30 GB. Explicitly approved Podman/nesting option, including its possible privileged/unconfined consequences. Native networkTenant=boxweave-v2-test; fresh Guard reports the same attached ACL as the application. No host-wide setting, old-box setting or new public route was changed.
- Outer create returned a client request timeout. Original-name reconciliation found this exact instance Running; it was not created again. Identity verification returned bw-v2-mcp-nested and bw-v2-mcp-nested-container, exit 0.
- Inner workload: v2-live-hello-container, user v2-live-hello, 10.138.33.2, creation 1793213048, 1 CPU / 1 GB / 8 GB. Exactly one authenticated MCP create was submitted, using this backend's own full managed public key. Its identity and Python command were then checked through MCP connect, not direct Incus execution.

Supporting bootstrap was pinned to 4318da6ea4e4e45c55cf43b64acbdd2ede65fc43b, installer SHA256 c76351524d33f430cdb189897b6107c698d6955a294334ad47c7700b8dcb7718, MCP release 0.114.2. Overall installation limit 1800 seconds; actual completion exit 0, observed by elapsed 753 seconds. The API wait retained its built-in 900-second bound. Incus/Containarium active; actual token/hostkey timers active/waiting/enabled; their completed oneshots Result=success/ExecMainStatus=0; no pending-update marker. PostgreSQL/security inner core instances Running without manual core repair.

Independent application key generated and retained in the application box; public half granted only restrict + command=/usr/local/bin/boxweave-mcp on the new backend. Host public key pinned from authenticated Platform MCP. StrictHostKeyChecking=yes, IdentitiesOnly=yes, BatchMode=yes. Private keys/JWTs were not read/exported. Each nested host retains its own workload identity.

## Actual results

| Check | Actual result | Limit |
| --- | --- | --- |
| initialize/initialized, tools/list, read-only tools/call | Passed; containarium-mcp-server 0.114.2, protocol 2024-11-05, 101 tools; empty inventory accepted before creation | Actual independent launcher as configured non-root user |
| Inner MCP create/get/connect | One create succeeded; original identity observed; whoami/hostname exit 0 | No duplicate create or host trust bypass |
| Inner Python hello | stdout hello world plus newline, exit 0 | Actual MCP connect, not an inferred result |
| New queue-race regression before fix | Reproduced failure: failed/DEADLINE_EXCEEDED | The first real HTTP operation also exposed this race; not erased |
| Linux suite after fix | 30 tests OK, exit 0, 7.597 seconds | Non-root assigned project LXC; provider fixtures in this suite |
| Fixed-candidate real HTTP chain | Passed, operation 6e918550adb84d80b138364c43f01227, evidence live_mcp, stdout hello world, exit 0 | Login/resources/binding/plan/confirm/worker/receipt/restart without replay |
| Fixed-candidate real Chromium chain | Passed, final operation 19de4b38666e4772a3e5ba863e8c9008, evidence live_mcp, stdout hello world, exit 0 | API/worker restart, binding and same receipt reload, one audit entry, desktop/mobile; no script errors/overflow |
| Wrong host pin | SSH exit 255, protocol not started; normal pin file unchanged | Deliberately wrong dedicated test pin, no trust repair |
| Wrong expected workload identity | IDENTITY_MISMATCH, actual tool calls only connect(whoami && hostname), zero hello dispatches | Instrumented client recording actual MCP calls; no fake provider |
| Retention | Temporary loopback 18641/18642 servers stopped; original three demo services active | No reboot/LXC autostart/security certification |

Actual identity stdout:

```text
v2-live-hello
v2-live-hello-container
```

Actual command stdout:

```text
hello world
```

Python 3.12.3, Flask 3.1.2, Waitress 3.0.2, Playwright 1.56.0, Chromium 141.0.7390.37. Supporting bootstrap reports BoxWeave 0.4.3; rewritten candidate is 0.1.1. These are different components. The original published demo continues to show fixture, not this isolated live candidate.

## Queue race repair (0.1.1)

work_one previously sampled now before opening its SQLite claim transaction. A concurrent HTTP confirmation could enqueue after that sample; when the worker then read the queue, now was older than the operation's creation time and was falsely treated as clock rollback. Move the sample after BEGIN IMMEDIATE and selection of the queued row. Persisted deadlines and genuine rollback rejection remain unchanged.

The regression interleaves a legitimate Service.confirm using a separate real SQLite repository immediately before the worker's claim. It fails on the old code and passes after the change. Existing rollback/expiry, unknown/no-replay, identity and session tests also pass. Product code was authored/tested in /home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a, then exact-byte synchronized back. VERSION and Python package metadata are 0.1.1.

The first HTTP receipt 9992a78ccb484edf873fdc9bcfa872c5 records failed/DEADLINE_EXCEEDED with deadline 120 seconds after creation, although the later inspection was only about 25 seconds after creation. Preserve it under before-fix/receipt.json and its original private database. It is not rewritten as success. A separate fresh HTTP state and explicit new harmless operation verified the fix.

Candidate /home/boxweave-v2-test/boxweave-v2-live-candidate was copied from unchanged deployed source after normalized Git-blob identity checks; only the four reviewed code/version files changed. Separate live test databases and loopback ports were used. Browser harness was tightened after visual review to wait for resource/project refresh after restart before capturing mobile evidence; final screenshot shows the persisted binding and receipt.

## Reproduction and artifacts

Inside the assigned development LXC:

```bash
cd /home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a
.venv/bin/python -m unittest discover -s tests -v
```

Inside boxweave-v2-test, retained operational harnesses live under /home/boxweave-v2-test/boxweave-v2-nested-lab. HTTP and browser scripts intentionally initialize private fresh test state; inspect their configured DB paths and choose a fresh path before another explicit test. Do not rerun create_inner.py: its marker means reconcile v2-live-hello on the original backend instead.

```bash
cd /home/boxweave-v2-test/boxweave-v2-live-candidate
PYTHONPATH=$PWD ../boxweave-v2-app/.venv/bin/python ../boxweave-v2-nested-lab/http_check.py
PYTHONPATH=$PWD ../boxweave-v2-app/.venv-lab/bin/python ../boxweave-v2-nested-lab/browser_check.py
```

These are reproduction commands with the fresh-state prerequisite above, not permission to replay an old operation. Actual runtime captures: protocol-result.json, http-result.json, browser-result.json, failure-gates.json, versions-source.json, live-desktop.png and live-mobile.png in that private lab directory. Sanitized machine-readable summaries and four exact source hashes are in [the retained result](nested-live-results.json). Ignored local review copies/screenshots are under .runtime/nested-live-20261028; no database, key, password or token is included in Git.

## Remaining boundaries and next step

- TASK-005 real Web/worker/MCP/command/receipt subset is now verified. Full actual Windows-native enrollment/registration and Agent UI acceptance are still unverified; no broad client/release support claim is made.
- Developer architecture/slice acceptance remains a separate gate. SRC-048 explicitly required the real chain before phase 5 and rejected a live-test deferral; this evidence fulfills the chain prerequisite, not a fabricated whole-stage acceptance. No Prompt Graph or other phase-5 feature was started.
- Old non-nesting bw-v2-mcp-test remains unchanged and incomplete for nested execution. This test succeeds in a separately approved nesting-enabled box; it is not a universal confinement claim or repair of the old box.
- The inner create requested enable_podman=false, but readback reports podmanEnabled=true. The test used only fixed read-only commands. Exact cause was not diagnosed; retain this provider compatibility limitation (RISK-020), do not claim the requested flag was honored or silently repair it.
- No fresh Windows/WSL enrollment, model request, Tenant policy edit, public exposure, bare-metal install or reboot was performed. New backend and inner workload remain available; original demo route remains unchanged. No push occurred.