# S2.5 RC6 real L2-to-L3 BoxWeave MCP recursion: positive and negative evidence

**Date:** 2026-11-15 PT
**Classification:** L2-controller-to-L3-installed-HTTP-MCP functional smoke PASS; BH-018 UNKNOWN classification needs review; BH-019 independent-per-depth secure isolation **NOT ACCEPTED**.
**Source:** immutable prerelease `v1.0.0-rc.6`, Linux amd64 binary SHA256 `0932ad434d2a1b113e2254f6ca02a218c2dff473f70d7ed3b2f45e93d0610174`, build commit `370e419c43f06b7488c7330ba61913ffec0e9f78`. Product code matches PR #146 checkpoint parent `89b0f8910376be65eb94153f41eecd5b81f6e6f6`; the prerelease commit adds only CI workflow changes.
**Controller:** installed official RC6 on independently administered L2 `rc6-l2-lab-1115`, accessed via authenticated `BoxWeave_L2_RC6` MCP. L2 native Incus 7.6, Project `user-1001`; `host_status.ready=true`. The L2 parent Guest itself is confined by inherited AppArmor `incus-user-1001_rc6-l2-lab-1115_</var/lib/incus> (enforce)`.
**Work reference:** `RC6-L2-L3-DEPLOY-20261115`.

## 1. Fail-closed negative controls and initial failure

All three distinct preflight operations ended `FAILED / SAFE_NATIVE_PREFLIGHT_DENIED`; no native instance was created by them:

| Input | Original operation ID |
| --- | --- |
| `lxc.apparmor.profile=unconfined` | `ade263c808c78c154a78ddb37b27e6a8` |
| `security.idmap.isolated=true` and size 4194304, unsupported allocator | `29f0cb03e17cb5842739b7339bb28f7b` |
| `lxc.apparmor.profile=unchanged` with Project lowlevel omitted | `9c391f208df35bc16273a286d251c088` |

A default, unprivileged `security.nesting=true` Box `rc6-l3-lab-1115`, UUID `0c2be850-9375-430b-9730-59377cc4bb96`, was correctly created stopped (operation `c4168b5387c5d2855a95cfa35f826578`). Its original Start operation `c6938fe17342ad02567ae11cf5642590`, native Incus operation `0f522d84-33b8-4535-bd0e-53fe0b093e26`, ended **UNKNOWN / POSTCONDITION_UNVERIFIED** and requested `HOST_LOCAL_REVIEW`. Independent read-only `box_get`, `incus info`, and `incus info --show-log` showed the instance **STOPPED** and the LXC startup failure:

- `cgroup2_devices ... bpf_program_load_kernel ... Operation not permitted`;
- `apparmor_process_label_set_at ... Permission denied - Failed to write AppArmor profile "unconfined"`;
- `Failed to change AppArmor profile to unconfined`, then `ABORTING`.

**Causal attribution:** the nested `incusd` configured with `INCUS_SECURITY_APPARMOR=false` otherwise falls into LXC's default attempt to switch to unconfined, correctly denied by inherited enforcing AppArmor. The BPF denial was concurrently observed but was **not proven to be fatal**; subsequent identical-environment start with `unchanged` succeeded. The aborted original Start **remains immutable UNKNOWN and was not retried, cancelled, replayed, reconciled by forced release, nor deleted**. Native read-only stop state is not a license to bypass its original BoxWeave recovery contract. This should become a targeted BH-018 regression: classify a native LXC startup error against original operation and native receipt without incorrectly implying success or erasing true uncertainty.

## 2. Narrow alternate path, successfully created and started L3

After confirming the restricted `user-1001` Project contained **only** this known stopped owned instance and no foreign native instances, the L2 root administrator (executed inside the scoped parent L2 Guest through authorized L1 BoxWeave Exec) set **only L2's** `restricted.containers.lowlevel=allow`; L0/L1 restricted Projects, host root sockets, physical AppArmor, privileges, and inherited L2 enforcing policy were not changed. L1 parent Exec operation `0ad289813a935755b950be0525a60ed7` SUCCEEDED. This **broadens a restricted Project's low-level admission**; it is not independently sufficient security acceptance. Product MCP continues to accept only typed/narrow LXC options, rejecting arbitrary `raw.lxc`.

