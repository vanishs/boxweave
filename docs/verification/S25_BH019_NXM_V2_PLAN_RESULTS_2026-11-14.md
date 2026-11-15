# BH-019 — N×M v2 sequential evidence matrix, L0 RC5 / L1 RC4 (2026-11-14 PT)

> **Purpose:** re-run the highest-information prior experiments with better fault localization, not to report recursive AppArmor delegation success. **Scope:** AppArmor / policy namespace, UserNS/NSFS, capabilities/IDs, read-only policy surface, and safe permission negatives only. Per owner instruction, do not research distribution/version matrices, patches, BPF or VM routes. No privileged Box, physical-host AppArmor/profile changes, host sysctl writes, exposed host Incus admin socket, or historical UNKNOWN replay.

## V2 matrix — sequential N with multiple M outputs

| N (in order) | Independent experiment/control | M: measure together (separate readouts) | Expected decision gate |
| --- | --- | --- | --- |
| **N0** | Authenticated L0/L1 baseline and two **new** equivalent internet-off, nesting-enabled, nonprivileged Ubuntu test Boxes | source build, native project, exact Box UUID, Incus native ID, pre-existing inventory, network, UID/GID map, kernel and AppArmor features | Stop if creation is not provably SUCCEEDED or fixture differs in security-relevant way. |
| **N1** | Root and nonroot identity × layer; read-only baseline | labels from /proc/self and PID1, visible AA policy NS, .ns_stacked, UserNS, mapped range, effective/bounding capabilities, NNP, seccomp, effective GID | Distinguish relative "unconfined" view from actual ancestor enforcement; no assumptions beyond visible label. |
| **N2** | **E5 repair:** one individual, sandboxed, time-limited stat/list/read operation at a time across a fixed whitelist of AppArmor securityfs nodes. Flush a marker **before/after every step** and isolate each probe in a separate subprocess. | path, op, errno, elapsed, child exit/timeout; directory metadata, one bounded profile text read, mountinfo | Do **not** do all-or-nothing virtual file enumeration. Halt on single timeout; no profile load/replace/remove. |
| **N3** | Root/nonroot NSFS queries with parent/owner negative and direct readbacks | NS_GET_USERNS on pid/mnt/net/uts/ipc/cgroup, NS_GET_PARENT errno, NS_GET_OWNER_UID overflow value, AA current/ns/stacking, maps, caps | Cannot infer unexposed numeric kernel depth. |
| **N4** | Root bounded nested `unshare -Ur` stages and nonroot one-level test | at depth 0/1/2: UserNS inode, owner_uid, uid/gid map, AA NS, label, capability, subprocess exit, timeout | A new UserNS alone must never be called new AppArmor policy admin. |
| **N5** | Raw `unshare(CLONE_NEWUSER)` under UID/GID 1001, 1002, 65534; multi-layer matched-pair controls; optionally UID0 for comparison | errno, uid/gid pair, map location, CapEff, NNP, SeccompFilters, AA label/ns, sysctl | Distinguish consistent layer policy from isolated UID/mapping and seccomp numerical mode; do not claim a specific LSM without audit. |
| **N6** | Disposable local file ownership/map check (only if prior safe) | stat owner before/after temp file owned by root/UID1001/1002, mapping interpretation, permission failures, cleanup | No shared mounts or host file modifications; merely verifies how user-visible IDs behave. |
| **N7** | Exact STOP/DELETE and independent inventory postcondition | BoxWeave original+native receipts, terminal states, unmodified original inventory 4 L0/9 L1, no newly created UNKNOWN | If effects unresolved, do not replay/delete; preserve fixture, identify blocker. |

### Execution contract

