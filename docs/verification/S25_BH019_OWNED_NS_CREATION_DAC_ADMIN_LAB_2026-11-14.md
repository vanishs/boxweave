# S2.5 BH-019 — owned AppArmor policy-namespace creation: independent DAC/admin gates (2026-11-14 PT)

> **Disposition:** A uniquely named, empty child AppArmor policy namespace **could not be created**, even from root in a freshly created L0 Box with its own visible AppArmor policy namespace. `mkdir` returned **EACCES (13)**, no child was created, and an independent control showed **W_OK=false** on the parent `policy/namespaces` inode. This is **positive denial evidence**, not proof of the specific rejecting kernel stack branch. All 7 durable operations and their original Incus IDs read back terminal SUCCEEDED, guest probes exited 0 while explicitly capturing the *inner* denial, the one-use Box was successfully stopped/deleted, final L0=4/L1=9 inventories unchanged. **BH-019 BLOCKED.**

## Scope and safe experimental preconditions

Owner excludes version/distro/patch matrices, BPF and VM alternatives. This study targets only the currently installed kernel AppArmor, securityfs, Incus/LXC, userns/UID mapping and scoped nonprivileged policy administration.

We did **not** run the potentially state-mutating namespace mkdir in L1: the L1-produced Box inherited an older outer AppArmor namespace, so a successful write could have targeted a shared ancestor policy view. No shared host settings, old Box, current policy profiles, parser writes, cap overrides, `raw.lxc`, privileged mode, SELinux, or historical UNKNOWN IDs were touched.

A new L0 RC5 (`1224845c`) one-use unprivileged `security.nesting=true`, internet-off Box was created:

| Item | Value |
| --- | --- |
| Exact test Box | `bh019-owned-aa-ns-1114`; UUID `b76fe801-8895-48f5-b353-b8f32157212d` |
| Work ID | `BH019-POLICY-DELEGATION-LAB-1114` |
| Original create BW / native Incus | `b8a2950db8d128a122a0efd88fdd951d` / `c02dd423-e18c-4230-9ea8-a8f15f361e56` |
| CPU / RAM / disk | 1 CPU, 1024MiB, 8GiB; no internet; nesting true |
| Precondition readback | `/sys/kernel/security/apparmor/.ns_name=incus-user-1001_bh019-owned-aa-ns-1114_<var-lib-incus>` (**names this new Box**), `.ns_level=1`, `.ns_stacked=yes`, guest `/proc/self/attr/current=unconfined` (relative label), UserNS inode `user:[4026533790]`, `CapEff=000001ffffffffff`, `Seccomp=2`, `Seccomp_filters=1` |
| Policy view/control | `/sys/kernel/security/apparmor/profiles` opened read-only; `policy/namespaces` list **empty**. `policy` is the dynamic magic symlink `apparmorfs:[1177]`. securityfs was **mounted rw**, which is **not** proof of per-directory write authorization |
| UserNS map | `0 1000000 1001\n1001 1001 1\n1002 1001002 999998998` |

All preflight checks yielded `safe_preconditions=true`. The test did **not** assume visible `unconfined` disproves enforced ancestor policy; it only required this Box's **own** separate named AppArmor policy-view namespace.

## N1 — one bounded real namespace creation attempt, precise result

Under guest root in the newly created Box, after checking namespace uniqueness and absence of existing child entries, the test attempted exactly:

```python
# Namespace name was fixed to this disposable test and checked absent.
os.mkdir("/sys/kernel/security/apparmor/policy/namespaces/bh019-ownchild-probe-1114", 0o700)
```

Events emitted before and after the syscall:

```json
{"phase":"BEGIN","precondition":true,"apparmor_level":"1","existing_children":[]}
{"phase":"MKDIR_START","name":"bh019-ownchild-probe-1114"}
{"phase":"MKDIR_DENIED","errno":13,"name":"EACCES","duration_ms":0.05}
{"phase":"END","children":[],"child_still_present":false,"apparmor_level":"1"}
```

No child namespace was created, so no profile loading, label transition, or `rmdir` against an existing namespace was attempted. A denied guest syscall was recorded inside a diagnostic process that itself exited 0; **outer BoxWeave `SUCCEEDED` is not a claim `mkdir` succeeded**.

