# BH-019 — Incus 7.6, enforced AppArmor inheritance, functional recursion to L5 (2026-11-14/15)

**Acceptance classification: native functional deep-recursion LAB PASS THROUGH L5; per-layer autonomous AppArmor policy namespace / sibling isolation SECURITY ACCEPTANCE NOT YET PASS.** The setup tested here **does not need complain, privileged instances, new kernel capability, host AppArmor disable, or upstream patches**. It differs materially from the owner-requested stronger model (independent per-depth AppArmor profiles), and does **not** establish infinite depth.

## Original laboratory topology and trust boundary

```text
L0 physical kernel 7.0.0-43-generic, Incus 7.6 (BoxWeave L0 RC5)
 └ L1: bh019-76-noaa-1114 [single disposable BW-owned Box, UUID c9f3912d-f523-4018-947c-ce774fda1537]
       2 CPUs, 2048 MiB RAM, 12 GiB disk, unprivileged, nesting=true,
       temporary Box-only internet for software; own AA namespace ns_level=1,
       locally visible unconfined, isolated Incus Project
     └ L1's own real Incus 7.6, ordinary enforced child Profile
       └ L2: bh019-noaa-enforce-child1 [RUNNING / exec PASS]
            AppArmor: incus-bh019-noaa-enforce-child1_</var/lib/incus> (enforce)
            UserNS and mountNS distinct; 4194304 delegated UIDs/GIDs
          └ L2's own real Incus 7.6 [daemon-only INCUS_SECURITY_APPARMOR=false]
            └ L3: bh019-noaa-wide-child2 [RUNNING / exec PASS]
                 inherited SAME L2 enforced AppArmor Profile
                 distinct UserNS/mountNS; UID/GID 4128768
               └ L3's own real Incus 7.6 [daemon-only INCUS_SECURITY_APPARMOR=false]
                 └ L4: bh019-noaa-wide-child3 [RUNNING / exec PASS]
                      inherited SAME L2 enforced AppArmor Profile
                      distinct UserNS/mountNS; UID/GID 4063232
                    └ L4's own real Incus 7.6 [daemon-only INCUS_SECURITY_APPARMOR=false]
                      └ L5: bh019-noaa-wide-child4 [RUNNING / exec PASS]
                           inherited SAME L2 enforced AppArmor Profile
                           distinct UserNS/mountNS; UID/GID 3997696
                           no independent Incus daemon installed at L5
```

All 4 internally independent Incus daemons at L1/L2/L3/L4 were **official Zabbly stable 7.6**: Ubuntu24.04 package `1:7.6-ubuntu24.04-202611072126`, with separate local Unix sockets and `dir` storage and private `incusbr0` networks. Each `incus admin init --auto --storage-backend=dir` **actually succeeded** at the recorded depth. No daemon, package, host security toggle, or container at L0 shared historical inventory was modified.

## Source-backed causal solution (Incus 7.6 tag v7.6.0)

1. **Daemon-only Incus integration switch.** `INCUS_SECURITY_APPARMOR=false` is an *official Incus environment variable*: it changes the **environment of each selected nested systemd `incus.service`**. The parent L0/L1 Incus daemon and the kernel AppArmor enforcing L2 were not switched off.
   - `internal/server/sys/apparmor.go::initAppArmor`: `util.IsFalse(os.Getenv("INCUS_SECURITY_APPARMOR"))` prevents `AppArmorAvailable` from becoming true.
   - `internal/server/apparmor/apparmor.go::runApparmor`: returns without calling `apparmor_parser` if `!sysOS.AppArmorAvailable`, thereby avoiding the earlier `dnsmasq` and generated child policy-write paths.
   - `internal/server/apparmor/archive.go::ArchiveWrapper`: returns no policy wrapper if `!sysOS.AppArmorAvailable`, thereby avoiding earlier archive unpack parser243.
   - `shared/subprocess/proc.go::hasApparmor`: checks the variable and skips spawning `aa-exec` when false.
   - `internal/server/instance/drivers/driver_lxc.go`: LXC AppArmor profile auto selection is conditional on `AppArmorAvailable`.
   Official docs: https://linuxcontainers.org/incus/docs/main/environment/ ; upstream v7.6.0 source: https://github.com/lxc/incus/tree/v7.6.0/internal/server .