A new unprivileged instance `rc6-l3-enforced-1115`, UUID `1762cebb-033b-4be2-a4f3-bc1af0caa8c6`, 2 CPUs, 2048 MiB RAM, 16 GiB requested disk, internet true, `security.nesting=true`, **exact** `lxc.apparmor.profile=unchanged`, `boot.autostart=false`, was admitted (Create operation `e9a6ece1c77eca1f648b339587cdb224`, SUCCEEDED). Native readback verified `lxc.apparmor.profile=unchanged` and resources. Start operation `627615fa11570bca33951d4bda9ae706` **SUCCEEDED**. No privileged or unconfined fallback used.

Read-only Guest security witness through actual L2 MCP Exec `476133831cb64213cf317dbb9da64dfd`:
- Ubuntu 24.04.5; PID 1 and Exec labels both `incus-user-1001_rc6-l2-lab-1115_</var/lib/incus> (enforce)`.
- AppArmor module `Y`, `Seccomp: 2`.
- Distinct nested UID/GID mappings (e.g. UID `0 1000000 1001`, `1002 1001002 997998998`; GID `0 1000000 990`, `991 1000991 997999009`). **These are observed maps, not evidence of safe transactional UID/GID reservations, sibling collision freedom or independent AppArmor policy namespaces.**
- Private eth0 IPv4 `10.13.12.3/24`, ULA/link-local IPv6, HTTPS signed package upstream HTTP 200.

Idempotent exact Create replay returned original UUID and operation with `duplicate=true` without creating another instance. Guest `exit 19` via Exec `9b1e61c50ad6e27427dcdd3de11cfdcf` produced original native Exec operation `SUCCEEDED`, `command_success=false`, `exit_code=19`, and retained expected stderr. This correctly distinguishes native Exec transport completion from guest command exit.

## 3. Incremental independent L3 Incus and official BoxWeave RC6 deployment

All steps executed **inside only the new L3 Guest** via real L2 BoxWeave `box_exec` and were recorded under the work reference above:

