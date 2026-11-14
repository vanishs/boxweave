# S2.5 BH-018 / BH-019 — isolated native A/B and Exec receipt experiments (2026-11-14)

**Window:** 2026-11-14 16:10:42–16:41:39 UTC (08:10–08:41 PST). **Purpose:** controlled owner-authorized repeated evidence, not a release/installed candidate verification. **Verdicts:** BH-018 PARTIAL (unsafe error-classification hypothesis disproved; precise 5-second receipt-retention boundary confirmed); BH-019 BLOCKED (BPF interception did not address AppArmor policy denial; nested L2 native startup not proved). No original UNKNOWN was retried, unlocked or reclassified.

## Topology, source and security controls

- External parent MCP: BoxWeave_L0_RC3, 1.0.0-rc.3, source 1ce1774deca99fac4a3bb1031779ccacea0289d7; outer Incus 7.6, restricted project user-1001. **Fresh disposable test Box C** name s25-bh01819-native-c-1114, exact UUID/incarnation **43b0852c-8747-44dc-9d49-253bd04b82fe**. Initial Create BW operation **06df9921e2627fa419d62d09756b5070**, outer native Incus operation **31b7bbdd-1440-41f6-bf24-8101fe8c3573**, SUCCEEDED; nesting=true, 2 CPUs, 4096 MiB, 32 GiB, internet=true.
- Box C Ubuntu 24.04 installed signed Zabbly Incus stable **7.6**, package **1:7.6-ubuntu24.04-202611072126**. Prior verified Zabbly signing fingerprint **4EFC590696CB15B87C73A3AD82CC8797C838DCFD** was checked on the fresh fetched key; Signed-By was bound to /etc/apt/keyrings/zabbly.asc. C native daemon initialized in minimal mode. An Ubuntu 24.04 child s25-c-base was created and **actually started** with nesting=true and no privileged setting, then successfully obtained IPv4 and resolved pkgs.zabbly.com. The same signed Incus 7.6 was installed in that guest.
- After stopping s25-c-base, C's native Incus cloned it into **s25-c-default** and **s25-c-bpf**, both nonprivileged, nesting=true, with identical installed Ubuntu/Incus. In C's *own* native Incus only, s25-c-bpf was configured with **security.syscalls.intercept.bpf=true** and **security.syscalls.intercept.bpf.devices=true**; default group retains neither option. Both actually started successfully.
- Both child Incus daemons ran under **enforced AppArmor** labels in the same outer namespace. Read-back proved no privileged mode and no diagnostic AppArmor disable: **systemctl show -p Environment incus.service** returned an empty Environment for C, default and BPF. No global AppArmor profile, parent Incus daemon policy, network ACL or production controller credential was modified. C's native root socket was used only inside this new disposable lab (not exposed by a BoxWeave MCP Tool).
- Note identity layers: **BoxWeave original Operation ID** and **outer L0 Incus Operation ID** record each Box C command. Within that command, a separate **inner Incus Operation ID** is created. Outer BW Exec SUCCEEDED means the wrapper completed; an inner CLI exit 1 / native Incus Failure is NOT an inner success.

## BH-019 — genuine one-variable BPF comparison

