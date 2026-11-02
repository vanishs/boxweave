# Code-first feature and acceptance audit — 2026-11-01

**Source baseline:** [main@`ca4898c26496`](https://github.com/vanishs/boxweave/commit/ca4898c26496a2618ac28bf0318175791307b564), v0.25.0. Code, not historical feature prose, determines what is inventory-listed. This is a dated baseline; an inventory token does not prove that a route works.

**Current acceptance supersession:** See the [post-v0.26.2 A01–A33 reacceptance](REACCEPTANCE_0262_TO_0264_2026-11-01.md) for the latest observed real Box evidence.  This document preserves a historical source/CI inventory and earlier scoped run. Read [the full A01–A30 live acceptance ledger](CODE_FIRST_LIVE_ACCEPTANCE_2026-11-01.md) for the 2026-11-01 new Box v0.26.0 fresh install, 0.24.0 and 0.26.0 upgrades to v0.26.1, real nested failure, network packet test, client generation, source-only fixes and explicit blocked/pending categories. Do not treat the historical claim "SSH blocked" as universal after direct guest-IP execution was verified.

## Actually executed evidence

- **T / PASS for source and fixtures:** GitHub Actions [run 41340089030](https://github.com/vanishs/boxweave/actions/runs/41340089030), exact baseline SHA above: verify and publish both succeeded. Document/metadata validation, shell syntax, unittest discovery, client generator grammar, frontend build, seven fixture browser flows, release ZIP membership/digest checks and Python 3.14 wheel smoke all succeeded. No source/fixture failure remains from this run.
- **L / bounded live read:** `boxweave_info` returned static guide source version 0.25.0; `list_containers` returned four visible running Boxes and `get_container` returned the two dedicated tests. `boxweave_diagnose(source=gateway,recent=5)` returned `state=observed`, `coverage=gateway_only`, five classified events (four successes, one `connect` outcome `unknown`). No raw arguments or private guest logs. This is a **gateway read PASS**, not installed-app, SSH, journal restart or durable retention acceptance.
- **L / D4 negative PASS:** `boxweave_project_diagnose(box=boxweave-test-cli-container,check=ports)` was denied with `PROJECT_DIAG_NOT_AUTHORIZED`. This demonstrates effective denial for that one unapproved Box/check, not all D4 policy/identity cases.
- **L / BLOCKED SSH:** `connect(box=boxweave-test-cli-container,exec='id -un && hostname && pwd')` returned `TRUST_OR_WORKLOAD_KEY_GATE`. Both test Box `debug_container` reads independently identified an absent host-user shell `/usr/local/bin/containarium-shell` on backend `dev`, daemon 0.120.1. Do not overwrite the original target or silently fix Provider host SSH under repository authorization.
- **U:** no current 0.25.0 real installed upgrade, end-to-end SSH/Python command, external desktop Agent, actual public TLS, reboot, privileged nesting or mutating Provider test was run in this audit. No general host shell or trusted checkout was available; code CI evidence does not close these gates.

**Evidence states:** S = source implemented; T = tests actually executed for an exact revision; L = authenticated live, scoped behavior. PASS requires the specified claim's observed success, FAIL an observed failure, PARTIAL mixed proof, BLOCKED a verified obstacle, UNVERIFIED no suitable run. An older release live success never proves the newer installed release.

## Source-backed feature-to-test-to-acceptance matrix

| ID | Implemented surface (not future design) | Source and test mapping | 0.25.0 source CI | Current end-to-end acceptance |
| --- | --- | --- | --- | --- |
| A01 | Install, repair, managed Provider adoption | online/bootstrap/distribution; test_online, test_host_bootstrap, test_distribution | CI PASS | PARTIAL: old reused install; clean first-host/current runtime U |
| A02 | Packaging, wheelhouse, digest and DB upgrades | distribution/deploy_check; test_distribution, test_platform_minimums | CI PASS | PARTIAL: 0.25.0 immutable published artifacts; host upgrade U |
| A03 | Host CLI health, status, doctor, report | host_cli/manage; test_host_cli, test_diagnostic_report | CI PASS | PARTIAL: older installed CLI; current 0.25.0 U |
| A04 | Client script enrollment, nine Apps, OS/SSH transports | client_profiles/client_setup; test_client_setup, test_profiles, test_public_profiles | CI PASS | PARTIAL: old ZCode read; current 9-App/OS matrix U |
| A05 | SSH pin, managed key, exact identity, Python execution | mcp_gate/hostkeys; test_mcp_gate_policy, test_workload_guidance | CI PASS | BLOCKED: real connect trust gate; backend login shell missing; Python exit unavailable |
| A06 | MCP discovery, capability catalog, host-wide allowlist | mcp/mcp_gate/capabilities; test_mcp_info, test_mcp_gate_policy | CI PASS | PARTIAL: info tool read works; 0.25.0 authenticated SSH chain U |
| A07 | Independent nesting helper and create extension | nesting/mcp_gate; test_nesting | CI PASS | BLOCKED: live nesting inspect failed NESTING_HELPER_FAILED_INSPECT_ORIGINAL_INSTANCE; privilege path U |
| A08 | Box lifecycle and reviewed action plans | lifecycle/box_actions/provider; test_lifecycle, test_box_actions | CI PASS | PARTIAL: direct Provider create/stop/start/resize proven 2026-11-01; application receipt path U |
| A09 | Auto-sleep and resource/capability observation | auto_sleep/capabilities/box_debug; test_capabilities, test_box_debug | CI PASS | PARTIAL: direct 2026-11-01 enable/disable ack; current effective idle stop U |
| A10 | Jobs/receipts/reconciliation/no replay | jobs/store/lifecycle; test_jobs, test_recovery, test_history | CI PASS | PARTIAL: fixture crash path; current live unknown recovery U |
| A11 | Web auth, CSRF, operator session | web/access; test_access, test_slice | CI PASS | PARTIAL: browser fixture; current installed browser U |
| A12 | Project binding, fixed hello, environment plans | service/lifecycle/web; test_bindings, test_slice, test_lifecycle | CI PASS | PARTIAL: fixture; actual SSH fixed hello blocked |
| A13 | Native Tenant policy, network and route plans | access/access_provider; test_access | CI PASS | PARTIAL: older native packet evidence; current isolation U |
| A14 | Prompt Graph, conflict control, export | graphs/PromptGraph; test_graphs | CI PASS | PARTIAL: fixture browser; current deployed Web U |
| A15 | Model wizard and native handoff | models/ModelWizard; test_models | CI PASS | PARTIAL: fixture and prior handoff; inference unclaimed |
| A16 | agent-box MCP file/resource read-only adapter | agent_box/MCPPanel; test_agent_box_adapter, test_mcp_resources | CI PASS | PARTIAL: real 0.24.0 nested read; 0.25.0 U |
| A17 | Remote MCP HTTP initialize, tools, protocol denials | remote_mcp/mcp_gate; test_remote_mcp | CI PASS | PARTIAL: prior 0.24.0 loopback; current public client U |
| A18 | OAuth PKCE, DCR, Bearer, URL link and revocation | remote_auth/remote_mcp; test_remote_mcp | CI PASS | PARTIAL: previous loopback OAuth success; external matrix U |
| A19 | Proxy/TLS/Caddy and remote Web enrollment | remote_tls/RemoteMCP; test_remote_mcp | CI PASS | PARTIAL: prior loopback TLS; public renewal, callback U |
| A20 | D0 host diagnostics; D1/D2 events/report, actor scope | diagnostics/host_cli; test_diagnostics, test_diagnostic_report | CI PASS | PARTIAL: prior 0.22.0 live; 0.24.5 actor change installed U |
| A21 | D3 gateway journal and app receipt query | gate_diagnostics/mcp_gate; test_gate_diagnostics | CI PASS | PARTIAL: live gateway classified read; app/restart/retention U |
| A22 | D4 default-deny, exact authorized Box diagnosis | gate_diagnostics/mcp_gate; test_gate_diagnostics | CI PASS | BLOCKED: default-deny observed; allowlisted live positive U |
| A23 | Selftest/worker fixed test and report paths | selftest/jobs; test_selftest | CI PASS | UNVERIFIED: real selftest mutates resources; no authority |
| A24 | Browser integration and real desktop client UI | web/src/*, scripts/*_browser_check.py | CI PASS, 7 fixture flows | PARTIAL: exact 0.25.0 installed client/browser U |

Code-first additions not fully represented in the older F-001–F-015 narrative include: direct `boxweave_info` and gateway policy, `boxweave_nesting`, `boxweave_diagnose` and `boxweave_project_diagnose`; agent-box Web read/resource endpoints; background worker, receipts and selftest; hosted CLI report output; remote MCP token and URL credential paths; nine-App portable Python client generator; and release manifest/wheel checks. These are **implemented entrypoints**, not newly claimed full acceptances.

## Source-derived inventory snapshots

The lists enumerate all Python app modules, Python unittest files, client helper scripts, support scripts, frontend sources, application/host CLI action names, Web and remote HTTP routes, gateway-owned MCP tools, client catalog and frontend views. Downstream Provider tool names are dynamic and are **not** mistaken for locally implemented BoxWeave tools. Each source-backed entry must be tested at the layer its acceptance claim requires.

### Python application modules (40)

<!-- inventory:python-modules:start -->
- `bwmini/__init__.py`
- `bwmini/__main__.py`
- `bwmini/access.py`
- `bwmini/access_provider.py`
- `bwmini/agent_box.py`
- `bwmini/auto_sleep.py`
- `bwmini/bootstrap.py`
- `bwmini/box_actions.py`
- `bwmini/box_debug.py`
- `bwmini/capabilities.py`
- `bwmini/client_profiles.py`
- `bwmini/client_setup.py`
- `bwmini/coordination_runtime.py`
- `bwmini/deploy_check.py`
- `bwmini/diagnostics.py`
- `bwmini/distribution.py`
- `bwmini/domain.py`
- `bwmini/gate_diagnostics.py`
- `bwmini/graphs.py`
- `bwmini/host_cli.py`
- `bwmini/http_manage.py`
- `bwmini/hostkeys.py`
- `bwmini/jobs.py`
- `bwmini/lifecycle.py`
- `bwmini/manage.py`
- `bwmini/mcp.py`
- `bwmini/mcp_gate.py`
- `bwmini/models.py`
- `bwmini/nesting.py`
- `bwmini/online.py`
- `bwmini/operation_coordination.py`
- `bwmini/profiles.py`
- `bwmini/provider.py`
- `bwmini/remote_auth.py`
- `bwmini/remote_mcp.py`
- `bwmini/remote_tls.py`
- `bwmini/selftest.py`
- `bwmini/service.py`
- `bwmini/store.py`
- `bwmini/web.py`
<!-- inventory:python-modules:end -->

### Python unittest files (44)

<!-- inventory:unit-tests:start -->
- `tests/test_access.py`
- `tests/test_agent_box_adapter.py`
- `tests/test_agent_source_bundle.py`
- `tests/test_bindings.py`
- `tests/test_bootstrap.py`
- `tests/test_box_actions.py`
- `tests/test_box_debug.py`
- `tests/test_capabilities.py`
- `tests/test_client_setup.py`
- `tests/test_coordination_runtime.py`
- `tests/test_diagnostic_golden.py`
- `tests/test_diagnostic_report.py`
- `tests/test_diagnostics.py`
- `tests/test_distribution.py`
- `tests/test_enrollment.py`
- `tests/test_gate_diagnostics.py`
- `tests/test_graphs.py`
- `tests/test_history.py`
- `tests/test_host_bootstrap.py`
- `tests/test_http_manage.py`
- `tests/test_host_cli.py`
- `tests/test_jobs.py`
- `tests/test_knowledge.py`
- `tests/test_lifecycle.py`
- `tests/test_manage.py`
- `tests/test_mcp_gate_policy.py`
- `tests/test_mcp_info.py`
- `tests/test_mcp_resources.py`
- `tests/test_models.py`
- `tests/test_nesting.py`
- `tests/test_online.py`
- `tests/test_online_adoption.py`
- `tests/test_operation_coordination.py`
- `tests/test_platform_minimums.py`
- `tests/test_portable_codex_merge.py`
- `tests/test_profiles.py`
- `tests/test_public_profiles.py`
- `tests/test_recovery.py`
- `tests/test_release_metadata.py`
- `tests/test_remote_mcp.py`
- `tests/test_selftest.py`
- `tests/test_slice.py`
- `tests/test_ssh_keepalive.py`
- `tests/test_workload_guidance.py`
<!-- inventory:unit-tests:end -->

### Standalone client code helpers (7)

<!-- inventory:client-scripts:start -->
- `clients/profile_client.py`
- `clients/windows/bootstrap-native.ps1`
- `clients/windows/diagnose-connection.ps1`
- `clients/windows/diagnose-native.ps1`
- `clients/windows/prepare-identity.ps1`
- `clients/windows/register-native.ps1`
- `clients/wsl/prepare_identity.py`
<!-- inventory:client-scripts:end -->

### Project scripts and live-test drivers (23)

<!-- inventory:support-scripts:start -->
- `scripts/access_browser_check.py`
- `scripts/agent_source_bundle.py`
- `scripts/authorize_native_key.py`
- `scripts/binding_browser_check.py`
- `scripts/browser_check.py`
- `scripts/build_release_manifest.py`
- `scripts/check_knowledge.py`
- `scripts/compare_local_receipts.py`
- `scripts/finish_access_check.py`
- `scripts/graph_browser_check.py`
- `scripts/history_browser_check.py`
- `scripts/lifecycle_browser_check.py`
- `scripts/live_access_check.py`
- `scripts/live_access_existing_check.py`
- `scripts/live_failure_check.py`
- `scripts/live_lifecycle_check.py`
- `scripts/live_slice_check.py`
- `scripts/model_browser_check.py`
- `scripts/package_live_check.py`
- `scripts/phase6_install_check.py`
- `scripts/release_profiles_check.py`
- `scripts/validate.py`
- `scripts/windows_profile_check.py`
<!-- inventory:support-scripts:end -->

### Frontend sources (9)

<!-- inventory:web-sources:start -->
- `web/src/AccessPanel.tsx`
- `web/src/ClientSetup.tsx`
- `web/src/Environments.tsx`
- `web/src/MCPPanel.tsx`
- `web/src/ModelWizard.tsx`
- `web/src/PromptGraph.tsx`
- `web/src/RemoteMCP.tsx`
- `web/src/main.tsx`
- `web/src/style.css`
<!-- inventory:web-sources:end -->

### Application CLI actions (36)

<!-- inventory:app-cli:start -->
- `access`
- `access-confirm`
- `access-memberships`
- `access-network-plan`
- `access-observe`
- `access-port-plan`
- `access-receipt`
- `access-receipts`
- `capabilities`
- `diagnose`
- `environment-confirm`
- `environment-observe`
- `environment-plan`
- `environment-receipt`
- `environment-receipts`
- `environments`
- `history`
- `init-admin`
- `job-receipts`
- `job-run`
- `model-confirm`
- `model-handoff`
- `model-select`
- `model-wizard`
- `projects`
- `receipts`
- `reconcile`
- `remote-caddyfile`
- `remote-https`
- `remote-mcp`
- `reset-admin`
- `selftest-plan`
- `selftest-run`
- `serve`
- `tenant-label`
- `worker`
<!-- inventory:app-cli:end -->

### Host CLI command identifiers (7)

<!-- inventory:host-cli:start -->
- `box`
- `client`
- `doctor`
- `provenance`
- `report`
- `status`
- `version`
<!-- inventory:host-cli:end -->

The direct `boxweave http` dispatcher is implemented as an early safe host
command branch rather than an argparse `add_parser` entry; it delegates managed
configuration and service control to the non-root application account.

### Local Web API routes (method and path) (56)

<!-- inventory:web-api:start -->
- `DELETE /api/graphs/<gid>`
- `DELETE /api/projects/<pid>/binding`
- `DELETE /api/remote-mcp/credentials/<ident>`
- `GET /`
- `GET /api/access`
- `GET /api/access-operations`
- `GET /api/access-operations/<oid>`
- `GET /api/access/memberships`
- `GET /api/box-action-receipts`
- `GET /api/box-action-receipts/<rid>`
- `GET /api/boxes/<logical>/agent-box`
- `GET /api/boxes/<logical>/agent-box/resource`
- `GET /api/boxes/<logical>/debug-summary`
- `GET /api/environment-operations`
- `GET /api/environment-operations/<oid>`
- `GET /api/environments`
- `GET /api/graphs`
- `GET /api/graphs/<gid>`
- `GET /api/graphs/<gid>/markdown`
- `GET /api/mcp-capabilities`
- `GET /api/model-wizard`
- `GET /api/model-wizard/handoff`
- `GET /api/operation-history`
- `GET /api/operations`
- `GET /api/operations/<oid>`
- `GET /api/operations/<oid>/observations`
- `GET /api/projects`
- `GET /api/remote-mcp`
- `GET /api/resources`
- `GET /api/session`
- `GET /assets/<path:asset>`
- `POST /api/access-operations/<oid>/observe`
- `POST /api/access-plans/<pid>/confirm`
- `POST /api/access-plans/network`
- `POST /api/access-plans/port`
- `POST /api/box-action-plans`
- `POST /api/box-action-plans/<pid>/confirm`
- `POST /api/boxes/<logical>/agent-box/read`
- `POST /api/client-profile`
- `POST /api/environment-operations/<oid>/observe`
- `POST /api/environment-plans`
- `POST /api/environment-plans/<pid>/confirm`
- `POST /api/graphs`
- `POST /api/graphs/<gid>/copies`
- `POST /api/login`
- `POST /api/logout`
- `POST /api/model-wizard/confirm`
- `POST /api/operations/<oid>/reconcile`
- `POST /api/plans`
- `POST /api/plans/<pid>/confirm`
- `POST /api/projects`
- `POST /api/remote-mcp/credentials`
- `PUT /api/graphs/<gid>`
- `PUT /api/model-wizard`
- `PUT /api/projects/<pid>`
- `PUT /api/tenants/<key>/label`
<!-- inventory:web-api:end -->

### Remote HTTP MCP/OAuth routes (method and path) (10)

<!-- inventory:remote-api:start -->
- `GET /.well-known/oauth-authorization-server`
- `GET /.well-known/oauth-protected-resource`
- `GET /.well-known/oauth-protected-resource/mcp`
- `GET /`
- `GET /authorize`
- `GET,POST,DELETE /mcp`
- `GET,POST,DELETE /mcp/link/<link>`
- `POST /authorize`
- `POST /register`
- `POST /token`
<!-- inventory:remote-api:end -->

### Gateway-owned MCP tools (4)

<!-- inventory:gateway-tools:start -->
- `boxweave_diagnose`
- `boxweave_info`
- `boxweave_nesting`
- `boxweave_project_diagnose`
<!-- inventory:gateway-tools:end -->

### Client App catalog (9)

<!-- inventory:client-apps:start -->
- `antigravity`
- `claude`
- `codex`
- `copilot`
- `cursor`
- `opencode`
- `qoder`
- `trae`
- `zcode`
<!-- inventory:client-apps:end -->

### Frontend views (8)

<!-- inventory:web-views:start -->
- `access`
- `clients`
- `environments`
- `graphs`
- `mcp`
- `models`
- `projects`
- `remote`
<!-- inventory:web-views:end -->

## Missing acceptance / safe closure order

1. **P1 BLOCKED:** ask the backend operator to investigate and explicitly authorize repair of the missing `containarium-shell` on `dev`; confirm account/target/SSH trust before rerunning the original Box's fixed `whoami`/Python hello and exit 0. No duplicate create, trust bypass or automated host repair.
2. **P1 U:** on a specifically authorized reused test Box, verify published v0.25.0 package digest, installed/running app+worker+gateway versions; real SSH/HTTP MCP initialize/list/read; D3 app query/fault/restart/retention; D4 default denial and approved exact Box/check positive. Record command, exit, actor and target for every result.
3. **P1 U:** separate clean Ubuntu 24.04/26.04 install/reinstall/reboot, cross-platform client App enrollment, genuine external HTTPS/OAuth client and split-host nesting tests. Prior local TLS and historical nested execution do not close these gates.
4. **P2 U:** fill remaining rows selectively from the actual code surface, including negative/authority/timeout/no-replay and data-preservation tests as applicable. Preserve existing state/evidence and distinguish fixture from live results.

**Excluded, not falsely passed:** E0–E4 Box snapshot/fingerprint/restore/clone proposal has not shipped; no arbitrary guest command/debugger, per-Agent RBAC, public Issue submission or universal provider/platform certification is implied.

Related records: [feature matrix](FEATURE_MATRIX.md), [test queue](REMEDIATION_TEST_QUEUE.md), [risks](OPEN_RISKS.md), [acceptance requirements](../product/ACCEPTANCE.md).

## Live MCP acceptance supplement — test Box created after baseline

The owner subsequently authorized all BoxWeave MCP commands against test Boxes. See [executed command/response matrix in the queue](REMEDIATION_TEST_QUEUE.md) for the full record. **Direct Provider L PASS:** create `bwaccept-20261101-container` (creation ID `1793567247`) with CPU2/RAM2GB/20GB, verify RUNNING and native Tenant `boxweave-test`; resize CPU3/RAM3GB and verify readback; stop and verify STOPPED; start and verify RUNNING with original identity. `get_metrics` returned data. Auto-sleep enable and disable were acknowledged but effective idle stop remains U. Test Box retained RUNNING, auto-sleep disabled, for next SSH test.

**Independent live negative observations:** D4 project ports request denied with `PROJECT_DIAG_NOT_AUTHORIZED`; nesting inspect failed `NESTING_HELPER_FAILED_INSPECT_ORIGINAL_INSTANCE`; connect/exec failed `TRUST_OR_WORKLOAD_KEY_GATE` and did not return Python output or exit. `debug_container` reported missing `/usr/local/bin/containarium-shell` for both old and new Box host users. Source `mcp_gate.trusted` also validates a fresh `targets.json` registration (45-second observation age), so the generic trust-gate code **does not prove which check rejected connect**; the missing backend login shell is a separately confirmed obstacle to ordinary SSH. No host shell, SSH trust, nesting helper, application installation, or unrelated Box was modified. Different inner BoxWeave versions cannot be compared until an actual guest execution path is established.

This live Provider evidence upgrades only the **direct API operation slice**, not the application-side BoxWeave v0.25.0 lifecycle/receipts, reboot persistence, network Guard enforcement, nested child execution or universal user-facing acceptance.