1. Switched official Ubuntu APT sources to HTTPS, `apt-get update` PASS and normally installed `gpg` plus `dnsmasq-base` (no bypass of APT's `_apt` account).
2. Verified the Zabbly signing-key fingerprint exactly `4EFC590696CB15B87C73A3AD82CC8797C838DCFD`, enabled signed Noble stable repository, and installed Incus/Incus client/base `1:7.6-ubuntu24.04-202611072126`. Both client and server report 7.6.
3. Added **only to L3's own** `incus.service` systemd drop-in `Environment=INCUS_SECURITY_APPARMOR=false`, restarted and verified the effective incusd process environment. Guest continued inherited `(enforce)`; the parent L2 daemon and physical host were unchanged. `incus admin init --auto --storage-backend=dir` passed, creating independent `default` dir storage pool and `incusbr0` bridge.
4. Created a dedicated OS `boxweave` user, UID 1001, non-admin `incus` GID 990 **without incus-admin**, and an automatically provisioned restricted `user-1001` project/isolated bridge `incusbr-1001`. No preexisting inner instances.
5. Downloaded the immutable public `v1.0.0-rc.6` Release installer and SHA256SUMS; installer checksum and syntax verified. `--plan` bound the correct exact release source commit and binary SHA, and the L3's own host-machine digest. L3 exact `install.sh --apply` completed; installed binary `build-info` `1.0.0-rc.6` / `370e419c43f06b7488c7330ba61913ffec0e9f78`, systemd active.
6. Generated an owner-reviewed `native-prepare-plan` for owned empty Project `user-1001`, existing `default` pool and `incusbr-1001` bridge, new exclusive ACL `bw-l3-rc6-1115`, review SHA256 `6128b1297f91b402ac886522c736f8e5afaa849331267c7a23e016ef725e4e02`. `native-prepare-apply` returned **APPLIED**; native prereq review SHA256 `42f74ce72a505b73531afbe6acdaea09a866641061bd9c8a13ac99ab4f8f5b7f` verified **8/8 checks**. After verification, actual `doctor` `ready=true`, `mcp_ready=true`, `native.ready=true`, Incus 7.6.

## 4. Signed Cloudflare Tunnel, OAuth and public read-only probes

- Installed Cloudflare's `cloudflared` signed APT source using fingerprint `CC94B39C77AE7342A68B89628A682D308D4E5E73`, installed exact `cloudflared 2026.11.0`.
- One initial precheck incorrectly assumed the `cloudflared tunnel --help` command advertised `--protocol`; the check returned guest exit 1, with BoxWeave original Exec **SUCCEEDED** as transport, and no systemd unit created. This is a test-harness assumption, not a product crash. Inspected CLI and found executable `/usr/local/bin/cloudflared`; selected previously validated default protocol negotiation instead of forcing an unverified CLI flag.
- A dedicated nonadmin `cf-quick` systemd service `boxweave-l3-quick-tunnel.service` now runs Quick Tunnel, without an elevated user. Network precheck observed one regional QUIC fail, other region success, both HTTP/2 paths success; Cloudflare automatically selected HTTP/2. Service active.
- Created the approval password **once** under `/home/boxweave/.boxweave-secrets/oauth-password` (directory 0700, file 0600, `boxweave` ownership); **password intentionally never printed to the model, PR, public logs or response**. Owner reads it locally in L3 via trusted nested terminal. Configured L3's OAuth public origin to its current ephemeral Quick Tunnel URL. Public hostname is intentionally omitted from this repo document.
- Real public postflight: local and public unauthenticated POST `/mcp` **HTTP 401**, HTTPS `/.well-known/oauth-authorization-server` **HTTP 200**, `/.well-known/oauth-protected-resource/mcp` **HTTP 200**. Actual local doctor still `ready=true`, `mcp_ready=true`; L3 Guest policy `(enforce)` retained. The owner still must authorize the external ChatGPT OAuth connection; HTTP probes are not proof of completed interactive OAuth login or L4 capability.

## 5. Terminal state and open acceptance gaps

L2 original inventory was **empty**. Final L2 `box_list`: exactly **two** test instances — original `rc6-l3-lab-1115` STOPPED with the **original unresolved UNKNOWN**; operational `rc6-l3-enforced-1115` RUNNING, with independent Incus 7.6, BoxWeave RC6, exclusive network ACL and Quick Tunnel. L1 inventory remained its previous **10 running Boxes**, including L2. L2 `host_status` stays ready.

L2 `operation_list(work_ref=RC6-L2-L3-DEPLOY-20261115)` returned **33 operations**: 29 `SUCCEEDED` (including Exec operations with guest exit 1/19), 3 preflight `FAILED`, 1 original `UNKNOWN`; no other UNKNOWN. No unsafe retry/replay/force-release of the original.

**Open:** true installed HTTP MCP-to-MCP multi-layer acceptance beyond L3 is NOT YET TESTED (owner still to connect L3). Effective per-depth AppArmor labels share one ancestor enforced profile, not independent policy namespaces. UID/GID lease reservation, collisions/exhaustion, cross-sibling packet isolation including IPv6 link-local, host/guest fault recovery, native unknown repair, quota and reboot testing remain unaccepted. Native `dir` pool quota verification remains WARN, and image may download on demand. Before L4, verify L3's own empty restricted Project and independently review its own lowlevel adapter to allow exact `unchanged` only; never preemptively weaken physical-host or L2/L1 AppArmor.

**Next investigation:** (a) explicit original FAILED native Start versus `UNKNOWN` semantics in BH-018, (b) BPF denial vs fatal/nonfatal AppArmor transitions, (c) compare zero-downgrade inherited-enforce with desired per-level policy separation. Never label BH-019 security PASS from this functional smoke.
