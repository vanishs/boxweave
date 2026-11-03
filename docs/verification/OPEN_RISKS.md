# Current risk register — 0.14.0 (historical evidence preserved)

Owner is the project maintainer/operator unless specified. P0/P1 = blocks the selected claim until resolved or explicitly cannot accept; P2 = bounded operational/coverage limitation. Source: [comparison](COMPARISON_REPORT.md), SRC-084..088. The historical phase6 P0/P1 items are closed within their original scope. Current remediation still has open P1 work, including RISK-610; use the current rows below. Residual P2 entries do not silently certify the unverified property.

| ID | Priority / status | Impact / scope | Evidence and action / owner |
| --- | --- | --- | --- |
| RISK-601 | P1 closed, fixed0.9.1 | Missing project binding edit/reselect/unbind violated REQ010; phase5 completion prose overstated coverage | New service/UI/revision7 tests, browser and actual installed-copy unbind preserve metadata/receipts. Maintainer: retain regressions and require backup on migration |
| RISK-602 | P1 closed, fixed0.9.1 | Empty target list could claim diagnostic success without contacting MCP and mutate local state | Actual0.9.0 dead-launcher exit0 versus patched exit2, positive real109-tool read with unchanged DB6; no Store in diagnose. Maintainer: preserve empty-list/unavailable-launcher cases |
| RISK-603 | P1 closed, user decision SRC-086 implemented | Missing promised Web client setup entry | Four profile generator, downloaded hash-gated scripts/replacement intent, strict validation/auth/CSRF and actual Windows registration/config-preservation. Operator still independently reviews source/pin and runs local setup; Web never enrolls or reads credential contents |
| RISK-604 | P2 open, legacy test maintenance | Readiness test at d8a09dd fails collection because its installer text extractor is stale |15 other isolated legacy tests pass; readiness assertions unexecuted. Legacy owner may repair its harness in its own authorized task. Do not change old expectations here or infer old runtime readiness |
| RISK-605 | P2 open, deployment handoff | Primary private test app remains0.9.0; editing source does not upgrade it, and new DB7 cannot be read by old binaries |0.9.1 migration/install tested on independent consistent copy; original bytes/PIDs unchanged. Operator: use reviewed0.9.1 package and consistent backup before using patched behavior; retain older artifacts as historical evidence |
| RISK-606 | P2 open, evidence boundary | Full old/new live fleet/network equivalence and other server platforms are unverified | Local old/new receipt execution, old mocks/static inspection and new prior real Provider0.120.1 packets are different lanes. No fabricated parity. Scope any future live legacy deployment separately |
| RISK-607 | P2 accepted boundary | Reboot/LXC boot.autostart, effective confinement, new public TLS rollout and model inference/SLA not demonstrated | Selected user-manager/private-LXC/SSH/native-model-handoff evidence only. Owner: obtain specific evidence before making any additional claim; no phase6 host changes or model spending |
| RISK-608 | P2 accepted trust assumption | Operator-delivered source/pin/digests and broad upstream credential authority remain trusted; generated descriptor does not grant per-tool least privilege | Single trusted operator DEC005. Public setup never contains private key/JWT; helper/payload hashes, exact versions, local ACL/pin and registration gates remain. No publisher signature or general secret-detector completeness claimed |
| RISK-609 | P2 accepted recovery/convergence boundary | Unknown commands cannot be replayed/adopted; dispatched work cannot be cancelled by unbinding; route acknowledgment is not instant packet convergence | UI warning and retained receipts/revision gates; previous real TCP/UDP delay evidence. Reconcile original identities and inspect saved outcomes before a fresh explicit action; no whole-host rollback promise |

Phase7 maintenance is authorized by SRC-089; its completion evidence belongs in [PHASE_07_REVIEW](PHASE_07_REVIEW.md). It does not resolve these operational bounds or authorize publication/deployment. No pending product decision remains after SRC-086. A future required claim becoming unsupported must be reopened as cannot accept, not hidden under a P2 label.

## New 0.10+ remediation risks (current; not closed by the old phase6 report)

| ID | Priority / status | Impact / scope | Evidence and action / owner |
| --- | --- | --- | --- |
| RISK-610 | P1 open / wider AC-019 acceptance | Source0.15.1 published streamed bootstrap/reinstall and separate peer MCP create/identity/Python passed in a dedicated nested Ubuntu LXC. This does not certify every fresh host, confinement or platform. New0.16.0 helper privilege acceptance is tracked separately in RISK-613 | [Streamed evidence](STREAMED_INSTALL_TEST.md), [nesting limits](NESTING_VERIFICATION.md); preserve existing upstream and require exact-version tests |
| RISK-611 | P1 closed within selected CLI acceptance, 0.14.0 | Actual installed wrapper/status/doctor, exact local Incus reads and missing/invalid targets, independent scoped MCP58-tool read, credential rotation and failed-install marker/repair passed in the dedicated LXC.192 unit tests/build/seven browser workflows and current four-profile Windows reads passed; final package identity remains separately recorded | [CI and next-Agent queue](REMEDIATION_TEST_QUEUE.md). Owner: retain these regressions; broader Provider/network/reboot acceptance remains in RISK-610/606/607, not implied by CLI checks |
| RISK-612 | P2 source CI gate verified at 0.13.4 / continuing | GitHub Actions run 40985891994 on 08efb069 succeeded for source 0.13.10; 0.14.0 working-tree changes require their own pipeline after publication. Fresh CI must still be checked for every later commit | [Exact source CI and handoff](REMEDIATION_TEST_QUEUE.md); no host, Provider, Windows, browser or security acceptance follows from a green source pipeline |

