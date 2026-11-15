# BH-019 — native Incus 7.6 → 7.6 scoped AppArmor complain recursion experiment

2026-11-14 Pacific / 2026-11-15 UTC. **Verdict: complain successfully applied to one newly owned child profile, but did NOT confer kernel AppArmor policy-admin permissions or enable the next nested Incus container. BH-019 BLOCKED.**

> This is the owner-requested real **7.6** follow-up, not the prior Ubuntu archive Incus 6.0 exploratory attempt. We used only a newly created nonprivileged nesting-enabled Box under L0 BoxWeave RC5, then only independent Incus instances it owned. No privileged container, global AppArmor complain, shared existing Box/profile, foreign incusd, host mount, or historical UNKNOWN was modified.

## Exact environment / ownership

- L0 connector BoxWeave RC5, server commit `1224845c139177f3986d1ca3d51032849422ed1c`, physical/native Incus `7.6`.
- New outer Box `bh019-incus76-complain-1114`, exact UUID `0af027ba-72da-4efa-9927-a9e061c321b1`, nonprivileged, `security.nesting=true`, CPU1, memory 2048 MiB, disk 12 GiB, initially `internet=false`; only this new Box temporarily enabled internet for signed packages and official guest image.
- Work reference `BH019-COMPLAIN-76-20261114`, agent reference `bh019-76-complain-lab`. Every 43 original BoxWeave Operation IDs AND original native Incus IDs are recorded in the final receipt table below.
- Installed independent outer-Box Incus client AND server **7.6**, pinned Zabbly package `1:7.6-ubuntu24.04-202611072126`, with Zabbly key fingerprint verified `4EFC590696CB15B87C73A3AD82CC8797C838DCFD`. The signed source was independently usable from the deeper child; installed its own independent Incus client AND server at the exact same 7.6 build.
- Nested child1 `bh019-aa76-child1` (native UUID `d29c1e70-72dd-4f45-b62f-911b978efe1e`) was independently owned by the disposable outer Box Incus, a NON-privileged `security.nesting=true` Ubuntu 24.04 container. L0 new Box AppArmor view: its OWN namespace `incus-user-1001_bh019-incus76-complain-1114_<var-lib-incus>`, `.ns_level=1`, `.ns_stacked=yes`; child1 is an enforced-profile (later changed to complain) child in the SAME inherited level1 AppArmor namespace.
- Attempted child2 under child1 independent Incus: `bh019-aa76-child2` with no NIC, `security.nesting=true`, `security.idmap.base=1`, `security.idmap.size=65536`, resource limits; **never created a running child or a durable instance in that nested Incus**.

## Evidence matrix, actual native outcomes