| Step in new Box C | Outer BW original | Outer native Incus original | Observed result |
|---|---|---|---|
| Init C internal Incus minimal | 185988497894c88f1372035633f8e497 | 21a67313-96f0-440f-b94b-40bd16509429 | Guest CLI exit 0 |
| Create s25-c-base native child | 806ac61b8053649aea71ee6ebd7e9f4d | bc45055d-7adc-47a3-a203-99a590a0884f | Guest CLI exit 0 |
| Start s25-c-base native child | 1a3dda00d4c81cd9285ad8f4dff102c9 | d3f8b44e-582d-4ab8-a755-c21c2d0326fd | Guest CLI exit 0; actual RUNNING |
| Check base guest network, DNS and apt | fc47779e3b70b25c684426a2772f702b | f957d947-7500-486f-ad4d-1be43802f2f4 | IPv4 10.78.67.117/24, DNS resolved, INNER_EXIT=0 |
| Install signed inner Incus 7.6 | db82cf2c50fa15dcffdde3c479603f26 | ff5801aa-3d0c-4e7d-bd8e-991c76905b4c | Inner apt exit 0 |
| Stop base for controlled clones | a3a87fa2e16cd9791dc525ca7fe70f46 | 65401794-8575-48cf-861b-c7f5564faf8f | Guest CLI exit 0 |
| Copy exact default clone | 4ff8bb8818260243bd2a7aa9fc18b5d2 | 9b58ab91-f0ed-4eab-8023-f3f705d49617 | Guest CLI exit 0 |
| Copy exact BPF clone | 60ae0e7d47c2a7211a7f855e634d1568 | f8fbd786-ff8c-4bf0-b261-9e91dd68bb40 | Guest CLI exit 0 |
| Enable BPF syscall intercept in BPF clone | 20bcdaf6d5a59795ca8b89eadcf39441 | 9fd75fac-ac8a-451b-ae6d-57844838222d | Guest CLI exit 0 |
| Enable BPF devices intercept in BPF clone | 0eff8671228096b8f2756161fc1d608f | ba7853f0-744a-480e-a269-a62cd565a02a | Guest CLI exit 0 |
| Start default clone | 780aff993d84f9bf5d029c83112996e8 | c2952a8b-421d-45e4-ace2-af901910b7ea | Guest CLI exit 0 |
| Start BPF clone | 5760e040fea51b1233f2f30a29eb2efb | 0443d66d-96a5-46ce-9431-b2b6a6b5e1dd | Guest CLI exit 0 |
| **Default:** inner Incus admin init --minimal | c730a66d203f9d8068e230200566faa4 | b7a00ba2-740a-4470-90f5-1d7dc46ab5be | **INNER_EXIT=1**; apparmor_parser profile incus_dnsmasq-incusbr0 exit **243** |
| **BPF:** same inner init | f28819335f4c52a31f8577c0db79fed6 | a528c079-cf25-48e2-a287-320a7d9f719d | **INNER_EXIT=1**; **identical** dnsmasq parser **243** |
| Default: add root disk-only default profile | f87d12f01edc87d35baad5e33b4ddfe1 | 96719aa5-3545-4ab3-a5d6-88077ba54cee | INNER_EXIT=0; no inner NIC/bridge |
| BPF: same root disk-only profile | cfd42f4ed39111e21782b2106c6fd5dd | 178507e9-332d-46da-bc24-d6a680e6fa4e | INNER_EXIT=0; no inner NIC/bridge |
| **Default:** create grandchild from signed Ubuntu 24.04 image with no NIC | 66571ab0cd47381768734456ec2578f8 | e17671fb-9a1f-438c-9a49-38a85f941bb2 | **INNER_EXIT=1**, archive unpack apparmor_parser incus_archive-* exit **243** |
| **BPF:** identical grandchild create, no NIC | f72a2026cd08ca8c502add8ee0317e15 | d811a486-eda8-4e85-875d-2879ba0e1f75 | **INNER_EXIT=1**, archive unpack apparmor_parser incus_archive-* exit **243** |
| Native flags and state postread | e94834628084d9fb434c881efa37c48b | 1dc6cbda-1fa0-4974-b7c8-e0a19d23fa4a | Both RUNNING, nesting=true; default BPF keys empty, BPF clone true/true; privileged empty |
| AppArmor daemon Environment postread | e0d5730318bb485a2ae1410ca756ddbc | ef7082cf-04ea-422f-8ef3-7d8c40fcc803 | All three daemon Environment values empty |

**What this proves:** The first nested Incus container can run with confinement. Both same-source second-layer daemon initializations were denied by AppArmor's **policy-admin** guard. Removing the inner NIC did NOT solve it: even image unpack invokes an **incus_archive** AppArmor policy-load. The paired BPF-device interception changes did not cure these **AppArmor** failures. This does **not** disprove BPF interception as a solution to the separate cgroup2 BPF EPERM: the controlled grandchild never reached that stage. No real L2 running lifecycle, independent I+N+V or inner BoxWeave install was claimed.

## BH-018 — real original Incus Exec Operation results

Native test harness: parent C's Incus 7.6 REST API queried via the **official Incus CLI raw-query mode** (NOT a claimed Go SDK execution). One accepted native POST to /1.0/instances/s25-c-default/exec per case; payload exactly **record-output=true**, **wait-for-websocket=false**, with command argv and optional cwd. The same original test process immediately read the exact **/1.0/operations/{native-id}/wait?timeout=20** response. The guest was RUNNING. Each case had a fresh outer BoxWeave caller key and separate native Operation ID.