Current remediation evidence: [0.14.0 review](MAIN_014_REVIEW.md). This separate test application does not upgrade the retained primary deployment in RISK-605.

Streamed source0.15.0 is now being implemented/tested under SRC-100. RISK-610 remains open until the exact streamed install and separate client creation/connect/Python path pass; [progress](STREAMED_INSTALL_TEST.md). Standalone Provider optional/native-policy/boot boundaries remain explicit.

Current SRC-100 experiment: source0.15.1 streamed reinstall and separate peer create/identity/Python passed. This resolves the specific experiment, not wider RISK-610 platform acceptance. [Measured follow-up](NESTING_VERIFICATION.md).

| RISK-613 | P1 open, new0.16.0 privilege path | New fixed helper/sudo rule and MCP extension need real installed-copy acceptance; old0.15.1 evidence does not certify this surface | [Validation and next checks](NESTING_VERIFICATION.md); no implicit restart or parent access |


## 0.17.0 existing Provider adoption risk

**RISK-614 (P1 open, existing-host acceptance):** Streamed installer source recognizes reviewed existing Containarium/Incus and retains Provider ownership across BoxWeave repair/update. The existing Provider's API must be reachable at the configured loopback endpoint and its signer readable only by the external Provider/token issuer. No real preexisting-host adoption, storage-source preservation, reboot or independent update/repair test is claimed. The current published artifact remains Ubuntu 24.04 / CPython 3.12; Ubuntu 26.04 compatibility is a separate package and acceptance task. [Test matrix](EXISTING_PROVIDER_INSTALL.md).


## 0.17.2 cross-version installation acceptance

**RISK-615 (P1 open, host verification):** Minimum host version is Ubuntu >= 24.04 / CPython >= 3.12, but the release bundles wheels only for CPython 3.12–3.15 and exact CI builds do not certify all future Python versions. The owner-provided Ubuntu 26.04 + bcachefs + existing Incus/Containarium/PostgreSQL must still pass a full real-host source-bound install, update/repair, container creation, and post-reboot mount/provider persistence check. **S**: source review; **U**: real-host acceptance. Do not bypass storage mount guards or silently replace external Provider state.


## 0.17.3 client setup live acceptance

**RISK-616 (P1 open, live client verification):** New Windows Codex + WSL SSH enrollment script generated by `boxweave client` must be exercised on an actual Windows Codex client and Ubuntu administrator SSH connection. The CI static PowerShell parse, fixtures and non-mutating source inspection do not certify interactive WSL password/sudo enrollment, forced-command SSH, MCP read or Codex app UI recognition. Required refusal tests: changed host pin, missing bootstrap sudo rights, mismatched WSL distro, keypair conflict, MCP unavailable, and existing Codex MCP entry without `-Replace`. Never bypass SSH host-key verification or auto-edit Incus/Containarium during client setup.


## 0.17.4 desktop MCP configuration regression

**RISK-617 (P1 live acceptance):** Confirm generated script works with the Windows ChatGPT/Codex desktop app **without** a standalone `codex` command on PATH. Verify app reads `%USERPROFILE%\\.codex\\config.toml`, password/keyboard-interactive administrator SSH auth, explicit-key admin fallback, backup and refusal of complex preexisting MCP TOML, and restart recognition. CI grammar parsing does not establish Windows desktop runtime correctness. Never modify WindowsApps ACLs or require installing a second Codex solely for MCP registration.


**RISK-617 (P1 open, workload managed SSH identity):** Historical `archive/0.5.1` clients exposed a separate full server-to-box public key in AGENTS, but the new client generator omitted it. A real `hello-world-test-container` creation returned Running, followed by an SSH public-key-authentication failure; actual supplied `ssh_keys` and guest `authorized_keys` are not yet independently attested. Version 0.17.8 adds a fail-closed create gate and restored client guidance, but **does not** modify existing resources or establish a successful `connect`/Python receipt. Operator must compare trusted full public key fingerprints, reconcile exactly the existing box with explicit approval if needed, and prove `whoami && hostname` and actual Python stdout/exit through MCP. This remains unverified until live evidence; no duplicate create or trust bypass.


**0.18.0 RISK-617 update:** The new Linux MCP gateway adds its own validated public key to each new create request, so Codex no longer needs BoxWeave-managed AGENTS guidance. The original `hello-world-test-container` still needs authorized investigation and live SSH/Python verification; neither the old block nor guest authorized keys are changed by installing this release.