- Collect **each** BoxWeave original Operation ID and original Incus native ID after per-tool `operation_get`; fetch `operation_output_get` and classify **outer operation state independently of Guest exit/command_success**. Original idempotency keys unique per distinct execution. Never execute old UNKNOWN identity.
- Phase order and isolation outrank attempt count. Do not run N3+ if N2 leaves a locked resource or ambiguous Guest. Later phases may be skipped with explicit reason.
- For N2: hard limit <2 seconds per filesystem node operation in a separately terminating child; **parent flushes marker first**, records precise timed-out path and exits. Avoid recursive/list-all-walks, profile writes and read-to-EOF on virtual files. A single abnormal child fails that cell; avoid mass retry.
- The L0 and L1 guest labels need **not** be assumed equivalent. Differences identify follow-up questions, not proof of enforcement or policy delegation.
- Preserve old inventory, no L0/L1 host Exec dispatch outside newly created owned disposable Boxes, no service/systemd mutation, no policy modifications.
- At end append observed N×M evidence, new distinctions, exact receipts and cleanup to this file, link in main BH019 hypothesis and PR conversation. Keep formal **BH-019 BLOCKED** unless genuine per-depth independent policy load/replace/remove/negative enforcement is verified.

## Execution results

### Final disposition (2026-11-14 PT)

**N0–N7 performed in sequence. N1–N6: 26/26 original BoxWeave Exec Operations with verified original native Incus IDs, guest `exit_code=0`, `command_success=true`, complete recorded stdout. N7: independently verified successful exact STOP and DELETE of both newly created test Boxes. Zero newly observed UNKNOWN; historical UNKNOWNs never touched.**

**Important classification:** N4's L0 UID1001 **inner unshare failed** and N5's three L0 nonroot raw unshares **failed EACCES**, deliberately captured inside a guest diagnostic wrapper that itself exited 0. Guest wrapper success is not the inner syscall outcome. N2 probes include expected read-only `EAGAIN` / `EACCES` negative results; these are not affirmative permissions or policy administration.

### Exact fixtures / baseline

| Layer | Controller source and native | Exact fixture UUID | Original BW Create / native Incus Create |
| --- | --- | --- | --- |
| L0 | RC5 `1224845c`; Incus 7.6, restricted project user-1001 | `bh019-nxm2-l0-1114` / `ac12052a-77ca-4c32-ad67-f1125846d824` | `2f7634d6090c340fb68b7d6da56be4eb` / `1bd1a209-68ad-46c7-bac6-e7666f9ac387` |
| L1 | RC4 `c3023a72`; Incus 7.6, restricted project user-1001 | `bh019-nxm2-l1-1114` / `4de6bf63-8c81-4c4b-bee8-f703d7281c09` | `3b14e023dddf3fa965607bc2743d66a7` / `80db3f3e-7528-4475-a985-8d14a4048789` |

Both new Boxes: nonprivileged, `security.nesting=true`, internet=false, Ubuntu24.04.5, 1 CPU, 1536 MiB, 10 GiB, kernel `7.0.0-43-generic`. The existing L0 inventory counted **4**, L1 **9** before this run.

### N × M observed comparison

