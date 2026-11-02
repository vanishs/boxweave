# Remediation validation and next-Agent queue

## Phase 1 unified operation base — P1-B1 identity/private storage adapter

Candidate source: `bwmini/coordination_runtime.py`, `tests/test_coordination_runtime.py`. Test on the exact candidate SHA: `python -m unittest discover -s tests -p test_coordination_runtime.py`, then full fixture/knowledge/packaging CI. The adapter must reject missing storage, symlink, unsafe directory/file, invalid identity and cross-incarnation mismatch; it must not create or authorize production shared storage. **Status: pending exact-SHA CI; live host and protected mutator integration NOT DONE.**

## Phase 1 unified operation base — P1-A initial source review (2026-11-01)

Source baseline: `1480b0671cab543d4664b57b0079f1d8962981fc` (v0.28.0). [Implementation/acceptance ledger](../process/PHASE1_UNIFIED_OPERATION_BASE.md). Initial new files: `bwmini/operation_coordination.py` and `tests/test_operation_coordination.py`. **Status: source/fixture PASS at `05a95aea`** via [CI run 41402675634](https://github.com/vanishs/boxweave/actions/runs/41402675634), verify job success; evidence-only follow-up commit requires new CI. Not an installed acceptance. Run `python -m unittest tests.test_operation_coordination` (or `python -m unittest discover -s tests -p test_operation_coordination.py`), `python scripts/check_knowledge.py`, and `python scripts/validate.py` in a prepared checkout. Check atomic multi-resource conflicts, stale/unknown retention and DB unavailability. No privileged host/Provider action or cross-principal SQLite grant has been performed. Pending P1-B: audited permissions and Web/gateway adapter coverage; pending P1-C: actual SSH/HTTP/Web mutual exclusion and migrations. Never treat this unit-core slice as live protection.


This is an append-only *planned vs executed* ledger. Do not write PASS unless the command actually ran, its exit/status was inspected, and the source/environment are recorded. The repository edit session operates through the GitHub connection **without a runnable trusted checkout**, so it cannot claim routine Linux or browser tests.

| Change | Test to run in authorized environment | Present evidence / acceptance |
| --- | --- | --- |
| 0.10.0 baseline, R-00/R-01 | `python scripts/check_knowledge.py`; link/README/VERSION/lock/scope consistency; `npm --prefix web ci && npm --prefix web run build` | U — not run in this source-only edit. Check especially retained historical links and IDs |
| Bootstrap/installer R-02 | Shell syntax, explicit-help/no-side-effect dry run, malformed inputs, existing Incus/Provider intact, isolated Ubuntu 24.04 fresh-host+reinstall+repair, signed/pinned upstream provenance, no-secret output | U — no host installation authorized |
| Management CLI R-05 | Read-only help, no config, missing service, version mismatch, forbidden target, no secret content, MCP-down; exact exit contracts | U — no host/user manager available |
| App distribution | `.venv/bin/python scripts/validate.py` and `tests/test_distribution.py`; isolated package build/install/upgrade, DB backup/marker, re-run and admin setup | U — old historical evidence does not count for new source |
| Web/clients | `.venv/bin/python scripts/validate.py --browser` seven fixture flows, plus separately approved four-profile Windows live registration/GUI | U — fixture is not real client evidence |
| Provider/Incus/Tenant | Independent MCP initialize/discovery/read; pinned upstream capabilities; authorized policy/packet, boot/confinement, public TLS | U — requires explicit live authorization |

Keep source SHAs, exact commands/exits, stdout/stderr redaction, any retained fixtures, known failures and required permissions in the actual Agent's result. Do not run live test drivers without their original scope and cleanup approval.

## 0.11.0 read-only management CLI — R-05 partial

Changed: `bwmini/manage.py`, `boxweave.sh`, distribution archive member list and five independent unit tests. An analogous standalone source copy was checked with `python -m py_compile` (exit 0), `bash -n boxweave.sh` (exit 0), and `PYTHONPATH=. python -m unittest -v tests/test_manage.py` (five passing tests, exit 0). The submitted shell wrapper differs in its equivalent `$0` location expression and the submitted tests have slightly different formatting; **full submitted-revision testing was not run**. Next Agent: execute project suite, real Incus exact inspect, missing/foreign user-service, real authenticated MCP read, build/reinstall archive containing CLI wrapper, and secret leak review.

## 0.12.0 host entry — R-02 partial

Changed: `install.sh`, safety tests, operator documentation. Analogous isolated copy: `bash -n install.sh` exit 0, `bash install.sh --help` exit 0, `PYTHONPATH=. python -m unittest -v tests/test_host_bootstrap.py` exit 0 (three tests). The submitted source and tests require new checkout validation; this environment is Debian and **no Ubuntu/Provider/app install was attempted**. Next Agent must test a malicious/missing symlinked artifact, user/session bus, deliberate root provisioning, exact package digest, first admin, successful MCP read, service restart, DB backup, interrupted update repair. Full new-host installer still blocked by secure launcher, credentials, account/bus/bootstrap and root installer integration acceptance.

## 0.13.0 CI and traceability

Changed: non-privileged GitHub Actions checks, new version/README metadata validation and unit test, REQ-019/020 and AC-019/020, ADR-007, risks/matrix/source register. The workflow has not been observed running for this new commit; actual runner SHA/status/logs must be retrieved before reporting success. Linux fixture browser and upstream/host tests remain separate.

## 0.13.1 corrective patch

GitHub Actions run 40980715266 for 0.13.0 commit ca3d06d failed `Documentation, release metadata, and traceability` with exit 1 and exact diagnostic `README does not declare the current release version`; subsequent steps were skipped. Cause: README front page still said 0.10.0 after multiple feature-version bumps. Patch updates README and VERSION/Python/Web/lock root to 0.13.1. Re-run CI on this SHA; do not infer build/test results from the earlier interrupted run.

## 0.13.2 semantic and browser audit

Added [F-001–F-015 parity table](FEATURE_PARITY_ARCHIVE_051.md), [Web source/navigation mapping](WEB_WORKFLOW_PARITY.md), and clarified the selected four Windows-hosted client profiles. All parity verdicts are source/decision judgments; **no new browser or Windows runtime acceptance**. Next Agent: run seven `--browser` fixture scripts individually and four selected Windows client scenarios. Keep fixture/source/historical/real evidence distinct, record actual stdout/exit and console/screenshot artifacts.

## 0.13.3 R-08/09 documentation-only delivery

Added explicit legacy migration stop/backup/parallel-deploy and rollback boundaries, operational diagnosis table and upstream ownership/capability contract. No legacy DB migration or host/Provider interaction executed. Next Agent must validate documented commands on a separately authorized non-root Ubuntu environment and full-host clean install lab; signed/pinned upstream artifacts, credential owner, effective Guard/packet and boot need separate evidence. Static docs are not new live passes.

## 0.13.4 installer source integrity fix

The host coordinator now demands an independent `--distribution-sha256` check and root ownership/re-hash of an explicitly invoked privileged Provider installer. Added missing-installer-digest negative test. A fresh GitHub CI run must validate the exact submitted revision; **no privileged upstream installer or application install was invoked in this source-only change**. Next Agent should test symlink/digest race, root-owned source and untrusted directory behavior in a disposable Ubuntu sandbox before relying on host integrity.

## Verified GitHub CI and next-Agent acceptance handoff (source 0.13.4)

**Actually executed** in GitHub Actions on Ubuntu 24.04 at source commit `17cf4c23f57afdcbf17ae6246270395935c01965`: [run 40981405630](https://github.com/vanishs/boxweave/actions/runs/40981405630), status `completed`, conclusion `success`, job `123822087557` conclusion `success`. The job steps `Documentation, release metadata, and traceability`, `Safe shell syntax`, `Install locked Python test dependencies`, `Python unit and contract tests (fixtures only)`, and `Frontend typecheck and build` each concluded `success`. The earlier initial run `40980715266` failed the README-version check and was corrected in 0.13.1. Subsequent release tests on 0.13.1/0.13.2/0.13.3 also had successful CI conclusions. This is actual **CI** evidence, not a real Provider, Incus, browser, Windows or new Ubuntu host test. This handoff-only documentation commit requires its own new CI status.

**Priority acceptance tasks for the next Agent (not completed):**
1. **P1, AC-019 / RISK-610:** On a specifically approved disposable clean Ubuntu 24.04 x86_64 host, design and implement the missing complete bootstrap path: trusted pinned Incus/Containarium prerequisites, app user and user-manager initialization, independent scoped MCP launcher/credential and SSH/host trust ownership. Verify no existing Provider/Incus/legacy instance can be silently replaced, no secret printed, first administrator, real authenticated MCP read, rerun/upgrades, interrupted-install repair, reboot/autostart, backup and rollback boundary. The current `install.sh` intentionally cannot do all of this.
2. **P1, AC-020 / RISK-611:** Verify installed `boxweave.sh` from the packaged archive and real (read-only) exact Incus inspection and service/failed-launcher cases. CLI remains a subset of the old interface; assess whether old job/selftest/client-generator goals require new approved implementations instead of unsafe aliases.
3. **P1, R-04/R-06:** Run `.venv/bin/python scripts/validate.py --browser` in an authorized prepared non-root Linux project environment (seven fixture Chromium workflows) and actual selected four Windows Codex/ZCode Native/WSL profiles with separate GUI/inference evidence; do not claim these from GitHub CI.
4. **P2, R-09:** Independently verify approved upstream/native Tenant/Guard capability, host-key/token ownership, packet enforcement and TLS/reboot only with corresponding live permissions. Preserve `archive/0.5.1`, all old deployment evidence, backups, keys and services. No automated legacy migration.

Failure reporting contract: change list + commit SHA, environment, exact command, exit code, sanitized log/evidence, confirmed/unknown category, affected REQ/AC and RISK owner. Unexpected blockers should be recorded and independent safe tasks continued without inventing test results.

## 0.13.5 README audience and onboarding order

Changed: the README now puts the real `install.sh plan/apply` workflow first, moves the isolated fixture into a final `Development & testing` section, distinguishes synthetic versus live Provider behavior, and explicitly preserves the incomplete fresh-host-install boundary. Added a source-level README ordering regression in `tests/test_release_metadata.py`. Bumped release metadata consistently to 0.13.5. **No live installation, Provider access or browser action was performed for this documentation change.** Next Agent: run `python scripts/check_knowledge.py`, release-metadata tests, full fixture-safe CI; manually verify the documented `install.sh --help` flags on an approved disposable Linux checkout, without performing host writes.

## 0.13.6 public documentation and piped-installer safety

Restructured the README with a clear desired single-command entry, an explicit not-yet-available release warning, real prepared-host command/flag table, client support matrix, and management command table; moved synthetic fixture to `docs/DEVELOPMENT.md`. A zero-argument `install.sh` invocation now fails explicitly with `DEFAULT_INSTALL_UNAVAILABLE`, without executing privileged host changes. Added no-argument and README layout regressions. This does **not** satisfy unattended new-host installation: no verified downloadable application artifacts, independent launcher/credential setup, or clean-Ubuntu acceptance exists. No real install or browser tests executed in this source edit. Next Agent must implement that complete path and verify it on an approved disposable machine before claiming the one-line installer works.

## 0.13.7 CI correction

Previous source check failed at GitHub CI run 40985248991 due to an incorrect relative link in the relocated developer guide. Fixed the link, normalized the default-installer error newline format, and corrected the README terminology regression's regex escaping. New CI is required; no real host deployment was attempted.

## 0.13.8 CLI package naming correction

CI run 40985401561 completed with one failure in the new README terminology test; it found an active `--archive` option in public copy. The public installer now accepts `--package` as the preferred alias while preserving the existing option for backward compatibility; standalone distribution accepts the same alias and retains the internal argument destination. Rewrote active installation examples and added a negative test that the new alias is parsed before any installation action. Full fresh-host bootstrapping remains not implemented. GitHub CI of this exact revision must be checked; no real host installation was run.

## 0.13.9 active guidance terminology cleanup

Replaced outdated stage/branch labels in active AGENT, product, scope, host, and application-install prose. Historical evidence and runtime identifiers are unchanged. The persisted managed service and installation directory identifiers must not be renamed in-place without a designed backward-compatible migration. Some older process/verification materials and path names retain historical labels; inventory and isolate them before attempting a repository-wide rename. No real host execution occurred; CI must verify this submitted source.

## 0.13.10 user-facing installation history separation

Preserved the previous operator installation guide in `docs/INSTALL_HISTORY.md` and reorganized current `docs/INSTALL.md` around explicit application preparation and a selected private prefix. Removed stale stage/evidence prose from current installation instructions without changing runtime file locations or schema. New installation examples use an explicitly selected prefix and the supported package alias; existing deployed locations must not be renamed without a separately reviewed migration. No host installation test performed. GitHub CI source checks are required.

## 0.14.0 current execution

The source-only environment description above belongs to the historical entries. The current authorized assigned-LXC, Windows and dedicated app bootstrap runs are recorded in [MAIN_014_REVIEW](MAIN_014_REVIEW.md). Preserve planned versus actually executed boundaries and keep RISK-610 open until the full published clean-host path exists.

## Corrected public-document convention (SRC-099)

The owner clarified that the earlier change was about current product narrative, mainly README, not forbidden substrings in all code/docs. The README regression now allows ordinary release archives, `--archive`, technical phases and upstream API v2, while rejecting obsolete archived-release/development-stage narratives and BoxWeave-v2 branding in prose. The compatible `--archive`/`--package` options are retained. Direct contributor links and historical records remain; the current installation/history split is retained.

## Streamed installer iteration

SRC-100 authorizes source fixes/commit/push/reinstall within the dedicated two-box test. First actual main9f03b9f streamed command failed installer2. Current0.15.0 source/package/CI and eventual live outcomes belong to [STREAMED_INSTALL_TEST](STREAMED_INSTALL_TEST.md); no pass is assumed from publishing source.

## 0.16.0 nesting slice

TASK-017/SRC-103: independent MCP create field, exact stale-checked nesting tool, Web choice, installer preflight and read-only diagnosis. [NESTING_VERIFICATION](NESTING_VERIFICATION.md) owns actual0.15.1 comparisons, new source validation and U privileged integration. Next: verify installed helper/sudo policy, peer inspect/plan, reviewed apply and unchanged target/no restart. No general parent-shell credentials or automatic recreation.

Actual0.16.0 source validation: .venv/bin/python scripts/validate.py --browser in the assigned non-root LXC, exit0;209 tests/build/seven browser workflows. Exact report and untested privilege surface: [NESTING_VERIFICATION](NESTING_VERIFICATION.md).


## 0.17.0 existing Containarium reuse and installer rerun

Changed: the streamed entry now detects a complete preinstalled Containarium/Incus as an external Provider, validates its loopback API, root-private secret-file metadata and nonempty local Incus storage, and skips Provider installation, Incus init, PostgreSQL setup and service mutation. With only a separately preinstalled Incus/pool, it provisions the pinned managed Provider without resetting storage. The recorded Provider mode is immutable across BoxWeave reruns. Updated streamed help, README and operator acceptance notes. Historical `archive/0.5.1` conditional-provider reuse was referenced, not copied. Added offline detection/refusal/read-only probe contract tests in `tests/test_online_adoption.py`.

**S:** source review via GitHub connection. **U:** exact submitted revision unit/shell/CI and any live host acceptance until results are inspected. Execute the tests and live scenarios from [EXISTING_PROVIDER_INSTALL](EXISTING_PROVIDER_INSTALL.md). The published CPython 3.12 / Ubuntu 24.04 release is not a verified Ubuntu 26.04 installer; do not pretend this feature change fixes that separate packaging problem.


## 0.17.1 custom Provider CLI path correction

Allow explicit `--provider-cli` to identify a reviewed installed Containarium binary outside PATH. Retain Incus and Containarium CLI detection and the external Provider read-only checks. Added an isolated path-selection regression. **S:** source-only GitHub edit; **U:** exact-commit CI and live custom-path acceptance until independently verified.


## 0.17.2 Ubuntu/Python minimum version gate and wheelhouse

Changed the **minimum** versions to Ubuntu >= 24.04, CPython >= 3.12 (x86_64) in the streamed entry, online host coordinator, scoped bootstrap and application installer. Updated the source-bound release selection, Ubuntu-specific signed Incus APT source, managed PostgreSQL version selection, and multi-ABI hash-locked offline wheelhouse. Existing Containarium remains externally owned; no Provider/Incus/secret mutation. Added `tests/test_platform_minimums.py` and a CPython 3.14 offline-installation CI smoke step.

**S**: code changes inspected; **U**: live Ubuntu 26.04 mount preflight, real existing-Provider reuse, authenticated MCP, Web, reboot and interruption/repair. CI and release evidence should be updated after the exact commit completes. On Python versions later than the bundled wheelhouse, the package manager must fail closed; no unverified network wheel download permitted.


## 0.17.3 interactive and non-interactive BoxWeave client generation

New `boxweave client` server-side wizard and explicit `--agent codex --transport wsl --admin-user USER --output FILE` mode use one generator and source-bound PowerShell template. Windows Codex stays native; SSH/keys live in the specified WSL distro. The script pins the exact server public key, calls `online.py enroll` only under separately authenticated Ubuntu sudo authorization, probes read-only MCP before touching Codex registration, and requires explicit `-Replace` for an existing MCP entry. `-NoEnroll` + `-AlreadyEnrolled` support two-person approval. Existing external Containarium, Incus, bcachefs and PostgreSQL remain out of scope.

**S**: source and fixture checks; **T**: exact-commit CI including PowerShell syntax; **U**: run on a real Windows native Codex + WSL SSH client, with multiple WSL distros, password and SSH-key administrator auth, conflict refusal and approved replacement, offline MCP, changed host pin, and round-trip list_containers. Confirm byte-identical Provider and Incus after generation. Non-WSL profiles still use the separate reviewed profile helper; this generator intentionally does not imply support for those profiles.


## 0.17.4 Windows ChatGPT/Codex desktop without CLI

Source update: generated Windows + WSL SSH installer does not require `Get-Command codex`, an external Codex CLI, or execution of a WindowsApps-packaged binary. The Windows desktop app shares `%USERPROFILE%\\.codex\\config.toml` (or the user's `CODEX_HOME`); the script updates only the BoxWeave MCP table, refusing ambiguous TOML, backing up existing configurations, and requiring `-Replace` for conflicts. Administrator SSH enrollment defaults to password authentication to prevent too many keys; `-AdminAuthentication publickey -BootstrapIdentity /absolute/wsl/path` is available for key-only administrator SSH.

**S**: updated source and fixture regression coverage. **T**: full CI and generated PowerShell grammar. **U**: real ChatGPT desktop recognition after restart, Windows/WSL password+sudo, optional authorized key login, preserve other Codex MCP settings, safe rerun and actual MCP read. Existing Incus and Containarium remain untouched.


## 0.17.5 Windows WSL executable selection

Fix generated PowerShell `$wsl` resolving from multiple `Get-Command wsl.exe` matches (System32 and WindowsApps). Select `$env:SystemRoot\System32\wsl.exe` as exactly one absolute executable, with a required file check. Regression test asserts no Get-Command WSL ambiguity and CI parses generated PowerShell. **S/T**: source and CI syntax/unit. **U**: actual Windows host with multiple `wsl.exe` matches, selected distro `Ubuntu`, real enrollment and MCP reading. Direct local patch to existing script is safe; no Containarium/Incus changes.


## 0.17.6 Windows Codex existing nested MCP TOML

Incident: real Windows + WSL client passed public-key enrollment and MCP initialize/tools/list/list_containers, then refused the client's valid `[mcp_servers.boxweave]` root plus two `[mcp_servers.boxweave.*]` subtables. Fixed overbroad nested-table rejection. A new narrow PowerShell merger rewrites only the four BoxWeave transport fields with explicit `-Replace` when the prior transport differs, preserves other root keys, comment lines, sibling MCP tables and nested subtable content byte-for-byte, and remains idempotent. Duplicate keys/tables, multiline transport arrays, invalid quoted/dotted table variants and nested-only tables refuse automatic changes. Added `tests/test_client_ps_merge.py` running `pwsh` with a real nested TOML fixture and parsing the resulting TOML using Python `tomllib` (unit fixtures, no SSH or provider calls). **U**: Windows ChatGPT/Codex App restart and real MCP tool recognition. Preserve enrolled key and invoke new setup script with `-AlreadyEnrolled -Replace` after reviewed transport conflict; do not re-enroll, reset Containarium or mutate Incus.


## 0.17.7 Windows Codex WSL SSH destination generated as two argv tokens

Observed on real Windows Codex: server enrollment and the independent Python SSH/MCP probe passed, yet the generated Codex MCP command put `boxweave@` and `192.168.2.233` in separate argument slots, leading to `Could not resolve hostname : Name or service not known`. Fix: construct `$sshTarget` as a scalar before `$sshArgs` and insert exactly that variable into the argument array. Add a `pwsh` execution fixture to assert `args[-2]` is exactly `boxweave@192.168.2.233`, without either fragment; this catches defects that grammar parsing alone cannot. The user manually corrected their configuration, then independently verified MCP initialize (service 0.120.1), 59 discovered tools, and a successful `list_containers` call returning `No containers found.` **U**: verify Codex desktop actually injects MCP tools in a new session; the independent launcher check does not prove that. Do not touch the client's now-working configuration or repeat enrollment for this generator-only fix.


## 0.17.8 Restore archived managed workload identity guidance and enforce the create contract

**Original incident:** a real Codex agent created `hello-world-test-container` (10.100.0.2, Running), but `connect` failed public-key authentication and Python `hello world` was **not executed**. The actual `create_container.ssh_keys` request and guest authorized keys have **not** been inspected; the key mismatch is a source-supported hypothesis, not a confirmed cause.

**Historical code S/D:** `archive/0.5.1/server/clients.py` exported the managed `container_public_key`; Windows/POSIX setup persisted a scoped AGENTS policy containing it; old `server/smoke.py` passed the exact key to create, checked `connect` identity and fixed Python output. The main generator had dropped public workload-key transfer/guidance; its MCP gate only checked Ed25519 syntax. The independent client transport key must never be confused with server->workload identity.

**Changes S:** `bwmini/client_setup.py` now requires the validated public `managed_public_key` in the server's root-owned state, bakes it into the public source-bound installer and merges an isolated Windows Codex AGENTS block; optional `-WslGuidance` targets the explicitly selected distro's Linux Codex home. Previous instructions are backed up; unrelated content and other config survive; symlinks and ambiguous markers fail closed. `bwmini/mcp_gate.py` reads its own non-root managed `.pub` only, verifies owner/regular file/safe mode and refuses missing managed key from `ssh_keys` before forwarding creation, while allowing extra explicit Ed25519 keys. No private key/token read or host/guest configuration change was performed through this repository edit.

**Tests T/U:** Newly authored offline tests cover absent/malformed generator public metadata, simulated managed key mismatch/extra keys/duplicate keys/unavailable file and PowerShell/WSL guidance merge execution in isolated temporary homes. Run exact GitHub CI and review failing logs before claiming T. Live acceptance remains **U** until the existing box is reconciled and MCP `connect` actually executes `python3 -c "print('hello world')"` with stdout `hello world\n`, exit0. This change never edits existing box authorized_keys and does not authorize real Incus/Provider/sshd mutation.


## 0.18.0 English artifact cleanup and Linux-owned workload SSH identity

Fixed source0.17.14's accidental Chinese in installation output, CLI onboarding, PowerShell banner, README and tests. `AGENTS.md` already required English project artifacts; the instruction now explicitly distinguishes conversational language from repository language.

The MCP create gateway reads its own trusted full workload public key and appends it to every new `create_container.ssh_keys` before forwarding. Additional distinct Ed25519 keys remain optional/validated; the missing-key Provider private-key fallback is not reachable through this gateway. The schema no longer requires clients to supply the host key. The generated Windows script no longer modifies client `AGENTS.md`/WSL instructions and makes the client-reachable `$ServerHost`/`$ServerPort` editable. The server host-key pin and restricted SSH enrollment remain unchanged. Approximately 6.7 KB of client generator source is removed. The old generator was shorter because it downloaded its implementation from a fixed commit at runtime; no new download dependency was introduced.

**S**: source review. **T**: exact-main CI Python tests, generated PowerShell parse and package validation (check published workflow before claiming pass). **U**: real Windows/WSL onboarding, authenticated new create without client `ssh_keys`, successful connect/Python stdout/exit, nesting identity, and existing-box SSH repair. Do not change old box credentials or client instruction files automatically.


## 0.18.2 Successful install version banner and Incus APT solver correction

A disposable GitHub Actions Ubuntu 24.04 install against v0.17.12 failed in `signed_apt_prerequisites`: three Incus packages were pinned at 7.5.1 while the unpinned `incus-base` candidate was 7.6, so APT rejected the inconsistent closure. This evidence is NOT a successful new-host installation and was gathered from the draft GA test PR #4 (workflow run 41141400914).

The managed fresh-host path now passes `incus`, `incus-client`, and `incus-extra` without partial exact-version pins, runs an identical-package `apt-get -s install --no-upgrade` first, then installs only if the simulation succeeds. Incus and Provider are never upgraded on an existing complete deployment; the minimum runtime Incus version check remains. Successful streamed installation now prints `BoxWeave vX.Y.Z installed successfully` with the manifest-verified installed version above the client setup commands.

**S:** source review of the new package solver call, ownership boundary and completion output. **T:** unit tests mock APT resolution order and refusal; CI validation pending on this exact commit. **U:** rerun the original disposable Ubuntu 24.04 real install, inspect APT simulation/selected packages and any subsequent failures, then verify Incus, Containarium, app, Web and authenticated MCP. No complete fresh-host success is claimed. Long-term design debt: Containarium should own its own dependency provisioning, but its current upstream installer can uninstall/reinstall Incus and modify system configuration. Do not delegate to it blindly or silently change external Incus/PostgreSQL.


## 0.19.0 Multi-App cross-platform client generator

The owner-approved alphabetic App list is Antigravity, Claude Code, Codex,
Copilot CLI, Cursor, OpenCode, Qoder IDE, TRAE IDE and ZCode. Native
client OpenSSH is supported on Windows, Linux and macOS; Windows-hosted
Apps may use explicitly selected WSL SSH. The previous Windows Codex+WSL
PowerShell generator remains compatible, and all other profiles emit
self-contained client-side Python 3.11+ source.

New client installer creates/pins a client-owned SSH identity, uses
administrator-authorized public-key enrollment, verifies an actual MCP
initialize/tools/list/list_containers read before scoped JSON/TOML updates,
backs up existing files, refuses conflicting entries without explicit
replacement and does not edit Codex AGENTS files. Qoder and TRAE use
official JSON manual import; no undocumented global path is guessed.
`AGENTS.md` enforces alphabetic App ordering in code/docs/UI. The new
helper is included in the release manifest and installed root-owned
helper inventory. Historic Web four-profile descriptors remain independent.

**S:** source/design and vendor MCP config reference inspection.
**T:** Python profile matrix/merge/refusal fixtures and exact commit CI
(verify workflow result). **U:** actual App-specific MCP injection on
Windows (native and WSL), Linux and macOS; SSH key ACLs and enrollment
approval; Copilot CLI vs VS Code discovery; OpenCode JSONC variants;
Qoder/TRAE import and client app restart. No live proof claimed.


## 0.19.1 Long-running MCP SSH keepalive restoration

The archived `archive/0.5.1` POSIX and Windows client installers used
`ServerAliveInterval=15` and `ServerAliveCountMax=4` in persistent
MCP SSH registrations and in probes. The 0.19.0 portable Python profile
generator already had these values. The separate Windows Codex+WSL
PowerShell template, Windows native/WSL descriptor helper, and Windows
native registration/diagnostic scripts were missing them.

**S:** restore the same options to every missing registered MCP SSH argv
and its matching verification probes; retain host-key pinning and
restricted-key enrollment. **T:** tests assert generated PowerShell
probe and persisted registration, evaluated PowerShell argv, WSL/native
profile argv and saved Codex TOML, all 36 portable App/OS/transport
combinations, and consistent native registration/diagnostic source.
Record exact commit CI completion before declaring pass. **U:** live
long-idle MCP behavior behind NAT/firewalls on Windows, Linux and macOS.
The client must restart/reconnect if SSH dies; no silent migration of
existing client registrations is performed by a server upgrade.


## 0.20.0 Unified Python client output and copyability

Owner requested that `boxweave client` stop making Windows Codex/WSL
a PowerShell-only exception. The installed wizard/noninteractive generator
now emits exactly the same self-contained Python 3.11+ source for all 36
app/OS/transport combinations. The shebang is **the first line**;
decorative `# =====` banners and usage notes appear afterwards.
Interactive prompts/status use `#`-prefixed stderr so the copied
script and redirected stdout are not polluted with un-commented wizard
text. The generator preserves `--no-replace` as the default, never
overwrites `--output`, retains strict SSH host-key pins and keepalive,
and verifies actual MCP reads before a scoped App registration. Updated
Codex TOML merge carries forward unrelated root options, comments and
nested tables (including per-tool approvals) rather than overwriting
them wholesale. The previous PowerShell generator and its regression
suite are no longer part of the installed CLI; manually maintained
legacy Windows scripts in clients/windows/ remain separate.

**S:** source review of generator/template, CLI stderr/stdout boundary
and the executable shebang. **T:** CI unit tests compile every generated
profile, inspect stdout/stderr and --output behavior, check replacement
policy, TOML preservation and keepalive; exact result must be checked.
**U:** real Windows native/WSL Python3.11+ setup, real macOS/Linux
clients, App tool injection and long-idle MCP connectivity. Python
availability and Windows OpenSSH ACL requirements are operator prerequisites.


## 0.20.1 Agent-channel client setup hang and nesting gate error clarity

A 2026-10-30 end-to-end test of the streamed installer, `boxweave client`
and the full MCP chain on fresh LXC boxes (server `boxweave-test-ser`,
client `boxweave-test-cli`, workload `hello-py`, shared network tenant)
succeeded overall but exposed two agent-facing defects, both fixed here.

`boxweave client` treated any TTY stdin as interactive. Agent/exec
channels expose a TTY stdin while capturing stdout, so generation
without explicit `--replace`/`--no-replace` blocked on the replace
prompt until the caller timed out, producing no output. Interactive
mode now additionally requires a TTY stdout and is disabled by a
non-empty `BOXWEAVE_BATCH` environment variable; the install summary,
host onboarding text and generator help state the automation rules.
The unattended enrollment recipe (`--no-enroll`, root
`online.py enroll --public-key` on the host, rerun with
`--already-enrolled`) is now printed by the installer instead of being
undiscoverable.

On a gate host without the BoxWeave server helper chain,
`create_container` with `enable_nesting` reported
`NESTING_CONFIGURATION_UNVERIFIED...; creation must not be repeated`
after the box was already created, and every `boxweave_nesting` or
nesting failure was masked as `TRUST_OR_WORKLOAD_KEY_GATE`. A missing
sudo/helper/timeout in `nesting.invoke` now raises
`NESTING_HELPER_UNAVAILABLE`, the gate surfaces the fixed nesting and
target-identity codes verbatim, and the post-create failure message is
`BOXWEAVE_NESTING_NOT_APPLIED; box creation itself succeeded and must
not be repeated; <code>`.

**S:** source review of `client_setup.py`, `nesting.py`, `mcp_gate.py`,
`online.py`, `host_cli.py`. **T:** unit tests cover captured-output
exec channels, `BOXWEAVE_BATCH`, helper-gap naming, created-box
survival, and the new summary text; run in WSL Python 3.10 plus
`scripts/check_knowledge.py` and `bash -n` locally. The pre-fix
behavior was reproduced on real boxes with release 0.20.0; the fixed
helpers require release 0.20.1 assets and have not been re-run on a
real host yet. **U:** gate-host repair (install the helper chain or
delegate nesting to the daemon), real Windows/macOS client profiles,
server reboot persistence, install.sh update/repair rerun. The
"PROVISIONING TAKES MINUTES / poll get_container" create text is
emitted by the upstream Containarium MCP server binary, not this
repository; clarification belongs upstream.


## 0.21.0 Legacy-only capability restoration (selftest, diagnose, resize, jobs, provenance)

The 2026-10-30 archive/0.5.1 gap review identified five capabilities with
no current counterpart; each is restored on the new architecture as an
independent slice. `boxweave selftest plan|run` produces a hash-confirmed
plan for a dedicated `boxweave-selftest-*` box and drives create,
identity+hello over the connect tool, stop and delete against the live
MCP provider only, with unconditional best-effort cleanup and a per-step
receipt that never records remote output. The generated client script
gains `--diagnose`: a read-only report of local tools, pinned-identity
state (no key material printed), live MCP stages and registration state.
`resize` joins the environment lifecycle as a reviewed plan/confirm
action over the existing resource vocabulary, verified against observed
provider inventory (unknown at deadline on persistent mismatch;
provider rejections such as disk shrink fail deterministically); the web
Environments panel and CLI `environment-plan --cpu/--memory/--disk`
expose it. `boxweave job [--budget N] -- COMMAND` runs one explicit
local command as the non-root application user with a durable receipt
(argv, state, exit code, duration, output hash/size only; store schema
user_version 8). `boxweave provenance` reads the installer-published
root-owned provenance.json (version, source, provider mode, endpoint,
phase history, managed-file count).

**U follow-ups (found in real-host acceptance):** the host wrappers for
`job` and `selftest` invoked `python -m bwmini` without the application
directory as the working directory, so the installed venv could not
import bwmini (0.21.1); the shared argparse guards rejected
selftest-run's --plan/--confirmation pair (0.21.2); and the CLI
--environment-action choices omitted resize (0.21.3). The long-running
user worker freezes lifecycle config at startup, so adding the
lifecycle block requires restarting the worker service before plans
dispatch.

**S:** source review of selftest.py, jobs.py, client_profiles.py,
lifecycle.py/provider.py, online.py/host_cli.py and Environments.tsx.
**T:** 60+ new and existing unit tests green on WSL Python 3.10 across
test_selftest, test_jobs, test_client_setup, test_lifecycle, test_access, test_bindings,
test_models, test_recovery and test_graphs (schema migration fixtures
updated for user_version 8), test_online, test_host_cli;
frontend tsc+vite build green; check_knowledge and bash -n clean. The
known WSL 3.10 gaps (tomllib import, interpreter-floor assertion in
test_platform_minimums, test_distribution suite) are pre-existing and
unrelated. **U:** real-host acceptance of all five features against a
fresh streamed v0.21.0 install (selftest live run, resize against a
real box, job via sudo, provenance after install, --diagnose from a
client box), web resize through a browser, and update/repair rerun of
install.sh from 0.20.x.

**T follow-up (real-host acceptance 2026-10-31):** all five features
passed against a fresh box (`boxweave-test-ser2`, shared tenant):
streamed install of v0.21.0, then streamed update reruns to v0.21.1,
v0.21.2 and v0.21.3 (the previously untested update/repair path, run
three times). `boxweave provenance` reports version, source commit,
endpoint and the merged phase history across installs. `sudo boxweave
selftest plan/run` passed end to end (create 42.7s, identity+hello
0.4s, stop, delete, cleanup not_needed). `sudo boxweave job` recorded
durable receipts for a succeeding and a failing command, surviving
process exit. The full lifecycle create of `demo-1` (identity_verified)
followed by `environment-plan --environment-action resize --memory 2GB`
succeeded online with verification=observed. `--diagnose` from a
separate client box printed 8/8 ok with registration=registered.
Acceptance surfaced and fixed three release-blocking defects (0.21.1
wrapper cwd, 0.21.2 argparse guards, 0.21.3 resize choice) and two
open findings: (1) corrected on review: `runtime.json` is a generated
artifact rebuilt from `config.json` on every install, and operator
keys belong in `config.json` (the byte-checked input). An update
refuses with OPERATOR_RUNTIME_KEYS_REQUIRE_CONFIG_JSON and names the
keys when a previous runtime.json would lose any key (0.21.4), instead
of silently dropping them; (2) the long-running user worker
freezes lifecycle configuration at startup, so config changes need a
worker restart. Client-side, a stale SSH MCP transport in the agent
chain wedged every MCP call until it was killed — transport
housekeeping, not a BoxWeave defect.

**T follow-up (0.21.4 live validation, 2026-10-31):** the runtime-key
guard shipped in v0.21.4 and the operator channel was exercised on real
boxes: on `boxweave-test-ser2` the `lifecycle` block was moved from
runtime.json into config.json (the supported channel), after a worker
restart `environments` reports enabled with the configured name. On
`boxweave-test-ser` the streamed update v0.20.0 -> v0.21.4 succeeded
across the major jump (doctor --mcp authenticated_read_passed,
provenance reporting 7ae7de7). The client migration flow was walked end
to end from `boxweave-test-cli`: `--diagnose` against the new server
correctly reported identity=mismatch and registration=differs before
migration; after the operator steps (out-of-band host-key review,
known_hosts pin update, root enroll of the unchanged client key,
regenerating the setup with --replace, running it with
--already-enrolled) diagnose reports 8/8 ok with
registration=registered against 10.100.0.3, and a real MCP connect
through the registered config executed `hello world` inside the
pre-existing `hello-py` box. One operator-facing note: `--replace` is a
generator flag baked into the emitted script; the generated script
itself only accepts --already-enrolled/--no-enroll/--diagnose/--distro.
## SRC-104 diagnostics design research — no implementation / no live test

Research-only documents: [source audit](DIAGNOSTICS_SOURCE_AUDIT.md) and [implementation proposal](../design/DIAGNOSTICS_PROPOSAL.md). Reviewed 0.21.4 source main@0fcfa6a3ea0f3d86cc929a5d7646bbc06c1b952b through the GitHub connector, plus published MCP protocol contract. **S/D only**, no package/code changes, fixture test, host/Provider connection, deployment, live Box change or CI execution. This source-only edit requires a new CI check before any future implementation confidence. Implementation is *not authorized* by the documentation commit. Before acting, owner must dispose Q-DIAG-001. After approval, validate DG-01..DG-14 in the proposal with actual exits and private-data safety review; preserve original unknown operations and nested targets.

## SRC-105 separate D0/D1/D2 coding specifications (documentation-only)

New design handoffs: [D0](../design/DIAGNOSTICS_D0_IMPLEMENTATION.md), [D1](../design/DIAGNOSTICS_D1_IMPLEMENTATION.md), [D2](../design/DIAGNOSTICS_D2_IMPLEMENTATION.md); cross-links inserted in [the parent proposal](../design/DIAGNOSTICS_PROPOSAL.md). Baseline source main@817273d98d8823fcd8489eb0d3f329a67929b91d. Evidence **S/D only**: GitHub repository read, source paths reviewed and proposed tests D0-T01..16, D1-T01..17, D2-T01..20 recorded. **U:** no local prepared checkout, pytests/unittest, knowledge checker, GitHub Actions validation, real install, authenticated MCP, process kill, report exporter, privacy penetration test or nested host validation was run. No production code, DB schema, MCP tool, version, host or deployed service was changed. Next Agent: obtain maintainer resolution of Q-DIAG-001; implement one slice at a time and log actual commands/exits/fixture vs live status with exact commit IDs; do not mark a proposed acceptance case passed.

## SRC-106 Box environment architecture (research-only / not implemented)

New [parent design](../design/BOX_ENVIRONMENT_PROPOSAL.md) and five independent proposed implementation batches: [E0 capability](../design/BOX_ENV_E0_CAPABILITIES.md), [E1 fingerprints](../design/BOX_ENV_E1_FINGERPRINTS.md), [E2 snapshots](../design/BOX_ENV_E2_SNAPSHOTS.md), [E3 restore](../design/BOX_ENV_E3_RESTORE.md), [E4 clone](../design/BOX_ENV_E4_CLONE.md), reciprocally linked from [D0-D2 diagnostics proposal](../design/DIAGNOSTICS_PROPOSAL.md) and [diagnostics source audit](DIAGNOSTICS_SOURCE_AUDIT.md). Baseline main@d72035837da1170d2fe5201ee4ea65075ea038ee. Evidence **S/D/public upstream docs**; **U**: no code, CLI, MCP catalog, Incus storage driver, nested guest, test deployment, trusted checkout/unit test, knowledge checker or live mutation was executed. The user requested **documentation and continued discussion**, not code implementation. Critical release blockers before E2-E4: actual Containarium authenticated snapshot/clone capability, managed source/destination identity, external/custom volume coverage, source serialization with existing Lifecycle, clone guest identity rekey and registry adoption. Never assert E0-E4 shipped or passed. Before code work resolve [Q-ENV-001](../knowledge/OPEN_QUESTIONS.md), and record exact tested source SHA/commands/exit/fixtures versus live acceptance in a future appended ledger entry.



## SRC-107 Agent-led rapid development working rules (documentation only)

Updated [AGENTS.md](../../AGENTS.md) and wrote [Agent test workflow](../process/AGENT_TEST_WORKFLOW.md), linked from [DEVELOPMENT](../DEVELOPMENT.md) and [INDEX](../INDEX.md). Defines two permitted styles (auto **within explicit pre-authorized scope**, or plan-first), reused and genuinely clean Box lanes, smallest relevant check, distinct installed/running/outer/inner identity, live hello-world evidence, uncertainty/no auto-retry and compact handoff record. Baseline main@16ef5ac7487a84c4e86163bb4852016088ec5299. Evidence **S/D** source/docs only; **U**: not run `scripts/check_knowledge.py`, unit/browser/CI, real installed CLI/MCP, nested Provider workflow or any host/Box mutations. No new runtime tool, database, snapshot, clone, diagnostics implementation or live authorization. Next Agent: apply workflow only to separately authorized work and append actual source SHA, exact commands/exits, environment, original resource disposition, S/D/T/U distinction and untested acceptance here.

## D0 diagnostics live Box checkpoint (owner-approved reused-Box upgrade lane, 2026-10-31)

**Goal:** D0 done-gate — the installed wrapper distinguishes a stopped app worker from a live MCP read. Mode: plan-first, owner approved the ser upgrade lane and the one-time manual host-file refresh after the GitHub-release-bound host publication path blocked a dev build. Source: main@5ae0febd050032325cad21a471d3f1a191e37dfd (0.21.5, `source_state=clean`), archive sha256 `bbb47e7922af8c6e568f9f43433bbb63f1f23d9a4f7c2bbfa1bb62b6c810cc59`. Lane: **reused** `boxweave-test-ser` (0.21.4, real data), verified `hello-py` workload untouched.

**T executed (all commands on the named Box, exits recorded):**
1. Build: WSL Ubuntu 22.04, CPython 3.12.16 venv, `pip download -r requirements.lock` + `python -m bwmini.distribution build` → exit 0, manifest source 5ae0feb/clean. Transfer via LAN HTTP with sha256 verification on the Box (all three digests matched).
2. Upgrade: `install.sh plan` exit 0 (read-only preflight), then `install.sh apply --activate` as app user → exit 0, `state=ready`, DB backup `backups/1793433705056714114.sqlite3` created, install-time MCP probe `authenticated_read_passed` (58 tools). Post-upgrade `distribution health` (app user, runuser): `diagnostic_schema=1`, `unit_states={serve:ready,worker:ready}`, `state=ready`, `user_version` still 8.
3. Definite-failure path (app layer): `systemctl --user stop` worker → health `unit_states.worker=inactive` typed classification, `state=starting` (legacy semantic preserved); restart → `ready` again.
4. Non-root `boxweave doctor`: exit **2**, `overall=incomplete`, public/systemd checks all `passed` (real unit names, no drift), app checks `not_verified/APP_HEALTH_REQUIRES_ADMIN`, coverage `not_verified_requires_admin`.
5. Root `sudo boxweave doctor`: exit **0**, all required checks `passed`. `sudo boxweave doctor --mcp`: exit **0**, `authenticated_mcp=MCP_READ_PASSED` + `admin_initialized=ADMIN_INITIALIZED` (single app-user transaction, 58-tool read).
6. D0 done-gate: worker stopped + `sudo boxweave doctor --mcp` → exit **1**, `overall=failed`, `app_worker_unit=failed/APP_UNIT_INACTIVE` while `authenticated_mcp` remains `passed` — the MCP pass does **not** mask the worker failure; recovery → exit 0.
7. Client live diagnose (D0-b): script rendered from 5ae0feb (zcode/linux/native, ser public descriptor data) run on `boxweave-test-cli` (real enrolled identity + registration) → exit **0**, all ten checks as designed: identity/ssh_transport/mcp_initialize/mcp_tools/mcp_read/registration all `passed` against the live server, `agent_ui`/`model_inference` explicitly `not_verified`; stderr carries only `#`-prefixed fixed codes, no raw SSH output.
8. One-time manual host refresh (owner-authorized): `host_cli.py` + `client_profiles.py` published to `/usr/local/lib/boxweave/default` replicating online.py semantics (FOREIGN guard, atomic 0o755, managed_files digest registry, atomic state.json write-back, `client-public.json` version aligned to 0.21.5); self-check verified all 19 managed digests. Managed keys, enrollment, sessions and operator config untouched.

**PARTIAL / U — remains unverified:** host `provenance.json` on ser still reports the 0.21.4 install (7ae7de7) until the next full streamed install — an observed instance of the documented installed-vs-running identity gap (F-07), acceptable and disclosed. Fresh-install lane, ser2 cross-version upgrade, nested workload behavior, report/selftest (D2), D1 v9 migration and browser lanes are not covered by this checkpoint. Fixture suite at 5ae0feb: 313 tests OK; `check_knowledge.py` passed. **Outcome: PASS for the D0 checkpoint scope as listed; next:** D1 implementation with its own fixture-then-live checkpoint at 0.22.0.

## D1/D2 0.22.0 live Box checkpoint (owner-approved, boxweave-test-ser, 2026-10-31)

**Goal:** the 0.22.0 diagnostics feature (D1 schema v9/v10 evidence, D2 report export, D2-b selftest journal) proves itself on the installed host. Mode: plan-first; owner approved the full scope (host-file refresh, live mutating selftest, Web-created operation attempt). Source: main@35fd5f1d9088ee224666873f66114b60bde47a01 (0.22.0, clean), archive sha256 `2c8c2d7a0edd063b4c9887642fd9882b9203cde5ac877e094673a71b1057a4f9`. Lane: **reused** boxweave-test-ser (0.21.5 from the D0 checkpoint).

**T executed:**
1. Upgrade: `install.sh apply` exit 0 → `state=ready`, DB backup `backups/1793437024151416432.sqlite3`, install-time MCP probe `authenticated_read_passed` (58 tools). Live schema migration observed: `user_version` 8 → **10**, `diagnostic_trace_links`/`diagnostic_events`/`selftest_runs` created, business rows unchanged.
2. Host refresh (owner-authorized one-time, repeated for 0.22.0): host_cli.py (report command) + client_profiles.py published with managed-digest registry update and descriptor aligned to 0.22.0; 19/19 managed files self-verified. A republish script defect initially left the registry stale; repaired in-session and re-verified (all digests match).
3. Doctor battery on 0.22.0: root `doctor` exit 0; non-root `doctor` exit 2 — D0 contracts hold.
4. **Live selftest (D2-b, owner-approved mutating test):** `selftest plan --budget 300` → confirmed run of box `boxweave-selftest-5392a9` on the real Provider: create 29.0s → hello 0.45s → stop 1.2s → delete 1.3s, `state=passed`, `cleanup=not_needed`; durable journal row `st-268a2eb8e1600dae … completed, delete:OK, not_needed`. No orphan box, no unreconciled rows.
5. **Live report (D2-a):** `sudo boxweave report --since --until --output` → exit 0, `app_reader=observed`, `collection_status=complete`, 0700 directory with seven 0600 files, honest omissions (gate/provider capture not installed, parent not verified). `report --kind hello --operation nonexistent-op` → exit 1 `REPORT_SOURCE_NOT_FOUND`, no directory created (fail-closed live).
6. **Live defect found and fixed (9162758):** the first export run reported `app_reader=unavailable`; root cause was a reader-CLI invocation mismatch (`read-report` token that the CLI never defined). Fixed, tests updated, republished, re-run: reader observed/complete.

**PARTIAL / U:** ser has no configured provider targets and lifecycle disabled, so no real hello/environment operation (and therefore a populated report timeline) could be created without operator config changes outside the approved scope — the populated-timeline export is proven at fixture level only; the live export pipeline, fail-closed path and zero-event partial are proven on the real host. Client-side live diagnose re-validation on 0.22.0 not rerun (client generator unchanged since the D0 checkpoint). Fixture suite at 9162758: 353 tests OK; `check_knowledge.py` passed. **Outcome: PASS for the 0.22.0 checkpoint scope as listed.** Next: none blocking; future slices inherit the installed-surface checkpoint rule.

## v0.23.0 + v0.24.0 live Box checkpoint (owner-approved reused-Box + nested-Box lane, boxweave-test-ser, 2026-11-01)

**Goal:** the 0.23.0 capability inventory and the 0.24.0 M0–M5 + Remote MCP R0–R4 features prove themselves on the installed host and against a real nested Box, fixing every release blocker found. Mode: agent self-executed within the owner's request to test both candidates on Boxes and publish v0.24.0. Source lane: main f98c192 → 2de4920 → 58ab5bc (all pushed; CI triggered per push). Lane: **reused** boxweave-test-ser (0.22.0 from the D1/D2 checkpoint) plus its real nested Box `hello-py` (Containarium 0.120.1 inner platform).

**Release blockers found and fixed (T + live I):**
1. **f98c192 — offline archives missed seven package modules** (`box_debug`, `capabilities`, `client_profiles`, `host_cli`, `remote_auth`, `remote_mcp`, `remote_tls`). A v0.24.0 candidate archive failed its install-time deploy_check with `ModuleNotFoundError: bwmini.remote_auth` (Store schema v12 imports it), leaving the reused Box's services stopped until repair — a fresh streamed install would have failed the same way. Fix: package every `bwmini/*.py`; regression test added.
2. **2de4920 — remote relay spawned the gateway with `python -I`**, whose implied `-P` strips the script directory from `sys.path`, so the gateway's `nesting` import failed and every authenticated remote tools call returned 503 `upstream_unavailable`. Proven live (exit 1 with ImportError under `-I`; works under `-E -s`). Fix: spawn with `-E -s`; relay regression test added that runs a fixture gateway importing a sibling module.

**T executed (all commands on the named Boxes, exits recorded):**
1. Fixture baseline at 3b237bd: WSL Python 3.12.16 `scripts/validate.py` → knowledge + 412 unit tests + web build, `state=passed`. Re-run `passed` after each fix (f98c192: 413 tests; 2de4920: 414; 58ab5bc: 415 with the caddyfile unit tests).
2. Upgrade: `install.sh plan` exit 0 (read-only), `install.sh apply` exit 0 → `state=ready`, DB backup created, install-time MCP probe `authenticated_read_passed` (58 tools). Live schema migration observed `user_version` 10 → **12**; v11 `box_action_plans`/`box_action_receipts` and v12 `remote_*` tables present, business rows unchanged.
3. Host refresh (established twice-approved procedure on this Box): published 0.24.0 `mcp_gate.py` (the only changed host helper; all six others byte-identical), aligned `client-public.json` to 0.24.0, updated `/var/lib/boxweave-default/state.json` (version/source/digests); **19/19 managed digests self-verified**.
4. Doctor battery on 0.24.0: root `doctor` exit 0 `overall=passed` version 0.24.0; non-root exit 2 `incomplete` with `APP_HEALTH_REQUIRES_ADMIN` (D0 contracts hold); `doctor --mcp` exit 0. First run before the host refresh showed `APP_RELEASE_DESCRIPTOR_MISMATCH` — the documented descriptor-vs-release gap, resolved by the refresh.
5. **v0.23.0 gate:** CLI `bwmini ... capabilities` exit 0 — live discovery through the app MCP client, `source=platform`, 58 tools, honest `authorization: not_established_by_discovery`. Web `GET /api/mcp-capabilities` 200 with the same catalogue; `GET /api/boxes/hello-py/debug-summary` 200 — redacted typed evidence (`sensitive_details: withheld`, `evidence: live_upstream_response`). Negatives: no cookie → 401 `AUTH_REQUIRED`; unknown logical → 403 `TARGET_NOT_ALLOWED`; stale target identity → 409 `TARGET_CHANGED`.
6. **M1 allowlist (live gateway process):** no policy → legacy 59 tools and unlisted calls pass; policy `{connect,get_container,list_containers}` → `tools/list` exactly 3, unlisted call → `-32601 GATE_TOOL_NOT_ALLOWED` never forwarded, listed call reaches upstream; unsafe policies (mode 0644, bad JSON, version 2, extra keys) → gate exits 1 with no output (fail closed, no legacy fallback); policy removed → legacy restored.
7. **M3 agent-box (real nested Box):** operator installed the reviewed Containarium `agent-box` binary (built from vanishs/Containarium feef1d00, sha256 0f0fa7f8…) into `hello-py` and configured the pinned-SSH launcher (dedicated key, strict host-key pin, keepalive argv). Web `GET /api/boxes/hello-py/agent-box` 200 live capabilities with per-tool `authorized` flags; `POST .../agent-box/read` `list_directory` and `read_file` returned real box content end to end (probe file read back byte-identical); `write_file` → 403 `AGENT_BOX_READ_ONLY`; `../../etc/passwd` → 403 `BOX_PATH_DENIED`; resource `containarium://ci-context` 200 with honest `available:false`.
8. **M2 auto-sleep:** plan 201 (120 s expiry, one-use) → wrong-logical confirm 409 `TARGET_CHANGED` → confirm 202 `queued` → worker receipt `acknowledged`/`AUTO_SLEEP_ACCEPTED_NOT_OBSERVED`. Independent out-of-band proof: inner Incus config shows `user.containarium.auto_sleep_enabled: "true"`, `idle_threshold_minutes: "15"`. Invalid args → 422 `AUTO_SLEEP_SPEC_INVALID`; replay confirm → 409 `PLAN_STALE`.
9. **M4 receipts/no-replay:** worker stopped → plan+confirm (restore toggle) queued → plan expired while worker down → worker restart marked the receipt `failed`/`PLAN_EXPIRED_BEFORE_DISPATCH` and the platform state remained unchanged (no execution, no replay). Mid-flight crash → `unknown` recovery stays fixture-proven only (2db1fcf/d660528); live timing not staged.
10. **Remote MCP R1/R2 (loopback 127.0.0.1:18844 only):** unauthenticated → 401; evil Origin → 403 `invalid_origin`; spoofed Host → 403 `invalid_host`. OAuth: resource + AS metadata 200 (issuer/audience = canonical origin, S256 only); DCR 201; authorize form 200; no-approve → 403 `access_denied`; evil redirect_uri → 400; approve → 302 with code + state; wrong verifier → 400 `invalid_grant`; replayed pending/code/refresh → 400/403; refresh grant 200. Authorized with protocol 2025-11-25: initialize 200 `boxweave/0.24.0`, `tools/list` **59** (R0 parity with the SSH gateway's 59), `tools/call list_containers` returns the real nested Box; GET/DELETE `/mcp` 405; bad Accept 406; wrong protocol header 400; notification 202; malformed body 400; wrong Bearer 401 with `WWW-Authenticate: Bearer resource_metadata=…`.
11. **R2 credentials:** Web-issued API Bearer (reveal-once) authorizes `tools/list` 59; opaque `/mcp/link/<secret>` authorizes 59; guessed path 401; Web list exposes no raw secrets; **database stores digests only** (`remote_credentials.digest`; raw token and raw link path absent from the DB row scan); revocation (Web DELETE) → both 401 immediately. Revocation state and panel data persist across a Web restart; an unrevoked OAuth access token remaining valid across restart is protocol-correct (tokens retain protocol-required expiry).
12. **R3:** `remote-caddyfile` on a host without Caddy → typed `CADDY_NOT_INSTALLED` exit 2 (fail closed). Independent local TLS proxy (self-signed cert, public Host preserved, no public exposure): full metadata → DCR → PKCE approve → token → initialize → `tools/list` 59 through the proxy; spoofed Host through the proxy still 403. Caddyfile shape now unit-covered (58ab5bc).
13. **R4/SSH parity:** after the host `mcp_gate.py` refresh and both upgrade repairs, the enrolled zcode client on `boxweave-test-cli` ran its live diagnose against ser → **exit 0, 8/8 ok** (`identity/ssh_transport/mcp_initialize/mcp_tools/mcp_read/registration`), proving the legacy SSH MCP chain unchanged.

**PARTIAL / U — remains unverified:** real public HTTPS (Caddy/TLS/DNS/firewall) and real external clients (ChatGPT/Claude/Cursor accounts) remain U per plan; the R3 evidence is an independent loopback TLS proxy, not a public origin. Web React panel UI on the live Box was exercised at API level only (live browser unreachable from the operator workstation); the seven fixture browser workflows cover the UI locally. Live `unknown`-recovery after a mid-dispatch worker crash stays fixture-proven. Platform-side `resources/list` is unsupported by th

## Published stable release pin correction (2026-11-01)

- **Change:** Source `VERSION` remains `0.24.1` (unreleased). Point `PUBLISHED_RELEASE` and the standalone streamed `install.sh` default at published stable `0.24.0`; preserve the existing manifest SHA-256, tag/commit and helper SHA-256 verification and fail-closed behavior.
- **S — repository/API inspection:** GitHub reported `v0.24.0` as latest non-draft, non-prerelease release, with uploaded `release.json` and `online.py` assets and recorded SHA-256 digests, plus a direct `refs/tags/v0.24.0` commit ref. GitHub returned 404 for `v0.24.1` release. This inspection does not prove downloaded bytes or real-host installation.
- **U — next checks:** Run `bash -n install.sh`, `python scripts/check_knowledge.py`, and `python -m unittest tests.test_platform_minimums -v` in a trusted checkout; compare `PUBLISHED_RELEASE` with the version literal in `install.sh`. GitHub Actions fixture validation can independently check source regression. In an **explicitly authorized** prepared Linux Box, run the minimal reused-Box installed-surface checkpoint per `AGENTS.md`: perform the reviewed `0.24.0` streamed installation/upgrade and read-only runtime checks, plus a definite-failure path. Record exact target, version, command exits and evidence. No live Box install/test or public network trust change was performed by this repository-only edit.

## v0.24.1 published package and default installer pin (2026-11-01)

- **Observed host failure:** During the user's v0.24.0 streamed installation, bootstrap reported `APPLICATION_INSTALL_INCOMPLETE_INSPECT_PRIVATE_LOG`, the private application log gave `CANDIDATE_CONFIG_OR_STATE_FAILED`, and a replica-only diagnostic raised `ModuleNotFoundError`. Exact missing module name in that host traceback was not captured. The v0.24.0 package module omission is a code-supported explanation, not a proved live repair.
- **S/T — release artifact check:** The published stable `v0.24.1` tag refers to `8eb6ab57863a5b1b5fc454051c244b549a95bf9b`. GitHub Actions run `41284747762` completed verification and publication successfully. The retained CI candidate `boxweave.zip` SHA-256 `1e1902e433293989db0f36e84b7ccfffcd40060f2e15c63bfbb22211c5289344` matches the GitHub Release asset digest. The ZIP contains 36 `bwmini/*.py` modules, including `remote_auth.py` and `remote_mcp.py`; inspected individual ZIP member digests match its manifest.
- **Change:** Promote the default streamed `install.sh` and `PUBLISHED_RELEASE` pins from `0.24.0` to verified published `0.24.1`. Add CI coverage of actual ZIP module membership and SHA-256 values. Permit CI to preserve an existing immutable release only when an installer diff is *exactly* a stable-pin retargeting to that same release (all other runnable source changes still require a version bump).
- **U — next acceptance:** Confirm PR and main CI verification/publish status. The user's existing partially installed host retains its original state and requires operator inspection before rerunning the reviewed installer. No live installation, host/service/provider change or cleanup was performed in this repository-only edit.

## Remote MCP v0.24.3 release readiness (2026-11-01)

Scope: Fix managed-Caddy's unconditional Referrer-Policy rewrite that caused browsers to submit the OAuth consent form with Origin:null; preserve the application-provided one-page same-origin setting while retaining the global POST Origin denial. Add a local-only bounded TCP reachability field to authenticated Web status and UI, explicitly not an external endpoint readiness claim. Update master R0-R4 statuses with previously observed real ChatGPT read and explicit remaining deviations. Tests and release: run GitHub Actions validation on exact v0.24.3 PR SHA, inspect Python 3.12 and 3.14, seven browser fixtures, JS build, offline ZIP/module/wheel verification and publishing result; record final URLs/SHA only after observed completion. No real host/Box write or deployment under this repository-only work. Separately test managed Caddy public TLS, 0.24.3 runtime callback and systemd persistence before closing R3/R4 host gates.

## D2 diagnostic actor scope hardening (2026-11-01; PR #17)

**Goal / change:** Prevent one actor's diagnostic trace, window or mixed-trace event from entering an operator-scoped D2 report. App-owned reader now requires an actor selector for all read shapes and filters both trace links and events; the installed host CLI supplies the fixed operator scope, without adding MCP authority, DB migrations or Provider operations. Updated the stale D0-D2 status text in AGENTS and its parent proposal. Source reviewed: `main@00d59f624ee0be73836c23df3212d47e1d5218f6`; branch patch `5a2bfaa75fb2cdedb6cb65063cadea98e243a6a5`.

**S — reviewed source:** `bwmini/diagnostics.py`, `bwmini/host_cli.py`, `tests/test_diagnostic_report.py`; new negative fixture covers a foreign actor's trace, forged cross-actor event under a valid operator trace, window leakage and missing actor scope.

**T — GitHub CI fixture, exact patch SHA:** Actions run `41330576611` completed with conclusion `success`. Run covers `python scripts/check_knowledge.py`, Python unit and contract tests, frontend build, browser fixtures, offline package creation, Python 3.14 smoke and archive/module checks. This is CI/fixture proof only, **not** an installed-surface checkpoint.

**U / blocker:** A read-only check of reused `boxweave-test-cli-container` confirmed it running but `connect(exec)` refused with `TRUST_OR_WORKLOAD_KEY_GATE`. `debug_container` classified the host account's configured `/usr/local/bin/containarium-shell` as absent on backend `dev`. No host shell/service/SSH repair, Provider mutation, nested test, or live report export was performed. The 0.22.0 D0-D2 historical live checkpoint applies only to that earlier release, not this patch. Next: independently approve/fix the backend shell deployment, then run the reused-Box installed-surface checkpoint (upgrade, read-only positive and cross-actor denial, report export); record exact exits and installed/running versions. D3 authenticated caller-to-trace mapping is still unimplemented; D4 remains out of scope. Do not claim their acceptance from this patch.

## v0.24.5 source release and stable pin promotion (2026-11-01)

**T — CI and GitHub publishing:** The versioned D2 actor-scope patch was merged through PR #17 to `main@f853c22b5f313c2ded868f0ae5c96efce424e987`; source verification on Actions run `41331107046` completed `success` for both `verify` and `publish`. GitHub reported the immutable release `v0.24.5`, published 2026-11-01T17:24:09Z, pointing at that exact commit; its `boxweave.zip` asset has reported SHA-256 `b1a314cacc8a3378b90cb61da27c72670e892ef64284f3696b0731a4ebca39e5`, and `release.json` and installer helper assets are present. These are GitHub/CI release observations, **not** installed runtime or byte-level independent asset download proof.

**S — default installer promotion:** Advance only the published stable selector `PUBLISHED_RELEASE` and the reviewed `install.sh` embedded default from `0.24.4` to `0.24.5`. Update source/published status documentation. Published-version preservation is intentionally immutable; the CI publisher allows this exact pin-only change without a new runtime version. No host, Box, private state or Provider operation was changed.

**U — live acceptance remains required:** The reused test-Box SSH command path was blocked by absent `containarium-shell` on backend `dev`; no 0.24.5 installed-host checkpoint occurred. D3 MCP trace authorization and D4 user-project debugging remain unimplemented/out of scope. The default installer selecting v0.24.5 does not prove a fresh install or authorize auto-upgrade.


## D3/D4 v0.25.0 source and pending live acceptance (2026-11-01)

- **S**: owner-approved [master](../design/DIAGNOSTICS_D3_D4_MASTER.md) and [implementation](../design/DIAGNOSTICS_D3_D4_IMPLEMENTATION.md). Branch PR #18. New gateway helper is root-installed via reviewed release manifest, runs as unprivileged gateway account, journals classified small events without raw tool data. D4 deep reads default DENY until administrator grants an exact identity and checks.
- **T (GitHub CI, exact SHA `60183f4341c8a3002b4a8df918898df7d4c795d9`)**: Actions run [41339083888](https://github.com/vanishs/boxweave/actions/runs/41339083888) concluded `success` for fixture Python contract/unit suite, docs, frontend/browser, archive/module digest and Python 3.14 wheel smoke. Earlier failing PR commits exposed a D4 identifier bug, stale catalog test, omitted release helper and synthetic transport fixture errors; all fixed before this successful exact source verification. This is **T fixture/CI, not live MCP**.
- **U (must execute before installed-surface acceptance)**: independently authorized reused Box upgrade using published reviewed artifact; verify real SSH and authenticated HTTP MCP initialize/list/diagnose; actual app report actor boundary; one original failed/unknown gateway write with no duplicate; journal restart/retention and capacity; D4 denial for unconfigured Box and success only for a specifically administrator-allowlisted real test Box, with guest output redacted. Record installed version, running process, exact test Box identity, negative and positive exits. The previous reused test Box's workload SSH trust was blocked by missing `containarium-shell`; do not infer current ability or mutate it to solve fixture acceptance. No host/Provider/Box operations performed here.
- **U/known limitation**: Gateway `_meta` was deliberately not added to ordinary upstream results because end-client compatibility is unverified; Agent uses bounded recent lookup. Automatic Agent follow-up is guidance, not mandatory client behavior. Underlying `connect(exec)` operator authority is unaffected by D4's new default-deny endpoint.


## v0.25.0 published diagnostic release and stable installer promotion (2026-11-01)

**T/CI:** PR #18 source merge `3491940921eaa4b2d3422e7bd3f4804b24f8290a` completed main Actions [41339810139](https://github.com/vanishs/boxweave/actions/runs/41339810139) with `verify=success` and `publish=success`. GitHub immutable [v0.25.0](https://github.com/vanishs/boxweave/releases/tag/v0.25.0) is bound to source SHA `3491940921eaa4b2d3422e7bd3f4804b24f8290a`, published 2026-11-01T19:06:07Z. Published `boxweave.zip` digest is `sha256:2426e91739751d165b62e09fcfb8046a90408a1687a7207ab1af2fd8f21da598`; the separate installed gateway helper `gate_diagnostics.py` is present as a verified release asset. Pin-only promotion selects v0.25.0 in `PUBLISHED_RELEASE` and in the embedded default `install.sh` selector. These are **GitHub-reported publication/CI observations**, not independent live installation or real-box proof.

**U:** D3.1 app query, D3.2 real gateway SSH/HTTP failure persistence, and D4 positive exact allowed Box/negative unallowed Box require separately authorized installed-host validation. User-visible Agent auto follow-up depends on client; no live authorization, client change or real Provider/Box/host mutation was performed by this release process.

## Code-first acceptance audit (2026-11-01)

**T:** source baseline `main@ca4898c26496a2618ac28bf0318175791307b564` completed [Actions run 41340089030](https://github.com/vanishs/boxweave/actions/runs/41340089030) with verify/publish success (Python unittest discovery, seven browser fixtures, client grammar, frontend, archive/module digests, Python 3.14 offline wheels). **L positive, bounded:** `boxweave_diagnose(source=gateway,recent=5)` returned five classified events (including an unknown connect result) with gateway-only coverage. **L negative, expected:** `boxweave_project_diagnose(box=boxweave-test-cli-container,check=ports)` rejected `PROJECT_DIAG_NOT_AUTHORIZED`. **BLOCKED:** direct `connect(exec)` rejected `TRUST_OR_WORKLOAD_KEY_GATE`; `debug_container` on two test Boxes found the missing `/usr/local/bin/containarium-shell` on backend `dev`, Containarium v0.120.1. No host repair, Box write, live Python stdout/exit or installed 0.25.0 app upgrade was done. The original resources remain untouched. **U:** D3 installed app/restart, D4 exact allowlisted positive, 0.25.0 host/SSH/HTTP/public client, nesting, true clean-host/reboot and external TLS.

See [code-first feature/endpoint/test register](CODE_FIRST_ACCEPTANCE_2026-11-01.md). Fixing the backend user's SSH shell requires explicit host-operator approval; reattempt exact original workload only after the trust boundary is verified, never duplicate create or bypass host pinning. Source and fixture PASS cannot close live U/BLOCKED rows.

## Authorized live MCP Box acceptance round (2026-11-01, scope SRC owner in conversation)

**Scope and identity:** The owner explicitly authorized arbitrary BoxWeave MCP actions, including new/nested Box creation and changes to test Boxes. The tested outer backend is `dev` (local/healthy; Containarium `0.120.1`, Incus `7.5.1`); all direct live actions targeted new dedicated test Box `bwaccept-20261101-container`, username `bwaccept-20261101`, `networkTenant=boxweave-test`, `Ubuntu 24.04`, creation identity `1793567247`. The outer gateway self-description advertises source version `0.25.0`, which alone **does not establish the installed inner application version**. Parent production data and older test Boxes were not modified.

**Actually performed through BoxWeave MCP (real Provider, result API rather than guest exit):**

| Test / exact MCP arguments | Observed result | Acceptance by scope |
| --- | --- | --- |
| `list_backends({})`, `list_containers({})`, `debug_container({username:"boxweave-test-cli"})` | One healthy local backend, existing test Boxes enumerated, missing backend host-user login shell `/usr/local/bin/containarium-shell` independently identified | PASS inventory/diagnosis only |
| `create_container({username:"bwaccept-20261101",cpu:"2",memory:"2GB",disk:"20GB",backend_id:"dev",network_tenant:"boxweave-test",enable_podman:false,disable_auto_sleep:true})` | Success response; `get_container` confirmed named `RUNNING`, `10.100.0.6`, requested CPU/RAM/disk, immutable creation identity | **PASS** direct new Box create and readback, no guest readiness claim |
| `get_metrics({username:"bwaccept-20261101"})` | CPU, memory, disk, network and process counters returned | **PASS** metrics read availability; accuracy not independently calibrated |
| `resize_container({username:"bwaccept-20261101",cpu:"3",memory:"3GB"})` | Success response; `get_container` readback `3` CPU / `3GB`, retained identity | **PASS** direct LXC resize/readback (not persistence after reboot) |
| `toggle_auto_sleep({username:"bwaccept-20261101",enabled:true,idleThresholdMinutes:120})`; then `enabled:false` | Both calls acknowledged configuration; final disabled, 120-minute metadata retained | **PARTIAL** API ack; independent idle behavior and metadata readback unverified |
| `stop_container({username:"bwaccept-20261101",force:false})` | Success; independent `get_container` returned `CONTAINER_STATE_STOPPED` | **PASS** direct stop/readback |
| `start_container({username:"bwaccept-20261101",waitForReady:false})` | Success; independent `get_container` returned `CONTAINER_STATE_RUNNING`, same creation identity | **PASS** direct start/readback; guest SSH/TCP readiness not established |
| `boxweave_project_diagnose({box:"bwaccept-20261101-container",check:"ports"})` | `PROJECT_DIAG_NOT_AUTHORIZED` | **PASS** D4 default-deny negative only; positive authorized read still U |
| `boxweave_nesting({action:"inspect",logical:"bwaccept-20261101"})` | `NESTING_HELPER_FAILED_INSPECT_ORIGINAL_INSTANCE` | **FAIL/BLOCKED** real helper read path; no nesting state proved; no plan/apply attempted |
| `connect({box:"bwaccept-20261101-container",exec:"id -un && hostname && python3 -c 'print(\\"hello world\\")'"})` | `TRUST_OR_WORKLOAD_KEY_GATE`; no guest execution/exit | **BLOCKED** SSH/real Python; never count Running as command PASS |
| `debug_container({username:"bwaccept-20261101"})` | Backend host account exists; login shell `/usr/local/bin/containarium-shell` **does not exist**, as also observed for old test-cli Box | **PASS** root-cause classification in Provider debug; backend host repair not performed |
| `boxweave_diagnose({source:"gateway",recent:8})` | Bounded journal included actual succeeded create/get and unknown connect/nesting events with trace IDs, no raw inputs | **PASS** D3 recent read for observed event types; retention, restart, app-side query still U |
| `network_tenant_capabilities({})`, `list_network_policies({})` | Authenticated native-network-tenant/v1 reports `sharedTenantCreate=true`, `atomicTenantAccess=true`, `networkPolicyWrite=true`; policies `{}` | **PASS** capability discovery/read, **U** network packet enforcement and multi-policy write |

**No code repair claimed:** The observed guest SSH failure is caused or at least independently blocked by a missing Containarium backend host-user shell outside the current BoxWeave MCP host-edit capabilities. Altering this host service requires its own authorized host administration channel. The nesting helper path is a distinct explicit failure, not permission to silently change host root/sudo configuration or create substitute Boxes. API response failures have no guest process exit code. **Retention:** leave `bwaccept-20261101-container` RUNNING, CPU3/RAM3GB/disk20GB, auto-sleep disabled for a post-repair SSH/Python and D4 positive test. Do not delete/recreate the retained failure-scene Box automatically. Recheck its original creation identity before the next mutation.

**Acceptance tracking:** Direct Provider create/resize/stop/start/list/get are now **live PASS in the exact scoped v0.120.1 backend**. BoxWeave **application-side** resource plans, installs, SSH/agent-box and D3/D4 comprehensive behavior are not upgraded by these direct MCP tests. Cross-version comparison, nested runtime execution and complete installed v0.25.0 acceptance remain **U/BLOCKED**. See [code-first register](CODE_FIRST_ACCEPTANCE_2026-11-01.md).

## Host prerequisite triage and non-blocking continuation (2026-11-01)

**Decision:** Continue all independent test/repair lanes; mark only the dependent live gates BLOCKED. The owner authorized arbitrary BoxWeave MCP Box actions and comparative test Boxes. Two Boxes in the same native Tenant can exercise Box-to-Box traffic **once a trusted guest execution or other authorized in-guest test mechanism exists**; same-Tenant placement alone cannot execute Python in either Box or bypass the first-hop gateway trust and absent backend jump-host login shell. Python does not need installing merely to resolve SSH: the Ubuntu image and actual guest executable must be checked once access works.

- **BLOCKED [host, issue #19](https://github.com/vanishs/boxweave/issues/19):** `debug_container` independently reports the host-account login shell `/usr/local/bin/containarium-shell` missing on local backend `dev` (Containarium 0.120.1), for both old and new test Box owners. `connect(exec)` also returns the generic `TRUST_OR_WORKLOAD_KEY_GATE` before guest execution, so the gateway's target-registry freshness/host-key layer is a **separate unresolved check**. Fixing the login shell is necessary for the ordinary jump-host path but is not yet sufficient evidence of SSH/Python success. The connected BoxWeave MCP has no privileged host shell/file/service repair action. Upstream `scripts/setup-ssh-container-proxy.sh` creates the missing component **and** changes broad sudo, sshd and MOTD configuration; review each operation before an authorized host operator repairs it. Do not blindly run this script or promote an unrelated backend binary. Current update discovery says the sentinel/latest update version is unknown.
- **BLOCKED [host, issue #20](https://github.com/vanishs/boxweave/issues/20):** `boxweave_nesting(inspect)` returned `NESTING_HELPER_FAILED_INSPECT_ORIGINAL_INSTANCE` for the retained `bwaccept-20261101` Box. The helper/source location and service-privilege boundary must be inspected on its actual parent host. No `plan/apply`, parent restart, sudo change or new Box was performed to mask this failure.
- **T / real negative, isolated Box:** `resize_container({username:"bwaccept-20261101",cpu:"invalid-cpu"})` rejected with API error `Invalid CPU limit syntax` (the external MCP classified it as UNAVAILABLE/500, not a clean input 4xx). A **fresh `get_container` readback confirmed** original identity `1793567247`, `RUNNING`, `cpu:"3"`, `memory:"3GB"`, `disk:"20GB"` unchanged. Mark **PASS for rejection + no observed configuration mutation**, **PARTIAL for error taxonomy** (upstream returned 500 rather than a client validation error). No retry of the invalid mutation.
- **L / native Tenant configuration read:** `network_guard_status({})` returned both core and Tenant `GUARD_MODE_ENFORCE`, driver `nftables`, `unsupported:false`, empty error/unresolved fields; all four `boxweave-test` Boxes (including the new one) share the same attached ACL, and the separate `hello-world-test` Box has a different attached ACL. This is **PASS for configured/attached ACL observation only**. Do not mark same-Tenant TCP connectivity or cross-Tenant packet isolation PASS without actual traffic.
- **L / diagnostic read:** `boxweave_diagnose(source=gateway,recent=5)` classified the rejected resize as `UPSTREAM_JSONRPC_ERROR` / `unknown` even though a subsequent direct container read disproved a resource-change effect for inspected CPU/RAM/disk. Preserve the distinction between gateway message classification and reconciled Provider state; no silent retry or inaccurate success declaration.

**Disposition:** Skip issues #19 and #20 while other independent acceptance proceeds. No host-level repair, runtime code patch, new release or upstream daemon upgrade has been executed or verified in this triage; this is a recorded live observation and documentation update only. Retain `bwaccept-20261101-container` RUNNING for later evidence.

## Additional independent MCP read checks (2026-11-01, no guest shell required)

Ran read-only checks after marking host-login/nesting gates as BLOCKED; no new Box, host change, daemon upgrade, route exposure or removal was performed.

| Check | Actual observation | Acceptance classification |
| --- | --- | --- |
| `network_guard_status({})` | `GUARD_MODE_ENFORCE` for core and Tenant, nftables, supported, `lastError:""`, ACLs attached to all visible Boxes; four `boxweave-test` entries share the same ACL | **PASS** effective configuration attachment/readback only; actual packets **U** |
| `get_network_policy({tenant:"boxweave-test"})` | API 404 `no network policy for tenant\"boxweave-test\"`; `list_network_policies({})` previously returned `{}` | **PASS** consistent absent explicit policy; **U** actual default packet isolation |
| `list_routes({username:"bwaccept-20261101"})` | Empty `routes:[]`, `totalCount:0` | **PASS** read-only Web exposure inventory for the new test Box |
| `list_passthrough_routes({})` | One still-active TCP passthrough, external port 39471 -> `10.100.0.3:39471`, target `boxweave-test-ser-container`; description claims it was already removed after delivery | **PARTIAL / follow-up:** actual route exists but historic description contradicts current state; inspect target/owner before any removal. No user-requested cleanup of this unrelated original route was inferred |
| `security_sentry_status({})` | `DISABLED`, reason `CONTAINARIUM_THREAT_SENTRY is not set` | **PASS** truthful status reporting; monitoring efficacy **NOT ACTIVE/NOT CLAIMED** |
| `get_metrics_export({})` | Cloud metrics export disabled | **PASS** status visibility; no export was enabled |

These independent checks **do not need a host repair and must not be deferred** because SSH/Python/nesting tests are BLOCKED. See [host issue #19](https://github.com/vanishs/boxweave/issues/19) and [nesting issue #20](https://github.com/vanishs/boxweave/issues/20). With no authorized privileged host-shell or repair tool, there was no code or deployed host change to version-bump or publish as a new release in this round.


## BoxWeave 0.25.1 source fixes and release verification (2026-11-01)

**Release:** [v0.25.1](https://github.com/vanishs/boxweave/releases/tag/v0.25.1) is published as an immutable tagged source-bound candidate with 14 release assets. Release tag target and verified CI head are both `91ea4439718be02b3dc4a883a4e148c43a446122`. [CI run #41352986393](https://github.com/vanishs/boxweave/actions/runs/41352986393) has `verify=success` and `publish=success`. This is **CI/fixture release proof**, not an installed-Box checkpoint.

**Source changes covered by this release (S + T/CI):**
- `bwmini/mcp_gate.py` now exposes only allowlisted non-secret `TRUST_REGISTRY_UNAVAILABLE`, `TRUST_OBSERVATION_STALE`, `TRUST_NOT_READY`, and `TARGET_OVERRIDE_REFUSED` errors. Reads of `targets.json` fail closed and never reveal keys. The exact-target identity, observed address, creation time and freshness checks are still mandatory; no trust bypass or TOFU was introduced.
- `bwmini/mcp_gate.py` rejects malformed `resize_container.cpu` input with `RESOURCE_CPU_INVALID` / JSON-RPC `-32602` **before** Provider dispatch. Supported count, decimal, millicpu and Incus CPU-set shapes are preserved; Provider remains authoritative for semantic capacity checks. This is a gateway-level improvement to the previously observed `invalid-cpu` upstream 500, not a claim that Containarium's native API error taxonomy was changed.
- `bwmini/nesting.py` makes the privileged fixed-argument helper return allowlisted safe `OWNED_PROVIDER_REQUIRED`, `MANAGED_INSTANCE_IDENTITY_REQUIRED`, `LOCAL_INCUS_OBSERVATION_OR_CHANGE_FAILED` and similar existing classifications through the sudo boundary. Unknown errors remain redacted; no privileged helper authority, sudoers setting or Incus nesting state was changed.
- Added synthetic regression tests to existing `tests/test_workload_guidance.py` and `tests/test_nesting.py`, including no-upstream-write on invalid CPU, all supported CPU forms, stale/missing registry, allowlisted nesting failure propagation, unknown failure redaction and preservation of previous D4 gateway behavior.

**Actually tested on GitHub hosted Ubuntu 24.04 runner (T/CI, exact commit above):** `python scripts/check_knowledge.py` exit 0; code-first source/acceptance inventory drift guard succeeded; `bash -n install.sh`, `bash -n boxweave.sh` succeeded; Python unittest discovery **436 tests, pass**, with new cases included; generated Python client grammar/keepalive checks succeeded; frontend typecheck/build succeeded; seven browser fixture workflows passed; release ZIP module SHA-256/package manifest check passed; Python 3.14 compatibility/offline wheel smoke passed. Tagged Release asset publication verified by GitHub API. Intermediate incomplete commits failed CI during the edit sequence; **only the full tagged commit is the release acceptance subject**.

**Still blocked / must not be marked PASS:** The installed `dev` host previously reported the absent `/usr/local/bin/containarium-shell` for both old and new test Box accounts ([issue #19](https://github.com/vanishs/boxweave/issues/19)). There is still no authorized host-filesystem or sshd repair execution channel through BoxWeave MCP. The source fixes do **not** install the missing host wrapper and do **not** repair the outer `dev` system; follow the owner-approved host-specific remediation with validated, least-privilege sudo/sshd configuration. Live `boxweave_nesting(inspect)` still requires the fixed privileged helper and returns a generic error in the deployed 0.25.0 runtime ([issue #20](https://github.com/vanishs/boxweave/issues/20)); the new source classification remains untested at the installed boundary. No guest `hello world` exit 0 was observed, and real cross-version/Tenant packet traffic was not exercised. The retained `bwaccept-20261101-container` was left untouched for post-repair tests.

**Stable pin:** `install.sh` and `PUBLISHED_RELEASE` intentionally remain pinned to the previously accepted stable `v0.25.0`; the new `v0.25.1` GitHub Release is published but **not yet promoted as the default streamed installer**. Revisit only after separate clean/reused-host installation and outer MCP live checkpoint, which cannot be truthfully completed by this CI run.

## Streamed installer version selection and stable-pin promotion (2026-11-01)

**Feature and release:** Added native `--version 0.X.Y` to the main streamed `install.sh` invocation (`curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash -s -- --version 0.25.2`). The parser validates exact `0.X.Y`, rejects missing/malformed/duplicate versions before network or privileged installer work, and preserves existing `--provider-url`, `--provider-secret-file`, `--provider-cli`. The requested version passes into the isolated release selector; GitHub published release identity, source tag and SHA-256 checks remain mandatory before staging the matching `online.py` helper. No TOFU or weakened checksum checks introduced.

**Test evidence:** Published source commit [`724ee493efe1591391f9d9d119ec2ddadbd1b793`](https://github.com/vanishs/boxweave/commit/724ee493efe1591391f9d9d119ec2ddadbd1b793), [CI run #41354228068](https://github.com/vanishs/boxweave/actions/runs/41354228068) `verify=success`, `publish=success`. New `tests/test_host_bootstrap.py` coverage exercises argument rejection before download, exact requested/default version selection using in-memory HTTP response fixtures, release-tag/version mismatch refusal, no helper execution on mismatch, and stable pin reconciliation. Existing Python/fixture browser, frontend build, release ZIP/hashes, Python 3.14/offline wheel tests all passed. Release [`v0.25.2`](https://github.com/vanishs/boxweave/releases/tag/v0.25.2) is non-draft, has 14 attached assets, and its Git tag target matches the passing CI source commit. **CI tests are synthetic/offline; no production install, host-side bootstrap or real provider SSH acceptance is claimed.**

**Default stable pin promotion:** Only **after verifying immutable v0.25.2 Release** was the main branch `PUBLISHED_RELEASE` and installer embedded default changed from `0.25.1` to `0.25.2`, with README updated to reflect this. This follow-on commit does not alter the v0.25.2 tag or its original release assets. Its CI publish guard accepts only the exact source-bound installer pin replacement and documentation changes; other executable changes would require a new version. `--version` can explicitly select an older published version despite a newer stable default. 

**Unverified / separate gates:** Active installed outer BoxWeave MCP was previously reporting 0.25.0; this release does not silently upgrade that host. Host login shell [#19](https://github.com/vanishs/boxweave/issues/19), nesting helper [#20](https://github.com/vanishs/boxweave/issues/20), cross-version install compatibility and live end-to-end guest Python remain BLOCKED/UNVERIFIED pending scoped real-host execution. No host changes were made during the version-selector rollout.

## Agent-led source delivery and real-Box test workflow v1.0 (2026-11-01)

**Scope:** Developer-only guidance and offline source-snapshot helper; not a customer MCP feature or a new always-on execution service. Source code: [`scripts/agent_source_bundle.py`](../../scripts/agent_source_bundle.py); tests: [`tests/test_agent_source_bundle.py`](../../tests/test_agent_source_bundle.py); runbook: [`docs/process/AGENT_TEST_DEPLOYMENT.md`](../process/AGENT_TEST_DEPLOYMENT.md). The runbook supersedes earlier preference for a reused historical Box: first run focused local checks; for real deployment start with a genuinely new disposable Box unless environment-specific need is evidenced. Local dirty/untracked source is prioritized, can be packaged with exact byte and SHA-256 records without being committed or publicly uploaded. SSH/HTTP MCP control is not assumed to provide SCP/file-upload; allowed private first-party delivery options are conditional on actual capabilities and authorization.

**Offline verification targets:** Git fixture includes tracked modifications, untracked files, ignored-file exclusion, manifest source identity, ZIP and member SHA-256 verification, clean and deleted worktree behavior, suspicious credential filenames/symlink refusal, output overwrite rejection, digest mismatch and output inside checkout rejection. Run exact unit test plus knowledge check, then CI build/fixture tests. A passing fixture does **not** certify private delivery over a real client, Box creation, SSH, Provider/Incus setup, installed/runtime version, nested guest hello or actual cleanup.

**Live acceptance still needed (U/BLOCKED where capability absent):** An authorized Agent with a local modified worktree and permitted transport must snapshot it, deliver its exact hash into a *new* isolated Box, build/install using existing valid entry points, verify running code and a representative failure path, run a suitably scoped real behavior test, then observe approved cleanup. Also test source-identity refusal under mismatch and cleanup on partial failure. The installed outer `dev` host still had an absent SSH wrapper (issue #19) and nesting helper limitation (issue #20); neither is repaired by these developer-only changes. Preserve any real failure scene and record actual commands, outputs, exit codes, and exact resources when exercised.

**0.26.0 publication and stable pin (T/CI):** Full new-workflow source/fixture validation passed on GitHub Actions [run #41357821284](https://github.com/vanishs/boxweave/actions/runs/41357821284): `verify=success`, `publish=success`; `tests/test_agent_source_bundle.py` exercised normal and refusal paths as part of **444 Python tests**. Release [v0.26.0](https://github.com/vanishs/boxweave/releases/tag/v0.26.0) is published, non-draft, with 14 source-bound assets; the tag points to `67da66800ed370555a298ba4d48a3e03b5ba3336`, the passing CI source commit. Only after that validation was the default installer `python3` fallback version and `PUBLISHED_RELEASE` promoted to `0.26.0`, with README synchronized. The immutable v0.26.0 tag was not changed. **Live external SSH/HTTP MCP, new Box creation, actual guest deployment, nested child hello and task-owned cleanup remain separately unverified**, not PASS.

## 2026-11-01 whole-project code-first sweep — new/old Box live tests and source remediation

**Authoritative per-feature matrix:** [A01–A30 live acceptance ledger](CODE_FIRST_LIVE_ACCEPTANCE_2026-11-01.md). Historical static source inventory remains [dated baseline](CODE_FIRST_ACCEPTANCE_2026-11-01.md). This pass did not invent a new BoxWeave MCP tool or assume that every upstream tool advertised is a verified BoxWeave feature.

**Actually tested L/PASS:** Created a distinct `bwaudit-1101-container` on `dev` at `10.100.0.7`, native Tenant `bwaudit-1101`, later resized to CPU2/RAM4GB/disk30GB; `get_container` returned exact identity. Direct `connect(exec)` Python success/exit0 on new and existing `boxweave-test-ser` even though outer host-hop shell is absent. Wrong host/user returned `TARGET_OVERRIDE_REFUSED`; invalid CPU `RESOURCE_CPU_INVALID`. Fresh in-guest `curl main/install.sh | bash -s -- --version 0.26.0` ran via a scoped `systemd-run`, ended `BWAUDIT_INSTALL_EXIT_CODE=0`, installed `boxweave version=0.26.0`; `sudo boxweave doctor --mcp` all 10 required checks passed, Provider/Incus and app/worker operational, Web unauthenticated HTTP401. **True upgrade L/PASS:** v0.26.0→v0.26.1 on new Box and v0.24.0→v0.26.1 on historical `boxweave-test-ser`; both streamed installer exit0, `boxweave version`/public provenance source match v0.26.1 tag, authenticated MCP doctor all passed. Old existing inner child Python remained successful/exit0. Separate old `boxweave-test-ser2` v0.21.3 retained unchanged; nested Python still exit0. 36 installed client generator outputs (9 Apps ×4 supported platform/transport combinations) produced valid standalone Python shebang/compiled. Installed operator job ran exact exit0 and exit7 with correct receipts, and 1s timeout produced `JOB_TIMEOUT` with null child exit. Private bounded report window exported 7 files; unsafe parent refused. Exact missing Incus instance was misclassified on v0.26.0 and confirmed corrected `EXACT_INSTANCE_NOT_FOUND` on upgraded v0.26.1. Current GitHub source helper packaged/verified 303 files including a new uncommitted Python source in test Box with SHA-256 and cleaned its checkout/ZIP afterwards (not a cross-machine transport test).

**Actual TCP network acceptance L/PASS, scoped:** Temporary `10.100.0.3:18761` HTTP service returned 200 from same native Tenant `boxweave-test-ser2`; independent Tenant `bwaudit-1101` could not connect, with Guard ENFORCE nftables and attached ACLs. Stopped the exact temporary service, verified zero listening sockets and removed the owned directory; no public route or Tenant grant created. This is not comprehensive cross-tenant policy certification.

**L/FAIL and retained scene:** Fresh in-Box Incus initialized with default pool and bridge, but `bwaudit-inner-0260` failed to launch with kernel cgroup/BPF and proc mount permission denials; `boxweave box diagnose` classified parent nesting/mount confinement. Outer `boxweave_nesting inspect/plan` failed. Original stopped nested child and `bwaudit-1101` are **RETAINED** for parent-authorized investigation; no blind retry, parent restart or broad host SSH repair was attempted. Prior host-hop missing wrapper is **not** a universal direct SSH failure (direct IP success observed).

**S/T fixes:** v0.26.1 [source commit `80ca2b24dc779a05c5150b14bc6a898acfe1b953`](https://github.com/vanishs/boxweave/commit/80ca2b24dc779a05c5150b14bc6a898acfe1b953), [CI #41359643857](https://github.com/vanishs/boxweave/actions/runs/41359643857) `verify=success`, `publish=success`; missing-instance classification, self-guide source version and developer guidance fixed. v0.26.2 [release](https://github.com/vanishs/boxweave/releases/tag/v0.26.2) and [CI #41360341680](https://github.com/vanishs/boxweave/actions/runs/41360341680) `verify=success`, `publish=success`, including **447 Python unit/contract tests** and remaining frontend/browser/package/offline-wheel stages. Its CLI report help now lists only reader-supported operation kinds; the prior `job/selftest` selections were advertised but not executable as reports. **Current v0.26.2 installed check is UNVERIFIED**; a follow-up guest installation request was rejected by the execution-channel safety check and was not replayed. Immutable publication and source CI are not a substitute for live checkpoint.

**Still U/BLOCKED:** Public external HTTP OAuth/TLS client, real GUI/Windows/macOS/WSL client integration, D4 allowlisted positive case, current-version app receipt/worker crash/no-replay, selftest requiring a healthy nested parent, paid model inference, cross-machine private local-source transfer and outer gateway upgrade. Tasks without a safe execution channel are documented and skipped; continue independent scopes. Mark each row only against evidence in the linked ledger. `PUBLISHED_RELEASE` defaults to v0.26.0 until any later promotion is separately validated, rather than falsely assuming all v0.26.2 guest acceptance succeeded.

## Post-host-v0.26.2 source-first whole-project reacceptance — A01–A33 (2026-11-01)

**Full current status matrix:** [REACCEPTANCE_0262_TO_0264_2026-11-01.md](REACCEPTANCE_0262_TO_0264_2026-11-01.md). This record supersedes the previous assumption that all current outer SSH is blocked: direct Box IP `connect(exec)` passed, while the distinct host-hop shell remains absent. Compared code and prior acceptance docs across all 30 original areas, adding A31–A33 for code paths not fully covered by earlier narrative.

**L/T actually executed:** (1) After user outer host upgrade, `boxweave_info` advertised static v0.26.2. (2) New test Box `bwaudit2-1101` was created with requested nesting but a partial error; exact Provider read proved created RUNNING, not safe to blindly re-create. Fresh Ubuntu 24.04.5 install **v0.26.2** succeeded and `doctor --mcp` passed 10/10. (3) In-Box upgraded **v0.26.2→v0.26.4**, verified version/provenance source `00bacdf755a07127d978ec0a6bda56fb33db544d`, 10/10 doctor and post-stop/start Python exit0, identity unchanged. (4) **Historical v0.21.3→v0.26.4** on `boxweave-test-ser2` succeeded; existing inner `demo-1-container` remained RUNNING, Python printed `UPGRADE_0213_TO_0264_INNER_HELLO`, exit0. Previous v0.21.3 installed comparison is no longer available because owner authorized the real upgrade. (5) 36 generated client profiles compiled as Python, no actual external client enrollment claimed. (6) SSH exact-user/host mismatch and malformed CPU denied, D4 default-deny passed, D3 gateway read and unavailable app receipt typed correctly. (7) Real same-Tenant fixed HTTP body 200 and other-Tenant connection refusal on `10.100.0.8:18761`; temporary service removed, listening count 0. Auto-sleep enabled then disabled (not an actual idle-sleep test). (8) The test Box executed **453 Python unittest cases PASS** after replacing an invalid root test execution with a dedicated unprivileged venv. (9) Source Incus query project bug demonstrated by exact v7.6 CLI comparison; v0.26.3 candidate helper from a pinned Commit actually performed inspect, plan, idempotent apply (changed=false) and refused stale plan on real existing inner Box. (10) **Installed v0.26.2 `GET /api/login` 500 reproduced**; v0.26.4 returned HTTP 405 `METHOD_NOT_ALLOWED`, unknown route 404 `NOT_FOUND`, unauthenticated projects 401, with Web/worker/MCP health still passed.

**Source fixes and publication:** [v0.26.3](https://github.com/vanishs/boxweave/releases/tag/v0.26.3) `c2983051...` source-bound CI [#41362067153](https://github.com/vanishs/boxweave/actions/runs/41362067153) verify/publish both success; [v0.26.4](https://github.com/vanishs/boxweave/releases/tag/v0.26.4) `00bacdf755...` CI [#41362313473](https://github.com/vanishs/boxweave/actions/runs/41362313473) verify/publish both success, with CI browser/packaging and Python suites; independent non-root Ubuntu guest local tests also passed 453/453. The published v0.26.4 tag remains immutable. The default installer release pin is being separately promoted to **0.26.4** only after these source/release/install/upgrade observations, not as proof of untested host nesting or external services.

**Cleanup and unchanged blockers:** Test-owned `bwaudit2-1101` successfully stopped, deleted and confirmed absent in `list_containers`. Network fixture service and files cleaned and independently observed, auto-sleep disabled. Earlier `bwaudit-1101` remains intentionally retained with failed stopped nested child; existing historical Boxes were not deleted. Outer gateway is still owner-installed v0.26.2, so the fixed `boxweave_nesting` helper needs a separately authorized outer host upgrade and *real* child creation/execution; HTTP OAuth/TLS/public clients, GUI, positive D4, model inference, external source transfer and genuine nested selftest remain **U/BLOCKED** and were skipped. One ambiguous post-create nesting error is OPEN: inspect the original Box before any retry. See exact status/risks in the linked report.

## Host v0.26.4 review and v0.26.5 partial-create correction (2026-11-01)

**True live nesting L/PASS:** Host outer BoxWeave v0.26.4 now runs `boxweave_nesting inspect` successfully. The original `bwaudit-1101` was confirmed nesting **false** and retained unchanged. New authorized `bwnestfix-1101` with `enable_nesting=true` returned an exact successful nesting-config receipt; independent inspect confirmed identity/config, followed by task-owned stop/start. Installed inner v0.26.4 and authenticated MCP doctor passed 10/10. `incus launch images:ubuntu/24.04 bw-nest-pass` yielded exit0 and Running instance; `incus exec` produced `BW_NESTED_PYTHON_OK 3.12.3` and child exit0. One immediately-after-start read returned `TRUST_NOT_READY`, and normal read/exec recovered after observing readiness; no destructive replay. This establishes nesting positive on an appropriately configured parent, not automatic repair of the old Box.

**Source fix A33 v0.26.5:** Preserve actual successful Provider create result and include explicit `isError=true` partial status when the independent nesting helper fails, with outcome `not_verified`, safe code, original username and no-repeat instruction. Add deterministic no-recreate/receipt-preservation/redaction tests. No host privileges or security policy changed. Exact source commit and CI evidence belong to the release pipeline; **live induced post-create failure remains UNVERIFIED**. Other U/BLOCKED items remain in the linked [acceptance record](NESTING_ACCEPTANCE_0265_2026-11-01.md).

## v0.26.5 published acceptance and stable default promotion (2026-11-01)

**T PASS:** [Source-bound CI #41365974385](https://github.com/vanishs/boxweave/actions/runs/41365974385) `verify=success`, `publish=success`; immutable [v0.26.5 Release](https://github.com/vanishs/boxweave/releases/tag/v0.26.5), tag at `115d48ed73dcfbe79f06cc40265fe7a1e35be15e`, 14 assets. 454 Python unittest tests recorded and all existing browser/frontend/package/offline checks passed. **L PASS (selected):** real `bwnestfix-1101` v0.26.4→v0.26.5 install/upgrade exit0, running version/provenance source verified, `doctor --mcp` 10/10 passed, inner child Python exit0 after upgrade. **Cleanup:** task-owned nested child and new outer Box deleted, independent exact-box list/readback confirms absence; old failing audit Box retained. Version/default installer pointer set to `v0.26.5` only after this check, with no rewriting of immutable tag. **L/U:** v0.26.5 deliberately induced post-create failure over outer gateway and client error rendering still not exercised; no user-host auto-upgrade claimed. [Full scoped report](NESTING_ACCEPTANCE_0265_2026-11-01.md).

## v0.26.6 deep reverse audit and newly completed selftest — 2026-11-01

**Authoritative new record:** [DEEP_AUDIT_0266_2026-11-01.md](DEEP_AUDIT_0266_2026-11-01.md), building on A01–A33. **A34 source defect:** `remote_mcp.approve → RemoteCredentials.approve → _rate → Store.db` stores each denied admin auth attempt in a transaction rolled back when `RemoteError` is raised, so 5/min rate limiting does not persist wrong-password denials. v0.26.6 fix commits the rate row in an isolated transaction and leaves grant validation/consumption atomic. Added a synthetic temporary-database Flask/OAuth regression for committed denials, 429, zero code issuance and success after the window. **No public credentials tested.**

**A23 true L/PASS:** On newly created authorized Ubuntu `bwdeep-1101` with `enable_nesting=true`, installed official v0.26.5, ten doctor checks passed, ran actual `sudo boxweave selftest plan` and confirmed `run` with unique plan hash; inner `boxweave-selftest-f014ec-container` was created, ran hello and was stopped/deleted. Receipt `state=passed`, cleanup `not_needed`, four step successes/OK; independent Incus list showed **0** inner instances. Negative confirmation mismatch returned exit2 with zero child instances, but host CLI only reported generic `SELFTEST_FAILED_INSPECT_STEPS` (**A35 P2**). Don't confuse refusal with a successful run.

**Pending before promotion:** CI new Python/HTTP fixture, browser/build/offline suite, published immutable v0.26.6 tag, then real v0.26.5→v0.26.6 guest upgrade/provenance/doctor and independently verified task-owned Box cleanup. Parent host remains owner-installed v0.26.5; no automatic host upgrade. All external/host ACL blockers remain U/BLOCKED.

## v0.26.7 SSH trust-readiness and typed selftest discovery (2026-11-01)

**A36 v0.26.6 L/FAIL reproduced:** On independent Box `bwdeep-1101`, v0.26.5→v0.26.6 upgrade and doctor 10/10 passed, but two distinct confirmed selftest runs failed after create. App-output repro `boxweave-selftest-249ff8`: `create:OK`, `hello:TRUST_FAILED`, `stop:OK`, `delete:OK`, service exit2, zero inner Box. Host wrapper printed generic `SELFTEST_FAILED_INSPECT_STEPS`. The source patch in v0.26.7 waits only around the exact idempotent identity/hello check on classified trust-ready faults, and exposes bounded safe typed failed-step/refusal output instead of discarding it. Real installed v0.26.7 full acceptance still needed. [Reproduction and cleanup](SELFTEST_TRUST_RACE_0267_2026-11-01.md).

## v0.26.7 immutable release and installed acceptance completed — 2026-11-01

**T PASS:** Release [v0.26.7](https://github.com/vanishs/boxweave/releases/tag/v0.26.7) has exact tag `526bc1a6625270390e1e1ebe7168e0fc4a49542b`, 14 source-bound assets; [CI 41369848703](https://github.com/vanishs/boxweave/actions/runs/41369848703) verify and publish succeeded. The earlier CI on `d4249c53...` was a real FAILED fixture run (two legacy tests incorrectly expected immediate readiness failure); corrected test/patch commit was independently verified, not silently treated as passing.

**L PASS on authorized disposable Box `bwdeep-1101`:** Upgraded installed v0.26.6→v0.26.7 via released installer, exited0; running version/provenance exactly v0.26.7/`526bc1a...`, doctor 10/10. Negative plan hash mismatch produced typed `SELFTEST_CONFIRMATION_MISMATCH`, exit2 and zero child Boxes. Positive app-owned `boxweave-selftest-e64976`: create, hello, stop, delete each succeeded/OK, exit0; SQLite `selftest_runs.state=completed`, `last_step=delete:OK`, cleanup `not_needed`, independent final Incus inner inventory **empty**; post-run doctor still passed. Host trust timer observed `targets=1` during new child existence, no known-host bypass.

**Cleanup L PASS:** The task-owned new parent `bwdeep-1101` was stopped/deleted, MCP `list_containers` independently confirmed exactly six original existing Boxes and absence of `bwdeep-1101`. Historical v0.26.6 failures retained only as records. Stable default installer pointer promoted to **0.26.7** in separate main commit only after release and live acceptance; source tag remained immutable. The owner's host (reported 0.26.5) was **not** upgraded by test work.

**Still U/BLOCKED:** Public OAuth/TLS client and admin login, D4 allowlisted positive, host-hop SSH wrapper, cross-platform real GUI setup, deep service-crash unknown/no-replay and private cross-machine dirty worktree delivery. See [current specific report](SELFTEST_TRUST_RACE_0267_2026-11-01.md).

## Whole-project 0.26.8 deep reacceptance — A01–A38 (2026-11-01 local / 2026-11-02 UTC)

**Authoritative scoped per-feature ledger:** [DEEP_REACCEPTANCE_0268_2026-11-01.md](DEEP_REACCEPTANCE_0268_2026-11-01.md). Reviewed all implemented A01–A36 surfaces against source/tests/older acceptance records, plus **A37** D4 inactive/failed/unknown systemd exit classification and **A38** public-key fixture umask independence. The new 0.26.8 source candidate is not a published release or installed owner host.

**T PASS (source candidate 0.26.8):** In test Box `bwaudit-1101`, `python3 scripts/check_knowledge.py` exit0 (138 docs), `.venv/bin/python scripts/validate.py` exit0 (458/458 unit tests and frontend build) under `UMask=0002`; D4 focused 11/11 and test key fixtures 13/13 plus 5/5; seven Playwright fixture browser workflows exit0 with temporary Web service stop/restore and separately recovered doctor 10/10. The original occupied-port browser attempt and 0002 key-fixture errors are recorded as real pre-fix failures, not counted as passing.

**L PASS on independent new nesting-enabled `bwfullaudit-1101`, installed released 0.26.7:** fresh streamed installer exit0; doctor --mcp 10/10, authenticated MCP read; invalid selftest confirmation exit2 with zero guest, positive app-owned selftest `boxweave-selftest-115305` create/hello/stop/delete all OK exit0 with independently zero children; 36/36 installed client profile scripts compiled; operator job success (child/host exit0), explicit exit7 (host exit1), and timeout (host exit1/null child exit). Old live 0.26.1 Web GET /api/login 500 contrasted with v0.26.4/v0.26.7 405, unauthenticated project 401. The original old Boxes were not upgraded by this reacceptance.

**U/BLOCKED:** 0.26.8 CI/release and installed candidate upgrade; D4 operator allowlisted positive read, public OAuth/TLS/client GUI, paid inference, outer host-hop, intentional crash/no-replay, true auto-sleep, uncommitted cross-host transport. Source/unit evidence for A37 cannot be called L positive. The new parent Box remains task-owned pending candidate upgrade; old failure Box kept. Do not claim complete end-to-end certification.

## v0.26.8 published source, installed upgrade, restart and cleanup — final checkpoint (2026-11-02 UTC)

**T PASS:** [CI #41384817175](https://github.com/vanishs/boxweave/actions/runs/41384817175) exact source `1ea072865222585df96875dd4ae5b2d4ff90ed71`, verify/publish success, immutable [Release v0.26.8](https://github.com/vanishs/boxweave/releases/tag/v0.26.8) tag same SHA, 14 assets. **L PASS:** installed v0.26.7 repeat-install exit0 followed by released v0.26.7→v0.26.8 upgrade exit0 on task-owned `bwfullaudit-1101`; installed provenance exactly tagged source, doctor 10/10. Installed D4 fixed parser gave bounded observed service-state classifications including actual Linux `inactive`/exit4 envelope; this is NOT an authorized outer D4 gateway positive. Post-upgrade wrong confirmation typed refusal exit2, valid application-controlled nested selftest `boxweave-selftest-9bcf53` create/hello/stop/delete all OK exit0, final inner list empty. Outer stop/start preserved v0.26.8 identity and doctor 10/10. Isolated HTTP test local HTTP200, other native Tenant endpoints timeout/HTTP000 with Guard ENFORCE; owned listener and files cleaned. Newly created parent stopped/deleted, `list_containers` independently saw the original six only; old failed scene retained. [Full A01–A38 matrix and blocked scopes](DEEP_REACCEPTANCE_0268_2026-11-01.md). Separate stable installer/default pointer promotion must leave released v0.26.8 tag immutable.

## 0.26.9 alternate transaction/state-transition sweep (2026-11-01 local / 2026-11-02 UTC)

See the [A39 cross-layer failure-injection ledger](STATE_TRANSITION_AUDIT_0269_2026-11-01.md). On source v0.26.8, an injected failure between issuing access and refresh tokens left a revoked predecessor and one orphan valid access credential; the same split invalidated OAuth one-use codes ahead of full issuance. Source v0.26.9 uses a single SQLite `BEGIN IMMEDIATE` transaction for consume/revoke plus both inserts. New real SQLite/Flask fault, duplicate digest, 8-thread racing refresh and eventual-success regressions **T PASS** (not public OAuth/TLS acceptance). Fresh authorized `bwstate-1101` nesting configured; first read after stop/start returned `TRUST_NOT_READY`, a subsequent exact read passed without trust bypass. Candidate release/installed upgrade and task Box cleanup are pending independently verified checkpoints; non-test host and credentials unchanged.

## 0.26.9 immutable release, live installed A39 acceptance and cleanup (2026-11-02 UTC)

**Source T/CI PASS:** [CI #41387108805](https://github.com/vanishs/boxweave/actions/runs/41387108805) verify/publish success; immutable [Release v0.26.9](https://github.com/vanishs/boxweave/releases/tag/v0.26.9), 14 files, tag `008f659593376c7df1608599bd523885197a6362`. Full source candidate 463/463 Python unit suite PASS, frontend build and browser fixtures through CI.

**Live actual version delta:** Task-owned Ubuntu nested Box `bwstate-1101`, installed v0.26.8 reproduced `revoked_old=1/orphan_access=1` from a controlled synthetic second issuance failure. Official v0.26.8→v0.26.9 installer exit0, exact provenance tag, doctor --mcp 10/10; same installed fault **revoked_old=0/orphan_access=0**, safe later rotation; code exchange partial failure preserved code, allowed recovery once and refused replay. Real loopback HTTP synthetic bearer POST 200, anonymous401, invalid Origin403, revoked401 on both versions; no public credentials. Outer online CPU2→3 and disk30→40 readback, app selftest full four OK before/after upgrade with independent zero children, post-stop/start version/doctor/resources stable despite transient `TRUST_NOT_READY` on first read. Task-owned `bwstate-1101` deleted and independently absent from the final six-Box inventory. **U/BLOCKED:** public OAuth/TLS/real client, worker crash/unknown write, D4 positive root-policy, true idle autosleep and cross-host dirty source. See [detailed A39 acceptance](STATE_TRANSITION_AUDIT_0269_2026-11-01.md). The stable pointer is promoted in a separate commit after the observed release/install rather than rewriting its tag.

## HTTP MCP quick-start v0.27.0 implementation checkpoint

Source scope: default authenticated loopback MCP 127.0.0.1:18844 with auto-managed user-systemd unit, explicit HTTPS setup via operator CLI, and Cloudflare Quick Tunnel guidance. Design: [HTTP MCP quick-start](../design/HTTP_MCP_QUICKSTART.md). Planned fixture checks: `python -m unittest discover -s tests -v`, `python scripts/check_knowledge.py`, full PR/main GitHub CI. No real host, firewall, tunnel process, Box or public TLS modified by source commits. **Record executed CI run/commit before marking fixture PASS**; real installed upgrade and Chrome OAuth callback remain U until independently observed.

## 0.27.0 HTTP MCP quick-start source checks (2026-11-01)

GitHub Actions PR [41390973974](https://github.com/vanishs/boxweave/actions/runs/41390973974) on source head `5540b85aaea42628982632717e75269431ad76ea`: completed success. Python unittest/contract, source inventory and authority, shell, frontend, seven browser fixture flows, release ZIP Python module/digest completeness and Python 3.14 offline wheel smoke passed. Merged into main as `f4e99d83c1e36393d13667f3cc8805b57298fcbc`. Evidence class T for fixtures and CI package; no newly executed installed systemd service, external Cloudflare callback or real Box write (I/E/L=U on 0.27.0). The published source tag and stable pin are checked separately; do not conflate promotion with real-host acceptance.

## v0.27.1 — standalone application installer import failure after default HTTP MCP (2026-11-01)

**Incident:** Operator-installed v0.27.0 stopped at `independent_application` with outer `INSTALL_OR_CONFIGURATION_FAILED`. Root cause in tagged code: `bootstrap.py` invokes downloaded `distribution.py` as an independent script via `/usr/bin/python3 <stage>/distribution install`; the new 0.27.0 `config_read` and install path execute `from .remote_mcp import validate_remote_config`, which cannot resolve a parent package in this invocation. `online.py` then attempts to parse the traceback-containing private application log as JSON and collapses the actionable error into an opaque error code.

**Repair:** v0.27.1 keeps the reviewed standalone installer dependency-free with a local strict remote-config validator and removes both package-relative imports. The stream installer now treats non-JSON private log content as a typed `APPLICATION_INSTALL_INCOMPLETE_INSPECT_PRIVATE_LOG` error while leaving original evidence intact. New fixture test executes installer via `runpy.run_path` with no package context and verifies HTTP default, custom HTTPS, invalid host/origin/port rejection. Source/CI pass must be recorded after the final commit. The user's installed v0.27.0 failure remains field-observed only; the actual private host log was not retrieved, so exact host stacktrace remains U. Exact v0.27.1 host repair and systemd auto-start remain U until reinstalled.

## v0.28.0 HTTP-first unified client setup

Task owner-approved: HFC-01..07 in [design](../design/HTTP_FIRST_CLIENT_SETUP.md). New opt-in `--method http` standalone client installer with HTTPS URL discovery and multi-App safe config merges; `--transport` and `--admin-user` preserve historical SSH behavior. Fixture unit tests include failed writes, conflict backup, generated script compile and 9×3 App/OS catalog. Do not label GitHub fixture checks as live Agent OAuth. Record exact PR CI run, release SHA and any independent installed-client acceptance separately. Existing real remote HTTP hosted ChatGPT tool reads are prior installed-revision evidence, not v0.28.0 client registration verification.

### v0.28.0 source and isolated client-script acceptance (2026-11-01)

- **T / PASS:** [PR #27 GitHub CI run 41400117885](https://github.com/vanishs/boxweave/actions/runs/41400117885) on head `c39425d1e41e2fb600a987320e059ddcc708eff3` completed successfully: source/knowledge inventory, shell syntax, Python unit/contract fixtures including HTTP profiles and old SSH regression, Python script grammar/shebang, frontend, seven fixture browser workflows, release ZIP Python inventory and Python 3.14 wheel smoke. Merged as `2fc27a5b70734bcc9664fb05ce7c72a979c68615`.
- **L / isolated client-script fixture PASS (NOT installed App OAuth):** Connected via authorized BoxWeave MCP `connect(box=boxweave-test-ser)` to the existing test Box, downloaded the candidate `bwmini/client_profiles.py` from the exact feature branch into a fresh temporary directory and executed generated Cursor, Codex and OpenCode HTTP client scripts with separately isolated `HOME` values. Each returned `exit_code=0`, created the expected scoped config, and emitted `awaiting_authorization`; no real client config was touched. The temporary directory was automatically removed.
- **U:** No installed v0.28.0 systemd app upgrade, real GUI/CLI Agent App OAuth, external MCP initialize/tools/list/read for each of the nine clients, or all-OS real device test. A generated registration does not verify that the App loads or authorizes it. The Qoder/TRAE UI-import path is deliberate rather than an undocumented automatic file edit.
