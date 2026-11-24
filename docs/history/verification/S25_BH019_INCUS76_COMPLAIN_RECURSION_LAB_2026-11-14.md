# BH-019 — actual Incus 7.6 recursive AppArmor complain experiment (2026-11-14/15 PT)

**Outcome: first real nested container successfully runs under complain; the next Incus 7.6 level remains blocked by AppArmor policy-administration `apparmor_parser exit 243`.** This is the **first actual per-Box complain** contrast in the BH-019 N×M program; prior Incus 6.0 lab did **not** reach complain. All commands ran inside a new disposable nonprivileged Box; no host-level direct Incus admin command, shared profile mutation, privileged host security change, or old Box touched. BH-019 still **BLOCKED**.

## Controlled topology and versions

```
Physical L0 (Incus 7.6; BoxWeave L0 RC5)
└── new one-use Box bh019-incus76-complain-lab-1114
    (UUID fa565109-7ca0-4987-a9bc-36e47028e23a;
     nonprivileged, security.nesting=true, CPU2, RAM2048MiB, disk12GiB,
     isolated BoxWeave-owned AA namespace level1, relative unconfined)
    └── independently installed Zabbly Incus 7.6, local dir pool + incusbr0
        └── Incus child bh019-incus76-child1 (RUNNING then safely deleted)
            (security.nesting=true, security.idmap.size=65536 at initial launch,
             actual uid_map "0 1000000 65536";
             generated incus-bh019-incus76-child1_</var/lib/incus> profile)
            └── child1 independently installed Incus 7.6
                ├── its own incusd and dir pool WORK
                ├── incusbr0 initialization BLOCKED on dnsmasq AppArmor policy load
                ├── normal image-based child2 BLOCKED on archive AppArmor policy load
                └── empty child2 CREATED but cannot START:
                    pre-start generated AppArmor profile load denied
```

Zabbly stable apt candidate and actual installed client/server: `1:7.6-ubuntu24.04-202611072126`, verified `Client version: 7.6; Server version: 7.6` at **both** independent Incus daemons. Ubuntu 24.04. Kernel `7.0.0-43-generic`. All source keys/repos and packages installed **only inside disposable Boxes**. Initial L0-owned Box had internet enabled for installation, but no public ingress or shared host profile edits.

## Stepwise original-native findings

1. In a fresh one-use outer Box, signed Zabbly Incus stable repository reported Incus 7.6. Installed Incus 7.6, `uidmap`, `dnsmasq-base`, `apparmor-utils`; `incus admin init --auto --storage-backend=dir` **succeeded**.
2. Real first child `incus launch images:ubuntu/24.04 bh019-incus76-child1 -c security.nesting=true -c security.idmap.size=65536` **succeeded**. Crucially, setting `security.idmap.size` **at initial launch**, not after a failed start, produced the actual native `volatile.idmap.current/next` range **65536**, rather than previous Incus6 lab's billion-ID range. `incus exec child1 -- cat /proc/self/attr/current` succeeded and returned `incus-bh019-incus76-child1_</var/lib/incus> (enforce)`; visible AA namespace level1.
3. Verified the generated AppArmor file belongs to **only this one-use child** and the outer Box's `.ns_name` explicitly belongs to the new Box. Ran exactly `apparmor_parser -r -C /var/lib/incus/security/apparmor/profiles/incus-bh019-incus76-child1` in the disposable parent. **exit 0**. Independently read both `aa-status --json` and the running child's `/proc/self/attr/current`: **`incus-bh019-incus76-child1_</var/lib/incus> (complain)`**, `aa-status` matching `complain`. This **is** a genuine runtime AppArmor mode switch for a running specific child, not a global or relative `unconfined` guess.
4. From within the **complain-mode child1**, independently installed Zabbly Incus **7.6** and started its own incusd. Child1's `incus version` returned 7.6/7.6. Its `incus admin init --auto --storage-backend=dir` failed when creating local `incusbr0` because `apparmor_parser -rWL .../incus_dnsmasq-incusbr0` returned **243: Access denied. You need policy admin privileges to manage profiles**. The local `dir` pool was nonetheless created; default root disk subsequently configured successfully.
5. To bypass unrelated network setup, attempted one offline child2 `incus init images:ubuntu/24.04 bh019-incus76-child2 -c security.nesting=true -c security.idmap.size=32768 -c security.idmap.base=10000` under child1. Failed on **archive unpack** before container creation: `apparmor_parser -rWL .../incus_archive-<unique>` returned **243 / policy admin access denied**. This proves the archive's temporary AppArmor profile is a separate gate, independent of dnsmasq.
6. To avoid the archive entirely, `incus init --empty bh019-incus76-child2-empty -c security.nesting=true -c security.idmap.size=32768 -c security.idmap.base=10000` **succeeded**. Its actual `volatile.idmap.next` was mapped to `Hostid=10000, Maprange=32768` for both UID and GID, showing the nested Incus accepted a bounded in-namespace map. But `incus start` **failed**. Native LXC log showed `lxc.hook.pre-start` failure; independent `journalctl -u incus.service` provided the exact underlying error: `apparmor_parser -rWL .../incus-bh019-incus76-child2-empty` returned **243 / policy admin access denied**.
7. As a single explicitly less-secure **one-use-only negative control**, set `raw.lxc=lxc.apparmor.profile = unconfined` on this **stopped, empty, never-started child2**, then retried `incus start` once. It **still failed** in the same pre-start stage, and daemon journal showed the same generated-profile parser243. **This raw.lxc override did not bypass the Incus profile-loader path.** This is **not** a proof that an actual running child with a different AppArmor profile can be safely delegated. No privileged container, `security.privileged=true`, global complain mode or host unconfined switch was used.
8. The independent child1 daemon also emitted `Per-container AppArmor profiles are disabled because Incus is already protected by AppArmor`. Importantly, this **warning is not sufficient evidence that all profile loads are skipped**: actual native dnsmasq, archive and child pre-start paths still called `apparmor_parser` and were denied. A precise Incus 7.6 source callgraph and profile-generation/loader separation is a follow-up.
9. **All experiments fully cleaned**: deleted stopped inner `child2-empty` and independently verified child1's `incus list=[]`; stopped/deleted running `child1` through its own outer Incus and independently verified that Incus's `incus list=[]`; finally BoxWeave L0 Stop/Delete on only the newly created outer Box both returned original native `SUCCEEDED`. Post-cleanup authenticated inventories exactly **L0=4 historical, L1=9 historical**. No new UNKNOWN, no old UNKNOWN replay, no shared host or L1 inherited AppArmor policy modified.

