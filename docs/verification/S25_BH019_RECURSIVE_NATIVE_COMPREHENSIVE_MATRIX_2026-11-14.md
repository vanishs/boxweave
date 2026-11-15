# BH-019: recursive Incus 7.6 comprehensive native acceptance / security matrix (2026-11-14 PT)

**Decision: FUNCTIONAL LAB PASS FOR TESTED PATHS; DEFAULT SIBLING NETWORK ISOLATION FAIL; BH-019 SECURITY ACCEPTANCE REMAINS BLOCKED.** This is follow-up to [real L5 recursion lab](S25_BH019_INCUS76_INHERITED_ENFORCE_RECURSIVE_L5_2026-11-14.md) and the actual [complain contrast](S25_BH019_INCUS76_COMPLAIN_RECURSION_LAB_2026-11-14.md). This report does not claim integrated BoxWeave API acceptance, arbitrary depth, separate per-layer AA namespaces, or a full adversarial isolation proof.

## Exact controlled setup and scope

- New one-use L0 BoxWeave RC5 Box: `bh019-acceptance-lab-1114`, exact UUID `e8257efb-62d9-4ac6-9262-77d820a64d6f`; unprivileged, `security.nesting=true`, CPU2, RAM3072MiB, root16GiB, independently authorized temporary internet. Not an existing/shared L0 or L1 box.
- L0 native Incus version 7.6; new L1 disposable Box installs Zabbly stable Incus **7.6** using signed repo, with independently initialized `dir` pool and `incusbr0`. Its first child `bh019-acc-L2` runs with an Incus-generated AppArmor **(enforce)** Profile; actual guest UID/GID map `0 1000000 4194304`. The L2 daemon uses the official `INCUS_SECURITY_APPARMOR=false` **only inside L2's own `incus.service`** and its own subuid/subgid range `root:65536:4128768`. L2's independent `dir` pool and `incusbr0` init PASS.
- Inside L2, two real siblings **A** (`bh019-acc-A`, UID/GID host-relative base65536 size1048576) and **B** (`bh019-acc-B`, base1179648 size1048576) are both nonprivileged `security.nesting=true`, exact `raw.lxc=lxc.apparmor.profile = unchanged`, `limits.cpu=1`, `limits.memory=512MiB`. Both inherited the same L2 enforced profile; no per-child new policy is loaded by L2's daemon.
- The upstream official LXC `unchanged` mode deliberately inherits the parent's current enforced profile; no parent global AppArmor/profile or host Incus policies were changed. This native L2 experiment explicitly bypasses BoxWeave's normal blanket denial of raw config only within its **fully owned internal Incus**. **Do not expose a generic `raw.lxc` MCP field**. The corresponding product interface is not implemented/accepted in BoxWeave.

## Actual positive, negative and incomplete acceptance matrix

