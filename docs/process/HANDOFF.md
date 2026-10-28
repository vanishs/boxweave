# Phase 4 handoff — retained partial minimum system

Date: 2026-10-28 America/Los_Angeles. **Result: partial implementation; real minimum-system gate not passed.** User SRC-030 approved the exact execution scope; after the missing independent application launcher was reported, SRC-032 confirmed no path is available and requested preservation. No new credential was minted or borrowed.

Legacy: refs/heads/main@4318da6ea4e4e45c55cf43b64acbdd2ede65fc43; last knowledge sync same SHA. Branch/worktree: rewrite/v2 at C:/Users/vanis/.codex/visualizations/2026/10/28/01a18158-8e09-7373-b64c-800d96ecb72e/boxweave-rewrite-v2. Phase 4 parent: 40a600a56a7fd3a08a0c71b6763bbd7d79f7f621. Exact implementation commit is reported after committing; locate with `git log --format=oneline --grep='feat: retain phase four project binding and receipt slice' rewrite/v2`. Historical phase 3 handoff is archived separately.

## Delivered implementation

- New Python 0.1.0 minimum package with typed target/binding contracts, private SQLite state, revisions, one-use action plans, separate worker lock/intent/receipt and unknown/no-replay recovery.
- Real authenticated Flask/Waitress API and React/TypeScript interface for project binding, exact hello confirmation and receipt observation. No arbitrary Web command. Fixtures are explicitly labeled and cannot silently replace missing live configuration.
- Bounded POSIX MCP stdio adapter with lifecycle, pagination, read-only observation, exact identity verification and fixed hello parsing. Real independent application credential/installed-provider compatibility is unverified.
- Native Windows connection-only PowerShell probe and three no-network rejection tests; no enrollment or registration modification. It never calls overall acceptance passed from connection success.
- Python runtime/test wheel hash locks, npm lock, runnable fixture/live-placeholder examples, README, reproducible commands and desktop/mobile fixture screenshots.

All product implementation was authored inside `/home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a` in the approved non-root Ubuntu LXC, then synchronized with 28 matching source SHA256 values. Old remote worktree at 7818668 and its existing 32 dirty/untracked entries were not modified. Local main/client WIP/archive remain unchanged.

## Actual validation

29 Linux tests passed (exit 0); same 29 passed in a clean offline/hash-locked virtual environment (exit 0). TypeScript/Vite production build and Python compilation passed. Real Chromium passed login/binding/preview/confirm/receipt/API+worker restart/no-replay/mobile-no-overflow with synthetic infrastructure; exactly one hello fixture call. Windows PowerShell 5.1.26100.5651 AST and three no-network preflight cases passed, exit 0. Dependencies checked without inconsistency.

Direct Platform MCP identity/hello returned the approved username, hostname and `hello world`, exit 0. This is **separate** from the fixture Web result and does not establish real Web->worker->MCP E2E. The live prerequisite fails explicitly as MCP_LAUNCHER_UNAVAILABLE. Full commands, stdout, exits, REQ/AC/TASK mappings and limits are in [VERTICAL_SLICE](../verification/VERTICAL_SLICE.md) and [FEATURE_MATRIX](../verification/FEATURE_MATRIX.md).

## Design adjustments and incomplete work

[ADR-005](../design/ADR/0005-phase-four-boundary.md) records the approved loopback fixture HTTP boundary, consolidated minimal binding record and preservation after missing credentials. The architecture's required production identity/HTTPS and real-client/live acceptance are unchanged obligations, not silently waived.

TASK-001 minimum contracts/package and TASK-004 minimum fixture UI are implemented. TASK-002 and TASK-003 remain partial (full reconciliation/DB-full coverage/enrollment/payload trust/real client still absent). TASK-005 is not satisfied. Current fixed operation budget is 60 seconds bounded at 600; no provisioning/create workflow is implemented. No full first-release AC is claimed passed merely from component tests.

The remaining prerequisite is a separately reviewed independent protected application MCP launcher/identity; provide/configure its supported path later without sharing secret text. Then complete the real application/native-client chain and ask the developer to accept architecture and experience. No phase 5 feature work can start before that gate. Tenant/ports/Graph/model wizard and four-profile acceptance remain selected future scope, not completed functionality.

## Cleanup and preservation

Owned temporary API/worker/browser processes were stopped; `ss` showed no listener on 18640. No private-address/public listener, route, policy, container lifecycle, persistent service, provider/Incus upgrade, model request, global trust or real client registration change occurred. Source, dependency artifacts, sanitized test logs and private synthetic state remain in the isolated remote directory for review. No secrets/database/raw old logs entered the commit.

Final source/link/ID/whitespace and repository preservation checks are performed before commit; their actual results are reported below. Stage 4 remains partial regardless of documentation-check success. No phase 5 work was started.

Final document/source validation passed: 35 Markdown files, 71 valid local links, source/risk/ADR references defined, supplied phase 4 prompt byte-identical and archived phase 3 handoff equal to the prior Git blob. All 28 synchronized source files match the remote SHA256 manifest. Tracked whitespace check passed. Private-key/JWT payload pattern scan found none; this narrow scan is not a universal secret detector. Final staging/commit checks cover newly added files as well. These checks do not change the partial/live-gate status above.

