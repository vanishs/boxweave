# Remediation validation and next-Agent queue

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