| Case | Outer BW original | Inner native original Incus ID | Immediate original final Operation / metadata |
|---|---|---|---|
| Existing /bin/true | c01cdaf1f642fa519d0046bf50d37932 | 532abf29-02b6-4e86-80dc-6c2762f00d9b | **Success**, return=0, recorded output paths |
| Existing /bin/false | e931a04df0a7f63156f20a0a0d7a6857 | e4cdd22c-e9ef-4572-9be3-7c17cea06905 | **Success**, return=1, recorded output paths |
| Missing absolute executable | 9943c9ed8f0b93f50493f85b99a97c66 | 51ca7483-1eb1-4cba-973a-dcb88c64fbf9 | **Failure**, err=Command not found, return=127 |
| Exists but not executable: /etc/passwd | 2df12d5516320f05dfbf67ab14f83e47 | 0bb36d1b-63f4-4267-8d1b-55ab3097c757 | **Failure**, err=Command not executable, return=126 |
| **Real shell starts and exits 127** | f8d98e68aa8b075aae3840d94ad3402e | 58cfe2bb-3ba7-4957-ba44-b8a4b47bc7cb | **Failure**, err=Command not found, return=127 (**identical terminal tuple to missing executable**) |
| **Real shell starts and exits 126** | 7cdb459f9ed7affadc38691bce706f57 | cc2d91ca-4a82-4ac0-b925-e075cec0d790 | **Failure**, err=Command not executable, return=126 (**identical to non-executable file**) |
| Missing unqualified command (PATH case) | 3c470b4fe11b57edbe711e902cbb7fea | b4cd9ee7-6ff7-40c1-8e74-f2d8c3b6c38e | **Failure**, err=Command not found, return=127 |
| /bin/true with absent absolute cwd | ecfe1564a676a4e1758def549ba43fa8 | e82d30c4-82d5-4d54-8bf4-1f6ecbc42224 | **Success**, return=0; cannot assume cwd was applied or verified |

**Decisive independent side-effect counterexample.**

1. Separate native read-preflight BW **49000d46b7763ba8c01307cbb2002f97**, outer native **7f5fa68e-d53e-415f-98d7-16f88eb9b2c9**, verified the path /tmp/s25-bh018-sideeffect-exit127-marker did **not** exist.
2. Original POST+WAIT test BW **9efcd423eb71aa200d248ab2d00fd714**, outer native **35e0dee5-7e28-475a-bbe9-e3c42854b57a**, *inner* original native Incus **801658fa-039c-43cc-971c-627afa2a8ec1**: /bin/sh writes an exact marker **BH018_MARKER_EXECUTED_127** into that file, then exits 127. Incus final status was **Failure / Command not found / return=127**, exactly matching the non-existent executable control.
3. Independent separate BW read **04510cd6e546496fadd860f5acb46a10**, outer native **82380e42-bf5a-44fb-a3f5-c55fed8c2b5b**, guest CLI exit 0, readback contained **BH018_MARKER_EXECUTED_127**. Thus an Operation reporting **Command not found** may have caused genuine durable guest effects. Do NOT infer "did not start", release the BoxWeave resource claim or auto-replay simply from native err/status/exit=127. Similarly 126 is not a proof of no guest effects.

### BH-018 exact Operation retention timing

The same native POST/GET/WAIT controls used a no-side-effect /bin/true. Delayed independent **GET /1.0/operations/{id}** and **GET /wait** both yielded:

| Delay after native POST | Outer BW original | Inner original native Incus ID | Native GET and WAIT result |
|---|---|---|---|
| Immediate | 7bc4b75f4e512f4d95c023107c1717b5 | 4cdc700c-2a31-40ee-8a90-4c98c5e1eef2 | Both 200, Success, return=0 |
| 0.25 s | 82341db20e878800887b4ca991eb6b60 | 4fe947e8-2e04-43b9-9815-3bdd8f33f0a8 | Both 200, Success, return=0 |
| 2 s | 287263379172d70936fe7d0074d01f29 | eb476e0a-4da6-4689-8d3e-76fd5c547ee3 | Both 200, Success, return=0 |
| 4 s | ac9a8f706f6049875ee91c488714edfe | 68d49147-f1b8-4b92-bfbd-cb43e951bdaa | Both 200, Success, return=0 |
| 6 s | e485e3b2f6927b4de17dc80005c78989 | b28363a6-0db8-4eec-b653-ae0bf45a9f06 | Both **404 Operation not found** |
| 9 s | befe184bd505119f1d81aaab38dfd8c4 | 5922befa-0f03-43b4-ae1b-f0cd2c59a8df | Both **404 Operation not found** |