**RISK-615/610 update (0.18.2):** The Ubuntu 24.04 GA run 41141400914 failed during APT resolution because a fixed Incus 7.5.1 subset conflicted with the Incus-base 7.6 candidate. The new managed fresh-host path removes partial pins and simulates the APT transaction before installation, but the exact revision still needs a complete isolated live install. The long-term owner-requested provider-only dependency boundary remains unresolved; the upstream Containarium installer itself changes Incus and host settings and is not silently executed.


**RISK-618 (P1, upstream installer integration, 2026-10-30):** Directly
executing the mutable upstream Containarium manual installer during a fresh
BoxWeave installation would open the REST listener on `0.0.0.0:8080` and
potentially UFW 8080, uninstall older Incus packages (including `apt-get
autoremove`), and install only the daemon, not the separate `mcp-server`.
Do not silently substitute it for the current reviewed managed Provider
bootstrap. Confirm upstream loopback/no-firewall support, immutable installer
identity, complete same-release MCP assets, exact provider compatibility,
owned-state separation, and real clean-host/update acceptance first. See
[the source-backed review](CONTAINARIUM_INSTALLER_BOUNDARY.md).


**RISK-619 (MCP SSH keepalive, 0.19.1):** Legacy `archive/0.5.1`
client SSH transports used `ServerAliveInterval=15` and
`ServerAliveCountMax=4` to mitigate idle session loss. Source-level
restoration of those flags in Windows Codex/WSL and legacy native/WSL
registration paths does not prove real long-idle reliability on any
network. Existing client config needs operator-approved re-registration;
the installer does not alter running SSH sessions or server sshd config.
Keepalive detects dead transports; it cannot automatically reconnect MCP.


**0.20.0 unified client generator note:** All new installed
`boxweave client` outputs are now Python 3.11+ including Codex/WSL.
This removes the old PowerShell-only Windows path from the installed
generator but does not install Python on clients or migrate old
`.ps1` registrations. Manual Windows PowerShell helpers remain
separate, and real cross-platform Python+SSH App acceptance is open.


**RISK-620 (gate-host nesting helper dependency, 0.20.1):** The
`enable_nesting` create extension and the `boxweave_nesting` tool run
`sudo .../nesting.py` on the host that hosts the MCP gate and require
that host to carry a complete BoxWeave server state
(`/var/lib/boxweave-default/state.json`, public helpers, sudoers). On
split deployments where the gate runs away from the Provider host —
observed on the 2026-10-30 dev backend — every nesting call fails with
`NESTING_HELPER_UNAVAILABLE` after the gate fix; before the fix the
failure was masked as a trust-gate error and the create response
suggested the box had not been created. Platform-created boxes were
empirically nested-capable, so workload creation was unaffected, but
the nesting tool contract remains non-functional until the gate host is
repaired or nesting is delegated to the daemon. Operators must not
delete and recreate boxes in response to `BOXWEAVE_NESTING_NOT_APPLIED`.

## Remote MCP v0.24.0 acceptance risks (explicitly deferred by owner)

**RISK-620 (P1 OPEN — public OAuth/client interoperability):** Source/fixture validation cannot certify real ChatGPT, Cursor, Claude, TLS/proxy authorization callback, certificate renewal, listener service ownership or actual Provider/Box execution. Verify against an approved real external client before production claims. No real Box tests were authorized.

**RISK-621 (P1 OPEN — broad trusted operator):** Legacy SSH and new authenticated remote MCP clients both have the existing broad operator-equivalent upstream tool authority; finer per-client/Box policy is deferred to a separate unified SSH+HTTP redesign. Credential-bearing URL is a bearer secret vulnerable to third-party proxy path logging; BoxWeave stores only a digest and permits revoke but cannot control external provider logs.

**RISK-622 (P2 OPEN — stateless Gateway performance):** The HTTP adapter spawns the existing trusted MCP gate per request rather than introducing a second long-lived privileged dispatcher. This preserves gate tool transforms but adds startup latency, does not offer resumable streams and cannot prove remote cancellation; lost mutating requests remain UNKNOWN and must not replay automatically.

## 2026-11-01 current code-first live sweep risks (not global certification)

**RISK-623 (P1 OPEN — fresh nested workload fails on parent confinement):** Owner-authorized `bwaudit-1101` fresh Ubuntu 24.04 streamed v0.26.0 installation and subsequent v0.26.1 upgrade passed, but an inner `incus launch` failed with cgroup2 BPF load `Operation not permitted` and `proc` mount denied. Inner guest is retained STOPPED; parent `security.nesting` not verified and outer `boxweave_nesting inspect/plan` returned `LOCAL_INCUS_OBSERVATION_OR_CHANGE_FAILED`. Historical nested Python works on different older parents, not evidence of this parent. An owner-authorized scoped parent-side policy inspection/correction with independent child create+execute check is required; no blind parent restart or host trust modification.

**RISK-624 (P1 OPEN — host-hop shell vs direct SSH trust are different):** `debug_container` confirmed the host account shell `/usr/local/bin/containarium-shell` absent on backend `dev`. Nonetheless the gateway direct IP `connect(exec)` succeeded on new and old Boxes with exact Python stdout and exit0; hostname/user mismatches were rejected. Do not call all SSH blocked. The separate host-hop/jump path remains unverified; broad upstream sudoers/sshd setup is not authorized as an automatic fix.