2. **Retain enforced parent, do not switch to `unconfined`.** When AppArmor integration is switched off for nested daemon, the default LXC path attempted `change_profile -> unconfined`; under actual L2 `enforce` this was **denied** (correctly). On **only** each further nonprivileged child, set exact `raw.lxc=lxc.apparmor.profile = unchanged`. LXC's official `unchanged` setting makes the child retain the existing AppArmor enforcement label. Reference: https://linuxcontainers.org/lxc/manpages/man5/lxc.container.conf.5.html . This is currently prohibited by BoxWeave's blanket `raw.lxc` admission; production design needs an **exact, fail-closed, narrowly validated option**, not raw config passthrough.
3. **Budget UID and GID subranges before each new level.** A normal Ubuntu24.04 needs mapped **65534** (e.g. `_apt:x:42:65534` and `dnsmasq` postinst `nogroup`). A too-small 32768 map allowed the child to boot but broke default APT (`setegid/setgroups 65534`) and later `dnsmasq-base` installation (`chown EINVAL`), even when bypassing APT sandbox for one exploratory negative control. Correct solution was a **wide delegated range**, preserving normal APT sandbox and system ownership, *not* changing system accounts or running APT as root permanently.
   - On L1 Incus L2 initially map 65536, then one time safely stopped L2, changed `security.idmap.size=4194304`, restarted; separately read `/proc/self/uid_map` and checked L2 remained `(enforce)`. Restarted L2 own daemon after updating its own `/etc/subuid` and `/etc/subgid` so the daemon actually recognized new ID-map pool (otherwise `System doesn't have a functional idmap setup`).
   - L2 allocated L3 with `security.idmap.base=65536`, `security.idmap.size=4128768`; L2's own root subuid/subgid `root:65536:4128768`.
   - L3 allocated L4 with `base=65536`, `size=4063232`; its own subuid/subgid `root:65536:4063232`.
   - L4 allocated L5 with `base=65536`, `size=3997696`; its own subuid/subgid `root:65536:3997696`.
   Each difference is **65536**. Native launch + real exit0 `exec` PASS at each final level; both the current and the next guest's UID/GID maps were read. UserNS max depth is separately kernel limited (commonly 32; `/proc/sys/user/max_user_namespaces` governs counts and userns nesting is hard-capped), so **there is no justified infinity claim**.
4. **Keep inherited AppArmor as an actual constraint.** The L2 `(enforce)` Profile was created by its parent Incus before AppArmor was disabled in the **nested** daemon. L2, L3, L4, L5 retained that exact label and AA namespace level1; their `/sys/kernel/security/apparmor/policy/namespaces` all had guest W_OK=false. We did NOT flip the L2 Profile to `complain` or `unconfined`. Neither the new policy namespace's independent creation nor per-depth policy authority was demonstrated.
5. **Scope safety.** The switch has an actual security cost: inner Incus no longer generates and loads its own per-child policy or per-helper archive/dnsmasq profiles. Parent L2 enforced policy **still applies**, but it may not protect siblings against each other to the same degree as truly independent per-instance policies; the inherited profile was configured to permit nesting. **Do not advertise this as completed high-assurance BoxWeave isolation.** Complete explicit ancestor/sibling/same-bridge filesystem and security negative tests before product admission, with no general privileged/raw escape hatch.

## Actual positive and negative native gates