Original Exec receipt: BW `7921b82afbafe9f7d67db9431587cbbd`, native Incus `aefe0798-c123-43b9-9b1f-0a1c871585cb`; guest exit0, `command_success=true`, inner mkdir EACCES.

## N2 — read-only inode/DAC evidence (no policy mutation)

`stat` and `os.access(... W_OK, effective_ids=true)` on relevant nodes:

| Path | stat mode | Guest-visible UID:GID | Guest root W_OK |
| --- | --- | --- | --- |
| `/sys/kernel/security` | `0755` | `65534:65534` | **false** |
| `/sys/kernel/security/apparmor` | `0755` | `65534:65534` | **false** |
| `/sys/kernel/security/apparmor/policy` (magic symlink dereferenced) | `0755` | `65534:65534` | **false** |
| **`/sys/kernel/security/apparmor/policy/namespaces`** | **`0755`** | **`65534:65534`** | **false** |
| `/sys/kernel/security/apparmor/.load` | `0666` | `65534:65534` | **true** |
| `/sys/kernel/security/apparmor/policy/.load` | `0640` | `65534:65534` | **false** |
| `/sys/kernel/security/apparmor/profiles` | `0444` | `65534:65534` | **false** |

`stat` **UID65534 is a guest-visible overflow conversion of an unmapped outside UID**, not proof the real securityfs owner has the host UID65534. This matches Linux `user_namespaces(7)` documented overflow behavior: https://manpages.debian.org/trixie/manpages/user_namespaces.7.en.html

The parent directory is 0755, not writable by mapped guest root; guest root capabilities are **local to its UserNS** and do not automatically give global securityfs DAC ownership or write authority. Hence the new mkdir EACCES can be caused **at a VFS/DAC or LSM pathname-permission gate before `ns_mkdir_op` executes its own `aa_may_manage_policy` check**. The evidence **does not** instrument the exact kernel return site. There may be **multiple** independent blockers.

Original Exec: BW `0224fbba2b5a8dab57e5de4411126442`, native `cac21d21-e791-41e5-93be-4f44795f5366`, guest exit0.

## N3 — UID/GID/capability contrast with OPEN ONLY (NO write)

Exactly three users were dispatched in short-lived local children within the **same one-use Box**: UID:GID = `0:0` (full local capabilities), `1001:0` (CapEff0), `1001:1001` (CapEff0). The only probes were `os.access(R_OK/W_OK)` and bounded `os.open` followed immediately by `os.close`:

| Subject | `policy/namespaces` W_OK | Top-level `.load` O_WRONLY open-only | `policy/.load` O_WRONLY open-only | `profiles` O_RDONLY open-only |
| --- | --- | --- | --- | --- |
| UID0/GID0 (full CapEff) | **false** | **OPEN OK** | EACCES | OPEN OK |
| UID1001/GID0 (CapEff0) | **false** | **OPEN OK** | EACCES | OPEN OK |
| UID1001/GID1001 (CapEff0) | **false** | **OPEN OK** | EACCES | EACCES |

This is important: ***opening*** a world-writable `.load` file is **not** permission to ***load*** a policy. The kernel's `profile_load → policy_update → aa_may_manage_policy` check runs during **write**, which was **never** attempted in this test. Similarly, directory visibility and a `rw` securityfs mount do not confer the ability to `mkdir` a child policy namespace.

Exact original Exec: BW `4f29bd891b853c0663dd3f1b44b054c9`, native `a6468b2d-206e-40f2-b3cc-565297d2367e`, guest exit0.

## Independent kernel/source alignment and limits