**RISK-625 (P2 OPEN — whole-project external acceptance):** All released source/unit/fixture checks cannot establish actual Windows/macOS/WSL GUI registration, TLS ACME/public HTTP MCP OAuth, D4 exact-positive allowlist, Web administrator session, paid model inference or cross-host private local-source transport. New source-bundle code was exercised with actual uncommitted content inside a Box, not transmitted across independent Agent/Box machines. Mark individual A rows U/BLOCKED and continue other checks; see [current ledger](CODE_FIRST_LIVE_ACCEPTANCE_2026-11-01.md).

**RISK-626 (P2 OPEN — source/installed version drift and typed CLI semantics):** The outer deployed gateway `boxweave_info` continued to report static source guide 0.25.0 while main advanced. Source v0.26.1 added source VERSION consistency tests and updated the developer guide, but the outer gateway is not updated by installing a nested Box. The v0.26.0 `boxweave box inspect` falsely classified an absent name as Incus unavailable; fixed and verified live in v0.26.1. The installed v0.26.1 `boxweave report --kind` advertised job/selftest despite the reader only allowing hello/environment/access; v0.26.2 limits the CLI help to real reader kinds and adds tests. v0.26.2 installed-surface checkpoint has not yet completed; do not claim it from CI alone.

## Post-v0.26.2 owner-host update: changed risk dispositions (2026-11-01)

[Latest A01–A33 code-first reacceptance](REACCEPTANCE_0262_TO_0264_2026-11-01.md) is the current source for exact test versions, command results and cleanup. The earlier RISK-623–626 text is retained as historical state, not rewritten into a false universal PASS.

- **RISK-623 remains P1 OPEN / PARTIAL:** On installed outer v0.26.2, `boxweave_nesting inspect/plan` still failed. Actual Incus 7.6 proved `--project default query` rejects with a specific usage error; v0.26.3 source changed it to `query '/1.0/instances/NAME?project=default'`, and an old installed Box ran the fixed candidate helper with real inspect, plan, no-change apply and stale-plan denial. **The outer host has not installed this corrected helper in this run** and a new nested child with proven launch/exec exit 0 is still required. Original failed `bwaudit-1101` child retained; parent BPF/proc confinement is separate.
- **RISK-624 remains P1 OPEN for host-hop only:** Direct `connect(exec)` into managed Box IP succeeded repeatedly with exact Python stdout/exit0, including after stop/start; `debug_container` still found missing `/usr/local/bin/containarium-shell` on host. No blanket SSH success/failure claim or unauthorized host shell repair.
- **RISK-625 remains P2 U/BLOCKED:** Authenticated external HTTP/OAuth/TLS clients, real native cross-platform registrations, Web administrator UI, D4 allowlisted positive diagnostic, paid model inference, real parent-nested selftest and unpublished Agent-local cross-machine delivery are still outside proven runtime acceptance. This does not block independent tests.
- **RISK-626 source guide updated, runtime distinction remains:** Outer `boxweave_info` now returns **static v0.26.2**, versus previous static v0.25.0. Installed app/worker/HTTP transport runtime on the actual outer host still requires a separately permitted host-level probe; v0.26.4 was installed and proved on a disposable Box and v0.21.3 historical Box, not thereby installed on the outer parent.
- **RISK-627 closed for the tested v0.26.4 Web path:** `GET /api/login` returned HTTP 500 in the fresh installed v0.26.2 Box due to blanket Flask exception classification. v0.26.4 special-cases Werkzeug `HTTPException`, with real post-upgrade 405 for login GET, 404 for an unknown route, and unchanged 401 for unauthenticated projects; 453 non-root source tests and full source CI passed.
- **RISK-628 OPEN — partial create acknowledgment:** A requested `create_container(enable_nesting=true)` left a new RUNNING Box even though the client reported an error, followed by an already-exists response to a repeated request. Exact resource was read back and eventually deleted. Source attempts to signal post-create nesting failure and not to retry, but this client's error was opaque. A reliable structured created-vs-nesting-not-applied outcome and installed gateway verification are still required. NEVER repeat an uncertain create until its original Box identity has been independently inspected.

## Host v0.26.4 nesting follow-up and source v0.26.5 (2026-11-01)

**RISK-623 partially closed on a new correctly configured Box:** The outer `boxweave_nesting(inspect)` now succeeds on v0.26.4, but the retained old failure Box `bwaudit-1101` has `security.nesting=false`. An isolated `bwnestfix-1101` was created with `enable_nesting=true`, independently inspected, stopped/restarted, installed inner BoxWeave/Incus v0.26.4, successfully launched nested Ubuntu `bw-nest-pass`, and executed Python `BW_NESTED_PYTHON_OK 3.12.3` with exit 0. This is real end-to-end PASS **only** for this correctly configured Box. Historical failed Box remains retained; do not infer its issue is fixed.

