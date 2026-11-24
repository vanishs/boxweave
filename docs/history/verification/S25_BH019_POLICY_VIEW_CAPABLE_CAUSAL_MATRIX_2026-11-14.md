# BH-019 — AppArmor `aa_policy_view_capable` causal N×M lab (2026-11-14 PT)

> **Status: read-only authorization path strongly reproduced, policy delegation NOT repaired.** Two newly created and deleted L0/L1 unprivileged Incus Boxes, no host profile or policy changes, no profile load/replace/remove, no new UNKNOWN, no old UNKNOWN interaction. No version/distro/patch/BPF/VM research. PR #146 remains Draft.

## Why this test is more decisive than earlier `apparmor_parser rc243`

The kernel AppArmor source contains these **separate** authorizations:

1. `labels_view(label)` is the AppArmor policy namespace associated with the current label, via `labels_ns` macro.
2. `aa_ns_visible(view_ns, ns, true)` is **true for the same namespace** (`curr == view`).
3. `aa_policy_view_capable(cred,label,ns)` requires a **UserNS-local root identity** (effective UID0 **or** effective GID0), the target policy namespace visible from the label's view, and for a noninitial UserNS both `unprivileged_userns_apparmor_policy != 0` and `user_ns->level == view_ns->level`.
4. `aa_policy_admin_capable` **additionally** requires both kernel `cap_capable(cred,userns,CAP_MAC_ADMIN)`, AppArmor `aa_capable(label,CAP_MAC_ADMIN)`, and that global policy is **not locked**.
5. `profiles_open` on `/sys/kernel/security/apparmor/profiles` explicitly calls `aa_current_policy_view_capable(NULL)`, returning `-EACCES` if it fails. **This makes opening the read-only profiles list a direct, minimally invasive view-permission probe rather than attempting to write any policy.** File-path mediation may still apply; a raw `EACCES` alone cannot prove which internal branch denied it.