| Stage | Actual native result | Causal boundary |
| --- | --- | --- |
| Preflight / exact 7.6 install | Signed packages, exact 7.6 client+server, `incus admin init --auto --storage-backend=dir` succeeded in outer new Box; its `incusbr0` and dir pool configured. | Eliminates the earlier Incus 6.0 package mismatch and missing dnsmasq confounder. |
| First child initial launch, default enforce | Image retrieved, but startup failed at `newuidmap: uid range [0-131072) -> [1000000-1131072) not allowed`. | Incus chose base 1000000 while inside this disposable Box root `/etc/subuid`/`/etc/subgid` explicitly permitted `root:200000:262144`. Prior cgroup BPF permission log also observed; cannot infer it is fatal once mapping corrected. |
| First child mapping correction | Changed ONLY stopped child1 `security.idmap.base=200000` while size `131072`; `incus start bh019-aa76-child1` returned 0 and read-back `RUNNING`, with exact guest UID/GID map `0 200000 131072`. | **Native unprivileged 7.6 L0-test-Box→child1 start is PASS**, without privileged/unconfined override. |
| Default AppArmor state | Guest PID1 and process label `incus-bh019-aa76-child1_</var/lib/incus> (enforce)`; child sees ancestor named AA namespace `.ns_level=1`, `.ns_stacked=yes`, local CapEff full. | Deeper UserNS did NOT automatically get new AA policy namespace; its root capabilities do not imply AA policy-admin privileges. |
| Switching only child1 to complain | Two ordinary `aa-complain` frontend attempts were non-effective (custom include path error or 'Profile ... not found, skipping'); safe parser syntax-only check returned 0. Exact **one generated child profile**, reloaded with AppArmor parser complain-mode flag in the disposable Box's own namespace, returned 0. `aa-status --json` independently reported child1 **complain**; `incus_dnsmasq-incusbr0_</var/lib/incus>` remained **enforce**. Inside child, both ordinary process and PID1 labels returned `(complain)`. | **The requested per-container complain toggle was genuinely achieved, not just a file edit or an AppArmor global switch.** Affected only the owned test profile in the disposable Box. |
| Independent Incus inside complain child | Signed packages installed, exact client AND server **7.6**, Unix daemon socket available. | Eliminates missing daemon, wrong version, no network as explanations. |
| Child1 `incus admin init --auto --storage-backend=dir` | Inner CLI rc=1: `Failed to create local member network incusbr0` → `apparmor_parser -rWL ... incus_dnsmasq-incusbr0` → **exit 243: Access denied. You need policy admin privileges to manage profiles.** | **Key negative result**: profile complain does not confer the kernel `aa_policy_admin_capable` authorization needed for inner daemon to load policy. |
| Networkless follow-up | Failed partial init left a valid `dir` pool, no managed NIC and empty default profile. Manually attached ONLY root disk to inner default Profile; no `incusbr0`. Child1 owned UID/GID subrange changed to `root:1:65536` inside its own FS, under its actual parent map `0 200000 131072`; all scoped to disposable descendants. | Tests skipping dnsmasq specifically rather than declaring it the sole blocker. |
| Child1 creates second child without NIC | Native `incus launch images:ubuntu/24.04 bh019-aa76-child2` rc=1 during **image unpack**: `Failed to load apparmor profile incus_archive-...`; `apparmor_parser` **exit 243**, policy admin denied. | **Second independent root-level operation requires policy admin** even when there is no nested managed bridge: archive extraction. No child2 instance in native inventory. |
| Independent negative/cleanup checks | Native inner `incus list --format=json` returned `[]`; child1 stopped/deleted after restoring profile to enforce; its parent inventory then `[]`. Outer owned Box stopped/deleted by exact L0 BoxWeave UUID and original durable native receipts. | No half-created child adopted, no shared host or historical UNKNOWN touched. |

## Important interpretation

1. This does **NOT** establish that AppArmor complain is globally ineffective; it establishes the precise narrow statement: **complain mode on a currently loaded, independently owned Incus child profile does not make the descendant's deeper UserNS policy-admin-capable**. The kernel namespace level/label-view and policy-admin gates are separate from ordinary complain enforcement behavior. See [previous kernel source call-graph and AA view tests](../research/S25_BH019_POLICY_NAMESPACE_DELEGATION_CALLGRAPH_2026-11-14.md).
2. The targeted complain toggle itself **passed real kernel mode inspection**, then was restored to enforce with the exact profile file, parser exit0 and `aa-status` enforce read-back **before deleting** the inner instance.
3. The *first* extra generation of native unprivileged nested 7.6 Incus DID run after correcting a specific permitted ID-map base. **This is not an L3 PASS**: next independent Incus initializes its daemon but cannot load required dnsmasq or archive profiles; no second-generation child under that daemon ran.
4. `box_exec` outer Operation state `SUCCEEDED` means an authorized diagnostic wrapper completed; **inner Incus child1 admin init and child2 launch both returned rc=1**. Original native receipt and command/output classifications are separated, not mislabeled PASS.
5. This does not prove deep recursion impossible on all kernels/configurations or under a design with truly separately delegated policy namespace levels. It **does refute this particular fully tested 7.6 complain-only workaround** as a solution for BoxWeave's normal native networking + image-unpack workflow.

## Exact recovery / teardown

- Stopped child1 by exact name with `incus stop --timeout 30`; independently confirmed its only instance state `Stopped`; deleted that one exact instance; read back outer new Box's independent Incus inventory `[]`.
- Restored only the new owned child1 AppArmor profile to `enforce` before its stop/delete; `incus_dnsmasq-incusbr0` also remained enforce. No AppArmor globally disabled.
- Stopped original outer test Box UUID via L0 BoxWeave Operation `aa68401debfe0170ca1ecce4af69a4fe`, native ID `0db2fde6-0d08-4b79-9ce3-ddad6cafafc0`, `SUCCEEDED`.
- Deleted the same original outer test Box via L0 BoxWeave Operation `0c2d7d334b77dd1f3cf6dd3150c30b77`, native ID `703f39d8-33ab-42df-b0bb-9f3d8c8ca84a`, `SUCCEEDED`.
- Independently verified exact baseline Box inventories **L0 4** and **L1 9**, with `bh019-incus76-complain-1114` absent. No new UNKNOWN, no old UNKNOWN replay.
- Scope query `operation_list(work_ref=BH019-COMPLAIN-76-20261114)` returned precisely **43 original immutable operations**, all with native IDs and outer status `SUCCEEDED`. This is a transport/lifecycle accounting claim only; separate inner native exits 1 remain real failures.