**RISK-628 source fix, runtime negative case still OPEN:** In v0.26.5, a successfully created Box with a failed/unverified post-create nesting step keeps the upstream Provider receipt and returns a structured MCP tool result marked `isError=true`, `state=partial`, `box_created=true`, `nesting_outcome=not_verified`, safe code and a no-retry instruction. No false atomic failure or automated replay. Source/CI tests exercise this; a live deliberately induced second-stage failure is still U/BLOCKED, and clients can differ in how they display errors.

**Separate remaining risks:** Parent host-hop shell `containarium-shell` is still missing but direct Box-IP SSH is functional; real external OAuth/TLS/client UI, positive D4, model inference, and private local-source-to-Box transfer remain unverified. See [latest scoped nesting acceptance](NESTING_ACCEPTANCE_0265_2026-11-01.md).

## v0.26.5 follow-up status (2026-11-01)

**Scoped nesting positive closed:** A new nesting-enabled Box successfully hosted/ran an inner Ubuntu/Python child and retained its execution after upgrading to v0.26.5; both test-owned Boxes were explicitly deleted and verified absent. **Partial-create negative path still U:** New structured `isError=true` tool output was CI-tested but not actually provoked against an installed v0.26.5 *outer* gateway. **Host owner-installed v0.26.4 unchanged:** Publication and nested-guest upgrade do not modify the user's host. Other earlier external/host-hop blockers remain, with original failed Box kept for comparison.

## v0.26.6 deep audit risk dispositions (2026-11-01)

**RISK-629 — OAuth admin denial throttle persistence, v0.26.6 source fix / pending source-bound test+release:** `RemoteCredentials.approve` inserted the failed authorization attempt inside the same transaction that raised `RemoteError`; `Store.db` rolls back on that error. Denials therefore failed to consume the advertised 5/min limit. The corrected code commits the rate row before the separate pending/password transaction. Synthetic SQLite/OAuth fixture verifies persistence, HTTP429, no authorization code, and recovery; no live administrator password or external OAuth client involved. **External protocol path remains U.**

**RISK-630 — selftest typed error visibility, OPEN/P2:** The newly real-tested v0.26.5 app selftest completed `create/hello/stop/delete` with `state=passed`, no nested child left. Deliberately incorrect confirmation was safely rejected with exit2/zero new children, but host CLI emitted generic `SELFTEST_FAILED_INSPECT_STEPS` due to dropping nonzero output. Fix typed error presentation in a separately reviewed bounded-reader slice; do not replace proven safe rejection with an unverified security claim. See [deep audit](DEEP_AUDIT_0266_2026-11-01.md).

Existing parent host-hop shell, D4 positive allowlist, real external OAuth/TLS clients, model inference, cross-platform native integrations, and private uncommitted-source transfer remain independently **U/BLOCKED**. These do not block the verified isolated scopes.

## RISK-631 — selftest SSH host-key trust-readiness race (2026-11-01)

**L FAIL on v0.26.6**: two new confirmed selftests on correct nesting Box returned failed `hello:TRUST_FAILED`, even after healthy 10/10 doctor and successful create. Both original tests stopped/deleted the child; zero residual instances. Host CLI reduced failure to generic `SELFTEST_FAILED_INSPECT_STEPS`. Source v0.26.7 adds bounded retry for only strict known-host identity/hello checks and typed response filtering. No host-key bypass, mutation replay or unverified host repair. Requires real installed v0.26.7 full-run PASS before closing. [Current evidence](SELFTEST_TRUST_RACE_0267_2026-11-01.md).

## v0.26.7 final risk disposition and cleanup (2026-11-01)

**RISK-631 closed for the tested installed happy path and typed refusal, not for all infrastructure:** [v0.26.7 source/CI/Release](SELFTEST_TRUST_RACE_0267_2026-11-01.md) succeeded; the owner-authorized `bwdeep-1101` v0.26.6→v0.26.7 upgrade completed with exact installed provenance and doctor 10/10. Application selftest `boxweave-selftest-e64976` completed its own create/hello/stop/delete with exit0 and persisted `state=completed`; immediate readback found zero children. Invalid confirmation returned typed `SELFTEST_CONFIRMATION_MISMATCH` exit2 without mutation. Trust timer saw `targets=1` while a Box existed; earlier `targets=0` observations at other times are not sufficient to conclude permanent failure. Original v0.26.6 `TRUST_FAILED` cases remain retained in evidence as regressions that led to the fix. **External SSH trust/path on other hosts still U.**

**RISK-629 source/CI fix accepted with installed v0.26.7 package:** The OAuth admin rate-ledger transaction fix included in v0.26.7 passed complete CI/fixtures and installed version checks; **real external administrator-OAuth + public TLS flow is U** because no live client or private credentials were used. Do not conflate a fixture's 429 behavior with external transport acceptance.

**Cleanup verified:** The disposable `bwdeep-1101` was stopped and deleted; the final MCP Box list contains exactly six historical Box entries and excludes it. Original `bwaudit-1101` failure environment intentionally preserved. [Full source/fixture/live acceptance](SELFTEST_TRUST_RACE_0267_2026-11-01.md).

## 0.26.8 deep reacceptance (2026-11-01 local / 2026-11-02 UTC)