- Kernel AppArmor `ns_mkdir_op` directly invokes `aa_may_manage_policy(...AA_MAY_LOAD_POLICY)` before `__aa_find_or_create_ns`; `ns_rmdir_op` also checks policy-admin rights. However, it only runs after the VFS accepts the directory creation operation. https://code.googlesource.com/linux/torvalds/linux/+/4bf54e47525dcacd4c6cdd97fb5902592414dd7a/security/apparmor/apparmorfs.c
- The `policy_update` / `.load` write handler independently calls `aa_may_manage_policy`; a successful **open-only** is not the privileged write. Same kernel source.
- For noninit userns, `aa_policy_view_capable` requires matching UserNS and AppArmor visible policy namespace levels plus root-effective UID or GID. `aa_policy_admin_capable` further requires suitable `CAP_MAC_ADMIN` in the proper policy context and unlocked policy. https://code.googlesource.com/linux/torvalds/linux/+/4bf54e47525dcacd4c6cdd97fb5902592414dd7a/security/apparmor/policy.c
- Incus v7.6's `AppArmorStacked` guards still skip automatic creation/profile transitions for deeper nested instances; LXC's own stack guard has the same issue. https://sources.debian.org/src/incus/7.6.0-1~exp1/internal/server/apparmor/apparmor.go
- **Proven**: guest root in this own-AA-namespace Box could view local policy, not `mkdir` a child through the normal filesystem path; namespaced directory W_OK=false; no child was created.
- **NOT proven**: exact instrumentation of VFS-versus-AA policy denial order, whether a parent-authorized helper could allocate a namespace securely, safe transition into child namespace with enforced ancestors, any child policy load/replace/remove, sibling/ancestor negative enforcement, or functioning L2→L3 independent incusd. **Do not change owner bits, mount options, host AppArmor, raw.lxc, privilege, or capabilities to force a green test.**

## Original receipts + safe cleanup

| Action | Original BW Operation ID | Native Incus Operation ID | Result |
| --- | --- | --- | --- |
| Create new Box | `b8a2950db8d128a122a0efd88fdd951d` | `c02dd423-e18c-4230-9ea8-a8f15f361e56` | SUCCEEDED |
| N0 preflight | `8b5ab56edbfb460cd72331da4fc6b41e` | `195cb6fe-04f8-4c6c-882d-f3eb69db144f` | SUCCEEDED, guest exit0 |
| N1 one exact namespace mkdir | `7921b82afbafe9f7d67db9431587cbbd` | `aefe0798-c123-43b9-9b1f-0a1c871585cb` | SUCCEEDED, guest exit0; **inner mkdir EACCES**, no namespace created |
| N2 mode/owner/access | `0224fbba2b5a8dab57e5de4411126442` | `cac21d21-e791-41e5-93be-4f44795f5366` | SUCCEEDED, guest exit0 |
| N3 UID/GID open-only contrast | `4f29bd891b853c0663dd3f1b44b054c9` | `a6468b2d-206e-40f2-b3cc-565297d2367e` | SUCCEEDED, guest exit0, **no writes** |
| Stop exact Box | `3d76dc561093e952f278a3bdd3b0652e` | `4bd894d2-cb0b-475d-986c-4b5ab6ea3cbc` | SUCCEEDED |
| Delete exact Box | `138a0dbc3c60a59bb9817cac1067b684` | `bd3efa97-d4e5-4792-a29b-082388e1a79e` | SUCCEEDED |

Authenticated `operation_list(work_ref=BH019-POLICY-DELEGATION-LAB-1114)` returned all **7** original receipts and their native IDs, all outer terminal `SUCCEEDED` (inner expected denial separately captured). Post-delete `box_list` returned **exact pre-existing 4 L0 and 9 L1 Boxes**, new Box absent. No new UNKNOWN, no prior UNKNOWN replay/alteration.

## Engineering decision

**New dual-gate research finding:** before designing AppArmor policy *admin* delegation, an unprivileged nested controller must have a **legitimate way to create a scoped AppArmor child namespace**, including both filesystem access to the relevant kernel namespace interface and LSM policy-admin authorization, while retaining ancestor enforcement. The current Box's userns-local CapEff, ownership of the Incus Box, `security.nesting=true`, `rw` securityfs mount and successful policy *view* do **not** satisfy this. A parent-helper/namespace transition idea remains an **unvalidated design**, not a supported BoxWeave fix. No unsafe security weakening or one-flag workaround is accepted.

**BH-019 BLOCKED; PR #146 Draft and unmerged; this is diagnostic evidence, not product acceptance.**