| Experiment | Real guest/native result | What it establishes |
| --- | --- | --- |
| Ordinary enforced L2 under separate Incus7.6 L1 with launch-time 65536 IDs | `incus launch` 0; `incus exec` 0; visible `(enforce)` | Baseline parent profile enforced; no complain needed |
| L2's own daemon environment `INCUS_SECURITY_APPARMOR=false` | `systemctl show` `Environment=INCUS_SECURITY_APPARMOR=false`, `incus version` 7.6/7.6 | Local daemon only, not kernel/host |
| L2 own init `dir + incusbr0` | `incus admin init --auto --storage-backend=dir` exit0 | Former dnsmasq policy admin gate bypassed |
| L2 own Incus image child creation, too-narrow IDmap | download/unpack/container DB creation succeeded; `forklxc` failed `newuidmap uid range ... not allowed` | Former archive policy admin gate bypassed, next independent IDmap gate |
| Narrow subordinate range corrected; but no `unchanged` | `Failed to change AppArmor profile to unconfined` EPERM | Existing ancestor enforcement actually blocked transition to unconfined |
| Explicit `raw.lxc=lxc.apparmor.profile = unchanged` in own child | `incus start` and `incus exec` both exit0 | Works while retaining the existing `(enforce)` label |
| Too-small 32768 UID/GID range | default APT `setgroups / setegid 65534` invalid; `dnsmasq-base` postinst `chown EINVAL` | Shallow container boot is insufficient for complete Linux deployment |
| L2 increase to 4194304, L3 map to 4128768 | launch/exec exit0; normal `apt update` 0, normal `apt install incus` 0 | Supported full Linux users, no sandbox bypass |
| L3 independent incusd7.6 + init dir+bridge | all exit0 | Second nested daemon independent and operational |
| L3 creates L4 map 4063232 with inherited profile | launch + exec exit0 | Functional recursion to L4 |
| L4 normal APT Incus7.6 installation + daemon/init | normal apt exit0, daemon env false, init dir+bridge exit0 | Third nested daemon independent and operational |
| L4 creates L5 map 3997696 with inherited profile | launch + exec exit0, same enforce label | Functional recursion to L5; **L5 has no daemon** |
| L1–L5 read-only AA/UserNS/MountNS/cap/DAC matrix | original native readbacks below | New UserNS/MountNS per depth, AA Profile shared and enforced, no local policy path write |

### Actual cross-layer readonly kernel observations

| Layer | UserNS inode | MountNS inode | Guest-visible AppArmor label | Guest root UID/GID map | AppArmor NS level | AA policy namespaces W_OK |
| --- | --- | --- | --- | --- | --- | --- |
| L1 | 4026533791 | 4026533793 | `unconfined` (relative to **its own** AA namespace) | several L1 initial separate mappings incl root0→host1000000 | 1 | false |
| L2 | 4026533859 | 4026533860 | `incus-bh019-noaa-enforce-child1_</var/lib/incus> (enforce)` | `0 1000000 4194304` | 1 | false |
| L3 | 4026533927 | 4026533928 | **same L2** `(enforce)` | `0 65536 4128768` | 1 | false |
| L4 | 4026533995 | 4026533996 | **same L2** `(enforce)` | `0 65536 4063232` | 1 | false |
| L5 | 4026534063 | 4026534064 | **same L2** `(enforce)` | `0 65536 3997696` | 1 | false |

All five reported Guest EUID0, `CapEff=000001ffffffffff`, seccomp mode2 / filter count1, `NoNewPrivs=0`. They **do not have the same host authority**, and this alone does **not** prove independent seccomp filter equivalence or host/sibling security. The AppArmor namespace name stays `incus-user-1001_bh019-76-noaa-1114_<var-lib-incus>` at all depths; thus **the nested layers did not gain per-depth AppArmor Policy Namespaces**. The read-only observation of the namespace write path is **not** a complete LSM negative authorization proof. Original BW op for matrix `1d4b6522d9462eaef694a52baf0b672d`, original native `d3ee40d3-426c-436f-a38e-86cc081eb0c0`, guest exit0. Real userns inodes all distinct.

## Resource and safety notes

The lab used one external managed 2 CPU / 2048 MiB RAM / 12GiB root disk Box. While L5 ran, measured cgroup `memory.current=1686794240`, `memory.max=2147483648`; Linux `free -m` reported 183MiB used processes and significant reclaimable cache, so no claim of unlimited actual available memory, disk, CPU or safe depth. `df -h` displayed underlying host filesystem sizes and is **not evidence the Box lacked a 12GiB root disk quota**. Every level was isolated and on new owned fixtures; nothing on shared L0/L1 existing Boxes changed. Existing BoxWeave native admission disallows arbitrary `raw.lxc`; this is still a **native Incus lab**, not an integrated BoxWeave feature PASS.

## Full cleanup and independent postconditions