**RISK-632 — A37 D4 service-state false unknown (P2, source-fixed / live positive U):** Existing service diagnosis treated `systemctl --user is-active` exit3 inactive/failed and exit4 unknown as generic command failure, despite bounded recognizable stdout. Added the explicit state/exit pair handler and negative regressions in candidate 0.26.8; source tests pass, **outer gateway allowlisted positive remains not verified**. See [A01–A38 ledger](DEEP_REACCEPTANCE_0268_2026-11-01.md).

**RISK-633 — A38 temporary-key fixture depends on umask (P3, source-fixed / T PASS):** Two fixture public keys defaulted to 0664 under umask0002, correctly refused by strict real key validator, causing seven incidental test errors. Set test-only 0644; no weakening of production security. Full candidate suite 458/458 passes under the reproducible umask0002.

**Other retained risks:** External desktop App/OAuth/TLS and real admin login, host-hop SSH, live D4 allowlisted read, deliberate partial-create failure, worker process-kill/no-replay, actual autosleep/wake, model inference, private cross-machine source transport remain U/BLOCKED; they do not block independent completed checks. Candidate 0.26.8 CI/immutable release/real upgrade and task-created Box cleanup require separate positive observations before the stable pin moves.

## v0.26.8 source-bound release and real installed risk dispositions (2026-11-02 UTC)

**RISK-632 source/installed parser closed within tested lower-level semantics, outer D4-positive still OPEN:** Full CI #41384817175 PASS, published v0.26.8 tag `1ea072865222585df96875dd4ae5b2d4ff90ed71`. Real guest v0.26.7→v0.26.8 upgrade exited 0 with exact provenance and doctor 10/10, installed parser recognized bounded `inactive`/exit4 service observation. Actual outer gateway default-deny still returns `PROJECT_DIAG_NOT_AUTHORIZED`; positive authorized-policy end-to-end read remains U/BLOCKED.

**RISK-633 T PASS closed:** 458/458 0.26.8 tests under umask0002 and successful published CI; strict production public-key validation remains unchanged. New independent nested selftest, stop/start persistence and exact task-owned cleanup all passed; historical Box failures preserved. Detailed evidence in [full new report](DEEP_REACCEPTANCE_0268_2026-11-01.md). All other external/host-hop/worker-crash/partial-create risks stay explicitly U/BLOCKED, not globally certified.

## v0.26.9 state-transition audit (2026-11-01 local / 2026-11-02 UTC)

**RISK-634 (P1, A39 source defect fixed / source candidate T PASS, public OAuth U):** v0.26.8 `RemoteCredentials.refresh` committed old-token revocation before the new pair was safely issued. A second-token exception left both a revoked refresh token and an undisclosed but valid access row. Authorization-code exchange likewise consumed a single-use code before issuance. v0.26.9 candidate puts revocation/code deletion and pair issuance under one transaction, with failure-injection, SQLite constraint and concurrent single-use/Flask HTTP regressions. Do not mark external TLS/OAuth ACCEPTED from fixture tests; v0.26.9 CI, immutable published artifact and independent installed checkpoint must complete. [Full A39 report](STATE_TRANSITION_AUDIT_0269_2026-11-01.md).

## v0.26.9 final A39 resolution and remaining limits (2026-11-02 UTC)

**RISK-634 closed within installed local credential transactions, external client still U:** Published v0.26.9 source `008f659593376c7df1608599bd523885197a6362`, source CI verify/publish success, 14 assets. A fresh installed v0.26.8 showed old refresh revoked and one orphan valid access token on second issuance failure. A real installed v0.26.9 upgrade and the identical SQLite fault returned zero revoked predecessor/zero new credential, preserved the authorization code on injected exchange fault, then allowed valid one-time recovery. Independent synthetic loopback HTTP bearer/Origin/revocation combinations passed before and after upgrade. This does **not** validate public OAuth/TLS callbacks, a real ChatGPT/desktop OAuth client, or actual operator credentials. See [A39 report](STATE_TRANSITION_AUDIT_0269_2026-11-01.md).

**Lifecycle and cleanup:** Same new test Box accepted online CPU/disk growth, successful selftest create/hello/stop/delete before/after upgrade and stable runtime after stop/start. Original test Box was independently deleted, original six retained. Post-restart `TRUST_NOT_READY` recovered by observation without trust bypass. Forced unknown-write worker crash, root-preauthorized D4 positive, external OAuth and real model inference remain U/BLOCKED.


## 2026-11-02 promoted stable installer scope (v0.42.1)

[Source-bound real-Box acceptance](STABLE_PROMOTION_2026-11-02.md)
records fresh v0.42.1 and in-place v0.28.0 -> v0.42.1 Ubuntu
test Box success, with authenticated Provider MCP list, private Web,
real guest selftest and cleanup. **Not closed:** default-disabled
global multi-Agent R-04 protection; external TLS/OAuth/desktop
client validation; snapshot/restore/clone upstream absence; production
fleet downgrade and unsupported older app schema; real cross-principal
SSH/HTTP/Web races. Post-restart outer SSH could briefly report
TRUST_NOT_READY independently of guest BoxWeave health; collect
independent read recovery evidence, do not weaken host-key checks.
No historical risk is globally closed by this scoped release test.