| Area / test | Real outcome | Classification / release implication |
| --- | --- | --- |
| Independent 7.6 L1/L2 daemon setup | Both real 7.6/7.6; L2 own dir and bridge init exit0 | **PASS** in one-use native lab |
| Concurrent creation | A and B launched simultaneously (max workers2), both native exit0, 37.1s elapsed wall; both subsequently `exec` exit0 | **PASS** for two concurrent children; not large-N/race stress |
| UID/GID isolation | A `0→65536` 1048576, B `0→1179648` 1048576, ranges disjoint; distinct UserNS and MountNS inodes | **PASS** for explicitly assigned maps |
| Nested root != physical host root | A/B guest UID0 show local capabilities, unique userns and non-initial relative maps | **Supported**; does not independently prove all host escape properties |
| AppArmor inheritance | Both `incus-bh019-acc-L2_</var/lib/incus> (enforce)`, AA NS level1 | **PASS** for ancestor enforcement inheritance; **FAIL TO MEET** per-depth independent AA namespaces |
| AA policy-directory ability | A/B `os.access(.../policy/namespaces, W_OK, effective_ids=True)=false` | **Read-only negative PASS**, not full kernel policy-admin penetration test |
| Parent Incus Unix socket exposure | `/var/lib/incus/unix.socket` absent in A/B | **Path-level check PASS**; no generalized parent mount attack testing |
| File-system private canary | A private canary not visible by pathname in B; B private canary not visible in A | **Basic PASS**; not a full mount/proc/file descriptor adversarial suite |
| CPU quota / affinity | `limits.cpu=1` led to `os.sched_getaffinity(0)` count1 on A/B, before and after recovery | **PASS** for CPU affinity, **not** proof of quota-based `cpu.max` (observed `max 100000`) |
| Memory quota | A/B cgroup `memory.max=536870912` bytes, stable after recovery | **PASS** for configured limit, **not** OOM/sustained load enforcement |
| Default sibling L2 bridge reachability | B connected to an A-owned, explicitly benign test server at IPv4 `10.216.165.120:18281` and GUA IPv6 `fd42:d477:1f3e:31b6:1266:6aff:fed0:5162:18282` | **FAIL** against default sibling isolation expectation; ordinary native Incus bridge is intentionally shared, this is not a claim Incus violates its own contract |
| BoxWeave L1 internet on → off | Before: external HTTPS from L1 and descendant B succeeded. After L0 `box_update(internet=false)`: both external HTTPS attempts timed out, with no successful request | **PASS** for tested TLS egress blocking across nested NAT; not exhaustive all ports/protocols |
| Sibling traffic with L1 internet off | After the above switch, B→A IPv4 and IPv6 test connections still succeeded | **FAIL** if assuming internet-off implies sibling local isolation; those are separate policies |
| BoxWeave L1 internet off → on | B external HTTPS to signed upstream key endpoint succeeded again | **PASS** for tested toggle restoration |
| Normal nested incusd restart | L2 daemon service restarted; A and B remained Running with unchanged native UUIDs, own canaries, exact enforced labels | **PASS** |
| Hard-crash nested incusd recovery | Exactly L2 `incus.service` MainPID **3709** received SIGKILL within the disposable L2; `systemctl restart` exit0, new MainPID **3896**, service environment retained, A/B `exec` and canaries passed | **PASS** for daemon-local crash/restart, **not** in-flight durable operation recovery |
| Fixed map conflict admission | `incus init --empty` accepted a **stopped** C config overlapping A's fixed range, and accepted a second stopped C config extending beyond the parent-owned map; both `volatile.idmap.next` records materialized; test records deleted | **GAP / design warning**: native Incus creation alone does **not** reject all explicit conflicting/out-of-bounds maps. No such bad-mapped guest was started. BoxWeave must preflight/deduplicate and distinguish persisted config from actual kernel-accepted start |
| Nonstateful snapshot / restore | `incus snapshot create A snap1` 0, canary changed, A stopped, `incus snapshot restore A snap1` 0, A start and exec0, canary restored, enforced label + cgroup memory/cpu intact, B remained running; snapshot deleted | **PASS** for `dir` filesystem rollback; **not tested**: CRIU memory-stateful snapshot, other storage backends, cross-host restore |
| Entire parent L2 stop/start | L1 stopped then started exact L2; L2 `enforce` retained; A/B with explicitly `boot.autostart=true` returned Running, native instance UUIDs and canaries unchanged | **PASS** after boot stabilization; first immediate `systemctl` probe raced systemd socket readiness |
| Service after whole-parent restart | L2 `incus.service` active, version 7.6/7.6, `Environment=INCUS_SECURITY_APPARMOR=false` verified from systemd *and actual incusd process environment* | **PASS** after stabilization |
| Ubuntu `systemctl is-system-running` after L2 restart | Returned **degraded**: `apparmor.service` failed. Boot log: stock system service tried to replace ordinary distro profiles; `apparmor_parser: Access denied. You need policy admin privileges to manage profiles.` | **REAL BOOT COMPATIBILITY DEFECT**; inherited ancestor enforcement remains active but nested distro AA policy loader is independently not functional. Needs an approved per-guest handling/monitoring policy, not global host AA weakening |
| Per-instance NIC ACL, only B IPv4 and global IPv6 CIDRs | Incus ACL created in disposable L2; attached **directly to A's overridden NIC**. B→A IPv4 and global IPv6 connection timed out; A local listener and outbound HTTPS passed | **PARTIAL PASS**; requires NIC-level ACL (network-wide bridge ACL cannot enforce intra-bridge according to upstream) |
| B→A IPv6 link-local after first ACL | B reached A server using A's `fe80::1266:6aff:fed0:5162` with interface scope, receiving exact test reply | **FAIL / BYPASS OF TOO-NARROW ACL**: global-IP-only deny rules are insufficient |
| Add `source=fe80::/10` to same A NIC ACL | B IPv4, GUA IPv6 **and link-local IPv6** all timed out; A local listener and outbound HTTPS still worked | **PARTIAL FIX PASS for these probes**, NOT proof of robust general network isolation, spoof resistance, RA/NDP/DHCP invariants or other protocols |
| Restart L2 incusd with NIC ACL active | Native rules/config persisted, B's tested IPv4/GUA/link-local connections remained denied | **PASS** for daemon restart persistence |
| ACL undo causal control | Removed exactly A local NIC override and deleted exactly test ACL. B reached A test server again over IPv4 and IPv6 | **PASS**: demonstrates observed denial actually due to own ACL |
| End-to-end cleanup | Exact A/B stop/delete; L2 independent `incus list=[]` and ACL list `[]`; exact L2 stop/delete; L1 `incus list=[]`; L0 controller stop/delete exact owned Box; L0 inventory 4 unchanged and L1 inventory 9 unchanged | **PASS**, no new UNKNOWN, no historical UNKNOWN replay |