## Interpretation and next direction

**The user's complain idea was tested successfully as a real per-Box mode switch, but it does not confer policy-admin authority to that child.** It can run independent Incus 7.6 but cannot use several normal Incus container creation paths because each loads/replaces AppArmor profiles in the *inherited* policy namespace. The previously unverified Incus 6.0 mapping blocker is separable: Incus7.6 first child works with a launch-time 65536 map; the deeper empty instance also accepts a correctly bounded map, but its start is denied **earlier at profile load**. This experiment does not establish an absolute maximum recursion depth; it establishes **this exact default 7.6 path** is blocked at the next child. To progress safely, source-audit Incus 7.6 `archive`, `dnsmasq`, and `pre-start` AppArmor load calls, evaluate an explicit **no-policy-load** mode limited to already-confined parent children, or parent-created child-owned policy namespace delegation. No proposal is validated without checking ancestor/sibling confinement and real running recursive children.

**Do not infer that AppArmor complain implies `CAP_MAC_ADMIN` or `AA_MAY_LOAD_POLICY`**; the live parser243 contradicts it. Do not globally switch host AppArmor mode to reproduce.

## Original BoxWeave and native Incus receipts

`work_ref=BH019-INCUS76-COMPLAIN-20261114`, `agent_ref=bh019-incus76-lab`; **33** original authenticated BoxWeave Operations, **33** original native Incus Operation IDs, all outer terminal **SUCCEEDED**; original Exec wrapper outputs capture and distinguish **inner incus exit1** and parser243 from outer command transport success. The inner daemon's own original `incus` errors and journal are transcribed above.