## Subsequent dedicated test deployment

User SRC-033 authorized a new dedicated deployment. New boxweave-v2-test / 10.206.30.6 runs source aa0857c with three active/enabled non-root application services. Verified local-CA TLS, login, binding and fixture receipt persistence across API/worker restart, exit 0. No real application MCP integration is claimed. See ../verification/TEST_DEPLOYMENT.md. This later scope permits those persistent test services; old resources were not changed.

Direct Windows access and the dedicated SSH tunnel were unavailable; read-only diagnostics reported missing host shell wrapper, which was not repaired. No host route exists. Proposed HTTPS passthrough on host TCP 18446 remains awaiting explicit confirmation of unfiltered exposure, then actual client connectivity testing. Credentials/public CA are retained in the private ignored test-access directory; no secret was committed. Deployment documentation is uncommitted, and no new product code revision was created by this deployment.

## Approved test access now available (SRC-036/037)

User approved the exact unfiltered host TCP 18446 -> 10.206.30.6:18446 route. Fresh target identity/services/conflict checks passed before creation; route inventory confirms active. The current Windows client verified normal TLS with the supplied public CA, homepage 200, protected API 401, successful login/Secure cookie, fixture resources and existing succeeded receipt, exit 0. Endpoint: https://192.168.2.233:18446. No browser/global root trust was installed, no real application MCP integration is claimed. Private local LOGIN.txt/public test CA/client-health summary remain in ignored test-access; password/session token not printed. The route and requested test services remain running for user review; no cleanup/delete is implied.

## Shared Tenant lab provisioning result

User confirmed the isolated application/MCP/nested workload plan (SRC-038). The first shared create, bw-v2-lab-app with network_tenant boxweave-v2-lab, failed immediately with provider native-Tenant readiness error. Original-name inventory/get reconciliation found no new lab box; no second create or fallback was issued. Current backend reports boxweave.tenant-network/v1 with true creation flags, whereas the reviewed upstream source requires native-network-tenant/v1. Likely compatibility mismatch, exact installed MCP binary unverified. Physical-host changes/upgrades are outside the agreed test scope and were not performed. Existing demo/route remain unchanged. Detailed evidence and next safe attestation/repair boundary: ../verification/SHARED_TENANT_LAB.md. No nested or live application MCP test ran in this attempt.

## Latest disposition: test deferred; do not execute phase 5

SRC-041 explicitly requests marking shared-Tenant/live MCP verification temporarily untested and skipping it, with readiness advice only. Recorded as DEFERRED / NOT EXECUTED; the failed prerequisite create remains factual history, not a passed or completed live test. No further host repair, resource creation or live test was attempted.

Technical next-iteration candidate: TASK-011 basic Prompt Graph, depending only on implemented TASK-004 and not independent MCP credentials. This does not close TASK-005/RISK-017/018 or certify the complete architecture. A future execution request should explicitly confirm acceptance of the demonstrated minimum architecture/slice with the deferred live test. This turn changed documentation only and did not execute phase 5.

## Shared-Tenant retry requested (SRC-042)

The developer resumed this test only. Fresh discovery, original-name reconciliation and one shared create retry again produced the same immediate capability rejection. Post-attempt inventory and exact-name lookup found no lab box. Shared creation remains blocked; nested/live application testing was not executed. No phase 5 implementation, host repair, existing-container changes or new routes occurred. See SHARED_TENANT_LAB for actual evidence.

## Existing shared-Tenant test resumed (SRC-043)

Existing box1/box2 native shared membership and real identities were verified (connect exit 0). One supporting backend installation was submitted in box2 with a 600-second runner timeout. Connect, progress query, get_container and debug_container later returned 180-second MCP transport timeouts. Installation outcome is UNKNOWN, not failed/passed; reconcile /home/box2/boxweave-v2-lab-backend/install.exit and package/job state before rerunning anything. No new application deployment or authenticated inner MCP test has completed. Phase 5 remains unstarted. See SHARED_TENANT_LAB.

## Latest reconciliation (SRC-044)

Platform MCP is responding again; native-network-tenant/v1 is now advertised. Existing box1/box2 share a native Tenant, while boxweave-v2-test remains separate. Existing Box reassignment is unsupported; no regrouping occurred. Original box2 installer exit is now known: 124, with staging-directory failure observed after API reachability; installation is incomplete. This supersedes the prior UNKNOWN installer exit observation. Real v2 MCP testing remains unexecuted; phase 5 has not started.

## Latest same-Tenant resource result (SRC-045)

New bw-v2-mcp-test-container (logical bw-v2-mcp-test, createdAt 1793202811, 10.206.30.7, dev, 2 CPU / 4 GB / 25 GB) was created once in networkTenant boxweave-v2-test, alongside the existing v2 application. Exact metadata, identity exit 0, same attached Guard ACL and app-to-new-box TCP22 connectivity verified. Earlier shared-create contract rejection no longer reproduces in this fresh native-network-tenant/v1 observation; no host repair was performed by this task.

