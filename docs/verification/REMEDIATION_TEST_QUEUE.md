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