| # | BW action | Original BoxWeave operation ID | Original native Incus operation ID | Outer terminal |
| --- | --- | --- | --- | --- |
| 1 | `box_create` | `420c8251c31a144cb2873e79163b2285` | `909407b8-1131-44da-9335-eb703b8a3a11` | SUCCEEDED |
| 2 | `box_exec` | `3238dece127057e9fe0705c3c5b684a9` | `67b691fe-1e91-4270-8546-944cd1e32c9e` | SUCCEEDED |
| 3 | `box_exec` | `e9437c3a1245adcafef6ba1567f3d501` | `097f120c-77ba-44ff-bf8d-52285653fec8` | SUCCEEDED |
| 4 | `box_exec` | `b8b8d997a330859d9cdc9830ecd8dc35` | `84214bc0-d4a7-4105-a14e-fe8192f5c5f5` | SUCCEEDED |
| 5 | `box_exec` | `f4a8881676dc67a1cc6f8f53b1cc6c4b` | `565fff94-569f-487a-8c43-95c357c61793` | SUCCEEDED |
| 6 | `box_exec` | `b56912c350ec63b9cb71921547d47383` | `7a716593-a66d-4a9f-bf7c-94db70a9bb46` | SUCCEEDED |
| 7 | `box_exec` | `2e51a3faba6a14a7ae69ea18e78e1360` | `fd2007ea-1e9a-444d-8470-c53d285509e2` | SUCCEEDED |
| 8 | `box_exec` | `299930b89ea88650e28a4f484b872b41` | `b1f7033b-8b37-41da-9081-fbce510e3c4b` | SUCCEEDED |
| 9 | `box_exec` | `27af67c774abca4f491decf4f2c4ca58` | `cf07a0ac-732d-4e96-b424-3cb261a42f1d` | SUCCEEDED |
| 10 | `box_exec` | `ee9ff64778480cb3377d4dd0cb8d76ba` | `b509b24e-556b-4f8b-a599-b1dfdcf52e9a` | SUCCEEDED |
| 11 | `box_exec` | `521b88660421f4e7e2a4ff561f594b10` | `6d352081-ef4e-4bf8-84eb-be3507fca285` | SUCCEEDED |
| 12 | `box_exec` | `f4c829614392615263174fd9411d1bb8` | `2885b192-c19e-473b-b732-1f754e3605e1` | SUCCEEDED |
| 13 | `box_exec` | `69917e3d20dcf2344c59d21874b798be` | `8562bd16-c581-4590-be99-d214b1e6ae76` | SUCCEEDED |
| 14 | `box_exec` | `4db976d349c02bd57922fffe81b59959` | `45c046cc-9cf1-43ec-8b9c-ed00bd12c083` | SUCCEEDED |
| 15 | `box_exec` | `4a598a1c66cfc7be240b2dfb2719e7d8` | `08897772-af3c-4558-bbbd-3ed09c11b3da` | SUCCEEDED |
| 16 | `box_exec` | `73cb2e713d6bdd89c895e8f28db2c4af` | `56ea5224-bc68-4c20-b24a-d1a9ed925f4a` | SUCCEEDED |
| 17 | `box_exec` | `0ac428ed11224857eb7809a491b1afbb` | `e129cc99-c056-474d-a1a3-6880e6fabd0e` | SUCCEEDED |
| 18 | `box_exec` | `538d39c4a1ab6bba6e6c74505ca6d8e3` | `51bbf4c7-1cda-4bb9-8acd-3d72eaa62ec6` | SUCCEEDED |
| 19 | `box_exec` | `54ee4e370dda3910abd94ad967e31ee5` | `729babc1-8a65-4806-ab58-da684bd9ed7e` | SUCCEEDED |
| 20 | `box_exec` | `40dab9f59c993168f8742253c616dc4f` | `99124f2d-a31d-4b33-b33a-e6edc0454b3c` | SUCCEEDED |
| 21 | `box_exec` | `0261b3cd1d0186acebcf842d4616489a` | `45f63e70-5851-4af1-a0ed-5a8d2f5eaefa` | SUCCEEDED |
| 22 | `box_exec` | `ba4794603a96beb67d520dfa9c226bd0` | `bc2b0c56-9a7b-4124-b297-38b19a378973` | SUCCEEDED |
| 23 | `box_exec` | `e92faac5b2986c6efec389140aaa6e66` | `2e49ef05-6ec7-4c84-bc6f-f99a14b5c6cc` | SUCCEEDED |
| 24 | `box_exec` | `ce8a64a1442bee2445cf0b7e4be91d91` | `f847bd24-7d68-4c3a-b53a-4debf82e1499` | SUCCEEDED |
| 25 | `box_exec` | `88f24cf3684044be30f0727ddbbd7456` | `5dac4d4f-675f-470c-a35b-2916c95774ee` | SUCCEEDED |
| 26 | `box_exec` | `3d751dc8812346c0416151c857eba916` | `535d2ef5-73d7-4dba-b1a1-8f15e4b64fda` | SUCCEEDED |
| 27 | `box_exec` | `5350d893a7563d584db45f85fad14de1` | `595af9a2-d72a-4c52-a5cc-9285a84d57cc` | SUCCEEDED |
| 28 | `box_exec` | `1e906681a5ce4a3fef326101987506d5` | `e27c7862-68a8-48f8-b263-3b7c23188ada` | SUCCEEDED |
| 29 | `box_exec` | `ad4ce36b2dacd750ca9d8ef06ff488b6` | `86d05e13-7627-43c7-826b-d412a7788b34` | SUCCEEDED |
| 30 | `box_exec` | `4e72f79dfcd460b1db3517ad1925f0b4` | `d3d738a8-3dd9-4bb9-ad95-9ba12c3139eb` | SUCCEEDED |
| 31 | `box_exec` | `8bfb1c82a8d92fcc954c8d870bb129d0` | `005bec73-2f47-4c99-97d4-acd62636f355` | SUCCEEDED |
| 32 | `box_stop` | `221a6cedc51ba5b0b4532fff62b262a5` | `c81d1ebf-f94c-43a6-9c8e-8fa7d1b84b4e` | SUCCEEDED |
| 33 | `box_delete` | `d6e468912fbd7e9b7425766ef3bc620e` | `125a176a-b725-44e0-a6c0-201bd4f82a2c` | SUCCEEDED |

**Classification:** first child RUNNING under complain PASS; second independent Incus7.6 daemon PASS; second child image/archive, network and start FAIL with specific AppArmor policy-admin parser243; per-depth complain recursion **NOT PASS**; BH-019 **BLOCKED**. PR #146 remains Draft and unmerged.
