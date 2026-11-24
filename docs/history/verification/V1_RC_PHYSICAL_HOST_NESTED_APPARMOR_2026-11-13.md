# v1.0.0-rc.1: owner-operated physical-host nested AppArmor experiment (2026-11-13 PT)

**Status: PARTIAL / L3 BLOCKED (historical snapshot, before the operator's planned full physical-host reinstall).** Real hosted ChatGPT OAuth/MCP and the v1-owned L2 lifecycle work. Incus 7.6 inside L2 starts, but its first `incus admin init` fails while loading a dnsmasq AppArmor policy. No third-level guest was created. This is evidence from a separate, owner-operated physical-host pilot, **not** a formal independent verifier, `HOST_GO=PASS`, or supported v1 release acceptance.

## Scope and evidence provenance

- The owner explicitly provisioned and configured the outer test L1 on the physical `dev` Incus host; the assistant did **not** receive physical-host execution access. Owner-supplied terminal output establishes the outer native configuration and installer results.
- The actual ChatGPT connector named **`v1rc`** was invoked directly for authenticated read-only `host_status`/`box_list`, a single new v1-owned L2 Create, guest Exec, operation/output retrieval, and read-only post-failure inspection.
- No old main gateway, historical UNKNOWN operation, other Project, physical-host policy, `raw.lxc`, `security.privileged`, or global AppArmor configuration was changed by the assistant.
- The experimental Cloudflare Quick Tunnel and OAuth approval password were operator-managed. Do not store the short-lived public hostname, password, access/refresh token, session/callback parameters or OAuth grants here.
- All observed guest-native effects were scoped to a new v1rc-owned L2. The guest **was last observed RUNNING on 2026-11-13**; failed inner Incus initialization was not retried, reset, or cleaned up during that experiment. **Do not assume it survives the planned host reinstall or is still running now.**

## Exact topology and versions

| Layer | Origin and versions | AppArmor / native observations | Result |
| --- | --- | --- | --- |
| L0 physical `dev` | **Incus client 7.5.1, server 7.5.1**, confirmed by owner using `sudo incus version`; unprivileged CLI alone said server unreachable | L0 AppArmor policy and kernel support were not directly inventoried in this exercise | Owner-created L1 succeeds |
| L1 `bw-v1rc-l1-1113` | Ubuntu 24.04, `security.nesting=true`, `security.privileged=false`, 4 CPUs, 6 GiB, root pool `default`; installed pinned Zabbly Incus **7.6 client/server** | `/proc/1/attr/current=unconfined` **as seen inside L1**; `.ns_level=1`, `.ns_stacked=yes`, namespace `incus-bw-v1rc-l1-1113_<var-lib-incus>`; unprivileged user mapping | Incus init, `incusbr0`, dnsmasq, restricted service Project and v1 RC installation succeed |
| L2 `v1rc-nest-l2-1113` | Created by actual `v1rc` authenticated MCP in L1's `user-1001` Project; `nesting=true`, 2 CPUs, 3072 MiB, requested 12 GiB, Internet enabled; pinned Zabbly Incus **7.6 client/server** installed | `/proc/1/attr/current=incus-user-1001_v1rc-nest-l2-1113_</var/lib/incus> (enforce)`; **same L1 namespace name/level=1**, `.ns_stacked=yes`; native default profile includes Incus-generated `raw.idmap` | v1 Create/Running/guest Exec PASS; inner Incus daemon active; `incus admin init` fails; no inner bridge or L3 |
| L3 | Not created | No AppArmor or workload evidence | **NOT TESTED / BLOCKED** |

`unconfined` inside L1 is a **namespace-relative observation**; it is **not proof that L1 is unrestricted by outer L0 policy**. Both L1 and L2 have stacking reported as `yes`, but only L2's process visibly has an enforced profile. BoxWeave does not choose `enforce` versus `unconfined` as a direct MCP setting; the creator Incus and its policy determine the effective profile. Do not equate `security.nesting=true` with AppArmor policy delegation.

## L1 provisioning: owner-supplied native readbacks

1. On physical L0, `sudo incus launch images:ubuntu/24.04 bw-v1rc-l1-1113 -c security.nesting=true -c security.privileged=false -c limits.cpu=4 -c limits.memory=6GiB`. Expanded device topology: default profile, `eth0` on L0 `incusbr0`, root disk on pool `default`; no user-added AppArmor/raw LXC overrides.
2. Inside L1, signed Zabbly Incus `1:7.6-ubuntu24.04-202611072126`; `incus admin init --auto --storage-backend=dir` PASS, `incusbr0` IPv4/IPv6 available and dnsmasq process running.
3. Created dedicated `boxweave` OS user UID 1001 / Incus GID 990, triggering restricted `user-1001` Project with `restricted.containers.nesting=allow`, dedicated `incusbr-1001` and Incus-generated service-user `raw.idmap`.
4. Copied approved Ubuntu 24.04 image into Project. Verified immutable Release `v1.0.0-rc.1` binary SHA256 `b0b1569941e58a7486b7337c5fb2eb3c8b73aed221697e23568b07896d5d2b86`, commit `a5e7b0e9d0dd8c3243d3edeb8c335594b29f1e07`; release `install.sh` SHA256 `1c6da7e341b810c65abd760c378f8bca1f139463090e4e358a9c58c476105bbc`.
5. `native-prepare-plan` / approved `native-prepare-apply` applied the **exclusive** `bw-v1rc-lab-1113` ACL to the empty restricted Project default NIC, with IPv4/IPv6 anti-spoof and explicit private/ULA deny. Readback `native-prereq-plan` and `native-prereq-verify`: **8/8 checks OK, `config_ready=true`**, review SHA256 `41b8206405c2e7bcf959ce41c6d7d2ec9412fc81115a58ef15efce6d7a7c7237`. These checks do **not** substitute for actual packet witnesses.
6. Operator-approved RC helper `--apply` completed on the separate L1, `install-status` reported `service_active=true`, `restart_required=false`, same binary and commit; local `oauth-setup` completed and a separate Quick Tunnel origin was configured. No password or public URL copied into the repository.

**Storage caveats:** `selected_storage_quota_verified=false`, `selected_storage_separate_mount=false`; the dir pool shares the L1 root backing filesystem and requested per-Box disk size is **not a hard verified quota**. No disk-exhaustion stress test is authorized by this evidence.

## Real ChatGPT v1rc MCP receipts (L1 controller, L2 guest)

- Authenticated `host_status`: `version=1.0.0-rc.1`, `commit=a5e7b0e`, `mcp_ready=true`, `native.ready=true`, `project=user-1001`, `incus_version=7.6`; initial `box_list` empty.
- One new `box_create` for `v1rc-nest-l2-1113` (`nesting=true`, start=true, Internet enabled), original **Operation `b124f0a29f7f13c17a980742c8343109`**, native Operation `075e4add-f605-4d5b-82e1-e97dc63cd9e0`: `SUCCEEDED`. Exact owned L2 **UUID `6eba364e-d8a0-4df8-9f81-7098aedf3046`**; `box_get` confirmed Running, nesting true.
- Guest profile/namespace read-only Exec: original Operation `6dff9f248c287be255fddd781321ace5`: `SUCCEEDED`, stdout exit 0. It showed enforced Incus profile and same level-1 namespace as L1.
- Pinned Incus 7.6 guest install + daemon start + **single** `incus admin init --auto --storage-backend=dir`: original Exec Operation `a5a959f12aa2c898996866205aa9fd27`. **Controller Operation state `SUCCEEDED` but recorded guest exit code `1` and `command_success=false`**. The distinction matters: successfully recording a native Exec does **not** mean its shell command succeeded.
- Original stdout stage markers reached `L2_STAGE=INSTALL_INCUS_7_6`, `INSTALLED`, `START_SERVICE`, `DAEMON_ACTIVE`, `INIT`. Client/server both 7.6 and `systemctl is-active incus.service` returned `active` before initialization failed.
- Original stderr decisive failure:

```text
Error: Failed to create local member network "incusbr0" in project "default":
Failed to run: apparmor_parser -rWL
/var/lib/incus/security/apparmor/cache
/var/lib/incus/security/apparmor/profiles/incus_dnsmasq-incusbr0:
exit status 243
apparmor_parser: Unable to replace "incus_dnsmasq-incusbr0_</var/lib/incus>".
apparmor_parser: Access denied. You need policy admin privileges
to manage profiles.
```

- Read-only postfailure Exec: original Operation `19c69acf865bbdc36c637ba7b3d4f718`, guest exit 0. Inner daemon still `active`, `incus network list` contains only unmanaged `lo`, no inner instances. Daemon logged `Per-container AppArmor profiles are disabled because Incus is already protected by AppArmor`. No second `incus admin init` was run.

## Interpretation: evidenced facts versus hypotheses

**Proven here:** L0 7.5.1 can create the unprivileged/nesting-enabled L1; L1 Incus 7.6 can initialize and serve a restricted v1 RC; the OAuth HTTP MCP can create and execute L2; L2 Incus 7.6 cannot load its required dnsmasq AppArmor policy in this topology. This is a concrete AppArmor policy-administration failure, not merely a missing Go installer check.

**Still unproven:** Whether the *difference* between L0 7.5.1 and inner Incus 7.6 causes or contributes to this failure. Changing a client version is not evidence; the owner separately confirmed `sudo incus version` on L0 is client **and server** 7.5.1. A paired `7.5.1 -> 7.5.1` control topology under the **same host kernel/AppArmor configuration** has **not** been run. Do not claim a specific Incus regression or that arbitrary-depth delegation is universally impossible. Kernel and LXC policy behavior may also matter.

**Source-level candidate explanation (not yet a validated fix):** In pinned Incus 7.6, confined daemon processes disable further per-container AppArmor policy setup; nested AppArmor namespace creation uses stacking predicates. This aligns with L2 inheriting level 1, and the inner network profile load denial. See [separate source study](../research/APPARMOR_MULTI_LEVEL_BOUNDARY_2026-11-13.md) and its upstream references.

## Next bounded experiment / gates

1. On the **newly reinstalled physical L0**, first record the *new actual* Incus client/server versions, kernel, AppArmor policy and defaults; do not assume the previous L0 `7.5.1` conditions persist. In separate disposable L1 environments, compare an accurately pinned matched-version control with the historical `7.5.1 -> 7.6 -> 7.6` result **only after acknowledging any host/kernel changes**. A difference alone is evidence for further analysis, **not** proof of causation.
2. Investigate whether upstream Incus/LXC/AppArmor can safely delegate a new policy namespace to L2 **without** lifting L0/L1 authority, using isolated/native negative tests. Do not solve by blanket `raw.lxc` unconfined, privileged LXC, global AppArmor disable or direct physical-host security-policy changes.
3. **PR #136's hard-fail preflight is deferred/archived, not adopted.** It is a static heuristic rather than actual Incus readiness proof. Owner discussion favors keeping factual diagnostics and authoritative native failure evidence for future review, rather than merging a mandatory reject based solely on process-profile appearance. Its original [research](../research/APPARMOR_MULTI_LEVEL_BOUNDARY_2026-11-13.md) and [experimental test receipts](APPARMOR_NATIVE_HOST_PREFLIGHT_2026-11-13.md) are preserved without code changes.
4. Native packet witnesses/quotas, independent verifier, actual L3 lifecycle, safe recovery/cleanup and relevant exact-head Actions remain **PENDING**; no formal S2.5/HOST_GO or third-level Host claim.

**Historical test resource:** `user-1001/v1rc-nest-l2-1113`, UUID `6eba364e-d8a0-4df8-9f81-7098aedf3046`, last observed Running on 2026-11-13. A physical-host reinstall may erase it. Do not automatically replay original operations; any follow-up on a fresh installation must discover the new native identities.