## Current v0.42.2 deep reacceptance boundaries (2026-11-02)

[Live and source evidence ledger](DEEP_REACCEPTANCE_0422_2026-11-02.md). Fresh v0.42.2 install, authenticated Provider read, real nested selftest, cold restart, and old v0.26.1/v0.26.4 read-only doctor checkpoints passed; the exact-source 640 tests and full GitHub CI passed. R-04 protected-mode same-Box races and Web/SSH/HTTP bypasses, R-02 provisioned request recovery, D4 authorized positive, public OAuth/TLS/desktop, E2-E4 snapshots/restore/clone, and current installed old-to-new migration remain explicitly UNVERIFIED or BLOCKED. Minor usability note: BOI-1 rejects absolute paths and symlink reads as intended but currently wraps those validation failures in a generic trust/workload gate error; clarify the relative-to-home input contract and consider typed non-disclosing validation codes in a separately versioned change. No new production outage or security bypass was demonstrated.


## Updated real host v0.42.3 Gateway acceptance (2026-11-02)

[Live observed host acceptance](HOST_0423_GATEWAY_REACCEPTANCE_2026-11-02.md) supersedes the v0.42.2 P3 generic-trust error caveat **within the tested R-01 scope**: absolute/traversal requests now return `BOX_PATH_INVALID`; file/directory symlinks fail closed with `BOX_READ_UNVERIFIED`; valid reads and old Box enumeration passed. **P3 still open, D3 observability only:** known denied JSON-RPC operations are stored as `UPSTREAM_JSONRPC_ERROR/unknown` in the Gateway event journal; avoid equating this label with an actually uncertain mutating Provider effect. Before addressing, preserve fault redaction and distinguish server-side deterministic denials from unknown dispatched writes in code/tests. Connector-discovered R-01 tool descriptions may remain stale until a fresh client connection; independent live catalog refresh is U. Root-authorized R-02/R-04/D4 positive and external client scenarios remain U/BLOCKED.


## D3 classification source fix (v0.42.4 candidate, 2026-11-02)

[Specific fixed-code classification and source tests](D3_CLASSIFICATION_FIX_0424_2026-11-02.md) resolve the v0.42.3 deterministic-refusal P3 issue at source level while preserving `unknown` on untrusted Provider JSON-RPC errors and potentially dispatched mutations. Existing historical rows are not rewritten. **Installed L/U** until exact release and pinned live Gateway journal observation; D4 positive, R-02 and R-04 remain separately unaccepted.


## D3 P3 fixed and independently installed — v0.42.4 (2026-11-02)

[Observed baseline -> fixed Gateway journal and cleanup](D3_CLASSIFICATION_LIVE_0424_2026-11-02.md) closes the earlier D3 deterministic-refusal misclassification within a verified pinned installed v0.42.3 -> v0.42.4 test Box upgrade. Provider-call failures and uncertain writes remain conservatively unknown; historical journal entries remain unchanged. **Deployment gap remains:** the owner's upgraded v0.42.3 host was not upgraded during this test; recheck D3 there only after independent host installation of v0.42.4. External clients and unrelated R-02/R-04/D4 positive gaps remain open.


## Post-v0.42.6 incident reacceptance and unresolved scopes (2026-11-02)

[Broad real-Box evidence and precise open issues](BROAD_REACCEPTANCE_0426_2026-11-02.md). R-02 bare exit_code and CAS stderr-only exit parsing are source+CI+live fixed by v0.42.5/v0.42.6, with no automatic replay of preexisting UNKNOWN and no broadened authority. New **OPEN uncertain-write**: two concurrent Provider create_container calls returned errors while both named Boxes appeared RUNNING in independent Incus, and root target registration briefly lagged. Treat both as UNKNOWN, investigate Provider timing/state/receipts; do not reinterpret error as proved failure or retry without observing the original. The incident Box bwfull-1102-e2e-container is stopped and retained, with private UNKNOWN metadata and nested test guests; all test-only root policies/config changes reverted. Not closed: globally protected Web/SSH/HTTP R-04 dispatch, external OAuth/desktop, old-version fleet migration and downgrade, real in-flight mutator crash, self-hosted sleep/wake, E2/E3/E4 missing Provider contracts, or real model inference. Stable v0.42.1 pin and operator production host unchanged.


### v0.28.0 installed direct migration reduction (2026-11-02)

[Live second Box upgrade addendum](BROAD_REACCEPTANCE_0426_2026-11-02.md) closes the missing **v0.28.0 -> v0.42.6 fresh-test installed upgrade** with exit-0 pinned installers, doctor 10/10 before/after, schema-12 actual DB to schema-13 runtime and preserved original admin password record. Stored v12 files are intact backups. v0.26.x direct upgrades, multi-tenant production data migration and actual downgrade/rollback remain U. The successful second Box was stopped/deleted, and original seven plus separately stopped incident Box remain.


## v0.42.7 independent test and D4 empty-log format correction (2026-11-02)