Cleanup **bottom-up**, never deleting an ancestor while descendants remained:

1. L4 Incus stopped/deleted only `bh019-noaa-wide-child4`; own `incus list --format=json` independently returned `[]`.
2. L3 Incus stopped/deleted only `bh019-noaa-wide-child3`; independent list `[]`.
3. L2 Incus stopped/deleted only `bh019-noaa-wide-child2`; independent list `[]`.
4. L1 Incus stopped/deleted only `bh019-noaa-enforce-child1`; independent list `[]`.
5. L0 BoxWeave stopped exactly the one-use Box `c9f3912d-f523-4018-947c-ce774fda1537`, then deleted; both original external operations confirmed terminal `SUCCEEDED` and original distinct native IDs.
6. Read-only authenticated inventories post cleanup: L0 **4 pre-existing Boxes**, L1 **9 pre-existing Boxes**, no `bh019-76-noaa-1114`, no new UNKNOWN. Neither historical UNKNOWN nor any old container touched.

## All original external BW/native operation receipts

`work_ref=BH019-INCUS76-NOAA-20261114`, `agent_ref=bh019-noaa-guarded`: **43** original BW operations, **43** separate native Incus Operation IDs, all outer terminal `SUCCEEDED`, no missing native IDs; original `operation_list` had no next page. The command wrappers deliberately report **inner command exit codes separately**; the negative experiments listed as FAIL above are **not successes** merely because the wrapper's own guest process exited0 or the outer BoxWeave operation succeeded.

