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

## Missing-instance compatibility regression

A subsequent read-only Platform MCP lookup of `boxweave-web-test` reproduced a
v0.114.2 response mismatch: a missing default-project instance returns HTTP 500
with the gateway JSON `code:500` and an exact nested `Instance not found` error,
rather than HTTP 404. The original fake-provider tests returned 404 and therefore
did not exercise this production behavior. The create-time duplicate check
stopped before POST /v1/containers.

The REST adapter now normalizes only this observed bounded JSON response for
GET /v1/containers/{username}, matching the exact requested instance and default
project. Other 500s, authorization errors, wrong names/projects, malformed or
oversized bodies, inventory errors, and write errors are not treated as absence.
Raw provider error bodies are never returned to the browser or logged.

Four regression tests cover the exact response, negative/error cases, failure
before mutation, and a complete authenticated Web plan/confirmation/batch through
a local HTTP fake provider. The latter verifies a missing-instance 500 is
followed by exactly one creation POST with explicit SSH public keys. This is a
real HTTP transport test against a simulated provider, not real infrastructure
creation. No existing failed receipt is replayed automatically, and ambiguous
creation safeguards remain intact.

Fix validation: `python -m unittest discover -s tests -p "test_web*.py" -v`
passed all 28 tests (exit 0). Python compilation and `git diff --check` also passed.

## Tenant console validation (schema 2)

Development and execution used the existing dedicated `boxweave-web-dev` LXC,
checkout `/home/boxweave-web-dev/boxweave-tenant`, based on `f891585`. No additional
Box, route, host configuration or Provider binary was created or changed.

Read-only real Platform MCP inventory, get/connect and network_guard_status
calls succeeded. At observation time the three existing web dev/test2/test3
instances resolved to three different native Tenant subjects and ACL names;
Tenant Guard reported ENFORCE/nftables with attached entries and no unresolved /
unmanaged entries. This is a point-in-time observation, not an installed-Web
integration test, shared-network test or bidirectional connectivity result.
Pinned-source investigation used Containarium v0.114.2 commit
`8107ea2724dd12786bbd73ff5371e8314f3d35d9`; see [upstream findings](TENANT-UPSTREAM.md).

Validation as the non-root project user (all exit 0):

- Full-regression checkpoint, `.venv/bin/python -m unittest discover -s tests -v`: 190 tests, 173 passed,
  17 PowerShell-dependent tests skipped. Installer rerun/staging and APT-lock
  regression protection passed in the unprivileged sandbox, not on the Host.
- `test_tenants.py`: 21 focused tests passed, including atomic schema 1 -> 2,
  byte-equivalent old table rows, retained sessions/admin/graphs/jobs/plans/audit,
  idempotency, rollback on migration conflict, Provider precedence/fallback,
  stale/unsupported/ambiguous/unattached ACL evidence, metadata revisions,
  reference deletion blockers, authenticated server-side scope validation,
  capability gates, non-replaying reconciliation and legacy/mixed graph export.
  These focused tests were rerun after final Guard/parser changes.
- `python -m unittest discover -s tests -p "test_web*.py" -v`: all 31 Web /
  installation helper regressions passed again after final backend changes.
- `npm --prefix web test`: 6 model tests passed, including per-row Tenant intent
  and expiry of open-tab verification without a successful network refresh.
- `npm --prefix web run build`: TypeScript and Vite production build passed.
- `npm --prefix web run test:browser`: 11 Chromium scenarios passed against an
  ephemeral authenticated Web server with a synthetic Provider. They exercise
  prior resource/graph behavior plus metadata CRUD/rename/reload persistence,
  empty metadata status, backend filtering, shared-creation gates, row selection
  retention and Tenant/Box mixed Markdown context. Mobile screenshot inspection
  is a layout check, not a production deployment test.
- Python compilation, `bash -n install.sh` and `git diff --check` passed.

No real shared-Tenant creation, birth-time ACL capture, A -> B / B -> A TCP test,
cross-Tenant TCP restriction test or Host management regression was performed.
No live Host Web upgrade, persistent service restart, LXC autostart, client LAN
connectivity, public exposure restriction or clean Windows deployment is claimed.
No upstream patch/fork was installed. The existing Web/Caddy/forced-command/token
services remain untouched. The [upstream proposal](TENANT-UPSTREAM.md) and
[upgrade/rollback instructions](TENANTS.md) describe the remaining release gates.

