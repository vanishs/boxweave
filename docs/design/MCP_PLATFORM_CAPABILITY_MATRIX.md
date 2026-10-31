# Containarium Platform MCP — source inventory and BoxWeave gap mapping

**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.


**Inventory:** exactly **109 unique `Name` declarations** gathered from the Platform MCP registration modules at pinned Containarium revision. This is a **source inventory**; installed `tools/list` may return a smaller/changed set due to version, token scopes, deployment and backend. This table neither authorizes nor verifies any tool.

### Reading this matrix

- **App adapter/read path:** directly referenced by BoxWeave `provider.py`, `access_provider.py` or deployment probing at the reviewed baseline; `get_system_info` is a deployment probe rather than an interactive Web function.
- **Candidate: selective reuse:** useful to existing BoxWeave functionality, but requires a typed, reviewed adapter, caller/target authorization and live capability validation. **NOT IMPLEMENTED**.
- **Agent relay only:** current gateway generally forwards ordinary upstream tool calls. This is *not* evidence of managed Web integration, authorization equivalence, or successful execution.
- **Risk:** static screening tags only; exact scope/permission is determined by source `toolScopeAssignments`, installed JWT, target and policy. Even a "READ" tool may disclose sensitive information.

| Platform tool | Registration module | Upstream stated purpose (excerpt / otherwise schema) | BoxWeave treatment | Proposed review risk |
| --- | --- | --- | --- | --- |
| `create_backup` | `backup_tools.go` | Back up a tenant's database off-host. Runs pg_dump inside  | Candidate: selective reuse | MUTATING/EXECUTION |
| `list_backups` | `backup_tools.go` | List stored database backups (newest first), optionally  | Candidate: selective reuse | READ/OBSERVE |
| `restore_backup` | `backup_tools.go` | Restore a stored dump back into its container's database  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `verify_backup` | `backup_tools.go` | Prove a backup is restorable, not merely intact. The recorded  | Candidate: selective reuse | MUTATING/EXECUTION |
| `code_egress_policy` | `code_egress_tools.go` | Show the coding tool's egress allowlist for a tenant (or the cluster default  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `compose_disable` | `compose_tools.go` | Stop + disable the systemd-user autostart unit for one compose  | Candidate: selective reuse | MUTATING/EXECUTION |
| `compose_discover` | `compose_tools.go` | Discover docker-compose / podman-compose stacks under a tenant's LXC.  | Candidate: selective reuse | READ/OBSERVE |
| `compose_enable` | `compose_tools.go` | Install + enable the systemd-user autostart unit for a compose  | Candidate: selective reuse | MUTATING/EXECUTION |
| `compose_status` | `compose_tools.go` | Show one compose dir's status without a filesystem walk —  | Candidate: selective reuse | READ/OBSERVE |
| `mint_gateway_token` | `gateway_tools.go` | Mint a short-lived, scoped model-gateway token for a box  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `guardrail_policy_get` | `guardrail_tools.go` | Show the server-side guardrail policy: its rules, trusted signer  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `kms_envelope_coverage` | `kms_tools.go` | Count stored secrets by encryption mode: total,  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `kms_migrate_to_envelope` | `kms_tools.go` | Re-wrap legacy secrets under the active KMS KEK.  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `kms_status` | `kms_tools.go` | Report the active KMS envelope-encryption backend  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `add_miner_signature` | `miner_signature_tools.go` | Add an operator entry to the miner-signature list, effective immediately and  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `list_miner_signatures` | `miner_signature_tools.go` | List the merged miner-signature list the miner rule matches against: the  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `remove_miner_signature` | `miner_signature_tools.go` | Remove a previously operator-added miner signature. Baseline entries cannot  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `network_tenant_capabilities` | `network_tenant_tools.go` | Read authenticated local native network Tenant capability. Shared creation and atomic cross-Tenant p | App adapter/read path + Agent relay | READ/OBSERVE |
| `security_process_policy_get` | `process_policy_tools.go` | Show what a HIGH mining finding does to its container: the stored process  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `security_process_policy_release` | `process_policy_tools.go` | Undo the open throttle or freeze on a container: restores the CPU limit  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `security_process_policy_set` | `process_policy_tools.go` | Set what a HIGH mining finding does to its container. Without `container`  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `list_runners` | `runner_tools.go` | List provisioned runner boxes. Merges the local (boxes whose  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `provision_runners` | `runner_tools.go` | Provision N Containarium boxes as ephemeral GitHub Actions  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `remove_runner` | `runner_tools.go` | Drain, deregister, and delete one runner box. The drain step is  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `terminate_user_sessions` | `sessions_tools.go` | End every live SSH session a user holds on the backend: their own box and  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `sprint_board` | `sprint_tools.go` | Read the sprint board mirrored on one umbrella issue: the items, their  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `sprint_list` | `sprint_tools.go` | List the organization's sprints (open issues labelled \ | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `set_ssh_keys` | `ssh_keys_tools.go` | Replace a box user's authorized SSH keys with exactly the given list.  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `get_network_policy` | `tenant_network_tools.go` | Read a destination Tenant policy and its full revision for a safe ingress-only edit. | App adapter/read path + Agent relay | READ/OBSERVE |
| `list_network_policies` | `tenant_network_tools.go` | Read Tenant network policies. Configured ingress permissions are not TCP test results. | Candidate: selective reuse | READ/OBSERVE |
| `patch_tenant_access` | `tenant_network_tools.go` | Admin-only atomic ingress-source add/remove with full revision checks. Source A -> destination B cha | App adapter/read path + Agent relay | MUTATING/EXECUTION |
| `add_bad_destination` | `tools.go` | Add an operator-supplied entry (exact IP or CIDR) to the known-bad-destination  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `add_collaborator` | `tools.go` | Grant a teammate SSH access to a container — the same capability a  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `add_passthrough_route` | `tools.go` | Create a TCP/UDP passthrough route: forwards an external port on  | App adapter/read path + Agent relay | MUTATING/EXECUTION |
| `backend_validate_gpu` | `tools.go` | Check that GPU passthrough works inside an LXC on a backend. The  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `call_agent` | `tools.go` | Delegate a task to a running peer agent over the  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `check_for_updates` | `tools.go` | Check whether a newer Containarium release is available. Returns the running daemon version, the lat | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `code_answer` | `tools.go` | Answer one open permission request of a code run: allow or deny.  | Candidate: selective reuse | MUTATING/EXECUTION |
| `code_attach` | `tools.go` | Resume a run's output from a byte offset. Pass the `next_offset` from your  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `code_logs` | `tools.go` | Read one window of a code run's output through the daemon API, no SSH  | Candidate: selective reuse | READ/OBSERVE |
| `code_pending` | `tools.go` | List the open permission requests of a code run started with  | Candidate: selective reuse | READ/OBSERVE |
| `code_run` | `tools.go` | Start a coding agent on a box and return the first window of its output.  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `code_runs` | `tools.go` | List the code runs on a box (containarium code run) through the daemon API,  | Candidate: selective reuse | MUTATING/EXECUTION |
| `code_status` | `tools.go` | Report a run's liveness and, once it has finished, its exit code — including for a run that finished | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `code_stop` | `tools.go` | Stop a run. Its log is left readable, so output can still be collected afterwards with code_attach. | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `connect` | `tools.go` | Get SSH access to one of your boxes using the token you're already  | App adapter/read path + Agent relay | READ/OBSERVE |
| `create_container` | `tools.go` | Create a new LXC container under a username. Returns the container's  | App adapter/read path + Agent relay | MUTATING/EXECUTION |
| `crew_logs` | `tools.go` | Read one window of a run's journal (crew or single-skill run):  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `debug_container` | `tools.go` | Diagnose why SSH to a container is failing. Call this BEFORE  | Candidate: selective reuse | READ/OBSERVE |
| `delete_container` | `tools.go` | Delete a container permanently | App adapter/read path + Agent relay | MUTATING/EXECUTION |
| `delete_passthrough_route` | `tools.go` | Remove a TCP/UDP passthrough route by its external port and  | App adapter/read path + Agent relay | MUTATING/EXECUTION |
| `delete_route` | `tools.go` | Remove a proxy route by its domain — the unexpose counterpart of  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `delete_secret` | `tools.go` | Remove a tenant secret from the store. Does NOT cascade to env-var stamps on running containers — ca | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `deploy_recipe` | `tools.go` | Deploy a recipe as a new dedicated container. Provisions  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `expose_port` | `tools.go` | Expose a container's port on a public hostname. Resolves the  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `get_backend` | `tools.go` | Get a single backend's details by ID — same fields as list_backends  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `get_container` | `tools.go` | Get detailed information about a specific container including metrics | App adapter/read path + Agent relay | READ/OBSERVE |
| `get_metrics` | `tools.go` | Get runtime metrics (CPU, memory, disk, network) for containers | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `get_metrics_export` | `tools.go` | Get the current cloud-native metrics export configuration and last-known health for this host (enabl | Candidate: selective reuse | READ/OBSERVE |
| `get_secret` | `tools.go` | Read a single tenant secret's plaintext value. Always audit-logged. Use this to verify what you just | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `get_system_info` | `tools.go` | Get information about the Containarium host system | App adapter/read path + Agent relay | READ/OBSERVE |
| `get_upgrade_status` | `tools.go` | Poll an upgrade started by upgrade_backend. Returns the status (in_progress, completed, failed, noop | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `install_clamav` | `tools.go` | Install ClamAV and its signatures inside ONE box so the tenant-run malware scan  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `install_zap` | `tools.go` | Download and install OWASP ZAP into this host's security container.  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `list_agent_engines` | `tools.go` | Report, for each agent engine (claude/codex/gemini), whether  | Candidate: selective reuse | READ/OBSERVE |
| `list_agent_skills` | `tools.go` | List the platform's built-in agent skills. A skill is a  | Candidate: selective reuse | READ/OBSERVE |
| `list_backends` | `tools.go` | List the cluster's backend hosts (the local daemon plus any  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `list_bad_destinations` | `tools.go` | List the known-bad-destination list (#1641) the threat-detection sentry's  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `list_collaborators` | `tools.go` | List everyone with SSH access to a container — the owner plus every  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `list_containers` | `tools.go` | List all containers with name, username, state, IP, and resources.  | App adapter/read path + Agent relay | READ/OBSERVE |
| `list_crews` | `tools.go` | List the platform's built-in crews. A crew is a  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `list_passthrough_routes` | `tools.go` | List the TCP/UDP passthrough routes currently registered on the  | App adapter/read path + Agent relay | READ/OBSERVE |
| `list_recipes` | `tools.go` | List the platform's built-in recipes. A recipe is a  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `list_routes` | `tools.go` | List the proxy routes currently registered on the sentinel — the  | Candidate: selective reuse | READ/OBSERVE |
| `list_secrets` | `tools.go` | List a tenant's secret names + versions + timestamps. Never returns the plaintext values — use get_s | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `list_security_sentry_findings` | `tools.go` | List security findings raised by the background threat-detection sentry  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `move_container` | `tools.go` | Migrate a container from this daemon to a peer daemon using  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `network_guard_status` | `tools.go` | Report the backend's tenant-isolation posture: the two Incus NIC-ACL  | App adapter/read path + Agent relay | READ/OBSERVE |
| `push` | `tools.go` | Push committed git history into a container via real `git push` over  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `refresh_secrets` | `tools.go` | Re-stamp env vars on the LXC from the current secret-store state. Useful after rotation when the nex | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `remove_bad_destination` | `tools.go` | Remove a previously operator-added entry from the known-bad-destination list.  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `remove_collaborator` | `tools.go` | Revoke a collaborator's SSH access to a container — the inverse of  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `resize_container` | `tools.go` | Change a container's CPU / memory / disk allocation in place. At least one of cpu, memory, disk, cpu | App adapter/read path + Agent relay | READ/OBSERVE |
| `resolve_security_sentry_finding` | `tools.go` | Mark an open threat-detection sentry finding as resolved, after triaging and handling it. | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `revoke_token` | `tools.go` | Admin: revoke a JWT by its jti. The token is rejected  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `run_agent_skill` | `tools.go` | Run an agent skill in a box. Provisions the skill's box,  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `run_crew` | `tools.go` | Run a crew: validate its topology against the members'  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `security_findings` | `tools.go` | List security findings for a container, normalized across the three  | Candidate: selective reuse | READ/OBSERVE |
| `security_remediate` | `tools.go` | Attempt to auto-fix a security finding. Currently only works on  | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `security_scan` | `tools.go` | Run a security scan on a container. Triggers one or more of the daemon's  | Candidate: selective reuse | MUTATING/EXECUTION |
| `security_sentry_status` | `tools.go` | Report the background threat-detection engine's on/off state (#1640):  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `set_metrics_export` | `tools.go` | Enable or disable opt-in export of this host's infra metrics to its cloud's native monitoring (GCP C | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `set_secret` | `tools.go` | Create or update a tenant secret stored encrypted (AES-256-GCM) on the daemon. The value will be sta | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `start_container` | `tools.go` | Start a stopped container. When waitForReady is true the daemon blocks until the container's primary | App adapter/read path + Agent relay | MUTATING/EXECUTION |
| `stop_container` | `tools.go` | Stop a running container | App adapter/read path + Agent relay | MUTATING/EXECUTION |
| `sync` | `tools.go` | Mirror a local directory into a container — rsync-style, but using  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `sync_ssh_config` | `tools.go` | Generate a self-contained ssh_config covering every reachable container  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `toggle_auto_sleep` | `tools.go` | Opt a container into auto-sleep. Enabled containers are stopped after `idleThresholdMinutes` of netw | Candidate: selective reuse | MUTATING/EXECUTION |
| `toggle_monitoring` | `tools.go` | Enable or disable application-emitted OpenTelemetry on an existing container without recreating it.  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `upgrade_backend` | `tools.go` | Upgrade a backend's daemon to the binary the sentinel currently serves, immediately instead of waiti | Agent relay only; no app domain wrapper | SENSITIVE/ADMIN |
| `tracker_claim` | `tracker_tools.go` | Claim a tracker issue for this run: posts a stamped claim comment and  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `tracker_comment` | `tracker_tools.go` | Post a comment on a tracker issue or change request, stamped with this  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `tracker_create_issue` | `tracker_tools.go` | File a follow-up issue on the tracker connection this run is bound to (#2024).  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `tracker_get_change` | `tracker_tools.go` | Read a single change request's (pull/merge request) state and  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `tracker_get_issue` | `tracker_tools.go` | Read a single tracker issue, including its comments, from the tracker  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `tracker_list_issues` | `tracker_tools.go` | List tracker issues, optionally filtered by state, labels (AND), or  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `tracker_route_list` | `tracker_tools.go` | List a tracker connection's scope routes: which agent skill each  | Agent relay only; no app domain wrapper | READ/OBSERVE |
| `tracker_set_labels` | `tracker_tools.go` | Add and/or remove labels on a tracker issue. Mirrors `containarium tracker issue label`. | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |
| `tracker_submit_change` | `tracker_tools.go` | Bundle this run's committed workspace out of its box, push it from a  | Agent relay only; no app domain wrapper | MUTATING/EXECUTION |

### Collisions and non-equivalences

- Platform `compose_discover` and agent-box `compose_discover` have the **same tool spelling** but different execution/authorization paths; never combine them into a name-only router.
- `create_backup`/`restore_backup` concern tenant **database** backups, **not** the Box Environment E0–E4 instance snapshot/restore/clone proposal. They do not replace or implement E2/E3/E4.
- `code_answer` applies to a particular upstream coding run. It does **not** authorize arbitrary BoxWeave workflow mutations or satisfy BoxWeave's host approval rules.
- `get_secret`, KMS, host upgrade, JWT revocation, arbitrary user session changes and privileged controls are not suitable for broad default Agent exposure. A gateway that forwards them is an authority risk to resolve before claiming least privilege.
- Provider/tool/Incus version changes can invalidate exact schemas and result parsing; only a tool that passes current actor, scope, Schema and actual execution preflight is callable.

**References:** [master](MCP_CAPABILITY_INTEGRATION_MASTER.md), [source audit](../verification/MCP_CAPABILITY_SOURCE_AUDIT.md), [policy design](MCP_INTEGRATION_SECURITY.md), [M0](mcp-integration/M0_SOURCE_AND_RUNTIME_CAPABILITY.md), [current architecture](ARCHITECTURE.md), [upstream registry](https://github.com/FootprintAI/Containarium/blob/f37358e631282ad90bd534796667af8d0a56b72c/internal/mcp/tools.go).