| N / measurements | L0 new Box evidence | L1 new Box evidence | Interpretation |
| --- | --- | --- | --- |
| **N1: root+UID1001 identity/policy/maps** | Relative label `unconfined`; `.ns_name=incus-user-1001_bh019-nxm2-l0-1114_<var-lib-incus>` (own Box); `.ns_stacked=yes`; root CapEff `000001ffffffffff`, UID1001 CapEff zero | Enforced own-box label `incus-user-1001_bh019-nxm2-l1-1114_</var/lib/incus> (enforce)`; `.ns_name=incus-user-1001_rc3-l1-nested-1113_<var-lib-incus>` (outer parent); same caps pattern | Distinct visible policy views; L0 `unconfined` **does not by itself prove no ancestor confinement** |
| **N2a: policy/feature directory metadata** | Exact isolated `stat/list` probes completed (13 steps); policy/namespaces listing empty. Detected previously uncollected `.ns_level` and `features/namespaces/userns_create` | All 13 stat/list steps completed, child policy namespaces empty | N2 metadata no hang; directory presence and mode `0755` do **not** prove policy administration |
| **N2b: bounded nonblocking read** | Direct `.ns_level=1`; `userns_create=pciu&`; `features/namespaces/mask=userns_create`, `features/namespaces/profile=yes`, `.ns_stacked=yes` | **Also `.ns_level=1`** and identical feature strings, yet Box own UserNS differs and AA name is outer parent's | Direct AppArmor namespace level: **same level 1 at both container depths**, not merely a name heuristic; absolute numeric kernel UserNS level still not measured |
| **N2c: profile visibility and revision event semantics** | `O_NONBLOCK` `/sys/kernel/security/apparmor/revision` and `policy/revision` both `EAGAIN(11)`, poll no readable event; `/sys/kernel/security/apparmor/profiles` read EOF, zero bytes | Revision and policy/revision both read `190`, poll readable; `profiles` open returns `EACCES(13)` | Strong new visibility split. Kernel `ns_revision_read` **waits** when revision unchanged and file is blocking; `O_NONBLOCK` returns EAGAIN. This is a strong *candidate* explanation for the old L0 E5 timeout; **old L1 E5 timeout exact path remains unproven**, and the new N2c did not hang |
| **N3: NSFS/owner/control** | For mount/pid/net/uts/ipc/cgroup, `NS_GET_USERNS` matches the Box's userns inode; `NS_GET_PARENT=EPERM`; original UserNS owner UID appears as `65534` in guest view; time namespace owner ioctl EPERM | Same pattern with different userns inode; AA `.ns_level=1` for both root/UID1001 | `65534` is mapped overflow view, **not an actual trusted host owner ID**, nor numeric kernel UserNS level |
| **N4: 0→1→2 ephemeral UserNS** | Root creates two further UserNS, with distinct inodes and new `0 0 1` maps; AppArmor namespace/name, enforced state and **`.ns_level=1` unchanged**. UID1001 first child `unshare -Ur` fails | Root likewise creates two UserNS and retains same AppArmor namespace/level. UID1001 succeeds creating one, mapping guest 1001 to child UID0 and retaining the outer enforced profile | Plain UserNS recursion does not automatically delegate per-generation AA policy namespace |
| **N5: three independent raw syscall subjects** | UID/GID 1001/1001, 1002/1002, 65534/65534, all zero effective caps: **raw `unshare(CLONE_NEWUSER)` = -1, EACCES(13) for all three** | Exactly the same matched pairs, same effective caps: **all three raw calls succeed**, create a new UserNS, but `.ns_level=1` and AA label/name remain unchanged | Not a UID1001-specific failure. Both show `Seccomp=2`, `Seccomp_filters=1`, same AA userns feature `pciu&`, `apparmor_restrict_unprivileged_userns=1`, `apparmor_restrict_unprivileged_unconfined=0`. **Filter count equality is NOT filter content equality** and errno alone does not identify the rejecting subsystem |
| **N6: local default UID/GID ownership / DAC** | root, UID1001, UID1002 each create distinct marker file `0600` with the expected visible owner UID/GID. Cross-user UID1001→UID1002 and reverse reads each `EACCES`; root reads all. Exact markers removed, none remaining | Same all-pass local file ownership and cross-user DAC controls; exact markers removed | No default local UID/GID file owner corruption observed; this does **not** prove host-layer ID ranges are isolated across different Boxes or establish policy delegation |

**New direct evidence absent in v1:** `.ns_level=1` both controller depths, unchanged at UserNS depth0/1/2; and `ns_revision_read` event semantics explaining why a naive blocking `read()` can hang. `userns_create=pciu&` is a feature bitmask string, **not** evidence that each process is authorized to create a UserNS or load AA policies. Prior root-with-CAP_MAC_ADMIN + `apparmor_parser` rc243 remains relevant.