## Native Tenant networking A + B: isolated validation, 2026-10-24

The following evidence supersedes earlier capability-blocked implementation
notes for the optional patched profile; it does not change historical live-test
results above. Dedicated box: boxweave-tenant-dev-container, logical username
boxweave-tenant-dev, 4 CPU / 8 GB / 40 GB. Development roots:
 /home/boxweave-tenant-dev/work/boxweave (base 7818668) and
 /home/boxweave-tenant-dev/work/containarium (v0.114.2 base
8107ea2724dd12786bbd73ff5371e8314f3d35d9).
MCP list and connect identity checks succeeded with actual exit code 0.

- Final non-root Python regression: 205 tests, OK, 17 Windows/PowerShell-dependent
  skips, TEST_EXIT_CODE=0 (120.911 seconds). Includes 11 native A/B API tests
  and 3 token tests, including absent-policy gateway default-field readback.
- Provider regression: pkg/core/container, pkg/core/box/lxc, pkg/core/incus,
  internal/tenantguard, internal/server, internal/mcp, internal/client,
  internal/cmd and internal/auth all passed; TEST_EXIT_CODE=0.
- Real PostgreSQL fixture used an extracted non-service PostgreSQL binary with
  a temporary non-root database on loopback port 18599. Tests exercise ingress
  field preservation, full revisions, atomic two-way rollback, competing writes
  and absent-row conflicts. This is actual SQL transaction evidence.
- Generated REST gateway plus in-process gRPC tests and an actual MCP stdio
  initialize/initialized/tools/list/tools/call test passed. Their Provider store
  and Guard evidence are controlled test fixtures, not the deployed host.
- TypeScript/Vite production build passed. Chromium: 15 tests passed (11 legacy,
  4 native A/B scenarios) against real Flask/Waitress with synthetic Provider.
  Covers shared batch creation, new Tenant in both modes, grant/both/revoke
  preservation and access-card context with no policy side effect.
- Pre-Start native config/Guard ordering, autostart deferral and fail-closed
  cleanup are tested with a fake Incus backend. No real A/B workloads were
  provisioned by these tests.

All three Linux amd64 candidate binaries built successfully without installation;
patch apply-check and application to a clean exact-base archive passed, with all
42 changed files byte-identical to the development source. Six frontend model
tests, shell syntax, Python compilation, local documentation links and diff
whitespace checks passed. The manifest records actual artifact checksums.

No deployed Provider upgrade, real policy mutation, same/cross-Tenant TCP
reachability, public/LAN route or production installation is claimed. The
source candidate requires the separately approved live acceptance plan in
[TENANT-UPSTREAM.md](TENANT-UPSTREAM.md). Stored grants and verified ACLs never
substitute for a real bidirectional TCP test.


## Historical fork contract validation (2026-10-25 UTC)

Dedicated development LXC: `boxweave-tenant-dev`.
BoxWeave checkout: `/home/boxweave-tenant-dev/work/boxweave-native`.
Provider checkout: `/home/boxweave-tenant-dev/work/containarium-boxweave`,
exact `c13582b5016e7471d83d50d1e09a49a2fe661395`.
The historical v0.114.2 patch/build records above remain historical.

Actual commands and final results:

- Provider: `go test -p 2 ./internal/server ./internal/mcp ./internal/cmd ./internal/client ./internal/auth`
  passed all five groups; server 57.015s. An isolated loopback PostgreSQL fixture
  was supplied through TEST_TENANT_PG_DSN, so transaction tests ran rather than skipped.
  `golangci-lint run` reported `0 issues.`; final log ends EXIT_CODE=0.
- Provider: project `make proto` regenerated bindings successfully; protobuf lint
  and breaking compatibility checks against the A-stage commit passed.
- BoxWeave: `.venv/bin/python -m unittest discover -s tests -v`:
  `Ran 212 tests in 136.940s; OK (skipped=18)`, EXIT_CODE=0.
  Seventeen skips require PowerShell; the optional source-contract test is the
  remaining skip and was run separately below.
