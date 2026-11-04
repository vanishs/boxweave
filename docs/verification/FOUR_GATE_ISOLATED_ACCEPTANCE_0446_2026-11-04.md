# v0.44.6 four-gate isolated installed acceptance — 2026-11-04

**Status: PARTIAL overall, with the tested isolated pathways PASS.** S = reviewed source; T = actually executed fixture tests; L = authenticated, exact-version installed environment/real Box operations; U = unverified/blocked. **No new runnable-source bug was demonstrated; no new version is necessary.** This document augments, never retroactively rewrites, [the original v0.44.6 release/reproduction](CONNECT_TARGET_ALIAS_0446_2026-11-04.md).

## Identity, authority and cleanup

Outer controller: BoxWeave Gateway self-reports **0.44.3**, Containarium daemon **0.128.0**, Incus **7.5.1**, backend `dev`; owner host remains unchanged. Current GitHub `main` baseline before this report `dfe5478aa4289ad12c025c37e7759f4738fff832`; immutable source-bound [v0.44.6](https://github.com/vanishs/boxweave/releases/tag/v0.44.6) SHA `20cd8782f8762b9ae51ae03527624d46da967784`, [CI 41855559707](https://github.com/vanishs/boxweave/actions/runs/41855559707) PASS. Stable `PUBLISHED_RELEASE` remains **0.43.1**.

New task-owned *clean* Ubuntu 24.04 Box `bwverify-0446-e2e-container`, logical `bwverify-0446-e2e`, UUID `e70ce653-6adc-4389-bd01-600248a8a789`, createdAt `1793806163`, initial outer IP `10.100.0.22`, **3 CPU / 6GB RAM / 45GB disk**. Parent `security.nesting=true` was applied using `enable_nesting=true`, inspected, deliberately **stopped then restarted**, and original incarnation independently verified. Final outer parent **STOPPED** to conserve resources yet preserve its installed test environment; stopped Box no longer has a reachable IP. No other historical Box was deleted, no outer host Provider/Incus/SSH/key/trust/firewall changed.

Nested Containarium v0.128.0 and Incus **7.6** were installed *inside* this newly created test Box only. Two named child test Boxes were operated inside: `bwchild-0446-container` via installed Gateway MCP and `bwweb-0446-container` via installed application lifecycle. Both were deliberately stopped/deleted, and final **inner `incus list --format=compact` empty**; the built-in selftest child `boxweave-selftest-7670b1-container` was independently cleaned. Temporary fixture/source and private task metadata remain in stopped test parent for follow-up. No private keys, JWTs, admin passwords or raw secret-bearing config were printed or exported.

## 1. Nesting configuration vs actual nested execution (L PASS)

Live outer `boxweave_nesting(action=inspect,logical=bwverify-0446-e2e)` observed nesting true, same UUID/createdAt after restart. On a *fresh installed* BoxWeave v0.44.5, authenticated `sudo boxweave selftest plan` returned a specific plan/confirmation; first execution returned `SELFTEST_PLAN_INVALID` because the app account could not read the plan saved under the unrelated user's private directory. Without changing the Box or plan contents, moved the exact confirmed plan into app-owned mode-0600 location and ran `sudo boxweave selftest run --plan /home/boxweave/selftest-plan-0445.json --confirmation <confirmed SHA> --budget 900` through a task-owned systemd unit.

Observed `state=passed`, `evidence=live_mcp`; child `boxweave-selftest-7670b1` created (~42.679s), Python **`hello world`** executed (exit 0, ~0.444s), stopped (~1.167s), deleted (~1.791s). Independent Incus absence read verified cleanup. **Nesting flag alone was not counted as execution proof**; this real child cycle is the pass evidence.

## 2. R-02 private task tracking and concurrent idempotence (L PASS after opt-in)

Before provisioning, installed Gateway `boxweave_task_list(box=bwchild-0446)` returned **`TASK_TRACKING_UNAVAILABLE`**. Executed `sudo boxweave task provision` **twice**, both exit 0 with `state=ready`, `private_metadata_only=true`, `coordination_prepared=true`, `coordination_enabled=false`; app-user-owned mode-0600 `gateway-tasks.sqlite3` and `claims.sqlite3` observed.

Via installed MCP Gateway, created `bwchild-0446` (physical `bwchild-0446-container`), observed RUNNING with `10.203.0.2` and disk **10GB**, authenticated logical `connect(exec)` returned exact nested username/hostname and Python stdout with exit 0. Submitted an exact `boxweave_task_submit` request with a 32-hex request key: task ID `4c5d1911eae3943703f00485b85bc802`, **completed/exit0**, metadata-only result available:false. Repeated same key/payload returned the **same ID**, task_get/task_list were consistent, same key/different payload failed **`REQUEST_KEY_CONFLICT`**, and the guest file held **one** appended line. After installed v0.44.6 upgrade, `task_get` returned the same completed ID and file remained one line, proving persistent receipt/guest continuity.

Two independent Gateway processes concurrently submitted a second identical task key and command. Both reported the **same** task ID `4a97063d5b557aceeeebb4d71b431f56` (one saw `recorded` during the race and the other `completed`); one line appeared in the separate guest side-effect file. This proves concurrent same-key metadata deduplication in this test. **Simulated disconnected/UNKNOWN dispatch and global Web/MCP/REST R-04 shared claims were not live tested**; the test's coordination was explicitly not enabled.

**Outer host still requires separate operator-approved provisioning**; publishing/upgrading BoxWeave does not automatically initialize its R-02 database.

## 3. MCP / REST / Web disk shrink protections (scoped L/T PASS; installed Web auth U)

Physical nested `bwchild-0446` initially **10GB**. On real installed Gateway **v0.44.5 and v0.44.6**, `resize_container(username=bwchild-0446,disk=8GB)` returned **`RESOURCE_DISK_SHRINK_REFUSED`**; invalid unit `8GBjunk` refused `RESOURCE_DISK_SIZE_UNVERIFIED`. Live `get_container` before/after retained **10GB**, identity/IP.

Using the installed app-account authenticated `RESTClient` against actual local Containarium REST, `mutate(resize,disk=8GB)` and `8GiB` returned **`RESOURCE_DISK_SHRINK_REFUSED`** and malformed `garbage` returned **`RESOURCE_DISK_SIZE_UNVERIFIED`**; an instrumented client counted only **GET** preflights and **0 PUT**, and real child remained 10GB.

For the Web-shared application path, backed up private task-owned guest `config.json` before adding an approved test-only lifecycle allowlist for `bwweb-0446`, and allowed the official installer to regenerate `runtime.json` on upgrade (never hand-edited generated state). With the **installed application lifecycle CLI and running worker**, created `bwweb-0446` as a managed real child through plan `8a4502ef81e54e91a7364b81fe8a1c06` and operation `4ca4ef7723684277ad287f0ba9992324`. Final receipt **succeeded/connection_ready**, exact username/hostname SSH identity exit 0; verified disk 10GB. Its installed lifecycle `environment-plan --environment-action resize --disk 8GB` then returned **exit 2 / `RESOURCE_DISK_SHRINK_REFUSED`**, with child disk unchanged.

Real installed unauthenticated `POST http://127.0.0.1:18640/api/environment-plans` returned **HTTP 401**, proving the auth boundary. Authenticated **Flask test-client** using synthetic fixture credentials (not the installed admin password) invoked the actual Web route with a fixture 20GB managed target: HTTP **422 `RESOURCE_DISK_SHRINK_REFUSED`**, **zero resize dispatches**; wrong Origin returned **403**. Exact v0.44.6 source `20cd878...` ran two targeted `test_lifecycle` Web/lifecycle tests in the installed application's dependency-complete venv, both PASS. A real installed-service authenticated Web HTTP POST **was not executed** because the private admin sign-in credential was not available within authorized scope. This is **L BLOCKED/U**, not an end-to-end HTTP claim. Note BoxWeave preflight guards are not global atomic Provider constraints: direct native Containarium REST or races outside these gates remain unverified.

## 4. Fresh installation, upgrade, installed regression and cleanup (L PASS)

On the new clean parent, downloaded exact public v0.44.5 `install.sh`, passed `bash -n`, and invoked `bash install-0445.sh --version 0.44.5` inside an isolated systemd-run unit. Real nested Incus/Postgres/local Containarium 0.128.0/BoxWeave app were provisioned; installer terminal state `ready` at source `156afa40ed278ee2af7a79dcd9d25bd5be425dbd`. `sudo boxweave doctor --mcp` **10/10 PASS** with authenticated upstream MCP version0.128.0 and REST inventory.

Installed `bash install-0446.sh --version 0.44.6` from immutable release in the **same** test guest; finished exit 0. Existing local Provider 0.128.0 retained (`upstream_upgrade.status=kept_compatible`), nested child and private task receipt survived, custom guest test lifecycle config preserved and propagated to generated runtime. State now `ready`, version `0.44.6`, exact source `20cd8782f8762b9ae51ae03527624d46da967784`. `doctor --mcp` again **10/10 PASS**.

Prior v0.44.5 installed Gateway's `connect(box="bwchild-0446-container",exec=...)` failed as the old generic `TRUST_OR_WORKLOAD_KEY_GATE`, while logical username succeeded. On installed v0.44.6, **same concrete alias now succeeds**, exit 0 and verified logical user/physical hostname; `boxweave_box_list_directory(box="bwchild-0446-container")` also succeeds; bad-IP override still **`TARGET_OVERRIDE_REFUSED`**. This proves the prior v0.44.6 bugfix in an actual installed isolated environment. It **does not** update or prove the outer owner's running version.

Finally observed app-managed stop receipt `6ee66a4350fa4cdba3326c086e627169` as **succeeded/stopped**, managed delete receipt `de9f039b14c844599eab3b98e1f5a24f` as **succeeded/absent**. Separate MCP child stopped/deleted. Final inner Incus empty, BoxWeave `doctor --mcp` remained 10/10 and the test parent was stopped, read back as STOPPED with original identity and resources.

## Acceptance summary and follow-up

| Requirement | Evidence scope | Outcome |
| --- | --- | --- |
| Nesting configuration, restarted parent, actual nested create/exec/stop/delete | installed v0.44.5, authenticated MCP, physical child | **L PASS** |
| R-02 explicit provisioning, task receipt/list, same-key dedup, persistence, parallel side effect once | installed v0.44.5→0.44.6, nested guest | **L PASS** |
| MCP no-shrink incl. invalid size | installed v0.44.5 and v0.44.6, real 10GB child | **L PASS** |
| Authenticated app REST no-shrink and no Provider PUT | installed v0.44.6 / live Provider0.128.0 | **L PASS** |
| Web-shared app lifecycle plan refusal on a real managed target | installed v0.44.6 CLI + worker | **L PASS** |
| Web HTTP handler authenticated shrink, no dispatch, wrong Origin | synthetic fixture auth on exact v0.44.6 source in installed venv | **T PASS** |
| Authenticated shrink POST to running installed Web service | initial admin password deliberately not accessed | **BLOCKED / U** |
| Clean streamed v0.44.5 install, upgrade to v0.44.6, authenticated doctor | isolated installed Linux host | **L PASS** |
| Installed v0.44.6 concrete Box-name exec/read and bad-IP refusal | installed nested Gateway and preserved Box incarnation | **L PASS** |
| Owner-host upgrade, R-02 host provisioning; global R-04 cross-entry/atomic Provider shrink guarantees | not modified/verified | **U / deferred** |

**Disposition:** no confirmed new software defect and therefore **no new patch release**, no change to v0.44.6 immutable Release or default stable 0.43.1. Keep `bwverify-0446-e2e-container` **STOPPED** with the installed 0.44.6 stack for future authorized Web auth work, and separately arrange an explicit operator upgrade/provisioning if owner-host acceptance is needed.
