# Development readiness research

Research date: 2026-10-22 (America/Los_Angeles).
Repository baseline: `b2c38cd08ce08b5adb7c89ed18955b586fca2df9`, VERSION `0.2.0`.

## Decision

Prioritize an independently runnable client diagnostic, followed by binding an
existing project to an existing remote workspace. Extend durable tasks after
those contracts stabilize. Prepare update recovery separately. Investigate
provider authorization early; make enforcement a release gate for shared use.

This is a development proposal, not an implementation or live acceptance result.
No workload, credential, route, host configuration or service was changed.
The working tree was clean before this report was added.

## Evidence and confidence

| Evidence | Observation | Limit |
| --- | --- | --- |
| Repository code and docs | Installation, client enrollment/probes, diagnostics, jobs and nested smoke scaffolding exist | Presence is not proof of runtime correctness |
| Current Platform MCP `list_containers({})` | Successful authenticated call; six resources, four running and two stopped | No connect, workload execution or installed-version verification |
| Current advertised tool schemas | `code_runs`, `code_status`, `code_attach` and `code_stop` exist; attach accepts an output offset | Not exercised in this research; applies to provider coding runs |
| Recorded nested test | `docs/TEST-BOXWEAVE-LXC.md` records inner MCP create/connect/Python success | Historical environment, not current checkout acceptance |
| Upstream official documentation | Scopes, roles and audit facilities are documented | Upstream main is mutable; installed support remains unverified |

Inventory candidates include `boxweave-e2e`, `boxweave-test` and `bw-acceptance`.
Names alone do not establish availability for modification. Inspect purpose,
files, ownership and Git state before selecting a development target.

No tests were rerun in this research. No source was transferred to a container.
README references `.github/workflows/ci.yml`, but that path is absent from this
checkout and `git ls-files .github` returns no entries. Do not assume automated
CI coverage exists from the README statement alone.

## Ranked scope

Scores are planning judgments out of ten, balancing user value, fault reduction,
effort and fit. Effort is a preliminary estimate for one developer, including
focused tests and docs, excluding waiting for hardware/client access or upstream
fixes. It is not a delivery commitment.

| Rank | Feature | Score | Estimated effort | Readiness |
| --- | --- | --- | --- | --- |
| 1 | Independent client diagnostics | 9.5 | 4-7 developer days | Strong reuse; client execution contract needed |
| 2 | Project/workspace binding | 9.2 | 5-8 developer days | Schema and target verification needed |
| 3 | Durable task discovery and output continuation | 8.9 | 4-7 developer days | Extend jobs; separate provider coding runs |
| 4 | Update preflight and controlled recovery | 8.6 | 6-10 developer days | Recovery boundary must be explicit |
| 5 | Scoped identities and audit integration | 8.5 | 2-4 days feasibility, then re-estimate | Installed provider enforcement unverified |

For multi-user deployment, move item 5 ahead of project mutations. Do not sum
these estimates into a release date before feasibility and client validation.

## 1. Independent client diagnostics

Existing building blocks:

- `clients/posix/setup.py:probe` and Windows `Test-BoxWeaveMcp` already perform
  initialize, initialized, tools/list and a read-only list_containers call.
- `server/cli.py` observes services and optional unauthenticated API reachability.
- `server/mcp_client.py` implements bounded POSIX stdio exchange for smoke tests.

Gaps: probes are embedded in enrollment; protocol readers are duplicated; current
probes require protocol `2024-11-05`; discovery checks only the first tools page.
These are compatibility limits, not evidence that the present server is broken.
The MCP specifications describe version negotiation and cursor pagination [1][2].

Proposed MVP: independently runnable Windows and POSIX diagnostics with equivalent
JSON output. The Ubuntu `boxweave client` generator can emit a diagnostic command;
the diagnostic itself must run in the actual client runtime. A server-side CLI
cannot establish Windows/macOS configuration correctness remotely.

Suggested output: schema_version, runtime, transport, stage results, negotiated
protocol, observed server identity, elapsed time, sanitized error code and
next action. Preserve existing status/doctor exit meanings; define separate
diagnostic exits explicitly (0 passed, 1 failed, 2 incomplete/unsupported).

Default checks must not enroll keys, replace config, rotate tokens or invoke
connect. The advertised connect tool may authorize a managed key on a plain
daemon, so it is not a strictly observation-only probe. SSH itself can also
produce provider-managed access records. Report GUI recognition and guidance
loading as unverified unless separately observed in the actual agent.