- `BOXWEAVE_PROVIDER_SOURCE=/home/boxweave-tenant-dev/work/containarium-boxweave .venv/bin/python -m unittest discover -s tests -p test_provider_branch.py -v`:
  `Ran 1 test in 43.513s; OK`, EXIT_CODE=0.
  This compiles the BoxWeave-owned Go fixture temporarily inside the exact clean
  Provider source, removes that helper, and invokes real generated REST handlers
  and actual policy service/store code through the real Python HTTP adapter and
  authenticated Web preview/confirm flow. It verifies typed pending state,
  nonempty local Backend identity, egressMode preservation, bidirectional update,
  stale revision rejection and precise revocation. Infrastructure/Guard/durability
  and the HTTP bearer identity are synthetic; this is not live Incus, production
  authentication, durable PostgreSQL or physical TCP acceptance.
- `cd web && npm run build && npm test && npm run test:browser`:
  production build passed, 6 model tests passed, `15 passed (46.0s)`,
  EXIT_CODE=0. Browser tests use synthetic legacy/native Providers.
- `bash patches/containarium/build-candidate.sh /home/boxweave-tenant-dev/work/containarium-boxweave /home/boxweave-tenant-dev/work/artifacts/native-c13582b`:
  all three binaries built; EXIT_CODE=0. Nothing installed.
- `bash -n patches/containarium/build-candidate.sh` and `git diff --check` passed.

Candidate SHA256 values from that build (not signed release hashes):

```text
9bf31ed5d07e0dff0b8eab227b531984dd2d0af25eeb91d39a5c9ee2e97a6ad8  containariumd
25f09fb035efaf9987cf01b77ee63289bebcf4695a914bca099808371214f8ea  containarium
390f4a718724b0183c253dd1c6cd3822595a6a851d92c7dd36d63f01a8af9de3  containarium-mcp
```

Raw logs remain under the dedicated LXC's `/home/boxweave-tenant-dev/work/logs/`:
`access-final-validation.log`, `access-proto-final.log`,
`box-native-all.log`, `box-source-contract.log`, `box-native-web.log` and
`box-candidate-build.log`. PostgreSQL and contract/browser fixture processes were
stopped after testing. No real Host deployment, existing instance change,
provider service upgrade, ACL edit, public route or TCP connectivity claim was made.
See [remaining deployment/live acceptance steps and suggested upstream PR descriptions](TENANT-UPSTREAM.md).

## Merged upstream compatibility and client configuration (2026-10-27)

Validated as non-root `containarium-build` in the dedicated `containarium-build`
LXC, against clean official Containarium source commit
`d4aad1a8a8cd748ab6211c67a3417f173316f9e1` and its
`native-network-tenant/v1` contract:

- Python unittest discovery: 224 tests, successful, with two native Windows
  OpenSSH diagnostic fixtures skipped. PowerShell 7.4.6 was extracted under the
  ignored project outputs directory; configuration and Linux WSL-shim fixtures
  ran, including initialize/initialized, tools/list and a read-only tools/call.
- The cross-repository fixture exercised upstream generated REST handlers and
  atomic policy service with synthetic infrastructure/authentication, including
  the Web token capability probe. It was executed, not skipped.
- Client configuration regression: replacing transport preserves per-tool
  approval tables and unrelated servers, keeps a backup, is repeatable, and
  rejects conflicting transport changes without explicit replacement. POSIX
  configuration tests also verify tool-table preservation and reject inline
  tool settings that would otherwise be lost.
- Frontend model tests: 6 passed; production asset build succeeded; Playwright
  browser suite: 15 passed. Browser libraries were extracted in user-owned
  outputs paths without changing host packages.
- The pinned-source build helper built daemon, client-tag CLI and Host MCP with
  source version `0.0.0-source.d4aad1a8a8cd` and checksums. A different source
  checkout was rejected with exit 2. Shell syntax, Python compilation and Git
  whitespace checks passed.

Logs and development binaries remain in ignored `outputs/upstream-3165/`.
These results cover source compatibility and isolated fixtures, not a real
Windows installation, live NIC ACL/eBPF enforcement or bidirectional TCP
isolation. No running Provider was replaced, no services were restarted and no
real network-policy or credential changes were made.