| # | Actual step / classification | Original BW Operation ID | Original native Incus Operation ID | BW terminal |
| --- | --- | --- | --- | --- |
| 1 | create one-use L1 Box, native isolated Incus, nesting enabled | `69cf4346aa538e68c09563a91ddb8a0f` | `ca1a9c26-c2ed-49d8-8f74-d3885c64f665` | SUCCEEDED |
| 2 | add signed Zabbly 7.6 repo L1 | `256bf91ef7cd9a6ea29ee9162aabc122` | `4f6a8739-dafb-4e08-9f53-57864670dadb` | SUCCEEDED |
| 3 | install Incus 7.6 in L1 | `14d95aae0233068b87138e3c563573e2` | `66da68ef-0872-48c3-946d-658fb6f79286` | SUCCEEDED |
| 4 | L1 Incus 7.6 dir/bridge init | `b358920eca39f41710cf1cb0d72a68d1` | `9c403123-206c-4b3c-8ba0-d55268169f5b` | SUCCEEDED |
| 5 | launch L2, enforced profile, map 65536 PASS | `e856df15f24aa8344cd0bb27edb3a985` | `ed6bd446-25f1-42b9-8275-664dfd9f4671` | SUCCEEDED |
| 6 | add Zabbly repo inside enforced L2 | `be416b28b58a28ff6709d1866fb85ada` | `6e937d93-b980-4eae-80fa-7de60fd26fea` | SUCCEEDED |
| 7 | install Incus 7.6 inside enforced L2 | `ba1863fe340345d46e95adaec34c517d` | `3b7c0323-6bc1-4591-a30e-e8fe0053371b` | SUCCEEDED |
| 8 | inspect L2 service and actual label | `7a6088e78b74a52f93b4fac8166fbdd2` | `673fb3f0-a498-43e0-b42a-658843070def` | SUCCEEDED |
| 9 | L2-only service override INCUS_SECURITY_APPARMOR=false PASS | `b048a1ba282b0381470fdfdff9098081` | `d1ab7d4a-0918-48fa-930c-da9025d1b71b` | SUCCEEDED |
| 10 | L2 Incus init dir+incusbr0 PASS without profile load | `8dfebd0606969a0a2404a00c967a4e37` | `5d1123f4-7051-448c-816d-b4e3035ab8f9` | SUCCEEDED |
| 11 | first narrow child launch FAIL (newuidmap) | `8942e57483140ec26cb6533b379a44a4` | `9b1828d9-4124-4d6b-9b08-669b0579b527` | SUCCEEDED |
| 12 | L3 first failure native log/readback | `ecbc597b7a5a306b503dea8844c56888` | `ee530be1-1a43-4e3f-a4a1-b613d1a679be` | SUCCEEDED |
| 13 | L2 read actual userns/subid/etc | `f3b0bec18a212cab0fac984ed14a654d` | `1a69c15a-5624-47fe-8617-4404ccdbc2d9` | SUCCEEDED |
| 14 | adjust only L2-owned subuid range, start still FAIL | `73a89950eac566c0aa5d783de4c1bdbe` | `e8b8dc8c-a124-460f-8054-1823bf2a9ea8` | SUCCEEDED |
| 15 | new failure LXC change_profile unconfined denied | `ecd728dc0d598513ab9718df02883a00` | `dcffcce8-b2aa-423e-b615-71d19fcf520a` | SUCCEEDED |
| 16 | L3 first child start PASS after raw.lxc=unchanged | `b0c8bfb19c04ce300ff4041d47b58b5d` | `b9cc2b8c-3c81-4f52-a93f-9791b7356fc5` | SUCCEEDED |
| 17 | L3 normal network and uid checks | `a90048f49640e6d683c26f3fb540a42f` | `0453cb0b-7fa6-4570-bda6-77ae149b9a2d` | SUCCEEDED |
| 18 | narrow L3 default apt update FAIL unmapped GID65534 | `51b37f50882a1373bfae37288fa4c29d` | `ebdeec1d-f47c-46ea-94a5-3f1a8c40d5f4` | SUCCEEDED |
| 19 | temporary exploratory apt sandbox contrast (NOT product) | `e50ed3082d702ff46278464463e30a3e` | `645a0977-4256-40ae-9dd5-d0bef37a93d5` | SUCCEEDED |
| 20 | narrow L3 installation FAIL dnsmasq chown unmapped nogroup GID65534 | `d992844e0622b4eb6c869257f34df362` | `2824f375-fbc8-4f88-b42d-29697e47a378` | SUCCEEDED |
| 21 | read dpkg/apt native failure logs | `9445e4636cf50ee8a11c3861f1ed2c18` | `b84d2ce4-b6fe-4b23-b058-a94ad1af6e62` | SUCCEEDED |
| 22 | delete narrow L3, verify its parent inventory empty | `71a6e721d5e4c0d02ec5c122097d294e` | `4c469468-a204-479b-81cd-663f515cfffa` | SUCCEEDED |
| 23 | stop/re-map/restart enforced L2 to 4,194,304 IDs PASS | `745e443a648d2caed41e494f8b189ef3` | `2f358aba-d101-43c4-abfe-c44b4f860312` | SUCCEEDED |
| 24 | read L2 service after restart | `b810571a15662ffab91b2804e63c066d` | `ec7b33d8-5537-408a-a9f5-13d938519b16` | SUCCEEDED |
| 25 | first wide L3 launch FAIL: Incus IdmapSet nil | `48e0a14c46d72dccc0d4ee2e7eb52b3a` | `b93972db-149b-45f8-afb1-0f4043ff53e9` | SUCCEEDED |
| 26 | correct only L2-owned subids and restart its incusd PASS | `71d75dd70191c268477d35b31bfc3a1c` | `c131a49b-0aeb-4a27-a123-46413411659c` | SUCCEEDED |
| 27 | launch widened L3 PASS, inherited enforce | `691448ee5696cfa68fb0d74cf1b701ef` | `79662e90-1e31-4e3b-9a23-6203b653782c` | SUCCEEDED |
| 28 | normal APT update in wide L3 PASS | `ae7469042c0d76b632f007cd734030e2` | `657d186e-7de2-45f7-a02e-9fda4dc9e039` | SUCCEEDED |
| 29 | normal apt install Incus 7.6 in L3 PASS | `5dea3838160ea7822788a2a2e2457848` | `4f214b2a-c1fd-40e8-86ec-f1fcbc49e01a` | SUCCEEDED |
| 30 | L3-only incusd env false, init dir/bridge PASS | `b84726de99dd7b177d97353e0353bf77` | `ca948d08-5782-484d-90aa-b1b86a491fa3` | SUCCEEDED |
| 31 | launch L4 via L3 incusd PASS | `08dec600d7a21c3b5acdef5adcdfbcbe` | `a7d48a03-07e2-42ba-b6d8-ebfcd9a37ec2` | SUCCEEDED |
| 32 | read resource and L4 enforced label | `d69d370b1bc6721888781003476e28fd` | `8acaea0a-24bd-4cdd-b856-02d5d274373a` | SUCCEEDED |
| 33 | L4 Zabbly signed repo and normal APT update PASS | `9e50e7ad3ed48dfceafa04263686370d` | `1552f54d-ef26-473e-be41-3a58e2c2887f` | SUCCEEDED |
| 34 | L4 normal apt install Incus 7.6 PASS | `cb2ee9b789d6e28555a2710ea185cc60` | `7744794a-9a8f-4dd6-aef0-368a1c6d7df2` | SUCCEEDED |
| 35 | L4-only incusd env false, init dir/bridge PASS | `609a5dcb8c7a7820abc93f3665c313c7` | `a87a674f-5ca7-4e92-b79e-91eeb08c106a` | SUCCEEDED |
| 36 | launch L5 via L4 incusd, exec PASS | `c2975084b2f98209a3cea1aee2870fc9` | `ee0f6465-4847-4111-9430-477623f7d608` | SUCCEEDED |
| 37 | read-only L1–L5 labels, userns/mntns, mapping and policy path DAC | `1d4b6522d9462eaef694a52baf0b672d` | `d3ee40d3-426c-436f-a38e-86cc081eb0c0` | SUCCEEDED |
| 38 | stop/delete L5 via L4, verify empty | `8ff70e2229692acdf9fd94e6b503798f` | `8e2615ff-176f-4ebf-9a49-5a93ac6f4b20` | SUCCEEDED |
| 39 | stop/delete L4 via L3, verify empty | `bf15c9285dd9ce468032c9bf482f5d36` | `75e92f5f-d3c7-4377-b80c-1c4959e23da3` | SUCCEEDED |
| 40 | stop/delete L3 via L2, verify empty | `02835853e8360f9e7ac3d2d99db02b46` | `6a8d3b1b-1478-4184-97c1-c91cd38b8703` | SUCCEEDED |
| 41 | stop/delete L2 via L1, verify empty | `22ab8893afb32ab6cfd9f8df887555e8` | `366ab060-0e19-46f8-b566-4a5aa8c70a8c` | SUCCEEDED |
| 42 | stop exact one-use L1 Box | `c9692265be186baf60e072d957393d4f` | `c37c62d0-c522-4b50-9eda-9fe09736602e` | SUCCEEDED |
| 43 | delete exact one-use L1 Box | `200b9fdfbdb411d4c0b4fb60b2ceb155` | `2d2675a1-ea38-4290-be30-2368d42e4274` | SUCCEEDED |

