# RACC-0461-20261105-C | Installed action-by-action reacceptance ledger

**Date:** 2026-11-05 (America/Los_Angeles). **Scope:** current outer Gateway + official installed v0.46.1 local-stdio inventory; isolated host is Containarium 0.128.0. This ledger is **NOT** a blanket PASS. All verification is same-Agent. A live installed local stdio MCP initialize/tools/list returned 68 names; direct per-method installed client calls were boxweave_info and get_container only; the per-action calls in this table are predominantly from the outer MCP ingress. Actual SSH wire and authenticated HTTP MCP positive paths remain unverified.

Statuses: PASS is only scoped to tested predicates; PARTIAL denotes incomplete positive/invalid/denial triad; FAIL denotes observed mismatch; BLOCKED denotes exact environment/authority constraint. Discovery alone does not count as a test.

## 68 MCP actions

| ID | Tool / mapped feature | Actual adversarial attempt and result | Oracle / limitation | State |
|---|---|---|---|---|
| MCP-001 | create_container / F07 | outer-write-batch1.json:ERR UNAVAILABLE: Error code: UNAVAILABLE; Error: McpServerError: Tool execution failed; outer-write-batch1.json:ERR UNAVAILABLE: Error code: UNAVAILABLE; Error: McpSe | New owned Box 1 CPU/2GB/20GB, UUID and createdAt; duplicate and CPU=0 returned 500 (#81); evidence outer-write-batch1.json | **PARTIAL** |
| MCP-002 | list_containers / F06 | Outer tool listing observed 29 Boxes; installed stdio catalog 68 entries | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-003 | get_container / F06 | outer-read-batch2.json:response; outer-read-batch2.json:ERR UNAVAILABLE: Error code: UNAVAILABLE; Error: McpServerError: Tool execution failed:; outer-write-batch1.json:response; outer-write | Absent Box incorrectly HTTP 500 (#81); evidence outer-lifecycle-batch3.json,outer-read-batch2.json,outer-write-batch1.json,outer-write-batch2.json | **FAIL** |
| MCP-004 | debug_container / X03 | outer-read-batch2.json:response; outer-read-batch2.json:response | Existing+absent target, typed missing; evidence outer-read-batch2.json | **PASS scoped** |
| MCP-005 | move_container / F07 | NOT INVOKED; authority/dependency gate | Single backend; no approved peer; evidence not captured | **BLOCKED** |
| MCP-006 | resize_container / F08 | outer-write-batch1.json:response; outer-write-batch2.json:ERR INVALID_ARGUMENT: Error code: INVALID_ARGUMENT; Error: McpServerError: RESOURCE_DI; outer-write-batch2.json:ERR INVALID_ARGUMENT | 2 CPU/3GB and restore: Provider+guest cgroup; invalid CPU and shrink refused; evidence outer-write-batch1.json,outer-write-batch2.json | **PASS scoped** |
| MCP-007 | toggle_monitoring / F17 | outer-write-batch2.json:ERR UNAVAILABLE: Error code: UNAVAILABLE; Error: McpServerError: Tool execution failed | No OTel collector; 400 prerequisite refusal; evidence outer-write-batch2.json | **BLOCKED** |
| MCP-008 | set_metrics_export / F17 | NOT INVOKED; authority/dependency gate | Host-cloud monitoring mutation forbidden; evidence not captured | **BLOCKED** |
| MCP-009 | get_metrics_export / F17 | cloud monitoring export disabled | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-010 | delete_container / F07 | NOT INVOKED; authority/dependency gate | Preserve owned failure scene; evidence not captured | **BLOCKED** |
| MCP-011 | start_container / F07 | outer-lifecycle-batch3.json:response | STOPPED to RUNNING, guest marker survived; waitForReady=true untested; evidence outer-lifecycle-batch3.json | **PARTIAL** |
| MCP-012 | toggle_auto_sleep / F16 | outer-write-batch2.json:response; outer-write-batch2.json:response | Enabled 120m then disabled; actual timer/wake not observed; evidence outer-write-batch2.json | **PARTIAL** |
| MCP-013 | stop_container / F07 | outer-lifecycle-batch3.json:response | RUNNING to STOPPED with separate get_container; denial untested; evidence outer-lifecycle-batch3.json | **PARTIAL** |
| MCP-014 | get_metrics / F17 | Own Box metrics positive; absent Box HTTP500 | Absent Box incorrectly HTTP 500 (#81); evidence not captured | **FAIL** |
| MCP-015 | get_system_info / F05 | Outer read: Containarium v0.128.0, Incus 7.5.1 | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-016 | check_for_updates / F05 | Read returns latest unknown | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-017 | upgrade_backend / F05 | NOT INVOKED; authority/dependency gate | Host upgrade forbidden; evidence not captured | **BLOCKED** |
| MCP-018 | get_upgrade_status / F05 | outer-read-batch2.json:response | Only unknown upgrade ID tested; live upgrade forbidden; evidence outer-read-batch2.json | **BLOCKED** |
| MCP-019 | list_backends / F05 | One healthy backend dev returned | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-020 | get_backend / F05 | dev positive; unknown backend denied | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-021 | backend_validate_gpu / X10 | NOT INVOKED; authority/dependency gate | No NVIDIA GPU; host-level test excluded; evidence not captured | **BLOCKED** |
| MCP-022 | security_findings / X08 | Owned Box returned zero findings | Positive + malformed + denial triad not independently completed; evidence outer-read-batch2.json | **PARTIAL** |
| MCP-023 | security_sentry_status / X08 | DISABLED because unset environment | Sentry disabled by environment; evidence not captured | **PARTIAL** |
| MCP-024 | list_bad_destinations / X08 | NOT INVOKED; requires next-Agent triad | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-025 | list_security_sentry_findings / X08 | Read returns HTTP503 finding store unavailable | Finding store unavailable (503); evidence outer-read-batch2.json | **BLOCKED** |
| MCP-026 | sync_ssh_config / F27 | NOT INVOKED; authority/dependency gate | Writes host SSH config, excluded; evidence not captured | **BLOCKED** |
| MCP-027 | code_runs / X09 | Owned Box returned no active code runs | No code runs on owned Box; evidence outer-read-batch2.json | **PARTIAL** |
| MCP-028 | code_logs / X09 | Known absent run typed 404 | No owned code run; absent run rejected; evidence outer-read-batch2.json | **BLOCKED** |
| MCP-029 | code_pending / X09 | Known absent run typed 404 | No owned code run; absent run rejected; evidence outer-read-batch2.json | **BLOCKED** |
| MCP-030 | connect / F11 | outer-lifecycle-batch3.json:guest exit 17; outer-lifecycle-batch3.json:ERR INVALID_ARGUMENT: Error code: INVALID_ARGUMENT; Error: McpServerError: TARGET_; outer-lifecycle-batch3.json:guest e | Guest exit 0 and 17; host override refused; marker survives restart; evidence outer-lifecycle-batch3.json | **PASS scoped** |
| MCP-031 | list_routes / F15 | Zero active HTTP routes | No HTTP routes; no test-owned publish; evidence not captured | **PARTIAL** |
| MCP-032 | network_guard_status / F14 | Both tenant guards reported enforce; no TCP oracle | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-033 | delete_route / F15 | NOT INVOKED; authority/dependency gate | No owned external route; host route excluded; evidence not captured | **BLOCKED** |
| MCP-034 | add_collaborator / F14 | NOT INVOKED; authority/dependency gate | SSH/jump account grant excluded; evidence not captured | **BLOCKED** |
| MCP-035 | list_collaborators / F14 | Owned Box returned zero collaborators | Empty list on owned Box; evidence outer-read-batch2.json | **PARTIAL** |
| MCP-036 | remove_collaborator / F14 | NOT INVOKED; authority/dependency gate | No approved collaborator; SSH trust change excluded; evidence not captured | **BLOCKED** |
| MCP-037 | expose_port / F15 | NOT INVOKED; authority/dependency gate | Public TLS/route mutation excluded; evidence not captured | **BLOCKED** |
| MCP-038 | list_passthrough_routes / F15 | NOT INVOKED; requires next-Agent triad | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-039 | add_passthrough_route / F15 | NOT INVOKED; authority/dependency gate | Host port exposure excluded; evidence not captured | **BLOCKED** |
| MCP-040 | delete_passthrough_route / F15 | NOT INVOKED; authority/dependency gate | No task-owned passthrough route; evidence not captured | **BLOCKED** |
| MCP-041 | list_recipes / X10 | Recipe IDs enumerated; no deployment | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-042 | deploy_recipe / X10 | NOT INVOKED; authority/dependency gate | GPU and public port/image dependencies unapproved; evidence not captured | **BLOCKED** |
| MCP-043 | compose_discover / X06 | Live Box with actual compose.yaml returned HTTP 404 | Existing compose.yaml returns HTTP 404 (#79); evidence outer-read-batch2.json | **FAIL** |
| MCP-044 | compose_enable / X06 | Actual compose directory and absent directory both returned HTTP 404 | Existing compose directory returns HTTP 404 (#79); evidence outer-compose-batch.json | **FAIL** |
| MCP-045 | compose_disable / X06 | Actual compose directory returned HTTP 404 | Existing compose directory returns HTTP 404 (#79); evidence outer-compose-batch.json | **FAIL** |
| MCP-046 | compose_status / X06 | Actual compose directory returned HTTP 404 | Existing compose directory returns HTTP 404 (#79); evidence outer-compose-batch.json,outer-read-batch2.json | **FAIL** |
| MCP-047 | provision_runners / X07 | NOT INVOKED; authority/dependency gate | No runner PAT or approved GitHub registration; evidence not captured | **BLOCKED** |
| MCP-048 | list_runners / X07 | NOT INVOKED; authority/dependency gate | GitHub PAT unavailable; evidence not captured | **BLOCKED** |
| MCP-049 | remove_runner / X07 | NOT INVOKED; authority/dependency gate | No owned runner and destructive deregistration; evidence not captured | **BLOCKED** |
| MCP-050 | guardrail_policy_get / F14 | Not configured | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-051 | set_ssh_keys / F11 | NOT INVOKED; authority/dependency gate | SSH trust change excluded; evidence not captured | **BLOCKED** |
| MCP-052 | security_process_policy_get / X08 | NOT INVOKED; requires next-Agent triad | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-053 | list_miner_signatures / X08 | NOT INVOKED; requires next-Agent triad | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-054 | terminate_user_sessions / F14 | NOT INVOKED; authority/dependency gate | Would end host/jump sessions; evidence not captured | **BLOCKED** |
| MCP-055 | network_tenant_capabilities / F14 | NOT INVOKED; requires next-Agent triad | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-056 | list_network_policies / F14 | NOT INVOKED; requires next-Agent triad | Positive + malformed + denial triad not independently completed; evidence not captured | **PARTIAL** |
| MCP-057 | get_network_policy / F14 | Two known selectors returned typed 404 | No policy configured; positive unavailable; evidence not captured | **BLOCKED** |
| MCP-058 | patch_tenant_access / F14 | NOT INVOKED; authority/dependency gate | Host Tenant ACL change excluded; evidence not captured | **BLOCKED** |
| MCP-059 | boxweave_nesting / F13 | outer-read-batch3.json:response; outer-read-batch3.json:ERR INVALID_ARGUMENT: Error code: INVALID_ARGUMENT; Error: McpServerError: TARGET_READ_; outer-lifecycle-batch3.json:response; outer-l | inspect/plan true; stale confirmation TARGET_CHANGED; no child proof; evidence outer-lifecycle-batch3.json,outer-read-batch3.json | **PARTIAL** |
| MCP-060 | boxweave_info / F24 | Outer read and installed local stdio tools/call succeeded | Positive + malformed + denial triad not independently completed; evidence inner-mcp-local-stdio-responses.jsonl | **PARTIAL** |
| MCP-061 | boxweave_diagnose / F37 | outer-read-batch3.json:response; outer-read-batch3.json:response; outer-read-batch3.json:ERR INVALID_ARGUMENT: Error code: INVALID_ARGUMENT; Error: McpServerError: DIAGNOSTIC_S | compat subset matched five tools, gateway events; app selector denied; evidence outer-read-batch3.json | **PARTIAL** |
| MCP-062 | boxweave_project_diagnose / F37 | outer-read-batch3.json:ERR INVALID_ARGUMENT: Error code: INVALID_ARGUMENT; Error: McpServerError: PROJECT_DIAG | Default-deny until root allowlist; evidence outer-read-batch3.json | **BLOCKED** |
| MCP-063 | boxweave_diagnostic_run / F36 | outer-read-batch3.json:response; outer-read-batch3.json:response; outer-read-batch3.json:ERR INVALID_ARGUMENT: Error code: INVALID_ARGUMENT; Error: McpServerError: COORDINATOR_ | preflight and rest_read PASS, invalid trace refused; evidence outer-read-batch3.json | **PARTIAL** |
| MCP-064 | boxweave_box_read_file / F12 | outer-read-batch3.json:ERR INVALID_ARGUMENT: Error code: INVALID_ARGUMENT; Error: McpServerError: BOX_PATH_INV; outer-read-batch3.json:response | 82-byte fixture and traversal denial, independent guest size; evidence outer-read-batch3.json | **PASS scoped** |
| MCP-065 | boxweave_box_list_directory / F12 | outer-read-batch3.json:response; outer-read-batch3.json:ERR INVALID_ARGUMENT: Error code: INVALID_ARGUMENT; Error: McpServerError: BOX_PATH_INV | Actual fixture listing, ../../etc rejected; evidence outer-read-batch3.json | **PASS scoped** |
| MCP-066 | boxweave_task_submit / F29 | outer-task-batch.json:response; outer-task-batch.json:response; outer-task-batch.json:ERR INVALID_ARGUMENT: Error code: INVALID_ARGUMENT; Error: McpServerError: REQUEST_KEY_C | Same key dedup, changed command refused; guest marker QX exactly 2 bytes; evidence outer-task-batch.json | **PASS scoped** |
| MCP-067 | boxweave_task_get / F29 | outer-read-batch3.json:ERR INVALID_ARGUMENT: Error code: INVALID_ARGUMENT; Error: McpServerError: TASK_SELECTO; outer-task-batch.json:response | Tracked receipt by key; invalid selector refused; evidence outer-read-batch3.json,outer-task-batch.json | **PARTIAL** |
| MCP-068 | boxweave_task_list / F29 | outer-read-batch3.json:response; outer-task-batch.json:response | Tracked receipt listing read; evidence outer-read-batch3.json,outer-task-batch.json | **PARTIAL** |

## 19 originally discovered host CLI leaves (plus 7 later-discovered leaves below)

Actual binary: /usr/local/bin/boxweave (0.46.1). Each leaf has normal/invalid invocation or a documented mutation blocker. For privileged host/recovery mutators, syntax rejection is not a positive acceptance.

| ID | CLI leaf | Normal/positive test | Invalid/negative test | Status / gap |
|---|---|---|---|---|
| CLI-H-001 | setup | exit 0 {"authenticated_mcp": "not_verified", "descriptor": "installed_public", "external_reacha | exit 2  | **PARTIAL**;  |
| CLI-H-002 | version | exit 0 BoxWeave 0.46.1  | exit 2  | **PARTIAL**;  |
| CLI-H-003 | status | exit 0 {"app_health": "not_checked", "application_user": "boxweave", "external_web_access": "no | exit 2  | **PARTIAL**;  |
| CLI-H-004 | doctor | exit 0 {"application_user": "boxweave", "checks": [{"code": "PUBLIC_DESCRIPTOR_VALID", "id": "p | exit 2  | **PASS scoped doctor**;  |
| CLI-H-005 | report | exit 1  | exit 2  | **PARTIAL**;  |
| CLI-H-006 | provenance | exit 0 {"application_user": "boxweave", "host": "10.100.0.30", "installed_utc": "2026-11-05T17: | exit 2  | **PARTIAL**;  |
| CLI-H-007 | box.inspect | exit 2 {"code": "EXACT_INSTANCE_NOT_FOUND", "name": "nonexistent-1105-container", "project": "d | exit 2  | **PARTIAL**;  |
| CLI-H-008 | box.diagnose | exit 2 {"code": "EXACT_INSTANCE_NOT_FOUND", "name": "nonexistent-1105-container", "project": "d | exit 2  | **PARTIAL**;  |
| CLI-H-009 | client | exit 0 #!/usr/bin/env python3 # =============================================================== | exit 2  | **PARTIAL**;  |
| CLI-H-010 | http.status | exit 0 {"enabled": true, "external_verified": false, "listener": "127.0.0.1", "local_discovery" | exit 0 {"enabled": true, "external_verified": false, "listener": "1 | **PARTIAL**;  |
| CLI-H-011 | http.doctor | exit 0 {"enabled": true, "external_verified": false, "listener": "127.0.0.1", "local_discovery" | exit 0 {"enabled": true, "external_verified": false, "listener": "1 | **PARTIAL**;  |
| CLI-H-012 | http.configure | not run: host_config_mutation | exit 2  | **BLOCKED**; BLOCKED scoped host/credential/recovery mutation |
| CLI-H-013 | http.restart | not run: service_restart | exit 2  | **BLOCKED**; BLOCKED scoped host/credential/recovery mutation |
| CLI-H-014 | task.provision | not run: host_coordination_mutation | exit 2  | **BLOCKED**; BLOCKED scoped host/credential/recovery mutation |
| CLI-H-015 | task.recover.inspect | exit 2  | exit 2  | **PARTIAL**;  |
| CLI-H-016 | task.recover.audit | exit 2  | exit 2  | **PARTIAL**;  |
| CLI-H-017 | task.recover.release | not run: unknown_release_protected | exit 2  | **BLOCKED**; BLOCKED scoped host/credential/recovery mutation |
| CLI-H-018 | selftest.plan | exit 0 {   "confirmation": "eb08d7b0dfa8841e4f1862f18309139bcbc3f86db29eb36fb9593bf7f46cbafd",  | exit 2  | **PARTIAL**;  |
| CLI-H-019 | selftest.run | not run: live_nested_write_positive_separate | exit 2  | **SUSPECT (inconsistent initial vs later result)**; BLOCKED scoped host/credential/recovery mutation |

## 36 installed application CLI actions

Installed path: /home/boxweave/boxweave/current/.venv/bin/python -m bwmini ACTION --config ... , executed as boxweave user. Every mode got the bad-config input; only bounded read actions got positive config. Missing-config refuses most modes with exit2; selftest-plan is an N/A exception (does not need config).

| ID | Application CLI leaf | Positive/safe call | Negative/config call | Status / next test |
|---|---|---|---|---|
| CLI-A-001 | capabilities | exit 0 {   "actor_boundary": "application_provider",   "authorization": "not_estab | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-002 | init-admin | NOT EXECUTED (write/long-lived) | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-003 | reset-admin | NOT EXECUTED (write/long-lived) | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-004 | serve | NOT EXECUTED (write/long-lived) | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-005 | remote-mcp | NOT EXECUTED (write/long-lived) | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-006 | remote-https | NOT EXECUTED (write/long-lived) | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-007 | remote-caddyfile | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-008 | worker | NOT EXECUTED (write/long-lived) | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-009 | projects | exit 0 []  | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-010 | receipts | exit 0 []  | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-011 | history | exit 0 {   "operations": [],   "next_cursor": null }  | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-012 | reconcile | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-013 | environments | exit 0 {   "enabled": false,   "environments": [] }  | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-014 | environment-plan | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-015 | environment-confirm | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-016 | environment-receipts | exit 0 []  | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-017 | environment-receipt | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-018 | environment-observe | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-019 | model-wizard | exit 0 {   "revision": 0,   "selection": null,   "status": "not_selected",   "evid | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-020 | model-select | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-021 | model-confirm | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-022 | model-handoff | exit 2 {"code": "MODEL_SELECTION_REQUIRED"}  | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-023 | access | exit 0 {   "enabled": false,   "tenants": [],   "port_profiles": [],   "network":  | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-024 | access-memberships | exit 2 {"code": "ACCESS_DISABLED"}  | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-025 | access-network-plan | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-026 | access-port-plan | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-027 | access-confirm | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-028 | access-receipts | exit 0 []  | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-029 | access-receipt | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-030 | access-observe | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-031 | tenant-label | NOT EXECUTED (write/long-lived) | exit 2  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-032 | diagnose | exit 0 {"mode": "mcp", "connection": {"state": "authenticated_read_passed", "serve | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-033 | job-run | NOT EXECUTED (write/long-lived) | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **BLOCKED**; live positive business state not fully proven |
| CLI-A-034 | job-receipts | exit 0 []  | exit 2 {"code": "CONFIG_OR_RUNTIME_UNAVAILABLE"}  | **PARTIAL**; live positive business state not fully proven |
| CLI-A-035 | selftest-plan | exit 0 {   "confirmation": "1797505a28e5c919403b3e8dcf12cf4d3f29164d088337db7f7340 | exit 0 {   "confirmation": "f48ed13437909a5f5293cf34bd0157e2ccc822a81bd5de8b2 | **PARTIAL**; live positive business state not fully proven |
| CLI-A-036 | selftest-run | NOT EXECUTED (write/long-lived) | exit 2  | **SUSPECT/UNKNOWN in separate installed host wrapper test**; live positive business state not fully proven |

## Remaining required discovery contexts

- Authenticated installed SSH client (not just local stdio) and Streamable HTTP MCP initialize/tools/list + per-method positive tests: **NOT VERIFIED**; unauthenticated /mcp returned 401.
- Authenticated Web admin session and positive plan/confirm/worker/receipt flow, CSRF/Origin verification, public HTTPS OAuth: **NOT VERIFIED**. Four unauthenticated Web routes returned 401.
- Exact source tag/installed identity separate from external clients and Provider; raw logs in retained test lab are sanitized samples, not claims of installed Web parity.
- Upstream direct provider CLI and each leaf is a separate catalog; this run discovered top-level CLI but DID NOT exhaustively exercise upstream leaves; BoxWeave's 68 MCP surface/55 CLI leaves are distinct.
- R02 owner host reboot, R04 protected mode enablement, credential/SSH policy, Tenant ACL, public routes, upstream direct shrink and historical UNKNOWN claims remain intentionally blocked.

## Test-owned evidence files

The isolated lab retained the raw JSON batch files outer-read-batch2/3, outer-write-batch1/2, outer-task-batch, outer-compose-batch, outer-lifecycle-batch3, web-http-negative-batch, inner-mcp-local-stdio-responses, installed-host-cli-ledger and installed-app-cli-ledger. No secrets were collected. Next Agent should rerun originally failing cases before any source repair.


## 7 additional discovered installed/forwarded CLI leaves

Further source dispatch review discovered upstream and job command families absent from the top-level host help, plus the root-installed online.py installation helper. These are distinct leaves, not covered by the first 19. The source-only CLI help candidate subsequently adds all hidden families to top-level help. It has not been installed as a release.

| ID | Installed leaf | Positive / actual exit | Invalid or denied / actual exit | Decision / next test |
|---|---|---|---|---|
| CLI-H-020 | boxweave upstream status | Current local provider baseline=installed=0.128.0, exit0 | upstream invalid exits2 with usage | PARTIAL: version policy checked only for current daemon |
| CLI-H-021 | boxweave upstream upgrade --version | Positive upgrade NOT RUN: host daemon restart excluded | Malformed version returns UPSTREAM_VERSION_INVALID, exit2 | BLOCKED: separate test-owned Provider host required |
| CLI-H-022 | boxweave upstream update --version | Positive update NOT RUN: host daemon restart excluded | Malformed version returns UPSTREAM_VERSION_INVALID, exit2 | BLOCKED: separate test-owned Provider host required |
| CLI-H-023 | boxweave job -- /usr/bin/true | Actual app-user bounded job succeeded exit0, receipt 0ac9f95b96014f30953b80304d734ebc, output_bytes=0 | --budget nonsense refused JOB_BUDGET_INVALID exit2 | PARTIAL: job receipts verified returned id, but failure and timeout branches remain |
| CLI-H-024 | root-installed online.py install | --help exits0 and advertises stage/version prerequisites | Positive install NOT RUN on retained existing lab | BLOCKED: new clean target and official artifact needed |
| CLI-H-025 | root-installed online.py enroll | --help exits0; no key changed | Invalid public key refused FULL_PUBLIC_ED25519_KEY_REQUIRED exit2 | BLOCKED: live enrollment/trust mutation excluded |
| CLI-H-026 | root-installed online.py client-config | --help exits0; required identity/known_hosts options observed | Positive client secret-path registration NOT RUN | BLOCKED: no approved client identity or trust changes |

**Updated audit total:** 68 exposed MCP tool rows + 26 installed host/helper CLI leaves + 36 application CLI modes = **130 action rows**, including blocked and negative-only rows. The installed upstream native Containarium CLI's hundreds of possible leaves are not implied covered; only BoxWeave-owned or direct shipped helpers listed above are accepted as inventoried in this run.