Implementation: extract probe-only entry points without executing setup on import;
share Python protocol logic where packaging allows; retain a native PowerShell
implementation to avoid introducing a Windows Python prerequisite. Use explicit
supported protocol versions, pagination with cursor-loop/page/byte budgets, and
bounded writes as well as reads. Never replace an error with raw SSH stderr.

Acceptance: success, missing runtime, changed host pin, denied provider call,
early exit, timeout, unsupported protocol, second-page tool discovery, repeated
cursor, oversized output and invalid JSON. Assert no config/key publication.
Run real Windows PowerShell 5.1 native/WSL and macOS checks separately from mocks.

## 2. Project/workspace binding

There is no project registry or general provider abstraction yet.
`providers/containarium/README.md` describes the existing integration;
`docs/ARCHITECTURE.md` explicitly labels its generic provider interface as future
design. Build the minimum adapter around current MCP rather than a new gateway.

MVP: `project init/status` binds an existing box and an existing absolute remote
directory. Treat `project open` initially as a machine-readable handoff, with no
promise of native IDE launch. Creation and dependency installation are later,
explicit actions. Do not silently clone, overwrite, change Git branches or install.

Proposed versioned manifest: project ID, provider, server profile ID, logical box
name, observed instance name/project, remote path and optional expected Git remote.
Keep host-specific overrides private and outside tracked project configuration.
Store no credentials. A tracked manifest is untrusted input: it must not select
arbitrary executable hooks or acquire authorization merely by being present.

At first execution, confirm the approved target and connect identity. Detect box
replacement and path/repository mismatch; preserve dirty work. UUIDs may be shared
by clones, so combine fresh host/project/name evidence and creation metadata.
Do not hard-code a container suffix or parse arbitrary guest text as trusted data.

Candidate files: new `server/projects.py`, a narrow provider client interface,
CLI wiring and schema documentation. Reuse metadata/receipt concepts from smoke,
but do not reuse its fixed hello-world parser as a general execution parser.

Acceptance: repeated binding, missing box, replaced box, wrong directory, dirty
repository, path quoting, absent metadata and manifest migration. No implicit
resource creation or destructive cleanup.

## 3. Durable tasks

`server/jobs.py` already supplies detached POSIX execution, private receipts,
locks, deadlines, bounded/redacted logs and actual exit codes. Its CLI currently
exposes start/status; status returns only a bounded log tail when requested.
`server/smoke.py` already persists creation intent and phase deadlines for one
specific workflow. Do not describe either as a general workflow engine.

MVP: job list with bounded pagination, incremental sanitized logs, and explicit
host/user/project association. Offset semantics must refer to persisted sanitized
log bytes; return next_offset, terminal state and truncation/rotation indicators.
Preserve the difference between query success and command success.

Provider code runs have their own run identity and code_attach offsets. Add a
display adapter if useful; do not silently convert an arbitrary job to code_run
or introduce another coding-agent supervisor.

Defer generic resume and cancellation until process identity and operation
reconciliation are specified. Never signal a PID solely because it appears in an
old receipt. A local timeout does not prove a remote request was cancelled.
Resume may continue a known workflow checkpoint; it must not replay arbitrary
commands or resubmit an ambiguously completed create.

Acceptance: disconnect/reconnect, nonzero exit, output limits, offset continuation,
worker disappearance, reboot/unknown state and concurrent readers. Preserve
existing tests in `tests/test_jobs.py`, `tests/test_smoke.py` and MCP fixtures.

## 4. Update preflight and recovery

`install.sh` already locks, stages, validates, backs up managed files, publishes
per file atomically and records update-in-progress. `server/provenance.py` records
source and hashes. Global rollback is explicitly absent in INSTALL-UPDATE.md.

MVP: read-only update plan with current/target provenance, managed-file differences,
required privileges and affected services. Download preparation, if offered, must
be an explicit separate action. Recovery plan validates a selected root-owned
backup and pending marker before proposing restoration of BoxWeave-managed files.

Do not claim rollback of package installation, JWT issuance, authorized keys,
verified known-host reconciliation, provider state or container data. Restoring
old client descriptors can invalidate generated download/checksum expectations;
validate that they match the restored source. Existing MCP processes retain old
executables until reconnected.

Candidate implementation: read-only planning module and reuse of the existing
installer publication/service checks. Avoid a second independent installer.
Service stop/restart and file restoration need a reviewed mutation plan.

Acceptance: corrupt/missing backup, symlink, incomplete publication, stale marker,
modified managed files, intentional service mask, interrupted recovery and
successful inactive oneshots. Installer sandbox tests run as non-root Linux;
real upgrade/recovery tests remain distinct from fake-host regression tests.