The exact [Incus v7.6.0 native operation code](https://github.com/lxc/incus/blob/v7.6.0/internal/server/operations/operations.go) (blob **46ff03d3f76344e19d74d73f63ac935ba5ec502d**) explicitly waits **time.Second * 5 after completion** before removing the original Operation from the internal map and database. Our 4-s positive and 6-s negative reproduce this. An initial two-tool call for /bin/true got a valid native ID **d4b0519f-3e4f-43c8-a1ea-304e19919302** on BW **6ab48249502310b7d9dd96ae3f457eb9**, then **404** on a delayed read through a separate BW operation **a5d833e3bfdb313c97fb33da029e5e59**. A 404 is a **retention event, not negative side-effect proof**.

The [official pinned Go SDK](https://github.com/lxc/incus/blob/v7.6.0/client/incus_operations.go) (blob **a3cd39c4eacab2760336a39c8db1fb144430a1ea**) exposes **GetOperationWait(id,timeout)** as a read-only GET of the original /wait endpoint, distinct from a high-level WaitContext helper that may cancel on a canceled context. An immediate bounded, non-canceling terminal read and durable recording should be investigated under existing BoxWeave operational safety constraints. A delayed /wait **also** returns 404 after expiry; simply switching endpoints is not a general recovery mechanism. Incus native terminal metadata cannot be recreated after loss. No SDK Go runtime test of a proposed patch has been asserted here.

## Engineering conclusions and acceptance boundary

- **BH-018:** Existing no-native-ID, proved local pre-dispatch validation can safely be FAILED. For a nonempty original native ID and native terminal Failure, **do not** classify as no-effect FAILED using only native **Command not found**, **Command not executable**, return=126/127 or absence of an Operation after retention; the marker counterexample demonstrates false safety. Keep UNKNOWN/locked unless a *separately proven no-effect or fully reconciled* criterion exists. Prior source guard in PR #146 correctly prevents a native-ID-bearing effect error from being treated as predispatch. Next focus: bounded non-canceling same-original native outcome collection and durable metadata, official SDK/context/restart safety regressions; allow successful process exit nonzero while retaining command_success=false as before.
- **BH-019:** AppArmor policy admin (dnsmasq and **archive image unpack**) is an independent mandatory gate beyond nesting and the official paired BPF devices option. No native BPF cgroup load was reached in the controlled grandchild trial, so its potential benefit to BPF EPERM is **UNVERIFIED**, not rejected. Need least-privilege outer AppArmor namespace/policy delegation before safe native L2 lifecycle. Do not use privileged instances, global unconfined AppArmor or exposed administrative sockets as an acceptance workaround. The earlier real L1 experiment already proved cgroup2 BPF and label-transition failures on an attempted child start; see [prior expanded proof](S25_BH018_BH019_L0_L1_EXPANSION_2026-11-14.md).
- All *new Box C* BW Operations above reached normal, recorded SUCCEEDED outer receipts; inner failed CLI commands were deliberately wrapped to preserve their output and **INNER_EXIT=1** evidence, not falsely relabeled as successful Incus operations. No new unresolved BoxWeave UNKNOWN was created in this run. The **prior original L1 UNKNOWN** BW 0e4ad1324a029fb58bbc5573e1a9588b / native 83c384e3-dade-4ef4-85da-4993cc08afa2 and **prior original L0 UNKNOWN** BW 4a5130ee9025e7ee39c1866d16ad1b39 / native fa2f54f1-bbe1-432d-88cc-52ad572e938c remain unchanged. Their original terminal evidence was not reconstructed.
- Box C and both cloned inner containers remain disposable and running for bounded follow-up; no release or merge. **Draft PR #146** remains Draft. Source and any future candidate native changes still require new exact-head CI, real installed MCP/native independent V, correct permissions, network/ACL/IPv4/IPv6 and negative isolation proof before BH signoff.