## 43 original BoxWeave/native Incus operation receipts

| # | Action | Original BoxWeave Operation ID | Original Incus native Operation ID | Outer state |
| ---: | --- | --- | --- | --- |
| 1 | box_create | `43c2068e9b95d14ee3e505e3d2467909` | `a8bc1385-365f-49a7-9975-b63e7f576f25` | SUCCEEDED |
| 2 | box_update | `1e0461ab2641775544e00f6acf91fae9` | `0735a070-e717-4059-b5c6-8dc2bed08a9a` | SUCCEEDED |
| 3 | box_exec | `87f4370eeb3c9317d5e6486907dae786` | `026b1e81-c674-4f67-b4dc-b809ae56ab24` | SUCCEEDED |
| 4 | box_exec | `9d5860d4e12043178a48a1c48e358292` | `fc12b42c-76e8-4869-aefa-4515da41483f` | SUCCEEDED |
| 5 | box_exec | `b19fde91d1b6ef49e37af40ba516fd02` | `ba2410c2-f0ca-400b-a600-069f1a2b31d0` | SUCCEEDED |
| 6 | box_exec | `3324cb49c3ac6cd73acf9d315a65dfef` | `1e4e5eb5-1ad7-4000-8789-c0e0b4c124d9` | SUCCEEDED |
| 7 | box_exec | `1d0155987f0b8b36c5e38a2cbe9b83c2` | `33a87437-726e-4059-9752-c8a6086dda56` | SUCCEEDED |
| 8 | box_exec | `b003662268e5696cc5579fed8aa73eee` | `3144997b-7b4d-4b73-84d7-814163b43e43` | SUCCEEDED |
| 9 | box_exec | `46c6f54f92d21081630b8247f44e52a2` | `3e0d958a-1427-411e-a2ba-1cf040f60d4c` | SUCCEEDED |
| 10 | box_exec | `410dd9c2c6b5d663e0a56f0de174342e` | `0c1ce7cb-ff9f-4011-a5fa-bd6d5a4c15a4` | SUCCEEDED |
| 11 | box_exec | `0c39ebfec21dc5ac33fe9b72912cde18` | `e45069cb-6b8d-4af9-b641-77bc8af083ed` | SUCCEEDED |
| 12 | box_exec | `eb6d2311b983f793cc5625808bb2045b` | `3b49dcce-1a70-4e2e-8573-6c5d99030561` | SUCCEEDED |
| 13 | box_exec | `30b5d25baca525537451d09995ef98c2` | `55ade315-f662-488e-80fd-8d8c5e6fb4ff` | SUCCEEDED |
| 14 | box_exec | `6d3aa69f9e05ff34b46558405f7606ca` | `aad8e468-2585-4dcb-9c02-cc72884f9526` | SUCCEEDED |
| 15 | box_exec | `edabaa007e8435a57c4c6324401b577f` | `a9dfb0f7-cf0b-4409-8fac-8d75fa2f19c6` | SUCCEEDED |
| 16 | box_exec | `5533806866ddc35e84167646716e59f0` | `7a89e090-f9e7-4f5b-927b-644b494133b5` | SUCCEEDED |
| 17 | box_exec | `06755e95cc7a78c4718c43a8efe73614` | `fbf3b084-985d-43f4-b40e-d52363a4938d` | SUCCEEDED |
| 18 | box_exec | `adbfbd924d7cd85e73be7b754628045b` | `895a46ed-b386-4cdb-baf4-7f99a8bc2cbc` | SUCCEEDED |
| 19 | box_exec | `dd48c20e44f9e42bfd5adfeedbcd1b77` | `0633e826-e87d-452c-8b4a-1b4d1bd76f1d` | SUCCEEDED |
| 20 | box_exec | `4dc4e9836ad4cf4bf6c19b735d99a9dd` | `aa2332f4-deb3-456a-abb2-b54221c3dcc2` | SUCCEEDED |
| 21 | box_exec | `fe992886f01429a13baffb662cfa418d` | `df0ea68d-777a-49d2-8854-6b6da5f23fb2` | SUCCEEDED |
| 22 | box_exec | `0becf24c98e22836a198d9d6b4cf1395` | `695b35fe-4586-4043-ba66-d97c9860eacc` | SUCCEEDED |
| 23 | box_exec | `6df9fbdbd97024923891fdb73cdd8e93` | `70e00808-00b9-4150-a936-b0873c3d91b2` | SUCCEEDED |
| 24 | box_exec | `133b48df4acecf97f102c72abb7f73ac` | `17edc94d-d66b-4e02-a0b0-e242cc99b0c9` | SUCCEEDED |
| 25 | box_exec | `6ce8f2c58837943e46c62dcd9a0bb311` | `e67945d8-9f29-486e-9963-18d12150d317` | SUCCEEDED |
| 26 | box_exec | `da305f40fedfaf43120da04f3761b32b` | `2dba165f-5f69-4197-8054-28651e6b448e` | SUCCEEDED |
| 27 | box_exec | `fa9f6c90da95f9e13b9cbbb8ff7537ae` | `9b93ed14-871c-47e5-b4d5-d891034669cb` | SUCCEEDED |
| 28 | box_exec | `01a323dfddbb94db46743f975ef28a38` | `7e3568bf-9f67-4836-bdab-b95dca64bff4` | SUCCEEDED |
| 29 | box_exec | `8993ba1cbd2d1af2ae47eaf84f88fc19` | `4ca8c4b3-d54b-4ad7-9a26-c463b817f8ec` | SUCCEEDED |
| 30 | box_exec | `cc532d502030f8ead6d6c1bf51bf13d9` | `cb02de6b-12c4-467f-9bec-6f0c8805c785` | SUCCEEDED |
| 31 | box_exec | `f7ae68e17eae23ba27f71482ecd1ed13` | `c6ecd2c2-a6d9-477f-8729-2a239301001f` | SUCCEEDED |
| 32 | box_exec | `66d770a391e4335b2cea35632b88bf82` | `0effb3ca-ac53-4ef6-961e-5b5d66b07b99` | SUCCEEDED |
| 33 | box_exec | `ba06cb2cc4cad9fe559fbb27113d8e5a` | `c7be49c4-f6ac-4df5-ad9c-2786bc28b3e5` | SUCCEEDED |
| 34 | box_exec | `c012607a137bc1f25b4d2e927dbddbbb` | `c33520e8-06d2-407c-bec9-0ab7c60b5104` | SUCCEEDED |
| 35 | box_exec | `cd510b5127975c80fb4b4f3437aa320c` | `6735a9be-7b9c-4e7e-ace5-d3f731c95ec8` | SUCCEEDED |
| 36 | box_exec | `1f46b648b6bdf2e63c2e098246c89b96` | `4b3155ae-a148-4edb-a5ab-cf7ab074ed25` | SUCCEEDED |
| 37 | box_exec | `031849db4652697182aecd55066acb0b` | `954db6d1-a6b1-46a4-880f-a2dade5d0198` | SUCCEEDED |
| 38 | box_exec | `5c8675ef9e484743dbfda3713a32c1ee` | `6df69fe1-7405-4dba-9c95-4a8de994b3c6` | SUCCEEDED |
| 39 | box_exec | `eede17d219e421f90c76e9749a400db3` | `8c0108d4-f45c-4660-9062-7b6efa5142d1` | SUCCEEDED |
| 40 | box_exec | `a3c4dc5d8cc74f7639df5f08a5b80d24` | `00397c4d-b792-4379-978f-4442ca00aedf` | SUCCEEDED |
| 41 | box_exec | `80e22045076eaf46d9822156a73c6bf3` | `90822c73-2505-450d-94a7-896095515f6e` | SUCCEEDED |
| 42 | box_stop | `aa68401debfe0170ca1ecce4af69a4fe` | `0db2fde6-0d08-4b79-9ce3-ddad6cafafc0` | SUCCEEDED |
| 43 | box_delete | `0c2d7d334b77dd1f3cf6dd3150c30b77` | `703f39d8-33ab-42df-b0bb-9f3d8c8ca84a` | SUCCEEDED |

## Engineering verdict and next action

**BH-019 remains BLOCKED.** The next meaningful line of investigation is coordinated kernel AppArmor policy namespace delegation / correct per-depth label transitions and policy administration, or a clearly bounded alternative design that does not require the normal native dnsmasq and image-extraction profile lifecycle. A simplistic per-Box `complain` mode toggle is now **experimentally falsified for full recursive Incus7.6 behavior**. Do not generalize this lab into a global complain/unconfined, privileged, or raw.lxc recommendation; don't change shared host policy, don't silently reduce BoxWeave's security acceptance criteria. No product code was changed; PR #146 stays Draft and unmerged.