## Shipping decision / follow-up evidence required

- **Functional native recursion: lab PASS up to L5** under one scoped inherited AppArmor `enforce`; independent daemons at L1–L4. This is stronger than prior complain-mode/5× failed policy-manager tests; no patch or privileged downgrade.
- **BH-019 safe recursive isolation: still NOT ACCEPTED.** No independent AppArmor Profile/namespace per nested layer; evaluate exact boundary against stated BoxWeave security model and perform actual sibling and ancestor negative tests, fail-closed start, after restart, concurrent creations and multi-level ops. Consider what security posture is acceptable if all deeper layers intentionally inherit one enforced Profile and skip child per-helper AppArmor profiles.
- **Implementation:** Introduce only reviewed, typed narrow-recursion options. Never expose arbitrary `raw.lxc`, `raw.apparmor` or privileged settings through general MCP; only delegate exact vetted `unchanged` when inherited parent enforce proven; place the Incus environment variable only in the nested daemon service, not the physical host or parent manager. Allocate/validate subordinate range with a per-level 65536 reserve and verify `/proc/self/{uid,gid}_map`; plan concurrency and UID-range exhaustion. This design is a **proposal**, not an accepted BoxWeave code change.
- **Max depth**: L5 actually verified. The Linux User Namespace nesting cap often defaults to **32**, subject to kernel configuration and resource limits; no extrapolation of L5 to 32 as tested.
- Keep PR #146 **Draft, unmerged**, and no green CI claimed for docs-only commit.
