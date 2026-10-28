# BoxWeave minimum system

**Phase 4 minimum system: real Web-to-MCP hello/receipt verified in an approved nested lab; developer/client acceptance remains incomplete.**
This is a new implementation on rewrite/v2, not a port of the legacy tree.

The current slice provides operator login, reviewed project/environment binding,
one-use hello plans, a separate worker, SQLite revisions and durable receipts.
Fixture mode is prominently labeled. Missing live configuration never silently
enables a fixture. Tenant writes, ports, model setup and the wider
client matrix remain required later features, not implemented controls.

## Verified environment

Development and Linux tests ran as the non-root boxweave-tenant-dev account in
the approved project Ubuntu 24.04 LXC: Python 3.12.3, Node 18.19.1, npm 9.2.0.
Runtime/test wheel locks target CPython 3.12 on Linux x86_64; do not claim they
are universal Windows/macOS locks. Frontend dependencies are pinned by npm lock.
Chromium/system library availability is a prerequisite for browser tests;
the scripts do not install privileged operating-system packages.

## Reproduce isolated verification

Run from this checkout inside the assigned Linux project environment:

```bash
python3 -m venv .venv
.venv/bin/pip install --require-hashes -r requirements-dev.lock
(cd web && npm ci --ignore-scripts --no-audit --no-fund && npm run build)
.venv/bin/python -m unittest discover -s tests -v
.venv/bin/python -m playwright install chromium
PYTHONPATH=. .venv/bin/python scripts/browser_check.py
```

The browser check creates private unique test state under .runtime/browser,
uses real Chromium/HTTP/application/SQLite and synthetic infrastructure,
restarts only its owned API/worker processes, verifies no replay and stops them.
It refuses an occupied loopback port 18640. Screenshots and state remain there
for review; no resource, policy, route, host trust or client registration changes.

## Run a local fixture for review

From the repository root, after building/installing as above:

```bash
.venv/bin/python -m bwmini init-admin --config examples/fixture.json
.venv/bin/python -m bwmini worker --config examples/fixture.json
# In a second terminal:
.venv/bin/python -m bwmini serve --config examples/fixture.json
```

Administrator initialization is interactive (12+ characters); no password in
arguments. Open http://127.0.0.1:18640 in that environment. Sign in, name a
project, save a binding, preview and confirm hello, then load the receipt.
This loopback fixture deliberately uses HTTP and non-Secure cookies; it is not
a production HTTPS deployment. Stop these foreground processes when finished.
The database parent must be private; symlinked/shared state is rejected.

CLI observation uses the same services and DB:

```bash
.venv/bin/python -m bwmini projects --config examples/fixture.json
.venv/bin/python -m bwmini receipts --config examples/fixture.json
```

## Live prerequisite

examples/live.json is a placeholder, not an operational or credential-bearing
configuration. A separately reviewed independent application MCP launcher,
protected runtime credential and exact target identity are required. The
original project container had no verified launcher at the first check. A later
approved independent nesting-enabled lab now verifies real MCP, HTTP and
Chromium hello/receipt/restart behavior with version 0.1.1; see
[actual live-chain evidence](docs/verification/LIVE_MCP_CHAIN.md). The bundled
example remains a placeholder and fails before creating state. No existing
Agent token/private key was borrowed or exported. The original public demo
remains fixture; this live result comes from isolated loopback test instances.

Live mode allows only explicitly listed targets, compares scoped creation/name
identity on observation, initializes/discovers/calls MCP, verifies exact
whoami/hostname before fixed hello, and rejects unsupported wire formats.
Its current parser accepts the observed exit_code/stdout envelope or structured
stdout/exit_code result, not arbitrary guest markers. The synthetic peer tests
do not prove installed provider compatibility. Only protocol 2024-11-05 has
fixtures; other versions fail closed.

## Windows connection probe

clients/windows/diagnose-connection.ps1 consumes an operator-reviewed public
descriptor, existing private-key path (presence only; never reads its contents),
and existing pinned public known_hosts. It uses native Windows OpenSSH with
BatchMode/strict trust and performs initialize/initialized/discovery/read-only
call. It does not enroll or edit registration/trust. Even a connection success
reports registration/UI unverified and overall incomplete. Cleanup requests
local SSH termination; remote descendants are not certified cancelled.
The descriptor example contains placeholders.

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File tests/windows_connection.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File clients/windows/diagnose-connection.ps1 -Descriptor C:\reviewed\public-profile.json
```

Only AST and three no-network rejection cases passed on Windows PowerShell 5.1.
Real native SSH/Agent recognition and enrollment remain unverified. Full
enrollment, setup-payload integrity distribution and WSL profiles are not
completed by this connection probe.

## Evidence and limits

See [vertical-slice evidence](docs/verification/VERTICAL_SLICE.md),
[task status](docs/process/IMPLEMENTATION_PLAN.md) and
[handoff](docs/process/HANDOFF.md). Linux tests cover normal/failure state paths,
auth/CSRF, target change, stale revision, no-replay recovery, MCP pagination and
malformed responses. No real create/policy/route/model call or production
installation/HTTPS/autostart/confinement acceptance is claimed. Phase 4 remains
partial until the real selected-client/application path and developer gate pass.

## Prompt Graph (0.2.0)

After login choose Prompt Graph. Create/edit a named graph, drag cards, add/edit comments and undirected relations, and save explicitly. Project/environment/Tenant context is text only and never executes infrastructure. Revision conflicts keep the local draft; reload explicitly to reconcile. Save before copying or exporting Markdown. Recognized secrets are redacted and Markdown is escaped; review exports because redaction is not complete secret detection. Clipboard failure never reports a successful copy. Delete removes graph data only.

Schema 2 migrates schema-1 data transactionally. Back up persistent SQLite state consistently before an update; older binaries refuse schema 2. This iteration is source/test delivery, not an update of the previously published demo. See [graph API, limits and actual tests](docs/verification/PHASE_05_TASK_011.md) and [ADR-006](docs/design/ADR/0006-prompt-composition.md).

```bash
PYTHONPATH=. .venv/bin/python scripts/graph_browser_check.py
```

The graph browser check uses owned loopback port 18643, real Chromium/HTTP/SQLite and fixture infrastructure, asserts zero execution operations, and stops its temporary server. Browser/system dependencies must already be installed as described above.

## Reviewed existing-identity native Codex registration (0.2.1)

The Windows-native client package now has an operator-trusted bootstrap that verifies payload bytes before executing them, probes strict authenticated MCP, and registers in an explicitly selected Codex home. Existing conflicting registration needs explicit -Replace; unrelated configuration/tool approval tables are preserved, unsupported forms stop. It does not provision SSH identities, copy sign-in data or claim GUI loading. Use the [scope, descriptor, recovery and actual native tests](docs/verification/TASK_003A_NATIVE_REGISTRATION.md); the example is intentionally non-operational until reviewed. Never trust a digest delivered only with an untrusted download.