Source links checked:
- [Linux `policy.c`: view, admin and policy manage checks](https://linux.googlesource.com/linux/kernel/git/torvalds/linux/+/54ac9ff8f1196afc49d644a1625e0af1c9fcf7f5/security/apparmor/policy.c)
- [Linux `apparmorfs.c`: `profiles_open` calls view-capable check](https://linux.googlesource.com/linux/kernel/git/torvalds/linux/+/441c63ff42c4e666304cdd32d23b5fc6bc1ea3cc/security/apparmor/apparmorfs.c)
- [Linux `policy_ns.c`: same-namespace visibility is reflexive](https://linux.googlesource.com/linux/kernel/git/hid/hid/+/63724e9519a312d7d0b8767d0aeb53bc15a7fdd5/security/apparmor/policy_ns.c)
- [Linux `label.h`: `labels_view(X) = labels_ns(X)`](https://kernel.googlesource.com/pub/scm/linux/kernel/git/powerpc/linux/+/d0b224cf9ab12e86a4d1ca55c760dfaa5c19cbe7/security/apparmor/include/label.h)
- [Incus `driver_lxc.go` AppArmor config branch](https://sources.debian.org/src/incus/7.6.0-1~exp1/internal/server/instance/drivers/driver_lxc.go): if currently confined **or** not AppArmorAdmin, it inherits current profile; in the independent profile path, a new namespace expression is chosen only for `AppArmorStacking && !AppArmorStacked`.

### Fixtures

| Layer controller | Isolated fixture | Original BoxWeave Create / native Incus Create |
| --- | --- | --- |
| L0 RC5 (`1224845c`) | `bh019-aa-view-l0-1114` / `0a66285e-c592-4ebb-9c7c-be070d442acd` | `262fcbdd80dad695bb06601307da59a1` / `80afd45e-b2d6-408a-8217-f205a1d749ae` |
| L1 RC4 (`c3023a72`) | `bh019-aa-view-l1-1114` / `5bcbdebf-9e43-4570-95d9-a9640d01c68f` | `d0e7776e44312c149818cc0294bbbbc9` / `a70e2519-53dd-42aa-a0bf-56e4aa73fcff` |

Both: 1 CPU, 1536 MiB, 10 GiB, internet disabled, nonprivileged, `nesting=true`. Authenticated L0 preexisting Box inventory was 4, L1 was 9, with none altered. The independent native Incus service is 7.6 at both controller layers. Each new Box belongs to original authorized restricted project `user-1001`.

## N1 × M1: actual read-only policy view by transient UserNS depth

Each root probe ran `/usr/bin/python3` and created **at most two ephemeral descendants** with `unshare --user --map-root-user --fork`. For every depth, it recorded UserNS inode, exact UID/GID map, `CapEff`, Seccomp, `/proc/self/attr/current`, AppArmor `.ns_name` and `.ns_level`, and **`open('/sys/kernel/security/apparmor/profiles', O_RDONLY|O_NONBLOCK)` with bounded 180-byte read**. No AppArmor policy write/open-for-write occurred.

| Layer/actual transient depth | UserNS inode | AppArmor namespace level | AppArmor label/namespace name | `CapEff` | `profiles` open |
| --- | --- | --- | --- | --- | --- |
| L0, initial Box UserNS (0) | `4026532378` | **1** | relative `unconfined`, own new Box AA namespace | `000001ffffffffff` | **OPEN OK, EOF, 0 bytes** |
| L0, first additional UserNS (1) | `4026533921` | **1** | **unchanged** | `000001ffffffffff` | **EACCES(13)** |
| L0, second additional UserNS (2) | `4026533926` | **1** | **unchanged** | `000001ffffffffff` | **EACCES(13)** |
| L1, initial Box UserNS (0) | `4026533859` | **1** | enforced own Box profile, inherited outer AA namespace | `000001ffffffffff` | **EACCES(13)** |
| L1, first additional UserNS (1) | `4026533855` | **1** | unchanged | `000001ffffffffff` | **EACCES(13)** |
| L1, second additional UserNS (2) | `4026533921` | **1** | unchanged | `000001ffffffffff` | **EACCES(13)** |

All children used one-ID `uid_map=0 0 1` and `gid_map=0 0 1`. Their per-guest process UserNS inode changed, while the AppArmor policy namespace's name, reported level and effective process label did **not**. They terminated automatically after producing results. L0 first-depth A/B is the strongest causal evidence: **same guest root EUID, effective caps, AA profile view/name/level, but new UserNS → view probe switches from success to EACCES**.

This is not a direct instrumented read of `user_ns->level`: the earlier `NS_GET_PARENT` remained EPERM. Under the verified **upstream** kernel policy check, since the original L0 root's `profiles_open` passes with a noninitial UserNS and AA `.ns_level=1`, its UserNS would also have level1; L1, which has the deeper UserNS but sees inherited AA `.ns_level=1`, matches the expected mismatch. This is **strong causal inference**, not a claim that the exact kernel's secret numeric `user_ns->level` was measured.

## N2 × M2: orthogonal effective UID/GID and CAP contrast

Separate `box_exec` identities used **explicit guest_uid/guest_gid**, without changing policy. Each recorded `Uid`, `Gid`, `CapEff`, UserNS, AA label/name/level, and read-only profiles-open status. `policy/namespaces` and `policy/profiles` could be opened read-only as directories in all cases, showing **filesystem directory presence is not equivalent to profile view authorization**.

| UID / GID, root view predicate expected | L0 `CapEff` / profiles open | L1 `CapEff` / profiles open |
| --- | --- | --- |
| `0/0` (prior depth0 baseline) | full / **OK** | full / EACCES |
| `0/1001` (UID0 true) | full / **OK** | full / EACCES |
| **`1001/0` (GID0 true, NO caps)** | **zero / OK** | zero / EACCES |
| `1001/1001` (both false) | zero / EACCES | zero / EACCES |

**Important new nuance:** UID1001/GID0 with **no effective capabilities** can view the empty L0 AppArmor profiles list. This is consistent with the kernel's explicit `effective UID 0 OR effective GID 0` check, while `CAP_MAC_ADMIN` is additionally required for **management**. It is NOT a claim that UID1001/GID0 can load policy. On L1 even root with full capabilities cannot open the list, supporting a second independent view boundary, consistent with AA/UserNS depth mismatch.

## Exactly verified original Exec receipts

All eight original BoxWeave Exec Operations had `state=SUCCEEDED`; the original native Incus operation IDs and original stdout were read back; `exit_code=0`, `command_success=true`. Inner kernel denial is accurately recorded **as EACCES**, not falsely marked as a successful policy view.

| Experiment | L0 original BW / native Incus | L1 original BW / native Incus |
| --- | --- | --- |
| N1 depth0→1→2 root tree | `363dd572fc997e2101128265d30acf82` / `735062fc-ce7f-4941-81e1-cb67488b168d` | `2f3fe80ed093938cbcf330537b81b131` / `d8458df8-3e1a-4637-b3dc-e605c866c3c5` |
| N2 `UID0/GID1001` | `c41e6216f5ce1a0620913dfef4118840` / `28c7ff01-9a50-41d1-b01c-dcedd7ef0366` | `fadd92e8c7717d4266e31e58d355578c` / `5891a66a-48f4-48fa-b41a-f940db500720` |
| N2 `UID1001/GID0` | `ee1dd8be25772d869c62f2468aa8952a` / `32b0f8b9-a1b0-42bd-a88d-8263abcb3529` | `b522d1c8ad5b4a57b75561951aee5a08` / `a837f477-0146-4aa9-87b9-b5164eb2713e` |
| N2 `UID1001/GID1001` | `1b6ad6d28f0db3add15f795973fa1f7b` / `aa474894-1311-4060-a80c-29d188376d0d` | `5a7ff87de2c3a69fc14c1b1edbaa1233` / `ece1c5b9-3991-456c-a6f0-4d0d8cc46a9a` |

## Formal teardown / no residual state

| Control | L0 original BW / native Incus | L1 original BW / native Incus |
| --- | --- | --- |
| STOP exact new UUID: SUCCEEDED | `779e6ae7e6f435cc18f63605b814ca2c` / `b7154a72-3fa0-41b3-a0c4-e8a202200aef` | `f410faca14ad426d6d29112e39cbc636` / `21644256-f21e-4202-9d55-fc23bfe6c362` |
| DELETE exact new UUID: SUCCEEDED | `cecae9a2fb72b50a23dfde12a28b732b` / `7f4031b2-c1ba-4b2f-9c36-36df29632c66` | `53ffb07808367a4b9e4b8484beb68e4c` / `5bd2bee0-4ee1-4981-91bf-d68a613a9aed` |

Authenticated `operation_list(work_ref=BH019-AA-VIEW-CHECK-1114)` on each layer returned **7 immutable original receipts = Create+4 Exec+Stop+Delete, all SUCCEEDED**. Independent `box_list` after deletion showed exact prior inventory **L0 4 / L1 9**, neither new Box present, no new UNKNOWN, no historical UNKNOWN touched.

## Root-cause conclusion and proper engineering boundary

**High-confidence, source-compatible mechanism:**

```
L0: current UserNS (likely kernel level 1) ≈ visible AppArmor NS level 1
    → can view policy (given UID0 or GID0)
    → one more transient UserNS: label/view AppArmor NS remains level 1
    → profiles_open EACCES despite CapEff full
L1: existing deeper UserNS with inherited AppArmor NS level 1
    → profiles_open EACCES at root
    → Incus source includes already-confined / !AppArmorAdmin inheritance branch
       and AppArmorStacked guard against creating another namespace
    → previous genuine parser "not policy admin" / exit243 is consistent
```

**Two independent chokepoints** remain: `aa_policy_view_capable` (identity, visible view and level alignment) and `aa_policy_admin_capable` (CAP_MAC_ADMIN under two policy checks + unlocked policy); Incus/LXC's current inherited/staked handling determines whether a correctly aligned deeper AppArmor namespace is provided. A green `security.nesting`, a deeper UserNS with fresh root capabilities, or switching process identity **does not solve safe per-depth policy administration**. Merely dropping or relaxing ancestor confinement, manually writing into inherited AA policy, or bypassing kernel gates would not meet BoxWeave's tenant safety goal.

**Remaining evidence required before BH-019 PASS:** a real independently owned AA policy namespace for the next depth with strict ancestor/sibling denies, safe local profile load/replace/remove confined to that namespace, and successful next-level independent Incus daemon/child lifecycle. These were intentionally **NOT RUN** on shared L0/L1. No final fix was developed here. BH-019 **BLOCKED**. PR #146 Draft and unmerged.