### In-depth interpretation and high-priority engineering actions

**P1: network.** Official [Incus network ACL documentation](https://linuxcontainers.org/incus/docs/main/howto/network_acls/) explicitly states that **network-level bridge ACLs cannot filter same-bridge sibling traffic**, but **NIC-attached ACLs can**. Our actual Incus 7.6 A/B tests reproduced both the default shared-bridge reachability and NIC ACL effectiveness. The IPv6 link-local bypass proves why "ban B's assigned global addresses" is not a complete fail-closed isolation policy. Required before shipping an isolated Box contract: a per-instance deny-by-default NIC policy, tested full IPv4, GUA IPv6, LL IPv6, DHCPv4/v6, RA/NDP, DNS, WAN, host-side and sibling ingress, service exceptions, dynamic address changes and active + post-restart enforcement; set and test source MAC/IPv4/IPv6 anti-spoofing and all possible ingress paths. Do **not** treat a few address-specific drop rules as a completed mitigation. Clarify intended tenant-to-tenant communication and safe opt-in.

**P1: UID/GID maps.** Reserve, validate and atomically persist per-tenant/per-depth subuid and subgid ranges, with explicit collision/overlap and bounds checks before native creation. The two bad fixed-map configurations passed native `incus init --empty` and were destroyed without start, so this is an **admission semantics observation**, not proof that the kernel would allow a bad map. The prior L5 lab proved at least 65536 reserved IDs per descending level were necessary for ordinary Ubuntu account GID65534 and distro package install; correctly size/cap the maximum possible depth and resources. Never invent unbounded UID allocations or pass arbitrary `raw.idmap`.

**P2: boot semantics.** The nested daemon-only environment flag prevents *Incus's* new policy loads; it does **not** stop Ubuntu's independent `apparmor.service` from attempting profile replacement at guest boot. That unit failed and left systemd `degraded`, despite all inner Incus functional gates passing. Investigate a scoped, well-documented, auditable guest boot strategy, not any physical-host kernel AppArmor switch or general profile escalation. Preserve the ancestor's `(enforce)` at every point.

**P2: recovery and state.** Normal daemon restart, SIGKILL recovery, snapshot rollback, and full parent stop/start passed this fixture. Future production gate must validate unexpected host/reboot/crash with in-flight create and restart, ACK loss/UNKNOWN classification and independent repair, repeated full chain restarts, rolling upgrade, and exact BoxWeave/MCP lifecycle rather than only native Incus CLI.

**Known limits:** Guest A/B had the same ancestor AppArmor `(enforce)` label, not separate per-tenant AppArmor namespaces. Per-sibling FileSystem canary absence, UserNS/MountNS separation and no visible parent socket are **not full ancestor/sibling security attestation**. Explicitly excluded/not tested: SELinux, privileged or host AA weakening, packet/IP/MAC spoofing, comprehensive cross-namespace filesystem escape, forced guest OOM, disk quota exhaustion, high-N races, stateful snapshot, other storage drivers, other kernel/architecture, complete installed HTTP MCP BoxWeave-to-BoxWeave recursion, and depth beyond prior L5. Keep BH-019 SECURITY **BLOCKED**.

**Safety and test-harness classification:** All work stayed in an owned disposable L0-managed Box and its own native child Incus project. No shared profile, outside host or existing L1 Box modified. Test harness made one initial benign Python listener syntax error and one outdated `incus snapshot` CLI invocation (the correct `incus snapshot create` later passed). One initial immediate systemctl read after full L2 restart failed due to service readiness; later independent readings confirmed active incusd and environment. These are recorded as **harness/tooling behavior**, not product PASS or false AppArmor fault. Every outer BW Exec wrapper had separate guest output and genuine inner `rc` captured; outer `SUCCEEDED` means transport success only.

## All original BoxWeave/native Incus receipts

`work_ref=BH019-RECURSION-ACCEPTANCE-20261114`, `agent_ref=bh019-acceptance-matrix`. **42** authenticated original BW operations, each terminal **SUCCEEDED**, each with an original immutable, distinct native Incus Operation ID, zero missing IDs, one complete operation-list page. These original outer receipts must **not** be equated with all inner test commands passing; the matrix above is authoritative for the command's actual success/error, relevant errno, and validation.

| # | Test step / classification | Original BoxWeave operation | Native Incus operation | Outer terminal |
| --- | --- | --- | --- | --- |
| 1 | Create exactly one L0-owned L1 test Box with nesting and quota | `8435c4cfbc403fea13fb1cc836f773a8` | `cf6dc8bf-8758-4a52-9d5d-941470bb0aff` | SUCCEEDED |
| 2 | L1 signed Zabbly stable repo; confirm candidate 7.6 | `04cce9f4fb1f1de65887bea69bf94eed` | `0d609c81-eecd-48da-b0aa-b933c4f01b63` | SUCCEEDED |
| 3 | L1 normal APT install Incus 7.6 / dnsmasq / uidmap | `c83b4c2678efc3ee8126e6a499b11f6f` | `0d5973e9-7b43-4966-bac2-fd62fd1492e2` | SUCCEEDED |
| 4 | L1 independent Incus 7.6 daemon / dir / bridge init | `4fb03a8cb50233154ffdac52ede662f2` | `39856190-1deb-4d73-a133-b5c8e7901f35` | SUCCEEDED |
| 5 | L1 launches enforced L2 with 4194304 UID/GID | `573a960db58934fe97b50d709336c151` | `993dcefd-d585-4b31-b180-1ef4209b76dc` | SUCCEEDED |
| 6 | L2 signed 7.6 repo, normal APT update | `7c167e4a940e452ebdd7b784014a5d7d` | `3285263f-5382-4147-befc-954a593763e6` | SUCCEEDED |
| 7 | L2 normal APT install Incus7.6 | `372fc26291d1614beefbbb1aa0449347` | `83b476fa-1c26-4059-9d46-72e4cfabb1e0` | SUCCEEDED |
| 8 | L2 exact daemon-only noAA + subuid/gid budget + dir/bridge init | `87077aee59557fd66aa3a49c937d2b5f` | `2d6032ec-0d74-4b2d-b634-6ab59d7e700f` | SUCCEEDED |
| 9 | L2 native concurrent launch of A/B; both inner exit0 | `42eb46a5f2c961dcd498543c4aaeaf75` | `1e00681e-e223-489e-8650-5657e45b4e03` | SUCCEEDED |
| 10 | Read A/B distinct userns/mntns, UID/GID, mem/cpu, policy path | `d0d76006b23308aafbed43b7aee6af58` | `dc1d0e92-b2d7-4e32-ad04-796a949481a3` | SUCCEEDED |
| 11 | First simple TCP harness attempt: Python syntax failure (NOT traffic verdict) | `271e80b20dda07bdc28af41c37ec0555` | `dceee688-a9f7-46fe-990c-6c2512b70a3e` | SUCCEEDED |
| 12 | Corrected A/B private-file and IPv4/IPv6 sibling TCP A/B | `8c06fb21584b3863e1b6659d15e3e59b` | `2c8b6c17-338f-4b68-aa69-bc8b78ee3284` | SUCCEEDED |
| 13 | Outer L1 + nested B baseline outbound HTTPS success | `e262797832848274744fdf8aea8180ee` | `a9c1e7a1-f458-4a5b-85c9-7c6be4126a74` | SUCCEEDED |
| 14 | Exact L1 BoxWeave internet true->false | `d792843a2d161ca7ec4f106b5dcf71af` | `c69128cd-7038-4eb7-873c-adc16de7eea3` | SUCCEEDED |
| 15 | Outer L1 + nested B HTTPS denied; sibling local remains reachable | `7a87ee00ca2623ed62cc3bab47a1d609` | `ff89f1ae-3df7-4ce1-8866-6d4d7254293e` | SUCCEEDED |
| 16 | Exact L1 BoxWeave internet false->true | `a5ad6c05e2e99223e6e9743da9022fcd` | `1daa6e60-7126-4f7c-985b-7caef3d6188f` | SUCCEEDED |
| 17 | Nested B external HTTPS restored | `d79201d15acbaebc91ab756872a576cd` | `d3223202-c5e7-418a-8925-643b4e254811` | SUCCEEDED |
| 18 | Normal L2 incusd restart; A/B UUID/canaries/profiles retained | `b5073bb5b2f9c209df867b9b69517f01` | `f43e24fb-74d9-4520-8851-66f5bad1d457` | SUCCEEDED |
| 19 | Native Incus init --empty accepts overlap and outside-range fixed maps; delete test records | `1e98e5bd0d4cc55a8239280c88553f3b` | `08b23285-1f37-4f9e-bd10-cfbda603d49c` | SUCCEEDED |
| 20 | SIGKILL exactly L2 incusd main PID; daemon restart; A/B canaries/profiles intact | `98ed91025abd44db8316296591dbd7e6` | `f46f34b3-29c5-4454-bcc5-a30d2e3d6e32` | SUCCEEDED |
| 21 | Old incus snapshot CLI syntax rejected (harness correction, no snapshot) | `4179ceb0c6705c5d5bf74c50d1c79e43` | `7089d6af-9b98-45d0-9aaf-e8c50a97a2f6` | SUCCEEDED |
| 22 | Read 7.6 snapshot subcommand help | `196e5bffd0727e20ff0598ba31025c5e` | `8883f8f6-88db-4645-9a2c-73ebe577b121` | SUCCEEDED |
| 23 | Actual nonstateful snapshot A created | `8b0af274b8e238620bde99ef0b87b9dc` | `53c36e09-f821-4929-8ccf-478ebbef78cc` | SUCCEEDED |
| 24 | Modify canary, stop A, restore snapshot, start, verify rollback, delete snapshot; B unaffected | `082c70fecd61dd55ed836dcb54723466` | `7cab4a19-00d8-4135-97c0-afd657aff5f8` | SUCCEEDED |
| 25 | Enable boot.autostart=true on A and B | `33893484b1f0479cdd3af777be2d911e` | `0b1089c0-1a53-4ad2-a1dc-c43e3f0700f1` | SUCCEEDED |
| 26 | Entire L2 stop/start; A/B exec and canaries recover; initial systemctl transient | `2ab17d31f41eb1a9a213ddf26a656483` | `056c2c41-afbe-40a6-bad3-ba5fde2ca860` | SUCCEEDED |
| 27 | After boot recheck incusd env, A/B Running, same UUID; systemd degraded | `067f30ca9dcb4badb4286a6db4a8fdac` | `8f9317e5-ce63-4a16-a298-5579067b5e5f` | SUCCEEDED |
| 28 | Read-only failed systemd unit inventory | `452442ef3ac8d08b2e2b594bdc389e3a` | `56b38860-4662-4fce-b0e3-483a8eefcd0f` | SUCCEEDED |
| 29 | Read boot apparmor.service parser policy-admin denial | `ed0f435f1a4ae784586357d0106acb79` | `2d0cef3a-a570-47b7-98c0-75c7c8607382` | SUCCEEDED |
| 30 | Inspect own L2 network ACL/NIC/profile config | `d9e8d6b2ebeea0e7e9477a26b40ff345` | `6900e1b1-5ca3-4994-ae7b-6da9c16f50bc` | SUCCEEDED |
| 31 | Restore A-only benign TCP listeners; B baseline IPv4/IPv6 reachability | `f7927b129d0874be18f12b5f55ca6cac` | `5d18cb7a-503b-40d4-8566-06f03b96dbf7` | SUCCEEDED |
| 32 | Create owned per-NIC ACL; drop B IPv4+GUA; override A NIC, default allow | `29bfd5a0e7c39d4c99b12e2533ced5b3` | `a0e9f26d-4a20-4450-940c-a7167a9e0ac8` | SUCCEEDED |
| 33 | B IPv4/GUA blocked; A localhost+HTTPS pass | `9ce5e6422bb9789524e656a52b2388b5` | `0541a299-5bf0-4fe2-86d0-c5a908162195` | SUCCEEDED |
| 34 | B to A IPv6 fe80 link-local still passes: security gap | `f3164b08a0f285b3b7b8ded5754c1edb` | `c2adf6d0-0b6f-417a-a779-a796594d6517` | SUCCEEDED |
| 35 | Add fe80::/10 ingress rule to one A NIC ACL | `7ae2725ce21cecabf2ff71b113f2f5b4` | `3d5f702e-e268-4463-85ec-ec1c9de24416` | SUCCEEDED |
| 36 | B IPv4/GUA/linklocal now blocked, A local+outbound works | `61d532b5e2baed5d0f0082bac7ece02a` | `2e173b6a-92b6-4379-a82b-34820cac0c88` | SUCCEEDED |
| 37 | Restart L2 incusd; ACL present and blocks tested three paths | `a14997dd0f3cd024e75a1496a5e39969` | `32be8704-ea31-48c5-898d-55efa00a82a1` | SUCCEEDED |
| 38 | Remove exact A NIC override + exact ACL; B IPv4/GUA reachability returns | `b40d7dfcbd79306d4f70a4973c35c9cd` | `7dcf5281-ccc2-47c1-b11d-01cd0e6a5ee2` | SUCCEEDED |
| 39 | Stop/delete A/B; L2 native child inventory + ACL list empty | `be1235bf8b18155d504b76a573f49b92` | `9beef3a2-bcb3-4a4a-a0e4-5c777d4aa00c` | SUCCEEDED |
| 40 | Stop/delete L2; L1 native inventory empty | `a73510747e3d726637dfe43162138906` | `59e3811f-2212-4fd6-a07a-673148ccc826` | SUCCEEDED |
| 41 | Stop exactly the one L0-owned L1 lab Box | `4aa58bda1e9c37fdf6e6278eb5606f0a` | `da605094-7fa1-4296-9033-40c9485001b2` | SUCCEEDED |
| 42 | Delete exactly the one L0-owned L1 lab Box | `25d2dc7b9794b3cd35f1bdd5369c8478` | `debfdaa5-0c6b-469b-8821-580d05efbd06` | SUCCEEDED |

**Final ledger:** all experimental nested instances removed bottom-up; original L0-owned Box stopped/deleted successfully; independently verified L0 **4** unchanged, L1 **9** unchanged. PR #146 remains **Draft**, not merged, BH-019 **BLOCKED for isolation acceptance**. Documentation-only commit; no claim of new Actions success.
