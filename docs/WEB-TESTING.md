# Web v1 test evidence and limitations

Date: 2026-10-23 (America/Los_Angeles). Dedicated BoxWeave MCP-created Ubuntu
24.04 LXC `boxweave-web-dev-container`, logical username `boxweave-web-dev`.
Project: `/home/boxweave-web-dev/boxweave`. Resources: 4 CPU, 8 GB RAM, 30 GB disk.
Platform MCP create returned RUNNING; connect `whoami && hostname` exited 0 and
returned the expected username/hostname. No passthrough/public route was added.

## Automated checks

Run from the dedicated Linux checkout as its ordinary non-root user:

```bash
python3 -m venv .venv
.venv/bin/python -m pip install --require-hashes -r server/webui/requirements.lock
.venv/bin/python -m unittest discover -s tests -v
bash -n install.sh
.venv/bin/python -m compileall -q server tests
(cd web && npm ci --ignore-scripts --no-audit --no-fund)
(cd web && npm run build && npm test)
(cd web && npx playwright install --with-deps chromium && npm run test:browser)
(cd web && npm audit --audit-level=moderate)
git diff --check
```

Full Python regression: 163 cases, exit 0, with 17 Windows/PowerShell-dependent
cases skipped on Linux. The final installer suite includes 15 cases and passed
with exit 0, including local-source staging and independent Web failure recovery.
The Python Web/security/installer helper tests cover 24 cases: protected APIs
before initialization/login, HTTPS/HTTP cookies, CSRF/Origin, login rate limits,
session expiry/logout, graph persistence/revisions/duplicates/free cards,
undirected relations/comments, history/manual deletion, Markdown escaping and
secret-pattern rejection, batch validation/partial success/unknown results,
one-time plans, restart recovery, unsupported operations, no generic proxy,
path traversal, private database/token permissions, initial/newer schema checks,
provider field allowlists, explicit sshKeys and pinned REST path/body contracts.

The existing installer sandbox also verifies explicit local-source installation
without publishing, as well as independent Web-stage failure and
Web-publish failure: build failure preserves installed files; publish failure
leaves MCP timers active and the pending marker present; rerun repairs it. The
sandbox deliberately stubs the Web provisioner; these are not live package,
provider, Incus or systemd integration results.

Frontend: TypeScript check and Vite production build succeed. Four Node built-in
unit tests cover independent batch rows and copy lifecycle ordering/failure.
Eight Chromium end-to-end tests pass against a real Flask/Waitress server with a
fake in-memory provider: search/details/confirmation, shared/individual batch
creation, repeat cards/autosave/reload, free cards/manual copy/new graph,
unsupported controls/mobile layout, graphical undirected connections/export,
recoverable competing revisions, and successful clipboard copy preserving history.
UI screenshots were inspected. npm audit
reports zero vulnerabilities for the complete locked dependency tree at the time
of this run. This is a point-in-time advisory observation, not a security proof.

## Real web-only deployment checks

In the dedicated LXC, the actual Web preparation/publication helper was run with
its own systemd units and APT-extracted Caddy binary. `/etc/boxweave/config.json`
in this box is explicitly a **test fixture**, referring to an unsigned synthetic
issuer and an unused loopback provider endpoint. No real provider signing secret,
MCP token, host credential or existing resource was read or changed.

Observed successful preparation/publication exits: 0. Dedicated backend and
Caddy services are active/enabled; the independent web token timer is
active/enabled. Waitress binds loopback; Caddy binds the container's private IP.
HTTPS was checked with curl using the actual generated public root CA (no `-k`).
`/healthz` returned HTTP 200 and `{schema:1,status:"ok"}`. `/api/status` reported
uninitialized before local CLI setup. The real getpass CLI was driven through a
local PTY with a disposable test password (never logged); exit 0. Login produced
a Secure cookie; graph creation worked; service restart preserved the graph.

Repeated publication preserved SQLite data. Explicit HTTP publication exited 0;
login correctly omitted Secure. Restoring HTTPS exited 0 and preserved both the
CA fingerprint and existing graph. A real occupied socket was rejected before
installed Web settings changed (package calls stubbed for that collision check).
Provider-token file mode was 0600; the runtime service user is `boxweave-web`,
ProtectHome is enabled, and provider signing configuration is inaccessible in its
systemd sandbox. Test-created administrator sessions are removed at handoff;
the retained console remains an uninitialized deployment fixture.

## What this does not establish

- No real provider infrastructure mutations were performed through Web. The
  v0.114.2 capability matrix is verified from the pinned upstream source tag,
  not from a live three-client infrastructure mutation test.
- The complete root installer has not been run against the real Ubuntu host.
  Fresh install/update/MCP-only upgrade are covered by sandbox staging and the
  dedicated Web helper tests, not a clean bare-metal full-stack installation.
- Existing host MCP availability was checked with real outer Platform MCP
  list/connect calls. This is not a fresh Codex/Claude client enrollment or a
  new nested installed-MCP authentication test.
- No host LAN port route was created, and no external LAN client connectivity
  was claimed. `https://10.206.30.2:18443` is a container-internal test address.
- LXC `boot.autostart`, firewall source filtering and public internet isolation
  were not independently verified. Application unit enablement is separate.
- Windows/PowerShell-specific tests require their supported runner; Linux skips
  do not constitute Windows validation. The checkout's existing research files
  and unrelated resources are not part of this change.

## Deliberate v1 limits

No metrics endpoint is used because upstream CLI metrics is unimplemented. No
route editing/toggles are shown because CLI/MCP lack corresponding operations.
No rename, force-delete shortcut, agent runtime or automatic prompt execution.
Unknown creates reserve their names; there is no automatic retry or automatic
rollback. Provider roles/scopes remain broader than individual Web actions.
Rotation expires older tokens within two hours, rather than immediate revocation.
Updates are atomic per file; a failed Web update can require rerun/manual
recovery. Free-text secret detection recognizes common patterns, not every
possible secret. IPv4 private-address deployment is supported; ambiguous LAN
interfaces require explicit --web-host. See [usage/recovery](WEB-UI.md).