## 5. Scoped identities and auditing

`server/tokens.py` currently generates an admin-role token with 168-hour expiry.
Restricted SSH constrains the host command, not the privilege of provider actions.
Upstream documents user roles, per-tool scopes, API-side gates and audit queries
[3]. This makes integration preferable to building a BoxWeave authorization proxy.
Documentation on main does not prove those features exist in this installed release.

Feasibility questions: installed daemon/backend versions; exact scope mapping for
connect and coding runs; subject-to-box ownership; denied-operation enforcement
through both MCP and direct API; audit identity and coverage. Shared launcher/user
credentials may collapse several clients into one provider subject. Per-client
attribution therefore requires an identity design, not merely an audit UI.

MVP after validation: explicit credential profiles, fail-closed unsupported-profile
handling, migration that preserves the existing working admin path until approved,
and sanitized audit queries/correlation. Do not copy upstream example secret-read
permissions into the default read-only profile. Token rotation must preserve scope.
Plan a lockout recovery path before changing the deployed identity.

Acceptance requires negative tests with dedicated identities/resources: list own
box succeeds; writes, secret reads, route creation and access to another subject's
box are denied as applicable; API cannot bypass MCP filtering; restart/rotation
does not broaden permissions. No existing credential or signing secret changes
are authorized by this report.

## Development sequence and gates

1. Establish a dedicated remote checkout and record Python/provider/MCP versions,
   Git status and a fresh baseline test result. Inspect resource suitability first.
2. Implement diagnostic contracts and probe extraction; test on remote Linux plus
   genuine client runtimes. Keep installer downloads/hashes aligned with new assets.
3. Add existing-workspace binding and then task discovery/log continuation.
4. Add update planning; perform scoped-authorization feasibility in parallel only
   as a workstream, without assuming approval for credential changes.
5. Before release, obtain actual client acceptance and installer rerun results.
   Restore or add CI deliberately; current checkout has no tracked CI workflow.

Primary tests: existing client/MCP tests for diagnostics; new behavioral project
tests for binding; existing job/smoke tests for task changes; non-root installer
sandbox tests for update work. Run relevant subsets, then the complete Linux
suite once integration is ready. Documentation-only research needs link,
consistency and diff checks, not a new test suite or a live provisioning run.

## Optional live validation plan (not executed)

The repository AGENTS.md requires reviewing a concrete experiment plan before
live research. No experiment is necessary to deliver this preparation report.
If approved, start with the following limited baseline, not a new installation:

- Target: existing `boxweave-test`, only after confirming its current purpose and
  suitability. First connect must return whoami/hostname and actual exit code.
- Observations: executable availability/version, selected service states, public
  provenance fields, existing checkout Git status and managed launcher identity.
  Do not dump config, environment, raw logs, JWTs or private keys.
- Protocol test: run the installed launcher as its configured non-root user,
  initialize/initialized, paginated tools/list and list_containers. Emit selected
  protocol/server metadata and success/error categories only.
- Expected effects: connect may authorize its managed public key; normal access
  logs may be written. No package installation, source modification, resource
  creation, service restart or port change. Use no new persistent background job.
- Limits: one 600-second overall deadline, at most 30 seconds per read/protocol
  request; retry transient readiness every 15-30 seconds without resetting it.
- Success: verified identity exit 0 and real authenticated read-only MCP call;
  explicitly record missing capabilities rather than treating HTTP as acceptance.
- Cleanup: no workload cleanup needed; any approved temporary report files must
  have exact recorded paths and be removed only within that approved scope.

Authorization-negative tests, task fixtures, client enrollment and update/recovery
experiments each need a separate concrete scope. They are not covered by the
observation plan above. Never modify unrelated or provider core containers.

## Sources

- Repository: README, CLI, CLIENTS, INSTALL-UPDATE, ARCHITECTURE, SECURITY,
  TASKS-AND-SELFTEST and TEST-BOXWEAVE-LXC; implementation files identified above.
- [1] [MCP lifecycle and version negotiation](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle).
  This is a versioned reference, not a claim that it is the latest revision.
- [2] [MCP list pagination](https://modelcontextprotocol.io/specification/2025-06-18/server/utilities/pagination).
- [3] [Containarium operator security runbook](https://github.com/FootprintAI/Containarium/blob/main/docs/security/OPERATOR-SECURITY-RUNBOOK.md).
  Mutable upstream main; compatibility must be checked against installed versions.