**Source evidence for revision behavior:** [Linux `apparmorfs.c`, `ns_revision_read`](https://code.googlesource.com/linux/torvalds/linux/+/abcbd3bfbbfe97a8912d0c929d4aa18f50d9bc52/security/apparmor/apparmorfs.c) returns `-EAGAIN` for nonblocking file when revision unchanged, otherwise waits on namespace revision notification. Thus the old E5 wrapper's read-to-end approach was an invalid probe of event-style files. No policy was modified to trigger a revision event.

### Complete original Exec pairs (13 per layer, 26 overall)

| N | Probe | L0 BoxWeave original / native Incus | L1 BoxWeave original / native Incus |
| --- | --- | --- | --- |
| N1 | Root | `0d8cef6cb3b4c903259158079ff109a1` / `ad988956-fe2f-436f-974c-c32af9ba73a5` | `8c91c2d83899444b57ad501980f95598` / `a3589f11-8c64-41ce-977f-6d2699ad987f` |
| N1 | UID1001 | `2d986841174b01c4db62680b8a54cf7f` / `dcf1db4b-5943-4688-b8de-b045fe846c0c` | `f33b5c79f0e21bd13ae6a23d1167cf40` / `9268bfd4-344a-4188-8978-ada0c23852f1` |
| N2 | Stat/list | `a951db95a8b2b4699409fe8ec8c32a5b` / `e38fdd7f-5e60-486b-accc-95f1fbc06d58` | `2ad6526cb30d07d35fe0f37d12f18cc4` / `39e98099-6c70-41e0-b1b1-eae3c0037805` |
| N2 | Namespace/features/revision values | `edb46b2f0fa0361e251411e21bcf8f6e` / `1efbb2a1-5886-4238-b37e-7840038c2d25` | `f75fefaadc9af3f630f58d76c7db9306` / `e02be05a-f874-481d-9f47-08d20ecda002` |
| N2 | Nonblocking profile/revision read + poll | `38c8be335c16fe50f9a837f58920fcf3` / `32af44e4-b7a9-418c-80c9-617d888f7cd5` | `a6d1c6aaaa6271dbc9b7d359a130f81b` / `71efcbea-5d48-4807-9575-a52f57a8c562` |
| N3 | Root NSFS owner | `a2d653e6c495271962ce9091fadd8fc9` / `9f3154df-b67b-4a09-8c9b-cf5439c3aa4f` | `788b49f1812d2d4c7ec8b55624f4cbec` / `773a0b1f-0e6a-4070-b110-7aa383ed697e` |
| N3 | UID1001 NSFS owner | `142b27cc648939bef520c0c5ae9593df` / `91746f0a-8264-4879-99e3-c148ede9136c` | `82b004a7de30f270013da64be73e0e28` / `1f466768-a80d-40db-b37a-74db2b5839c1` |
| N4 | Root two-level UserNS | `0ee455423dcbeb7dd03a69bb6b937e0b` / `f51ca586-8a70-4575-9e1f-b44c31e7a873` | `6980d014bd3e588b2bfeb204b2625c9d` / `f591ef64-8dc6-49fe-9301-7142849106bf` |
| N4 | UID1001 one-level attempt | `5c0b6b7e7daab689eb60960f01f6e7dc` / `ec2f2c51-13dc-4285-bcd2-9b87cc86bbf4` | `7f1f5104a2e839cd9f06f51eb4cab303` / `23178570-756f-4f68-af8f-51efb3a362d4` |
| N5 | UID/GID1001/1001 | `d51f79cd639496cf1e38339567fc15e1` / `2396b829-a8e3-4163-b693-580c22d99de5` | `a1cacde2f2e56c479c332c644a9aaca9` / `ad316fd4-4281-41a2-85d6-dba8eb0b1941` |
| N5 | UID/GID1002/1002 | `53ed9b50a8df7fa9b946b94a3bd5e8b1` / `808314d6-a931-4cf0-a1b9-a026345dd8d5` | `61e88eae3b033f12d35c14f7d1cbe52e` / `aa7e4618-62b2-4a7d-ab55-9a2c74bcdfc9` |
| N5 | UID/GID65534/65534 | `27abccc2feae9b6c4ff605ca8da5f707` / `c5153fba-21d9-4593-a275-1c7e89922a3e` | `ef10b388a0934d4c7168be0b6f12ca08` / `a5623684-579c-4dfb-94b4-67d246b69b7c` |
| N6 | Disposable marker/DAC matrix | `2d170258ae2be101d04c5ac0868e31cf` / `f61a0f89-74a0-493c-a6e7-3ab37ac2c0c0` | `641ecdd98538f0508292ed52e1770254` / `25b0eefe-618e-4050-9f11-1cbdfa742579` |

Each above has an independently read `operation_get: state=SUCCEEDED` and an original output page with `exit_code=0`, `command_success=true`. Scripts explicitly journal their own **inner** success/denial separately. N2's own step-level marker-before/after design identified the exact path and op for every probe and **no timeouts occurred**. N6 created/read/deleted ONLY six exact marker files within these new Boxes, three per layer.

### N7 — exact safe teardown and independent postcondition

| Control | L0 original BW / native Incus | L1 original BW / native Incus |
| --- | --- | --- |
| STOP — SUCCEEDED | `aecc37904d5435d35393ac0f3d75eea6` / `8fc1ed58-dcb0-4521-ab2c-bcd696852b26` | `964be29f0c0cda2237908fa747badcd1` / `2884aeb6-0499-4cf7-77281b17c486` |
| DELETE — SUCCEEDED | `9fa202ef3b8d2330d67d4b17e63b47e3` / `249836f9-772d-4795-99a4-4823ccb24afd` | `068f97093fbfdf25c9ddca8c825cd98c` / `43b69b84-0e45-42b2-bd03-7f132aa12c96` |

Authenticated `operation_list(work_ref=BH019-NXM-V2-1114)`: **16 original operations per controller** (Create+13 Exec+Stop+Delete), **all state SUCCEEDED** with original native Incus IDs. New independent final `box_list` shows exactly **4** pre-existing L0 and **9** pre-existing L1 Boxes; neither `bh019-nxm2-*` exists. No original UNKNOWN touched, no new UNKNOWN observed.

### What v2 learned / prioritized follow-up

1. **Most decisive new read-only source observation:** AA `.ns_level=1` on both parent and deeper child views, and still 1 through two further real UserNS creations; no AppArmor child policy namespace magically appeared. This reinforces stock Incus 7.6 `AppArmorStacked` guard / inherited-policy-view hypotheses without claiming directly measured absolute UserNS kernel `level`.
2. **E5 is a *measurement harness flaw*, not proven kernel hang:** reading event-driven AppArmor revision file as ordinary blocking text is unsafe, especially when revision unchanged (L0 O_NONBLOCK EAGAIN). On L1, revision=190; the precise old L1 E5 timeout path remains undetermined, so do not retroactively assert both timeouts had the same root cause. Retain bounded, nonblocking per-path reads and staged markers in future verification tooling.
3. **Nonroot raw UserNS creation denial difference repeated across 3 UID/GID pairs**, with same exposed seccomp mode/filter count and feature flags; cannot distinguish underlying LSM/seccomp filters or host audit without more appropriate read-only evidence. **Do not change policies or guess an AppArmor rule** from user-visible `unconfined`.
4. **UID/GID local DAC baseline passed**, so local owner corruption is not established as a current BH019 blocker. Distinct mapped parent IDs and cross-Box storage/tenant isolation remain unproven, not waived.
5. **Missing decisive security acceptance:** no independent per-depth child AppArmor policy namespace, no unique local profile load/reload/remove, no verified ancestor/sibling denial, no safe L2→L3 daemon lifecycle proof. Do not manually write into an inherited AA namespace. **BH-019 remains BLOCKED; PR #146 stays Draft and unmerged.**

This closes the **N×M v2 diagnostic run**, not the BoxWeave product bug.
