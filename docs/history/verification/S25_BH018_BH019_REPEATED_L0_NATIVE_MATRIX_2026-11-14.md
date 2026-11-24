# S2.5 BH-018/BH-019 — repeated L0 native Exec, AppArmor and BPF A/B matrix

**Run window:** 2026-11-14 **16:10–17:16 UTC** (08:10–09:16 America/Los_Angeles). **Owner authority:** approved repeat L0/L1 disposable experiments. **Verdict:** `BH-018 PARTIAL`; `BH-019 BLOCKED`; source/native evidence improved, but no secure restricted L2→L3 BoxWeave-controller acceptance, independent Verifier or host signoff. This supplement is an exact original-operation record, **not** a blanket nesting PASS.

**Parent MCP:** `BoxWeave_L0_RC3`, restricted Incus Project `user-1001`, native Incus 7.6. Every outer guest command had a distinct idempotency key and original BoxWeave Operation ID plus outer native Incus Operation ID. Here `BoxWeave SUCCEEDED` means a recorded Guest Exec lifecycle; a nested `incus` CLI can still exit nonzero. The nested Incus IDs recorded below are **third**, distinct IDs created by the `incus query` API inside the Guest.

This follows [the initial L1 RC4 evidence](S25_BH018_BH019_L1_RC4_2026-11-14.md), [the owner-authorized L0/L1 expansion](S25_BH018_BH019_L0_L1_EXPANSION_2026-11-14.md), and [upstream source reasoning](../research/S25_INCUS_UPSTREAM_NESTING_EXEC_2026-11-14.md).

## Disposable L0 Box C — native Exec semantics and further AppArmor depth

- Exact Box name `s25-bh01819-native-c-1114`; UUID/incarnation **`43b0852c-8747-44dc-9d49-253bd04b82fe`**, Ubuntu 24.04, 2 CPUs, 4096 MiB, 32 GiB, `security.nesting=true`.
- Original outer Box Create **BW `06df9921e2627fa419d62d09756b5070`**, **outer Incus `31b7bbdd-1440-41f6-bf24-8101fe8c3573`**, **SUCCEEDED** at 16:10:42 UTC.
- Parent installed signed Incus 7.6. This Box had other already-created native children/changes during the run; **no BPF causality is attributed to those other children**. Own independently named running child `s25-bh018-exec-own` was created at 16:46:52 UTC (outer BW `715d91acc8686faf8ab72f4071f1ba22`, outer native `c3b4a223-2398-41e7-885b-db5ba664baf3`) and started at 16:47:10 (outer BW `3c52dfa368d35d83fdfd34ec36381c00`, native `165ef364-c6c7-4564-8603-234fc18e6855`).

### BH-018 — original native Incus Operation outcomes, same `RecordOutput=true,WaitForWS=false`