Supporting backend bootstrap reached its 600-second total deadline, install.exit=124, while mandatory Web dependencies were still preparing. APT/dpkg children remained; do not rerun until original state is reconciled. Managed MCP launcher absent; live MCP and prepared loopback HTTP integration harness not executed. Existing demo services active, homepage 200. RISK-017 remains open; RISK-018 has current successful shared-create evidence but no full compatibility certification. New box and dedicated lab artifacts retained, no new public route, no phase 5 implementation. Details in SHARED_TENANT_LAB.

## Read-only installation status follow-up (SRC-046)

User asked whether bw-v2-mcp-test launcher installation was still running. Fresh connect identity passed, exit 0. install.exit remains 124; /usr/local/bin/boxweave-mcp is absent. No apt-get/dpkg/timeout/python3/npm/node installer processes appeared in the selected process snapshot. dpkg.log ends with npm and libc-bin installed; no whole-package audit was run. The remaining preparation process reported FileNotFoundError for its removed temporary staging directory. Installation has stopped and will not automatically continue. Containarium daemon remains active/running with Result=success and ExecMainStatus=0; this does not establish complete launcher installation or authenticated MCP operation. No repair/retry was performed by this status check.

## Latest no-total-deadline retry outcome (SRC-047)

The supporting backend installer was retried once at explicit developer request without an installation total deadline. It completed exit 0; actual managed timers/oneshots and marker checks passed. Independent strict SSH MCP initialize/initialized/tools/list and an authenticated read-only list_containers call passed from the rewritten application container (101 tools).

One inner v2-live-hello creation was rejected; original-name reconciliation and inventory observed no workload. Read-only inner Incus logs confirm its PostgreSQL core instance cannot start: BPF load and fatal proc mount permission failures. No core/host configuration was changed. Launcher availability/read-only authentication is now established for this lab, superseding the earlier missing-launcher observation, but execution and live HTTP worker receipt remain unverified. RISK-019 records this nested startup blocker. No phase 5 work.

## Current developer direction — complete real chain before phase 5 (SRC-048)

The developer requested phase 5 and authorized project boxes, with up to 30 minutes for long installation/preparation. The phase-5 prompt startup gate requires recorded developer architecture/slice acceptance. Asked explicitly whether to accept the demonstrated slice with deferred live execution and start TASK-011, the developer selected: complete the real execution chain first, then start phase 5. Therefore no phase-5 task/feature was started, no gate waiver was inferred, and the current work remains real phase-4 integration preparation.

Read-only source diagnosis found the current failed test box had enable_podman=false. The reviewed provider source maps EnablePodman to EnableNesting; nesting adds security.nesting and syscall intercepts. Its server policy can also map enable_podman to privileged mode and an unconfined AppArmor label for administrators. This is a candidate explanation/alternative, not attestation of all active outer policy. Existing child label is unconfined but namespace stacking is reported, so that label does not establish complete parent permissions. No source-based security workaround was applied.

A concrete alternative was presented for explicit approval: new bw-v2-mcp-nested, 2 CPU / 4 GB / 30 GB, same boxweave-v2-test Tenant, Podman/nesting enabled, possible privileged/unconfined behavior disclosed, installation preparation up to 30 minutes, no new public port, no existing-box/host setting changes. Approval is pending; no create for that name was submitted. Do not treat elapsed time or the general box-creation authorization as approval of the disclosed isolation change.

## Current result — real chain verified (SRC-049/050)

Version 0.1.1 fixes a real queue-claim timing race. Thirty Linux tests pass, including a new regression that failed before the fix. An explicitly approved nesting-enabled backend now supports the real application -> independent managed MCP -> inner workload path. HTTP and Chromium binding/confirm/hello/receipt/restart/no-replay plus live trust/identity rejection passed. See [LIVE_MCP_CHAIN](../verification/LIVE_MCP_CHAIN.md) for exact identities, operation IDs, versions, commands and limits.

The previous missing-launcher/nested-startup blocker is resolved for this new lab only, not the old box or every LXC. TASK-005 selected Web path has live evidence; real Windows-native enrollment/Agent GUI acceptance remains incomplete. Developer required this real chain before phase 5 (SRC-048); no phase 5 feature was started and architecture/slice acceptance must still be recorded before TASK-011 or other new features. Original published demo remains fixture; isolated test servers are stopped, resources retained, no new public port or host-security change.

Final validation for SRC-049/050: 30 Linux tests passed in the assigned project LXC; real HTTP and final Chromium live-MCP checks passed; live trust/identity negative paths passed; 38 Markdown documents' local links and the four changed source hashes match. Original legacy HEAD d8a09dd/client source hash/archive status are unchanged by this task. Ignored screenshots/state remain outside Git. The real-chain repair and related accumulated test/deployment records are committed on rewrite/v2; final commit identity is reported by Git/final response. No phase5 feature or push occurred.