[Exact real-Box discovery and source fix](REACCEPTANCE_FULL_RETEST_0427_2026-11-02.md). v0.42.6 D4 wrongly classifies exact successful empty Provider journal `exit_code: 0\n` and genuine stderr-only service nonzero as unsupported wire format. v0.42.7 source accepts only exact successful empty logs and classified nonzero stderr-only without exposing raw guest stderr, preserving fail-closed under missing user bus. CI/installed pending at source checkpoint. Simulated two-client R-02/R-04/CAS outcomes PASS within tested protected Gateway routes, but global multi-client protections and actual Web/HTTP OAuth do not follow. Two concurrent Provider creates succeeded on this rerun; prior HTTP-500 + created cases remain unresolved upstream intermittent risk, not safe to auto-retry.


### v0.42.7 bounded D4 live closure and residual risks (2026-11-02)

[Independent v0.42.7 live upgrade and full risk/acceptance ledger](REACCEPTANCE_FULL_RETEST_0427_2026-11-02.md). The v0.42.6 empty successful journal/provider result and stderr-only failed user-service format defect is **closed** at source and installed Gateway after pinned v0.42.7 upgrade, with 10/10 doctor, no raw stderr, unchanged D4 root allowlist/security and task-owned cleanup. Real guest user systemd service positive remains blocked by absent user manager/lingering denied in this lab; E2-E4 still lack authenticated upstream snapshot/restore/clone/rekey contracts. HTTP OAuth desktop, real worker cross-transport conflict, complete rollback, sleep/wake and model inference stay U. Prior intermittent Provider HTTP-500 with successfully created Boxes remains an OPEN upstream async race: new two-client run succeeded but does not prove root cause repaired.


## 2026-11-02: D4 positive closed; cold-create sleep and upstream environment gaps

[Verified v0.42.8 user-service and 0.120.1 auto-sleep evidence](BLOCKED_REACCEPTANCE_0428_2026-11-02.md). The previous D4 positive-service BLOCKED state is **closed within an isolated installed Gateway**: v0.42.8 sends fixed service/journal commands with the guest UID's `/run/user/<uid>`, not the caller's environment; active/inactive/user-bus-unavailable and unauthorized service cases were observed with no raw logs. A separate remaining Provider **cold-create idle tracking gap** was reproduced: `toggle_auto_sleep` on newly created guest had no last-start stamp/traffic and never stopped; a test-only explicit stop/start stamped a time and the actual daemon ticker then stopped the Box. Do not automatically restart production Boxes as a workaround. HTTP request wake needs an authorized app-hosting route, Caddy/gateway and a reachable host, not available in this lab; still U/BLOCKED. Existing 0.120.1 CLI offers snapshots but authenticated MCP lacks those methods and test storage is `dir`; E2/E3 remain unimplemented pending tested authenticated adapter/ZFS storage. Whole-box restore and clone/rekey remain future v0.126.0 discovery. [Future upgrade plan](../design/CONTAINARIUM_0126_UPGRADE_E2_E4_HANDOFF.md). The owner host and stable v0.42.1 pin were unchanged.


## 2026-11-02 upstream candidate P-01 verified locally, not released

[Detailed nested v0.126.0 cold-create/wake acceptance](CONTAINARIUM_0126_AUTOSLEEP_WAKE_LIVE_2026-11-02.md): historical 0.120.1 no-start-stamp and first HTTP wake 502 are now **fixed T+L in a locally compiled v0.126.0 candidate** committed to the owner's [Containarium fork](https://github.com/vanishs/Containarium/commit/40b9d650584ceef66ea196ec6ee6dd9d88263f2f). This is **not** an upstream released fix, official upgrade, or BoxWeave shipped feature. The GitHub integration returned 403 creating a PR in FootprintAI/Containarium; an authorized owner must submit for upstream review, then await merge/release and separately test upgrade/rollback. Cold birth/toggle clock and routed first-request HTTP 200 passed live; untrusted wake HTTP 403. Caddy app-hosting wiring is required; the test distro Caddy catch-all needed removal. Source-only/local CLI `create --idle-stop`, public TLS and concurrent wake remain unverified; protect the stable pin and production Provider until acceptance. [Upgrade handoff](../design/CONTAINARIUM_0126_UPGRADE_E2_E4_HANDOFF.md).

## Current main-based upstream patch awaiting PR/merge — 2026-11-02

The [upstream latest-main source candidate `28b18edc`](https://github.com/vanishs/Containarium/commit/28b18edc550f79a049cfa7f08fcf5fd09b926a1e) has a direct parent of official `main` `4be7f81a`. It passed focused security/auto-sleep Go tests, full server Go package tests and final daemon compilation (details in [handoff](../design/CONTAINARIUM_0126_UPGRADE_E2_E4_HANDOFF.md)); two tenant names in newly added logs were changed to `%q` per latest upstream guard. New branch has **NOT** been independently installed/live-tested, while the earlier v0.126.0-tag patch has real nested Box HTTP wake evidence. Connected GitHub integration returned 403 attempting to open an upstream PR, so official merge/release and host upgrade still remain gated. No production/provider changes; task-owned build Box deleted.