Each inner Exec was a fresh `POST /1.0/instances/s25-bh018-exec-own/exec` against **Incus 7.6** with recorded stdout/stderr. The exact original **inner native Operation** was independently read via `GET /1.0/operations/<id>/wait?timeout=5` (not the box_exec wrapper's own native ID).

| UTC | Guest command / witness | Original **inner** Incus Operation | Parent **BW** Operation / parent native Incus Operation | Terminal native outcome |
|---|---|---|---|---|
| 16:48:54 | Nonexistent `/usr/bin/bh018-missing-binary-unique-v2` | `b0f1e649-1d2f-459f-aeb9-15a5174f0c3c` | `e7aa927a4b0e32e3ebcdbee39fa62e9e` / `3ad5dffc-fa1b-4913-b423-d7096e200259` | **Failure**, code **400**, `err="Command not found"`, `metadata.return=127`, recorded output paths `1,2` |
| 16:49:33 | Existing `/bin/false` | `d24e3c18-c19d-4c95-9525-ad09179edc3c` | `14e0f99912c9abde0fb47094b1152b31` / `e2e58395-0533-4597-8670-4d79e6cac577` | **Success**, code **200**, `err=""`, **return=1**, output paths present |
| 16:49:48 | Existing `/bin/sh -c 'exit 127'` | `98ae9ea6-1621-440d-b8f3-36d4eccbba02` | `dd7667b82584ddfd66fee2d64a5ff496` / `ed48b8db-3826-4e88-b169-4ac70e37cc72` | **Failure**, code **400**, **same `Command not found` and return=127** |
| 16:50:27 | Existing shell writes `EFFECT_OCCURRED` to `/tmp/s25-bh018-127-sideeffect`, **then exits 127** | `d08d88ff-ebc2-46b9-a959-971789759eab` | `398de60c74929655378c1b19c5e09e96` / `8d3cd000-9c8d-4060-9259-f675b028639b` | **Failure**, code **400**, **same `Command not found` and return=127**, **but independent `incus exec ... cat` confirms `EFFECT_OCCURRED`** |

**Decisive negative:** native `Failure + Command not found + return 127` **cannot** prove a Guest executable never ran or did not mutate Guest state. An actually launched shell performing a persistent file write produced those exact same fields. Inferring pre-dispatch failure or safely retrying on this signature would be a real unsafe classification. A terminal `api.Failure` with apparent exit 127 must **not** be converted to successful command output or proof of no side effects. An improved result taxonomy needs **additional authoritative process-start witness**; until then the original BoxWeave `UNKNOWN` resource lock remains correct.

An earlier raw-POST **debugging** experiment returned original inner operation `3815f2a1-c445-4c46-8eb8-62ea2d794fce` (initial Running) but test parser mistakenly expected a `operation` field instead of `id`. Later native GET said `Operation not found`. Its terminal outcome was **not observed**, and was neither reconstructed nor labelled a confirmed failure. Its outer BW `64f2981f1a1b3b46a0f1a7418a496685`, outer native `9667fb83-6253-448b-9c7f-8e9c7bd2782d` have their original independent lifecycle receipt. The corrected v2 test above is **a separate test with different command**, not a replay of this original operation.

### BH-019 — nested correct-flag, no-network, no-image controls

Within C's separately named child `s25-bh018-exec-own`, installed Incus **7.6** using Zabbly's signature-verified APT source/key, copied with matching SHA256 from the parent, then examined distinct failure stages:

| UTC | Inner L2 step (outer BW original / outer native original) | Observed Guest Incus CLI and independent evidence |
|---|---|---|
| 16:56:09 | `incus admin init --auto --storage-backend=dir` — `f3407ed52eab631c1450f2adacb4047a` / `17a60af8-e202-45ae-8273-ea7f2016797e` | **Guest exit 1**, `apparmor_parser -rWL ... incus_dnsmasq-incusbr0` **exit 243 policy-admin denied**. Root service, not only `incus-user.service` |
| 16:58:11 | Root-only Profile, no NIC, `incus init images:ubuntu/24.04 s25-bh019-l3-own` — `0e6167712e596d3177f9efb830fbd1fd` / `bb98a4bd-db03-4a4f-96f6-80af21334482` | **Guest exit 1**, `incus_archive-*` AppArmor profile-loading **243** in image unpack, even with no network |
| 16:59:28 / 16:59:42 / 16:59:54 | **Stop, set `security.nesting=true`, start** own L2: `f958c3b38f8d6ce48738bb9db85ec390` / `bcca9ab7-5b97-42d3-8757-cc282848c39a`; `f039bed9a85b6f9b20df47a01de1cd04` / `04be81a5-63f4-4992-bb2d-1f89db3954fb`; `de0cd306f6f3b81656977708368e6db4` / `b0ff37ae-60ec-4994-9a3a-9c55275b9947` | All original wrapper operations succeeded; independent readback `061d5147472aca1fb29020035d19a7d4` / `20af375b-6e0b-4e84-bf6a-fe191ec41c1b` **proves nesting=true on restarted L2** and inherited AppArmor namespace with enforced label |
| 17:00:41 | **New** `incus network create s25-l3-net-own` — `b9d03abcebea11c1c7e12947abc0ccde` / `717ca696-4d79-422f-8fa4-fd7740270759` | **Guest exit 1**, dnsmasq profile load **243 persists after nesting=true + full restart** |
| 17:01:53 | `incus init --empty s25-bh019-l3-empty-own` — `2eaff2d503afff2a14fb903789465187` / `d9f14648-75ed-469b-b39b-4727518b4edc` | **Guest exit 0**, empty child created without image/network |
| 17:02:06 | `incus start s25-bh019-l3-empty-own` — `d54da635e97a6e22d905d34b008e2147` / `f1d10599-1046-4716-b450-9db317c2c5ac` | **Guest exit 1**, LXC pre-start hook failed |
| 17:02:41 | Native service journal readback — `e790b10379db5fc952edf2edf783d373` / `7eca6bc1-4036-4d6a-824d-b1919f59f8d7` | Confirmed `incus_<empty-child>` **AppArmor parser exit 243** during pre-start. **No BPF causal inference possible** because startup blocked earlier |

**Reproducible 243 path:** DNS bridge policy, image archive policy and empty child start pre-hook each require AppArmor policy-admin beyond current inherited namespace. `security.nesting=true` does not grant it by itself. This is not simply networking, image download, user socket or UID exhaustion.

**C safety stop:** After the last real C operation, a later independent new `box_exec` request was **rejected synchronously by BoxWeave** with `original unresolved resource claim is still locked`, before native dispatch. The available operation list for this caller showed no visible C UNKNOWN/active ID and no safe unlock evidence. No native effect was attempted after denial. **C retained**, no reset, adoption, unlock or deletion; test moved to new D. Earlier historical L1 missing-gpg original `0e4ad1324a029fb58bbc5573e1a9588b`/native `83c384e3-dade-4ef4-85da-4993cc08afa2` and prior L0 `4a5130ee9025e7ee39c1866d16ad1b39`/native `fa2f54f1-bbe1-432d-88cc-52ad572e938c` also untouched.

## Disposable L0 Box D — clean BPF-device syscall intercept A/B

- Name `s25-bh019-bpf-ab-d-1114`, UUID/incarnation **`0d0ffd60-2fd6-4734-b40c-6eabdb561033`**, Ubuntu24.04, CPU2, RAM3072 MiB, disk20GiB, `security.nesting=true`. **Original Create BW `347a29db731d92575b0ca834ea018ca6` / native `5a71df72-a3f3-41b7-b7cf-8690bcacce9f`**, 17:05:22 UTC, SUCCEEDED. Initial Guest had **no Incus or third-party APT source**.
- Ubuntu signed `gpg` installed, Zabbly public signing key fingerprint verified **`4EFC590696CB15B87C73A3AD82CC8797C838DCFD`**, APT `Signed-By=/etc/apt/keyrings/zabbly.asc` against signed `https://pkgs.zabbly.com/incus/stable`; installed exact `incus=1:7.6-ubuntu24.04-202611072126`. Original install BW `f137c9ccad99f0b6d7c92e47e22cada5` / native `7c2cef0f-aa10-4cf7-92e3-4a098f867d88`, Guest exit **0**.
- `incus admin init --auto --storage-backend=dir`: BW `79531f41e2f18c40c510fb48d19be52e` / native `33cc6201-989f-4a89-aa24-39ec22fe37b3`, Guest exit **0**. Independent Profile shows bridge `incusbr0` and dir root pool. This is **root of a disposable D Guest using native default Project**, not proof of BoxWeave restricted nonroot service or per-Agent isolation.

Two cases use the same Ubuntu 24.04 official image, same default profile/root disk/NIC, same `security.nesting=true`, `limits.cpu=1`, `limits.memory=384MiB`, same native Incus 7.6 parent D. Only experimental difference is **paired BPF device-cgroup syscall interception**:

| A/B case | Native keys | Original Create BW / original outer native | Original Start BW / original outer native | Real results |
|---|---|---|---|---|
| **A, default** `s25-bpf-ab-default` | `security.syscalls.intercept.bpf` and `.devices` **absent**, defaults false | `246d6007b9407e0cd936a75366834797` / `fe69dfde-6370-4f01-a6e1-5afcf674e307` | `38c9ef7e51a5a2bdf7542f465b684b91` / `382e0eab-baf0-408b-903d-6908876bfa4c` | Create and start Guest CLI **exit 0**, native `RUNNING`, Guest `/bin/true` works |
| **B, BPF enabled** `s25-bpf-ab-enabled` | `security.syscalls.intercept.bpf=true` **and** `security.syscalls.intercept.bpf.devices=true` | `f17a6796d125c71b18db005eace3d0d7` / `9834a53c-3b76-41f8-a2ec-15c4c4ca0737` | `7bfceb6a0de4b80054ba32254bf44030` / `4298540e-31f8-457b-a82c-853f147ff4b9` | Create and start Guest CLI **exit 0**, native `RUNNING`, Guest `/bin/true` works |

- **Original independent status and effective-config readback:** BW **`b6285c0ba3b00d1a7dce15d5f7932ebf`** / native **`f6cfad86-07c1-4373-b4b2-ed058ab22178`**, 17:14:58 UTC, verifies both RUNNING, profile enforced and exact BPF key difference. Separate AppArmor readback BW **`68a2530a810bb015a86d5ee92d747994`** / native **`465eeddd-7965-446e-bf5d-ea8c39ad7bbf`**, 17:16:18 UTC: `/proc/self/attr/current` is an enforced per-child Incus profile in each case; `.ns_stacked=yes`, same namespace name `incus-user-1001_s25-bh019-bpf-ab-d-1114_<var-lib-incus>` on both.
- **Strict interpretation:** Paired interception **is not necessary for child Start in D**, because both succeed. This A/B has **no existing cgroup-device BPF EPERM at this depth**, so it does **not establish** whether interception cures deeper BPF failure. With no safe AppArmor delegation at the L2→L3 path, deeper BPF remains **INCONCLUSIVE**, and enabling this high-risk capability globally is not justified.

## Accepted lessons and next safe gate

1. **BH-018:** preserve existing conservative `ExecEvidenceFromFinal` behavior for non-`api.Success` final Operations: `Failure/Command not found/return=127` does not mean no guest effects. Do not classify or unlock historic unknown on code127 alone. Any future `FAILED` for dispatched Exec requires new non-ambiguous independent *process-not-started* evidence (not the final Operation signature).
2. **BH-019:** obtain a formally supported, least-privilege **AppArmor namespace/policy delegation** path from the actual enclosing L0 host/Incus layer that preserves parent/child confinement and restricted Project. Independently prove dnsmasq, archive and empty-container pre-start profiles load, **then** independently test cgroup-device BPF in the deeper LXC start. Root/default-Project D A/B is a diagnostic control, not admissible security I+N+V.
3. Do not add unrestricted host Incus socket, `incus-admin`, `security.privileged=true`, `apparmor=unconfined` or permanent `INCUS_SECURITY_APPARMOR=false` to bypass the gate. No host-global profile modification was made during these experiments.
4. Both new D A/B native child instances and original D Box are retained for readback; C is retained because BoxWeave refused a later new command with a resource-lock diagnostic. No original UNKNOWN was replayed, unlocked or destroyed. No claim of nested BoxWeave RC4 deployment or secure L3.

**Run status:** BH-018 **PARTIAL** (critical unsafe alternative disproven); BH-019 **BLOCKED** (actual multiple 243 failures, BPF-only A/B inconclusive at deeper depth). No S2.5/LAB/HOST_GO or release PASS granted by this report.
